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

	// networkdLeaseDir is where systemd-networkd keeps each link's DHCPv4
	// lease, one file per interface index. It is the DHCP client on Ubuntu
	// Server through netplan, so most Linux probes have no dhclient file.
	networkdLeaseDir = "/run/systemd/netif/leases"

	networkdDay  = 24 * time.Hour
	networkdWeek = 7 * networkdDay
)

// leaseFilePathTemplates returns the lease-file locations searched for an
// interface, each with a %s where the interface name goes.
func leaseFilePathTemplates() []string {
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

// hostInterface returns the host's own entry for the named interface, or an
// error when the host has no such interface.
//
// The point is the source of the returned name and index: they come from the
// kernel's interface list, not from the caller. findAndParseLeaseFile
// interpolates them into lease-file paths, so a caller-supplied name carrying
// traversal segments would otherwise steer [os.Stat] and [os.Open] outside the
// lease directories. Returning the matched entry means the values that reach a
// path can only ever be a real interface's.
func hostInterface(name string) (net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, &InterfaceError{Message: "failed to list interfaces: " + err.Error()}
	}

	for _, iface := range interfaces {
		if iface.Name == name {
			return iface, nil
		}
	}

	return net.Interface{}, &InterfaceError{Message: "interface not found: " + name}
}

// findAndParseLeaseFile finds and parses the DHCP lease file for an interface.
//
// The name is resolved against the host's interface list first. The API handler
// checks this too, but the guarantee belongs here: this is the function that
// builds the path, and a second caller that forgets the check must not turn it
// into an arbitrary-file read.
func findAndParseLeaseFile(requestedName string) (*LeaseInfo, error) {
	return findLeaseFileIn(networkdLeaseDir, leaseFilePathTemplates(), requestedName)
}

// findLeaseFileIn is findAndParseLeaseFile over a given networkd lease
// directory and set of dhclient/dhcpcd path templates. Both are parameters so a
// test can point the search at a temp directory and demonstrate the traversal
// the name check prevents.
//
// networkd is searched first. Its directory is on tmpfs and networkd removes a
// link's file when the lease goes, so a file there is the live lease; dhclient
// files under /var/lib outlive both the lease and the client that wrote them.
func findLeaseFileIn(networkdDir string, templates []string, requestedName string) (*LeaseInfo, error) {
	iface, resolveErr := hostInterface(requestedName)
	if resolveErr != nil {
		return nil, resolveErr
	}
	interfaceName := iface.Name

	networkdPath := filepath.Join(networkdDir, strconv.Itoa(iface.Index))
	if _, err := os.Stat(networkdPath); err == nil {
		return parseNetworkdLeaseFile(networkdPath, interfaceName)
	}

	for _, pathTemplate := range templates {
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

// parseNetworkdLeaseFile parses a systemd-networkd DHCPv4 lease: KEY=VALUE
// lines, with list values separated by spaces.
//
// networkd heads the file "This is private data. Do not parse." Its varlink
// API (io.systemd.Network.Describe) was checked first and does not carry the
// router, DNS or domain options, so the file is the only complete source; the
// format has been stable since networkd shipped, and cloud-init reads it too.
//
// Expiry is left unset. networkd rewrites the file whenever it saves the link's
// state, not only when the lease changes, so its modification time is not the
// lease's acquisition time and an expiry derived from it would be invented.
func parseNetworkdLeaseFile(path, interfaceName string) (*LeaseInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lease := &LeaseInfo{Interface: interfaceName}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, found := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
		if !found {
			continue
		}

		switch key {
		case "ADDRESS":
			lease.IPAddress = value
		case "NETMASK":
			lease.SubnetMask = value
		case "ROUTER":
			if routers := strings.Fields(value); len(routers) > 0 {
				lease.Gateway = routers[0]
			}
		case "SERVER_ADDRESS":
			lease.ServerIP = value
		case "DNS":
			lease.DNSServers = strings.Fields(value)
		case "DOMAINNAME":
			lease.DomainName = value
		case "LIFETIME":
			if lifetime, ok := parseNetworkdTimespan(value); ok {
				lease.LeaseTime = lifetime
				lease.LeaseTimeSec = int(lifetime.Seconds())
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, scanErr
	}

	if lease.IPAddress == "" {
		return nil, &InterfaceError{Message: "no valid lease found in file"}
	}

	return lease, nil
}

// networkdTimespanUnit maps the suffixes systemd's timespan formatter emits at
// the magnitudes a DHCP lease has.
func networkdTimespanUnit(suffix string) (time.Duration, bool) {
	switch suffix {
	case "ms":
		return time.Millisecond, true
	case "s":
		return time.Second, true
	case "min":
		return time.Minute, true
	case "h":
		return time.Hour, true
	case "d":
		return networkdDay, true
	case "w":
		return networkdWeek, true
	}

	return 0, false
}

// parseNetworkdTimespan parses a networkd lease duration. Current systemd
// writes a formatted timespan ("1d", "12h 30min"); older releases wrote plain
// seconds ("86400"). An "infinity" lease, or anything else, reports false.
func parseNetworkdTimespan(value string) (time.Duration, bool) {
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second, true
	}

	var total time.Duration
	fields := strings.Fields(value)
	for _, field := range fields {
		digits := strings.TrimRightFunc(field, func(r rune) bool { return r < '0' || r > '9' })
		amount, err := strconv.Atoi(digits)
		unit, known := networkdTimespanUnit(field[len(digits):])
		if err != nil || !known {
			return 0, false
		}
		total += time.Duration(amount) * unit
	}

	return total, len(fields) > 0
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

// getCurrentLeasePlatform retrieves the current DHCP lease on Linux. An
// interface with no lease file holds no lease: its address, if it has one, is
// not one DHCP gave it, and reporting it as a lease would grade a static
// configuration as an incomplete lease.
func getCurrentLeasePlatform(interfaceName string) (*LeaseInfo, error) {
	return findAndParseLeaseFile(interfaceName)
}
