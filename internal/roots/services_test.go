package roots_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots"
)

func TestNewAnalysisService(t *testing.T) {
	svc := roots.NewAnalysisService(nil, nil)
	if svc == nil {
		t.Fatal("NewAnalysisService() returned nil")
	}
}

func TestAnalysisService_AnalyzePath(t *testing.T) {
	tests := []analyzePathTestCase{
		{
			name:    "nil result returns error",
			result:  nil,
			wantErr: true,
		},
		{
			name: "empty hops list",
			result: &roots.TracerouteResult{
				Target:   "8.8.8.8",
				Hops:     []roots.TracerouteHop{},
				Complete: true,
			},
			wantErr:        false,
			wantHops:       0,
			wantScoreMin:   100,
			wantScoreMax:   100,
			wantAnalysisOK: true,
		},
		{
			name: "single hop with low latency",
			result: &roots.TracerouteResult{
				Target: "192.168.1.1",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 1.0, Lost: false},
				},
				Complete: true,
			},
			wantErr:        false,
			wantHops:       1,
			wantScoreMin:   90,
			wantScoreMax:   100,
			wantAnalysisOK: true,
		},
		{
			name: "multiple hops with increasing latency",
			result: &roots.TracerouteResult{
				Target: "8.8.8.8",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 1.0, Lost: false},
					{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 5.0, Lost: false},
					{Number: 3, Address: net.ParseIP("172.16.0.1"), RTTMs: 10.0, Lost: false},
					{Number: 4, Address: net.ParseIP("8.8.8.8"), RTTMs: 15.0, Lost: false},
				},
				Complete: true,
			},
			wantErr:        false,
			wantHops:       4,
			wantScoreMin:   90,
			wantScoreMax:   100,
			wantAnalysisOK: true,
		},
		{
			name: "path with packet loss",
			result: &roots.TracerouteResult{
				Target: "8.8.8.8",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 1.0, Lost: false},
					{Number: 2, Address: nil, RTTMs: 0, Lost: true},
					{Number: 3, Address: net.ParseIP("10.0.0.1"), RTTMs: 10.0, Lost: false},
					{Number: 4, Address: nil, RTTMs: 0, Lost: true},
					{Number: 5, Address: net.ParseIP("8.8.8.8"), RTTMs: 20.0, Lost: false},
				},
				Complete: true,
			},
			wantErr:        false,
			wantHops:       5,
			wantScoreMin:   50, // 40% packet loss = -40 points
			wantScoreMax:   65,
			wantAnalysisOK: true,
		},
		{
			name: "path with bottleneck",
			result: &roots.TracerouteResult{
				Target: "8.8.8.8",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 1.0, Lost: false},
					{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 5.0, Lost: false},
					{Number: 3, Address: net.ParseIP("172.16.0.1"), RTTMs: 100.0, Lost: false}, // Bottleneck
					{Number: 4, Address: net.ParseIP("8.8.8.8"), RTTMs: 110.0, Lost: false},
				},
				Complete: true,
			},
			wantErr:        false,
			wantHops:       4,
			wantScoreMin:   80, // May have bottleneck penalty
			wantScoreMax:   100,
			wantAnalysisOK: true,
		},
		{
			name: "path with high latency",
			result: &roots.TracerouteResult{
				Target: "example.com",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 50.0, Lost: false},
					{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 100.0, Lost: false},
					{Number: 3, Address: net.ParseIP("172.16.0.1"), RTTMs: 150.0, Lost: false},
					{Number: 4, Address: net.ParseIP("93.184.216.34"), RTTMs: 200.0, Lost: false},
				},
				Complete: true,
			},
			wantErr:        false,
			wantHops:       4,
			wantScoreMin:   70, // High RTT penalty
			wantScoreMax:   100,
			wantAnalysisOK: true,
		},
		{
			name: "all hops lost",
			result: &roots.TracerouteResult{
				Target: "unreachable.example",
				Hops: []roots.TracerouteHop{
					{Number: 1, Lost: true},
					{Number: 2, Lost: true},
					{Number: 3, Lost: true},
				},
				Complete: false,
			},
			wantErr:        false,
			wantHops:       3,
			wantScoreMin:   0,
			wantScoreMax:   10, // 100% packet loss
			wantAnalysisOK: true,
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := svc.AnalyzePath(context.Background(), tt.result)
			assertAnalyzePathResult(t, tt, analysis, err)
		})
	}
}

