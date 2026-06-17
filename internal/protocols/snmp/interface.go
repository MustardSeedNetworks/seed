package snmp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Interface table OIDs (IF-MIB).
const (
	OIDIfIndex       = "1.3.6.1.2.1.2.2.1.1"    // ifIndex
	OIDIfDescr       = "1.3.6.1.2.1.2.2.1.2"    // ifDescr
	OIDIfType        = "1.3.6.1.2.1.2.2.1.3"    // ifType
	OIDIfSpeed       = "1.3.6.1.2.1.2.2.1.5"    // ifSpeed (bps)
	OIDIfPhysAddress = "1.3.6.1.2.1.2.2.1.6"    // ifPhysAddress (MAC)
	OIDIfAdminStatus = "1.3.6.1.2.1.2.2.1.7"    // ifAdminStatus
	OIDIfOperStatus  = "1.3.6.1.2.1.2.2.1.8"    // ifOperStatus
	OIDIfLastChange  = "1.3.6.1.2.1.2.2.1.9"    // ifLastChange (TimeTicks)
	OIDIfName        = "1.3.6.1.2.1.31.1.1.1.1" // ifName (IF-MIB)

	// OIDDot3StatsDuplexStatus is the EtherLike-MIB OID for duplex status.
	OIDDot3StatsDuplexStatus = "1.3.6.1.2.1.10.7.2.1.19"

	// OIDDot1dTpFdbAddress is the BRIDGE-MIB OID for MAC address table entries.
	OIDDot1dTpFdbAddress = "1.3.6.1.2.1.17.4.3.1.1"
	// OIDDot1dTpFdbPort is the BRIDGE-MIB OID for MAC table port mapping.
	OIDDot1dTpFdbPort = "1.3.6.1.2.1.17.4.3.1.2"
	// OIDDot1dTpFdbStatus is the BRIDGE-MIB OID for MAC table entry status.
	OIDDot1dTpFdbStatus = "1.3.6.1.2.1.17.4.3.1.3"

	// OIDDot1qTpFdbPort is the Q-BRIDGE-MIB OID for VLAN-aware MAC address table port.
	OIDDot1qTpFdbPort = "1.3.6.1.2.1.17.7.1.2.2.1.2"
	// OIDDot1qTpFdbStatus is the Q-BRIDGE-MIB OID for VLAN-aware MAC address table status.
	OIDDot1qTpFdbStatus = "1.3.6.1.2.1.17.7.1.2.2.1.3"

	// OIDDot1qVlanCurrentEgressPorts is the Q-BRIDGE-MIB OID for VLAN egress ports.
	OIDDot1qVlanCurrentEgressPorts = "1.3.6.1.2.1.17.7.1.4.2.1.4"
	// OIDDot1qVlanCurrentUntaggedPorts is the Q-BRIDGE-MIB OID for VLAN untagged ports.
	OIDDot1qVlanCurrentUntaggedPorts = "1.3.6.1.2.1.17.7.1.4.2.1.5"

	// OIDDot1dBasePortIfIndex is the Bridge port to ifIndex mapping OID.
	OIDDot1dBasePortIfIndex = "1.3.6.1.2.1.17.1.4.1.2"
)

// Interface status values.
const (
	StatusUp      = "up"
	StatusDown    = "down"
	StatusTesting = "testing"
	StatusUnknown = "unknown"
)

// MAC entry type values.
const (
	MACTypeLearned = "learned"
	MACTypeStatic  = "static"
	MACTypeOther   = "other"
)

// ID subtype values for LLDP.
const (
	IDSubtypeLocal = "local"
)

// OID parsing constants for validating minimum required parts.
const (
	// minOIDPartsForIndex is the minimum OID parts needed to extract an index (e.g., ifIndex).
	minOIDPartsForIndex = 2
	// minOIDPartsQBridge is the minimum OID parts for Q-BRIDGE-MIB MAC table entries
	// (OID base + VLAN + 6 MAC octets = 8 parts minimum).
	minOIDPartsQBridge = 8
	// minOIDPartsBridge is the minimum OID parts for BRIDGE-MIB MAC table entries
	// (OID base + 6 MAC octets = 7 parts minimum).
	minOIDPartsBridge = 7
	// vlanOIDOffset is the offset from end of OID parts to locate the VLAN ID
	// in Q-BRIDGE-MIB (VLAN + 6 MAC octets = 7 positions back).
	vlanOIDOffset = 7
)

