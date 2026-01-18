package snmp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
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

// GetMACTable retrieves the MAC address table from a device.
// It tries Q-BRIDGE-MIB first (VLAN-aware), then falls back to BRIDGE-MIB.
func GetMACTable(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]MACEntry, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try Q-BRIDGE-MIB first (VLAN-aware).
	entries, err := getMACTableQBridge(ctx, ip, cfg)
	if err == nil && len(entries) > 0 {
		return entries, nil
	}

	// Fall back to BRIDGE-MIB (non-VLAN-aware).
	return getMACTableBridge(ctx, ip, cfg)
}

// getMACTableQBridge retrieves MAC table using Q-BRIDGE-MIB (VLAN-aware).
// Security: SNMPv3 is preferred over v2c when both are configured.
func getMACTableQBridge(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) ([]MACEntry, error) {
	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		entries, err := walkMACTableQBridgeV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return entries, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		entries, err := walkMACTableQBridge(ctx, ip, community, cfg)
		if err == nil {
			return entries, nil
		}
	}

	return nil, errors.New("failed to query Q-BRIDGE MAC table with all configured credentials")
}

// getMACTableBridge retrieves MAC table using BRIDGE-MIB (non-VLAN-aware).
// Security: SNMPv3 is preferred over v2c when both are configured.
func getMACTableBridge(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]MACEntry, error) {
	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		entries, err := walkMACTableBridgeV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return entries, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		entries, err := walkMACTableBridge(ctx, ip, community, cfg)
		if err == nil {
			return entries, nil
		}
	}

	return nil, errors.New("failed to query BRIDGE MAC table with all configured credentials")
}

// walkMACTableQBridge walks Q-BRIDGE-MIB MAC table using SNMPv2c.
func walkMACTableQBridge(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) ([]MACEntry, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkQBridgeMACTable(params)
}

// walkMACTableQBridgeV3 walks Q-BRIDGE-MIB MAC table using SNMPv3.
func walkMACTableQBridgeV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]MACEntry, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkQBridgeMACTable(params)
}

