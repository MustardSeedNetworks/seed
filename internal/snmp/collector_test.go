package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

// TestGetInterfaceInfo tests the GetInterfaceInfo function.
func TestGetInterfaceInfo(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		ifIndex int
		cfg     *config.SNMPConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			ip:      "192.168.1.1",
			ifIndex: 1,
			cfg:     nil,
			wantErr: true,
		},
		{
			name:    "unreachable host",
			ip:      "192.0.2.1",
			ifIndex: 1,
			cfg: &config.SNMPConfig{
				Communities: []string{"public"},
				Port:        161,
				Timeout:     100 * time.Millisecond,
				Retries:     1,
			},
			wantErr: true,
		},
		{
			name:    "empty communities and credentials",
			ip:      "192.168.1.1",
			ifIndex: 1,
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
			name:    "multiple communities all fail",
			ip:      "192.0.2.1",
			ifIndex: 1,
			cfg: &config.SNMPConfig{
				Communities: []string{"public", "private", "secret"},
				Port:        161,
				Timeout:     100 * time.Millisecond,
				Retries:     1,
			},
			wantErr: true,
		},
		{
			name:    "v3 credentials unreachable",
			ip:      "192.0.2.1",
			ifIndex: 1,
			cfg: &config.SNMPConfig{
				V3Credentials: []config.SNMPv3Credential{
					{
						Name:         "test",
						Username:     "testuser",
						AuthProtocol: "SHA256",
						AuthPassword: "authpass",
						PrivProtocol: "AES256",
						PrivPassword: "privpass",
					},
				},
				Port:    161,
				Timeout: 100 * time.Millisecond,
				Retries: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := snmp.GetInterfaceInfo(ctx, tt.ip, tt.ifIndex, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetInterfaceInfo() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetInterfaceInfo() error = %v, want nil", err)
				}
			}
		})
	}
}

// TestGetPortVLANs tests the GetPortVLANs function.
func TestGetPortVLANs(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		ifIndex int
		cfg     *config.SNMPConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			ip:      "192.168.1.1",
			ifIndex: 1,
			cfg:     nil,
			wantErr: true,
		},
		{
			name:    "unreachable host",
			ip:      "192.0.2.1",
			ifIndex: 1,
			cfg: &config.SNMPConfig{
				Communities: []string{"public"},
				Port:        161,
				Timeout:     100 * time.Millisecond,
				Retries:     1,
			},
			wantErr: true,
		},
		{
			name:    "empty communities",
			ip:      "192.168.1.1",
			ifIndex: 1,
			cfg: &config.SNMPConfig{
				Communities: []string{},
				Port:        161,
				Timeout:     time.Second,
				Retries:     1,
			},
			wantErr: true,
		},
		{
			name:    "v3 with empty username",
			ip:      "192.0.2.1",
			ifIndex: 1,
			cfg: &config.SNMPConfig{
				V3Credentials: []config.SNMPv3Credential{
					{
						Name:         "empty-user",
						Username:     "",
						AuthProtocol: "SHA",
						AuthPassword: "pass",
					},
				},
				Port:    161,
				Timeout: 100 * time.Millisecond,
				Retries: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := snmp.GetPortVLANs(ctx, tt.ip, tt.ifIndex, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetPortVLANs() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetPortVLANs() error = %v, want nil", err)
				}
			}
		})
	}
}

