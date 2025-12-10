// Package network handles network interface management.
package network

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
func NewManager(defaultInterface string) (*Manager, error) {
	m := &Manager{
		currentInterface: defaultInterface,
		interfaces:       make(map[string]*InterfaceInfo),
	}
	if err := m.RefreshInterfaces(); err != nil {
		return nil, fmt.Errorf("failed to refresh interfaces during manager initialization: %w", err)
	}
	return m, nil
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
	carrier := info.Running                     // Physical link/carrier detected
	hasIP := hasRoutableAddress(info.Addresses) // Has routable IP address
	linkUp := carrier && hasIP                  // Legacy: both conditions met

	status := &LinkStatus{
		LinkUp:  linkUp,
		Carrier: carrier,
		HasIP:   hasIP,
	}

	// Try to read speed from sysfs (Linux only)
	speedPath := filepath.Join("sys", "class", "net", name, "speed")
	speedPath = string(os.PathSeparator) + speedPath
	//nolint:gosec // G304: speedPath is constructed from validated interface name
	if data, err := os.ReadFile(speedPath); err == nil {
		speed := strings.TrimSpace(string(data))
		if speed != "" && speed != "-1" {
			status.Speed = speed + "Mb/s"
		}
	}

	// Try to read duplex from sysfs (Linux only)
	duplexPath := filepath.Join("sys", "class", "net", name, "duplex")
	duplexPath = string(os.PathSeparator) + duplexPath
	//nolint:gosec // G304: duplexPath is constructed from validated interface name
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

	// Get ethtool settings (autoneg, advertised modes) on Linux
	autoNeg, advertised := getEthtoolSettings(name)
	status.AutoNeg = autoNeg
	status.Advertised = advertised

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
// Implementation is platform-specific (interfaces_linux.go, interfaces_darwin.go).
func (m *Manager) ConfigureStaticIP(iface string, cfg *StaticIPConfig) error {
	// Validate input
	if err := validateIPConfig(cfg); err != nil {
		return err
	}

	return configureStaticIPPlatform(iface, cfg)
}

// ConfigureDHCP switches an interface to DHCP mode.
// Requires root/administrator privileges.
// Implementation is platform-specific (interfaces_linux.go, interfaces_darwin.go).
func (m *Manager) ConfigureDHCP(iface string) error {
	return configureDHCPPlatform(iface)
}

// SetMTU sets the MTU (Maximum Transmission Unit) for an interface.
// Valid MTU range is typically 68-9000 (Ethernet jumbo frames).
// Requires root/administrator privileges.
// Implementation is platform-specific (interfaces_linux.go, interfaces_darwin.go).
func (m *Manager) SetMTU(iface string, mtu int) error {
	// Validate MTU range
	if mtu < 68 || mtu > 9000 {
		return fmt.Errorf("invalid MTU %d: must be between 68 and 9000", mtu)
	}

	return setMTUPlatform(iface, mtu)
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
	var prefix int
	_, err := fmt.Sscanf(netmask, "%d", &prefix)
	if err == nil {
		return prefix >= 0 && prefix <= 32
	}

	// Check if it's dotted notation (e.g., "255.255.255.0")
	ip := net.ParseIP(netmask)
	return ip != nil && ip.To4() != nil
}

// cidrToNetmask converts a CIDR prefix to dotted decimal netmask.
func cidrToNetmask(prefix int) string {
	mask := net.CIDRMask(prefix, 32)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}
