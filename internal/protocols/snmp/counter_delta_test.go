package snmp_test

import (
	"math"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

func TestCounter32Delta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		prev, cur uint64
		prevUp    uint32
		curUp     uint32
		wantDelta uint64
		wantOK    bool
	}{
		{name: "advance", prev: 100, cur: 160, prevUp: 1000, curUp: 7000, wantDelta: 60, wantOK: true},
		{name: "unchanged", prev: 1_000_000, cur: 1_000_000, prevUp: 1000, curUp: 7000, wantDelta: 0, wantOK: true},
		{
			name:      "wrap through 2^32",
			prev:      math.MaxUint32 - 4,
			cur:       30,
			prevUp:    1000,
			curUp:     7000,
			wantDelta: 35,
			wantOK:    true,
		},
		{
			name:      "wrap from max to zero",
			prev:      math.MaxUint32,
			cur:       0,
			prevUp:    1000,
			curUp:     7000,
			wantDelta: 1,
			wantOK:    true,
		},
		{name: "agent restarted", prev: 1_000_000, cur: 20, prevUp: 9_000_000, curUp: 500},
		{name: "agent restarted and counter grew past prev", prev: 10, cur: 900, prevUp: 9_000_000, curUp: 500},
		{name: "prev wider than 32 bits", prev: math.MaxUint32 + 1, cur: 5, prevUp: 1000, curUp: 7000},
		{name: "cur wider than 32 bits", prev: 5, cur: math.MaxUint32 + 1, prevUp: 1000, curUp: 7000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			delta, ok := snmp.Counter32Delta(tt.prev, tt.cur, tt.prevUp, tt.curUp)
			if ok != tt.wantOK || delta != tt.wantDelta {
				t.Fatalf("Counter32Delta(%d, %d, %d, %d) = (%d, %v), want (%d, %v)",
					tt.prev, tt.cur, tt.prevUp, tt.curUp, delta, ok, tt.wantDelta, tt.wantOK)
			}
		})
	}
}