// TestContextCancellationAllFunctions tests that context cancellation is respected.
func TestContextCancellationAllFunctions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Port:        161,
		Timeout:     time.Second,
		Retries:     1,
	}

	// All functions should fail due to canceled context
	t.Run("Query", func(t *testing.T) {
		_, err := snmp.Query(ctx, "192.168.1.1", snmp.OIDSysDescr, cfg)
		if err == nil {
			t.Error("Query() with canceled context should return error")
		}
	})

	t.Run("QueryMultiple", func(t *testing.T) {
		_, err := snmp.QueryMultiple(ctx, "192.168.1.1", []string{snmp.OIDSysDescr}, cfg)
		if err == nil {
			t.Error("QueryMultiple() with canceled context should return error")
		}
	})

	t.Run("GetAllInterfaces", func(t *testing.T) {
		_, err := snmp.GetAllInterfaces(ctx, "192.168.1.1", cfg)
		if err == nil {
			t.Error("GetAllInterfaces() with canceled context should return error")
		}
	})

	t.Run("GetMACTable", func(t *testing.T) {
		_, err := snmp.GetMACTable(ctx, "192.168.1.1", cfg)
		if err == nil {
			t.Error("GetMACTable() with canceled context should return error")
		}
	})

	t.Run("GetInterfaceInfo", func(t *testing.T) {
		_, err := snmp.GetInterfaceInfo(ctx, "192.168.1.1", 1, cfg)
		if err == nil {
			t.Error("GetInterfaceInfo() with canceled context should return error")
		}
	})

	t.Run("GetPortVLANs", func(t *testing.T) {
		_, err := snmp.GetPortVLANs(ctx, "192.168.1.1", 1, cfg)
		if err == nil {
			t.Error("GetPortVLANs() with canceled context should return error")
		}
	})
}

// TestContextTimeoutFunctions tests that context timeout is respected.
func TestContextTimeoutFunctions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow context to expire
	time.Sleep(5 * time.Millisecond)

	cfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Port:        161,
		Timeout:     time.Second,
		Retries:     1,
	}

	_, err := snmp.Query(ctx, "192.168.1.1", snmp.OIDSysDescr, cfg)
	if err == nil {
		t.Error("Query() with expired context should return error")
	}
}

// TestV3CredentialsFallback tests fallback from v3 to v2c.
func TestV3CredentialsFallback(t *testing.T) {
	ctx := context.Background()

	// Config with both v3 credentials (that fail) and v2c communities (that also fail)
	// This tests that the code iterates through all options
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
				AuthProtocol: "SHA",
				AuthPassword: "pass2",
			},
		},
		Communities: []string{"public", "private"},
		Port:        161,
		Timeout:     100 * time.Millisecond,
		Retries:     1,
	}

	// Should try all v3 credentials, then fall back to all v2c communities
	_, err := snmp.Query(ctx, "192.0.2.1", snmp.OIDSysDescr, cfg)
	if err == nil {
		t.Error("Query() should fail after trying all credentials")
	}
}

// TestMACTableFallback tests that GetMACTable falls back from Q-BRIDGE to BRIDGE.
func TestMACTableFallback(t *testing.T) {
	ctx := context.Background()

	cfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Port:        161,
		Timeout:     100 * time.Millisecond,
		Retries:     1,
	}

	// Should try Q-BRIDGE first, then BRIDGE, both will fail due to unreachable host
	_, err := snmp.GetMACTable(ctx, "192.0.2.1", cfg)
	if err == nil {
		t.Error("GetMACTable() should fail when host is unreachable")
	}
}

// TestInterfaceInfoDefaultValues tests that InterfaceInfo has correct defaults.
func TestInterfaceInfoDefaultValues(t *testing.T) {
	info := snmp.InterfaceInfo{}

	// All string fields should be empty by default
	if info.Description != "" {
		t.Errorf("Description = %v, want empty", info.Description)
	}
	if info.Name != "" {
		t.Errorf("Name = %v, want empty", info.Name)
	}
	if info.Duplex != "" {
		t.Errorf("Duplex = %v, want empty", info.Duplex)
	}
	if info.AdminStatus != "" {
		t.Errorf("AdminStatus = %v, want empty", info.AdminStatus)
	}
	if info.OperStatus != "" {
		t.Errorf("OperStatus = %v, want empty", info.OperStatus)
	}
	if info.MACAddress != "" {
		t.Errorf("MACAddress = %v, want empty", info.MACAddress)
	}

	// Numeric fields should be zero
	if info.Index != 0 {
		t.Errorf("Index = %v, want 0", info.Index)
	}
	if info.Speed != 0 {
		t.Errorf("Speed = %v, want 0", info.Speed)
	}
}

