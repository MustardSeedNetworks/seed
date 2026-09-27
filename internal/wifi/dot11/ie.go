package dot11

import "encoding/binary"

// ie.go parses 802.11 information elements (the tagged fields in beacon and
// probe-response bodies) into the BSS view. Every parser is bounds-checked —
// real-world IEs are frequently truncated or malformed, and a decoder fed
// adversarial frames must never panic.

// Information-element IDs (IEEE 802.11-2020 Table 9-92, plus the Element ID
// Extension space for HE/EHT). Only the ones we decode are named.
const (
	ieSSID             = 0
	ieSupportedRates   = 1
	ieDSParameterSet   = 3
	ieCountry          = 7
	ieBSSLoad          = 11
	ieHTCapabilities   = 45
	ieRSN              = 48
	ieExtSupportRates  = 50
	ieMobilityDomain   = 54 // 802.11r — Fast BSS Transition
	ieHTOperation      = 61
	ieRMEnabledCaps    = 70 // 802.11k — Radio Resource Measurement
	ieExtendedCaps     = 127
	ieVHTCapabilities  = 191
	ieVHTOperation     = 192
	ieVendorSpecific   = 221
	ieElementExtension = 255 // Element ID Extension follows in the first Info byte
)

// Element ID Extension sub-IDs (within IE 255) for the modern PHYs. Wi-Fi 8
// (802.11bn / UHR) will add new sub-IDs here — extend this list, no rewrite.
const (
	extHECapabilities  = 35 // 802.11ax — Wi-Fi 6
	extHEOperation     = 36
	extEHTOperation    = 106
	extEHTCapabilities = 108 // 802.11be — Wi-Fi 7
)

// RSN AKM suite selectors (suite OUI 00-0F-AC, last byte = type).
const (
	akmDot1X     = 1
	akmPSK       = 2
	akmFTDot1X   = 3
	akmFTPSK     = 4
	akmSAE       = 8 // WPA3-Personal
	akmFTSAE     = 9
	akmSuiteB    = 11
	akmSuiteB192 = 12
	akmOWE       = 18 // Opportunistic Wireless Encryption (WPA3 "enhanced open")
)

// Field sizes, offsets, and bit positions used by the IE parsers.
const (
	countryCodeLen   = 2 // 802.11d country string prefix
	bssLoadMinLen    = 3 // station count (2) + channel-util byte (1)
	rsnVersionLen    = 2
	cipherSuiteLen   = 4 // a cipher / AKM suite selector is OUI(3) + type(1)
	countFieldLen    = 2 // the uint16 "count" fields in the RSN IE
	rsnCapsLen       = 2
	selectorTypeOff  = 3  // type byte within a 4-byte suite selector
	vendorMinLen     = 4  // OUI(3) + vendor type(1)
	mfprBit          = 6  // RSN capabilities: Management Frame Protection Required
	mfpcBit          = 7  // RSN capabilities: Management Frame Protection Capable
	btmBit           = 19 // Extended Capabilities: BSS Transition (802.11v)
	bitsPerByte      = 8
	rateBasicMask    = 0x7f // strips the "basic rate" high bit
	maxDSSSRateUnits = 22   // 11 Mbps in 500 kbps units; above this implies OFDM
	wpaVendorType    = 1    // 00-50-F2 type 1 = WPA1
	wpsVendorType    = 4    // 00-50-F2 type 4 = WPS
)

// The Microsoft / Wi-Fi-Alliance OUI (00-50-F2) prefixes the WPA1 and WPS
// vendor-specific elements.
const (
	msftWFAOUI0 = 0x00
	msftWFAOUI1 = 0x50
	msftWFAOUI2 = 0xF2
)

// Operating channel widths in MHz.
const (
	width20MHz  = 20
	width40MHz  = 40
	width80MHz  = 80
	width160MHz = 160
	width320MHz = 320
)

