package dot11

import (
	"encoding/binary"
	"net"
	"testing"
)

// radiotapMinimal is an 8-byte radiotap header advertising no present fields
// (version, pad, len=8, present=0). Radio measurements are then absent and the
// channel comes from the DS Parameter Set IE — which is what we want to test
// without hand-assembling radiotap's alignment-sensitive value block.
func radiotapMinimal() []byte {
	return []byte{0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00}
}

// ie assembles one information element: id, length, body.
func ie(id byte, body ...byte) []byte {
	return append([]byte{id, byte(len(body))}, body...)
}

// rsnIE builds a minimal RSN element (CCMP group + pairwise, a single AKM, and
// the RSN capabilities word) so the security/PMF classifier can be exercised.
func rsnIE(akmType byte, rsnCaps uint16) []byte {
	body := []byte{
		0x01, 0x00, // version 1
		0x00, 0x0f, 0xac, 0x04, // group cipher: CCMP
		0x01, 0x00, // pairwise count 1
		0x00, 0x0f, 0xac, 0x04, // pairwise: CCMP
		0x01, 0x00, // AKM count 1
		0x00, 0x0f, 0xac, akmType, // AKM selector
	}
	caps := make([]byte, 2)
	binary.LittleEndian.PutUint16(caps, rsnCaps)
	body = append(body, caps...)
	return ie(ieRSN, body...)
}

// beacon assembles radiotapMinimal + an 802.11 beacon (broadcast DA, the given
// BSSID, the capability word, the supplied IEs) + a 4-byte dummy FCS (gopacket
// strips the trailing 4 bytes as the frame checksum).
func beacon(bssid net.HardwareAddr, capability uint16, ies ...[]byte) []byte {
	frame := []byte{0x80, 0x00, 0x00, 0x00}                   // FC (mgmt/beacon) + duration
	frame = append(frame, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff) // Address1 broadcast
	frame = append(frame, bssid...)                           // Address2
	frame = append(frame, bssid...)                           // Address3 (BSSID)
	frame = append(frame, 0x00, 0x00)                         // sequence control
	frame = append(frame, make([]byte, 8)...)                 // timestamp
	frame = append(frame, 0x64, 0x00)                         // beacon interval
	capWord := make([]byte, 2)
	binary.LittleEndian.PutUint16(capWord, capability)
	frame = append(frame, capWord...) // capability information
	for _, e := range ies {
		frame = append(frame, e...)
	}
	frame = append(frame, 0xde, 0xad, 0xbe, 0xef) // dummy FCS

	rt := radiotapMinimal()
	out := make([]byte, 0, len(rt)+len(frame))
	out = append(out, rt...)
	return append(out, frame...)
}

const (
	capESS     = 0x0001
	capPrivacy = 0x0010
)

// htCapIE's mere presence marks the BSS as 802.11n (HT). The 26-byte body is
// the HT Capabilities element size; its contents are irrelevant to presence.
func htCapIE() []byte { return ie(ieHTCapabilities, make([]byte, 26)...) }

// ratesOFDM is a Supported Rates IE that includes OFDM rates (distinguishes g/a).
func ratesOFDM() []byte { return ie(ieSupportedRates, 0x8c, 0x12, 0x98, 0x24) }

// dsChan6 is a DS Parameter Set IE advertising channel 6.
func dsChan6() []byte { return ie(ieDSParameterSet, 6) }

func mustDecode(t *testing.T, data []byte) *Frame {
	t.Helper()
	f, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return f
}

func TestDecodeBeaconWPA2N(t *testing.T) {
	t.Parallel()
	bssid := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	data := beacon(bssid, capESS|capPrivacy,
		ie(ieSSID, []byte("TestNet")...),
		ratesOFDM(), dsChan6(),
		rsnIE(akmPSK, 0),
		htCapIE(),
	)

	f := mustDecode(t, data)
	if f.Kind != KindBeacon {
		t.Fatalf("Kind = %v, want beacon", f.Kind)
	}
	if f.BSSID.String() != bssid.String() {
		t.Errorf("BSSID = %v, want %v", f.BSSID, bssid)
	}
	if f.BSS == nil {
		t.Fatal("BSS is nil")
	}
	if f.BSS.SSID != "TestNet" {
		t.Errorf("SSID = %q, want TestNet", f.BSS.SSID)
	}
	if f.BSS.ChannelNum != 6 {
		t.Errorf("ChannelNum = %d, want 6", f.BSS.ChannelNum)
	}
	if f.BSS.Security != SecurityWPA2 {
		t.Errorf("Security = %v, want WPA2", f.BSS.Security)
	}
	if f.BSS.Standard != Standard80211n {
		t.Errorf("Standard = %v, want 802.11n", f.BSS.Standard)
	}
}

