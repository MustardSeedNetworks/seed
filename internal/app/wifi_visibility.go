package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
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

// WiFiWiredSource feeds the rogue-AP-on-LAN rule the MAC addresses known on the
// wired side: every device in discovery's wired registry (the probe's own
// ARP/NDP view) plus the newest ARP table and forwarding database each polled
// SNMP target reported. A nil database contributes no polled MACs.
func WiFiWiredSource(
	devices func() []*discovery.DiscoveredDevice,
	persist func() *Persistence,
) visibility.WiredSource {
	return func(ctx context.Context) ([]string, error) {
		var macs []string
		for _, d := range devices() {
			if d.MAC != "" {
				macs = append(macs, d.MAC)
			}
		}
		conn := lazyHandle(persist)()
		if conn == nil {
			return macs, nil
		}
		for _, kind := range []string{sink.KindARP, sink.KindFDB} {
			polled, err := latestPolledMACs(ctx, conn.SNMPObservations(), kind)
			if err != nil {
				return nil, err
			}
			macs = append(macs, polled...)
		}
		return macs, nil
	}
}

// polledObservationPage is the largest page the observation repository returns.
// Rows come newest first, so every target that polled recently is in it.
const polledObservationPage = 1000

// polledTable is the part of an ARP or FDB observation payload the wired set
// reads; ARP rows carry no status.
type polledTable struct {
	Entries []struct {
		MACAddress string
		Status     int
	}
}

// latestPolledMACs returns the MACs in the newest observation of kind from each
// polled target. Older observations are what the target used to hold.
func latestPolledMACs(ctx context.Context, repo *database.SNMPObservationsRepository, kind string) ([]string, error) {
	rows, err := repo.List(ctx, observation.ListOptions{Kind: kind, Limit: polledObservationPage})
	if err != nil {
		return nil, fmt.Errorf("list %s observations: %w", kind, err)
	}
	seen := make(map[string]struct{})
	var macs []string
	for _, row := range rows {
		target := row.ClientID + "/" + row.TargetID
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		var table polledTable
		if decodeErr := json.Unmarshal([]byte(row.PayloadJSON), &table); decodeErr != nil {
			return nil, fmt.Errorf("decode %s observation %d: %w", kind, row.ID, decodeErr)
		}
		for _, e := range table.Entries {
			if e.MACAddress != "" && e.Status != fdb.StatusInvalid {
				macs = append(macs, e.MACAddress)
			}
		}
	}
	return macs, nil
}
