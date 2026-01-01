// Package discovery provides network device discovery functionality.
// This file implements discovery metrics and coverage tracking.
package discovery

import (
	"maps"
	"slices"
	"sync"
	"time"
)

// Discovery method constants.
const (
	MethodICMP = "icmp"
)

// Health status constants.
const (
	HealthCritical = "critical"
	HealthDegraded = "degraded"
)

// Metrics tracks discovery statistics and coverage.
type Metrics struct {
	mu sync.RWMutex

	// Enumeration metrics.
	TotalDiscovered int            `json:"totalDiscovered"` // Total unique devices found
	ByMethod        map[string]int `json:"byMethod"`        // Devices found per method (arp, icmp, lldp, etc.)
	ByVendor        map[string]int `json:"byVendor"`        // Devices by OUI vendor

	// Coverage metrics.
	SubnetsScanned   int     `json:"subnetsScanned"`   // Number of subnets scanned
	HostsProbed      int     `json:"hostsProbed"`      // Total IPs probed
	HostsResponding  int     `json:"hostsResponding"`  // IPs that responded
	CoveragePercent  float64 `json:"coveragePercent"`  // Responding / probed * 100
	UnreachableCount int     `json:"unreachableCount"` // IPs that didn't respond

	// Error metrics.
	ARPErrors      int `json:"arpErrors"`
	ICMPErrors     int `json:"icmpErrors"`
	DNSErrors      int `json:"dnsErrors"`
	SNMPErrors     int `json:"snmpErrors"`
	TimeoutCount   int `json:"timeoutCount"`
	RetryCount     int `json:"retryCount"`
	SuccessRetries int `json:"successRetries"` // Retries that eventually succeeded

	// Profiling metrics.
	ProfilesCompleted int           `json:"profilesCompleted"`
	ProfilesFailed    int           `json:"profilesFailed"`
	AvgProfileTimeMs  int64         `json:"avgProfileTimeMs"`
	TotalProfileTime  time.Duration `json:"-"` // Internal tracking

	// Port scanning metrics.
	PortScansCompleted int            `json:"portScansCompleted"`
	OpenPortsFound     int            `json:"openPortsFound"`
	CommonPorts        map[int]int    `json:"commonPorts"`  // Port -> count of devices with that port open
	ServiceTypes       map[string]int `json:"serviceTypes"` // Service -> count

	// SNMP metrics.
	SNMPSuccessful   int `json:"snmpSuccessful"`
	SNMPv2cUsed      int `json:"snmpv2cUsed"`
	SNMPv3Used       int `json:"snmpv3Used"`
	MIBsCollected    int `json:"mibsCollected"`
	InterfacesFound  int `json:"interfacesFound"`
	IPAddressesFound int `json:"ipAddressesFound"`
	EntitiesFound    int `json:"entitiesFound"` // Physical entities from ENTITY-MIB

	// Timing metrics.
	LastScanStart    time.Time     `json:"lastScanStart"`
	LastScanDuration time.Duration `json:"lastScanDuration"`
	FastestScan      time.Duration `json:"fastestScan"`
	SlowestScan      time.Duration `json:"slowestScan"`

	// Degradation tracking.
	MethodsAvailable []string `json:"methodsAvailable"` // Methods that are working
	MethodsFailed    []string `json:"methodsFailed"`    // Methods that failed completely
}

// NewMetrics creates a new metrics instance with initialized maps.
func NewMetrics() *Metrics {
	return &Metrics{
		ByMethod:     make(map[string]int),
		ByVendor:     make(map[string]int),
		CommonPorts:  make(map[int]int),
		ServiceTypes: make(map[string]int),
	}
}

// Reset clears all metrics for a new scan.
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalDiscovered = 0
	m.ByMethod = make(map[string]int)
	m.ByVendor = make(map[string]int)
	m.SubnetsScanned = 0
	m.HostsProbed = 0
	m.HostsResponding = 0
	m.CoveragePercent = 0
	m.UnreachableCount = 0
	m.ARPErrors = 0
	m.ICMPErrors = 0
	m.DNSErrors = 0
	m.SNMPErrors = 0
	m.TimeoutCount = 0
	m.RetryCount = 0
	m.SuccessRetries = 0
	m.ProfilesCompleted = 0
	m.ProfilesFailed = 0
	m.AvgProfileTimeMs = 0
	m.TotalProfileTime = 0
	m.PortScansCompleted = 0
	m.OpenPortsFound = 0
	m.CommonPorts = make(map[int]int)
	m.ServiceTypes = make(map[string]int)
	m.SNMPSuccessful = 0
	m.SNMPv2cUsed = 0
	m.SNMPv3Used = 0
	m.MIBsCollected = 0
	m.InterfacesFound = 0
	m.IPAddressesFound = 0
	m.EntitiesFound = 0
	m.MethodsAvailable = nil
	m.MethodsFailed = nil
}

