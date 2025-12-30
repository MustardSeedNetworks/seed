// Package snmp provides SNMP query functionality for network device discovery.
package snmp

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
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

	// EtherLike-MIB for duplex status.
	OIDDot3StatsDuplexStatus = "1.3.6.1.2.1.10.7.2.1.19" // dot3StatsDuplexStatus

	// BRIDGE-MIB for MAC address table.
	OIDDot1dTpFdbAddress = "1.3.6.1.2.1.17.4.3.1.1" // dot1dTpFdbAddress
	OIDDot1dTpFdbPort    = "1.3.6.1.2.1.17.4.3.1.2" // dot1dTpFdbPort
	OIDDot1dTpFdbStatus  = "1.3.6.1.2.1.17.4.3.1.3" // dot1dTpFdbStatus

	// Q-BRIDGE-MIB for VLAN-aware MAC address table.
	OIDDot1qTpFdbPort   = "1.3.6.1.2.1.17.7.1.2.2.1.2" // dot1qTpFdbPort
	OIDDot1qTpFdbStatus = "1.3.6.1.2.1.17.7.1.2.2.1.3" // dot1qTpFdbStatus

	// Q-BRIDGE-MIB for VLAN membership.
	OIDDot1qVlanCurrentEgressPorts   = "1.3.6.1.2.1.17.7.1.4.2.1.4" // dot1qVlanCurrentEgressPorts
	OIDDot1qVlanCurrentUntaggedPorts = "1.3.6.1.2.1.17.7.1.4.2.1.5" // dot1qVlanCurrentUntaggedPorts

	// Bridge port mapping (dot1dBasePortIfIndex).
	OIDDot1dBasePortIfIndex = "1.3.6.1.2.1.17.1.4.1.2" // Maps bridge port to ifIndex
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
func GetInterfaceInfo(ctx context.Context, ip string, ifIndex int, cfg *config.SNMPConfig) (*InterfaceInfo, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SNMP config is nil")
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
			if speed, err := strconv.ParseInt(value, 10, 64); err == nil {
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
func GetAllInterfaces(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]InterfaceInfo, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SNMP config is nil")
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

	return nil, fmt.Errorf("failed to query interfaces with all configured credentials")
}

// walkInterfaces performs a bulk walk of interface table using SNMPv2c.
func walkInterfaces(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]InterfaceInfo, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community:      community,
		Version:        gosnmp.Version2c,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkInterfaceTable(params)
}

// walkInterfacesV3 performs a bulk walk of interface table using SNMPv3.
func walkInterfacesV3(ctx context.Context, ip string, cred *config.SNMPv3Credential, cfg *config.SNMPConfig) ([]InterfaceInfo, error) {
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
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol), //nolint:staticcheck // Internal usage of deprecated field is expected
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkInterfaceTable(params)
}

// walkInterfaceTable walks the interface table for the given SNMP connection.
func walkInterfaceTable(params *gosnmp.GoSNMP) ([]InterfaceInfo, error) {
	interfaces := make(map[int]*InterfaceInfo)

	// Walk ifIndex to discover all interfaces.
	err := params.BulkWalk(OIDIfIndex, func(pdu gosnmp.SnmpPDU) error {
		// Extract ifIndex from OID.
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < 2 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}
		ifIndex, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			log.Printf("Failed to parse ifIndex: %v", err)
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
		if speed, err := strconv.ParseInt(value, 10, 64); err == nil {
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
	walkIfAttribute(params, OIDDot3StatsDuplexStatus, interfaces, func(info *InterfaceInfo, value string) {
		info.Duplex = parseDuplexStatus(value)
	})

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
func walkIfAttribute(params *gosnmp.GoSNMP, oid string, interfaces map[int]*InterfaceInfo, updateFunc func(*InterfaceInfo, string)) {
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		// Extract ifIndex from OID.
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < 2 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}
		ifIndex, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			log.Printf("Failed to parse ifIndex: %v", err)
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
		log.Printf("Failed to walk OID %s: %v", oid, err)
	}
}

// GetMACTable retrieves the MAC address table from a device.
// It tries Q-BRIDGE-MIB first (VLAN-aware), then falls back to BRIDGE-MIB.
func GetMACTable(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]MACEntry, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SNMP config is nil")
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
func getMACTableQBridge(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]MACEntry, error) {
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

	return nil, fmt.Errorf("failed to query Q-BRIDGE MAC table with all configured credentials")
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

	return nil, fmt.Errorf("failed to query BRIDGE MAC table with all configured credentials")
}

// walkMACTableQBridge walks Q-BRIDGE-MIB MAC table using SNMPv2c.
func walkMACTableQBridge(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]MACEntry, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community:      community,
		Version:        gosnmp.Version2c,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkQBridgeMACTable(params)
}

// walkMACTableQBridgeV3 walks Q-BRIDGE-MIB MAC table using SNMPv3.
func walkMACTableQBridgeV3(ctx context.Context, ip string, cred *config.SNMPv3Credential, cfg *config.SNMPConfig) ([]MACEntry, error) {
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
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol), //nolint:staticcheck // Internal usage of deprecated field is expected
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkQBridgeMACTable(params)
}

