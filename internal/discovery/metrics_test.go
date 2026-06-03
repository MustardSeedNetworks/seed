package discovery_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewMetrics(t *testing.T) {
	m := discovery.NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}
	if m.ByMethod == nil {
		t.Error("ByMethod map should be initialized")
	}
	if m.ByVendor == nil {
		t.Error("ByVendor map should be initialized")
	}
	if m.CommonPorts == nil {
		t.Error("CommonPorts map should be initialized")
	}
	if m.ServiceTypes == nil {
		t.Error("ServiceTypes map should be initialized")
	}
}

func TestMetrics_Reset(t *testing.T) {
	m := discovery.NewMetrics()

	// Add some data
	m.RecordDevice("arp", "Apple")
	m.RecordProbe(true)
	m.RecordError("icmp")
	m.RecordRetry(true)
	m.RecordProfile(true, time.Second)
	m.RecordOpenPort(22, "ssh")
	m.RecordSNMP(true, "v2c", 5, 3, 2)
	m.RecordSubnetScanned()
	m.RecordMethodAvailable("arp")
	m.RecordMethodFailed("snmp")

	// Reset
	m.Reset()

	// Verify all fields are reset
	if m.TotalDiscovered != 0 {
		t.Errorf("TotalDiscovered should be 0, got %d", m.TotalDiscovered)
	}
	if len(m.ByMethod) != 0 {
		t.Errorf("ByMethod should be empty, got %d", len(m.ByMethod))
	}
	if len(m.ByVendor) != 0 {
		t.Errorf("ByVendor should be empty, got %d", len(m.ByVendor))
	}
	if m.HostsProbed != 0 {
		t.Errorf("HostsProbed should be 0, got %d", m.HostsProbed)
	}
	if m.HostsResponding != 0 {
		t.Errorf("HostsResponding should be 0, got %d", m.HostsResponding)
	}
	if m.ProfilesCompleted != 0 {
		t.Errorf("ProfilesCompleted should be 0, got %d", m.ProfilesCompleted)
	}
	if m.SNMPSuccessful != 0 {
		t.Errorf("SNMPSuccessful should be 0, got %d", m.SNMPSuccessful)
	}
}

func TestMetrics_StartEndScan(t *testing.T) {
	m := discovery.NewMetrics()

	m.StartScan()

	// Check scan start time is set
	if m.LastScanStart.IsZero() {
		t.Error("LastScanStart should be set after StartScan")
	}

	// Simulate some activity
	m.RecordProbe(true)
	m.RecordProbe(true)
	m.RecordProbe(false)

	time.Sleep(10 * time.Millisecond)

	m.EndScan()

	// Check scan duration
	if m.LastScanDuration == 0 {
		t.Error("LastScanDuration should be non-zero after EndScan")
	}
	if m.FastestScan == 0 {
		t.Error("FastestScan should be set after first scan")
	}
	if m.SlowestScan == 0 {
		t.Error("SlowestScan should be set after first scan")
	}

	// Coverage should be calculated
	expectedCoverage := float64(2) / float64(3) * 100
	if m.CoveragePercent != expectedCoverage {
		t.Errorf("CoveragePercent should be %.2f, got %.2f", expectedCoverage, m.CoveragePercent)
	}
}

func TestMetrics_RecordDevice(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordDevice("arp", "Apple")
	m.RecordDevice("lldp", "Cisco")
	m.RecordDevice("arp", "Dell")

	if m.TotalDiscovered != 3 {
		t.Errorf("TotalDiscovered should be 3, got %d", m.TotalDiscovered)
	}
	if m.ByMethod["arp"] != 2 {
		t.Errorf("ByMethod[arp] should be 2, got %d", m.ByMethod["arp"])
	}
	if m.ByMethod["lldp"] != 1 {
		t.Errorf("ByMethod[lldp] should be 1, got %d", m.ByMethod["lldp"])
	}
	if m.ByVendor["Apple"] != 1 {
		t.Errorf("ByVendor[Apple] should be 1, got %d", m.ByVendor["Apple"])
	}

	// Test empty vendor
	m.RecordDevice("arp", "")
	if m.TotalDiscovered != 4 {
		t.Errorf("TotalDiscovered should be 4, got %d", m.TotalDiscovered)
	}
}

