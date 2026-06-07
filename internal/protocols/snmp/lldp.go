package snmp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// LLDP-MIB OIDs (lldpRemTable - remote device information).
const (
	// OIDLldpRemChassisIDSubtype is the LLDP-MIB OID for chassis ID subtype.
	OIDLldpRemChassisIDSubtype = "1.0.8802.1.1.2.1.4.1.1.4"
	// OIDLldpRemChassisID is the LLDP-MIB OID for chassis ID.
	OIDLldpRemChassisID = "1.0.8802.1.1.2.1.4.1.1.5"
	// OIDLldpRemPortIDSubtype is the LLDP-MIB OID for port ID subtype.
	OIDLldpRemPortIDSubtype = "1.0.8802.1.1.2.1.4.1.1.6"
	// OIDLldpRemPortID is the LLDP-MIB OID for port ID.
	OIDLldpRemPortID = "1.0.8802.1.1.2.1.4.1.1.7"
	// OIDLldpRemPortDesc is the LLDP-MIB OID for port description.
	OIDLldpRemPortDesc = "1.0.8802.1.1.2.1.4.1.1.8"
	// OIDLldpRemSysName is the LLDP-MIB OID for system name.
	OIDLldpRemSysName = "1.0.8802.1.1.2.1.4.1.1.9"
	// OIDLldpRemSysDesc is the LLDP-MIB OID for system description.
	OIDLldpRemSysDesc = "1.0.8802.1.1.2.1.4.1.1.10"

	// OIDLldpRemManAddrIfSubtype is the LLDP-MIB OID for management address interface subtype.
	OIDLldpRemManAddrIfSubtype = "1.0.8802.1.1.2.1.4.2.1.3"
	// OIDLldpRemManAddrIfID is the LLDP-MIB OID for management address interface ID.
	OIDLldpRemManAddrIfID = "1.0.8802.1.1.2.1.4.2.1.4"
)

// LLDP OID parsing constants.
const (
	// minOIDPartsLLDPIndex is the minimum OID parts to extract LLDP index components.
	// Format: ...TimeMark.LocalPortNum.RemoteIndex (need at least 3 trailing parts).
	minOIDPartsLLDPIndex = 3
)

// Network address length constants for chassis ID formatting.
const (
	// macAddressLength is the byte length of a MAC address.
	macAddressLength = 6
	// ipv4AddressLength is the byte length of an IPv4 address.
	ipv4AddressLength = 4
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
func GetLLDPNeighbors(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) ([]LLDPNeighbor, error) {
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
func walkLLDP(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) ([]LLDPNeighbor, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkLLDPTable(params)
}

// walkLLDPV3 walks the LLDP-MIB tables using SNMPv3.
func walkLLDPV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]LLDPNeighbor, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkLLDPTable(params)
}

// walkLLDPTable walks the LLDP remote device table.
func walkLLDPTable(params *gosnmp.GoSNMP) ([]LLDPNeighbor, error) {
	neighbors := make(map[string]*LLDPNeighbor)

	// Walk lldpRemChassisId to discover all neighbors.
	err := params.BulkWalk(OIDLldpRemChassisID, func(pdu gosnmp.SnmpPDU) error {
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
	walkLLDPAttribute(
		params,
		OIDLldpRemChassisIDSubtype,
		neighbors,
		func(n *LLDPNeighbor, value string) {
			n.ChassisIDType = parseChassisIDSubtype(value)
		},
	)

	// Walk lldpRemPortId.
	walkLLDPAttribute(params, OIDLldpRemPortID, neighbors, func(n *LLDPNeighbor, value string) {
		n.PortID = value
	})

	// Walk lldpRemPortIdSubtype.
	walkLLDPAttribute(
		params,
		OIDLldpRemPortIDSubtype,
		neighbors,
		func(n *LLDPNeighbor, value string) {
			n.PortIDType = parsePortIDSubtype(value)
		},
	)

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
		logging.GetLogger().Debug("Failed to walk LLDP attribute", "oid", oid, "error", err)
	}
}

// extractLLDPIndex extracts local port and remote index from LLDP OID.
// OID format: ...TimeMark.LocalPortNum.RemoteIndex.
func extractLLDPIndex(oid string) (int, int) {
	parts := strings.Split(oid, ".")
	if len(parts) < minOIDPartsLLDPIndex {
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
	if len(bytes) == macAddressLength {
		return net.HardwareAddr(bytes).String()
	}

	// Check if it's an IP address (4 bytes)
	if len(bytes) == ipv4AddressLength {
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
