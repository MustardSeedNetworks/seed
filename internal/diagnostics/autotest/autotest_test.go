package autotest_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/cable"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/netif"
)

func fixed(name autotest.StageName, o autotest.Outcome) autotest.Stage {
	return autotest.Stage{Name: name, Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
		return autotest.StageReport{Outcome: o}
	}}
}

func TestRunGradesTheSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		outcomes []autotest.Outcome
		want     autotest.Outcome
	}{
		{"all pass", []autotest.Outcome{autotest.OutcomePass, autotest.OutcomePass}, autotest.OutcomePass},
		{
			"skipped does not fail",
			[]autotest.Outcome{autotest.OutcomePass, autotest.OutcomeSkipped},
			autotest.OutcomePass,
		},
		{"a warning warns", []autotest.Outcome{autotest.OutcomeWarn, autotest.OutcomePass}, autotest.OutcomeWarn},
		{"a failure fails", []autotest.Outcome{autotest.OutcomeWarn, autotest.OutcomeFail}, autotest.OutcomeFail},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var stages []autotest.Stage
			for _, o := range tc.outcomes {
				stages = append(stages, fixed(autotest.StageLink, o))
			}
			res := autotest.Run(t.Context(), "eth0", autotest.Spec{}, stages, func(float64) {})
			if res.Verdict != tc.want {
				t.Fatalf("verdict = %q, want %q", res.Verdict, tc.want)
			}
		})
	}
}

// The acceptance: a failing stage names itself and the stages after it still
// run.
func TestRunKeepsGoingPastAFailure(t *testing.T) {
	t.Parallel()

	var progress []float64
	stages := []autotest.Stage{
		fixed(autotest.StageLink, autotest.OutcomeFail),
		fixed(autotest.StageDHCP, autotest.OutcomePass),
		fixed(autotest.StageDNS, autotest.OutcomeWarn),
	}
	res := autotest.Run(
		t.Context(),
		"eth0",
		autotest.Spec{},
		stages,
		func(f float64) { progress = append(progress, f) },
	)

	if res.Verdict != autotest.OutcomeFail || res.Interface != "eth0" {
		t.Fatalf("result = %+v, want a failed run on eth0", res)
	}
	got := make([]autotest.StageName, 0, len(res.Stages))
	for _, s := range res.Stages {
		got = append(got, s.Stage)
	}
	if want := []autotest.StageName{
		autotest.StageLink,
		autotest.StageDHCP,
		autotest.StageDNS,
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("stages = %v, want %v", got, want)
	}
	if res.Stages[0].Outcome != autotest.OutcomeFail || res.Stages[2].Outcome != autotest.OutcomeWarn {
		t.Fatalf("stage outcomes = %+v", res.Stages)
	}
	if want := []float64{1.0 / 3, 2.0 / 3, 1}; !slices.Equal(progress, want) {
		t.Fatalf("progress = %v, want %v", progress, want)
	}
}

func TestRunSkipsTheRestOnceCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	stopper := autotest.Stage{
		Name: autotest.StageLink,
		Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
			cancel()
			return autotest.StageReport{Outcome: autotest.OutcomePass}
		},
	}
	ran := false
	after := autotest.Stage{
		Name: autotest.StageDHCP,
		Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
			ran = true
			return autotest.StageReport{Outcome: autotest.OutcomeFail}
		},
	}
	res := autotest.Run(ctx, "eth0", autotest.Spec{}, []autotest.Stage{stopper, after}, func(float64) {})

	if ran {
		t.Fatal("a stage ran after the run was cancelled")
	}
	if s := res.Stages[1]; s.Outcome != autotest.OutcomeSkipped || s.Reason != autotest.ReasonCancelled {
		t.Fatalf("stage after cancel = %+v, want skipped/cancelled", s)
	}
}

func TestRunBoundsEachStage(t *testing.T) {
	t.Parallel()

	st := autotest.Stage{
		Name: autotest.StageDNS,
		Run: func(ctx context.Context, _ string, _ autotest.Spec) autotest.StageReport {
			if _, ok := ctx.Deadline(); !ok {
				return autotest.StageReport{Outcome: autotest.OutcomeFail}
			}
			return autotest.StageReport{Outcome: autotest.OutcomePass}
		},
	}
	res := autotest.Run(t.Context(), "eth0", autotest.Spec{}, []autotest.Stage{st}, func(float64) {})
	if res.Verdict != autotest.OutcomePass {
		t.Fatal("stage context carried no deadline")
	}
}

