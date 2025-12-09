package wifi

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("en0")

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.interfaceName != "en0" {
		t.Errorf("expected interfaceName 'en0', got %q", manager.interfaceName)
	}
}

func TestManagerSetInterface(t *testing.T) {
	manager := NewManager("en0")

	manager.SetInterface("wlan0")
	if manager.interfaceName != "wlan0" {
		t.Errorf("expected interfaceName 'wlan0', got %q", manager.interfaceName)
	}
}

func TestManagerIsWireless(t *testing.T) {
	manager := NewManager("en0")

	// Result depends on system, just verify it doesn't panic
	_ = manager.IsWireless()
}

func TestManagerGetInfo(t *testing.T) {
	manager := NewManager("en0")

	// Result depends on system, just verify it doesn't panic
	_ = manager.GetInfo()
}

func TestInfoFields(t *testing.T) {
	info := Info{
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
			result := mapSecurityType(tt.input)
			if result != tt.expected {
				t.Errorf("mapSecurityType(%q) = %q, want %q", tt.input, result, tt.expected)
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
			result := channelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("channelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
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
			result := frequencyToChannel(tt.freq)
			if result != tt.expected {
				t.Errorf("frequencyToChannel(%d) = %d, want %d", tt.freq, result, tt.expected)
			}
		})
	}
}

func TestChannelFrequencyRoundTrip(t *testing.T) {
	// Test that converting channel -> frequency -> channel gives original
	channels := []int{1, 6, 11, 36, 40, 44, 48, 149, 153, 157, 161, 165}

	for _, ch := range channels {
		freq := channelToFrequency(ch)
		if freq == 0 {
			continue
		}
		result := frequencyToChannel(freq)
		if result != ch {
			t.Errorf("roundtrip failed: channel %d -> freq %d -> channel %d", ch, freq, result)
		}
	}
}

func TestConcurrentManagerAccess(t *testing.T) {
	manager := NewManager("en0")

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				manager.SetInterface("en" + string(rune('0'+id)))
				_ = manager.IsWireless()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestIsWirelessPlatform(t *testing.T) {
	// This will vary by system, just verify it doesn't panic
	_ = isWirelessPlatform("en0")
}

func TestGetInfoPlatform(t *testing.T) {
	// This will vary by system, just verify it doesn't panic
	_ = getInfoPlatform("en0")
}

func TestGetInfo(t *testing.T) {
	manager := NewManager("en0")
	info := manager.GetInfo()
	// Just verify it doesn't panic - result depends on system
	_ = info
}

func TestInfoAllFields(t *testing.T) {
	info := Info{
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
			result := mapSecurityType(tt.input)
			if result != tt.expected {
				t.Errorf("mapSecurityType(%q) = %q, want %q", tt.input, result, tt.expected)
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
			result := channelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("channelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
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
			result := channelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("channelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
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
			result := frequencyToChannel(tt.freq)
			if result != tt.expected {
				t.Errorf("frequencyToChannel(%d) = %d, want %d", tt.freq, result, tt.expected)
			}
		})
	}
}

func TestManagerInterface(t *testing.T) {
	manager := NewManager("en0")

	if manager.interfaceName != "en0" {
		t.Errorf("expected interfaceName 'en0', got %q", manager.interfaceName)
	}

	manager.SetInterface("wlan0")
	if manager.interfaceName != "wlan0" {
		t.Errorf("expected interfaceName 'wlan0', got %q", manager.interfaceName)
	}
}

func TestIsWirelessResult(t *testing.T) {
	manager := NewManager("lo0")

	// Loopback is not wireless
	result := manager.IsWireless()
	// Result depends on system, just verify it returns a boolean
	_ = result
}

func TestConcurrentWifiManagerAccess(t *testing.T) {
	manager := NewManager("en0")

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				manager.SetInterface("wlan0")
				_ = manager.IsWireless()
				_ = manager.GetInfo()
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
