package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// Wire-shape pins for the Wi-Fi handlers; response_dto_shape_internal_test.go
// explains why these exist.

type statusRadio struct{ troubleshooting.Hardware }

func (statusRadio) ManagerAvailable() bool          { return true }
func (statusRadio) IsWireless() bool                { return true }
func (statusRadio) IsWirelessInterface(string) bool { return true }

type wirelessNames []string

func (n wirelessNames) WirelessInterfaceNames() []string { return n }

type fixedWiFiInterface string

func (f fixedWiFiInterface) ResolvedWiFiInterface() string { return string(f) }
func (fixedWiFiInterface) SaveWiFiInterface(string) error  { return nil }

func TestWiFiStatusResponseShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		adapters wirelessNames
		want     string
	}{
		{
			name:     "ready",
			adapters: wirelessNames{"wlan0"},
			want: `{"status":"ready","message":"Wireless adapter ready for scanning.",` +
				`"currentInterface":"wlan0","isWireless":true,"availableAdapters":["wlan0"],"canScan":true}`,
		},
		{
			name: "no adapter",
			want: `{"status":"unavailable",` +
				`"message":"No wireless adapter detected. Connect a WiFi adapter to scan networks.",` +
				`"currentInterface":"wlan0","isWireless":true,"availableAdapters":[],"canScan":false}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{wifiManagement: troubleshooting.NewManagement(
				statusRadio{}, tt.adapters, fixedWiFiInterface("wlan0"))}
			w := serve(t, s.handleWiFiStatus, "/api/v1/wifi/wifi/status")
			require.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

type noScanRadio struct{ statusRadio }

func (noScanRadio) ScannerAvailable() bool { return false }

type scanRadio struct{ statusRadio }

func (scanRadio) ScannerAvailable() bool { return true }
func (scanRadio) SetInterface(string)    {}
func (scanRadio) Scan(string) ([]*wifi.ScannedNetwork, error) {
	return []*wifi.ScannedNetwork{{
		SSID: "lab", BSSID: "aa:bb:cc:dd:ee:ff", Signal: -51, Channel: 36,
		Frequency: 5180, Security: "WPA2", ChannelWidth: 80, LastSeen: time.Unix(0, 0).UTC(),
	}}, nil
}

func (scanRadio) Disconnect() (*wifi.ConnectionResult, error) {
	return &wifi.ConnectionResult{Success: true, Message: "Disconnected", SSID: "lab"}, nil
}

func TestWiFiScanResponseShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		radio troubleshooting.Hardware
		want  string
	}{
		{
			name:  "scanned",
			radio: scanRadio{},
			want: `{"interface":"wlan0","available":true,"networks":[{"ssid":"lab","bssid":"aa:bb:cc:dd:ee:ff",` +
				`"signal":-51,"channel":36,"frequency":5180,"security":"WPA2","channelWidth":80,"noiseFloor":0,` +
				`"snr":0,"htMode":"","isDFS":false,"lastSeen":"1970-01-01T00:00:00Z","hidden":false,` +
				`"pmfRequired":false,"wpsEnabled":false,"rrmNeighbor":false,"btmSupported":false,` +
				`"ftSupported":false,"hasBssLoad":false,"channelUtil":0,"advertisedStations":0}]}`,
		},
		{
			name:  "no scanner",
			radio: noScanRadio{},
			want:  `{"interface":"wlan0","available":false,"networks":[],"error":"WiFi scanner not initialized"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{wifiManagement: troubleshooting.NewManagement(tt.radio, nil, fixedWiFiInterface("wlan0"))}
			w := serve(t, s.handleWiFiScan, "/api/v1/wifi/wifi/scan")
			require.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

// The channel graph answered each network with snake_case keys the UI never
// mapped (it renamed only the band lists), so every curve was drawn from an
// undefined signal and channel width.
func TestWiFiChannelGraphResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{
		wifiManagement: troubleshooting.NewManagement(scanRadio{}, nil, fixedWiFiInterface("wlan0")),
		wifiMgr:        wifi.NewManager("seed-test-absent"),
	}
	w := serve(t, s.handleWiFiChannelGraph, "/api/v1/wifi/wifi/channel-graph")

	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Contains(t, body.Data, "scanTime")
	delete(body.Data, "scanTime")
	data, err := json.Marshal(body.Data)
	require.NoError(t, err)
	require.JSONEq(t, `{"networks24Ghz":[],"networks6Ghz":[],"networks5Ghz":[{"ssid":"lab",`+
		`"bssid":"aa:bb:cc:dd:ee:ff","channel":36,"centerFreqMhz":5180,"channelWidthMhz":80,`+
		`"signalDbm":-51,"band":"5GHz","isConnected":false}]}`, string(data))
}

func TestWiFiChannelGraphUnavailableHasNoData(t *testing.T) {
	t.Parallel()
	s := &Server{wifiManagement: troubleshooting.NewManagement(noScanRadio{}, nil, fixedWiFiInterface("wlan0"))}
	w := serve(t, s.handleWiFiChannelGraph, "/api/v1/wifi/wifi/channel-graph")
	require.JSONEq(t, `{"interface":"wlan0","available":false,"error":"WiFi scanner not initialized"}`,
		w.Body.String())
}

func TestWiFiDisconnectResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{wifiManagement: troubleshooting.NewManagement(scanRadio{}, nil, fixedWiFiInterface("wlan0"))}
	w := serve(t, s.handleWiFiDisconnect, "/api/v1/wifi/wifi/disconnect")
	require.JSONEq(t, `{"success":true,"message":"Disconnected","ssid":"lab"}`, w.Body.String())
}

func TestWiFiSettingsUpdateResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{wifiManagement: troubleshooting.NewManagement(scanRadio{}, nil, fixedWiFiInterface("wlan0"))}
	w := httptest.NewRecorder()
	s.handleWiFiSettings(w, httptest.NewRequest(http.MethodPut, "/api/v1/wifi/wifi/settings",
		strings.NewReader(`{"interface":"wlan1"}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"status":"success","message":"WiFi settings updated"}`, w.Body.String())
}

func TestWiFiSavedNetworksWithoutManagerIsEmptyArray(t *testing.T) {
	t.Parallel()
	w := serve(t, (&Server{}).handleWiFiSavedNetworks, "/api/v1/wifi/wifi/saved")
	require.JSONEq(t, `{"networks":[]}`, w.Body.String())
}
