package dot11

import (
	"bytes"
	"testing"
)

const (
	freq24Chan6  = 2437
	freq5Chan36  = 5180
	freq6Chan37  = 6135
	wpsOUIType   = 0x04
	extCapsBTM   = 0x08 // Extended Capabilities byte 2, bit 3 = bit 19 overall
	vhtCCFS0Ch42 = 42
	vhtCCFS1Ch50 = 50
)

func ies(elements ...[]byte) []byte { return bytes.Join(elements, nil) }

// ext assembles an Element ID Extension element with the given sub-ID.
func ext(extID byte, body ...byte) []byte {
	return ie(ieElementExtension, append([]byte{extID}, body...)...)
}

// htOp is an HT Operation element on primary channel ch with the given
// Information byte 1 (secondary offset in bits 0-1, STA channel width bit 2).
func htOp(ch, info byte) []byte {
	return ie(ieHTOperation, append([]byte{ch, info}, make([]byte, 20)...)...)
}

func vhtOp(code, ccfs0, ccfs1 byte) []byte { return ie(ieVHTOperation, code, ccfs0, ccfs1, 0xfc, 0xff) }

// heOp is an HE Operation element with the given 24-bit parameters and the
// optional fields that follow the fixed part.
func heOp(params uint32, optional ...byte) []byte {
	body := []byte{byte(params), byte(params >> 8), byte(params >> 16), 0x01, 0xfc, 0xff}
	return ext(extHEOperation, append(body, optional...)...)
}

// ehtOp is an EHT Operation element carrying Operation Information with the
// given control byte.
func ehtOp(control byte) []byte {
	return ext(extEHTOperation, ehtInfoPresentBit, 0x11, 0x11, 0x11, 0x11, control, 0x2a, 0x00)
}

func TestDecodeIEsOperatingWidth(t *testing.T) {
	tests := []struct {
		name     string
		freq     int
		elements [][]byte
		standard Standard
		width    int
	}{
		{
			// The regression: HE capability alone used to report 160 MHz.
			name:     "Wi-Fi 6 on 2.4 GHz states 20 MHz",
			freq:     freq24Chan6,
			elements: [][]byte{htCapIE(), htOp(6, 0), ext(extHECapabilities, make([]byte, 20)...), heOp(0)},
			standard: Standard80211ax,
			width:    width20MHz,
		},
		{
			name:     "HT capability without an operation element is 20 MHz",
			freq:     freq24Chan6,
			elements: [][]byte{htCapIE()},
			standard: Standard80211n,
			width:    width20MHz,
		},
		{
			name:     "HT secondary above with 40 MHz allowed",
			freq:     freq5Chan36,
			elements: [][]byte{htCapIE(), htOp(36, htSecondaryAbove|htAnyWidthBit)},
			standard: Standard80211n,
			width:    width40MHz,
		},
		{
			name:     "HT secondary below with 40 MHz allowed",
			freq:     freq5Chan36,
			elements: [][]byte{htOp(40, htSecondaryBelow|htAnyWidthBit)},
			standard: Standard80211n,
			width:    width40MHz,
		},
		{
			name:     "HT secondary named but 20 MHz only",
			freq:     freq24Chan6,
			elements: [][]byte{htOp(6, htSecondaryAbove)},
			standard: Standard80211n,
			width:    width20MHz,
		},
		{
			name:     "VHT 80 MHz",
			freq:     freq5Chan36,
			elements: [][]byte{htOp(36, htSecondaryAbove|htAnyWidthBit), vhtOp(vhtWidth80Plus, vhtCCFS0Ch42, 0)},
			standard: Standard80211ac,
			width:    width80MHz,
		},
		{
			name:     "VHT 160 MHz via CCFS1",
			freq:     freq5Chan36,
			elements: [][]byte{vhtOp(vhtWidth80Plus, vhtCCFS0Ch42, vhtCCFS1Ch50)},
			standard: Standard80211ac,
			width:    width160MHz,
		},
		{
			name:     "VHT deprecated 160 MHz code",
			freq:     freq5Chan36,
			elements: [][]byte{vhtOp(vhtWidth160Old, vhtCCFS1Ch50, 0)},
			standard: Standard80211ac,
			width:    width160MHz,
		},
		{
			name:     "VHT code 0 defers to HT 40 MHz",
			freq:     freq5Chan36,
			elements: [][]byte{htOp(36, htSecondaryAbove|htAnyWidthBit), vhtOp(0, 0, 0)},
			standard: Standard80211ac,
			width:    width40MHz,
		},
		{
			name:     "HE carrying VHT Operation Information",
			freq:     freq5Chan36,
			elements: [][]byte{heOp(1<<heVHTInfoBit, vhtWidth80Plus, vhtCCFS0Ch42, 0)},
			standard: Standard80211ax,
			width:    width80MHz,
		},
		{
			name:     "HE 6 GHz Operation Information 160 MHz",
			freq:     freq6Chan37,
			elements: [][]byte{heOp(1<<he6GHzInfoBit, 37, 3, 39, 47, 0x01)},
			standard: Standard80211ax,
			width:    width160MHz,
		},
		{
			name:     "HE 6 GHz information after a co-hosted indicator",
			freq:     freq6Chan37,
			elements: [][]byte{heOp(1<<heCoHostedBit|1<<he6GHzInfoBit, 0x02, 37, 2, 39, 0, 0x01)},
			standard: Standard80211ax,
			width:    width80MHz,
		},
		{
			name:     "EHT 320 MHz",
			freq:     freq6Chan37,
			elements: [][]byte{heOp(1<<he6GHzInfoBit, 37, 3, 39, 47, 0x01), ehtOp(4)},
			standard: Standard80211be,
			width:    width320MHz,
		},
		{
			name:     "EHT without Operation Information states nothing",
			freq:     freq6Chan37,
			elements: [][]byte{ext(extEHTOperation, 0x00, 0x11, 0x11, 0x11, 0x11)},
			standard: Standard80211be,
			width:    width20MHz,
		},
		{
			name:     "EHT reserved width code is ignored",
			freq:     freq6Chan37,
			elements: [][]byte{ehtOp(7)},
			standard: Standard80211be,
			width:    width20MHz,
		},
		{
			name:     "legacy 802.11g",
			freq:     freq24Chan6,
			elements: [][]byte{ratesOFDM()},
			standard: Standard80211g,
			width:    width20MHz,
		},
		{
			name:     "legacy 802.11a on 5 GHz",
			freq:     freq5Chan36,
			elements: nil,
			standard: Standard80211a,
			width:    width20MHz,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bss := DecodeIEs(ies(tt.elements...), false, tt.freq)
			if bss.Standard != tt.standard {
				t.Errorf("Standard = %v, want %v", bss.Standard, tt.standard)
			}
			if bss.ChannelWidthM != tt.width {
				t.Errorf("ChannelWidthM = %d, want %d", bss.ChannelWidthM, tt.width)
			}
		})
	}
}

