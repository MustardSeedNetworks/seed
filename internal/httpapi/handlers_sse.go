package httpapi

//
// Server-Sent Events (SSE) Implementation
//
// This file implements SSE for real-time updates to connected clients, replacing
// the more complex WebSocket implementation. SSE provides a simpler, HTTP-based
// approach for server-to-client streaming.
//
// SSE Protocol:
//   - Endpoint: /api/events
//   - Content-Type: text/event-stream
//   - Message format: "data: {json}\n\n"
//   - Automatic reconnection built into browser EventSource API
//   - Heartbeat: Comment every 30 seconds to keep connection alive
//
// Message Types:
//   - initial_state: Full state on connection (interface, cards data)
//   - card_update: Real-time dashboard card updates
//   - pipeline: Discovery pipeline events
//   - traceHop: Path tracing hop updates
//   - log_entry: Real-time log streaming
//
// Security:
//   - Cookie-based authentication (httpOnly cookies)
//   - CORS/Origin validation against configured allowed origins
//   - Same-origin policy enforced by browser for EventSource
//
// Advantages over WebSocket:
//   - Simpler protocol (standard HTTP)
//   - Automatic reconnection handled by browser
//   - Works through HTTP proxies and load balancers
//   - No special upgrade handshake required
//   - Simpler server implementation
//

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

const (
	// sseHeartbeatInterval is how often to send heartbeat comments to keep connection alive.
	sseHeartbeatInterval = 30 * time.Second

	// sseBroadcastBufferSize is the buffer size for the SSE broadcast channel.
	sseBroadcastBufferSize = 256

	// sseClientBufferSize is the buffer size for each client's message channel.
	sseClientBufferSize = 64

	// sseBroadcastTimeoutMs is the timeout in milliseconds for broadcast operations.
	sseBroadcastTimeoutMs = 100
)

// SSEClient represents a single SSE client connection.
type SSEClient struct {
	id       uint64
	messages chan []byte
	done     chan struct{}
	hub      *SSEHub
}

// SSEHub maintains the set of active SSE clients and broadcasts messages.
type SSEHub struct {
	clients    map[*SSEClient]bool
	broadcast  chan []byte
	register   chan *SSEClient
	unregister chan *SSEClient
	shutdown   chan struct{}
	mu         sync.RWMutex
	nextID     uint64
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:    make(map[*SSEClient]bool),
		broadcast:  make(chan []byte, sseBroadcastBufferSize),
		register:   make(chan *SSEClient),
		unregister: make(chan *SSEClient),
		shutdown:   make(chan struct{}),
	}
}

// Run starts the SSE hub's main loop.
func (h *SSEHub) Run() {
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
func (h *SSEHub) handleRegister(client *SSEClient) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
	logging.GetLogger().Debug("SSE client connected", "client_id", client.id, "total_clients", len(h.clients))
}

// handleUnregister removes a client from the hub.
func (h *SSEHub) handleUnregister(client *SSEClient) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.messages)
	}
	h.mu.Unlock()
	logging.GetLogger().Debug("SSE client disconnected", "client_id", client.id, "total_clients", len(h.clients))
}

// handleBroadcast sends a message to all connected clients.
func (h *SSEHub) handleBroadcast(message []byte) {
	var slowClients []*SSEClient
	h.mu.RLock()
	for client := range h.clients {
		select {
		case client.messages <- message:
		default:
			// Client is slow - collect for removal
			slowClients = append(slowClients, client)
		}
	}
	h.mu.RUnlock()

	// Remove slow clients
	if len(slowClients) > 0 {
		h.mu.Lock()
		for _, client := range slowClients {
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.messages)
				logging.GetLogger().Warn("SSE client removed - too slow", "client_id", client.id)
			}
		}
		h.mu.Unlock()
	}
}

// handleShutdown closes all client connections.
func (h *SSEHub) handleShutdown() {
	h.mu.Lock()
	for client := range h.clients {
		close(client.messages)
		delete(h.clients, client)
	}
	h.mu.Unlock()
}

// Shutdown stops the SSE hub.
func (h *SSEHub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	select {
	case <-h.shutdown:
		return // Already closed
	default:
		close(h.shutdown)
	}
}

