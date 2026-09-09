package enumerate

import "testing"

// TestMergeNDPNeighborsSkipsUnresolved covers the second half of #2337: a
// neighbour whose link layer never resolved is not a device. Every such row
// normalises to the same empty MAC, so without a guard they collapse into one
// phantom device that carries IPv6 addresses from unrelated interfaces.
func TestMergeNDPNeighborsSkipsUnresolved(t *testing.T) {
	d := &DeviceDiscovery{devices: map[string]*DiscoveredDevice{}}

	d.mergeNDPNeighbors(map[string]*NDPNeighbor{
		"fe80::1":  {IPv6: "fe80::1", MAC: "", IsRouter: true},
		"fe80::2":  {IPv6: "fe80::2", MAC: ""},
		"fe80::99": {IPv6: "fe80::99", MAC: "aa:bb:cc:dd:ee:ff"},
	})

	if _, phantom := d.devices[""]; phantom {
		t.Errorf("unresolved neighbours created a device keyed on an empty MAC: %+v", d.devices[""])
	}

	if len(d.devices) != 1 {
		t.Fatalf("want only the resolved neighbour, got %d devices: %+v", len(d.devices), d.devices)
	}

	if got := d.devices["AA:BB:CC:DD:EE:FF"]; got == nil || got.IPv6Address != "fe80::99" {
		t.Errorf("the resolved neighbour did not survive the merge: %+v", d.devices)
	}
}
