// Package network handles network interface management, monitoring, and configuration.
// Provides cross-platform interface enumeration, property detection (type, speed, duplex),
// and platform-specific implementations for Linux and macOS interface introspection.
package network

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network/detection"
)

// InterfaceType represents the type of network interface.
type InterfaceType string

// Network interface type constants.
const (
	InterfaceTypeEthernet InterfaceType = "ethernet"
	InterfaceTypeWiFi     InterfaceType = "wifi"
	InterfaceTypeLoopback InterfaceType = "loopback"
	InterfaceTypeVirtual  InterfaceType = "virtual"
	InterfaceTypeOther    InterfaceType = "other"
)

// ipv4BitLength is the number of bits in an IPv4 address (32 bits).
// Used for CIDR mask calculations and netmask validation.
const ipv4BitLength = 32

// InterfaceInfo contains information about a network interface.
type InterfaceInfo struct {
	Name          string        `json:"name"`
	FriendlyName  string        `json:"friendlyName,omitempty"` // Human-readable name (e.g., "Intel I225-V")
	Description   string        `json:"description,omitempty"`  // Brief description (e.g., "2.5 Gbps Ethernet")
	Type          InterfaceType `json:"type"`
	Up            bool          `json:"up"`
	Running       bool          `json:"running"`
	HardwareAddr  string        `json:"hardwareAddr"`
	MTU           int           `json:"mtu"`
	Addresses     []string      `json:"addresses"`
	Speed         int64         `json:"speed,omitempty"`         // Speed in bits per second
	SpeedDisplay  string        `json:"speedDisplay,omitempty"`  // Human-readable speed (e.g., "2.5 Gbps")
	ChipsetVendor string        `json:"chipsetVendor,omitempty"` // NIC vendor (e.g., "Intel")
	ChipsetModel  string        `json:"chipsetModel,omitempty"`  // NIC model (e.g., "I225-V")
	HasTDR        bool          `json:"hasTDR,omitempty"`        // Supports cable diagnostics
	HasDOM        bool          `json:"hasDOM,omitempty"`        // Supports fiber optics monitoring
	Score         int           `json:"score,omitempty"`         // Detection score for auto-selection
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
	detector         *detection.Detector

	// Callback management for interface change notifications
	callbackMu sync.RWMutex
	callbacks  []InterfaceChangeCallback
}

// NewManager creates a new network manager.
func NewManager(defaultInterface string) (*Manager, error) {
	m := &Manager{
		currentInterface: defaultInterface,
		interfaces:       make(map[string]*InterfaceInfo),
		detector:         detection.NewDetector(),
	}
	if err := m.RefreshInterfaces(); err != nil {
		return nil, fmt.Errorf(
			"failed to refresh interfaces during manager initialization: %w",
			err,
		)
	}
	return m, nil
}

