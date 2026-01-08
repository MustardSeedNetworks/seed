package traceroute_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots/traceroute"
)

func TestNewService(t *testing.T) {
	svc := traceroute.NewService()
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	if svc.Tracer() == nil {
		t.Error("Tracer() should not be nil")
	}
}

func TestNewServiceWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		maxHops int
	}{
		{
			name:    "custom timeout and max hops",
			timeout: 5 * time.Second,
			maxHops: 20,
		},
		{
			name:    "zero values use defaults",
			timeout: 0,
			maxHops: 0,
		},
		{
			name:    "small values",
			timeout: 100 * time.Millisecond,
			maxHops: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := traceroute.NewServiceWithConfig(tt.timeout, tt.maxHops)
			if svc == nil {
				t.Fatal("NewServiceWithConfig() returned nil")
			}
			if svc.Tracer() == nil {
				t.Error("Tracer() should not be nil")
			}
		})
	}
}

func TestHop_Structure(t *testing.T) {
	tests := []struct {
		name string
		hop  traceroute.Hop
	}{
		{
			name: "complete hop with all fields",
			hop: traceroute.Hop{
				Number:    5,
				Address:   net.ParseIP("192.168.1.1"),
				Hostname:  "router.local",
				RTT:       10 * time.Millisecond,
				RTTMs:     10.0,
				Lost:      false,
				ASN:       15169,
				ASName:    "Google LLC",
				GeoCity:   "Mountain View",
				GeoRegion: "California",
				ISP:       "Google",
			},
		},
		{
			name: "lost hop",
			hop: traceroute.Hop{
				Number: 3,
				Lost:   true,
			},
		},
		{
			name: "hop with only IP",
			hop: traceroute.Hop{
				Number:  2,
				Address: net.ParseIP("10.0.0.1"),
				RTT:     5 * time.Millisecond,
				RTTMs:   5.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify fields are set correctly
			if tt.hop.Number == 0 && tt.name != "lost hop" {
				t.Error("hop number should be set")
			}
			if tt.hop.Lost && tt.hop.RTT > 0 {
				t.Error("lost hop should not have RTT")
			}
		})
	}
}

func TestResult_Structure(t *testing.T) {
	now := time.Now()
	result := &traceroute.Result{
		Target:      "example.com",
		ResolvedIP:  "93.184.216.34",
		Hops:        make([]traceroute.Hop, 0),
		Complete:    false,
		Duration:    100 * time.Millisecond,
		DurationMs:  100.0,
		StartedAt:   now,
		CompletedAt: now.Add(100 * time.Millisecond),
	}

	if result.Target != "example.com" {
		t.Errorf("Target = %q, want %q", result.Target, "example.com")
	}
	if result.ResolvedIP != "93.184.216.34" {
		t.Errorf("ResolvedIP = %q, want %q", result.ResolvedIP, "93.184.216.34")
	}
	if len(result.Hops) != 0 {
		t.Errorf("Hops length = %d, want 0", len(result.Hops))
	}
	if result.Complete {
		t.Error("Complete should be false")
	}
	if result.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want %v", result.Duration, 100*time.Millisecond)
	}
	if result.DurationMs != 100.0 {
		t.Errorf("DurationMs = %f, want 100.0", result.DurationMs)
	}
	if !result.StartedAt.Equal(now) {
		t.Errorf("StartedAt = %v, want %v", result.StartedAt, now)
	}
	if !result.CompletedAt.Equal(now.Add(100 * time.Millisecond)) {
		t.Errorf("CompletedAt = %v, want %v", result.CompletedAt, now.Add(100*time.Millisecond))
	}
}

func TestOptions_Defaults(t *testing.T) {
	opts := &traceroute.Options{}

	// Verify default values when not explicitly set
	if opts.MaxHops != 0 {
		t.Errorf("MaxHops default = %d, want 0", opts.MaxHops)
	}
	if opts.Timeout != 0 {
		t.Errorf("Timeout default = %v, want 0", opts.Timeout)
	}
	if opts.UseUDP {
		t.Error("UseUDP default should be false")
	}
	if opts.DontResolve {
		t.Error("DontResolve default should be false")
	}
}

