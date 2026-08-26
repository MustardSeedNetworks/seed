//go:build ignore

// generate_lldp.go builds the LLDP fixtures in this directory.
//
// Run with: go run ./internal/discovery/enumerate/testdata/generate_lldp.go
//
// The CDP and EDP fixtures beside these are real captures taken from live gear
// while fixing seed#1922 and seed#1937. LLDP's are generated, because there was
// no capture to hand — a weaker guarantee, so the tests that consume them
// assert the decoded values rather than trusting the bytes because we wrote
// them.
//
// The frames are assembled byte by byte rather than with
// gopacket.SerializeLayers. LinkLayerDiscovery.SerializeTo does not write its
// Values field, so optional TLVs built that way are silently dropped and land
// as trailing garbage — the first attempt here produced a frame that decoded
// with an empty SysName and a DecodeFailure layer. Writing the TLV stream
// directly is both correct and closer to what a switch actually emits.
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

// LLDP TLV types from IEEE 802.1AB.
const (
	tlvEnd             = 0
	tlvChassisID       = 1
	tlvPortID          = 2
	tlvTTL             = 3
	tlvPortDescription = 4
	tlvSysName         = 5
	tlvSysDescription  = 6
	tlvSysCapabilities = 7
	tlvMgmtAddress     = 8
)

var (
	// lldpMulticast is the destination every LLDP advertisement is sent to.
	lldpMulticast = []byte{0x01, 0x80, 0xC2, 0x00, 0x00, 0x0E}
	// switchMAC stands in for the neighbour's port MAC.
	switchMAC = []byte{0x00, 0x1B, 0x54, 0xC1, 0x3E, 0x0F}
)

// tlv encodes one LLDP TLV: seven bits of type, nine of length, then the value.
func tlv(typ int, value []byte) []byte {
	header := uint16(typ)<<9 | uint16(len(value))&0x01FF
	out := make([]byte, 2, 2+len(value))
	binary.BigEndian.PutUint16(out, header)
	return append(out, value...)
}

// mandatoryTLVs are the three every advertisement must carry.
func mandatoryTLVs(portID string, ttl uint16) []byte {
	var out []byte
	// Chassis ID: subtype 4 (MAC address), then the address.
	out = append(out, tlv(tlvChassisID, append([]byte{4}, switchMAC...))...)
	// Port ID: subtype 5 (interface name), then the name.
	out = append(out, tlv(tlvPortID, append([]byte{5}, portID...))...)

	seconds := make([]byte, 2)
	binary.BigEndian.PutUint16(seconds, ttl)
	out = append(out, tlv(tlvTTL, seconds)...)
	return out
}

// optionalTLVs are the ones seed reads beyond the mandatory three.
func optionalTLVs() []byte {
	// System capabilities: supported then enabled, each a uint16 bitmap.
	// Bridge is bit 2 (0x0004), router bit 4 (0x0010).
	caps := make([]byte, 4)
	binary.BigEndian.PutUint16(caps[0:2], 0x0004|0x0010)
	binary.BigEndian.PutUint16(caps[2:4], 0x0004)

	// Management address: the length byte counts the family byte plus the
	// address, so an IPv4 address gives 5.
	mgmt := []byte{
		5,             // address string length
		1,             // IANA address family: IPv4
		10, 44, 20, 5, // the address
		2,           // interface numbering subtype: ifIndex
		0, 0, 0, 24, // interface number
		0, // OID length
	}

	var out []byte
	out = append(out, tlv(tlvPortDescription, []byte("Uplink to core"))...)
	out = append(out, tlv(tlvSysName, []byte("access-sw-01"))...)
	out = append(out, tlv(tlvSysDescription,
		[]byte("Cisco IOS Software, C2960X Software, Version 15.2(7)E3"))...)
	out = append(out, tlv(tlvSysCapabilities, caps)...)
	out = append(out, tlv(tlvMgmtAddress, mgmt)...)
	return out
}

// ethernet builds the header. A non-zero vlan wraps the payload in 802.1Q.
func ethernet(vlan uint16, payload []byte) []byte {
	var frame []byte
	frame = append(frame, lldpMulticast...)
	frame = append(frame, switchMAC...)

	ethType := make([]byte, 2)
	if vlan == 0 {
		binary.BigEndian.PutUint16(ethType, 0x88CC) // LLDP
		frame = append(frame, ethType...)
		return append(frame, payload...)
	}

	binary.BigEndian.PutUint16(ethType, 0x8100) // 802.1Q
	frame = append(frame, ethType...)

	tag := make([]byte, 2)
	binary.BigEndian.PutUint16(tag, vlan&0x0FFF)
	frame = append(frame, tag...)

	inner := make([]byte, 2)
	binary.BigEndian.PutUint16(inner, 0x88CC)
	frame = append(frame, inner...)
	return append(frame, payload...)
}

func main() {
	dir := "internal/discovery/enumerate/testdata"

	full := append(mandatoryTLVs("GigabitEthernet1/0/24", 120), optionalTLVs()...)
	full = append(full, tlv(tlvEnd, nil)...)

	sparse := append(mandatoryTLVs("e1", 30), tlv(tlvEnd, nil)...)

	for _, f := range []struct {
		name  string
		bytes []byte
	}{
		// The ordinary access-port case.
		{"lldp_untagged.bin", ethernet(0, full)},
		// The same advertisement on a trunk. seed#1922 was exactly this case
		// going undiscovered for CDP, so LLDP deserves the fixture too.
		{"lldp_dot1q_vlan300.bin", ethernet(300, full)},
		// Only the mandatory TLVs, as a sparse or embedded device sends. Every
		// optional field must come back empty rather than the decode failing.
		{"lldp_minimal.bin", ethernet(0, sparse)},
	} {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, f.bytes, 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%d bytes)\n", f.name, len(f.bytes))
	}
}
