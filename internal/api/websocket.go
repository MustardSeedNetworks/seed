// Package api provides WebSocket functionality for real-time network monitoring updates.
//
// This file implements a WebSocket hub using the gorilla/websocket library to broadcast
// real-time updates to connected clients. The hub manages multiple concurrent client
// connections and handles message routing, connection lifecycle, and ping/pong heartbeats.
//
// WebSocket Protocol:
//   - Endpoint: /ws
//   - Protocol: RFC 6455 WebSocket over HTTP/HTTPS
//   - Message format: JSON-encoded Message structs
//   - Heartbeat: Ping every 54 seconds, expect pong within 60 seconds
//   - Message size limit: 512 bytes for client->server messages
//   - Write timeout: 10 seconds per message
//
// Message Types (client receives):
//   - "linkState": Network link up/down events
//   - "dhcpLease": DHCP lease changes
//   - "rogueDetected": Rogue DHCP server alerts
//   - "speedtestProgress": Real-time speedtest updates
//   - "iperfProgress": iPerf throughput test updates
//   - "scanProgress": Network device scan progress
//   - "discovery": New neighbor discovered via LLDP/CDP
//
// Security:
//   - CORS/Origin validation against configured allowed origins
//   - Default: RFC 1918 private network addresses only
//   - Configurable via Security.AllowedOrigins in config
//   - Wildcard "*" allows all origins (use only in development)
//
// Clients must:
//   - Respond to ping frames with pong within 60 seconds
//   - Handle JSON-decoded messages with "type" and "payload" fields
//   - Reconnect on connection loss (server does not persist messages)
//
// Implementation details:
//   - Hub runs in a dedicated goroutine managing all client connections
//   - Each client has separate read/write goroutines with independent timeouts
//   - Broadcast messages are sent to all connected clients concurrently
//   - Slow clients are automatically disconnected if write buffer fills
//   - No message persistence - clients must handle reconnection gaps
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait is the maximum time allowed to write a message to the peer.
	// If a write takes longer, the connection is considered dead and closed.
	// Set to 10 seconds to accommodate slow networks while preventing indefinite hangs.
	writeWait = 10 * time.Second

	// pongWait is the maximum time allowed to read the next pong message from the peer.
	// If no pong is received within this duration, the connection is considered dead.
	// Set to 60 seconds to allow for network latency and busy clients.
	pongWait = 60 * time.Second

	// pingPeriod specifies how often to send ping messages to the peer.
	// Must be less than pongWait to ensure pings are sent before timeout.
	// Calculated as 90% of pongWait (54 seconds) to provide safety margin.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum size in bytes allowed for messages from peer.
	// Client-to-server messages are currently unused, so this is kept small.
	// Server-to-client broadcasts have no size limit (but should remain reasonable).
	maxMessageSize = 512
)

// configuredOrigins holds explicitly configured WebSocket/CORS origins from the config file.
//
// Origin validation behavior:
//   - Empty slice (default): Use RFC 1918 private network ranges (10.x, 172.16-31.x, 192.168.x)
//   - Contains "*": Allow all origins (development/testing only - security risk in production)
//   - Contains specific origins: Match exactly against Origin header (case-sensitive)
//
// Origins should include protocol and port: "https://192.168.1.100:8443"
var configuredOrigins []string

// SetAllowedOrigins configures the allowed WebSocket/CORS origins from application config.
//
// This function is called during server initialization with values from the
// Security.AllowedOrigins configuration field. Origin validation is enforced
// in the WebSocket upgrader CheckOrigin function and CORS middleware.
//
// Parameters:
//   - origins: List of allowed origin strings, or ["*"] to allow all
//
// Thread-safety: Should only be called during server initialization before
// accepting WebSocket connections. Not protected by mutex.
func SetAllowedOrigins(origins []string) {
	configuredOrigins = origins
}

