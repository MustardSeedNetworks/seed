//go:build windows

package dhcp

// Windows-specific DHCP implementation.
// Uses ipconfig and netsh commands for DHCP operations.

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for DHCP operations.
const dhcpTimeoutSeconds = 60

// colonSplitParts caps SplitN at 2 pieces when parsing "key: value"
// ipconfig output lines, so a value containing ':' is not split further.
const colonSplitParts = 2

// dhcpInfo contains DHCP lease information.
type dhcpInfo struct {
	Enabled     bool      `json:"enabled"`
	Server      string    `json:"server,omitempty"`
	LeaseStart  time.Time `json:"lease_start,omitzero"`
	LeaseExpiry time.Time `json:"lease_expiry,omitzero"`
	Gateway     string    `json:"gateway,omitempty"`
	DNS         []string  `json:"dns,omitempty"`
}

// getDHCPInfo retrieves DHCP lease information for an interface on Windows.
func getDHCPInfo(iface string) (*dhcpInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dhcpTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "ipconfig", "/all").Output()
	if err != nil {
		return nil, fmt.Errorf("ipconfig failed: %w", err)
	}

	return parseDHCPInfo(string(output), iface), nil
}

// parseDHCPInfo parses ipconfig /all output for DHCP information.
func parseDHCPInfo(output, targetIface string) *dhcpInfo {
	info := &dhcpInfo{}

	lines := strings.Split(output, "\n")
	inTargetAdapter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect adapter sections
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, " ") {
			// New adapter section
			adapterName := strings.TrimSuffix(strings.TrimSpace(line), ":")
			inTargetAdapter = targetIface == "" || strings.Contains(adapterName, targetIface)
			continue
		}

		if !inTargetAdapter {
			continue
		}

		switch {
		case strings.Contains(trimmed, "DHCP Enabled") || strings.Contains(trimmed, "DHCP 有効"):
			info.Enabled = strings.Contains(trimmed, "Yes") || strings.Contains(trimmed, "はい")
		case strings.Contains(trimmed, "DHCP Server") || strings.Contains(trimmed, "DHCP サーバー"):
			parseDHCPServerLine(trimmed, info)
		case strings.Contains(trimmed, "Lease Obtained") || strings.Contains(trimmed, "リース取得"):
			parseDHCPLeaseStartLine(trimmed, info)
		case strings.Contains(trimmed, "Lease Expires") || strings.Contains(trimmed, "リース期限"):
			parseDHCPLeaseExpiryLine(trimmed, info)
		case strings.Contains(trimmed, "Default Gateway") || strings.Contains(trimmed, "デフォルト ゲートウェイ"):
			parseDHCPGatewayLine(trimmed, info)
		}
	}

	return info
}

// parseDHCPServerLine reads the DHCP Server field into info.
func parseDHCPServerLine(trimmed string, info *dhcpInfo) {
	parts := strings.SplitN(trimmed, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		info.Server = strings.TrimSpace(parts[1])
	}
}

// parseDHCPLeaseStartLine reads the Lease Obtained field into info.
func parseDHCPLeaseStartLine(trimmed string, info *dhcpInfo) {
	parts := strings.SplitN(trimmed, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		info.LeaseStart = parseWindowsDate(strings.TrimSpace(parts[1]))
	}
}

// parseDHCPLeaseExpiryLine reads the Lease Expires field into info.
func parseDHCPLeaseExpiryLine(trimmed string, info *dhcpInfo) {
	parts := strings.SplitN(trimmed, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		info.LeaseExpiry = parseWindowsDate(strings.TrimSpace(parts[1]))
	}
}

// parseDHCPGatewayLine reads the Default Gateway field into info.
func parseDHCPGatewayLine(trimmed string, info *dhcpInfo) {
	parts := strings.SplitN(trimmed, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		info.Gateway = strings.TrimSpace(parts[1])
	}
}

// parseWindowsDate parses Windows ipconfig date format.
// Example: "Wednesday, January 15, 2025 10:30:00 AM".
func parseWindowsDate(s string) time.Time {
	// Try common Windows formats
	formats := []string{
		"Monday, January 2, 2006 3:04:05 PM",
		"January 2, 2006 3:04:05 PM",
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05",
	}

	s = strings.TrimSpace(s)
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}

