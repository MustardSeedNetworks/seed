package autotest

// stages.go grades each existing check's answer. The checks are passed in as
// functions so a stage can be exercised with a fixed answer; the server binds
// them to the same testers its per-check pages use.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/cable"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/netif"
)

// LinkReader returns the link state of an interface.
type LinkReader func(iface string) (*netif.LinkStatus, error)

// LinkStage checks for carrier and a full-duplex negotiation.
func LinkStage(read LinkReader) Stage {
	return Stage{Name: StageLink, Run: func(_ context.Context, iface string, _ Spec) StageReport {
		st, err := read(iface)
		if err != nil {
			return StageReport{Outcome: OutcomeFail, Reason: ReasonError, Message: err.Error()}
		}
		if !st.Carrier {
			return StageReport{Outcome: OutcomeFail, Reason: ReasonNoCarrier}
		}
		r := StageReport{Outcome: OutcomePass, Reason: ReasonOK, Readings: compact([]Reading{
			{Label: "speed", Value: st.Speed},
			{Label: "duplex", Value: st.Duplex},
			{Label: "autoNegotiation", Value: strconv.FormatBool(st.AutoNeg)},
		})}
		if strings.EqualFold(st.Duplex, "half") {
			r.Outcome, r.Reason = OutcomeWarn, ReasonHalfDuplex
		}
		return r
	}}
}

// CableStage runs a TDR cable test, but only when the link has no carrier. A
// TDR test takes the link down while it runs, which would fail every stage
// after it on a port that was working; with carrier present the cable has
// already proved itself well enough for this run.
func CableStage(read LinkReader, test func() *cable.TestResult) Stage {
	return Stage{Name: StageCable, Run: func(_ context.Context, iface string, _ Spec) StageReport {
		if st, err := read(iface); err == nil && st.Carrier {
			return StageReport{Outcome: OutcomeSkipped, Reason: ReasonLinkUp}
		}
		res := test()
		if !res.TDR {
			return StageReport{Outcome: OutcomeSkipped, Reason: ReasonUnsupported}
		}
		r := StageReport{Outcome: OutcomePass, Reason: ReasonOK}
		for _, p := range res.Pairs {
			r.Readings = append(r.Readings, Reading{Label: "pair " + p.Pair, Value: pairValue(p)})
		}
		if res.Status != cable.StatusOK {
			r.Outcome, r.Reason = OutcomeFail, ReasonCableFault
			r.Message = strings.Join(res.Faults, "; ")
		}
		return r
	}}
}

func pairValue(p cable.PairResult) string {
	if p.LengthM == nil {
		return string(p.Status)
	}
	return fmt.Sprintf("%s %.1f m", p.Status, *p.LengthM)
}

// LeaseReader returns the DHCP lease an interface holds; an
// *dhcp.InterfaceError means it holds none.
type LeaseReader func(iface string) (*dhcp.LeaseInfo, error)

// DHCPStage reads the lease the interface holds. A lease missing the gateway,
// the resolvers or the mask warns, as on the DHCP page. No lease on an
// interface that has an address is most often a static configuration, so it
// is skipped rather than failed; no lease and no address is the failure.
func DHCPStage(read LinkReader, lease LeaseReader) Stage {
	return Stage{Name: StageDHCP, Run: func(_ context.Context, iface string, _ Spec) StageReport {
		l, err := lease(iface)
		if noLease := (*dhcp.InterfaceError)(nil); errors.As(err, &noLease) {
			if st, linkErr := read(iface); linkErr == nil && st.HasIP {
				return StageReport{Outcome: OutcomeSkipped, Reason: ReasonNoLeaseFound}
			}
			return StageReport{Outcome: OutcomeFail, Reason: ReasonNoAddress, Message: err.Error()}
		}
		if err != nil {
			return StageReport{Outcome: OutcomeFail, Reason: ReasonError, Message: err.Error()}
		}
		r := StageReport{Outcome: OutcomePass, Reason: ReasonOK, Readings: compact([]Reading{
			{Label: "server", Value: l.ServerIP},
			{Label: "address", Value: l.IPAddress},
			{Label: "subnetMask", Value: l.SubnetMask},
			{Label: "gateway", Value: l.Gateway},
			{Label: "dnsServers", Value: strings.Join(l.DNSServers, ", ")},
		})}
		switch {
		case !l.Expiry.IsZero() && l.Expiry.Before(time.Now()):
			r.Outcome, r.Reason = OutcomeFail, ReasonLeaseExpired
		case l.Gateway == "" || len(l.DNSServers) == 0 || l.SubnetMask == "":
			r.Outcome, r.Reason = OutcomeWarn, ReasonIncompleteLease
		}
		return r
	}}
}

// DNSStage resolves the configured test hostname through the interface's
// resolvers. Either address family answering is a pass.
func DNSStage(test func(ctx context.Context) *dns.TestResult) Stage {
	return Stage{Name: StageDNS, Run: func(ctx context.Context, _ string, _ Spec) StageReport {
		res := test(ctx)
		if res.Forward == nil && res.ForwardIPv6 == nil {
			return StageReport{Outcome: OutcomeFail, Reason: ReasonNoResolvers}
		}
		best := bestLookup(res.Forward, res.ForwardIPv6)
		r := StageReport{Readings: compact([]Reading{
			{Label: "server", Value: res.Server},
			{Label: "hostname", Value: res.TestHostname},
			{Label: "resolved", Value: strings.Join(best.Resolved, ", ")},
			{Label: "timeMs", Value: strconv.FormatInt(best.TimeMs, 10)},
		})}
		switch {
		case best.Outcome != dns.OutcomeResolved:
			r.Outcome, r.Reason, r.Message = OutcomeFail, ReasonNoAnswer, best.Error
		case best.Status == dns.StatusWarning || best.Status == dns.StatusError:
			r.Outcome, r.Reason = OutcomeWarn, ReasonSlow
		default:
			r.Outcome, r.Reason = OutcomePass, ReasonOK
		}
		return r
	}}
}

