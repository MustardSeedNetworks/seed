//go:build linux

package dhcp

import (
	"bufio"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	leaseLineParts = 2
	dhcpTimeParts  = 3
)

func linuxLeaseFilePaths() []string {
	return []string{
		"/var/lib/dhcp/dhclient.%s.leases",
		"/var/lib/dhclient/dhclient-%s.leases",
		"/var/lib/NetworkManager/dhclient-%s.lease",
		"/var/lib/dhcpcd/dhcpcd-%s.lease",
		"/run/dhcpcd-%s.lease",
	}
}

// testDHCPPlatform performs platform-specific DHCP testing on Linux.
func testDHCPPlatform(ctx context.Context, interfaceName string) *TestResult {
	result := &TestResult{
		Interface: interfaceName,
		TestedAt:  time.Now(),
	}

	// Try to find and parse lease file
	leaseInfo, err := findAndParseLeaseFile(interfaceName)
	if err == nil && leaseInfo != nil {
		result.Success = true
		result.OfferedIP = leaseInfo.IPAddress
		result.SubnetMask = leaseInfo.SubnetMask
		result.Gateway = leaseInfo.Gateway
		result.ServerIP = leaseInfo.ServerIP
		result.DNSServers = leaseInfo.DNSServers
		result.DomainName = leaseInfo.DomainName
		result.LeaseTime = leaseInfo.LeaseTime
		result.LeaseTimeSec = leaseInfo.LeaseTimeSec
		return result
	}

	// Fallback: try dhclient to probe DHCP (dry-run mode if available)
	// Note: This requires root privileges
	cmd := exec.CommandContext(ctx, "dhclient", "-v", "-d", "-1", "-sf", "/bin/true", interfaceName)
	output, cmdErr := cmd.CombinedOutput()
	if cmdErr == nil {
		parseLinuxDHClientOutput(string(output), result)
		if result.OfferedIP != "" {
			result.Success = true
			return result
		}
	}

	// Try to get current IP info from interface as fallback
	iface, ifaceErr := net.InterfaceByName(interfaceName)
	if ifaceErr == nil {
		addrs, addrsErr := iface.Addrs()
		if addrsErr == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					result.OfferedIP = ipnet.IP.String()
					result.SubnetMask = net.IP(ipnet.Mask).String()
					result.Success = true
					return result
				}
			}
		}
	}

	result.Success = false
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Error = "no DHCP lease found"
	}
	return result
}

// findAndParseLeaseFile finds and parses the DHCP lease file for an interface.
func findAndParseLeaseFile(interfaceName string) (*LeaseInfo, error) {
	for _, pathTemplate := range linuxLeaseFilePaths() {
		path := strings.ReplaceAll(pathTemplate, "%s", interfaceName)
		if _, err := os.Stat(path); err == nil {
			return parseLeaseFile(path, interfaceName)
		}
	}

	// Also check for wildcard lease files
	patterns := []string{
		"/var/lib/dhcp/dhclient*.leases",
		"/var/lib/dhclient/*.leases",
		"/var/lib/NetworkManager/*.lease",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if strings.Contains(match, interfaceName) {
				return parseLeaseFile(match, interfaceName)
			}
		}
	}

	return nil, &InterfaceError{Message: "no lease file found"}
}

// parseLeaseFile parses a dhclient lease file.
func parseLeaseFile(path, interfaceName string) (*LeaseInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lease := &LeaseInfo{
		Interface: interfaceName,
	}

	scanner := bufio.NewScanner(file)
	inLease := false
	var currentLease *LeaseInfo

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "lease {") {
			inLease = true
			currentLease = &LeaseInfo{Interface: interfaceName}
			continue
		}

		if line == "}" && inLease {
			inLease = false
			// Keep the most recent lease
			if currentLease != nil && currentLease.IPAddress != "" {
				lease = currentLease
			}
			continue
		}

		if inLease && currentLease != nil {
			parseLeaseFileLine(line, currentLease)
		}
	}

	if lease.IPAddress == "" {
		return nil, &InterfaceError{Message: "no valid lease found in file"}
	}

	return lease, nil
}

