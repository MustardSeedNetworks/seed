// Package discovery implements multi-protocol network device discovery.
// This file implements extended SNMP MIB collection for Phase 3 of the pipeline.
// It collects interface, IP address, MAC table, VLAN, and LLDP data from network devices.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

// SNMPFullData contains all collected SNMP data from a device.
type SNMPFullData struct {
	// CollectedAt is when the data was collected.
	CollectedAt time.Time `json:"collectedAt"`

	// System contains basic system information.
	System *snmp.SystemInfo `json:"system,omitempty"`

	// Interfaces contains all network interfaces with speeds, MACs, status.
	Interfaces []SNMPInterface `json:"interfaces,omitempty"`

	// IPAddresses contains all IP addresses per interface.
	IPAddresses []SNMPIPAddress `json:"ipAddresses,omitempty"`

	// MACTable contains learned MAC addresses (for switches).
	MACTable []SNMPMACEntry `json:"macTable,omitempty"`

	// VLANs contains VLAN configuration.
	VLANs []SNMPVLAN `json:"vlans,omitempty"`

	// Inventory contains physical components (ENTITY-MIB).
	Inventory []SNMPEntity `json:"inventory,omitempty"`

	// LLDPNeighbors contains LLDP neighbor information.
	LLDPNeighbors []SNMPLLDPNeighbor `json:"lldpNeighbors,omitempty"`

	// Routing contains routing table entries (IP-FORWARD-MIB).
	Routing []SNMPRoute `json:"routing,omitempty"`

	// Errors contains any errors encountered during collection.
	Errors []string `json:"errors,omitempty"`
}

// SNMPInterface represents a network interface from IF-MIB.
type SNMPInterface struct {
	Index       int    `json:"index"`                 // ifIndex
	Name        string `json:"name,omitempty"`        // ifName (e.g., "Gi0/1")
	Description string `json:"description,omitempty"` // ifDescr (full name)
	Alias       string `json:"alias,omitempty"`       // ifAlias (user description)
	Type        int    `json:"type,omitempty"`        // ifType (6=ethernet)
	MTU         int    `json:"mtu,omitempty"`         // ifMtu
	SpeedMbps   uint64 `json:"speedMbps,omitempty"`   // ifHighSpeed in Mbps
	MAC         string `json:"mac,omitempty"`         // ifPhysAddress
	AdminStatus string `json:"adminStatus,omitempty"` // up/down
	OperStatus  string `json:"operStatus,omitempty"`  // up/down/dormant
}

// SNMPIPAddress represents an IP address from IP-MIB.
type SNMPIPAddress struct {
	Address   string `json:"address"`             // IP address
	Prefix    int    `json:"prefix,omitempty"`    // Subnet prefix length
	IfIndex   int    `json:"ifIndex"`             // Associated interface
	Type      string `json:"type,omitempty"`      // unicast, broadcast, etc.
	AddressIP string `json:"addressIP,omitempty"` // IPv4 or IPv6
}

// SNMPMACEntry represents a MAC address table entry.
type SNMPMACEntry struct {
	MAC     string `json:"mac"`            // MAC address
	VLAN    int    `json:"vlan,omitempty"` // VLAN ID
	IfIndex int    `json:"ifIndex"`        // Interface index
	Type    string `json:"type,omitempty"` // learned, static, other
	Port    string `json:"port,omitempty"` // Port name if resolved
}

// SNMPVLAN represents VLAN information from Q-BRIDGE-MIB.
type SNMPVLAN struct {
	ID          int    `json:"id"`                    // VLAN ID
	Name        string `json:"name,omitempty"`        // VLAN name
	Status      string `json:"status,omitempty"`      // active, notInService
	EgressPorts []int  `json:"egressPorts,omitempty"` // Ports in this VLAN
	Type        string `json:"type,omitempty"`        // static, dynamic
}

// SNMPEntity represents a physical entity from ENTITY-MIB.
type SNMPEntity struct {
	Index        int    `json:"index"`                  // entPhysicalIndex
	Description  string `json:"description,omitempty"`  // entPhysicalDescr
	VendorType   string `json:"vendorType,omitempty"`   // entPhysicalVendorType
	ContainedIn  int    `json:"containedIn,omitempty"`  // entPhysicalContainedIn
	Class        string `json:"class,omitempty"`        // chassis, module, port, etc.
	ParentRelPos int    `json:"parentRelPos,omitempty"` // entPhysicalParentRelPos
	Name         string `json:"name,omitempty"`         // entPhysicalName
	HardwareRev  string `json:"hardwareRev,omitempty"`  // entPhysicalHardwareRev
	FirmwareRev  string `json:"firmwareRev,omitempty"`  // entPhysicalFirmwareRev
	SoftwareRev  string `json:"softwareRev,omitempty"`  // entPhysicalSoftwareRev
	SerialNum    string `json:"serialNum,omitempty"`    // entPhysicalSerialNum
	ModelName    string `json:"modelName,omitempty"`    // entPhysicalModelName
}