func TestBottleneck_Structure(t *testing.T) {
	tests := []struct {
		name       string
		bottleneck traceroute.Bottleneck
	}{
		{
			name: "typical bottleneck",
			bottleneck: traceroute.Bottleneck{
				HopNumber:   5,
				Address:     "192.168.1.1",
				RTTIncrease: 75.5,
				Reason:      "Significant latency increase",
			},
		},
		{
			name: "bottleneck without address",
			bottleneck: traceroute.Bottleneck{
				HopNumber:   3,
				RTTIncrease: 100.0,
				Reason:      "Significant latency increase",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.bottleneck.HopNumber <= 0 {
				t.Error("HopNumber should be positive")
			}
			if tt.bottleneck.RTTIncrease <= 0 {
				t.Error("RTTIncrease should be positive")
			}
			if tt.bottleneck.Reason == "" {
				t.Error("Reason should not be empty")
			}
		})
	}
}

func TestPathAnalysis_Structure(t *testing.T) {
	analysis := &traceroute.PathAnalysis{
		Target:         "example.com",
		Hops:           10,
		AverageRTT:     25.5,
		PacketLoss:     10.0,
		ASNTransitions: 3,
		Bottlenecks: []traceroute.Bottleneck{
			{HopNumber: 5, RTTIncrease: 50.0, Reason: "Significant latency increase"},
		},
		Analysis: "Good path quality with acceptable latency.",
		Score:    75,
	}

	if analysis.Target != "example.com" {
		t.Errorf("Target = %q, want %q", analysis.Target, "example.com")
	}
	if analysis.Hops != 10 {
		t.Errorf("Hops = %d, want 10", analysis.Hops)
	}
	if analysis.AverageRTT != 25.5 {
		t.Errorf("AverageRTT = %f, want 25.5", analysis.AverageRTT)
	}
	if analysis.PacketLoss != 10.0 {
		t.Errorf("PacketLoss = %f, want 10.0", analysis.PacketLoss)
	}
	if analysis.ASNTransitions != 3 {
		t.Errorf("ASNTransitions = %d, want 3", analysis.ASNTransitions)
	}
	if len(analysis.Bottlenecks) != 1 {
		t.Errorf("Bottlenecks length = %d, want 1", len(analysis.Bottlenecks))
	}
	if analysis.Analysis != "Good path quality with acceptable latency." {
		t.Errorf("Analysis = %q, want %q", analysis.Analysis, "Good path quality with acceptable latency.")
	}
	if analysis.Score != 75 {
		t.Errorf("Score = %d, want 75", analysis.Score)
	}
}

func TestAnalyzePath_NilResult(t *testing.T) {
	_, err := traceroute.AnalyzePath(nil)
	if err == nil {
		t.Error("AnalyzePath(nil) should return error")
	}
}

func TestAnalyzePath_EmptyHops(t *testing.T) {
	result := &traceroute.Result{Target: "example.com", Hops: []traceroute.Hop{}}
	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() error = %v", err)
	}
	if analysis.Hops != 0 {
		t.Errorf("Hops = %d, want 0", analysis.Hops)
	}
	if analysis.Score != traceroute.MaxScore {
		t.Errorf("Score = %d, want %d", analysis.Score, traceroute.MaxScore)
	}
}

func TestAnalyzePath_AllHopsResponding(t *testing.T) {
	result := &traceroute.Result{
		Target: "example.com",
		Hops: []traceroute.Hop{
			{Number: 1, RTTMs: 1.0, Lost: false},
			{Number: 2, RTTMs: 5.0, Lost: false},
			{Number: 3, RTTMs: 10.0, Lost: false},
		},
	}
	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() error = %v", err)
	}
	if analysis.Hops != 3 {
		t.Errorf("Hops = %d, want 3", analysis.Hops)
	}
	if analysis.PacketLoss != 0 {
		t.Errorf("PacketLoss = %f, want 0", analysis.PacketLoss)
	}
	expectedAvg := (1.0 + 5.0 + 10.0) / 3
	if analysis.AverageRTT != expectedAvg {
		t.Errorf("AverageRTT = %f, want %f", analysis.AverageRTT, expectedAvg)
	}
}