func TestDecodeIEsFields(t *testing.T) {
	bssLoad := ie(ieBSSLoad, 0x07, 0x00, 0xa8, 0x00, 0x00) // 7 stations, util 168/255
	elements := ies(
		ie(ieSSID, []byte("corp")...),
		ie(ieCountry, 'U', 'S', ' '),
		bssLoad,
		rsnIE(akmSAE, 1<<mfprBit|1<<mfpcBit),
		ie(ieMobilityDomain, 0x01, 0x02, 0x00),
		ie(ieRMEnabledCaps, 0x73, 0x00, 0x00, 0x00, 0x00),
		ie(ieExtendedCaps, 0x00, 0x00, extCapsBTM),
		ie(ieVendorSpecific, msftWFAOUI0, msftWFAOUI1, msftWFAOUI2, wpsOUIType, 0x10),
	)
	bss := DecodeIEs(elements, true, freq5Chan36)

	if bss.SSID != "corp" || bss.Hidden {
		t.Errorf("SSID = %q hidden=%v, want corp visible", bss.SSID, bss.Hidden)
	}
	if bss.CountryCode != "US" {
		t.Errorf("CountryCode = %q, want US", bss.CountryCode)
	}
	if !bss.HasBSSLoad || bss.StationCount != 7 || bss.ChannelUtilByte != 0xa8 {
		t.Errorf("BSS Load = %v/%d/%d, want true/7/168", bss.HasBSSLoad, bss.StationCount, bss.ChannelUtilByte)
	}
	if bss.Security != SecurityWPA3 || !bss.PMFRequired || !bss.PMFCapable {
		t.Errorf("security = %v pmf=%v/%v, want WPA3 required", bss.Security, bss.PMFRequired, bss.PMFCapable)
	}
	if !bss.FTSupported || !bss.RRMNeighbor || !bss.BTMSupported || !bss.WPSEnabled {
		t.Errorf("FT/RRM/BTM/WPS = %v/%v/%v/%v, want all true",
			bss.FTSupported, bss.RRMNeighbor, bss.BTMSupported, bss.WPSEnabled)
	}
	if bss.ChannelNum != 36 {
		t.Errorf("ChannelNum = %d, want 36 from the frequency (no DS Parameter Set)", bss.ChannelNum)
	}
}

func TestDecodeIEsChannelPrefersDSParameterSet(t *testing.T) {
	// A 2.4 GHz scan entry can be heard on an adjacent channel; the element
	// names the channel the AP operates on.
	if got := DecodeIEs(dsChan6(), false, 2432).ChannelNum; got != 6 {
		t.Errorf("ChannelNum = %d, want 6 from the DS Parameter Set", got)
	}
}

func TestDecodeIEsSecurityFromPrivacyBit(t *testing.T) {
	hidden := ie(ieSSID, 0, 0, 0)
	if bss := DecodeIEs(hidden, true, freq24Chan6); bss.Security != SecurityWEP || !bss.Hidden {
		t.Errorf("privacy without RSN = %v hidden=%v, want WEP hidden", bss.Security, bss.Hidden)
	}
	if bss := DecodeIEs(hidden, false, freq24Chan6); bss.Security != SecurityOpen {
		t.Errorf("no privacy = %v, want Open", bss.Security)
	}
}

func TestDecodeIEsTruncatedKeepsEarlierElements(t *testing.T) {
	elements := ies(ie(ieSSID, []byte("lab")...), ie(ieCountry, 'D', 'E', ' '))
	elements = append(elements, ieRSN, 40, 0x01) // claims 40 bytes, carries 1
	bss := DecodeIEs(elements, false, freq24Chan6)
	if bss.SSID != "lab" || bss.CountryCode != "DE" {
		t.Errorf("SSID/country = %q/%q, want lab/DE", bss.SSID, bss.CountryCode)
	}
	if bss.Security != SecurityOpen {
		t.Errorf("Security = %v, want Open: the truncated RSN element was not read", bss.Security)
	}
}

func FuzzDecodeIEs(f *testing.F) {
	f.Add(ies(htCapIE(), htOp(6, 0x07), heOp(1<<he6GHzInfoBit|1<<heVHTInfoBit|1<<heCoHostedBit, 1, 2, 3, 4, 5)), 2437)
	f.Add(ehtOp(4), 6135)
	f.Fuzz(func(_ *testing.T, data []byte, freq int) {
		_ = DecodeIEs(data, false, freq)
	})
}
