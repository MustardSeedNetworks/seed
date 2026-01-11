package discovery

// bluetooth.go extends the discovery system with Bluetooth/BLE device tracking.
// This integrates with the existing DiscoveredDevice system by correlating
// Bluetooth devices with their wired/WiFi counterparts where possible.
//
// Bluetooth discovery complements existing ARP/LLDP/WiFi discovery:
// - BluetoothDevice tracks classic Bluetooth and BLE devices
// - Supports device classification (phone, computer, audio, IoT, etc.)
// - Tracks signal strength for proximity estimation
// - Links to DiscoveredDevice when MAC correlation is possible

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/krisarmstrong/seed/internal/logging"
)

// BluetoothType represents the Bluetooth protocol type.
type BluetoothType string

const (
	BluetoothTypeClassic BluetoothType = "classic" // BR/EDR (Basic Rate/Enhanced Data Rate)
	BluetoothTypeBLE     BluetoothType = "ble"     // Bluetooth Low Energy
	BluetoothTypeDual    BluetoothType = "dual"    // Supports both
)

// BluetoothDeviceClass represents major device classes per Bluetooth spec.
type BluetoothDeviceClass string

const (
	BluetoothClassMisc          BluetoothDeviceClass = "miscellaneous"
	BluetoothClassComputer      BluetoothDeviceClass = "computer"
	BluetoothClassPhone         BluetoothDeviceClass = "phone"
	BluetoothClassLAN           BluetoothDeviceClass = "lan_access"
	BluetoothClassAudioVideo    BluetoothDeviceClass = "audio_video"
	BluetoothClassPeripheral    BluetoothDeviceClass = "peripheral"
	BluetoothClassImaging       BluetoothDeviceClass = "imaging"
	BluetoothClassWearable      BluetoothDeviceClass = "wearable"
	BluetoothClassToy           BluetoothDeviceClass = "toy"
	BluetoothClassHealth        BluetoothDeviceClass = "health"
	BluetoothClassUncategorized BluetoothDeviceClass = "uncategorized"
)