type analyzePathTestCase struct {
	name           string
	result         *roots.TracerouteResult
	wantErr        bool
	wantHops       int
	wantScoreMin   int // minimum acceptable score
	wantScoreMax   int // maximum acceptable score
	wantAnalysisOK bool
}

func assertAnalyzePathResult(t *testing.T, tc analyzePathTestCase, analysis *roots.PathAnalysis, err error) {
	t.Helper()

	if (err != nil) != tc.wantErr {
		t.Errorf("AnalyzePath() error = %v, wantErr %v", err, tc.wantErr)
		return
	}
	if tc.wantErr {
		return
	}
	if analysis == nil {
		t.Fatal("AnalyzePath() returned nil analysis")
	}
	if analysis.Hops != tc.wantHops {
		t.Errorf("Hops = %d, want %d", analysis.Hops, tc.wantHops)
	}
	if analysis.Score < tc.wantScoreMin || analysis.Score > tc.wantScoreMax {
		t.Errorf(
			"Score = %d, want between %d and %d",
			analysis.Score,
			tc.wantScoreMin,
			tc.wantScoreMax,
		)
	}
	if tc.wantAnalysisOK && analysis.Analysis == "" {
		t.Error("Analysis text should not be empty")
	}
}

func TestAnalysisService_AnalyzePath_PacketLoss(t *testing.T) {
	tests := []struct {
		name          string
		hops          []roots.TracerouteHop
		wantLossMin   float64
		wantLossMax   float64
		wantAvgRTTMin float64
		wantAvgRTTMax float64
	}{
		{
			name: "no packet loss",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 10.0, Lost: false},
				{Number: 2, RTTMs: 20.0, Lost: false},
				{Number: 3, RTTMs: 30.0, Lost: false},
			},
			wantLossMin:   0,
			wantLossMax:   0,
			wantAvgRTTMin: 19.9,
			wantAvgRTTMax: 20.1,
		},
		{
			name: "50% packet loss",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 10.0, Lost: false},
				{Number: 2, Lost: true},
				{Number: 3, RTTMs: 20.0, Lost: false},
				{Number: 4, Lost: true},
			},
			wantLossMin:   49.9,
			wantLossMax:   50.1,
			wantAvgRTTMin: 14.9,
			wantAvgRTTMax: 15.1,
		},
		{
			name: "single responding hop",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 50.0, Lost: false},
				{Number: 2, Lost: true},
				{Number: 3, Lost: true},
			},
			wantLossMin:   66.0,
			wantLossMax:   67.0,
			wantAvgRTTMin: 49.9,
			wantAvgRTTMax: 50.1,
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &roots.TracerouteResult{
				Target: "test",
				Hops:   tt.hops,
			}

			analysis, err := svc.AnalyzePath(context.Background(), result)
			if err != nil {
				t.Fatalf("AnalyzePath() error = %v", err)
			}

			if analysis.PacketLoss < tt.wantLossMin || analysis.PacketLoss > tt.wantLossMax {
				t.Errorf(
					"PacketLoss = %.2f%%, want between %.2f%% and %.2f%%",
					analysis.PacketLoss,
					tt.wantLossMin,
					tt.wantLossMax,
				)
			}

			if analysis.AverageRTT < tt.wantAvgRTTMin || analysis.AverageRTT > tt.wantAvgRTTMax {
				t.Errorf(
					"AverageRTT = %.2f ms, want between %.2f and %.2f",
					analysis.AverageRTT,
					tt.wantAvgRTTMin,
					tt.wantAvgRTTMax,
				)
			}
		})
	}
}

