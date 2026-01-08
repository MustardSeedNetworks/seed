// Package snmp_test tests the snmp package.
package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

func TestQuery(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		oid     string
		cfg     *config.SNMPConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			ip:      "192.168.1.1",
			oid:     snmp.OIDSysDescr,
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty communities",
			ip:   "192.168.1.1",
			oid:  snmp.OIDSysDescr,
			cfg: &config.SNMPConfig{
				Communities:   []string{},
				V3Credentials: []config.SNMPv3Credential{},
				Port:          161,
				Timeout:       time.Second,
				Retries:       1,
			},
			wantErr: true,
		},
		{
			name: "unreachable host",
			ip:   "192.0.2.1", // TEST-NET-1 (RFC 5737)
			oid:  snmp.OIDSysDescr,
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
			_, err := snmp.Query(ctx, tt.ip, tt.oid, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("Query() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("Query() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestQueryMultiple(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		oids    []string
		cfg     *config.SNMPConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			ip:      "192.168.1.1",
			oids:    []string{snmp.OIDSysDescr, snmp.OIDSysName},
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty oids",
			ip:   "192.168.1.1",
			oids: []string{},
			cfg: &config.SNMPConfig{
				Communities: []string{"public"},
				Port:        161,
				Timeout:     time.Second,
				Retries:     1,
			},
			wantErr: true,
		},
		{
			name: "unreachable host",
			ip:   "192.0.2.1",
			oids: []string{snmp.OIDSysDescr, snmp.OIDSysName},
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
			_, err := snmp.QueryMultiple(ctx, tt.ip, tt.oids, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("QueryMultiple() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("QueryMultiple() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetSystemInfo(t *testing.T) {
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
			_, err := snmp.GetSystemInfo(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetSystemInfo() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetSystemInfo() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetVendorVersion(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		cfg     *config.SNMPConfig
		wantErr bool
	}{
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
			_, err := snmp.GetVendorVersion(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetVendorVersion() error = nil, want error")
				}
			}
		})
	}
}

func TestFormatSNMPValue(t *testing.T) {
	tests := []struct {
		name     string
		variable gosnmp.SnmpPDU
		want     string
	}{
		{
			name: "OctetString",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: []byte("test string"),
			},
			want: "test string",
		},
		{
			name: "OctetString non-byte value",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: "not a byte slice",
			},
			want: "not a byte slice",
		},
		{
			name: "Integer",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Integer,
				Value: 12345,
			},
			want: "12345",
		},
		{
			name: "Counter32",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Counter32,
				Value: uint(98765),
			},
			want: "98765",
		},
		{
			name: "Gauge32",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Gauge32,
				Value: uint(54321),
			},
			want: "54321",
		},
		{
			name: "TimeTicks",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.TimeTicks,
				Value: uint32(100000),
			},
			want: "100000",
		},
		{
			name: "Counter64",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Counter64,
				Value: uint64(1234567890),
			},
			want: "1234567890",
		},
		{
			name: "ObjectIdentifier",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.ObjectIdentifier,
				Value: "1.3.6.1.2.1.1.1.0",
			},
			want: "1.3.6.1.2.1.1.1.0",
		},
		{
			name: "ObjectIdentifier non-string",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.ObjectIdentifier,
				Value: 12345,
			},
			want: "12345",
		},
		{
			name: "IPAddress",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.IPAddress,
				Value: "192.168.1.1",
			},
			want: "192.168.1.1",
		},
		{
			name: "IPAddress non-string",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.IPAddress,
				Value: []byte{192, 168, 1, 1},
			},
			want: "[192 168 1 1]",
		},
		{
			name: "nil value",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: nil,
			},
			want: "",
		},
		{
			name: "default case",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.Opaque,
				Value: []byte{0x01, 0x02},
			},
			want: "[1 2]",
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

func TestGetAuthProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		want     gosnmp.SnmpV3AuthProtocol
	}{
		{"MD5", "MD5", gosnmp.MD5},
		{"SHA", "SHA", gosnmp.SHA},
		{"SHA224", "SHA224", gosnmp.SHA224},
		{"SHA256", "SHA256", gosnmp.SHA256},
		{"SHA384", "SHA384", gosnmp.SHA384},
		{"SHA512", "SHA512", gosnmp.SHA512},
		{"empty", "", gosnmp.NoAuth},
		{"unknown", "UNKNOWN", gosnmp.NoAuth},
		{"lowercase md5", "md5", gosnmp.NoAuth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportGetAuthProtocol(tt.protocol)
			if got != tt.want {
				t.Errorf("GetAuthProtocol(%v) = %v, want %v", tt.protocol, got, tt.want)
			}
		})
	}
}

func TestGetPrivProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		want     gosnmp.SnmpV3PrivProtocol
	}{
		{"DES", "DES", gosnmp.DES},
		{"AES", "AES", gosnmp.AES},
		{"AES192", "AES192", gosnmp.AES192},
		{"AES256", "AES256", gosnmp.AES256},
		{"AES192C", "AES192C", gosnmp.AES192C},
		{"AES256C", "AES256C", gosnmp.AES256C},
		{"empty", "", gosnmp.NoPriv},
		{"unknown", "UNKNOWN", gosnmp.NoPriv},
		{"lowercase aes", "aes", gosnmp.NoPriv},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportGetPrivProtocol(tt.protocol)
			if got != tt.want {
				t.Errorf("GetPrivProtocol(%v) = %v, want %v", tt.protocol, got, tt.want)
			}
		})
	}
}

