// Package roots_test provides tests for the AnalysisService.
// Test suite validates path analysis, bottleneck detection, and scoring.
package roots_test

import (
	"context"
	"net"
	"testing"

	"github.com/krisarmstrong/seed/internal/roots"
)

// TestAnalysisService_Creation validates service creation.
func TestAnalysisService_Creation(t *testing.T) {
	t.Parallel()

	svc := roots.NewAnalysisService(nil, nil)
	if svc == nil {
		t.Fatal("NewAnalysisService() returned nil")
	}
}

// TestAnalysisService_ExportAnalyzeHops validates internal analyzeHops function.
func TestAnalysisService_ExportAnalyzeHops(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		hops              []roots.TracerouteHop
		wantTotalRTT      float64
		wantLostHops      int
		wantBottleneckCnt int
	}{
		{
			name:              "empty hops",
			hops:              []roots.TracerouteHop{},
			wantTotalRTT:      0,
			wantLostHops:      0,
			wantBottleneckCnt: 0,
		},
		{
			name: "all responding hops no bottleneck",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, RTTMs: 8.0, Lost: false},  // 1.6x ratio < 2x, 3ms < 50ms
				{Number: 3, RTTMs: 10.0, Lost: false}, // 1.25x ratio < 2x, 2ms < 50ms
			},
			wantTotalRTT:      23.0,
			wantLostHops:      0,
			wantBottleneckCnt: 0,
		},
		{
			name: "all lost hops",
			hops: []roots.TracerouteHop{
				{Number: 1, Lost: true},
				{Number: 2, Lost: true},
				{Number: 3, Lost: true},
			},
			wantTotalRTT:      0,
			wantLostHops:      3,
			wantBottleneckCnt: 0,
		},
		{
			name: "mixed with bottleneck from absolute increase",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, RTTMs: 60.0, Lost: false}, // 55ms > 50ms threshold
				{Number: 3, RTTMs: 65.0, Lost: false},
			},
			wantTotalRTT:      130.0,
			wantLostHops:      0,
			wantBottleneckCnt: 1,
		},
		{
			name: "mixed with bottleneck from ratio",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 10.0, Lost: false},
				{Number: 2, RTTMs: 25.0, Lost: false}, // 2.5x ratio > 2x threshold
				{Number: 3, RTTMs: 30.0, Lost: false},
			},
			wantTotalRTT:      65.0,
			wantLostHops:      0,
			wantBottleneckCnt: 1,
		},
		{
			name: "lost hop between responding hops",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, Lost: true},
				{Number: 3, RTTMs: 8.0, Lost: false}, // 1.6x ratio < 2x, but comparing to hop 1
			},
			wantTotalRTT:      13.0,
			wantLostHops:      1,
			wantBottleneckCnt: 0, // No bottleneck because 8/5 = 1.6x < 2x
		},
		{
			name: "lost hop with subsequent bottleneck",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, Lost: true},
				{Number: 3, RTTMs: 20.0, Lost: false}, // 4x ratio > 2x threshold
			},
			wantTotalRTT:      25.0,
			wantLostHops:      1,
			wantBottleneckCnt: 1,
		},
		{
			name: "multiple bottlenecks",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 1.0, Lost: false},
				{Number: 2, RTTMs: 60.0, Lost: false},  // Bottleneck (>50ms increase)
				{Number: 3, RTTMs: 70.0, Lost: false},  // No bottleneck (10ms increase)
				{Number: 4, RTTMs: 200.0, Lost: false}, // Bottleneck (>50ms increase)
			},
			wantTotalRTT:      331.0,
			wantLostHops:      0,
			wantBottleneckCnt: 2,
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bottlenecks := make([]roots.PathBottleneck, 0)
			totalRTT, lostHops := svc.ExportAnalyzeHops(tt.hops, &bottlenecks)

			if totalRTT != tt.wantTotalRTT {
				t.Errorf("totalRTT = %f, want %f", totalRTT, tt.wantTotalRTT)
			}
			if lostHops != tt.wantLostHops {
				t.Errorf("lostHops = %d, want %d", lostHops, tt.wantLostHops)
			}
			if len(bottlenecks) != tt.wantBottleneckCnt {
				t.Errorf("bottleneck count = %d, want %d", len(bottlenecks), tt.wantBottleneckCnt)
			}
		})
	}
}