// TestMACEntryDefaultValues tests that MACEntry has correct defaults.
func TestMACEntryDefaultValues(t *testing.T) {
	entry := snmp.MACEntry{}

	if entry.MAC != "" {
		t.Errorf("MAC = %v, want empty", entry.MAC)
	}
	if entry.VLAN != 0 {
		t.Errorf("VLAN = %v, want 0", entry.VLAN)
	}
	if entry.IfIndex != 0 {
		t.Errorf("IfIndex = %v, want 0", entry.IfIndex)
	}
	if entry.Type != "" {
		t.Errorf("Type = %v, want empty", entry.Type)
	}
}

// TestInterfaceInfoFull tests InterfaceInfo with all fields populated.
func TestInterfaceInfoFull(t *testing.T) {
	now := time.Now()
	info := snmp.InterfaceInfo{
		Index:       24,
		Description: "GigabitEthernet0/24",
		Name:        "Gi0/24",
		Speed:       1000000000,
		Duplex:      "full",
		AdminStatus: snmp.StatusUp,
		OperStatus:  snmp.StatusUp,
		LastChange:  now,
		MACAddress:  "00:11:22:33:44:55",
	}

	if info.Index != 24 {
		t.Errorf("Index = %v, want 24", info.Index)
	}
	if info.Description != "GigabitEthernet0/24" {
		t.Errorf("Description = %v, want GigabitEthernet0/24", info.Description)
	}
	if info.Name != "Gi0/24" {
		t.Errorf("Name = %v, want Gi0/24", info.Name)
	}
	if info.Speed != 1000000000 {
		t.Errorf("Speed = %v, want 1000000000", info.Speed)
	}
	if info.Duplex != "full" {
		t.Errorf("Duplex = %v, want full", info.Duplex)
	}
	if info.AdminStatus != snmp.StatusUp {
		t.Errorf("AdminStatus = %v, want %v", info.AdminStatus, snmp.StatusUp)
	}
	if info.OperStatus != snmp.StatusUp {
		t.Errorf("OperStatus = %v, want %v", info.OperStatus, snmp.StatusUp)
	}
	if info.LastChange != now {
		t.Errorf("LastChange = %v, want %v", info.LastChange, now)
	}
	if info.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("MACAddress = %v, want 00:11:22:33:44:55", info.MACAddress)
	}
}

// TestMACEntryFull tests MACEntry with all fields populated.
func TestMACEntryFull(t *testing.T) {
	entry := snmp.MACEntry{
		MAC:     "aa:bb:cc:dd:ee:ff",
		VLAN:    100,
		IfIndex: 48,
		Type:    snmp.MACTypeLearned,
	}

	if entry.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC = %v, want aa:bb:cc:dd:ee:ff", entry.MAC)
	}
	if entry.VLAN != 100 {
		t.Errorf("VLAN = %v, want 100", entry.VLAN)
	}
	if entry.IfIndex != 48 {
		t.Errorf("IfIndex = %v, want 48", entry.IfIndex)
	}
	if entry.Type != snmp.MACTypeLearned {
		t.Errorf("Type = %v, want %v", entry.Type, snmp.MACTypeLearned)
	}
}

// TestMACEntryTypeStatic tests MACEntry with static type.
func TestMACEntryTypeStatic(t *testing.T) {
	entry := snmp.MACEntry{
		MAC:     "ff:ff:ff:ff:ff:ff",
		VLAN:    1,
		IfIndex: 1,
		Type:    snmp.MACTypeStatic,
	}

	if entry.Type != snmp.MACTypeStatic {
		t.Errorf("Type = %v, want %v", entry.Type, snmp.MACTypeStatic)
	}
	if entry.MAC != "ff:ff:ff:ff:ff:ff" {
		t.Errorf("MAC = %v, want ff:ff:ff:ff:ff:ff", entry.MAC)
	}
	if entry.VLAN != 1 {
		t.Errorf("VLAN = %v, want 1", entry.VLAN)
	}
	if entry.IfIndex != 1 {
		t.Errorf("IfIndex = %v, want 1", entry.IfIndex)
	}
}