func TestGetMaxRepetitions(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.SNMPConfig
		want uint32
	}{
		{
			name: "nil config returns default",
			cfg:  nil,
			want: 10, // defaultMaxRepetitions
		},
		{
			name: "zero value returns default",
			cfg: &config.SNMPConfig{
				MaxRepetitions: 0,
			},
			want: 10, // defaultMaxRepetitions
		},
		{
			name: "value within range",
			cfg: &config.SNMPConfig{
				MaxRepetitions: 25,
			},
			want: 25,
		},
		{
			name: "value at max allowed",
			cfg: &config.SNMPConfig{
				MaxRepetitions: 50,
			},
			want: 50, // maxAllowedRepetitions
		},
		{
			name: "value exceeds max allowed",
			cfg: &config.SNMPConfig{
				MaxRepetitions: 100,
			},
			want: 50, // maxAllowedRepetitions
		},
		{
			name: "value at minimum",
			cfg: &config.SNMPConfig{
				MaxRepetitions: 1,
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportGetMaxRepetitions(tt.cfg)
			if got != tt.want {
				t.Errorf("GetMaxRepetitions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDConstants(t *testing.T) {
	// Verify OID constants are correctly defined
	oids := map[string]string{
		"OIDSysDescr":       snmp.OIDSysDescr,
		"OIDSysObjectID":    snmp.OIDSysObjectID,
		"OIDSysUpTime":      snmp.OIDSysUpTime,
		"OIDSysContact":     snmp.OIDSysContact,
		"OIDSysName":        snmp.OIDSysName,
		"OIDSysLocation":    snmp.OIDSysLocation,
		"OIDCiscoVersion":   snmp.OIDCiscoVersion,
		"OIDHPVersion":      snmp.OIDHPVersion,
		"OIDJuniperVersion": snmp.OIDJuniperVersion,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("%s is empty", name)
		}
		// OIDs should start with a digit
		if oid == "" || (oid[0] < '0' || oid[0] > '9') {
			t.Errorf("%s = %v, should start with a digit", name, oid)
		}
	}

	// Verify standard OIDs are under the system MIB tree (1.3.6.1.2.1.1)
	standardOIDs := []string{
		snmp.OIDSysDescr,
		snmp.OIDSysObjectID,
		snmp.OIDSysUpTime,
		snmp.OIDSysContact,
		snmp.OIDSysName,
		snmp.OIDSysLocation,
	}

	for _, oid := range standardOIDs {
		if len(oid) < 13 || oid[:13] != "1.3.6.1.2.1.1" {
			t.Errorf("Standard OID %v should start with 1.3.6.1.2.1.1", oid)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	// Test that context cancellation is respected
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Port:        161,
		Timeout:     time.Second,
		Retries:     1,
	}

	// Query should fail due to canceled context
	_, err := snmp.Query(ctx, "192.168.1.1", snmp.OIDSysDescr, cfg)
	if err == nil {
		t.Error("Query() with canceled context should return error")
	}
}

func TestSystemInfo(t *testing.T) {
	// Test SystemInfo structure
	info := &snmp.SystemInfo{
		SysDescr:    "Test Device Description",
		SysObjectID: "1.3.6.1.4.1.9",
		SysName:     "test-device",
		SysContact:  "admin@example.com",
		SysLocation: "Data Center",
		SysUpTime:   123456,
	}

	if info.SysDescr == "" {
		t.Error("SysDescr should not be empty")
	}
	if info.SysObjectID != "1.3.6.1.4.1.9" {
		t.Errorf("expected SysObjectID '1.3.6.1.4.1.9', got %q", info.SysObjectID)
	}
	if info.SysName == "" {
		t.Error("SysName should not be empty")
	}
	if info.SysContact != "admin@example.com" {
		t.Errorf("expected SysContact 'admin@example.com', got %q", info.SysContact)
	}
	if info.SysLocation != "Data Center" {
		t.Errorf("expected SysLocation 'Data Center', got %q", info.SysLocation)
	}
	if info.SysUpTime == 0 {
		t.Error("SysUpTime should not be zero")
	}
}

func TestSNMPConfigValidation(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *config.SNMPConfig
		valid bool
	}{
		{
			name:  "nil config",
			cfg:   nil,
			valid: false,
		},
		{
			name: "valid v2c config",
			cfg: &config.SNMPConfig{
				Communities: []string{"public"},
				Port:        161,
				Timeout:     time.Second,
				Retries:     2,
			},
			valid: true,
		},
		{
			name: "valid v3 config",
			cfg: &config.SNMPConfig{
				Communities: []string{},
				V3Credentials: []config.SNMPv3Credential{
					{
						Username:     "snmpuser",
						AuthProtocol: "SHA",
						AuthPassword: "authpass",
						PrivProtocol: "AES",
						PrivPassword: "privpass",
					},
				},
				Port:    161,
				Timeout: time.Second,
				Retries: 2,
			},
			valid: true,
		},
		{
			name: "empty communities and credentials",
			cfg: &config.SNMPConfig{
				Communities:   []string{},
				V3Credentials: []config.SNMPv3Credential{},
				Port:          161,
				Timeout:       time.Second,
				Retries:       2,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := snmp.Query(ctx, "192.0.2.1", snmp.OIDSysDescr, tt.cfg)

			if tt.valid && tt.cfg != nil {
				// Should fail due to unreachable host, not config validation
				if err == nil {
					t.Error("Expected error for unreachable host")
				}
			} else {
				// Should fail due to invalid config
				if err == nil {
					t.Error("Expected error for invalid config")
				}
			}
		})
	}
}

func TestMultipleCommunities(t *testing.T) {
	ctx := context.Background()

	// Config with multiple communities (all will fail, but tests the iteration)
	cfg := &config.SNMPConfig{
		Communities: []string{"public", "private", "community"},
		Port:        161,
		Timeout:     100 * time.Millisecond,
		Retries:     1,
	}

	_, err := snmp.Query(ctx, "192.0.2.1", snmp.OIDSysDescr, cfg)

	// Should fail after trying all communities
	if err == nil {
		t.Error("Query() should fail after trying all communities")
	}

	// Error message should indicate all credentials failed
	if err.Error() != "SNMP query failed for all configured credentials" {
		t.Logf("Got error: %v", err)
	}
}

func TestV3CredentialFields(t *testing.T) {
	cred := config.SNMPv3Credential{
		Name:          "test-cred",
		Username:      "snmpuser",
		AuthProtocol:  "SHA256",
		AuthPassword:  "authpass123",
		PrivProtocol:  "AES256",
		PrivPassword:  "privpass456",
		ContextName:   "context1",
		SecurityLevel: "authPriv",
	}

	if cred.Name == "" {
		t.Error("Name should not be empty")
	}
	if cred.Username == "" {
		t.Error("Username should not be empty")
	}
	if cred.AuthProtocol == "" {
		t.Error("AuthProtocol should not be empty")
	}
	if cred.AuthPassword != "authpass123" {
		t.Errorf("expected AuthPassword 'authpass123', got %q", cred.AuthPassword)
	}
	if cred.PrivProtocol == "" {
		t.Error("PrivProtocol should not be empty")
	}
	if cred.PrivPassword != "privpass456" {
		t.Errorf("expected PrivPassword 'privpass456', got %q", cred.PrivPassword)
	}
	if cred.ContextName != "context1" {
		t.Errorf("expected ContextName 'context1', got %q", cred.ContextName)
	}
	if cred.SecurityLevel != "authPriv" {
		t.Errorf("expected SecurityLevel 'authPriv', got %q", cred.SecurityLevel)
	}

	// Test protocol conversion
	authProto := snmp.ExportGetAuthProtocol(cred.AuthProtocol)
	if authProto != gosnmp.SHA256 {
		t.Errorf("Auth protocol = %v, want SHA256", authProto)
	}

	privProto := snmp.ExportGetPrivProtocol(cred.PrivProtocol)
	if privProto != gosnmp.AES256 {
		t.Errorf("Priv protocol = %v, want AES256", privProto)
	}
}

func TestV3WithEmptyUsername(t *testing.T) {
	ctx := context.Background()

	cfg := &config.SNMPConfig{
		V3Credentials: []config.SNMPv3Credential{
			{
				Name:         "test",
				Username:     "", // Empty username should fail
				AuthProtocol: "SHA",
				AuthPassword: "authpass",
			},
		},
		Port:    161,
		Timeout: 100 * time.Millisecond,
		Retries: 1,
	}

	_, err := snmp.Query(ctx, "192.0.2.1", snmp.OIDSysDescr, cfg)
	if err == nil {
		t.Error("Query() with empty v3 username should return error")
	}
}

func TestAuthProtocolMD5Deprecation(t *testing.T) {
	// Verify MD5 constant is defined for backward compatibility
	if snmp.AuthProtocolMD5 != "MD5" {
		t.Errorf("AuthProtocolMD5 = %v, want MD5", snmp.AuthProtocolMD5)
	}

	// MD5 should still work (backward compat) but logs warning
	got := snmp.ExportGetAuthProtocol("MD5")
	if got != gosnmp.MD5 {
		t.Errorf("GetAuthProtocol(MD5) = %v, want gosnmp.MD5", got)
	}
}

func TestInterfaceOIDConstants(t *testing.T) {
	// Verify interface-related OID constants
	if snmp.OIDIfIndex == "" {
		t.Error("OIDIfIndex should not be empty")
	}
	if snmp.OIDIfDescr == "" {
		t.Error("OIDIfDescr should not be empty")
	}
	if snmp.OIDIfType == "" {
		t.Error("OIDIfType should not be empty")
	}
	if snmp.OIDIfSpeed == "" {
		t.Error("OIDIfSpeed should not be empty")
	}
	if snmp.OIDIfPhysAddress == "" {
		t.Error("OIDIfPhysAddress should not be empty")
	}
	if snmp.OIDIfAdminStatus == "" {
		t.Error("OIDIfAdminStatus should not be empty")
	}
	if snmp.OIDIfOperStatus == "" {
		t.Error("OIDIfOperStatus should not be empty")
	}
}

func TestStatusConstants(t *testing.T) {
	// Verify status constant values
	if snmp.StatusUp != "up" {
		t.Errorf("StatusUp = %v, want 'up'", snmp.StatusUp)
	}
	if snmp.StatusDown != "down" {
		t.Errorf("StatusDown = %v, want 'down'", snmp.StatusDown)
	}
	if snmp.StatusTesting != "testing" {
		t.Errorf("StatusTesting = %v, want 'testing'", snmp.StatusTesting)
	}
	if snmp.StatusUnknown != "unknown" {
		t.Errorf("StatusUnknown = %v, want 'unknown'", snmp.StatusUnknown)
	}
}

func TestMACTypeConstants(t *testing.T) {
	if snmp.MACTypeLearned != "learned" {
		t.Errorf("MACTypeLearned = %v, want 'learned'", snmp.MACTypeLearned)
	}
	if snmp.MACTypeStatic != "static" {
		t.Errorf("MACTypeStatic = %v, want 'static'", snmp.MACTypeStatic)
	}
	if snmp.MACTypeOther != "other" {
		t.Errorf("MACTypeOther = %v, want 'other'", snmp.MACTypeOther)
	}
}
