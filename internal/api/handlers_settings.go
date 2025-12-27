// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
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
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
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
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "") // fixes #694, #699
	}
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	// Lock config for read access
	s.config.RLock()
	defer s.config.RUnlock()

	settings := map[string]interface{}{
		"interface": map[string]interface{}{
			"current":   s.config.Interface.Default,
			"available": []string{}, // Will be populated by network module
		},
		"vlan": map[string]interface{}{
			"enabled": s.config.VLAN.Enabled,
			"id":      s.config.VLAN.ID,
		},
		"ip": map[string]interface{}{
			"mode": s.config.IP.Mode,
		},
		"thresholds": map[string]interface{}{
			"dns": map[string]int64{
				"good":    s.config.Thresholds.DNS.Warning.Milliseconds(),
				"warning": s.config.Thresholds.DNS.Critical.Milliseconds(),
			},
			"gateway": map[string]int64{
				"good":    s.config.Thresholds.Ping.Warning.Milliseconds(),
				"warning": s.config.Thresholds.Ping.Critical.Milliseconds(),
			},
			"wifi": map[string]int{
				"good":    s.config.Thresholds.WiFi.Signal.Warning,
				"warning": s.config.Thresholds.WiFi.Signal.Critical,
			},
			"customPing": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.Ping.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.Ping.Critical.Milliseconds(),
			},
			"customTcp": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.TCP.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.TCP.Critical.Milliseconds(),
			},
			"customHttp": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.HTTP.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.HTTP.Critical.Milliseconds(),
			},
			"httpTimings": map[string]map[string]int64{
				"dns": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
				},
				"tcp": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
				},
				"tls": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
				},
				"ttfb": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
				},
			},
		},
		"healthChecks": map[string]interface{}{
			"runPerformance": true,
			"runSpeedtest":   true,
			"runIperf":       false, // iPerf disabled by default (requires server)
			"runDiscovery":   true,
		},
		"speedtest": map[string]interface{}{
			"serverId":      s.config.Speedtest.ServerID,
			"autoRunOnLink": true, // Always auto-run speedtest on link
		},
		"iperf": map[string]interface{}{
			"autoRunOnLink": s.config.Iperf.AutoRunOnLink,
			"server":        s.config.Iperf.Server,
			"port":          s.config.Iperf.Port,
			"protocol":      s.config.Iperf.Protocol,
			"direction":     s.config.Iperf.Direction,
			"duration":      s.config.Iperf.Duration,
			"serverPort":    s.config.Iperf.ServerPort,
			"enableServer":  s.config.Iperf.EnableServer,
		},
		// Card visibility settings - all cards visible by default
		// Card visibility is managed entirely by the frontend via DEFAULT_CARD_SETTINGS
		"cardSettings": map[string]interface{}{
			"link": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"switch": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"vlan": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"network": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"gateway": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"dns": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"healthChecks": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"networkDiscovery": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
			},
			"performance": map[string]interface{}{
				"visible":       true,
				"autoRunOnLink": true,
				"speedtest": map[string]interface{}{
					"enabled":       true,
					"autoRunOnLink": true,
				},
				"iperf": map[string]interface{}{
					"enabled":       false, // iPerf disabled by default (requires server)
					"autoRunOnLink": false,
				},
			},
		},
		"displayOptions": map[string]interface{}{
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

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", "")
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
		logger.Warn("Invalid settings format", "errors", applyErrors)
		errMsg := "Invalid settings format. Check server logs for details."
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation, errMsg, "") // fixes #H7
		return
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Failed to save config", "")
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
			logger.Debug("No active or default profile to save settings to", "reason", getDefaultErr.Error())
			return nil
		}
		activeID = defaultProfile.ID
	}

	// Get the profile
	profile, err := s.db.Profiles().Get(ctx, activeID)
	if err != nil {
		logger.Warn("Failed to get active profile for settings save", "error", err, "profile_id", activeID)
		return nil
	}

	// Extract current settings from config
	profileSettings := config.NewProfileSettings()
	profileSettings.FromConfig(s.config)

	// Preserve existing notes if any
	if profile.ConfigJSON != "" {
		existingSettings, err := config.ParseProfileSettings(profile.ConfigJSON)
		if err == nil && existingSettings.Notes != "" {
			profileSettings.Notes = existingSettings.Notes
		}
	}

	// Serialize to JSON
	configJSON, err := profileSettings.ToJSON()
	if err != nil {
		logger.Warn("Failed to serialize profile settings", "error", err)
		return nil
	}

	// Update profile
	profile.ConfigJSON = configJSON
	if err := s.db.Profiles().Update(ctx, profile); err != nil {
		logger.Error("Failed to save settings to profile", "error", err, "profile_id", profile.ID)
		return err
	}

	logger.Debug("Saved settings to active profile", "profile_id", profile.ID, "profile_name", profile.Name)
	return nil
}

