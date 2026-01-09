//go:build linux

package wifi_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// channelGraphTestCase defines a test case for GetChannelGraphData.
type channelGraphTestCase struct {
	name           string
	networks       []*wifi.ScannedNetwork
	connectedBSSID string
	wantNum2_4     int
	wantNum5       int
	wantNum6       int
	wantConnected  string
}

// assertBandCounts verifies that the channel graph data has the expected number
// of networks in each frequency band.
func assertBandCounts(t *testing.T, got *wifi.ChannelGraphData, tc channelGraphTestCase) {
	t.Helper()

	if len(got.Networks2_4GHz) != tc.wantNum2_4 {
		t.Errorf(
			"GetChannelGraphData() 2.4GHz count = %d, want %d",
			len(got.Networks2_4GHz),
			tc.wantNum2_4,
		)
	}
	if len(got.Networks5GHz) != tc.wantNum5 {
		t.Errorf(
			"GetChannelGraphData() 5GHz count = %d, want %d",
			len(got.Networks5GHz),
			tc.wantNum5,
		)
	}
	if len(got.Networks6GHz) != tc.wantNum6 {
		t.Errorf(
			"GetChannelGraphData() 6GHz count = %d, want %d",
			len(got.Networks6GHz),
			tc.wantNum6,
		)
	}
}

// assertConnectedBSSID verifies that the connected BSSID field matches expected.
func assertConnectedBSSID(t *testing.T, got *wifi.ChannelGraphData, wantConnected string) {
	t.Helper()

	if got.ConnectedBSSID != wantConnected {
		t.Errorf(
			"GetChannelGraphData() ConnectedBSSID = %q, want %q",
			got.ConnectedBSSID,
			wantConnected,
		)
	}
}

// assertScanTimeRecent verifies that the scan time is within the last second.
func assertScanTimeRecent(t *testing.T, got *wifi.ChannelGraphData) {
	t.Helper()

	if time.Since(got.ScanTime) > time.Second {
		t.Errorf("GetChannelGraphData() ScanTime is too old: %v", got.ScanTime)
	}
}

// collectAllNetworks combines networks from all bands into a single slice.
func collectAllNetworks(got *wifi.ChannelGraphData) []wifi.ChannelNetwork {
	allNetworks := make(
		[]wifi.ChannelNetwork,
		0,
		len(got.Networks2_4GHz)+len(got.Networks5GHz)+len(got.Networks6GHz),
	)
	allNetworks = append(allNetworks, got.Networks2_4GHz...)
	allNetworks = append(allNetworks, got.Networks5GHz...)
	allNetworks = append(allNetworks, got.Networks6GHz...)
	return allNetworks
}

// assertConnectedNetworkMarked verifies that the correct network is marked as connected.
func assertConnectedNetworkMarked(t *testing.T, allNetworks []wifi.ChannelNetwork, connectedBSSID string) {
	t.Helper()

	foundConnected := false
	for _, cn := range allNetworks {
		if cn.BSSID == connectedBSSID {
			if !cn.IsConnected {
				t.Errorf(
					"Network with BSSID %s should be marked as connected",
					connectedBSSID,
				)
			}
			foundConnected = true
		} else if cn.IsConnected {
			t.Errorf("Network with BSSID %s should not be marked as connected", cn.BSSID)
		}
	}
	if !foundConnected && len(allNetworks) > 0 {
		t.Errorf(
			"Connected network with BSSID %s not found in results",
			connectedBSSID,
		)
	}
}

func TestGetBand(t *testing.T) {
	tests := []struct {
		name string
		freq int
		want string
	}{
		{"2.4 GHz channel 1", 2412, "2.4GHz"},
		{"2.4 GHz channel 6", 2437, "2.4GHz"},
		{"2.4 GHz channel 11", 2462, "2.4GHz"},
		{"2.4 GHz channel 14", 2484, "2.4GHz"},
		{"5 GHz channel 36", 5180, "5GHz"},
		{"5 GHz channel 149", 5745, "5GHz"},
		{"6 GHz channel 1", 5955, "6GHz"},
		{"6 GHz channel 93", 6415, "6GHz"},
		{"Unknown frequency", 1000, ""},
		{"Zero frequency", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.GetBand(tt.freq)
			if got != tt.want {
				t.Errorf("GetBand(%d) = %q, want %q", tt.freq, got, tt.want)
			}
		})
	}
}

