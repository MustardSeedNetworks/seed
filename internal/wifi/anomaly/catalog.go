package wifianomaly

import "github.com/krisarmstrong/seed/internal/anomaly"

// Exported def IDs are the stable catalog keys. Detections reference these, the
// API/UI key off them, and tests assert against them — so they live as exported
// constants rather than string literals scattered through the rules.
const (
	DefOpenNetwork             = "wifi-open-network"
	DefWEPInUse                = "wifi-wep-in-use"
	DefWPSEnabled              = "wifi-wps-enabled"
	DefPMFNotRequired          = "wifi-pmf-not-required"
	DefSecurityMismatch        = "wifi-security-mismatch"
	DefEvilTwin                = "wifi-evil-twin"
	DefCoChannelContention     = "wifi-co-channel-contention"
	DefAdjacentChannelOverlap  = "wifi-adjacent-channel-overlap"
	DefHiddenSSID              = "wifi-hidden-ssid"
	DefCountryConflict         = "wifi-country-conflict"
	DefStandardMismatch        = "wifi-standard-mismatch"
	DefWPA3TransitionDowngrade = "wifi-wpa3-transition-downgrade"
	DefDefaultSSIDName         = "wifi-default-ssid-name"
	DefSSIDSprawl              = "wifi-ssid-sprawl"
	DefInconsistentRoaming     = "wifi-inconsistent-roaming"
	DefRegulatoryViolation     = "wifi-regulatory-violation"
	DefBSSLoadSaturation       = "wifi-bss-load-saturation"
	DefWideChannel24GHz        = "wifi-wide-channel-2ghz"
	DefChannelWidthMismatch    = "wifi-channel-width-mismatch"
	DefDeauthFlood             = "wifi-deauth-flood"
)

// CapActiveTest names the platform capability required to run an active
// follow-up that transmits frames (e.g. a controlled deauthentication probe to
// confirm a PMF gap). Where the capture adapter does not register it — most
// best-effort/monitor-only setups — the engine degrades the follow-up to a
// guided manual prompt (ADR-0011).
const CapActiveTest = "wifi_active_test"

// Catalog builds and validates the Wi-Fi anomaly catalog. It fails fast if a
// definition is malformed, so a typo in the data below cannot ship a blank card.
func Catalog() (*anomaly.Catalog, error) {
	return anomaly.NewCatalog(Defs()...)
}

