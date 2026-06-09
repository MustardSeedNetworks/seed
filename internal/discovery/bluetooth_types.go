package discovery

// bluetooth_types.go holds the Bluetooth result/data types that stay in the
// discovery kernel (ADR-0018, Phase 6 enumerate split). The Bluetooth scanner
// logic lives in internal/discovery/enumerate, but these types remain here
// because they are referenced by kernel-resident code: the enumerate-stage
// converter (bluetoothDeviceToDevice in stages.go), the DiscoveredDevice
// BluetoothPresence field (devices_types.go), and the BluetoothCollectorPort
// interface the Engine drives. Moving them would invert the kernel→stage
// direction and create an import cycle. The enumerate package aliases them.

import "time"

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
	DeviceID string `json:"deviceId,omitempty"` // Links to DiscoveredDevice if correlated

	// Identity
	Address     string `json:"address"`     // MAC address (AA:BB:CC:DD:EE:FF)
	Name        string `json:"name"`        // Advertised device name
	Alias       string `json:"alias"`       // User-assigned alias
	Vendor      string `json:"vendor"`      // OUI vendor lookup
	IsConnected bool   `json:"isConnected"` // Currently connected to this host

	// Classification
	Type        BluetoothType        `json:"type"`                    // classic, ble, dual
	DeviceClass BluetoothDeviceClass `json:"deviceClass"`             // Major device class
	Appearance  uint16               `json:"appearance"`              // BLE appearance value
	ClassOfDev  uint32               `json:"classOfDevice,omitempty"` // Classic CoD

	// Signal
	RSSI         int     `json:"rssi"`         // Signal strength in dBm
	TxPower      int     `json:"txPower"`      // Advertised TX power (BLE)
	EstDistanceM float64 `json:"estDistanceM"` // Estimated distance in meters

	// BLE-specific
	IsConnectable    bool     `json:"isConnectable"`
	ServiceUUIDs     []string `json:"serviceUuids,omitempty"`
	ManufacturerID   uint16   `json:"manufacturerId,omitempty"`
	ManufacturerData []byte   `json:"manufacturerData,omitempty"`

	// Authorization
	IsAuthorized bool `json:"isAuthorized"`
	IsTrusted    bool `json:"isTrusted"`
	IsPaired     bool `json:"isPaired"`
	IsBlocked    bool `json:"isBlocked"`

	// Timestamps
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`

	Metadata map[string]any `json:"metadata,omitempty"`
}

// BluetoothScanResult contains results from a Bluetooth scan.
type BluetoothScanResult struct {
	Devices      []BluetoothDevice `json:"devices"`
	ScanTime     time.Time         `json:"scanTime"`
	ScanDuration time.Duration     `json:"scanDuration"`
	AdapterName  string            `json:"adapterName"`
	ScanType     string            `json:"scanType"` // "passive", "active"
}

// BluetoothDiscoveryStats contains aggregated Bluetooth discovery statistics.
type BluetoothDiscoveryStats struct {
	TotalDevices      int            `json:"totalDevices"`
	ClassicDevices    int            `json:"classicDevices"`
	BLEDevices        int            `json:"bleDevices"`
	DualDevices       int            `json:"dualDevices"`
	ConnectedDevices  int            `json:"connectedDevices"`
	AuthorizedCount   int            `json:"authorizedCount"`
	UnauthorizedCount int            `json:"unauthorizedCount"`
	DevicesByClass    map[string]int `json:"devicesByClass"`
	VendorBreakdown   map[string]int `json:"vendorBreakdown"`
	LastScanTime      time.Time      `json:"lastScanTime"`
}
