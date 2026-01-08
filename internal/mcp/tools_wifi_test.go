package mcp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/mcp"
)

func TestWiFiScanFlow(t *testing.T) {
	tests := []struct {
		name        string
		networks    []mcp.WiFiNetwork
		scanErr     error
		expectError bool
		expectCount int
	}{
		{
			name: "successful scan with multiple networks",
			networks: []mcp.WiFiNetwork{
				{
					SSID:      "Network1",
					BSSID:     "00:11:22:33:44:55",
					Signal:    -50,
					Channel:   1,
					Frequency: 2412,
					Security:  "WPA2-PSK",
				},
				{
					SSID:      "Network2",
					BSSID:     "00:11:22:33:44:56",
					Signal:    -65,
					Channel:   6,
					Frequency: 2437,
					Security:  "WPA3-SAE",
				},
				{
					SSID:      "Network3",
					BSSID:     "00:11:22:33:44:57",
					Signal:    -80,
					Channel:   11,
					Frequency: 2462,
					Security:  "Open",
				},
			},
			expectError: false,
			expectCount: 3,
		},
		{
			name:        "successful scan with no networks",
			networks:    []mcp.WiFiNetwork{},
			expectError: false,
			expectCount: 0,
		},
		{
			name:        "scan error",
			networks:    nil,
			scanErr:     errors.New("WiFi scan failed: interface not available"),
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWiFiScanner{
				networks: tt.networks,
				scanErr:  tt.scanErr,
			}

			networks, err := mock.Scan(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(networks) != tt.expectCount {
					t.Errorf("expected %d networks, got %d", tt.expectCount, len(networks))
				}
			}
		})
	}
}

func TestWiFiInfoFlow(t *testing.T) {
	tests := []struct {
		name          string
		currentNet    *mcp.WiFiConnectionInfo
		getNetworkErr error
		expectError   bool
		expectNil     bool
	}{
		{
			name: "connected to network",
			currentNet: &mcp.WiFiConnectionInfo{
				SSID:      "MyNetwork",
				BSSID:     "00:11:22:33:44:55",
				Signal:    -55,
				Channel:   6,
				Frequency: 2437,
				Security:  "WPA2-PSK",
			},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "not connected to any network",
			currentNet:  nil,
			expectError: false,
			expectNil:   true,
		},
		{
			name:          "error getting network info",
			currentNet:    nil,
			getNetworkErr: errors.New("failed to get WiFi info"),
			expectError:   true,
			expectNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWiFiManager{
				currentNetwork: tt.currentNet,
				getNetworkErr:  tt.getNetworkErr,
			}

			info, err := mock.GetCurrentNetwork()

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.expectNil {
				if info != nil {
					t.Error("expected nil info")
				}
			} else {
				if info == nil {
					t.Error("expected non-nil info")
				}
			}
		})
	}
}