func TestAnalysisService_AnalyzePath_BottleneckDetection(t *testing.T) {
	tests := []struct {
		name               string
		hops               []roots.TracerouteHop
		wantBottleneckHops []int // Expected hop numbers with bottlenecks
	}{
		{
			name: "no bottleneck with gradual increase",
			hops: []roots.TracerouteHop{
				// Use RTT values where each hop is less than 2x previous and increase is <50ms
				{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 40.0, Lost: false},
				{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 55.0, Lost: false},   // 1.37x, +15ms
				{Number: 3, Address: net.ParseIP("172.16.0.1"), RTTMs: 70.0, Lost: false}, // 1.27x, +15ms
			},
			wantBottleneckHops: nil,
		},
		{
			name: "bottleneck with large RTT jump",
			hops: []roots.TracerouteHop{
				{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 5.0, Lost: false},
				{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 10.0, Lost: false},
				{Number: 3, Address: net.ParseIP("172.16.0.1"), RTTMs: 100.0, Lost: false}, // >50ms jump
			},
			wantBottleneckHops: []int{3},
		},
		{
			name: "bottleneck with RTT doubling",
			hops: []roots.TracerouteHop{
				{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 30.0, Lost: false},
				{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 70.0, Lost: false}, // >2x previous
			},
			wantBottleneckHops: []int{2},
		},
		{
			name: "multiple bottlenecks",
			hops: []roots.TracerouteHop{
				{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 1.0, Lost: false},
				{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 100.0, Lost: false}, // Bottleneck
				{Number: 3, Address: net.ParseIP("172.16.0.1"), RTTMs: 110.0, Lost: false},
				{Number: 4, Address: net.ParseIP("8.8.8.8"), RTTMs: 300.0, Lost: false}, // Bottleneck
			},
			wantBottleneckHops: []int{2, 4},
		},
		{
			name: "bottleneck skips lost hops",
			hops: []roots.TracerouteHop{
				{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 5.0, Lost: false},
				{Number: 2, Lost: true},
				{Number: 3, Address: net.ParseIP("10.0.0.1"), RTTMs: 100.0, Lost: false}, // Bottleneck
			},
			wantBottleneckHops: []int{3},
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &roots.TracerouteResult{
				Target: "test",
				Hops:   tt.hops,
			}

			analysis, err := svc.AnalyzePath(context.Background(), result)
			if err != nil {
				t.Fatalf("AnalyzePath() error = %v", err)
			}

			if len(analysis.Bottlenecks) != len(tt.wantBottleneckHops) {
				t.Errorf(
					"got %d bottlenecks, want %d",
					len(analysis.Bottlenecks),
					len(tt.wantBottleneckHops),
				)
				for i, b := range analysis.Bottlenecks {
					t.Logf("  bottleneck[%d]: hop %d, increase %.2f ms", i, b.HopNumber, b.RTTIncrease)
				}
				return
			}

			for i, wantHop := range tt.wantBottleneckHops {
				if analysis.Bottlenecks[i].HopNumber != wantHop {
					t.Errorf(
						"bottleneck[%d].HopNumber = %d, want %d",
						i,
						analysis.Bottlenecks[i].HopNumber,
						wantHop,
					)
				}
			}
		})
	}
}

func TestAnalysisService_AnalyzePath_ScoreDescriptions(t *testing.T) {
	tests := []struct {
		name            string
		hops            []roots.TracerouteHop
		wantDescription string
	}{
		{
			name: "excellent path",
			hops: []roots.TracerouteHop{
				{Number: 1, RTTMs: 1.0, Lost: false},
				{Number: 2, RTTMs: 2.0, Lost: false},
			},
			wantDescription: "Excellent path quality with low latency and no packet loss.",
		},
		{
			name: "very poor path with total packet loss",
			hops: []roots.TracerouteHop{
				{Number: 1, Lost: true},
				{Number: 2, Lost: true},
				{Number: 3, Lost: true},
				{Number: 4, Lost: true},
				{Number: 5, Lost: true},
			},
			wantDescription: "Very poor path quality. Consider using an alternative route.",
		},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &roots.TracerouteResult{
				Target: "test",
				Hops:   tt.hops,
			}

			analysis, err := svc.AnalyzePath(context.Background(), result)
			if err != nil {
				t.Fatalf("AnalyzePath() error = %v", err)
			}

			if analysis.Analysis != tt.wantDescription {
				t.Errorf("Analysis = %q, want %q", analysis.Analysis, tt.wantDescription)
			}
		})
	}
}

