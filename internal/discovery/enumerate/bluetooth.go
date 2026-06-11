// Package enumerate is the discovery pipeline's host/link enumeration stage
// (ADR-0018, Phase 6): the device collectors the Engine drives via narrow kernel
// ports (BluetoothCollectorPort and siblings). It depends inward on the discovery
// kernel for the shared Bluetooth result/data types and on
// internal/discovery/resolve for OUI lookups; the kernel never imports this
// package (depguard-enforced one-way direction).
package enumerate

// bluetooth.go extends the discovery system with Bluetooth/BLE device tracking.
// This integrates with the existing DiscoveredDevice system by correlating
// Bluetooth devices with their wired/WiFi counterparts where possible.
//
// Bluetooth discovery complements existing ARP/LLDP/WiFi discovery:
// - BluetoothDevice tracks classic Bluetooth and BLE devices
// - Supports device classification (phone, computer, audio, IoT, etc.)
// - Tracks signal strength for proximity estimation
// - Links to DiscoveredDevice when MAC correlation is possible
//
// The Bluetooth result/data types (BluetoothDevice, BluetoothScanResult,
// BluetoothDiscoveryStats, BluetoothType, BluetoothDeviceClass) live in the
// discovery kernel and are aliased here via aliases.go.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Bluetooth configuration constants.
const (
	// defaultScanDurationSec is the default Bluetooth scan duration in seconds.
	defaultScanDurationSec = 10
	// defaultMinRSSI is the default minimum RSSI threshold for device detection.
	defaultMinRSSI = -90
	// closeProximityDistance is the distance returned when device is very close.
	closeProximityDistance = 0.1
	// pathLossMultiplier is the multiplier for path loss calculation (10 * n).
	pathLossMultiplier = 10
	// codMajorClassMask is the mask for extracting major class from Class of Device.
	codMajorClassMask = 0x1F
	// codMajorClassShift is the bit shift for major class in Class of Device.
	codMajorClassShift = 8
)

// Bluetooth major device class constants per Bluetooth spec.
const (
	btMajorClassMisc       = 0
	btMajorClassComputer   = 1
	btMajorClassPhone      = 2
	btMajorClassLAN        = 3
	btMajorClassAudioVideo = 4
	btMajorClassPeripheral = 5
	btMajorClassImaging    = 6
	btMajorClassWearable   = 7
	btMajorClassToy        = 8
	btMajorClassHealth     = 9
)

// BluetoothScanConfig configures Bluetooth scanning behavior. JSON keys are
// camelCase (API convention); YAML keys stay snake_case (config-file convention),
// matching the repo pattern in scan_config.go.
type BluetoothScanConfig struct {
	// ScanDurationSec is how long to scan in seconds
	ScanDurationSec int `json:"scanDurationSec" yaml:"scan_duration_sec"`

	// ScanType: "passive" (listen only) or "active" (send inquiries)
	ScanType string `json:"scanType" yaml:"scan_type"`

	// IncludeClassic enables classic Bluetooth discovery
	IncludeClassic bool `json:"includeClassic" yaml:"include_classic"`

	// IncludeBLE enables BLE scanning
	IncludeBLE bool `json:"includeBle" yaml:"include_ble"`

	// MinRSSI filters out devices below this signal strength
	MinRSSI int `json:"minRssi" yaml:"min_rssi"`

	// AuthorizedAddresses lists MAC addresses to mark as authorized
	AuthorizedAddresses []string `json:"authorizedAddresses" yaml:"authorized_addresses"`
}

// DefaultBluetoothScanConfig returns sensible defaults for Bluetooth scanning.
func DefaultBluetoothScanConfig() *BluetoothScanConfig {
	return &BluetoothScanConfig{
		ScanDurationSec:     defaultScanDurationSec,
		ScanType:            "active",
		IncludeClassic:      true,
		IncludeBLE:          true,
		MinRSSI:             defaultMinRSSI,
		AuthorizedAddresses: []string{},
	}
}

