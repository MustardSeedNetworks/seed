package discovery

// This file implements extended SNMP MIB collection for Phase 3 of the pipeline.
// It collects interface, IP address, MAC table, VLAN, and LLDP data from network devices.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/snmp"
)

// SNMP collector constants.
const (
	snmpCollectorTimeoutS = 30      // Default timeout for SNMP walks in seconds
	snmpCollectorMaxOIDs  = 10      // Default maximum OIDs per SNMP request
	snmpSpeedMbpsDivisor  = 1000000 // Divisor to convert speed to Mbps
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

	// Interface counters for bandwidth monitoring (IF-MIB/IF-MIB-X)
	InOctets    uint64 `json:"inOctets,omitempty"`    // ifHCInOctets (64-bit) or ifInOctets (32-bit)
	OutOctets   uint64 `json:"outOctets,omitempty"`   // ifHCOutOctets (64-bit) or ifOutOctets (32-bit)
	InErrors    uint64 `json:"inErrors,omitempty"`    // ifInErrors
	OutErrors   uint64 `json:"outErrors,omitempty"`   // ifOutErrors
	InDiscards  uint64 `json:"inDiscards,omitempty"`  // ifInDiscards
	OutDiscards uint64 `json:"outDiscards,omitempty"` // ifOutDiscards
	LastUpdated int64  `json:"lastUpdated,omitempty"` // Unix timestamp when counters were collected
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
	MfgName      string `json:"mfgName,omitempty"`      // entPhysicalMfgName
	ModelName    string `json:"modelName,omitempty"`    // entPhysicalModelName
	IsFRU        bool   `json:"isFRU,omitempty"`        // entPhysicalIsFRU (Field Replaceable Unit)
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
		timeout:    snmpCollectorTimeoutS * time.Second,
		maxOIDsReq: snmpCollectorMaxOIDs,
	}
}

// SetTimeout sets the timeout for MIB walks.
func (c *SNMPCollector) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// SetMaxOIDsPerRequest sets the maximum OIDs per SNMP request.
// This value is passed through to the gosnmp MaxRepetitions field
// via the SNMPConfig.MaxRepetitions setting when collecting MIB data.
func (c *SNMPCollector) SetMaxOIDsPerRequest(maxOIDs int) {
	c.maxOIDsReq = maxOIDs
	// Update config MaxRepetitions so walks use the new value
	if c.config != nil && maxOIDs > 0 {
		c.config.MaxRepetitions = uint32(maxOIDs) // #nosec G115 -- maxOIDs validated to be positive
	}
}

// collectionTask represents a single MIB collection operation.
type collectionTask struct {
	enabled   bool
	name      string
	collector func()
}

// Collect gathers all enabled MIB data from a device.
func (c *SNMPCollector) Collect(ctx context.Context, ip string) (*SNMPFullData, error) {
	if c.config == nil {
		return nil, errors.New("SNMP config is nil")
	}

	data := &SNMPFullData{
		CollectedAt: time.Now(),
	}

	collectCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	tasks := c.buildCollectionTasks(collectCtx, ip, data, &mu)
	c.executeCollectionTasks(&wg, tasks)
	wg.Wait()

	return data, nil
}

// buildCollectionTasks creates the list of MIB collection tasks based on configuration.
func (c *SNMPCollector) buildCollectionTasks(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) []collectionTask {
	return []collectionTask{
		{
			enabled: c.mibConfig.System,
			name:    "system",
			collector: func() {
				c.collectAndStoreSystem(ctx, ip, data, mu)
			},
		},
		{
			enabled: c.mibConfig.Interfaces,
			name:    "interfaces",
			collector: func() {
				c.collectAndStoreInterfaces(ctx, ip, data, mu)
			},
		},
		{
			enabled: c.mibConfig.IPAddresses,
			name:    "ipAddresses",
			collector: func() {
				c.collectAndStoreIPAddresses(ctx, ip, data, mu)
			},
		},
		{
			enabled: c.mibConfig.Bridge,
			name:    "macTable",
			collector: func() {
				c.collectAndStoreMACTable(ctx, ip, data, mu)
			},
		},
		{
			enabled: c.mibConfig.VLAN,
			name:    "vlans",
			collector: func() {
				c.collectAndStoreVLANs(ctx, ip, data, mu)
			},
		},
		{
			enabled: c.mibConfig.Entity,
			name:    "inventory",
			collector: func() {
				c.collectAndStoreInventory(ctx, ip, data, mu)
			},
		},
		{
			enabled: c.mibConfig.LLDP,
			name:    "lldp",
			collector: func() {
				c.collectAndStoreLLDP(ctx, ip, data, mu)
			},
		},
		{
			enabled: c.mibConfig.Routing,
			name:    "routing",
			collector: func() {
				c.collectAndStoreRoutes(ctx, ip, data, mu)
			},
		},
	}
}