// walkQBridgeMACTable walks the Q-BRIDGE MAC table.
func walkQBridgeMACTable(params *gosnmp.GoSNMP) ([]MACEntry, error) {
	macToEntry := make(map[string]*MACEntry)
	bridgePortMap := getBridgePortMapping(params)

	// Walk dot1qTpFdbPort (MAC to bridge port mapping).
	err := params.BulkWalk(OIDDot1qTpFdbPort, func(pdu gosnmp.SnmpPDU) error {
		return processQBridgeFdbPortPDU(pdu, bridgePortMap, macToEntry)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk Q-BRIDGE MAC table: %w", err)
	}

	// Walk dot1qTpFdbStatus to get MAC entry type.
	walkErr := params.BulkWalk(OIDDot1qTpFdbStatus, func(pdu gosnmp.SnmpPDU) error {
		return processQBridgeFdbStatusPDU(pdu, macToEntry)
	})
	if walkErr != nil {
		logging.GetLogger().Warn("failed to walk MAC status", "error", walkErr)
	}

	return collectMACEntries(macToEntry), nil
}

// processQBridgeFdbPortPDU processes a single PDU from dot1qTpFdbPort walk.
// OID format: .1.3.6.1.2.1.17.7.1.2.2.1.2.VLAN.MAC1.MAC2.MAC3.MAC4.MAC5.MAC6.
func processQBridgeFdbPortPDU(
	pdu gosnmp.SnmpPDU,
	bridgePortMap map[int]int,
	macToEntry map[string]*MACEntry,
) error {
	parts := strings.Split(pdu.Name, ".")
	if len(parts) < minOIDPartsQBridge {
		logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
		return nil
	}

	vlan, mac, ok := parseVLANAndMAC(parts)
	if !ok {
		return nil
	}

	bridgePortNum, ok := parseBridgePort(pdu)
	if !ok {
		return nil
	}

	// Map bridge port to ifIndex (fallback to bridgePortNum if mapping not available).
	ifIndex := bridgePortMap[bridgePortNum]
	if ifIndex == 0 {
		ifIndex = bridgePortNum
	}

	macToEntry[mac] = &MACEntry{
		MAC:     mac,
		VLAN:    vlan,
		IfIndex: ifIndex,
	}
	return nil
}

// processQBridgeFdbStatusPDU processes a single PDU from dot1qTpFdbStatus walk.
func processQBridgeFdbStatusPDU(pdu gosnmp.SnmpPDU, macToEntry map[string]*MACEntry) error {
	parts := strings.Split(pdu.Name, ".")
	if len(parts) < minOIDPartsQBridge {
		logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
		return nil
	}

	_, mac, ok := parseVLANAndMAC(parts)
	if !ok {
		return nil
	}

	entry, exists := macToEntry[mac]
	if !exists {
		return nil
	}

	status := formatSNMPValue(pdu)
	entry.Type = parseMACStatus(status)
	return nil
}

// parseVLANAndMAC extracts VLAN and MAC address from OID parts.
// Returns (vlan, mac, ok).
func parseVLANAndMAC(parts []string) (int, string, bool) {
	vlanIdx := len(parts) - vlanOIDOffset
	vlan, err := strconv.Atoi(parts[vlanIdx])
	if err != nil {
		logging.GetLogger().Warn("failed to parse VLAN", "error", err)
		return 0, "", false
	}

	mac := parseMACFromOID(parts[vlanIdx+1:])
	if mac == "" {
		return 0, "", false
	}

	return vlan, mac, true
}

// parseBridgePort extracts the bridge port number from an SNMP PDU.
// Returns (bridgePort, ok) where ok is false if parsing fails.
func parseBridgePort(pdu gosnmp.SnmpPDU) (int, bool) {
	bridgePort := formatSNMPValue(pdu)
	bridgePortNum, err := strconv.Atoi(bridgePort)
	if err != nil {
		logging.GetLogger().Warn("failed to parse bridge port", "error", err)
		return 0, false
	}
	return bridgePortNum, true
}

// collectMACEntries converts a MAC entry map to a slice, applying default type if needed.
func collectMACEntries(macToEntry map[string]*MACEntry) []MACEntry {
	entries := make([]MACEntry, 0, len(macToEntry))
	for _, entry := range macToEntry {
		if entry.Type == "" {
			entry.Type = MACTypeLearned
		}
		entries = append(entries, *entry)
	}
	return entries
}

// walkMACTableBridge walks BRIDGE-MIB MAC table using SNMPv2c.
func walkMACTableBridge(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) ([]MACEntry, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkBridgeMACTable(params)
}

// walkMACTableBridgeV3 walks BRIDGE-MIB MAC table using SNMPv3.
func walkMACTableBridgeV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]MACEntry, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkBridgeMACTable(params)
}

// walkBridgeMACTable walks the BRIDGE-MIB MAC table.
func walkBridgeMACTable(params *gosnmp.GoSNMP) ([]MACEntry, error) {
	macToEntry := make(map[string]*MACEntry)
	bridgePortMap := getBridgePortMapping(params)

	// Walk dot1dTpFdbPort (MAC to bridge port mapping).
	err := params.BulkWalk(OIDDot1dTpFdbPort, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.17.4.3.1.2.MAC1.MAC2.MAC3.MAC4.MAC5.MAC6
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < minOIDPartsBridge {
			logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
			return nil
		}

		mac := parseMACFromOID(parts[len(parts)-macOctetCount:])
		if mac == "" {
			return nil
		}

		bridgePort := formatSNMPValue(pdu)
		bridgePortNum, err := strconv.Atoi(bridgePort)
		if err != nil {
			logging.GetLogger().Warn("failed to parse bridge port", "error", err)
			return nil
		}

		// Map bridge port to ifIndex.
		ifIndex := bridgePortMap[bridgePortNum]
		if ifIndex == 0 {
			ifIndex = bridgePortNum // Fallback if mapping not available
		}

		entry := &MACEntry{
			MAC:     mac,
			VLAN:    0, // BRIDGE-MIB doesn't include VLAN info
			IfIndex: ifIndex,
		}

		macToEntry[mac] = entry
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk BRIDGE MAC table: %w", err)
	}

	// Walk dot1dTpFdbStatus to get MAC entry type.
	walkErr := params.BulkWalk(OIDDot1dTpFdbStatus, func(pdu gosnmp.SnmpPDU) error {
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < minOIDPartsBridge {
			logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
			return nil
		}

		mac := parseMACFromOID(parts[len(parts)-macOctetCount:])
		if mac == "" {
			return nil
		}

		entry, exists := macToEntry[mac]
		if !exists {
			return nil
		}

		status := formatSNMPValue(pdu)
		entry.Type = parseMACStatus(status)
		return nil
	})
	if walkErr != nil {
		logging.GetLogger().Warn("failed to walk MAC status", "error", walkErr)
	}

	// Convert map to slice.
	entries := make([]MACEntry, 0, len(macToEntry))
	for _, entry := range macToEntry {
		if entry.Type == "" {
			entry.Type = MACTypeLearned
		}
		entries = append(entries, *entry)
	}

	return entries, nil
}