func TestScoreToDescription(t *testing.T) {
	tests := []struct {
		score       int
		wantContain string
	}{
		{score: 100, wantContain: "Excellent"},
		{score: 95, wantContain: "Excellent"},
		{score: 90, wantContain: "Excellent"},
		{score: 89, wantContain: "Good"},
		{score: 75, wantContain: "Good"},
		{score: 70, wantContain: "Good"},
		{score: 69, wantContain: "Fair"},
		{score: 55, wantContain: "Fair"},
		{score: 50, wantContain: "Fair"},
		{score: 49, wantContain: "Poor"},
		{score: 35, wantContain: "Poor"},
		{score: 30, wantContain: "Poor"},
		{score: 29, wantContain: "Very poor"},
		{score: 10, wantContain: "Very poor"},
		{score: 0, wantContain: "Very poor"},
	}

	svc := roots.NewAnalysisService(nil, nil)

	for _, tt := range tests {
		t.Run("score_"+string(rune('0'+tt.score/10)), func(t *testing.T) {
			// Create a path that roughly achieves this score
			hops := createHopsForScore(tt.score)

			result := &roots.TracerouteResult{
				Target: "test",
				Hops:   hops,
			}

			analysis, err := svc.AnalyzePath(context.Background(), result)
			if err != nil {
				t.Fatalf("AnalyzePath() error = %v", err)
			}

			// Verify description matches expected pattern
			desc := roots.ExportScoreToDescription(analysis.Score)
			if desc == "" {
				t.Error("scoreToDescription returned empty string")
			}
		})
	}
}

// createHopsForScore creates a hop list that should approximately achieve the target score.
func createHopsForScore(targetScore int) []roots.TracerouteHop {
	if targetScore >= 90 {
		// Excellent: low latency, no packet loss
		return []roots.TracerouteHop{
			{Number: 1, RTTMs: 1.0, Lost: false},
			{Number: 2, RTTMs: 2.0, Lost: false},
		}
	}

	if targetScore >= 70 {
		// Good: moderate latency
		return []roots.TracerouteHop{
			{Number: 1, RTTMs: 50.0, Lost: false},
			{Number: 2, RTTMs: 80.0, Lost: false},
			{Number: 3, RTTMs: 100.0, Lost: false},
		}
	}

	if targetScore >= 50 {
		// Fair: some packet loss or high latency
		return []roots.TracerouteHop{
			{Number: 1, RTTMs: 50.0, Lost: false},
			{Number: 2, Lost: true},
			{Number: 3, RTTMs: 100.0, Lost: false},
			{Number: 4, Lost: true},
		}
	}

	if targetScore >= 30 {
		// Poor: significant packet loss
		return []roots.TracerouteHop{
			{Number: 1, RTTMs: 100.0, Lost: false},
			{Number: 2, Lost: true},
			{Number: 3, Lost: true},
			{Number: 4, RTTMs: 200.0, Lost: false},
		}
	}

	// Very poor: mostly packet loss
	return []roots.TracerouteHop{
		{Number: 1, Lost: true},
		{Number: 2, Lost: true},
		{Number: 3, Lost: true},
		{Number: 4, Lost: true},
		{Number: 5, RTTMs: 500.0, Lost: false},
	}
}

func TestPathAnalysis_Fields(t *testing.T) {
	analysis := roots.PathAnalysis{
		Target:         "8.8.8.8",
		Hops:           10,
		AverageRTT:     25.5,
		PacketLoss:     5.0,
		ASNTransitions: 3,
		Bottlenecks: []roots.PathBottleneck{
			{HopNumber: 5, Address: "10.0.0.1", RTTIncrease: 50.0, Reason: "Test"},
		},
		Analysis: "Good path quality",
		Score:    85,
	}

	if analysis.Target != "8.8.8.8" {
		t.Errorf("Target = %q, want %q", analysis.Target, "8.8.8.8")
	}
	if analysis.Hops != 10 {
		t.Errorf("Hops = %d, want %d", analysis.Hops, 10)
	}
	if analysis.AverageRTT != 25.5 {
		t.Errorf("AverageRTT = %f, want %f", analysis.AverageRTT, 25.5)
	}
	if analysis.PacketLoss != 5.0 {
		t.Errorf("PacketLoss = %f, want %f", analysis.PacketLoss, 5.0)
	}
	if analysis.Analysis != "Good path quality" {
		t.Errorf("Analysis = %q, want %q", analysis.Analysis, "Good path quality")
	}
	if analysis.ASNTransitions != 3 {
		t.Errorf("ASNTransitions = %d, want %d", analysis.ASNTransitions, 3)
	}
	if len(analysis.Bottlenecks) != 1 {
		t.Errorf("len(Bottlenecks) = %d, want %d", len(analysis.Bottlenecks), 1)
	}
	if analysis.Score != 85 {
		t.Errorf("Score = %d, want %d", analysis.Score, 85)
	}
}

