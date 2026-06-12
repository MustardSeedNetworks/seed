// Package management is the application service for the main settings endpoint
// (ADR-0020 clean-hexagonal, WS-A2). It owns the read/apply/persist logic the
// transport layer used to carry inline: the HTTP handler decodes the request,
// calls Get or Update here, and encodes the result. Persistence is reached
// through the consumer-defined Store port, satisfied by an adapter in the
// composition root (internal/app).
package management

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// Sentinel errors the transport layer maps to HTTP status codes.
var (
	// ErrConflict is returned when an If-Match ETag does not match the current
	// settings token (optimistic concurrency, HTTP 412).
	ErrConflict = errors.New("management: settings ETag mismatch")
	// ErrValidation is returned when one or more apply helpers reject the
	// update payload (HTTP 400).
	ErrValidation = errors.New("management: invalid update fields")
)

// Store reads and persists the main application settings. Read runs fn under
// the config read-lock; Write runs fn under the write-lock, then saves to
// disk if fn returns nil.
type Store interface {
	// Read calls fn with the live config held under the config RLock.
	Read(fn func(*config.Config))
	// Write calls fn with the live config held under the config Lock. If fn
	// returns nil, the config is saved to disk. The lock is released before
	// Save acquires its own RLock (fixes #783 deadlock pattern).
	Write(fn func(*config.Config) error) error
}

// Service is the main-settings application service.
type Service struct {
	store Store
}

// NewService builds the settings management service over its Store port.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Get returns the current settings map and the ETag header value. The map
// mirrors the getSettings read model verbatim (behavior-preserving).
func (s *Service) Get() (map[string]any, string) {
	var settings map[string]any
	var etag string
	s.store.Read(func(cfg *config.Config) {
		settings = map[string]any{
			"interface": map[string]any{
				"current":   cfg.Interface.Default,
				"available": []string{},
			},
			"vlan":       map[string]any{"enabled": cfg.VLAN.Enabled, "id": cfg.VLAN.ID},
			"ip":         map[string]any{"mode": cfg.IP.Mode},
			"thresholds": buildThresholdSettings(cfg),
			// Reflect the live config — these were previously hardcoded, so a GET
			// never echoed what a prior PUT wrote (and any concurrency token derived
			// from them would be incoherent).
			"healthChecks": map[string]any{
				"runPerformance": cfg.HealthChecks.RunPerformance,
				"runSpeedtest":   cfg.HealthChecks.RunSpeedtest,
				"runIperf":       cfg.HealthChecks.RunIperf,
				"runDiscovery":   cfg.HealthChecks.RunDiscovery,
			},
			"speedtest": map[string]any{
				"serverId":      cfg.Speedtest.ServerID,
				"autoRunOnLink": cfg.Speedtest.AutoRunOnLink,
			},
			"iperf": map[string]any{
				"autoRunOnLink": cfg.Iperf.AutoRunOnLink, "server": cfg.Iperf.Server,
				"port": cfg.Iperf.Port, "protocol": cfg.Iperf.Protocol,
				"direction": cfg.Iperf.Direction, "duration": cfg.Iperf.Duration,
				"serverPort": cfg.Iperf.ServerPort, "enableServer": cfg.Iperf.EnableServer,
			},
			"cardSettings": buildCardSettings(),
			"displayOptions": map[string]any{
				"showPublicIP": cfg.DisplayOptions.ShowPublicIP,
				"unitSystem":   cfg.DisplayOptions.UnitSystem,
			},
		}
		etag = cfg.SettingsETagLocked()
	})
	return settings, etag
}

// Update applies updates to the live config and persists it. When ifMatch is
// non-empty it is compared to the current ETag; a mismatch returns ErrConflict.
// A type error in any apply helper returns ErrValidation.
func (s *Service) Update(updates map[string]any, ifMatch string) error {
	return s.store.Write(func(cfg *config.Config) error {
		// Compare-and-apply is atomic under the write lock: a concurrent writer
		// cannot slip between the ETag check and the mutations below.
		if ifMatch != "" && ifMatch != strings.Trim(cfg.SettingsETagLocked(), `"`) {
			return ErrConflict
		}

		var applyErrors []error
		if err := applyThresholdUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyHealthChecksUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applySpeedtestUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyIperfUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyFABOptionsUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyDisplayOptionsUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}

		if len(applyErrors) > 0 {
			return ErrValidation
		}
		return nil
	})
}

// ============================================================================
// Read helpers
// ============================================================================

// Threshold-tier keys in the settings read model: "good" carries the
// warning-boundary value, "warning" the critical-boundary value (the wire
// contract, preserved from the original handler).
const (
	tierGood    = "good"
	tierWarning = "warning"
)

// buildThresholdSettings builds the threshold section of the settings read
// model. Moved from *Server.buildThresholdSettings; behavior-identical.
func buildThresholdSettings(cfg *config.Config) map[string]any {
	t := &cfg.Thresholds
	return map[string]any{
		"dns": map[string]int64{
			tierGood:    t.DNS.Warning.Milliseconds(),
			tierWarning: t.DNS.Critical.Milliseconds(),
		},
		"gateway": map[string]int64{
			tierGood:    t.Ping.Warning.Milliseconds(),
			tierWarning: t.Ping.Critical.Milliseconds(),
		},
		"wifi": map[string]int{
			tierGood:    t.WiFi.Signal.Warning,
			tierWarning: t.WiFi.Signal.Critical,
		},
		"customPing": map[string]int64{
			tierGood:    t.CustomTests.Ping.Warning.Milliseconds(),
			tierWarning: t.CustomTests.Ping.Critical.Milliseconds(),
		},
		"customTcp": map[string]int64{
			tierGood:    t.CustomTests.TCP.Warning.Milliseconds(),
			tierWarning: t.CustomTests.TCP.Critical.Milliseconds(),
		},
		"customHttp": map[string]int64{
			tierGood:    t.CustomTests.HTTP.Warning.Milliseconds(),
			tierWarning: t.CustomTests.HTTP.Critical.Milliseconds(),
		},
		"httpTimings": map[string]map[string]int64{
			"dns": {
				tierGood:    t.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
				tierWarning: t.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
			},
			"tcp": {
				tierGood:    t.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
				tierWarning: t.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
			},
			"tls": {
				tierGood:    t.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
				tierWarning: t.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
			},
			"ttfb": {
				tierGood:    t.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
				tierWarning: t.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
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

// ============================================================================
// Apply helpers (write path)
// ============================================================================

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
