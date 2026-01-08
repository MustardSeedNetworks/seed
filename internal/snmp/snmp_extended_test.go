// Package snmp_test contains extended tests for the snmp package.
package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

// TestParseIPAddressFromOIDWithValidIPv4 tests parsing valid IPv4 addresses from OID.
func TestParseIPAddressFromOIDWithValidIPv4(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		wantAddr string
		wantType string
	}{
		{
			name:     "valid IPv4 in ipAddressIfIndex OID",
			oid:      "1.3.6.1.2.1.4.34.1.3.1.4.192.168.1.1",
			wantAddr: "192.168.1.1",
			wantType: "ipv4",
		},
		{
			name:     "valid IPv4 loopback",
			oid:      "1.3.6.1.2.1.4.34.1.3.1.4.127.0.0.1",
			wantAddr: "127.0.0.1",
			wantType: "ipv4",
		},
		{
			name:     "valid IPv4 all zeros",
			oid:      "1.3.6.1.2.1.4.34.1.3.1.4.0.0.0.0",
			wantAddr: "0.0.0.0",
			wantType: "ipv4",
		},
		{
			name:     "valid IPv4 broadcast",
			oid:      "1.3.6.1.2.1.4.34.1.3.1.4.255.255.255.255",
			wantAddr: "255.255.255.255",
			wantType: "ipv4",
		},
		{
			name:     "type not ipv4 (type=3)",
			oid:      "1.3.6.1.2.1.4.34.1.3.3.4.192.168.1.1",
			wantAddr: "",
			wantType: "",
		},
		{
			name:     "wrong length (not 4)",
			oid:      "1.3.6.1.2.1.4.34.1.3.1.5.192.168.1.1.1",
			wantAddr: "",
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAddr, gotType := snmp.ExportParseIPAddressFromOID(tt.oid)
			if gotAddr != tt.wantAddr {
				t.Errorf("ParseIPAddressFromOID(%v) addr = %v, want %v", tt.oid, gotAddr, tt.wantAddr)
			}
			if gotType != tt.wantType {
				t.Errorf("ParseIPAddressFromOID(%v) type = %v, want %v", tt.oid, gotType, tt.wantType)
			}
		})
	}
}

// TestParseIPAddressFromOIDWithValidIPv6 tests parsing valid IPv6 addresses from OID.
func TestParseIPAddressFromOIDWithValidIPv6(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		wantAddr string
		wantType string
	}{
		{
			name:     "valid IPv6 loopback",
			oid:      "1.3.6.1.2.1.4.34.1.3.2.16.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1",
			wantAddr: "0000:0000:0000:0000:0000:0000:0000:0001",
			wantType: "ipv6",
		},
		{
			name:     "valid IPv6 all zeros",
			oid:      "1.3.6.1.2.1.4.34.1.3.2.16.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",
			wantAddr: "0000:0000:0000:0000:0000:0000:0000:0000",
			wantType: "ipv6",
		},
		{
			name:     "valid IPv6 link-local",
			oid:      "1.3.6.1.2.1.4.34.1.3.2.16.254.128.0.0.0.0.0.0.0.0.0.0.0.0.0.1",
			wantAddr: "fe80:0000:0000:0000:0000:0000:0000:0001",
			wantType: "ipv6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAddr, gotType := snmp.ExportParseIPAddressFromOID(tt.oid)
			if gotAddr != tt.wantAddr {
				t.Errorf("ParseIPAddressFromOID(%v) addr = %v, want %v", tt.oid, gotAddr, tt.wantAddr)
			}
			if gotType != tt.wantType {
				t.Errorf("ParseIPAddressFromOID(%v) type = %v, want %v", tt.oid, gotType, tt.wantType)
			}
		})
	}
}

