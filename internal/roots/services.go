package roots

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/roots/publicip"
)

// TracerouteService handles network path tracing.
type TracerouteService struct {
	cfg    *config.Config
	tracer *discovery.Tracer
}

// NewTracerouteService creates a new traceroute service.
func NewTracerouteService(cfg *config.Config) *TracerouteService {
	// Default timeout from config or fallback
	timeout := 2 * time.Second
	maxHops := 30

	return &TracerouteService{
		cfg:    cfg,
		tracer: discovery.NewTracer(timeout, maxHops),
	}
}

// Trace performs a traceroute to the target with the given options.
func (s *TracerouteService) Trace(
	ctx context.Context,
	target string,
	opts *TracerouteOptions,
) (*TracerouteResult, error) {
	if s.tracer == nil {
		return nil, ErrNotInitialized
	}

	startTime := time.Now()

	// Apply options to create appropriate tracer
	timeout := 2 * time.Second
	maxHops := 30
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
		port := 33434 // Default UDP traceroute port
		discoveryResult = tracer.TraceUDP(ctx, target, port)
	} else {
		discoveryResult = tracer.TraceICMP(ctx, target)
	}

	// Convert discovery result to roots result
	result := &TracerouteResult{
		Target:      discoveryResult.Target,
		ResolvedIP:  discoveryResult.TargetIP,
		Complete:    discoveryResult.Completed,
		Duration:    time.Since(startTime),
		DurationMs:  float64(time.Since(startTime).Milliseconds()),
		StartedAt:   startTime,
		CompletedAt: time.Now(),
		Hops:        make([]TracerouteHop, 0, len(discoveryResult.Hops)),
	}

	for _, hop := range discoveryResult.Hops {
		rootsHop := TracerouteHop{
			Number:   hop.TTL,
			Hostname: hop.Hostname,
			RTT:      hop.RTT,
			RTTMs:    float64(hop.RTT.Milliseconds()),
			Lost:     hop.State == "timeout" || hop.State == "unreachable",
		}

		// Parse IP address
		if hop.IP != "" {
			rootsHop.Address = net.ParseIP(hop.IP)
		}

		result.Hops = append(result.Hops, rootsHop)
	}

	return result, nil
}

// TopologyService manages network topology discovery and storage.
type TopologyService struct {
	cfg    *config.Config
	db     *database.DB
	cancel context.CancelFunc
}

// NewTopologyService creates a new topology service.
func NewTopologyService(cfg *config.Config, db *database.DB) *TopologyService {
	return &TopologyService{
		cfg: cfg,
		db:  db,
	}
}

// Start begins background topology discovery.
func (s *TopologyService) Start(ctx context.Context) error {
	_, s.cancel = context.WithCancel(ctx)
	// Topology discovery requires combining data from multiple sources:
	// - Device discovery results
	// - Traceroute paths
	// - LLDP/CDP neighbor data
	// This would be implemented as a data aggregation service
	return nil
}