// walkQBridgeMACTable walks the Q-BRIDGE MAC table.
func walkQBridgeMACTable(params *gosnmp.GoSNMP) ([]MACEntry, error) {
	macToEntry := make(map[string]*MACEntry)
	bridgePortMap := getBridgePortMapping(params)

	// Walk dot1qTpFdbPort (MAC to bridge port mapping).
	err := params.BulkWalk(OIDDot1qTpFdbPort, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.17.7.1.2.2.1.2.VLAN.MAC1.MAC2.MAC3.MAC4.MAC5.MAC6
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < 8 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}

		// Extract VLAN and MAC address.
		vlanIdx := len(parts) - 7
		vlan, err := strconv.Atoi(parts[vlanIdx])
		if err != nil {
			log.Printf("Failed to parse VLAN: %v", err)
			return nil
		}

		mac := parseMACFromOID(parts[vlanIdx+1:])
		if mac == "" {
			return nil
		}

		bridgePort := formatSNMPValue(pdu)
		bridgePortNum, err := strconv.Atoi(bridgePort)
		if err != nil {
			log.Printf("Failed to parse bridge port: %v", err)
			return nil
		}

		// Map bridge port to ifIndex.
		ifIndex := bridgePortMap[bridgePortNum]
		if ifIndex == 0 {
			ifIndex = bridgePortNum // Fallback if mapping not available
		}

		entry := &MACEntry{
			MAC:     mac,
			VLAN:    vlan,
			IfIndex: ifIndex,
		}

		macToEntry[mac] = entry
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk Q-BRIDGE MAC table: %w", err)
	}

	// Walk dot1qTpFdbStatus to get MAC entry type.
	walkErr := params.BulkWalk(OIDDot1qTpFdbStatus, func(pdu gosnmp.SnmpPDU) error {
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < 8 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}

		vlanIdx := len(parts) - 7
		mac := parseMACFromOID(parts[vlanIdx+1:])
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
		log.Printf("Failed to walk MAC status: %v", walkErr)
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

// walkMACTableBridge walks BRIDGE-MIB MAC table using SNMPv2c.
func walkMACTableBridge(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]MACEntry, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community:      community,
		Version:        gosnmp.Version2c,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkBridgeMACTable(params)
}

// walkMACTableBridgeV3 walks BRIDGE-MIB MAC table using SNMPv3.
func walkMACTableBridgeV3(ctx context.Context, ip string, cred *config.SNMPv3Credential, cfg *config.SNMPConfig) ([]MACEntry, error) {
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
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol), //nolint:staticcheck // Internal usage of deprecated field is expected
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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
		if len(parts) < 7 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}

		mac := parseMACFromOID(parts[len(parts)-6:])
		if mac == "" {
			return nil
		}

		bridgePort := formatSNMPValue(pdu)
		bridgePortNum, err := strconv.Atoi(bridgePort)
		if err != nil {
			log.Printf("Failed to parse bridge port: %v", err)
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
		if len(parts) < 7 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}

		mac := parseMACFromOID(parts[len(parts)-6:])
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
		log.Printf("Failed to walk MAC status: %v", walkErr)
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
func GetPortVLANs(ctx context.Context, ip string, ifIndex int, cfg *config.SNMPConfig) ([]int, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SNMP config is nil")
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

	return nil, fmt.Errorf("failed to query port VLANs with all configured credentials")
}

// getPortVLANsWithCommunity retrieves port VLANs using SNMPv2c.
func getPortVLANsWithCommunity(ctx context.Context, ip string, ifIndex int, community string, cfg *config.SNMPConfig) ([]int, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community:      community,
		Version:        gosnmp.Version2c,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkPortVLANs(params, ifIndex)
}

// getPortVLANsWithV3 retrieves port VLANs using SNMPv3.
func getPortVLANsWithV3(ctx context.Context, ip string, ifIndex int, cred *config.SNMPv3Credential, cfg *config.SNMPConfig) ([]int, error) {
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
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol), //nolint:staticcheck // Internal usage of deprecated field is expected
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkPortVLANs(params, ifIndex)
}

// walkPortVLANs walks the port VLAN table.
func walkPortVLANs(params *gosnmp.GoSNMP, ifIndex int) ([]int, error) {
	vlans := make([]int, 0)

	// Walk dot1qVlanCurrentEgressPorts to find VLANs for this port.
	err := params.BulkWalk(OIDDot1qVlanCurrentEgressPorts, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.17.7.1.4.2.1.4.VLAN_ID
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < 2 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}

		vlanID, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			log.Printf("Failed to parse VLAN ID: %v", err)
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
		if len(parts) < 2 {
			log.Printf("Invalid OID format: %s", pdu.Name)
			return nil
		}

		bridgePort, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			log.Printf("Failed to parse bridge port: %v", err)
			return nil
		}

		ifIndex, err := strconv.Atoi(formatSNMPValue(pdu))
		if err != nil {
			log.Printf("Failed to parse ifIndex: %v", err)
			return nil
		}

		mapping[bridgePort] = ifIndex
		return nil
	})
	if err != nil {
		log.Printf("Failed to get bridge port mapping: %v", err)
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
	if len(parts) < 6 {
		return ""
	}

	mac := make([]string, 6)
	for i := range 6 {
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
	duration := time.Duration(ticks) * 10 * time.Millisecond

	// Return time relative to now (approximation).
	return time.Now().Add(-duration)
}

// portListContainsPort checks if a port list bitmap contains the specified port.
// Port lists are byte arrays where each bit represents a port.
func portListContainsPort(portList interface{}, portNum int) bool {
	bytes, ok := portList.([]byte)
	if !ok {
		return false
	}

	// Port numbering starts at 1, bitmap starts at 0.
	portIdx := portNum - 1
	if portIdx < 0 {
		return false
	}

	byteIdx := portIdx / 8
	bitIdx := 7 - (portIdx % 8) // #nosec G115 -- portIdx%8 is always 0-7

	if byteIdx >= len(bytes) {
		return false
	}

	return (bytes[byteIdx] & (1 << bitIdx)) != 0
}
