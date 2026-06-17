package dhcp

//
// Lease helpers: read DHCP lease information from the OS (darwin/linux lease
// files). Split out of dhcp.go to keep both files focused.

import (
	"bufio"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// LeaseInfo contains DHCP lease information from the system.
type LeaseInfo struct {
	DHCPServer string
	Gateway    string
	LeaseTime  int // seconds
	DNS        []string
}

// GetLeaseInfo retrieves DHCP lease information for an interface.
// Returns (nil, nil) for unsupported platforms - this is not an error.
func GetLeaseInfo(interfaceName string) (*LeaseInfo, error) {
	switch runtime.GOOS {
	case "darwin":
		return getLeaseInfoDarwin(interfaceName)
	case "linux":
		return getLeaseInfoLinux(interfaceName)
	default:
		//nolint:nilnil // Unsupported platform returns no info, not an error
		return nil, nil
	}
}

// getLeaseInfoDarwin reads DHCP info on macOS from lease files.
// macOS stores DHCP leases in /var/db/dhcpclient/leases/ as plist-like files.
// The filename format is: ifname-1,<hardware_address>.
func getLeaseInfoDarwin(interfaceName string) (*LeaseInfo, error) {
	// Get hardware address for the interface to find the lease file
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	// macOS lease files are named like: en0-1,aa:bb:cc:dd:ee:ff
	hwAddr := iface.HardwareAddr.String()

	// Try both old and new lease file locations
	// Note: Using filepath.Join with absolute path is intentional to construct full paths
	leasePaths := []string{
		"/var/db/dhcpclient/leases/" + interfaceName + "-1," + hwAddr,
		"/private/var/db/dhcpclient/leases/" + interfaceName + "-1," + hwAddr,
	}

	for _, path := range leasePaths {
		if info := parseDarwinLeaseFile(path); info != nil {
			return info, nil
		}
	}

	// Fallback: scan the leases directory for any file matching the interface
	leaseDir := "/var/db/dhcpclient/leases"
	entries, err := os.ReadDir(leaseDir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), interfaceName+"-") {
				if info := parseDarwinLeaseFile(filepath.Join(leaseDir, entry.Name())); info != nil {
					return info, nil
				}
			}
		}
	}

	return &LeaseInfo{}, nil
}

// parseDarwinLeaseFile parses a macOS DHCP lease file (plist-like format).
func parseDarwinLeaseFile(path string) *LeaseInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	info := &LeaseInfo{}
	content := string(data)

	// The lease file is XML plist format
	// Extract values using simple string parsing (avoiding plist dependency)

	// <key>ServerIdentifier</key> followed by <data>hex_encoded_ip</data>
	if server := extractPlistIP(content, "ServerIdentifier"); server != "" {
		info.DHCPServer = server
	}

	// <key>RouterIPAddress</key> followed by <data>hex_encoded_ip</data>
	if router := extractPlistIP(content, "RouterIPAddress"); router != "" {
		info.Gateway = router
	}
	// Also try Router key
	if info.Gateway == "" {
		if router := extractPlistIP(content, "Router"); router != "" {
			info.Gateway = router
		}
	}

	// <key>LeaseLength</key> followed by <integer>value</integer>
	if lease := extractPlistInteger(content, "LeaseLength"); lease > 0 {
		info.LeaseTime = lease
	}

	// DNS servers - may be in array format
	if dns := extractPlistIPArray(content, "DomainNameServer"); len(dns) > 0 {
		info.DNS = dns
	}

	if info.DHCPServer != "" || info.Gateway != "" {
		return info
	}
	return nil
}

// extractPlistIP extracts an IP address from a plist data field.
func extractPlistIP(content, key string) string {
	keyTag := "<key>" + key + "</key>"
	_, remaining, found := strings.Cut(content, keyTag)
	if !found {
		return ""
	}

	// Look for <data> tag after the key
	dataStart := strings.Index(remaining, "<data>")
	if dataStart == -1 {
		return ""
	}
	dataEnd := strings.Index(remaining[dataStart:], "</data>")
	if dataEnd == -1 {
		return ""
	}

	hexData := strings.TrimSpace(remaining[dataStart+6 : dataStart+dataEnd])
	return hexToIP(hexData)
}

// extractPlistInteger extracts an integer value from a plist.
func extractPlistInteger(content, key string) int {
	keyTag := "<key>" + key + "</key>"
	_, remaining, found := strings.Cut(content, keyTag)
	if !found {
		return 0
	}

	intStart := strings.Index(remaining, "<integer>")
	if intStart == -1 {
		return 0
	}
	intEnd := strings.Index(remaining[intStart:], "</integer>")
	if intEnd == -1 {
		return 0
	}

	valStr := strings.TrimSpace(remaining[intStart+9 : intStart+intEnd])
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0
	}
	return val
}

