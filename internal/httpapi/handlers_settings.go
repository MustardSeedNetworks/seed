package httpapi

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
	if s.db() != nil {
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
	activeID, err := s.db().Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil || activeID == "" {
		// No active profile, try to get default
		defaultProfile, getDefaultErr := s.db().Profiles().GetDefault(ctx)
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
	profile, err := s.db().Profiles().Get(ctx, activeID)
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

	// Export current settings from config using single source of truth
	configJSON, jsonErr := s.config.ToProfileJSON()
	if jsonErr != nil {
		logger.WarnContext(ctx, "Failed to serialize profile settings", "error", jsonErr)
		return nil
	}

	// Update profile
	profile.ConfigJSON = configJSON
	if updateErr := s.db().Profiles().Update(ctx, profile); updateErr != nil {
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

// thresholdPair holds warning and critical threshold pointers for updates.
type thresholdPair struct {
	warning  *time.Duration
	critical *time.Duration
}

// applyThresholdPair extracts good/warning values and applies them to threshold pointers.
// Returns error if the threshold object or values have invalid types.
func applyThresholdPair(data map[string]any, key, prefix string, pair thresholdPair) error {
	val, exists := data[key]
	if !exists {
		return nil
	}
	threshMap, ok := val.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", prefix)
	}

	if goodVal, goodExists := threshMap["good"]; goodExists {
		good, goodOK := goodVal.(float64)
		if !goodOK {
			return fmt.Errorf("%s.good must be a number", prefix)
		}
		*pair.warning = time.Duration(good) * time.Millisecond
	}

	if warnVal, warnExists := threshMap["warning"]; warnExists {
		warn, warnOK := warnVal.(float64)
		if !warnOK {
			return fmt.Errorf("%s.warning must be a number", prefix)
		}
		*pair.critical = time.Duration(warn) * time.Millisecond
	}

	return nil
}

// applyCustomTestThresholds applies custom test threshold updates.
// Returns error if any custom test key exists but has invalid type (fixes #784, G3).
func applyCustomTestThresholds(thresholds map[string]any, cfg *config.Config) error {
	if err := applyThresholdPair(thresholds, "customPing", "thresholds.customPing", thresholdPair{
		warning:  &cfg.Thresholds.CustomTests.Ping.Warning,
		critical: &cfg.Thresholds.CustomTests.Ping.Critical,
	}); err != nil {
		return err
	}

	if err := applyThresholdPair(thresholds, "customTcp", "thresholds.customTcp", thresholdPair{
		warning:  &cfg.Thresholds.CustomTests.TCP.Warning,
		critical: &cfg.Thresholds.CustomTests.TCP.Critical,
	}); err != nil {
		return err
	}

	return applyThresholdPair(thresholds, "customHttp", "thresholds.customHttp", thresholdPair{
		warning:  &cfg.Thresholds.CustomTests.HTTP.Warning,
		critical: &cfg.Thresholds.CustomTests.HTTP.Critical,
	})
}

// httpTimingThreshold represents a single HTTP timing threshold with Warning and Critical values.
type httpTimingThreshold struct {
	Warning  *time.Duration
	Critical *time.Duration
}

// parseHTTPTimingThreshold extracts good/warning values from a timing object.
// Returns (result, true, nil) if found, (nil, false, nil) if not found,
// or (nil, false, error) if the timing key exists but has invalid type.
func parseHTTPTimingThreshold(httpTimings map[string]any, key string) (*httpTimingThreshold, bool, error) {
	val, exists := httpTimings[key]
	if !exists {
		return nil, false, nil
	}

	timingObj, ok := val.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("thresholds.httpTimings.%s must be an object", key)
	}

	result := &httpTimingThreshold{}

	good, found, err := extractDurationField(timingObj, "good", key)
	if err != nil {
		return nil, false, err
	}
	if found {
		result.Warning = good
	}

	warning, found, err := extractDurationField(timingObj, "warning", key)
	if err != nil {
		return nil, false, err
	}
	if found {
		result.Critical = warning
	}

	return result, true, nil
}

// extractDurationField extracts a duration field from a timing object.
// Returns (duration, true, nil) if found, (nil, false, nil) if not found,
// or (nil, false, error) if the field has invalid type.
func extractDurationField(obj map[string]any, field, parentKey string) (*time.Duration, bool, error) {
	val, exists := obj[field]
	if !exists {
		return nil, false, nil
	}

	num, ok := val.(float64)
	if !ok {
		return nil, false, fmt.Errorf("thresholds.httpTimings.%s.%s must be a number", parentKey, field)
	}

	d := time.Duration(num) * time.Millisecond
	return &d, true, nil
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

	if err := applyDNSTimingThreshold(httpTimings, cfg); err != nil {
		return err
	}
	if err := applyTCPTimingThreshold(httpTimings, cfg); err != nil {
		return err
	}
	if err := applyTLSTimingThreshold(httpTimings, cfg); err != nil {
		return err
	}
	return applyTTFBTimingThreshold(httpTimings, cfg)
}

// applyDNSTimingThreshold applies DNS timing threshold updates.
func applyDNSTimingThreshold(httpTimings map[string]any, cfg *config.Config) error {
	threshold, found, err := parseHTTPTimingThreshold(httpTimings, "dns")
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if threshold.Warning != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning = *threshold.Warning
	}
	if threshold.Critical != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical = *threshold.Critical
	}
	return nil
}