func TestAnalyzePath_SomeHopsLost(t *testing.T) {
	result := &traceroute.Result{
		Target: "example.com",
		Hops: []traceroute.Hop{
			{Number: 1, RTTMs: 1.0, Lost: false},
			{Number: 2, Lost: true},
			{Number: 3, RTTMs: 10.0, Lost: false},
			{Number: 4, Lost: true},
		},
	}
	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() error = %v", err)
	}
	if analysis.Hops != 4 {
		t.Errorf("Hops = %d, want 4", analysis.Hops)
	}
	expectedLoss := 50.0 // 2 out of 4 hops lost
	if analysis.PacketLoss != expectedLoss {
		t.Errorf("PacketLoss = %f, want %f", analysis.PacketLoss, expectedLoss)
	}
	expectedAvg := (1.0 + 10.0) / 2
	if analysis.AverageRTT != expectedAvg {
		t.Errorf("AverageRTT = %f, want %f", analysis.AverageRTT, expectedAvg)
	}
}

func TestAnalyzePath_BottleneckDetection(t *testing.T) {
	result := &traceroute.Result{
		Target: "example.com",
		Hops: []traceroute.Hop{
			{Number: 1, RTTMs: 5.0, Lost: false},
			{Number: 2, RTTMs: 60.0, Lost: false}, // 55ms increase > threshold
			{Number: 3, RTTMs: 65.0, Lost: false},
		},
	}
	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() error = %v", err)
	}
	if len(analysis.Bottlenecks) != 1 {
		t.Errorf("Bottlenecks count = %d, want 1", len(analysis.Bottlenecks))
	}
	if len(analysis.Bottlenecks) > 0 && analysis.Bottlenecks[0].HopNumber != 2 {
		t.Errorf("Bottleneck hop = %d, want 2", analysis.Bottlenecks[0].HopNumber)
	}
}

func TestAnalyzePath_RatioBasedBottleneck(t *testing.T) {
	result := &traceroute.Result{
		Target: "example.com",
		Hops: []traceroute.Hop{
			{Number: 1, RTTMs: 10.0, Lost: false},
			{Number: 2, RTTMs: 25.0, Lost: false}, // 2.5x ratio > threshold
			{Number: 3, RTTMs: 30.0, Lost: false},
		},
	}
	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() error = %v", err)
	}
	if len(analysis.Bottlenecks) != 1 {
		t.Errorf("Bottlenecks count = %d, want 1", len(analysis.Bottlenecks))
	}
}

func TestAnalyzePath_HighRTTPenalty(t *testing.T) {
	result := &traceroute.Result{
		Target: "example.com",
		Hops: []traceroute.Hop{
			{Number: 1, RTTMs: 150.0, Lost: false}, // High RTT
			{Number: 2, RTTMs: 160.0, Lost: false},
			{Number: 3, RTTMs: 170.0, Lost: false},
		},
	}
	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() error = %v", err)
	}
	// Average RTT is 160ms, above 100ms threshold
	if analysis.Score >= traceroute.MaxScore {
		t.Errorf("Score = %d should be less than max due to high RTT", analysis.Score)
	}
}

func TestScoreToDescription(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		contains string
	}{
		{
			name:     "excellent score",
			score:    95,
			contains: "Excellent",
		},
		{
			name:     "excellent boundary",
			score:    90,
			contains: "Excellent",
		},
		{
			name:     "good score",
			score:    80,
			contains: "Good",
		},
		{
			name:     "good boundary",
			score:    70,
			contains: "Good",
		},
		{
			name:     "fair score",
			score:    60,
			contains: "Fair",
		},
		{
			name:     "fair boundary",
			score:    50,
			contains: "Fair",
		},
		{
			name:     "poor score",
			score:    40,
			contains: "Poor",
		},
		{
			name:     "poor boundary",
			score:    30,
			contains: "Poor",
		},
		{
			name:     "very poor score",
			score:    20,
			contains: "Very poor",
		},
		{
			name:     "zero score",
			score:    0,
			contains: "Very poor",
		},
		{
			name:     "negative score",
			score:    -10,
			contains: "Very poor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := traceroute.ScoreToDescription(tt.score)
			if desc == "" {
				t.Error("ScoreToDescription() returned empty string")
			}
			if !containsSubstring(desc, tt.contains) {
				t.Errorf("ScoreToDescription(%d) = %q, want to contain %q", tt.score, desc, tt.contains)
			}
		})
	}
}