// BluetoothDevice represents a discovered Bluetooth device.
type BluetoothDevice struct {
	ID       string `json:"id"`
	DeviceID string `json:"device_id,omitempty"` // Links to DiscoveredDevice if correlated

	// Identity
	Address     string `json:"address"`      // MAC address (AA:BB:CC:DD:EE:FF)
	Name        string `json:"name"`         // Advertised device name
	Alias       string `json:"alias"`        // User-assigned alias
	Vendor      string `json:"vendor"`       // OUI vendor lookup
	IsConnected bool   `json:"is_connected"` // Currently connected to this host

	// Classification
	Type        BluetoothType        `json:"type"`                      // classic, ble, dual
	DeviceClass BluetoothDeviceClass `json:"device_class"`              // Major device class
	Appearance  uint16               `json:"appearance"`                // BLE appearance value
	ClassOfDev  uint32               `json:"class_of_device,omitempty"` // Classic CoD

	// Signal
	RSSI         int     `json:"rssi"`           // Signal strength in dBm
	TxPower      int     `json:"tx_power"`       // Advertised TX power (BLE)
	EstDistanceM float64 `json:"est_distance_m"` // Estimated distance in meters

	// BLE-specific
	IsConnectable    bool     `json:"is_connectable"`
	ServiceUUIDs     []string `json:"service_uuids,omitempty"`
	ManufacturerID   uint16   `json:"manufacturer_id,omitempty"`
	ManufacturerData []byte   `json:"manufacturer_data,omitempty"`

	// Authorization
	IsAuthorized bool `json:"is_authorized"`
	IsTrusted    bool `json:"is_trusted"`
	IsPaired     bool `json:"is_paired"`
	IsBlocked    bool `json:"is_blocked"`

	// Timestamps
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`

	Metadata map[string]any `json:"metadata,omitempty"`
}

// BluetoothScanResult contains results from a Bluetooth scan.
type BluetoothScanResult struct {
	Devices      []BluetoothDevice `json:"devices"`
	ScanTime     time.Time         `json:"scan_time"`
	ScanDuration time.Duration     `json:"scan_duration"`
	AdapterName  string            `json:"adapter_name"`
	ScanType     string            `json:"scan_type"` // "passive", "active"
}

// BluetoothDiscoveryStats contains aggregated Bluetooth discovery statistics.
type BluetoothDiscoveryStats struct {
	TotalDevices      int            `json:"total_devices"`
	ClassicDevices    int            `json:"classic_devices"`
	BLEDevices        int            `json:"ble_devices"`
	DualDevices       int            `json:"dual_devices"`
	ConnectedDevices  int            `json:"connected_devices"`
	AuthorizedCount   int            `json:"authorized_count"`
	UnauthorizedCount int            `json:"unauthorized_count"`
	DevicesByClass    map[string]int `json:"devices_by_class"`
	VendorBreakdown   map[string]int `json:"vendor_breakdown"`
	LastScanTime      time.Time      `json:"last_scan_time"`
}

// BluetoothScanConfig configures Bluetooth scanning behavior.
type BluetoothScanConfig struct {
	// ScanDurationSec is how long to scan in seconds
	ScanDurationSec int `json:"scan_duration_sec" yaml:"scan_duration_sec"`

	// ScanType: "passive" (listen only) or "active" (send inquiries)
	ScanType string `json:"scan_type" yaml:"scan_type"`

	// IncludeClassic enables classic Bluetooth discovery
	IncludeClassic bool `json:"include_classic" yaml:"include_classic"`

	// IncludeBLE enables BLE scanning
	IncludeBLE bool `json:"include_ble" yaml:"include_ble"`

	// MinRSSI filters out devices below this signal strength
	MinRSSI int `json:"min_rssi" yaml:"min_rssi"`

	// AuthorizedAddresses lists MAC addresses to mark as authorized
	AuthorizedAddresses []string `json:"authorized_addresses" yaml:"authorized_addresses"`
}

// DefaultBluetoothScanConfig returns sensible defaults for Bluetooth scanning.
func DefaultBluetoothScanConfig() *BluetoothScanConfig {
	return &BluetoothScanConfig{
		ScanDurationSec:     10,
		ScanType:            "active",
		IncludeClassic:      true,
		IncludeBLE:          true,
		MinRSSI:             -90,
		AuthorizedAddresses: []string{},
	}
}

// BluetoothScanner discovers Bluetooth devices.
type BluetoothScanner struct {
	mu                sync.RWMutex
	adapterName       string
	config            *BluetoothScanConfig
	oui               *OUIDatabase
	lastScan          *BluetoothScanResult
	lastScanTime      time.Time
	authorizedDevices map[string]bool // Authorized MAC addresses
}

// NewBluetoothScanner creates a new Bluetooth scanner.
func NewBluetoothScanner(adapterName string, config *BluetoothScanConfig, oui *OUIDatabase) *BluetoothScanner {
	if config == nil {
		config = DefaultBluetoothScanConfig()
	}

	authorized := make(map[string]bool)
	for _, addr := range config.AuthorizedAddresses {
		authorized[normalizeMAC(addr)] = true
	}

	return &BluetoothScanner{
		adapterName:       adapterName,
		config:            config,
		oui:               oui,
		authorizedDevices: authorized,
	}
}

// SetAdapter updates the Bluetooth adapter to use.
func (s *BluetoothScanner) SetAdapter(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adapterName = name
}

// SetAuthorizedDevices sets the list of authorized Bluetooth addresses.
func (s *BluetoothScanner) SetAuthorizedDevices(addresses []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authorizedDevices = make(map[string]bool)
	for _, addr := range addresses {
		s.authorizedDevices[normalizeMAC(addr)] = true
	}
}

// Scan performs a Bluetooth scan and returns discovered devices.
func (s *BluetoothScanner) Scan(ctx context.Context) (*BluetoothScanResult, error) {
	s.mu.Lock()
	adapter := s.adapterName
	config := s.config
	s.mu.Unlock()

	logger := logging.GetLogger()
	logger.InfoContext(ctx, "Starting Bluetooth scan", "adapter", adapter)

	start := time.Now()

	// Perform platform-specific scan
	rawDevices, err := s.scanPlatform(ctx, adapter, config)
	if err != nil {
		return nil, fmt.Errorf("bluetooth scan failed: %w", err)
	}

	// Filter by minimum RSSI
	filteredDevices := make([]BluetoothDevice, 0, len(rawDevices))
	for _, dev := range rawDevices {
		if dev.RSSI >= config.MinRSSI {
			filteredDevices = append(filteredDevices, dev)
		}
	}

	// Enrich devices
	s.enrichDevices(filteredDevices)

	result := &BluetoothScanResult{
		Devices:      filteredDevices,
		ScanTime:     start,
		ScanDuration: time.Since(start),
		AdapterName:  adapter,
		ScanType:     config.ScanType,
	}

	// Cache result
	s.mu.Lock()
	s.lastScan = result
	s.lastScanTime = start
	s.mu.Unlock()

	logger.InfoContext(ctx, "Bluetooth scan complete",
		"devices", len(filteredDevices),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// GetLastScan returns the most recent scan result.
func (s *BluetoothScanner) GetLastScan() *BluetoothScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScan
}

// GetStats returns Bluetooth discovery statistics.
func (s *BluetoothScanner) GetStats() *BluetoothDiscoveryStats {
	s.mu.RLock()
	lastScan := s.lastScan
	lastTime := s.lastScanTime
	s.mu.RUnlock()

	if lastScan == nil {
		return &BluetoothDiscoveryStats{
			DevicesByClass:  make(map[string]int),
			VendorBreakdown: make(map[string]int),
		}
	}

	stats := &BluetoothDiscoveryStats{
		TotalDevices:    len(lastScan.Devices),
		DevicesByClass:  make(map[string]int),
		VendorBreakdown: make(map[string]int),
		LastScanTime:    lastTime,
	}

	for _, dev := range lastScan.Devices {
		switch dev.Type {
		case BluetoothTypeClassic:
			stats.ClassicDevices++
		case BluetoothTypeBLE:
			stats.BLEDevices++
		case BluetoothTypeDual:
			stats.DualDevices++
		}

		if dev.IsConnected {
			stats.ConnectedDevices++
		}

		if dev.IsAuthorized {
			stats.AuthorizedCount++
		} else {
			stats.UnauthorizedCount++
		}

		stats.DevicesByClass[string(dev.DeviceClass)]++
		if dev.Vendor != "" {
			stats.VendorBreakdown[dev.Vendor]++
		}
	}

	return stats
}

// enrichDevices adds vendor info and authorization status.
func (s *BluetoothScanner) enrichDevices(devices []BluetoothDevice) {
	s.mu.RLock()
	authorized := s.authorizedDevices
	s.mu.RUnlock()

	now := time.Now()

	for i := range devices {
		// Vendor lookup
		if s.oui != nil && devices[i].Vendor == "" {
			devices[i].Vendor = s.oui.Lookup(devices[i].Address)
		}

		// Check authorization
		normalized := normalizeMAC(devices[i].Address)
		if authorized[normalized] || devices[i].IsTrusted || devices[i].IsPaired {
			devices[i].IsAuthorized = true
		}

		// Generate ID if missing
		if devices[i].ID == "" {
			devices[i].ID = uuid.New().String()
		}

		// Set timestamps if not set
		if devices[i].FirstSeen.IsZero() {
			devices[i].FirstSeen = now
		}
		if devices[i].LastSeen.IsZero() {
			devices[i].LastSeen = now
		}

		// Estimate distance from RSSI and TX power
		if devices[i].TxPower != 0 && devices[i].RSSI != 0 {
			devices[i].EstDistanceM = estimateDistance(devices[i].TxPower, devices[i].RSSI)
		}
	}
}

// estimateDistance calculates approximate distance from RSSI using path loss model.
// Formula: distance = 10^((TxPower - RSSI) / (10 * n))
// where n is the path loss exponent (typically 2-4 for indoor environments).
func estimateDistance(txPower, rssi int) float64 {
	const pathLossExponent = 2.5 // Indoor environment estimate
	if rssi >= txPower {
		return 0.1 // Very close
	}
	ratio := float64(txPower-rssi) / (10 * pathLossExponent)
	distance := 1.0
	for i := 0; i < int(ratio); i++ {
		distance *= 10
	}
	// Interpolate for fractional part
	frac := ratio - float64(int(ratio))
	for f := 0.1; f <= frac; f += 0.1 {
		distance *= 1.258925 // 10^0.1
	}
	return distance
}

// ClassOfDeviceToClass converts Bluetooth Class of Device to our class enum.
func ClassOfDeviceToClass(cod uint32) BluetoothDeviceClass {
	majorClass := (cod >> 8) & 0x1F
	switch majorClass {
	case 0:
		return BluetoothClassMisc
	case 1:
		return BluetoothClassComputer
	case 2:
		return BluetoothClassPhone
	case 3:
		return BluetoothClassLAN
	case 4:
		return BluetoothClassAudioVideo
	case 5:
		return BluetoothClassPeripheral
	case 6:
		return BluetoothClassImaging
	case 7:
		return BluetoothClassWearable
	case 8:
		return BluetoothClassToy
	case 9:
		return BluetoothClassHealth
	default:
		return BluetoothClassUncategorized
	}
}

// BLEAppearanceToClass converts BLE appearance value to our class enum.
func BLEAppearanceToClass(appearance uint16) BluetoothDeviceClass {
	category := appearance >> 6 // High 10 bits are category
	switch category {
	case 0:
		return BluetoothClassMisc
	case 1:
		return BluetoothClassPhone
	case 2:
		return BluetoothClassComputer
	case 3:
		return BluetoothClassWearable // Watch
	case 4:
		return BluetoothClassWearable // Clock
	case 5:
		return BluetoothClassMisc // Display
	case 6:
		return BluetoothClassMisc // Remote Control
	case 7:
		return BluetoothClassMisc // Eye-glasses
	case 8:
		return BluetoothClassMisc // Tag
	case 9:
		return BluetoothClassPeripheral // Keyring
	case 10:
		return BluetoothClassMisc // Media player
	case 11:
		return BluetoothClassMisc // Barcode scanner
	case 12:
		return BluetoothClassHealth // Thermometer
	case 13:
		return BluetoothClassHealth // Heart rate
	case 14:
		return BluetoothClassHealth // Blood pressure
	case 15:
		return BluetoothClassHealth // HID
	case 16:
		return BluetoothClassHealth // Glucose
	case 17:
		return BluetoothClassHealth // Running/walking
	case 18:
		return BluetoothClassHealth // Cycling
	case 49:
		return BluetoothClassHealth // Pulse oximeter
	case 50:
		return BluetoothClassHealth // Weight scale
	case 51:
		return BluetoothClassHealth // Personal mobility
	case 52:
		return BluetoothClassHealth // Continuous glucose
	case 53:
		return BluetoothClassHealth // Insulin pump
	case 54:
		return BluetoothClassHealth // Medication delivery
	case 81:
		return BluetoothClassMisc // Outdoor sports
	default:
		return BluetoothClassUncategorized
	}
}
