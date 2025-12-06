// Package network handles network interface management.
package network

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/krisarmstrong/netscope/internal/validation"
)

// InterfaceType represents the type of network interface.
type InterfaceType string

const (
	InterfaceTypeEthernet InterfaceType = "ethernet"
	InterfaceTypeWiFi     InterfaceType = "wifi"
	InterfaceTypeLoopback InterfaceType = "loopback"
	InterfaceTypeOther    InterfaceType = "other"
)

// InterfaceInfo contains information about a network interface.
type InterfaceInfo struct {
	Name         string        `json:"name"`
	Type         InterfaceType `json:"type"`
	Up           bool          `json:"up"`
	Running      bool          `json:"running"`
	HardwareAddr string        `json:"hardwareAddr"`
	MTU          int           `json:"mtu"`
	Addresses    []string      `json:"addresses"`
}

// LinkStatus contains link layer status information.
type LinkStatus struct {
	Speed      string   `json:"speed"`      // e.g., "1000Mb/s"
	Duplex     string   `json:"duplex"`     // "full" or "half"
	LinkUp     bool     `json:"linkUp"`     // Deprecated: use Carrier && HasIP for accurate status
	Carrier    bool     `json:"carrier"`    // Physical link/carrier detected (Layer 2)
	HasIP      bool     `json:"hasIP"`      // Has routable IP address (Layer 3)
	Advertised []string `json:"advertised"` // Advertised link modes
	AutoNeg    bool     `json:"autoNeg"`    // Auto-negotiation enabled
}

// Manager handles network interface operations.
type Manager struct {
	mu               sync.RWMutex
	currentInterface string
	interfaces       map[string]*InterfaceInfo
}

// NewManager creates a new network manager.
func NewManager(defaultInterface string) *Manager {
	m := &Manager{
		currentInterface: defaultInterface,
		interfaces:       make(map[string]*InterfaceInfo),
	}
	m.RefreshInterfaces()
	return m
}

// RefreshInterfaces updates the list of available interfaces.
func (m *Manager) RefreshInterfaces() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get interfaces: %w", err)
	}

	// Build new map first, then swap under lock
	newInterfaces := make(map[string]*InterfaceInfo)

	for _, iface := range ifaces {
		info := &InterfaceInfo{
			Name:         iface.Name,
			Type:         detectInterfaceType(iface.Name),
			Up:           iface.Flags&net.FlagUp != 0,
			Running:      iface.Flags&net.FlagRunning != 0,
			HardwareAddr: iface.HardwareAddr.String(),
			MTU:          iface.MTU,
			Addresses:    []string{},
		}

		// Get IP addresses
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				info.Addresses = append(info.Addresses, addr.String())
			}
		}

		newInterfaces[iface.Name] = info
	}

	// Swap under lock
	m.mu.Lock()
	m.interfaces = newInterfaces
	m.mu.Unlock()

	return nil
}

// GetInterfaces returns all available interfaces.
func (m *Manager) GetInterfaces() []*InterfaceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*InterfaceInfo, 0, len(m.interfaces))
	for _, info := range m.interfaces {
		result = append(result, info)
	}
	return result
}

// GetInterface returns information about a specific interface.
func (m *Manager) GetInterface(name string) (*InterfaceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.interfaces[name]
	if !ok {
		return nil, fmt.Errorf("interface %s not found", name)
	}
	return info, nil
}

// GetCurrentInterface returns the currently selected interface.
func (m *Manager) GetCurrentInterface() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentInterface
}

// SetCurrentInterface sets the active interface.
func (m *Manager) SetCurrentInterface(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.interfaces[name]; !ok {
		return fmt.Errorf("interface %s not found", name)
	}
	m.currentInterface = name
	return nil
}

// FindFirstAvailable finds the first available interface from a list.
func (m *Manager) FindFirstAvailable(preferred []string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, name := range preferred {
		if info, ok := m.interfaces[name]; ok && info.Up {
			return name
		}
	}

	// Fall back to first non-loopback interface
	for name, info := range m.interfaces {
		if info.Type != InterfaceTypeLoopback && info.Up {
			return name
		}
	}

	return ""
}