// Operation-element layouts (IEEE 802.11-2020 9.4.2.56, 9.4.2.158, 9.4.2.249;
// 802.11be 9.4.2.311). Each names the field that carries the operating width.
const (
	htOpMinLen         = 2    // primary channel(1) + HT Operation Information byte 1
	htSecondaryMask    = 0x03 // byte 1 bits 0-1: secondary channel offset
	htSecondaryAbove   = 1
	htSecondaryBelow   = 3
	htAnyWidthBit      = 0x04 // byte 1 bit 2: STA Channel Width (40 MHz allowed)
	vhtOpInfoLen       = 3    // channel width(1) + CCFS0(1) + CCFS1(1)
	vhtWidth80Plus     = 1    // 80 MHz, or 160 / 80+80 when CCFS1 is set
	vhtWidth160Old     = 2    // deprecated 160 MHz encoding
	vhtWidth8080Old    = 3    // deprecated 80+80 MHz encoding
	heOpFixedLen       = 6    // parameters(3) + BSS color(1) + basic HE-MCS(2)
	heVHTInfoBit       = 14   // parameters: VHT Operation Information present
	heCoHostedBit      = 15   // parameters: Max Co-Hosted BSSID Indicator present
	he6GHzInfoBit      = 17   // parameters: 6 GHz Operation Information present
	he6GHzControlOff   = 1    // control byte within the 6 GHz Operation Information
	he6GHzWidthMask    = 0x03
	ehtOpFixedLen      = 5 // parameters(1) + basic EHT-MCS and NSS set(4)
	ehtInfoPresentBit  = 0x01
	ehtWidthMask       = 0x07
	shiftBitsPerOctet2 = 16
)

// Width codes shared by the HE 6 GHz and EHT Operation Information fields.
const (
	widthCode20 = iota
	widthCode40
	widthCode80
	widthCode160
	widthCode320
)

// widthForCode maps the width codes of the HE 6 GHz and EHT Operation
// Information control fields, which both count 20, 40, 80, 160, 320 MHz from 0.
// A reserved code states nothing.
func widthForCode(code byte) int {
	switch code {
	case widthCode20:
		return width20MHz
	case widthCode40:
		return width40MHz
	case widthCode80:
		return width80MHz
	case widthCode160:
		return width160MHz
	case widthCode320:
		return width320MHz
	default:
		return 0
	}
}

// rawIE is one parsed tag from the IE list: its ID, the extension sub-ID (only
// meaningful when ID == 255), and the element body.
type rawIE struct {
	id    uint8
	extID uint8
	body  []byte
}

// phyPresence accumulates which PHY generations a BSS advertised so the highest
// is chosen once every IE has been folded in.
type phyPresence struct {
	hasHT, hasVHT, hasHE, hasEHT bool
	hasOFDMRate                  bool
	// opWidth is the widest operating width an Operation element states.
	opWidth int
}

// applyIE folds an identity/security/capability information element into the
// BSS. PHY-generation elements are handled separately by markPHY (keeping each
// function's branch count low).
func applyIE(bss *BSS, ie rawIE) {
	switch ie.id {
	case ieSSID:
		// A zero-length or all-NUL SSID IE is a cloaked (hidden) network.
		if len(ie.body) == 0 || allZero(ie.body) {
			bss.Hidden = true
		} else {
			bss.SSID = string(ie.body)
		}
	case ieDSParameterSet:
		if len(ie.body) >= 1 {
			bss.ChannelNum = int(ie.body[0])
		}
	case ieCountry:
		if len(ie.body) >= countryCodeLen {
			bss.CountryCode = string(ie.body[:countryCodeLen])
		}
	case ieBSSLoad:
		applyBSSLoad(bss, ie.body)
	case ieRSN:
		applyRSN(bss, ie.body)
	case ieMobilityDomain:
		bss.FTSupported = true
	case ieRMEnabledCaps:
		bss.RRMNeighbor = true
	case ieExtendedCaps:
		if btmSupported(ie.body) {
			bss.BTMSupported = true
		}
	case ieVendorSpecific:
		applyVendor(bss, ie.body)
	}
}

// markPHY records the presence of a PHY-generation element (or OFDM rates) so
// the highest advertised standard can be chosen in finalize, and the operating
// width each Operation element states.
func markPHY(phy *phyPresence, ie rawIE) {
	switch ie.id {
	case ieSupportedRates, ieExtSupportRates:
		phy.hasOFDMRate = phy.hasOFDMRate || hasOFDMRate(ie.body)
	case ieHTCapabilities:
		phy.hasHT = true
	case ieHTOperation:
		phy.hasHT = true
		phy.opWidth = maxWidth(phy.opWidth, htOperationWidth(ie.body))
	case ieVHTCapabilities:
		phy.hasVHT = true
	case ieVHTOperation:
		phy.hasVHT = true
		phy.opWidth = maxWidth(phy.opWidth, vhtOperationWidth(ie.body))
	case ieElementExtension:
		markExtension(phy, ie)
	}
}

