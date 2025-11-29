package handler

import (
	"context"
	"encoding/json"
	"log"
	// "net/http"
	"sync"
	"time"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// OrderbookMessage for client subscribe/unsubscribe
type OrderbookMessage struct {
	Type      string   `json:"type"`       // "subscribe", "unsubscribe"
	MarketIDs []string `json:"market_ids"` // List of market UUIDs
}

// OrderbookClient represents a WebSocket client connection
type OrderbookClient struct {
	hub        *OrderbookHub
	conn       *websocket.Conn
	send       chan []byte
	marketIDs  map[string]bool // Subscribed markets
	marketsLock sync.RWMutex
}

// OrderbookHub manages all orderbook WebSocket connections
type OrderbookHub struct {
	clients    map[*OrderbookClient]bool
	broadcast  chan map[string]*repo.OrderBook // market_id -> orderbook
	register   chan *OrderbookClient
	unregister chan *OrderbookClient
	mu         sync.RWMutex
	orderRepo  *repo.OrderRepo
}

// NewOrderbookHub creates a new orderbook hub
func NewOrderbookHub(orderRepo *repo.OrderRepo) *OrderbookHub {
	return &OrderbookHub{
		clients:    make(map[*OrderbookClient]bool),
		broadcast:  make(chan map[string]*repo.OrderBook, 256),
		register:   make(chan *OrderbookClient),
		unregister: make(chan *OrderbookClient),
		orderRepo:  orderRepo,
	}
}

// Run starts the hub's main loop
func (h *OrderbookHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Orderbook client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Orderbook client disconnected. Total clients: %d", len(h.clients))

		case orderbooks := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Filter orderbooks based on client's subscriptions
				filteredBooks := client.filterOrderbooks(orderbooks)

				if len(filteredBooks) > 0 {
					data, err := json.Marshal(filteredBooks)
					if err != nil {
						log.Printf("Error marshaling orderbooks: %v", err)
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

// filterOrderbooks returns only orderbooks for subscribed markets
func (c *OrderbookClient) filterOrderbooks(orderbooks map[string]*repo.OrderBook) map[string]*repo.OrderBook {
	c.marketsLock.RLock()
	defer c.marketsLock.RUnlock()

	// If no markets subscribed, return empty (don't send anything)
	if len(c.marketIDs) == 0 {
		return nil
	}

	filtered := make(map[string]*repo.OrderBook)
	for marketID, orderbook := range orderbooks {
		if c.marketIDs[marketID] {
			filtered[marketID] = orderbook
		}
	}
	return filtered
}

// writePump sends messages to the WebSocket connection
func (c *OrderbookClient) writePump() {
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

// readPump reads messages from the WebSocket connection
func (c *OrderbookClient) readPump() {
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
		var msg OrderbookMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		c.handleMessage(msg)
	}
}

// handleMessage processes subscribe/unsubscribe requests
func (c *OrderbookClient) handleMessage(msg OrderbookMessage) {
	c.marketsLock.Lock()
	
	switch msg.Type {
	case "subscribe":
		for _, marketID := range msg.MarketIDs {
			c.marketIDs[marketID] = true
		}
		log.Printf("Client subscribed to markets: %v (total: %d)", msg.MarketIDs, len(c.marketIDs))
		
		// Immediately fetch and send orderbook for subscribed markets
		c.marketsLock.Unlock() // Unlock before async operations
		go c.sendImmediateOrderbook(msg.MarketIDs)
		return

	case "unsubscribe":
		for _, marketID := range msg.MarketIDs {
			delete(c.marketIDs, marketID)
		}
		log.Printf("Client unsubscribed from markets: %v (remaining: %d)", msg.MarketIDs, len(c.marketIDs))

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
	
	c.marketsLock.Unlock()
}

// sendImmediateOrderbook fetches and sends orderbook immediately for specified markets
func (c *OrderbookClient) sendImmediateOrderbook(marketIDs []string) {
	ctx := context.Background()
	orderbooks := make(map[string]*repo.OrderBook)
	
	for _, marketID := range marketIDs {
		orderbook, err := c.hub.orderRepo.GetOrderBook(ctx, marketID, 20)
		if err != nil {
			log.Printf("Error fetching immediate orderbook for market %s: %v", marketID, err)
			continue
		}
		orderbooks[marketID] = orderbook
		log.Printf("Fetched orderbook for market %s: %d bids, %d asks", marketID, len(orderbook.Bids), len(orderbook.Asks))
	}
	
	if len(orderbooks) > 0 {
		data, err := json.Marshal(orderbooks)
		if err != nil {
			log.Printf("Error marshaling immediate orderbook: %v", err)
			return
		}
		
		select {
		case c.send <- data:
			log.Printf("Sent immediate orderbook to client for %d markets", len(orderbooks))
		default:
			log.Printf("Client send channel full, skipping immediate orderbook")
		}
	} else {
		log.Printf("No orderbook data available for markets: %v", marketIDs)
	}
}

// HandleWebSocket handles new WebSocket connections
func (h *OrderbookHub) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &OrderbookClient{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 256),
		marketIDs: make(map[string]bool),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// StartOrderbookBroadcaster periodically fetches and broadcasts orderbooks
func (h *OrderbookHub) StartOrderbookBroadcaster(marketRepo *repo.MarketRepo) {
	ticker := time.NewTicker(1 * time.Second) // Update every second
	defer ticker.Stop()

	log.Println("Orderbook broadcaster started")

	for range ticker.C {
		ctx := context.Background()

		// Get all active markets
		markets, err := marketRepo.GetAllActiveMarkets(ctx)
		if err != nil {
			log.Printf("Error fetching markets: %v", err)
			continue
		}

		// Fetch orderbook for each market
		orderbooks := make(map[string]*repo.OrderBook)
		for _, market := range markets {
			orderbook, err := h.orderRepo.GetOrderBook(ctx, market.ID, 20) // Top 20 levels
			if err != nil {
				log.Printf("Error fetching orderbook for market %s: %v", market.Symbol, err)
				continue
			}
			orderbooks[market.ID] = orderbook
		}

		// Broadcast to clients
		if len(orderbooks) > 0 {
			h.broadcast <- orderbooks
		}
	}
}
