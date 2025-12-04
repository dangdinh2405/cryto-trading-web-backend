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

// HasCandle checks if a candle exists for symbol
func (ct *CandleTracker) HasCandle(symbol string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	_, exists := ct.candles[symbol]
	return exists
}

// GetCandle returns candle for symbol (or nil)
func (ct *CandleTracker) GetCandle(symbol string) *models.OHLCV {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.candles[symbol]
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
	clients           map[*Client]bool
	broadcast         chan []models.OHLCV
	register          chan *Client
	unregister        chan *Client
	mu                sync.RWMutex
	marketRepo        *repo.MarketRepo
	cache             interface{} // Cache service for candles
	currentMinute     time.Time
	subscribedSymbols map[string]bool // track all subscribed symbols
	symbolsLock       sync.RWMutex    // protect subscribedSymbols
	lastTradeCheck    time.Time       // Last time we checked for trades
}

func NewHub(marketRepo *repo.MarketRepo, cache interface{}) *Hub {
	now := time.Now()
	return &Hub{
		clients:           make(map[*Client]bool),
		broadcast:         make(chan []models.OHLCV, 256),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		marketRepo:        marketRepo,
		cache:             cache,
		currentMinute:     now.Truncate(time.Minute),
		subscribedSymbols: make(map[string]bool),
		lastTradeCheck:    now,
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

	ctx := context.Background()

	switch msg.Type {
	case "subscribe":
		for _, symbol := range msg.Symbols {
			// Validate symbol BEFORE subscribing
			exists, err := c.hub.marketRepo.ValidateSymbol(ctx, symbol)
			if err != nil {
				log.Printf("Error validating symbol %s: %v", symbol, err)
				continue
			}
			if !exists {
				log.Printf("Rejected subscription to invalid symbol: %s (not in markets table)", symbol)
				continue
			}

			// Symbol is valid, add to subscriptions
			c.symbols[symbol] = true
			
			// Track globally subscribed symbols and initialize candle if new
			c.hub.symbolsLock.Lock()
			if !c.hub.subscribedSymbols[symbol] {
				c.hub.subscribedSymbols[symbol] = true
				go c.hub.InitializeSymbolCandle(ctx, symbol)
			}
			c.hub.symbolsLock.Unlock()
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

// InitializeSymbolCandle initializes a candle for a symbol from database
// Note: Assumes symbol has already been validated before calling this method
func (h *Hub) InitializeSymbolCandle(ctx context.Context, symbol string) error {
	// Check if candle already exists using cache service
	type cacheService interface {
		HasCandle(ctx context.Context, symbol string) (bool, error)
		ResetCandle(ctx context.Context, symbol string, newMinute time.Time, lastClose float64) error
	}
	
	cs, ok := h.cache.(cacheService)
	if !ok || cs == nil {
		return nil // No cache, skip
	}
	
	has, err := cs.HasCandle(ctx, symbol)
	if err == nil && has {
		return nil // Already exists
	}
	
	log.Printf("[InitializeSymbolCandle] Initializing candle for %s", symbol)

	// Try to get latest candle from DB
	candles, err := h.marketRepo.GetCandles(ctx, symbol, "1m", 1, nil)
	if err == nil && len(candles) > 0 {
		lastCandle := candles[0]
		cs.ResetCandle(ctx, symbol, h.currentMinute, lastCandle.Close)
		log.Printf("[InitializeSymbolCandle] Initialized %s with last close price: %.2f", symbol, lastCandle.Close)
		return nil
	}

	// Fallback: try to get last trade price
	since := time.Now().Add(-24 * time.Hour)
	trades, err := h.marketRepo.GetLatestTrades(ctx, since)
	if err == nil {
		for i := len(trades) - 1; i >= 0; i-- {
			if trades[i].Symbol == symbol {
				cs.ResetCandle(ctx, symbol, h.currentMinute, trades[i].Price)
				log.Printf("[InitializeSymbolCandle] Initialized %s with last trade price: %.2f", symbol, trades[i].Price)
				return nil
			}
		}
	}

	// No data available, initialize with 0
	cs.ResetCandle(ctx, symbol, h.currentMinute, 0)
	log.Printf("[InitializeSymbolCandle] Initialized %s with default price: 0", symbol)
	return nil
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

		// Get cache service interface
		type cacheService interface {
			GetAllCandles(ctx context.Context) ([]models.OHLCV, error)
			ResetCandle(ctx context.Context, symbol string, newMinute time.Time, lastClose float64) error
			UpdateCandleWithTrade(ctx context.Context, trade repo.Trade, currentMinute time.Time) error
			HasCandle(ctx context.Context, symbol string) (bool, error)
		}
		
		cs, ok := h.cache.(cacheService)
		if !ok || cs == nil {
			continue // Skip if no cache service
		}

		// Check if we've moved to a new minute
		if newMinute.After(h.currentMinute) {
			log.Printf("New minute detected: %v", newMinute)

			// Save all completed candles from previous minute
			oldCandles, err := cs.GetAllCandles(ctx)
			if err != nil {
				log.Printf("Error getting candles: %v", err)
			} else {
				for _, candle := range oldCandles {
					// Validate symbol before saving
					exists, err := h.marketRepo.ValidateSymbol(ctx, candle.Symbol)
					if err != nil {
						log.Printf("Error validating symbol %s: %v", candle.Symbol, err)
						continue
					}
					if !exists {
						log.Printf("Skipping save for invalid symbol: %s", candle.Symbol)
						continue
					}

					// Save valid candles
					if err := h.marketRepo.SaveOHLCV(ctx, &candle); err != nil {
						log.Printf("Error saving candle for %s: %v", candle.Symbol, err)
					} else {
						log.Printf("Saved candle for %s: O=%.2f H=%.2f L=%.2f C=%.2f V=%.2f",
							candle.Symbol, candle.Open, candle.High, candle.Low, candle.Close, candle.Volume)
					}

					// Reset candle for new minute (use last close as new open)
					cs.ResetCandle(ctx, candle.Symbol, newMinute, candle.Close)
				}
			}

			// Ensure all subscribed symbols have candles for the new minute
			h.symbolsLock.RLock()
			for symbol := range h.subscribedSymbols {
				has, _ := cs.HasCandle(ctx, symbol)
				if !has {
					// Initialize candle for this symbol
					go h.InitializeSymbolCandle(ctx, symbol)
				}
			}
			h.symbolsLock.RUnlock()

			h.currentMinute = newMinute
		}

		// Fetch trades since last check
		trades, err := h.marketRepo.GetLatestTrades(ctx, h.lastTradeCheck)
		if err != nil {
			log.Printf("Error fetching trades: %v", err)
		} else {
			// Update candles with new trades
			for _, trade := range trades {
				cs.UpdateCandleWithTrade(ctx, trade, h.currentMinute)
			}
		}

		h.lastTradeCheck = now

		// Always broadcast current candle state (even if no trades)
		candles, err := cs.GetAllCandles(ctx)
		if err == nil && len(candles) > 0 {
			h.broadcast <- candles
		}
	}
}
