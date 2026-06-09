package survey

import (
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// Band boundary frequencies in MHz used to label a scanned network's band so the
// anomaly rules (which classify off the airspace band string) see the same
// tokens the live capture path produces.
const (
	band24MinMHz = 2400
	band24MaxMHz = 2500
	band5MinMHz  = 4900
	band5MaxMHz  = 5900
	channel24Max = 14 // highest 2.4 GHz channel, used when a scan omits frequency
)

// AnalyzeAnomalies runs the Wi-Fi anomaly engine over the access points captured
// in a survey's passive samples and returns the detected anomalies. It reuses
// the same rule catalog as the live visibility capture path (via
// troubleshooting.AnalyzeBSSes), so a site survey surfaces security/RF/standards
// problems alongside its coverage (dead-zone) analysis.
//
// A survey with no passive AP observations yields no anomalies (and no error).
func AnalyzeAnomalies(survey *Survey) ([]anomaly.Anomaly, error) {
	if survey == nil {
		return nil, nil
	}
	bsses, at := surveyBSSViews(survey)
	if len(bsses) == 0 {
		return nil, nil
	}
	if at.IsZero() {
		at = survey.UpdatedAt
	}
	return troubleshooting.AnalyzeBSSes(bsses, nil, at)
}

// surveyBSSViews flattens a survey's passive-scan observations into the airspace
// BSS view the anomaly detector consumes, de-duplicated by BSSID keeping the
// strongest-signal sighting. It also returns the most recent passive-sample
// timestamp so the caller can stamp the synthetic observations.
func surveyBSSViews(survey *Survey) ([]airspace.BSSView, time.Time) {
	type seen struct {
		view   airspace.BSSView
		signal int
	}
	byBSSID := make(map[string]seen)
	var latest time.Time

	for _, sp := range survey.GetAllSamples() {
		passive := getPassiveSampleFromPoint(sp)
		if passive == nil {
			continue
		}
		if sp.Timestamp.After(latest) {
			latest = sp.Timestamp
		}
		for _, n := range passive.Networks {
			if n == nil || n.BSSID == "" {
				continue
			}
			// Keep the strongest sighting; signals are negative dBm, so a larger
			// (closer to zero) value is stronger.
			if prev, ok := byBSSID[n.BSSID]; ok && n.Signal <= prev.signal {
				continue
			}
			byBSSID[n.BSSID] = seen{view: scannedNetworkToBSSView(n), signal: n.Signal}
		}
	}

	views := make([]airspace.BSSView, 0, len(byBSSID))
	for _, s := range byBSSID {
		views = append(views, s.view)
	}
	return views, latest
}

// scannedNetworkToBSSView maps one OS-scan result onto the airspace BSS view.
// Fields a passive OS scan cannot observe (PMF/RRM/BTM/FT/BSS-Load/country, the
// decoded 802.11 standard) are left at their zero values, so the rules that need
// them simply do not fire — passive surveys flag the security/RF/channel issues
// they can actually see without false positives. Security is passed through
// verbatim; recognition is best-effort against the detector's known suites.
func scannedNetworkToBSSView(n *wifi.ScannedNetwork) airspace.BSSView {
	return airspace.BSSView{
		BSSID:           n.BSSID,
		SSID:            n.SSID,
		Hidden:          n.SSID == "",
		Band:            bandLabel(n.Frequency, n.Channel),
		Channel:         n.Channel,
		Security:        n.Security,
		ChannelWidthMHz: n.ChannelWidth,
		SignalDBm:       n.Signal,
	}
}

// bandLabel maps a scanned network's frequency (with a channel fallback) onto
// the airspace band token ("2.4 GHz" / "5 GHz" / "6 GHz" / "Unknown").
func bandLabel(freqMHz, channel int) string {
	switch {
	case freqMHz >= band24MinMHz && freqMHz < band24MaxMHz:
		return "2.4 GHz"
	case freqMHz >= band5MinMHz && freqMHz < band5MaxMHz:
		return "5 GHz"
	case freqMHz >= band5MaxMHz:
		return "6 GHz"
	case freqMHz == 0 && channel >= 1 && channel <= channel24Max:
		return "2.4 GHz"
	default:
		return "Unknown"
	}
}