// BluetoothScanner discovers Bluetooth devices.
type BluetoothScanner struct {
	mu                sync.RWMutex
	adapterName       string
	config            *BluetoothScanConfig
	oui               *resolve.OUIDatabase
	lastScan          *BluetoothScanResult
	lastScanTime      time.Time
	authorizedDevices map[string]bool // Authorized MAC addresses
}

// NewBluetoothScanner creates a new Bluetooth scanner.
func NewBluetoothScanner(adapterName string, config *BluetoothScanConfig, oui *resolve.OUIDatabase) *BluetoothScanner {
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
		return closeProximityDistance // Very close
	}
	ratio := float64(txPower-rssi) / (pathLossMultiplier * pathLossExponent)
	distance := 1.0
	for range int(ratio) {
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
	majorClass := (cod >> codMajorClassShift) & codMajorClassMask
	switch majorClass {
	case btMajorClassMisc:
		return BluetoothClassMisc
	case btMajorClassComputer:
		return BluetoothClassComputer
	case btMajorClassPhone:
		return BluetoothClassPhone
	case btMajorClassLAN:
		return BluetoothClassLAN
	case btMajorClassAudioVideo:
		return BluetoothClassAudioVideo
	case btMajorClassPeripheral:
		return BluetoothClassPeripheral
	case btMajorClassImaging:
		return BluetoothClassImaging
	case btMajorClassWearable:
		return BluetoothClassWearable
	case btMajorClassToy:
		return BluetoothClassToy
	case btMajorClassHealth:
		return BluetoothClassHealth
	default:
		return BluetoothClassUncategorized
	}
}

// getBLEAppearanceCategoryMap returns the BLE appearance categories to device classes mapping.
func getBLEAppearanceCategoryMap() map[uint16]BluetoothDeviceClass {
	return map[uint16]BluetoothDeviceClass{
		0:  BluetoothClassMisc,       // Generic
		1:  BluetoothClassPhone,      // Phone
		2:  BluetoothClassComputer,   // Computer
		3:  BluetoothClassWearable,   // Watch
		4:  BluetoothClassWearable,   // Clock
		5:  BluetoothClassMisc,       // Display
		6:  BluetoothClassMisc,       // Remote Control
		7:  BluetoothClassMisc,       // Eye-glasses
		8:  BluetoothClassMisc,       // Tag
		9:  BluetoothClassPeripheral, // Keyring
		10: BluetoothClassMisc,       // Media player
		11: BluetoothClassMisc,       // Barcode scanner
		12: BluetoothClassHealth,     // Thermometer
		13: BluetoothClassHealth,     // Heart rate
		14: BluetoothClassHealth,     // Blood pressure
		15: BluetoothClassHealth,     // HID
		16: BluetoothClassHealth,     // Glucose
		17: BluetoothClassHealth,     // Running/walking
		18: BluetoothClassHealth,     // Cycling
		49: BluetoothClassHealth,     // Pulse oximeter
		50: BluetoothClassHealth,     // Weight scale
		51: BluetoothClassHealth,     // Personal mobility
		52: BluetoothClassHealth,     // Continuous glucose
		53: BluetoothClassHealth,     // Insulin pump
		54: BluetoothClassHealth,     // Medication delivery
		81: BluetoothClassMisc,       // Outdoor sports
	}
}

// bleAppearanceCategoryBits is the number of bits to shift for BLE appearance category.
const bleAppearanceCategoryBits = 6

// BLEAppearanceToClass converts BLE appearance value to our class enum.
func BLEAppearanceToClass(appearance uint16) BluetoothDeviceClass {
	category := appearance >> bleAppearanceCategoryBits // High 10 bits are category
	if class, ok := getBLEAppearanceCategoryMap()[category]; ok {
		return class
	}
	return BluetoothClassUncategorized
}
