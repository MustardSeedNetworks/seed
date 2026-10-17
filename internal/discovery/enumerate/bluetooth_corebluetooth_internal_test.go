package enumerate

import (
	"reflect"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

func TestDevicesFromPeripherals(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	tx := -8
	got := devicesFromPeripherals([]wifihelper.Peripheral{
		{
			Identifier:  "5A1C0F2E-6B4D-4E8A-9C1F-3D2B7A6E9F01",
			Name:        "Tag",
			RSSI:        -61,
			Connectable: true,
			TxPower:     &tx,
			// Apple's company identifier 0x004C, little-endian, then an
			// iBeacon prefix.
			ManufacturerData: []byte{0x4c, 0x00, 0x02, 0x15},
			ServiceUUIDs:     []string{"180F", "6E400001-B5A3-F393-E0A9-E50E24DCCA9E", "180f"},
		},
		// No TX power, no manufacturer field: both stay unset rather than
		// reading as 0 dBm or company 0 (Ericsson).
		{Identifier: "0B7E2C44-1F3A-4C2D-8E5B-6A9D0C1E2F33", RSSI: -88},
	}, now)

	want := []BluetoothDevice{
		{
			Address:          "5A1C0F2E-6B4D-4E8A-9C1F-3D2B7A6E9F01",
			Name:             "Tag",
			Type:             BluetoothTypeBLE,
			DeviceClass:      BluetoothClassUncategorized,
			RSSI:             -61,
			TxPower:          -8,
			IsConnectable:    true,
			ServiceUUIDs:     []string{"180f", "6e400001-b5a3-f393-e0a9-e50e24dcca9e"},
			ManufacturerID:   0x004c,
			ManufacturerData: []byte{0x02, 0x15},
			FirstSeen:        now,
			LastSeen:         now,
			Metadata:         map[string]any{},
		},
		{
			Address:     "0B7E2C44-1F3A-4C2D-8E5B-6A9D0C1E2F33",
			Type:        BluetoothTypeBLE,
			DeviceClass: BluetoothClassUncategorized,
			RSSI:        -88,
			FirstSeen:   now,
			LastSeen:    now,
			Metadata:    map[string]any{},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("devicesFromPeripherals() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestBLEScanDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seconds int
		want    time.Duration
	}{
		{"configured", 15, 15 * time.Second},
		{"unset takes the default", 0, defaultScanDurationSec * time.Second},
		{"negative takes the default", -3, defaultScanDurationSec * time.Second},
		// The helper refuses a longer scan, which would fail every scan.
		{"held to the helper's limit", 600, wifihelper.MaxBLEScan},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := bleScanDuration(&BluetoothScanConfig{ScanDurationSec: tc.seconds}); got != tc.want {
				t.Errorf("bleScanDuration(%d s) = %s, want %s", tc.seconds, got, tc.want)
			}
		})
	}
}
