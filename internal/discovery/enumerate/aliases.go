package enumerate

import (
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// Kernel type aliases. The Bluetooth result/data types live in the discovery
// kernel (they sit in the BluetoothCollectorPort signatures and the
// DiscoveredDevice BluetoothPresence field, so they must live there). Aliasing
// them here lets the collector logic reference them unqualified without
// duplicating the definitions or inverting the dependency direction.
type (
	// BluetoothType is the kernel Bluetooth protocol-type enum.
	BluetoothType = discovery.BluetoothType
	// BluetoothDeviceClass is the kernel major-device-class enum.
	BluetoothDeviceClass = discovery.BluetoothDeviceClass
	// BluetoothDevice is a discovered Bluetooth device (kernel result type).
	BluetoothDevice = discovery.BluetoothDevice
	// BluetoothScanResult is a complete Bluetooth scan result (kernel result type).
	BluetoothScanResult = discovery.BluetoothScanResult
	// BluetoothDiscoveryStats is aggregated Bluetooth statistics (kernel result type).
	BluetoothDiscoveryStats = discovery.BluetoothDiscoveryStats
)

// Kernel constant aliases, re-exported so the collector logic references them
// unqualified.
const (
	BluetoothTypeClassic = discovery.BluetoothTypeClassic
	BluetoothTypeBLE     = discovery.BluetoothTypeBLE
	BluetoothTypeDual    = discovery.BluetoothTypeDual

	BluetoothClassMisc          = discovery.BluetoothClassMisc
	BluetoothClassComputer      = discovery.BluetoothClassComputer
	BluetoothClassPhone         = discovery.BluetoothClassPhone
	BluetoothClassLAN           = discovery.BluetoothClassLAN
	BluetoothClassAudioVideo    = discovery.BluetoothClassAudioVideo
	BluetoothClassPeripheral    = discovery.BluetoothClassPeripheral
	BluetoothClassImaging       = discovery.BluetoothClassImaging
	BluetoothClassWearable      = discovery.BluetoothClassWearable
	BluetoothClassToy           = discovery.BluetoothClassToy
	BluetoothClassHealth        = discovery.BluetoothClassHealth
	BluetoothClassUncategorized = discovery.BluetoothClassUncategorized
)

// normalizeMAC canonicalizes a MAC address to upper-case colon-separated form.
// The enumerate stage keeps its own private copy: the kernel's normalizeMAC
// (in wifi.go) is unexported and shared by kernel-resident code, so it cannot
// be referenced across the package boundary. A later wifi relocation will lift
// the kernel definition into a dedicated mac.go.
func normalizeMAC(mac string) string {
	return strings.ToUpper(strings.ReplaceAll(mac, "-", ":"))
}
