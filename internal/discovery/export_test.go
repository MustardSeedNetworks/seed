// Package discovery exports internal types and functions for testing.
// This file is only compiled during testing (due to _test.go suffix)
// and provides access to internal implementation details.
package discovery

import (
	"net"
	"sync"
)

// ExportIncrementIP exposes incrementIP for testing.
func ExportIncrementIP(ip net.IP, n int) net.IP {
	return incrementIP(ip, n)
}

// ExportNormalizeMac exposes normalizeMac for testing.
func ExportNormalizeMac(mac string) string {
	return normalizeMac(mac)
}

// ExportGuessOSFromTTL exposes guessOSFromTTL for testing.
func ExportGuessOSFromTTL(ttl int) string {
	return guessOSFromTTL(ttl)
}

// ExportSplitSubnetIntoChunks exposes splitSubnetIntoChunks for testing.
func ExportSplitSubnetIntoChunks(subnet *net.IPNet, maxChunks int) []*net.IPNet {
	return splitSubnetIntoChunks(subnet, maxChunks)
}

// Export NVD rate limit constants for testing.
const (
	NVDRateLimitNoKey   = nvdRateLimitNoKey
	NVDRateLimitWithKey = nvdRateLimitWithKey
)

// ARPScannerTestAccessor provides access to ARPScanner's private fields for testing.
type ARPScannerTestAccessor struct {
	Scanner *ARPScanner
}

// GetInterfaceName returns the scanner's interface name.
func (a *ARPScannerTestAccessor) GetInterfaceName() string {
	return a.Scanner.interfaceName
}

// GetOUI returns the scanner's OUI database.
func (a *ARPScannerTestAccessor) GetOUI() *OUIDatabase {
	return a.Scanner.oui
}

// GetEntries returns the scanner's entries map.
func (a *ARPScannerTestAccessor) GetEntries() map[string]*ARPEntry {
	return a.Scanner.entries
}

// GetSubnet returns the scanner's subnet.
func (a *ARPScannerTestAccessor) GetSubnet() *net.IPNet {
	return a.Scanner.subnet
}

// SetSubnet sets the scanner's subnet.
func (a *ARPScannerTestAccessor) SetSubnet(subnet *net.IPNet) {
	a.Scanner.subnet = subnet
}

// GetLocalIP returns the scanner's local IP.
func (a *ARPScannerTestAccessor) GetLocalIP() net.IP {
	return a.Scanner.localIP
}

// IsScanning returns the scanner's scanning state.
func (a *ARPScannerTestAccessor) IsScanning() bool {
	return a.Scanner.scanning
}

// SetScanning sets the scanner's scanning state.
func (a *ARPScannerTestAccessor) SetScanning(scanning bool) {
	a.Scanner.scanning = scanning
}

// Lock locks the scanner's mutex.
func (a *ARPScannerTestAccessor) Lock() {
	a.Scanner.mu.Lock()
}

// Unlock unlocks the scanner's mutex.
func (a *ARPScannerTestAccessor) Unlock() {
	a.Scanner.mu.Unlock()
}

// IsInSubnet exposes the private isInSubnet method for testing.
func (s *ARPScanner) IsInSubnet(ip string) bool {
	return s.isInSubnet(ip)
}

// TracerTestAccessor provides access to Tracer's private fields for testing.
type TracerTestAccessor struct {
	Tracer *Tracer
}

// GetTimeout returns the tracer's timeout.
func (t *TracerTestAccessor) GetTimeout() any {
	return t.Tracer.timeout
}

// GetMaxHops returns the tracer's maxHops.
func (t *TracerTestAccessor) GetMaxHops() int {
	return t.Tracer.maxHops
}

// VulnerabilityScannerTestAccessor provides access to VulnerabilityScanner's private fields for testing.
type VulnerabilityScannerTestAccessor struct {
	Scanner *VulnerabilityScanner
}

// GetMu returns the scanner's mutex for testing.
func (v *VulnerabilityScannerTestAccessor) GetMu() *sync.RWMutex {
	return &v.Scanner.mu
}

// GetDeviceResults returns the scanner's deviceResults map.
func (v *VulnerabilityScannerTestAccessor) GetDeviceResults() map[string]*DeviceVulnerabilities {
	return v.Scanner.deviceResults
}

// SetDeviceResults sets the scanner's deviceResults map.
func (v *VulnerabilityScannerTestAccessor) SetDeviceResults(
	results map[string]*DeviceVulnerabilities,
) {
	v.Scanner.deviceResults = results
}

// GetConfig returns the scanner's config.
func (v *VulnerabilityScannerTestAccessor) GetConfig() *VulnerabilityScannerConfig {
	return v.Scanner.config
}

// FilterBySeverity exposes the private filterBySeverity method.
func (v *VulnerabilityScannerTestAccessor) FilterBySeverity(vulns []Vulnerability) []Vulnerability {
	return v.Scanner.filterBySeverity(vulns)
}

// NVDProviderTestAccessor provides access to NVDProvider's private fields for testing.
type NVDProviderTestAccessor struct {
	Provider *NVDProvider
}

// GetAPIKey returns the provider's API key.
func (n *NVDProviderTestAccessor) GetAPIKey() string {
	return n.Provider.apiKey
}

// GetRateLimit returns the provider's rate limit.
func (n *NVDProviderTestAccessor) GetRateLimit() int {
	return n.Provider.rateLimit
}

// DeviceDiscoveryTestAccessor provides access to DeviceDiscovery's private fields for testing.
type DeviceDiscoveryTestAccessor struct {
	Discovery *DeviceDiscovery
}

// GetProtoManager returns the discovery's protocol manager.
func (d *DeviceDiscoveryTestAccessor) GetProtoManager() *Manager {
	return d.Discovery.protoManager
}