// parseLeaseFileLine parses a single line from a dhclient lease file.
func parseLeaseFileLine(line string, lease *LeaseInfo) {
	line = strings.TrimSuffix(line, ";")
	parts := strings.SplitN(line, " ", leaseLineParts)
	if len(parts) < leaseLineParts {
		return
	}

	key := parts[0]
	value := strings.Trim(parts[1], "\"")

	switch key {
	case "fixed-address":
		lease.IPAddress = value
	case "option":
		parseLeaseOption(value, lease)
	case "renew":
		// Format: renew 3 2024/01/15 10:30:00;
		if t, err := parseDHCPTime(value); err == nil {
			lease.RenewTime = time.Until(t)
		}
	case "expire":
		// Format: expire 3 2024/01/15 18:30:00;
		if t, err := parseDHCPTime(value); err == nil {
			lease.Expiry = t
		}
	}
}

func parseLeaseOption(value string, lease *LeaseInfo) {
	switch {
	case strings.HasPrefix(value, "subnet-mask "):
		lease.SubnetMask = strings.TrimPrefix(value, "subnet-mask ")
	case strings.HasPrefix(value, "routers "):
		lease.Gateway = strings.TrimSpace(strings.Split(strings.TrimPrefix(value, "routers "), ",")[0])
	case strings.HasPrefix(value, "domain-name-servers "):
		for server := range strings.SplitSeq(strings.TrimPrefix(value, "domain-name-servers "), ",") {
			if server = strings.TrimSpace(server); server != "" {
				lease.DNSServers = append(lease.DNSServers, server)
			}
		}
	case strings.HasPrefix(value, "domain-name "):
		lease.DomainName = strings.Trim(strings.TrimPrefix(value, "domain-name "), "\"")
	case strings.HasPrefix(value, "dhcp-server-identifier "):
		lease.ServerIP = strings.TrimPrefix(value, "dhcp-server-identifier ")
	case strings.HasPrefix(value, "dhcp-lease-time "):
		setLeaseDuration(strings.TrimPrefix(value, "dhcp-lease-time "), lease)
	}
}

func setLeaseDuration(value string, lease *LeaseInfo) {
	seconds, err := strconv.Atoi(value)
	if err == nil {
		lease.LeaseTime = time.Duration(seconds) * time.Second
		lease.LeaseTimeSec = seconds
	}
}

// parseDHCPTime parses a time from dhclient lease file format.
func parseDHCPTime(value string) (time.Time, error) {
	// Format: "3 2024/01/15 10:30:00" (weekday date time)
	parts := strings.Fields(value)
	if len(parts) < dhcpTimeParts {
		return time.Time{}, &InterfaceError{Message: "invalid time format"}
	}
	// Parse date and time, skipping weekday
	dateTime := parts[1] + " " + parts[2]
	return time.Parse("2006/01/02 15:04:05", dateTime)
}

// parseLinuxDHClientOutput parses dhclient verbose output.
func parseLinuxDHClientOutput(output string, result *TestResult) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		parseDHClientLine(scanner.Text(), result)
	}
}

func parseDHClientLine(line string, result *TestResult) {
	parts := strings.Fields(line)
	for index, part := range parts {
		if index+1 >= len(parts) {
			continue
		}
		switch {
		case strings.Contains(line, "DHCPOFFER") && part == "of":
			result.OfferedIP = parts[index+1]
		case strings.Contains(line, "DHCPOFFER") && part == "from":
			result.ServerIP = parts[index+1]
		case strings.Contains(line, "bound to") && part == "to":
			result.OfferedIP = parts[index+1]
		}
	}
}

// getCurrentLeasePlatform retrieves the current DHCP lease on Linux.
func getCurrentLeasePlatform(interfaceName string) (*LeaseInfo, error) {
	// First try to find and parse the lease file
	lease, err := findAndParseLeaseFile(interfaceName)
	if err == nil && lease != nil {
		return lease, nil
	}

	// Fallback: get current IP info from interface
	lease = &LeaseInfo{
		Interface: interfaceName,
	}

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, &InterfaceError{Message: "interface not found: " + err.Error()}
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, &InterfaceError{Message: "failed to get addresses: " + err.Error()}
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			lease.IPAddress = ipnet.IP.String()
			lease.SubnetMask = net.IP(ipnet.Mask).String()
			return lease, nil
		}
	}

	return nil, &InterfaceError{Message: "no IPv4 address found"}
}
