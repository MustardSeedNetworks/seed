package listener

import "time"

// VoIPQualityKind is the event kind the VoIP analyser publishes when one
// window of an RTP stream scores at or below its alert MOS (P-A7). The
// event's SourceAddr is the stream's sender, so alert suppression holds
// one alert per sending host per window.
const VoIPQualityKind = "voip-quality"

// VoIPQuality is the payload of a [VoIPQualityKind] event.
type VoIPQuality struct {
	Interface   string    `json:"interface"`
	Src         string    `json:"src"`
	Dst         string    `json:"dst"`
	SSRC        uint32    `json:"ssrc"`
	Codec       string    `json:"codec"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	LossPct     float64   `json:"lossPct"`
	JitterMs    float64   `json:"jitterMs"`
	MaxJitterMs float64   `json:"maxJitterMs"`
	RFactor     float64   `json:"rFactor"`
	MOS         float64   `json:"mos"`
}
