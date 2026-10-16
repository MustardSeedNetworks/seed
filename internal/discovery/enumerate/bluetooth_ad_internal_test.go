package enumerate

import (
	"encoding/hex"
	"slices"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestApplyAdvertisingData decodes payloads field by field. The first two are
// advertisements heard on dev-srv-ubuntu's hci0 on 2026-10-16.
func TestApplyAdvertisingData(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		want BluetoothDevice
	}{
		{
			name: "flags, TX power and Apple manufacturer data",
			data: "02011a020a080bff4c0010063f1e18958e24",
			want: BluetoothDevice{
				TxPower:          8,
				ManufacturerID:   0x004C,
				ManufacturerData: []byte{0x10, 0x06, 0x3f, 0x1e, 0x18, 0x95, 0x8e, 0x24},
			},
		},
		{
			name: "negative TX power",
			data: "020af4",
			want: BluetoothDevice{TxPower: -12},
		},
		{
			name: "16-bit and 128-bit service UUIDs, deduplicated",
			data: "05030f180a18" + "0302" + "0f18" +
				"1107" + "fb349b5f80000080001000000d180000",
			want: BluetoothDevice{ServiceUUIDs: []string{
				"180f", "180a", "0000180d-0000-1000-8000-00805f9b34fb",
			}},
		},
		{
			name: "complete name replaces a short name",
			data: "040853656e" + "0a0953656e736f72203132",
			want: BluetoothDevice{Name: "Sensor 12"},
		},
		{
			name: "short name does not replace a complete name",
			data: "0a0953656e736f72203132" + "040853656e",
			want: BluetoothDevice{Name: "Sensor 12"},
		},
		{
			name: "classic class of device: phone",
			data: "040d0c025a",
			want: BluetoothDevice{ClassOfDev: 0x5a020c, DeviceClass: BluetoothClassPhone},
		},
		{
			name: "appearance classifies a device without a class of device",
			data: "0319c103",
			want: BluetoothDevice{Appearance: 0x03c1, DeviceClass: BluetoothClassPeripheral},
		},
		{
			name: "zero-length padding ends an EIR block",
			data: "020af4" + "00" + "020a08",
			want: BluetoothDevice{TxPower: -12},
		},
		{
			name: "a field running past the payload is dropped",
			data: "020af4" + "0509414243",
			want: BluetoothDevice{TxPower: -12},
		},
		{
			name: "manufacturer data shorter than a company ID is ignored",
			data: "02ff4c",
			want: BluetoothDevice{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got BluetoothDevice
			applyAdvertisingData(&got, mustHex(t, tc.data))
			if got.Name != tc.want.Name || got.TxPower != tc.want.TxPower ||
				got.ClassOfDev != tc.want.ClassOfDev || got.DeviceClass != tc.want.DeviceClass ||
				got.Appearance != tc.want.Appearance || got.ManufacturerID != tc.want.ManufacturerID ||
				!slices.Equal(got.ManufacturerData, tc.want.ManufacturerData) ||
				!slices.Equal(got.ServiceUUIDs, tc.want.ServiceUUIDs) {
				t.Errorf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}