func TestPathBottleneck_Fields(t *testing.T) {
	bottleneck := roots.PathBottleneck{
		HopNumber:   7,
		Address:     "203.0.113.1",
		RTTIncrease: 75.5,
		Reason:      "Significant latency increase",
	}

	if bottleneck.HopNumber != 7 {
		t.Errorf("HopNumber = %d, want %d", bottleneck.HopNumber, 7)
	}
	if bottleneck.Address != "203.0.113.1" {
		t.Errorf("Address = %q, want %q", bottleneck.Address, "203.0.113.1")
	}
	if bottleneck.RTTIncrease != 75.5 {
		t.Errorf("RTTIncrease = %f, want %f", bottleneck.RTTIncrease, 75.5)
	}
	if bottleneck.Reason != "Significant latency increase" {
		t.Errorf("Reason = %q, want %q", bottleneck.Reason, "Significant latency increase")
	}
}

func TestTracerouteHop_Fields(t *testing.T) {
	hop := roots.TracerouteHop{
		Number:    5,
		Address:   net.ParseIP("192.0.2.1"),
		Hostname:  "router.example.com",
		RTT:       25 * time.Millisecond,
		RTTMs:     25.0,
		Lost:      false,
		ASN:       15169,
		ASName:    "GOOGLE",
		GeoCity:   "Mountain View",
		GeoRegion: "California",
		ISP:       "Google LLC",
	}

	if hop.Number != 5 {
		t.Errorf("Number = %d, want %d", hop.Number, 5)
	}
	if hop.Address.String() != "192.0.2.1" {
		t.Errorf("Address = %s, want %s", hop.Address, "192.0.2.1")
	}
	if hop.Hostname != "router.example.com" {
		t.Errorf("Hostname = %q, want %q", hop.Hostname, "router.example.com")
	}
	if hop.RTT != 25*time.Millisecond {
		t.Errorf("RTT = %v, want %v", hop.RTT, 25*time.Millisecond)
	}
	if hop.RTTMs != 25.0 {
		t.Errorf("RTTMs = %f, want %f", hop.RTTMs, 25.0)
	}
	if hop.Lost {
		t.Error("Lost = true, want false")
	}
	if hop.ASN != 15169 {
		t.Errorf("ASN = %d, want %d", hop.ASN, 15169)
	}
	if hop.ASName != "GOOGLE" {
		t.Errorf("ASName = %q, want %q", hop.ASName, "GOOGLE")
	}
	if hop.GeoCity != "Mountain View" {
		t.Errorf("GeoCity = %q, want %q", hop.GeoCity, "Mountain View")
	}
	if hop.GeoRegion != "California" {
		t.Errorf("GeoRegion = %q, want %q", hop.GeoRegion, "California")
	}
	if hop.ISP != "Google LLC" {
		t.Errorf("ISP = %q, want %q", hop.ISP, "Google LLC")
	}
}

func TestTracerouteResult_Fields(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Second)

	result := roots.TracerouteResult{
		Target:     "google.com",
		ResolvedIP: "142.250.80.46",
		Hops: []roots.TracerouteHop{
			{Number: 1, RTTMs: 1.0},
			{Number: 2, RTTMs: 5.0},
		},
		Complete:    true,
		Duration:    5 * time.Second,
		DurationMs:  5000.0,
		StartedAt:   now,
		CompletedAt: later,
	}

	if result.Target != "google.com" {
		t.Errorf("Target = %q, want %q", result.Target, "google.com")
	}
	if result.ResolvedIP != "142.250.80.46" {
		t.Errorf("ResolvedIP = %q, want %q", result.ResolvedIP, "142.250.80.46")
	}
	if len(result.Hops) != 2 {
		t.Errorf("len(Hops) = %d, want %d", len(result.Hops), 2)
	}
	if !result.Complete {
		t.Error("Complete = false, want true")
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want %v", result.Duration, 5*time.Second)
	}
	if result.DurationMs != 5000.0 {
		t.Errorf("DurationMs = %f, want %f", result.DurationMs, 5000.0)
	}
	if !result.StartedAt.Equal(now) {
		t.Errorf("StartedAt = %v, want %v", result.StartedAt, now)
	}
	if !result.CompletedAt.Equal(later) {
		t.Errorf("CompletedAt = %v, want %v", result.CompletedAt, later)
	}
}

