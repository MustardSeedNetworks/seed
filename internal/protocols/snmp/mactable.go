package snmp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

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
