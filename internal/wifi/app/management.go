package wifiapp

import (
	"errors"

	"github.com/krisarmstrong/seed/internal/wifi"
)

// ErrRadioUnavailable is returned by the management use-cases when the Wi-Fi
// manager (radio control surface) is not present. The handler maps it to a
// 503 Service Unavailable, preserving the pre-strangle behavior.
var ErrRadioUnavailable = errors.New("wifi manager not available")

// Hardware is the Wi-Fi radio + scan surface the management use-cases need. It
// is defined here at the consumer (ADR-0016 interface-segregation) and satisfied
// by an adapter over *wifi.Manager + *wifi.Scanner. The Available reporters let
// the use-case mirror the "adapter not present" degraded responses without the
// handler reaching into the service container.
type Hardware interface {
	ManagerAvailable() bool
	ScannerAvailable() bool
	IsWireless() bool
	SetInterface(name string)
	Scan() ([]*wifi.ScannedNetwork, error)
	Connect(ssid, password string) (*wifi.ConnectionResult, error)
	Disconnect() (*wifi.ConnectionResult, error)
}

// InterfaceLister enumerates the host's wireless interface names. Satisfied by
// an adapter over the network interface manager.
type InterfaceLister interface {
	WirelessInterfaceNames() []string
}

// InterfaceStore reads and persists the configured Wi-Fi interface selection.
// The port speaks only in strings so the use-case stays free of the config
// package; the adapter owns the locking and on-disk save.
type InterfaceStore interface {
	// ResolvedWiFiInterface returns the configured Wi-Fi interface, falling back
	// to the default interface when no dedicated Wi-Fi interface is set.
	ResolvedWiFiInterface() string
	// SaveWiFiInterface persists name as the Wi-Fi interface selection.
	SaveWiFiInterface(name string) error
}

// Management is the Wi-Fi settings/scan/status/connect use-case. Handlers depend
// on it instead of on the service container, the config, or the network manager.
type Management struct {
	hw     Hardware
	lister InterfaceLister
	store  InterfaceStore
}

// NewManagement builds the management use-case over its narrow dependencies.
func NewManagement(hw Hardware, lister InterfaceLister, store InterfaceStore) *Management {
	return &Management{hw: hw, lister: lister, store: store}
}

// SettingsResult is the Wi-Fi settings read model: the active interface, the
// available wireless interfaces, and whether the active one is wireless.
type SettingsResult struct {
	Interface     string
	AvailableWiFi []string
	IsWireless    bool
}

// Settings returns the current Wi-Fi settings.
func (m *Management) Settings() SettingsResult {
	return SettingsResult{
		Interface:     m.store.ResolvedWiFiInterface(),
		AvailableWiFi: m.lister.WirelessInterfaceNames(),
		IsWireless:    m.hw.ManagerAvailable() && m.hw.IsWireless(),
	}
}

// UpdateInterface sets and persists the Wi-Fi interface selection, also pointing
// the radio at the new interface when one is given.
func (m *Management) UpdateInterface(name string) error {
	if name != "" && m.hw.ManagerAvailable() {
		m.hw.SetInterface(name)
	}
	return m.store.SaveWiFiInterface(name)
}

// ScanResult is the neighbor-scan read model. Error is empty on success; when
// set it carries the user-facing reason the scan could not run or complete.
// Networks is always non-nil so it serializes as [] rather than null.
type ScanResult struct {
	Interface string
	Available bool
	Error     string
	Networks  []*wifi.ScannedNetwork
}

// Scan performs a neighbor-AP scan on the requested interface, falling back to
// the configured interface when requestedIface is empty. It degrades (never
// errors) to an Available=false / Error-bearing result when the scanner or a
// wireless adapter is absent, mirroring the pre-strangle handler.
func (m *Management) Scan(requestedIface string) ScanResult {
	iface := requestedIface
	if iface == "" {
		iface = m.store.ResolvedWiFiInterface()
	}
	empty := []*wifi.ScannedNetwork{}

	switch {
	case !m.hw.ScannerAvailable():
		return ScanResult{Interface: iface, Available: false, Error: "WiFi scanner not initialized", Networks: empty}
	case !m.hw.ManagerAvailable() || !m.hw.IsWireless():
		return ScanResult{
			Interface: iface,
			Available: false,
			Error:     "No wireless adapter available. Connect a WiFi adapter to scan networks.",
			Networks:  empty,
		}
	}

	networks, err := m.hw.Scan()
	if err != nil {
		return ScanResult{
			Interface: iface,
			Available: true,
			Error:     "Wi-Fi scan failed. Check permissions and interface availability.",
			Networks:  empty,
		}
	}
	return ScanResult{Interface: iface, Available: true, Networks: networks}
}

// StatusResult is the adapter-status read model returned by Status.
type StatusResult struct {
	Status            string
	Message           string
	CurrentInterface  string
	IsWireless        bool
	AvailableAdapters []string
	CanScan           bool
}

// Status reports the wireless adapter state without performing a scan, resolving
// the current interface the same way Scan does.
func (m *Management) Status(requestedIface string) StatusResult {
	adapters := m.lister.WirelessInterfaceNames()
	iface := requestedIface
	if iface == "" {
		iface = m.store.ResolvedWiFiInterface()
	}
	isWireless := m.hw.ManagerAvailable() && m.hw.IsWireless()

	var status, message string
	switch {
	case len(adapters) == 0:
		status = "unavailable"
		message = "No wireless adapter detected. Connect a WiFi adapter to perform surveys."
	case !isWireless:
		status = "available"
		message = "Wireless adapter available but not selected as current interface."
	default:
		status = "ready"
		message = "Wireless adapter ready for scanning."
	}

	return StatusResult{
		Status:            status,
		Message:           message,
		CurrentInterface:  iface,
		IsWireless:        isWireless,
		AvailableAdapters: adapters,
		CanScan:           isWireless && len(adapters) > 0,
	}
}

// Connect attempts to join an SSID. It returns ErrRadioUnavailable when the
// radio is absent; any other error is the underlying connect failure.
func (m *Management) Connect(ssid, password string) (*wifi.ConnectionResult, error) {
	if !m.hw.ManagerAvailable() {
		return nil, ErrRadioUnavailable
	}
	return m.hw.Connect(ssid, password)
}

// Disconnect drops the current association, returning ErrRadioUnavailable when
// the radio is absent.
func (m *Management) Disconnect() (*wifi.ConnectionResult, error) {
	if !m.hw.ManagerAvailable() {
		return nil, ErrRadioUnavailable
	}
	return m.hw.Disconnect()
}
