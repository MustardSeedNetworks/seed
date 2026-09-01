package enumerate

import (
	"testing"

	"github.com/gopacket/gopacket/layers"
)

// The chassis and port identifiers below are the two rows captured on the lab
// trunk in #1932, where whichever field arrived as binary rendered as
// unprintable bytes while the other read fine. Both must be readable now.
func TestFormatLLDPChassisID(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		subtype layers.LLDPChassisIDSubType
		id      []byte
		want    string
	}{
		{
			name:    "mac address is colon separated hex",
			subtype: layers.LLDPChassisIDSubTypeMACAddr,
			id:      []byte{0x00, 0x1B, 0x54, 0xC1, 0x3E, 0x0F},
			want:    "00:1b:54:c1:3e:0f",
		},
		{
			name:    "the simulated router's binary chassis id",
			subtype: layers.LLDPChassisIDSubTypeMACAddr,
			id:      []byte{0x00, 0x00, 0x0C, 0x00, 0x01, 0x01},
			want:    "00:00:0c:00:01:01",
		},
		{
			name:    "network address carries an ipv4 family byte",
			subtype: layers.LLDPChassisIDSubTypeNetworkAddr,
			id:      []byte{1, 10, 44, 20, 5},
			want:    "10.44.20.5",
		},
		{
			name:    "network address carries an ipv6 family byte",
			subtype: layers.LLDPChassisIDSubTypeNetworkAddr,
			id: []byte{
				2, 0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0x01,
			},
			want: "2001:db8::1",
		},
		{
			name:    "a text subtype is unchanged",
			subtype: layers.LLDPChassisIDSubTypeLocal,
			id:      []byte("access-sw-01"),
			want:    "access-sw-01",
		},
		{
			name:    "a mac of the wrong width falls back to hex rather than vanishing",
			subtype: layers.LLDPChassisIDSubTypeMACAddr,
			id:      []byte{0xde, 0xad, 0xbe},
			want:    "deadbe",
		},
		{
			name:    "a text subtype carrying bytes that are not text falls back to hex",
			subtype: layers.LLDPChassisIDSubTypeChassisComp,
			id:      []byte{0xff, 0xfe, 0xfd},
			want:    "fffefd",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := formatLLDPChassisID(layers.LLDPChassisID{Subtype: tc.subtype, ID: tc.id})
			if got != tc.want {
				t.Errorf("formatLLDPChassisID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatLLDPPortID(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		subtype layers.LLDPPortIDSubType
		id      []byte
		want    string
	}{
		{
			// The UniFi gateway's port id, which used to render as "t\xac\xb9;\xaf>".
			name:    "mac address is colon separated hex",
			subtype: layers.LLDPPortIDSubtypeMACAddr,
			id:      []byte{0x74, 0xAC, 0xB9, 0x3B, 0xAF, 0x3E},
			want:    "74:ac:b9:3b:af:3e",
		},
		{
			name:    "an interface name is unchanged",
			subtype: layers.LLDPPortIDSubtypeIfaceName,
			id:      []byte("TenGigabitEthernet0/0/0"),
			want:    "TenGigabitEthernet0/0/0",
		},
		{
			name:    "network address carries an ipv4 family byte",
			subtype: layers.LLDPPortIDSubtypeNetworkAddr,
			id:      []byte{1, 192, 0, 2, 1},
			want:    "192.0.2.1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := formatLLDPPortID(layers.LLDPPortID{Subtype: tc.subtype, ID: tc.id})
			if got != tc.want {
				t.Errorf("formatLLDPPortID() = %q, want %q", got, tc.want)
			}
		})
	}
}