// executeCollectionTasks runs all enabled collection tasks concurrently.
func (c *SNMPCollector) executeCollectionTasks(wg *sync.WaitGroup, tasks []collectionTask) {
	for _, task := range tasks {
		if task.enabled {
			wg.Go(task.collector)
		}
	}
}

// collectAndStoreSystem collects system info and stores it in data.
func (c *SNMPCollector) collectAndStoreSystem(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	sysInfo, err := snmp.GetSystemInfo(ctx, ip, c.config)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("system: %v", err))
		return
	}
	data.System = sysInfo
	logging.GetLogger().DebugContext(ctx, "Collected system info", "ip", ip, "sysName", sysInfo.SysName)
}

// collectAndStoreInterfaces collects interfaces and stores them in data.
func (c *SNMPCollector) collectAndStoreInterfaces(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	interfaces, err := c.collectInterfaces(ctx, ip)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("interfaces: %v", err))
		return
	}
	data.Interfaces = interfaces
	logging.GetLogger().DebugContext(ctx, "Collected interfaces", "ip", ip, "count", len(interfaces))
}

// collectAndStoreIPAddresses collects IP addresses and stores them in data.
func (c *SNMPCollector) collectAndStoreIPAddresses(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	ipAddrs, err := c.collectIPAddresses(ctx, ip)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("ipAddresses: %v", err))
		return
	}
	data.IPAddresses = ipAddrs
	logging.GetLogger().DebugContext(ctx, "Collected IP addresses", "ip", ip, "count", len(ipAddrs))
}

// collectAndStoreMACTable collects MAC table and stores it in data.
func (c *SNMPCollector) collectAndStoreMACTable(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	macTable, err := c.collectMACTable(ctx, ip)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("macTable: %v", err))
		return
	}
	data.MACTable = macTable
	logging.GetLogger().DebugContext(ctx, "Collected MAC table", "ip", ip, "count", len(macTable))
}

// collectAndStoreVLANs collects VLANs and stores them in data.
func (c *SNMPCollector) collectAndStoreVLANs(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	vlans, err := c.collectVLANs(ctx, ip)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("vlans: %v", err))
		return
	}
	data.VLANs = vlans
	logging.GetLogger().DebugContext(ctx, "Collected VLANs", "ip", ip, "count", len(vlans))
}

// collectAndStoreInventory collects inventory and stores it in data.
func (c *SNMPCollector) collectAndStoreInventory(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	entities, err := c.collectInventory(ctx, ip)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("inventory: %v", err))
		return
	}
	data.Inventory = entities
	logging.GetLogger().DebugContext(ctx, "Collected inventory", "ip", ip, "count", len(entities))
}

// collectAndStoreLLDP collects LLDP neighbors and stores them in data.
func (c *SNMPCollector) collectAndStoreLLDP(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	neighbors, err := c.collectLLDPNeighbors(ctx, ip)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("lldp: %v", err))
		return
	}
	data.LLDPNeighbors = neighbors
	logging.GetLogger().DebugContext(ctx, "Collected LLDP neighbors", "ip", ip, "count", len(neighbors))
}

// collectAndStoreRoutes collects routes and stores them in data.
func (c *SNMPCollector) collectAndStoreRoutes(
	ctx context.Context,
	ip string,
	data *SNMPFullData,
	mu *sync.Mutex,
) {
	routes, err := c.collectRoutes(ctx, ip)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		data.Errors = append(data.Errors, fmt.Sprintf("routing: %v", err))
		return
	}
	data.Routing = routes
	logging.GetLogger().DebugContext(ctx, "Collected routes", "ip", ip, "count", len(routes))
}

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

// CollectorResult contains the result of SNMP collection for a device.
type CollectorResult struct {
	IP    string        `json:"ip"`
	Data  *SNMPFullData `json:"data,omitempty"`
	Error error         `json:"error,omitempty"`
}

// CollectBatch collects SNMP data from multiple devices concurrently.
//

func (c *SNMPCollector) CollectBatch(
	ctx context.Context,
	ips []string,
	maxConcurrent int,
) []CollectorResult {
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