// applyThresholdUpdates applies threshold configuration updates.
// Returns error if thresholds key exists but has invalid type (fixes #784).
func applyThresholdUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["thresholds"]
	if !exists {
		return nil // Field not provided - valid for partial updates
	}
	thresholds, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("thresholds must be an object")
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
//nolint:dupl // Similar pattern to other threshold functions - intentional for clarity
func applyDNSThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	val, exists := thresholds["dns"]
	if !exists {
		return nil
	}
	dnsThresh, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("thresholds.dns must be an object")
	}

	// Validate "good" field if present
	if goodVal, exists := dnsThresh["good"]; exists {
		good, ok := goodVal.(float64)
		if !ok {
			return fmt.Errorf("thresholds.dns.good must be a number")
		}
		cfg.Thresholds.DNS.Warning = time.Duration(good) * time.Millisecond
	}

	// Validate "warning" field if present
	if warningVal, exists := dnsThresh["warning"]; exists {
		warning, ok := warningVal.(float64)
		if !ok {
			return fmt.Errorf("thresholds.dns.warning must be a number")
		}
		cfg.Thresholds.DNS.Critical = time.Duration(warning) * time.Millisecond
	}

	return nil
}

// applyGatewayThresholds applies gateway ping threshold updates.
// Returns error if gateway key exists but has invalid type (fixes #784, G3).
//
//nolint:dupl // Similar pattern to other threshold functions - intentional for clarity
func applyGatewayThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	val, exists := thresholds["gateway"]
	if !exists {
		return nil
	}
	gwThresh, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("thresholds.gateway must be an object")
	}

	// Validate "good" field if present
	if goodVal, exists := gwThresh["good"]; exists {
		good, ok := goodVal.(float64)
		if !ok {
			return fmt.Errorf("thresholds.gateway.good must be a number")
		}
		cfg.Thresholds.Ping.Warning = time.Duration(good) * time.Millisecond
	}

	// Validate "warning" field if present
	if warningVal, exists := gwThresh["warning"]; exists {
		warning, ok := warningVal.(float64)
		if !ok {
			return fmt.Errorf("thresholds.gateway.warning must be a number")
		}
		cfg.Thresholds.Ping.Critical = time.Duration(warning) * time.Millisecond
	}

	return nil
}

// applyWiFiThresholds applies WiFi signal threshold updates.
// Returns error if wifi key exists but has invalid type (fixes #784, G3).
func applyWiFiThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	val, exists := thresholds["wifi"]
	if !exists {
		return nil
	}
	wifi, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("thresholds.wifi must be an object")
	}

	// Validate "good" field if present
	if goodVal, exists := wifi["good"]; exists {
		good, ok := goodVal.(float64)
		if !ok {
			return fmt.Errorf("thresholds.wifi.good must be a number")
		}
		cfg.Thresholds.WiFi.Signal.Warning = int(good)
	}

	// Validate "warning" field if present
	if warningVal, exists := wifi["warning"]; exists {
		warning, ok := warningVal.(float64)
		if !ok {
			return fmt.Errorf("thresholds.wifi.warning must be a number")
		}
		cfg.Thresholds.WiFi.Signal.Critical = int(warning)
	}

	return nil
}

// applyCustomTestThresholds applies custom test threshold updates.
// Returns error if any custom test key exists but has invalid type (fixes #784, G3).
func applyCustomTestThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	// Custom ping thresholds
	if val, exists := thresholds["customPing"]; exists {
		customPing, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.customPing must be an object")
		}

		// Validate "good" field if present
		if goodVal, exists := customPing["good"]; exists {
			good, ok := goodVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.customPing.good must be a number")
			}
			cfg.Thresholds.CustomTests.Ping.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, exists := customPing["warning"]; exists {
			warning, ok := warningVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.customPing.warning must be a number")
			}
			cfg.Thresholds.CustomTests.Ping.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom TCP thresholds
	if val, exists := thresholds["customTcp"]; exists {
		customTCP, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.customTcp must be an object")
		}

		// Validate "good" field if present
		if goodVal, exists := customTCP["good"]; exists {
			good, ok := goodVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.customTcp.good must be a number")
			}
			cfg.Thresholds.CustomTests.TCP.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, exists := customTCP["warning"]; exists {
			warning, ok := warningVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.customTcp.warning must be a number")
			}
			cfg.Thresholds.CustomTests.TCP.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom HTTP thresholds
	if val, exists := thresholds["customHttp"]; exists {
		customHTTP, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.customHttp must be an object")
		}

		// Validate "good" field if present
		if goodVal, exists := customHTTP["good"]; exists {
			good, ok := goodVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.customHttp.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTP.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, exists := customHTTP["warning"]; exists {
			warning, ok := warningVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.customHttp.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTP.Critical = time.Duration(warning) * time.Millisecond
		}
	}
	return nil
}