// Broadcast sends a message to all connected clients.
func (h *SSEHub) Broadcast(msg Message) {
	// Check shutdown first to avoid work during shutdown
	select {
	case <-h.shutdown:
		logging.GetLogger().Debug("SSE broadcast dropped - hub already shut down")
		return
	default:
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logging.GetLogger().Error("Error marshaling SSE message", "error", err)
		return
	}

	// Non-blocking send with timeout
	select {
	case h.broadcast <- data:
		// Message sent successfully
	case <-h.shutdown:
		logging.GetLogger().Debug("SSE broadcast dropped - hub shutting down")
	case <-time.After(sseBroadcastTimeoutMs * time.Millisecond):
		logging.GetLogger().Warn("SSE broadcast timeout - hub may be overloaded")
	}
}

// BroadcastCardUpdate sends a card update to all clients.
func (h *SSEHub) BroadcastCardUpdate(cardID string, data any) {
	h.Broadcast(Message{
		Type: "card_update",
		Payload: CardUpdate{
			CardID: cardID,
			Data:   data,
		},
	})
}

// BroadcastCardUpdateForInterface sends a card update scoped to a specific interface.
func (h *SSEHub) BroadcastCardUpdateForInterface(cardID string, data any, iface string) {
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
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastLogEntry sends a log entry to all connected clients.
func (h *SSEHub) BroadcastLogEntry(entry any) {
	h.Broadcast(Message{
		Type:    "log_entry",
		Payload: entry,
	})
}

// newSSEClient creates a new SSE client.
func (h *SSEHub) newClient() *SSEClient {
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	h.mu.Unlock()

	return &SSEClient{
		id:       id,
		messages: make(chan []byte, sseClientBufferSize),
		done:     make(chan struct{}),
		hub:      h,
	}
}

// handleSSE handles Server-Sent Events connections.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	// Validate authentication via httpOnly cookie
	token, source := auth.GetTokenFromRequest(r)
	if token == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized,
			localizer.T("errors.auth.noToken"), "")
		return
	}

	claims, err := s.authManager().ValidateToken(r.Context(), token)
	if err != nil {
		logger.Warn("SSE auth failed", "error", err, "source", source)
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized,
			localizer.T("errors.auth.invalidToken"), "")
		return
	}

	logger.Debug("SSE authenticated", "username", claims.Username, "source", source)

	// Check if the client supports SSE (streaming)
	flusher, ok := w.(http.Flusher)
	if !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal,
			"Streaming not supported", "")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create and register client
	client := s.sseHub().newClient()
	s.sseHub().register <- client

	// Ensure client is unregistered on exit
	defer func() {
		close(client.done)
		s.sseHub().unregister <- client
	}()

	// Send initial state
	s.sendSSEInitialState(w, flusher)

	// Create context for cleanup
	ctx := r.Context()

	// Start heartbeat ticker
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return

		case message, msgOK := <-client.messages:
			if !msgOK {
				// Channel closed, client removed
				return
			}
			// Write SSE formatted message
			if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", message); writeErr != nil {
				logger.Debug("SSE write error", "error", writeErr)
				return
			}
			flusher.Flush()

		case <-heartbeat.C:
			// Send heartbeat comment to keep connection alive
			if _, writeErr := fmt.Fprintf(w, ": heartbeat\n\n"); writeErr != nil {
				logger.Debug("SSE heartbeat error", "error", writeErr)
				return
			}
			flusher.Flush()
		}
	}
}

// sendSSEInitialState sends the current state to a newly connected SSE client.
func (s *Server) sendSSEInitialState(w http.ResponseWriter, flusher http.Flusher) {
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
		logging.GetLogger().Error("Error marshaling SSE initial state", "error", err)
		return
	}

	// Write SSE formatted message
	if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", data); writeErr != nil {
		logging.GetLogger().Debug("SSE initial state write error", "error", writeErr)
		return
	}
	flusher.Flush()
}

// sseLogBroadcastAdapter wraps the SSEHub to implement logging.Broadcaster interface.
type sseLogBroadcastAdapter struct {
	hub *SSEHub
}

// BroadcastLogEntry implements logging.Broadcaster interface.
func (a *sseLogBroadcastAdapter) BroadcastLogEntry(entry *logging.LogEntry) {
	if a.hub != nil {
		a.hub.BroadcastLogEntry(entry)
	}
}
