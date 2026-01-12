package httpapi

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

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

const (
	// websocketBufferSize is the read/write buffer size for WebSocket connections.
	websocketBufferSize = 1024

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
	pingPeriod = (pongWait * pingPeriodRatio) / pingPeriodDivisor

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

	// pingPeriodRatio is the numerator for calculating ping period (90% of pongWait).
	pingPeriodRatio = 9

	// pingPeriodDivisor is the denominator for calculating ping period (to get 90%).
	pingPeriodDivisor = 10

	// ipPartsClassC is the expected number of IP parts for Class C address validation.
	ipPartsClassC = 2

	// ipPartsClassAB is the expected number of IP parts for Class A/B address validation.
	ipPartsClassAB = 3

	// hexBase is the base for hexadecimal number parsing.
	hexBase = 16

	// decimalParseBase is the base for decimal digit parsing.
	decimalParseBase = 10

	// maxIPOctetValue is the maximum valid value for an IP address octet (255).
	maxIPOctetValue = 255

	// wsBroadcastBufferSize is the buffer size for the WebSocket broadcast channel.
	wsBroadcastBufferSize = 256

	// wsBroadcastTimeoutMs is the timeout in milliseconds for broadcast operations.
	wsBroadcastTimeoutMs = 100

	// multicastOctetMax is the maximum second octet value for multicast addresses (172.16-31.x.x range).
	multicastOctetMax = 31
)

// wsConfig holds WebSocket configuration state to satisfy gochecknoglobals.
// Access via getWSState().getConfiguredOrigins(), getWSState().getUpgrader(), getWSState().setAllowedOrigins().
type wsConfig struct {
	originMu     sync.RWMutex
	origins      []string
	upgraderOnce sync.Once
	upgraderInst *websocket.Upgrader
}

// WebSocket state accessor functions use closure-encapsulated state for thread-safe singleton access.
// getWSState returns the global WebSocket configuration instance.
// setWSState sets the global WebSocket configuration instance.
// _ (clearWSState) resets the global WebSocket configuration to nil (unused but required for pattern).
//
//nolint:gochecknoglobals // Intentional thread-safe singleton using closure pattern
var (
	getWSState, _, _ = func() (
		func() *wsConfig,
		func(*wsConfig),
		func(),
	) {
		var (
			mu    sync.RWMutex
			state *wsConfig
		)
		// Initialize with default state
		state = &wsConfig{}

		return func() *wsConfig {
				mu.RLock()
				defer mu.RUnlock()
				return state
			}, func(s *wsConfig) {
				mu.Lock()
				defer mu.Unlock()
				state = s
			}, func() {
				mu.Lock()
				defer mu.Unlock()
				state = nil
			}
	}()
)

// getConfiguredOrigins returns the configured origins (thread-safe).
func (ws *wsConfig) getConfiguredOrigins() []string {
	ws.originMu.RLock()
	defer ws.originMu.RUnlock()
	return ws.origins
}

// getUpgrader returns the lazily-initialized WebSocket upgrader.
func (ws *wsConfig) getUpgrader() *websocket.Upgrader {
	ws.upgraderOnce.Do(func() {
		ws.upgraderInst = &websocket.Upgrader{
			ReadBufferSize:  websocketBufferSize,
			WriteBufferSize: websocketBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				// Call isAllowedWSOriginWithGetter to avoid init cycle
				return isAllowedWSOriginWithGetter(origin, ws.getConfiguredOrigins)
			},
		}
	})
	return ws.upgraderInst
}

// setAllowedOrigins sets the allowed origins (thread-safe).
func (ws *wsConfig) setAllowedOrigins(origins []string) {
	ws.originMu.Lock()
	defer ws.originMu.Unlock()
	ws.origins = origins
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
	return isAllowedWSOriginWithGetter(origin, getWSState().getConfiguredOrigins)
}

// isAllowedWSOriginWithGetter is the internal implementation that accepts a getter function.
// This allows it to be called during initialization without causing an init cycle.
func isAllowedWSOriginWithGetter(origin string, getOrigins func() []string) bool {
	// Get configured origins via provided getter
	origins := getOrigins()

	// Default: Allow localhost and RFC 1918 private networks if no explicit origins configured
	if len(origins) == 0 {
		return isRFC1918Origin(origin)
	}

	// If explicit origins are configured, use them exclusively
	for _, allowed := range origins {
		if matchesAllowedOrigin(origin, allowed) {
			return true
		}
	}
	return false
}