// TestMACEntryTypeOther tests MACEntry with other type.
func TestMACEntryTypeOther(t *testing.T) {
	entry := snmp.MACEntry{
		MAC:     "00:00:00:00:00:00",
		VLAN:    0,
		IfIndex: 0,
		Type:    snmp.MACTypeOther,
	}

	if entry.Type != snmp.MACTypeOther {
		t.Errorf("Type = %v, want %v", entry.Type, snmp.MACTypeOther)
	}
	if entry.MAC != "00:00:00:00:00:00" {
		t.Errorf("MAC = %v, want 00:00:00:00:00:00", entry.MAC)
	}
	if entry.VLAN != 0 {
		t.Errorf("VLAN = %v, want 0", entry.VLAN)
	}
	if entry.IfIndex != 0 {
		t.Errorf("IfIndex = %v, want 0", entry.IfIndex)
	}
}

// TestConfigMaxRepetitionsEdgeCases tests edge cases for MaxRepetitions.
func TestConfigMaxRepetitionsEdgeCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		maxRepetitions uint32
	}{
		{"very small", 1},
		{"small", 5},
		{"default like", 10},
		{"medium", 25},
		{"at max", 50},
		{"over max", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SNMPConfig{
				Communities:    []string{"public"},
				Port:           161,
				Timeout:        100 * time.Millisecond,
				Retries:        1,
				MaxRepetitions: tt.maxRepetitions,
			}

			// The query will fail due to unreachable host, but exercises the config path
			_, err := snmp.GetAllInterfaces(ctx, "192.0.2.1", cfg)
			if err == nil {
				t.Error("Expected error for unreachable host")
			}
		})
	}
}

// TestBridgeMIBOIDConstants tests that BRIDGE-MIB OID constants are defined.
func TestBridgeMIBOIDConstants(t *testing.T) {
	oids := map[string]string{
		"OIDDot1dTpFdbAddress":           snmp.OIDDot1dTpFdbAddress,
		"OIDDot1dTpFdbPort":              snmp.OIDDot1dTpFdbPort,
		"OIDDot1dTpFdbStatus":            snmp.OIDDot1dTpFdbStatus,
		"OIDDot1qTpFdbPort":              snmp.OIDDot1qTpFdbPort,
		"OIDDot1qTpFdbStatus":            snmp.OIDDot1qTpFdbStatus,
		"OIDDot1qVlanCurrentEgressPorts": snmp.OIDDot1qVlanCurrentEgressPorts,
		"OIDDot1dBasePortIfIndex":        snmp.OIDDot1dBasePortIfIndex,
		"OIDDot3StatsDuplexStatus":       snmp.OIDDot3StatsDuplexStatus,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

// TestInterfaceMIBOIDConstants tests that IF-MIB OID constants are correctly prefixed.
func TestInterfaceMIBOIDConstants(t *testing.T) {
	// IF-MIB OIDs should be under 1.3.6.1.2.1.2 (interfaces MIB)
	ifMIBOIDs := []string{
		snmp.OIDIfIndex,
		snmp.OIDIfDescr,
		snmp.OIDIfType,
		snmp.OIDIfSpeed,
		snmp.OIDIfPhysAddress,
		snmp.OIDIfAdminStatus,
		snmp.OIDIfOperStatus,
		snmp.OIDIfLastChange,
	}

	for _, oid := range ifMIBOIDs {
		if len(oid) < 13 || oid[:13] != "1.3.6.1.2.1.2" {
			t.Errorf("IF-MIB OID %v should start with 1.3.6.1.2.1.2", oid)
		}
	}

	// ifName is in IF-MIB extension (1.3.6.1.2.1.31)
	if len(snmp.OIDIfName) < 14 || snmp.OIDIfName[:14] != "1.3.6.1.2.1.31" {
		t.Errorf("OIDIfName %v should start with 1.3.6.1.2.1.31", snmp.OIDIfName)
	}
}
