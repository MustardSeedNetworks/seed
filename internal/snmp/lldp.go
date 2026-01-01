// Package snmp provides SNMP query functionality for network device discovery.
// This file implements LLDP-MIB collection for L2 topology discovery.
// LLDP (Link Layer Discovery Protocol) provides neighbor information for
// building network topology diagrams.
package snmp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
)

// LLDP-MIB OIDs.
const (
	// lldpRemTable - remote device information.
	OIDLldpRemChassisIdSubtype = "1.0.8802.1.1.2.1.4.1.1.4"  // lldpRemChassisIdSubtype
	OIDLldpRemChassisId        = "1.0.8802.1.1.2.1.4.1.1.5"  // lldpRemChassisId
	OIDLldpRemPortIdSubtype    = "1.0.8802.1.1.2.1.4.1.1.6"  // lldpRemPortIdSubtype
	OIDLldpRemPortId           = "1.0.8802.1.1.2.1.4.1.1.7"  // lldpRemPortId
	OIDLldpRemPortDesc         = "1.0.8802.1.1.2.1.4.1.1.8"  // lldpRemPortDesc
	OIDLldpRemSysName          = "1.0.8802.1.1.2.1.4.1.1.9"  // lldpRemSysName
	OIDLldpRemSysDesc          = "1.0.8802.1.1.2.1.4.1.1.10" // lldpRemSysDesc

	// lldpRemManAddrTable - remote management addresses.
	OIDLldpRemManAddrIfSubtype = "1.0.8802.1.1.2.1.4.2.1.3" // lldpRemManAddrIfSubtype
	OIDLldpRemManAddrIfId      = "1.0.8802.1.1.2.1.4.2.1.4" // lldpRemManAddrIfId
)

// LLDPNeighbor contains LLDP neighbor information from LLDP-MIB.
type LLDPNeighbor struct {
	LocalIfIndex    int    // Local interface index (from OID)
	LocalPortNum    int    // Local port number (from OID)
	RemoteIndex     int    // Remote neighbor index (from OID)
	ChassisIDType   string // Type of chassis ID (macAddress, networkAddress, etc.)
	ChassisID       string // Remote chassis ID
	PortIDType      string // Type of port ID
	PortID          string // Remote port ID
	PortDescription string // Remote port description
	SystemName      string // Remote system name
	SystemDesc      string // Remote system description
	MgmtAddress     string // Remote management address
}

// GetLLDPNeighbors retrieves all LLDP neighbors from a device.
// Security: SNMPv3 is preferred over v2c when both are configured.
func GetLLDPNeighbors(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]LLDPNeighbor, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		neighbors, err := walkLLDPV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return neighbors, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		neighbors, err := walkLLDP(ctx, ip, community, cfg)
		if err == nil {
			return neighbors, nil
		}
	}

	return nil, errors.New("failed to query LLDP neighbors with all configured credentials")
}

// walkLLDP walks the LLDP-MIB tables using SNMPv2c.
func walkLLDP(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]LLDPNeighbor, error) {
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

	return walkLLDPTable(params)
}

// walkLLDPV3 walks the LLDP-MIB tables using SNMPv3.
func walkLLDPV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]LLDPNeighbor, error) {
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

	return walkLLDPTable(params)
}

