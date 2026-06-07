package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// TestParseIPAddressFromOIDEdgeCases tests parseIPAddressFromOID with additional edge cases.
func TestParseIPAddressFromOIDEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		wantAddr string
		wantType string
	}{
		{
			name:     "very short OID",
			oid:      "1.3",
			wantAddr: "",
			wantType: "",
		},
		{
			name:     "OID with only prefix",
			oid:      "1.3.6.1.2.1.4.34.1",
			wantAddr: "",
			wantType: "",
		},
		{
			name:     "OID with spaces",
			oid:      "1.3.6. 1.2.1.4",
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

// TestParseInetCidrRouteIndexEdgeCases tests parseInetCidrRouteIndex with additional edge cases.
func TestParseInetCidrRouteIndexEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		oid        string
		wantDst    string
		wantPrefix int
		wantNH     string
	}{
		{
			name:       "OID with no parts",
			oid:        "",
			wantDst:    "",
			wantPrefix: 0,
			wantNH:     "",
		},
		{
			name:       "OID with single part",
			oid:        "1",
			wantDst:    "",
			wantPrefix: 0,
			wantNH:     "",
		},
		{
			name:       "OID with minimum parts but invalid",
			oid:        "1.2.3.4.5.6.7.8.9.10",
			wantDst:    "",
			wantPrefix: 0,
			wantNH:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDst, gotPrefix, gotNH := snmp.ExportParseInetCidrRouteIndex(tt.oid)
			if gotDst != tt.wantDst {
				t.Errorf("ParseInetCidrRouteIndex(%v) dest = %v, want %v", tt.oid, gotDst, tt.wantDst)
			}
			if gotPrefix != tt.wantPrefix {
				t.Errorf(
					"ParseInetCidrRouteIndex(%v) prefix = %v, want %v",
					tt.oid,
					gotPrefix,
					tt.wantPrefix,
				)
			}
			if gotNH != tt.wantNH {
				t.Errorf(
					"ParseInetCidrRouteIndex(%v) nexthop = %v, want %v",
					tt.oid,
					gotNH,
					tt.wantNH,
				)
			}
		})
	}
}

// TestNetmaskToPrefixEdgeCases tests netmaskToPrefix with additional edge cases.
func TestNetmaskToPrefixEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		mask string
		want int
	}{
		{"valid /26", "255.255.255.192", 26},
		{"valid /27", "255.255.255.224", 27},
		{"valid /29", "255.255.255.248", 29},
		{"valid /31", "255.255.255.254", 31},
		{"valid /23", "255.255.254.0", 23},
		{"valid /22", "255.255.252.0", 22},
		{"valid /12", "255.240.0.0", 12},
		{"negative octet", "-1.0.0.0", 0},
		{"spaces in mask", " 255.255.255.0 ", 0},
		{"dots only", "...", 0},
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

// TestFormatIPv6FromOctetsEdgeCases tests formatIPv6FromOctets with additional edge cases.
func TestFormatIPv6FromOctetsEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		octets []string
		want   string
	}{
		{
			name:   "exactly 15 octets (too short)",
			octets: []string{"0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0"},
			want:   "",
		},
		{
			name: "non-numeric octet",
			octets: []string{
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"0",
				"abc",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportFormatIPv6FromOctets(tt.octets)
			if got != tt.want {
				t.Errorf("FormatIPv6FromOctets(%v) = %v, want %v", tt.octets, got, tt.want)
			}
		})
	}
}