// parseDataTagsFromArray extracts IP addresses from <data> tags within array content.
func parseDataTagsFromArray(arrayContent string) []string {
	var ips []string
	content := arrayContent

	for {
		dataStart := strings.Index(content, "<data>")
		if dataStart == -1 {
			break
		}
		dataEnd := strings.Index(content[dataStart:], "</data>")
		if dataEnd == -1 {
			break
		}
		hexData := strings.TrimSpace(content[dataStart+6 : dataStart+dataEnd])
		if ip := hexToIP(hexData); ip != "" {
			ips = append(ips, ip)
		}
		content = content[dataStart+dataEnd+7:]
	}

	return ips
}

// extractArrayContent finds and extracts the content between <array> and </array> tags.
func extractArrayContent(remaining string) (string, bool) {
	arrayStart := strings.Index(remaining, "<array>")
	if arrayStart == -1 {
		return "", false
	}

	arrayEnd := strings.Index(remaining[arrayStart:], "</array>")
	if arrayEnd == -1 {
		return "", false
	}

	return remaining[arrayStart+7 : arrayStart+arrayEnd], true
}

// extractPlistIPArray extracts array of IPs from plist.
func extractPlistIPArray(content, key string) []string {
	keyTag := "<key>" + key + "</key>"
	_, remaining, found := strings.Cut(content, keyTag)
	if !found {
		return nil
	}

	// Try array format first
	if arrayContent, arrayFound := extractArrayContent(remaining); arrayFound {
		if ips := parseDataTagsFromArray(arrayContent); len(ips) > 0 {
			return ips
		}
	}

	// Try single data format
	if ip := extractPlistIP(content, key); ip != "" {
		return []string{ip}
	}

	return nil
}

// hexToIP converts hex-encoded IP address to string.
func hexToIP(hexStr string) string {
	// Remove any whitespace/newlines from base64-ish encoding
	hexStr = strings.ReplaceAll(hexStr, "\n", "")
	hexStr = strings.ReplaceAll(hexStr, "\t", "")
	hexStr = strings.ReplaceAll(hexStr, " ", "")

	// Try decoding as hex
	if bytes, err := hex.DecodeString(hexStr); err == nil && len(bytes) == 4 {
		return net.IP(bytes).String()
	}

	// macOS sometimes uses raw bytes in base64 - try that too
	// The data might actually be raw bytes interpreted as string
	if len(hexStr) == hexIPv4Len {
		return net.IP([]byte(hexStr)).String()
	}

	return ""
}

// getLeaseInfoLinux reads DHCP info on Linux from lease files.
func getLeaseInfoLinux(interfaceName string) (*LeaseInfo, error) {
	info := &LeaseInfo{}

	// Try NetworkManager lease file first
	nmLeasePath := "/var/lib/NetworkManager/internal-" + interfaceName + ".lease"
	if _, err := os.Stat(nmLeasePath); err == nil {
		if lease := parseNMLeaseFile(nmLeasePath); lease != nil {
			return lease, nil
		}
	}

	// Try dhclient lease file
	dhclientPaths := []string{
		"/var/lib/dhcp/dhclient." + interfaceName + ".leases",
		"/var/lib/dhclient/dhclient." + interfaceName + ".leases",
		"/var/lib/dhcp/dhclient.leases",
	}

	for _, path := range dhclientPaths {
		if _, err := os.Stat(path); err == nil {
			if lease := parseDHClientLeaseFile(path, interfaceName); lease != nil {
				return lease, nil
			}
		}
	}

	// Try systemd-networkd lease file
	networkdPath := "/run/systemd/netif/leases/"
	if entries, err := os.ReadDir(networkdPath); err == nil {
		for _, entry := range entries {
			if lease := parseNetworkdLeaseFile(networkdPath + entry.Name()); lease != nil {
				return lease, nil
			}
		}
	}

	return info, nil
}

// parseDHClientLeaseLine parses a single line from a dhclient lease file.
func parseDHClientLeaseLine(line string, info *LeaseInfo) {
	switch {
	case strings.HasPrefix(line, "option dhcp-server-identifier"):
		info.DHCPServer = extractValue(line)
	case strings.HasPrefix(line, "option routers"):
		val := extractValue(line)
		if parts := strings.Split(val, ","); len(parts) > 0 {
			info.Gateway = strings.TrimSpace(parts[0])
		}
	case strings.HasPrefix(line, "option dhcp-lease-time"):
		if lease, err := strconv.Atoi(extractValue(line)); err == nil {
			info.LeaseTime = lease
		}
	case strings.HasPrefix(line, "option domain-name-servers"):
		val := extractValue(line)
		for dns := range strings.SplitSeq(val, ",") {
			dns = strings.TrimSpace(dns)
			if dns != "" {
				info.DNS = append(info.DNS, dns)
			}
		}
	}
}