// StartScan records the start of a new scan.
func (m *Metrics) StartScan() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastScanStart = time.Now()
}

// EndScan records the end of a scan and updates timing metrics.
func (m *Metrics) EndScan() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LastScanDuration = time.Since(m.LastScanStart)
	if m.FastestScan == 0 || m.LastScanDuration < m.FastestScan {
		m.FastestScan = m.LastScanDuration
	}
	if m.LastScanDuration > m.SlowestScan {
		m.SlowestScan = m.LastScanDuration
	}

	// Calculate coverage.
	if m.HostsProbed > 0 {
		m.CoveragePercent = float64(m.HostsResponding) / float64(m.HostsProbed) * 100
	}
}

// RecordDevice records a discovered device.
func (m *Metrics) RecordDevice(method, vendor string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalDiscovered++
	m.ByMethod[method]++
	if vendor != "" {
		m.ByVendor[vendor]++
	}
}

// RecordProbe records a host probe attempt and result.
func (m *Metrics) RecordProbe(responded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HostsProbed++
	if responded {
		m.HostsResponding++
	} else {
		m.UnreachableCount++
	}
}

// RecordError records an error by category.
func (m *Metrics) RecordError(category string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch category {
	case "arp":
		m.ARPErrors++
	case MethodICMP:
		m.ICMPErrors++
	case "dns":
		m.DNSErrors++
	case "snmp":
		m.SNMPErrors++
	case "timeout":
		m.TimeoutCount++
	}
}

// RecordRetry records a retry attempt and whether it succeeded.
func (m *Metrics) RecordRetry(succeeded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RetryCount++
	if succeeded {
		m.SuccessRetries++
	}
}

// RecordProfile records a profiling operation.
func (m *Metrics) RecordProfile(succeeded bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if succeeded {
		m.ProfilesCompleted++
	} else {
		m.ProfilesFailed++
	}
	m.TotalProfileTime += duration

	total := m.ProfilesCompleted + m.ProfilesFailed
	if total > 0 {
		m.AvgProfileTimeMs = m.TotalProfileTime.Milliseconds() / int64(total)
	}
}

// RecordOpenPort records an open port found on a device.
func (m *Metrics) RecordOpenPort(port int, service string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.OpenPortsFound++
	m.CommonPorts[port]++
	if service != "" {
		m.ServiceTypes[service]++
	}
}

// RecordSNMP records an SNMP collection result.
func (m *Metrics) RecordSNMP(succeeded bool, version string, interfaces, ipAddrs, entities int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if succeeded {
		m.SNMPSuccessful++
		switch version {
		case "v2c":
			m.SNMPv2cUsed++
		case "v3":
			m.SNMPv3Used++
		}
		m.MIBsCollected++
		m.InterfacesFound += interfaces
		m.IPAddressesFound += ipAddrs
		m.EntitiesFound += entities
	}
}

// RecordSubnetScanned records a subnet scan completion.
func (m *Metrics) RecordSubnetScanned() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SubnetsScanned++
}

// RecordMethodAvailable records a working discovery method.
func (m *Metrics) RecordMethodAvailable(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if slices.Contains(m.MethodsAvailable, method) {
		return
	}
	m.MethodsAvailable = append(m.MethodsAvailable, method)
}

// RecordMethodFailed records a failed discovery method.
func (m *Metrics) RecordMethodFailed(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if slices.Contains(m.MethodsFailed, method) {
		return
	}
	m.MethodsFailed = append(m.MethodsFailed, method)
}

// Clone returns a copy of the current metrics (alias for GetSnapshot).
func (m *Metrics) Clone() *Metrics {
	return m.GetSnapshot()
}