func TestMetrics_RecordProbe(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordProbe(true)
	m.RecordProbe(true)
	m.RecordProbe(false)

	if m.HostsProbed != 3 {
		t.Errorf("HostsProbed should be 3, got %d", m.HostsProbed)
	}
	if m.HostsResponding != 2 {
		t.Errorf("HostsResponding should be 2, got %d", m.HostsResponding)
	}
	if m.UnreachableCount != 1 {
		t.Errorf("UnreachableCount should be 1, got %d", m.UnreachableCount)
	}
}

func TestMetrics_RecordError(t *testing.T) {
	m := discovery.NewMetrics()

	categories := []string{"arp", "icmp", "dns", "snmp", "timeout", "unknown"}
	for _, cat := range categories {
		m.RecordError(cat)
	}

	if m.ARPErrors != 1 {
		t.Errorf("ARPErrors should be 1, got %d", m.ARPErrors)
	}
	if m.ICMPErrors != 1 {
		t.Errorf("ICMPErrors should be 1, got %d", m.ICMPErrors)
	}
	if m.DNSErrors != 1 {
		t.Errorf("DNSErrors should be 1, got %d", m.DNSErrors)
	}
	if m.SNMPErrors != 1 {
		t.Errorf("SNMPErrors should be 1, got %d", m.SNMPErrors)
	}
	if m.TimeoutCount != 1 {
		t.Errorf("TimeoutCount should be 1, got %d", m.TimeoutCount)
	}
}

func TestMetrics_RecordRetry(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordRetry(true)
	m.RecordRetry(true)
	m.RecordRetry(false)

	if m.RetryCount != 3 {
		t.Errorf("RetryCount should be 3, got %d", m.RetryCount)
	}
	if m.SuccessRetries != 2 {
		t.Errorf("SuccessRetries should be 2, got %d", m.SuccessRetries)
	}
}

func TestMetrics_RecordProfile(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordProfile(true, 100*time.Millisecond)
	m.RecordProfile(true, 200*time.Millisecond)
	m.RecordProfile(false, 50*time.Millisecond)

	if m.ProfilesCompleted != 2 {
		t.Errorf("ProfilesCompleted should be 2, got %d", m.ProfilesCompleted)
	}
	if m.ProfilesFailed != 1 {
		t.Errorf("ProfilesFailed should be 1, got %d", m.ProfilesFailed)
	}

	// Avg should be (100+200+50)/3 = 116ms
	expectedAvg := int64(116)
	if m.AvgProfileTimeMs != expectedAvg {
		t.Errorf("AvgProfileTimeMs should be %d, got %d", expectedAvg, m.AvgProfileTimeMs)
	}
}

func TestMetrics_RecordOpenPort(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordOpenPort(22, "ssh")
	m.RecordOpenPort(80, "http")
	m.RecordOpenPort(22, "ssh")
	m.RecordOpenPort(443, "")

	if m.OpenPortsFound != 4 {
		t.Errorf("OpenPortsFound should be 4, got %d", m.OpenPortsFound)
	}
	if m.CommonPorts[22] != 2 {
		t.Errorf("CommonPorts[22] should be 2, got %d", m.CommonPorts[22])
	}
	if m.ServiceTypes["ssh"] != 2 {
		t.Errorf("ServiceTypes[ssh] should be 2, got %d", m.ServiceTypes["ssh"])
	}
	if m.ServiceTypes["http"] != 1 {
		t.Errorf("ServiceTypes[http] should be 1, got %d", m.ServiceTypes["http"])
	}
}

func TestMetrics_RecordSNMP(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordSNMP(true, "v2c", 10, 5, 3)
	m.RecordSNMP(true, "v3", 8, 4, 2)
	m.RecordSNMP(false, "", 0, 0, 0)

	if m.SNMPSuccessful != 2 {
		t.Errorf("SNMPSuccessful should be 2, got %d", m.SNMPSuccessful)
	}
	if m.SNMPv2cUsed != 1 {
		t.Errorf("SNMPv2cUsed should be 1, got %d", m.SNMPv2cUsed)
	}
	if m.SNMPv3Used != 1 {
		t.Errorf("SNMPv3Used should be 1, got %d", m.SNMPv3Used)
	}
	if m.InterfacesFound != 18 {
		t.Errorf("InterfacesFound should be 18, got %d", m.InterfacesFound)
	}
	if m.IPAddressesFound != 9 {
		t.Errorf("IPAddressesFound should be 9, got %d", m.IPAddressesFound)
	}
	if m.EntitiesFound != 5 {
		t.Errorf("EntitiesFound should be 5, got %d", m.EntitiesFound)
	}
	if m.MIBsCollected != 2 {
		t.Errorf("MIBsCollected should be 2, got %d", m.MIBsCollected)
	}
}

