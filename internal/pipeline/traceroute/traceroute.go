// Package traceroute provides traceroute functionality for the roots module.
// It wraps the discovery package's tracer and provides additional features
// such as path analysis, bottleneck detection, and hop enrichment.
package traceroute

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/krisarmstrong/seed/internal/services/discovery"
)

// Hop state constants.
const (
	StateReply       = "reply"
	StateTimeout     = "timeout"
	StateUnreachable = "unreachable"
	StateError       = "error"
)

// Default configuration values.
const (
	DefaultHopTimeout = 2 * time.Second
	DefaultMaxHops    = 30
	DefaultUDPPort    = 33434
)

// Analysis thresholds.
const (
	BottleneckRTTThreshold = 50.0 // Minimum RTT increase (ms) to flag as bottleneck
	BottleneckRTTRatio     = 2.0  // Minimum ratio of current/previous RTT to flag as bottleneck
	HighRTTThreshold       = 100  // RTT threshold (ms) above which score is penalized
	RTTPenaltyDivisor      = 10   // Divisor for calculating RTT penalty
	BottleneckPenalty      = 5    // Score deduction per bottleneck
	MaxScore               = 100  // Maximum possible path quality score
	ScoreExcellent         = 90   // Threshold for excellent path quality
	ScoreGood              = 70   // Threshold for good path quality
	ScoreFair              = 50   // Threshold for fair path quality
	ScorePoor              = 30   // Threshold for poor path quality
	PercentMultiplier      = 100  // Multiplier to convert ratio to percentage
)

// ErrNotInitialized is returned when a service is accessed before initialization.
var ErrNotInitialized = errors.New("traceroute service not initialized")

// ErrNilResult is returned when attempting to analyze a nil result.
var ErrNilResult = errors.New("traceroute result is nil")

// Hop represents a single hop in a traceroute.
type Hop struct {
	Number    int           `json:"number"`
	Address   net.IP        `json:"address,omitempty"`
	Hostname  string        `json:"hostname,omitempty"`
	RTT       time.Duration `json:"rtt"`
	RTTMs     float64       `json:"rttMs"`
	Lost      bool          `json:"lost"`
	ASN       uint32        `json:"asn,omitempty"`
	ASName    string        `json:"asName,omitempty"`
	GeoCity   string        `json:"geoCity,omitempty"`
	GeoRegion string        `json:"geoRegion,omitempty"`
	ISP       string        `json:"isp,omitempty"`
}

// Result contains the full result of a traceroute.
type Result struct {
	Target      string        `json:"target"`
	ResolvedIP  string        `json:"resolvedIp"`
	Hops        []Hop         `json:"hops"`
	Complete    bool          `json:"complete"`
	Duration    time.Duration `json:"duration"`
	DurationMs  float64       `json:"durationMs"`
	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt time.Time     `json:"completedAt"`
}

// Options configures a traceroute execution.
type Options struct {
	MaxHops     int           `json:"maxHops"`
	Timeout     time.Duration `json:"timeout"`
	Probes      int           `json:"probes"`
	PacketSize  int           `json:"packetSize"`
	EnrichHops  bool          `json:"enrichHops"` // Add ASN/geo data
	UseUDP      bool          `json:"useUdp"`
	SourceAddr  string        `json:"sourceAddr,omitempty"`
	DontResolve bool          `json:"dontResolve"`
}

// Bottleneck identifies a potential bottleneck in the path.
type Bottleneck struct {
	HopNumber   int     `json:"hopNumber"`
	Address     string  `json:"address"`
	RTTIncrease float64 `json:"rttIncreaseMs"`
	Reason      string  `json:"reason"`
}

// PathAnalysis contains analysis results for a network path.
type PathAnalysis struct {
	Target         string       `json:"target"`
	Hops           int          `json:"hops"`
	AverageRTT     float64      `json:"averageRttMs"`
	PacketLoss     float64      `json:"packetLossPercent"`
	ASNTransitions int          `json:"asnTransitions"`
	Bottlenecks    []Bottleneck `json:"bottlenecks,omitempty"`
	Analysis       string       `json:"analysis"`
	Score          int          `json:"score"` // 0-100 path quality score
}

// Service handles network path tracing.
type Service struct {
	tracer *discovery.Tracer
}

// NewService creates a new traceroute service with default configuration.
func NewService() *Service {
	return &Service{
		tracer: discovery.NewTracer(DefaultHopTimeout, DefaultMaxHops),
	}
}

// NewServiceWithConfig creates a new traceroute service with custom configuration.
func NewServiceWithConfig(timeout time.Duration, maxHops int) *Service {
	return &Service{
		tracer: discovery.NewTracer(timeout, maxHops),
	}
}