// matchesAllowedOrigin checks if an origin matches a single allowed origin pattern.
// Returns true if the origin matches the allowed pattern.
func matchesAllowedOrigin(origin, allowed string) bool {
	// Skip empty strings to prevent matching all origins (fixes #893)
	if allowed == "" {
		return false
	}
	// "*" allows all origins
	if allowed == "*" {
		return true
	}
	// Exact match
	if origin == allowed {
		return true
	}
	// Check prefix match
	return matchesOriginPrefix(origin, allowed)
}

// matchesOriginPrefix checks if an origin matches an allowed prefix pattern.
// Prefix match for patterns like "http://192.168."
// Fixes #929: Ensure prefix match ends at a valid boundary (port, path, or exact).
func matchesOriginPrefix(origin, allowed string) bool {
	// Origin must be at least as long as allowed prefix
	if len(origin) < len(allowed) {
		return false
	}
	// Prefix must match
	if origin[:len(allowed)] != allowed {
		return false
	}

	// Check remainder to prevent partial domain match
	// e.g., allowed "http://192.168." should not match "http://192.168.evil.com"
	remainder := origin[len(allowed):]

	// Empty remainder or valid boundary character
	if remainder == "" || remainder[0] == ':' || remainder[0] == '/' {
		return true
	}

	// Fixes #940: For IP prefixes ending in '.', validate octet format
	return matchesIPPrefixOctet(allowed, remainder)
}

// matchesIPPrefixOctet validates that a remainder after an IP prefix is a valid octet.
// Ensures remainder starts with a valid octet (1-3 digits) followed by valid boundary.
// Only applies when the allowed pattern ends with '.'.
func matchesIPPrefixOctet(allowed, remainder string) bool {
	// Only check if allowed ends with '.' and remainder starts with a digit
	if allowed[len(allowed)-1] != '.' {
		return false
	}
	if len(remainder) == 0 || remainder[0] < '0' || remainder[0] > '9' {
		return false
	}

	// Find end of potential octet (first non-digit, max 3 digits)
	octetEnd := findOctetEnd(remainder)

	// Valid octet is 1-3 digits; next char must be valid boundary (. : / or end)
	if octetEnd == 0 || octetEnd > 3 {
		return false
	}
	return octetEnd == len(remainder) || isValidOctetBoundary(remainder[octetEnd])
}

// findOctetEnd finds the end of a numeric octet in a string.
// Returns the index of the first non-digit character (max 3 digits).
func findOctetEnd(s string) int {
	octetEnd := 0
	for ; octetEnd < len(s) && octetEnd < 3; octetEnd++ {
		if s[octetEnd] < '0' || s[octetEnd] > '9' {
			break
		}
	}
	return octetEnd
}