// applyHTTPTimingThresholds applies HTTP timing threshold updates.
// Returns error if httpTimings key exists but has invalid type (fixes #784, G3).
func applyHTTPTimingThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	val, exists := thresholds["httpTimings"]
	if !exists {
		return nil
	}
	httpTimings, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("thresholds.httpTimings must be an object")
	}

	// DNS timing thresholds
	if dnsVal, exists := httpTimings["dns"]; exists {
		dnsT, ok := dnsVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.httpTimings.dns must be an object")
		}

		// Validate "good" field if present
		if goodVal, exists := dnsT["good"]; exists {
			good, ok := goodVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.dns.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, exists := dnsT["warning"]; exists {
			warning, ok := warningVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.dns.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// TCP timing thresholds
	if tcpVal, exists := httpTimings["tcp"]; exists {
		tcpT, ok := tcpVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.httpTimings.tcp must be an object")
		}

		// Validate "good" field if present
		if goodVal, exists := tcpT["good"]; exists {
			good, ok := goodVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.tcp.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, exists := tcpT["warning"]; exists {
			warning, ok := warningVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.tcp.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// TLS timing thresholds
	if tlsVal, exists := httpTimings["tls"]; exists {
		tlsT, ok := tlsVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.httpTimings.tls must be an object")
		}

		// Validate "good" field if present
		if goodVal, exists := tlsT["good"]; exists {
			good, ok := goodVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.tls.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, exists := tlsT["warning"]; exists {
			warning, ok := warningVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.tls.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// TTFB timing thresholds
	if ttfbVal, exists := httpTimings["ttfb"]; exists {
		ttfb, ok := ttfbVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.httpTimings.ttfb must be an object")
		}

		// Validate "good" field if present
		if goodVal, exists := ttfb["good"]; exists {
			good, ok := goodVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.ttfb.good must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning = time.Duration(good) * time.Millisecond
		}

		// Validate "warning" field if present
		if warningVal, exists := ttfb["warning"]; exists {
			warning, ok := warningVal.(float64)
			if !ok {
				return fmt.Errorf("thresholds.httpTimings.ttfb.warning must be a number")
			}
			cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical = time.Duration(warning) * time.Millisecond
		}
	}
	return nil
}

// applyHealthChecksUpdates applies health check toggle updates.
// Returns error if healthChecks key exists but has invalid type (fixes #784, G3).
func applyHealthChecksUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["healthChecks"]
	if !exists {
		return nil
	}
	healthChecks, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("healthChecks must be an object")
	}

	if val, exists := healthChecks["runPerformance"]; exists {
		runPerformance, ok := val.(bool)
		if !ok {
			return fmt.Errorf("healthChecks.runPerformance must be a boolean")
		}
		cfg.HealthChecks.RunPerformance = runPerformance
	}

	if val, exists := healthChecks["runSpeedtest"]; exists {
		runSpeedtest, ok := val.(bool)
		if !ok {
			return fmt.Errorf("healthChecks.runSpeedtest must be a boolean")
		}
		cfg.HealthChecks.RunSpeedtest = runSpeedtest
	}

	if val, exists := healthChecks["runIperf"]; exists {
		runIperf, ok := val.(bool)
		if !ok {
			return fmt.Errorf("healthChecks.runIperf must be a boolean")
		}
		cfg.HealthChecks.RunIperf = runIperf
	}

	if val, exists := healthChecks["runDiscovery"]; exists {
		runDiscovery, ok := val.(bool)
		if !ok {
			return fmt.Errorf("healthChecks.runDiscovery must be a boolean")
		}
		cfg.HealthChecks.RunDiscovery = runDiscovery
	}

	return nil
}

// applySpeedtestUpdates applies speedtest configuration updates.
// Returns error if speedtest key exists but has invalid type (fixes #784, G3).
func applySpeedtestUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["speedtest"]
	if !exists {
		return nil
	}
	speedtest, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("speedtest must be an object")
	}

	if val, exists := speedtest["serverId"]; exists {
		serverID, ok := val.(string)
		if !ok {
			return fmt.Errorf("speedtest.serverId must be a string")
		}
		cfg.Speedtest.ServerID = serverID
	}

	if val, exists := speedtest["autoRunOnLink"]; exists {
		autoRunOnLink, ok := val.(bool)
		if !ok {
			return fmt.Errorf("speedtest.autoRunOnLink must be a boolean")
		}
		cfg.Speedtest.AutoRunOnLink = autoRunOnLink
	}

	return nil
}

