// Package airspace aggregates decoded 802.11 frames (internal/wifi/dot11) into
// a live, cross-referenced SSID → AP → BSSID → client model — the Wi-Fi
// analogue of the wired discovery device tree (ADR-0012). It is pure data and
// pure aggregation: CGO-free, no I/O, no hidden clock (callers pass the
// observation time), so it is fully unit-testable off a monitor-mode host.
package airspace

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/wifi/dot11"
)

// apGroupMask groups the BSSIDs of one physical AP. Many APs assign their
// per-band/per-SSID BSSIDs as a run that differs only in the low bits of the
// last octet, so masking those bits clusters an AP's radios. It is a heuristic
// (the only signal available from passive capture) — documented, not exact.
const apGroupMask = 0xF0

// station is a client observed on a BSS (internal aggregation state).
type station struct {
	mac       string
	signalDBm int8
	firstSeen time.Time
	lastSeen  time.Time
	frames    int
}

// bss is a single BSSID (one radio advertising one SSID) with its decoded
// capabilities and the clients seen on it (internal aggregation state).
type bss struct {
	bssid       string
	ssid        string
	hidden      bool
	band        dot11.Band
	channel     int
	security    dot11.Security
	standard    dot11.Standard
	countryCode string
	pmfRequired bool
	rrmNeighbor bool
	btm         bool
	ft          bool
	wps         bool
	signalDBm   int8
	// Channel width + BSS Load (802.11e), decoded from the beacon IEs.
	channelWidthM  int
	chanUtil       int // raw 0..255 channel utilization from the BSS Load IE
	advertStations int // station count the AP advertises in the BSS Load IE
	hasBSSLoad     bool
	firstSeen      time.Time
	lastSeen       time.Time
	beacons        int
	stations       map[string]*station
	// deauthTimes holds the observation times of deauthentication/disassociation
	// frames attributed to this BSSID, kept as a sliding window (Prune drops
	// entries older than the cutoff). A spike feeds the deauth-flood rule (W4e).
	deauthTimes []time.Time
}

// Airspace is the live aggregate of everything seen on the air. Safe for
// concurrent Observe/Tree/Prune (the capture loop writes; the API reads).
type Airspace struct {
	mu    sync.RWMutex
	bsses map[string]*bss
}

// New returns an empty Airspace.
func New() *Airspace {
	return &Airspace{bsses: make(map[string]*bss)}
}

// Observe folds one decoded frame into the model at observation time `at`.
// Beacons/probe-responses build and refresh BSSes; association and data frames
// attribute client stations to the right BSSID (via the frame's DS-aware
// BSSIDOf/StationOf). A nil frame is a no-op.
func (a *Airspace) Observe(f *dot11.Frame, at time.Time) {
	if f == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if f.BSS != nil {
		a.applyBeacon(f, at)
	}
	if f.Kind == dot11.KindDeauth || f.Kind == dot11.KindDisassoc {
		a.applyDeauth(f, at)
	}
	a.applyStation(f, at)
}

// applyDeauth records a deauthentication/disassociation frame against its BSS
// (creating a thin entry if needed), keeping the sliding window the deauth-flood
// rule reads. Frames that do not resolve to a BSSID are ignored.
func (a *Airspace) applyDeauth(f *dot11.Frame, at time.Time) {
	bssid := macKey(f.BSSIDOf())
	if bssid == "" {
		return
	}
	b := a.ensureBSS(bssid, at)
	b.deauthTimes = append(b.deauthTimes, at)
	if at.After(b.lastSeen) {
		b.lastSeen = at
	}
}

