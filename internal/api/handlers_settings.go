// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
)

// ============================================================================
// Settings Handlers (fixes #544 - split from handlers.go)
// ============================================================================

// handleSettingsDefaults returns all default settings as the single source of truth.
// This eliminates the need for duplicated DEFAULT_* constants in the frontend.
// The defaults are served from the backend's DefaultConfig() function.
func (s *Server) handleSettingsDefaults(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
		return
	}

	defaults := config.GetDefaultSettings()
	sendJSONResponse(w, logger, http.StatusOK, defaults)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r)
	case http.MethodPut:
		s.updateSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
	}
}

// buildThresholdSettings builds threshold configuration for API response.
func (s *Server) buildThresholdSettings() map[string]any {
	t := &s.config.Thresholds
	return map[string]any{
		"dns": map[string]int64{
			"good":    t.DNS.Warning.Milliseconds(),
			"warning": t.DNS.Critical.Milliseconds(),
		},
		"gateway": map[string]int64{
			"good":    t.Ping.Warning.Milliseconds(),
			"warning": t.Ping.Critical.Milliseconds(),
		},
		"wifi": map[string]int{
			"good":    t.WiFi.Signal.Warning,
			"warning": t.WiFi.Signal.Critical,
		},
		"customPing": map[string]int64{
			"good":    t.CustomTests.Ping.Warning.Milliseconds(),
			"warning": t.CustomTests.Ping.Critical.Milliseconds(),
		},
		"customTcp": map[string]int64{
			"good":    t.CustomTests.TCP.Warning.Milliseconds(),
			"warning": t.CustomTests.TCP.Critical.Milliseconds(),
		},
		"customHttp": map[string]int64{
			"good":    t.CustomTests.HTTP.Warning.Milliseconds(),
			"warning": t.CustomTests.HTTP.Critical.Milliseconds(),
		},
		"httpTimings": map[string]map[string]int64{
			"dns": {
				"good":    t.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
				"warning": t.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
			},
			"tcp": {
				"good":    t.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
				"warning": t.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
			},
			"tls": {
				"good":    t.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
				"warning": t.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
			},
			"ttfb": {
				"good":    t.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
				"warning": t.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
			},
		},
	}
}

