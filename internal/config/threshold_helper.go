// Package config provides threshold extraction helpers.
package config

// ThresholdMsValues contains extracted millisecond threshold values.
// This helper eliminates duplication between ProfileSettings.FromConfig and GetDefaultSettings.
type ThresholdMsValues struct {
	DNSWarning      int64
	DNSCritical     int64
	GatewayWarning  int64
	GatewayCritical int64
	WiFiWarning     int
	WiFiCritical    int
	PingWarning     int64
	PingCritical    int64
	TCPWarning      int64
	TCPCritical     int64
	HTTPWarning     int64
	HTTPCritical    int64
	TimingDNSWarn   int64
	TimingDNSCrit   int64
	TimingTCPWarn   int64
	TimingTCPCrit   int64
	TimingTLSWarn   int64
	TimingTLSCrit   int64
	TimingTTFBWarn  int64
	TimingTTFBCrit  int64
}

// ExtractThresholdMs extracts all threshold millisecond values from config.
func ExtractThresholdMs(cfg *Config) ThresholdMsValues {
	return ThresholdMsValues{
		DNSWarning:      cfg.Thresholds.DNS.Warning.Milliseconds(),
		DNSCritical:     cfg.Thresholds.DNS.Critical.Milliseconds(),
		GatewayWarning:  cfg.Thresholds.Ping.Warning.Milliseconds(),
		GatewayCritical: cfg.Thresholds.Ping.Critical.Milliseconds(),
		WiFiWarning:     cfg.Thresholds.WiFi.Signal.Warning,
		WiFiCritical:    cfg.Thresholds.WiFi.Signal.Critical,
		PingWarning:     cfg.Thresholds.CustomTests.Ping.Warning.Milliseconds(),
		PingCritical:    cfg.Thresholds.CustomTests.Ping.Critical.Milliseconds(),
		TCPWarning:      cfg.Thresholds.CustomTests.TCP.Warning.Milliseconds(),
		TCPCritical:     cfg.Thresholds.CustomTests.TCP.Critical.Milliseconds(),
		HTTPWarning:     cfg.Thresholds.CustomTests.HTTP.Warning.Milliseconds(),
		HTTPCritical:    cfg.Thresholds.CustomTests.HTTP.Critical.Milliseconds(),
		TimingDNSWarn:   cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
		TimingDNSCrit:   cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
		TimingTCPWarn:   cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
		TimingTCPCrit:   cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
		TimingTLSWarn:   cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
		TimingTLSCrit:   cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
		TimingTTFBWarn:  cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
		TimingTTFBCrit:  cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
	}
}
