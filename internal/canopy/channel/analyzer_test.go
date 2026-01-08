package channel_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/channel"
)

// TestGetBand verifies frequency band detection.
func TestGetBand(t *testing.T) {
	tests := []struct {
		name string
		freq int
		want channel.Band
	}{
		// 2.4 GHz band
		{"2.4 GHz channel 1", 2412, channel.Band24GHz},
		{"2.4 GHz channel 6", 2437, channel.Band24GHz},
		{"2.4 GHz channel 11", 2462, channel.Band24GHz},
		{"2.4 GHz channel 14", 2484, channel.Band24GHz},
		{"2.4 GHz lower bound", 2400, channel.Band24GHz},
		{"2.4 GHz upper bound", 2500, channel.Band24GHz},

		// 5 GHz band
		{"5 GHz channel 36", 5180, channel.Band5GHz},
		{"5 GHz channel 52 (DFS)", 5260, channel.Band5GHz},
		{"5 GHz channel 100 (DFS)", 5500, channel.Band5GHz},
		{"5 GHz channel 149", 5745, channel.Band5GHz},
		{"5 GHz channel 165", 5825, channel.Band5GHz},
		{"5 GHz lower bound", 5150, channel.Band5GHz},
		{"5 GHz upper bound", 5895, channel.Band5GHz},

		// 6 GHz band
		{"6 GHz channel 1", 5955, channel.Band6GHz},
		{"6 GHz channel 93", 6415, channel.Band6GHz},
		{"6 GHz channel 233", 7115, channel.Band6GHz},
		{"6 GHz lower bound", 5925, channel.Band6GHz},
		{"6 GHz upper bound", 7125, channel.Band6GHz},

		// Unknown frequencies
		{"Unknown low frequency", 1000, ""},
		{"Zero frequency", 0, ""},
		{"Between 2.4 and 5 GHz", 3000, ""},
		{"Above 6 GHz", 8000, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.GetBand(tt.freq)
			if got != tt.want {
				t.Errorf("GetBand(%d) = %q, want %q", tt.freq, got, tt.want)
			}
		})
	}
}

// TestChannelToFrequency verifies channel to frequency conversion.
func TestChannelToFrequency(t *testing.T) {
	tests := []struct {
		name    string
		channel int
		band    channel.Band
		want    int
	}{
		// 2.4 GHz band
		{"2.4 GHz channel 1", 1, channel.Band24GHz, 2412},
		{"2.4 GHz channel 6", 6, channel.Band24GHz, 2437},
		{"2.4 GHz channel 11", 11, channel.Band24GHz, 2462},
		{"2.4 GHz channel 13", 13, channel.Band24GHz, 2472},
		{"2.4 GHz channel 14 (Japan)", 14, channel.Band24GHz, 2484},
		{"2.4 GHz invalid channel 0", 0, channel.Band24GHz, 0},
		{"2.4 GHz invalid channel 15", 15, channel.Band24GHz, 0},

		// 5 GHz band
		{"5 GHz channel 36", 36, channel.Band5GHz, 5180},
		{"5 GHz channel 40", 40, channel.Band5GHz, 5200},
		{"5 GHz channel 44", 44, channel.Band5GHz, 5220},
		{"5 GHz channel 48", 48, channel.Band5GHz, 5240},
		{"5 GHz channel 52 (DFS)", 52, channel.Band5GHz, 5260},
		{"5 GHz channel 100 (DFS)", 100, channel.Band5GHz, 5500},
		{"5 GHz channel 149", 149, channel.Band5GHz, 5745},
		{"5 GHz channel 165", 165, channel.Band5GHz, 5825},
		{"5 GHz invalid channel 35", 35, channel.Band5GHz, 0},
		{"5 GHz invalid channel 50", 50, channel.Band5GHz, 0},

		// 6 GHz band
		{"6 GHz channel 1", 1, channel.Band6GHz, 5955},
		{"6 GHz channel 5", 5, channel.Band6GHz, 5975},
		{"6 GHz channel 233", 233, channel.Band6GHz, 7115},
		{"6 GHz invalid channel 0", 0, channel.Band6GHz, 0},
		{"6 GHz invalid channel 234", 234, channel.Band6GHz, 0},

		// Unknown band
		{"Unknown band", 1, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.ChannelToFrequency(tt.channel, tt.band)
			if got != tt.want {
				t.Errorf(
					"ChannelToFrequency(%d, %q) = %d, want %d",
					tt.channel,
					tt.band,
					got,
					tt.want,
				)
			}
		})
	}
}