// TestFormatSNMPValueExtended tests formatSNMPValue with additional SNMP PDU types.
func TestFormatSNMPValueExtended(t *testing.T) {
	tests := []struct {
		name     string
		variable gosnmp.SnmpPDU
		want     string
	}{
		{
			name: "OctetString with special chars",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: []byte("Cisco IOS Software, C3750E"),
			},
			want: "Cisco IOS Software, C3750E",
		},
		{
			name: "OctetString with unicode",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: []byte("Test-Device\x00"),
			},
			want: "Test-Device\x00",
		},
		{
			name: "OctetString empty",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: []byte(""),
			},
			want: "",
		},
		{
			name: "Counter32 zero",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Counter32,
				Value: uint(0),
			},
			want: "0",
		},
		{
			name: "Counter32 max",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Counter32,
				Value: uint(4294967295),
			},
			want: "4294967295",
		},
		{
			name: "Counter64 large value",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Counter64,
				Value: uint64(18446744073709551615),
			},
			want: "18446744073709551615",
		},
		{
			name: "TimeTicks large value",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.TimeTicks,
				Value: uint32(4294967295),
			},
			want: "4294967295",
		},
		{
			name: "Integer negative",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Integer,
				Value: -1,
			},
			want: "-1",
		},
		{
			name: "Integer zero",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Integer,
				Value: 0,
			},
			want: "0",
		},
		{
			name: "Gauge32 value",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Gauge32,
				Value: uint(1000000),
			},
			want: "1000000",
		},
		{
			name: "NoSuchObject",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.NoSuchObject,
				Value: nil,
			},
			want: "",
		},
		{
			name: "NoSuchInstance",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.NoSuchInstance,
				Value: nil,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportFormatSNMPValue(tt.variable)
			if got != tt.want {
				t.Errorf("FormatSNMPValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSecurityLevelMappings tests various security level configurations.
func TestSecurityLevelMappings(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		cfg  *config.SNMPConfig
	}{
		{
			name: "authNoPriv configuration",
			cfg: &config.SNMPConfig{
				V3Credentials: []config.SNMPv3Credential{
					{
						Name:          "authNoPriv",
						Username:      "user1",
						AuthProtocol:  "SHA256",
						AuthPassword:  "password123",
						PrivProtocol:  "",
						PrivPassword:  "",
						SecurityLevel: "authNoPriv",
					},
				},
				Port:    161,
				Timeout: 50 * time.Millisecond,
				Retries: 1,
			},
		},
		{
			name: "noAuthNoPriv configuration",
			cfg: &config.SNMPConfig{
				V3Credentials: []config.SNMPv3Credential{
					{
						Name:          "noAuthNoPriv",
						Username:      "user1",
						AuthProtocol:  "",
						AuthPassword:  "",
						PrivProtocol:  "",
						PrivPassword:  "",
						SecurityLevel: "noAuthNoPriv",
					},
				},
				Port:    161,
				Timeout: 50 * time.Millisecond,
				Retries: 1,
			},
		},
		{
			name: "authPriv configuration with AES192C",
			cfg: &config.SNMPConfig{
				V3Credentials: []config.SNMPv3Credential{
					{
						Name:          "authPrivAES192C",
						Username:      "user1",
						AuthProtocol:  "SHA384",
						AuthPassword:  "password123",
						PrivProtocol:  "AES192C",
						PrivPassword:  "privpass456",
						SecurityLevel: "authPriv",
					},
				},
				Port:    161,
				Timeout: 50 * time.Millisecond,
				Retries: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Query should fail due to unreachable host, but should exercise config path
			_, err := snmp.Query(ctx, "192.0.2.1", snmp.OIDSysDescr, tt.cfg)
			if err == nil {
				t.Error("Query() should fail for unreachable host")
			}
		})
	}
}

// TestGetSystemInfoNilConfig tests GetSystemInfo with nil config.
func TestGetSystemInfoNilConfig(t *testing.T) {
	ctx := context.Background()

	_, err := snmp.GetSystemInfo(ctx, "192.168.1.1", nil)
	if err == nil {
		t.Error("GetSystemInfo() with nil config should return error")
	}
}

// TestGetVendorVersionNilConfig tests GetVendorVersion with nil config.
func TestGetVendorVersionNilConfig(t *testing.T) {
	ctx := context.Background()

	_, err := snmp.GetVendorVersion(ctx, "192.168.1.1", nil)
	if err == nil {
		t.Error("GetVendorVersion() with nil config should return error")
	}
}

// TestQueryMultipleNilOids tests QueryMultiple with nil OIDs.
func TestQueryMultipleNilOids(t *testing.T) {
	ctx := context.Background()

	cfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Port:        161,
		Timeout:     100 * time.Millisecond,
		Retries:     1,
	}

	_, err := snmp.QueryMultiple(ctx, "192.0.2.1", nil, cfg)
	if err == nil {
		t.Error("QueryMultiple() with nil OIDs should return error")
	}
}

// TestMACTypeMappings verifies MAC type constant values.
func TestMACTypeMappings(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"MACTypeLearned", snmp.MACTypeLearned, "learned"},
		{"MACTypeStatic", snmp.MACTypeStatic, "static"},
		{"MACTypeOther", snmp.MACTypeOther, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.want)
			}
		})
	}
}

