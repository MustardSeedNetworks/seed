package enumerate_test

import (
	_ "embed"
	"io"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// Two CDPv2 advertisements from the lab trunk, captured on CT313 eth0 with
//
//	tcpdump -i eth0 -s 0 'ether dst 01:00:0c:cc:cc:cc and vlan'
//
// and stored as raw Ethernet frames rather than a pcap savefile so the test
// needs no pcap reader: gopacket/pcapgo is off-limits here under the capture-port
// confinement rule. Both carry the same advertisement on a different VLAN.
var (
	//go:embed testdata/cdp_dot1q_vlan200.bin
	cdpTaggedVLAN200 []byte

	//go:embed testdata/cdp_dot1q_vlan203.bin
	cdpTaggedVLAN203 []byte
)

// replayHandle serves a fixed set of frames, then reports [io.EOF]. It ignores
// the BPF filter: kernel filtering is exercised in internal/capture/pcap, the
// only package that links libpcap.
type replayHandle struct {
	frames [][]byte
	next   int
}

func (h *replayHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	if h.next >= len(h.frames) {
		return nil, gopacket.CaptureInfo{}, io.EOF
	}
	frame := h.frames[h.next]
	h.next++
	return frame, gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: len(frame),
		Length:        len(frame),
	}, nil
}

func (h *replayHandle) SetBPFFilter(string) error { return nil }
func (h *replayHandle) LinkType() layers.LinkType { return layers.LinkTypeEthernet }
func (h *replayHandle) Close()                    {}

type replayOpener struct{ frames [][]byte }

func (o *replayOpener) OpenLive(string, int32, bool, time.Duration) (capture.Handle, error) {
	return &replayHandle{frames: o.frames}, nil
}

// TestCDPCaptureDecodesVLANTaggedFrames pins seed#1922: CDP advertisements on an
// 802.1Q trunk must be discovered.
//
// gopacket's Dot1Q layer passes its inner type field straight to
// EthernetType.LayerType(). The Ethernet layer treats a value below 0x0600 as an
// 802.3 length and continues into LLC, but Dot1Q has no such branch, so a tagged
// frame carrying LLC/SNAP — which is how CDP is framed — stopped at Dot1Q with
// its payload undecoded, and no neighbour was ever recorded.
func TestCDPCaptureDecodesVLANTaggedFrames(t *testing.T) {
	t.Parallel()

	opener := &replayOpener{frames: [][]byte{cdpTaggedVLAN203, cdpTaggedVLAN200}}
	cdp := enumerate.NewCDPCapture(opener, "eth0")

	if err := cdp.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(cdp.Stop)

	var neighbors []*enumerate.CDPNeighbor
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if neighbors = cdp.GetNeighbors(); len(neighbors) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(neighbors) == 0 {
		t.Fatal("no CDP neighbour discovered from 802.1Q-tagged frames")
	}
	if got := neighbors[0].DeviceID; got != "LAB-EDGE-R1" {
		t.Errorf("DeviceID = %q, want %q", got, "LAB-EDGE-R1")
	}
	// The fixture's later TLVs — PortID, Platform, Version — are not asserted:
	// the emitter that produced it writes the IPv4 NLPID (0xcc) into the Address
	// TLV's protocol-type field, where the protocol type NLPID (0x01) belongs, so
	// a conforming decoder abandons the TLV walk after DeviceID. That is a defect
	// in the emitter (niac#1323), not in this capture path.
	if got := neighbors[0].SourceMAC; got != "00:00:0c:00:01:01" {
		t.Errorf("SourceMAC = %q, want %q", got, "00:00:0c:00:01:01")
	}
	// The fixtures are VLAN 203 and 200; whichever arrived last wins the map
	// entry, but it must be one of them and never zero — a trunk neighbour with
	// no recorded VLAN is the gap #1929 closed.
	if got := neighbors[0].ObservedVLAN; got != 203 && got != 200 {
		t.Errorf("ObservedVLAN = %d, want 203 or 200", got)
	}
}

// TestVLANStripperLeavesUntaggedFramesIntact guards the other direction: the
// stripper must not disturb frames that carry no tag.
func TestVLANStripperLeavesUntaggedFramesIntact(t *testing.T) {
	t.Parallel()

	untagged := make([]byte, 0, len(cdpTaggedVLAN200)-4)
	untagged = append(untagged, cdpTaggedVLAN200[:12]...)
	untagged = append(untagged, cdpTaggedVLAN200[16:]...)

	opener := &replayOpener{frames: [][]byte{untagged}}
	cdp := enumerate.NewCDPCapture(opener, "eth0")

	if err := cdp.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(cdp.Stop)

	var neighbors []*enumerate.CDPNeighbor
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if neighbors = cdp.GetNeighbors(); len(neighbors) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(neighbors) == 0 {
		t.Fatal("untagged CDP frame was not discovered")
	}
	if got := neighbors[0].DeviceID; got != "LAB-EDGE-R1" {
		t.Errorf("DeviceID = %q, want %q", got, "LAB-EDGE-R1")
	}
	if got := neighbors[0].ObservedVLAN; got != enumerate.VLANUntagged {
		t.Errorf("ObservedVLAN = %d, want VLANUntagged for a frame with no tag", got)
	}
}