// Trace performs a traceroute to the target with the given options.
func (s *Service) Trace(ctx context.Context, target string, opts *Options) (*Result, error) {
	if s.tracer == nil {
		return nil, ErrNotInitialized
	}

	startTime := time.Now()

	// Apply options to create appropriate tracer
	timeout := DefaultHopTimeout
	maxHops := DefaultMaxHops
	if opts != nil {
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
		if opts.MaxHops > 0 {
			maxHops = opts.MaxHops
		}
	}

	// Create tracer with options
	var tracer *discovery.Tracer
	if opts != nil && !opts.DontResolve {
		tracer = discovery.NewTracerWithPTR(timeout, maxHops)
	} else {
		tracer = discovery.NewTracer(timeout, maxHops)
	}

	// Perform the trace based on protocol
	var discoveryResult *discovery.TracerouteResult
	if opts != nil && opts.UseUDP {
		discoveryResult = tracer.TraceUDP(ctx, target, DefaultUDPPort)
	} else {
		discoveryResult = tracer.TraceICMP(ctx, target)
	}

	// Convert discovery result to traceroute result
	result := &Result{
		Target:      discoveryResult.Target,
		ResolvedIP:  discoveryResult.TargetIP,
		Complete:    discoveryResult.Completed,
		Duration:    time.Since(startTime),
		DurationMs:  float64(time.Since(startTime).Milliseconds()),
		StartedAt:   startTime,
		CompletedAt: time.Now(),
		Hops:        make([]Hop, 0, len(discoveryResult.Hops)),
	}

	for _, hop := range discoveryResult.Hops {
		trHop := Hop{
			Number:   hop.TTL,
			Hostname: hop.Hostname,
			RTT:      hop.RTT,
			RTTMs:    float64(hop.RTT.Milliseconds()),
			Lost:     hop.State == StateTimeout || hop.State == StateUnreachable,
		}

		// Parse IP address
		if hop.IP != "" {
			trHop.Address = net.ParseIP(hop.IP)
		}

		result.Hops = append(result.Hops, trHop)
	}

	return result, nil
}

// Tracer returns the underlying tracer for dependency injection.
func (s *Service) Tracer() *discovery.Tracer {
	return s.tracer
}

// AnalyzePath performs quality analysis on a traceroute result.
func AnalyzePath(result *Result) (*PathAnalysis, error) {
	if result == nil {
		return nil, ErrNilResult
	}

	analysis := &PathAnalysis{
		Target:      result.Target,
		Hops:        len(result.Hops),
		Bottlenecks: make([]Bottleneck, 0),
	}

	// Calculate statistics and detect bottlenecks
	totalRTT, lostHops := analyzeHops(result.Hops, &analysis.Bottlenecks)

	// Calculate average RTT (excluding lost hops)
	respondingHops := len(result.Hops) - lostHops
	if respondingHops > 0 {
		analysis.AverageRTT = totalRTT / float64(respondingHops)
	}

	// Calculate packet loss percentage
	if len(result.Hops) > 0 {
		analysis.PacketLoss = float64(lostHops) / float64(len(result.Hops)) * PercentMultiplier
	}

	// Calculate path quality score and generate analysis text
	analysis.Score = calculateScore(analysis)
	analysis.Analysis = ScoreToDescription(analysis.Score)

	return analysis, nil
}

// analyzeHops processes hops to calculate RTT stats and detect bottlenecks.
func analyzeHops(hops []Hop, bottlenecks *[]Bottleneck) (float64, int) {
	var previousRTT float64
	var totalRTT float64
	var lostHops int

	for i, hop := range hops {
		if hop.Lost {
			lostHops++
			continue
		}

		rttMs := hop.RTTMs
		totalRTT += rttMs

		// Detect bottlenecks (significant RTT increase from previous hop)
		if bottleneck := detectBottleneck(i, hop, previousRTT, rttMs); bottleneck != nil {
			*bottlenecks = append(*bottlenecks, *bottleneck)
		}

		previousRTT = rttMs
	}

	return totalRTT, lostHops
}

// detectBottleneck checks if a hop represents a bottleneck based on RTT increase.
func detectBottleneck(hopIndex int, hop Hop, previousRTT, currentRTT float64) *Bottleneck {
	if hopIndex == 0 || previousRTT <= 0 || currentRTT <= 0 {
		return nil
	}

	increase := currentRTT - previousRTT
	isSignificantIncrease := increase > BottleneckRTTThreshold ||
		currentRTT/previousRTT > BottleneckRTTRatio

	if !isSignificantIncrease {
		return nil
	}

	bottleneck := &Bottleneck{
		HopNumber:   hop.Number,
		RTTIncrease: increase,
		Reason:      "Significant latency increase",
	}
	if hop.Address != nil {
		bottleneck.Address = hop.Address.String()
	}
	return bottleneck
}

// calculateScore computes the path quality score (0-100).
func calculateScore(analysis *PathAnalysis) int {
	score := MaxScore

	// Deduct for packet loss
	score -= int(analysis.PacketLoss)

	// Deduct for high average RTT
	if analysis.AverageRTT > HighRTTThreshold {
		score -= int((analysis.AverageRTT - HighRTTThreshold) / RTTPenaltyDivisor)
	}

	// Deduct for bottlenecks
	score -= len(analysis.Bottlenecks) * BottleneckPenalty

	// Ensure score is within bounds
	return max(0, min(MaxScore, score))
}

// ScoreToDescription converts a score to a human-readable description.
func ScoreToDescription(score int) string {
	switch {
	case score >= ScoreExcellent:
		return "Excellent path quality with low latency and no packet loss."
	case score >= ScoreGood:
		return "Good path quality with acceptable latency."
	case score >= ScoreFair:
		return "Fair path quality. Some latency or packet loss detected."
	case score >= ScorePoor:
		return "Poor path quality. High latency or significant packet loss."
	default:
		return "Very poor path quality. Consider using an alternative route."
	}
}

// IsHopLost determines if a hop was lost based on its state.
func IsHopLost(state string) bool {
	return state == StateTimeout || state == StateUnreachable
}

// CalculateRTTMs converts a duration to milliseconds as float64.
func CalculateRTTMs(rtt time.Duration) float64 {
	return float64(rtt.Milliseconds())
}