func TestMetrics_RecordSubnetScanned(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordSubnetScanned()
	m.RecordSubnetScanned()

	if m.SubnetsScanned != 2 {
		t.Errorf("SubnetsScanned should be 2, got %d", m.SubnetsScanned)
	}
}

func TestMetrics_RecordMethodAvailable(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordMethodAvailable("arp")
	m.RecordMethodAvailable("icmp")
	m.RecordMethodAvailable("arp") // Duplicate should be ignored

	if len(m.MethodsAvailable) != 2 {
		t.Errorf("MethodsAvailable should have 2 entries, got %d", len(m.MethodsAvailable))
	}
}

func TestMetrics_RecordMethodFailed(t *testing.T) {
	m := discovery.NewMetrics()

	m.RecordMethodFailed("snmp")
	m.RecordMethodFailed("lldp")
	m.RecordMethodFailed("snmp") // Duplicate should be ignored

	if len(m.MethodsFailed) != 2 {
		t.Errorf("MethodsFailed should have 2 entries, got %d", len(m.MethodsFailed))
	}
}

func TestMetrics_GetSnapshot(t *testing.T) {
	m := discovery.NewMetrics()

	// Add some data
	m.RecordDevice("arp", "Apple")
	m.RecordProbe(true)
	m.RecordMethodAvailable("arp")
	m.RecordMethodFailed("snmp")

	snapshot := m.GetSnapshot()

	// Verify it's a copy
	if snapshot == m {
		t.Error("GetSnapshot should return a copy, not the same instance")
	}

	// Verify values are copied
	if snapshot.TotalDiscovered != m.TotalDiscovered {
		t.Errorf("TotalDiscovered mismatch: %d vs %d", snapshot.TotalDiscovered, m.TotalDiscovered)
	}
	if len(snapshot.ByMethod) != len(m.ByMethod) {
		t.Errorf("ByMethod length mismatch: %d vs %d", len(snapshot.ByMethod), len(m.ByMethod))
	}
	if len(snapshot.MethodsAvailable) != len(m.MethodsAvailable) {
		t.Errorf("MethodsAvailable length mismatch: %d vs %d", len(snapshot.MethodsAvailable), len(m.MethodsAvailable))
	}

	// Modify original - snapshot should not change
	m.RecordDevice("lldp", "Cisco")
	if snapshot.TotalDiscovered == m.TotalDiscovered {
		t.Error("Snapshot should be independent of original")
	}
}

func TestMetrics_Clone(t *testing.T) {
	m := discovery.NewMetrics()
	m.RecordDevice("arp", "Apple")

	clone := m.Clone()
	if clone.TotalDiscovered != m.TotalDiscovered {
		t.Error("Clone should have same values as original")
	}
}

func TestMetrics_UpdateFromDevices(t *testing.T) {
	m := discovery.NewMetrics()

	devices := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", Vendor: "Apple"},
		{MAC: "00:11:22:33:44:56", Vendor: "Dell"},
		{MAC: "00:11:22:33:44:57", Vendor: "Apple"},
		{
			MAC:    "00:11:22:33:44:58",
			Vendor: "",
			Profile: &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{
					{Port: 22, Service: "ssh"},
					{Port: 80, Service: "http"},
				},
			},
		},
	}

	m.UpdateFromDevices(devices)

	if m.TotalDiscovered != 4 {
		t.Errorf("TotalDiscovered should be 4, got %d", m.TotalDiscovered)
	}
	if m.ByVendor["Apple"] != 2 {
		t.Errorf("ByVendor[Apple] should be 2, got %d", m.ByVendor["Apple"])
	}
	if m.ByVendor["Dell"] != 1 {
		t.Errorf("ByVendor[Dell] should be 1, got %d", m.ByVendor["Dell"])
	}
	if m.CommonPorts[22] != 1 {
		t.Errorf("CommonPorts[22] should be 1, got %d", m.CommonPorts[22])
	}
	if m.ServiceTypes["ssh"] != 1 {
		t.Errorf("ServiceTypes[ssh] should be 1, got %d", m.ServiceTypes["ssh"])
	}
}

