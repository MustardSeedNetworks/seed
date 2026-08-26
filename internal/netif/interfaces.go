// Package netif handles network interface management, monitoring, and configuration.
// Provides cross-platform interface enumeration, property detection (type, speed, duplex),
// and platform-specific implementations for Linux and macOS interface introspection.
package netif

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
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

	// Seams for ConfigureStaticIP's rollback path, so it can be tested without
	// reconfiguring a live interface. Nil means the real implementations.
	snapshotter configSnapshotter
	applier     configApplier
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
// For mock managers (detector is nil), this is a no-op to preserve pre-defined test data.
func (m *Manager) RefreshInterfaces() error {
	// Skip refresh for mock managers (detector is nil) to preserve pre-defined test data
	if m.detector == nil {
		return nil
	}

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
//
// The previous configuration is captured first and restored if any step of the
// apply fails. Each platform applies address, gateway and DNS in sequence and
// returns on the first error, so without this a failure partway through leaves
// the interface on its new address with its old default route — unreachable,
// and needing physical access to recover (#50).
//
// Snapshotting is best-effort: an interface with no current address has nothing
// to restore, and the apply proceeds without a rollback rather than being
// refused, since configuring an unconfigured interface is the ordinary case.
func (m *Manager) ConfigureStaticIP(iface string, cfg *StaticIPConfig) error {
	// Validate input
	if err := validateIPConfig(cfg); err != nil {
		return err
	}

	previous, err := m.snapshotFor().Snapshot(iface)
	if err != nil {
		previous = nil
	}

	return applyWithRollback(m.applierFor(), iface, cfg, previous)
}

// snapshotFor returns the configured snapshotter, defaulting to the real one.
func (m *Manager) snapshotFor() configSnapshotter {
	if m.snapshotter != nil {
		return m.snapshotter
	}
	return systemSnapshotter{manager: m}
}

// applierFor returns the configured applier, defaulting to the platform one.
func (m *Manager) applierFor() configApplier {
	if m.applier != nil {
		return m.applier
	}
	return platformApplier{}
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
	_, ok := parseNetmask(netmask)
	return ok
}

// parseNetmask converts any of the three forms an operator may reasonably type
// into a prefix length. All three are accepted everywhere:
//
//	"24"              a bare prefix
//	"/24"             a prefix with the separator, as it appears in CIDR
//	"255.255.255.0"   dotted decimal
//
// One parser, used by validation and by every platform's apply path, because
// the two disagreeing is how a netmask passes validation and then fails to
// apply. Both halves previously used [fmt.Sscanf] with "%d", which succeeds on
// "255.255.255.0" by reading 255 and stopping at the dot — so dotted masks were
// rejected as "invalid CIDR prefix: 255" despite being the documented form.
func parseNetmask(netmask string) (int, bool) {
	netmask = strings.TrimSpace(netmask)
	if netmask == "" {
		return 0, false
	}

	// strconv.Atoi, not fmt.Sscanf: Atoi requires the whole string to be the
	// number, which is what keeps a dotted mask out of this branch.
	if prefix, err := strconv.Atoi(strings.TrimPrefix(netmask, "/")); err == nil {
		if prefix < 0 || prefix > ipv4BitLength {
			return 0, false
		}
		return prefix, true
	}

	ip := net.ParseIP(netmask)
	if ip == nil {
		return 0, false
	}
	v4 := ip.To4()
	if v4 == nil {
		return 0, false
	}

	// A netmask is a contiguous run of ones. Size reports (0, 0) for a
	// non-contiguous mask such as 255.0.255.0, which is what rejects it.
	ones, bits := net.IPMask(v4).Size()
	if bits != ipv4BitLength {
		return 0, false
	}
	return ones, true
}

// cidrToNetmask converts a CIDR prefix to dotted decimal netmask.
func cidrToNetmask(prefix int) string {
	mask := net.CIDRMask(prefix, ipv4BitLength)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
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
