// Package channel provides WiFi channel analysis and recommendation functionality.
package channel

import (
	"sort"
	"time"
)

// Band represents WiFi frequency bands.
type Band string

// Band constants.
const (
	Band24GHz Band = "2.4GHz"
	Band5GHz  Band = "5GHz"
	Band6GHz  Band = "6GHz"
)

// Channel width constants in MHz.
const (
	Width20MHz  = 20
	Width40MHz  = 40
	Width80MHz  = 80
	Width160MHz = 160
	Width320MHz = 320
)

// Frequency ranges for band detection (MHz).
const (
	Freq24GHzMin = 2400
	Freq24GHzMax = 2500
	Freq5GHzMin  = 5150
	Freq5GHzMax  = 5895
	Freq6GHzMin  = 5925
	Freq6GHzMax  = 7125
)

// Channel frequency conversion constants.
const (
	Freq24GHzBase        = 2407
	Freq24GHzChannel14   = 2484
	Freq5GHzBase         = 5000
	Freq6GHzBase         = 5950
	ChannelSpacing       = 5
	Channel24GHzMax      = 13
	Channel24GHzJapan    = 14
	Channel5GHzUNII1Min  = 36
	Channel5GHzUNII1Max  = 48
	Channel5GHzUNII2Min  = 52
	Channel5GHzUNII2Max  = 64
	Channel5GHzUNII2EMin = 100
	Channel5GHzUNII2EMax = 144
	Channel5GHzUNII3Min  = 149
	Channel5GHzUNII3Max  = 165
)

// DFS channel ranges in 5 GHz band.
const (
	DFSChannelMin1 = 52
	DFSChannelMax1 = 64
	DFSChannelMin2 = 100
	DFSChannelMax2 = 144
)

// InterferenceScore calculation constants.
const (
	BaseUtilizationPerNetwork = 10.0 // Base utilization percentage per network on channel
	InterferenceMultiplier    = 1.5  // Multiplier for co-channel interference
	AdjacentInterference      = 5.0  // Interference from adjacent channels (2.4 GHz)
	OverlapPenalty            = 2.0  // Penalty multiplier for overlapping channels
)

// Channel overlap and utilization constants.
const (
	Channel24GHzWidth     = 5    // Number of channels a 2.4 GHz signal overlaps
	MaxUtilizationPercent = 100  // Maximum utilization percentage cap
	Channel6GHzCount      = 59   // Number of 6 GHz channels (every 4th from 1-233)
	MaxInterferenceScore  = 100  // Maximum interference score
	SignalStrongThreshold = -30  // Signal strength threshold for "very strong" (dBm)
	SignalWeakThreshold   = -90  // Signal strength threshold for "very weak" (dBm)
	InterferenceScaleMin  = 1.0  // Minimum interference factor
	InterferenceScaleMax  = 10.0 // Maximum interference factor
	SignalRange           = 60   // Range between strong and weak thresholds
)

// NetworkInfo contains network data for channel analysis.
type NetworkInfo struct {
	SSID         string
	BSSID        string
	Channel      int
	Frequency    int // MHz
	ChannelWidth int // MHz
	Signal       int // dBm
}

// ChannelInfo contains information about a specific channel.
//
//nolint:revive // Intentional: explicit naming for clarity when used outside package
type ChannelInfo struct {
	Number        int     `json:"number"`
	CenterFreqMHz int     `json:"centerFreqMHz"`
	Band          Band    `json:"band"`
	NetworkCount  int     `json:"networkCount"`
	Utilization   float64 `json:"utilizationPercent"`
	Interference  float64 `json:"interferenceScore"` // 0-100
	IsRecommended bool    `json:"isRecommended"`
	IsDFS         bool    `json:"isDfs"`
	MaxPowerDbm   int     `json:"maxPowerDbm,omitempty"`
}

// Analysis contains channel utilization analysis results.
type Analysis struct {
	Band               Band          `json:"band"`
	Channels           []ChannelInfo `json:"channels"`
	RecommendedChannel int           `json:"recommendedChannel"`
	AnalyzedAt         time.Time     `json:"analyzedAt"`
}

// Analyzer performs WiFi channel analysis.
type Analyzer struct {
	// Configuration options
	includeDFS bool
}

// NewAnalyzer creates a new channel analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		includeDFS: false,
	}
}

// WithDFS configures whether to include DFS channels in recommendations.
func (a *Analyzer) WithDFS(include bool) *Analyzer {
	a.includeDFS = include
	return a
}

