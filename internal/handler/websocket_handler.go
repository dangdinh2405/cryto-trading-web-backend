package handler

import (
	"context"
	"encoding/json"
	"log"
	// "math"
	"net/http"
	"sync"
	"time"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Message types for client-server communication
type WSMessage struct {
	Type    string   `json:"type"`    // "subscribe", "unsubscribe"
	Symbols []string `json:"symbols"` // ["BTC/USDT", "ETH/USDT"]
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	symbols     map[string]bool // Subscribed symbols
	symbolsLock sync.RWMutex
}

// CandleTracker tracks current 1-minute candles for each symbol
type CandleTracker struct {
	candles        map[string]*models.OHLCV // symbol -> current candle
	mu             sync.RWMutex
	lastTradeCheck time.Time // Last time we checked for trades
}

func NewCandleTracker() *CandleTracker {
	return &CandleTracker{
		candles:        make(map[string]*models.OHLCV),
		lastTradeCheck: time.Now(),
	}
}

// UpdateWithTrade updates candle with a new trade
func (ct *CandleTracker) UpdateWithTrade(trade repo.Trade, currentMinute time.Time) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	candle, exists := ct.candles[trade.Symbol]
	if !exists {
		// Create new candle
		candle = &models.OHLCV{
			Symbol:    trade.Symbol,
			OpenTime:  currentMinute,
			CloseTime: currentMinute.Add(time.Minute),
			Open:      trade.Price,
			High:      trade.Price,
			Low:       trade.Price,
			Close:     trade.Price,
			Volume:    trade.QuoteAmount,
		}
		ct.candles[trade.Symbol] = candle
	} else {
		// Update existing candle
		if trade.Price > candle.High {
			candle.High = trade.Price
		}
		if trade.Price < candle.Low {
			candle.Low = trade.Price
		}
		candle.Close = trade.Price
		candle.Volume += trade.QuoteAmount
	}
}

// GetCurrentCandles returns copy of all current candles
func (ct *CandleTracker) GetCurrentCandles() []models.OHLCV {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	candles := make([]models.OHLCV, 0, len(ct.candles))
	for _, candle := range ct.candles {
		candles = append(candles, *candle)
	}
	return candles
}

// ResetCandle resets candle for a symbol (start new minute)
func (ct *CandleTracker) ResetCandle(symbol string, newMinute time.Time, lastClose float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.candles[symbol] = &models.OHLCV{
		Symbol:    symbol,
		OpenTime:  newMinute,
		CloseTime: newMinute.Add(time.Minute),
		Open:      lastClose,
		High:      lastClose,
		Low:       lastClose,
		Close:     lastClose,
		Volume:    0,
	}
}

type Hub struct {
	clients       map[*Client]bool
	broadcast     chan []models.OHLCV
	register      chan *Client
	unregister    chan *Client
	mu            sync.RWMutex
	marketRepo    *repo.MarketRepo
	candleTracker *CandleTracker
	currentMinute time.Time
}

func NewHub(marketRepo *repo.MarketRepo) *Hub {
	now := time.Now()
	return &Hub{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan []models.OHLCV, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		marketRepo:    marketRepo,
		candleTracker: NewCandleTracker(),
		currentMinute: now.Truncate(time.Minute),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

		case candles := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Filter candles based on client's subscriptions
				filteredCandles := client.filterCandles(candles)

				if len(filteredCandles) > 0 {
					data, err := json.Marshal(filteredCandles)
					if err != nil {
						log.Printf("Error marshaling candles: %v", err)
						continue
					}

					select {
					case client.send <- data:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// filterCandles returns only candles for symbols the client subscribed to
func (c *Client) filterCandles(candles []models.OHLCV) []models.OHLCV {
	c.symbolsLock.RLock()
	defer c.symbolsLock.RUnlock()

	// If no symbols subscribed, return all (default behavior)
	if len(c.symbols) == 0 {
		return candles
	}

	var filtered []models.OHLCV
	for _, candle := range candles {
		if c.symbols[candle.Symbol] {
			filtered = append(filtered, candle)
		}
	}
	return filtered
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle subscribe/unsubscribe messages
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		c.handleMessage(wsMsg)
	}
}

func (c *Client) handleMessage(msg WSMessage) {
	c.symbolsLock.Lock()
	defer c.symbolsLock.Unlock()

	switch msg.Type {
	case "subscribe":
		for _, symbol := range msg.Symbols {
			c.symbols[symbol] = true
		}
		log.Printf("Client subscribed to: %v (total: %d)", msg.Symbols, len(c.symbols))

	case "unsubscribe":
		for _, symbol := range msg.Symbols {
			delete(c.symbols, symbol)
		}
		log.Printf("Client unsubscribed from: %v (remaining: %d)", msg.Symbols, len(c.symbols))

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

func (h *Hub) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		hub:     h,
		conn:    conn,
		send:    make(chan []byte, 256),
		symbols: make(map[string]bool),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// StartCandleBroadcaster aggregates trades into candles and broadcasts
func (h *Hub) StartCandleBroadcaster() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Println("Candle broadcaster started")

	for range ticker.C {
		ctx := context.Background()
		now := time.Now()
		newMinute := now.Truncate(time.Minute)

		// Check if we've moved to a new minute
		if newMinute.After(h.currentMinute) {
			log.Printf("New minute detected: %v", newMinute)

			// Save all completed candles from previous minute
			oldCandles := h.candleTracker.GetCurrentCandles()
			for _, candle := range oldCandles {
				if err := h.marketRepo.SaveOHLCV(ctx, &candle); err != nil {
					log.Printf("Error saving candle for %s: %v", candle.Symbol, err)
				} else {
					log.Printf("Saved candle for %s: O=%.2f H=%.2f L=%.2f C=%.2f V=%.2f",
						candle.Symbol, candle.Open, candle.High, candle.Low, candle.Close, candle.Volume)
				}

				// Reset candle for new minute (use last close as new open)
				h.candleTracker.ResetCandle(candle.Symbol, newMinute, candle.Close)
			}

			h.currentMinute = newMinute
		}

		// Fetch trades since last check
		trades, err := h.marketRepo.GetLatestTrades(ctx, h.candleTracker.lastTradeCheck)
		if err != nil {
			log.Printf("Error fetching trades: %v", err)
			continue
		}

		// Update candles with new trades
		for _, trade := range trades {
			h.candleTracker.UpdateWithTrade(trade, h.currentMinute)
		}

		h.candleTracker.lastTradeCheck = now

		// Broadcast current candle state
		candles := h.candleTracker.GetCurrentCandles()
		if len(candles) > 0 {
			h.broadcast <- candles
		}
	}
}
