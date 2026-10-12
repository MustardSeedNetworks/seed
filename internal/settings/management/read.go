package management

import "github.com/MustardSeedNetworks/seed/internal/config"

// Settings is the read model GET /settings serves: the settings this service
// writes, plus the VLAN and IP mode the dashboard shows beside them.
type Settings struct {
	VLAN           VLANSetting         `json:"vlan"`
	IP             IPSetting           `json:"ip"`
	Thresholds     ThresholdSettings   `json:"thresholds"`
	HealthChecks   HealthCheckSettings `json:"healthChecks"`
	Speedtest      SpeedtestSettings   `json:"speedtest"`
	Iperf          IperfTestSettings   `json:"iperf"`
	Alerts         AlertSettings       `json:"alerts"`
	DisplayOptions DisplaySettings     `json:"displayOptions"`
	Identity       IdentitySettings    `json:"identity"`
}

// VLANSetting is the configured VLAN tag.
type VLANSetting struct {
	Enabled bool `json:"enabled"`
	ID      int  `json:"id"`
}

// IPSetting is the configured addressing mode, "dhcp" or "static".
type IPSetting struct {
	Mode string `json:"mode"`
}

// ThresholdPair is one metric's two boundaries in milliseconds (dBm for
// Wi-Fi signal): "good" carries the warning boundary and "warning" the
// critical one, the wire contract the original handler set.
type ThresholdPair struct {
	Good    int64 `json:"good"`
	Warning int64 `json:"warning"`
}

// ThresholdSettings are the dashboard's good/warning boundaries.
type ThresholdSettings struct {
	DNS         ThresholdPair      `json:"dns"`
	Gateway     ThresholdPair      `json:"gateway"`
	WiFi        ThresholdPair      `json:"wifi"`
	CustomPing  ThresholdPair      `json:"customPing"`
	CustomTCP   ThresholdPair      `json:"customTcp"`
	CustomHTTP  ThresholdPair      `json:"customHttp"`
	HTTPTimings HTTPTimingSettings `json:"httpTimings"`
}

// HTTPTimingSettings are the boundaries for each phase of an HTTP check.
type HTTPTimingSettings struct {
	DNS  ThresholdPair `json:"dns"`
	TCP  ThresholdPair `json:"tcp"`
	TLS  ThresholdPair `json:"tls"`
	TTFB ThresholdPair `json:"ttfb"`
}

// HealthCheckSettings are the suites a health-check run includes.
type HealthCheckSettings struct {
	RunPerformance bool `json:"runPerformance"`
	RunSpeedtest   bool `json:"runSpeedtest"`
	RunIperf       bool `json:"runIperf"`
	RunDiscovery   bool `json:"runDiscovery"`
}

// SpeedtestSettings choose the speed test server and when it runs.
type SpeedtestSettings struct {
	ServerID      string `json:"serverId"`
	AutoRunOnLink bool   `json:"autoRunOnLink"`
}

// IperfTestSettings are the iperf3 client test and the local server.
type IperfTestSettings struct {
	AutoRunOnLink bool   `json:"autoRunOnLink"`
	Server        string `json:"server"`
	Port          int    `json:"port"`
	Protocol      string `json:"protocol"`
	Direction     string `json:"direction"`
	Duration      int    `json:"duration"`
	ServerPort    int    `json:"serverPort"`
	EnableServer  bool   `json:"enableServer"`
}

// DisplaySettings are the dashboard's presentation choices.
type DisplaySettings struct {
	ShowPublicIP bool   `json:"showPublicIP"`
	UnitSystem   string `json:"unitSystem"`
}

func buildSettings(cfg *config.Config) Settings {
	return Settings{
		VLAN:       VLANSetting{Enabled: cfg.VLAN.Enabled, ID: cfg.VLAN.ID},
		IP:         IPSetting{Mode: cfg.IP.Mode},
		Thresholds: buildThresholdSettings(&cfg.Thresholds),
		HealthChecks: HealthCheckSettings{
			RunPerformance: cfg.HealthChecks.RunPerformance,
			RunSpeedtest:   cfg.HealthChecks.RunSpeedtest,
			RunIperf:       cfg.HealthChecks.RunIperf,
			RunDiscovery:   cfg.HealthChecks.RunDiscovery,
		},
		Speedtest: SpeedtestSettings{ServerID: cfg.Speedtest.ServerID, AutoRunOnLink: cfg.Speedtest.AutoRunOnLink},
		Iperf: IperfTestSettings{
			AutoRunOnLink: cfg.Iperf.AutoRunOnLink, Server: cfg.Iperf.Server,
			Port: cfg.Iperf.Port, Protocol: cfg.Iperf.Protocol,
			Direction: cfg.Iperf.Direction, Duration: cfg.Iperf.Duration,
			ServerPort: cfg.Iperf.ServerPort, EnableServer: cfg.Iperf.EnableServer,
		},
		Alerts: buildAlertSettings(cfg),
		DisplayOptions: DisplaySettings{
			ShowPublicIP: cfg.DisplayOptions.ShowPublicIP,
			UnitSystem:   cfg.DisplayOptions.UnitSystem,
		},
		Identity: IdentitySettings{Name: cfg.Identity.Name, Location: cfg.Identity.Location},
	}
}

func durationPair(t config.Threshold) ThresholdPair {
	return ThresholdPair{Good: t.Warning.Milliseconds(), Warning: t.Critical.Milliseconds()}
}

func buildThresholdSettings(t *config.ThresholdsConfig) ThresholdSettings {
	timings := &t.CustomTests.HTTPTimings
	return ThresholdSettings{
		DNS:     durationPair(t.DNS),
		Gateway: durationPair(t.Ping),
		WiFi: ThresholdPair{
			Good:    int64(t.WiFi.Signal.Warning),
			Warning: int64(t.WiFi.Signal.Critical),
		},
		CustomPing: durationPair(t.CustomTests.Ping),
		CustomTCP:  durationPair(t.CustomTests.TCP),
		CustomHTTP: durationPair(t.CustomTests.HTTP),
		HTTPTimings: HTTPTimingSettings{
			DNS:  durationPair(timings.DNS),
			TCP:  durationPair(timings.TCP),
			TLS:  durationPair(timings.TLS),
			TTFB: durationPair(timings.TTFB),
		},
	}
}
