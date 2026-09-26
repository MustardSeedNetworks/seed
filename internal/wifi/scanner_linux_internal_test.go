//go:build linux

package wifi

import (
	"bytes"
	"testing"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/wifi"
	"golang.org/x/sys/unix"

	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
)

// element assembles one 802.11 information element.
func element(id byte, body ...byte) []byte { return append([]byte{id, byte(len(body))}, body...) }

// heOnly24 is what a Wi-Fi 6 AP on 2.4 GHz channel 6 advertises: HT
// capabilities and a 20 MHz HT Operation element, plus HE capabilities.
func heOnly24(ssid string) []byte {
	return bytes.Join([][]byte{
		element(0, []byte(ssid)...),
		element(3, 6),
		element(7, 'U', 'S', ' '),
		element(11, 0x02, 0x00, 0x2c, 0x00, 0x00),
		element(45, make([]byte, 26)...),
		element(61, append([]byte{6, 0x00}, make([]byte, 20)...)...),
		element(70, 0x73, 0, 0, 0, 0),
		element(255, append([]byte{35}, make([]byte, 20)...)...),
	}, nil)
}

type bssAttrs struct {
	bssid      []byte
	freq       uint32
	mbm        *int32
	pct        *uint8
	capability uint16
	ies        []byte
	beaconIEs  []byte
}

func scanMessage(t *testing.T, b bssAttrs) genetlink.Message {
	t.Helper()
	ae := netlink.NewAttributeEncoder()
	ae.Uint32(unix.NL80211_ATTR_IFINDEX, 4)
	ae.Nested(unix.NL80211_ATTR_BSS, func(nae *netlink.AttributeEncoder) error {
		if b.bssid != nil {
			nae.Bytes(unix.NL80211_BSS_BSSID, b.bssid)
		}
		nae.Uint32(unix.NL80211_BSS_FREQUENCY, b.freq)
		if b.mbm != nil {
			nae.Int32(unix.NL80211_BSS_SIGNAL_MBM, *b.mbm)
		}
		if b.pct != nil {
			nae.Uint8(unix.NL80211_BSS_SIGNAL_UNSPEC, *b.pct)
		}
		nae.Uint16(unix.NL80211_BSS_CAPABILITY, b.capability)
		if b.ies != nil {
			nae.Bytes(unix.NL80211_BSS_INFORMATION_ELEMENTS, b.ies)
		}
		if b.beaconIEs != nil {
			nae.Bytes(unix.NL80211_BSS_BEACON_IES, b.beaconIEs)
		}
		return nil
	})
	data, err := ae.Encode()
	if err != nil {
		t.Fatal(err)
	}
	return genetlink.Message{Data: data}
}

func testBSSID() []byte { return []byte{0x18, 0xa5, 0xff, 0x85, 0x3f, 0x7c} }

func TestParseScanDumpDecodesInformationElements(t *testing.T) {
	msgs := []genetlink.Message{scanMessage(t, bssAttrs{
		bssid: testBSSID(), freq: 2437, mbm: new(int32(-5400)), capability: 0x0431, ies: heOnly24("corp"),
	})}
	networks, err := parseScanDump(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 1 {
		t.Fatalf("got %d networks, want 1", len(networks))
	}
	n := networks[0]
	checks := []struct {
		field     string
		got, want any
	}{
		{"SSID", n.SSID, "corp"},
		{"BSSID", n.BSSID, "18:A5:FF:85:3F:7C"},
		{"Signal", n.Signal, -54},
		{"Channel", n.Channel, 6},
		{"Frequency", n.Frequency, 2437},
		{"Standard", n.Standard, dot11.Standard80211ax.String()},
		{"ChannelWidth", n.ChannelWidth, 20},
		{"HTMode", n.HTMode, "HE20"},
		{"CountryCode", n.CountryCode, "US"},
		{"HasBSSLoad", n.HasBSSLoad, true},
		{"ChannelUtil", n.ChannelUtil, 0x2c},
		{"AdvertisedStations", n.AdvertisedStations, 2},
		{"RRMNeighbor", n.RRMNeighbor, true},
		{"Security", n.Security, "WEP"}, // Privacy set in 0x0431, no RSN element
		{"IsDFS", n.IsDFS, false},
		{"NoiseFloor", n.NoiseFloor, 0},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.field, c.got, c.want)
		}
	}
}

