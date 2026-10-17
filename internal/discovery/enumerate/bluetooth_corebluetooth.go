package enumerate

// bluetooth_corebluetooth.go maps the LE devices a macOS CoreBluetooth scan
// heard onto BluetoothDevice. The scan itself runs in the Seed Wi-Fi Helper
// (bluetooth_darwin.go); the mapping has no platform dependency, so it is
// tested everywhere.

import (
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

// bleScanDuration is how long the helper listens: the configured duration,
// held to the helper's limit.
func bleScanDuration(config *BluetoothScanConfig) time.Duration {
	d := time.Duration(config.ScanDurationSec) * time.Second
	if d <= 0 {
		d = defaultScanDurationSec * time.Second
	}
	return min(d, wifihelper.MaxBLEScan)
}

// devicesFromPeripherals keys each device on its CoreBluetooth identifier.
// macOS never reveals an LE device's Bluetooth address, and the identifier is
// stable on this Mac, so it is what keeps a device's identity across scans. It
// leaves the address type empty: the identifier is neither a public nor a
// random address.
func devicesFromPeripherals(peripherals []wifihelper.Peripheral, now time.Time) []BluetoothDevice {
	devices := make([]BluetoothDevice, 0, len(peripherals))
	for _, p := range peripherals {
		dev := BluetoothDevice{
			Address:       p.Identifier,
			Name:          p.Name,
			Type:          BluetoothTypeBLE,
			DeviceClass:   BluetoothClassUncategorized,
			RSSI:          p.RSSI,
			IsConnectable: p.Connectable,
			FirstSeen:     now,
			LastSeen:      now,
			Metadata:      make(map[string]any),
		}
		if p.TxPower != nil {
			dev.TxPower = *p.TxPower
		}
		// CoreBluetooth spells UUIDs in upper case; the advertising-data
		// decoder, which every other platform uses, writes lower case.
		for _, u := range p.ServiceUUIDs {
			appendServiceUUID(&dev, strings.ToLower(u))
		}
		// The manufacturer field arrives as advertised: company identifier
		// first, so the advertising-data decoder reads it unchanged.
		applyADField(&dev, adTypeManufacturerData, p.ManufacturerData)
		devices = append(devices, dev)
	}
	return devices
}