// applyBeacon upserts the BSS advertised by a beacon/probe-response.
func (a *Airspace) applyBeacon(f *dot11.Frame, at time.Time) {
	key := macKey(f.BSSID)
	if key == "" {
		return
	}
	b := a.ensureBSS(key, at)
	info := f.BSS
	if info.SSID != "" {
		b.ssid = info.SSID
	}
	b.hidden = info.Hidden && info.SSID == ""
	b.band = f.Band
	if f.ChannelNum != 0 {
		b.channel = f.ChannelNum
	}
	b.security = info.Security
	b.standard = info.Standard
	b.countryCode = info.CountryCode
	b.pmfRequired = info.PMFRequired
	b.rrmNeighbor = info.RRMNeighbor
	b.btm = info.BTMSupported
	b.ft = info.FTSupported
	b.wps = info.WPSEnabled
	b.channelWidthM = info.ChannelWidthM
	if info.HasBSSLoad {
		b.hasBSSLoad = true
		b.chanUtil = info.ChannelUtilByte
		b.advertStations = info.StationCount
	}
	if f.SignalDBm != 0 {
		b.signalDBm = f.SignalDBm
	}
	b.beacons++
	b.lastSeen = at
}

// applyStation attributes a client station to its BSS, if the frame identifies
// one. Frames where the "station" is actually the AP (mgmt from the BSSID
// itself) are ignored.
func (a *Airspace) applyStation(f *dot11.Frame, at time.Time) {
	bssid := macKey(f.BSSIDOf())
	sta := macKey(f.StationOf())
	if bssid == "" || sta == "" || sta == bssid {
		return
	}
	b := a.ensureBSS(bssid, at)
	s, ok := b.stations[sta]
	if !ok {
		s = &station{mac: sta, firstSeen: at}
		b.stations[sta] = s
	}
	if f.SignalDBm != 0 {
		s.signalDBm = f.SignalDBm
	}
	s.frames++
	s.lastSeen = at
	if at.After(b.lastSeen) {
		b.lastSeen = at
	}
}

// ensureBSS returns the BSS for a key, creating a thin entry (no beacon yet)
// if a client was seen on a BSSID we have not yet decoded a beacon for.
func (a *Airspace) ensureBSS(key string, at time.Time) *bss {
	b, ok := a.bsses[key]
	if !ok {
		b = &bss{bssid: key, firstSeen: at, stations: make(map[string]*station)}
		a.bsses[key] = b
	}
	return b
}

// Prune removes stations not seen since cutoff, and then BSSes that have no
// beacon and no remaining stations (so transient client-only entries age out).
func (a *Airspace) Prune(cutoff time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for key, b := range a.bsses {
		for mac, s := range b.stations {
			if s.lastSeen.Before(cutoff) {
				delete(b.stations, mac)
			}
		}
		b.deauthTimes = pruneDeauthTimes(b.deauthTimes, cutoff)
		if b.beacons == 0 && len(b.stations) == 0 && len(b.deauthTimes) == 0 &&
			b.lastSeen.Before(cutoff) {
			delete(a.bsses, key)
		}
	}
}

// pruneDeauthTimes drops deauth observations older than cutoff, sliding the
// window forward. It compacts in place and returns nil once the window is empty
// so a long-lived (beaconed) BSS does not accumulate stale timestamps.
func pruneDeauthTimes(times []time.Time, cutoff time.Time) []time.Time {
	kept := times[:0]
	for _, t := range times {
		if !t.Before(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// macKey normalizes a hardware address to a lowercase string key, or "" if nil.
func macKey(mac net.HardwareAddr) string {
	if len(mac) == 0 {
		return ""
	}
	return mac.String()
}

// apKey clusters the BSSIDs of one physical AP (see apGroupMask). Falls back to
// the full BSSID when it cannot be parsed.
func apKey(bssid string) string {
	hw, err := net.ParseMAC(bssid)
	if err != nil || len(hw) == 0 {
		return bssid
	}
	masked := make(net.HardwareAddr, len(hw))
	copy(masked, hw)
	masked[len(masked)-1] &= apGroupMask
	return masked.String()
}

// vendorOUI returns the 24-bit OUI prefix of a BSSID (e.g. "00:11:22"), the
// hook the vendor-mismatch anomaly rule (W4) resolves to a vendor name.
func vendorOUI(bssid string) string {
	hw, err := net.ParseMAC(bssid)
	const ouiOctets = 3
	if err != nil || len(hw) < ouiOctets {
		return ""
	}
	return fmt.Sprintf("%02x:%02x:%02x", hw[0], hw[1], hw[2])
}
