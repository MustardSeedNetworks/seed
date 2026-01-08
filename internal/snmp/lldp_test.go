package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

func TestExtractLLDPIndex(t *testing.T) {
	tests := []struct {
		name          string
		oid           string
		wantLocalPort int
		wantRemoteIdx int
	}{
		{
			name:          "valid LLDP OID",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.24.1",
			wantLocalPort: 24,
			wantRemoteIdx: 1,
		},
		{
			name:          "multi-digit indices",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.100.25",
			wantLocalPort: 100,
			wantRemoteIdx: 25,
		},
		{
			name:          "short OID",
			oid:           "1.0",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
		{
			name:          "empty OID",
			oid:           "",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
		{
			name:          "invalid port number",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.invalid.1",
			wantLocalPort: 0,
			wantRemoteIdx: 0,
		},
		{
			name:          "invalid remote index",
			oid:           "1.0.8802.1.1.2.1.4.1.1.5.0.24.invalid",
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

func TestFormatChassisID(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "MAC address bytes",
			value: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			want:  "aa:bb:cc:dd:ee:ff",
		},
		{
			name:  "IPv4 address bytes",
			value: []byte{192, 168, 1, 1},
			want:  "192.168.1.1",
		},
		{
			name:  "printable string",
			value: []byte("switch-01"),
			want:  "switch-01",
		},
		{
			name:  "non-printable bytes",
			value: []byte{0x01, 0x02, 0x03},
			want:  "010203",
		},
		{
			name:  "non-byte value",
			value: "string-value",
			want:  "string-value",
		},
		{
			name:  "integer value",
			value: 12345,
			want:  "12345",
		},
		{
			name:  "nil value",
			value: nil,
			want:  "<nil>",
		},
		{
			name:  "empty bytes",
			value: []byte{},
			want:  "",
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

func TestIsPrintable(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"printable ASCII", []byte("Hello World"), true},
		{"all lowercase", []byte("abcdefghijklmnopqrstuvwxyz"), true},
		{"all uppercase", []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), true},
		{"digits", []byte("0123456789"), true},
		{"special chars", []byte("!@#$%^&*()_+-=[]{}|;':\",./<>?"), true},
		{"space", []byte(" "), true},
		{"with tab", []byte("hello\tworld"), false},
		{"with newline", []byte("hello\nworld"), false},
		{"with null", []byte("hello\x00world"), false},
		{"binary data", []byte{0x00, 0x01, 0x02}, false},
		{"empty", []byte{}, false},
		{"single printable", []byte("a"), true},
		{"high ASCII", []byte{0x80, 0x81}, false},
		{"DEL character", []byte{0x7f}, false},
		{"control character", []byte{0x1f}, false},
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

func TestParseChassisIDSubtype(t *testing.T) {
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
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high value", "99", snmp.StatusUnknown},
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

func TestParsePortIDSubtype(t *testing.T) {
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
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high value", "99", snmp.StatusUnknown},
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

func TestLLDPNeighborStruct(t *testing.T) {
	neighbor := snmp.LLDPNeighbor{
		LocalIfIndex:    1,
		LocalPortNum:    24,
		RemoteIndex:     1,
		ChassisIDType:   "macAddress",
		ChassisID:       "aa:bb:cc:dd:ee:ff",
		PortIDType:      "interfaceName",
		PortID:          "GigabitEthernet0/1",
		PortDescription: "Uplink Port",
		SystemName:      "switch-01",
		SystemDesc:      "Cisco IOS Switch",
		MgmtAddress:     "192.168.1.1",
	}

	// Verify all fields to avoid linter warnings
	if neighbor.LocalIfIndex != 1 {
		t.Errorf("LocalIfIndex = %v, want 1", neighbor.LocalIfIndex)
	}
	if neighbor.LocalPortNum != 24 {
		t.Errorf("LocalPortNum = %v, want 24", neighbor.LocalPortNum)
	}
	if neighbor.RemoteIndex != 1 {
		t.Errorf("RemoteIndex = %v, want 1", neighbor.RemoteIndex)
	}
	if neighbor.ChassisIDType != "macAddress" {
		t.Errorf("ChassisIDType = %v, want 'macAddress'", neighbor.ChassisIDType)
	}
	if neighbor.ChassisID != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("ChassisID = %v, want 'aa:bb:cc:dd:ee:ff'", neighbor.ChassisID)
	}
	if neighbor.PortIDType != "interfaceName" {
		t.Errorf("PortIDType = %v, want 'interfaceName'", neighbor.PortIDType)
	}
	if neighbor.PortID != "GigabitEthernet0/1" {
		t.Errorf("PortID = %v, want 'GigabitEthernet0/1'", neighbor.PortID)
	}
	if neighbor.PortDescription != "Uplink Port" {
		t.Errorf("PortDescription = %v, want 'Uplink Port'", neighbor.PortDescription)
	}
	if neighbor.SystemName != "switch-01" {
		t.Errorf("SystemName = %v, want 'switch-01'", neighbor.SystemName)
	}
	if neighbor.SystemDesc != "Cisco IOS Switch" {
		t.Errorf("SystemDesc = %v, want 'Cisco IOS Switch'", neighbor.SystemDesc)
	}
	if neighbor.MgmtAddress != "192.168.1.1" {
		t.Errorf("MgmtAddress = %v, want '192.168.1.1'", neighbor.MgmtAddress)
	}
}

func TestGetLLDPNeighbors(t *testing.T) {
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
			_, err := snmp.GetLLDPNeighbors(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetLLDPNeighbors() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetLLDPNeighbors() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestLLDPOIDConstants(t *testing.T) {
	// Verify LLDP-MIB OID constants are defined
	oids := map[string]string{
		"OIDLldpRemChassisIDSubtype": snmp.OIDLldpRemChassisIDSubtype,
		"OIDLldpRemChassisID":        snmp.OIDLldpRemChassisID,
		"OIDLldpRemPortIDSubtype":    snmp.OIDLldpRemPortIDSubtype,
		"OIDLldpRemPortID":           snmp.OIDLldpRemPortID,
		"OIDLldpRemPortDesc":         snmp.OIDLldpRemPortDesc,
		"OIDLldpRemSysName":          snmp.OIDLldpRemSysName,
		"OIDLldpRemSysDesc":          snmp.OIDLldpRemSysDesc,
		"OIDLldpRemManAddrIfSubtype": snmp.OIDLldpRemManAddrIfSubtype,
		"OIDLldpRemManAddrIfID":      snmp.OIDLldpRemManAddrIfID,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("%s is empty", name)
		}
		// All LLDP OIDs should start with 1.0.8802 (ISO 802.1)
		if len(oid) < 8 || oid[:8] != "1.0.8802" {
			t.Errorf("%s = %v, should start with 1.0.8802", name, oid)
		}
	}
}

func TestIDSubtypeLocalConstant(t *testing.T) {
	if snmp.IDSubtypeLocal != "local" {
		t.Errorf("IDSubtypeLocal = %v, want 'local'", snmp.IDSubtypeLocal)
	}
}
