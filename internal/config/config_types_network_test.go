package config_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestAllEthernet and TestAllWiFi guard the contract the two helpers actually
// promise — Default/WiFi folded in first, blanks skipped, duplicates dropped,
// order otherwise preserved.
//
// Neither had any test coverage, which is why the capacity hints in them could
// be changed with nothing to catch a mistake (CodeQL 407-410).
func TestAllEthernet(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.InterfaceConfig
		want []string
	}{
		{"empty", config.InterfaceConfig{}, []string{}},
		{"default only", config.InterfaceConfig{Default: "eth0"}, []string{"eth0"}},
		{"list only", config.InterfaceConfig{Ethernet: []string{"eth1", "eth2"}}, []string{"eth1", "eth2"}},
		{
			"default is first even when it also appears in the list",
			config.InterfaceConfig{Default: "eth0", Ethernet: []string{"eth1", "eth0"}},
			[]string{"eth0", "eth1"},
		},
		{
			"blanks skipped and duplicates dropped",
			config.InterfaceConfig{Default: "eth0", Ethernet: []string{"eth1", "", "eth1", "eth2"}},
			[]string{"eth0", "eth1", "eth2"},
		},
		{
			"blank default is not folded in",
			config.InterfaceConfig{Default: "", Ethernet: []string{"eth1"}},
			[]string{"eth1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.AllEthernet(); !slices.Equal(got, tt.want) {
				t.Errorf("AllEthernet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllWiFi(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.InterfaceConfig
		want []string
	}{
		{"empty", config.InterfaceConfig{}, []string{}},
		{"primary only", config.InterfaceConfig{WiFi: "wlan0"}, []string{"wlan0"}},
		{"list only", config.InterfaceConfig{WiFiList: []string{"wlan1"}}, []string{"wlan1"}},
		{
			"primary is first even when it also appears in the list",
			config.InterfaceConfig{WiFi: "wlan0", WiFiList: []string{"wlan1", "wlan0"}},
			[]string{"wlan0", "wlan1"},
		},
		{
			"blanks skipped and duplicates dropped",
			config.InterfaceConfig{WiFi: "wlan0", WiFiList: []string{"wlan1", "", "wlan1"}},
			[]string{"wlan0", "wlan1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.AllWiFi(); !slices.Equal(got, tt.want) {
				t.Errorf("AllWiFi() = %v, want %v", got, tt.want)
			}
		})
	}
}