// Defs returns the Wi-Fi anomaly definitions. The copy is authored originally
// with 802.11 citations; severities are the catalog defaults (the engine may
// escalate on recurrence, and a rule may override per detection).
func Defs() []anomaly.Def {
	return []anomaly.Def{
		{
			ID:              DefOpenNetwork,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11-2020 §12"},
			Title:           "Open network (no encryption)",
			Description: "The BSS advertises no robust security network (no RSN information " +
				"element) and no privacy, so all traffic is transmitted in the clear and any " +
				"device in range can join and read it.",
			Impact: "Anyone within radio range can capture user traffic and associate without " +
				"credentials. Acceptable only for a deliberately public/guest segment that is " +
				"isolated from internal resources.",
			Recommendation: "If this network is not intentionally public, enable WPA2-PSK at " +
				"minimum and prefer WPA3-SAE. If it is a guest network, confirm it is firewalled " +
				"off the internal LAN and consider OWE (Enhanced Open) for opportunistic encryption.",
			FollowUps: []anomaly.FollowUp{{
				Kind:   anomaly.FollowUpPrompt,
				Label:  "Confirm intent",
				Action: "Verify whether this open SSID is an intentional, isolated guest network.",
			}},
		},
		{
			ID:              DefWEPInUse,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityCritical,
			Standards:       []string{"IEEE 802.11-2016 (WEP deprecated)"},
			Title:           "WEP encryption in use",
			Description: "The BSS sets the Privacy bit but advertises no RSN/WPA element, " +
				"indicating Wired Equivalent Privacy. WEP's RC4 keystream and weak IV scheme " +
				"are broken; its use was deprecated by 802.11 and the key is recoverable in minutes.",
			Impact: "An attacker can passively recover the WEP key and decrypt all traffic, then " +
				"join the network as a trusted device. WEP provides effectively no protection.",
			Recommendation: "Retire WEP immediately. Reconfigure the AP for WPA2-PSK/802.1X or " +
				"WPA3 and re-provision clients. If the hardware cannot do WPA2, replace it.",
		},
		{
			ID:              DefWPSEnabled,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"Wi-Fi Simple Configuration (WPS)"},
			Title:           "WPS enabled",
			Description: "The BSS advertises Wi-Fi Protected Setup. The WPS external-registrar PIN " +
				"is vulnerable to online brute force and to offline Pixie-Dust recovery of weak " +
				"nonces, either of which yields the PSK regardless of its strength.",
			Impact: "An attacker can recover the WPA passphrase by attacking WPS rather than the " +
				"passphrase itself, bypassing an otherwise strong key.",
			Recommendation: "Disable WPS on the AP. Provision clients with the passphrase or " +
				"802.1X instead of the WPS PIN/push-button flow.",
		},
		{
			ID:              DefPMFNotRequired,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11w-2009", "IEEE 802.11-2020 §12.2.7"},
			Title:           "Protected Management Frames not required",
			Description: "The BSS runs an RSN security suite but does not require Protected " +
				"Management Frames (802.11w / MFPR=0), so deauthentication and disassociation " +
				"management frames remain unauthenticated.",
			Impact: "Forged deauth/disassoc frames can knock clients off the network, enabling " +
				"denial of service and the disconnect step that evil-twin and handshake-capture " +
				"attacks rely on.",
			Recommendation: "Require PMF (MFPR=1) on the SSID. WPA3 mandates it; for WPA2 enable " +
				"the 802.11w 'required' (not merely 'capable') setting once clients support it.",
			FollowUps: []anomaly.FollowUp{{
				Kind:       anomaly.FollowUpAuto,
				Label:      "Deauth-response test",
				Action:     "Transmit a spoofed deauthentication and observe whether associated clients drop.",
				Capability: CapActiveTest,
			}},
		},
		{
			ID:              DefSecurityMismatch,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11i", "IEEE 802.11-2020 §12"},
			Title:           "Inconsistent security across an SSID",
			Description: "BSSes advertising the same SSID offer materially different security " +
				"suites — at least one strong (WPA2/WPA3) and at least one weak (Open/WEP/WPA). " +
				"A client may silently associate to the weakest radio, and the disparity can " +
				"signal a rogue impersonator advertising a downgraded variant.",
			Impact: "Users believe they are on the protected network while their device joins the " +
				"weak BSS, exposing traffic and credentials to interception or to an attacker's AP.",
			Recommendation: "Make the security configuration identical across every AP serving the " +
				"SSID. If a weak BSS is not yours, treat it as a rogue/evil-twin and locate it.",
			FollowUps: []anomaly.FollowUp{{
				Kind:   anomaly.FollowUpPrompt,
				Label:  "Audit AP configs",
				Action: "Compare the security suite on every sanctioned AP advertising this SSID.",
			}},
		},
		{
			ID:              DefEvilTwin,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11 (BSSID/SSID semantics)"},
			Title:           "Possible evil twin (vendor mismatch)",
			Description: "One SSID is served by access points whose BSSIDs resolve to different " +
				"hardware vendors (OUIs). A homogeneous enterprise WLAN is normally single-vendor; " +
				"an outlier vendor advertising the same network name is a classic evil-twin / " +
				"honeypot signature.",
			Impact: "Clients may associate to an attacker-controlled AP impersonating the trusted " +
				"SSID, exposing them to interception, captive-portal credential theft, and " +
				"on-path attacks.",
			Recommendation: "Verify every BSSID/vendor advertising this SSID against your sanctioned " +
				"AP inventory. Investigate and physically locate any AP you do not recognize.",
			FollowUps: []anomaly.FollowUp{{
				Kind:   anomaly.FollowUpPrompt,
				Label:  "Verify inventory",
				Action: "Cross-check the BSSIDs and vendors for this SSID against the AP allowlist.",
			}},
		},
		{
			ID:              DefCoChannelContention,
			Category:        anomaly.CategoryRF,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11 (CSMA/CA, DCF)"},
			Title:           "Co-channel contention",
			Description: "Several BSSes share one channel in the same band. Because 802.11 uses " +
				"CSMA/CA, every radio on a channel contends for the same airtime and must defer " +
				"to the others, so capacity is divided rather than added.",
			Impact: "Throughput drops and latency/jitter rise for all networks on the channel as " +
				"the count of co-channel radios grows — the dominant cause of poor Wi-Fi in dense " +
				"deployments.",
			Recommendation: "Re-plan channel assignments to spread APs across non-overlapping " +
				"channels, reduce transmit power to shrink cells, or move capacity to 5/6 GHz.",
		},
		{
			ID:              DefAdjacentChannelOverlap,
			Category:        anomaly.CategoryRF,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"IEEE 802.11 (2.4 GHz channel plan)"},
			Title:           "Adjacent-channel overlap (2.4 GHz)",
			Description: "A 2.4 GHz BSS operates on a channel other than 1, 6, or 11. In the " +
				"2.4 GHz band a 20 MHz channel overlaps its neighbours, so any channel off the " +
				"1/6/11 plan partially overlaps two non-overlapping channels at once.",
			Impact: "Overlapping energy is treated as noise rather than decodable traffic, raising " +
				"the noise floor and retransmissions for every nearby network — worse than clean " +
				"co-channel sharing.",
			Recommendation: "Move 2.4 GHz radios onto channels 1, 6, or 11 only, and avoid 40 MHz " +
				"width in 2.4 GHz.",
		},
		{
			ID:              DefHiddenSSID,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"IEEE 802.11 (SSID element)"},
			Title:           "Hidden (cloaked) SSID",
			Description: "The BSS beacons with a null/zero-length SSID. SSID cloaking is security " +
				"through obscurity: the network name is still revealed in probe and association " +
				"frames whenever a client connects, and it is trivially recovered by passive " +
				"observation.",
			Impact: "Cloaking provides no real protection while it forces clients to actively probe " +
				"for the network — which leaks the SSID from the client side and can degrade " +
				"battery life and roaming.",
			Recommendation: "Do not rely on hiding the SSID for security. Broadcast the SSID and " +
				"protect the network with WPA2/WPA3 and PMF instead.",
		},
		{
			ID:              DefCountryConflict,
			Category:        anomaly.CategoryStandards,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11d", "IEEE 802.11-2020 (Country element)"},
			Title:           "Conflicting regulatory domain",
			Description: "Access points in the same airspace advertise different 802.11d country " +
				"codes. Co-located APs should agree on the regulatory domain; a disagreement " +
				"indicates a misconfigured AP or one spoofing a different domain to use channels " +
				"or power levels that are not permitted locally.",
			Impact: "Clients honouring the wrong country element may use illegal channels or " +
				"transmit power, causing interference and potential regulatory violation; a " +
				"spoofed domain can also be a rogue-AP indicator.",
			Recommendation: "Set the correct, identical country/regulatory domain on every AP. " +
				"Investigate any AP advertising an unexpected country code.",
		},
		{
			ID:              DefStandardMismatch,
			Category:        anomaly.CategoryStandards,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"IEEE 802.11n/ac/ax/be"},
			Title:           "Mixed 802.11 generations across an SSID",
			Description: "BSSes advertising the same SSID support different 802.11 generations " +
				"(e.g. one 802.11n radio alongside an 802.11ax radio). Clients that land on the " +
				"older radio get lower rates and fewer efficiency features (OFDMA, MU-MIMO).",
			Impact: "Inconsistent performance and roaming across the WLAN; the slowest radios can " +
				"also pull down airtime efficiency for the whole cell via protection mechanisms.",
			Recommendation: "Standardise AP hardware/firmware where practical, or confirm the " +
				"mixed generations are intentional and the older radios are scoped to legacy " +
				"clients only.",
		},
		{
			ID:              DefWPA3TransitionDowngrade,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"WPA3 Specification (transition mode)", "IEEE 802.11-2020 §12"},
			Title:           "WPA3 transition mode (downgrade-capable)",
			Description: "The BSS runs WPA3/WPA2 transition mode, advertising both SAE (WPA3) and " +
				"PSK (WPA2) so legacy clients can still join. An on-path attacker can strip the " +
				"WPA3 offer and force a WPA2 association, defeating SAE's offline-dictionary " +
				"resistance for that client.",
			Impact: "Clients capable of WPA3 may be downgraded to WPA2, exposing the passphrase to " +
				"offline cracking that pure WPA3-SAE would prevent.",
			Recommendation: "Once all clients support WPA3, move the SSID to WPA3-only (SAE). Keep " +
				"transition mode only as long as WPA2-only clients genuinely remain.",
		},
		{
			ID:              DefDefaultSSIDName,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"IEEE 802.11 (SSID element)"},
			Title:           "Default/manufacturer SSID name",
			Description: "The SSID matches a well-known router/manufacturer default (e.g. " +
				"\"linksys\", \"NETGEAR\", \"dlink\"). A default name signals an unconfigured or " +
				"lightly-managed device and often correlates with default credentials and a " +
				"vendor-derivable default passphrase.",
			Impact: "Default-named networks advertise the hardware vendor to an attacker and are " +
				"more likely to retain default management credentials and weak factory keys.",
			Recommendation: "Rename the SSID to a non-identifying value and confirm the device's " +
				"admin credentials and Wi-Fi passphrase have been changed from the factory defaults.",
		},
		{
			ID:              DefSSIDSprawl,
			Category:        anomaly.CategoryCapacity,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"IEEE 802.11 (beacon/airtime)"},
			Title:           "SSID sprawl on one AP",
			Description: "A single access point advertises many SSIDs. Every SSID needs its own " +
				"beacon at the beacon interval on every radio, so each extra SSID consumes fixed " +
				"management airtime whether or not any client uses it.",
			Impact: "Excess beacons reduce the airtime available for data, most noticeably in the " +
				"2.4 GHz band and in dense deployments, degrading throughput for every nearby cell.",
			Recommendation: "Consolidate SSIDs — use one or two with role-based VLANs/RADIUS rather " +
				"than a separate SSID per purpose, and disable unused SSIDs.",
		},
		{
			ID:              DefInconsistentRoaming,
			Category:        anomaly.CategoryRoaming,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11r", "IEEE 802.11k", "IEEE 802.11v"},
			Title:           "Inconsistent roaming support across an SSID",
			Description: "BSSes advertising the same SSID disagree on roaming assistance: some " +
				"advertise 802.11r (fast transition), 802.11k (neighbor reports), or 802.11v (BSS " +
				"transition management) while others do not. Clients roam best when every AP in the " +
				"WLAN offers the same assistance.",
			Impact: "Clients moving between a supporting and a non-supporting AP get slow or failed " +
				"roams, dropped real-time sessions, and sticky-client behaviour.",
			Recommendation: "Enable the same roaming features (802.11r/k/v) consistently on every AP " +
				"serving the SSID, or disable them uniformly if a legacy client cannot cope.",
		},
		{
			ID:              DefRegulatoryViolation,
			Category:        anomaly.CategoryStandards,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11d", "ITU/regional 2.4 GHz channel plans"},
			Title:           "Channel not permitted in the declared regulatory domain",
			Description: "The BSS operates on a 2.4 GHz channel that is not allowed in the country " +
				"it advertises (802.11d). Channels 12-13 are not permitted under the US/Canada FCC " +
				"domain, and channel 14 is permitted only in Japan; using them elsewhere violates " +
				"the local regulatory plan.",
			Impact: "Operating outside the permitted channel set can cause interference with " +
				"licensed services and other networks, and is a regulatory-compliance violation; it " +
				"can also indicate a misconfigured or spoofed regulatory domain.",
			Recommendation: "Move the radio to a channel permitted in the deployment's country " +
				"(1-11 under FCC) and set the correct regulatory domain on the AP.",
		},
		{
			ID:              DefBSSLoadSaturation,
			Category:        anomaly.CategoryCapacity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11e (BSS Load element)"},
			Title:           "BSS Load channel saturation",
			Description: "The BSS advertises a high channel-utilization figure in its 802.11e BSS " +
				"Load element. Channel utilization measures how much airtime is already busy; a " +
				"sustained high value means little airtime is left for new traffic.",
			Impact: "Clients on a saturated channel see low throughput and high latency/jitter " +
				"regardless of signal strength — the channel itself is the bottleneck.",
			Recommendation: "Reduce the load on the channel: move clients or APs to a less-utilized " +
				"channel or band, add capacity, or lower cell size so fewer clients share airtime.",
		},
		{
			ID:              DefWideChannel24GHz,
			Category:        anomaly.CategoryRF,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"IEEE 802.11n (2.4 GHz 40 MHz)"},
			Title:           "Wide channel in 2.4 GHz",
			Description: "The BSS uses a 40 MHz (or wider) channel in the 2.4 GHz band. With only " +
				"three non-overlapping 20 MHz channels (1/6/11), a 40 MHz channel occupies most of " +
				"the band and necessarily overlaps neighbouring networks.",
			Impact: "Wide 2.4 GHz channels raise the noise floor for every nearby network and are " +
				"frequently forced back to 20 MHz by coexistence rules — net throughput often drops.",
			Recommendation: "Use 20 MHz channels in 2.4 GHz; reserve 40/80/160 MHz widths for 5 and " +
				"6 GHz where the spectrum exists.",
		},
		{
			ID:              DefChannelWidthMismatch,
			Category:        anomaly.CategoryRF,
			DefaultSeverity: anomaly.SeverityInfo,
			Standards:       []string{"IEEE 802.11n/ac/ax (channel width)"},
			Title:           "Inconsistent channel width across an SSID",
			Description: "BSSes advertising the same SSID operate at different channel widths. " +
				"Mixed widths across an ESS can cause uneven throughput and complicate channel " +
				"planning as clients roam between radios.",
			Impact: "Clients get inconsistent peak rates depending on which radio they land on, and " +
				"wider radios may overlap channels the planning assumed were clear.",
			Recommendation: "Standardise the channel width across APs serving the SSID per band, or " +
				"confirm the mix is intentional for capacity tiering.",
		},
		{
			ID:              DefDeauthFlood,
			Category:        anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11-2020 §9.3.3.3", "IEEE 802.11w-2009"},
			Title:           "Deauthentication/disassociation flood",
			Description: "An unusually high number of deauthentication or disassociation management " +
				"frames was attributed to this BSSID within a short window. On a BSS that does not " +
				"require Protected Management Frames these frames are unauthenticated, so a burst is a " +
				"classic management-frame denial-of-service — and the forced-disconnect step that " +
				"evil-twin and handshake-capture attacks rely on.",
			Impact: "Clients are repeatedly knocked off the network, causing dropped sessions and roam " +
				"churn, and the disconnects create the window an attacker uses to capture handshakes or " +
				"lure clients onto a rogue AP.",
			Recommendation: "Require Protected Management Frames (802.11w) on the SSID to authenticate " +
				"deauth/disassoc frames, and investigate the source — a misbehaving client/AP or an " +
				"active attacker — using the BSSID and channel.",
			FollowUps: []anomaly.FollowUp{{
				Kind:  anomaly.FollowUpPrompt,
				Label: "Locate the deauth source",
				Action: "Watch the affected channel and correlate the deauth burst with a transmitter; " +
					"confirm whether the BSS requires PMF.",
			}},
		},
	}
}