// MAC address parsing constants.
const (
	// macOctetCount is the number of octets in a MAC address.
	macOctetCount = 6
)

// Time conversion constants.
const (
	// timeTicksToMilliseconds converts SNMP TimeTicks (hundredths of a second) to milliseconds.
	timeTicksToMilliseconds = 10
)

// Port bitmap parsing constants.
const (
	// bitsPerByte is the number of bits in a byte for port bitmap calculations.
	bitsPerByte = 8
	// highBitIndex is the index of the highest bit in a byte (0-7).
	highBitIndex = 7
)

// InterfaceInfo contains network interface details from IF-MIB.
type InterfaceInfo struct {
	Index       int       // ifIndex
	Description string    // ifDescr (e.g., "GigabitEthernet0/1")
	Name        string    // ifName (more concise name)
	Speed       int64     // ifSpeed in bps
	Duplex      string    // "full", "half", "auto", "unknown"
	AdminStatus string    // "up", "down", "testing"
	OperStatus  string    // "up", "down", "testing", etc.
	LastChange  time.Time // ifLastChange converted to timestamp
	MACAddress  string    // ifPhysAddress (MAC address)
}

// MACEntry contains a MAC address table entry from BRIDGE-MIB or Q-BRIDGE-MIB.
type MACEntry struct {
	MAC     string // MAC address (formatted as xx:xx:xx:xx:xx:xx)
	VLAN    int    // VLAN ID (0 if not available)
	IfIndex int    // Interface index
	Type    string // "learned", "static", "other"
}

// GetInterfaceInfo retrieves detailed information for a specific interface by ifIndex.
func GetInterfaceInfo(
	ctx context.Context,
	ip string,
	ifIndex int,
	cfg *config.SNMPConfig,
) (*InterfaceInfo, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Build OIDs with the ifIndex appended.
	oids := []string{
		fmt.Sprintf("%s.%d", OIDIfDescr, ifIndex),
		fmt.Sprintf("%s.%d", OIDIfName, ifIndex),
		fmt.Sprintf("%s.%d", OIDIfSpeed, ifIndex),
		fmt.Sprintf("%s.%d", OIDIfAdminStatus, ifIndex),
		fmt.Sprintf("%s.%d", OIDIfOperStatus, ifIndex),
		fmt.Sprintf("%s.%d", OIDIfLastChange, ifIndex),
		fmt.Sprintf("%s.%d", OIDIfPhysAddress, ifIndex),
	}

	results, err := QueryMultiple(ctx, ip, oids, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to query interface %d: %w", ifIndex, err)
	}

	info := &InterfaceInfo{
		Index: ifIndex,
	}

	// Parse results.
	for oid, value := range results {
		switch {
		case strings.HasPrefix(oid, OIDIfDescr):
			info.Description = value
		case strings.HasPrefix(oid, OIDIfName):
			info.Name = value
		case strings.HasPrefix(oid, OIDIfSpeed):
			if speed, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
				info.Speed = speed
			}
		case strings.HasPrefix(oid, OIDIfAdminStatus):
			info.AdminStatus = parseInterfaceStatus(value)
		case strings.HasPrefix(oid, OIDIfOperStatus):
			info.OperStatus = parseInterfaceStatus(value)
		case strings.HasPrefix(oid, OIDIfLastChange):
			info.LastChange = parseTimeTicks(value)
		case strings.HasPrefix(oid, OIDIfPhysAddress):
			info.MACAddress = value
		}
	}

	// Try to get duplex status (may not be available on all devices).
	duplexOID := fmt.Sprintf("%s.%d", OIDDot3StatsDuplexStatus, ifIndex)
	duplex, err := Query(ctx, ip, duplexOID, cfg)
	if err == nil {
		info.Duplex = parseDuplexStatus(duplex)
	} else {
		info.Duplex = StatusUnknown
	}

	return info, nil
}