// isValidOctetBoundary checks if a character is a valid boundary after an IP octet.
// Valid boundaries are '.', ':', and '/'.
func isValidOctetBoundary(c byte) bool {
	return c == '.' || c == ':' || c == '/'
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
func isRFC1918Origin(origin string) bool {
	// Reject null origin (fixes #709)
	if origin == "null" {
		return false
	}

	// Extract and validate host from origin URL
	host, ok := extractHostFromOrigin(origin)
	if !ok {
		return false
	}

	// Check for localhost addresses
	if isLocalhostAddress(host) {
		return true
	}

	// Check for RFC 1918 private network ranges
	return isPrivateNetworkAddress(host)
}

// extractHostFromOrigin parses an origin URL and extracts the hostname.
// Returns the hostname and true if successful, empty string and false otherwise.
// Origin format: scheme://host[:port][/path]
func extractHostFromOrigin(origin string) (string, bool) {
	var host string

	// Extract hostname from origin using manual parsing to avoid import overhead
	switch {
	case strings.HasPrefix(origin, "http://"):
		host = origin[7:]
	case strings.HasPrefix(origin, "https://"):
		host = origin[8:]
	default:
		// Invalid scheme - reject
		return "", false
	}

	// Remove port if present (everything after first colon)
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// Remove path if present (everything after first slash)
	if slashIdx := strings.Index(host, "/"); slashIdx != -1 {
		host = host[:slashIdx]
	}

	return host, true
}

// isLocalhostAddress checks if the host is a localhost address.
// Supports "localhost", "127.0.0.1", and "[::1]" (IPv6 loopback).
func isLocalhostAddress(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "[::1]"
}

// isPrivateNetworkAddress checks if the host is an RFC 1918 private network address.
// This prevents subdomain attacks like "192.168.1.1.evil.com" by validating
// the complete IP address structure.
func isPrivateNetworkAddress(host string) bool {
	// Class C: 192.168.0.0/16
	if strings.HasPrefix(host, "192.168.") {
		return isValidClassCAddress(host)
	}

	// Class A: 10.0.0.0/8
	if strings.HasPrefix(host, "10.") {
		return isValidClassAAddress(host)
	}

	// Class B: 172.16.0.0/12 (172.16.0.0 - 172.31.255.255)
	if strings.HasPrefix(host, "172.") {
		return isValidClassBAddress(host)
	}

	return false
}

// isValidClassCAddress validates a 192.168.x.x address.
// Returns true if the host is a valid Class C private address.
func isValidClassCAddress(host string) bool {
	remainder := host[8:] // After "192.168."
	// Should be X.Y where X and Y are 0-255
	parts := strings.Split(remainder, ".")
	if len(parts) != ipPartsClassC {
		return false
	}
	return isValidIPOctet(parts[0]) && isValidIPOctet(parts[1])
}

// isValidClassAAddress validates a 10.x.x.x address.
// Returns true if the host is a valid Class A private address.
func isValidClassAAddress(host string) bool {
	remainder := host[3:] // After "10."
	parts := strings.Split(remainder, ".")
	if len(parts) != ipPartsClassAB {
		return false
	}
	return isValidIPOctet(parts[0]) && isValidIPOctet(parts[1]) && isValidIPOctet(parts[2])
}

// isValidClassBAddress validates a 172.16-31.x.x address.
// Returns true if the host is a valid Class B private address (172.16.0.0/12).
func isValidClassBAddress(host string) bool {
	remainder := host[4:] // After "172."
	parts := strings.Split(remainder, ".")
	if len(parts) != ipPartsClassAB {
		return false
	}

	// Validate and parse second octet to verify range 16-31
	secondOctet, ok := parseOctetInRange(parts[0], hexBase, multicastOctetMax)
	if !ok || secondOctet < 16 || secondOctet > 31 {
		return false
	}

	return isValidIPOctet(parts[1]) && isValidIPOctet(parts[2])
}

// parseOctetInRange parses an octet string and checks if it's within the given range.
// Returns the parsed value and true if valid, 0 and false otherwise.
func parseOctetInRange(s string, minVal, maxVal int) (int, bool) {
	if s == "" || len(s) > 3 {
		return 0, false
	}

	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		val = val*decimalParseBase + int(c-'0')
		if val > maxIPOctetValue {
			return 0, false
		}
	}

	if val < minVal || val > maxVal {
		return val, false
	}

	return val, true
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
		val = val*decimalParseBase + int(c-'0')
		if val > maxIPOctetValue {
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
	Type    string `json:"type"`    // Message type identifier
	Payload any    `json:"payload"` // Type-specific data (varies by Type)
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
	CardID    string `json:"cardId"`              // Unique card identifier (e.g., "link", "dns", "gateway")
	Data      any    `json:"data"`                // Complete card data (structure varies by CardID)
	Interface string `json:"interface,omitempty"` // Network interface name (e.g., "eth0", "wlan0")
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
		broadcast:  make(chan []byte, wsBroadcastBufferSize),
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
			h.handleRegister(client)
		case client := <-h.unregister:
			h.handleUnregister(client)
		case message := <-h.broadcast:
			h.handleBroadcast(message)
		case <-h.shutdown:
			h.handleShutdown()
			return
		}
	}
}

