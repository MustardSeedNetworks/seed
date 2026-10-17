//go:build darwin

package enumerate

import (
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

type fakeBLEHelper struct {
	asked time.Duration
	found []wifihelper.Peripheral
	err   error
}

func (f *fakeBLEHelper) ScanBLE(d time.Duration) ([]wifihelper.Peripheral, error) {
	f.asked = d
	return f.found, f.err
}

func TestScanBLE(t *testing.T) {
	t.Parallel()

	helper := &fakeBLEHelper{found: []wifihelper.Peripheral{
		{Identifier: "5A1C0F2E-6B4D-4E8A-9C1F-3D2B7A6E9F01", RSSI: -61},
	}}
	s := NewBluetoothScanner("", nil, nil)
	s.SetBLEHelper(helper)

	devices, err := s.scanBLE(&BluetoothScanConfig{ScanDurationSec: 7})
	if err != nil {
		t.Fatalf("scanBLE() unexpected error: %v", err)
	}
	if helper.asked != 7*time.Second {
		t.Errorf("helper asked to listen %s, want 7s", helper.asked)
	}
	if len(devices) != 1 || devices[0].Address != "5A1C0F2E-6B4D-4E8A-9C1F-3D2B7A6E9F01" ||
		devices[0].Type != BluetoothTypeBLE {
		t.Errorf("scanBLE() = %+v, want the helper's peripheral as an LE device", devices)
	}
}

// Without a helper, or with one that lacks the grant, the scan must fail with
// the reason rather than report a room with no LE devices in it.
func TestScanBLEFailures(t *testing.T) {
	t.Parallel()

	_, err := NewBluetoothScanner("", nil, nil).scanBLE(DefaultBluetoothScanConfig())
	if !errors.Is(err, errNoBLEHelper) {
		t.Errorf("scanBLE() without a helper error = %v, want errNoBLEHelper", err)
	}

	denied := errors.New("corebluetooth: Bluetooth authorization required")
	s := NewBluetoothScanner("", nil, nil)
	s.SetBLEHelper(&fakeBLEHelper{err: denied})
	if _, err = s.scanBLE(DefaultBluetoothScanConfig()); !errors.Is(err, denied) {
		t.Errorf("scanBLE() with a refused helper error = %v, want the helper's reason", err)
	}
}
