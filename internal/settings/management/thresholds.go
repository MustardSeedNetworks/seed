package management

import (
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

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
