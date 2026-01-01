// Package snmp provides SNMP query functionality for network device discovery.
// This file implements ENTITY-MIB (RFC 6933) collection for physical inventory.
package snmp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
)

// ENTITY-MIB OIDs (RFC 6933).
const (
	OIDEntPhysicalDescr        = "1.3.6.1.2.1.47.1.1.1.1.2"  // entPhysicalDescr
	OIDEntPhysicalVendorType   = "1.3.6.1.2.1.47.1.1.1.1.3"  // entPhysicalVendorType
	OIDEntPhysicalContainedIn  = "1.3.6.1.2.1.47.1.1.1.1.4"  // entPhysicalContainedIn
	OIDEntPhysicalClass        = "1.3.6.1.2.1.47.1.1.1.1.5"  // entPhysicalClass
	OIDEntPhysicalParentRelPos = "1.3.6.1.2.1.47.1.1.1.1.6"  // entPhysicalParentRelPos
	OIDEntPhysicalName         = "1.3.6.1.2.1.47.1.1.1.1.7"  // entPhysicalName
	OIDEntPhysicalHardwareRev  = "1.3.6.1.2.1.47.1.1.1.1.8"  // entPhysicalHardwareRev
	OIDEntPhysicalFirmwareRev  = "1.3.6.1.2.1.47.1.1.1.1.9"  // entPhysicalFirmwareRev
	OIDEntPhysicalSoftwareRev  = "1.3.6.1.2.1.47.1.1.1.1.10" // entPhysicalSoftwareRev
	OIDEntPhysicalSerialNum    = "1.3.6.1.2.1.47.1.1.1.1.11" // entPhysicalSerialNum
	OIDEntPhysicalMfgName      = "1.3.6.1.2.1.47.1.1.1.1.12" // entPhysicalMfgName
	OIDEntPhysicalModelName    = "1.3.6.1.2.1.47.1.1.1.1.13" // entPhysicalModelName
	OIDEntPhysicalIsFRU        = "1.3.6.1.2.1.47.1.1.1.1.16" // entPhysicalIsFRU
)

// PhysicalEntity represents a physical entity from ENTITY-MIB.
type PhysicalEntity struct {
	Index        int    // entPhysicalIndex
	Description  string // entPhysicalDescr
	VendorType   string // entPhysicalVendorType (OID)
	ContainedIn  int    // entPhysicalContainedIn (parent index)
	Class        string // chassis, module, port, powerSupply, etc.
	ParentRelPos int    // entPhysicalParentRelPos
	Name         string // entPhysicalName
	HardwareRev  string // entPhysicalHardwareRev
	FirmwareRev  string // entPhysicalFirmwareRev
	SoftwareRev  string // entPhysicalSoftwareRev
	SerialNum    string // entPhysicalSerialNum
	MfgName      string // entPhysicalMfgName
	ModelName    string // entPhysicalModelName
	IsFRU        bool   // entPhysicalIsFRU (Field Replaceable Unit)
}

// GetPhysicalEntities retrieves all physical entities from ENTITY-MIB.
// Security: SNMPv3 is preferred over v2c when both are configured.
func GetPhysicalEntities(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]PhysicalEntity, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		entities, err := walkEntityTableV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return entities, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		entities, err := walkEntityTable(ctx, ip, community, cfg)
		if err == nil {
			return entities, nil
		}
	}

	return nil, errors.New("failed to query ENTITY-MIB with all configured credentials")
}

// walkEntityTable walks the entPhysicalTable using SNMPv2c.
func walkEntityTable(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]PhysicalEntity, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community:      community,
		Version:        gosnmp.Version2c,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
	}

	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkPhysicalEntities(params)
}

// walkEntityTableV3 walks the entPhysicalTable using SNMPv3.
func walkEntityTableV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]PhysicalEntity, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Version:        gosnmp.Version3,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
		SecurityModel:  gosnmp.UserSecurityModel,
		MsgFlags:       gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 cred.Username,
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol),
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkPhysicalEntities(params)
}