// handleRegister adds a new client to the hub.
func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
	logging.GetLogger().Debug("WebSocket client connected", "total_clients", len(h.clients))
}

// handleUnregister removes a client from the hub and closes its send channel.
func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	h.mu.Unlock()
	logging.GetLogger().Debug("WebSocket client disconnected", "total_clients", len(h.clients))
}

// handleBroadcast sends a message to all connected clients and removes slow ones.
func (h *Hub) handleBroadcast(message []byte) {
	slowClients := h.sendToClients(message)
	h.removeSlowClients(slowClients)
}

// sendToClients attempts to send a message to all clients and returns any slow clients.
func (h *Hub) sendToClients(message []byte) []*Client {
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
	return slowClients
}

// removeSlowClients removes clients that couldn't receive messages in time.
func (h *Hub) removeSlowClients(slowClients []*Client) {
	if len(slowClients) == 0 {
		return
	}
	h.mu.Lock()
	for _, client := range slowClients {
		if _, ok := h.clients[client]; ok {
			delete(h.clients, client)
			close(client.send)
		}
	}
	h.mu.Unlock()
}

// handleShutdown closes all client connections and cleans up the hub.
func (h *Hub) handleShutdown() {
	h.mu.Lock()
	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
	h.mu.Unlock()
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
		logging.GetLogger().Debug("Broadcast dropped - hub already shut down")
		return
	default:
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logging.GetLogger().Error("Error marshaling message", "error", err)
		return
	}

	// Non-blocking send with timeout to prevent indefinite blocking if hub exits (fixes #858)
	select {
	case h.broadcast <- data:
		// Message sent successfully
	case <-h.shutdown:
		// Hub is shutting down, drop the message
		logging.GetLogger().Debug("Broadcast dropped - hub shutting down")
	case <-time.After(wsBroadcastTimeoutMs * time.Millisecond):
		logging.GetLogger().Warn("Broadcast timeout - hub may be overloaded or stopped")
	}
}