// UpdateFromDevices updates metrics based on discovered devices.
func (m *Metrics) UpdateFromDevices(devices []*DiscoveredDevice) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalDiscovered = len(devices)

	// Reset and rebuild vendor counts
	m.ByVendor = make(map[string]int)
	for _, d := range devices {
		if d.Vendor != "" {
			m.ByVendor[d.Vendor]++
		}

		// Count open ports from profiles
		if d.Profile != nil {
			for _, port := range d.Profile.OpenPorts {
				m.CommonPorts[port.Port]++
				if port.Service != "" {
					m.ServiceTypes[port.Service]++
				}
			}
		}
	}
}

// GetSnapshot returns a copy of the current metrics.
func (m *Metrics) GetSnapshot() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create deep copy of maps.
	byMethod := make(map[string]int, len(m.ByMethod))
	maps.Copy(byMethod, m.ByMethod)

	byVendor := make(map[string]int, len(m.ByVendor))
	maps.Copy(byVendor, m.ByVendor)

	commonPorts := make(map[int]int, len(m.CommonPorts))
	maps.Copy(commonPorts, m.CommonPorts)

	serviceTypes := make(map[string]int, len(m.ServiceTypes))
	maps.Copy(serviceTypes, m.ServiceTypes)

	// Copy slices.
	methodsAvailable := make([]string, len(m.MethodsAvailable))
	copy(methodsAvailable, m.MethodsAvailable)

	methodsFailed := make([]string, len(m.MethodsFailed))
	copy(methodsFailed, m.MethodsFailed)

	return &Metrics{
		TotalDiscovered:    m.TotalDiscovered,
		ByMethod:           byMethod,
		ByVendor:           byVendor,
		SubnetsScanned:     m.SubnetsScanned,
		HostsProbed:        m.HostsProbed,
		HostsResponding:    m.HostsResponding,
		CoveragePercent:    m.CoveragePercent,
		UnreachableCount:   m.UnreachableCount,
		ARPErrors:          m.ARPErrors,
		ICMPErrors:         m.ICMPErrors,
		DNSErrors:          m.DNSErrors,
		SNMPErrors:         m.SNMPErrors,
		TimeoutCount:       m.TimeoutCount,
		RetryCount:         m.RetryCount,
		SuccessRetries:     m.SuccessRetries,
		ProfilesCompleted:  m.ProfilesCompleted,
		ProfilesFailed:     m.ProfilesFailed,
		AvgProfileTimeMs:   m.AvgProfileTimeMs,
		PortScansCompleted: m.PortScansCompleted,
		OpenPortsFound:     m.OpenPortsFound,
		CommonPorts:        commonPorts,
		ServiceTypes:       serviceTypes,
		SNMPSuccessful:     m.SNMPSuccessful,
		SNMPv2cUsed:        m.SNMPv2cUsed,
		SNMPv3Used:         m.SNMPv3Used,
		MIBsCollected:      m.MIBsCollected,
		InterfacesFound:    m.InterfacesFound,
		IPAddressesFound:   m.IPAddressesFound,
		EntitiesFound:      m.EntitiesFound,
		LastScanStart:      m.LastScanStart,
		LastScanDuration:   m.LastScanDuration,
		FastestScan:        m.FastestScan,
		SlowestScan:        m.SlowestScan,
		MethodsAvailable:   methodsAvailable,
		MethodsFailed:      methodsFailed,
	}
}

// ScanDelta represents changes between discovery scans.
type ScanDelta struct {
	NewDevices     []*DiscoveredDevice `json:"newDevices"`
	UpdatedDevices []*DiscoveredDevice `json:"updatedDevices"`
	RemovedDevices []*DiscoveredDevice `json:"removedDevices"`
	ScanTime       time.Time           `json:"scanTime"`
}

