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
	BSSID        string `json:"bssid"`
	SSID         string `json:"ssid"`
	Hidden       bool   `json:"hidden"`
	Band         string `json:"band"`
	Channel      int    `json:"channel"`
	Security     string `json:"security"`
	Standard     string `json:"standard"`
	CountryCode  string `json:"countryCode,omitempty"`
	PMFRequired  bool   `json:"pmfRequired"`
	RRMNeighbor  bool   `json:"rrmNeighbor"`
	BTMSupported bool   `json:"btmSupported"`
	FTSupported  bool   `json:"ftSupported"`
	WPSEnabled   bool   `json:"wpsEnabled"`
	// ChannelWidthMHz is the operating width (20/40/80/160/320), 0 if unknown.
	ChannelWidthMHz int `json:"channelWidthMhz"`
	// ChannelUtil is the BSS Load channel-utilization figure (0-255); valid only
	// when HasBSSLoad. AdvertisedStations is the AP's advertised association count.
	ChannelUtil        int  `json:"channelUtil"`
	AdvertisedStations int  `json:"advertisedStations"`
	HasBSSLoad         bool `json:"hasBssLoad"`
	SignalDBm          int  `json:"signalDbm"`
	Beacons            int  `json:"beacons"`
	// RecentDeauths is the number of deauth/disassoc frames seen for this BSSID
	// within the current retention window; a spike feeds the deauth-flood rule.
	RecentDeauths int           `json:"recentDeauths"`
	LastSeen      time.Time     `json:"lastSeen"`
	Stations      []StationView `json:"stations"`
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
		// Convert the internal BSSes to views, then assemble with the shared
		// builder so the live Tree and TreeFromBSSViews stay identical.
		viewBuckets := make(map[string][]BSSView, len(apBuckets))
		for key, bsses := range apBuckets {
			views := make([]BSSView, 0, len(bsses))
			for _, b := range bsses {
				views = append(views, toBSSView(b))
			}
			viewBuckets[key] = views
		}
		groups = append(groups, assembleSSIDGroup(ssid, viewBuckets))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].SSID < groups[j].SSID })
	return groups
}

// TreeFromBSSViews assembles the sorted SSID → AP → BSSID hierarchy from a flat
// set of BSS views (e.g. captured during a site survey), using the same
// AP-clustering and ordering as the live Tree. Stations are carried through from
// each view as-is. It is a pure function of its input — no airspace state.
func TreeFromBSSViews(bsses []BSSView) []SSIDGroup {
	bySSID := make(map[string]map[string][]BSSView)
	for _, b := range bsses {
		apBuckets, ok := bySSID[b.SSID]
		if !ok {
			apBuckets = make(map[string][]BSSView)
			bySSID[b.SSID] = apBuckets
		}
		k := apKey(b.BSSID)
		apBuckets[k] = append(apBuckets[k], b)
	}

	groups := make([]SSIDGroup, 0, len(bySSID))
	for ssid, apBuckets := range bySSID {
		groups = append(groups, assembleSSIDGroup(ssid, apBuckets))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].SSID < groups[j].SSID })
	return groups
}

// assembleSSIDGroup builds one SSIDGroup from AP-keyed BSS views, applying the
// deterministic BSS/AP ordering and the cross-reference counts.
func assembleSSIDGroup(ssid string, apBuckets map[string][]BSSView) SSIDGroup {
	g := SSIDGroup{SSID: ssid}
	for key, views := range apBuckets {
		ap := APGroup{Key: key, Vendor: vendorOUI(key)}
		for _, v := range views {
			ap.BSSes = append(ap.BSSes, v)
			g.BSSCount++
			g.StationCount += len(v.Stations)
			if v.Hidden {
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
		BSSID:              b.bssid,
		SSID:               b.ssid,
		Hidden:             b.hidden,
		Band:               b.band.String(),
		Channel:            b.channel,
		Security:           b.security.String(),
		Standard:           b.standard.String(),
		CountryCode:        b.countryCode,
		PMFRequired:        b.pmfRequired,
		RRMNeighbor:        b.rrmNeighbor,
		BTMSupported:       b.btm,
		FTSupported:        b.ft,
		WPSEnabled:         b.wps,
		ChannelWidthMHz:    b.channelWidthM,
		ChannelUtil:        b.chanUtil,
		AdvertisedStations: b.advertStations,
		HasBSSLoad:         b.hasBSSLoad,
		SignalDBm:          int(b.signalDBm),
		Beacons:            b.beacons,
		RecentDeauths:      len(b.deauthTimes),
		LastSeen:           b.lastSeen,
		Stations:           make([]StationView, 0, len(b.stations)),
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
