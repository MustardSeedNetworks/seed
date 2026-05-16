package api

// server_shutdown.go contains the graceful shutdown path, the periodic data
// retention goroutine, and the link-state-change broadcaster that runs while
// the server is alive.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/netif"
)

// getClientIP extracts the client IP from a request, considering trusted proxies.
// If trusted proxies are configured and the request comes from one, uses X-Forwarded-For.
// Otherwise, uses RemoteAddr (the only secure option).
func (s *Server) getClientIP(r *http.Request) string {
	return GetClientIPWithTrustedProxies(r, s.trustedProxies())
}

// onLinkStateChange handles link up/down events.
func (s *Server) onLinkStateChange(event netif.LinkEvent) {
	logging.GetLogger().
		Info("Link state change", "interface", event.Interface, "state", event.State)

	switch event.State {
	case netif.LinkStateUp:
		// Link came up - reload discovery service to restart protocol capture
		logging.GetLogger().Info("Link up - reloading discovery service")
		if err := s.discoveryService().Reload(); err != nil {
			logging.GetLogger().Warn("Failed to reload discovery service", "error", err)
		}

		// Notify clients with linkState message (SSE primary, WebSocket for backwards compat)
		linkStateMsg := Message{
			Type: "linkState",
			Payload: map[string]any{
				jsonKeyInterface: event.Interface,
				"state":          "up",
				jsonKeyTimestamp: event.Timestamp.Format(time.RFC3339),
			},
		}
		s.sseHub().Broadcast(linkStateMsg)

		// Also broadcast link card update immediately to trigger frontend auto-run tests.
		// The frontend listens for card_update messages on the "link" card to detect
		// link-up transitions and run speedtest/iperf tests.
		// Multi-interface support (#754): Include interface in broadcast.
		if linkData := s.collectLinkData(); linkData != nil {
			s.sseHub().BroadcastCardUpdateForInterface("link", linkData, event.Interface)
		}
	case netif.LinkStateDown:
		// Link went down - notify clients
		logging.GetLogger().Info("Link down - notifying clients")
		linkStateMsg := Message{
			Type: "linkState",
			Payload: map[string]any{
				jsonKeyInterface: event.Interface,
				"state":          "down",
				jsonKeyTimestamp: event.Timestamp.Format(time.RFC3339),
			},
		}
		s.sseHub().Broadcast(linkStateMsg)

		// Also broadcast link card update for proper state tracking.
		// Frontend uses this to track DOWN state for detecting DOWN→UP transitions.
		// Multi-interface support (#754): Include interface in broadcast.
		if linkData := s.collectLinkData(); linkData != nil {
			s.sseHub().BroadcastCardUpdateForInterface("link", linkData, event.Interface)
		}
	case netif.LinkStateUnknown:
		// Unknown state - log but don't take action
		logging.GetLogger().Warn("Link state unknown", "interface", event.Interface)
	}
}

// Shutdown gracefully shuts down the server (fixes #515, #524).
func (s *Server) Shutdown(ctx context.Context) error {
	logging.GetLogger().InfoContext(ctx, "Shutting down server...")

	// Shutdown HTTP redirect server if running (fixes #515)
	if s.redirectServer != nil {
		logging.GetLogger().InfoContext(ctx, "Shutting down HTTP redirect server...")
		if err := s.redirectServer.Shutdown(ctx); err != nil {
			logging.GetLogger().
				ErrorContext(ctx, "Error shutting down redirect server", "error", err)
		}
	}

	// Shutdown ACME HTTP-01 challenge server if running (fixes #837)
	if s.acmeChallengeServer != nil {
		logging.GetLogger().InfoContext(ctx, "Shutting down ACME challenge server...")
		if err := s.acmeChallengeServer.Shutdown(ctx); err != nil {
			logging.GetLogger().
				ErrorContext(ctx, "Error shutting down ACME challenge server", "error", err)
		}
	}

	// Stop all services (fixes #524 - services will complete gracefully)
	logging.GetLogger().InfoContext(ctx, "Stopping SSE hub...")
	s.sseHub().Shutdown()

	logging.GetLogger().InfoContext(ctx, "Stopping link monitor...")
	s.linkMonitor().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping discovery service...")
	s.discoveryService().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping VLAN traffic monitor...")
	s.vlanTrafficMonitor().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping rate limiters...")
	s.loginRateLimiter().Stop()
	s.endpointRateLimiter().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping CSRF manager...")
	s.csrfManager().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping auth manager (token blacklist)...")
	s.authManager().Stop()

	// Stop data retention goroutine (fixes #848)
	if s.services.Database.RetentionStopCh != nil {
		logging.GetLogger().InfoContext(ctx, "Stopping data retention goroutine...")
		close(s.services.Database.RetentionStopCh)
		s.services.Database.RetentionStopCh = nil
	}

	// Close database connection (#755)
	if s.db() != nil {
		logging.GetLogger().InfoContext(ctx, "Closing database connection...")
		if err := s.db().Close(); err != nil {
			logging.GetLogger().ErrorContext(ctx, "Error closing database", "error", err)
		}
	}

	// Shutdown main HTTP server
	logging.GetLogger().InfoContext(ctx, "Shutting down main HTTP server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown main server: %w", err)
	}
	return nil
}

// startDataRetention runs periodic data cleanup based on retention policy (#755).
// The goroutine respects shutdown signals to avoid leaks (fixes #848).
func (s *Server) startDataRetention(retentionDays int) {
	// Run cleanup every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	policy := database.RetentionPolicy{
		MetricsDays:        retentionDays,
		AlertsDays:         retentionDays * retentionAlertsMultiplier, // Keep alerts longer
		SpeedTestDays:      retentionDays,
		DNSResultDays:      retentionDays,
		GatewayResultDays:  retentionDays,
		AuditLogDays:       retentionDays * retentionAuditLogMultiplier,       // Keep audit logs longest
		InactiveDeviceDays: retentionDays * retentionInactiveDeviceMultiplier, // Keep inactive device records longer
	}

	for {
		select {
		case <-s.services.Database.RetentionStopCh:
			logging.GetLogger().Debug("Data retention goroutine shutting down")
			return
		case <-ticker.C:
			if s.db() == nil {
				return
			}
			result, err := s.db().RunCleanup(context.Background(), policy)
			if err != nil {
				logging.GetLogger().Error("Data retention cleanup failed", "error", err)
				continue
			}
			totalDeleted := result.MetricsDeleted + result.AlertsDeleted +
				result.SpeedTestsDeleted + result.DNSResultsDeleted +
				result.GatewayResultsDeleted + result.AuditLogsDeleted +
				result.DevicesDeleted
			if totalDeleted > 0 {
				logging.GetLogger().Info("Data retention cleanup completed",
					"metrics_deleted", result.MetricsDeleted,
					"alerts_deleted", result.AlertsDeleted,
					"devices_deleted", result.DevicesDeleted,
					"speedtests_deleted", result.SpeedTestsDeleted,
					"dns_deleted", result.DNSResultsDeleted,
					"gateway_deleted", result.GatewayResultsDeleted,
					"audit_deleted", result.AuditLogsDeleted,
					"duration", result.Duration)
			}
		}
	}
}