func markExtension(phy *phyPresence, ie rawIE) {
	switch ie.extID {
	case extHECapabilities:
		phy.hasHE = true
	case extHEOperation:
		phy.hasHE = true
		phy.opWidth = maxWidth(phy.opWidth, heOperationWidth(ie.body))
	case extEHTCapabilities:
		phy.hasEHT = true
	case extEHTOperation:
		phy.hasEHT = true
		phy.opWidth = maxWidth(phy.opWidth, ehtOperationWidth(ie.body))
	}
}

// finalize sets the BSS standard from the PHY flags, the band and the rates,
// newest first (EHT→HE→VHT→HT→legacy), and the operating width from the
// Operation elements. A generation's capability ceiling is not its operating
// width: a Wi-Fi 6 AP on 2.4 GHz runs 20 MHz, so the width comes only from what
// the AP states it operates at, and 20 MHz (the primary channel every BSS
// occupies) when it states nothing wider.
func (p phyPresence) finalize(bss *BSS, band Band) {
	switch {
	case p.hasEHT:
		bss.Standard = Standard80211be
	case p.hasHE:
		bss.Standard = Standard80211ax
	case p.hasVHT:
		bss.Standard = Standard80211ac
	case p.hasHT:
		bss.Standard = Standard80211n
	case band == Band5GHz:
		bss.Standard = Standard80211a
	case p.hasOFDMRate:
		bss.Standard = Standard80211g
	default:
		bss.Standard = Standard80211b
	}
	bss.ChannelWidthM = maxWidth(width20MHz, p.opWidth)
}

// htOperationWidth is 40 when the HT Operation element names a secondary
// channel and allows a 40 MHz STA channel width, else 20.
func htOperationWidth(body []byte) int {
	if len(body) < htOpMinLen {
		return 0
	}
	info := body[1]
	secondary := info & htSecondaryMask
	if (secondary == htSecondaryAbove || secondary == htSecondaryBelow) && info&htAnyWidthBit != 0 {
		return width40MHz
	}
	return width20MHz
}

// vhtOperationWidth reads the VHT Operation Information field, which leads the
// VHT Operation element.
func vhtOperationWidth(body []byte) int {
	if len(body) < vhtOpInfoLen {
		return 0
	}
	return vhtChannelWidth(body[0], body[2])
}

// vhtChannelWidth decodes a VHT Operation Information width. Code 0 defers to
// the HT Operation element (20 or 40 MHz), so it contributes nothing here. With
// code 1 a non-zero CCFS1 means 160 or 80+80 MHz (802.11-2020 Table 9-274);
// both occupy 160 MHz of spectrum.
func vhtChannelWidth(code, ccfs1 byte) int {
	switch code {
	case vhtWidth80Plus:
		if ccfs1 != 0 {
			return width160MHz
		}
		return width80MHz
	case vhtWidth160Old, vhtWidth8080Old:
		return width160MHz
	default:
		return 0
	}
}

// heOperationWidth reads the width from the optional fields of the HE
// Operation element: the VHT Operation Information a 5 GHz HE AP may carry, and
// the 6 GHz Operation Information a 6 GHz AP carries instead of HT/VHT
// Operation. On 2.4 and most 5 GHz APs neither is present and the HT/VHT
// Operation elements state the width.
func heOperationWidth(body []byte) int {
	if len(body) < heOpFixedLen {
		return 0
	}
	params := uint32(body[0]) | uint32(body[1])<<bitsPerByte | uint32(body[2])<<shiftBitsPerOctet2
	off := heOpFixedLen
	width := 0
	if params&(1<<heVHTInfoBit) != 0 {
		if len(body) < off+vhtOpInfoLen {
			return 0
		}
		width = vhtChannelWidth(body[off], body[off+2])
		off += vhtOpInfoLen
	}
	if params&(1<<heCoHostedBit) != 0 {
		off++
	}
	if params&(1<<he6GHzInfoBit) != 0 && len(body) > off+he6GHzControlOff {
		width = maxWidth(width, widthForCode(body[off+he6GHzControlOff]&he6GHzWidthMask))
	}
	return width
}

// ehtOperationWidth reads the EHT Operation Information control field, present
// when the parameters say so; it is the only element that can state 320 MHz.
func ehtOperationWidth(body []byte) int {
	if len(body) <= ehtOpFixedLen || body[0]&ehtInfoPresentBit == 0 {
		return 0
	}
	return widthForCode(body[ehtOpFixedLen] & ehtWidthMask)
}

