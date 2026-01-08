// Package wifi_test provides wireless network information functionality tests.
// Test suite validates WiFi scanning, interface detection, and property extraction.
package wifi_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestNewManager(t *testing.T) {
	manager := wifi.NewManager("en0")
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.InterfaceName() != "en0" {
		t.Errorf("expected InterfaceName 'en0', got %q", manager.InterfaceName())
	}
}

func TestManagerSetInterface(t *testing.T) {
	manager := wifi.NewManager("en0")

	manager.SetInterface("wlan0")
	if manager.InterfaceName() != "wlan0" {
		t.Errorf("expected InterfaceName 'wlan0', got %q", manager.InterfaceName())
	}
}

func TestManagerIsWireless(_ *testing.T) {
	manager := wifi.NewManager("en0")

	// Result depends on system, just verify it doesn't panic
	_ = manager.IsWireless()
}

func TestManagerGetInfo(_ *testing.T) {
	manager := wifi.NewManager("en0")

	// Result depends on system, just verify it doesn't panic
	_ = manager.GetInfo()
}

func TestInfoFields(t *testing.T) {
	info := wifi.Info{
		SSID:      "TestNetwork",
		BSSID:     "00:11:22:33:44:55",
		Signal:    -65,
		Channel:   6,
		Frequency: 2437,
		Security:  "WPA2",
	}

	if info.SSID != "TestNetwork" {
		t.Errorf("expected SSID 'TestNetwork', got %q", info.SSID)
	}
	if info.BSSID != "00:11:22:33:44:55" {
		t.Errorf("expected BSSID '00:11:22:33:44:55', got %q", info.BSSID)
	}
	if info.Signal != -65 {
		t.Errorf("expected Signal -65, got %d", info.Signal)
	}
	if info.Channel != 6 {
		t.Errorf("expected Channel 6, got %d", info.Channel)
	}
	if info.Frequency != 2437 {
		t.Errorf("expected Frequency 2437, got %d", info.Frequency)
	}
	if info.Security != "WPA2" {
		t.Errorf("expected Security 'WPA2', got %q", info.Security)
	}
}

func TestMapSecurityType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SAE", "WPA3"},
		{"wpa3-personal", "WPA3"},
		{"WPA2-PSK", "WPA2"},
		{"WPA-PSK", "WPA"},
		{"WEP", "WEP"},
		{"open", "Open"},
		{"NONE", "Open"},
		{"unknown-type", "UNKNOWN-TYPE"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := wifi.MapSecurityType(tt.input)
			if result != tt.expected {
				t.Errorf("MapSecurityType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestChannelToFrequency(t *testing.T) {
	tests := []struct {
		channel  int
		expected int
	}{
		// 2.4 GHz band
		{1, 2412},
		{6, 2437},
		{11, 2462},
		{13, 2472},
		{14, 2484},
		// 5 GHz band
		{36, 5180},
		{40, 5200},
		{44, 5220},
		{48, 5240},
		{149, 5745},
		{153, 5765},
		{157, 5785},
		{161, 5805},
		{165, 5825},
		// Invalid channel
		{0, 0},
		{-1, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := wifi.ChannelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
			}
		})
	}
}

func TestFrequencyToChannel(t *testing.T) {
	tests := []struct {
		freq     int
		expected int
	}{
		// 2.4 GHz band
		{2412, 1},
		{2437, 6},
		{2462, 11},
		{2472, 13},
		{2484, 14},
		// 5 GHz band
		{5180, 36},
		{5200, 40},
		{5220, 44},
		{5240, 48},
		{5745, 149},
		{5765, 153},
		{5785, 157},
		{5805, 161},
		{5825, 165},
		// Invalid frequency
		{0, 0},
		{1000, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := wifi.FrequencyToChannel(tt.freq)
			if result != tt.expected {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, result, tt.expected)
			}
		})
	}
}