// TestAnalysisService_ExportDetectBottleneck validates internal detectBottleneck function.
func TestAnalysisService_ExportDetectBottleneck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		hopIndex    int
		hop         roots.TracerouteHop
		previousRTT float64
		currentRTT  float64
		wantNil     bool
	}{
		{
			name:        "first hop never bottleneck",
			hopIndex:    0,
			hop:         roots.TracerouteHop{Number: 1, RTTMs: 100.0},
			previousRTT: 0,
			currentRTT:  100.0,
			wantNil:     true,
		},
		{
			name:        "zero previous RTT",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: 100.0},
			previousRTT: 0,
			currentRTT:  100.0,
			wantNil:     true,
		},
		{
			name:        "zero current RTT",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: 0},
			previousRTT: 10.0,
			currentRTT:  0,
			wantNil:     true,
		},
		{
			name:        "negative previous RTT",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: 50.0},
			previousRTT: -1.0,
			currentRTT:  50.0,
			wantNil:     true,
		},
		{
			name:        "negative current RTT",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: -1.0},
			previousRTT: 10.0,
			currentRTT:  -1.0,
			wantNil:     true,
		},
		{
			name:        "exactly 2x ratio not bottleneck",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: 20.0},
			previousRTT: 10.0,
			currentRTT:  20.0,
			wantNil:     true, // 2x ratio == threshold, need > 2x
		},
		{
			name:        "exactly 50ms increase not bottleneck alone",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: 100.0},
			previousRTT: 50.0,
			currentRTT:  100.0,
			wantNil:     true, // 50ms == threshold, need > 50ms, and ratio is 2x not > 2x
		},
		{
			name:        "just above 2x ratio is bottleneck",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: 20.1},
			previousRTT: 10.0,
			currentRTT:  20.1,
			wantNil:     false,
		},
		{
			name:        "just above 50ms is bottleneck",
			hopIndex:    1,
			hop:         roots.TracerouteHop{Number: 2, RTTMs: 60.1},
			previousRTT: 10.0,
			currentRTT:  60.1,
			wantNil:     false,
		},
		{
			name:        "bottleneck with IP address",
			hopIndex:    2,
			hop:         roots.TracerouteHop{Number: 3, Address: net.ParseIP("192.168.1.1"), RTTMs: 70.0},
			previousRTT: 10.0,
			currentRTT:  70.0,
			wantNil:     false,
		},
		{
			name:        "bottleneck without IP address",
			hopIndex:    2,
			hop:         roots.TracerouteHop{Number: 3, Address: nil, RTTMs: 70.0},
			previousRTT: 10.0,
			currentRTT:  70.0,
			wantNil:     false,
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := svc.ExportDetectBottleneck(tt.hopIndex, tt.hop, tt.previousRTT, tt.currentRTT)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Error("expected bottleneck, got nil")
				return
			}

			if result.HopNumber != tt.hop.Number {
				t.Errorf("HopNumber = %d, want %d", result.HopNumber, tt.hop.Number)
			}

			expectedIncrease := tt.currentRTT - tt.previousRTT
			if result.RTTIncrease != expectedIncrease {
				t.Errorf("RTTIncrease = %f, want %f", result.RTTIncrease, expectedIncrease)
			}

			if result.Reason == "" {
				t.Error("Reason should not be empty")
			}
		})
	}
}