func TestTracerouteOptions_Fields(t *testing.T) {
	opts := roots.TracerouteOptions{
		MaxHops:     30,
		Timeout:     2 * time.Second,
		Probes:      3,
		PacketSize:  60,
		EnrichHops:  true,
		UseUDP:      false,
		SourceAddr:  "192.168.1.100",
		DontResolve: false,
	}

	if opts.MaxHops != 30 {
		t.Errorf("MaxHops = %d, want %d", opts.MaxHops, 30)
	}
	if opts.Timeout != 2*time.Second {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 2*time.Second)
	}
	if opts.Probes != 3 {
		t.Errorf("Probes = %d, want %d", opts.Probes, 3)
	}
	if opts.PacketSize != 60 {
		t.Errorf("PacketSize = %d, want %d", opts.PacketSize, 60)
	}
	if !opts.EnrichHops {
		t.Error("EnrichHops = false, want true")
	}
	if opts.UseUDP {
		t.Error("UseUDP = true, want false")
	}
	if opts.SourceAddr != "192.168.1.100" {
		t.Errorf("SourceAddr = %q, want %q", opts.SourceAddr, "192.168.1.100")
	}
	if opts.DontResolve {
		t.Error("DontResolve = true, want false")
	}
}

func TestRootsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrNotImplemented",
			err:  roots.ErrNotImplemented,
			want: "not implemented: pending migration",
		},
		{
			name: "ErrNotInitialized",
			err:  roots.ErrNotInitialized,
			want: "service not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("error = %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}

// ============================================================================
// Enrichment Service Tests
// ============================================================================

func TestNewEnrichmentService(t *testing.T) {
	t.Parallel()

	svc := roots.NewEnrichmentService(nil)
	if svc == nil {
		t.Fatal("NewEnrichmentService() returned nil")
	}

	if svc.Checker() == nil {
		t.Error("Checker() should not be nil after initialization")
	}
}

func TestEnrichmentService_GetPublicIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupSvc   func() *roots.EnrichmentService
		wantErr    bool
		errContain string
	}{
		{
			name: "nil checker returns error",
			setupSvc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentServiceWithChecker(nil, nil)
			},
			wantErr:    true,
			errContain: "not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.setupSvc()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result, err := svc.GetPublicIP(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errContain != "" && !containsSubstring(err.Error(), tt.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContain)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("result should not be nil on success")
			}
		})
	}
}

func TestEnrichmentService_Enrich(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupSvc   func() *roots.EnrichmentService
		ip         string
		wantErr    bool
		errContain string
	}{
		{
			name: "nil checker returns error",
			setupSvc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentServiceWithChecker(nil, nil)
			},
			ip:         "8.8.8.8",
			wantErr:    true,
			errContain: "not initialized",
		},
		{
			name: "arbitrary IP returns not implemented",
			setupSvc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentService(nil)
			},
			ip:         "192.0.2.1",
			wantErr:    true,
			errContain: "not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.setupSvc()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result, err := svc.Enrich(ctx, tt.ip)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errContain != "" && !containsSubstring(err.Error(), tt.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContain)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("result should not be nil on success")
			}
		})
	}
}

func TestIPEnrichment_StructFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	enrichment := roots.IPEnrichment{
		IP:          "8.8.8.8",
		ASN:         15169,
		ASName:      "GOOGLE",
		ISP:         "Google LLC",
		Org:         "Google LLC",
		City:        "Mountain View",
		Region:      "California",
		Country:     "United States",
		CountryCode: "US",
		Latitude:    37.386,
		Longitude:   -122.084,
		Timezone:    "America/Los_Angeles",
		IsProxy:     false,
		IsHosting:   true,
		IsTor:       false,
		QueryTime:   now,
	}

	if enrichment.IP != "8.8.8.8" {
		t.Errorf("IP = %q, want %q", enrichment.IP, "8.8.8.8")
	}
	if enrichment.ASN != 15169 {
		t.Errorf("ASN = %d, want %d", enrichment.ASN, 15169)
	}
	if enrichment.ASName != "GOOGLE" {
		t.Errorf("ASName = %q, want %q", enrichment.ASName, "GOOGLE")
	}
	if enrichment.ISP != "Google LLC" {
		t.Errorf("ISP = %q, want %q", enrichment.ISP, "Google LLC")
	}
	if enrichment.Org != "Google LLC" {
		t.Errorf("Org = %q, want %q", enrichment.Org, "Google LLC")
	}
	if enrichment.City != "Mountain View" {
		t.Errorf("City = %q, want %q", enrichment.City, "Mountain View")
	}
	if enrichment.Region != "California" {
		t.Errorf("Region = %q, want %q", enrichment.Region, "California")
	}
	if enrichment.Country != "United States" {
		t.Errorf("Country = %q, want %q", enrichment.Country, "United States")
	}
	if enrichment.CountryCode != "US" {
		t.Errorf("CountryCode = %q, want %q", enrichment.CountryCode, "US")
	}
	if enrichment.Latitude != 37.386 {
		t.Errorf("Latitude = %f, want %f", enrichment.Latitude, 37.386)
	}
	if enrichment.Longitude != -122.084 {
		t.Errorf("Longitude = %f, want %f", enrichment.Longitude, -122.084)
	}
	if enrichment.Timezone != "America/Los_Angeles" {
		t.Errorf("Timezone = %q, want %q", enrichment.Timezone, "America/Los_Angeles")
	}
	if enrichment.IsProxy {
		t.Error("IsProxy should be false")
	}
	if !enrichment.IsHosting {
		t.Error("IsHosting should be true")
	}
	if enrichment.IsTor {
		t.Error("IsTor should be false")
	}
	if !enrichment.QueryTime.Equal(now) {
		t.Errorf("QueryTime = %v, want %v", enrichment.QueryTime, now)
	}
}