// applyIperfUpdates applies iperf configuration updates.
// Returns error if iperf key exists but has invalid type (fixes #784, G3).
func applyIperfUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["iperf"]
	if !exists {
		return nil
	}
	iperf, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("iperf must be an object")
	}

	if val, exists := iperf["autoRunOnLink"]; exists {
		autoRunOnLink, ok := val.(bool)
		if !ok {
			return fmt.Errorf("iperf.autoRunOnLink must be a boolean")
		}
		cfg.Iperf.AutoRunOnLink = autoRunOnLink
	}

	if val, exists := iperf["server"]; exists {
		server, ok := val.(string)
		if !ok {
			return fmt.Errorf("iperf.server must be a string")
		}
		cfg.Iperf.Server = server
	}

	if val, exists := iperf["port"]; exists {
		port, ok := val.(float64)
		if !ok {
			return fmt.Errorf("iperf.port must be a number")
		}
		p := int(port)
		if err := validation.ValidatePort(p); err != nil {
			return fmt.Errorf("iperf.port: %w", err)
		}
		cfg.Iperf.Port = p
	}

	if val, exists := iperf["protocol"]; exists {
		protocol, ok := val.(string)
		if !ok {
			return fmt.Errorf("iperf.protocol must be a string")
		}
		cfg.Iperf.Protocol = protocol
	}

	if val, exists := iperf["direction"]; exists {
		direction, ok := val.(string)
		if !ok {
			return fmt.Errorf("iperf.direction must be a string")
		}
		cfg.Iperf.Direction = direction
	}

	if val, exists := iperf["duration"]; exists {
		duration, ok := val.(float64)
		if !ok {
			return fmt.Errorf("iperf.duration must be a number")
		}
		cfg.Iperf.Duration = int(duration)
	}

	if val, exists := iperf["serverPort"]; exists {
		serverPort, ok := val.(float64)
		if !ok {
			return fmt.Errorf("iperf.serverPort must be a number")
		}
		p := int(serverPort)
		if validation.ValidatePort(p) == nil {
			cfg.Iperf.ServerPort = p
		}
	}

	if val, exists := iperf["enableServer"]; exists {
		enableServer, ok := val.(bool)
		if !ok {
			return fmt.Errorf("iperf.enableServer must be a boolean")
		}
		cfg.Iperf.EnableServer = enableServer
	}

	return nil
}

// applyFABOptionsUpdates applies FAB options updates.
// Returns error if fabOptions key exists but has invalid type (fixes #784, G3).
func applyFABOptionsUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["fabOptions"]
	if !exists {
		return nil
	}
	fabOptions, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("fabOptions must be an object")
	}

	if val, exists := fabOptions["runLink"]; exists {
		runLink, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runLink must be a boolean")
		}
		cfg.FABOptions.RunLink = runLink
	}

	if val, exists := fabOptions["runSwitch"]; exists {
		runSwitch, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runSwitch must be a boolean")
		}
		cfg.FABOptions.RunSwitch = runSwitch
	}

	if val, exists := fabOptions["runVLAN"]; exists {
		runVLAN, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runVLAN must be a boolean")
		}
		cfg.FABOptions.RunVLAN = runVLAN
	}

	if val, exists := fabOptions["runIPConfig"]; exists {
		runIPConfig, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runIPConfig must be a boolean")
		}
		cfg.FABOptions.RunIPConfig = runIPConfig
	}

	if val, exists := fabOptions["runGateway"]; exists {
		runGateway, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runGateway must be a boolean")
		}
		cfg.FABOptions.RunGateway = runGateway
	}

	if val, exists := fabOptions["runDNS"]; exists {
		runDNS, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runDNS must be a boolean")
		}
		cfg.FABOptions.RunDNS = runDNS
	}

	if val, exists := fabOptions["runHealthChecks"]; exists {
		runHealthChecks, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runHealthChecks must be a boolean")
		}
		cfg.FABOptions.RunHealthChecks = runHealthChecks
	}

	if val, exists := fabOptions["runNetworkDiscovery"]; exists {
		runNetworkDiscovery, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runNetworkDiscovery must be a boolean")
		}
		cfg.FABOptions.RunNetworkDiscovery = runNetworkDiscovery
	}

	if val, exists := fabOptions["runSpeedtest"]; exists {
		runSpeedtest, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runSpeedtest must be a boolean")
		}
		cfg.FABOptions.RunSpeedtest = runSpeedtest
	}

	if val, exists := fabOptions["runIperf"]; exists {
		runIperf, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runIperf must be a boolean")
		}
		cfg.FABOptions.RunIperf = runIperf
	}

	if val, exists := fabOptions["runPerformance"]; exists {
		runPerformance, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.runPerformance must be a boolean")
		}
		cfg.FABOptions.RunPerformance = runPerformance
	}

	if val, exists := fabOptions["autoScanOnLink"]; exists {
		autoScanOnLink, ok := val.(bool)
		if !ok {
			return fmt.Errorf("fabOptions.autoScanOnLink must be a boolean")
		}
		cfg.FABOptions.AutoScanOnLink = autoScanOnLink
	}

	return nil
}

