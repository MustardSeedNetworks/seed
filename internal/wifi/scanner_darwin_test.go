//go:build darwin

package wifi_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

func TestNetworkFromCoreWLAN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   corewlan.Network
		want wifi.ScannedNetwork
	}{
		{
			name: "5GHz 802.11ax",
			in: corewlan.Network{
				SSID: "Neuroplasticity", BSSID: "24:5a:4c:6b:b5:c9",
				RSSI: -54, Noise: -87, Channel: 149, ChannelWidth: 40,
				Band: corewlan.Band5GHz, Security: "wpa3Transition",
			},
			want: wifi.ScannedNetwork{
				SSID: "Neuroplasticity", BSSID: "24:5a:4c:6b:b5:c9",
				Signal: -54, Channel: 149, Frequency: 5745, Security: "WPA3",
				ChannelWidth: 40, NoiseFloor: -87, SNR: 33, HTMode: "HT40", IsDFS: false,
			},
		},
		{
			// Channel numbers collide across bands: 6 GHz channel 1 is 5955 MHz,
			// not the 2.4 GHz 2412 MHz. Band must drive the conversion.
			name: "6GHz channel 1 does not collide with 2.4GHz",
			in: corewlan.Network{
				BSSID: "aa:bb:cc:dd:ee:01", RSSI: -60, Noise: -90,
				Channel: 1, ChannelWidth: 160, Band: corewlan.Band6GHz, Security: "wpa3Personal",
			},
			want: wifi.ScannedNetwork{
				BSSID: "aa:bb:cc:dd:ee:01", Signal: -60, Channel: 1, Frequency: 5955,
				Security: "WPA3", ChannelWidth: 160, NoiseFloor: -90, SNR: 30,
				HTMode: "HE160", IsDFS: false,
			},
		},
		{
			name: "2.4GHz channel 1",
			in: corewlan.Network{
				SSID: "TMOBILE-3F7A", BSSID: "18:a5:ff:85:3f:7c", RSSI: -45, Noise: -92,
				Channel: 1, ChannelWidth: 20, Band: corewlan.Band2GHz, Security: "wpa2Personal",
			},
			want: wifi.ScannedNetwork{
				SSID: "TMOBILE-3F7A", BSSID: "18:a5:ff:85:3f:7c", Signal: -45,
				Channel: 1, Frequency: 2412, Security: "WPA2", ChannelWidth: 20,
				NoiseFloor: -92, SNR: 47, HTMode: "HT20",
			},
		},
		{
			// CoreWLAN omits the noise floor for scanned networks on some adapters.
			// Fall back to the conservative estimate rather than reporting 0 dBm,
			// which would make SNR nonsense.
			name: "unreported noise falls back to estimate",
			in: corewlan.Network{
				SSID: "NoNoise", BSSID: "aa:bb:cc:dd:ee:02", RSSI: -50,
				Channel: 36, ChannelWidth: 80, Band: corewlan.Band5GHz, Security: "wpa2Personal",
			},
			want: wifi.ScannedNetwork{
				SSID: "NoNoise", BSSID: "aa:bb:cc:dd:ee:02", Signal: -50,
				Channel: 36, Frequency: 5180, Security: "WPA2", ChannelWidth: 80,
				NoiseFloor: -95, SNR: 45, HTMode: "VHT80",
			},
		},
		{
			name: "DFS channel flagged",
			in: corewlan.Network{
				SSID: "Radar", BSSID: "aa:bb:cc:dd:ee:03", RSSI: -70, Noise: -95,
				Channel: 52, ChannelWidth: 20, Band: corewlan.Band5GHz, Security: "none",
			},
			want: wifi.ScannedNetwork{
				SSID: "Radar", BSSID: "aa:bb:cc:dd:ee:03", Signal: -70, Channel: 52,
				Frequency: 5260, Security: "Open", ChannelWidth: 20, NoiseFloor: -95,
				SNR: 25, HTMode: "HT20", IsDFS: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := wifi.NetworkFromCoreWLAN(tc.in)
			if got == nil {
				t.Fatal("NetworkFromCoreWLAN() = nil")
			}
			got.LastSeen = tc.want.LastSeen // set from the clock; not under test
			if *got != tc.want {
				t.Errorf("NetworkFromCoreWLAN()\n got = %+v\nwant = %+v", *got, tc.want)
			}
		})
	}
}

func TestChannelToFrequencyInBand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		channel int
		band    int
		want    int
	}{
		{"2.4GHz ch1", 1, 2, 2412},
		{"2.4GHz ch14", 14, 2, 2484},
		{"5GHz ch36", 36, 5, 5180},
		{"5GHz ch149", 149, 5, 5745},
		{"6GHz ch1", 1, 6, 5955},
		{"6GHz ch233", 233, 6, 7115},
		// An unknown band cannot be resolved from the channel number alone.
		{"unknown band falls back", 149, 0, 5745},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := wifi.ChannelToFrequencyInBand(tc.channel, tc.band); got != tc.want {
				t.Errorf("ChannelToFrequencyInBand(%d, %d) = %d, want %d", tc.channel, tc.band, got, tc.want)
			}
		})
	}
}

func TestAssociationResult(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		got := wifi.AssociationResult("Neuroplasticity", nil)
		if !got.Success || got.SSID != "Neuroplasticity" {
			t.Errorf("AssociationResult() = %+v, want success for the named network", got)
		}
	})

	// A refused association is an outcome the caller renders, not a Go error.
	t.Run("failure carries the reason", func(t *testing.T) {
		t.Parallel()

		got := wifi.AssociationResult("Ghost", errors.New("network Ghost not found in range"))
		if got.Success {
			t.Error("AssociationResult() reported success for a failed join")
		}
		if !strings.Contains(got.Message, "not found in range") {
			t.Errorf("AssociationResult() message = %q, want the underlying reason", got.Message)
		}
		if got.SSID != "Ghost" {
			t.Errorf("AssociationResult() SSID = %q, want %q", got.SSID, "Ghost")
		}
	})
}

func TestDisassociationResult(t *testing.T) {
	t.Parallel()

	if got := wifi.DisassociationResult(nil); !got.Success {
		t.Errorf("DisassociationResult(nil) = %+v, want success", got)
	}

	got := wifi.DisassociationResult(errors.New("no Wi-Fi interface"))
	if got.Success {
		t.Error("DisassociationResult() reported success despite an error")
	}
	if !strings.Contains(got.Message, "no Wi-Fi interface") {
		t.Errorf("DisassociationResult() message = %q, want the underlying reason", got.Message)
	}
}
