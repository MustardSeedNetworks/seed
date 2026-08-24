//go:build darwin

package wifi_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

type fakeHelper struct{}

func (fakeHelper) Scan() ([]wifihelper.Network, error)  { return nil, nil }
func (fakeHelper) Current() (wifihelper.Network, error) { return wifihelper.Network{}, nil }
func (fakeHelper) Saved() ([]string, error)             { return nil, nil }

// A scanner carries its own helper, so two scanners cannot disturb each other
// the way shared package state would allow.
func TestSetHelperIsPerScanner(t *testing.T) {
	t.Parallel()

	withHelper := wifi.NewScanner("en0")
	withHelper.SetHelper(fakeHelper{})

	without := wifi.NewScanner("en0")

	if withHelper.CurrentHelper() == nil {
		t.Error("SetHelper did not register the helper")
	}
	if without.CurrentHelper() != nil {
		t.Error("a second scanner inherited another scanner's helper")
	}

	withHelper.SetHelper(nil)
	if withHelper.CurrentHelper() != nil {
		t.Error("SetHelper(nil) did not clear the helper")
	}
}

// Only a missing Location grant is worth retrying through the helper. Any other
// failure would fail the same way there, and delegating would replace a precise
// message with a vaguer one.
func TestShouldDelegate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"location denied delegates", corewlan.ErrLocationDenied, true},
		{"wrapped location denied delegates", errors.Join(errors.New("scan"), corewlan.ErrLocationDenied), true},
		{"no interface does not delegate", corewlan.ErrNoInterface, false},
		{"unsupported does not delegate", corewlan.ErrUnsupported, false},
		{"unrelated error does not delegate", errors.New("boom"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := wifi.ShouldDelegate(tc.err); got != tc.want {
				t.Errorf("ShouldDelegate(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// A helper observation must derive the same fields as an in-process one, so an
// operator sees identical data whichever path served the scan.
func TestNetworkFromHelperMatchesDirect(t *testing.T) {
	t.Parallel()

	via := wifi.NetworkFromHelper(wifihelper.Network{
		SSID: "Neuroplasticity", BSSID: "24:5a:4c:6b:b5:c9", RSSI: -54, Noise: -87,
		Channel: 149, ChannelWidth: 40, Band: 5, Security: "wpa3Transition",
	})
	direct := wifi.NetworkFromCoreWLAN(corewlan.Network{
		SSID: "Neuroplasticity", BSSID: "24:5a:4c:6b:b5:c9", RSSI: -54, Noise: -87,
		Channel: 149, ChannelWidth: 40, Band: corewlan.Band5GHz, Security: "wpa3Transition",
	})

	if *via != *direct {
		t.Errorf("helper path\n got = %+v\nwant = %+v", *via, *direct)
	}
}
