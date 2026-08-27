package enumerate_test

import (
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// byProtocol indexes a neighbour slice so assertions do not depend on the
// map-iteration order GetNeighbors inherits from each capture.
func byProtocol(t *testing.T, neighbors []*enumerate.Neighbor) map[enumerate.Protocol]*enumerate.Neighbor {
	t.Helper()

	out := make(map[enumerate.Protocol]*enumerate.Neighbor, len(neighbors))
	for _, n := range neighbors {
		if _, dup := out[n.Protocol]; dup {
			t.Fatalf("two neighbours share protocol %q; test fixture expects one each", n.Protocol)
		}
		out[n.Protocol] = n
	}
	return out
}

// TestGetNeighborsMapsProtocolLowercase pins the protocol strings the API hands
// the UI. The UI lowercases defensively, so an uppercase regression here would
// not show on screen -- only a client reading the documented value would break.
func TestGetNeighborsMapsProtocolLowercase(t *testing.T) {
	t.Parallel()

	mgr := enumerate.NewManagerWithNeighbors(
		[]*enumerate.LLDPNeighbor{{ChassisID: "chassis-1", PortID: "Gi0/1"}},
		[]*enumerate.CDPNeighbor{{DeviceID: "switch-1", PortID: "Gi0/2"}},
		[]*enumerate.EDPNeighbor{{DeviceID: "extreme-1", PortID: "1:1"}},
	)

	got := byProtocol(t, mgr.GetNeighbors())

	// Literal values, not the constants: comparing a constant to itself would
	// pass even if someone changed it to "LLDP".
	for _, want := range []string{"lldp", "cdp", "edp"} {
		if _, ok := got[enumerate.Protocol(want)]; !ok {
			t.Errorf("no neighbour mapped to protocol %q; got %v", want, keysOf(got))
		}
	}
}

func keysOf(m map[enumerate.Protocol]*enumerate.Neighbor) []enumerate.Protocol {
	out := make([]enumerate.Protocol, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestGetNeighborsOmitsBlankSystemDescription is a regression test: CDP and EDP
// build SystemDescription by joining Platform and SoftwareVersion with a space.
// When both are empty that produced a single space, which is not empty, so the
// `omitempty` tag emitted `"systemDescription": " "` instead of dropping it.
func TestGetNeighborsOmitsBlankSystemDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		platform string
		version  string
		want     string
	}{
		{"both empty", "", "", ""},
		{"platform only", "WS-C2960", "", "WS-C2960"},
		{"version only", "", "15.0(2)SE", "15.0(2)SE"},
		{"both present", "WS-C2960", "15.0(2)SE", "WS-C2960 15.0(2)SE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := enumerate.NewManagerWithNeighbors(
				nil,
				[]*enumerate.CDPNeighbor{{
					DeviceID: "switch-1", PortID: "Gi0/2",
					Platform: tt.platform, SoftwareVersion: tt.version,
				}},
				[]*enumerate.EDPNeighbor{{
					DeviceID: "extreme-1", PortID: "1:1",
					Platform: tt.platform, SoftwareVersion: tt.version,
				}},
			)

			for proto, n := range byProtocol(t, mgr.GetNeighbors()) {
				if n.SystemDescription != tt.want {
					t.Errorf("%s SystemDescription = %q, want %q",
						proto, n.SystemDescription, tt.want)
				}
			}
		})
	}
}

// TestGetNeighborsSystemNameFallbacks pins the identity field the UI renders as
// the switch name. CDP has no SystemName of its own and EDP's DisplayName is
// optional, so both fall back -- a broken fallback leaves the card blank.
func TestGetNeighborsSystemNameFallbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		displayName string
		wantEDP     string
	}{
		{"display name present", "extreme-display", "extreme-display"},
		{"display name empty falls back to device id", "", "extreme-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := enumerate.NewManagerWithNeighbors(
				nil,
				[]*enumerate.CDPNeighbor{{DeviceID: "switch-1", PortID: "Gi0/2"}},
				[]*enumerate.EDPNeighbor{{
					DeviceID: "extreme-1", PortID: "1:1", DisplayName: tt.displayName,
				}},
			)

			got := byProtocol(t, mgr.GetNeighbors())

			if name := got[enumerate.ProtocolCDP].SystemName; name != "switch-1" {
				t.Errorf("CDP SystemName = %q, want the DeviceID %q", name, "switch-1")
			}
			if name := got[enumerate.ProtocolEDP].SystemName; name != tt.wantEDP {
				t.Errorf("EDP SystemName = %q, want %q", name, tt.wantEDP)
			}
			if id := got[enumerate.ProtocolCDP].ChassisID; id != "switch-1" {
				t.Errorf("CDP ChassisID = %q, want the DeviceID %q", id, "switch-1")
			}
		})
	}
}

func TestParseCDPCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		caps layers.CDPCapabilities
		want []string
	}{
		{"none set", layers.CDPCapabilities{}, nil},
		{"router", layers.CDPCapabilities{L3Router: true}, []string{"Router"}},
		{
			"switch and bridge",
			layers.CDPCapabilities{TBBridge: true, L2Switch: true},
			[]string{"Bridge", "Switch"},
		},
		{"host", layers.CDPCapabilities{IsHost: true}, []string{"Host"}},
		{"phone", layers.CDPCapabilities{IsPhone: true}, []string{"Phone"}},
		{
			"source route bridge",
			layers.CDPCapabilities{SPBridge: true},
			[]string{"Source Route Bridge"},
		},
		{"igmp filter", layers.CDPCapabilities{IGMPFilter: true}, []string{"IGMP Filter"}},
		{"repeater", layers.CDPCapabilities{L1Repeater: true}, []string{"Repeater"}},
		{
			"all set preserves declaration order",
			layers.CDPCapabilities{
				L3Router: true, TBBridge: true, SPBridge: true, L2Switch: true,
				IsHost: true, IGMPFilter: true, L1Repeater: true, IsPhone: true,
			},
			[]string{
				"Router", "Bridge", "Source Route Bridge", "Switch",
				"Host", "IGMP Filter", "Repeater", "Phone",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertStrings(t, enumerate.ExportParseCDPCapabilities(tt.caps), tt.want)
		})
	}
}

func TestParseSystemCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		caps layers.LLDPCapabilities
		want []string
	}{
		{"none set", layers.LLDPCapabilities{}, nil},
		{
			"bridge and router",
			layers.LLDPCapabilities{Bridge: true, Router: true},
			[]string{"Bridge", "Router"},
		},
		{"wlan ap", layers.LLDPCapabilities{WLANAP: true}, []string{"WLAN AP"}},
		{"phone", layers.LLDPCapabilities{Phone: true}, []string{"Phone"}},
		{"docsis", layers.LLDPCapabilities{DocSis: true}, []string{"DOCSIS"}},
		{"station only", layers.LLDPCapabilities{StationOnly: true}, []string{"Station"}},
		{
			"vlan bridges",
			layers.LLDPCapabilities{CVLAN: true, SVLAN: true},
			[]string{"C-VLAN", "S-VLAN"},
		},
		{
			"other and repeater",
			layers.LLDPCapabilities{Other: true, Repeater: true},
			[]string{"Other", "Repeater"},
		},
		{
			"all set preserves declaration order",
			layers.LLDPCapabilities{
				Other: true, Repeater: true, Bridge: true, WLANAP: true, Router: true,
				Phone: true, DocSis: true, StationOnly: true, CVLAN: true, SVLAN: true,
			},
			[]string{
				"Other", "Repeater", "Bridge", "WLAN AP", "Router",
				"Phone", "DOCSIS", "Station", "C-VLAN", "S-VLAN",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertStrings(t, enumerate.ExportParseSystemCapabilities(tt.caps), tt.want)
		})
	}
}