// BroadcastCardUpdate sends a card update to all clients.
// For interface-specific updates, use BroadcastCardUpdateForInterface instead.
func (h *Hub) BroadcastCardUpdate(cardID string, data any) {
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
func (h *Hub) BroadcastCardUpdateForInterface(cardID string, data any, iface string) {
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
func (h *Hub) BroadcastLogEntry(entry any) {
	h.Broadcast(Message{
		Type:    "log_entry",
		Payload: entry,
	})
}

// dbLogWriterAdapter implements logging.DBLogWriter interface for database persistence.
// This adapter connects the logging package to the database package.
type dbLogWriterAdapter struct {
	db *database.DB
}

// WriteLog implements logging.DBLogWriter interface - writes a single log entry.
func (a *dbLogWriterAdapter) WriteLog(ctx context.Context, entry *logging.LogEntry) error {
	if a.db == nil {
		return nil
	}

	dbEntry := &database.LogEntry{
		Timestamp:  entry.Timestamp,
		Level:      entry.Level,
		Layer:      entry.Layer,
		Message:    entry.Message,
		Component:  entry.Component,
		RequestID:  entry.RequestID,
		SessionID:  entry.SessionID,
		DurationMs: entry.DurationMs,
		Metadata:   database.ConvertMetadataToJSON(entry.Metadata),
		Stack:      entry.Stack,
	}

	return a.db.Logs().Create(ctx, dbEntry)
}

// WriteBatch implements logging.DBLogWriter interface - writes multiple log entries.
func (a *dbLogWriterAdapter) WriteBatch(ctx context.Context, entries []*logging.LogEntry) error {
	if a.db == nil || len(entries) == 0 {
		return nil
	}

	dbEntries := make([]*database.LogEntry, len(entries))
	for i, entry := range entries {
		dbEntries[i] = &database.LogEntry{
			Timestamp:  entry.Timestamp,
			Level:      entry.Level,
			Layer:      entry.Layer,
			Message:    entry.Message,
			Component:  entry.Component,
			RequestID:  entry.RequestID,
			SessionID:  entry.SessionID,
			DurationMs: entry.DurationMs,
			Metadata:   database.ConvertMetadataToJSON(entry.Metadata),
			Stack:      entry.Stack,
		}
	}

	return a.db.Logs().BatchCreate(ctx, dbEntries)
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

// dbDeviceWriterAdapter implements discovery.DBDeviceWriter for database persistence.
type dbDeviceWriterAdapter struct {
	db *database.DB
}

// PersistDevices implements discovery.DBDeviceWriter interface.
func (a *dbDeviceWriterAdapter) PersistDevices(
	ctx context.Context,
	devices []*discovery.DiscoveredDevice,
) error {
	if a.db == nil || len(devices) == 0 {
		return nil
	}

	for _, d := range devices {
		dbDevice := &database.Device{
			IPAddress:  d.IP,
			MACAddress: d.MAC,
			Hostname:   d.DisplayName,
			Vendor:     d.Vendor,
			DeviceType: guessDeviceType(d),
			OSFamily:   d.OSGuess,
			LastSeen:   d.LastSeen,
			IsActive:   true,
		}

		// Upsert by MAC if available, otherwise by IP
		if d.MAC != "" {
			if err := a.db.Devices().UpsertByMAC(ctx, dbDevice); err != nil {
				// Log but don't fail - continue with other devices
				continue
			}
		} else if d.IP != "" {
			if err := a.db.Devices().Upsert(ctx, dbDevice); err != nil {
				continue
			}
		}
	}

	return nil
}

// guessDeviceType infers a device type from discovery information.
func guessDeviceType(d *discovery.DiscoveredDevice) string {
	if d.LLDPInfo != nil {
		for _, cap := range d.LLDPInfo.Capabilities {
			switch cap {
			case "Router":
				return "router"
			case "Bridge":
				return "switch"
			case "Telephone":
				return "phone"
			case "WLAN Access Point":
				return "access_point"
			}
		}
	}
	if d.CDPInfo != nil {
		if d.CDPInfo.Platform != "" {
			return "network_device"
		}
	}
	if d.IsRouter {
		return "router"
	}
	return "host"
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
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusUnauthorized,
			ErrCodeUnauthorized,
			localizer.T("errors.auth.noToken"),
			"",
		) // fixes #694
		return
	}

	claims, err := s.authManager().ValidateToken(r.Context(), token)
	if err != nil {
		logging.GetLogger().Warn("WebSocket auth failed", "error", err, "source", source)
		logger := logging.FromContext(r.Context())
		localizer := i18n.FromRequest(r)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusUnauthorized,
			ErrCodeUnauthorized,
			localizer.T("errors.auth.invalidToken"),
			"",
		)
		return
	}

	logging.GetLogger().
		Debug("WebSocket authenticated", "username", claims.Username, "source", source)

	// No response header needed for cookie auth
	conn, err := getWSState().getUpgrader().Upgrade(w, r, nil)
	if err != nil {
		logging.GetLogger().Error("WebSocket upgrade error", "error", err)
		return
	}

	client := &Client{
		hub:     s.wsHub(),
		conn:    conn,
		send:    make(chan []byte, wsBroadcastBufferSize),
		limiter: rate.NewLimiter(messageRateLimit, messageRateBurst),
	}

	s.wsHub().register <- client

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
	if s.wifiManager() != nil {
		isWireless = s.wifiManager().IsWireless()
	}

	// Build initial state with actual card data
	cards := make(map[string]any)

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
		Payload: map[string]any{
			"status":     "connected",
			"interface":  s.config.Interface.Default,
			"isWireless": isWireless,
			"cards":      cards,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logging.GetLogger().Error("Error marshaling initial state", "error", err)
		return
	}

	// Safely attempt to send; channel may already be closed if the client dropped.
	func() {
		defer func() {
			if r := recover(); r != nil {
				logging.GetLogger().Debug("Skipped initial state send (client gone)", "recover", r)
			}
		}()
		select {
		case client.send <- data:
		default:
			logging.GetLogger().Warn("Failed to send initial state to client")
		}
	}()
}

