//go:build windows

package enumerate

import "testing"

// Captured verbatim from `Get-NetNeighbor -AddressFamily IPv6 | ConvertTo-Csv`
// on dev-win11 (Windows 11, 10.0.26200). Real output rather than an invented
// shape, so the parser is tested against what Windows actually emits.
const realNetNeighborCSV = `"InterfaceAlias","IPAddress","LinkLayerAddress","State"
"Bluetooth Network Connection","ff02::1:2","33-33-00-01-00-02","Permanent"
"Bluetooth Network Connection","fe80::2b19:2dd6:5f8a:96e0","00-00-00-00-00-00","Unreachable"
"Ethernet 2","ff02::1:ff8a:96e0","33-33-FF-8A-96-E0","Permanent"
"Ethernet 2","fe80::76ac:b9ff:fe3b:af40","74-AC-B9-3B-AF-40","Stale"
"Ethernet 2","fe80::2b19:2dd6:5f8a:96e0","BC-24-11-0B-21-B2","Stale"
"Local Area Connection* 10","fe80::2b19:2dd6:5f8a:96e0","00-00-00-00-00-00","Unreachable"
`

func TestParseNetNeighbors_ReadsTheTableNotTheHost(t *testing.T) {
	got := parseNetNeighbors(realNetNeighborCSV, "Ethernet 2", nil)

	// Two unicast neighbours on that interface; the ff02:: entry is a multicast
	// group membership, not a host.
	if len(got) != 2 {
		t.Fatalf("neighbours = %d, want 2: %+v", len(got), got)
	}

	byIP := map[string]NDPNeighbor{}
	for _, n := range got {
		byIP[n.IPv6] = n
	}

	// The defect this replaces reported the host's own addresses with the local
	// MAC. This address belongs to another host and carries its own MAC.
	peer, ok := byIP["fe80::76ac:b9ff:fe3b:af40"]
	if !ok {
		t.Fatalf("real neighbour missing: %+v", got)
	}
	if peer.MAC != "74:ac:b9:3b:af:40" {
		t.Errorf("MAC = %q, want the neighbour's own address 74:ac:b9:3b:af:40", peer.MAC)
	}
	if peer.State != "STALE" {
		t.Errorf("State = %q, want STALE (the NUD vocabulary Linux reports)", peer.State)
	}
	if peer.IsRouter {
		t.Error("IsRouter = true with no default route supplied")
	}
}

func TestParseNetNeighbors_FiltersOtherInterfaces(t *testing.T) {
	// The same link-local address appears on three interfaces. Asking for one
	// must not return the others' entries.
	got := parseNetNeighbors(realNetNeighborCSV, "Local Area Connection* 10", nil)

	if len(got) != 1 {
		t.Fatalf("neighbours = %d, want 1: %+v", len(got), got)
	}
	// Windows reports all-zero for an unresolved entry; Linux yields empty for
	// the same state, and a caller should not have to know which platform spoke.
	if got[0].MAC != "" {
		t.Errorf("MAC = %q, want empty for an unresolved neighbour", got[0].MAC)
	}
	if got[0].State != "FAILED" {
		t.Errorf("State = %q, want FAILED for Windows 'Unreachable'", got[0].State)
	}
}

func TestParseNetNeighbors_MarksDefaultRouteNextHopsAsRouters(t *testing.T) {
	routers := map[string]bool{"fe80::76ac:b9ff:fe3b:af40": true}
	got := parseNetNeighbors(realNetNeighborCSV, "Ethernet 2", routers)

	var seen bool
	for _, n := range got {
		if n.IPv6 == "fe80::76ac:b9ff:fe3b:af40" {
			seen = true
			if !n.IsRouter {
				t.Error("a default-route next hop must be flagged as a router")
			}
		}
	}
	if !seen {
		t.Fatal("router candidate not returned at all")
	}
}

func TestWindowsNeighborState_MapsOntoTheNUDVocabulary(t *testing.T) {
	for in, want := range map[string]string{
		"Reachable": "REACHABLE", "Stale": "STALE", "Permanent": "PERMANENT",
		"Unreachable": "FAILED", "Incomplete": "INCOMPLETE", "Probe": "PROBE",
		"Delay": "DELAY", "something-new": "UNKNOWN",
	} {
		if got := windowsNeighborState(in); got != want {
			t.Errorf("windowsNeighborState(%q) = %q, want %q", in, got, want)
		}
	}
}
