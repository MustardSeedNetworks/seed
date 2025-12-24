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
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", err.Error()) // fixes #694, #699
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
		errMsg := "Invalid settings format: "
		for i, err := range applyErrors {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.Error()
		}
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation, errMsg, "")
		return
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Failed to save config", err.Error()) // fixes #694, #699
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
			logger.Debug("No active or default profile to save settings to")
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
// Returns error if dns key exists but has invalid type (fixes #784).
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
	if good, ok := dnsThresh["good"].(float64); ok {
		cfg.Thresholds.DNS.Warning = time.Duration(good) * time.Millisecond
	}
	if warning, ok := dnsThresh["warning"].(float64); ok {
		cfg.Thresholds.DNS.Critical = time.Duration(warning) * time.Millisecond
	}
	return nil
}

// applyGatewayThresholds applies gateway ping threshold updates.
// Returns error if gateway key exists but has invalid type (fixes #784).
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
	if good, ok := gwThresh["good"].(float64); ok {
		cfg.Thresholds.Ping.Warning = time.Duration(good) * time.Millisecond
	}
	if warning, ok := gwThresh["warning"].(float64); ok {
		cfg.Thresholds.Ping.Critical = time.Duration(warning) * time.Millisecond
	}
	return nil
}

// applyWiFiThresholds applies WiFi signal threshold updates.
// Returns error if wifi key exists but has invalid type (fixes #784).
func applyWiFiThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	val, exists := thresholds["wifi"]
	if !exists {
		return nil
	}
	wifi, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("thresholds.wifi must be an object")
	}
	if good, ok := wifi["good"].(float64); ok {
		cfg.Thresholds.WiFi.Signal.Warning = int(good)
	}
	if warning, ok := wifi["warning"].(float64); ok {
		cfg.Thresholds.WiFi.Signal.Critical = int(warning)
	}
	return nil
}

