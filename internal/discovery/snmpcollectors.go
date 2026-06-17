package discovery

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// collectInterfaces retrieves interface information from IF-MIB.
func (c *SNMPCollector) collectInterfaces(ctx context.Context, ip string) ([]SNMPInterface, error) {
	interfaces, err := snmp.GetAllInterfaces(ctx, ip, c.config)
	if err != nil {
		return nil, fmt.Errorf("get interfaces: %w", err)
	}

	// Also collect interface counters for bandwidth monitoring
	counters, countersErr := snmp.GetInterfaceCounters(ctx, ip, c.config)
	if countersErr != nil {
		logging.GetLogger().DebugContext(ctx, "Failed to get interface counters", "ip", ip, "error", countersErr)
		// Don't fail - counters are optional enhancement
	}

	result := make([]SNMPInterface, len(interfaces))
	for i := range interfaces {
		iface := &interfaces[i]
		// Convert bps to Mbps, handling negative speeds (which shouldn't occur but be safe)
		speedMbps := uint64(0)
		if iface.Speed > 0 {
			speedMbps = uint64(iface.Speed) / snmpSpeedMbpsDivisor
		}
		result[i] = SNMPInterface{
			Index:       iface.Index,
			Name:        iface.Name,
			Description: iface.Description,
			SpeedMbps:   speedMbps,
			MAC:         iface.MACAddress,
			AdminStatus: iface.AdminStatus,
			OperStatus:  iface.OperStatus,
		}

		// Add counter data if available (for bandwidth monitoring)
		if counters != nil {
			if counter, ok := counters[iface.Index]; ok {
				result[i].InOctets = counter.InOctets
				result[i].OutOctets = counter.OutOctets
				result[i].InErrors = counter.InErrors
				result[i].OutErrors = counter.OutErrors
				result[i].InDiscards = counter.InDiscards
				result[i].OutDiscards = counter.OutDiscards
				result[i].LastUpdated = counter.Timestamp
			}
		}
	}

	return result, nil
}

// collectIPAddresses retrieves IP addresses from IP-MIB.
func (c *SNMPCollector) collectIPAddresses(
	ctx context.Context,
	ip string,
) ([]SNMPIPAddress, error) {
	entries, err := snmp.GetIPAddresses(ctx, ip, c.config)
	if err != nil {
		return nil, fmt.Errorf("get IP addresses: %w", err)
	}

	result := make([]SNMPIPAddress, len(entries))
	for i, entry := range entries {
		result[i] = SNMPIPAddress{
			Address:   entry.Address,
			Prefix:    entry.Prefix,
			IfIndex:   entry.IfIndex,
			Type:      entry.Type,
			AddressIP: entry.AddressIP,
		}
	}

	return result, nil
}

// collectMACTable retrieves the MAC address table.
func (c *SNMPCollector) collectMACTable(ctx context.Context, ip string) ([]SNMPMACEntry, error) {
	macEntries, err := snmp.GetMACTable(ctx, ip, c.config)
	if err != nil {
		return nil, fmt.Errorf("get MAC table: %w", err)
	}

	result := make([]SNMPMACEntry, len(macEntries))
	for i, entry := range macEntries {
		result[i] = SNMPMACEntry{
			MAC:     entry.MAC,
			VLAN:    entry.VLAN,
			IfIndex: entry.IfIndex,
			Type:    entry.Type,
		}
	}

	return result, nil
}

// collectVLANs retrieves VLAN information from Q-BRIDGE-MIB.
func (c *SNMPCollector) collectVLANs(ctx context.Context, ip string) ([]SNMPVLAN, error) {
	vlans, err := snmp.GetVLANs(ctx, ip, c.config)
	if err != nil {
		return nil, fmt.Errorf("get VLANs: %w", err)
	}

	result := make([]SNMPVLAN, len(vlans))
	for i, vlan := range vlans {
		result[i] = SNMPVLAN{
			ID:          vlan.ID,
			Name:        vlan.Name,
			Status:      vlan.Status,
			EgressPorts: vlan.EgressPorts,
			Type:        vlan.Type,
		}
	}

	return result, nil
}

// collectInventory retrieves physical inventory from ENTITY-MIB.
func (c *SNMPCollector) collectInventory(ctx context.Context, ip string) ([]SNMPEntity, error) {
	entities, err := snmp.GetPhysicalEntities(ctx, ip, c.config)
	if err != nil {
		return nil, fmt.Errorf("get physical entities: %w", err)
	}

	result := make([]SNMPEntity, len(entities))
	for i, entity := range entities {
		result[i] = SNMPEntity{
			Index:        entity.Index,
			Description:  entity.Description,
			VendorType:   entity.VendorType,
			ContainedIn:  entity.ContainedIn,
			Class:        entity.Class,
			ParentRelPos: entity.ParentRelPos,
			Name:         entity.Name,
			HardwareRev:  entity.HardwareRev,
			FirmwareRev:  entity.FirmwareRev,
			SoftwareRev:  entity.SoftwareRev,
			SerialNum:    entity.SerialNum,
			MfgName:      entity.MfgName,
			ModelName:    entity.ModelName,
			IsFRU:        entity.IsFRU,
		}
	}

	return result, nil
}

// collectLLDPNeighbors retrieves LLDP neighbor information.
func (c *SNMPCollector) collectLLDPNeighbors(
	ctx context.Context,
	ip string,
) ([]SNMPLLDPNeighbor, error) {
	neighbors, err := snmp.GetLLDPNeighbors(ctx, ip, c.config)
	if err != nil {
		return nil, fmt.Errorf("get LLDP neighbors: %w", err)
	}

	result := make([]SNMPLLDPNeighbor, len(neighbors))
	for i, n := range neighbors {
		result[i] = SNMPLLDPNeighbor{
			LocalIfIndex:    n.LocalIfIndex,
			LocalPortID:     strconv.Itoa(n.LocalPortNum),
			RemoteChassisID: n.ChassisID,
			RemotePortID:    n.PortID,
			RemoteSysName:   n.SystemName,
			RemoteSysDescr:  n.SystemDesc,
			RemoteMgmtAddr:  n.MgmtAddress,
		}
	}

	return result, nil
}

// collectRoutes retrieves routing table from IP-FORWARD-MIB.
func (c *SNMPCollector) collectRoutes(ctx context.Context, ip string) ([]SNMPRoute, error) {
	routes, err := snmp.GetRoutes(ctx, ip, c.config)
	if err != nil {
		return nil, fmt.Errorf("get routes: %w", err)
	}

	result := make([]SNMPRoute, len(routes))
	for i, route := range routes {
		result[i] = SNMPRoute{
			Destination: route.Destination,
			Prefix:      route.Prefix,
			NextHop:     route.NextHop,
			IfIndex:     route.IfIndex,
			Type:        route.Type,
			Protocol:    route.Protocol,
			Metric:      route.Metric,
		}
	}

	return result, nil
}
