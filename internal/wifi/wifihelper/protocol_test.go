package wifihelper_test

import (
	"errors"
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
