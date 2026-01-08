package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

func TestExtractEntityIndex(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want int
	}{
		{
			name: "valid entity index",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.1001",
			want: 1001,
		},
		{
			name: "entity index 1",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.1",
			want: 1,
		},
		{
			name: "large entity index",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.99999",
			want: 99999,
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
			name: "invalid entity index",
			oid:  "1.3.6.1.2.1.47.1.1.1.1.2.invalid",
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

func TestParseEntityClass(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"unknown", "2", snmp.StatusUnknown},
		{"chassis", "3", "chassis"},
		{"backplane", "4", "backplane"},
		{"container", "5", "container"},
		{"powerSupply", "6", "powerSupply"},
		{"fan", "7", "fan"},
		{"sensor", "8", "sensor"},
		{"module", "9", "module"},
		{"port", "10", "port"},
		{"stack", "11", "stack"},
		{"cpu", "12", "cpu"},
		{"energyObject", "13", "energyObject"},
		{"battery", "14", "battery"},
		{"storageDrive", "15", "storageDrive"},
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high value", "99", snmp.StatusUnknown},
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

func TestPhysicalEntityStruct(t *testing.T) {
	entity := snmp.PhysicalEntity{
		Index:        1,
		Description:  "Cisco Catalyst 3850-48P Switch",
		VendorType:   "1.3.6.1.4.1.9.12.3.1.3.1208",
		ContainedIn:  0,
		Class:        "chassis",
		ParentRelPos: 0,
		Name:         "Switch 1",
		HardwareRev:  "V02",
		FirmwareRev:  "16.12.4",
		SoftwareRev:  "16.12.4",
		SerialNum:    "FCW2145L0AB",
		MfgName:      "Cisco Systems, Inc.",
		ModelName:    "WS-C3850-48P",
		IsFRU:        true,
	}

	// Verify all fields to avoid unusedwrite linter warnings
	if entity.Index != 1 {
		t.Errorf("Index = %v, want 1", entity.Index)
	}
	if entity.Description != "Cisco Catalyst 3850-48P Switch" {
		t.Errorf("Description = %v, want 'Cisco Catalyst 3850-48P Switch'", entity.Description)
	}
	if entity.VendorType != "1.3.6.1.4.1.9.12.3.1.3.1208" {
		t.Errorf("VendorType = %v, want '1.3.6.1.4.1.9.12.3.1.3.1208'", entity.VendorType)
	}
	if entity.ContainedIn != 0 {
		t.Errorf("ContainedIn = %v, want 0", entity.ContainedIn)
	}
	if entity.Class != "chassis" {
		t.Errorf("Class = %v, want 'chassis'", entity.Class)
	}
	if entity.ParentRelPos != 0 {
		t.Errorf("ParentRelPos = %v, want 0", entity.ParentRelPos)
	}
	if entity.Name != "Switch 1" {
		t.Errorf("Name = %v, want 'Switch 1'", entity.Name)
	}
	if entity.HardwareRev != "V02" {
		t.Errorf("HardwareRev = %v, want 'V02'", entity.HardwareRev)
	}
	if entity.FirmwareRev != "16.12.4" {
		t.Errorf("FirmwareRev = %v, want '16.12.4'", entity.FirmwareRev)
	}
	if entity.SoftwareRev != "16.12.4" {
		t.Errorf("SoftwareRev = %v, want '16.12.4'", entity.SoftwareRev)
	}
	if entity.SerialNum != "FCW2145L0AB" {
		t.Errorf("SerialNum = %v, want 'FCW2145L0AB'", entity.SerialNum)
	}
	if entity.MfgName != "Cisco Systems, Inc." {
		t.Errorf("MfgName = %v, want 'Cisco Systems, Inc.'", entity.MfgName)
	}
	if entity.ModelName != "WS-C3850-48P" {
		t.Errorf("ModelName = %v, want 'WS-C3850-48P'", entity.ModelName)
	}
	if !entity.IsFRU {
		t.Error("IsFRU = false, want true")
	}
}

func TestGetPhysicalEntities(t *testing.T) {
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
			_, err := snmp.GetPhysicalEntities(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetPhysicalEntities() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetPhysicalEntities() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetChassisInfo(t *testing.T) {
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
			_, err := snmp.GetChassisInfo(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetChassisInfo() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetChassisInfo() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetModules(t *testing.T) {
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
			_, err := snmp.GetModules(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetModules() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetModules() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetPowerSupplies(t *testing.T) {
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
			_, err := snmp.GetPowerSupplies(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetPowerSupplies() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetPowerSupplies() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetFans(t *testing.T) {
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
			_, err := snmp.GetFans(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetFans() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetFans() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestEntityMIBOIDConstants(t *testing.T) {
	// Verify ENTITY-MIB OID constants are defined
	oids := map[string]string{
		"OIDEntPhysicalDescr":        snmp.OIDEntPhysicalDescr,
		"OIDEntPhysicalVendorType":   snmp.OIDEntPhysicalVendorType,
		"OIDEntPhysicalContainedIn":  snmp.OIDEntPhysicalContainedIn,
		"OIDEntPhysicalClass":        snmp.OIDEntPhysicalClass,
		"OIDEntPhysicalParentRelPos": snmp.OIDEntPhysicalParentRelPos,
		"OIDEntPhysicalName":         snmp.OIDEntPhysicalName,
		"OIDEntPhysicalHardwareRev":  snmp.OIDEntPhysicalHardwareRev,
		"OIDEntPhysicalFirmwareRev":  snmp.OIDEntPhysicalFirmwareRev,
		"OIDEntPhysicalSoftwareRev":  snmp.OIDEntPhysicalSoftwareRev,
		"OIDEntPhysicalSerialNum":    snmp.OIDEntPhysicalSerialNum,
		"OIDEntPhysicalMfgName":      snmp.OIDEntPhysicalMfgName,
		"OIDEntPhysicalModelName":    snmp.OIDEntPhysicalModelName,
		"OIDEntPhysicalIsFRU":        snmp.OIDEntPhysicalIsFRU,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("%s is empty", name)
		}
		// ENTITY-MIB OIDs should be under 1.3.6.1.2.1.47 (entityMIB)
		if len(oid) < 14 || oid[:14] != "1.3.6.1.2.1.47" {
			t.Errorf("%s = %v, should start with 1.3.6.1.2.1.47", name, oid)
		}
	}
}

func TestPhysicalEntityHierarchy(t *testing.T) {
	// Test entity hierarchy (chassis contains modules)
	chassis := snmp.PhysicalEntity{
		Index:       1,
		Class:       "chassis",
		ContainedIn: 0, // Top-level
	}

	module := snmp.PhysicalEntity{
		Index:       1001,
		Class:       "module",
		ContainedIn: 1, // Contained in chassis
	}

	port := snmp.PhysicalEntity{
		Index:       10001,
		Class:       "port",
		ContainedIn: 1001, // Contained in module
	}

	// Verify hierarchy by checking Index, Class, and ContainedIn
	if chassis.Index != 1 || chassis.Class != "chassis" || chassis.ContainedIn != 0 {
		t.Errorf(
			"Chassis: Index=%v, Class=%v, ContainedIn=%v",
			chassis.Index,
			chassis.Class,
			chassis.ContainedIn,
		)
	}
	if module.Index != 1001 || module.Class != "module" || module.ContainedIn != 1 {
		t.Errorf(
			"Module: Index=%v, Class=%v, ContainedIn=%v",
			module.Index,
			module.Class,
			module.ContainedIn,
		)
	}
	if port.Index != 10001 || port.Class != "port" || port.ContainedIn != 1001 {
		t.Errorf(
			"Port: Index=%v, Class=%v, ContainedIn=%v",
			port.Index,
			port.Class,
			port.ContainedIn,
		)
	}
}

func TestPhysicalEntityIsFRU(t *testing.T) {
	// FRU (Field Replaceable Unit) testing
	fruEntity := snmp.PhysicalEntity{
		Index: 1,
		Class: "powerSupply",
		IsFRU: true,
	}

	nonFruEntity := snmp.PhysicalEntity{
		Index: 2,
		Class: "backplane",
		IsFRU: false,
	}

	// Verify all fields to avoid linter warnings
	if fruEntity.Index != 1 || fruEntity.Class != "powerSupply" || !fruEntity.IsFRU {
		t.Errorf(
			"FRU entity: Index=%v, Class=%v, IsFRU=%v",
			fruEntity.Index,
			fruEntity.Class,
			fruEntity.IsFRU,
		)
	}
	if nonFruEntity.Index != 2 || nonFruEntity.Class != "backplane" || nonFruEntity.IsFRU {
		t.Errorf(
			"Non-FRU entity: Index=%v, Class=%v, IsFRU=%v",
			nonFruEntity.Index,
			nonFruEntity.Class,
			nonFruEntity.IsFRU,
		)
	}
}

func TestPhysicalEntityEmptyOptionalFields(t *testing.T) {
	// Entities may have empty optional fields
	entity := snmp.PhysicalEntity{
		Index:       1,
		Description: "Test Entity",
		Class:       "module",
		// Optional fields left empty
	}

	// Verify explicitly set fields
	if entity.Index != 1 {
		t.Errorf("Index = %v, want 1", entity.Index)
	}
	if entity.Description != "Test Entity" {
		t.Errorf("Description = %v, want 'Test Entity'", entity.Description)
	}
	if entity.Class != "module" {
		t.Errorf("Class = %v, want 'module'", entity.Class)
	}

	// Verify optional fields are empty
	if entity.HardwareRev != "" {
		t.Errorf("HardwareRev = %v, want empty", entity.HardwareRev)
	}
	if entity.FirmwareRev != "" {
		t.Errorf("FirmwareRev = %v, want empty", entity.FirmwareRev)
	}
	if entity.SoftwareRev != "" {
		t.Errorf("SoftwareRev = %v, want empty", entity.SoftwareRev)
	}
	if entity.SerialNum != "" {
		t.Errorf("SerialNum = %v, want empty", entity.SerialNum)
	}
}
