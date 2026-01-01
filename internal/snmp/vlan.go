// Package snmp provides SNMP query functionality for network device discovery.
// This file implements Q-BRIDGE-MIB (IEEE 802.1Q) VLAN collection for L2 topology.
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

// Q-BRIDGE-MIB OIDs (IEEE 802.1Q).
const (
	// dot1qVlanStaticTable - static VLAN configuration.
	OIDDot1qVlanStaticName        = "1.3.6.1.2.1.17.7.1.4.3.1.1" // dot1qVlanStaticName
	OIDDot1qVlanStaticEgressPorts = "1.3.6.1.2.1.17.7.1.4.3.1.2" // dot1qVlanStaticEgressPorts
	OIDDot1qVlanStaticRowStatus   = "1.3.6.1.2.1.17.7.1.4.3.1.5" // dot1qVlanStaticRowStatus

	// dot1qVlanCurrentTable - current VLAN state.
	OIDDot1qVlanFdbID = "1.3.6.1.2.1.17.7.1.4.2.1.3" // dot1qVlanFdbId

	// dot1qPvid - port VLAN ID.
	OIDDot1qPvid = "1.3.6.1.2.1.17.7.1.4.5.1.1" // dot1qPvid
)

// VLANInfo contains VLAN information from Q-BRIDGE-MIB.
type VLANInfo struct {
	ID          int    // VLAN ID
	Name        string // VLAN name
	Status      string // active, notInService, notReady, createAndGo, createAndWait, destroy
	EgressPorts []int  // List of port indices that are members
	Type        string // static, dynamic
}

// GetVLANs retrieves all VLANs from a device using Q-BRIDGE-MIB.
// Security: SNMPv3 is preferred over v2c when both are configured.
func GetVLANs(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]VLANInfo, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		vlans, err := walkVLANsV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return vlans, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		vlans, err := walkVLANs(ctx, ip, community, cfg)
		if err == nil {
			return vlans, nil
		}
	}

	return nil, errors.New("failed to query Q-BRIDGE VLANs with all configured credentials")
}

// walkVLANs walks the Q-BRIDGE VLAN tables using SNMPv2c.
func walkVLANs(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]VLANInfo, error) {
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

	return walkVLANTable(params), nil
}

// walkVLANsV3 walks the Q-BRIDGE VLAN tables using SNMPv3.
func walkVLANsV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]VLANInfo, error) {
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

	return walkVLANTable(params), nil
}

// walkVLANTable walks the Q-BRIDGE VLAN tables.
// Errors during walks are logged but not returned since we want to collect as much data as possible.
func walkVLANTable(params *gosnmp.GoSNMP) []VLANInfo {
	vlans := make(map[int]*VLANInfo)

	// Walk dot1qVlanStaticName to discover all VLANs.
	err := params.BulkWalk(OIDDot1qVlanStaticName, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.17.7.1.4.3.1.1.VLAN_ID
		vlanID := extractVLANIndex(pdu.Name)
		if vlanID <= 0 {
			return nil
		}

		vlans[vlanID] = &VLANInfo{
			ID:   vlanID,
			Name: formatSNMPValue(pdu),
			Type: "static",
		}
		return nil
	})
	if err != nil {
		// If static table fails, try current table
		slog.Debug("Failed to walk dot1qVlanStaticName, trying current table", "error", err)
	}

	// Walk dot1qVlanStaticEgressPorts to get port membership.
	walkErr := params.BulkWalk(OIDDot1qVlanStaticEgressPorts, func(pdu gosnmp.SnmpPDU) error {
		vlanID := extractVLANIndex(pdu.Name)
		if vlanID <= 0 {
			return nil
		}

		vlan, exists := vlans[vlanID]
		if !exists {
			vlan = &VLANInfo{
				ID:   vlanID,
				Type: "static",
			}
			vlans[vlanID] = vlan
		}

		// Parse port bitmap
		vlan.EgressPorts = parsePortBitmap(pdu.Value)
		return nil
	})
	if walkErr != nil {
		slog.Debug("Failed to walk dot1qVlanStaticEgressPorts", "error", walkErr)
	}

	// Walk dot1qVlanStaticRowStatus to get VLAN status.
	walkErr = params.BulkWalk(OIDDot1qVlanStaticRowStatus, func(pdu gosnmp.SnmpPDU) error {
		vlanID := extractVLANIndex(pdu.Name)
		if vlanID <= 0 {
			return nil
		}

		vlan, exists := vlans[vlanID]
		if !exists {
			return nil
		}

		vlan.Status = parseRowStatus(formatSNMPValue(pdu))
		return nil
	})
	if walkErr != nil {
		slog.Debug("Failed to walk dot1qVlanStaticRowStatus", "error", walkErr)
	}

	// Convert map to slice.
	result := make([]VLANInfo, 0, len(vlans))
	for _, vlan := range vlans {
		if vlan.Status == "" {
			vlan.Status = "active"
		}
		result = append(result, *vlan)
	}

	return result
}

// extractVLANIndex extracts VLAN ID from OID.
func extractVLANIndex(oid string) int {
	parts := strings.Split(oid, ".")
	if len(parts) < 2 {
		return 0
	}

	vlanID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return vlanID
}

// parsePortBitmap parses a port bitmap into a list of port indices.
func parsePortBitmap(value any) []int {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	ports := make([]int, 0)
	for byteIdx, b := range bytes {
		for bitIdx := 7; bitIdx >= 0; bitIdx-- {
			if b&(1<<bitIdx) != 0 {
				// Port numbering starts at 1, bit 7 of byte 0 is port 1
				portNum := byteIdx*8 + (7 - bitIdx) + 1
				ports = append(ports, portNum)
			}
		}
	}

	return ports
}

// parseRowStatus converts SNMP RowStatus to string.
func parseRowStatus(value string) string {
	switch value {
	case "1":
		return "active"
	case "2":
		return "notInService"
	case "3":
		return "notReady"
	case "4":
		return "createAndGo"
	case "5":
		return "createAndWait"
	case "6":
		return "destroy"
	default:
		return StatusUnknown
	}
}
