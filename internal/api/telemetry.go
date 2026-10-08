package api

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
)

// telemetryDNSTimeout bounds the DNS half of one sample so a dead resolver
// cannot stretch a sample past the next tick.
const telemetryDNSTimeout = 10 * time.Second

// initTelemetry registers the engine that records link, gateway and DNS
// health into store every [telemetry.Interval].
func (s *Server) initTelemetry(store telemetry.Store) {
	eng := telemetry.New(serverSampler{s: s}, store, telemetry.Interval, logging.GetLogger())
	if err := s.registerEngineIfLicensed(eng); err != nil {
		logging.GetLogger().Warn("telemetry engine registry registration failed", "error", err)
	}
}

// serverSampler measures the selected interface with the server's own
// testers, the ones the dashboard cards read.
type serverSampler struct {
	s *Server
}

func (ss serverSampler) Sample(ctx context.Context) telemetry.Sample {
	s := ss.s
	sample := telemetry.Sample{At: time.Now()}
	if s.netManager() == nil {
		return sample
	}
	if err := s.netManager().RefreshInterfaces(); err != nil {
		return sample
	}
	sample.Interface = s.netManager().GetCurrentInterface()
	if sample.Interface == "" {
		return sample
	}

	if status, err := s.netManager().GetLinkStatus(sample.Interface); err == nil {
		sample.Points = append(sample.Points, linkPoints(status)...)
	}
	if s.gatewayTester() != nil {
		sample.Points = append(sample.Points, gatewayPoints(s.gatewayTester().Test())...)
	}
	if s.dnsTester() != nil {
		dnsCtx, cancel := context.WithTimeout(ctx, telemetryDNSTimeout)
		sample.Points = append(sample.Points, dnsPoints(s.dnsTester().Test(dnsCtx))...)
		cancel()
	}
	return sample
}

func linkPoints(status *netif.LinkStatus) []telemetry.Point {
	return []telemetry.Point{{Type: telemetry.LinkCarrier, Value: boolValue(status.Carrier), Unit: telemetry.UnitBool}}
}

// gatewayPoints records nothing when there is no gateway to ping, and no
// latency when no reply came back.
func gatewayPoints(stats *gateway.PingStats) []telemetry.Point {
	if stats == nil || stats.Gateway == "" || stats.Sent == 0 {
		return nil
	}
	points := []telemetry.Point{
		{Type: telemetry.GatewayReachable, Value: boolValue(stats.Reachable), Unit: telemetry.UnitBool},
		{Type: telemetry.GatewayLossPct, Value: stats.LossPercent, Unit: telemetry.UnitPercent},
	}
	if stats.Received > 0 {
		points = append(
			points,
			telemetry.Point{Type: telemetry.GatewayLatencyMs, Value: stats.AvgTime, Unit: telemetry.UnitMs},
		)
	}
	return points
}

// dnsPoints reads the IPv4 forward lookup, the one every resolver answers.
// The query time is recorded only when the name resolved.
func dnsPoints(result *dns.TestResult) []telemetry.Point {
	if result == nil || result.Forward == nil {
		return nil
	}
	resolved := result.Forward.Outcome == dns.OutcomeResolved
	points := []telemetry.Point{{Type: telemetry.DNSResolved, Value: boolValue(resolved), Unit: telemetry.UnitBool}}
	if resolved {
		points = append(
			points,
			telemetry.Point{Type: telemetry.DNSQueryMs, Value: float64(result.Forward.TimeMs), Unit: telemetry.UnitMs},
		)
	}
	return points
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