// GetPortVLANs retrieves VLAN membership for a specific port.
// Returns a list of VLAN IDs that the port is a member of.
// Security: SNMPv3 is preferred over v2c when both are configured.
func GetPortVLANs(
	ctx context.Context,
	ip string,
	ifIndex int,
	cfg *config.SNMPConfig,
) ([]int, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		vlans, err := getPortVLANsWithV3(ctx, ip, ifIndex, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return vlans, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		vlans, err := getPortVLANsWithCommunity(ctx, ip, ifIndex, community, cfg)
		if err == nil {
			return vlans, nil
		}
	}

	return nil, errors.New("failed to query port VLANs with all configured credentials")
}

// getPortVLANsWithCommunity retrieves port VLANs using SNMPv2c.
func getPortVLANsWithCommunity(
	ctx context.Context,
	ip string,
	ifIndex int,
	community string,
	cfg *config.SNMPConfig,
) ([]int, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkPortVLANs(params, ifIndex)
}

// getPortVLANsWithV3 retrieves port VLANs using SNMPv3.
func getPortVLANsWithV3(
	ctx context.Context,
	ip string,
	ifIndex int,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]int, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkPortVLANs(params, ifIndex)
}

// walkPortVLANs walks the port VLAN table.
func walkPortVLANs(params *gosnmp.GoSNMP, ifIndex int) ([]int, error) {
	vlans := make([]int, 0)

	// Walk dot1qVlanCurrentEgressPorts to find VLANs for this port.
	err := params.BulkWalk(OIDDot1qVlanCurrentEgressPorts, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.17.7.1.4.2.1.4.VLAN_ID
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < minOIDPartsForIndex {
			logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
			return nil
		}

		vlanID, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			logging.GetLogger().Warn("failed to parse VLAN ID", "error", err)
			return nil
		}

		// The value is a port list bitmap.
		portList := pdu.Value
		if portListContainsPort(portList, ifIndex) {
			vlans = append(vlans, vlanID)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk port VLANs: %w", err)
	}

	return vlans, nil
}

// getBridgePortMapping retrieves the mapping from bridge port number to ifIndex.
func getBridgePortMapping(params *gosnmp.GoSNMP) map[int]int {
	mapping := make(map[int]int)

	err := params.BulkWalk(OIDDot1dBasePortIfIndex, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.17.1.4.1.2.BRIDGE_PORT
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < minOIDPartsForIndex {
			logging.GetLogger().Warn("invalid OID format", "oid", pdu.Name)
			return nil
		}

		bridgePort, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			logging.GetLogger().Warn("failed to parse bridge port", "error", err)
			return nil
		}

		ifIndex, err := strconv.Atoi(formatSNMPValue(pdu))
		if err != nil {
			logging.GetLogger().Warn("failed to parse ifIndex", "error", err)
			return nil
		}

		mapping[bridgePort] = ifIndex
		return nil
	})
	if err != nil {
		logging.GetLogger().Warn("failed to get bridge port mapping", "error", err)
	}

	return mapping
}

// parseInterfaceStatus converts SNMP interface status value to string.
func parseInterfaceStatus(value string) string {
	switch value {
	case "1":
		return StatusUp
	case "2":
		return StatusDown
	case "3":
		return StatusTesting
	default:
		return StatusUnknown
	}
}

// parseDuplexStatus converts SNMP duplex status value to string.
func parseDuplexStatus(value string) string {
	switch value {
	case "1":
		return StatusUnknown
	case "2":
		return "half"
	case "3":
		return "full"
	default:
		return StatusUnknown
	}
}

// parseMACStatus converts SNMP MAC status value to entry type.
func parseMACStatus(value string) string {
	switch value {
	case "1":
		return MACTypeOther
	case "2":
		return MACTypeLearned // invalid
	case "3":
		return MACTypeLearned
	case "4":
		return MACTypeStatic
	case "5":
		return MACTypeStatic // mgmt
	default:
		return MACTypeOther
	}
}