// RenewDHCP renews the DHCP lease for an interface.
func RenewDHCP(iface string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dhcpTimeoutSeconds*time.Second)
	defer cancel()

	args := []string{"/renew"}
	if iface != "" {
		args = append(args, iface)
	}

	output, err := exec.CommandContext(ctx, "ipconfig", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipconfig /renew failed: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// testDHCPPlatform performs platform-specific DHCP testing on Windows.
// Uses ipconfig /all to query the current DHCP lease information.
// interfaceIPAndMask returns the first IPv4 address and subnet mask bound to
// the named interface, or two empty strings if the interface or its
// addresses cannot be read. Both testDHCPPlatform and getCurrentLeasePlatform
// use ipconfig for lease timing but read the live address off the interface
// itself, since ipconfig's own address line is harder to parse reliably.
func interfaceIPAndMask(interfaceName string) (string, string) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", ""
	}
	addrs, err := iface.Addrs()
	if err != nil || len(addrs) == 0 {
		return "", ""
	}
	ip, ipNet, err := net.ParseCIDR(addrs[0].String())
	if err != nil || ip == nil {
		return "", ""
	}
	if ipNet == nil {
		return ip.String(), ""
	}
	return ip.String(), net.IP(ipNet.Mask).String()
}

func testDHCPPlatform(ctx context.Context, interfaceName string) *TestResult {
	start := time.Now()
	result := &TestResult{
		Interface: interfaceName,
		TestedAt:  start,
	}

	output, err := exec.CommandContext(ctx, "ipconfig", "/all").Output()
	if err != nil {
		result.Success = false
		result.Status = StatusError
		result.Error = "ipconfig failed: " + err.Error()
		return result
	}

	info := parseDHCPInfo(string(output), interfaceName)
	result.ResponseTime = time.Since(start)
	result.ResponseMs = float64(result.ResponseTime.Milliseconds())

	if !info.Enabled {
		result.Success = false
		result.Status = StatusWarning
		result.Error = "DHCP is not enabled on this interface"
		return result
	}

	if info.Server != "" {
		result.Success = true
		result.Status = StatusSuccess
		result.ServerIP = info.Server
		result.Gateway = info.Gateway
		result.DNSServers = info.DNS

		// Parse offered IP from interface
		result.OfferedIP, result.SubnetMask = interfaceIPAndMask(interfaceName)

		if !info.LeaseExpiry.IsZero() && !info.LeaseStart.IsZero() {
			result.LeaseTime = info.LeaseExpiry.Sub(info.LeaseStart)
			result.LeaseTimeSec = int(result.LeaseTime.Seconds())
		}
	} else {
		result.Success = false
		result.Status = StatusWarning
		result.Error = "DHCP enabled but no server found"
	}

	return result
}

// getCurrentLeasePlatform retrieves the current DHCP lease on Windows.
func getCurrentLeasePlatform(interfaceName string) (*LeaseInfo, error) {
	info, err := getDHCPInfo(interfaceName)
	if err != nil {
		return nil, err
	}

	if !info.Enabled {
		return nil, &InterfaceError{Message: "DHCP not enabled on " + interfaceName}
	}

	lease := &LeaseInfo{
		Interface:  interfaceName,
		ServerIP:   info.Server,
		Gateway:    info.Gateway,
		DNSServers: info.DNS,
	}

	if !info.LeaseStart.IsZero() {
		lease.ObtainedAt = info.LeaseStart
	}
	if !info.LeaseExpiry.IsZero() {
		lease.Expiry = info.LeaseExpiry
	}
	if !info.LeaseStart.IsZero() && !info.LeaseExpiry.IsZero() {
		lease.LeaseTime = info.LeaseExpiry.Sub(info.LeaseStart)
		lease.LeaseTimeSec = int(lease.LeaseTime.Seconds())
	}

	// Get IP address from interface
	lease.IPAddress, lease.SubnetMask = interfaceIPAndMask(interfaceName)

	if lease.IPAddress == "" && lease.ServerIP == "" {
		return nil, &InterfaceError{Message: "no DHCP lease found for " + interfaceName}
	}

	return lease, nil
}
