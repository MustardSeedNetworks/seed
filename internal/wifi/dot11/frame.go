// Package dot11 decodes IEEE 802.11 radiotap + management frames into
// structured, pure-data results that the Wi-Fi airspace model and the anomaly
// engine consume (ADR-0012). It is CGO-free — gopacket's radiotap/dot11 layers
// are pure Go; only live capture (internal/capture) needs libpcap. The result
// types carry no behaviour and no I/O, so they are trivially unit-tested off a
// monitor-mode host against captured-frame fixtures.
package dot11

import "net"

// Kind classifies an 802.11 frame at the granularity the airspace model needs
// (build the SSID→AP→BSSID tree from beacons/probe-responses; attribute clients
// from association/data frames; surface deauth/disassoc for anomaly rules).
type Kind uint8

const (
	KindOther Kind = iota
	KindBeacon
	KindProbeRequest
	KindProbeResponse
	KindAssocRequest
	KindAssocResponse
	KindReassocRequest
	KindReassocResponse
	KindAuth
	KindDeauth
	KindDisassoc
	KindAction
	KindData
)

// String renders the frame kind for logs and JSON.
func (k Kind) String() string {
	switch k {
	case KindBeacon:
		return "beacon"
	case KindProbeRequest:
		return "probe-request"
	case KindProbeResponse:
		return "probe-response"
	case KindAssocRequest:
		return "assoc-request"
	case KindAssocResponse:
		return "assoc-response"
	case KindReassocRequest:
		return "reassoc-request"
	case KindReassocResponse:
		return "reassoc-response"
	case KindAuth:
		return "auth"
	case KindDeauth:
		return "deauth"
	case KindDisassoc:
		return "disassoc"
	case KindAction:
		return "action"
	case KindData:
		return "data"
	case KindOther:
		return "other"
	default:
		return "other"
	}
}

// IsManagement reports whether the frame is an 802.11 management frame (the
// class the visibility feature primarily decodes).
func (k Kind) IsManagement() bool {
	switch k {
	case KindBeacon, KindProbeRequest, KindProbeResponse, KindAssocRequest,
		KindAssocResponse, KindReassocRequest, KindReassocResponse, KindAuth,
		KindDeauth, KindDisassoc, KindAction:
		return true
	case KindData, KindOther:
		return false
	default:
		return false
	}
}

// Security is the access-protection suite a BSS advertises, derived from the
// RSN/vendor-WPA information elements and the Privacy capability bit.
type Security uint8

const (
	SecurityUnknown  Security = iota
	SecurityOpen              // no encryption
	SecurityWEP               // Privacy bit set, no RSN/WPA IE — deprecated (802.11-2016)
	SecurityWPA               // vendor WPA1 / TKIP — deprecated (802.11-2012)
	SecurityWPA2              // RSN, CCMP, PSK or 802.1X
	SecurityWPA3              // RSN with SAE / OWE / Suite-B
	SecurityWPA2WPA3          // WPA3 transition mode (both offered)
)