// parseMACFromOID extracts MAC address from OID suffix.
// Expects 6 octets in the last parts of the OID.
func parseMACFromOID(parts []string) string {
	if len(parts) < macOctetCount {
		return ""
	}

	mac := make([]string, macOctetCount)
	for i := range macOctetCount {
		octet, err := strconv.Atoi(parts[i])
		if err != nil || octet < 0 || octet > 255 {
			return ""
		}
		mac[i] = fmt.Sprintf("%02x", octet) // #nosec G602 -- i is bounded by range [0,5]
	}

	return strings.Join(mac, ":")
}

// parseTimeTicks converts SNMP TimeTicks to a timestamp.
// TimeTicks are in hundredths of a second since system startup.
func parseTimeTicks(value string) time.Time {
	ticks, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}
	}

	// Convert hundredths of a second to duration.
	duration := time.Duration(ticks) * timeTicksToMilliseconds * time.Millisecond

	// Return time relative to now (approximation).
	return time.Now().Add(-duration)
}

// portListContainsPort checks if a port list bitmap contains the specified port.
// Port lists are byte arrays where each bit represents a port.
func portListContainsPort(portList any, portNum int) bool {
	bytes, ok := portList.([]byte)
	if !ok {
		return false
	}

	// Port numbering starts at 1, bitmap starts at 0.
	portIdx := portNum - 1
	if portIdx < 0 {
		return false
	}

	byteIdx := portIdx / bitsPerByte
	bitIdx := highBitIndex - (portIdx % bitsPerByte) // #nosec G115 -- portIdx%8 is always 0-7

	if byteIdx >= len(bytes) {
		return false
	}

	return (bytes[byteIdx] & (1 << bitIdx)) != 0
}

// Interface counter OIDs for bandwidth monitoring (IF-MIB).
const (
	OIDIfInOctets    = "1.3.6.1.2.1.2.2.1.10"    // ifInOctets (32-bit)
	OIDIfOutOctets   = "1.3.6.1.2.1.2.2.1.16"    // ifOutOctets (32-bit)
	OIDIfInErrors    = "1.3.6.1.2.1.2.2.1.14"    // ifInErrors
	OIDIfOutErrors   = "1.3.6.1.2.1.2.2.1.20"    // ifOutErrors
	OIDIfInDiscards  = "1.3.6.1.2.1.2.2.1.13"    // ifInDiscards
	OIDIfOutDiscards = "1.3.6.1.2.1.2.2.1.19"    // ifOutDiscards
	OIDIfHCInOctets  = "1.3.6.1.2.1.31.1.1.1.6"  // ifHCInOctets (64-bit, ifXTable)
	OIDIfHCOutOctets = "1.3.6.1.2.1.31.1.1.1.10" // ifHCOutOctets (64-bit, ifXTable)
)

// InterfaceCounters contains interface traffic counters for bandwidth monitoring.
type InterfaceCounters struct {
	Index       int    // ifIndex
	InOctets    uint64 // Input bytes (64-bit preferred if available)
	OutOctets   uint64 // Output bytes (64-bit preferred if available)
	InErrors    uint64 // Input errors
	OutErrors   uint64 // Output errors
	InDiscards  uint64 // Input discards
	OutDiscards uint64 // Output discards
	Timestamp   int64  // Unix timestamp when counters were collected
}

// GetInterfaceCounters retrieves traffic counters for all interfaces.
// It prefers 64-bit counters (ifHCInOctets/ifHCOutOctets) when available.
func GetInterfaceCounters(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) (map[int]*InterfaceCounters, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first.
	for i := range cfg.V3Credentials {
		counters, err := walkCountersV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return counters, nil
		}
	}

	// Fall back to v2c community strings.
	for _, community := range cfg.Communities {
		counters, err := walkCounters(ctx, ip, community, cfg)
		if err == nil {
			return counters, nil
		}
	}

	return nil, errors.New("failed to query interface counters with all configured credentials")
}

// walkCounters walks interface counters using SNMPv2c.
func walkCounters(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) (map[int]*InterfaceCounters, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return collectCounters(params)
}

// walkCountersV3 walks interface counters using SNMPv3.
func walkCountersV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) (map[int]*InterfaceCounters, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return collectCounters(params)
}