// applyTCPTimingThreshold applies TCP timing threshold updates.
func applyTCPTimingThreshold(httpTimings map[string]any, cfg *config.Config) error {
	threshold, found, err := parseHTTPTimingThreshold(httpTimings, "tcp")
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if threshold.Warning != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning = *threshold.Warning
	}
	if threshold.Critical != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical = *threshold.Critical
	}
	return nil
}

// applyTLSTimingThreshold applies TLS timing threshold updates.
func applyTLSTimingThreshold(httpTimings map[string]any, cfg *config.Config) error {
	threshold, found, err := parseHTTPTimingThreshold(httpTimings, "tls")
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if threshold.Warning != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning = *threshold.Warning
	}
	if threshold.Critical != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical = *threshold.Critical
	}
	return nil
}

// applyTTFBTimingThreshold applies TTFB timing threshold updates.
func applyTTFBTimingThreshold(httpTimings map[string]any, cfg *config.Config) error {
	threshold, found, err := parseHTTPTimingThreshold(httpTimings, "ttfb")
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if threshold.Warning != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning = *threshold.Warning
	}
	if threshold.Critical != nil {
		cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical = *threshold.Critical
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

// extractBool extracts a boolean value from a map, returning an error if the key exists but is not a bool.
func extractBool(data map[string]any, key, prefix string) (bool, bool, error) {
	val, exists := data[key]
	if !exists {
		return false, false, nil
	}
	b, ok := val.(bool)
	if !ok {
		return false, false, fmt.Errorf("%s.%s must be a boolean", prefix, key)
	}
	return b, true, nil
}

// extractString extracts a string value from a map, returning an error if the key exists but is not a string.
func extractString(data map[string]any, key, prefix string) (string, bool, error) {
	val, exists := data[key]
	if !exists {
		return "", false, nil
	}
	s, ok := val.(string)
	if !ok {
		return "", false, fmt.Errorf("%s.%s must be a string", prefix, key)
	}
	return s, true, nil
}

// extractInt extracts an integer from a float64 value in a map.
func extractInt(data map[string]any, key, prefix string) (int, bool, error) {
	val, exists := data[key]
	if !exists {
		return 0, false, nil
	}
	f, ok := val.(float64)
	if !ok {
		return 0, false, fmt.Errorf("%s.%s must be a number", prefix, key)
	}
	return int(f), true, nil
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

	return applyIperfFields(iperf, cfg)
}

// applyIperfFields applies individual iperf configuration fields.
func applyIperfFields(iperf map[string]any, cfg *config.Config) error {
	const prefix = "iperf"

	if autoRun, found, err := extractBool(iperf, "autoRunOnLink", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.AutoRunOnLink = autoRun
	}

	if server, found, err := extractString(iperf, "server", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Server = server
	}

	if port, found, err := extractInt(iperf, "port", prefix); err != nil {
		return err
	} else if found {
		if validationErr := validation.ValidatePort(port); validationErr != nil {
			return fmt.Errorf("iperf.port: %w", validationErr)
		}
		cfg.Iperf.Port = port
	}

	if proto, found, err := extractString(iperf, "protocol", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Protocol = proto
	}

	if dir, found, err := extractString(iperf, "direction", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Direction = dir
	}

	if dur, found, err := extractInt(iperf, "duration", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Duration = dur
	}

	if srvPort, found, err := extractInt(iperf, "serverPort", prefix); err != nil {
		return err
	} else if found && validation.ValidatePort(srvPort) == nil {
		cfg.Iperf.ServerPort = srvPort
	}

	if enable, found, err := extractBool(iperf, "enableServer", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.EnableServer = enable
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

	const prefix = "fabOptions"

	if err := applyFABRunOptions(fabOptions, prefix, cfg); err != nil {
		return err
	}

	return applyFABMiscOptions(fabOptions, prefix, cfg)
}

// applyFABRunOptions applies the "run*" boolean options for FAB.
func applyFABRunOptions(fabOptions map[string]any, prefix string, cfg *config.Config) error {
	// Define field mappings: key -> pointer to config field
	type boolField struct {
		key   string
		field *bool
	}

	fields := []boolField{
		{"runLink", &cfg.FABOptions.RunLink},
		{"runSwitch", &cfg.FABOptions.RunSwitch},
		{"runVLAN", &cfg.FABOptions.RunVLAN},
		{"runIPConfig", &cfg.FABOptions.RunIPConfig},
		{"runGateway", &cfg.FABOptions.RunGateway},
		{"runDNS", &cfg.FABOptions.RunDNS},
	}

	for _, f := range fields {
		if val, found, err := extractBool(fabOptions, f.key, prefix); err != nil {
			return err
		} else if found {
			*f.field = val
		}
	}

	return nil
}

// applyFABMiscOptions applies the remaining FAB boolean options.
func applyFABMiscOptions(fabOptions map[string]any, prefix string, cfg *config.Config) error {
	type boolField struct {
		key   string
		field *bool
	}

	fields := []boolField{
		{"runHealthChecks", &cfg.FABOptions.RunHealthChecks},
		{"runNetworkDiscovery", &cfg.FABOptions.RunNetworkDiscovery},
		{"runSpeedtest", &cfg.FABOptions.RunSpeedtest},
		{"runIperf", &cfg.FABOptions.RunIperf},
		{"runPerformance", &cfg.FABOptions.RunPerformance},
		{"autoScanOnLink", &cfg.FABOptions.AutoScanOnLink},
	}

	for _, f := range fields {
		if val, found, err := extractBool(fabOptions, f.key, prefix); err != nil {
			return err
		} else if found {
			*f.field = val
		}
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

// getLinkSettings returns current link settings from config.
func (s *Server) getLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	// Read directly from Config (single source of truth)
	s.config.RLock()
	settings := s.config.Link
	s.config.RUnlock()

	// Default to "auto" if not set
	if settings.Mode == "" {
		settings.Mode = "auto"
	}

	sendJSONResponse(w, logger, http.StatusOK, settings)
}

// updateLinkSettings updates link settings in config and saves to active profile.
func (s *Server) updateLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var updates config.LinkConfig
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

	// Update Config directly (single source of truth)
	s.config.Lock()
	s.config.Link = updates
	s.config.Unlock()

	// Save to active profile in database
	if s.db() != nil {
		if err := s.saveSettingsToActiveProfile(ctx, logger); err != nil {
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

// getCableTestSettings returns current cable test settings from config.
func (s *Server) getCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	// Read directly from Config (single source of truth)
	s.config.RLock()
	settings := s.config.CableTest
	s.config.RUnlock()

	sendJSONResponse(w, logger, http.StatusOK, settings)
}

// updateCableTestSettings updates cable test settings in config and saves to active profile.
func (s *Server) updateCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var updates config.CableTestConfig
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

	// Update Config directly (single source of truth)
	s.config.Lock()
	s.config.CableTest = updates
	s.config.Unlock()

	// Save to active profile in database
	if s.db() != nil {
		if err := s.saveSettingsToActiveProfile(ctx, logger); err != nil {
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