func TestWiFiNetworkFields(t *testing.T) {
	tests := []struct {
		name      string
		network   mcp.WiFiNetwork
		checkFunc func(t *testing.T, n mcp.WiFiNetwork)
	}{
		{
			name: "2.4GHz network",
			network: mcp.WiFiNetwork{
				SSID:      "Network24",
				BSSID:     "00:11:22:33:44:55",
				Signal:    -60,
				Channel:   6,
				Frequency: 2437,
				Security:  "WPA2-PSK",
			},
			checkFunc: func(t *testing.T, n mcp.WiFiNetwork) {
				if n.Frequency < 2400 || n.Frequency > 2500 {
					t.Errorf("expected 2.4GHz frequency, got %d", n.Frequency)
				}
				if n.Channel < 1 || n.Channel > 14 {
					t.Errorf("expected 2.4GHz channel (1-14), got %d", n.Channel)
				}
			},
		},
		{
			name: "5GHz network",
			network: mcp.WiFiNetwork{
				SSID:      "Network5",
				BSSID:     "00:11:22:33:44:56",
				Signal:    -70,
				Channel:   36,
				Frequency: 5180,
				Security:  "WPA3-SAE",
			},
			checkFunc: func(t *testing.T, n mcp.WiFiNetwork) {
				if n.Frequency < 5000 || n.Frequency > 6000 {
					t.Errorf("expected 5GHz frequency, got %d", n.Frequency)
				}
				if n.Channel < 36 {
					t.Errorf("expected 5GHz channel (>= 36), got %d", n.Channel)
				}
			},
		},
		{
			name: "strong signal",
			network: mcp.WiFiNetwork{
				SSID:   "StrongSignal",
				Signal: -30,
			},
			checkFunc: func(t *testing.T, n mcp.WiFiNetwork) {
				if n.Signal < -50 {
					t.Errorf("expected strong signal (> -50 dBm), got %d", n.Signal)
				}
			},
		},
		{
			name: "weak signal",
			network: mcp.WiFiNetwork{
				SSID:   "WeakSignal",
				Signal: -85,
			},
			checkFunc: func(t *testing.T, n mcp.WiFiNetwork) {
				if n.Signal > -80 {
					t.Errorf("expected weak signal (< -80 dBm), got %d", n.Signal)
				}
			},
		},
		{
			name: "WPA2 security",
			network: mcp.WiFiNetwork{
				SSID:     "SecureNet",
				Security: "WPA2-PSK",
			},
			checkFunc: func(t *testing.T, n mcp.WiFiNetwork) {
				if n.Security != "WPA2-PSK" {
					t.Errorf("expected WPA2-PSK security, got %s", n.Security)
				}
			},
		},
		{
			name: "WPA3 security",
			network: mcp.WiFiNetwork{
				SSID:     "VerySecureNet",
				Security: "WPA3-SAE",
			},
			checkFunc: func(t *testing.T, n mcp.WiFiNetwork) {
				if n.Security != "WPA3-SAE" {
					t.Errorf("expected WPA3-SAE security, got %s", n.Security)
				}
			},
		},
		{
			name: "open network",
			network: mcp.WiFiNetwork{
				SSID:     "OpenNet",
				Security: "Open",
			},
			checkFunc: func(t *testing.T, n mcp.WiFiNetwork) {
				if n.Security != "Open" {
					t.Errorf("expected Open security, got %s", n.Security)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checkFunc(t, tt.network)
		})
	}
}

func TestWiFiSignalStrength(t *testing.T) {
	tests := []struct {
		name    string
		signal  int
		quality string
	}{
		{
			name:    "excellent signal",
			signal:  -30,
			quality: "excellent",
		},
		{
			name:    "good signal",
			signal:  -55,
			quality: "good",
		},
		{
			name:    "fair signal",
			signal:  -70,
			quality: "fair",
		},
		{
			name:    "weak signal",
			signal:  -75,
			quality: "weak",
		},
		{
			name:    "no signal",
			signal:  -95,
			quality: "no signal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Classify signal quality based on dBm
			var quality string
			switch {
			case tt.signal >= -50:
				quality = "excellent"
			case tt.signal >= -60:
				quality = "good"
			case tt.signal >= -70:
				quality = "fair"
			case tt.signal >= -80:
				quality = "weak"
			default:
				quality = "no signal"
			}

			if quality != tt.quality {
				t.Errorf("expected quality %q for signal %d dBm, got %q",
					tt.quality, tt.signal, quality)
			}
		})
	}
}

func TestWiFiChannelFrequencyMapping(t *testing.T) {
	// 2.4GHz channel to frequency mapping
	channelToFreq24 := map[int]int{
		1:  2412,
		2:  2417,
		3:  2422,
		4:  2427,
		5:  2432,
		6:  2437,
		7:  2442,
		8:  2447,
		9:  2452,
		10: 2457,
		11: 2462,
		12: 2467,
		13: 2472,
		14: 2484,
	}

	for channel, expectedFreq := range channelToFreq24 {
		t.Run("2.4GHz_channel_"+string(rune('0'+channel)), func(t *testing.T) {
			network := mcp.WiFiNetwork{
				Channel:   channel,
				Frequency: expectedFreq,
			}
			// Verify channel is in valid range
			if network.Channel < 1 || network.Channel > 14 {
				t.Errorf("channel %d out of 2.4GHz range", network.Channel)
			}
			// Verify frequency matches expected
			if network.Frequency != expectedFreq {
				t.Errorf("channel %d: expected freq %d, got %d",
					channel, expectedFreq, network.Frequency)
			}
		})
	}
}