func TestIsHopLost(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{
			name:  "timeout state",
			state: traceroute.StateTimeout,
			want:  true,
		},
		{
			name:  "unreachable state",
			state: traceroute.StateUnreachable,
			want:  true,
		},
		{
			name:  "reply state",
			state: traceroute.StateReply,
			want:  false,
		},
		{
			name:  "error state",
			state: traceroute.StateError,
			want:  false,
		},
		{
			name:  "empty state",
			state: "",
			want:  false,
		},
		{
			name:  "unknown state",
			state: "unknown",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := traceroute.IsHopLost(tt.state)
			if got != tt.want {
				t.Errorf("IsHopLost(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestCalculateRTTMs(t *testing.T) {
	tests := []struct {
		name string
		rtt  time.Duration
		want float64
	}{
		{
			name: "zero duration",
			rtt:  0,
			want: 0,
		},
		{
			name: "one millisecond",
			rtt:  1 * time.Millisecond,
			want: 1.0,
		},
		{
			name: "100 milliseconds",
			rtt:  100 * time.Millisecond,
			want: 100.0,
		},
		{
			name: "one second",
			rtt:  1 * time.Second,
			want: 1000.0,
		},
		{
			name: "sub-millisecond rounds down",
			rtt:  500 * time.Microsecond,
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := traceroute.CalculateRTTMs(tt.rtt)
			if got != tt.want {
				t.Errorf("CalculateRTTMs(%v) = %f, want %f", tt.rtt, got, tt.want)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have expected values
	tests := []struct {
		name  string
		got   any
		want  any
		check func() bool
	}{
		{
			name: "DefaultHopTimeout",
			got:  traceroute.DefaultHopTimeout,
			want: 2 * time.Second,
			check: func() bool {
				return traceroute.DefaultHopTimeout == 2*time.Second
			},
		},
		{
			name: "DefaultMaxHops",
			got:  traceroute.DefaultMaxHops,
			want: 30,
			check: func() bool {
				return traceroute.DefaultMaxHops == 30
			},
		},
		{
			name: "DefaultUDPPort",
			got:  traceroute.DefaultUDPPort,
			want: 33434,
			check: func() bool {
				return traceroute.DefaultUDPPort == 33434
			},
		},
		{
			name: "MaxScore",
			got:  traceroute.MaxScore,
			want: 100,
			check: func() bool {
				return traceroute.MaxScore == 100
			},
		},
		{
			name: "BottleneckRTTThreshold",
			got:  traceroute.BottleneckRTTThreshold,
			want: 50.0,
			check: func() bool {
				return traceroute.BottleneckRTTThreshold == 50.0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestStateConstants(t *testing.T) {
	// Verify state constants are distinct and not empty
	states := []string{
		traceroute.StateReply,
		traceroute.StateTimeout,
		traceroute.StateUnreachable,
		traceroute.StateError,
	}

	seen := make(map[string]bool)
	for _, state := range states {
		if state == "" {
			t.Error("state constant should not be empty")
		}
		if seen[state] {
			t.Errorf("duplicate state constant: %s", state)
		}
		seen[state] = true
	}
}

func TestAnalyzePath_AllHopsLost(t *testing.T) {
	result := &traceroute.Result{
		Target: "example.com",
		Hops: []traceroute.Hop{
			{Number: 1, Lost: true},
			{Number: 2, Lost: true},
			{Number: 3, Lost: true},
		},
	}

	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() unexpected error: %v", err)
	}

	if analysis.PacketLoss != 100.0 {
		t.Errorf("PacketLoss = %f, want 100.0", analysis.PacketLoss)
	}
	if analysis.AverageRTT != 0 {
		t.Errorf("AverageRTT = %f, want 0 (no responding hops)", analysis.AverageRTT)
	}
	// Score should be heavily penalized due to 100% packet loss
	if analysis.Score > 0 {
		t.Errorf("Score = %d, expected 0 due to 100%% packet loss", analysis.Score)
	}
}

func TestAnalyzePath_WithAddresses(t *testing.T) {
	result := &traceroute.Result{
		Target: "example.com",
		Hops: []traceroute.Hop{
			{Number: 1, Address: net.ParseIP("10.0.0.1"), RTTMs: 5.0, Lost: false},
			{Number: 2, Address: net.ParseIP("192.168.1.1"), RTTMs: 60.0, Lost: false}, // Bottleneck
			{Number: 3, Address: net.ParseIP("8.8.8.8"), RTTMs: 65.0, Lost: false},
		},
	}

	analysis, err := traceroute.AnalyzePath(result)
	if err != nil {
		t.Fatalf("AnalyzePath() unexpected error: %v", err)
	}

	if len(analysis.Bottlenecks) != 1 {
		t.Fatalf("Bottlenecks count = %d, want 1", len(analysis.Bottlenecks))
	}

	bottleneck := analysis.Bottlenecks[0]
	if bottleneck.Address != "192.168.1.1" {
		t.Errorf("Bottleneck address = %q, want %q", bottleneck.Address, "192.168.1.1")
	}
}

func TestService_Trace_InvalidTarget(t *testing.T) {
	svc := traceroute.NewService()
	ctx := context.Background()

	result, err := svc.Trace(ctx, "invalid.hostname.that.does.not.exist.example", nil)
	if err != nil {
		t.Fatalf("Trace() returned error: %v", err)
	}
	// The result should contain an error message in the result itself (from discovery layer)
	if result.Complete {
		t.Error("trace to invalid hostname should not complete successfully")
	}
}

func TestService_Trace_ContextCancellation(t *testing.T) {
	svc := traceroute.NewService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := svc.Trace(ctx, "8.8.8.8", nil)
	if err != nil {
		t.Fatalf("Trace() returned error: %v", err)
	}
	// Result should indicate incomplete trace
	t.Logf("Trace result after cancellation: complete=%v, hops=%d", result.Complete, len(result.Hops))
}

func TestService_Trace_WithOptions(t *testing.T) {
	tests := []struct {
		name string
		opts *traceroute.Options
	}{
		{
			name: "nil options",
			opts: nil,
		},
		{
			name: "custom timeout",
			opts: &traceroute.Options{
				Timeout: 1 * time.Second,
			},
		},
		{
			name: "custom max hops",
			opts: &traceroute.Options{
				MaxHops: 5,
			},
		},
		{
			name: "UDP mode",
			opts: &traceroute.Options{
				UseUDP: true,
			},
		},
		{
			name: "don't resolve",
			opts: &traceroute.Options{
				DontResolve: true,
			},
		},
		{
			name: "all options",
			opts: &traceroute.Options{
				MaxHops:     10,
				Timeout:     500 * time.Millisecond,
				UseUDP:      true,
				DontResolve: true,
			},
		},
	}

	svc := traceroute.NewService()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a non-routable IP to make the test fast
			result, err := svc.Trace(ctx, "127.0.0.1", tt.opts)
			if err != nil {
				t.Fatalf("Trace() returned error: %v", err)
			}
			if result == nil {
				t.Fatal("Trace() returned nil result")
			}
			if result.Target != "127.0.0.1" {
				t.Errorf("Target = %q, want %q", result.Target, "127.0.0.1")
			}
		})
	}
}

func TestErrors(t *testing.T) {
	// Test that errors are properly defined
	if traceroute.ErrNotInitialized == nil {
		t.Error("ErrNotInitialized should not be nil")
	}
	if traceroute.ErrNilResult == nil {
		t.Error("ErrNilResult should not be nil")
	}

	// Test error messages
	if traceroute.ErrNotInitialized.Error() == "" {
		t.Error("ErrNotInitialized should have a message")
	}
	if traceroute.ErrNilResult.Error() == "" {
		t.Error("ErrNilResult should have a message")
	}
}

// containsSubstring is a helper to check if a string contains a substring.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
