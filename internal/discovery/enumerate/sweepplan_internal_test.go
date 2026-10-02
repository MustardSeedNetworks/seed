package enumerate

import (
	"net"
	"net/netip"
	"slices"
	"testing"
)

func addrs(t *testing.T, ss ...string) []netip.Addr {
	t.Helper()
	out := make([]netip.Addr, 0, len(ss))
	for _, s := range ss {
		out = append(out, netip.MustParseAddr(s))
	}
	return out
}

func blockStrings(plan sweepPlan) []string {
	out := make([]string, 0, len(plan.blocks))
	for _, block := range plan.blocks {
		out = append(out, block.String())
	}
	return out
}

// hospitalSummary is the shape behind seed#2832: a learned /16, swept under
// the default one-/24 budget.
func hospitalSummary() netip.Prefix {
	return netip.MustParsePrefix("10.51.0.0/16")
}

// sweepSummary runs one sweep's worth of rotation over the hospitalSummary(), as
// pingSweep does when the sweep succeeds.
func sweepSummary(r *rotation, evidence []netip.Addr) sweepPlan {
	plan := r.planSweep(hospitalSummary(), evidence, DefaultMaxHostsPerSubnet)
	r.commit(hospitalSummary(), plan)
	return plan
}

func TestPlanSweepEvidencedBlocksRotateMostEvidenceFirst(t *testing.T) {
	evidence := addrs(t,
		"10.51.30.2",
		"10.51.200.2", "10.51.200.3", "10.51.200.9",
		"10.51.10.2", "10.51.10.3",
		"10.52.0.1",   // outside the target
		"192.168.1.1", // outside the target
	)
	r := &rotation{}

	var got []string
	for range 4 {
		plan := sweepSummary(r, evidence)
		if plan.probe {
			t.Fatalf("plan probes first hosts only, want whole /24s with evidence present")
		}
		got = append(got, blockStrings(plan)...)
	}

	want := []string{"10.51.200.0/24", "10.51.10.0/24", "10.51.30.0/24", "10.51.200.0/24"}
	if !slices.Equal(got, want) {
		t.Errorf("four sweeps covered %v, want %v", got, want)
	}
}

// A /24 named for the first time goes ahead of every /24 already swept, so
// learning a new network inside the summary does not wait out a full cycle.
func TestPlanSweepNewEvidenceGoesFirst(t *testing.T) {
	r := &rotation{}
	sweepSummary(r, addrs(t, "10.51.10.2", "10.51.10.3"))

	plan := r.planSweep(hospitalSummary(), addrs(t, "10.51.10.2", "10.51.10.3", "10.51.40.7"), DefaultMaxHostsPerSubnet)

	if got := blockStrings(plan); !slices.Equal(got, []string{"10.51.40.0/24"}) {
		t.Errorf("plan = %v, want the newly evidenced 10.51.40.0/24", got)
	}
}

func TestPlanSweepBudgetBoundsWholeBlocks(t *testing.T) {
	evidence := addrs(t, "10.51.1.1", "10.51.2.1", "10.51.3.1")
	cases := []struct {
		maxHosts int
		want     int
	}{
		{DefaultMaxHostsPerSubnet, 1},
		{2 * DefaultMaxHostsPerSubnet, 2},
		{2*DefaultMaxHostsPerSubnet + 1, 3},
		{10 * DefaultMaxHostsPerSubnet, 3}, // only three /24s have evidence
	}
	for _, tc := range cases {
		plan := (&rotation{}).planSweep(hospitalSummary(), evidence, tc.maxHosts)
		if len(plan.blocks) != tc.want || plan.probe {
			t.Errorf("maxHosts %d: plan = %v (probe %v), want %d whole /24s",
				tc.maxHosts, blockStrings(plan), plan.probe, tc.want)
		}
	}
}

// With nothing naming a /24 inside the hospitalSummary(), the sweep probes .1-.4 of as
// many /24s as the budget covers and moves on each sweep, wrapping at the top.
func TestPlanSweepWithoutEvidenceProbesTheFirstHosts(t *testing.T) {
	r := &rotation{}

	first := sweepSummary(r, addrs(t, "10.52.0.1"))

	if !first.probe {
		t.Fatal("plan sweeps whole /24s, want a probe of the first hosts")
	}
	if len(first.blocks) != DefaultMaxHostsPerSubnet/probeHostsPerBlock {
		t.Fatalf("probe covers %d /24s, want %d", len(first.blocks), DefaultMaxHostsPerSubnet/probeHostsPerBlock)
	}
	if got := len(first.hosts()); got > DefaultMaxHostsPerSubnet {
		t.Errorf("probe sends %d pings, over the per-sweep cap of %d", got, DefaultMaxHostsPerSubnet)
	}
	if got := first.blocks[0].String(); got != "10.51.0.0/24" {
		t.Errorf("first probed /24 = %s, want 10.51.0.0/24", got)
	}
	hosts := first.hosts()
	if got := hosts[:5]; !slices.Equal(got, addrs(t, "10.51.0.1", "10.51.0.2", "10.51.0.3", "10.51.0.4", "10.51.1.1")) {
		t.Errorf("first probed hosts = %v, want .1-.4 of each /24 in order", got)
	}

	second := sweepSummary(r, nil)
	if got := second.blocks[0].String(); got != "10.51.63.0/24" {
		t.Errorf("second probe starts at %s, want 10.51.63.0/24", got)
	}

	covered := map[netip.Prefix]bool{}
	for _, plan := range []sweepPlan{first, second} {
		for _, block := range plan.blocks {
			covered[block] = true
		}
	}
	for range 3 {
		for _, block := range sweepSummary(r, nil).blocks {
			covered[block] = true
		}
	}
	if len(covered) != 256 {
		t.Errorf("five probes covered %d of the /16's 256 /24s, want all of them", len(covered))
	}
	if got := r.planSweep(hospitalSummary(), nil, DefaultMaxHostsPerSubnet).blocks[0].String(); got != "10.51.59.0/24" {
		t.Errorf("sixth probe starts at %s, want 10.51.59.0/24 after wrapping", got)
	}
}

