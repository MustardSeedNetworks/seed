package enumerate_test

import (
	_ "embed"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// LLDP fixtures, built by testdata/generate_lldp.go.
//
// Unlike the CDP and EDP frames beside them these are generated rather than
// captured, so the tests below assert decoded values field by field rather than
// trusting the bytes because we wrote them.
var (
	//go:embed testdata/lldp_untagged.bin
	lldpUntagged []byte

	//go:embed testdata/lldp_dot1q_vlan300.bin
	lldpTaggedVLAN300 []byte

	//go:embed testdata/lldp_minimal.bin
	lldpMinimal []byte
)

// collectLLDP replays frames and waits for the capture to decode them.
func collectLLDP(t *testing.T, frames ...[]byte) []*enumerate.LLDPNeighbor {
	t.Helper()

	lldp := enumerate.NewLLDPCapture(&replayOpener{frames: frames}, "eth0")
	if err := lldp.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(lldp.Stop)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if neighbors := lldp.GetNeighbors(); len(neighbors) > 0 {
			return neighbors
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no LLDP neighbour decoded within 5s")
	return nil
}

// TestLLDPCaptureDecodesAdvertisement covers processPacket, which had no test
// at all: every field seed surfaces from an advertisement, asserted against the
// fixture that produced it.
func TestLLDPCaptureDecodesAdvertisement(t *testing.T) {
	t.Parallel()

	neighbors := collectLLDP(t, lldpUntagged)
	if len(neighbors) != 1 {
		t.Fatalf("decoded %d neighbours, want 1", len(neighbors))
	}
	n := neighbors[0]

	for _, tc := range []struct{ field, got, want string }{
		// Chassis subtype 4 is a MAC, and it is the same address SourceMAC
		// reports below. It used to be pinned here as the raw bytes it arrives
		// as -- "\x00\x1bT\xc1>\x0f" -- which is #1932.
		{"ChassisID", n.ChassisID, "00:1b:54:c1:3e:0f"},
		{"PortID", n.PortID, "GigabitEthernet1/0/24"},
		{"PortDescription", n.PortDescription, "Uplink to core"},
		{"SystemName", n.SystemName, "access-sw-01"},
		{
			"SystemDescription", n.SystemDescription,
			"Cisco IOS Software, C2960X Software, Version 15.2(7)E3",
		},
		{"ManagementAddress", n.ManagementAddress, "10.44.20.5"},
		{"SourceMAC", n.SourceMAC, "00:1b:54:c1:3e:0f"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.field, tc.got, tc.want)
		}
	}

	if n.TTL != 120 {
		t.Errorf("TTL = %d, want 120", n.TTL)
	}

	// The fixture advertises bridge and router as supported. Both must survive
	// the bitmap decode — parseSystemCapabilities was also entirely untested.
	for _, want := range []string{"Bridge", "Router"} {
		if !slices.Contains(n.SystemCapabilities, want) {
			t.Errorf("SystemCapabilities %v is missing %q", n.SystemCapabilities, want)
		}
	}
}

// TestLLDPCaptureDecodesVLANTaggedFrames is the LLDP counterpart of the CDP
// case in seed#1922: an advertisement on an 802.1Q trunk must be discovered,
// and the VLAN it arrived on must be recorded. On a trunk that tag is the only
// way to tell which VLAN a neighbour advertises into.
func TestLLDPCaptureDecodesVLANTaggedFrames(t *testing.T) {
	t.Parallel()

	neighbors := collectLLDP(t, lldpTaggedVLAN300)
	if len(neighbors) != 1 {
		t.Fatalf("decoded %d neighbours, want 1", len(neighbors))
	}
	n := neighbors[0]

	if n.ObservedVLAN != 300 {
		t.Errorf("ObservedVLAN = %d, want 300", n.ObservedVLAN)
	}
	if n.SystemName != "access-sw-01" {
		t.Errorf("SystemName = %q; the tagged frame decoded differently from "+
			"the untagged one", n.SystemName)
	}
}

// TestLLDPCaptureAcceptsMandatoryTLVsOnly pins that a sparse advertisement is
// decoded rather than rejected. Plenty of embedded devices send only chassis,
// port and TTL, and a neighbour with no system name is still a neighbour.
func TestLLDPCaptureAcceptsMandatoryTLVsOnly(t *testing.T) {
	t.Parallel()

	neighbors := collectLLDP(t, lldpMinimal)
	if len(neighbors) != 1 {
		t.Fatalf("decoded %d neighbours, want 1", len(neighbors))
	}
	n := neighbors[0]

	if n.PortID != "e1" {
		t.Errorf("PortID = %q, want %q", n.PortID, "e1")
	}
	if n.TTL != 30 {
		t.Errorf("TTL = %d, want 30", n.TTL)
	}
	// Absent optional TLVs must be empty, not garbage read past the end.
	if n.SystemName != "" || n.PortDescription != "" || n.ManagementAddress != "" {
		t.Errorf("optional fields are populated from a frame that carries none: "+
			"SystemName=%q PortDescription=%q ManagementAddress=%q",
			n.SystemName, n.PortDescription, n.ManagementAddress)
	}
	if len(n.SystemCapabilities) != 0 {
		t.Errorf("SystemCapabilities = %v, want none", n.SystemCapabilities)
	}
}

// TestLLDPCaptureSurvivesMalformedFrames is the robustness criterion. A decoder
// reading unauthenticated broadcast traffic must not panic or invent a
// neighbour from rubbish, and must still read the good frame behind it.
func TestLLDPCaptureSurvivesMalformedFrames(t *testing.T) {
	t.Parallel()

	truncated := lldpUntagged[:20]
	// An LLDP ethertype with nothing behind it.
	headerOnly := append([]byte{}, lldpUntagged[:14]...)
	// A TLV claiming far more length than the frame holds.
	overrun := append(append([]byte{}, lldpUntagged[:14]...), 0x02, 0xFF, 0x01, 0x02)

	neighbors := collectLLDP(t, truncated, headerOnly, overrun, lldpUntagged)

	if len(neighbors) != 1 {
		t.Fatalf("decoded %d neighbours, want 1 — malformed frames should be "+
			"dropped, not turned into neighbours", len(neighbors))
	}
	if neighbors[0].SystemName != "access-sw-01" {
		t.Errorf("SystemName = %q; the good frame behind the malformed ones was "+
			"not decoded", neighbors[0].SystemName)
	}
}