func TestChannelFrequencyRoundTrip(t *testing.T) {
	// Test that converting channel -> frequency -> channel gives original
	channels := []int{1, 6, 11, 36, 40, 44, 48, 149, 153, 157, 161, 165}

	for _, ch := range channels {
		freq := wifi.ChannelToFrequency(ch)
		if freq == 0 {
			continue
		}
		result := wifi.FrequencyToChannel(freq)
		if result != ch {
			t.Errorf("roundtrip failed: channel %d -> freq %d -> channel %d", ch, freq, result)
		}
	}
}

func TestConcurrentManagerAccess(_ *testing.T) {
	manager := wifi.NewManager("en0")

	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			for range 50 {
				manager.SetInterface("en" + string(rune('0'+id)))
				_ = manager.IsWireless()
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}
}

func TestIsWirelessPlatform(_ *testing.T) {
	// This will vary by system, just verify it doesn't panic
	_ = wifi.IsWirelessPlatform("en0")
}

func TestGetInfoPlatform(_ *testing.T) {
	// This will vary by system, just verify it doesn't panic
	_ = wifi.GetInfoPlatform("en0")
}

func TestGetInfo(_ *testing.T) {
	manager := wifi.NewManager("en0")
	info := manager.GetInfo()
	// Just verify it doesn't panic - result depends on system
	_ = info
}

func TestInfoAllFields(t *testing.T) {
	info := wifi.Info{
		SSID:      "MyNetwork",
		BSSID:     "00:11:22:33:44:55",
		Signal:    -50,
		Channel:   11,
		Frequency: 2462,
		Security:  "WPA3",
	}

	if info.SSID != "MyNetwork" {
		t.Errorf("expected SSID 'MyNetwork', got %q", info.SSID)
	}
	if info.BSSID != "00:11:22:33:44:55" {
		t.Errorf("expected BSSID '00:11:22:33:44:55', got %q", info.BSSID)
	}
	if info.Signal != -50 {
		t.Errorf("expected Signal -50, got %d", info.Signal)
	}
	if info.Channel != 11 {
		t.Errorf("expected Channel 11, got %d", info.Channel)
	}
	if info.Frequency != 2462 {
		t.Errorf("expected Frequency 2462, got %d", info.Frequency)
	}
	if info.Security != "WPA3" {
		t.Errorf("expected Security 'WPA3', got %q", info.Security)
	}
}

func TestMapSecurityTypeMoreCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"WPA2-PSK-CCMP", "WPA2"},
		{"WPA-ENTERPRISE", "WPA"},
		{"wep40", "WEP"},
		{"Open", "Open"},
		{"none", "Open"},
		{"MIXED", "MIXED"},
		{"RSN", "RSN"},
		{"PSK", "PSK"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := wifi.MapSecurityType(tt.input)
			if result != tt.expected {
				t.Errorf("MapSecurityType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestChannelToFrequency2GHz(t *testing.T) {
	tests := []struct {
		channel  int
		expected int
	}{
		{1, 2412},
		{2, 2417},
		{3, 2422},
		{4, 2427},
		{5, 2432},
		{7, 2442},
		{8, 2447},
		{9, 2452},
		{10, 2457},
		{12, 2467},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := wifi.ChannelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
			}
		})
	}
}

func TestChannelToFrequency5GHz(t *testing.T) {
	tests := []struct {
		channel  int
		expected int
	}{
		{52, 5260},
		{56, 5280},
		{60, 5300},
		{64, 5320},
		{100, 5500},
		{104, 5520},
		{108, 5540},
		{112, 5560},
		{116, 5580},
		{120, 5600},
		{124, 5620},
		{128, 5640},
		{132, 5660},
		{136, 5680},
		{140, 5700},
		{144, 5720},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := wifi.ChannelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
			}
		})
	}
}

func TestFrequencyToChannelEdgeCases(t *testing.T) {
	tests := []struct {
		freq     int
		expected int
	}{
		// Out of 2.4 GHz range
		{2400, 0},
		{2500, 0},
		// Out of 5 GHz range
		{5100, 0},
		{5900, 0},
		// Valid 6 GHz
		{5955, 1},
		{6000, 10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := wifi.FrequencyToChannel(tt.freq)
			if result != tt.expected {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, result, tt.expected)
			}
		})
	}
}

