package api

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
)

// TestTelemetry_DaemonRecordsAndRollsUp is the #2623 acceptance: a server
// over a database records each telemetry series within one cadence of its
// engines starting, and the hourly rollup then has a bucket per series.
func TestTelemetry_DaemonRecordsAndRollsUp(t *testing.T) {
	db := newTestDB(t)
	gw := gateway.NewTester(gateway.DefaultThresholds())
	t.Cleanup(gw.Close)
	gw.SetGateway("127.0.0.1")
	s := &Server{
		engines:     engine.NewRegistry(nil),
		netMgr:      netif.NewMockManager(netif.DefaultMockConfig()),
		gatewayTest: gw,
		// localhost resolves from the hosts file, so the lookup needs no network.
		dnsTest: dns.NewTester("", "localhost", dns.DefaultThresholds()),
	}
	s.initDatabaseDependentServices(db)

	ctx := t.Context()
	if err := s.engines.Start(ctx); err != nil {
		t.Fatalf("start engines: %v", err)
	}
	t.Cleanup(func() { _ = s.engines.Stop(context.Background()) })

	want := []string{
		telemetry.LinkCarrier,
		telemetry.GatewayReachable,
		telemetry.GatewayLossPct,
		telemetry.DNSResolved,
		telemetry.DNSQueryMs,
	}
	var got []string
	deadline := time.Now().Add(telemetry.Interval)
	for time.Now().Before(deadline) {
		types, err := db.Metrics().GetDistinctTypes(ctx)
		if err != nil {
			t.Fatalf("distinct types: %v", err)
		}
		got = types
		if containsAll(got, want) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !containsAll(got, want) {
		t.Fatalf("metrics series after one cadence = %v, want all of %v", got, want)
	}

	hour := time.Now().UTC().Truncate(time.Hour)
	if _, err := database.NewMetricsRollupSource(db).RollupHour(ctx, hour); err != nil {
		t.Fatalf("rollup hour: %v", err)
	}
	for _, metricType := range want {
		var n int
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*) FROM metrics_hourly
			WHERE metric_type = ? AND target_kind = ? AND target_id = ? AND hour_bucket = ?
		`, metricType, telemetry.TargetKind, "eth0", hour.Format("2006-01-02T15:00:00Z")).Scan(&n); err != nil {
			t.Fatalf("count hourly %s: %v", metricType, err)
		}
		if n != 1 {
			t.Errorf("metrics_hourly buckets for %s = %d, want 1", metricType, n)
		}
	}
}

func containsAll(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

func TestGatewayPoints(t *testing.T) {
	tests := []struct {
		name  string
		stats *gateway.PingStats
		want  []telemetry.Point
	}{
		{name: "no stats", stats: nil},
		{name: "no gateway detected", stats: &gateway.PingStats{}},
		{
			name: "replies",
			stats: &gateway.PingStats{
				Gateway:     "10.0.0.1",
				Sent:        4,
				Received:    3,
				LossPercent: 25,
				AvgTime:     1.5,
				Reachable:   true,
			},
			want: []telemetry.Point{
				{Type: telemetry.GatewayReachable, Value: 1, Unit: telemetry.UnitBool},
				{Type: telemetry.GatewayLossPct, Value: 25, Unit: telemetry.UnitPercent},
				{Type: telemetry.GatewayLatencyMs, Value: 1.5, Unit: telemetry.UnitMs},
			},
		},
		{
			name:  "no replies has no latency",
			stats: &gateway.PingStats{Gateway: "10.0.0.1", Sent: 4, LossPercent: 100},
			want: []telemetry.Point{
				{Type: telemetry.GatewayReachable, Value: 0, Unit: telemetry.UnitBool},
				{Type: telemetry.GatewayLossPct, Value: 100, Unit: telemetry.UnitPercent},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gatewayPoints(tt.stats); !slices.Equal(got, tt.want) {
				t.Errorf("gatewayPoints = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDNSPoints(t *testing.T) {
	tests := []struct {
		name   string
		result *dns.TestResult
		want   []telemetry.Point
	}{
		{name: "no result", result: nil},
		{name: "interface has no resolvers", result: &dns.TestResult{}},
		{
			name:   "resolved",
			result: &dns.TestResult{Forward: &dns.LookupResult{Outcome: dns.OutcomeResolved, TimeMs: 12}},
			want: []telemetry.Point{
				{Type: telemetry.DNSResolved, Value: 1, Unit: telemetry.UnitBool},
				{Type: telemetry.DNSQueryMs, Value: 12, Unit: telemetry.UnitMs},
			},
		},
		{
			name:   "failed has no query time",
			result: &dns.TestResult{Forward: &dns.LookupResult{Outcome: dns.OutcomeFailed, TimeMs: 5000}},
			want:   []telemetry.Point{{Type: telemetry.DNSResolved, Value: 0, Unit: telemetry.UnitBool}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dnsPoints(tt.result); !slices.Equal(got, tt.want) {
				t.Errorf("dnsPoints = %v, want %v", got, tt.want)
			}
		})
	}
}