// TestAnalysisService_ExportCalculateScore validates internal calculateScore function.
func TestAnalysisService_ExportCalculateScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		analysis *roots.PathAnalysis
		want     int
	}{
		{
			name: "perfect score",
			analysis: &roots.PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  10.0,
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 100,
		},
		{
			name: "10% packet loss",
			analysis: &roots.PathAnalysis{
				PacketLoss:  10.0,
				AverageRTT:  10.0,
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 90,
		},
		{
			name: "50% packet loss",
			analysis: &roots.PathAnalysis{
				PacketLoss:  50.0,
				AverageRTT:  10.0,
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 50,
		},
		{
			name: "100% packet loss",
			analysis: &roots.PathAnalysis{
				PacketLoss:  100.0,
				AverageRTT:  0,
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 0,
		},
		{
			name: "high RTT exactly at threshold",
			analysis: &roots.PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  100.0, // At threshold
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 100, // No penalty at exactly 100ms
		},
		{
			name: "high RTT above threshold",
			analysis: &roots.PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  150.0, // 50ms above threshold
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 95, // 100 - (50/10) = 95
		},
		{
			name: "very high RTT",
			analysis: &roots.PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  200.0, // 100ms above threshold
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 90, // 100 - (100/10) = 90
		},
		{
			name: "one bottleneck",
			analysis: &roots.PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  10.0,
				Bottlenecks: []roots.PathBottleneck{{HopNumber: 1}},
			},
			want: 95, // 100 - 5 = 95
		},
		{
			name: "three bottlenecks",
			analysis: &roots.PathAnalysis{
				PacketLoss: 0,
				AverageRTT: 10.0,
				Bottlenecks: []roots.PathBottleneck{
					{HopNumber: 1},
					{HopNumber: 2},
					{HopNumber: 3},
				},
			},
			want: 85, // 100 - 15 = 85
		},
		{
			name: "combined penalties",
			analysis: &roots.PathAnalysis{
				PacketLoss:  20.0,                                   // -20
				AverageRTT:  150.0,                                  // -5
				Bottlenecks: []roots.PathBottleneck{{HopNumber: 1}}, // -5
			},
			want: 70, // 100 - 20 - 5 - 5 = 70
		},
		{
			name: "score clamped at 0",
			analysis: &roots.PathAnalysis{
				PacketLoss: 80.0,
				AverageRTT: 500.0, // 400ms above threshold = -40
				Bottlenecks: []roots.PathBottleneck{
					{HopNumber: 1},
					{HopNumber: 2},
					{HopNumber: 3},
				},
			},
			want: 0, // Would be negative but clamped
		},
		{
			name: "score clamped at 100",
			analysis: &roots.PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  1.0,
				Bottlenecks: []roots.PathBottleneck{},
			},
			want: 100,
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := svc.ExportCalculateScore(tt.analysis)
			if got != tt.want {
				t.Errorf("calculateScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestScoreToDescription_AllBoundaries validates all score boundaries.
func TestScoreToDescription_AllBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		score       int
		wantContain string
	}{
		{name: "score 100", score: 100, wantContain: "Excellent"},
		{name: "score 95", score: 95, wantContain: "Excellent"},
		{name: "score 90", score: 90, wantContain: "Excellent"},
		{name: "score 89", score: 89, wantContain: "Good"},
		{name: "score 80", score: 80, wantContain: "Good"},
		{name: "score 70", score: 70, wantContain: "Good"},
		{name: "score 69", score: 69, wantContain: "Fair"},
		{name: "score 60", score: 60, wantContain: "Fair"},
		{name: "score 50", score: 50, wantContain: "Fair"},
		{name: "score 49", score: 49, wantContain: "Poor"},
		{name: "score 40", score: 40, wantContain: "Poor"},
		{name: "score 30", score: 30, wantContain: "Poor"},
		{name: "score 29", score: 29, wantContain: "Very poor"},
		{name: "score 20", score: 20, wantContain: "Very poor"},
		{name: "score 10", score: 10, wantContain: "Very poor"},
		{name: "score 0", score: 0, wantContain: "Very poor"},
		{name: "negative score", score: -10, wantContain: "Very poor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			desc := roots.ExportScoreToDescription(tt.score)
			if desc == "" {
				t.Error("description should not be empty")
			}
			if !containsString(desc, tt.wantContain) {
				t.Errorf("description = %q, want to contain %q", desc, tt.wantContain)
			}
		})
	}
}

// TestAnalysisService_AnalyzePath_Comprehensive validates full path analysis.
func TestAnalysisService_AnalyzePath_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		result         *roots.TracerouteResult
		wantErr        bool
		wantScoreMin   int
		wantScoreMax   int
		wantHops       int
		wantLossMin    float64
		wantLossMax    float64
		wantAvgRTTMin  float64
		wantAvgRTTMax  float64
		wantBottleneck int
	}{
		{
			name:    "nil result",
			result:  nil,
			wantErr: true,
		},
		{
			name: "empty hops",
			result: &roots.TracerouteResult{
				Target: "test",
				Hops:   []roots.TracerouteHop{},
			},
			wantErr:       false,
			wantScoreMin:  100,
			wantScoreMax:  100,
			wantHops:      0,
			wantLossMin:   0,
			wantLossMax:   0,
			wantAvgRTTMin: 0,
			wantAvgRTTMax: 0,
		},
		{
			name: "single responding hop",
			result: &roots.TracerouteResult{
				Target: "test",
				Hops: []roots.TracerouteHop{
					{Number: 1, RTTMs: 5.0, Lost: false},
				},
			},
			wantErr:       false,
			wantScoreMin:  95,
			wantScoreMax:  100,
			wantHops:      1,
			wantLossMin:   0,
			wantLossMax:   0,
			wantAvgRTTMin: 4.9,
			wantAvgRTTMax: 5.1,
		},
		{
			name: "single lost hop",
			result: &roots.TracerouteResult{
				Target: "test",
				Hops: []roots.TracerouteHop{
					{Number: 1, Lost: true},
				},
			},
			wantErr:      false,
			wantScoreMin: 0,
			wantScoreMax: 5, // 100% loss = -100
			wantHops:     1,
			wantLossMin:  99.9,
			wantLossMax:  100.1,
		},
		{
			name: "path with bottleneck and loss",
			result: &roots.TracerouteResult{
				Target: "test",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("10.0.0.1"), RTTMs: 5.0, Lost: false},
					{Number: 2, Lost: true},
					{Number: 3, Address: net.ParseIP("10.0.0.2"), RTTMs: 100.0, Lost: false}, // Bottleneck
					{Number: 4, Address: net.ParseIP("8.8.8.8"), RTTMs: 105.0, Lost: false},
				},
			},
			wantErr:        false,
			wantScoreMin:   60,
			wantScoreMax:   80,
			wantHops:       4,
			wantLossMin:    24.9,
			wantLossMax:    25.1,
			wantBottleneck: 1,
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			analysis, err := svc.AnalyzePath(context.Background(), tt.result)

			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzePath() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if analysis == nil {
				t.Fatal("analysis should not be nil")
			}

			if analysis.Hops != tt.wantHops {
				t.Errorf("Hops = %d, want %d", analysis.Hops, tt.wantHops)
			}

			if analysis.Score < tt.wantScoreMin || analysis.Score > tt.wantScoreMax {
				t.Errorf("Score = %d, want between %d and %d",
					analysis.Score, tt.wantScoreMin, tt.wantScoreMax)
			}

			if tt.wantLossMax > 0 {
				if analysis.PacketLoss < tt.wantLossMin || analysis.PacketLoss > tt.wantLossMax {
					t.Errorf("PacketLoss = %.2f%%, want between %.2f%% and %.2f%%",
						analysis.PacketLoss, tt.wantLossMin, tt.wantLossMax)
				}
			}

			if tt.wantAvgRTTMax > 0 {
				if analysis.AverageRTT < tt.wantAvgRTTMin || analysis.AverageRTT > tt.wantAvgRTTMax {
					t.Errorf("AverageRTT = %.2f, want between %.2f and %.2f",
						analysis.AverageRTT, tt.wantAvgRTTMin, tt.wantAvgRTTMax)
				}
			}

			if tt.wantBottleneck > 0 {
				if len(analysis.Bottlenecks) < tt.wantBottleneck {
					t.Errorf("Bottlenecks count = %d, want at least %d",
						len(analysis.Bottlenecks), tt.wantBottleneck)
				}
			}

			if analysis.Analysis == "" {
				t.Error("Analysis text should not be empty")
			}
		})
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
