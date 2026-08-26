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

	return sweepCredentials(ctx, cfg, "failed to query port VLANs with all configured credentials",
		func(cred *config.SNMPv3Credential) ([]int, error) {
			return getPortVLANsWithV3(ctx, ip, ifIndex, cred, cfg)
		},
		func(community string) ([]int, error) {
			return getPortVLANsWithCommunity(ctx, ip, ifIndex, community, cfg)
		},
	)
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