// Analyze performs channel analysis for the specified band.
func (a *Analyzer) Analyze(networks []NetworkInfo, band Band) *Analysis {
	analysis := &Analysis{
		Band:       band,
		Channels:   []ChannelInfo{},
		AnalyzedAt: time.Now(),
	}

	// Get channels for the band
	channels := getChannelsForBand(band)
	if len(channels) == 0 {
		return analysis
	}

	// Group networks by channel
	channelNetworks := groupNetworksByChannel(networks, band)

	// Build channel info for each channel
	channelInfos := make([]ChannelInfo, 0, len(channels))
	for _, ch := range channels {
		info := buildChannelInfo(ch, band, channelNetworks, networks)

		// Skip DFS channels if not included
		if info.IsDFS && !a.includeDFS {
			continue
		}

		channelInfos = append(channelInfos, info)
	}

	// Find best channel (lowest interference)
	bestChannel := findBestChannel(channelInfos)
	if bestChannel > 0 {
		for i := range channelInfos {
			if channelInfos[i].Number == bestChannel {
				channelInfos[i].IsRecommended = true
				break
			}
		}
	}

	analysis.Channels = channelInfos
	analysis.RecommendedChannel = bestChannel

	return analysis
}

// GetBand determines the frequency band from frequency in MHz.
func GetBand(freq int) Band {
	switch {
	case freq >= Freq24GHzMin && freq <= Freq24GHzMax:
		return Band24GHz
	case freq >= Freq5GHzMin && freq <= Freq5GHzMax:
		return Band5GHz
	case freq >= Freq6GHzMin && freq <= Freq6GHzMax:
		return Band6GHz
	default:
		return ""
	}
}

// ChannelToFrequency converts a WiFi channel number to center frequency in MHz.
//
//nolint:revive // Intentional: explicit naming for clarity when used outside package
func ChannelToFrequency(channel int, band Band) int {
	switch band {
	case Band24GHz:
		if channel >= 1 && channel <= Channel24GHzMax {
			return Freq24GHzBase + (channel * ChannelSpacing)
		}
		if channel == Channel24GHzJapan {
			return Freq24GHzChannel14
		}
	case Band5GHz:
		if isValid5GHzChannel(channel) {
			return Freq5GHzBase + (channel * ChannelSpacing)
		}
	case Band6GHz:
		if channel >= 1 && channel <= 233 {
			return Freq6GHzBase + (channel * ChannelSpacing)
		}
	}
	return 0
}

// FrequencyToChannel converts frequency in MHz to WiFi channel number.
func FrequencyToChannel(freq int) int {
	band := GetBand(freq)
	switch band {
	case Band24GHz:
		if freq == Freq24GHzChannel14 {
			return Channel24GHzJapan
		}
		return (freq - Freq24GHzBase) / ChannelSpacing
	case Band5GHz:
		return (freq - Freq5GHzBase) / ChannelSpacing
	case Band6GHz:
		return (freq - Freq6GHzBase) / ChannelSpacing
	}
	return 0
}

// IsDFSChannel returns true if the channel is a DFS channel.
func IsDFSChannel(channel int) bool {
	return (channel >= DFSChannelMin1 && channel <= DFSChannelMax1) ||
		(channel >= DFSChannelMin2 && channel <= DFSChannelMax2)
}

// IsNonOverlapping24GHz returns true if the channel is a non-overlapping 2.4 GHz channel.
func IsNonOverlapping24GHz(channel int) bool {
	// Channels 1, 6, and 11 are the standard non-overlapping channels
	return channel == 1 || channel == 6 || channel == 11
}

// GetChannelOverlap calculates the channel overlap for 2.4 GHz channels.
// Returns the number of overlapping channels (0-4).
func GetChannelOverlap(channel1, channel2 int) int {
	if channel1 < 1 || channel1 > Channel24GHzMax || channel2 < 1 || channel2 > Channel24GHzMax {
		return 0
	}
	diff := abs(channel1 - channel2)
	if diff >= Channel24GHzWidth {
		return 0 // No overlap
	}
	return Channel24GHzWidth - diff // 5 channels wide in 2.4 GHz
}