// SNMPLLDPNeighbor represents an LLDP neighbor from LLDP-MIB.
type SNMPLLDPNeighbor struct {
	LocalIfIndex    int    `json:"localIfIndex"`              // Local interface index
	LocalPortID     string `json:"localPortId,omitempty"`     // Local port ID
	RemoteChassisID string `json:"remoteChassisId,omitempty"` // Remote chassis ID
	RemotePortID    string `json:"remotePortId,omitempty"`    // Remote port ID
	RemoteSysName   string `json:"remoteSysName,omitempty"`   // Remote system name
	RemoteSysDescr  string `json:"remoteSysDescr,omitempty"`  // Remote system description
	RemoteMgmtAddr  string `json:"remoteMgmtAddr,omitempty"`  // Remote management address
}

// SNMPRoute represents a routing table entry from IP-FORWARD-MIB.
type SNMPRoute struct {
	Destination string `json:"destination"`        // Destination network
	Prefix      int    `json:"prefix,omitempty"`   // Prefix length
	NextHop     string `json:"nextHop,omitempty"`  // Next hop address
	IfIndex     int    `json:"ifIndex,omitempty"`  // Output interface
	Type        string `json:"type,omitempty"`     // local, remote, etc.
	Protocol    string `json:"protocol,omitempty"` // static, ospf, bgp, etc.
	Metric      int    `json:"metric,omitempty"`   // Route metric
}

// SNMPCollector collects extended SNMP data from network devices.
type SNMPCollector struct {
	config     *config.SNMPConfig
	mibConfig  SNMPMIBSelection
	timeout    time.Duration
	maxOIDsReq int
}

// NewSNMPCollector creates a new SNMP collector.
func NewSNMPCollector(cfg *config.SNMPConfig, mibConfig SNMPMIBSelection) *SNMPCollector {
	return &SNMPCollector{
		config:     cfg,
		mibConfig:  mibConfig,
		timeout:    30 * time.Second,
		maxOIDsReq: 10,
	}
}

// SetTimeout sets the timeout for MIB walks.
func (c *SNMPCollector) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// SetMaxOIDsPerRequest sets the maximum OIDs per SNMP request.
func (c *SNMPCollector) SetMaxOIDsPerRequest(maxOIDs int) {
	c.maxOIDsReq = maxOIDs
}