// parseDHClientLeaseFile parses a dhclient lease file.
func parseDHClientLeaseFile(path, _ string) *LeaseInfo {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	info := &LeaseInfo{}
	scanner := bufio.NewScanner(file)
	inLease := false

	// Parse the last lease block for the interface
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "lease {") {
			inLease = true
			info = &LeaseInfo{} // Reset for each lease block
			continue
		}
		if !inLease {
			continue
		}
		if line == "}" {
			inLease = false
			continue
		}
		parseDHClientLeaseLine(line, info)
	}

	if info.DHCPServer != "" || info.Gateway != "" {
		return info
	}
	return nil
}

// leaseFieldMapping defines how to extract DHCP lease fields from a file format.
type leaseFieldMapping struct {
	serverKey    string
	routerKey    string
	leaseTimeKey string
	dnsKey       string
}

// parseLeaseLineServer extracts the DHCP server from a line if it matches the server key.
func parseLeaseLineServer(line, serverKey string) (string, bool) {
	after, ok := strings.CutPrefix(line, serverKey)
	if !ok {
		return "", false
	}
	return after, true
}

// parseLeaseLineRouter extracts the gateway from a line if it matches the router key.
func parseLeaseLineRouter(line, routerKey string) (string, bool) {
	after, ok := strings.CutPrefix(line, routerKey)
	if !ok {
		return "", false
	}
	parts := strings.Split(after, " ")
	if len(parts) == 0 {
		return "", false
	}
	return parts[0], true
}

// parseLeaseLineTime extracts the lease time from a line if it matches the lease time key.
func parseLeaseLineTime(line, leaseTimeKey string) (int, bool) {
	after, ok := strings.CutPrefix(line, leaseTimeKey)
	if !ok {
		return 0, false
	}
	lease, err := strconv.Atoi(after)
	if err != nil {
		return 0, false
	}
	return lease, true
}

// parseLeaseLineDNS extracts DNS servers from a line if it matches the DNS key.
func parseLeaseLineDNS(line, dnsKey string) []string {
	after, ok := strings.CutPrefix(line, dnsKey)
	if !ok {
		return nil
	}
	var servers []string
	for dns := range strings.SplitSeq(after, " ") {
		dns = strings.TrimSpace(dns)
		if dns != "" {
			servers = append(servers, dns)
		}
	}
	return servers
}

// processLeaseLine processes a single lease file line and updates the LeaseInfo.
func processLeaseLine(line string, mapping leaseFieldMapping, info *LeaseInfo) {
	if server, ok := parseLeaseLineServer(line, mapping.serverKey); ok {
		info.DHCPServer = server
	}
	if gateway, ok := parseLeaseLineRouter(line, mapping.routerKey); ok {
		info.Gateway = gateway
	}
	if leaseTime, ok := parseLeaseLineTime(line, mapping.leaseTimeKey); ok {
		info.LeaseTime = leaseTime
	}
	if dnsServers := parseLeaseLineDNS(line, mapping.dnsKey); dnsServers != nil {
		info.DNS = append(info.DNS, dnsServers...)
	}
}

// parseLeaseFileWithMapping parses a lease file using the given field mappings.
func parseLeaseFileWithMapping(path string, mapping leaseFieldMapping) *LeaseInfo {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	info := &LeaseInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		processLeaseLine(line, mapping, info)
	}

	if info.DHCPServer != "" || info.Gateway != "" {
		return info
	}
	return nil
}

// parseNMLeaseFile parses NetworkManager internal lease file.
func parseNMLeaseFile(path string) *LeaseInfo {
	return parseLeaseFileWithMapping(path, leaseFieldMapping{
		serverKey:    "DHCP4_SERVER_ID=",
		routerKey:    "DHCP4_ROUTERS=",
		leaseTimeKey: "DHCP4_LEASE_TIME=",
		dnsKey:       "DHCP4_DOMAIN_NAME_SERVERS=",
	})
}

// parseNetworkdLeaseFile parses systemd-networkd lease file.
func parseNetworkdLeaseFile(path string) *LeaseInfo {
	return parseLeaseFileWithMapping(path, leaseFieldMapping{
		serverKey:    "SERVER_ADDRESS=",
		routerKey:    "ROUTER=",
		leaseTimeKey: "LIFETIME=",
		dnsKey:       "DNS=",
	})
}

// extractValue extracts value from dhclient option line.
func extractValue(line string) string {
	// Remove trailing semicolon
	line = strings.TrimSuffix(line, ";")
	// Get last space-separated value
	parts := strings.Fields(line)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