// ============================================================================
// Traceroute Service Tests
// ============================================================================

func TestNewTracerouteService(t *testing.T) {
	t.Parallel()

	svc := roots.NewTracerouteService(nil)
	if svc == nil {
		t.Fatal("NewTracerouteService() returned nil")
	}

	if svc.Tracer() == nil {
		t.Error("Tracer() should not be nil after initialization")
	}
}

// ============================================================================
// Topology Service Tests
// ============================================================================

func TestNewTopologyService(t *testing.T) {
	t.Parallel()

	svc := roots.NewTopologyService(nil, nil)
	if svc == nil {
		t.Fatal("NewTopologyService() returned nil")
	}
}

func TestTopologyService_StartStop(t *testing.T) {
	t.Parallel()

	svc := roots.NewTopologyService(nil, nil)
	ctx := context.Background()

	// Start should not error
	if err := svc.Start(ctx); err != nil {
		t.Errorf("Start() unexpected error: %v", err)
	}

	// Stop should not panic
	svc.Stop()
}

func TestTopologyService_GetTopology(t *testing.T) {
	t.Parallel()

	svc := roots.NewTopologyService(nil, nil)
	ctx := context.Background()

	_, err := svc.GetTopology(ctx)
	if err == nil {
		t.Error("GetTopology() should return an error for unimplemented feature")
	}
}

func TestTopologyNodeType_Constants(t *testing.T) {
	t.Parallel()

	nodeTypes := []roots.TopologyNodeType{
		roots.NodeTypeRouter,
		roots.NodeTypeSwitch,
		roots.NodeTypeHost,
		roots.NodeTypeGateway,
		roots.NodeTypeFirewall,
		roots.NodeTypeAP,
		roots.NodeTypeCloud,
		roots.NodeTypeUnknown,
	}

	for _, nt := range nodeTypes {
		if nt == "" {
			t.Error("TopologyNodeType should not be empty")
		}
	}
}

func TestTopologyLinkType_Constants(t *testing.T) {
	t.Parallel()

	linkTypes := []roots.TopologyLinkType{
		roots.LinkTypeEthernet,
		roots.LinkTypeWiFi,
		roots.LinkTypeFiber,
		roots.LinkTypeWAN,
		roots.LinkTypeVPN,
		roots.LinkTypeUnknown,
	}

	for _, lt := range linkTypes {
		if lt == "" {
			t.Error("TopologyLinkType should not be empty")
		}
	}
}

