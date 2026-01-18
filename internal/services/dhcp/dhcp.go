package dhcp

import (
	"context"
	"net"
	"sync"
	"time"
)

// Status represents the status of a DHCP operation.
type Status string

// DHCP operation status constants.
const (
	StatusSuccess Status = "success"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
)

// DHCP timeout bounds for validation.
const (
	MinDHCPTimeout = 1 * time.Second
	MaxDHCPTimeout = 60 * time.Second
)

// Default threshold values for DHCP response times.
const (
	DefaultWarningThresholdMs  = 500  // milliseconds - response times above this trigger warning
	DefaultCriticalThresholdMs = 2000 // milliseconds - response times above this trigger critical
)

// DefaultTestTimeout is the default timeout for DHCP tests.
const DefaultTestTimeout = 10 * time.Second

// microsecondsPerMillisecond is used for time conversion.
const microsecondsPerMillisecond = 1000.0

// ipv4Len is the length of an IPv4 address in bytes.
const ipv4Len = 4

// bitsPerByte is the number of bits in a byte (for shift operations).
const bitsPerByte = 8

// ValidateDHCPTimeout validates that a DHCP timeout is within acceptable bounds.
func ValidateDHCPTimeout(timeout time.Duration) error {
	if timeout < MinDHCPTimeout || timeout > MaxDHCPTimeout {
		return &TimeoutError{timeout, MinDHCPTimeout, MaxDHCPTimeout}
	}
	return nil
}

// TimeoutError represents an invalid timeout configuration.
type TimeoutError struct {
	Value time.Duration
	Min   time.Duration
	Max   time.Duration
}

func (e *TimeoutError) Error() string {
	return "DHCP timeout " + e.Value.String() + " must be between " + e.Min.String() + " and " + e.Max.String()
}

// LeaseInfo contains DHCP lease information.
type LeaseInfo struct {
	Interface    string        `json:"interface"`
	IPAddress    string        `json:"ipAddress"`
	SubnetMask   string        `json:"subnetMask"`
	Gateway      string        `json:"gateway,omitempty"`
	ServerIP     string        `json:"serverIp,omitempty"`
	DNSServers   []string      `json:"dnsServers,omitempty"`
	DomainName   string        `json:"domainName,omitempty"`
	LeaseTime    time.Duration `json:"leaseTime,omitempty"`
	LeaseTimeSec int           `json:"leaseTimeSec,omitempty"`
	RenewTime    time.Duration `json:"renewTime,omitempty"`
	RebindTime   time.Duration `json:"rebindTime,omitempty"`
	Expiry       time.Time     `json:"expiry,omitzero"`
	ObtainedAt   time.Time     `json:"obtainedAt,omitzero"`
}

// TestResult contains the result of a DHCP test.
type TestResult struct {
	Interface    string        `json:"interface"`
	Success      bool          `json:"success"`
	Status       Status        `json:"status"`
	ServerIP     string        `json:"serverIp,omitempty"`
	OfferedIP    string        `json:"offeredIp,omitempty"`
	SubnetMask   string        `json:"subnetMask,omitempty"`
	Gateway      string        `json:"gateway,omitempty"`
	DNSServers   []string      `json:"dnsServers,omitempty"`
	DomainName   string        `json:"domainName,omitempty"`
	LeaseTime    time.Duration `json:"leaseTime,omitempty"`
	LeaseTimeSec int           `json:"leaseTimeSec,omitempty"`
	ResponseTime time.Duration `json:"responseTime"`
	ResponseMs   float64       `json:"responseTimeMs"`
	Error        string        `json:"error,omitempty"`
	TestedAt     time.Time     `json:"testedAt"`
}

// Thresholds defines timing thresholds for DHCP responses.
type Thresholds struct {
	Warning  time.Duration
	Critical time.Duration
}

// DefaultThresholds returns reasonable default thresholds for DHCP.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Warning:  DefaultWarningThresholdMs * time.Millisecond,
		Critical: DefaultCriticalThresholdMs * time.Millisecond,
	}
}