func TestManagerInterface(t *testing.T) {
	manager := wifi.NewManager("en0")

	if manager.InterfaceName() != "en0" {
		t.Errorf("expected InterfaceName 'en0', got %q", manager.InterfaceName())
	}

	manager.SetInterface("wlan0")
	if manager.InterfaceName() != "wlan0" {
		t.Errorf("expected InterfaceName 'wlan0', got %q", manager.InterfaceName())
	}
}

func TestIsWirelessResult(_ *testing.T) {
	manager := wifi.NewManager("lo0")

	// Loopback is not wireless
	result := manager.IsWireless()
	// Result depends on system, just verify it returns a boolean
	_ = result
}

func TestConcurrentWifiManagerAccess(_ *testing.T) {
	manager := wifi.NewManager("en0")

	done := make(chan bool)
	for range 5 {
		go func() {
			for range 20 {
				manager.SetInterface("wlan0")
				_ = manager.IsWireless()
				_ = manager.GetInfo()
			}
			done <- true
		}()
	}

	for range 5 {
		<-done
	}
}

func TestChannelToFrequency6GHz(t *testing.T) {
	// 6 GHz band tests - channels not in 2.4/5 GHz specific ranges fall through to 6 GHz
	// The function checks 2.4 GHz (1-13, 14) and 5 GHz (36-64, 100-144, 149-165) first,
	// then any remaining channels 1-233 get 6 GHz frequencies
	tests := []struct {
		name     string
		channel  int
		expected int
	}{
		// Channels that fall through to 6 GHz band (not matching 2.4 or 5 GHz specific ranges)
		{"6 GHz channel 15", 15, 6025},   // 5950 + 15*5
		{"6 GHz channel 35", 35, 6125},   // 5950 + 35*5
		{"6 GHz channel 65", 65, 6275},   // 5950 + 65*5
		{"6 GHz channel 99", 99, 6445},   // 5950 + 99*5
		{"6 GHz channel 145", 145, 6675}, // 5950 + 145*5
		{"6 GHz channel 148", 148, 6690}, // 5950 + 148*5
		{"6 GHz channel 166", 166, 6780}, // 5950 + 166*5
		{"6 GHz channel 193", 193, 6915}, // 5950 + 193*5
		{"6 GHz channel 233", 233, 7115}, // 5950 + 233*5 = 7115

		// Channels outside all valid ranges
		{"Channel 0 (invalid)", 0, 0},
		{"Channel 234 (invalid)", 234, 0},
		{"Channel 250 (invalid)", 250, 0},
		{"Negative channel", -100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wifi.ChannelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
			}
		})
	}
}

func TestChannelToFrequencyBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		channel  int
		expected int
	}{
		// 2.4 GHz boundaries
		{"Channel 1 lower bound", 1, 2412},
		{"Channel 13 upper bound", 13, 2472},
		{"Channel 14 special", 14, 2484},

		// 5 GHz UNII-1 boundaries
		{"Channel 36 UNII-1 start", 36, 5180},
		{"Channel 64 UNII-1 end", 64, 5320},

		// 5 GHz UNII-2E boundaries
		{"Channel 100 UNII-2E start", 100, 5500},
		{"Channel 144 UNII-2E end", 144, 5720},

		// 5 GHz UNII-3 boundaries
		{"Channel 149 UNII-3 start", 149, 5745},
		{"Channel 165 UNII-3 end", 165, 5825},

		// Invalid ranges - Channel 0 and negative are truly invalid
		{"Channel 0", 0, 0},

		// Note: Channels not in 2.4 GHz (1-13,14) or 5 GHz (36-64, 100-144, 149-165)
		// but within 1-233 fall through to 6 GHz band
		// These are tested in TestChannelToFrequency6GHz
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wifi.ChannelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
			}
		})
	}
}