// ComputeDelta computes changes between previous and current device lists.
//
// NOTE: Delta tracking uses MAC address as the unique device identifier.
// Devices discovered via ICMP-only (no ARP response = no MAC) are excluded
// from delta computation because they lack a stable identifier.
// This is intentional: ICMP-only devices typically indicate:
//   - Remote hosts beyond local subnet (L3 routing)
//   - Hosts with firewall rules blocking ARP
//   - Temporary/transient connections
//
// IP addresses alone are not used for tracking because they can be
// reassigned via DHCP, causing false positives in new/removed counts.
func ComputeDelta(previous, current []*DiscoveredDevice) *ScanDelta {
	delta := &ScanDelta{
		NewDevices:     make([]*DiscoveredDevice, 0),
		UpdatedDevices: make([]*DiscoveredDevice, 0),
		RemovedDevices: make([]*DiscoveredDevice, 0),
		ScanTime:       time.Now(),
	}

	// Build maps for quick lookup by MAC.
	// Devices without MAC (ICMP-only) are excluded from delta tracking.
	prevMap := make(map[string]*DiscoveredDevice)
	for _, d := range previous {
		if d.MAC != "" {
			prevMap[d.MAC] = d
		}
	}

	currMap := make(map[string]*DiscoveredDevice)
	for _, d := range current {
		if d.MAC != "" {
			currMap[d.MAC] = d
		}
	}

	// Find new and updated devices.
	for mac, curr := range currMap {
		prev, existed := prevMap[mac]
		if !existed {
			delta.NewDevices = append(delta.NewDevices, curr)
		} else if deviceChanged(prev, curr) {
			delta.UpdatedDevices = append(delta.UpdatedDevices, curr)
		}
	}

	// Find removed devices.
	for mac, prev := range prevMap {
		if _, exists := currMap[mac]; !exists {
			delta.RemovedDevices = append(delta.RemovedDevices, prev)
		}
	}

	return delta
}

// deviceChanged checks if a device has meaningful changes.
func deviceChanged(prev, curr *DiscoveredDevice) bool {
	// Check for IP change.
	if prev.IP != curr.IP {
		return true
	}

	// Check for hostname change.
	if prev.Hostname != curr.Hostname {
		return true
	}

	// Check for vendor change (rare but possible with OUI updates).
	if prev.Vendor != curr.Vendor {
		return true
	}

	// Check for open ports change (if profiles available).
	if prev.Profile != nil && curr.Profile != nil {
		if len(prev.Profile.OpenPorts) != len(curr.Profile.OpenPorts) {
			return true
		}
	}

	return false
}

// DegradationStatus represents the health of discovery methods.
type DegradationStatus struct {
	OverallHealth   string   `json:"overallHealth"` // healthy, degraded, critical
	HealthyMethods  []string `json:"healthyMethods"`
	FailedMethods   []string `json:"failedMethods"`
	Warnings        []string `json:"warnings"`
	Recommendations []string `json:"recommendations"`
}

// GetDegradationStatus analyzes metrics and returns health status.
func (m *Metrics) GetDegradationStatus() *DegradationStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := &DegradationStatus{
		OverallHealth:   "healthy",
		HealthyMethods:  make([]string, 0),
		FailedMethods:   make([]string, 0),
		Warnings:        make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Copy method lists.
	status.HealthyMethods = append(status.HealthyMethods, m.MethodsAvailable...)
	status.FailedMethods = append(status.FailedMethods, m.MethodsFailed...)

	// Analyze error rates.
	totalErrors := m.ARPErrors + m.ICMPErrors + m.DNSErrors + m.SNMPErrors
	totalOps := m.HostsProbed + m.ProfilesCompleted

	if totalOps > 0 {
		errorRate := float64(totalErrors) / float64(totalOps) * 100
		if errorRate > 50 {
			status.OverallHealth = HealthCritical
			status.Warnings = append(status.Warnings, "High error rate detected (>50%)")
		} else if errorRate > 20 {
			status.OverallHealth = HealthDegraded
			status.Warnings = append(status.Warnings, "Elevated error rate detected (>20%)")
		}
	}

	// Check coverage.
	if m.CoveragePercent < 50 && m.HostsProbed > 10 {
		if status.OverallHealth == "healthy" {
			status.OverallHealth = HealthDegraded
		}
		status.Warnings = append(status.Warnings, "Low network coverage (<50% of hosts responding)")
		status.Recommendations = append(status.Recommendations, "Check firewall rules for ICMP")
	}

	// Check for method failures.
	if len(m.MethodsFailed) > 0 {
		if len(m.MethodsAvailable) == 0 {
			status.OverallHealth = HealthCritical
			status.Warnings = append(status.Warnings, "All discovery methods failed")
		} else if status.OverallHealth == "healthy" {
			status.OverallHealth = HealthDegraded
		}
	}

	// Add recommendations based on failures.
	for _, method := range m.MethodsFailed {
		switch method {
		case "arp":
			status.Recommendations = append(status.Recommendations,
				"ARP scanning requires same broadcast domain - check interface configuration")
		case MethodICMP:
			status.Recommendations = append(status.Recommendations,
				"ICMP requires CAP_NET_RAW - check permissions")
		case "snmp":
			status.Recommendations = append(status.Recommendations,
				"SNMP collection failed - verify community strings and network access")
		}
	}

	return status
}
