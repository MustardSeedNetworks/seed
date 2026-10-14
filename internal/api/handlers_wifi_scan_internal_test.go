package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// recordingRadio remembers which interface each scan ran on; wlan0 and wlan1
// are its radios.
type recordingRadio struct {
	statusRadio

	scanned *string
}

func (recordingRadio) ScannerAvailable() bool { return true }
func (recordingRadio) IsWirelessInterface(name string) bool {
	return slices.Contains([]string{"wlan0", "wlan1"}, name)
}

func (r recordingRadio) Scan(iface string) ([]*wifi.ScannedNetwork, error) {
	*r.scanned = iface
	return []*wifi.ScannedNetwork{}, nil
}

// The scan handlers ran the configured radio whatever the request named, and
// with no parameter they reported the wired interface as the one scanned
// (#2881). Each case asserts the reported interface is the scanned one.
func TestWiFiScanHandlersScanTheNamedRadio(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		query       string
		wantCode    int
		wantScanned string
		wantError   string
	}{
		{name: "no parameter uses the Wi-Fi radio, not eth0", wantCode: http.StatusOK, wantScanned: "wlan0"},
		{name: "second radio", query: "?interface=wlan1", wantCode: http.StatusOK, wantScanned: "wlan1"},
		{
			name:      "wired interface is refused",
			query:     "?interface=eth0",
			wantCode:  http.StatusOK,
			wantError: "eth0 is not a Wi-Fi adapter. Choose a Wi-Fi adapter to scan.",
		},
		{name: "malformed name", query: "?interface=wlan0%3Brm", wantCode: http.StatusBadRequest},
	}
	for _, tt := range tests {
		for _, route := range []string{"scan", "channel-graph"} {
			t.Run(tt.name+"/"+route, func(t *testing.T) {
				t.Parallel()
				var scanned string
				s := &Server{
					wifiManagement: troubleshooting.NewManagement(
						recordingRadio{scanned: &scanned}, nil, fixedWiFiInterface("wlan0")),
					// The wired interface is current, as on a probe with a radio.
					netMgr:  netif.NewMockManager(netif.DefaultMockConfig()),
					wifiMgr: wifi.NewManager("seed-test-absent"),
				}
				handler := s.handleWiFiScan
				if route == "channel-graph" {
					handler = s.handleWiFiChannelGraph
				}
				w := httptest.NewRecorder()
				handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/wifi/wifi/"+route+tt.query, http.NoBody))

				require.Equal(t, tt.wantCode, w.Code, w.Body.String())
				require.Equal(t, tt.wantScanned, scanned)
				if tt.wantCode != http.StatusOK {
					return
				}
				var body struct {
					Interface string `json:"interface"`
					Error     string `json:"error"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				if tt.wantScanned != "" {
					require.Equal(t, tt.wantScanned, body.Interface)
				}
				require.Equal(t, tt.wantError, body.Error)
			})
		}
	}
}