// Stop halts topology discovery.
func (s *TopologyService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// GetTopology returns the current network topology.
func (s *TopologyService) GetTopology(_ context.Context) (*Topology, error) {
	// Topology building requires aggregating:
	// 1. Discovered devices (from discovery service)
	// 2. Traceroute paths (showing route to internet)
	// 3. LLDP/CDP neighbors (showing L2 connectivity)
	// This is a complex feature that would need dedicated implementation
	return nil, ErrNotImplemented
}

// EnrichmentService provides IP address enrichment (ASN, geo, etc.).
type EnrichmentService struct {
	cfg     *config.Config
	checker *publicip.Checker
}

// NewEnrichmentService creates a new enrichment service.
func NewEnrichmentService(cfg *config.Config) *EnrichmentService {
	return &EnrichmentService{
		cfg:     cfg,
		checker: publicip.NewChecker(),
	}
}

// Enrich looks up enrichment data for an IP address.
func (s *EnrichmentService) Enrich(ctx context.Context, ip string) (*IPEnrichment, error) {
	if s.checker == nil {
		return nil, ErrNotInitialized
	}

	// For the current public IP, use the checker
	// For other IPs, we would need a separate geo lookup service
	// The publicip.Checker is designed for public IP detection, not arbitrary IP lookup

	// Check if this is our public IP
	result := s.checker.GetPublicIP(ctx)
	if result.IPv4 == ip || result.IPv6 == ip {
		enrichment := &IPEnrichment{
			IP:          ip,
			ISP:         result.ISP,
			Org:         result.Org,
			City:        result.City,
			Region:      result.Region,
			Country:     result.Country,
			CountryCode: result.CountryCode,
			Latitude:    result.Lat,
			Longitude:   result.Lon,
			QueryTime:   result.LastChecked,
		}

		// Parse ASN from string format
		if result.ASN != "" {
			var asn uint32
			if _, err := fmt.Sscanf(result.ASN, "%d", &asn); err == nil {
				enrichment.ASN = asn
			}
		}

		return enrichment, nil
	}

	// For other IPs, we'd need an external geo API
	// This is a placeholder for future implementation
	return nil, ErrNotImplemented
}

// GetPublicIP returns the current public IP with enrichment.
func (s *EnrichmentService) GetPublicIP(ctx context.Context) (*IPEnrichment, error) {
	if s.checker == nil {
		return nil, ErrNotInitialized
	}

	result := s.checker.GetPublicIP(ctx)
	if result.Error != "" {
		return nil, fmt.Errorf("public IP lookup failed: %s", result.Error)
	}

	enrichment := &IPEnrichment{
		IP:          result.IPv4,
		ISP:         result.ISP,
		Org:         result.Org,
		City:        result.City,
		Region:      result.Region,
		Country:     result.Country,
		CountryCode: result.CountryCode,
		Latitude:    result.Lat,
		Longitude:   result.Lon,
		QueryTime:   result.LastChecked,
	}

	// Parse ASN from string format
	if result.ASN != "" {
		var asn uint32
		if _, err := fmt.Sscanf(result.ASN, "%d", &asn); err == nil {
			enrichment.ASN = asn
		}
	}

	return enrichment, nil
}

// AnalysisService provides path quality analysis.
type AnalysisService struct {
	cfg *config.Config
	db  *database.DB
}

// NewAnalysisService creates a new analysis service.
func NewAnalysisService(cfg *config.Config, db *database.DB) *AnalysisService {
	return &AnalysisService{cfg: cfg, db: db}
}

// AnalyzePath performs quality analysis on a traceroute result.
func (s *AnalysisService) AnalyzePath(_ context.Context, result *TracerouteResult) (*PathAnalysis, error) {
	if result == nil {
		return nil, errors.New("traceroute result is nil")
	}

	analysis := &PathAnalysis{
		Target:      result.Target,
		Hops:        len(result.Hops),
		Bottlenecks: make([]PathBottleneck, 0),
	}

	// Calculate statistics and detect bottlenecks
	totalRTT, lostHops := s.analyzeHops(result.Hops, &analysis.Bottlenecks)

	// Calculate average RTT (excluding lost hops)
	respondingHops := len(result.Hops) - lostHops
	if respondingHops > 0 {
		analysis.AverageRTT = totalRTT / float64(respondingHops)
	}

	// Calculate packet loss percentage
	if len(result.Hops) > 0 {
		analysis.PacketLoss = float64(lostHops) / float64(len(result.Hops)) * 100
	}

	// Calculate path quality score and generate analysis text
	analysis.Score = s.calculateScore(analysis)
	analysis.Analysis = scoreToDescription(analysis.Score)

	return analysis, nil
}

// analyzeHops processes hops to calculate RTT stats and detect bottlenecks.
func (s *AnalysisService) analyzeHops(
	hops []TracerouteHop,
	bottlenecks *[]PathBottleneck,
) (float64, int) {
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
		if bottleneck := s.detectBottleneck(i, hop, previousRTT, rttMs); bottleneck != nil {
			*bottlenecks = append(*bottlenecks, *bottleneck)
		}

		previousRTT = rttMs
	}

	return totalRTT, lostHops
}

// detectBottleneck checks if a hop represents a bottleneck based on RTT increase.
func (s *AnalysisService) detectBottleneck(
	hopIndex int,
	hop TracerouteHop,
	previousRTT, currentRTT float64,
) *PathBottleneck {
	if hopIndex == 0 || previousRTT <= 0 || currentRTT <= 0 {
		return nil
	}

	increase := currentRTT - previousRTT
	isSignificantIncrease := increase > 50 || currentRTT/previousRTT > 2

	if !isSignificantIncrease {
		return nil
	}

	bottleneck := &PathBottleneck{
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
func (s *AnalysisService) calculateScore(analysis *PathAnalysis) int {
	score := 100

	// Deduct for packet loss
	score -= int(analysis.PacketLoss)

	// Deduct for high average RTT
	if analysis.AverageRTT > 100 {
		score -= int((analysis.AverageRTT - 100) / 10)
	}

	// Deduct for bottlenecks
	score -= len(analysis.Bottlenecks) * 5

	// Ensure score is within bounds
	return max(0, min(100, score))
}

// scoreToDescription converts a score to a human-readable description.
func scoreToDescription(score int) string {
	switch {
	case score >= 90:
		return "Excellent path quality with low latency and no packet loss."
	case score >= 70:
		return "Good path quality with acceptable latency."
	case score >= 50:
		return "Fair path quality. Some latency or packet loss detected."
	case score >= 30:
		return "Poor path quality. High latency or significant packet loss."
	default:
		return "Very poor path quality. Consider using an alternative route."
	}
}

// Tracer returns the underlying tracer for dependency injection.
func (s *TracerouteService) Tracer() *discovery.Tracer {
	return s.tracer
}

// Checker returns the underlying public IP checker for dependency injection.
func (s *EnrichmentService) Checker() *publicip.Checker {
	return s.checker
}
