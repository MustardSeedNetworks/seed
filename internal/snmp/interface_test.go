package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

func TestParseInterfaceStatus(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"up", "1", snmp.StatusUp},
		{"down", "2", snmp.StatusDown},
		{"testing", "3", snmp.StatusTesting},
		{"unknown_4", "4", snmp.StatusUnknown},
		{"dormant_5", "5", snmp.StatusUnknown},
		{"notPresent_6", "6", snmp.StatusUnknown},
		{"lowerLayerDown_7", "7", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high", "99", snmp.StatusUnknown},
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

func TestParseDuplexStatus(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"unknown", "1", snmp.StatusUnknown},
		{"halfDuplex", "2", "half"},
		{"fullDuplex", "3", "full"},
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high", "99", snmp.StatusUnknown},
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

func TestParseMACStatus(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"invalid_status", "2", snmp.MACTypeLearned}, // invalid maps to learned
		{"learned", "3", snmp.MACTypeLearned},
		{"self", "4", snmp.MACTypeStatic}, // self maps to static
		{"mgmt", "5", snmp.MACTypeStatic}, // mgmt maps to static
		{"empty", "", snmp.MACTypeOther},
		{"invalid_string", "invalid", snmp.MACTypeOther},
		{"negative", "-1", snmp.MACTypeOther},
		{"high", "99", snmp.MACTypeOther},
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

func TestParseMACFromOID(t *testing.T) {
	// ParseMACFromOID takes 6 MAC octets, not the full OID
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "valid MAC address",
			parts: []string{"170", "187", "204", "221", "238", "255"},
			want:  "aa:bb:cc:dd:ee:ff",
		},
		{
			name:  "all zeros",
			parts: []string{"0", "0", "0", "0", "0", "0"},
			want:  "00:00:00:00:00:00",
		},
		{
			name:  "short parts",
			parts: []string{"1", "2", "3"},
			want:  "",
		},
		{
			name:  "invalid MAC parts non-numeric",
			parts: []string{"xx", "bb", "cc", "dd", "ee", "ff"},
			want:  "",
		},
		{
			name:  "empty",
			parts: []string{},
			want:  "",
		},
		{
			name:  "value out of range",
			parts: []string{"256", "0", "0", "0", "0", "0"},
			want:  "",
		},
		{
			name:  "broadcast MAC",
			parts: []string{"255", "255", "255", "255", "255", "255"},
			want:  "ff:ff:ff:ff:ff:ff",
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

func TestParseTimeTicks(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantIsZero bool
	}{
		{"valid", "12345", false},
		{"zero_ticks", "0", false}, // Zero ticks is valid - returns current time
		{"large", "999999999", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
		{"negative", "-1", false}, // Negative is still parseable as int64
		{"float", "123.45", true},
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

func TestPortListContainsPort(t *testing.T) {
	tests := []struct {
		name     string
		portList any
		portNum  int
		want     bool
	}{
		{
			name:     "empty byte slice",
			portList: []byte{},
			portNum:  1,
			want:     false,
		},
		{
			name:     "port 1 in first byte bit 7",
			portList: []byte{0x80}, // 10000000
			portNum:  1,
			want:     true,
		},
		{
			name:     "port 2 in first byte bit 6",
			portList: []byte{0x40}, // 01000000
			portNum:  2,
			want:     true,
		},
		{
			name:     "port 8 in first byte bit 0",
			portList: []byte{0x01}, // 00000001
			portNum:  8,
			want:     true,
		},
		{
			name:     "port 9 in second byte bit 7",
			portList: []byte{0x00, 0x80}, // 00000000 10000000
			portNum:  9,
			want:     true,
		},
		{
			name:     "port 16 in second byte bit 0",
			portList: []byte{0x00, 0x01}, // 00000000 00000001
			portNum:  16,
			want:     true,
		},
		{
			name:     "port not present",
			portList: []byte{0x80, 0x00}, // Port 1 only
			portNum:  2,
			want:     false,
		},
		{
			name:     "port beyond bitmap",
			portList: []byte{0xFF}, // Only 8 ports
			portNum:  20,
			want:     false,
		},
		{
			name:     "non-byte slice type",
			portList: "not a byte slice",
			portNum:  1,
			want:     false,
		},
		{
			name:     "nil portList",
			portList: nil,
			portNum:  1,
			want:     false,
		},
		{
			name:     "port 0 invalid",
			portList: []byte{0xFF},
			portNum:  0,
			want:     false,
		},
		{
			name:     "negative port",
			portList: []byte{0xFF},
			portNum:  -1,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportPortListContainsPort(tt.portList, tt.portNum)
			if got != tt.want {
				t.Errorf(
					"PortListContainsPort(%v, %v) = %v, want %v",
					tt.portList,
					tt.portNum,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestInterfaceInfoStruct(t *testing.T) {
	now := time.Now()
	iface := snmp.InterfaceInfo{
		Index:       1,
		Description: "GigabitEthernet0/0",
		Name:        "Gi0/0",
		Speed:       1000000000,
		Duplex:      "full",
		AdminStatus: "up",
		OperStatus:  "up",
		LastChange:  now,
		MACAddress:  "00:11:22:33:44:55",
	}

	// Verify all fields to avoid linter warnings
	if iface.Index != 1 {
		t.Errorf("Index = %v, want 1", iface.Index)
	}
	if iface.Description != "GigabitEthernet0/0" {
		t.Errorf("Description = %v, want 'GigabitEthernet0/0'", iface.Description)
	}
	if iface.Name != "Gi0/0" {
		t.Errorf("Name = %v, want 'Gi0/0'", iface.Name)
	}
	if iface.Speed != 1000000000 {
		t.Errorf("Speed = %v, want 1000000000", iface.Speed)
	}
	if iface.Duplex != "full" {
		t.Errorf("Duplex = %v, want 'full'", iface.Duplex)
	}
	if iface.AdminStatus != "up" {
		t.Errorf("AdminStatus = %v, want 'up'", iface.AdminStatus)
	}
	if iface.OperStatus != "up" {
		t.Errorf("OperStatus = %v, want 'up'", iface.OperStatus)
	}
	if iface.LastChange != now {
		t.Errorf("LastChange = %v, want %v", iface.LastChange, now)
	}
	if iface.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("MACAddress = %v, want '00:11:22:33:44:55'", iface.MACAddress)
	}
}

func TestMACEntryStruct(t *testing.T) {
	entry := snmp.MACEntry{
		MAC:     "00:11:22:33:44:55",
		VLAN:    100,
		IfIndex: 24,
		Type:    snmp.MACTypeLearned,
	}

	if entry.MAC != "00:11:22:33:44:55" {
		t.Errorf("MAC = %v, want '00:11:22:33:44:55'", entry.MAC)
	}
	if entry.VLAN != 100 {
		t.Errorf("VLAN = %v, want 100", entry.VLAN)
	}
	if entry.IfIndex != 24 {
		t.Errorf("IfIndex = %v, want 24", entry.IfIndex)
	}
	if entry.Type != snmp.MACTypeLearned {
		t.Errorf("Type = %v, want %v", entry.Type, snmp.MACTypeLearned)
	}
}

func TestGetAllInterfaces(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		cfg     *config.SNMPConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			ip:      "192.168.1.1",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "unreachable host",
			ip:   "192.0.2.1",
			cfg: &config.SNMPConfig{
				Communities: []string{"public"},
				Port:        161,
				Timeout:     100 * time.Millisecond,
				Retries:     1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := snmp.GetAllInterfaces(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetAllInterfaces() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetAllInterfaces() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetMACTable(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		cfg     *config.SNMPConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			ip:      "192.168.1.1",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "unreachable host",
			ip:   "192.0.2.1",
			cfg: &config.SNMPConfig{
				Communities: []string{"public"},
				Port:        161,
				Timeout:     100 * time.Millisecond,
				Retries:     1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := snmp.GetMACTable(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetMACTable() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetMACTable() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestParseVLANAndMAC(t *testing.T) {
	tests := []struct {
		name    string
		parts   []string
		wantVLN int
		wantMAC string
		wantOK  bool
	}{
		{
			name: "valid VLAN and MAC",
			parts: []string{
				"1",
				"3",
				"6",
				"1",
				"2",
				"1",
				"17",
				"7",
				"1",
				"2",
				"2",
				"1",
				"2",
				"100",
				"170",
				"187",
				"204",
				"221",
				"238",
				"255",
			},
			wantVLN: 100,
			wantMAC: "aa:bb:cc:dd:ee:ff",
			wantOK:  true,
		},
		{
			name:    "short OID",
			parts:   []string{"1", "2", "3"},
			wantVLN: 0,
			wantMAC: "",
			wantOK:  false,
		},
		{
			name: "invalid VLAN",
			parts: []string{
				"1",
				"3",
				"6",
				"1",
				"2",
				"1",
				"17",
				"7",
				"1",
				"2",
				"2",
				"1",
				"2",
				"xx",
				"170",
				"187",
				"204",
				"221",
				"238",
				"255",
			},
			wantVLN: 0,
			wantMAC: "",
			wantOK:  false,
		},
		{
			name:    "empty",
			parts:   []string{},
			wantVLN: 0,
			wantMAC: "",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVLAN, gotMAC, gotOK := snmp.ExportParseVLANAndMAC(tt.parts)
			if gotVLAN != tt.wantVLN {
				t.Errorf("ParseVLANAndMAC() VLAN = %v, want %v", gotVLAN, tt.wantVLN)
			}
			if gotMAC != tt.wantMAC {
				t.Errorf("ParseVLANAndMAC() MAC = %v, want %v", gotMAC, tt.wantMAC)
			}
			if gotOK != tt.wantOK {
				t.Errorf("ParseVLANAndMAC() OK = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestParseBridgePort(t *testing.T) {
	tests := []struct {
		name    string
		pdu     gosnmp.SnmpPDU
		wantPrt int
		wantOK  bool
	}{
		{
			name: "integer value",
			pdu: gosnmp.SnmpPDU{
				Type:  gosnmp.Integer,
				Value: 24,
			},
			wantPrt: 24,
			wantOK:  true,
		},
		{
			name: "zero value",
			pdu: gosnmp.SnmpPDU{
				Type:  gosnmp.Integer,
				Value: 0,
			},
			wantPrt: 0,
			wantOK:  true, // Zero is valid - parseable integer
		},
		{
			name: "negative value",
			pdu: gosnmp.SnmpPDU{
				Type:  gosnmp.Integer,
				Value: -1,
			},
			wantPrt: -1,
			wantOK:  true, // Negative is valid - parseable integer
		},
		{
			name: "octet string numeric",
			pdu: gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: []byte("24"),
			},
			wantPrt: 24,
			wantOK:  true, // formatSNMPValue converts to string "24", parseable
		},
		{
			name: "nil value",
			pdu: gosnmp.SnmpPDU{
				Type:  gosnmp.Integer,
				Value: nil,
			},
			wantPrt: 0,
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPort, gotOK := snmp.ExportParseBridgePort(tt.pdu)
			if gotPort != tt.wantPrt {
				t.Errorf("ParseBridgePort() port = %v, want %v", gotPort, tt.wantPrt)
			}
			if gotOK != tt.wantOK {
				t.Errorf("ParseBridgePort() ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestCollectMACEntries(t *testing.T) {
	tests := []struct {
		name       string
		macToEntry map[string]*snmp.MACEntry
		wantLen    int
	}{
		{
			name:       "empty map",
			macToEntry: map[string]*snmp.MACEntry{},
			wantLen:    0,
		},
		{
			name: "single entry",
			macToEntry: map[string]*snmp.MACEntry{
				"00:11:22:33:44:55": {MAC: "00:11:22:33:44:55", VLAN: 1, IfIndex: 1},
			},
			wantLen: 1,
		},
		{
			name: "multiple entries",
			macToEntry: map[string]*snmp.MACEntry{
				"00:11:22:33:44:55": {MAC: "00:11:22:33:44:55", VLAN: 1, IfIndex: 1},
				"aa:bb:cc:dd:ee:ff": {MAC: "aa:bb:cc:dd:ee:ff", VLAN: 2, IfIndex: 2},
				"11:22:33:44:55:66": {MAC: "11:22:33:44:55:66", VLAN: 3, IfIndex: 3},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportCollectMACEntries(tt.macToEntry)
			if len(got) != tt.wantLen {
				t.Errorf("CollectMACEntries() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestInterfaceOIDConstantsExist(t *testing.T) {
	// Test that all interface OID constants are defined
	oids := []string{
		snmp.OIDIfIndex,
		snmp.OIDIfDescr,
		snmp.OIDIfType,
		snmp.OIDIfSpeed,
		snmp.OIDIfPhysAddress,
		snmp.OIDIfAdminStatus,
		snmp.OIDIfOperStatus,
		snmp.OIDIfLastChange,
		snmp.OIDIfName,
	}

	for _, oid := range oids {
		if oid == "" {
			t.Error("Expected OID to be non-empty")
		}
	}
}

func TestMACBridgeOIDConstants(t *testing.T) {
	// Test that MAC table OID constants are defined
	if snmp.OIDDot1dTpFdbAddress == "" {
		t.Error("OIDDot1dTpFdbAddress should not be empty")
	}
	if snmp.OIDDot1dTpFdbPort == "" {
		t.Error("OIDDot1dTpFdbPort should not be empty")
	}
	if snmp.OIDDot1dTpFdbStatus == "" {
		t.Error("OIDDot1dTpFdbStatus should not be empty")
	}
}
