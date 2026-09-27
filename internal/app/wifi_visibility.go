package app

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// NewWiFiVisibility builds the Wi-Fi airspace visibility component, fed decoded
// frames by the capture source (not the DB). It is a producer into the shared,
// server-owned anomaly Coordinator (ADR-0029): the cmd layer builds it before the
// server owns the merged engine, so the server injects the Coordinator later via
// Service.SetCoordinator. Until then it runs airspace-only.
func NewWiFiVisibility() *visibility.Service {
	return visibility.New()
}

// WiFiScanSource feeds the visibility component from the Wi-Fi management scan
// use-case, on the configured Wi-Fi interface. A scan that could not run (no
// adapter, withheld details, a failed scan) is an error rather than an empty
// airspace, so it never resolves the anomalies the last good scan raised.
func WiFiScanSource(m *troubleshooting.Management) visibility.ScanSource {
	return func() ([]airspace.BSSView, error) {
		res := m.Scan("")
		if !res.Available || res.Error != "" {
			return nil, fmt.Errorf("scan on %q: %s", res.Interface, res.Error)
		}
		views := make([]airspace.BSSView, 0, len(res.Networks))
		for _, n := range res.Networks {
			views = append(views, scanBSSView(n))
		}
		return views, nil
	}
}

// scanBSSView maps one scanned network onto the view the Wi-Fi rules read. The
// BSSID is lowercased to the key capture uses, so a BSS seen both ways merges.
func scanBSSView(n *wifi.ScannedNetwork) airspace.BSSView {
	band, _ := dot11.BandAndChannel(n.Frequency)
	return airspace.BSSView{
		BSSID:              strings.ToLower(n.BSSID),
		SSID:               n.SSID,
		Hidden:             n.Hidden,
		Band:               band.String(),
		Channel:            n.Channel,
		Security:           n.Security,
		Standard:           n.Standard,
		CountryCode:        n.CountryCode,
		PMFRequired:        n.PMFRequired,
		RRMNeighbor:        n.RRMNeighbor,
		BTMSupported:       n.BTMSupported,
		FTSupported:        n.FTSupported,
		WPSEnabled:         n.WPSEnabled,
		ChannelWidthMHz:    n.ChannelWidth,
		ChannelUtil:        n.ChannelUtil,
		AdvertisedStations: n.AdvertisedStations,
		HasBSSLoad:         n.HasBSSLoad,
		SignalDBm:          n.Signal,
		LastSeen:           n.LastSeen,
		Stations:           []airspace.StationView{},
	}
}