func runOne(t *testing.T, st autotest.Stage, req autotest.Spec) autotest.StageReport {
	t.Helper()
	res := autotest.Run(t.Context(), "eth0", req, []autotest.Stage{st}, func(float64) {})
	if res.Stages[0].Stage != st.Name {
		t.Fatalf("stage named %q, want %q", res.Stages[0].Stage, st.Name)
	}
	return res.Stages[0]
}

func check(t *testing.T, got autotest.StageReport, outcome autotest.Outcome, reason autotest.Reason) {
	t.Helper()
	if got.Outcome != outcome || got.Reason != reason {
		t.Fatalf("got %s/%s, want %s/%s (%+v)", got.Outcome, got.Reason, outcome, reason, got)
	}
}

func linkReader(st *netif.LinkStatus, err error) autotest.LinkReader {
	return func(string) (*netif.LinkStatus, error) { return st, err }
}

func TestLinkStage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		st      *netif.LinkStatus
		err     error
		outcome autotest.Outcome
		reason  autotest.Reason
	}{
		{
			"full duplex", &netif.LinkStatus{Carrier: true, Speed: "1000Mb/s", Duplex: "full"}, nil,
			autotest.OutcomePass, autotest.ReasonOK,
		},
		{
			"half duplex", &netif.LinkStatus{Carrier: true, Speed: "100Mb/s", Duplex: "half"}, nil,
			autotest.OutcomeWarn, autotest.ReasonHalfDuplex,
		},
		{"no carrier", &netif.LinkStatus{}, nil, autotest.OutcomeFail, autotest.ReasonNoCarrier},
		{"read error", nil, errors.New("no such interface"), autotest.OutcomeFail, autotest.ReasonError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			check(t, runOne(t, autotest.LinkStage(linkReader(tc.st, tc.err)), autotest.Spec{}), tc.outcome, tc.reason)
		})
	}
}

func TestCableStage(t *testing.T) {
	t.Parallel()

	length := 12.5
	tests := []struct {
		name    string
		carrier bool
		res     *cable.TestResult
		outcome autotest.Outcome
		reason  autotest.Reason
	}{
		{"link up is not interrupted", true, nil, autotest.OutcomeSkipped, autotest.ReasonLinkUp},
		{"no TDR", false, &cable.TestResult{}, autotest.OutcomeSkipped, autotest.ReasonUnsupported},
		{
			"link state is not a measurement", false, &cable.TestResult{Supported: true, Status: cable.StatusOpen},
			autotest.OutcomeSkipped, autotest.ReasonUnsupported,
		},
		{
			"good cable", false, &cable.TestResult{Supported: true, TDR: true, Status: cable.StatusOK},
			autotest.OutcomePass, autotest.ReasonOK,
		},
		{"open pair", false, &cable.TestResult{
			Supported: true, TDR: true, Status: cable.StatusOpen, Faults: []string{"pair A open"},
			Pairs: []cable.PairResult{{Pair: "1-2", Status: cable.StatusOpen, LengthM: &length}},
		}, autotest.OutcomeFail, autotest.ReasonCableFault},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tested := false
			st := autotest.CableStage(
				linkReader(&netif.LinkStatus{Carrier: tc.carrier}, nil),
				func() *cable.TestResult {
					tested = true
					return tc.res
				},
			)
			got := runOne(t, st, autotest.Spec{})
			check(t, got, tc.outcome, tc.reason)
			if tc.carrier && tested {
				t.Fatal("TDR ran on a link with carrier")
			}
		})
	}

	got := runOne(t, autotest.CableStage(linkReader(&netif.LinkStatus{}, nil), func() *cable.TestResult {
		return &cable.TestResult{
			Supported: true, TDR: true, Status: cable.StatusOpen, Faults: []string{"pair A open"},
			Pairs: []cable.PairResult{{Pair: "1-2", Status: cable.StatusOpen, LengthM: &length}},
		}
	}), autotest.Spec{})
	if got.Message != "pair A open" || got.Readings[0] != (autotest.Reading{Label: "pair 1-2", Value: "open 12.5 m"}) {
		t.Fatalf("fault detail = %+v", got)
	}
}