func TestMetrics_GetDegradationStatus_Healthy(t *testing.T) {
	m := discovery.NewMetrics()
	m.RecordMethodAvailable("arp")
	m.RecordMethodAvailable("icmp")
	m.RecordProbe(true)
	m.RecordProbe(true)
	m.RecordProbe(true)

	status := m.GetDegradationStatus()

	if status.OverallHealth != "healthy" {
		t.Errorf("OverallHealth should be 'healthy', got %q", status.OverallHealth)
	}
	if len(status.HealthyMethods) != 2 {
		t.Errorf("HealthyMethods should have 2 entries, got %d", len(status.HealthyMethods))
	}
	if len(status.FailedMethods) != 0 {
		t.Errorf("FailedMethods should be empty, got %d", len(status.FailedMethods))
	}
}

func TestMetrics_GetDegradationStatus_Degraded(t *testing.T) {
	m := discovery.NewMetrics()

	// Record available methods first - this is important so we don't go critical
	m.RecordMethodAvailable("arp")
	m.RecordMethodAvailable("icmp")
	m.RecordMethodAvailable("snmp")

	// Add some healthy probes to avoid zero division
	for range 10 {
		m.RecordProbe(true)
	}

	// Add errors (> 20% but < 50% to trigger degraded not critical)
	// With 10 probes, we need > 2 errors for > 20% but <= 5 for <= 50%
	for range 3 {
		m.RecordError("icmp")
	}

	m.RecordMethodFailed("snmp")

	status := m.GetDegradationStatus()

	if status.OverallHealth != "degraded" {
		t.Errorf("OverallHealth should be 'degraded', got %q", status.OverallHealth)
	}
	if len(status.FailedMethods) != 1 {
		t.Errorf("FailedMethods should have 1 entry, got %d", len(status.FailedMethods))
	}
	if len(status.Recommendations) == 0 {
		t.Error("Should have recommendations for failed methods")
	}
}

func TestMetrics_GetDegradationStatus_Critical(t *testing.T) {
	m := discovery.NewMetrics()

	// Add probes
	for range 10 {
		m.RecordProbe(true)
		m.RecordProfile(true, time.Millisecond)
	}

	// Add critical level errors (> 50%)
	for range 15 {
		m.RecordError("icmp")
	}

	status := m.GetDegradationStatus()

	if status.OverallHealth != "critical" {
		t.Errorf("OverallHealth should be 'critical', got %q", status.OverallHealth)
	}
	if len(status.Warnings) == 0 {
		t.Error("Should have warnings for critical state")
	}
}

func TestMetrics_GetDegradationStatus_AllMethodsFailed(t *testing.T) {
	m := discovery.NewMetrics()
	m.RecordMethodFailed("arp")
	m.RecordMethodFailed("icmp")
	m.RecordMethodFailed("snmp")
	m.RecordProbe(true) // Need at least some activity

	status := m.GetDegradationStatus()

	if status.OverallHealth != "critical" {
		t.Errorf("OverallHealth should be 'critical' when all methods failed, got %q", status.OverallHealth)
	}
}

func TestMetrics_GetDegradationStatus_LowCoverage(t *testing.T) {
	m := discovery.NewMetrics()

	// Simulate low coverage (< 50% with > 10 hosts)
	for range 15 {
		m.RecordProbe(false) // Only non-responding
	}
	for range 3 {
		m.RecordProbe(true) // Few responding (20%)
	}

	m.EndScan() // Calculate coverage

	status := m.GetDegradationStatus()

	// Should be degraded due to low coverage
	if status.OverallHealth == "healthy" {
		t.Error("OverallHealth should not be 'healthy' with low coverage")
	}

	// Check for coverage warning
	hasWarning := false
	for _, w := range status.Warnings {
		if contains(w, "coverage") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("Should have warning about low coverage")
	}
}

func TestComputeDelta_NewDevices(t *testing.T) {
	prev := []*discovery.DiscoveredDevice{}
	curr := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10"},
		{MAC: "00:11:22:33:44:56", IP: "192.168.1.11"},
	}

	delta := discovery.ComputeDelta(prev, curr)

	if len(delta.NewDevices) != 2 {
		t.Errorf("NewDevices should have 2 entries, got %d", len(delta.NewDevices))
	}
	if len(delta.RemovedDevices) != 0 {
		t.Errorf("RemovedDevices should be empty, got %d", len(delta.RemovedDevices))
	}
	if len(delta.UpdatedDevices) != 0 {
		t.Errorf("UpdatedDevices should be empty, got %d", len(delta.UpdatedDevices))
	}
}