// TestFrequencyToChannel verifies frequency to channel conversion.
func TestFrequencyToChannel(t *testing.T) {
	tests := []struct {
		name string
		freq int
		want int
	}{
		// 2.4 GHz band
		{"2.4 GHz 2412 MHz", 2412, 1},
		{"2.4 GHz 2437 MHz", 2437, 6},
		{"2.4 GHz 2462 MHz", 2462, 11},
		{"2.4 GHz 2472 MHz", 2472, 13},
		{"2.4 GHz 2484 MHz (channel 14)", 2484, 14},

		// 5 GHz band
		{"5 GHz 5180 MHz", 5180, 36},
		{"5 GHz 5200 MHz", 5200, 40},
		{"5 GHz 5260 MHz", 5260, 52},
		{"5 GHz 5500 MHz", 5500, 100},
		{"5 GHz 5745 MHz", 5745, 149},
		{"5 GHz 5825 MHz", 5825, 165},

		// 6 GHz band
		{"6 GHz 5955 MHz", 5955, 1},
		{"6 GHz 5975 MHz", 5975, 5},
		{"6 GHz 7115 MHz", 7115, 233},

		// Unknown frequencies
		{"Unknown frequency 1000", 1000, 0},
		{"Zero frequency", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.FrequencyToChannel(tt.freq)
			if got != tt.want {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, got, tt.want)
			}
		})
	}
}