// buildCardSettings builds default card visibility settings.
func buildCardSettings() map[string]any {
	defaultCard := map[string]any{"visible": true, "autoRunOnLink": true}
	return map[string]any{
		"link": defaultCard, "switch": defaultCard, "vlan": defaultCard,
		"network": defaultCard, "gateway": defaultCard, "dns": defaultCard,
		"healthChecks": defaultCard, "networkDiscovery": defaultCard,
		"performance": map[string]any{
			"visible": true, "autoRunOnLink": true,
			"speedtest": map[string]any{"enabled": true, "autoRunOnLink": true},
			"iperf":     map[string]any{"enabled": false, "autoRunOnLink": false},
		},
	}
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	s.config.RLock()
	defer s.config.RUnlock()

	settings := map[string]any{
		"interface": map[string]any{
			"current":   s.config.Interface.Default,
			"available": []string{},
		},
		"vlan":       map[string]any{"enabled": s.config.VLAN.Enabled, "id": s.config.VLAN.ID},
		"ip":         map[string]any{"mode": s.config.IP.Mode},
		"thresholds": s.buildThresholdSettings(),
		"healthChecks": map[string]any{
			"runPerformance": true,
			"runSpeedtest":   true,
			"runIperf":       false,
			"runDiscovery":   true,
		},
		"speedtest": map[string]any{
			"serverId":      s.config.Speedtest.ServerID,
			"autoRunOnLink": true,
		},
		"iperf": map[string]any{
			"autoRunOnLink": s.config.Iperf.AutoRunOnLink, "server": s.config.Iperf.Server,
			"port": s.config.Iperf.Port, "protocol": s.config.Iperf.Protocol,
			"direction": s.config.Iperf.Direction, "duration": s.config.Iperf.Duration,
			"serverPort": s.config.Iperf.ServerPort, "enableServer": s.config.Iperf.EnableServer,
		},
		"cardSettings": buildCardSettings(),
		"displayOptions": map[string]any{
			"showPublicIP": s.config.DisplayOptions.ShowPublicIP,
			"unitSystem":   s.config.DisplayOptions.UnitSystem,
		},
	}

	sendJSONResponse(w, logger, http.StatusOK, settings)
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		logger.WarnContext(ctx, "Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Invalid request body",
			"",
		)
		return
	}

	// Lock config for write access
	// NOTE: Must unlock before Save() - Save() acquires RLock internally (fixes #783)
	s.config.Lock()

	// Apply updates using helper functions (fixes #784 - return errors for invalid types)
	var applyErrors []error
	if err := applyThresholdUpdates(updates, s.config); err != nil {
		applyErrors = append(applyErrors, err)
	}
	if err := applyHealthChecksUpdates(updates, s.config); err != nil {
		applyErrors = append(applyErrors, err)
	}
	if err := applySpeedtestUpdates(updates, s.config); err != nil {
		applyErrors = append(applyErrors, err)
	}
	if err := applyIperfUpdates(updates, s.config); err != nil {
		applyErrors = append(applyErrors, err)
	}
	if err := applyFABOptionsUpdates(updates, s.config); err != nil {
		applyErrors = append(applyErrors, err)
	}
	if err := applyDisplayOptionsUpdates(updates, s.config); err != nil {
		applyErrors = append(applyErrors, err)
	}

	// Unlock before Save() to avoid deadlock - Save() acquires RLock internally
	s.config.Unlock()

	// Check for validation errors (fixes #784)
	if len(applyErrors) > 0 {
		logger.WarnContext(ctx, "Invalid settings format", "errors", applyErrors)
		errMsg := "Invalid settings format. Check server logs for details."
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			errMsg,
			"",
		) // fixes #H7
		return
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		logger.ErrorContext(ctx, "Failed to save config", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Failed to save config",
			"",
		)
		return
	}

	// Also save settings to the active profile (fixes #781)
	if s.db != nil {
		if err := s.saveSettingsToActiveProfile(ctx, logger); err != nil {
			sendJSONResponse(w, logger, http.StatusInternalServerError, map[string]string{
				"error": "Failed to save settings",
			})
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

// saveSettingsToActiveProfile saves current settings to the active profile's ConfigJSON.
// This ensures profile-specific settings are persisted (fixes #781).
func (s *Server) saveSettingsToActiveProfile(ctx context.Context, logger *slog.Logger) error {
	// Get active profile ID
	activeID, err := s.db.Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil || activeID == "" {
		// No active profile, try to get default
		defaultProfile, getDefaultErr := s.db.Profiles().GetDefault(ctx)
		if getDefaultErr != nil {
			// No profile exists - this is not an error, just nothing to save to
			logger.DebugContext(ctx,
				"No active or default profile to save settings to",
				"reason",
				getDefaultErr.Error(),
			)
			return nil
		}
		activeID = defaultProfile.ID
	}

	// Get the profile
	profile, err := s.db.Profiles().Get(ctx, activeID)
	if err != nil {
		logger.WarnContext(ctx,
			"Failed to get active profile for settings save",
			"error",
			err,
			"profile_id",
			activeID,
		)
		return nil
	}

	// Extract current settings from config
	profileSettings := config.NewProfileSettings()
	profileSettings.FromConfig(s.config)

	// Preserve existing notes if any
	if profile.ConfigJSON != "" {
		existingSettings, parseErr := config.ParseProfileSettings(profile.ConfigJSON)
		if parseErr == nil && existingSettings.Notes != "" {
			profileSettings.Notes = existingSettings.Notes
		}
	}

	// Serialize to JSON
	configJSON, jsonErr := profileSettings.ToJSON()
	if jsonErr != nil {
		logger.WarnContext(ctx, "Failed to serialize profile settings", "error", jsonErr)
		return nil
	}

	// Update profile
	profile.ConfigJSON = configJSON
	if updateErr := s.db.Profiles().Update(ctx, profile); updateErr != nil {
		logger.ErrorContext(ctx,
			"Failed to save settings to profile",
			"error",
			updateErr,
			"profile_id",
			profile.ID,
		)
		return updateErr
	}

	logger.DebugContext(ctx,
		"Saved settings to active profile",
		"profile_id",
		profile.ID,
		"profile_name",
		profile.Name,
	)
	return nil
}

// applyThresholdUpdates applies threshold configuration updates.
// Returns error if thresholds key exists but has invalid type (fixes #784).
func applyThresholdUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["thresholds"]
	if !exists {
		return nil // Field not provided - valid for partial updates
	}
	thresholds, ok := val.(map[string]any)
	if !ok {
		return errors.New("thresholds must be an object")
	}

	if err := applyDNSThresholds(thresholds, cfg); err != nil {
		return err
	}
	if err := applyGatewayThresholds(thresholds, cfg); err != nil {
		return err
	}
	if err := applyWiFiThresholds(thresholds, cfg); err != nil {
		return err
	}
	if err := applyCustomTestThresholds(thresholds, cfg); err != nil {
		return err
	}
	return applyHTTPTimingThresholds(thresholds, cfg)
}

// applyDNSThresholds applies DNS threshold updates.
// Returns error if dns key exists but has invalid type (fixes #784, G3).
//

func applyDNSThresholds(thresholds map[string]any, cfg *config.Config) error {
	val, exists := thresholds["dns"]
	if !exists {
		return nil
	}
	dnsThresh, ok := val.(map[string]any)
	if !ok {
		return errors.New("thresholds.dns must be an object")
	}

	// Validate "good" field if present
	if goodVal, goodExists := dnsThresh["good"]; goodExists {
		good, goodOK := goodVal.(float64)
		if !goodOK {
			return errors.New("thresholds.dns.good must be a number")
		}
		cfg.Thresholds.DNS.Warning = time.Duration(good) * time.Millisecond
	}

	// Validate "warning" field if present
	if warningVal, warnExists := dnsThresh["warning"]; warnExists {
		warning, warnOK := warningVal.(float64)
		if !warnOK {
			return errors.New("thresholds.dns.warning must be a number")
		}
		cfg.Thresholds.DNS.Critical = time.Duration(warning) * time.Millisecond
	}

	return nil
}

// applyGatewayThresholds applies gateway ping threshold updates.
// Returns error if gateway key exists but has invalid type (fixes #784, G3).
//

func applyGatewayThresholds(thresholds map[string]any, cfg *config.Config) error {
	val, exists := thresholds["gateway"]
	if !exists {
		return nil
	}
	gwThresh, ok := val.(map[string]any)
	if !ok {
		return errors.New("thresholds.gateway must be an object")
	}

	// Validate "good" field if present
	if goodVal, goodExists := gwThresh["good"]; goodExists {
		good, goodOK := goodVal.(float64)
		if !goodOK {
			return errors.New("thresholds.gateway.good must be a number")
		}
		cfg.Thresholds.Ping.Warning = time.Duration(good) * time.Millisecond
	}

	// Validate "warning" field if present
	if warningVal, warnExists := gwThresh["warning"]; warnExists {
		warning, warnOK := warningVal.(float64)
		if !warnOK {
			return errors.New("thresholds.gateway.warning must be a number")
		}
		cfg.Thresholds.Ping.Critical = time.Duration(warning) * time.Millisecond
	}

	return nil
}

// applyWiFiThresholds applies WiFi signal threshold updates.
// Returns error if wifi key exists but has invalid type (fixes #784, G3).
func applyWiFiThresholds(thresholds map[string]any, cfg *config.Config) error {
	val, exists := thresholds["wifi"]
	if !exists {
		return nil
	}
	wifi, ok := val.(map[string]any)
	if !ok {
		return errors.New("thresholds.wifi must be an object")
	}

	// Validate "good" field if present
	if goodVal, goodExists := wifi["good"]; goodExists {
		good, goodOK := goodVal.(float64)
		if !goodOK {
			return errors.New("thresholds.wifi.good must be a number")
		}
		cfg.Thresholds.WiFi.Signal.Warning = int(good)
	}

	// Validate "warning" field if present
	if warningVal, warnExists := wifi["warning"]; warnExists {
		warning, warnOK := warningVal.(float64)
		if !warnOK {
			return errors.New("thresholds.wifi.warning must be a number")
		}
		cfg.Thresholds.WiFi.Signal.Critical = int(warning)
	}

	return nil
}

// applyCustomTestThresholds applies custom test threshold updates.
// Returns error if any custom test key exists but has invalid type (fixes #784, G3).
func applyCustomTestThresholds(thresholds map[string]any, cfg *config.Config) error {
	// Custom ping thresholds
	if val, pingExists := thresholds["customPing"]; pingExists {
		customPing, pingOK := val.(map[string]any)
		if !pingOK {
			return errors.New("thresholds.customPing must be an object")
		}

		// Validate "good" field if present
		if goodVal, goodExists := customPing["good"]; goodExists {
			good, goodOK := goodVal.(float64)
			if !goodOK {
				return errors.New("thresholds.customPing.good must be a number")
			}
			cfg.Thresholds.CustomTests.Ping.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, warnExists := customPing["warning"]; warnExists {
			warning, warnOK := warningVal.(float64)
			if !warnOK {
				return errors.New("thresholds.customPing.warning must be a number")
			}
			cfg.Thresholds.CustomTests.Ping.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom TCP thresholds
	if val, tcpExists := thresholds["customTcp"]; tcpExists {
		customTCP, tcpOK := val.(map[string]any)
		if !tcpOK {
			return errors.New("thresholds.customTcp must be an object")
		}

		// Validate "good" field if present
		if goodVal, goodExists := customTCP["good"]; goodExists {
			good, goodOK := goodVal.(float64)
			if !goodOK {
				return errors.New("thresholds.customTcp.good must be a number")
			}
			cfg.Thresholds.CustomTests.TCP.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, warnExists := customTCP["warning"]; warnExists {
			warning, warnOK := warningVal.(float64)
			if !warnOK {
				return errors.New("thresholds.customTcp.warning must be a number")
			}
			cfg.Thresholds.CustomTests.TCP.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom HTTP thresholds
	if val, httpExists := thresholds["customHttp"]; httpExists {
		customHTTP, httpOK := val.(map[string]any)
		if !httpOK {
			return errors.New("thresholds.customHttp must be an object")
		}

		// Validate "good" field if present
		if goodVal, goodExists := customHTTP["good"]; goodExists {
			good, goodOK := goodVal.(float64)
			if !goodOK {
				return errors.New("thresholds.customHttp.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTP.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, warnExists := customHTTP["warning"]; warnExists {
			warning, warnOK := warningVal.(float64)
			if !warnOK {
				return errors.New("thresholds.customHttp.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTP.Critical = time.Duration(warning) * time.Millisecond
		}
	}
	return nil
}

// applyHTTPTimingThresholds applies HTTP timing threshold updates.
// Returns error if httpTimings key exists but has invalid type (fixes #784, G3).
func applyHTTPTimingThresholds(thresholds map[string]any, cfg *config.Config) error {
	val, exists := thresholds["httpTimings"]
	if !exists {
		return nil
	}
	httpTimings, ok := val.(map[string]any)
	if !ok {
		return errors.New("thresholds.httpTimings must be an object")
	}

	// DNS timing thresholds
	if dnsVal, dnsExists := httpTimings["dns"]; dnsExists {
		dnsT, dnsOK := dnsVal.(map[string]any)
		if !dnsOK {
			return errors.New("thresholds.httpTimings.dns must be an object")
		}

		// Validate "good" field if present
		if goodVal, goodExists := dnsT["good"]; goodExists {
			good, goodOK := goodVal.(float64)
			if !goodOK {
				return errors.New("thresholds.httpTimings.dns.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning = time.Duration(
				good,
			) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, warnExists := dnsT["warning"]; warnExists {
			warning, warnOK := warningVal.(float64)
			if !warnOK {
				return errors.New("thresholds.httpTimings.dns.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical = time.Duration(
				warning,
			) * time.Millisecond
		}
	}

	// TCP timing thresholds
	if tcpVal, tcpExists := httpTimings["tcp"]; tcpExists {
		tcpT, tcpOK := tcpVal.(map[string]any)
		if !tcpOK {
			return errors.New("thresholds.httpTimings.tcp must be an object")
		}

		// Validate "good" field if present
		if goodVal, goodExists := tcpT["good"]; goodExists {
			good, goodOK := goodVal.(float64)
			if !goodOK {
				return errors.New("thresholds.httpTimings.tcp.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning = time.Duration(
				good,
			) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, warnExists := tcpT["warning"]; warnExists {
			warning, warnOK := warningVal.(float64)
			if !warnOK {
				return errors.New("thresholds.httpTimings.tcp.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical = time.Duration(
				warning,
			) * time.Millisecond
		}
	}

	// TLS timing thresholds
	if tlsVal, tlsExists := httpTimings["tls"]; tlsExists {
		tlsT, tlsOK := tlsVal.(map[string]any)
		if !tlsOK {
			return errors.New("thresholds.httpTimings.tls must be an object")
		}

		// Validate "good" field if present
		if goodVal, goodExists := tlsT["good"]; goodExists {
			good, goodOK := goodVal.(float64)
			if !goodOK {
				return errors.New("thresholds.httpTimings.tls.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning = time.Duration(
				good,
			) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, warnExists := tlsT["warning"]; warnExists {
			warning, warnOK := warningVal.(float64)
			if !warnOK {
				return errors.New("thresholds.httpTimings.tls.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical = time.Duration(
				warning,
			) * time.Millisecond
		}
	}

	// TTFB timing thresholds
	if ttfbVal, ttfbExists := httpTimings["ttfb"]; ttfbExists {
		ttfb, ttfbOK := ttfbVal.(map[string]any)
		if !ttfbOK {
			return errors.New("thresholds.httpTimings.ttfb must be an object")
		}

		// Validate "good" field if present
		if goodVal, goodExists := ttfb["good"]; goodExists {
			good, goodOK := goodVal.(float64)
			if !goodOK {
				return errors.New("thresholds.httpTimings.ttfb.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning = time.Duration(
				good,
			) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, warnExists := ttfb["warning"]; warnExists {
			warning, warnOK := warningVal.(float64)
			if !warnOK {
				return errors.New("thresholds.httpTimings.ttfb.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical = time.Duration(
				warning,
			) * time.Millisecond
		}
	}
	return nil
}

// applyHealthChecksUpdates applies health check toggle updates.
// Returns error if healthChecks key exists but has invalid type (fixes #784, G3).
func applyHealthChecksUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["healthChecks"]
	if !exists {
		return nil
	}
	healthChecks, ok := val.(map[string]any)
	if !ok {
		return errors.New("healthChecks must be an object")
	}

	if perfVal, perfExists := healthChecks["runPerformance"]; perfExists {
		runPerformance, perfOK := perfVal.(bool)
		if !perfOK {
			return errors.New("healthChecks.runPerformance must be a boolean")
		}
		cfg.HealthChecks.RunPerformance = runPerformance
	}

	if speedVal, speedExists := healthChecks["runSpeedtest"]; speedExists {
		runSpeedtest, speedOK := speedVal.(bool)
		if !speedOK {
			return errors.New("healthChecks.runSpeedtest must be a boolean")
		}
		cfg.HealthChecks.RunSpeedtest = runSpeedtest
	}

	if iperfVal, iperfExists := healthChecks["runIperf"]; iperfExists {
		runIperf, iperfOK := iperfVal.(bool)
		if !iperfOK {
			return errors.New("healthChecks.runIperf must be a boolean")
		}
		cfg.HealthChecks.RunIperf = runIperf
	}

	if discVal, discExists := healthChecks["runDiscovery"]; discExists {
		runDiscovery, discOK := discVal.(bool)
		if !discOK {
			return errors.New("healthChecks.runDiscovery must be a boolean")
		}
		cfg.HealthChecks.RunDiscovery = runDiscovery
	}

	return nil
}

// applySpeedtestUpdates applies speedtest configuration updates.
// Returns error if speedtest key exists but has invalid type (fixes #784, G3).
func applySpeedtestUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["speedtest"]
	if !exists {
		return nil
	}
	speedtest, ok := val.(map[string]any)
	if !ok {
		return errors.New("speedtest must be an object")
	}

	if serverIDVal, serverIDExists := speedtest["serverId"]; serverIDExists {
		serverID, serverIDOK := serverIDVal.(string)
		if !serverIDOK {
			return errors.New("speedtest.serverId must be a string")
		}
		cfg.Speedtest.ServerID = serverID
	}

	if autoRunVal, autoRunExists := speedtest["autoRunOnLink"]; autoRunExists {
		autoRunOnLink, autoRunOK := autoRunVal.(bool)
		if !autoRunOK {
			return errors.New("speedtest.autoRunOnLink must be a boolean")
		}
		cfg.Speedtest.AutoRunOnLink = autoRunOnLink
	}

	return nil
}

// applyIperfUpdates applies iperf configuration updates.
// Returns error if iperf key exists but has invalid type (fixes #784, G3).
func applyIperfUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["iperf"]
	if !exists {
		return nil
	}
	iperf, ok := val.(map[string]any)
	if !ok {
		return errors.New("iperf must be an object")
	}

	if autoRunVal, autoRunExists := iperf["autoRunOnLink"]; autoRunExists {
		autoRunOnLink, autoRunOK := autoRunVal.(bool)
		if !autoRunOK {
			return errors.New("iperf.autoRunOnLink must be a boolean")
		}
		cfg.Iperf.AutoRunOnLink = autoRunOnLink
	}

	if serverVal, serverExists := iperf["server"]; serverExists {
		server, serverOK := serverVal.(string)
		if !serverOK {
			return errors.New("iperf.server must be a string")
		}
		cfg.Iperf.Server = server
	}

	if portVal, portExists := iperf["port"]; portExists {
		port, portOK := portVal.(float64)
		if !portOK {
			return errors.New("iperf.port must be a number")
		}
		p := int(port)
		if err := validation.ValidatePort(p); err != nil {
			return fmt.Errorf("iperf.port: %w", err)
		}
		cfg.Iperf.Port = p
	}

	if protoVal, protoExists := iperf["protocol"]; protoExists {
		protocol, protoOK := protoVal.(string)
		if !protoOK {
			return errors.New("iperf.protocol must be a string")
		}
		cfg.Iperf.Protocol = protocol
	}

	if dirVal, dirExists := iperf["direction"]; dirExists {
		direction, dirOK := dirVal.(string)
		if !dirOK {
			return errors.New("iperf.direction must be a string")
		}
		cfg.Iperf.Direction = direction
	}

	if durVal, durExists := iperf["duration"]; durExists {
		duration, durOK := durVal.(float64)
		if !durOK {
			return errors.New("iperf.duration must be a number")
		}
		cfg.Iperf.Duration = int(duration)
	}

	if srvPortVal, srvPortExists := iperf["serverPort"]; srvPortExists {
		serverPort, srvPortOK := srvPortVal.(float64)
		if !srvPortOK {
			return errors.New("iperf.serverPort must be a number")
		}
		p := int(serverPort)
		if validation.ValidatePort(p) == nil {
			cfg.Iperf.ServerPort = p
		}
	}

	if enableVal, enableExists := iperf["enableServer"]; enableExists {
		enableServer, enableOK := enableVal.(bool)
		if !enableOK {
			return errors.New("iperf.enableServer must be a boolean")
		}
		cfg.Iperf.EnableServer = enableServer
	}

	return nil
}

// applyFABOptionsUpdates applies FAB options updates.
// Returns error if fabOptions key exists but has invalid type (fixes #784, G3).
func applyFABOptionsUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["fabOptions"]
	if !exists {
		return nil
	}
	fabOptions, ok := val.(map[string]any)
	if !ok {
		return errors.New("fabOptions must be an object")
	}

	if linkVal, linkExists := fabOptions["runLink"]; linkExists {
		runLink, linkOK := linkVal.(bool)
		if !linkOK {
			return errors.New("fabOptions.runLink must be a boolean")
		}
		cfg.FABOptions.RunLink = runLink
	}

	if switchVal, switchExists := fabOptions["runSwitch"]; switchExists {
		runSwitch, switchOK := switchVal.(bool)
		if !switchOK {
			return errors.New("fabOptions.runSwitch must be a boolean")
		}
		cfg.FABOptions.RunSwitch = runSwitch
	}

	if vlanVal, vlanExists := fabOptions["runVLAN"]; vlanExists {
		runVLAN, vlanOK := vlanVal.(bool)
		if !vlanOK {
			return errors.New("fabOptions.runVLAN must be a boolean")
		}
		cfg.FABOptions.RunVLAN = runVLAN
	}

	if ipVal, ipExists := fabOptions["runIPConfig"]; ipExists {
		runIPConfig, ipOK := ipVal.(bool)
		if !ipOK {
			return errors.New("fabOptions.runIPConfig must be a boolean")
		}
		cfg.FABOptions.RunIPConfig = runIPConfig
	}

	if gwVal, gwExists := fabOptions["runGateway"]; gwExists {
		runGateway, gwOK := gwVal.(bool)
		if !gwOK {
			return errors.New("fabOptions.runGateway must be a boolean")
		}
		cfg.FABOptions.RunGateway = runGateway
	}

	if dnsVal, dnsExists := fabOptions["runDNS"]; dnsExists {
		runDNS, dnsOK := dnsVal.(bool)
		if !dnsOK {
			return errors.New("fabOptions.runDNS must be a boolean")
		}
		cfg.FABOptions.RunDNS = runDNS
	}

	if hcVal, hcExists := fabOptions["runHealthChecks"]; hcExists {
		runHealthChecks, hcOK := hcVal.(bool)
		if !hcOK {
			return errors.New("fabOptions.runHealthChecks must be a boolean")
		}
		cfg.FABOptions.RunHealthChecks = runHealthChecks
	}

	if ndVal, ndExists := fabOptions["runNetworkDiscovery"]; ndExists {
		runNetworkDiscovery, ndOK := ndVal.(bool)
		if !ndOK {
			return errors.New("fabOptions.runNetworkDiscovery must be a boolean")
		}
		cfg.FABOptions.RunNetworkDiscovery = runNetworkDiscovery
	}

	if stVal, stExists := fabOptions["runSpeedtest"]; stExists {
		runSpeedtest, stOK := stVal.(bool)
		if !stOK {
			return errors.New("fabOptions.runSpeedtest must be a boolean")
		}
		cfg.FABOptions.RunSpeedtest = runSpeedtest
	}

	if iperfVal, iperfExists := fabOptions["runIperf"]; iperfExists {
		runIperf, iperfOK := iperfVal.(bool)
		if !iperfOK {
			return errors.New("fabOptions.runIperf must be a boolean")
		}
		cfg.FABOptions.RunIperf = runIperf
	}

	if perfVal, perfExists := fabOptions["runPerformance"]; perfExists {
		runPerformance, perfOK := perfVal.(bool)
		if !perfOK {
			return errors.New("fabOptions.runPerformance must be a boolean")
		}
		cfg.FABOptions.RunPerformance = runPerformance
	}

	if autoVal, autoExists := fabOptions["autoScanOnLink"]; autoExists {
		autoScanOnLink, autoOK := autoVal.(bool)
		if !autoOK {
			return errors.New("fabOptions.autoScanOnLink must be a boolean")
		}
		cfg.FABOptions.AutoScanOnLink = autoScanOnLink
	}

	return nil
}

// applyDisplayOptionsUpdates applies display options updates.
// Returns error if displayOptions key exists but has invalid type (fixes #784, G3).
func applyDisplayOptionsUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["displayOptions"]
	if !exists {
		return nil
	}
	displayOptions, ok := val.(map[string]any)
	if !ok {
		return errors.New("displayOptions must be an object")
	}

	if pubIPVal, pubIPExists := displayOptions["showPublicIP"]; pubIPExists {
		showPublicIP, pubIPOK := pubIPVal.(bool)
		if !pubIPOK {
			return errors.New("displayOptions.showPublicIP must be a boolean")
		}
		cfg.DisplayOptions.ShowPublicIP = showPublicIP
	}

	if unitVal, unitExists := displayOptions["unitSystem"]; unitExists {
		unitSystem, unitOK := unitVal.(string)
		if !unitOK {
			return errors.New("displayOptions.unitSystem must be a string")
		}
		// Validate unit system (only "sae" or "metric" allowed)
		if unitSystem == "sae" || unitSystem == "metric" {
			cfg.DisplayOptions.UnitSystem = unitSystem
		}
	}

	return nil
}

// ============================================================================
// Link Settings Handlers (fixes #734)
// ============================================================================

// handleLinkSettings handles GET/PUT for /api/settings/link.
// Link settings control interface speed/duplex configuration.
func (s *Server) handleLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getLinkSettings(w, r)
	case http.MethodPut:
		s.updateLinkSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
	}
}

// getLinkSettings returns current link settings from the active profile.
func (s *Server) getLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Try to get settings from active profile
	settings := config.ProfileLinkSettings{
		Mode:           "auto",
		AvailableModes: []string{},
	}

	if s.db != nil {
		profileSettings, err := s.getActiveProfileSettings(ctx)
		if err == nil && profileSettings != nil {
			settings = profileSettings.Link
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, settings)
}

// updateLinkSettings updates link settings in the active profile.
func (s *Server) updateLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var updates config.ProfileLinkSettings
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		logger.WarnContext(ctx, "Invalid link settings request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Invalid request body",
			"",
		)
		return
	}

	// Validate mode value (combined speed/duplex format like "100/full" or "auto")
	validModes := map[string]bool{
		"auto": true, "10/half": true, "10/full": true, "100/half": true, "100/full": true,
		"1000/full": true, "2500/full": true, "5000/full": true, "10000/full": true,
	}
	if updates.Mode != "" && !validModes[updates.Mode] {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			"Invalid mode value",
			"",
		)
		return
	}

	// Save to active profile
	if s.db != nil {
		if err := s.updateActiveProfileLinkSettings(ctx, logger, updates); err != nil {
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusInternalServerError,
				ErrCodeInternal,
				"Failed to save link settings",
				"",
			)
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

// updateActiveProfileLinkSettings saves link settings to the active profile.
func (s *Server) updateActiveProfileLinkSettings(
	ctx context.Context,
	logger *slog.Logger,
	settings config.ProfileLinkSettings,
) error {
	profileSettings, err := s.getActiveProfileSettings(ctx)
	if err != nil {
		logger.WarnContext(ctx, "Failed to get active profile settings", "error", err)
		return err
	}

	// Update link settings
	profileSettings.Link = settings

	return s.saveActiveProfileSettings(ctx, logger, profileSettings)
}

// ============================================================================
// Cable Test Settings Handlers (fixes #740)
// ============================================================================

// handleCableTestSettings handles GET/PUT for /api/settings/cable.
// Cable test settings control TDR cable diagnostics behavior.
func (s *Server) handleCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getCableTestSettings(w, r)
	case http.MethodPut:
		s.updateCableTestSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
	}
}

// getCableTestSettings returns current cable test settings from the active profile.
func (s *Server) getCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Default settings
	settings := config.ProfileCableTestSettings{
		Enabled: true,
	}

	if s.db != nil {
		profileSettings, err := s.getActiveProfileSettings(ctx)
		if err == nil && profileSettings != nil {
			settings = profileSettings.CableTest
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, settings)
}

// updateCableTestSettings updates cable test settings in the active profile.
func (s *Server) updateCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var updates config.ProfileCableTestSettings
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		logger.WarnContext(ctx, "Invalid cable test settings request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Invalid request body",
			"",
		)
		return
	}

	// Save to active profile
	if s.db != nil {
		if err := s.updateActiveProfileCableTestSettings(ctx, logger, updates); err != nil {
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusInternalServerError,
				ErrCodeInternal,
				"Failed to save cable test settings",
				"",
			)
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

// updateActiveProfileCableTestSettings saves cable test settings to the active profile.
func (s *Server) updateActiveProfileCableTestSettings(
	ctx context.Context,
	logger *slog.Logger,
	settings config.ProfileCableTestSettings,
) error {
	profileSettings, err := s.getActiveProfileSettings(ctx)
	if err != nil {
		logger.WarnContext(ctx, "Failed to get active profile settings", "error", err)
		return err
	}

	// Update cable test settings
	profileSettings.CableTest = settings

	return s.saveActiveProfileSettings(ctx, logger, profileSettings)
}

// ============================================================================
// Profile Settings Helpers
// ============================================================================

// getActiveProfileSettings retrieves settings from the active profile.
func (s *Server) getActiveProfileSettings(ctx context.Context) (*config.ProfileSettings, error) {
	// Get active profile ID
	activeID, err := s.db.Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil || activeID == "" {
		// Try to get default profile
		defaultProfile, getErr := s.db.Profiles().GetDefault(ctx)
		if getErr != nil {
			return nil, getErr
		}
		activeID = defaultProfile.ID
	}

	// Get the profile
	profile, err := s.db.Profiles().Get(ctx, activeID)
	if err != nil {
		return nil, err
	}

	// Parse settings
	if profile.ConfigJSON == "" {
		return config.NewProfileSettings(), nil
	}

	return config.ParseProfileSettings(profile.ConfigJSON)
}

// saveActiveProfileSettings saves settings to the active profile.
func (s *Server) saveActiveProfileSettings(
	ctx context.Context,
	logger *slog.Logger,
	settings *config.ProfileSettings,
) error {
	// Get active profile ID
	activeID, err := s.db.Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil || activeID == "" {
		// Try to get default profile
		defaultProfile, getErr := s.db.Profiles().GetDefault(ctx)
		if getErr != nil {
			logger.DebugContext(ctx,
				"No active or default profile to save settings to",
				"reason",
				getErr.Error(),
			)
			return nil
		}
		activeID = defaultProfile.ID
	}

	// Get the profile
	profile, err := s.db.Profiles().Get(ctx, activeID)
	if err != nil {
		return err
	}

	// Serialize to JSON
	configJSON, err := settings.ToJSON()
	if err != nil {
		return err
	}

	// Update profile
	profile.ConfigJSON = configJSON
	if updateErr := s.db.Profiles().Update(ctx, profile); updateErr != nil {
		logger.ErrorContext(ctx,
			"Failed to save settings to profile",
			"error",
			updateErr,
			"profile_id",
			profile.ID,
		)
		return updateErr
	}

	logger.DebugContext(ctx,
		"Saved settings to active profile",
		"profile_id",
		profile.ID,
		"profile_name",
		profile.Name,
	)
	return nil
}
