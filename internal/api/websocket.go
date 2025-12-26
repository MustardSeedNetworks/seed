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
//   - Rate limit: 100 messages/second per connection with burst of 20
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
//   - Per-connection rate limiting prevents message flooding attacks
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
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
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

	// messageRateLimit is the maximum number of messages per second per connection.
	// Set to 100 messages/second to prevent resource exhaustion via rapid message flooding.
	messageRateLimit = 100

	// messageRateBurst is the maximum burst size for the rate limiter.
	// Allows brief bursts of up to 20 messages before rate limiting kicks in.
	messageRateBurst = 20
)

// configuredOrigins holds explicitly configured WebSocket/CORS origins from the config file.
//
// Origin validation behavior:
//   - Empty slice (default): Use RFC 1918 private network ranges (10.x, 172.16-31.x, 192.168.x)
//   - Contains "*": Allow all origins (development/testing only - security risk in production)
//   - Contains specific origins: Match exactly against Origin header (case-sensitive)
//
// Origins should include protocol and port: "https://192.168.1.100:8443"
var (
	configuredOrigins   []string
	configuredOriginsMu sync.RWMutex // Protects configuredOrigins (fixes #847)
)

// SetAllowedOrigins configures the allowed WebSocket/CORS origins from application config.
//
// This function is called during server initialization with values from the
// Security.AllowedOrigins configuration field. Origin validation is enforced
// in the WebSocket upgrader CheckOrigin function and CORS middleware.
//
// Parameters:
//   - origins: List of allowed origin strings, or ["*"] to allow all
//
// Thread-safety: Protected by mutex for safe concurrent access (fixes #847).
func SetAllowedOrigins(origins []string) {
	configuredOriginsMu.Lock()
	defer configuredOriginsMu.Unlock()
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
	// Copy origins under lock to avoid race condition (fixes #847)
	configuredOriginsMu.RLock()
	origins := make([]string, len(configuredOrigins))
	copy(origins, configuredOrigins)
	configuredOriginsMu.RUnlock()

	// If explicit origins are configured, use them exclusively
	if len(origins) > 0 {
		for _, allowed := range origins {
			// Skip empty strings to prevent matching all origins (fixes #893)
			if allowed == "" {
				continue
			}
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

// getConfiguredOrigins returns a copy of the configured origins for CORS middleware.
// Thread-safe for concurrent access (fixes #847).
func getConfiguredOrigins() []string {
	configuredOriginsMu.RLock()
	defer configuredOriginsMu.RUnlock()
	origins := make([]string, len(configuredOrigins))
	copy(origins, configuredOrigins)
	return origins
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
// Uses proper URL parsing to prevent subdomain bypass attacks (fixes #710).
// Rejects malicious origins like "http://192.168.1.1.evil.com".
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
//
//nolint:gocyclo,gocritic // Complexity is necessary for proper IP validation security (fixes #710)
func isRFC1918Origin(origin string) bool {
	// Reject null origin (fixes #709)
	if origin == "null" {
		return false
	}

	// Parse the origin URL to extract the hostname (fixes #710)
	// This prevents subdomain bypass attacks like "http://192.168.1.1.evil.com"
	var host string

	// Extract hostname from origin using manual parsing to avoid import overhead
	// Origin format: scheme://host[:port]
	if strings.HasPrefix(origin, "http://") {
		host = origin[7:]
	} else if strings.HasPrefix(origin, "https://") {
		host = origin[8:]
	} else {
		// Invalid scheme - reject
		return false
	}

	// Remove port if present (everything after first colon)
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// Remove path if present (everything after first slash)
	if slashIdx := strings.Index(host, "/"); slashIdx != -1 {
		host = host[:slashIdx]
	}

	// Check for localhost
	if host == "localhost" || host == "127.0.0.1" || host == "[::1]" {
		return true
	}

	// Check for RFC 1918 private networks using exact IP address validation
	// This prevents subdomain attacks like "192.168.1.1.evil.com"

	// Class C: 192.168.0.0/16
	if strings.HasPrefix(host, "192.168.") {
		// Verify it's actually an IP (no dots after the third octet portion)
		remainder := host[8:] // After "192.168."
		// Should be X.Y where X and Y are 0-255
		parts := strings.Split(remainder, ".")
		if len(parts) == 2 {
			// Valid format - ensure no additional domain components
			return isValidIPOctet(parts[0]) && isValidIPOctet(parts[1])
		}
		return false
	}

	// Class A: 10.0.0.0/8
	if strings.HasPrefix(host, "10.") {
		remainder := host[3:] // After "10."
		parts := strings.Split(remainder, ".")
		if len(parts) == 3 {
			return isValidIPOctet(parts[0]) && isValidIPOctet(parts[1]) && isValidIPOctet(parts[2])
		}
		return false
	}

	// Class B: 172.16.0.0/12 (172.16.0.0 - 172.31.255.255)
	if strings.HasPrefix(host, "172.") {
		remainder := host[4:] // After "172."
		parts := strings.Split(remainder, ".")
		if len(parts) == 3 {
			// Check second octet is 16-31
			if !isValidIPOctet(parts[0]) {
				return false
			}
			// Parse second octet to verify range 16-31
			var secondOctet int
			for _, c := range parts[0] {
				if c < '0' || c > '9' {
					return false
				}
				secondOctet = secondOctet*10 + int(c-'0')
				if secondOctet > 255 {
					return false
				}
			}
			if secondOctet >= 16 && secondOctet <= 31 {
				return isValidIPOctet(parts[1]) && isValidIPOctet(parts[2])
			}
		}
		return false
	}

	return false
}

// isValidIPOctet checks if a string is a valid IP octet (0-255).
// Helper function for proper IP validation (fixes #710).
func isValidIPOctet(s string) bool {
	if s == "" || len(s) > 3 {
		return false
	}

	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
		val = val*10 + int(c-'0')
		if val > 255 {
			return false
		}
	}

	return true
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
//
// Multi-interface support (#754): The Interface field identifies which network
// interface the card data pertains to. Clients can filter updates based on their
// currently selected interface.
type CardUpdate struct {
	CardID    string      `json:"cardId"`              // Unique card identifier (e.g., "link", "dns", "gateway")
	Data      interface{} `json:"data"`                // Complete card data (structure varies by CardID)
	Interface string      `json:"interface,omitempty"` // Network interface name (e.g., "eth0", "wlan0")
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
//   - Message rate limit is exceeded (100 msg/sec with burst of 20)
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	limiter   *rate.Limiter // Per-connection rate limiter to prevent message flooding
	closeOnce sync.Once     // Ensures connection is closed only once
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
			slog.Debug("WebSocket client connected", "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			slog.Debug("WebSocket client disconnected", "total_clients", len(h.clients))

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
	// Protect against double-close panic (fixes #833)
	h.mu.Lock()
	defer h.mu.Unlock()
	select {
	case <-h.shutdown:
		// Already closed
		return
	default:
		close(h.shutdown)
	}
}

// Broadcast sends a message to all connected clients.
// Uses non-blocking send with timeout to prevent goroutine hangs if hub stops (fixes #858).
// Fixes #881: Check shutdown first to avoid creating timers during shutdown.
func (h *Hub) Broadcast(msg Message) {
	// Fixes #881: Check shutdown first to avoid timer allocation during shutdown
	select {
	case <-h.shutdown:
		slog.Debug("Broadcast dropped - hub already shut down")
		return
	default:
	}

	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Error marshaling message", "error", err)
		return
	}

	// Non-blocking send with timeout to prevent indefinite blocking if hub exits (fixes #858)
	select {
	case h.broadcast <- data:
		// Message sent successfully
	case <-h.shutdown:
		// Hub is shutting down, drop the message
		slog.Debug("Broadcast dropped - hub shutting down")
	case <-time.After(100 * time.Millisecond):
		slog.Warn("Broadcast timeout - hub may be overloaded or stopped")
	}
}

// BroadcastCardUpdate sends a card update to all clients.
// For interface-specific updates, use BroadcastCardUpdateForInterface instead.
func (h *Hub) BroadcastCardUpdate(cardID string, data interface{}) {
	h.Broadcast(Message{
		Type: "card_update",
		Payload: CardUpdate{
			CardID: cardID,
			Data:   data,
		},
	})
}

// BroadcastCardUpdateForInterface sends a card update scoped to a specific interface.
// This allows clients to filter updates based on their currently selected interface.
// Multi-interface support (#754).
func (h *Hub) BroadcastCardUpdateForInterface(cardID string, data interface{}, iface string) {
	h.Broadcast(Message{
		Type: "card_update",
		Payload: CardUpdate{
			CardID:    cardID,
			Data:      data,
			Interface: iface,
		},
	})
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastLogEntry sends a log entry to all connected clients.
// This is used by the logging package to stream logs in real-time.
func (h *Hub) BroadcastLogEntry(entry interface{}) {
	h.Broadcast(Message{
		Type:    "log_entry",
		Payload: entry,
	})
}

// logBroadcastAdapter wraps the Hub to implement logging.Broadcaster interface.
// This adapter allows the logging package to broadcast logs without importing the api package.
type logBroadcastAdapter struct {
	hub *Hub
}

// BroadcastLogEntry implements logging.Broadcaster interface.
func (a *logBroadcastAdapter) BroadcastLogEntry(entry *logging.LogEntry) {
	if a.hub != nil {
		a.hub.BroadcastLogEntry(entry)
	}
}

// pipelineBroadcastAdapter implements discovery.EventBroadcaster for pipeline events.
type pipelineBroadcastAdapter struct {
	hub *Hub
}

// BroadcastPipelineEvent implements discovery.EventBroadcaster interface.
func (a *pipelineBroadcastAdapter) BroadcastPipelineEvent(event discovery.PipelineEvent) {
	if a.hub != nil {
		a.hub.Broadcast(Message{
			Type:    "pipeline",
			Payload: event,
		})
	}
}

// handleWebSocket handles WebSocket connections.
// Security fix #660: Uses httpOnly cookie authentication.
// Cookies are automatically sent by the browser and are httpOnly (not accessible to JS)
// to prevent token exposure in logs, browser dev tools, and proxy servers.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract token from httpOnly cookie (fixes #660 - secure method)
	token, source := auth.GetTokenFromRequest(r)

	// Validate token
	if token == "" {
		logger := logging.FromContext(r.Context())
		localizer := i18n.FromRequest(r)
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized, localizer.T("errors.auth.noToken"), "") // fixes #694
		return
	}

	claims, err := s.authManager.ValidateToken(token)
	if err != nil {
		slog.Warn("WebSocket auth failed", "error", err, "source", source)
		logger := logging.FromContext(r.Context())
		localizer := i18n.FromRequest(r)
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized, localizer.T("errors.auth.invalidToken"), "")
		return
	}

	slog.Debug("WebSocket authenticated", "username", claims.Username, "source", source)

	// No response header needed for cookie auth
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return
	}

	client := &Client{
		hub:     s.wsHub,
		conn:    conn,
		send:    make(chan []byte, 256),
		limiter: rate.NewLimiter(messageRateLimit, messageRateBurst),
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
		slog.Error("Error marshaling initial state", "error", err)
		return
	}

	// Safely attempt to send; channel may already be closed if the client dropped.
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Debug("Skipped initial state send (client gone)", "recover", r)
			}
		}()
		select {
		case client.send <- data:
		default:
			slog.Warn("Failed to send initial state to client")
		}
	}()
}