// TestStatusMappings verifies status constant values.
func TestStatusMappings(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"StatusUp", snmp.StatusUp, "up"},
		{"StatusDown", snmp.StatusDown, "down"},
		{"StatusTesting", snmp.StatusTesting, "testing"},
		{"StatusUnknown", snmp.StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.want)
			}
		})
	}
}

// TestIPCidrRouteIndexParsing tests parsing of IP CIDR route table OIDs.
func TestIPCidrRouteIndexParsing(t *testing.T) {
	tests := []struct {
		name    string
		oid     string
		wantDst string
		wantMsk string
		wantNH  string
	}{
		{
			name:    "class A route",
			oid:     "1.3.6.1.2.1.4.24.4.1.1.10.0.0.0.255.0.0.0.0.10.0.0.1",
			wantDst: "10.0.0.0",
			wantMsk: "255.0.0.0",
			wantNH:  "10.0.0.1",
		},
		{
			name:    "class B route",
			oid:     "1.3.6.1.2.1.4.24.4.1.1.172.16.0.0.255.255.0.0.0.172.16.0.1",
			wantDst: "172.16.0.0",
			wantMsk: "255.255.0.0",
			wantNH:  "172.16.0.1",
		},
		{
			name:    "class C route",
			oid:     "1.3.6.1.2.1.4.24.4.1.1.192.168.1.0.255.255.255.0.0.192.168.1.254",
			wantDst: "192.168.1.0",
			wantMsk: "255.255.255.0",
			wantNH:  "192.168.1.254",
		},
		{
			name:    "host route (/32)",
			oid:     "1.3.6.1.2.1.4.24.4.1.1.192.168.1.100.255.255.255.255.0.192.168.1.1",
			wantDst: "192.168.1.100",
			wantMsk: "255.255.255.255",
			wantNH:  "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDst, gotMsk, gotNH := snmp.ExportParseIPCidrRouteIndex(tt.oid)
			if gotDst != tt.wantDst {
				t.Errorf("ParseIPCidrRouteIndex() dest = %v, want %v", gotDst, tt.wantDst)
			}
			if gotMsk != tt.wantMsk {
				t.Errorf("ParseIPCidrRouteIndex() mask = %v, want %v", gotMsk, tt.wantMsk)
			}
			if gotNH != tt.wantNH {
				t.Errorf("ParseIPCidrRouteIndex() nexthop = %v, want %v", gotNH, tt.wantNH)
			}
		})
	}
}

// TestEntityIndexEdgeCases tests entity index parsing edge cases.
func TestEntityIndexEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want int
	}{
		{
			name: "single digit index",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.1",
			want: 1,
		},
		{
			name: "double digit index",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.99",
			want: 99,
		},
		{
			name: "index 0",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.0",
			want: 0,
		},
		{
			name: "very long OID",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.123456789",
			want: 123456789,
		},
		{
			name: "just dots",
			oid:  "....",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportExtractEntityIndex(tt.oid)
			if got != tt.want {
				t.Errorf("ExtractEntityIndex(%v) = %v, want %v", tt.oid, got, tt.want)
			}
		})
	}
}

// TestGetIPAddressesNilConfig tests GetIPAddresses with nil config.
func TestGetIPAddressesNilConfig(t *testing.T) {
	ctx := context.Background()

	_, err := snmp.GetIPAddresses(ctx, "192.168.1.1", nil)
	if err == nil {
		t.Error("GetIPAddresses() with nil config should return error")
	}
}

// TestGetRoutesNilConfig tests GetRoutes with nil config.
func TestGetRoutesNilConfig(t *testing.T) {
	ctx := context.Background()

	_, err := snmp.GetRoutes(ctx, "192.168.1.1", nil)
	if err == nil {
		t.Error("GetRoutes() with nil config should return error")
	}
}

