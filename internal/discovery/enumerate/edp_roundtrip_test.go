package enumerate_test

import (
	_ "embed"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// A real EDP advertisement as niac emits it: 802.3 + LLC/SNAP (OUI 00:E0:2B,
// PID 0x00BB) + the 16-byte EDP header + 0x99-marker TLVs. Wireshark decodes
// this exact frame as eth:llc:edp with a good checksum.
//
//go:embed testdata/edp_llcsnap.bin
var edpLLCSNAPFrame []byte

// TestEDPCaptureParsesLLCSNAPFrame pins seed#1937. The parser previously read a
// 2-byte version and a machine-ID *length* at offsets that do not exist in the
// real header, and conditionally skipped 5 bytes of SNAP that gopacket had
// already decoded — so it read none of these frames.
func TestEDPCaptureParsesLLCSNAPFrame(t *testing.T) {
	t.Parallel()

	opener := &replayOpener{frames: [][]byte{edpLLCSNAPFrame}}
	edp := enumerate.NewEDPCapture(opener, "eth0")

	if err := edp.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(edp.Stop)

	var neighbors []*enumerate.EDPNeighbor
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if neighbors = edp.GetNeighbors(); len(neighbors) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(neighbors) == 0 {
		t.Fatal("no EDP neighbour parsed from a correctly framed advertisement")
	}
	if got := neighbors[0].DisplayName; got != "Switch-1 (switch)" {
		t.Errorf("DisplayName = %q, want %q", got, "Switch-1 (switch)")
	}
	if got := neighbors[0].MachineID; got != "00:1a:2b:3c:4d:5e" {
		t.Errorf("MachineID = %q, want the header MAC", got)
	}
}