// applyBSSLoad parses the 802.11e BSS Load IE: station count (uint16 LE) +
// channel-utilization byte + available admission capacity (uint16, ignored).
func applyBSSLoad(bss *BSS, body []byte) {
	if len(body) < bssLoadMinLen {
		return
	}
	bss.StationCount = int(binary.LittleEndian.Uint16(body[0:countFieldLen]))
	bss.ChannelUtilByte = int(body[countFieldLen])
	bss.HasBSSLoad = true
}

// applyRSN parses the RSN IE (802.11i / RSN) to classify WPA2 vs WPA3 (vs the
// transition mode that offers both) and the PMF (802.11w) capability/required
// bits. Layout: version(2) | group cipher(4) | pairwise count(2)+suites |
// AKM count(2)+suites | RSN capabilities(2) | ... (optional fields ignored).
func applyRSN(bss *BSS, body []byte) {
	off := rsnVersionLen + cipherSuiteLen // skip version + group cipher suite
	if len(body) < off+countFieldLen {
		bss.Security = highestSecurity(bss.Security, SecurityWPA2)
		return
	}
	pairwiseCount := int(binary.LittleEndian.Uint16(body[off : off+countFieldLen]))
	off += countFieldLen + cipherSuiteLen*pairwiseCount // skip pairwise suites
	if len(body) < off+countFieldLen {
		bss.Security = highestSecurity(bss.Security, SecurityWPA2)
		return
	}
	akmCount := int(binary.LittleEndian.Uint16(body[off : off+countFieldLen]))
	off += countFieldLen

	sae, psk, owe := false, false, false
	for range akmCount {
		if len(body) < off+cipherSuiteLen {
			break
		}
		switch body[off+selectorTypeOff] { // suite type (last byte of the selector)
		case akmSAE, akmFTSAE:
			sae = true
		case akmPSK, akmFTPSK, akmDot1X, akmFTDot1X:
			psk = true
		case akmOWE, akmSuiteB, akmSuiteB192:
			owe = true
		}
		off += cipherSuiteLen
	}
	switch {
	case sae && psk:
		bss.Security = highestSecurity(bss.Security, SecurityWPA2WPA3)
	case sae || owe:
		bss.Security = highestSecurity(bss.Security, SecurityWPA3)
	default:
		bss.Security = highestSecurity(bss.Security, SecurityWPA2)
	}
	if len(body) >= off+rsnCapsLen {
		caps := binary.LittleEndian.Uint16(body[off : off+rsnCapsLen])
		bss.PMFRequired = caps&(1<<mfprBit) != 0
		bss.PMFCapable = caps&(1<<mfpcBit) != 0
	}
}

// applyVendor handles the vendor-specific IEs we care about: legacy WPA1 (TKIP)
// and WPS presence. Both use the Microsoft/Wi-Fi-Alliance OUI 00-50-F2.
func applyVendor(bss *BSS, body []byte) {
	if len(body) < vendorMinLen {
		return
	}
	isMSOUI := body[0] == msftWFAOUI0 && body[1] == msftWFAOUI1 && body[2] == msftWFAOUI2
	if !isMSOUI {
		return
	}
	switch body[selectorTypeOff] {
	case wpaVendorType:
		bss.Security = highestSecurity(bss.Security, SecurityWPA)
	case wpsVendorType:
		bss.WPSEnabled = true
	}
}

// btmSupported reads the BSS Transition (802.11v) bit of the Extended
// Capabilities IE.
func btmSupported(body []byte) bool {
	byteIdx := btmBit / bitsPerByte
	if len(body) <= byteIdx {
		return false
	}
	return body[byteIdx]&(1<<(btmBit%bitsPerByte)) != 0
}

// hasOFDMRate reports whether a (extended) supported-rates IE lists any OFDM
// rate (>11 Mbps), which distinguishes 802.11g/a from 802.11b-only.
func hasOFDMRate(body []byte) bool {
	for _, r := range body {
		if r&rateBasicMask > maxDSSSRateUnits {
			return true
		}
	}
	return false
}

// highestSecurity keeps the strongest suite seen (an AP may carry both a vendor
// WPA1 IE and an RSN IE; RSN wins).
func highestSecurity(a, b Security) Security {
	if b > a {
		return b
	}
	return a
}

func maxWidth(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
