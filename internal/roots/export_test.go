package roots

import (
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/roots/publicip"
)

// ExportScoreToDescription exports scoreToDescription for testing.
func ExportScoreToDescription(score int) string {
	return scoreToDescription(score)
}

// NewEnrichmentServiceWithChecker creates an EnrichmentService with a custom checker for testing.
func NewEnrichmentServiceWithChecker(cfg any, checker *publicip.Checker) *EnrichmentService {
	return &EnrichmentService{
		cfg:     nil,
		checker: checker,
	}
}

// ExportAnalyzeHops exports AnalysisService.analyzeHops for testing.
func (s *AnalysisService) ExportAnalyzeHops(
	hops []TracerouteHop,
	bottlenecks *[]PathBottleneck,
) (float64, int) {
	return s.analyzeHops(hops, bottlenecks)
}

// ExportDetectBottleneck exports AnalysisService.detectBottleneck for testing.
func (s *AnalysisService) ExportDetectBottleneck(
	hopIndex int,
	hop TracerouteHop,
	previousRTT, currentRTT float64,
) *PathBottleneck {
	return s.detectBottleneck(hopIndex, hop, previousRTT, currentRTT)
}

// ExportCalculateScore exports AnalysisService.calculateScore for testing.
func (s *AnalysisService) ExportCalculateScore(analysis *PathAnalysis) int {
	return s.calculateScore(analysis)
}

// NewTracerouteServiceWithTracer creates a TracerouteService with a custom tracer for testing.
func NewTracerouteServiceWithTracer(tracer *discovery.Tracer) *TracerouteService {
	return &TracerouteService{
		cfg:    nil,
		tracer: tracer,
	}
}

// NewTracerouteServiceNilTracer creates a TracerouteService with nil tracer for testing errors.
func NewTracerouteServiceNilTracer() *TracerouteService {
	return &TracerouteService{
		cfg:    nil,
		tracer: nil,
	}
}