func TestParseEDPTLV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tlvType uint8
		data    []byte
		want    enumerate.EDPNeighbor
	}{
		{
			name:    "display name strips trailing nulls",
			tlvType: enumerate.EDPTLVDisplay,
			data:    []byte("summit-1\x00\x00\x00"),
			want:    enumerate.EDPNeighbor{DisplayName: "summit-1"},
		},
		{
			name:    "display name ignores empty payload",
			tlvType: enumerate.EDPTLVDisplay,
			data:    nil,
			want:    enumerate.EDPNeighbor{},
		},
		{
			name:    "info yields slot:port",
			tlvType: enumerate.EDPTLVInfo,
			data:    []byte{0x00, 0x02, 0x00, 0x07},
			want:    enumerate.EDPNeighbor{PortID: "2:7"},
		},
		{
			name:    "info shorter than slot+port is ignored",
			tlvType: enumerate.EDPTLVInfo,
			data:    []byte{0x00, 0x02},
			want:    enumerate.EDPNeighbor{},
		},
		{
			name:    "info long enough also yields vlan",
			tlvType: enumerate.EDPTLVInfo,
			data:    []byte{0x00, 0x02, 0x00, 0x07, 0x00, 0x00, 0x01, 0x2c},
			want:    enumerate.EDPNeighbor{PortID: "2:7", VLAN: 300},
		},
		{
			name:    "vlan id only",
			tlvType: enumerate.EDPTLVVlan,
			data:    []byte{0x01, 0x2c},
			want:    enumerate.EDPNeighbor{VLAN: 300},
		},
		{
			name:    "vlan id with name",
			tlvType: enumerate.EDPTLVVlan,
			data:    append([]byte{0x01, 0x2c, 0x00, 0x00}, []byte("voice\x00")...),
			want:    enumerate.EDPNeighbor{VLAN: 300, VLANName: "voice"},
		},
		{
			name:    "vlan shorter than id is ignored",
			tlvType: enumerate.EDPTLVVlan,
			data:    []byte{0x01},
			want:    enumerate.EDPNeighbor{},
		},
		{
			name:    "ipv4 management address",
			tlvType: enumerate.EDPTLVIPAddr,
			data:    []byte{192, 168, 1, 10},
			want:    enumerate.EDPNeighbor{ManagementAddress: "192.168.1.10"},
		},
		{
			name:    "short ip address is ignored",
			tlvType: enumerate.EDPTLVIPAddr,
			data:    []byte{192, 168, 1},
			want:    enumerate.EDPNeighbor{},
		},
		{
			name:    "null tlv is a no-op",
			tlvType: enumerate.EDPTLVNull,
			data:    []byte("ignored"),
			want:    enumerate.EDPNeighbor{},
		},
		{
			name:    "unhandled tlv type is a no-op",
			tlvType: enumerate.EDPTLVESRP,
			data:    []byte{0x01, 0x02, 0x03, 0x04},
			want:    enumerate.EDPNeighbor{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got enumerate.EDPNeighbor
			enumerate.ExportParseEDPTLV(tt.tlvType, tt.data, &got)

			if got.DisplayName != tt.want.DisplayName {
				t.Errorf("DisplayName = %q, want %q", got.DisplayName, tt.want.DisplayName)
			}
			if got.PortID != tt.want.PortID {
				t.Errorf("PortID = %q, want %q", got.PortID, tt.want.PortID)
			}
			if got.VLAN != tt.want.VLAN {
				t.Errorf("VLAN = %d, want %d", got.VLAN, tt.want.VLAN)
			}
			if got.VLANName != tt.want.VLANName {
				t.Errorf("VLANName = %q, want %q", got.VLANName, tt.want.VLANName)
			}
			if got.ManagementAddress != tt.want.ManagementAddress {
				t.Errorf("ManagementAddress = %q, want %q",
					got.ManagementAddress, tt.want.ManagementAddress)
			}
		})
	}
}

func TestTrimNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no nulls", "summit", "summit"},
		{"trailing nulls", "summit\x00\x00", "summit"},
		{"all nulls", "\x00\x00\x00", ""},
		{"empty", "", ""},
		{"interior null is kept", "a\x00b\x00", "a\x00b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := enumerate.ExportTrimNull(tt.in); got != tt.want {
				t.Errorf("ExportTrimNull(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %v (%d entries), want %v (%d entries)", got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %q, want %q", i, got[i], want[i])
		}
	}
}
