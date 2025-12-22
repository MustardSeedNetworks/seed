// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
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
			"runPerformance": s.config.HealthChecks.RunPerformance,
			"runSpeedtest":   s.config.HealthChecks.RunSpeedtest,
			"runIperf":       s.config.HealthChecks.RunIperf,
			"runDiscovery":   s.config.HealthChecks.RunDiscovery,
		},
		"speedtest": map[string]interface{}{
			"serverId":      s.config.Speedtest.ServerID,
			"autoRunOnLink": s.config.Speedtest.AutoRunOnLink,
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
		// Card visibility and auto-run settings (renamed from fabOptions for clarity)
		"cardSettings": map[string]interface{}{
			"link": map[string]interface{}{
				"visible":       s.config.FABOptions.RunLink,
				"autoRunOnLink": s.config.FABOptions.RunLink,
			},
			"switch": map[string]interface{}{
				"visible":       s.config.FABOptions.RunSwitch,
				"autoRunOnLink": s.config.FABOptions.RunSwitch,
			},
			"vlan": map[string]interface{}{
				"visible":       s.config.FABOptions.RunVLAN,
				"autoRunOnLink": s.config.FABOptions.RunVLAN,
			},
			"network": map[string]interface{}{
				"visible":       s.config.FABOptions.RunIPConfig,
				"autoRunOnLink": s.config.FABOptions.RunIPConfig,
			},
			"gateway": map[string]interface{}{
				"visible":       s.config.FABOptions.RunGateway,
				"autoRunOnLink": s.config.FABOptions.RunGateway,
			},
			"dns": map[string]interface{}{
				"visible":       s.config.FABOptions.RunDNS,
				"autoRunOnLink": s.config.FABOptions.RunDNS,
			},
			"healthChecks": map[string]interface{}{
				"visible":       s.config.FABOptions.RunHealthChecks,
				"autoRunOnLink": s.config.FABOptions.RunHealthChecks,
			},
			"networkDiscovery": map[string]interface{}{
				"visible":       s.config.FABOptions.RunNetworkDiscovery,
				"autoRunOnLink": s.config.FABOptions.AutoScanOnLink,
			},
			"performance": map[string]interface{}{
				"visible":       s.config.FABOptions.RunPerformance,
				"autoRunOnLink": s.config.FABOptions.RunPerformance,
				"speedtest": map[string]interface{}{
					"enabled":       s.config.FABOptions.RunSpeedtest,
					"autoRunOnLink": s.config.FABOptions.RunSpeedtest,
				},
				"iperf": map[string]interface{}{
					"enabled":       s.config.FABOptions.RunIperf,
					"autoRunOnLink": s.config.FABOptions.RunIperf,
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
	s.config.Lock()
	defer s.config.Unlock()

	// Apply updates using helper functions
	applyThresholdUpdates(updates, s.config)
	applyHealthChecksUpdates(updates, s.config)
	applySpeedtestUpdates(updates, s.config)
	applyIperfUpdates(updates, s.config)
	applyFABOptionsUpdates(updates, s.config)
	applyDisplayOptionsUpdates(updates, s.config)

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Failed to save config", err.Error()) // fixes #694, #699
		return
	}

	// Also save settings to the active profile (fixes #781)
	if s.db != nil {
		s.saveSettingsToActiveProfile(ctx, logger)
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

// saveSettingsToActiveProfile saves current settings to the active profile's ConfigJSON.
// This ensures profile-specific settings are persisted (fixes #781).
func (s *Server) saveSettingsToActiveProfile(ctx context.Context, logger *slog.Logger) {
	// Get active profile ID
	activeID, err := s.db.Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil || activeID == "" {
		// No active profile, try to get default
		defaultProfile, err := s.db.Profiles().GetDefault(ctx)
		if err != nil {
			logger.Debug("No active or default profile to save settings to")
			return
		}
		activeID = defaultProfile.ID
	}

	// Get the profile
	profile, err := s.db.Profiles().Get(ctx, activeID)
	if err != nil {
		logger.Warn("Failed to get active profile for settings save", "error", err, "profile_id", activeID)
		return
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
		return
	}

	// Update profile
	profile.ConfigJSON = configJSON
	if err := s.db.Profiles().Update(ctx, profile); err != nil {
		logger.Warn("Failed to save settings to profile", "error", err, "profile_id", profile.ID)
		return
	}

	logger.Debug("Saved settings to active profile", "profile_id", profile.ID, "profile_name", profile.Name)
}

// applyThresholdUpdates applies threshold configuration updates.
func applyThresholdUpdates(updates map[string]interface{}, cfg *config.Config) {
	thresholds, ok := updates["thresholds"].(map[string]interface{})
	if !ok {
		return
	}

	applyDNSThresholds(thresholds, cfg)
	applyGatewayThresholds(thresholds, cfg)
	applyWiFiThresholds(thresholds, cfg)
	applyCustomTestThresholds(thresholds, cfg)
	applyHTTPTimingThresholds(thresholds, cfg)
}

// applyDNSThresholds applies DNS threshold updates.
func applyDNSThresholds(thresholds map[string]interface{}, cfg *config.Config) {
	dnsThresh, ok := thresholds["dns"].(map[string]interface{})
	if !ok {
		return
	}
	if good, ok := dnsThresh["good"].(float64); ok {
		cfg.Thresholds.DNS.Warning = time.Duration(good) * time.Millisecond
	}
	if warning, ok := dnsThresh["warning"].(float64); ok {
		cfg.Thresholds.DNS.Critical = time.Duration(warning) * time.Millisecond
	}
}

// applyGatewayThresholds applies gateway ping threshold updates.
func applyGatewayThresholds(thresholds map[string]interface{}, cfg *config.Config) {
	gwThresh, ok := thresholds["gateway"].(map[string]interface{})
	if !ok {
		return
	}
	if good, ok := gwThresh["good"].(float64); ok {
		cfg.Thresholds.Ping.Warning = time.Duration(good) * time.Millisecond
	}
	if warning, ok := gwThresh["warning"].(float64); ok {
		cfg.Thresholds.Ping.Critical = time.Duration(warning) * time.Millisecond
	}
}

// applyWiFiThresholds applies WiFi signal threshold updates.
func applyWiFiThresholds(thresholds map[string]interface{}, cfg *config.Config) {
	wifi, ok := thresholds["wifi"].(map[string]interface{})
	if !ok {
		return
	}
	if good, ok := wifi["good"].(float64); ok {
		cfg.Thresholds.WiFi.Signal.Warning = int(good)
	}
	if warning, ok := wifi["warning"].(float64); ok {
		cfg.Thresholds.WiFi.Signal.Critical = int(warning)
	}
}

// applyCustomTestThresholds applies custom test threshold updates.
func applyCustomTestThresholds(thresholds map[string]interface{}, cfg *config.Config) {
	// Custom ping thresholds
	if customPing, ok := thresholds["customPing"].(map[string]interface{}); ok {
		if good, ok := customPing["good"].(float64); ok {
			cfg.Thresholds.CustomTests.Ping.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := customPing["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.Ping.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom TCP thresholds
	if customTCP, ok := thresholds["customTcp"].(map[string]interface{}); ok {
		if good, ok := customTCP["good"].(float64); ok {
			cfg.Thresholds.CustomTests.TCP.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := customTCP["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.TCP.Critical = time.Duration(warning) * time.Millisecond
		}
	}

	// Custom HTTP thresholds
	if customHTTP, ok := thresholds["customHttp"].(map[string]interface{}); ok {
		if good, ok := customHTTP["good"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTP.Warning = time.Duration(good) * time.Millisecond
		}
		if warning, ok := customHTTP["warning"].(float64); ok {
			cfg.Thresholds.CustomTests.HTTP.Critical = time.Duration(warning) * time.Millisecond
		}
	}
}

// applyHTTPTimingThresholds applies HTTP timing threshold updates.
func applyHTTPTimingThresholds(thresholds map[string]interface{}, cfg *config.Config) {
	httpTimings, ok := thresholds["httpTimings"].(map[string]interface{})
	if !ok {
		return
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
}

// applyHealthChecksUpdates applies health check toggle updates.
func applyHealthChecksUpdates(updates map[string]interface{}, cfg *config.Config) {
	healthChecks, ok := updates["healthChecks"].(map[string]interface{})
	if !ok {
		return
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
}

// applySpeedtestUpdates applies speedtest configuration updates.
func applySpeedtestUpdates(updates map[string]interface{}, cfg *config.Config) {
	speedtest, ok := updates["speedtest"].(map[string]interface{})
	if !ok {
		return
	}
	if serverID, ok := speedtest["serverId"].(string); ok {
		cfg.Speedtest.ServerID = serverID
	}
	if autoRunOnLink, ok := speedtest["autoRunOnLink"].(bool); ok {
		cfg.Speedtest.AutoRunOnLink = autoRunOnLink
	}
}

// applyIperfUpdates applies iperf configuration updates.
func applyIperfUpdates(updates map[string]interface{}, cfg *config.Config) {
	iperf, ok := updates["iperf"].(map[string]interface{})
	if !ok {
		return
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
}

// applyFABOptionsUpdates applies FAB options updates.
func applyFABOptionsUpdates(updates map[string]interface{}, cfg *config.Config) {
	fabOptions, ok := updates["fabOptions"].(map[string]interface{})
	if !ok {
		return
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
}

// applyDisplayOptionsUpdates applies display options updates.
func applyDisplayOptionsUpdates(updates map[string]interface{}, cfg *config.Config) {
	displayOptions, ok := updates["displayOptions"].(map[string]interface{})
	if !ok {
		return
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
}