// String renders the security suite (protocol nouns kept verbatim).
func (s Security) String() string {
	switch s {
	case SecurityOpen:
		return "Open"
	case SecurityWEP:
		return "WEP"
	case SecurityWPA:
		return "WPA"
	case SecurityWPA2:
		return "WPA2"
	case SecurityWPA3:
		return "WPA3"
	case SecurityWPA2WPA3:
		return "WPA2/WPA3"
	case SecurityUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// Standard is the highest 802.11 generation a BSS advertises, derived from the
// presence of HT/VHT/HE/EHT information elements. The set runs through Wi-Fi 7
// and is forward-ready for Wi-Fi 8 (802.11bn / UHR, in draft) — adding a
// generation is a table addition, not a rewrite.
type Standard uint8

const (
	StandardUnknown Standard = iota
	Standard80211a
	Standard80211b
	Standard80211g
	Standard80211n  // Wi-Fi 4 — HT
	Standard80211ac // Wi-Fi 5 — VHT
	Standard80211ax // Wi-Fi 6 / 6E — HE
	Standard80211be // Wi-Fi 7 — EHT (320 MHz, MLO, 4K-QAM)
	Standard80211bn // Wi-Fi 8 — UHR (draft); forward-ready
)

// String renders the marketing + spec label (kept verbatim, DNT).
func (s Standard) String() string {
	switch s {
	case Standard80211a:
		return "802.11a"
	case Standard80211b:
		return "802.11b"
	case Standard80211g:
		return "802.11g"
	case Standard80211n:
		return "802.11n (Wi-Fi 4)"
	case Standard80211ac:
		return "802.11ac (Wi-Fi 5)"
	case Standard80211ax:
		return "802.11ax (Wi-Fi 6)"
	case Standard80211be:
		return "802.11be (Wi-Fi 7)"
	case Standard80211bn:
		return "802.11bn (Wi-Fi 8)"
	case StandardUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// Band is the frequency band a channel sits in.
type Band uint8

const (
	BandUnknown Band = iota
	Band24GHz
	Band5GHz
	Band6GHz
)

// String renders the band label.
func (b Band) String() string {
	switch b {
	case Band24GHz:
		return "2.4 GHz"
	case Band5GHz:
		return "5 GHz"
	case Band6GHz:
		return "6 GHz"
	case BandUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// Frame is a decoded 802.11 frame. Radio fields come from the radiotap header;
// the rest from the 802.11 header and (for beacons/probe-responses) the BSS
// information elements. BSS is non-nil only for frames that carry a full BSS
// advertisement (beacon, probe-response).
type Frame struct {
	Kind Kind

	// 802.11 addresses as carried in the header (nil when not present). For
	// management frames Address3 is the BSSID; for data frames the BSSID and
	// station addresses depend on ToDS/FromDS — use BSSIDOf/StationOf rather
	// than reading these directly for data frames.
	BSSID       net.HardwareAddr // Address3
	Transmitter net.HardwareAddr // Address2 — who sent it
	Receiver    net.HardwareAddr // Address1 — intended recipient

	// ToDS/FromDS distinguish the data-frame addressing modes (client→AP,
	// AP→client, IBSS, WDS) so a station attributes to the right BSSID.
	ToDS   bool
	FromDS bool

	// Radio (radiotap) measurements; zero when the header omitted them.
	SignalDBm  int8
	NoiseDBm   int8
	ChannelMHz int
	ChannelNum int
	Band       Band

	Retry bool // the 802.11 Retry bit (feeds the retry-rate anomaly rule)

	// BSS advertisement, present for beacon/probe-response frames.
	BSS *BSS
}

// BSSIDOf returns the BSSID the frame belongs to, accounting for data-frame
// ToDS/FromDS addressing. Returns nil when it cannot be determined (e.g. a
// 4-address WDS frame). For management frames the BSSID is Address3.
func (f *Frame) BSSIDOf() net.HardwareAddr {
	if f.Kind != KindData {
		return f.BSSID // management/control: Address3
	}
	switch {
	case f.ToDS && !f.FromDS: // client → AP: Address1 is the BSSID
		return f.Receiver
	case !f.ToDS && f.FromDS: // AP → client: Address2 is the BSSID
		return f.Transmitter
	case !f.ToDS && !f.FromDS: // IBSS: Address3 is the BSSID
		return f.BSSID
	default: // WDS (4-address): no single BSSID
		return nil
	}
}

// StationOf returns the non-AP station (client) MAC this frame attributes to a
// BSS, or nil if the frame does not identify a client. It covers the common
// passive sources: data frames (by DS direction) and (re)association frames.
func (f *Frame) StationOf() net.HardwareAddr {
	switch f.Kind {
	case KindData:
		switch {
		case f.ToDS && !f.FromDS: // client → AP: Address2 is the client
			return f.Transmitter
		case !f.ToDS && f.FromDS: // AP → client: Address1 is the client
			return f.Receiver
		default:
			return nil
		}
	case KindAssocRequest, KindReassocRequest, KindProbeRequest:
		return f.Transmitter // client transmits to the AP
	case KindAssocResponse, KindReassocResponse:
		return f.Receiver // AP responds to the client
	case KindOther, KindBeacon, KindProbeResponse, KindAuth, KindDeauth,
		KindDisassoc, KindAction:
		return nil
	default:
		return nil
	}
}

// BSS is everything a beacon/probe-response advertises about a basic service
// set: identity, the protection suite, the generation, and the capability flags
// the anomaly rules key off (PMF, RRM/BTM/FT roaming support, country).
type BSS struct {
	SSID     string // empty + Hidden=true for a cloaked SSID
	Hidden   bool
	Security Security
	Standard Standard

	ChannelNum    int
	ChannelWidthM int    // operating width in MHz (20/40/80/160/320), 0 if unknown
	CountryCode   string // 802.11d, "" if absent

	// Capability / roaming-support flags (decoded from IEs).
	PMFRequired  bool // 802.11w — Management Frame Protection required
	PMFCapable   bool
	RRMNeighbor  bool // 802.11k — RM Enabled Capabilities present
	BTMSupported bool // 802.11v — BSS Transition Management (Ext Cap)
	FTSupported  bool // 802.11r — Mobility Domain IE present
	WPSEnabled   bool // vendor WPS IE present

	// BSS Load (802.11e) — station count + channel utilization (0 if absent).
	StationCount    int
	ChannelUtilByte int // raw 0..255 channel-utilization figure from the BSS Load IE
	HasBSSLoad      bool
}
