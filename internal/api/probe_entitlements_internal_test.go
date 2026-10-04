package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// dnsServers is n enabled configured DNS servers on loopback, which refuse at
// once rather than time out.
func dnsServers(n int) []DNSServerResponse {
	out := make([]DNSServerResponse, n)
	for i := range out {
		out[i] = DNSServerResponse{Address: fmt.Sprintf("127.0.0.%d", i+2), Enabled: true}
	}
	return out
}

// dns_monitoring is a count (seed#2327): saving up to the tier's limit of
// enabled DNS servers succeeds and one more is refused with the upgrade hint.
// Disabled servers do not count, and Pro is unlimited.
func TestDNSServerSettingsCappedByTier(t *testing.T) {
	for _, tt := range []struct {
		tier  license.Tier
		limit int // 0 = unlimited; probed past starterDNSServers
	}{
		{license.TierFree, freeDNSServers},
		{license.TierStarter, starterDNSServers},
		{license.TierPro, 0},
	} {
		t.Run(tt.tier.String(), func(t *testing.T) {
			s := newHealthSettingsServer(t)
			s.licenseMgr = licenseManagerAt(t, tt.tier)

			saved := tt.limit
			if saved == 0 {
				saved = starterDNSServers + 1
			}
			servers := append(dnsServers(saved), DNSServerResponse{Address: "127.0.0.99", Enabled: false})
			if w := putHealthChecksSettings(t, s, TestsSettingsResponse{DNSServers: servers}); w.Code != http.StatusOK {
				t.Fatalf("save %d servers: status %d, body=%s", saved, w.Code, w.Body.String())
			}
			if tt.limit == 0 {
				return
			}
			w := putHealthChecksSettings(t, s, TestsSettingsResponse{DNSServers: dnsServers(tt.limit + 1)})
			requireFeatureGate(t, w, "dns_monitoring")
			if got := len(s.config.DNS.Servers); got != len(servers) {
				t.Errorf("config holds %d servers after the refused save, want the %d saved before", got, len(servers))
			}
		})
	}
}

// The DNS tester offers no more configured servers than the licence allows,
// so a configuration saved under a larger licence stops being tested past the
// limit when the licence shrinks.
func TestDNSTesterHonoursServerLimit(t *testing.T) {
	configured := dnsServers(3)
	for _, tt := range []struct {
		tier license.Tier
		want int
	}{
		{license.TierFree, freeDNSServers},
		{license.TierPro, len(configured)},
	} {
		t.Run(tt.tier.String(), func(t *testing.T) {
			s := &Server{licenseMgr: licenseManagerAt(t, tt.tier)}
			s.dnsTest = dns.NewTester("", "localhost", dns.DefaultThresholds())
			cfg := &config.Config{}
			for _, c := range configured {
				cfg.DNS.Servers = append(cfg.DNS.Servers, config.DNSServer{Address: c.Address, Enabled: c.Enabled})
			}
			s.initNetworkServices(cfg)

			offered := s.dnsTester().Test(t.Context()).Servers
			got := 0
			for _, c := range configured {
				if slices.Contains(offered, c.Address) {
					got++
				}
			}
			if got != tt.want {
				t.Errorf("tested %d configured servers (%v), want %d", got, offered, tt.want)
			}
		})
	}
}

// httpsResult is a successful https probe result whose certificate expires in
// days.
func httpsResult(t *testing.T, name string, days int) (probe.Probe, probe.Result) {
	t.Helper()
	meta, err := json.Marshal(map[string]any{
		"status_code": 200,
		"tls":         map[string]any{"days_remaining": days, "tls_version": "TLS 1.3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return probe.Probe{Kind: probe.KindHTTPS, DisplayName: name},
		probe.Result{Kind: probe.KindHTTPS, Success: true, LatencyMs: 5, Metadata: meta}
}

// ssl_cert_monitoring is a count (seed#2327): the https checks past the
// licence's limit still run and report, but their certificate is withheld,
// including an expiring one, which would otherwise warn. Plain http results
// do not use up the limit.
func TestCertSummaryWithheldPastLimit(t *testing.T) {
	th := lenientThresholds()
	var resp HealthCheckRunResponse
	certs := &certAllowance{limit: 2}

	plain := probe.Result{
		Kind: probe.KindHTTP, Success: true, LatencyMs: 5, Metadata: json.RawMessage(`{"status_code":200}`),
	}
	mapRunResult(&resp, probe.Probe{Kind: probe.KindHTTP, DisplayName: "plain"}, plain, &th, certs)
	for i, days := range []int{200, 200, 3} {
		p, r := httpsResult(t, fmt.Sprintf("https-%d", i), days)
		mapRunResult(&resp, p, r, &th, certs)
	}

	if len(resp.HTTPResults) != 4 {
		t.Fatalf("got %d http results, want 4", len(resp.HTTPResults))
	}
	for _, res := range resp.HTTPResults[:3] {
		if res.CertWithheld {
			t.Errorf("%s: certificate withheld inside the limit", res.Name)
		}
	}
	for _, res := range resp.HTTPResults[1:3] {
		if res.CertStatus != statusSuccess || res.CertDaysLeft != 200 {
			t.Errorf("%s: cert status %q, %d days; want a summary", res.Name, res.CertStatus, res.CertDaysLeft)
		}
	}
	past := resp.HTTPResults[3]
	if !past.CertWithheld || past.CertStatus != "" || past.CertDaysLeft != 0 || past.TLSVersion != "" {
		t.Errorf("past the limit: %+v, want the certificate withheld", past)
	}
	if !past.Success || past.Status != 200 || past.TestStatus != statusSuccess {
		t.Errorf("past the limit: %+v, want the check itself still reported", past)
	}
}

// The certificate limit follows the licence: Free covers the two HTTPS
// endpoints a new install ships with, Starter more, Pro all of them.
func TestCertCheckLimitByTier(t *testing.T) {
	for _, tt := range []struct {
		tier license.Tier
		want int
	}{
		{license.TierFree, freeCertChecks},
		{license.TierStarter, starterCertChecks},
		{license.TierPro, 0},
	} {
		t.Run(tt.tier.String(), func(t *testing.T) {
			s := &Server{licenseMgr: licenseManagerAt(t, tt.tier)}
			if got := s.certCheckLimit(); got != tt.want {
				t.Errorf("certCheckLimit() = %d, want %d", got, tt.want)
			}
		})
	}
	https := 0
	for _, e := range config.DefaultConfig().HealthChecks.HTTPEndpoints {
		if strings.HasPrefix(e.URL, "https://") {
			https++
		}
	}
	if https > freeCertChecks {
		t.Errorf("a new install ships %d HTTPS endpoints, more than Free's %d certificates", https, freeCertChecks)
	}
}