func TestTopologyNode_StructFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	node := roots.TopologyNode{
		ID:        "node-1",
		Type:      roots.NodeTypeRouter,
		Label:     "Gateway Router",
		IP:        "192.168.1.1",
		MAC:       "00:11:22:33:44:55",
		Vendor:    "Cisco",
		Metadata:  map[string]string{"model": "ISR4451"},
		X:         100.0,
		Y:         200.0,
		UpdatedAt: now,
	}

	if node.ID != "node-1" {
		t.Errorf("ID = %q, want %q", node.ID, "node-1")
	}
	if node.Type != roots.NodeTypeRouter {
		t.Errorf("Type = %q, want %q", node.Type, roots.NodeTypeRouter)
	}
	if node.Label != "Gateway Router" {
		t.Errorf("Label = %q, want %q", node.Label, "Gateway Router")
	}
	if node.IP != "192.168.1.1" {
		t.Errorf("IP = %q, want %q", node.IP, "192.168.1.1")
	}
	if node.MAC != "00:11:22:33:44:55" {
		t.Errorf("MAC = %q, want %q", node.MAC, "00:11:22:33:44:55")
	}
	if node.Vendor != "Cisco" {
		t.Errorf("Vendor = %q, want %q", node.Vendor, "Cisco")
	}
	if node.Metadata["model"] != "ISR4451" {
		t.Errorf("Metadata[model] = %q, want %q", node.Metadata["model"], "ISR4451")
	}
	if node.X != 100.0 {
		t.Errorf("X = %f, want %f", node.X, 100.0)
	}
	if node.Y != 200.0 {
		t.Errorf("Y = %f, want %f", node.Y, 200.0)
	}
	if !node.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", node.UpdatedAt, now)
	}
}

func TestTopologyLink_StructFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	link := roots.TopologyLink{
		ID:        "link-1",
		SourceID:  "node-1",
		TargetID:  "node-2",
		Type:      roots.LinkTypeEthernet,
		Label:     "Primary Link",
		Bandwidth: "1Gbps",
		Latency:   0.5,
		Metadata:  map[string]string{"cable": "Cat6"},
		UpdatedAt: now,
	}

	if link.ID != "link-1" {
		t.Errorf("ID = %q, want %q", link.ID, "link-1")
	}
	if link.SourceID != "node-1" {
		t.Errorf("SourceID = %q, want %q", link.SourceID, "node-1")
	}
	if link.TargetID != "node-2" {
		t.Errorf("TargetID = %q, want %q", link.TargetID, "node-2")
	}
	if link.Type != roots.LinkTypeEthernet {
		t.Errorf("Type = %q, want %q", link.Type, roots.LinkTypeEthernet)
	}
	if link.Label != "Primary Link" {
		t.Errorf("Label = %q, want %q", link.Label, "Primary Link")
	}
	if link.Bandwidth != "1Gbps" {
		t.Errorf("Bandwidth = %q, want %q", link.Bandwidth, "1Gbps")
	}
	if link.Latency != 0.5 {
		t.Errorf("Latency = %f, want %f", link.Latency, 0.5)
	}
	if link.Metadata["cable"] != "Cat6" {
		t.Errorf("Metadata[cable] = %q, want %q", link.Metadata["cable"], "Cat6")
	}
	if !link.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", link.UpdatedAt, now)
	}
}

func TestTopology_StructFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	topology := roots.Topology{
		Nodes: []roots.TopologyNode{
			{ID: "node-1", Type: roots.NodeTypeRouter},
			{ID: "node-2", Type: roots.NodeTypeHost},
		},
		Links: []roots.TopologyLink{
			{ID: "link-1", SourceID: "node-1", TargetID: "node-2"},
		},
		UpdatedAt: now,
	}

	if len(topology.Nodes) != 2 {
		t.Errorf("len(Nodes) = %d, want %d", len(topology.Nodes), 2)
	}
	if len(topology.Links) != 1 {
		t.Errorf("len(Links) = %d, want %d", len(topology.Links), 1)
	}
	if !topology.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", topology.UpdatedAt, now)
	}
}

// ============================================================================
// Module Tests
// ============================================================================

func TestNewModule(t *testing.T) {
	t.Parallel()

	m := roots.New(nil, nil)
	if m == nil {
		t.Fatal("New() returned nil")
	}

	if m.Traceroute() == nil {
		t.Error("Traceroute() should not be nil")
	}
	if m.Topology() == nil {
		t.Error("Topology() should not be nil")
	}
	if m.Enrichment() == nil {
		t.Error("Enrichment() should not be nil")
	}
	if m.Analysis() == nil {
		t.Error("Analysis() should not be nil")
	}
}

func TestModule_StartStop(t *testing.T) {
	t.Parallel()

	m := roots.New(nil, nil)
	ctx := context.Background()

	// Start should not error
	if err := m.Start(ctx); err != nil {
		t.Errorf("Start() unexpected error: %v", err)
	}

	// Stop should not error
	if err := m.Stop(); err != nil {
		t.Errorf("Stop() unexpected error: %v", err)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
