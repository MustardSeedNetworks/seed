package airspace

import (
	"sort"
	"time"
)

// The Tree view is the read model the API + UI consume: the cross-referenced
// SSID → AP → BSSID → client hierarchy. All slices are sorted deterministically
// (no map-iteration or time ordering leaks) so equal inputs render identically.

// StationView is one client under a BSS.
type StationView struct {
	MAC       string    `json:"mac"`
	SignalDBm int       `json:"signalDbm"`
	Frames    int       `json:"frames"`
	LastSeen  time.Time `json:"lastSeen"`
}

// BSSView is one BSSID (radio) under an AP.
type BSSView struct {
	BSSID        string        `json:"bssid"`
	SSID         string        `json:"ssid"`
	Hidden       bool          `json:"hidden"`
	Band         string        `json:"band"`
	Channel      int           `json:"channel"`
	Security     string        `json:"security"`
	Standard     string        `json:"standard"`
	CountryCode  string        `json:"countryCode,omitempty"`
	PMFRequired  bool          `json:"pmfRequired"`
	RRMNeighbor  bool          `json:"rrmNeighbor"`
	BTMSupported bool          `json:"btmSupported"`
	FTSupported  bool          `json:"ftSupported"`
	WPSEnabled   bool          `json:"wpsEnabled"`
	SignalDBm    int           `json:"signalDbm"`
	Beacons      int           `json:"beacons"`
	LastSeen     time.Time     `json:"lastSeen"`
	Stations     []StationView `json:"stations"`
}

// APGroup clusters the BSSIDs believed to belong to one physical AP.
type APGroup struct {
	Key    string    `json:"key"`
	Vendor string    `json:"vendor,omitempty"`
	BSSes  []BSSView `json:"bsses"`
}

// SSIDGroup is the top level: one advertised SSID and the APs serving it.
type SSIDGroup struct {
	SSID         string    `json:"ssid"`
	Hidden       bool      `json:"hidden"`
	APCount      int       `json:"apCount"`
	BSSCount     int       `json:"bssCount"`
	StationCount int       `json:"stationCount"`
	APs          []APGroup `json:"aps"`
}

// Tree renders the current airspace as the sorted SSID → AP → BSSID → client
// hierarchy. Hidden/un-named BSSes are grouped under an empty SSID string.
func (a *Airspace) Tree() []SSIDGroup {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Bucket BSSes by SSID, then by AP key.
	bySSID := make(map[string]map[string][]*bss)
	for _, b := range a.bsses {
		apBuckets, ok := bySSID[b.ssid]
		if !ok {
			apBuckets = make(map[string][]*bss)
			bySSID[b.ssid] = apBuckets
		}
		k := apKey(b.bssid)
		apBuckets[k] = append(apBuckets[k], b)
	}

	groups := make([]SSIDGroup, 0, len(bySSID))
	for ssid, apBuckets := range bySSID {
		groups = append(groups, buildSSIDGroup(ssid, apBuckets))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].SSID < groups[j].SSID })
	return groups
}

func buildSSIDGroup(ssid string, apBuckets map[string][]*bss) SSIDGroup {
	g := SSIDGroup{SSID: ssid}
	for key, bsses := range apBuckets {
		ap := APGroup{Key: key, Vendor: vendorOUI(key)}
		for _, b := range bsses {
			ap.BSSes = append(ap.BSSes, toBSSView(b))
			g.BSSCount++
			g.StationCount += len(b.stations)
			if b.hidden {
				g.Hidden = true
			}
		}
		sort.Slice(ap.BSSes, func(i, j int) bool { return ap.BSSes[i].BSSID < ap.BSSes[j].BSSID })
		g.APs = append(g.APs, ap)
	}
	sort.Slice(g.APs, func(i, j int) bool { return g.APs[i].Key < g.APs[j].Key })
	g.APCount = len(g.APs)
	return g
}

func toBSSView(b *bss) BSSView {
	v := BSSView{
		BSSID:        b.bssid,
		SSID:         b.ssid,
		Hidden:       b.hidden,
		Band:         b.band.String(),
		Channel:      b.channel,
		Security:     b.security.String(),
		Standard:     b.standard.String(),
		CountryCode:  b.countryCode,
		PMFRequired:  b.pmfRequired,
		RRMNeighbor:  b.rrmNeighbor,
		BTMSupported: b.btm,
		FTSupported:  b.ft,
		WPSEnabled:   b.wps,
		SignalDBm:    int(b.signalDBm),
		Beacons:      b.beacons,
		LastSeen:     b.lastSeen,
		Stations:     make([]StationView, 0, len(b.stations)),
	}
	for _, s := range b.stations {
		v.Stations = append(v.Stations, StationView{
			MAC:       s.mac,
			SignalDBm: int(s.signalDBm),
			Frames:    s.frames,
			LastSeen:  s.lastSeen,
		})
	}
	sort.Slice(v.Stations, func(i, j int) bool { return v.Stations[i].MAC < v.Stations[j].MAC })
	return v
}