// GetLinkStatus returns the link status for an interface.
func (m *Manager) GetLinkStatus(name string) (*LinkStatus, error) {
	m.mu.RLock()
	info, ok := m.interfaces[name]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("interface %s not found", name)
	}

	// Separate carrier (Layer 2) from IP assignment (Layer 3)
	carrier := info.Running                        // Physical link/carrier detected
	hasIP := hasRoutableAddress(info.Addresses)    // Has routable IP address
	linkUp := carrier && hasIP                     // Legacy: both conditions met

	status := &LinkStatus{
		LinkUp:  linkUp,
		Carrier: carrier,
		HasIP:   hasIP,
	}

	// Try to read speed from sysfs (Linux only)
	speedPath := filepath.Join("/sys/class/net", name, "speed")
	if data, err := os.ReadFile(speedPath); err == nil {
		speed := strings.TrimSpace(string(data))
		if speed != "" && speed != "-1" {
			status.Speed = speed + "Mb/s"
		}
	}

	// Try to read duplex from sysfs (Linux only)
	duplexPath := filepath.Join("/sys/class/net", name, "duplex")
	if data, err := os.ReadFile(duplexPath); err == nil {
		status.Duplex = strings.TrimSpace(string(data))
	}

	// macOS: try to get link info from ifconfig
	if status.Speed == "" {
		speed, duplex := getLinkInfoFromIfconfig(name)
		if speed != "" {
			status.Speed = speed
		}
		if duplex != "" {
			status.Duplex = duplex
		}
	}

	return status, nil
}

// hasRoutableAddress checks if any address is routable (not link-local).
func hasRoutableAddress(addresses []string) bool {
	for _, addr := range addresses {
		// Parse the address (remove CIDR suffix if present)
		ipStr := addr
		if idx := strings.Index(addr, "/"); idx != -1 {
			ipStr = addr[:idx]
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		// Skip loopback
		if ip.IsLoopback() {
			continue
		}
		// Skip link-local (169.254.x.x for IPv4, fe80:: for IPv6)
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		// Found a routable address
		return true
	}
	return false
}

// getLinkInfoFromIfconfig parses ifconfig output on macOS.
func getLinkInfoFromIfconfig(name string) (speed, duplex string) {
	// This is a placeholder - actual implementation would exec ifconfig
	// and parse "media: autoselect (1000baseT <full-duplex>)"
	return "", ""
}

// detectInterfaceType determines the type of interface from its name.
func detectInterfaceType(name string) InterfaceType {
	// Loopback
	if name == "lo" || name == "lo0" {
		return InterfaceTypeLoopback
	}

	// WiFi interfaces
	wifiPrefixes := []string{"wlan", "wlp", "wifi", "ath", "ra", "wl"}
	for _, prefix := range wifiPrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeWiFi
		}
	}

	// Ethernet interfaces
	ethPrefixes := []string{"eth", "enp", "ens", "eno", "em", "en"}
	for _, prefix := range ethPrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeEthernet
		}
	}

	return InterfaceTypeOther
}

// IsWireless returns true if the interface is a wireless interface.
func (m *Manager) IsWireless(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.interfaces[name]
	if !ok {
		return false
	}
	return info.Type == InterfaceTypeWiFi
}

