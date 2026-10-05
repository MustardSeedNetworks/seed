package voip

// Codec is an RTP audio payload type the analyser scores.
type Codec struct {
	Name string
	// ClockRate is the RTP timestamp rate in Hz (RFC 3551).
	ClockRate uint32
	// Ie and Bpl are the codec's equipment impairment factor and packet
	// loss robustness, from ITU-T G.113 Appendix I (G.711 with its
	// Appendix I packet loss concealment).
	Ie, Bpl float64
	// LookaheadMs is the encoder's algorithmic delay beyond one frame.
	LookaheadMs float64
}

// The scored static payload types (RFC 3551) and their shared clock.
const (
	ptPCMU          = 0
	ptPCMA          = 8
	ptG729          = 18
	narrowbandClock = 8000
)

// Codec impairments from ITU-T G.113 Appendix I.
const (
	g711Bpl         = 25.1
	g729Ie          = 11
	g729Bpl         = 19
	g729LookaheadMs = 5
)

// codecFor returns the codec of a static RTP payload type. Dynamic types
// (96-127) carry no codec without the call's SDP, so they are not scored.
// G.722 is wideband and needs the G.107.1 model, so it is not scored
// either.
func codecFor(pt uint8) (Codec, bool) {
	switch pt {
	case ptPCMU:
		return Codec{Name: "PCMU", ClockRate: narrowbandClock, Bpl: g711Bpl}, true
	case ptPCMA:
		return Codec{Name: "PCMA", ClockRate: narrowbandClock, Bpl: g711Bpl}, true
	case ptG729:
		return Codec{
			Name: "G729", ClockRate: narrowbandClock,
			Ie: g729Ie, Bpl: g729Bpl, LookaheadMs: g729LookaheadMs,
		}, true
	}
	return Codec{}, false
}

// The E-model reduced to its delay and loss terms. With every other
// ITU-T G.107 parameter at its default, Ro − Is is 93.2, and the delay
// impairment Id is 0.024·d plus 0.11·(d − 177.3) past the knee (Cole and
// Rosenbluth, "Voice over IP performance monitoring", 2001). Against the
// full G.107 equations that stays within 0.25 R up to 300 ms and drifts
// past it, where a call is already rated poor; the tests hold it there.
const (
	rDefault      = 93.2
	idSlope       = 0.024
	idKneeMs      = 177.3
	idKneeSlope   = 0.11
	ieEffCeiling  = 95
	burstRatio    = 1 // random loss
	mosFloor      = 1
	mosCeiling    = 4.5
	rCeiling      = 100
	mosLinear     = 0.035
	mosCubicKnee  = 60
	mosCubicScale = 7e-6
)

// RFactor is the E-model transmission rating for a narrowband call with
// one-way delay delayMs and random packet loss of lossPct percent through
// codec c.
func RFactor(c Codec, delayMs, lossPct float64) float64 {
	id := idSlope * delayMs
	if delayMs > idKneeMs {
		id += idKneeSlope * (delayMs - idKneeMs)
	}
	// G.107 equation 7-29.
	ieEff := c.Ie + (ieEffCeiling-c.Ie)*lossPct/(lossPct/burstRatio+c.Bpl)
	return rDefault - id - ieEff
}

// MOS converts an R factor to the G.107 Annex B conversational quality
// estimate, 1 to 4.5.
func MOS(r float64) float64 {
	switch {
	case r <= 0:
		return mosFloor
	case r >= rCeiling:
		return mosCeiling
	}
	return mosFloor + mosLinear*r + r*(r-mosCubicKnee)*(rCeiling-r)*mosCubicScale
}
