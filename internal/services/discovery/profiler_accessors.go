package discovery

// profiler_accessors.go contains the read-only / state-inspection methods on
// DeviceProfiler used by HTTP handlers and tests, plus the small helpers that
// expose profiling status and SNMP data.

import (
	"maps"
)

// GetProfile returns the profile for an IP address.
func (p *DeviceProfiler) GetProfile(ip string) *DeviceProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.profiles[ip]
}

// GetAllProfiles returns all collected profiles.
func (p *DeviceProfiler) GetAllProfiles() map[string]*DeviceProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]*DeviceProfile, len(p.profiles))
	maps.Copy(result, p.profiles)
	return result
}

// ClearProfiles removes all stored profiles.
func (p *DeviceProfiler) ClearProfiles() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.profiles = make(map[string]*DeviceProfile)
}

// IsProfiled returns true if the IP has been profiled.
func (p *DeviceProfiler) IsProfiled(ip string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.profiles[ip]
	return exists
}

// IsProfiling returns true if the IP is currently being profiled.
func (p *DeviceProfiler) IsProfiling(ip string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.profiling[ip]
}

// hasSNMPCredentials returns true if SNMP credentials are configured.
func (p *DeviceProfiler) hasSNMPCredentials() bool {
	if p.snmpConfig == nil {
		return false
	}
	return len(p.snmpConfig.Communities) > 0 || len(p.snmpConfig.V3Credentials) > 0
}

// GetSNMPData returns the full SNMP MIB data for an IP address.
func (p *DeviceProfiler) GetSNMPData(ip string) *SNMPFullData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.snmpData[ip]
}

// GetAllSNMPData returns all collected SNMP MIB data.
func (p *DeviceProfiler) GetAllSNMPData() map[string]*SNMPFullData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]*SNMPFullData, len(p.snmpData))
	maps.Copy(result, p.snmpData)
	return result
}

// ClearSNMPData removes all stored SNMP data.
func (p *DeviceProfiler) ClearSNMPData() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snmpData = make(map[string]*SNMPFullData)
}

// GetProfilingStatus returns comprehensive status about profiling operations.
func (p *DeviceProfiler) GetProfilingStatus() *ProfilingStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profilingIPs := make([]string, 0, len(p.profiling))
	for ip := range p.profiling {
		profilingIPs = append(profilingIPs, ip)
	}

	// Get port count based on intensity
	ports := p.config.GetPortsForIntensity()
	portsCount := len(ports)
	if ports == nil {
		portsCount = len(p.config.QuickPorts) // Fallback to quick ports
	}

	return &ProfilingStatus{
		TotalProfiled: len(p.profiles),
		InProgress:    len(p.profiling),
		QueueLength:   len(p.queue),
		ProfilingIPs:  profilingIPs,
		Enabled:       p.config.Enabled,
		MaxConcurrent: p.config.MaxConcurrent,
		PortsToScan:   portsCount,
		ScanIntensity: string(p.config.PortScanIntensity),
	}
}

// GetDeviceProfilingState returns the profiling state for a specific device IP.
func (p *DeviceProfiler) GetDeviceProfilingState(ip string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.profiling[ip] {
		return "profiling"
	}
	if _, exists := p.profiles[ip]; exists {
		return "completed"
	}
	return "pending"
}