// StaticIPConfig contains static IP configuration.
type StaticIPConfig struct {
	Address string   `json:"address"`
	Netmask string   `json:"netmask"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
}

// ConfigureStaticIP applies a static IP configuration to an interface.
// Requires root/administrator privileges.
func (m *Manager) ConfigureStaticIP(iface string, cfg *StaticIPConfig) error {
	// Validate input
	if err := validateIPConfig(cfg); err != nil {
		return err
	}

	switch runtime.GOOS {
	case "linux":
		return configureStaticIPLinux(iface, cfg)
	case "darwin":
		return configureStaticIPDarwin(iface, cfg)
	default:
		return fmt.Errorf("static IP configuration not supported on %s", runtime.GOOS)
	}
}

// ConfigureDHCP switches an interface to DHCP mode.
// Requires root/administrator privileges.
func (m *Manager) ConfigureDHCP(iface string) error {
	switch runtime.GOOS {
	case "linux":
		return configureDHCPLinux(iface)
	case "darwin":
		return configureDHCPDarwin(iface)
	default:
		return fmt.Errorf("DHCP configuration not supported on %s", runtime.GOOS)
	}
}

// validateIPConfig validates the static IP configuration.
func validateIPConfig(cfg *StaticIPConfig) error {
	if cfg.Address == "" {
		return fmt.Errorf("IP address is required")
	}
	if cfg.Netmask == "" {
		return fmt.Errorf("netmask is required")
	}

	// Validate IP address
	if net.ParseIP(cfg.Address) == nil {
		return fmt.Errorf("invalid IP address: %s", cfg.Address)
	}

	// Validate netmask (can be CIDR prefix or dotted notation)
	if !isValidNetmask(cfg.Netmask) {
		return fmt.Errorf("invalid netmask: %s", cfg.Netmask)
	}

	// Validate gateway if provided
	if cfg.Gateway != "" {
		if net.ParseIP(cfg.Gateway) == nil {
			return fmt.Errorf("invalid gateway: %s", cfg.Gateway)
		}
	}

	// Validate DNS servers if provided
	for _, dns := range cfg.DNS {
		if net.ParseIP(dns) == nil {
			return fmt.Errorf("invalid DNS server: %s", dns)
		}
	}

	return nil
}

// isValidNetmask checks if the netmask is valid (CIDR or dotted notation).
func isValidNetmask(netmask string) bool {
	// Check if it's a CIDR prefix (e.g., "24")
	if _, err := fmt.Sscanf(netmask, "%d", new(int)); err == nil {
		var prefix int
		fmt.Sscanf(netmask, "%d", &prefix)
		return prefix >= 0 && prefix <= 32
	}

	// Check if it's dotted notation (e.g., "255.255.255.0")
	ip := net.ParseIP(netmask)
	return ip != nil && ip.To4() != nil
}

// netmaskToCIDR converts a dotted netmask to CIDR prefix length.
func netmaskToCIDR(netmask string) (int, error) {
	// If already a number, return it
	var prefix int
	if _, err := fmt.Sscanf(netmask, "%d", &prefix); err == nil {
		return prefix, nil
	}

	// Parse dotted notation
	ip := net.ParseIP(netmask)
	if ip == nil {
		return 0, fmt.Errorf("invalid netmask: %s", netmask)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("invalid IPv4 netmask: %s", netmask)
	}

	// Count bits
	ones, _ := net.IPv4Mask(ip4[0], ip4[1], ip4[2], ip4[3]).Size()
	return ones, nil
}

// configureStaticIPLinux applies static IP on Linux using ip command.
func configureStaticIPLinux(iface string, cfg *StaticIPConfig) error {
	// Validate interface name to prevent command injection
	if err := validation.ValidateInterface(iface); err != nil {
		return fmt.Errorf("invalid interface: %w", err)
	}

	// Validate IP address
	if !validation.IsValidIPv4(cfg.Address) {
		return fmt.Errorf("invalid IP address: %s", cfg.Address)
	}

	// Validate gateway if provided
	if cfg.Gateway != "" && !validation.IsValidIPv4(cfg.Gateway) {
		return fmt.Errorf("invalid gateway address: %s", cfg.Gateway)
	}

	prefix, err := netmaskToCIDR(cfg.Netmask)
	if err != nil {
		return err
	}

	// Flush existing addresses
	if err := exec.Command("ip", "addr", "flush", "dev", iface).Run(); err != nil {
		return fmt.Errorf("failed to flush addresses: %w", err)
	}

	// Add new address
	addr := fmt.Sprintf("%s/%d", cfg.Address, prefix)
	if err := exec.Command("ip", "addr", "add", addr, "dev", iface).Run(); err != nil {
		return fmt.Errorf("failed to add address: %w", err)
	}

	// Bring interface up
	if err := exec.Command("ip", "link", "set", iface, "up").Run(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	// Add default gateway if provided
	if cfg.Gateway != "" {
		// Remove existing default route first (ignore errors)
		exec.Command("ip", "route", "del", "default").Run()

		if err := exec.Command("ip", "route", "add", "default", "via", cfg.Gateway, "dev", iface).Run(); err != nil {
			return fmt.Errorf("failed to add default gateway: %w", err)
		}
	}

	// Configure DNS (update /etc/resolv.conf)
	if len(cfg.DNS) > 0 {
		if err := updateResolvConf(cfg.DNS); err != nil {
			return fmt.Errorf("failed to configure DNS: %w", err)
		}
	}

	return nil
}

// configureDHCPLinux switches to DHCP on Linux.
func configureDHCPLinux(iface string) error {
	// Validate interface name to prevent command injection
	if err := validation.ValidateInterface(iface); err != nil {
		return fmt.Errorf("invalid interface: %w", err)
	}

	// Try dhclient first
	if err := exec.Command("dhclient", "-r", iface).Run(); err == nil {
		return exec.Command("dhclient", iface).Run()
	}

	// Try dhcpcd
	if err := exec.Command("dhcpcd", "-k", iface).Run(); err == nil {
		return exec.Command("dhcpcd", iface).Run()
	}

	return fmt.Errorf("no DHCP client found (tried dhclient, dhcpcd)")
}

// configureStaticIPDarwin applies static IP on macOS using networksetup.
func configureStaticIPDarwin(iface string, cfg *StaticIPConfig) error {
	// Get the network service name for the interface
	service, err := getNetworkServiceName(iface)
	if err != nil {
		return err
	}

	// Convert CIDR to dotted netmask if needed
	netmask := cfg.Netmask
	if net.ParseIP(netmask) == nil {
		// It's a CIDR prefix, convert to dotted
		var prefix int
		fmt.Sscanf(netmask, "%d", &prefix)
		netmask = cidrToNetmask(prefix)
	}

	// Set manual IP
	args := []string{"-setmanual", service, cfg.Address, netmask}
	if cfg.Gateway != "" {
		args = append(args, cfg.Gateway)
	}
	if err := exec.Command("networksetup", args...).Run(); err != nil {
		return fmt.Errorf("failed to set static IP: %w", err)
	}

	// Configure DNS if provided
	if len(cfg.DNS) > 0 {
		dnsArgs := append([]string{"-setdnsservers", service}, cfg.DNS...)
		if err := exec.Command("networksetup", dnsArgs...).Run(); err != nil {
			return fmt.Errorf("failed to configure DNS: %w", err)
		}
	}

	return nil
}

// configureDHCPDarwin switches to DHCP on macOS.
func configureDHCPDarwin(iface string) error {
	service, err := getNetworkServiceName(iface)
	if err != nil {
		return err
	}

	return exec.Command("networksetup", "-setdhcp", service).Run()
}

// getNetworkServiceName gets the macOS network service name for an interface.
func getNetworkServiceName(iface string) (string, error) {
	// Common mappings
	serviceNames := map[string]string{
		"en0": "Ethernet",
		"en1": "Wi-Fi",
	}

	if name, ok := serviceNames[iface]; ok {
		return name, nil
	}

	// Try to find the service by listing all
	output, err := exec.Command("networksetup", "-listnetworkserviceorder").Output()
	if err != nil {
		return "", fmt.Errorf("failed to list network services: %w", err)
	}

	// Parse output looking for our interface
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Device: "+iface) && i > 0 {
			// Service name is in the previous line
			prev := lines[i-1]
			// Format: "(1) Service Name"
			if idx := strings.Index(prev, ") "); idx != -1 {
				return strings.TrimSpace(prev[idx+2:]), nil
			}
		}
	}

	return "", fmt.Errorf("network service not found for interface %s", iface)
}

// cidrToNetmask converts a CIDR prefix to dotted decimal netmask.
func cidrToNetmask(prefix int) string {
	mask := net.CIDRMask(prefix, 32)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

// updateResolvConf updates /etc/resolv.conf with new DNS servers.
func updateResolvConf(servers []string) error {
	var content strings.Builder
	content.WriteString("# Generated by NetScope\n")
	for _, server := range servers {
		content.WriteString(fmt.Sprintf("nameserver %s\n", server))
	}
	return os.WriteFile("/etc/resolv.conf", []byte(content.String()), 0644)
}