// TestExtractLLDPIndexEdgeCases tests extractLLDPIndex with additional edge cases.
func TestExtractLLDPIndexEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		oid           string
		wantLocalPort int
		wantRemoteIdx int
	}{
		{
			name:          "only dots",
			oid:           "...",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
		{
			name:          "leading dot",
			oid:           ".1.0.8802.1.1.2.1.4.1.1.5.0.24.1",
			wantLocalPort: 24,
			wantRemoteIdx: 1,
		},
		{
			name:          "single element OID",
			oid:           "1",
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

// TestIsPrintableEdgeCases tests isPrintable with additional edge cases.
func TestIsPrintableEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"exactly at printable boundary (space)", []byte{0x20}, true},
		{"exactly at printable boundary (tilde)", []byte{0x7E}, true},
		{"just below printable (unit separator)", []byte{0x1F}, false},
		{"just above printable (DEL)", []byte{0x7F}, false},
		{"mixed printable with one non-printable", []byte{'a', 0x01, 'b'}, false},
		{"all printable symbols", []byte{'!', '@', '#', '$', '%'}, true},
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

// TestFormatChassisIDEdgeCases tests formatChassisID with additional edge cases.
func TestFormatChassisIDEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "single byte",
			value: []byte{0xAB},
			want:  "ab",
		},
		{
			name:  "two bytes",
			value: []byte{0xAB, 0xCD},
			want:  "abcd",
		},
		{
			name:  "all zeros MAC",
			value: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want:  "00:00:00:00:00:00",
		},
		{
			name:  "float value",
			value: 3.14,
			want:  "3.14",
		},
		{
			name:  "bool value",
			value: true,
			want:  "true",
		},
		{
			name:  "slice of ints",
			value: []int{1, 2, 3},
			want:  "[1 2 3]",
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

// TestExtractVLANIndexEdgeCases tests extractVLANIndex with additional edge cases.
func TestExtractVLANIndexEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want int
	}{
		{
			name: "VLAN 0",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.0",
			want: 0,
		},
		{
			name: "max VLAN 4095",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.4095",
			want: 4095,
		},
		{
			name: "OID with trailing dot",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.",
			want: 0,
		},
		{
			name: "non-numeric VLAN",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.abc",
			want: 0,
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

// TestParsePortBitmapEdgeCases tests parsePortBitmap with additional edge cases.
func TestParsePortBitmapEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []int
	}{
		{
			name:  "single bit in middle byte",
			value: []byte{0x00, 0x80, 0x00}, // Port 9
			want:  []int{9},
		},
		{
			name:  "alternating bits",
			value: []byte{0xAA}, // 10101010 = ports 1, 3, 5, 7
			want:  []int{1, 3, 5, 7},
		},
		{
			name:  "map type (not supported)",
			value: map[string]int{"port": 1},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParsePortBitmap(tt.value)
			if len(got) != len(tt.want) {
				t.Errorf("ParsePortBitmap(%v) len = %v, want %v", tt.value, len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ParsePortBitmap(%v)[%d] = %v, want %v", tt.value, i, v, tt.want[i])
				}
			}
		})
	}
}

// TestParseRowStatusEdgeCases tests parseRowStatus with additional edge cases.
func TestParseRowStatusEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"zero", "0", snmp.StatusUnknown},
		{"seven", "7", snmp.StatusUnknown},
		{"large number", "1000", snmp.StatusUnknown},
		{"float string", "1.5", snmp.StatusUnknown},
		{"whitespace", " ", snmp.StatusUnknown},
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

// TestParseMACFromOIDEdgeCases tests parseMACFromOID with additional edge cases.
func TestParseMACFromOIDEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "negative value",
			parts: []string{"-1", "0", "0", "0", "0", "0"},
			want:  "",
		},
		{
			name:  "value at max boundary 255",
			parts: []string{"255", "255", "255", "255", "255", "255"},
			want:  "ff:ff:ff:ff:ff:ff",
		},
		{
			name:  "mixed valid and invalid",
			parts: []string{"0", "0", "0", "0", "0", "abc"},
			want:  "",
		},
		{
			name:  "hex values (not supported - expects decimal)",
			parts: []string{"aa", "bb", "cc", "dd", "ee", "ff"},
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

// TestParseTimeTicksEdgeCases tests parseTimeTicks with additional edge cases.
func TestParseTimeTicksEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantIsZero bool
	}{
		{"very large value", "999999999999", false},
		{"hex string", "0xFF", true},
		{"scientific notation", "1e10", true},
		{"just whitespace", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseTimeTicks(tt.value)
			gotTime, ok := got.(time.Time)
			if !ok {
				t.Fatalf("ParseTimeTicks(%v) returned non-time.Time type: %T", tt.value, got)
			}
			if tt.wantIsZero {
				if !gotTime.IsZero() {
					t.Errorf("ParseTimeTicks(%v) = %v, want zero time", tt.value, gotTime)
				}
			} else {
				if gotTime.IsZero() {
					t.Errorf("ParseTimeTicks(%v) = zero time, want non-zero", tt.value)
				}
			}
		})
	}
}