// walkPhysicalEntities walks the entPhysicalTable.
func walkPhysicalEntities(params *gosnmp.GoSNMP) ([]PhysicalEntity, error) {
	entities := make(map[int]*PhysicalEntity)

	// Walk entPhysicalDescr to discover all physical entities.
	err := params.BulkWalk(OIDEntPhysicalDescr, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.47.1.1.1.1.2.INDEX
		idx := extractEntityIndex(pdu.Name)
		if idx <= 0 {
			return nil
		}

		entities[idx] = &PhysicalEntity{
			Index:       idx,
			Description: formatSNMPValue(pdu),
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk entPhysicalDescr: %w", err)
	}

	// Walk other attributes.
	walkEntityAttribute(params, OIDEntPhysicalVendorType, entities, func(e *PhysicalEntity, v string) {
		e.VendorType = v
	})
	walkEntityAttribute(params, OIDEntPhysicalContainedIn, entities, func(e *PhysicalEntity, v string) {
		if idx, parseErr := strconv.Atoi(v); parseErr == nil {
			e.ContainedIn = idx
		}
	})
	walkEntityAttribute(params, OIDEntPhysicalClass, entities, func(e *PhysicalEntity, v string) {
		e.Class = parseEntityClass(v)
	})
	walkEntityAttribute(params, OIDEntPhysicalParentRelPos, entities, func(e *PhysicalEntity, v string) {
		if pos, parseErr := strconv.Atoi(v); parseErr == nil {
			e.ParentRelPos = pos
		}
	})
	walkEntityAttribute(params, OIDEntPhysicalName, entities, func(e *PhysicalEntity, v string) {
		e.Name = v
	})
	walkEntityAttribute(params, OIDEntPhysicalHardwareRev, entities, func(e *PhysicalEntity, v string) {
		e.HardwareRev = v
	})
	walkEntityAttribute(params, OIDEntPhysicalFirmwareRev, entities, func(e *PhysicalEntity, v string) {
		e.FirmwareRev = v
	})
	walkEntityAttribute(params, OIDEntPhysicalSoftwareRev, entities, func(e *PhysicalEntity, v string) {
		e.SoftwareRev = v
	})
	walkEntityAttribute(params, OIDEntPhysicalSerialNum, entities, func(e *PhysicalEntity, v string) {
		e.SerialNum = v
	})
	walkEntityAttribute(params, OIDEntPhysicalMfgName, entities, func(e *PhysicalEntity, v string) {
		e.MfgName = v
	})
	walkEntityAttribute(params, OIDEntPhysicalModelName, entities, func(e *PhysicalEntity, v string) {
		e.ModelName = v
	})
	walkEntityAttribute(params, OIDEntPhysicalIsFRU, entities, func(e *PhysicalEntity, v string) {
		e.IsFRU = (v == "1" || v == "true")
	})

	// Convert map to slice, sorted by index.
	result := make([]PhysicalEntity, 0, len(entities))
	for _, entity := range entities {
		result = append(result, *entity)
	}

	return result, nil
}

// walkEntityAttribute walks an entity attribute and applies a function.
func walkEntityAttribute(
	params *gosnmp.GoSNMP,
	oid string,
	entities map[int]*PhysicalEntity,
	updateFunc func(*PhysicalEntity, string),
) {
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		idx := extractEntityIndex(pdu.Name)
		if idx <= 0 {
			return nil
		}

		entity, exists := entities[idx]
		if !exists {
			return nil
		}

		updateFunc(entity, formatSNMPValue(pdu))
		return nil
	})
	if err != nil {
		slog.Debug("Failed to walk entity attribute", "oid", oid, "error", err)
	}
}

// extractEntityIndex extracts entity index from OID.
func extractEntityIndex(oid string) int {
	parts := strings.Split(oid, ".")
	if len(parts) < 2 {
		return 0
	}

	idx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return idx
}

// parseEntityClass converts entPhysicalClass value to string.
func parseEntityClass(value string) string {
	switch value {
	case "1":
		return MACTypeOther
	case "2":
		return StatusUnknown
	case "3":
		return "chassis"
	case "4":
		return "backplane"
	case "5":
		return "container"
	case "6":
		return "powerSupply"
	case "7":
		return "fan"
	case "8":
		return "sensor"
	case "9":
		return "module"
	case "10":
		return "port"
	case "11":
		return "stack"
	case "12":
		return "cpu"
	case "13":
		return "energyObject"
	case "14":
		return "battery"
	case "15":
		return "storageDrive"
	default:
		return StatusUnknown
	}
}

// GetChassisInfo retrieves the main chassis entity (class=chassis).
func GetChassisInfo(ctx context.Context, ip string, cfg *config.SNMPConfig) (*PhysicalEntity, error) {
	entities, err := GetPhysicalEntities(ctx, ip, cfg)
	if err != nil {
		return nil, err
	}

	// Find the chassis entity (first one with class=chassis).
	for i := range entities {
		if entities[i].Class == "chassis" {
			return &entities[i], nil
		}
	}

	// Fall back to first entity if no chassis found.
	if len(entities) > 0 {
		return &entities[0], nil
	}

	return nil, errors.New("no physical entities found")
}

// GetModules retrieves all module entities (class=module).
func GetModules(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]PhysicalEntity, error) {
	entities, err := GetPhysicalEntities(ctx, ip, cfg)
	if err != nil {
		return nil, err
	}

	modules := make([]PhysicalEntity, 0)
	for _, entity := range entities {
		if entity.Class == "module" {
			modules = append(modules, entity)
		}
	}

	return modules, nil
}

// GetPowerSupplies retrieves all power supply entities.
func GetPowerSupplies(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]PhysicalEntity, error) {
	entities, err := GetPhysicalEntities(ctx, ip, cfg)
	if err != nil {
		return nil, err
	}

	psus := make([]PhysicalEntity, 0)
	for _, entity := range entities {
		if entity.Class == "powerSupply" {
			psus = append(psus, entity)
		}
	}

	return psus, nil
}

// GetFans retrieves all fan entities.
func GetFans(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]PhysicalEntity, error) {
	entities, err := GetPhysicalEntities(ctx, ip, cfg)
	if err != nil {
		return nil, err
	}

	fans := make([]PhysicalEntity, 0)
	for _, entity := range entities {
		if entity.Class == "fan" {
			fans = append(fans, entity)
		}
	}

	return fans, nil
}