func TestComputeDelta_RemovedDevices(t *testing.T) {
	prev := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10"},
		{MAC: "00:11:22:33:44:56", IP: "192.168.1.11"},
	}
	curr := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10"},
	}

	delta := discovery.ComputeDelta(prev, curr)

	if len(delta.NewDevices) != 0 {
		t.Errorf("NewDevices should be empty, got %d", len(delta.NewDevices))
	}
	if len(delta.RemovedDevices) != 1 {
		t.Errorf("RemovedDevices should have 1 entry, got %d", len(delta.RemovedDevices))
	}
	if delta.RemovedDevices[0].MAC != "00:11:22:33:44:56" {
		t.Errorf("Removed device should be 00:11:22:33:44:56, got %s", delta.RemovedDevices[0].MAC)
	}
}

func TestComputeDelta_UpdatedDevices(t *testing.T) {
	prev := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10", Hostname: "host1"},
	}
	curr := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10", Hostname: "host1-new"},
	}

	delta := discovery.ComputeDelta(prev, curr)

	if len(delta.UpdatedDevices) != 1 {
		t.Errorf("UpdatedDevices should have 1 entry, got %d", len(delta.UpdatedDevices))
	}
	if len(delta.NewDevices) != 0 {
		t.Errorf("NewDevices should be empty, got %d", len(delta.NewDevices))
	}
}

func TestComputeDelta_IPChange(t *testing.T) {
	prev := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10"},
	}
	curr := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.20"}, // IP changed
	}

	delta := discovery.ComputeDelta(prev, curr)

	if len(delta.UpdatedDevices) != 1 {
		t.Errorf("UpdatedDevices should have 1 entry for IP change, got %d", len(delta.UpdatedDevices))
	}
}

func TestComputeDelta_VendorChange(t *testing.T) {
	prev := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", Vendor: "Unknown"},
	}
	curr := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", Vendor: "Apple"}, // Vendor updated
	}

	delta := discovery.ComputeDelta(prev, curr)

	if len(delta.UpdatedDevices) != 1 {
		t.Errorf("UpdatedDevices should have 1 entry for vendor change, got %d", len(delta.UpdatedDevices))
	}
}

func TestComputeDelta_NoChanges(t *testing.T) {
	prev := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10", Hostname: "host1"},
	}
	curr := []*discovery.DiscoveredDevice{
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.10", Hostname: "host1"},
	}

	delta := discovery.ComputeDelta(prev, curr)

	if len(delta.NewDevices) != 0 {
		t.Errorf("NewDevices should be empty, got %d", len(delta.NewDevices))
	}
	if len(delta.RemovedDevices) != 0 {
		t.Errorf("RemovedDevices should be empty, got %d", len(delta.RemovedDevices))
	}
	if len(delta.UpdatedDevices) != 0 {
		t.Errorf("UpdatedDevices should be empty, got %d", len(delta.UpdatedDevices))
	}
}

func TestComputeDelta_NoMACDevicesExcluded(t *testing.T) {
	// Devices without MAC (ICMP-only) should be excluded from delta
	prev := []*discovery.DiscoveredDevice{
		{MAC: "", IP: "192.168.1.10"}, // No MAC - should be excluded
	}
	curr := []*discovery.DiscoveredDevice{
		{MAC: "", IP: "192.168.1.10"},
		{MAC: "00:11:22:33:44:55", IP: "192.168.1.11"},
	}

	delta := discovery.ComputeDelta(prev, curr)

	// Only the device with MAC should show as new
	if len(delta.NewDevices) != 1 {
		t.Errorf("NewDevices should have 1 entry (only devices with MAC), got %d", len(delta.NewDevices))
	}
}

func TestComputeDelta_ProfileChange(t *testing.T) {
	prev := []*discovery.DiscoveredDevice{
		{
			MAC: "00:11:22:33:44:55",
			Profile: &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{{Port: 22}},
			},
		},
	}
	curr := []*discovery.DiscoveredDevice{
		{
			MAC: "00:11:22:33:44:55",
			Profile: &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{{Port: 22}, {Port: 80}}, // Added port
			},
		},
	}

	delta := discovery.ComputeDelta(prev, curr)

	if len(delta.UpdatedDevices) != 1 {
		t.Errorf("UpdatedDevices should have 1 entry for profile change, got %d", len(delta.UpdatedDevices))
	}
}

// Helper function for string contains check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