// bestLookup prefers a lookup that resolved, then the faster one. At least
// one of a and b is non-nil.
func bestLookup(a, b *dns.LookupResult) *dns.LookupResult {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	case (a.Outcome == dns.OutcomeResolved) != (b.Outcome == dns.OutcomeResolved):
		if a.Outcome == dns.OutcomeResolved {
			return a
		}
		return b
	case b.TimeMs < a.TimeMs:
		return b
	}
	return a
}

// GatewayStage pings the interface's gateway and grades loss and latency
// against the gateway tester's thresholds.
func GatewayStage(test func() *gateway.PingStats) Stage {
	return Stage{Name: StageGateway, Run: func(_ context.Context, _ string, _ Spec) StageReport {
		st := test()
		if st.Gateway == "" {
			return StageReport{Outcome: OutcomeFail, Reason: ReasonNoGateway}
		}
		r := StageReport{Readings: []Reading{
			{Label: "gateway", Value: st.Gateway},
			{Label: "source", Value: string(st.Source)},
			{Label: "lossPercent", Value: strconv.FormatFloat(st.LossPercent, 'f', 0, 64)},
			{Label: "avgMs", Value: strconv.FormatFloat(st.AvgTime, 'f', 1, 64)},
		}}
		switch {
		case !st.Reachable:
			r.Outcome, r.Reason = OutcomeFail, ReasonUnreachable
			if len(st.Results) > 0 {
				r.Message = st.Results[len(st.Results)-1].Error
			}
		case st.Status == gateway.StatusSuccess:
			r.Outcome, r.Reason = OutcomePass, ReasonOK
		default:
			r.Outcome, r.Reason = OutcomeWarn, ReasonLossy
		}
		return r
	}}
}

// VLANStage reports the native and voice VLAN the switch advertised over
// LLDP or CDP and, when the request names one, checks the native VLAN
// against it. A switch that advertised nothing leaves nothing to check.
func VLANStage(advertised func() (native, voice *int)) Stage {
	return Stage{Name: StageVLAN, Run: func(_ context.Context, _ string, req Spec) StageReport {
		native, voice := advertised()
		if native == nil {
			return StageReport{Outcome: OutcomeSkipped, Reason: ReasonNotAdvertised}
		}
		r := StageReport{Outcome: OutcomePass, Reason: ReasonOK, Readings: []Reading{
			{Label: "nativeVlan", Value: strconv.Itoa(*native)},
		}}
		if voice != nil {
			r.Readings = append(r.Readings, Reading{Label: "voiceVlan", Value: strconv.Itoa(*voice)})
		}
		if req.ExpectedVLAN != 0 {
			r.Readings = append(r.Readings, Reading{Label: "expectedVlan", Value: strconv.Itoa(req.ExpectedVLAN)})
			if *native != req.ExpectedVLAN {
				r.Outcome, r.Reason = OutcomeFail, ReasonVLANMismatch
			}
		}
		return r
	}}
}

// Target is an internet reachability target: the operator's enabled ping
// targets from the health-check settings.
type Target struct {
	Name string
	Host string
}

// Pinger sends one echo to host through iface and returns its round-trip
// time.
type Pinger func(ctx context.Context, iface, host string) (time.Duration, error)

// InternetStage pings every target through the interface under test. All
// answering is a pass, some a warning, none a failure.
func InternetStage(targets func(ctx context.Context) ([]Target, error), ping Pinger) Stage {
	return Stage{Name: StageInternet, Run: func(ctx context.Context, iface string, _ Spec) StageReport {
		ts, err := targets(ctx)
		if err != nil {
			return StageReport{Outcome: OutcomeFail, Reason: ReasonError, Message: err.Error()}
		}
		if len(ts) == 0 {
			return StageReport{Outcome: OutcomeSkipped, Reason: ReasonNoTargets}
		}
		r := StageReport{}
		reached := 0
		for _, t := range ts {
			rtt, pingErr := ping(ctx, iface, t.Host)
			value := "unreachable"
			if pingErr == nil {
				reached++
				value = strconv.FormatFloat(float64(rtt.Microseconds())/1000, 'f', 1, 64) + " ms"
			}
			r.Readings = append(r.Readings, Reading{Label: t.Name + " (" + t.Host + ")", Value: value})
		}
		switch reached {
		case len(ts):
			r.Outcome, r.Reason = OutcomePass, ReasonOK
		case 0:
			r.Outcome, r.Reason = OutcomeFail, ReasonUnreachable
		default:
			r.Outcome, r.Reason = OutcomeWarn, ReasonPartial
		}
		return r
	}}
}

// compact drops details whose value is empty, so a stage lists only what the
// check actually reported.
func compact(ds []Reading) []Reading {
	out := ds[:0]
	for _, d := range ds {
		if d.Value != "" {
			out = append(out, d)
		}
	}
	return out
}