func TestDHCPStage(t *testing.T) {
	t.Parallel()

	full := &dhcp.LeaseInfo{
		ServerIP: "10.0.0.1", IPAddress: "10.0.0.5", SubnetMask: "255.255.255.0",
		Gateway: "10.0.0.1", DNSServers: []string{"10.0.0.53"},
	}
	noGateway := *full
	noGateway.Gateway = ""
	expired := *full
	expired.Expiry = time.Now().Add(-time.Hour)
	noLease := &dhcp.InterfaceError{Message: "no lease found"}

	tests := []struct {
		name    string
		lease   *dhcp.LeaseInfo
		err     error
		hasIP   bool
		outcome autotest.Outcome
		reason  autotest.Reason
	}{
		{"complete lease", full, nil, true, autotest.OutcomePass, autotest.ReasonOK},
		{"lease without a gateway", &noGateway, nil, true, autotest.OutcomeWarn, autotest.ReasonIncompleteLease},
		{"expired lease", &expired, nil, true, autotest.OutcomeFail, autotest.ReasonLeaseExpired},
		{
			"no lease on an addressed interface",
			nil,
			noLease,
			true,
			autotest.OutcomeSkipped,
			autotest.ReasonNoLeaseFound,
		},
		{"no lease, no address", nil, noLease, false, autotest.OutcomeFail, autotest.ReasonNoAddress},
		{"lease unreadable", nil, errors.New("permission denied"), true, autotest.OutcomeFail, autotest.ReasonError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotIface string
			lease := func(iface string) (*dhcp.LeaseInfo, error) {
				gotIface = iface
				return tc.lease, tc.err
			}
			st := autotest.DHCPStage(linkReader(&netif.LinkStatus{Carrier: true, HasIP: tc.hasIP}, nil), lease)
			got := runOne(t, st, autotest.Spec{})
			check(t, got, tc.outcome, tc.reason)
			if gotIface != "eth0" {
				t.Fatalf("lease read for %q, want eth0", gotIface)
			}
		})
	}

	got := runOne(t, autotest.DHCPStage(linkReader(&netif.LinkStatus{}, nil),
		func(string) (*dhcp.LeaseInfo, error) { return &noGateway, nil }), autotest.Spec{})
	if !slices.Contains(got.Readings, autotest.Reading{Label: "address", Value: "10.0.0.5"}) ||
		slices.ContainsFunc(got.Readings, func(d autotest.Reading) bool { return d.Value == "" }) {
		t.Fatalf("details = %+v, want the address and no empty readings", got.Readings)
	}
}

func TestDNSStage(t *testing.T) {
	t.Parallel()

	resolved := func(ms int64, status dns.Status) *dns.LookupResult {
		return &dns.LookupResult{
			Outcome:  dns.OutcomeResolved,
			Status:   status,
			TimeMs:   ms,
			Resolved: []string{"192.0.2.1"},
		}
	}
	failed := &dns.LookupResult{Outcome: dns.OutcomeFailed, Status: dns.StatusError, Error: "timeout"}
	tests := []struct {
		name    string
		res     *dns.TestResult
		outcome autotest.Outcome
		reason  autotest.Reason
	}{
		{
			"resolved",
			&dns.TestResult{Forward: resolved(20, dns.StatusSuccess)},
			autotest.OutcomePass,
			autotest.ReasonOK,
		},
		{
			"IPv6 alone resolving passes",
			&dns.TestResult{Forward: failed, ForwardIPv6: resolved(30, dns.StatusSuccess)},
			autotest.OutcomePass,
			autotest.ReasonOK,
		},
		{"slow", &dns.TestResult{Forward: resolved(300, dns.StatusWarning)}, autotest.OutcomeWarn, autotest.ReasonSlow},
		{
			"no answer",
			&dns.TestResult{Forward: failed, ForwardIPv6: failed},
			autotest.OutcomeFail,
			autotest.ReasonNoAnswer,
		},
		{"no resolvers", &dns.TestResult{}, autotest.OutcomeFail, autotest.ReasonNoResolvers},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			st := autotest.DNSStage(func(context.Context) *dns.TestResult { return tc.res })
			check(t, runOne(t, st, autotest.Spec{}), tc.outcome, tc.reason)
		})
	}
}