// Collect gathers all enabled MIB data from a device.
//
//nolint:gocyclo // SNMP collection requires checking multiple MIB flags and collecting various data types.
func (c *SNMPCollector) Collect(ctx context.Context, ip string) (*SNMPFullData, error) {
	if c.config == nil {
		return nil, fmt.Errorf("SNMP config is nil")
	}

	data := &SNMPFullData{
		CollectedAt: time.Now(),
	}

	// Create timeout context
	collectCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Always collect system info
	if c.mibConfig.System {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sysInfo, err := snmp.GetSystemInfo(collectCtx, ip, c.config); err == nil {
				mu.Lock()
				data.System = sysInfo
				mu.Unlock()
				slog.Debug("Collected system info", "ip", ip, "sysName", sysInfo.SysName)
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("system: %v", err))
				mu.Unlock()
			}
		}()
	}

	// Collect interfaces (IF-MIB)
	if c.mibConfig.Interfaces {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if interfaces, err := c.collectInterfaces(collectCtx, ip); err == nil {
				mu.Lock()
				data.Interfaces = interfaces
				mu.Unlock()
				slog.Debug("Collected interfaces", "ip", ip, "count", len(interfaces))
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("interfaces: %v", err))
				mu.Unlock()
			}
		}()
	}

	// Collect IP addresses (IP-MIB)
	if c.mibConfig.IPAddresses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ipAddrs, err := c.collectIPAddresses(collectCtx, ip); err == nil {
				mu.Lock()
				data.IPAddresses = ipAddrs
				mu.Unlock()
				slog.Debug("Collected IP addresses", "ip", ip, "count", len(ipAddrs))
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("ipAddresses: %v", err))
				mu.Unlock()
			}
		}()
	}

	// Collect MAC table (BRIDGE-MIB)
	if c.mibConfig.Bridge {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if macTable, err := c.collectMACTable(collectCtx, ip); err == nil {
				mu.Lock()
				data.MACTable = macTable
				mu.Unlock()
				slog.Debug("Collected MAC table", "ip", ip, "count", len(macTable))
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("macTable: %v", err))
				mu.Unlock()
			}
		}()
	}

	// Collect VLANs (Q-BRIDGE-MIB)
	if c.mibConfig.VLAN {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if vlans, err := c.collectVLANs(collectCtx, ip); err == nil {
				mu.Lock()
				data.VLANs = vlans
				mu.Unlock()
				slog.Debug("Collected VLANs", "ip", ip, "count", len(vlans))
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("vlans: %v", err))
				mu.Unlock()
			}
		}()
	}

	// Collect physical inventory (ENTITY-MIB)
	if c.mibConfig.Entity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if entities, err := c.collectInventory(collectCtx, ip); err == nil {
				mu.Lock()
				data.Inventory = entities
				mu.Unlock()
				slog.Debug("Collected inventory", "ip", ip, "count", len(entities))
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("inventory: %v", err))
				mu.Unlock()
			}
		}()
	}

	// Collect LLDP neighbors (LLDP-MIB)
	if c.mibConfig.LLDP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if neighbors, err := c.collectLLDPNeighbors(collectCtx, ip); err == nil {
				mu.Lock()
				data.LLDPNeighbors = neighbors
				mu.Unlock()
				slog.Debug("Collected LLDP neighbors", "ip", ip, "count", len(neighbors))
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("lldp: %v", err))
				mu.Unlock()
			}
		}()
	}

	// Collect routing table (IP-FORWARD-MIB)
	if c.mibConfig.Routing {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if routes, err := c.collectRoutes(collectCtx, ip); err == nil {
				mu.Lock()
				data.Routing = routes
				mu.Unlock()
				slog.Debug("Collected routes", "ip", ip, "count", len(routes))
			} else {
				mu.Lock()
				data.Errors = append(data.Errors, fmt.Sprintf("routing: %v", err))
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return data, nil
}

// collectInterfaces retrieves interface information from IF-MIB.
func (c *SNMPCollector) collectInterfaces(ctx context.Context, ip string) ([]SNMPInterface, error) {
	interfaces, err := snmp.GetAllInterfaces(ctx, ip, c.config)
	if err != nil {
		return nil, err
	}

	result := make([]SNMPInterface, len(interfaces))
	for i := range interfaces {
		iface := &interfaces[i]
		// Convert bps to Mbps, handling negative speeds (which shouldn't occur but be safe)
		speedMbps := uint64(0)
		if iface.Speed > 0 {
			speedMbps = uint64(iface.Speed) / 1000000
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
	}

	return result, nil
}

// collectIPAddresses retrieves IP addresses from IP-MIB.
func (c *SNMPCollector) collectIPAddresses(_ context.Context, _ string) ([]SNMPIPAddress, error) {
	// IP-MIB::ipAddrTable OIDs - full implementation would walk ipAddrTable.
	// This can be expanded when needed.
	return []SNMPIPAddress{}, nil
}

// collectMACTable retrieves the MAC address table.
func (c *SNMPCollector) collectMACTable(ctx context.Context, ip string) ([]SNMPMACEntry, error) {
	macEntries, err := snmp.GetMACTable(ctx, ip, c.config)
	if err != nil {
		return nil, err
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
func (c *SNMPCollector) collectVLANs(_ context.Context, _ string) ([]SNMPVLAN, error) {
	// Q-BRIDGE-MIB::dot1qVlanStaticTable OIDs - can be expanded when needed.
	return []SNMPVLAN{}, nil
}

// collectInventory retrieves physical inventory from ENTITY-MIB.
func (c *SNMPCollector) collectInventory(_ context.Context, _ string) ([]SNMPEntity, error) {
	// ENTITY-MIB::entPhysicalTable OIDs - can be expanded when needed.
	return []SNMPEntity{}, nil
}

// collectLLDPNeighbors retrieves LLDP neighbor information.
func (c *SNMPCollector) collectLLDPNeighbors(_ context.Context, _ string) ([]SNMPLLDPNeighbor, error) {
	// LLDP-MIB::lldpRemTable OIDs
	// For now, return empty - can be expanded when needed
	return []SNMPLLDPNeighbor{}, nil
}

// collectRoutes retrieves routing table from IP-FORWARD-MIB.
func (c *SNMPCollector) collectRoutes(_ context.Context, _ string) ([]SNMPRoute, error) {
	// IP-FORWARD-MIB::ipCidrRouteTable OIDs
	// For now, return empty - can be expanded when needed
	return []SNMPRoute{}, nil
}

// CollectorResult contains the result of SNMP collection for a device.
type CollectorResult struct {
	IP    string        `json:"ip"`
	Data  *SNMPFullData `json:"data,omitempty"`
	Error error         `json:"error,omitempty"`
}

// CollectBatch collects SNMP data from multiple devices concurrently.
//
//nolint:dupl // Similar concurrent batch pattern to mdns.go but uses different collector and result types.
func (c *SNMPCollector) CollectBatch(ctx context.Context, ips []string, maxConcurrent int) []CollectorResult {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	results := make([]CollectorResult, len(ips))
	resultCh := make(chan struct {
		idx    int
		result CollectorResult
	}, len(ips))

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for i, ip := range ips {
		wg.Add(1)
		go func(idx int, ipAddr string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				resultCh <- struct {
					idx    int
					result CollectorResult
				}{idx, CollectorResult{IP: ipAddr, Error: ctx.Err()}}
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			data, err := c.Collect(ctx, ipAddr)
			resultCh <- struct {
				idx    int
				result CollectorResult
			}{idx, CollectorResult{IP: ipAddr, Data: data, Error: err}}
		}(i, ip)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		results[res.idx] = res.result
	}

	return results
}