func TestFrequencyToChannel6GHz(t *testing.T) {
	tests := []struct {
		name     string
		freq     int
		expected int
	}{
		// 6 GHz band (5955 MHz - 7115 MHz)
		{"6 GHz channel 1", 5955, 1},
		{"6 GHz channel 5", 5975, 5},
		{"6 GHz channel 9", 5995, 9},
		{"6 GHz channel 13", 6015, 13},
		{"6 GHz channel 17", 6035, 17},
		{"6 GHz channel 21", 6055, 21},
		{"6 GHz channel 93", 6415, 93},
		{"6 GHz channel 233", 7115, 233},

		// Boundaries
		{"6 GHz lower bound", 5955, 1},
		{"6 GHz upper bound", 7115, 233},

		// Just outside 6 GHz band
		{"Below 6 GHz band", 5954, 0},
		{"Above 6 GHz band", 7116, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wifi.FrequencyToChannel(tt.freq)
			if result != tt.expected {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, result, tt.expected)
			}
		})
	}
}

func TestFrequencyToChannelBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		freq     int
		expected int
	}{
		// 2.4 GHz boundaries
		{"2.4 GHz lower bound", 2412, 1},
		{"2.4 GHz upper bound", 2472, 13},
		{"Channel 14", 2484, 14},
		{"Below 2.4 GHz", 2411, 0},
		{"Above 2.4 GHz normal", 2473, 0},
		{"Between normal and ch14", 2483, 0},
		{"Above channel 14", 2485, 0},

		// 5 GHz boundaries
		{"5 GHz lower bound", 5180, 36},
		{"5 GHz upper bound", 5825, 165},
		{"Below 5 GHz", 5179, 0},
		{"Above 5 GHz", 5826, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wifi.FrequencyToChannel(tt.freq)
			if result != tt.expected {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, result, tt.expected)
			}
		})
	}
}

func TestDetectChannelWidthMoreHTModes(t *testing.T) {
	tests := []struct {
		name   string
		freq   int
		htMode string
		want   int
	}{
		// VHT modes
		{"VHT20", 5180, "VHT20", 20},
		{"VHT40", 5180, "VHT40", 40},

		// HE (WiFi 6) modes
		{"HE20", 5180, "HE20", 20},
		{"HE40", 5180, "HE40", 40},
		{"HE80", 5180, "HE80", 80},

		// EHT (WiFi 7) modes
		{"EHT20", 5955, "EHT20", 20},
		{"EHT40", 5955, "EHT40", 40},
		{"EHT80", 5955, "EHT80", 80},
		{"EHT160", 5955, "EHT160", 160},

		// Unknown/empty modes with different bands
		{"Empty mode 2.4 GHz", 2437, "", 20},
		{"Empty mode 5 GHz", 5745, "", 80},
		{"Empty mode 6 GHz", 6000, "", 160},
		{"Unknown mode", 5180, "UNKNOWN", 80}, // Falls through to band detection
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.DetectChannelWidth(tt.freq, tt.htMode)
			if got != tt.want {
				t.Errorf("DetectChannelWidth(%d, %q) = %d, want %d", tt.freq, tt.htMode, got, tt.want)
			}
		})
	}
}

func TestGetBandBoundaries(t *testing.T) {
	tests := []struct {
		name string
		freq int
		want string
	}{
		// 2.4 GHz band boundaries (2400-2500 MHz)
		{"2.4 GHz lower bound", 2400, "2.4GHz"},
		{"2.4 GHz upper bound", 2500, "2.4GHz"},
		{"Below 2.4 GHz", 2399, ""},
		{"Above 2.4 GHz", 2501, ""},

		// 5 GHz band boundaries (5150-5895 MHz)
		{"5 GHz lower bound", 5150, "5GHz"},
		{"5 GHz upper bound", 5895, "5GHz"},
		{"Below 5 GHz", 5149, ""},
		{"Above 5 GHz", 5896, ""},

		// 6 GHz band boundaries (5925-7125 MHz)
		{"6 GHz lower bound", 5925, "6GHz"},
		{"6 GHz upper bound", 7125, "6GHz"},
		{"Below 6 GHz", 5924, ""},
		{"Above 6 GHz", 7126, ""},

		// Gap between bands
		{"Between 2.4 and 5 GHz", 3000, ""},
		{"Between 5 and 6 GHz", 5910, ""},
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
