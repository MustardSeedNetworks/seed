//go:build windows

// Windows-specific DHCP implementation.
// Uses ipconfig and netsh commands for DHCP operations.
package dhcp

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

// DHCPInfo contains DHCP lease information.
type DHCPInfo struct {
	Enabled     bool      `json:"enabled"`
	Server      string    `json:"server,omitempty"`
	LeaseStart  time.Time `json:"lease_start,omitempty"`
	LeaseExpiry time.Time `json:"lease_expiry,omitempty"`
	Gateway     string    `json:"gateway,omitempty"`
	DNS         []string  `json:"dns,omitempty"`
}

// GetDHCPInfo retrieves DHCP lease information for an interface on Windows.
func GetDHCPInfo(iface string) (*DHCPInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dhcpTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "ipconfig", "/all").Output()
	if err != nil {
		return nil, fmt.Errorf("ipconfig failed: %w", err)
	}

	return parseDHCPInfo(string(output), iface), nil
}

// parseDHCPInfo parses ipconfig /all output for DHCP information.
func parseDHCPInfo(output, targetIface string) *DHCPInfo {
	info := &DHCPInfo{}

	lines := strings.Split(output, "\n")
	inTargetAdapter := false
	adapterName := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect adapter sections
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, " ") {
			// New adapter section
			adapterName = strings.TrimSuffix(strings.TrimSpace(line), ":")
			inTargetAdapter = targetIface == "" || strings.Contains(adapterName, targetIface)
			continue
		}

		if !inTargetAdapter {
			continue
		}

		// Parse DHCP-related fields
		if strings.Contains(trimmed, "DHCP Enabled") || strings.Contains(trimmed, "DHCP 有効") {
			info.Enabled = strings.Contains(trimmed, "Yes") || strings.Contains(trimmed, "はい")
		} else if strings.Contains(trimmed, "DHCP Server") || strings.Contains(trimmed, "DHCP サーバー") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				info.Server = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(trimmed, "Lease Obtained") || strings.Contains(trimmed, "リース取得") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				// Parse Windows date format
				info.LeaseStart = parseWindowsDate(strings.TrimSpace(parts[1]))
			}
		} else if strings.Contains(trimmed, "Lease Expires") || strings.Contains(trimmed, "リース期限") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				info.LeaseExpiry = parseWindowsDate(strings.TrimSpace(parts[1]))
			}
		} else if strings.Contains(trimmed, "Default Gateway") || strings.Contains(trimmed, "デフォルト ゲートウェイ") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				info.Gateway = strings.TrimSpace(parts[1])
			}
		}
	}

	return info
}

// parseWindowsDate parses Windows ipconfig date format.
// Example: "Wednesday, January 15, 2025 10:30:00 AM"
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

// ReleaseDHCP releases the DHCP lease for an interface.
func ReleaseDHCP(iface string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dhcpTimeoutSeconds*time.Second)
	defer cancel()

	args := []string{"/release"}
	if iface != "" {
		args = append(args, iface)
	}

	output, err := exec.CommandContext(ctx, "ipconfig", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipconfig /release failed: %s", strings.TrimSpace(string(output)))
	}

	return nil
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

// FlushDNS flushes the DNS resolver cache on Windows.
func FlushDNS() error {
	ctx, cancel := context.WithTimeout(context.Background(), dhcpTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "ipconfig", "/flushdns").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipconfig /flushdns failed: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// testDHCPPlatform performs platform-specific DHCP testing on Windows.
// Uses ipconfig /all to query the current DHCP lease information.
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
		if iface, err := net.InterfaceByName(interfaceName); err == nil {
			if addrs, err := iface.Addrs(); err == nil && len(addrs) > 0 {
				ip, ipNet, _ := net.ParseCIDR(addrs[0].String())
				if ip != nil {
					result.OfferedIP = ip.String()
					if ipNet != nil {
						result.SubnetMask = net.IP(ipNet.Mask).String()
					}
				}
			}
		}

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
	info, err := GetDHCPInfo(interfaceName)
	if err != nil {
		return nil, err
	}

	if !info.Enabled {
		return nil, &InterfaceError{Message: "DHCP not enabled on " + interfaceName}
	}

	lease := &LeaseInfo{
		Interface: interfaceName,
		ServerIP:  info.Server,
		Gateway:   info.Gateway,
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
	if iface, err := net.InterfaceByName(interfaceName); err == nil {
		if addrs, err := iface.Addrs(); err == nil && len(addrs) > 0 {
			ip, ipNet, _ := net.ParseCIDR(addrs[0].String())
			if ip != nil {
				lease.IPAddress = ip.String()
				if ipNet != nil {
					lease.SubnetMask = net.IP(ipNet.Mask).String()
				}
			}
		}
	}

	if lease.IPAddress == "" && lease.ServerIP == "" {
		return nil, &InterfaceError{Message: "no DHCP lease found for " + interfaceName}
	}

	return lease, nil
}
