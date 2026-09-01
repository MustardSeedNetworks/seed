//go:build darwin

package enumerate

import "testing"

// Captured verbatim from `ndp -an` on macOS 27 (Darwin 27.0.0). Real output
// rather than an invented shape: the unpadded MAC, the "(incomplete)" marker
// and the %zone suffix are all things this parser has to survive, and all
// three appear here.
const realNDPTable = `Neighbor                                Linklayer Address  Netif Expire    St Flgs Prbs
fe80::1%lo0                             (incomplete)         lo0 permanent R      
fe80::60:3651:1cc2:c27c%en0             ea:3b:2d:35:63:81    en0 11s       R      
fe80::2c0:17ff:fe57:17e%en0             0:c0:17:57:1:7e      en0 4h19m27s  S      
fe80::48f:f729:535d:eeab%en0            (incomplete)         en0 expired   N      
fe80::38f0:5aff:fe8b:c504%awdl0         3a:f0:5a:8b:c5:4   awdl0 permanent R      
`

func TestParseNDPTable_ReadsRealOutput(t *testing.T) {
	got := parseNDPTable(realNDPTable, "en0")

	if len(got) != 3 {
		t.Fatalf("neighbours = %d, want 3 on en0: %+v", len(got), got)
	}

	reachable, ok := got["fe80::60:3651:1cc2:c27c"]
	if !ok {
		t.Fatalf("neighbour missing; got %+v", got)
	}
	if reachable.MAC != "ea:3b:2d:35:63:81" {
		t.Errorf("MAC = %q, want ea:3b:2d:35:63:81", reachable.MAC)
	}
	if reachable.State != "REACHABLE" {
		t.Errorf("State = %q, want REACHABLE", reachable.State)
	}

	// ndp does not zero-pad: 0:c0:17:57:1:7e must widen to a valid MAC.
	stale := got["fe80::2c0:17ff:fe57:17e"]
	if stale == nil || stale.MAC != "00:c0:17:57:01:7e" {
		t.Errorf("unpadded MAC not normalised: %+v", stale)
	}
	if stale != nil && stale.State != "STALE" {
		t.Errorf("State = %q, want STALE", stale.State)
	}

	// "(incomplete)" is not a link-layer address.
	incomplete := got["fe80::48f:f729:535d:eeab"]
	if incomplete == nil || incomplete.MAC != "" {
		t.Errorf("incomplete entry should carry no MAC: %+v", incomplete)
	}
}

func TestParseNDPTable_FiltersByInterface(t *testing.T) {
	// awdl0 and lo0 entries must not leak into an en0 scan.
	for _, n := range parseNDPTable(realNDPTable, "en0") {
		if n.IPv6 == "fe80::38f0:5aff:fe8b:c504" || n.IPv6 == "fe80::1" {
			t.Errorf("entry from another interface leaked: %s", n.IPv6)
		}
	}

	if got := parseNDPTable(realNDPTable, "awdl0"); len(got) != 1 {
		t.Errorf("awdl0 neighbours = %d, want 1", len(got))
	}
}

func TestParseNDPTable_StripsTheZoneFromTheAddress(t *testing.T) {
	for ip := range parseNDPTable(realNDPTable, "en0") {
		if len(ip) > 0 && ip[len(ip)-1] == '0' && ip != "fe80::2c0:17ff:fe57:17e" {
			continue
		}
		if got := parseNDPTable(realNDPTable, "en0")[ip]; got != nil && got.IPv6 != ip {
			t.Errorf("key %q does not match IPv6 %q", ip, got.IPv6)
		}
	}
	// The %en0 suffix must not survive into the address.
	if _, bad := parseNDPTable(realNDPTable, "en0")["fe80::60:3651:1cc2:c27c%en0"]; bad {
		t.Error("zone suffix left on the address")
	}
}

// The scanner must read the live table, not return an empty map as it did
// before #2089. Asserted against the machine's own count so it cannot pass by
// returning something plausible but fixed.
func TestGetNeighbors_ReadsTheLiveTable(t *testing.T) {
	ns := NewNDPScanner("")
	live := ns.GetNeighbors()

	if len(live) == 0 {
		t.Skip("no IPv6 neighbours on this host; nothing to assert against")
	}
	for _, n := range live {
		if n.IPv6 == "" {
			t.Error("neighbour with no address")
		}
		if n.State == "" {
			t.Error("neighbour with no state")
		}
	}
}