// Tester performs DHCP tests.
type Tester struct {
	interfaceName string
	thresholds    Thresholds
	testTimeout   time.Duration
	lastResult    *TestResult
	mu            sync.RWMutex
}

// NewTester creates a new DHCP tester.
func NewTester(interfaceName string, thresholds Thresholds) *Tester {
	return &Tester{
		interfaceName: interfaceName,
		thresholds:    thresholds,
		testTimeout:   DefaultTestTimeout,
	}
}

// SetInterface updates the interface to test.
func (t *Tester) SetInterface(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.interfaceName = name
}

// GetInterface returns the current interface name.
func (t *Tester) GetInterface() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.interfaceName
}

// SetTimeout sets the test timeout.
func (t *Tester) SetTimeout(timeout time.Duration) error {
	if err := ValidateDHCPTimeout(timeout); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.testTimeout = timeout
	return nil
}

// GetTimeout returns the current test timeout.
func (t *Tester) GetTimeout() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.testTimeout
}

// SetThresholds updates the timing thresholds.
func (t *Tester) SetThresholds(thresholds Thresholds) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.thresholds = thresholds
}

// GetThresholds returns the current timing thresholds.
func (t *Tester) GetThresholds() Thresholds {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.thresholds
}

// GetLastResult returns the last test result.
func (t *Tester) GetLastResult() *TestResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.lastResult == nil {
		return nil
	}
	// Return a copy
	result := *t.lastResult
	return &result
}

// getStatus determines status based on timing and thresholds.
func (t *Tester) getStatus(duration time.Duration, hasError bool) Status {
	if hasError {
		return StatusError
	}

	t.mu.RLock()
	th := t.thresholds
	t.mu.RUnlock()

	if duration >= th.Critical {
		return StatusError
	}
	if duration >= th.Warning {
		return StatusWarning
	}
	return StatusSuccess
}

// storeAndReturnError stores an error result and returns it.
func (t *Tester) storeAndReturnError(result *TestResult, errMsg string) *TestResult {
	result.Success = false
	result.Status = StatusError
	result.Error = errMsg
	t.mu.Lock()
	t.lastResult = result
	t.mu.Unlock()
	return result
}

// mergePlatformResult copies fields from platform result to our result.
func mergePlatformResult(result, platformResult *TestResult) {
	result.Success = platformResult.Success
	result.ServerIP = platformResult.ServerIP
	result.OfferedIP = platformResult.OfferedIP
	result.SubnetMask = platformResult.SubnetMask
	result.Gateway = platformResult.Gateway
	result.DNSServers = platformResult.DNSServers
	result.DomainName = platformResult.DomainName
	result.LeaseTime = platformResult.LeaseTime
	result.LeaseTimeSec = int(platformResult.LeaseTime.Seconds())
	result.Error = platformResult.Error
}

// Test performs a DHCP test on the configured interface.
func (t *Tester) Test(ctx context.Context) *TestResult {
	t.mu.RLock()
	iface := t.interfaceName
	timeout := t.testTimeout
	t.mu.RUnlock()

	result := &TestResult{
		Interface: iface,
		TestedAt:  time.Now(),
	}

	if iface == "" {
		return t.storeAndReturnError(result, "no interface specified")
	}

	// Validate interface exists
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		return t.storeAndReturnError(result, "interface not found: "+err.Error())
	}

	// Check if interface is up
	if netIface.Flags&net.FlagUp == 0 {
		return t.storeAndReturnError(result, "interface is down")
	}

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Perform platform-specific DHCP test
	start := time.Now()
	platformResult := testDHCPPlatform(testCtx, iface)
	elapsed := time.Since(start)

	// Merge platform result with our result
	mergePlatformResult(result, platformResult)
	result.ResponseTime = elapsed
	result.ResponseMs = float64(elapsed.Microseconds()) / microsecondsPerMillisecond

	// Determine status
	result.Status = t.getStatus(elapsed, !result.Success)

	// Store result
	t.mu.Lock()
	t.lastResult = result
	t.mu.Unlock()

	return result
}