// close safely closes the client connection exactly once.
func (c *Client) close() {
	c.closeOnce.Do(func() {
		// Close connection first to force writePump to exit via conn.Close() (fixes #835)
		// This ensures the writePump goroutine doesn't block forever on c.send
		_ = c.conn.Close()

		// Then try to unregister - if this times out, the client is already
		// disconnected so it won't receive messages anyway (fixes #686)
		select {
		case c.hub.unregister <- c:
			// Successfully sent unregister request
		case <-time.After(wsBroadcastTimeoutMs * time.Millisecond):
			// Channel full or hub not responding, client already disconnected
			logging.GetLogger().Debug("Client unregister timeout, connection already closed")
		}
	})
}

// configureReadPump sets up the connection for reading messages.
// Returns false if configuration fails and the read pump should exit.
func (c *Client) configureReadPump() bool {
	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logging.GetLogger().Error("Failed to set initial read deadline", "error", err)
		return false
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			logging.GetLogger().Error("Failed to extend read deadline", "error", err)
		}
		return nil
	})
	return true
}

// handleReadError processes WebSocket read errors and logs unexpected closures.
// Returns true if the error is expected (normal close), false otherwise.
func (c *Client) handleReadError(err error) {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		logging.GetLogger().Warn("WebSocket error", "error", err)
	}
}

// checkRateLimit verifies the client hasn't exceeded message rate limits.
// Returns true if the message should be processed, false if rate limit exceeded.
func (c *Client) checkRateLimit() bool {
	if c.limiter.Allow() {
		return true
	}
	logging.GetLogger().Warn("WebSocket rate limit exceeded, closing connection")
	closeMsg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "rate limit exceeded")
	if writeErr := c.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(writeWait)); writeErr != nil {
		logging.GetLogger().Error("Failed to send rate limit close message", "error", writeErr)
	}
	return false
}

// parseMessage unmarshals a raw WebSocket message into a Message struct.
// Returns the parsed message and true on success, or an empty message and false on failure.
func (c *Client) parseMessage(data []byte) (Message, bool) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		logging.GetLogger().Warn("Error parsing message", "error", err)
		return Message{}, false
	}
	return msg, true
}

// handlePingMessage responds to client ping with a pong message.
func (c *Client) handlePingMessage() {
	response := Message{Type: "pong", Payload: time.Now().UnixMilli()}
	data, err := json.Marshal(response)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		logging.GetLogger().Warn("Client send buffer full, dropping pong")
	}
}

// handleCardUpdateRequest processes a request for a specific card update.
func (c *Client) handleCardUpdateRequest(msg Message) {
	if cardID, ok := msg.Payload.(string); ok {
		logging.GetLogger().Debug("Card update requested", "card_id", cardID)
		// The server will send the next scheduled update for this card
	}
}

// dispatchMessage routes a parsed message to the appropriate handler.
func (c *Client) dispatchMessage(msg Message) {
	logging.GetLogger().Debug("Received message", "type", msg.Type)
	switch msg.Type {
	case "ping":
		c.handlePingMessage()
	case "requestCardUpdate":
		c.handleCardUpdateRequest(msg)
	default:
		logging.GetLogger().Warn("Unknown message type", "type", msg.Type)
	}
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump() {
	defer c.close()

	if !c.configureReadPump() {
		return
	}

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.handleReadError(err)
			return
		}

		if !c.checkRateLimit() {
			return
		}

		msg, ok := c.parseMessage(message)
		if !ok {
			continue
		}

		c.dispatchMessage(msg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
// Fixes #869: Ensure writer is closed on all error paths to prevent resource leaks.
//
//nolint:gocognit // WebSocket pump handles multiple message types
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
				logging.GetLogger().Error("Failed to set write deadline", "error", err)
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logging.GetLogger().Error("Failed to send close message", "error", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Fixes #869: Use a closure to ensure writer is closed on all paths
			writeErr := func() error {
				defer func() { _ = w.Close() }() // Always close writer (fixes #869)

				if _, wErr := w.Write(message); wErr != nil {
					return wErr
				}

				// Add queued messages to the current WebSocket message
				n := len(c.send)
				for range n {
					if _, nlErr := w.Write([]byte{'\n'}); nlErr != nil {
						return nlErr
					}
					if _, msgErr := w.Write(<-c.send); msgErr != nil {
						return msgErr
					}
				}
				return nil
			}()

			if writeErr != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logging.GetLogger().Error("Failed to set ping write deadline", "error", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