// TestIsDFSChannel verifies DFS channel detection.
func TestIsDFSChannel(t *testing.T) {
	tests := []struct {
		name    string
		channel int
		want    bool
	}{
		// Non-DFS UNII-1 channels
		{"Channel 36 (non-DFS)", 36, false},
		{"Channel 40 (non-DFS)", 40, false},
		{"Channel 44 (non-DFS)", 44, false},
		{"Channel 48 (non-DFS)", 48, false},

		// DFS UNII-2 channels (52-64)
		{"Channel 52 (DFS)", 52, true},
		{"Channel 56 (DFS)", 56, true},
		{"Channel 60 (DFS)", 60, true},
		{"Channel 64 (DFS)", 64, true},

		// DFS UNII-2 Extended channels (100-144)
		{"Channel 100 (DFS)", 100, true},
		{"Channel 116 (DFS)", 116, true},
		{"Channel 132 (DFS)", 132, true},
		{"Channel 144 (DFS)", 144, true},

		// Non-DFS UNII-3 channels
		{"Channel 149 (non-DFS)", 149, false},
		{"Channel 153 (non-DFS)", 153, false},
		{"Channel 157 (non-DFS)", 157, false},
		{"Channel 161 (non-DFS)", 161, false},
		{"Channel 165 (non-DFS)", 165, false},

		// Edge cases
		{"Channel 51 (non-DFS)", 51, false},
		{"Channel 65 (non-DFS)", 65, false},
		{"Channel 99 (non-DFS)", 99, false},
		{"Channel 145 (non-DFS)", 145, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.IsDFSChannel(tt.channel)
			if got != tt.want {
				t.Errorf("IsDFSChannel(%d) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

// TestIsNonOverlapping24GHz verifies non-overlapping channel detection.
func TestIsNonOverlapping24GHz(t *testing.T) {
	tests := []struct {
		name    string
		channel int
		want    bool
	}{
		{"Channel 1 (non-overlapping)", 1, true},
		{"Channel 2", 2, false},
		{"Channel 3", 3, false},
		{"Channel 4", 4, false},
		{"Channel 5", 5, false},
		{"Channel 6 (non-overlapping)", 6, true},
		{"Channel 7", 7, false},
		{"Channel 8", 8, false},
		{"Channel 9", 9, false},
		{"Channel 10", 10, false},
		{"Channel 11 (non-overlapping)", 11, true},
		{"Channel 12", 12, false},
		{"Channel 13", 13, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.IsNonOverlapping24GHz(tt.channel)
			if got != tt.want {
				t.Errorf("IsNonOverlapping24GHz(%d) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

// TestGetChannelOverlap verifies channel overlap calculation.
func TestGetChannelOverlap(t *testing.T) {
	tests := []struct {
		name     string
		channel1 int
		channel2 int
		want     int
	}{
		// Same channel
		{"Same channel 1", 1, 1, 5},
		{"Same channel 6", 6, 6, 5},

		// Adjacent channels
		{"Channels 1 and 2", 1, 2, 4},
		{"Channels 6 and 7", 6, 7, 4},

		// 2 apart
		{"Channels 1 and 3", 1, 3, 3},
		{"Channels 6 and 8", 6, 8, 3},

		// 3 apart
		{"Channels 1 and 4", 1, 4, 2},
		{"Channels 6 and 9", 6, 9, 2},

		// 4 apart
		{"Channels 1 and 5", 1, 5, 1},
		{"Channels 6 and 10", 6, 10, 1},

		// 5+ apart (no overlap)
		{"Channels 1 and 6 (non-overlapping)", 1, 6, 0},
		{"Channels 1 and 11 (non-overlapping)", 1, 11, 0},
		{"Channels 6 and 11 (non-overlapping)", 6, 11, 0},

		// Order independence
		{"Channels 6 and 1 (reversed)", 6, 1, 0},
		{"Channels 11 and 6 (reversed)", 11, 6, 0},

		// Invalid channels
		{"Invalid channel 0", 0, 1, 0},
		{"Invalid channel 14", 14, 1, 0},
		{"Both invalid", 0, 14, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.GetChannelOverlap(tt.channel1, tt.channel2)
			if got != tt.want {
				t.Errorf(
					"GetChannelOverlap(%d, %d) = %d, want %d",
					tt.channel1,
					tt.channel2,
					got,
					tt.want,
				)
			}
		})
	}
}

// TestCalculateInterference verifies interference calculation.
func TestCalculateInterference(t *testing.T) {
	tests := []struct {
		name     string
		channel  int
		band     channel.Band
		networks []channel.NetworkInfo
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "No networks",
			channel:  6,
			band:     channel.Band24GHz,
			networks: nil,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:    "Single co-channel network",
			channel: 6,
			band:    channel.Band24GHz,
			networks: []channel.NetworkInfo{
				{Channel: 6, Frequency: 2437, Signal: -50},
			},
			wantMin: 1,
			wantMax: 20,
		},
		{
			name:    "Adjacent channel interference (2.4 GHz)",
			channel: 6,
			band:    channel.Band24GHz,
			networks: []channel.NetworkInfo{
				{Channel: 5, Frequency: 2432, Signal: -50},
			},
			wantMin: 20,
			wantMax: 40,
		},
		{
			name:    "No interference from distant channel",
			channel: 1,
			band:    channel.Band24GHz,
			networks: []channel.NetworkInfo{
				{Channel: 11, Frequency: 2462, Signal: -50},
			},
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "Strong signal causes more interference",
			channel: 36,
			band:    channel.Band5GHz,
			networks: []channel.NetworkInfo{
				{Channel: 36, Frequency: 5180, Signal: -30},
			},
			wantMin: 10,
			wantMax: 20,
		},
		{
			name:    "Weak signal causes less interference",
			channel: 36,
			band:    channel.Band5GHz,
			networks: []channel.NetworkInfo{
				{Channel: 36, Frequency: 5180, Signal: -90},
			},
			wantMin: 0.5,
			wantMax: 5,
		},
		{
			name:    "Different band ignored",
			channel: 36,
			band:    channel.Band5GHz,
			networks: []channel.NetworkInfo{
				{Channel: 6, Frequency: 2437, Signal: -30},
			},
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.CalculateInterference(tt.channel, tt.band, tt.networks)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf(
					"CalculateInterference(%d, %q, networks) = %f, want between %f and %f",
					tt.channel,
					tt.band,
					got,
					tt.wantMin,
					tt.wantMax,
				)
			}
		})
	}
}

// TestNewAnalyzer verifies analyzer creation.
func TestNewAnalyzer(t *testing.T) {
	analyzer := channel.NewAnalyzer()
	if analyzer == nil {
		t.Fatal("NewAnalyzer() returned nil")
	}
}

// TestAnalyzerWithDFS verifies DFS configuration.
func TestAnalyzerWithDFS(t *testing.T) {
	analyzer := channel.NewAnalyzer().WithDFS(true)
	if analyzer == nil {
		t.Fatal("WithDFS() returned nil")
	}

	// Verify DFS channels are included in analysis
	networks := []channel.NetworkInfo{}
	analysis := analyzer.Analyze(networks, channel.Band5GHz)

	hasDFS := false
	for _, ch := range analysis.Channels {
		if ch.IsDFS {
			hasDFS = true
			break
		}
	}
	if !hasDFS {
		t.Error("WithDFS(true) should include DFS channels in analysis")
	}
}

// TestAnalyzerWithoutDFS verifies DFS exclusion.
func TestAnalyzerWithoutDFS(t *testing.T) {
	analyzer := channel.NewAnalyzer().WithDFS(false)

	networks := []channel.NetworkInfo{}
	analysis := analyzer.Analyze(networks, channel.Band5GHz)

	for _, ch := range analysis.Channels {
		if ch.IsDFS {
			t.Errorf("WithDFS(false) should exclude DFS channel %d", ch.Number)
		}
	}
}

// TestAnalyze verifies channel analysis.
func TestAnalyze(t *testing.T) {
	tests := []struct {
		name                   string
		networks               []channel.NetworkInfo
		band                   channel.Band
		includeDFS             bool
		wantRecommendedChannel int
		wantChannelCount       int
	}{
		{
			name:                   "Empty networks 2.4 GHz",
			networks:               nil,
			band:                   channel.Band24GHz,
			includeDFS:             false,
			wantRecommendedChannel: 1, // First available channel
			wantChannelCount:       13,
		},
		{
			name: "Single network on channel 6",
			networks: []channel.NetworkInfo{
				{SSID: "Test", BSSID: "00:11:22:33:44:55", Channel: 6, Frequency: 2437, Signal: -50},
			},
			band:                   channel.Band24GHz,
			includeDFS:             false,
			wantRecommendedChannel: 1, // Avoid channel 6
			wantChannelCount:       13,
		},
		{
			name: "Multiple networks on same channel",
			networks: []channel.NetworkInfo{
				{SSID: "Net1", BSSID: "00:11:22:33:44:55", Channel: 1, Frequency: 2412, Signal: -50},
				{SSID: "Net2", BSSID: "00:11:22:33:44:56", Channel: 1, Frequency: 2412, Signal: -60},
				{SSID: "Net3", BSSID: "00:11:22:33:44:57", Channel: 1, Frequency: 2412, Signal: -70},
			},
			band:                   channel.Band24GHz,
			includeDFS:             false,
			wantRecommendedChannel: 0, // Will verify channel is not 1-5 (adjacent to congested channel)
			wantChannelCount:       13,
		},
		{
			name:                   "5 GHz without DFS",
			networks:               nil,
			band:                   channel.Band5GHz,
			includeDFS:             false,
			wantRecommendedChannel: 36, // First non-DFS channel
			wantChannelCount:       9,  // Only non-DFS channels
		},
		{
			name:                   "5 GHz with DFS",
			networks:               nil,
			band:                   channel.Band5GHz,
			includeDFS:             true,
			wantRecommendedChannel: 36, // Still prefer non-DFS
			wantChannelCount:       25, // All channels including DFS
		},
		{
			name:             "6 GHz band",
			networks:         nil,
			band:             channel.Band6GHz,
			includeDFS:       false,
			wantChannelCount: 59, // 6 GHz has 59 channels (every 4th channel from 1-233)
		},
		{
			name:             "Unknown band",
			networks:         nil,
			band:             "",
			includeDFS:       false,
			wantChannelCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := channel.NewAnalyzer().WithDFS(tt.includeDFS)
			analysis := analyzer.Analyze(tt.networks, tt.band)

			assertAnalysisNotNil(t, analysis)
			assertAnalysisBand(t, analysis, tt.band)
			assertChannelCount(t, analysis, tt.wantChannelCount)
			assertRecommendedChannel(t, analysis, tt.wantRecommendedChannel)
			assertCongestedChannelAvoidance(t, tt.name, analysis)
			assertAnalysisTimeRecent(t, analysis)
		})
	}
}

// assertAnalysisNotNil verifies that analysis is not nil.
func assertAnalysisNotNil(t *testing.T, analysis *channel.Analysis) {
	t.Helper()
	if analysis == nil {
		t.Fatal("Analyze() returned nil")
	}
}

// assertAnalysisBand verifies the analysis band matches expected.
func assertAnalysisBand(t *testing.T, analysis *channel.Analysis, want channel.Band) {
	t.Helper()
	if analysis.Band != want {
		t.Errorf("analysis.Band = %q, want %q", analysis.Band, want)
	}
}

// assertChannelCount verifies the number of channels in the analysis.
func assertChannelCount(t *testing.T, analysis *channel.Analysis, want int) {
	t.Helper()
	if len(analysis.Channels) != want {
		t.Errorf("len(analysis.Channels) = %d, want %d", len(analysis.Channels), want)
	}
}

// assertRecommendedChannel verifies the recommended channel if specified.
func assertRecommendedChannel(t *testing.T, analysis *channel.Analysis, want int) {
	t.Helper()
	if want > 0 && analysis.RecommendedChannel != want {
		t.Errorf("analysis.RecommendedChannel = %d, want %d", analysis.RecommendedChannel, want)
	}
}

// assertCongestedChannelAvoidance verifies channel recommendation avoids congested areas.
func assertCongestedChannelAvoidance(t *testing.T, testName string, analysis *channel.Analysis) {
	t.Helper()
	if testName == "Multiple networks on same channel" {
		// Recommended channel should be far from channel 1 (>= channel 6)
		if analysis.RecommendedChannel < 6 {
			t.Errorf(
				"analysis.RecommendedChannel = %d, should be >= 6 to avoid congested channel 1",
				analysis.RecommendedChannel,
			)
		}
	}
}

// assertAnalysisTimeRecent verifies the analysis timestamp is recent.
func assertAnalysisTimeRecent(t *testing.T, analysis *channel.Analysis) {
	t.Helper()
	if time.Since(analysis.AnalyzedAt) > time.Second {
		t.Errorf("analysis.AnalyzedAt is too old: %v", analysis.AnalyzedAt)
	}
}

// TestAnalyzeChannelInfoFields verifies channel info fields are populated correctly.
func TestAnalyzeChannelInfoFields(t *testing.T) {
	networks := []channel.NetworkInfo{
		{SSID: "Test", BSSID: "00:11:22:33:44:55", Channel: 6, Frequency: 2437, Signal: -50},
	}

	analyzer := channel.NewAnalyzer()
	analysis := analyzer.Analyze(networks, channel.Band24GHz)

	// Find channel 6 info
	var ch6 *channel.ChannelInfo
	for i := range analysis.Channels {
		if analysis.Channels[i].Number == 6 {
			ch6 = &analysis.Channels[i]
			break
		}
	}

	if ch6 == nil {
		t.Fatal("Channel 6 not found in analysis")
	}

	if ch6.Number != 6 {
		t.Errorf("ChannelInfo.Number = %d, want 6", ch6.Number)
	}

	if ch6.CenterFreqMHz != 2437 {
		t.Errorf("ChannelInfo.CenterFreqMHz = %d, want 2437", ch6.CenterFreqMHz)
	}

	if ch6.Band != channel.Band24GHz {
		t.Errorf("ChannelInfo.Band = %q, want %q", ch6.Band, channel.Band24GHz)
	}

	if ch6.NetworkCount != 1 {
		t.Errorf("ChannelInfo.NetworkCount = %d, want 1", ch6.NetworkCount)
	}

	if ch6.Utilization < 10 {
		t.Errorf("ChannelInfo.Utilization = %f, want >= 10", ch6.Utilization)
	}

	if ch6.Interference <= 0 {
		t.Errorf("ChannelInfo.Interference = %f, want > 0", ch6.Interference)
	}

	if ch6.IsDFS {
		t.Error("ChannelInfo.IsDFS = true, want false for 2.4 GHz channel")
	}
}

// TestSignalToInterference verifies signal strength to interference conversion.
func TestSignalToInterference(t *testing.T) {
	tests := []struct {
		name    string
		signal  int
		wantMin float64
		wantMax float64
	}{
		{"Very strong signal (-30 dBm)", -30, 9.5, 10.5},
		{"Strong signal (-40 dBm)", -40, 8, 10},
		{"Medium signal (-60 dBm)", -60, 4, 7},
		{"Weak signal (-80 dBm)", -80, 1.5, 4},
		{"Very weak signal (-90 dBm)", -90, 0.5, 1.5},
		{"Extremely weak signal (-100 dBm)", -100, 0.5, 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.SignalToInterference(tt.signal)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf(
					"SignalToInterference(%d) = %f, want between %f and %f",
					tt.signal,
					got,
					tt.wantMin,
					tt.wantMax,
				)
			}
		})
	}
}

// TestGetChannelsForBand verifies channel list for each band.
func TestGetChannelsForBand(t *testing.T) {
	tests := []struct {
		name          string
		band          channel.Band
		wantMinCount  int
		wantFirstChan int
		wantLastChan  int
	}{
		{"2.4 GHz band", channel.Band24GHz, 13, 1, 13},
		{"5 GHz band", channel.Band5GHz, 20, 36, 165},
		{"6 GHz band", channel.Band6GHz, 50, 1, 233},
		{"Unknown band", "", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channels := channel.GetChannelsForBand(tt.band)

			if len(channels) < tt.wantMinCount {
				t.Errorf(
					"GetChannelsForBand(%q) returned %d channels, want at least %d",
					tt.band,
					len(channels),
					tt.wantMinCount,
				)
			}

			if tt.wantMinCount > 0 {
				if channels[0] != tt.wantFirstChan {
					t.Errorf(
						"GetChannelsForBand(%q) first channel = %d, want %d",
						tt.band,
						channels[0],
						tt.wantFirstChan,
					)
				}
				if channels[len(channels)-1] != tt.wantLastChan {
					t.Errorf(
						"GetChannelsForBand(%q) last channel = %d, want %d",
						tt.band,
						channels[len(channels)-1],
						tt.wantLastChan,
					)
				}
			}
		})
	}
}

// TestGroupNetworksByChannel verifies network grouping.
func TestGroupNetworksByChannel(t *testing.T) {
	networks := []channel.NetworkInfo{
		{SSID: "Net1", Channel: 1, Frequency: 2412},
		{SSID: "Net2", Channel: 1, Frequency: 2412},
		{SSID: "Net3", Channel: 6, Frequency: 2437},
		{SSID: "Net4", Channel: 36, Frequency: 5180}, // Different band
	}

	grouped := channel.GroupNetworksByChannel(networks, channel.Band24GHz)

	if len(grouped[1]) != 2 {
		t.Errorf("Channel 1 has %d networks, want 2", len(grouped[1]))
	}

	if len(grouped[6]) != 1 {
		t.Errorf("Channel 6 has %d networks, want 1", len(grouped[6]))
	}

	if len(grouped[36]) != 0 {
		t.Errorf("Channel 36 (5 GHz) should not be in 2.4 GHz grouping")
	}
}

// TestFindBestChannel verifies best channel selection.
func TestFindBestChannel(t *testing.T) {
	tests := []struct {
		name     string
		channels []channel.ChannelInfo
		want     int
	}{
		{
			name:     "Empty channels",
			channels: nil,
			want:     0,
		},
		{
			name: "Single channel",
			channels: []channel.ChannelInfo{
				{Number: 6, Interference: 50},
			},
			want: 6,
		},
		{
			name: "Choose lowest interference",
			channels: []channel.ChannelInfo{
				{Number: 1, Interference: 80},
				{Number: 6, Interference: 20},
				{Number: 11, Interference: 50},
			},
			want: 6,
		},
		{
			name: "Equal interference, choose lower network count",
			channels: []channel.ChannelInfo{
				{Number: 1, Interference: 50, NetworkCount: 3},
				{Number: 6, Interference: 50, NetworkCount: 1},
				{Number: 11, Interference: 50, NetworkCount: 2},
			},
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.FindBestChannel(tt.channels)
			if got != tt.want {
				t.Errorf("FindBestChannel() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestIsValid5GHzChannel verifies 5 GHz channel validation.
func TestIsValid5GHzChannel(t *testing.T) {
	tests := []struct {
		name    string
		channel int
		want    bool
	}{
		// UNII-1
		{"Channel 36 (UNII-1)", 36, true},
		{"Channel 40 (UNII-1)", 40, true},
		{"Channel 44 (UNII-1)", 44, true},
		{"Channel 48 (UNII-1)", 48, true},

		// UNII-2
		{"Channel 52 (UNII-2)", 52, true},
		{"Channel 64 (UNII-2)", 64, true},

		// UNII-2 Extended
		{"Channel 100 (UNII-2E)", 100, true},
		{"Channel 144 (UNII-2E)", 144, true},

		// UNII-3
		{"Channel 149 (UNII-3)", 149, true},
		{"Channel 165 (UNII-3)", 165, true},

		// Invalid channels
		{"Channel 35 (invalid)", 35, false},
		{"Channel 50 (invalid)", 50, false},
		{"Channel 66 (invalid)", 66, false},
		{"Channel 99 (invalid)", 99, false},
		{"Channel 145 (invalid)", 145, false},
		{"Channel 170 (invalid)", 170, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.IsValid5GHzChannel(tt.channel)
			if got != tt.want {
				t.Errorf("IsValid5GHzChannel(%d) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

// TestAbs verifies absolute value function.
func TestAbs(t *testing.T) {
	tests := []struct {
		name string
		x    int
		want int
	}{
		{"Positive number", 5, 5},
		{"Negative number", -5, 5},
		{"Zero", 0, 0},
		{"Large positive", 1000000, 1000000},
		{"Large negative", -1000000, 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.Abs(tt.x)
			if got != tt.want {
				t.Errorf("Abs(%d) = %d, want %d", tt.x, got, tt.want)
			}
		})
	}
}

// TestAnalyzeRecommendedChannelMarking verifies that recommended channel is marked correctly.
func TestAnalyzeRecommendedChannelMarking(t *testing.T) {
	analyzer := channel.NewAnalyzer()
	analysis := analyzer.Analyze(nil, channel.Band24GHz)

	recommendedCount := 0
	for _, ch := range analysis.Channels {
		if ch.IsRecommended {
			recommendedCount++
			if ch.Number != analysis.RecommendedChannel {
				t.Errorf(
					"Channel %d is marked recommended but RecommendedChannel is %d",
					ch.Number,
					analysis.RecommendedChannel,
				)
			}
		}
	}

	if recommendedCount != 1 {
		t.Errorf("Expected exactly 1 recommended channel, got %d", recommendedCount)
	}
}

// TestChannelWidthConstants verifies channel width constant values.
func TestChannelWidthConstants(t *testing.T) {
	if channel.Width20MHz != 20 {
		t.Errorf("Width20MHz = %d, want 20", channel.Width20MHz)
	}
	if channel.Width40MHz != 40 {
		t.Errorf("Width40MHz = %d, want 40", channel.Width40MHz)
	}
	if channel.Width80MHz != 80 {
		t.Errorf("Width80MHz = %d, want 80", channel.Width80MHz)
	}
	if channel.Width160MHz != 160 {
		t.Errorf("Width160MHz = %d, want 160", channel.Width160MHz)
	}
	if channel.Width320MHz != 320 {
		t.Errorf("Width320MHz = %d, want 320", channel.Width320MHz)
	}
}

// TestBandConstants verifies band constant values.
func TestBandConstants(t *testing.T) {
	if channel.Band24GHz != "2.4GHz" {
		t.Errorf("Band24GHz = %q, want %q", channel.Band24GHz, "2.4GHz")
	}
	if channel.Band5GHz != "5GHz" {
		t.Errorf("Band5GHz = %q, want %q", channel.Band5GHz, "5GHz")
	}
	if channel.Band6GHz != "6GHz" {
		t.Errorf("Band6GHz = %q, want %q", channel.Band6GHz, "6GHz")
	}
}