func TestGatewayStage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		stats   *gateway.PingStats
		outcome autotest.Outcome
		reason  autotest.Reason
	}{
		{
			"reachable", &gateway.PingStats{Gateway: "10.0.0.1", Reachable: true, Status: gateway.StatusSuccess},
			autotest.OutcomePass, autotest.ReasonOK,
		},
		{
			"lossy",
			&gateway.PingStats{Gateway: "10.0.0.1", Reachable: true, Status: gateway.StatusWarning, LossPercent: 33},
			autotest.OutcomeWarn,
			autotest.ReasonLossy,
		},
		{"unreachable", &gateway.PingStats{
			Gateway: "10.0.0.1", Status: gateway.StatusError,
			Results: []gateway.PingResult{{Error: "ping timeout"}},
		}, autotest.OutcomeFail, autotest.ReasonUnreachable},
		{"no gateway", &gateway.PingStats{Status: gateway.StatusError}, autotest.OutcomeFail, autotest.ReasonNoGateway},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			st := autotest.GatewayStage(func() *gateway.PingStats { return tc.stats })
			check(t, runOne(t, st, autotest.Spec{}), tc.outcome, tc.reason)
		})
	}
}

func TestVLANStage(t *testing.T) {
	t.Parallel()

	v := func(n int) *int { return &n }
	tests := []struct {
		name     string
		native   *int
		expected int
		outcome  autotest.Outcome
		reason   autotest.Reason
	}{
		{"nothing advertised", nil, 0, autotest.OutcomeSkipped, autotest.ReasonNotAdvertised},
		{"nothing advertised to compare", nil, 10, autotest.OutcomeSkipped, autotest.ReasonNotAdvertised},
		{"advertised, no expectation", v(20), 0, autotest.OutcomePass, autotest.ReasonOK},
		{"matches", v(10), 10, autotest.OutcomePass, autotest.ReasonOK},
		{"mismatch", v(20), 10, autotest.OutcomeFail, autotest.ReasonVLANMismatch},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			st := autotest.VLANStage(func() (*int, *int) { return tc.native, nil })
			check(t, runOne(t, st, autotest.Spec{ExpectedVLAN: tc.expected}), tc.outcome, tc.reason)
		})
	}
}

func TestInternetStage(t *testing.T) {
	t.Parallel()

	targets := []autotest.Target{{Name: "A", Host: "192.0.2.1"}, {Name: "B", Host: "192.0.2.2"}}
	ping := func(up ...string) autotest.Pinger {
		return func(_ context.Context, host string) (time.Duration, error) {
			if slices.Contains(up, host) {
				return 1500 * time.Microsecond, nil
			}
			return 0, errors.New("no echo reply")
		}
	}
	tests := []struct {
		name    string
		targets []autotest.Target
		err     error
		ping    autotest.Pinger
		outcome autotest.Outcome
		reason  autotest.Reason
	}{
		{"all answer", targets, nil, ping("192.0.2.1", "192.0.2.2"), autotest.OutcomePass, autotest.ReasonOK},
		{"some answer", targets, nil, ping("192.0.2.2"), autotest.OutcomeWarn, autotest.ReasonPartial},
		{"none answer", targets, nil, ping(), autotest.OutcomeFail, autotest.ReasonUnreachable},
		{"no targets", nil, nil, ping(), autotest.OutcomeSkipped, autotest.ReasonNoTargets},
		{"settings unreadable", nil, errors.New("db closed"), ping(), autotest.OutcomeFail, autotest.ReasonError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			st := autotest.InternetStage(
				func(context.Context) ([]autotest.Target, error) { return tc.targets, tc.err },
				tc.ping,
			)
			check(t, runOne(t, st, autotest.Spec{}), tc.outcome, tc.reason)
		})
	}

	st := autotest.InternetStage(
		func(context.Context) ([]autotest.Target, error) { return targets, nil },
		ping("192.0.2.1"),
	)
	got := runOne(t, st, autotest.Spec{})
	want := []autotest.Reading{
		{Label: "A (192.0.2.1)", Value: "1.5 ms"},
		{Label: "B (192.0.2.2)", Value: "unreachable"},
	}
	if !slices.Equal(got.Readings, want) {
		t.Fatalf("details = %+v, want %+v", got.Readings, want)
	}
}
