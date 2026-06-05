package dot11

import (
	"encoding/binary"
	"errors"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// ErrNotDot11 is returned when the bytes are not a radiotap-framed 802.11 frame
// (e.g. a different link type, or a frame too short/corrupt to parse a header).
var ErrNotDot11 = errors.New("dot11: not a radiotap/802.11 frame")

const (
	// capabilityPrivacyBit is the Privacy bit (bit 4) of the 802.11 Capability
	// Information field in beacon/probe-response bodies; set => link-layer
	// encryption is in use (WEP when no RSN/WPA IE is present).
	capabilityPrivacyBit = 0x0010
	// beaconFixedParamLen is the beacon/probe-response fixed-parameter block:
	// timestamp(8) + beacon interval(2) + capability(2).
	beaconFixedParamLen = 12
	// capabilityFieldOffset is the byte offset of the Capability Information
	// word within that fixed-parameter block.
	capabilityFieldOffset = 10

	// Wi-Fi channel-plan reference frequencies (MHz) for band/channel mapping.
	freqChannel14         = 2484 // 2.4 GHz channel 14 (Japan), numbered specially
	channel14             = 14
	freq24Base            = 2407 // 2.4 GHz: channel = (freq-2407)/5
	freq5Base             = 5000 // 5 GHz: channel = (freq-5000)/5
	freq6Base             = 5950 // 6 GHz: channel = (freq-5950)/5
	freq24Low, freq24High = 2412, 2472
	freq5Low, freq5High   = 5160, 5895
	freq6Low, freq6High   = 5955, 7115
	channelSpacingMHz     = 5
)

// Decode parses one radiotap-framed 802.11 frame into a Frame. It never panics
// on malformed input — every field is bounds-checked — and returns ErrNotDot11
// when the bytes do not contain an 802.11 header.
func Decode(data []byte) (*Frame, error) {
	pkt := gopacket.NewPacket(data, layers.LayerTypeRadioTap, gopacket.DecodeOptions{
		Lazy:   false, // eager: we walk every information-element layer
		NoCopy: true,
	})

	d11, ok := pkt.Layer(layers.LayerTypeDot11).(*layers.Dot11)
	if !ok || d11 == nil {
		return nil, ErrNotDot11
	}

	f := &Frame{
		Kind:        classify(d11.Type),
		Retry:       d11.Flags.Retry(),
		ToDS:        d11.Flags.ToDS(),
		FromDS:      d11.Flags.FromDS(),
		Receiver:    d11.Address1,
		Transmitter: d11.Address2,
		BSSID:       d11.Address3, // for management frames, Address3 is the BSSID
	}

	rt, rtOK := pkt.Layer(layers.LayerTypeRadioTap).(*layers.RadioTap)
	if rtOK && rt != nil && len(rt.RadioTapValues) > 0 {
		// gopacket v1.6 nests the standard fields in RadioTapValues; the first
		// segment carries the regular namespace fields. (Present-bit refinement
		// for multi-segment headers is a later improvement.)
		v := rt.RadioTapValues[0]
		f.SignalDBm = v.DBMAntennaSignal
		f.NoiseDBm = v.DBMAntennaNoise
		f.ChannelMHz = int(v.ChannelFrequency)
		f.Band, f.ChannelNum = bandAndChannel(int(v.ChannelFrequency))
	}

	if f.Kind == KindBeacon || f.Kind == KindProbeResponse {
		f.BSS = decodeBSS(pkt, f.Band)
		// The DS Parameter Set IE channel and the radiotap channel can each be
		// absent; let one fill the other.
		if f.BSS != nil {
			if f.BSS.ChannelNum == 0 {
				f.BSS.ChannelNum = f.ChannelNum
			} else if f.ChannelNum == 0 {
				f.ChannelNum = f.BSS.ChannelNum
			}
		}
	}

	return f, nil
}

// classify maps a gopacket Dot11Type to our coarser Kind.
func classify(t layers.Dot11Type) Kind {
	switch t {
	case layers.Dot11TypeMgmtBeacon:
		return KindBeacon
	case layers.Dot11TypeMgmtProbeReq:
		return KindProbeRequest
	case layers.Dot11TypeMgmtProbeResp:
		return KindProbeResponse
	case layers.Dot11TypeMgmtAssociationReq:
		return KindAssocRequest
	case layers.Dot11TypeMgmtAssociationResp:
		return KindAssocResponse
	case layers.Dot11TypeMgmtReassociationReq:
		return KindReassocRequest
	case layers.Dot11TypeMgmtReassociationResp:
		return KindReassocResponse
	case layers.Dot11TypeMgmtAuthentication:
		return KindAuth
	case layers.Dot11TypeMgmtDeauthentication:
		return KindDeauth
	case layers.Dot11TypeMgmtDisassociation:
		return KindDisassoc
	case layers.Dot11TypeMgmtAction, layers.Dot11TypeMgmtActionNoAck:
		return KindAction
	default:
		if t.MainType() == layers.Dot11TypeData {
			return KindData
		}
		return KindOther
	}
}

// decodeBSS builds the BSS view from a beacon/probe-response: the Privacy bit
// from the Capability Information field plus every information element.
func decodeBSS(pkt gopacket.Packet, band Band) *BSS {
	bss := &BSS{}
	privacy := capabilityPrivacy(pkt)

	phy := phyPresence{}
	for _, l := range pkt.Layers() {
		ieLayer, ok := l.(*layers.Dot11InformationElement)
		if !ok {
			continue
		}
		r := toRawIE(ieLayer)
		applyIE(bss, r)
		markPHY(&phy, r)
	}
	phy.finalize(bss, band)

	// If no RSN/WPA IE classified the suite, the Privacy bit decides WEP vs Open.
	if bss.Security == SecurityUnknown {
		if privacy {
			bss.Security = SecurityWEP
		} else {
			bss.Security = SecurityOpen
		}
	}
	return bss
}

// toRawIE adapts a gopacket information element to our parser shape, splitting
// out the Element ID Extension sub-ID when ID == 255.
func toRawIE(ie *layers.Dot11InformationElement) rawIE {
	r := rawIE{id: uint8(ie.ID), body: ie.Info}
	if r.id == ieElementExtension && len(ie.Info) >= 1 {
		r.extID = ie.Info[0]
		r.body = ie.Info[1:]
	}
	return r
}

// capabilityPrivacy reads the Privacy bit from the Capability Information field
// in a beacon/probe-response fixed-parameter block (timestamp[8] + beacon
// interval[2] + capability[2]). Returns false if the block is absent/short.
func capabilityPrivacy(pkt gopacket.Packet) bool {
	for _, lt := range []gopacket.LayerType{
		layers.LayerTypeDot11MgmtBeacon,
		layers.LayerTypeDot11MgmtProbeResp,
	} {
		if l := pkt.Layer(lt); l != nil {
			body := l.LayerContents()
			if len(body) >= beaconFixedParamLen {
				capInfo := binary.LittleEndian.Uint16(body[capabilityFieldOffset : capabilityFieldOffset+2])
				return capInfo&capabilityPrivacyBit != 0
			}
		}
	}
	return false
}

// bandAndChannel derives the band and channel number from a center frequency in
// MHz (radiotap). Returns BandUnknown/0 for frequencies outside the Wi-Fi bands.
func bandAndChannel(freqMHz int) (Band, int) {
	switch {
	case freqMHz == freqChannel14:
		return Band24GHz, channel14
	case freqMHz >= freq24Low && freqMHz <= freq24High:
		return Band24GHz, (freqMHz - freq24Base) / channelSpacingMHz
	case freqMHz >= freq5Low && freqMHz <= freq5High:
		return Band5GHz, (freqMHz - freq5Base) / channelSpacingMHz
	case freqMHz >= freq6Low && freqMHz <= freq6High:
		// 6 GHz (Wi-Fi 6E / 7): channels numbered from 5950 MHz.
		return Band6GHz, (freqMHz - freq6Base) / channelSpacingMHz
	default:
		return BandUnknown, 0
	}
}
