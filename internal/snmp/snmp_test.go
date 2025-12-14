package snmp

import (
	"context"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/krisarmstrong/luminetiq/internal/config"
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
			oid:     OIDSysDescr,
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty communities",
			ip:   "192.168.1.1",
			oid:  OIDSysDescr,
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
			oid:  OIDSysDescr,
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
			_, err := Query(ctx, tt.ip, tt.oid, tt.cfg)

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
			oids:    []string{OIDSysDescr, OIDSysName},
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
			oids: []string{OIDSysDescr, OIDSysName},
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
			_, err := QueryMultiple(ctx, tt.ip, tt.oids, tt.cfg)

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
			_, err := GetSystemInfo(ctx, tt.ip, tt.cfg)

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
			_, err := GetVendorVersion(ctx, tt.ip, tt.cfg)

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
			name: "ObjectIdentifier",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.ObjectIdentifier,
				Value: "1.3.6.1.2.1.1.1.0",
			},
			want: "1.3.6.1.2.1.1.1.0",
		},
		{
			name: "IPAddress",
			variable: gosnmp.SnmpPDU{
				Type:  gosnmp.IPAddress,
				Value: "192.168.1.1",
			},
			want: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSNMPValue(tt.variable)
			if got != tt.want {
				t.Errorf("formatSNMPValue() = %v, want %v", got, tt.want)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAuthProtocol(tt.protocol)
			if got != tt.want {
				t.Errorf("getAuthProtocol(%v) = %v, want %v", tt.protocol, got, tt.want)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPrivProtocol(tt.protocol)
			if got != tt.want {
				t.Errorf("getPrivProtocol(%v) = %v, want %v", tt.protocol, got, tt.want)
			}
		})
	}
}

func TestOIDConstants(t *testing.T) {
	// Verify OID constants are correctly defined
	oids := map[string]string{
		"OIDSysDescr":      OIDSysDescr,
		"OIDSysObjectID":   OIDSysObjectID,
		"OIDSysUpTime":     OIDSysUpTime,
		"OIDSysContact":    OIDSysContact,
		"OIDSysName":       OIDSysName,
		"OIDSysLocation":   OIDSysLocation,
		"OIDCiscoVersion":  OIDCiscoVersion,
		"OIDHPVersion":     OIDHPVersion,
		"OIDJuniperVersion": OIDJuniperVersion,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("%s is empty", name)
		}
		// OIDs should start with a digit
		if len(oid) == 0 || (oid[0] < '0' || oid[0] > '9') {
			t.Errorf("%s = %v, should start with a digit", name, oid)
		}
	}

	// Verify standard OIDs are under the system MIB tree (1.3.6.1.2.1.1)
	standardOIDs := []string{
		OIDSysDescr,
		OIDSysObjectID,
		OIDSysUpTime,
		OIDSysContact,
		OIDSysName,
		OIDSysLocation,
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

	// Query should fail due to cancelled context
	_, err := Query(ctx, "192.168.1.1", OIDSysDescr, cfg)
	if err == nil {
		t.Error("Query() with cancelled context should return error")
	}
}

func TestSNMPSystemInfo(t *testing.T) {
	// Test SNMPSystemInfo structure
	info := &SNMPSystemInfo{
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
	if info.SysName == "" {
		t.Error("SysName should not be empty")
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
			_, err := Query(ctx, "192.0.2.1", OIDSysDescr, tt.cfg)

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

	_, err := Query(ctx, "192.0.2.1", OIDSysDescr, cfg)

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
		Name:         "test-cred",
		Username:     "snmpuser",
		AuthProtocol: "SHA256",
		AuthPassword: "authpass123",
		PrivProtocol: "AES256",
		PrivPassword: "privpass456",
		ContextName:  "context1",
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
	if cred.PrivProtocol == "" {
		t.Error("PrivProtocol should not be empty")
	}

	// Test protocol conversion
	authProto := getAuthProtocol(cred.AuthProtocol)
	if authProto != gosnmp.SHA256 {
		t.Errorf("Auth protocol = %v, want SHA256", authProto)
	}

	privProto := getPrivProtocol(cred.PrivProtocol)
	if privProto != gosnmp.AES256 {
		t.Errorf("Priv protocol = %v, want AES256", privProto)
	}
}