func TestDecodeBeaconWPA3PMFRequired(t *testing.T) {
	t.Parallel()
	const mfpr = 1 << 6 // RSN capabilities: Management Frame Protection Required
	bssid := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	data := beacon(bssid, capESS|capPrivacy,
		ie(ieSSID, []byte("Secure")...),
		dsChan6(),
		rsnIE(akmSAE, mfpr),
	)

	f := mustDecode(t, data)
	if f.BSS.Security != SecurityWPA3 {
		t.Errorf("Security = %v, want WPA3", f.BSS.Security)
	}
	if !f.BSS.PMFRequired {
		t.Error("PMFRequired = false, want true")
	}
}

func TestDecodeBeaconWPA2WPA3Transition(t *testing.T) {
	t.Parallel()
	// Two AKMs — PSK + SAE — in one RSN IE marks WPA3 transition mode.
	bssid := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x01}
	rsn := []byte{
		0x01, 0x00,
		0x00, 0x0f, 0xac, 0x04,
		0x01, 0x00, 0x00, 0x0f, 0xac, 0x04,
		0x02, 0x00, // AKM count 2
		0x00, 0x0f, 0xac, akmPSK,
		0x00, 0x0f, 0xac, akmSAE,
		0x00, 0x00,
	}
	data := beacon(bssid, capESS|capPrivacy, ie(ieSSID, []byte("Mixed")...), ie(ieRSN, rsn...))
	f := mustDecode(t, data)
	if f.BSS.Security != SecurityWPA2WPA3 {
		t.Errorf("Security = %v, want WPA2/WPA3", f.BSS.Security)
	}
}

func TestDecodeBeaconOpenHidden(t *testing.T) {
	t.Parallel()
	// No RSN/WPA IE + Privacy clear => Open. Zero-length SSID => hidden.
	bssid := net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	data := beacon(bssid, capESS, ie(ieSSID), ratesOFDM(), dsChan6())
	f := mustDecode(t, data)
	if f.BSS.Security != SecurityOpen {
		t.Errorf("Security = %v, want Open", f.BSS.Security)
	}
	if !f.BSS.Hidden {
		t.Error("Hidden = false, want true")
	}
	if f.BSS.Standard != Standard80211g {
		t.Errorf("Standard = %v, want 802.11g (OFDM rates, no HT)", f.BSS.Standard)
	}
}

func TestDecodeBeaconWEP(t *testing.T) {
	t.Parallel()
	// Privacy set, no RSN/WPA IE => WEP.
	bssid := net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x03}
	data := beacon(bssid, capESS|capPrivacy, ie(ieSSID, []byte("OldNet")...), dsChan6())
	f := mustDecode(t, data)
	if f.BSS.Security != SecurityWEP {
		t.Errorf("Security = %v, want WEP", f.BSS.Security)
	}
}

func TestDecodeDeauthHasNoBSS(t *testing.T) {
	t.Parallel()
	// A deauthentication frame: FC subtype 12 (0xC0), no IE body.
	frame := []byte{0xc0, 0x00, 0x00, 0x00}
	frame = append(frame, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55) // Address1
	frame = append(frame, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff) // Address2
	frame = append(frame, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff) // Address3 (BSSID)
	frame = append(frame, 0x00, 0x00)                         // sequence
	frame = append(frame, 0x07, 0x00)                         // reason code
	frame = append(frame, 0xde, 0xad, 0xbe, 0xef)             // FCS
	data := append(radiotapMinimal(), frame...)

	f := mustDecode(t, data)
	if f.Kind != KindDeauth {
		t.Errorf("Kind = %v, want deauth", f.Kind)
	}
	if f.BSS != nil {
		t.Error("BSS should be nil for a deauth frame")
	}
}

func TestDecodeRejectsNonDot11(t *testing.T) {
	t.Parallel()
	if _, err := Decode([]byte{0x01, 0x02, 0x03}); err == nil {
		t.Error("expected an error decoding garbage, got nil")
	}
}

func TestBandAndChannel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		freq    int
		band    Band
		channel int
	}{
		{2412, Band24GHz, 1},
		{2437, Band24GHz, 6},
		{2484, Band24GHz, 14},
		{5180, Band5GHz, 36},
		{5955, Band6GHz, 1},
		{6175, Band6GHz, 45},
		{9999, BandUnknown, 0},
	}
	for _, c := range cases {
		band, ch := bandAndChannel(c.freq)
		if band != c.band || ch != c.channel {
			t.Errorf("bandAndChannel(%d) = %v/%d, want %v/%d", c.freq, band, ch, c.band, c.channel)
		}
	}
}