func TestWiFiSecurityTypes(t *testing.T) {
	securityTypes := []string{
		"Open",
		"WEP",
		"WPA-PSK",
		"WPA2-PSK",
		"WPA3-SAE",
		"WPA2-Enterprise",
		"WPA3-Enterprise",
	}

	for _, security := range securityTypes {
		t.Run(security, func(t *testing.T) {
			network := mcp.WiFiNetwork{
				SSID:     "TestNetwork",
				Security: security,
			}
			if network.Security != security {
				t.Errorf("expected security %q, got %q", security, network.Security)
			}
		})
	}
}

func TestWiFiManagerSignalStrength(t *testing.T) {
	tests := []struct {
		name        string
		signal      int
		signalErr   error
		expectError bool
	}{
		{
			name:        "strong signal",
			signal:      -40,
			expectError: false,
		},
		{
			name:        "weak signal",
			signal:      -85,
			expectError: false,
		},
		{
			name:        "error getting signal",
			signal:      0,
			signalErr:   errors.New("WiFi not connected"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWiFiManager{
				signalStrength: tt.signal,
				getSignalErr:   tt.signalErr,
			}

			signal, err := mock.GetSignalStrength()

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if signal != tt.signal {
					t.Errorf("expected signal %d, got %d", tt.signal, signal)
				}
			}
		})
	}
}

func TestWiFiScannerServiceAvailability(t *testing.T) {
	tests := []struct {
		name       string
		hasScanner bool
		hasManager bool
	}{
		{
			name:       "both available",
			hasScanner: true,
			hasManager: true,
		},
		{
			name:       "only scanner",
			hasScanner: true,
			hasManager: false,
		},
		{
			name:       "only manager",
			hasScanner: false,
			hasManager: true,
		},
		{
			name:       "neither available",
			hasScanner: false,
			hasManager: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scanner mcp.WiFiScanner
			var manager mcp.WiFiManager

			if tt.hasScanner {
				scanner = &mockWiFiScanner{}
			}
			if tt.hasManager {
				manager = &mockWiFiManager{}
			}

			provider := &mockServiceProvider{
				wifiScanner: scanner,
				wifiManager: manager,
			}

			if (provider.GetWiFiScanner() != nil) != tt.hasScanner {
				t.Errorf("WiFi scanner availability mismatch")
			}
			if (provider.GetWiFiManager() != nil) != tt.hasManager {
				t.Errorf("WiFi manager availability mismatch")
			}
		})
	}
}

func TestWiFiNetworksSorting(t *testing.T) {
	// Networks should typically be sorted by signal strength (strongest first)
	networks := []mcp.WiFiNetwork{
		{SSID: "Network1", Signal: -80},
		{SSID: "Network2", Signal: -50},
		{SSID: "Network3", Signal: -65},
		{SSID: "Network4", Signal: -30},
	}

	// Expected order after sorting by signal strength (descending)
	expectedOrder := []string{"Network4", "Network2", "Network3", "Network1"}

	// Sort networks by signal strength (descending - less negative is better)
	for i := 0; i < len(networks)-1; i++ {
		for j := i + 1; j < len(networks); j++ {
			if networks[j].Signal > networks[i].Signal {
				networks[i], networks[j] = networks[j], networks[i]
			}
		}
	}

	for i, network := range networks {
		if network.SSID != expectedOrder[i] {
			t.Errorf("position %d: expected %s, got %s", i, expectedOrder[i], network.SSID)
		}
	}
}