func TestParseScanDumpEntries(t *testing.T) {
	hidden := element(0, 0, 0, 0, 0)
	tests := []struct {
		name  string
		attrs bssAttrs
		check func(t *testing.T, got []*ScannedNetwork)
	}{
		{
			name:  "beacon elements stand in when the probe response carried none",
			attrs: bssAttrs{bssid: testBSSID(), freq: 2437, mbm: new(int32(-6000)), beaconIEs: heOnly24("beacon")},
			check: func(t *testing.T, got []*ScannedNetwork) {
				t.Helper()
				if got[0].SSID != "beacon" {
					t.Errorf("SSID = %q, want beacon", got[0].SSID)
				}
			},
		},
		{
			name:  "unspecified signal maps onto dBm",
			attrs: bssAttrs{bssid: testBSSID(), freq: 2437, pct: new(uint8(50)), ies: hidden},
			check: func(t *testing.T, got []*ScannedNetwork) {
				t.Helper()
				if got[0].Signal != -65 {
					t.Errorf("Signal = %d, want -65", got[0].Signal)
				}
				if !got[0].Hidden || got[0].Security != "Open" {
					t.Errorf("hidden/security = %v/%s, want true/Open", got[0].Hidden, got[0].Security)
				}
			},
		},
		{
			name:  "measured dBm wins over an unspecified figure",
			attrs: bssAttrs{bssid: testBSSID(), freq: 2437, mbm: new(int32(-4800)), pct: new(uint8(10)), ies: hidden},
			check: func(t *testing.T, got []*ScannedNetwork) {
				t.Helper()
				if got[0].Signal != -48 {
					t.Errorf("Signal = %d, want -48", got[0].Signal)
				}
			},
		},
		{
			name:  "DFS channel and legacy 802.11a",
			attrs: bssAttrs{bssid: testBSSID(), freq: 5260, mbm: new(int32(-7000)), ies: element(0, 'x')},
			check: func(t *testing.T, got []*ScannedNetwork) {
				t.Helper()
				if !got[0].IsDFS || got[0].Channel != 52 || got[0].HTMode != "" {
					t.Errorf("DFS/channel/HTMode = %v/%d/%q, want true/52/empty",
						got[0].IsDFS, got[0].Channel, got[0].HTMode)
				}
			},
		},
		{
			name:  "an entry without a BSSID is dropped",
			attrs: bssAttrs{freq: 2437, mbm: new(int32(-6000)), ies: hidden},
			check: func(t *testing.T, got []*ScannedNetwork) {
				t.Helper()
				if len(got) != 0 {
					t.Errorf("got %d networks, want 0", len(got))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScanDump([]genetlink.Message{scanMessage(t, tt.attrs)})
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, got)
		})
	}
}

func TestParseScanDumpSkipsMessagesWithoutBSS(t *testing.T) {
	ae := netlink.NewAttributeEncoder()
	ae.Uint32(unix.NL80211_ATTR_IFINDEX, 4)
	data, err := ae.Encode()
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseScanDump([]genetlink.Message{{Data: data}})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %d networks, err %v; want none, nil", len(got), err)
	}
}

func TestParseScanDumpRejectsMalformedAttributes(t *testing.T) {
	if _, err := parseScanDump([]genetlink.Message{{Data: []byte{0xff, 0x00, 0x01}}}); err == nil {
		t.Fatal("malformed attributes parsed without error")
	}
}

func TestApplySurveyNoise(t *testing.T) {
	networks := []*ScannedNetwork{
		{Frequency: 2437, Signal: -50},
		{Frequency: 2462, Signal: -60},
		{Frequency: 5180, Signal: -70},
	}
	applySurveyNoise(networks, []*wifi.SurveyInfo{
		{Frequency: 2437, Noise: -92},
		{Frequency: 2462, Noise: 0}, // not measured on this channel
	})
	want := [][2]int{{-92, 42}, {0, 0}, {0, 0}}
	for i, n := range networks {
		if n.NoiseFloor != want[i][0] || n.SNR != want[i][1] {
			t.Errorf("network %d noise/SNR = %d/%d, want %d/%d", i, n.NoiseFloor, n.SNR, want[i][0], want[i][1])
		}
	}
}

func TestHTModeFor(t *testing.T) {
	tests := []struct {
		standard dot11.Standard
		width    int
		want     string
	}{
		{dot11.Standard80211n, 40, "HT40"},
		{dot11.Standard80211ac, 80, "VHT80"},
		{dot11.Standard80211ax, 20, "HE20"},
		{dot11.Standard80211be, 320, "EHT320"},
		{dot11.Standard80211g, 20, ""},
		{dot11.StandardUnknown, 0, ""},
	}
	for _, tt := range tests {
		if got := htModeFor(tt.standard, tt.width); got != tt.want {
			t.Errorf("htModeFor(%v, %d) = %q, want %q", tt.standard, tt.width, got, tt.want)
		}
	}
}