// walkLLDPTable walks the LLDP remote device table.
func walkLLDPTable(params *gosnmp.GoSNMP) ([]LLDPNeighbor, error) {
	neighbors := make(map[string]*LLDPNeighbor)

	// Walk lldpRemChassisId to discover all neighbors.
	err := params.BulkWalk(OIDLldpRemChassisId, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.0.8802.1.1.2.1.4.1.1.5.TimeMark.LocalPortNum.RemoteIndex
		localPort, remoteIdx := extractLLDPIndex(pdu.Name)
		if localPort <= 0 || remoteIdx <= 0 {
			return nil
		}

		key := fmt.Sprintf("%d-%d", localPort, remoteIdx)
		neighbors[key] = &LLDPNeighbor{
			LocalPortNum: localPort,
			RemoteIndex:  remoteIdx,
			ChassisID:    formatChassisID(pdu.Value),
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk lldpRemChassisId: %w", err)
	}

	// Walk lldpRemChassisIdSubtype.
	walkLLDPAttribute(params, OIDLldpRemChassisIdSubtype, neighbors, func(n *LLDPNeighbor, value string) {
		n.ChassisIDType = parseChassisIDSubtype(value)
	})

	// Walk lldpRemPortId.
	walkLLDPAttribute(params, OIDLldpRemPortId, neighbors, func(n *LLDPNeighbor, value string) {
		n.PortID = value
	})

	// Walk lldpRemPortIdSubtype.
	walkLLDPAttribute(params, OIDLldpRemPortIdSubtype, neighbors, func(n *LLDPNeighbor, value string) {
		n.PortIDType = parsePortIDSubtype(value)
	})

	// Walk lldpRemPortDesc.
	walkLLDPAttribute(params, OIDLldpRemPortDesc, neighbors, func(n *LLDPNeighbor, value string) {
		n.PortDescription = value
	})

	// Walk lldpRemSysName.
	walkLLDPAttribute(params, OIDLldpRemSysName, neighbors, func(n *LLDPNeighbor, value string) {
		n.SystemName = value
	})

	// Walk lldpRemSysDesc.
	walkLLDPAttribute(params, OIDLldpRemSysDesc, neighbors, func(n *LLDPNeighbor, value string) {
		n.SystemDesc = value
	})

	// Convert map to slice.
	result := make([]LLDPNeighbor, 0, len(neighbors))
	for _, neighbor := range neighbors {
		result = append(result, *neighbor)
	}

	return result, nil
}

// walkLLDPAttribute walks an LLDP attribute and applies a function.
func walkLLDPAttribute(
	params *gosnmp.GoSNMP,
	oid string,
	neighbors map[string]*LLDPNeighbor,
	updateFunc func(*LLDPNeighbor, string),
) {
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		localPort, remoteIdx := extractLLDPIndex(pdu.Name)
		if localPort <= 0 || remoteIdx <= 0 {
			return nil
		}

		key := fmt.Sprintf("%d-%d", localPort, remoteIdx)
		neighbor, exists := neighbors[key]
		if !exists {
			return nil
		}

		updateFunc(neighbor, formatSNMPValue(pdu))
		return nil
	})
	if err != nil {
		slog.Debug("Failed to walk LLDP attribute", "oid", oid, "error", err)
	}
}

// extractLLDPIndex extracts local port and remote index from LLDP OID.
// OID format: ...TimeMark.LocalPortNum.RemoteIndex.
func extractLLDPIndex(oid string) (int, int) {
	parts := strings.Split(oid, ".")
	if len(parts) < 3 {
		return 0, 0
	}

	// Last element is RemoteIndex
	remoteIdx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, 0
	}

	// Second to last is LocalPortNum
	localPort, err := strconv.Atoi(parts[len(parts)-2])
	if err != nil {
		return 0, 0
	}

	return localPort, remoteIdx
}

// formatChassisID formats chassis ID based on its content.
func formatChassisID(value any) string {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Sprintf("%v", value)
	}

	// Check if it's a MAC address (6 bytes)
	if len(bytes) == 6 {
		return net.HardwareAddr(bytes).String()
	}

	// Check if it's an IP address (4 bytes)
	if len(bytes) == 4 {
		return net.IP(bytes).String()
	}

	// Try to interpret as string
	if isPrintable(bytes) {
		return string(bytes)
	}

	// Fall back to hex encoding
	return hex.EncodeToString(bytes)
}

// isPrintable checks if all bytes are printable ASCII.
func isPrintable(data []byte) bool {
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return len(data) > 0
}

// parseChassisIDSubtype converts chassis ID subtype to string.
func parseChassisIDSubtype(value string) string {
	switch value {
	case "1":
		return "chassisComponent"
	case "2":
		return "interfaceAlias"
	case "3":
		return "portComponent"
	case "4":
		return "macAddress"
	case "5":
		return "networkAddress"
	case "6":
		return "interfaceName"
	case "7":
		return IDSubtypeLocal
	default:
		return StatusUnknown
	}
}

// parsePortIDSubtype converts port ID subtype to string.
func parsePortIDSubtype(value string) string {
	switch value {
	case "1":
		return "interfaceAlias"
	case "2":
		return "portComponent"
	case "3":
		return "macAddress"
	case "4":
		return "networkAddress"
	case "5":
		return "interfaceName"
	case "6":
		return "agentCircuitId"
	case "7":
		return IDSubtypeLocal
	default:
		return StatusUnknown
	}
}
