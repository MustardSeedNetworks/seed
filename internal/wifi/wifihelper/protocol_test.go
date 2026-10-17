package wifihelper_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

func TestDecodeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		line    string
		want    wifihelper.Op
		wantErr bool
	}{
		{"scan", `{"op":"scan"}`, wifihelper.OpScan, false},
		{"current", `{"op":"current"}`, wifihelper.OpCurrent, false},
		{"saved", `{"op":"saved"}`, wifihelper.OpSaved, false},
		{"ble scan", `{"op":"bleScan","scanMillis":10000}`, wifihelper.OpBLEScan, false},
		{"ble scan at the cap", `{"op":"bleScan","scanMillis":60000}`, wifihelper.OpBLEScan, false},
		// The daemon waits for the reply, so the helper must not accept a scan
		// that never ends or one that cannot listen at all.
		{"ble scan without a duration refused", `{"op":"bleScan"}`, "", true},
		{"ble scan with a negative duration refused", `{"op":"bleScan","scanMillis":-1}`, "", true},
		{"ble scan past the cap refused", `{"op":"bleScan","scanMillis":60001}`, "", true},
		// An op the helper cannot answer must be refused, not silently treated
		// as an empty result.
		{"unknown op refused", `{"op":"reboot"}`, "", true},
		{"missing op refused", `{}`, "", true},
		{"malformed json", `{"op":`, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := wifihelper.DecodeRequest([]byte(tc.line))
			if tc.wantErr {
				if !errors.Is(err, wifihelper.ErrProtocol) {
					t.Fatalf("DecodeRequest(%s) error = %v, want ErrProtocol", tc.line, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeRequest(%s) unexpected error: %v", tc.line, err)
			}
			if got.Op != tc.want {
				t.Errorf("DecodeRequest(%s).Op = %q, want %q", tc.line, got.Op, tc.want)
			}
		})
	}
}

func TestResponseErr(t *testing.T) {
	t.Parallel()

	if err := (wifihelper.Response{}).Err(); err != nil {
		t.Errorf("Response{}.Err() = %v, want nil", err)
	}

	// The helper's failure reason must survive the wire, so an operator is told
	// the Location grant is missing rather than shown an empty network list.
	err := wifihelper.Response{Error: "Location Services authorization required"}.Err()
	if err == nil || err.Error() != "Location Services authorization required" {
		t.Errorf("Response.Err() = %v, want the helper's reason", err)
	}
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()

	resp, err := wifihelper.DecodeResponse([]byte(
		`{"networks":[{"ssid":"n","bssid":"aa:bb:cc:dd:ee:ff","rssi":-54,"channel":149,"band":5}]}`))
	if err != nil {
		t.Fatalf("DecodeResponse() unexpected error: %v", err)
	}
	if len(resp.Networks) != 1 || resp.Networks[0].BSSID != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("DecodeResponse() = %+v, want one network with its BSSID", resp)
	}

	_, decodeErr := wifihelper.DecodeResponse([]byte(`{"networks":`))
	if !errors.Is(decodeErr, wifihelper.ErrProtocol) {
		t.Errorf("DecodeResponse(malformed) error = %v, want ErrProtocol", decodeErr)
	}
}

// A peripheral must cross the socket intact: the manufacturer bytes carry the
// company identifier the vendor is named from, and an absent TX power must stay
// absent rather than read as 0 dBm.
func TestPeripheralRoundTrip(t *testing.T) {
	t.Parallel()

	tx := -8
	sent := wifihelper.Response{Peripherals: []wifihelper.Peripheral{
		{
			Identifier: "5A1C0F2E-6B4D-4E8A-9C1F-3D2B7A6E9F01", Name: "Tag", RSSI: -61, Connectable: true,
			TxPower: &tx, ManufacturerData: []byte{0x4c, 0x00, 0x02, 0x15}, ServiceUUIDs: []string{"180F"},
		},
		{Identifier: "0B7E2C44-1F3A-4C2D-8E5B-6A9D0C1E2F33", RSSI: -88},
	}}

	line, err := json.Marshal(sent)
	if err != nil {
		t.Fatalf("Marshal() unexpected error: %v", err)
	}
	got, err := wifihelper.DecodeResponse(line)
	if err != nil {
		t.Fatalf("DecodeResponse() unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, sent) {
		t.Errorf("DecodeResponse() = %+v, want %+v", got, sent)
	}
}