// RefreshInterfaces updates the list of available interfaces.
// Enriches interface information with detection data including friendly names,
// chipset info, TDR/DOM capabilities, and scoring for auto-selection.
func (m *Manager) RefreshInterfaces() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get interfaces: %w", err)
	}

	// Get enriched detection data for all interfaces
	detectedScores, err := m.detector.DetectAll()
	if err != nil {
		logging.GetLogger().Warn("interface detection failed", "error", err)
		// Continue with empty detection - graceful degradation
	}
	scoreMap := make(map[string]*detection.InterfaceScore)
	for i := range detectedScores {
		scoreMap[detectedScores[i].Name] = &detectedScores[i]
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
		addrs, addrErr := iface.Addrs()
		if addrErr == nil {
			for _, addr := range addrs {
				info.Addresses = append(info.Addresses, addr.String())
			}
		}

		// Enrich with detection data if available
		if score := scoreMap[iface.Name]; score != nil {
			info.FriendlyName = score.FriendlyName
			info.Description = score.Description
			info.Speed = score.Speed
			info.SpeedDisplay = score.SpeedDisplay
			info.ChipsetVendor = score.ChipsetVendor
			info.ChipsetModel = score.ChipsetModel
			info.HasTDR = score.HasTDR
			info.HasDOM = score.HasDOM
			info.Score = score.Score
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

// GetPhysicalInterfaces returns only physical network interfaces (ethernet and wifi).
// Excludes loopback, virtual, and other non-physical interfaces.
func (m *Manager) GetPhysicalInterfaces() []*InterfaceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*InterfaceInfo, 0, len(m.interfaces))
	for _, info := range m.interfaces {
		// Only include ethernet and wifi interfaces
		if info.Type == InterfaceTypeEthernet || info.Type == InterfaceTypeWiFi {
			result = append(result, info)
		}
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

// InterfaceChangeCallback is called when the active interface changes.
// #756: Used to notify modules to rebind when auto-detection switches interfaces.
type InterfaceChangeCallback func(oldInterface, newInterface string)

// OnInterfaceChange registers a callback to be notified when the active interface changes.
// #756: Modules use this to rebind when auto-detection switches interfaces.
func (m *Manager) OnInterfaceChange(callback InterfaceChangeCallback) {
	m.callbackMu.Lock()
	defer m.callbackMu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// notifyInterfaceChange notifies all registered callbacks of an interface change.
func (m *Manager) notifyInterfaceChange(oldInterface, newInterface string) {
	m.callbackMu.RLock()
	callbacks := make([]InterfaceChangeCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.callbackMu.RUnlock()

	for _, cb := range callbacks {
		go cb(oldInterface, newInterface)
	}
}

// AutoRedetect refreshes interfaces and auto-detects the best available interface.
// Returns the new interface name and whether it changed from the previous one.
// #756: Called when link state changes to automatically switch to a working interface.
func (m *Manager) AutoRedetect() (string, bool) {
	// Refresh the interface list first
	if err := m.RefreshInterfaces(); err != nil {
		logging.GetLogger().Warn("Failed to refresh interfaces during auto-redetect", "error", err)
	}

	m.mu.Lock()
	oldInterface := m.currentInterface

	// Find the best available interface (no preferred list - pure auto-detection)
	candidates := m.collectCandidates()
	newInterface := candidates.selectBest()

	if newInterface == "" {
		m.mu.Unlock()
		logging.GetLogger().Warn("Auto-redetect found no suitable interface")
		return oldInterface, false
	}

	changed := newInterface != oldInterface
	if changed {
		m.currentInterface = newInterface
		logging.GetLogger().Info("Auto-redetect switched interface (#756)",
			"old", oldInterface,
			"new", newInterface,
			"reason", "link state change or better interface available")
	}
	m.mu.Unlock()

	// Notify callbacks if interface changed
	if changed {
		m.notifyInterfaceChange(oldInterface, newInterface)
	}

	return newInterface, changed
}

// interfaceCandidates holds categorized interface names for selection.
type interfaceCandidates struct {
	ethernetWithIP, wifiWithIP, ethernetUp, wifiUp []string
}

// FindFirstAvailable finds the first available interface from a list.
// If no preferred interface is found, it auto-detects the best physical interface:
// Priority: Ethernet with IP > WiFi with IP > Ethernet up > WiFi up
// Virtual interfaces (docker, bridge, veth, etc.) are excluded from auto-detection.
func (m *Manager) FindFirstAvailable(preferred []string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, name := range preferred {
		if info, ok := m.interfaces[name]; ok && info.Up {
			return name
		}
	}

	candidates := m.collectCandidates()
	return candidates.selectBest()
}

// collectCandidates categorizes interfaces by type and connectivity.
func (m *Manager) collectCandidates() *interfaceCandidates {
	c := &interfaceCandidates{}
	for name, info := range m.interfaces {
		if info.Type == InterfaceTypeLoopback || info.Type == InterfaceTypeVirtual || !info.Up {
			continue
		}
		hasIP := hasRoutableAddress(info.Addresses)
		switch info.Type {
		case InterfaceTypeEthernet:
			if hasIP {
				c.ethernetWithIP = append(c.ethernetWithIP, name)
			} else {
				c.ethernetUp = append(c.ethernetUp, name)
			}
		case InterfaceTypeWiFi:
			if hasIP {
				c.wifiWithIP = append(c.wifiWithIP, name)
			} else {
				c.wifiUp = append(c.wifiUp, name)
			}
		case InterfaceTypeOther:
			if hasIP {
				c.ethernetWithIP = append(c.ethernetWithIP, name)
			}
		case InterfaceTypeLoopback, InterfaceTypeVirtual:
			// Already filtered
		}
	}
	return c
}

// selectBest returns the best interface in priority order.
func (c *interfaceCandidates) selectBest() string {
	if len(c.ethernetWithIP) > 0 {
		return c.ethernetWithIP[0]
	}
	if len(c.wifiWithIP) > 0 {
		return c.wifiWithIP[0]
	}
	if len(c.ethernetUp) > 0 {
		return c.ethernetUp[0]
	}
	if len(c.wifiUp) > 0 {
		return c.wifiUp[0]
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

	if data, err := os.ReadFile(speedPath); err == nil {
		speed := strings.TrimSpace(string(data))
		if speed != "" && speed != "-1" {
			status.Speed = speed + "Mb/s"
		}
	}

	// Try to read duplex from sysfs (Linux only)
	duplexPath := filepath.Join("sys", "class", "net", name, "duplex")
	duplexPath = string(os.PathSeparator) + duplexPath

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
		ipStr, _, _ := strings.Cut(addr, "/")
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
func getLinkInfoFromIfconfig(_ string) (string, string) {
	// This is a placeholder - actual implementation would exec ifconfig
	// and parse "media: autoselect (1000baseT <full-duplex>)"
	return "", ""
}

// detectInterfaceType determines the type of interface from its name.
func detectInterfaceType(name string) InterfaceType {
	// Loopback (lo, lo0, lo1, etc.)
	if name == "lo" || strings.HasPrefix(name, "lo") && len(name) <= 3 {
		// Check if remaining chars are digits (lo0, lo1, lo2)
		if len(name) == 2 || (len(name) == 3 && name[2] >= '0' && name[2] <= '9') {
			return InterfaceTypeLoopback
		}
	}

	// Virtual interfaces (docker, bridge, veth, tun, tap, virbr, etc.)
	virtualPrefixes := []string{
		"docker",
		"br-",
		"veth",
		"virbr",
		"tun",
		"tap",
		"vnet",
		"vmnet",
		"vboxnet",
		"utun",
	}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return InterfaceTypeVirtual
		}
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
		return errors.New("IP address is required")
	}
	if cfg.Netmask == "" {
		return errors.New("netmask is required")
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
		return prefix >= 0 && prefix <= ipv4BitLength
	}

	// Check if it's dotted notation (e.g., "255.255.255.0")
	ip := net.ParseIP(netmask)
	return ip != nil && ip.To4() != nil
}

// cidrToNetmask converts a CIDR prefix to dotted decimal netmask.
func cidrToNetmask(prefix int) string {
	mask := net.CIDRMask(prefix, ipv4BitLength)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}