// applyCustomTestThresholds applies custom test threshold updates.
// Returns error if any custom test key exists but has invalid type (fixes #784).
func applyCustomTestThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	// Custom ping thresholds
	if val, exists := thresholds["customPing"]; exists {
		customPing, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.customPing must be an object")
		}
		if good, ok := customPing["good"].(float64); ok {
			cfg.Thresholds.CustomTests.Ping.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := customPing["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.Ping.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom TCP thresholds
	if val, exists := thresholds["customTcp"]; exists {
		customTCP, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.customTcp must be an object")
		}
		if good, ok := customTCP["good"].(float64); ok {
			cfg.Thresholds.CustomTests.TCP.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := customTCP["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.TCP.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom HTTP thresholds
	if val, exists := thresholds["customHttp"]; exists {
		customHTTP, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("thresholds.customHttp must be an object")
		}
		if good, ok := customHTTP["good"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTP.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := customHTTP["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTP.Critical = time.Duration(warning) * time.Millisecond
		}
	}
	return nil
}

// applyHTTPTimingThresholds applies HTTP timing threshold updates.
// Returns error if httpTimings key exists but has invalid type (fixes #784).
func applyHTTPTimingThresholds(thresholds map[string]interface{}, cfg *config.Config) error {
	val, exists := thresholds["httpTimings"]
	if !exists {
		return nil
	}
	httpTimings, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("thresholds.httpTimings must be an object")
	}

	if dnsT, ok := httpTimings["dns"].(map[string]interface{}); ok {
		if good, ok := dnsT["good"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := dnsT["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	if tcpT, ok := httpTimings["tcp"].(map[string]interface{}); ok {
		if good, ok := tcpT["good"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := tcpT["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	if tlsT, ok := httpTimings["tls"].(map[string]interface{}); ok {
		if good, ok := tlsT["good"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := tlsT["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	if ttfb, ok := httpTimings["ttfb"].(map[string]interface{}); ok {
		if good, ok := ttfb["good"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := ttfb["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical = time.Duration(warning) * time.Millisecond
		}
	}
	return nil
}

// applyHealthChecksUpdates applies health check toggle updates.
// Returns error if healthChecks key exists but has invalid type (fixes #784).
func applyHealthChecksUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["healthChecks"]
	if !exists {
		return nil
	}
	healthChecks, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("healthChecks must be an object")
	}
	if runPerformance, ok := healthChecks["runPerformance"].(bool); ok {
		cfg.HealthChecks.RunPerformance = runPerformance
	}
	if runSpeedtest, ok := healthChecks["runSpeedtest"].(bool); ok {
		cfg.HealthChecks.RunSpeedtest = runSpeedtest
	}
	if runIperf, ok := healthChecks["runIperf"].(bool); ok {
		cfg.HealthChecks.RunIperf = runIperf
	}
	if runDiscovery, ok := healthChecks["runDiscovery"].(bool); ok {
		cfg.HealthChecks.RunDiscovery = runDiscovery
	}
	return nil
}

// applySpeedtestUpdates applies speedtest configuration updates.
// Returns error if speedtest key exists but has invalid type (fixes #784).
func applySpeedtestUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["speedtest"]
	if !exists {
		return nil
	}
	speedtest, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("speedtest must be an object")
	}
	if serverID, ok := speedtest["serverId"].(string); ok {
		cfg.Speedtest.ServerID = serverID
	}
	if autoRunOnLink, ok := speedtest["autoRunOnLink"].(bool); ok {
		cfg.Speedtest.AutoRunOnLink = autoRunOnLink
	}
	return nil
}

// applyIperfUpdates applies iperf configuration updates.
// Returns error if iperf key exists but has invalid type (fixes #784).
func applyIperfUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["iperf"]
	if !exists {
		return nil
	}
	iperf, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("iperf must be an object")
	}
	if autoRunOnLink, ok := iperf["autoRunOnLink"].(bool); ok {
		cfg.Iperf.AutoRunOnLink = autoRunOnLink
	}
	if server, ok := iperf["server"].(string); ok {
		cfg.Iperf.Server = server
	}
	if port, ok := iperf["port"].(float64); ok {
		p := int(port)
		if validation.ValidatePort(p) == nil {
			cfg.Iperf.Port = p
		}
	}
	if protocol, ok := iperf["protocol"].(string); ok {
		cfg.Iperf.Protocol = protocol
	}
	if direction, ok := iperf["direction"].(string); ok {
		cfg.Iperf.Direction = direction
	}
	if duration, ok := iperf["duration"].(float64); ok {
		cfg.Iperf.Duration = int(duration)
	}
	if serverPort, ok := iperf["serverPort"].(float64); ok {
		p := int(serverPort)
		if validation.ValidatePort(p) == nil {
			cfg.Iperf.ServerPort = p
		}
	}
	if enableServer, ok := iperf["enableServer"].(bool); ok {
		cfg.Iperf.EnableServer = enableServer
	}
	return nil
}

// applyFABOptionsUpdates applies FAB options updates.
// Returns error if fabOptions key exists but has invalid type (fixes #784).
func applyFABOptionsUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["fabOptions"]
	if !exists {
		return nil
	}
	fabOptions, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("fabOptions must be an object")
	}
	if runLink, ok := fabOptions["runLink"].(bool); ok {
		cfg.FABOptions.RunLink = runLink
	}
	if runSwitch, ok := fabOptions["runSwitch"].(bool); ok {
		cfg.FABOptions.RunSwitch = runSwitch
	}
	if runVLAN, ok := fabOptions["runVLAN"].(bool); ok {
		cfg.FABOptions.RunVLAN = runVLAN
	}
	if runIPConfig, ok := fabOptions["runIPConfig"].(bool); ok {
		cfg.FABOptions.RunIPConfig = runIPConfig
	}
	if runGateway, ok := fabOptions["runGateway"].(bool); ok {
		cfg.FABOptions.RunGateway = runGateway
	}
	if runDNS, ok := fabOptions["runDNS"].(bool); ok {
		cfg.FABOptions.RunDNS = runDNS
	}
	if runHealthChecks, ok := fabOptions["runHealthChecks"].(bool); ok {
		cfg.FABOptions.RunHealthChecks = runHealthChecks
	}
	if runNetworkDiscovery, ok := fabOptions["runNetworkDiscovery"].(bool); ok {
		cfg.FABOptions.RunNetworkDiscovery = runNetworkDiscovery
	}
	if runSpeedtest, ok := fabOptions["runSpeedtest"].(bool); ok {
		cfg.FABOptions.RunSpeedtest = runSpeedtest
	}
	if runIperf, ok := fabOptions["runIperf"].(bool); ok {
		cfg.FABOptions.RunIperf = runIperf
	}
	if runPerformance, ok := fabOptions["runPerformance"].(bool); ok {
		cfg.FABOptions.RunPerformance = runPerformance
	}
	if autoScanOnLink, ok := fabOptions["autoScanOnLink"].(bool); ok {
		cfg.FABOptions.AutoScanOnLink = autoScanOnLink
	}
	return nil
}

// applyDisplayOptionsUpdates applies display options updates.
// Returns error if displayOptions key exists but has invalid type (fixes #784).
func applyDisplayOptionsUpdates(updates map[string]interface{}, cfg *config.Config) error {
	val, exists := updates["displayOptions"]
	if !exists {
		return nil
	}
	displayOptions, ok := val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("displayOptions must be an object")
	}
	if showPublicIP, ok := displayOptions["showPublicIP"].(bool); ok {
		cfg.DisplayOptions.ShowPublicIP = showPublicIP
	}
	if unitSystem, ok := displayOptions["unitSystem"].(string); ok {
		// Validate unit system (only "sae" or "metric" allowed)
		if unitSystem == "sae" || unitSystem == "metric" {
			cfg.DisplayOptions.UnitSystem = unitSystem
		}
	}
	return nil
}
