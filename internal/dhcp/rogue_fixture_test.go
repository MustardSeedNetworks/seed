package dhcp_test

import (
	_ "embed"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/dhcp"
)

// Rogue DHCP detection had nine tests and not one fed a packet through the
// parser (#498) — every scenario the detector exists for was unexercised, and
// DHCP server-identifier extraction had no coverage at all.
//
// These replay captured-shape DHCP OFFER frames through the real parse path,
// with no network and no privileges, following the pattern in
// internal/discovery/enumerate/testdata.
//
// Regenerating the fixtures: see testdata/README.md.

//go:embed testdata/dhcp_offer_authorized.bin
var offerAuthorized []byte

//go:embed testdata/dhcp_offer_rogue.bin
var offerRogue []byte

//go:embed testdata/dhcp_offer_no_serverid.bin
var offerNoServerID []byte

//go:embed testdata/dhcp_offer_truncated.bin
var offerTruncated []byte

const (
	authorizedIP = "192.0.2.1"
	rogueIP      = "192.0.2.66"
	noServerIDIP = "192.0.2.77"
)

func decode(t *testing.T, frame []byte) gopacket.Packet {
	t.Helper()
	return gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)
}

func detectorWithKnown(known ...string) *dhcp.RogueDetector {
	return dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
		Interface:        "test0",
		KnownServers:     known,
		AlertOnDetection: true,
	})
}

func detectedIPs(servers []*dhcp.RogueServer) []string {
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		out = append(out, s.IP)
	}
	return out
}

func TestRogueDetectorAuthorizedOfferRaisesNoAlert(t *testing.T) {
	rd := detectorWithKnown(authorizedIP)
	rd.RogueProcessPacket(decode(t, offerAuthorized))

	if got := detectedIPs(rd.GetDetectedServers()); len(got) != 1 || got[0] != authorizedIP {
		t.Fatalf("detected = %v, want exactly [%s]", got, authorizedIP)
	}
	if rogues := rd.GetRogueServers(); len(rogues) != 0 {
		t.Errorf("an authorised server was reported rogue: %v", detectedIPs(rogues))
	}
}

func TestRogueDetectorUnknownOfferIsRogue(t *testing.T) {
	rd := detectorWithKnown(authorizedIP)
	rd.RogueProcessPacket(decode(t, offerRogue))

	rogues := detectedIPs(rd.GetRogueServers())
	if len(rogues) != 1 || rogues[0] != rogueIP {
		t.Fatalf("rogues = %v, want exactly [%s]", rogues, rogueIP)
	}
}

func TestRogueDetectorFlagsOnlyTheUnauthorizedServer(t *testing.T) {
	rd := detectorWithKnown(authorizedIP)
	rd.RogueProcessPacket(decode(t, offerAuthorized))
	rd.RogueProcessPacket(decode(t, offerRogue))

	if got := len(rd.GetDetectedServers()); got != 2 {
		t.Fatalf("detected %d servers, want 2", got)
	}
	rogues := detectedIPs(rd.GetRogueServers())
	if len(rogues) != 1 || rogues[0] != rogueIP {
		t.Errorf("rogues = %v, want exactly [%s] — the authorised server must not be flagged",
			rogues, rogueIP)
	}
}

func TestRogueDetectorStopsAlertingOnceServerIsAuthorized(t *testing.T) {
	rd := detectorWithKnown(authorizedIP)
	rd.RogueProcessPacket(decode(t, offerRogue))
	if len(rd.GetRogueServers()) != 1 {
		t.Fatalf("expected the server to start out rogue")
	}

	// Operator adds it to the known-good list; alerting must stop without the
	// detector needing to see the server again.
	rd.UpdateKnownServers([]string{authorizedIP, rogueIP})

	if rogues := detectedIPs(rd.GetRogueServers()); len(rogues) != 0 {
		t.Errorf("still rogue after authorisation: %v", rogues)
	}
	if got := len(rd.GetDetectedServers()); got != 1 {
		t.Errorf("authorising a server dropped it from the detected list (%d)", got)
	}
}

func TestRogueDetectorFallsBackToSourceIPWithoutOption54(t *testing.T) {
	rd := detectorWithKnown(authorizedIP)
	rd.RogueProcessPacket(decode(t, offerNoServerID))

	// No server identifier, so the detector must fall back to the frame's
	// source IP rather than silently dropping the offer — a rogue server can
	// simply omit option 54.
	rogues := detectedIPs(rd.GetRogueServers())
	if len(rogues) != 1 || rogues[0] != noServerIDIP {
		t.Fatalf("rogues = %v, want exactly [%s] from the source IP fallback",
			rogues, noServerIDIP)
	}
}

func TestRogueDetectorIgnoresTruncatedFrame(t *testing.T) {
	rd := detectorWithKnown(authorizedIP)
	rd.RogueProcessPacket(decode(t, offerTruncated))

	if got := len(rd.GetDetectedServers()); got != 0 {
		t.Errorf("a truncated frame produced %d detected server(s), want 0", got)
	}
}

func TestRogueServerIdentifierExtraction(t *testing.T) {
	rd := detectorWithKnown()

	withID := decode(t, offerAuthorized).Layer(layers.LayerTypeDHCPv4)
	dhcpLayer, ok := withID.(*layers.DHCPv4)
	if !ok {
		t.Fatal("fixture did not decode a DHCPv4 layer")
	}
	if got := rd.RogueServerIdentifier(dhcpLayer); got != authorizedIP {
		t.Errorf("server identifier = %q, want %q", got, authorizedIP)
	}

	without := decode(t, offerNoServerID).Layer(layers.LayerTypeDHCPv4)
	noIDLayer, ok := without.(*layers.DHCPv4)
	if !ok {
		t.Fatal("fixture did not decode a DHCPv4 layer")
	}
	if got := rd.RogueServerIdentifier(noIDLayer); got != "" {
		t.Errorf("server identifier = %q for a frame with no option 54, want empty", got)
	}
}
