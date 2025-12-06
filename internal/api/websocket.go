package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// configuredOrigins holds explicitly configured origins from config.
// Empty slice means use RFC 1918 defaults. "*" means allow all.
var configuredOrigins []string

// SetAllowedOrigins configures the allowed WebSocket/CORS origins.
// Called during server initialization with config values.
func SetAllowedOrigins(origins []string) {
	configuredOrigins = origins
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Allow requests with no origin (same-origin)
		if origin == "" {
			return true
		}
		// Check against allowed patterns using shared function
		return isAllowedWSOrigin(origin)
	},
}

// isAllowedWSOrigin checks if the WebSocket origin is allowed.
// Priority: 1) Configured origins, 2) RFC 1918 defaults if no config.
func isAllowedWSOrigin(origin string) bool {
	// If explicit origins are configured, use them exclusively
	if len(configuredOrigins) > 0 {
		for _, allowed := range configuredOrigins {
			// "*" allows all origins
			if allowed == "*" {
				return true
			}
			// Exact match
			if origin == allowed {
				return true
			}
			// Prefix match for patterns like "http://192.168."
			if len(origin) >= len(allowed) && origin[:len(allowed)] == allowed {
				return true
			}
		}
		return false
	}

	// Default: Allow localhost and RFC 1918 private networks
	return isRFC1918Origin(origin)
}

// isRFC1918Origin checks if origin is localhost or RFC 1918 private network.
func isRFC1918Origin(origin string) bool {
	allowedPatterns := []string{
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
		"http://[::1]",
		"https://[::1]",
		// Private network ranges (RFC 1918)
		"http://192.168.",
		"https://192.168.",
		"http://10.",
		"https://10.",
		"http://172.16.",
		"https://172.16.",
		"http://172.17.",
		"https://172.17.",
		"http://172.18.",
		"https://172.18.",
		"http://172.19.",
		"https://172.19.",
		"http://172.20.",
		"https://172.20.",
		"http://172.21.",
		"https://172.21.",
		"http://172.22.",
		"https://172.22.",
		"http://172.23.",
		"https://172.23.",
		"http://172.24.",
		"https://172.24.",
		"http://172.25.",
		"https://172.25.",
		"http://172.26.",
		"https://172.26.",
		"http://172.27.",
		"https://172.27.",
		"http://172.28.",
		"https://172.28.",
		"http://172.29.",
		"https://172.29.",
		"http://172.30.",
		"https://172.30.",
		"http://172.31.",
		"https://172.31.",
	}

	for _, pattern := range allowedPatterns {
		if len(origin) >= len(pattern) && origin[:len(pattern)] == pattern {
			// Allow localhost with any port, and private IPs
			remainder := origin[len(pattern):]
			if remainder == "" || (len(remainder) > 0 && (remainder[0] == ':' || (remainder[0] >= '0' && remainder[0] <= '9'))) {
				return true
			}
		}
	}
	return false
}

// Message represents a WebSocket message.
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// CardUpdate represents a card data update.
type CardUpdate struct {
	CardID string      `json:"cardId"`
	Data   interface{} `json:"data"`
}

// Client represents a WebSocket client.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	shutdown   chan struct{}
	mu         sync.RWMutex
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		shutdown:   make(chan struct{}),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			// Collect slow clients under read lock, then remove them under write lock
			var slowClients []*Client
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client is slow - collect for removal
					slowClients = append(slowClients, client)
				}
			}
			h.mu.RUnlock()

			// Remove slow clients under write lock
			if len(slowClients) > 0 {
				h.mu.Lock()
				for _, client := range slowClients {
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						close(client.send)
					}
				}
				h.mu.Unlock()
			}

		case <-h.shutdown:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// Shutdown stops the hub.
func (h *Hub) Shutdown() {
	close(h.shutdown)
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	h.broadcast <- data
}

// BroadcastCardUpdate sends a card update to all clients.
func (h *Hub) BroadcastCardUpdate(cardID string, data interface{}) {
	h.Broadcast(Message{
		Type: "card_update",
		Payload: CardUpdate{
			CardID: cardID,
			Data:   data,
		},
	})
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// handleWebSocket handles WebSocket connections.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  s.wsHub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.wsHub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	// Send initial state
	s.sendInitialState(client)
}

// sendInitialState sends the current state to a newly connected client.
func (s *Server) sendInitialState(client *Client) {
	// Check if current interface is wireless
	isWireless := false
	if s.wifiManager != nil {
		isWireless = s.wifiManager.IsWireless()
	}

	// Build initial state with actual card data
	cards := make(map[string]interface{})

	// Collect current card data
	if linkData := s.collectLinkData(); linkData != nil {
		cards["link"] = linkData
	}
	if gatewayData := s.collectGatewayData(); gatewayData != nil {
		cards["gateway"] = gatewayData
	}
	if dnsData := s.collectDNSData(); dnsData != nil {
		cards["dns"] = dnsData
	}
	if switchData := s.collectDiscoveryData(); switchData != nil {
		cards["switch"] = switchData
	}

	msg := Message{
		Type: "initial_state",
		Payload: map[string]interface{}{
			"status":     "connected",
			"interface":  s.config.Interface.Default,
			"isWireless": isWireless,
			"cards":      cards,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling initial state: %v", err)
		return
	}

	select {
	case client.send <- data:
	default:
		log.Printf("Failed to send initial state to client")
	}
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
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

		// Handle incoming messages (e.g., settings updates, test triggers)
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		log.Printf("Received message: %s", msg.Type)
		// TODO: Handle different message types
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