func TestDetectChannelWidth(t *testing.T) {
	tests := []struct {
		name   string
		freq   int
		htMode string
		want   int
	}{
		// Explicit HTMode parsing
		{"HT20", 2412, "HT20", 20},
		{"HT40", 2412, "HT40", 40},
		{"HT40+", 2412, "HT40+", 40},
		{"HT40-", 2412, "HT40-", 40},
		{"VHT80", 5180, "VHT80", 80},
		{"VHT160", 5180, "VHT160", 160},
		{"HE160", 5180, "HE160", 160},
		{"EHT320", 5955, "EHT320", 320},

		// Fallback to band-based detection
		{"2.4 GHz default", 2412, "", 20},
		{"5 GHz default", 5180, "", 80},
		{"6 GHz default", 5955, "", 160},
		{"Unknown band default", 1000, "", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.DetectChannelWidth(tt.freq, tt.htMode)
			if got != tt.want {
				t.Errorf(
					"DetectChannelWidth(%d, %q) = %d, want %d",
					tt.freq,
					tt.htMode,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestGetChannelGraphData(t *testing.T) {
	now := time.Now()
	tests := buildChannelGraphTestCases(now)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.GetChannelGraphData(tt.networks, tt.connectedBSSID)

			assertBandCounts(t, got, tt)
			assertConnectedBSSID(t, got, tt.wantConnected)
			assertScanTimeRecent(t, got)

			if tt.connectedBSSID != "" {
				allNetworks := collectAllNetworks(got)
				assertConnectedNetworkMarked(t, allNetworks, tt.connectedBSSID)
			}
		})
	}
}

// buildChannelGraphTestCases creates the test cases for GetChannelGraphData.
func buildChannelGraphTestCases(now time.Time) []channelGraphTestCase {
	return []channelGraphTestCase{
		{
			name:           "Empty networks",
			networks:       []*wifi.ScannedNetwork{},
			connectedBSSID: "",
			wantNum2_4:     0,
			wantNum5:       0,
			wantNum6:       0,
			wantConnected:  "",
		},
		{
			name:           "Nil networks",
			networks:       nil,
			connectedBSSID: "",
			wantNum2_4:     0,
			wantNum5:       0,
			wantNum6:       0,
			wantConnected:  "",
		},
		buildSingle24GHzTestCase(now),
		buildSingle5GHzTestCase(now),
		buildSingle6GHzTestCase(now),
		buildMixedBandTestCase(now),
		buildAutoWidthTestCase(now),
		buildUnknownFreqTestCase(now),
	}
}

// buildSingle24GHzTestCase creates a test case for a single 2.4 GHz network.
func buildSingle24GHzTestCase(now time.Time) channelGraphTestCase {
	return channelGraphTestCase{
		name: "Single 2.4 GHz network",
		networks: []*wifi.ScannedNetwork{
			{
				SSID:         "TestNet24",
				BSSID:        "00:11:22:33:44:55",
				Signal:       -50,
				Channel:      6,
				Frequency:    2437,
				ChannelWidth: 20,
				HTMode:       "HT20",
				LastSeen:     now,
			},
		},
		connectedBSSID: "",
		wantNum2_4:     1,
		wantNum5:       0,
		wantNum6:       0,
		wantConnected:  "",
	}
}

// buildSingle5GHzTestCase creates a test case for a single 5 GHz network.
func buildSingle5GHzTestCase(now time.Time) channelGraphTestCase {
	return channelGraphTestCase{
		name: "Single 5 GHz network",
		networks: []*wifi.ScannedNetwork{
			{
				SSID:         "TestNet5",
				BSSID:        "AA:BB:CC:DD:EE:FF",
				Signal:       -60,
				Channel:      36,
				Frequency:    5180,
				ChannelWidth: 80,
				HTMode:       "VHT80",
				LastSeen:     now,
			},
		},
		connectedBSSID: "",
		wantNum2_4:     0,
		wantNum5:       1,
		wantNum6:       0,
		wantConnected:  "",
	}
}

// buildSingle6GHzTestCase creates a test case for a single 6 GHz network.
func buildSingle6GHzTestCase(now time.Time) channelGraphTestCase {
	return channelGraphTestCase{
		name: "Single 6 GHz network",
		networks: []*wifi.ScannedNetwork{
			{
				SSID:         "TestNet6",
				BSSID:        "11:22:33:44:55:66",
				Signal:       -40,
				Channel:      1,
				Frequency:    5955,
				ChannelWidth: 160,
				HTMode:       "HE160",
				LastSeen:     now,
			},
		},
		connectedBSSID: "",
		wantNum2_4:     0,
		wantNum5:       0,
		wantNum6:       1,
		wantConnected:  "",
	}
}

// buildMixedBandTestCase creates a test case with networks across all bands.
func buildMixedBandTestCase(now time.Time) channelGraphTestCase {
	return channelGraphTestCase{
		name: "Mixed band networks with connected",
		networks: []*wifi.ScannedNetwork{
			{
				SSID:         "Home24",
				BSSID:        "00:11:22:33:44:55",
				Signal:       -45,
				Channel:      1,
				Frequency:    2412,
				ChannelWidth: 20,
				HTMode:       "HT20",
				LastSeen:     now,
			},
			{
				SSID:         "Home5",
				BSSID:        "00:11:22:33:44:56",
				Signal:       -50,
				Channel:      36,
				Frequency:    5180,
				ChannelWidth: 80,
				HTMode:       "VHT80",
				LastSeen:     now,
			},
			{
				SSID:         "Home6",
				BSSID:        "00:11:22:33:44:57",
				Signal:       -55,
				Channel:      1,
				Frequency:    5955,
				ChannelWidth: 160,
				HTMode:       "HE160",
				LastSeen:     now,
			},
		},
		connectedBSSID: "00:11:22:33:44:56",
		wantNum2_4:     1,
		wantNum5:       1,
		wantNum6:       1,
		wantConnected:  "00:11:22:33:44:56",
	}
}

// buildAutoWidthTestCase creates a test case for auto-detecting channel width.
func buildAutoWidthTestCase(now time.Time) channelGraphTestCase {
	return channelGraphTestCase{
		name: "Network with zero channel width (auto-detect)",
		networks: []*wifi.ScannedNetwork{
			{
				SSID:         "AutoWidth",
				BSSID:        "AA:BB:CC:DD:EE:FF",
				Signal:       -60,
				Channel:      36,
				Frequency:    5180,
				ChannelWidth: 0, // Should be auto-detected
				HTMode:       "VHT80",
				LastSeen:     now,
			},
		},
		connectedBSSID: "",
		wantNum2_4:     0,
		wantNum5:       1,
		wantNum6:       0,
		wantConnected:  "",
	}
}

// buildUnknownFreqTestCase creates a test case for unknown frequency (should be skipped).
func buildUnknownFreqTestCase(now time.Time) channelGraphTestCase {
	return channelGraphTestCase{
		name: "Network with unknown frequency (skipped)",
		networks: []*wifi.ScannedNetwork{
			{
				SSID:         "Unknown",
				BSSID:        "00:00:00:00:00:00",
				Signal:       -70,
				Channel:      0,
				Frequency:    1000, // Unknown frequency
				ChannelWidth: 20,
				HTMode:       "",
				LastSeen:     now,
			},
		},
		connectedBSSID: "",
		wantNum2_4:     0,
		wantNum5:       0,
		wantNum6:       0,
		wantConnected:  "",
	}
}

func TestGetChannelGraphDataFieldMapping(t *testing.T) {
	// Test that fields are correctly mapped from ScannedNetwork to ChannelNetwork
	network := &wifi.ScannedNetwork{
		SSID:         "TestSSID",
		BSSID:        "AA:BB:CC:DD:EE:FF",
		Signal:       -55,
		Channel:      6,
		Frequency:    2437,
		ChannelWidth: 20,
		HTMode:       "HT20",
		LastSeen:     time.Now(),
	}

	data := wifi.GetChannelGraphData([]*wifi.ScannedNetwork{network}, "AA:BB:CC:DD:EE:FF")

	if len(data.Networks2_4GHz) != 1 {
		t.Fatalf("Expected 1 network in 2.4GHz band, got %d", len(data.Networks2_4GHz))
	}

	cn := data.Networks2_4GHz[0]

	if cn.SSID != network.SSID {
		t.Errorf("SSID = %q, want %q", cn.SSID, network.SSID)
	}
	if cn.BSSID != network.BSSID {
		t.Errorf("BSSID = %q, want %q", cn.BSSID, network.BSSID)
	}
	if cn.Signal != network.Signal {
		t.Errorf("Signal = %d, want %d", cn.Signal, network.Signal)
	}
	if cn.Channel != network.Channel {
		t.Errorf("Channel = %d, want %d", cn.Channel, network.Channel)
	}
	if cn.CenterFreq != network.Frequency {
		t.Errorf("CenterFreq = %d, want %d", cn.CenterFreq, network.Frequency)
	}
	if cn.ChannelWidth != network.ChannelWidth {
		t.Errorf("ChannelWidth = %d, want %d", cn.ChannelWidth, network.ChannelWidth)
	}
	if cn.Band != "2.4GHz" {
		t.Errorf("Band = %q, want %q", cn.Band, "2.4GHz")
	}
	if !cn.IsConnected {
		t.Errorf("IsConnected = false, want true")
	}
}