// GetCurrentLease retrieves the current DHCP lease information for the interface.
func (t *Tester) GetCurrentLease() (*LeaseInfo, error) {
	t.mu.RLock()
	iface := t.interfaceName
	t.mu.RUnlock()

	if iface == "" {
		return nil, &InterfaceError{Message: "no interface specified"}
	}

	return getCurrentLeasePlatform(iface)
}

// InterfaceError represents an interface-related error.
type InterfaceError struct {
	Message string
}

func (e *InterfaceError) Error() string {
	return e.Message
}

// GetSystemInterfaces returns a list of network interfaces that can use DHCP.
func GetSystemInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, iface := range interfaces {
		// Skip loopback and point-to-point interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// Include interfaces that are up or could be brought up
		if iface.Flags&net.FlagBroadcast != 0 || iface.Flags&net.FlagMulticast != 0 {
			result = append(result, iface.Name)
		}
	}

	return result, nil
}

// IsValidIPAddress checks if a string is a valid IP address.
func IsValidIPAddress(addr string) bool {
	return net.ParseIP(addr) != nil
}

// IsValidSubnetMask checks if a string is a valid subnet mask.
func IsValidSubnetMask(mask string) bool {
	ip := net.ParseIP(mask)
	if ip == nil {
		return false
	}

	// Convert to 4-byte representation for IPv4
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	// Check that mask is contiguous (no gaps between 1s and 0s)
	return isContiguousMask(ip4)
}

// isContiguousMask checks if a subnet mask is contiguous.
func isContiguousMask(mask net.IP) bool {
	ip4 := mask.To4()
	if ip4 == nil {
		return false
	}

	// Convert to uint32 for bit operations
	var m uint32
	for _, b := range ip4 {
		m = m<<bitsPerByte | uint32(b)
	}

	// A valid mask has all 1s followed by all 0s
	// Invert: all 0s followed by all 1s
	// Adding 1 should result in a power of 2
	inverted := ^m
	return (inverted & (inverted + 1)) == 0
}

// ParseCIDR parses a CIDR notation and returns the IP and subnet mask.
func ParseCIDR(cidr string) (string, string, error) {
	ipNet, network, parseErr := net.ParseCIDR(cidr)
	if parseErr != nil {
		return "", "", parseErr
	}
	return ipNet.String(), net.IP(network.Mask).String(), nil
}

// CalculateNetworkAddress calculates the network address from an IP and subnet mask.
func CalculateNetworkAddress(ipStr, maskStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", &InterfaceError{Message: "invalid IP address"}
	}

	mask := net.ParseIP(maskStr)
	if mask == nil {
		return "", &InterfaceError{Message: "invalid subnet mask"}
	}

	ip4 := ip.To4()
	mask4 := mask.To4()
	if ip4 == nil || mask4 == nil {
		return "", &InterfaceError{Message: "IPv4 addresses required"}
	}

	network := make(net.IP, ipv4Len)
	for i := range ipv4Len {
		network[i] = ip4[i] & mask4[i]
	}

	return network.String(), nil
}

// CalculateBroadcastAddress calculates the broadcast address from an IP and subnet mask.
func CalculateBroadcastAddress(ipStr, maskStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", &InterfaceError{Message: "invalid IP address"}
	}

	mask := net.ParseIP(maskStr)
	if mask == nil {
		return "", &InterfaceError{Message: "invalid subnet mask"}
	}

	ip4 := ip.To4()
	mask4 := mask.To4()
	if ip4 == nil || mask4 == nil {
		return "", &InterfaceError{Message: "IPv4 addresses required"}
	}

	broadcast := make(net.IP, ipv4Len)
	for i := range ipv4Len {
		broadcast[i] = ip4[i] | ^mask4[i]
	}

	return broadcast.String(), nil
}