// A target the budget covers whole is swept whole, evidence or not.
func TestPlanSweepTargetWithinBudgetIsSweptWhole(t *testing.T) {
	plan := (&rotation{}).planSweep(netip.MustParsePrefix("10.51.6.0/23"), nil, 2*DefaultMaxHostsPerSubnet)

	if got := blockStrings(plan); plan.probe || !slices.Equal(got, []string{"10.51.6.0/24", "10.51.7.0/24"}) {
		t.Errorf("plan = %v (probe %v), want both /24s whole", got, plan.probe)
	}
	hosts := plan.hosts()
	if len(hosts) != 2*hostsPerSubnet24 || hosts[0] != netip.MustParseAddr("10.51.6.1") ||
		hosts[len(hosts)-1] != netip.MustParseAddr("10.51.7.254") {
		t.Errorf("hosts = %d from %v to %v, want 508 from 10.51.6.1 to 10.51.7.254",
			len(hosts), hosts[0], hosts[len(hosts)-1])
	}
}

// A sweep that could not run commits nothing, so the rotation repeats it.
func TestPlanSweepWithoutCommitRepeats(t *testing.T) {
	r := &rotation{}
	evidence := addrs(t, "10.51.1.1", "10.51.2.1", "10.51.2.2")

	first := r.planSweep(hospitalSummary(), evidence, DefaultMaxHostsPerSubnet)
	again := r.planSweep(hospitalSummary(), evidence, DefaultMaxHostsPerSubnet)
	if !slices.Equal(blockStrings(first), blockStrings(again)) {
		t.Errorf("uncommitted plans differ: %v then %v", blockStrings(first), blockStrings(again))
	}
	first = r.planSweep(hospitalSummary(), nil, DefaultMaxHostsPerSubnet)
	again = r.planSweep(hospitalSummary(), nil, DefaultMaxHostsPerSubnet)
	if first.blocks[0] != again.blocks[0] {
		t.Errorf("uncommitted probes start at %v then %v", first.blocks[0], again.blocks[0])
	}
}

func TestIPv4Prefix(t *testing.T) {
	cases := []struct {
		name   string
		subnet *net.IPNet
		want   string
	}{
		{"nil", nil, ""},
		{"ipv4", mustCIDR(t, "10.51.0.0/16"), "10.51.0.0/16"},
		{"ipv4 unmasked", &net.IPNet{IP: net.ParseIP("10.51.3.9"), Mask: net.CIDRMask(16, 32)}, "10.51.0.0/16"},
		{"ipv6", mustCIDR(t, "2001:db8::/32"), ""},
	}
	for _, tc := range cases {
		got, ok := ipv4Prefix(tc.subnet)
		if (tc.want == "") == ok || (ok && got.String() != tc.want) {
			t.Errorf("%s: ipv4Prefix = %v, %v; want %q", tc.name, got, ok, tc.want)
		}
	}
}

// A target that stays configured keeps its place in the rotation; one that is
// removed does not leave its state behind.
func TestSetTargetNetworksKeepsRotationOfKeptTargets(t *testing.T) {
	s := NewARPScanner("", nil)
	if err := s.SetTargetNetworks([]string{"10.51.0.0/16", "10.60.0.0/16"}); err != nil {
		t.Fatal(err)
	}
	kept := &rotation{probeNext: 63}
	s.rotations = map[netip.Prefix]*rotation{
		hospitalSummary():                     kept,
		netip.MustParsePrefix("10.60.0.0/16"): {probeNext: 5},
	}

	if err := s.SetTargetNetworks([]string{"10.51.0.0/16", "10.51.200.0/24"}); err != nil {
		t.Fatal(err)
	}

	if len(s.rotations) != 1 || s.rotations[hospitalSummary()] != kept {
		t.Errorf("rotations = %v, want only the kept 10.51.0.0/16 one", s.rotations)
	}
}