// upgrader configures the WebSocket protocol upgrade from HTTP.
//
// The upgrader handles the HTTP handshake to upgrade a connection to WebSocket protocol.
// It validates the Origin header to prevent Cross-Site WebSocket Hijacking (CSWSH) attacks.
//
// Buffer sizes (1024 bytes each) are appropriate for JSON messages containing network
// status updates. Adjust if implementing file transfer or large data payloads.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // Buffer for incoming messages (client->server, currently unused)
	WriteBufferSize: 1024, // Buffer for outgoing broadcasts (server->client, main data flow)

	// CheckOrigin validates the WebSocket upgrade request's Origin header.
	// This prevents malicious web pages from connecting to our WebSocket endpoint.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		// Allow requests with no Origin header (same-origin requests, native apps, tools)
		if origin == "" {
			return true
		}

		// Validate origin against configured allowed list
		return isAllowedWSOrigin(origin)
	},
}

// isAllowedWSOrigin checks if the WebSocket origin is allowed to connect.
//
// Origin validation follows this priority order:
//  1. If configuredOrigins is set, use it exclusively (no fallback to defaults)
//  2. If configuredOrigins is empty, allow RFC 1918 private networks only
//
// Configured origins matching:
//   - Wildcard "*": Allow all origins (insecure, development only)
//   - Exact match: "https://192.168.1.100:8443" matches exactly
//   - Prefix match: "http://192.168." matches "http://192.168.1.x", "http://192.168.10.x", etc.
//
// RFC 1918 default matching (when no config):
//   - 10.0.0.0/8: http://10.x.x.x
//   - 172.16.0.0/12: http://172.16-31.x.x
//   - 192.168.0.0/16: http://192.168.x.x
//   - localhost: http://localhost, http://127.0.0.1, http://[::1]
//
// Security note: Origin validation is crucial to prevent Cross-Site WebSocket
// Hijacking (CSWSH). Always configure explicit origins in production environments.
//
// Parameters:
//   - origin: The Origin header value from the WebSocket upgrade request
//
// Returns:
//   - true if the origin is allowed to establish a WebSocket connection
//   - false if the origin should be rejected (connection will fail with 403)
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

// isRFC1918Origin checks if the origin is localhost or an RFC 1918 private network address.
//
// This function implements the default origin validation when no explicit origins are configured.
// It allows connections only from:
//   - Localhost: http(s)://localhost, http(s)://127.0.0.1, http(s)://[::1]
//   - Class A private: http(s)://10.0.0.0/8 (10.x.x.x)
//   - Class B private: http(s)://172.16.0.0/12 (172.16.x.x through 172.31.x.x)
//   - Class C private: http(s)://192.168.0.0/16 (192.168.x.x)
//
// All ports are accepted for these origins - the check is prefix-based to allow
// various port numbers commonly used in development (3000, 8080, 8443, etc.).
//
// Security rationale: Private networks (RFC 1918) are assumed to be trusted local networks.
// This provides reasonable security for home/office LANs while not requiring explicit
// configuration. For public-facing deployments or zero-trust networks, explicitly
// configure allowed origins.
//
// Parameters:
//   - origin: The Origin header value from the WebSocket upgrade request
//
// Returns:
//   - true if the origin is localhost or a private network address
//   - false for public IP addresses or non-matching origins
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
			if remainder == "" || (remainder != "" && (remainder[0] == ':' || (remainder[0] >= '0' && remainder[0] <= '9'))) {
				return true
			}
		}
	}
	return false
}

// Message represents a WebSocket message sent from server to client.
//
// All messages follow this JSON structure:
//
//	{
//	  "type": "messageType",
//	  "payload": { ... type-specific data ... }
//	}
//
// Standard message types:
//   - "linkState": Network interface up/down events
//     Payload: {interface: string, state: "up"|"down", timestamp: string}
//   - "dhcpLease": DHCP lease acquisition/renewal events
//     Payload: {ip: string, mac: string, hostname: string, leaseTime: number}
//   - "rogueDetected": Rogue DHCP server alert
//     Payload: {serverIP: string, serverMAC: string, interface: string}
//   - "speedtestProgress": Real-time speedtest updates
//     Payload: {phase: string, bytesTransferred: number, elapsedMs: number}
//   - "iperfProgress": iPerf3 throughput test updates
//     Payload: {direction: "upload"|"download", mbps: number, progress: number}
//   - "scanProgress": Network device scan progress
//     Payload: {scanned: number, total: number, currentIP: string}
//   - "discovery": New neighbor discovered via LLDP/CDP
//     Payload: DiscoveryNeighborInfo object
//
// Clients must handle unknown message types gracefully (ignore or log).
type Message struct {
	Type    string      `json:"type"`    // Message type identifier
	Payload interface{} `json:"payload"` // Type-specific data (varies by Type)
}

