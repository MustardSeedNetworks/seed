package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

func TestExtractVLANIndex(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want int
	}{
		{
			name: "valid VLAN ID",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.100",
			want: 100,
		},
		{
			name: "VLAN 1",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.1",
			want: 1,
		},
		{
			name: "VLAN 4094",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.4094",
			want: 4094,
		},
		{
			name: "short OID",
			oid:  "1",
			want: 0,
		},
		{
			name: "empty OID",
			oid:  "",
			want: 0,
		},
		{
			name: "invalid VLAN ID",
			oid:  "1.3.6.1.2.1.17.7.1.4.3.1.1.invalid",
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

func TestParsePortBitmap(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []int
	}{
		{
			name:  "single port (port 1)",
			value: []byte{0x80}, // 10000000
			want:  []int{1},
		},
		{
			name:  "ports 1 and 8",
			value: []byte{0x81}, // 10000001
			want:  []int{1, 8},
		},
		{
			name:  "all first 8 ports",
			value: []byte{0xFF}, // 11111111
			want:  []int{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:  "ports 9 and 16",
			value: []byte{0x00, 0x81}, // 00000000 10000001
			want:  []int{9, 16},
		},
		{
			name:  "port 24",
			value: []byte{0x00, 0x00, 0x01}, // 00000000 00000000 00000001
			want:  []int{24},
		},
		{
			name:  "empty bitmap",
			value: []byte{0x00},
			want:  []int{},
		},
		{
			name:  "no bytes",
			value: []byte{},
			want:  []int{},
		},
		{
			name:  "non-byte value",
			value: "not bytes",
			want:  nil,
		},
		{
			name:  "nil value",
			value: nil,
			want:  nil,
		},
		{
			name:  "integer value",
			value: 12345,
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

func TestParseRowStatus(t *testing.T) {
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
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high value", "99", snmp.StatusUnknown},
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

func TestVLANInfoStruct(t *testing.T) {
	vlan := snmp.VLANInfo{
		ID:          100,
		Name:        "Management",
		Status:      "active",
		EgressPorts: []int{1, 2, 3, 24},
		Type:        "static",
	}

	if vlan.ID != 100 {
		t.Errorf("ID = %v, want 100", vlan.ID)
	}
	if vlan.Name != "Management" {
		t.Errorf("Name = %v, want 'Management'", vlan.Name)
	}
	if vlan.Status != "active" {
		t.Errorf("Status = %v, want 'active'", vlan.Status)
	}
	if len(vlan.EgressPorts) != 4 {
		t.Errorf("EgressPorts len = %v, want 4", len(vlan.EgressPorts))
	}
	if vlan.Type != "static" {
		t.Errorf("Type = %v, want 'static'", vlan.Type)
	}
}

func TestGetVLANs(t *testing.T) {
	ctx := context.Background()

	t.Run("nil config", func(t *testing.T) {
		_, err := snmp.GetVLANs(ctx, "192.168.1.1", nil)
		if err == nil {
			t.Error("GetVLANs() with nil config should return error")
		}
	})

	t.Run("empty communities", func(t *testing.T) {
		cfg := &config.SNMPConfig{
			Communities: []string{},
			Port:        161,
			Timeout:     100 * time.Millisecond,
			Retries:     1,
		}
		_, err := snmp.GetVLANs(ctx, "192.168.1.1", cfg)
		// With empty communities and no v3 credentials, should fail
		if err == nil {
			t.Error("GetVLANs() with empty communities should return error")
		}
	})
}

func TestVLANOIDConstants(t *testing.T) {
	// Verify Q-BRIDGE-MIB OID constants are defined
	oids := map[string]string{
		"OIDDot1qVlanStaticName":        snmp.OIDDot1qVlanStaticName,
		"OIDDot1qVlanStaticEgressPorts": snmp.OIDDot1qVlanStaticEgressPorts,
		"OIDDot1qVlanStaticRowStatus":   snmp.OIDDot1qVlanStaticRowStatus,
		"OIDDot1qVlanFdbID":             snmp.OIDDot1qVlanFdbID,
		"OIDDot1qPvid":                  snmp.OIDDot1qPvid,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("%s is empty", name)
		}
		// Q-BRIDGE OIDs should be under 1.3.6.1.2.1.17.7 (BRIDGE-MIB.7)
		if len(oid) < 16 || oid[:16] != "1.3.6.1.2.1.17.7" {
			t.Errorf("%s = %v, should start with 1.3.6.1.2.1.17.7", name, oid)
		}
	}
}

func TestVLANInfoDefaultStatus(t *testing.T) {
	// When status is not explicitly set, it should default to empty
	vlan := snmp.VLANInfo{
		ID:   1,
		Name: "default",
	}

	// Verify explicitly set fields
	if vlan.ID != 1 {
		t.Errorf("ID = %v, want 1", vlan.ID)
	}
	if vlan.Name != "default" {
		t.Errorf("Name = %v, want 'default'", vlan.Name)
	}
	// Verify default value
	if vlan.Status != "" {
		t.Errorf("Default Status = %v, want empty string", vlan.Status)
	}
}

func TestVLANPortBitmapLargeValues(t *testing.T) {
	// Test parsing a large port bitmap (48 ports)
	bitmap := make([]byte, 6) // 48 ports
	bitmap[0] = 0xFF          // ports 1-8
	bitmap[5] = 0xFF          // ports 41-48

	ports := snmp.ExportParsePortBitmap(bitmap)

	// Should have 16 ports (8 from first byte + 8 from last byte)
	if len(ports) != 16 {
		t.Errorf("ParsePortBitmap large bitmap = %v ports, want 16", len(ports))
	}

	// Check first and last port numbers
	if ports[0] != 1 {
		t.Errorf("First port = %v, want 1", ports[0])
	}
	if ports[len(ports)-1] != 48 {
		t.Errorf("Last port = %v, want 48", ports[len(ports)-1])
	}
}

func TestVLANInfoWithEmptyEgressPorts(t *testing.T) {
	vlan := snmp.VLANInfo{
		ID:          999,
		Name:        "isolated",
		Status:      "active",
		EgressPorts: []int{},
		Type:        "static",
	}

	// Verify all fields to avoid linter warnings
	if vlan.ID != 999 {
		t.Errorf("ID = %v, want 999", vlan.ID)
	}
	if vlan.Name != "isolated" {
		t.Errorf("Name = %v, want 'isolated'", vlan.Name)
	}
	if vlan.Status != "active" {
		t.Errorf("Status = %v, want 'active'", vlan.Status)
	}
	if vlan.Type != "static" {
		t.Errorf("Type = %v, want 'static'", vlan.Type)
	}
	if vlan.EgressPorts == nil {
		t.Error("EgressPorts should be empty slice, not nil")
	}
	if len(vlan.EgressPorts) != 0 {
		t.Errorf("EgressPorts len = %v, want 0", len(vlan.EgressPorts))
	}
}