// CalculateInterference calculates interference score for a channel.
func CalculateInterference(channel int, band Band, networks []NetworkInfo) float64 {
	var score float64

	for _, n := range networks {
		networkBand := GetBand(n.Frequency)
		if networkBand != band {
			continue
		}

		if n.Channel == channel {
			// Co-channel interference
			score += InterferenceMultiplier * signalToInterference(n.Signal)
		} else if band == Band24GHz {
			// Adjacent channel interference (2.4 GHz only)
			overlap := GetChannelOverlap(channel, n.Channel)
			if overlap > 0 {
				overlapFactor := float64(overlap) / float64(Channel24GHzWidth)
				score += overlapFactor * AdjacentInterference * signalToInterference(n.Signal)
			}
		}
	}

	// Normalize to 0-100 scale
	if score > MaxInterferenceScore {
		score = MaxInterferenceScore
	}

	return score
}

// Helper functions

func getChannelsForBand(band Band) []int {
	switch band {
	case Band24GHz:
		// Standard 2.4 GHz channels (1-11 for US, 1-13 for most others)
		return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	case Band5GHz:
		// UNII-1, UNII-2, UNII-2 Extended, UNII-3
		return []int{
			36, 40, 44, 48, // UNII-1
			52, 56, 60, 64, // UNII-2 (DFS)
			100, 104, 108, 112, 116, // UNII-2 Extended (DFS)
			120, 124, 128, 132, 136, // UNII-2 Extended (DFS)
			140, 144, // UNII-2 Extended (DFS)
			149, 153, 157, 161, 165, // UNII-3
		}
	case Band6GHz:
		// 6 GHz channels (simplified set)
		channels := make([]int, 0, Channel6GHzCount)
		for ch := 1; ch <= 233; ch += 4 {
			channels = append(channels, ch)
		}
		return channels
	default:
		return nil
	}
}

func isValid5GHzChannel(channel int) bool {
	return (channel >= Channel5GHzUNII1Min && channel <= Channel5GHzUNII1Max) ||
		(channel >= Channel5GHzUNII2Min && channel <= Channel5GHzUNII2Max) ||
		(channel >= Channel5GHzUNII2EMin && channel <= Channel5GHzUNII2EMax) ||
		(channel >= Channel5GHzUNII3Min && channel <= Channel5GHzUNII3Max)
}

func groupNetworksByChannel(networks []NetworkInfo, band Band) map[int][]NetworkInfo {
	result := make(map[int][]NetworkInfo)
	for _, n := range networks {
		networkBand := GetBand(n.Frequency)
		if networkBand != band {
			continue
		}
		result[n.Channel] = append(result[n.Channel], n)
	}
	return result
}

func buildChannelInfo(
	channel int,
	band Band,
	channelNetworks map[int][]NetworkInfo,
	allNetworks []NetworkInfo,
) ChannelInfo {
	networks := channelNetworks[channel]
	networkCount := len(networks)

	info := ChannelInfo{
		Number:        channel,
		CenterFreqMHz: ChannelToFrequency(channel, band),
		Band:          band,
		NetworkCount:  networkCount,
		Utilization:   float64(networkCount) * BaseUtilizationPerNetwork,
		IsDFS:         band == Band5GHz && IsDFSChannel(channel),
	}

	// Cap utilization at 100%
	if info.Utilization > MaxUtilizationPercent {
		info.Utilization = MaxUtilizationPercent
	}

	// Calculate interference score
	info.Interference = CalculateInterference(channel, band, allNetworks)

	return info
}

func findBestChannel(channels []ChannelInfo) int {
	if len(channels) == 0 {
		return 0
	}

	// Sort by interference score (lowest first), then by network count
	sorted := make([]ChannelInfo, len(channels))
	copy(sorted, channels)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Interference != sorted[j].Interference {
			return sorted[i].Interference < sorted[j].Interference
		}
		return sorted[i].NetworkCount < sorted[j].NetworkCount
	})

	return sorted[0].Number
}

func signalToInterference(signal int) float64 {
	// Stronger signals cause more interference
	// Convert dBm to interference factor (stronger = higher interference)
	// Typical range: -30 dBm (very strong) to -90 dBm (weak)
	if signal >= SignalStrongThreshold {
		return InterferenceScaleMax
	}
	if signal <= SignalWeakThreshold {
		return InterferenceScaleMin
	}
	// Linear interpolation from 1.0 (at -90 dBm) to 10.0 (at -30 dBm)
	// For signal = -40: 1.0 + 9.0 * ((-40) - (-90)) / 60.0 = 1.0 + 9.0 * 50/60 = 8.5
	scaleRange := InterferenceScaleMax - InterferenceScaleMin
	return InterferenceScaleMin + scaleRange*float64(signal-SignalWeakThreshold)/float64(SignalRange)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