// CardUpdate represents a dashboard card data update for periodic refreshes.
//
// Card updates are sent via "cardUpdate" messages to refresh specific dashboard
// cards without requiring full page reloads. The Data field contains the complete
// updated state for the identified card.
//
// This is used by the broadcast loop to push periodic updates for:
//   - Link status (speed, duplex, carrier)
//   - DNS test results (latency, failures)
//   - Gateway ping (latency, packet loss)
//   - DHCP monitor status
//   - WiFi signal strength
//   - Interface statistics
type CardUpdate struct {
	CardID string      `json:"cardId"` // Unique card identifier (e.g., "link", "dns", "gateway")
	Data   interface{} `json:"data"`   // Complete card data (structure varies by CardID)
}

// Client represents a single WebSocket client connection.
//
// Each client runs two goroutines:
//   - writePump: Sends messages from the send channel to the WebSocket connection
//   - readPump: Reads messages from the WebSocket connection (currently only for ping/pong)
//
// The client is automatically removed from the hub and cleaned up when:
//   - The send channel is closed (server-initiated disconnect)
//   - A write error occurs (network failure, slow client)
//   - A read error occurs (client disconnected, ping/pong timeout)
//   - No pong is received within pongWait (60 seconds)
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once // Ensures connection is closed only once
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
	// Extract token from Sec-WebSocket-Protocol header (modern, secure method)
	// Frontend sends: new WebSocket(url, ["access_token", token])
	// This comes through as: Sec-WebSocket-Protocol: access_token, <token>
	var token string
	protocols := r.Header.Get("Sec-WebSocket-Protocol")
	if protocols != "" {
		parts := strings.Split(protocols, ",")
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "access_token" {
			token = strings.TrimSpace(parts[1])
		}
	}

	// Fallback: check query parameter (deprecated but kept for compatibility)
	if token == "" {
		token = r.URL.Query().Get("token")
		if token != "" {
			log.Println("Warning: WebSocket auth via query param is deprecated, use Sec-WebSocket-Protocol")
		}
	}

	// Validate token
	if token == "" {
		http.Error(w, "Unauthorized: no token provided", http.StatusUnauthorized)
		return
	}

	if _, err := s.authManager.ValidateToken(token); err != nil {
		log.Printf("WebSocket auth failed: %v", err)
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// Set response header to accept the subprotocol
	responseHeader := http.Header{}
	if protocols != "" && strings.Contains(protocols, "access_token") {
		responseHeader.Set("Sec-WebSocket-Protocol", "access_token")
	}

	conn, err := upgrader.Upgrade(w, r, responseHeader)
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

	// Safely attempt to send; channel may already be closed if the client dropped.
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Skipped initial state send (client gone): %v", r)
			}
		}()
		select {
		case client.send <- data:
		default:
			log.Printf("Failed to send initial state to client")
		}
	}()
}

// close safely closes the client connection exactly once.
func (c *Client) close() {
	c.closeOnce.Do(func() {
		c.hub.unregister <- c
		c.conn.Close()
	})
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump() {
	defer c.close()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("failed to set initial read deadline: %v", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("failed to extend read deadline: %v", err)
		}
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
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("failed to set write deadline: %v", err)
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("failed to send close message: %v", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for range n {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					return
				}
				if _, err := w.Write(<-c.send); err != nil {
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("failed to set ping write deadline: %v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
