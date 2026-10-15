package ipam_test

import (
	"net/netip"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/ipam"
)

func scope(start, end string) ipam.Scope {
	return ipam.Scope{Start: netip.MustParseAddr(start), End: netip.MustParseAddr(end)}
}

func TestParseScope(t *testing.T) {
	for name, tc := range map[string]struct {
		start, end string
		want       ipam.Scope
		wantErr    bool
	}{
		"an ascending IPv4 range":         {start: "10.44.20.100", end: "10.44.20.199", want: scope("10.44.20.100", "10.44.20.199")},
		"a single address":                {start: "10.44.20.5", end: "10.44.20.5", want: scope("10.44.20.5", "10.44.20.5")},
		"a range across an octet":         {start: "10.44.20.250", end: "10.44.21.10", want: scope("10.44.20.250", "10.44.21.10")},
		"the ends reversed":               {start: "10.44.20.199", end: "10.44.20.100", wantErr: true},
		"an IPv6 pool":                    {start: "2001:db8::10", end: "2001:db8::ff", wantErr: true},
		"an IPv4 start with an IPv6 end":  {start: "10.44.20.1", end: "2001:db8::ff", wantErr: true},
		"not an address":                  {start: "10.44.20", end: "10.44.20.9", wantErr: true},
		"an empty end":                    {start: "10.44.20.1", wantErr: true},
		"an IPv4-mapped IPv6 address end": {start: "10.44.20.1", end: "::ffff:10.44.20.9", wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := ipam.ParseScope("", tc.start, tc.end)
			if tc.wantErr {
				require.ErrorIs(t, err, ipam.ErrInvalidScope)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestScopeSizeAndOverlap(t *testing.T) {
	require.Equal(t, uint64(100), scope("10.44.20.100", "10.44.20.199").Size())
	require.Equal(t, uint64(17), scope("10.44.20.250", "10.44.21.10").Size())
	require.Equal(t, uint64(1)<<32, scope("0.0.0.0", "255.255.255.255").Size())

	pool := scope("10.44.20.100", "10.44.20.199")
	for other, want := range map[ipam.Scope]bool{
		scope("10.44.20.199", "10.44.20.210"): true,
		scope("10.44.20.1", "10.44.20.100"):   true,
		scope("10.44.20.120", "10.44.20.130"): true,
		scope("10.44.20.1", "10.44.20.99"):    false,
		scope("10.44.20.200", "10.44.20.250"): false,
	} {
		require.Equal(t, want, pool.Overlaps(other), "%v-%v", other.Start, other.End)
		require.Equal(t, want, other.Overlaps(pool), "%v-%v", other.Start, other.End)
	}
}

// The acceptance of P-D5's second half: a scope's count is a hand-count of
// the addresses seen inside its range, and the levels follow from it.
func TestScopeUtilization(t *testing.T) {
	ten := scope("10.44.20.100", "10.44.20.109")
	usages := ipam.ScopeUtilization([]ipam.Scope{ten, scope("10.44.30.1", "10.44.30.4")},
		addrs("10.44.20.99", "10.44.20.100", "10.44.20.105", "10.44.20.105", "10.44.20.109", "10.44.20.110",
			"10.44.30.1", "10.44.30.2", "10.44.30.3", "10.44.30.4", "2001:db8::1"))

	require.Equal(t, []ipam.ScopeUsage{
		{Scope: ten, InUse: addrs("10.44.20.100", "10.44.20.105", "10.44.20.109")},
		{
			Scope: scope("10.44.30.1", "10.44.30.4"),
			InUse: addrs("10.44.30.1", "10.44.30.2", "10.44.30.3", "10.44.30.4"),
		},
	}, usages)
	require.Equal(t, uint64(7), usages[0].Free())
	require.InDelta(t, 30.0, usages[0].Percent(), 1e-9)
	require.Equal(t, ipam.ExhaustionNone, usages[0].Exhaustion())
	require.Equal(t, uint64(0), usages[1].Free())
	require.Equal(t, ipam.ExhaustionFull, usages[1].Exhaustion())
}

func TestScopeExhaustion(t *testing.T) {
	ten := scope("10.44.20.100", "10.44.20.109")
	for used, want := range map[int]ipam.Exhaustion{
		0:  ipam.ExhaustionNone,
		8:  ipam.ExhaustionNone,
		9:  ipam.ExhaustionNear,
		10: ipam.ExhaustionFull,
	} {
		t.Run(strconv.Itoa(used), func(t *testing.T) {
			usage := ipam.ScopeUsage{Scope: ten, InUse: make([]netip.Addr, used)}
			require.Equal(t, want, usage.Exhaustion())
		})
	}
}

func TestWatchReportsEachClimbOnce(t *testing.T) {
	ten := scope("10.44.20.100", "10.44.20.109")
	usage := func(used int) ipam.ScopeUsage {
		return ipam.ScopeUsage{Scope: ten, InUse: make([]netip.Addr, used)}
	}
	var w ipam.Watch

	steps := []struct {
		used   int
		rising bool
	}{
		{used: 3, rising: false},  // a scope with room says nothing
		{used: 9, rising: true},   // near
		{used: 9, rising: false},  // still near: already reported
		{used: 10, rising: true},  // full is a further climb
		{used: 9, rising: false},  // falling back is not news
		{used: 10, rising: true},  // full again after falling back
		{used: 2, rising: false},  // recovered
		{used: 10, rising: true},  // straight to full
		{used: -1, rising: false}, // the scope was deleted
		{used: 10, rising: true},  // re-entered: reported afresh
	}
	for i, step := range steps {
		var usages []ipam.ScopeUsage
		if step.used >= 0 {
			usages = []ipam.ScopeUsage{usage(step.used)}
		}
		got := w.Rising(usages)
		if step.rising {
			require.Equal(t, usages, got, "step %d", i)
		} else {
			require.Empty(t, got, "step %d", i)
		}
	}
}