// GetAllInterfaces retrieves information for all interfaces on a device.
// It performs a bulk walk of the interface table for efficiency.
// Security: SNMPv3 is preferred over v2c when both are configured.
func GetAllInterfaces(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) ([]InterfaceInfo, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		interfaces, err := walkInterfacesV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return interfaces, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		interfaces, err := walkInterfaces(ctx, ip, community, cfg)
		if err == nil {
			return interfaces, nil
		}
	}

	return nil, errors.New("failed to query interfaces with all configured credentials")
}

// walkInterfaces performs a bulk walk of interface table using SNMPv2c.
func walkInterfaces(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) ([]InterfaceInfo, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkInterfaceTable(params)
}

// walkInterfacesV3 performs a bulk walk of interface table using SNMPv3.
func walkInterfacesV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]InterfaceInfo, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkInterfaceTable(params)
}

// walkInterfaceTable walks the interface table for the given SNMP connection.
func walkInterfaceTable(params *gosnmp.GoSNMP) ([]InterfaceInfo, error) {
	interfaces := make(map[int]*InterfaceInfo)

	// Walk ifIndex to discover all interfaces.
	err := params.BulkWalk(OIDIfIndex, func(pdu gosnmp.SnmpPDU) error {
		// Extract ifIndex from OID.
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < minOIDPartsForIndex {
			logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
			return nil
		}
		ifIndex, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			logging.GetLogger().Warn("failed to parse ifIndex", "error", err)
			return nil
		}

		interfaces[ifIndex] = &InterfaceInfo{Index: ifIndex}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk ifIndex: %w", err)
	}

	// Walk other interface attributes.
	walkIfAttribute(params, OIDIfDescr, interfaces, func(info *InterfaceInfo, value string) {
		info.Description = value
	})
	walkIfAttribute(params, OIDIfName, interfaces, func(info *InterfaceInfo, value string) {
		info.Name = value
	})
	walkIfAttribute(params, OIDIfSpeed, interfaces, func(info *InterfaceInfo, value string) {
		if speed, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
			info.Speed = speed
		}
	})
	walkIfAttribute(params, OIDIfAdminStatus, interfaces, func(info *InterfaceInfo, value string) {
		info.AdminStatus = parseInterfaceStatus(value)
	})
	walkIfAttribute(params, OIDIfOperStatus, interfaces, func(info *InterfaceInfo, value string) {
		info.OperStatus = parseInterfaceStatus(value)
	})
	walkIfAttribute(params, OIDIfLastChange, interfaces, func(info *InterfaceInfo, value string) {
		info.LastChange = parseTimeTicks(value)
	})
	walkIfAttribute(params, OIDIfPhysAddress, interfaces, func(info *InterfaceInfo, value string) {
		info.MACAddress = value
	})

	// Try to get duplex status for each interface.
	walkIfAttribute(
		params,
		OIDDot3StatsDuplexStatus,
		interfaces,
		func(info *InterfaceInfo, value string) {
			info.Duplex = parseDuplexStatus(value)
		},
	)

	// Convert map to slice.
	result := make([]InterfaceInfo, 0, len(interfaces))
	for _, info := range interfaces {
		if info.Duplex == "" {
			info.Duplex = StatusUnknown
		}
		result = append(result, *info)
	}

	return result, nil
}

// walkIfAttribute walks an SNMP OID and applies a function to update interface info.
func walkIfAttribute(
	params *gosnmp.GoSNMP,
	oid string,
	interfaces map[int]*InterfaceInfo,
	updateFunc func(*InterfaceInfo, string),
) {
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		// Extract ifIndex from OID.
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < minOIDPartsForIndex {
			logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
			return nil
		}
		ifIndex, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			logging.GetLogger().Warn("failed to parse ifIndex", "error", err)
			return nil
		}

		info, exists := interfaces[ifIndex]
		if !exists {
			return nil
		}

		value := formatSNMPValue(pdu)
		updateFunc(info, value)
		return nil
	})
	if err != nil {
		logging.GetLogger().Warn("failed to walk OID", "oid", oid, "error", err)
	}
}