// collectCounters collects interface counters via SNMP bulk walk.
func collectCounters(params *gosnmp.GoSNMP) (map[int]*InterfaceCounters, error) {
	counters := make(map[int]*InterfaceCounters)
	timestamp := time.Now().Unix()

	// Helper to get or create counter entry.
	getCounter := func(index int) *InterfaceCounters {
		if c, ok := counters[index]; ok {
			return c
		}
		c := &InterfaceCounters{Index: index, Timestamp: timestamp}
		counters[index] = c
		return c
	}

	// Walk 64-bit counters first (preferred).
	walkCounter64(
		params,
		OIDIfHCInOctets,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.InOctets = v },
	)
	walkCounter64(
		params,
		OIDIfHCOutOctets,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.OutOctets = v },
	)

	// Walk 32-bit counters as fallback for devices without 64-bit support.
	walkCounter32Fallback(
		params,
		OIDIfInOctets,
		getCounter,
		func(c *InterfaceCounters) *uint64 { return &c.InOctets },
	)
	walkCounter32Fallback(
		params,
		OIDIfOutOctets,
		getCounter,
		func(c *InterfaceCounters) *uint64 { return &c.OutOctets },
	)

	// Walk error and discard counters.
	walkCounter32(
		params,
		OIDIfInErrors,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.InErrors = v },
	)
	walkCounter32(
		params,
		OIDIfOutErrors,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.OutErrors = v },
	)
	walkCounter32(
		params,
		OIDIfInDiscards,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.InDiscards = v },
	)
	walkCounter32(
		params,
		OIDIfOutDiscards,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.OutDiscards = v },
	)

	if len(counters) == 0 {
		return nil, errors.New("no interface counters retrieved")
	}

	return counters, nil
}

// walkCounter64 walks a 64-bit counter OID and applies the value using the setter.
func walkCounter64(
	params *gosnmp.GoSNMP,
	oid string,
	getCounter func(int) *InterfaceCounters,
	setter func(*InterfaceCounters, uint64),
) {
	_ = params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		if index := extractIndexFromOID(pdu.Name); index > 0 {
			setter(getCounter(index), parseCounter64(pdu))
		}
		return nil
	})
}

// walkCounter32 walks a 32-bit counter OID and applies the value using the setter.
func walkCounter32(
	params *gosnmp.GoSNMP,
	oid string,
	getCounter func(int) *InterfaceCounters,
	setter func(*InterfaceCounters, uint64),
) {
	_ = params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		if index := extractIndexFromOID(pdu.Name); index > 0 {
			setter(getCounter(index), parseCounter32(pdu))
		}
		return nil
	})
}

// walkCounter32Fallback walks a 32-bit counter OID but only sets value if the field is currently 0.
func walkCounter32Fallback(
	params *gosnmp.GoSNMP,
	oid string,
	getCounter func(int) *InterfaceCounters,
	fieldPtr func(*InterfaceCounters) *uint64,
) {
	_ = params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		if index := extractIndexFromOID(pdu.Name); index > 0 {
			c := getCounter(index)
			if ptr := fieldPtr(c); *ptr == 0 {
				*ptr = parseCounter32(pdu)
			}
		}
		return nil
	})
}

// extractIndexFromOID extracts the interface index from an OID string.
func extractIndexFromOID(oid string) int {
	parts := strings.Split(oid, ".")
	if len(parts) < minOIDPartsForIndex {
		return 0
	}
	idx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return idx
}

// parseCounter32 parses a 32-bit counter value from an SNMP PDU.
func parseCounter32(pdu gosnmp.SnmpPDU) uint64 {
	switch v := pdu.Value.(type) {
	case uint:
		return uint64(v)
	case uint32:
		return uint64(v)
	case int:
		if v >= 0 {
			return uint64(v)
		}
	case int64:
		if v >= 0 {
			return uint64(v)
		}
	}
	return 0
}

// parseCounter64 parses a 64-bit counter value from an SNMP PDU.
func parseCounter64(pdu gosnmp.SnmpPDU) uint64 {
	switch v := pdu.Value.(type) {
	case uint64:
		return v
	case uint:
		return uint64(v)
	case uint32:
		return uint64(v)
	case int64:
		if v >= 0 {
			return uint64(v)
		}
	case int:
		if v >= 0 {
			return uint64(v)
		}
	}
	return 0
}