// applyDisplayOptionsUpdates applies display options updates.
// Returns error if displayOptions key exists but has invalid type (fixes #784, G3).
func applyDisplayOptionsUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["displayOptions"]
	if !exists {
		return nil
	}
	displayOptions, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("displayOptions must be an object")
	}

	if val, exists := displayOptions["showPublicIP"]; exists {
		showPublicIP, ok := val.(bool)
		if !ok {
			return fmt.Errorf("displayOptions.showPublicIP must be a boolean")
		}
		cfg.DisplayOptions.ShowPublicIP = showPublicIP
	}

	if val, exists := displayOptions["unitSystem"]; exists {
		unitSystem, ok := val.(string)
		if !ok {
			return fmt.Errorf("displayOptions.unitSystem must be a string")
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
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
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
		logger.Warn("Invalid link settings request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", "")
		return
	}

	// Validate mode value (combined speed/duplex format like "100/full" or "auto")
	validModes := map[string]bool{
		"auto": true, "10/half": true, "10/full": true, "100/half": true, "100/full": true,
		"1000/full": true, "2500/full": true, "5000/full": true, "10000/full": true,
	}
	if updates.Mode != "" && !validModes[updates.Mode] {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation, "Invalid mode value", "")
		return
	}

	// Save to active profile
	if s.db != nil {
		if err := s.updateActiveProfileLinkSettings(ctx, logger, updates); err != nil {
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Failed to save link settings", "")
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

// updateActiveProfileLinkSettings saves link settings to the active profile.
func (s *Server) updateActiveProfileLinkSettings(ctx context.Context, logger *slog.Logger, settings config.ProfileLinkSettings) error {
	profileSettings, err := s.getActiveProfileSettings(ctx)
	if err != nil {
		logger.Warn("Failed to get active profile settings", "error", err)
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
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
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
		logger.Warn("Invalid cable test settings request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", "")
		return
	}

	// Save to active profile
	if s.db != nil {
		if err := s.updateActiveProfileCableTestSettings(ctx, logger, updates); err != nil {
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Failed to save cable test settings", "")
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

// updateActiveProfileCableTestSettings saves cable test settings to the active profile.
func (s *Server) updateActiveProfileCableTestSettings(ctx context.Context, logger *slog.Logger, settings config.ProfileCableTestSettings) error {
	profileSettings, err := s.getActiveProfileSettings(ctx)
	if err != nil {
		logger.Warn("Failed to get active profile settings", "error", err)
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
func (s *Server) saveActiveProfileSettings(ctx context.Context, logger *slog.Logger, settings *config.ProfileSettings) error {
	// Get active profile ID
	activeID, err := s.db.Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil || activeID == "" {
		// Try to get default profile
		defaultProfile, getErr := s.db.Profiles().GetDefault(ctx)
		if getErr != nil {
			logger.Debug("No active or default profile to save settings to", "reason", getErr.Error())
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
	if err := s.db.Profiles().Update(ctx, profile); err != nil {
		logger.Error("Failed to save settings to profile", "error", err, "profile_id", profile.ID)
		return err
	}

	logger.Debug("Saved settings to active profile", "profile_id", profile.ID, "profile_name", profile.Name)
	return nil
}