// TestGetLLDPNeighborsNilConfig tests GetLLDPNeighbors with nil config.
func TestGetLLDPNeighborsNilConfig(t *testing.T) {
	ctx := context.Background()

	_, err := snmp.GetLLDPNeighbors(ctx, "192.168.1.1", nil)
	if err == nil {
		t.Error("GetLLDPNeighbors() with nil config should return error")
	}
}

// TestGetVLANsNilConfig tests GetVLANs with nil config.
func TestGetVLANsNilConfig(t *testing.T) {
	ctx := context.Background()

	_, err := snmp.GetVLANs(ctx, "192.168.1.1", nil)
	if err == nil {
		t.Error("GetVLANs() with nil config should return error")
	}
}

// TestParseEntityClassAllCases tests all entity class values.
func TestParseEntityClassAllCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other 1", "1", snmp.MACTypeOther},
		{"unknown 2", "2", snmp.StatusUnknown},
		{"chassis 3", "3", "chassis"},
		{"backplane 4", "4", "backplane"},
		{"container 5", "5", "container"},
		{"powerSupply 6", "6", "powerSupply"},
		{"fan 7", "7", "fan"},
		{"sensor 8", "8", "sensor"},
		{"module 9", "9", "module"},
		{"port 10", "10", "port"},
		{"stack 11", "11", "stack"},
		{"cpu 12", "12", "cpu"},
		{"energyObject 13", "13", "energyObject"},
		{"battery 14", "14", "battery"},
		{"storageDrive 15", "15", "storageDrive"},
		{"invalid 16", "16", snmp.StatusUnknown},
		{"invalid 100", "100", snmp.StatusUnknown},
		{"empty string", "", snmp.StatusUnknown},
		{"non-numeric", "abc", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseEntityClass(tt.value)
			if got != tt.want {
				t.Errorf("ParseEntityClass(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestLLDPExtractIndexAllCases tests LLDP index extraction with various OID patterns.
func TestLLDPExtractIndexAllCases(t *testing.T) {
	tests := []struct {
		name          string
		oid           string
		wantLocalPort int
		wantRemoteIdx int
	}{
		{
			name:          "standard LLDP OID format",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.1.1",
			wantLocalPort: 1,
			wantRemoteIdx: 1,
		},
		{
			name:          "high port number",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.48.1",
			wantLocalPort: 48,
			wantRemoteIdx: 1,
		},
		{
			name:          "high remote index",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.1.99",
			wantLocalPort: 1,
			wantRemoteIdx: 99,
		},
		{
			name:          "both high values",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.999.999",
			wantLocalPort: 999,
			wantRemoteIdx: 999,
		},
		{
			name:          "zero values in middle",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.0.0",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
		{
			name:          "empty string",
			oid:           "",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
		{
			name:          "only one part",
			oid:           "1",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
		{
			name:          "two parts",
			oid:           "1.2",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocalPort, gotRemoteIdx := snmp.ExportExtractLLDPIndex(tt.oid)
			if gotLocalPort != tt.wantLocalPort {
				t.Errorf(
					"ExtractLLDPIndex(%v) localPort = %v, want %v",
					tt.oid,
					gotLocalPort,
					tt.wantLocalPort,
				)
			}
			if gotRemoteIdx != tt.wantRemoteIdx {
				t.Errorf(
					"ExtractLLDPIndex(%v) remoteIdx = %v, want %v",
					tt.oid,
					gotRemoteIdx,
					tt.wantRemoteIdx,
				)
			}
		})
	}
}

// TestVLANIndexExtractionAllCases tests VLAN ID extraction from OIDs.
func TestVLANIndexExtractionAllCases(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want int
	}{
		{
			name: "default VLAN",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.1",
			want: 1,
		},
		{
			name: "management VLAN",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.10",
			want: 10,
		},
		{
			name: "user VLAN",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.100",
			want: 100,
		},
		{
			name: "voice VLAN",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.101",
			want: 101,
		},
		{
			name: "max standard VLAN",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.4094",
			want: 4094,
		},
		{
			name: "only base OID",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1",
			want: 1, // Last part is "1"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportExtractVLANIndex(tt.oid)
			if got != tt.want {
				t.Errorf("ExtractVLANIndex(%v) = %v, want %v", tt.oid, got, tt.want)
			}
		})
	}
}

// TestPortBitmapParsingExtended tests port bitmap parsing with more edge cases.
func TestPortBitmapParsingExtended(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []int
	}{
		{
			name:  "all zeros 3 bytes",
			value: []byte{0x00, 0x00, 0x00},
			want:  []int{},
		},
		{
			name:  "all ones single byte",
			value: []byte{0xFF},
			want:  []int{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:  "all ones two bytes",
			value: []byte{0xFF, 0xFF},
			want:  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			name:  "every other port first byte",
			value: []byte{0x55}, // 01010101
			want:  []int{2, 4, 6, 8},
		},
		{
			name:  "only first port in each byte",
			value: []byte{0x80, 0x80}, // 10000000 10000000
			want:  []int{1, 9},
		},
		{
			name:  "string value (invalid)",
			value: "invalid",
			want:  nil,
		},
		{
			name:  "int value (invalid)",
			value: 12345,
			want:  nil,
		},
		{
			name:  "nil value",
			value: nil,
			want:  nil,
		},
		{
			name:  "empty byte slice",
			value: []byte{},
			want:  []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParsePortBitmap(tt.value)
			if len(got) != len(tt.want) {
				t.Errorf("ParsePortBitmap(%v) len = %v, want %v", tt.value, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParsePortBitmap(%v)[%d] = %v, want %v", tt.value, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestRowStatusParsingAllValues tests all row status values.
func TestRowStatusParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"active", "1", "active"},
		{"notInService", "2", "notInService"},
		{"notReady", "3", "notReady"},
		{"createAndGo", "4", "createAndGo"},
		{"createAndWait", "5", "createAndWait"},
		{"destroy", "6", "destroy"},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 7", "7", snmp.StatusUnknown},
		{"undefined 99", "99", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
		{"non-numeric", "abc", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseRowStatus(tt.value)
			if got != tt.want {
				t.Errorf("ParseRowStatus(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestChassisIDFormattingEdgeCases tests chassis ID formatting with more edge cases.
func TestChassisIDFormattingEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "standard MAC",
			value: []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			want:  "00:11:22:33:44:55",
		},
		{
			name:  "broadcast MAC",
			value: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			want:  "ff:ff:ff:ff:ff:ff",
		},
		{
			name:  "standard IPv4",
			value: []byte{192, 168, 1, 1},
			want:  "192.168.1.1",
		},
		{
			name:  "loopback IPv4",
			value: []byte{127, 0, 0, 1},
			want:  "127.0.0.1",
		},
		{
			name:  "printable hostname",
			value: []byte("switch-core-01"),
			want:  "switch-core-01",
		},
		{
			name:  "printable with numbers",
			value: []byte("R1-2960X-48P"),
			want:  "R1-2960X-48P",
		},
		{
			name:  "binary data 3 bytes",
			value: []byte{0x00, 0x01, 0x02},
			want:  "000102",
		},
		{
			name:  "binary data 5 bytes",
			value: []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE},
			want:  "aabbccddee",
		},
		{
			name:  "integer value",
			value: 42,
			want:  "42",
		},
		{
			name:  "string value",
			value: "already-a-string",
			want:  "already-a-string",
		},
		{
			name:  "nil",
			value: nil,
			want:  "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportFormatChassisID(tt.value)
			if got != tt.want {
				t.Errorf("FormatChassisID(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestNetmaskToPrefixAllCommon tests common netmask to prefix conversions.
func TestNetmaskToPrefixAllCommon(t *testing.T) {
	tests := []struct {
		name string
		mask string
		want int
	}{
		{"/32", "255.255.255.255", 32},
		{"/31", "255.255.255.254", 31},
		{"/30", "255.255.255.252", 30},
		{"/29", "255.255.255.248", 29},
		{"/28", "255.255.255.240", 28},
		{"/27", "255.255.255.224", 27},
		{"/26", "255.255.255.192", 26},
		{"/25", "255.255.255.128", 25},
		{"/24", "255.255.255.0", 24},
		{"/23", "255.255.254.0", 23},
		{"/22", "255.255.252.0", 22},
		{"/21", "255.255.248.0", 21},
		{"/20", "255.255.240.0", 20},
		{"/19", "255.255.224.0", 19},
		{"/18", "255.255.192.0", 18},
		{"/17", "255.255.128.0", 17},
		{"/16", "255.255.0.0", 16},
		{"/15", "255.254.0.0", 15},
		{"/14", "255.252.0.0", 14},
		{"/13", "255.248.0.0", 13},
		{"/12", "255.240.0.0", 12},
		{"/11", "255.224.0.0", 11},
		{"/10", "255.192.0.0", 10},
		{"/9", "255.128.0.0", 9},
		{"/8", "255.0.0.0", 8},
		{"/0", "0.0.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportNetmaskToPrefix(tt.mask)
			if got != tt.want {
				t.Errorf("NetmaskToPrefix(%v) = %v, want %v", tt.mask, got, tt.want)
			}
		})
	}
}

// TestRouteTypeParsingAllValues tests all route type values.
func TestRouteTypeParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"reject", "2", "reject"},
		{"local", "3", snmp.IDSubtypeLocal},
		{"remote", "4", "remote"},
		{"blackhole", "5", "blackhole"},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 6", "6", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseRouteType(tt.value)
			if got != tt.want {
				t.Errorf("ParseRouteType(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestRouteProtocolParsingAllValues tests all route protocol values.
func TestRouteProtocolParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"local", "2", snmp.IDSubtypeLocal},
		{"netmgmt", "3", "netmgmt"},
		{"icmp", "4", "icmp"},
		{"egp", "5", "egp"},
		{"ggp", "6", "ggp"},
		{"hello", "7", "hello"},
		{"rip", "8", "rip"},
		{"is-is", "9", "is-is"},
		{"es-is", "10", "es-is"},
		{"ciscoIgrp", "11", "ciscoIgrp"},
		{"bbnSpfIgp", "12", "bbnSpfIgp"},
		{"ospf", "13", "ospf"},
		{"bgp", "14", "bgp"},
		{"idpr", "15", "idpr"},
		{"ciscoEigrp", "16", "ciscoEigrp"},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 17", "17", snmp.StatusUnknown},
		{"undefined 99", "99", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseRouteProtocol(tt.value)
			if got != tt.want {
				t.Errorf("ParseRouteProtocol(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestChassisIDSubtypeParsingAllValues tests all chassis ID subtype values.
func TestChassisIDSubtypeParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"chassisComponent", "1", "chassisComponent"},
		{"interfaceAlias", "2", "interfaceAlias"},
		{"portComponent", "3", "portComponent"},
		{"macAddress", "4", "macAddress"},
		{"networkAddress", "5", "networkAddress"},
		{"interfaceName", "6", "interfaceName"},
		{"local", "7", snmp.IDSubtypeLocal},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 8", "8", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseChassisIDSubtype(tt.value)
			if got != tt.want {
				t.Errorf("ParseChassisIDSubtype(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestPortIDSubtypeParsingAllValues tests all port ID subtype values.
func TestPortIDSubtypeParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"interfaceAlias", "1", "interfaceAlias"},
		{"portComponent", "2", "portComponent"},
		{"macAddress", "3", "macAddress"},
		{"networkAddress", "4", "networkAddress"},
		{"interfaceName", "5", "interfaceName"},
		{"agentCircuitId", "6", "agentCircuitId"},
		{"local", "7", snmp.IDSubtypeLocal},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 8", "8", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParsePortIDSubtype(tt.value)
			if got != tt.want {
				t.Errorf("ParsePortIDSubtype(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestIPAddressTypeParsingAllValues tests all IP address type values.
func TestIPAddressTypeParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"unicast", "1", "unicast"},
		{"anycast", "2", "anycast"},
		{"broadcast", "3", "broadcast"},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 4", "4", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseIPAddressType(tt.value)
			if got != tt.want {
				t.Errorf("ParseIPAddressType(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestIPAddressOriginParsingAllValues tests all IP address origin values.
func TestIPAddressOriginParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"manual", "2", "manual"},
		{"dhcp", "4", "dhcp"},
		{"linklayer", "5", "linklayer"},
		{"random", "6", "random"},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 3", "3", snmp.StatusUnknown},
		{"undefined 7", "7", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseIPAddressOrigin(tt.value)
			if got != tt.want {
				t.Errorf("ParseIPAddressOrigin(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestIPAddressStatusParsingAllValues tests all IP address status values.
func TestIPAddressStatusParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"preferred", "1", "preferred"},
		{"deprecated", "2", "deprecated"},
		{"invalid", "3", "invalid"},
		{"inaccessible", "4", "inaccessible"},
		{"unknown", "5", snmp.StatusUnknown},
		{"tentative", "6", "tentative"},
		{"duplicate", "7", "duplicate"},
		{"optimistic", "8", "optimistic"},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 9", "9", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseIPAddressStatus(tt.value)
			if got != tt.want {
				t.Errorf("ParseIPAddressStatus(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestInterfaceStatusParsingAllValues tests all interface status values.
func TestInterfaceStatusParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"up", "1", snmp.StatusUp},
		{"down", "2", snmp.StatusDown},
		{"testing", "3", snmp.StatusTesting},
		{"dormant 4", "4", snmp.StatusUnknown},
		{"notPresent 5", "5", snmp.StatusUnknown},
		{"lowerLayerDown 6", "6", snmp.StatusUnknown},
		{"undefined 7", "7", snmp.StatusUnknown},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseInterfaceStatus(tt.value)
			if got != tt.want {
				t.Errorf("ParseInterfaceStatus(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestDuplexStatusParsingAllValues tests all duplex status values.
func TestDuplexStatusParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"unknown", "1", snmp.StatusUnknown},
		{"halfDuplex", "2", "half"},
		{"fullDuplex", "3", "full"},
		{"undefined 0", "0", snmp.StatusUnknown},
		{"undefined 4", "4", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseDuplexStatus(tt.value)
			if got != tt.want {
				t.Errorf("ParseDuplexStatus(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestMACStatusParsingAllValues tests all MAC status values.
func TestMACStatusParsingAllValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"invalid", "2", snmp.MACTypeLearned},
		{"learned", "3", snmp.MACTypeLearned},
		{"self", "4", snmp.MACTypeStatic},
		{"mgmt", "5", snmp.MACTypeStatic},
		{"undefined 0", "0", snmp.MACTypeOther},
		{"undefined 6", "6", snmp.MACTypeOther},
		{"undefined 99", "99", snmp.MACTypeOther},
		{"empty", "", snmp.MACTypeOther},
		{"non-numeric", "xyz", snmp.MACTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseMACStatus(tt.value)
			if got != tt.want {
				t.Errorf("ParseMACStatus(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestIsPrintableEdgeCasesExtended tests isPrintable with more edge cases.
func TestIsPrintableEdgeCasesExtended(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty slice", []byte{}, false},
		{"single space", []byte{0x20}, true},
		{"single tilde", []byte{0x7E}, true},
		{"single exclamation", []byte{0x21}, true},
		{"single tab", []byte{0x09}, false},
		{"single newline", []byte{0x0A}, false},
		{"single carriage return", []byte{0x0D}, false},
		{"single null", []byte{0x00}, false},
		{"single DEL", []byte{0x7F}, false},
		{"single unit separator", []byte{0x1F}, false},
		{"ascii letters", []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"), true},
		{"ascii digits", []byte("0123456789"), true},
		{"ascii symbols", []byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"), true},
		{"space and printable", []byte(" Hello World "), true},
		{"printable with trailing null", []byte("Hello\x00"), false},
		{"printable with internal null", []byte("Hel\x00lo"), false},
		{"high ASCII", []byte{0x80}, false},
		{"all high ASCII", []byte{0x80, 0x81, 0xFF}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportIsPrintable(tt.data)
			if got != tt.want {
				t.Errorf("IsPrintable(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

// TestPortListContainsPortExtended tests port list bitmap checking with edge cases.
func TestPortListContainsPortExtended(t *testing.T) {
	tests := []struct {
		name     string
		portList any
		portNum  int
		want     bool
	}{
		{"port 1 set", []byte{0x80}, 1, true},
		{"port 1 not set", []byte{0x7F}, 1, false},
		{"port 8 set", []byte{0x01}, 8, true},
		{"port 8 not set", []byte{0xFE}, 8, false},
		{"port 9 set", []byte{0x00, 0x80}, 9, true},
		{"port 16 set", []byte{0x00, 0x01}, 16, true},
		{"port 24 set", []byte{0x00, 0x00, 0x01}, 24, true},
		{"port beyond bitmap", []byte{0xFF}, 9, false},
		{"port 0 invalid", []byte{0xFF}, 0, false},
		{"negative port", []byte{0xFF}, -5, false},
		{"nil portList", nil, 1, false},
		{"string portList", "not bytes", 1, false},
		{"int portList", 12345, 1, false},
		{"empty portList", []byte{}, 1, false},
		{"large port number", []byte{0xFF, 0xFF, 0xFF, 0xFF}, 32, true},
		{"large port number not set", []byte{0xFF, 0xFF, 0xFF, 0xFE}, 32, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportPortListContainsPort(tt.portList, tt.portNum)
			if got != tt.want {
				t.Errorf("PortListContainsPort(%v, %v) = %v, want %v", tt.portList, tt.portNum, got, tt.want)
			}
		})
	}
}

// TestParseTimeTicksExtended tests time ticks parsing with various values.
func TestParseTimeTicksExtended(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantIsZero bool
	}{
		{"zero", "0", false},
		{"positive small", "100", false},
		{"positive large", "86400000", false},   // 10 days in centiseconds
		{"very large", "3155760000000", false},  // ~100 years in centiseconds
		{"negative", "-100", false},             // Should parse but result in negative duration
		{"empty string", "", true},              // Should fail to parse
		{"non-numeric", "abc", true},            // Should fail to parse
		{"decimal", "100.5", true},              // Should fail to parse
		{"scientific notation", "1e5", true},    // Should fail to parse
		{"hex", "0xFF", true},                   // Should fail to parse
		{"whitespace only", "   ", true},        // Should fail to parse
		{"leading whitespace", " 100", true},    // Should fail to parse
		{"trailing whitespace", "100 ", true},   // Should fail to parse
		{"overflow prevention", "9999999999999999999999", true}, // Too large
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseTimeTicks(tt.value)
			gotTime, ok := got.(time.Time)
			if !ok {
				t.Fatalf("ParseTimeTicks(%v) returned non-time.Time type: %T", tt.value, got)
			}
			if tt.wantIsZero && !gotTime.IsZero() {
				t.Errorf("ParseTimeTicks(%v) = non-zero time, want zero time", tt.value)
			}
			if !tt.wantIsZero && gotTime.IsZero() {
				t.Errorf("ParseTimeTicks(%v) = zero time, want non-zero time", tt.value)
			}
		})
	}
}

// TestMACFromOIDExtended tests MAC address parsing from OID parts.
func TestMACFromOIDExtended(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "all zeros",
			parts: []string{"0", "0", "0", "0", "0", "0"},
			want:  "00:00:00:00:00:00",
		},
		{
			name:  "all 255",
			parts: []string{"255", "255", "255", "255", "255", "255"},
			want:  "ff:ff:ff:ff:ff:ff",
		},
		{
			name:  "mixed values",
			parts: []string{"170", "187", "204", "221", "238", "255"},
			want:  "aa:bb:cc:dd:ee:ff",
		},
		{
			name:  "single digit values",
			parts: []string{"1", "2", "3", "4", "5", "6"},
			want:  "01:02:03:04:05:06",
		},
		{
			name:  "common MAC",
			parts: []string{"0", "17", "34", "51", "68", "85"},
			want:  "00:11:22:33:44:55",
		},
		{
			name:  "too few parts",
			parts: []string{"0", "0", "0", "0", "0"},
			want:  "",
		},
		{
			name:  "empty parts",
			parts: []string{},
			want:  "",
		},
		{
			name:  "non-numeric value",
			parts: []string{"0", "0", "0", "0", "0", "xx"},
			want:  "",
		},
		{
			name:  "value too large",
			parts: []string{"0", "0", "0", "0", "0", "256"},
			want:  "",
		},
		{
			name:  "negative value",
			parts: []string{"0", "0", "0", "0", "0", "-1"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseMACFromOID(tt.parts)
			if got != tt.want {
				t.Errorf("ParseMACFromOID(%v) = %v, want %v", tt.parts, got, tt.want)
			}
		})
	}
}