// close safely closes the client connection exactly once.
func (c *Client) close() {
	c.closeOnce.Do(func() {
		// Close connection first to force writePump to exit via conn.Close() (fixes #835)
		// This ensures the writePump goroutine doesn't block forever on c.send
		c.conn.Close()

		// Then try to unregister - if this times out, the client is already
		// disconnected so it won't receive messages anyway (fixes #686)
		select {
		case c.hub.unregister <- c:
			// Successfully sent unregister request
		case <-time.After(100 * time.Millisecond):
			// Channel full or hub not responding, client already disconnected
			slog.Debug("Client unregister timeout, connection already closed")
		}
	})
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump() {
	defer c.close()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Error("Failed to set initial read deadline", "error", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			slog.Error("Failed to extend read deadline", "error", err)
		}
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket error", "error", err)
			}
			break
		}

		// Check rate limit before processing message
		if !c.limiter.Allow() {
			slog.Warn("WebSocket rate limit exceeded, closing connection")
			// Send close message with policy violation code (1008)
			closeMsg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "rate limit exceeded")
			if err := c.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(writeWait)); err != nil {
				slog.Error("Failed to send rate limit close message", "error", err)
			}
			break
		}

		// Handle incoming messages (e.g., settings updates, test triggers)
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Warn("Error parsing message", "error", err)
			continue
		}

		slog.Debug("Received message", "type", msg.Type)

		// Handle different message types (issue #608 resolved)
		switch msg.Type {
		case "ping":
			// Respond with pong for client-side keep-alive
			response := Message{Type: "pong", Payload: time.Now().UnixMilli()}
			if data, err := json.Marshal(response); err == nil {
				select {
				case c.send <- data:
				default:
					slog.Warn("Client send buffer full, dropping pong")
				}
			}

		case "requestCardUpdate":
			// Client requesting a specific card update
			if cardID, ok := msg.Payload.(string); ok {
				slog.Debug("Card update requested", "card_id", cardID)
				// The server will send the next scheduled update for this card
			}

		default:
			slog.Warn("Unknown message type", "type", msg.Type)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
// Fixes #869: Ensure writer is closed on all error paths to prevent resource leaks.
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
				slog.Error("Failed to set write deadline", "error", err)
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					slog.Error("Failed to send close message", "error", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Fixes #869: Use a closure to ensure writer is closed on all paths
			writeErr := func() error {
				defer w.Close() // Always close writer (fixes #869)

				if _, err := w.Write(message); err != nil {
					return err
				}

				// Add queued messages to the current WebSocket message
				n := len(c.send)
				for range n {
					if _, err := w.Write([]byte{'\n'}); err != nil {
						return err
					}
					if _, err := w.Write(<-c.send); err != nil {
						return err
					}
				}
				return nil
			}()

			if writeErr != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Error("Failed to set ping write deadline", "error", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
