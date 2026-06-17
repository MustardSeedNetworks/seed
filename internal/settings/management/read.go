package management

import "github.com/MustardSeedNetworks/seed/internal/config"

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