// TestCollectMACEntriesDefaultType tests that collectMACEntries sets default type.
func TestCollectMACEntriesDefaultType(t *testing.T) {
	// Entry without Type should default to "learned"
	macToEntry := map[string]*snmp.MACEntry{
		"00:11:22:33:44:55": {MAC: "00:11:22:33:44:55", VLAN: 1, IfIndex: 1, Type: ""},
		"aa:bb:cc:dd:ee:ff": {
			MAC:     "aa:bb:cc:dd:ee:ff",
			VLAN:    2,
			IfIndex: 2,
			Type:    snmp.MACTypeStatic,
		}, // Already has type
	}

	got := snmp.ExportCollectMACEntries(macToEntry)

	for _, entry := range got {
		if entry.Type == "" {
			t.Errorf("Entry %v has empty type, expected default 'learned'", entry.MAC)
		}
	}
}

// TestAllV3CredentialsFailThenV2cFails tests fallback behavior.
func TestAllV3CredentialsFailThenV2cFails(t *testing.T) {
	ctx := context.Background()

	cfg := &config.SNMPConfig{
		V3Credentials: []config.SNMPv3Credential{
			{
				Name:         "cred1",
				Username:     "user1",
				AuthProtocol: "SHA256",
				AuthPassword: "pass1",
				PrivProtocol: "AES256",
				PrivPassword: "priv1",
			},
			{
				Name:         "cred2",
				Username:     "user2",
				AuthProtocol: "SHA512",
				AuthPassword: "pass2",
				PrivProtocol: "AES192C",
				PrivPassword: "priv2",
			},
		},
		Communities: []string{"public", "private", "secret"},
		Port:        161,
		Timeout:     50 * time.Millisecond,
		Retries:     1,
	}

	// All should fail because of unreachable host
	_, err := snmp.GetAllInterfaces(ctx, "192.0.2.1", cfg)
	if err == nil {
		t.Error("Expected error when all credentials fail")
	}
}

// TestContextDeadlineExceeded tests that context deadline is respected.
func TestContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	cfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Port:        161,
		Timeout:     time.Second,
		Retries:     1,
	}

	// Context already expired
	_, err := snmp.Query(ctx, "192.168.1.1", snmp.OIDSysDescr, cfg)
	if err == nil {
		t.Error("Expected error with expired context deadline")
	}
}

// TestEmptyV3CredentialsList tests behavior with empty v3 credentials.
func TestEmptyV3CredentialsList(t *testing.T) {
	ctx := context.Background()

	cfg := &config.SNMPConfig{
		V3Credentials: []config.SNMPv3Credential{}, // Empty v3
		Communities:   []string{"public"},
		Port:          161,
		Timeout:       50 * time.Millisecond,
		Retries:       1,
	}

	// Should skip v3 and try v2c directly
	_, err := snmp.Query(ctx, "192.0.2.1", snmp.OIDSysDescr, cfg)
	if err == nil {
		t.Error("Expected error for unreachable host")
	}
}

// TestNilV3Credentials tests behavior with nil v3 credentials slice.
func TestNilV3Credentials(t *testing.T) {
	ctx := context.Background()

	cfg := &config.SNMPConfig{
		V3Credentials: nil, // Nil v3
		Communities:   []string{"public"},
		Port:          161,
		Timeout:       50 * time.Millisecond,
		Retries:       1,
	}

	// Should skip v3 and try v2c directly
	_, err := snmp.Query(ctx, "192.0.2.1", snmp.OIDSysDescr, cfg)
	if err == nil {
		t.Error("Expected error for unreachable host")
	}
}
