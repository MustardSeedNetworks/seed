package api

//
// This file contains shared message types and adapters used by the SSE broadcasting system.
// These types are used for real-time updates to connected clients.
//

import (
	"context"
	"strings"
	"sync"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/services/discovery"
)

// Message represents a broadcast message sent to SSE clients.
//
// All real-time updates use this structure with a type discriminator
// and variable payload. Clients should switch on Type to decode Payload.
type Message struct {
	Type    string `json:"type"`    // Message type identifier
	Payload any    `json:"payload"` // Type-specific data (varies by Type)
}

// CardUpdate represents a dashboard card data update for periodic refreshes.
//
// Card updates are sent via "card_update" messages to refresh specific dashboard
// cards without requiring full page reloads. The Data field contains the complete
// updated state for the identified card.
//
// Multi-interface support (#754): The Interface field identifies which network
// interface the card data pertains to. Clients can filter updates based on their
// selected interface.
type CardUpdate struct {
	CardID    string `json:"cardId"`              // Unique card identifier (e.g., "link", "dns", "gateway")
	Data      any    `json:"data"`                // Complete card data (structure varies by CardID)
	Interface string `json:"interface,omitempty"` // Network interface name (e.g., "eth0", "wlan0")
}

// Constants for origin validation.
const (
	// multicastOctetMax is the maximum second octet value for Class B addresses (172.16-31.x.x range).
	multicastOctetMax = 31

	// ipPartsClassC is the expected number of IP parts for Class C address validation.
	ipPartsClassC = 2

	// ipPartsClassAB is the expected number of IP parts for Class A/B address validation.
	ipPartsClassAB = 3

	// maxIPOctetValue is the maximum valid value for an IP address octet (255).
	maxIPOctetValue = 255

	// decimalParseBase is the base for decimal digit parsing.
	decimalParseBase = 10
)

// originConfig holds origin validation configuration state.
// Access via getOriginState().
type originConfig struct {
	mu      sync.RWMutex
	origins []string
}

// Origin configuration state accessor functions.
//
//nolint:gochecknoglobals // Intentional thread-safe singleton using closure pattern
var (
	getOriginState, _, _ = func() (
		func() *originConfig,
		func(*originConfig),
		func(),
	) {
		var (
			mu    sync.RWMutex
			state *originConfig
		)
		return func() *originConfig {
				mu.Lock()
				defer mu.Unlock()
				if state == nil {
					state = &originConfig{}
				}
				return state
			}, func(s *originConfig) {
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

// setAllowedOrigins configures the list of allowed origins for CORS validation.
func (c *originConfig) setAllowedOrigins(origins []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.origins = origins
}

// getConfiguredOrigins returns the configured allowed origins.
func (c *originConfig) getConfiguredOrigins() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.origins
}

// isAllowedOrigin checks if the origin is allowed based on configuration.
// Used by CORS middleware for origin validation.
func isAllowedOrigin(origin string) bool {
	return isAllowedOriginWithGetter(origin, getOriginState().getConfiguredOrigins)
}

// isAllowedOriginWithGetter is the internal implementation that accepts a getter function.
func isAllowedOriginWithGetter(origin string, getOrigins func() []string) bool {
	origins := getOrigins()

	// Default: Allow localhost and RFC 1918 private networks if no explicit origins configured
	if len(origins) == 0 {
		return isRFC1918Origin(origin)
	}

	// Check against explicit origin list
	for _, allowed := range origins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	return false
}

// isRFC1918Origin checks if origin is from localhost or RFC 1918 private network.
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
func extractHostFromOrigin(origin string) (string, bool) {
	var host string

	switch {
	case strings.HasPrefix(origin, "http://"):
		host = origin[7:]
	case strings.HasPrefix(origin, "https://"):
		host = origin[8:]
	default:
		return "", false
	}

	// Remove port if present
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// Remove path if present
	if slashIdx := strings.Index(host, "/"); slashIdx != -1 {
		host = host[:slashIdx]
	}

	return host, true
}

// isLocalhostAddress checks if the host is a localhost address.
func isLocalhostAddress(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "[::1]"
}

// isPrivateNetworkAddress checks if the host is an RFC 1918 private network address.
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
func isValidClassCAddress(host string) bool {
	parts := strings.Split(host[8:], ".") // Skip "192.168."
	if len(parts) != ipPartsClassC {
		return false
	}
	return isValidOctet(parts[0]) && isValidOctet(parts[1])
}

// isValidClassAAddress validates a 10.x.x.x address.
func isValidClassAAddress(host string) bool {
	parts := strings.Split(host[3:], ".") // Skip "10."
	if len(parts) != ipPartsClassAB {
		return false
	}
	return isValidOctet(parts[0]) && isValidOctet(parts[1]) && isValidOctet(parts[2])
}

// isValidClassBAddress validates a 172.16-31.x.x address.
func isValidClassBAddress(host string) bool {
	parts := strings.Split(host[4:], ".") // Skip "172."
	if len(parts) != ipPartsClassAB {
		return false
	}

	// Validate second octet is in range 16-31
	secondOctet := parseOctet(parts[0])
	if secondOctet < 16 || secondOctet > multicastOctetMax {
		return false
	}

	return isValidOctet(parts[1]) && isValidOctet(parts[2])
}

// isValidOctet checks if the string is a valid IP octet (0-255).
func isValidOctet(s string) bool {
	if len(s) == 0 || len(s) > 3 {
		return false
	}
	n := parseOctet(s)
	return n >= 0 && n <= maxIPOctetValue
}

// parseOctet parses a string as an IP octet, returning -1 on error.
func parseOctet(s string) int {
	if len(s) == 0 {
		return -1
	}
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*decimalParseBase + int(c-'0')
		if n > maxIPOctetValue {
			return -1
		}
	}
	return n
}

// pipelineBroadcastAdapter implements discovery.EventBroadcaster using SSE for real-time updates.
type pipelineBroadcastAdapter struct {
	hub *SSEHub
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

// dbLogWriterAdapter implements logging.DBLogWriter for database persistence.
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

	// Persist each device individually using upsert
	for _, d := range devices {
		dbDevice := &database.Device{
			ID:         d.MAC, // Use MAC as unique ID
			IPAddress:  d.IP,
			MACAddress: d.MAC,
			Hostname:   d.Hostname,
			Vendor:     d.Vendor,
			DeviceType: d.OSGuess,
			LastSeen:   d.LastSeen,
			IsActive:   true,
		}
		if err := a.db.Devices().Upsert(ctx, dbDevice); err != nil {
			return err
		}
	}

	return nil
}
