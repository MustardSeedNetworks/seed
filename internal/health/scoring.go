package health

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

// Score weight constants for composite health calculation.
const (
	// AvailabilityWeight is the weight given to availability in composite score (40%).
	AvailabilityWeight = 0.40

	// LatencyWeight is the weight given to latency score in composite score (30%).
	LatencyWeight = 0.30

	// CriticalityWeight is the weight given to criticality in composite score (30%).
	CriticalityWeight = 0.30
)

// Status thresholds for health determination.
const (
	// HealthyThreshold is the minimum composite score for healthy status.
	HealthyThreshold = 80.0

	// DegradedThreshold is the minimum composite score for degraded status.
	// Below this threshold is considered critical.
	DegradedThreshold = 50.0
)

// Latency scoring constants.
const (
	// DefaultLatencyThresholdMs is the default P95 latency threshold for scoring.
	DefaultLatencyThresholdMs = 500.0

	// ExcellentLatencyRatio is the ratio below which latency is considered excellent.
	ExcellentLatencyRatio = 0.2 // 20% of threshold

	// GoodLatencyRatio is the ratio below which latency is considered good.
	GoodLatencyRatio = 0.5 // 50% of threshold

	// AcceptableLatencyRatio is the ratio below which latency is considered acceptable.
	AcceptableLatencyRatio = 1.0 // 100% of threshold

	// PoorLatencyRatio is the ratio above which latency is considered poor.
	PoorLatencyRatio = 2.0

	// VeryPoorLatencyRatio is the ratio above which latency score becomes zero.
	VeryPoorLatencyRatio = 4.0
)

// Criticality range constants.
const (
	minCriticality        = 1
	maxCriticality        = 10
	criticalityToScoreMul = 10.0
)

// Score tier boundaries for latency calculation.
const (
	scoreMax       = 100.0
	scoreExcellent = 90.0
	scoreGood      = 70.0
	scoreAccept    = 50.0
	scorePoor      = 25.0
	scoreTierGap   = 20.0
	scoreSmallGap  = 10.0
	scorePoorGap   = 25.0
	percentageMul  = 100.0
)

// DefaultCriticality is the default criticality for endpoints without explicit configuration.
const DefaultCriticality = 5

// Health status constants.
const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusCritical = "critical"
	StatusUnknown  = "unknown"
)

// AvailabilityTimeRange is the time range for availability calculation.
const AvailabilityTimeRange = 24 * time.Hour

// EndpointHealthScore represents the computed health score for an endpoint.
type EndpointHealthScore struct {
	EndpointName     string    `json:"endpointName"`
	CheckType        string    `json:"checkType"`
	AvailabilityPct  float64   `json:"availabilityPct"`  // Last 24h uptime percentage
	LatencyScore     float64   `json:"latencyScore"`     // 0-100 based on P95 vs threshold
	CriticalityScore float64   `json:"criticalityScore"` // User-defined 1-10 scale (normalized to 0-100)
	CompositeScore   float64   `json:"compositeScore"`   // Weighted combination
	Status           string    `json:"status"`           // healthy, degraded, critical
	LastCheck        time.Time `json:"lastCheck"`
	P95LatencyMs     float64   `json:"p95LatencyMs"`
	TotalChecks      int64     `json:"totalChecks"`
	SuccessfulChecks int64     `json:"successfulChecks"`
}

// CriticalityConfig defines the criticality level for an endpoint.
type CriticalityConfig struct {
	EndpointName     string `json:"endpointName"`
	CheckType        string `json:"checkType,omitempty"`
	Criticality      int    `json:"criticality"`                // 1-10 scale
	LatencyThreshold int    `json:"latencyThreshold,omitempty"` // Optional custom threshold in ms
}

// ScoringService calculates health scores for endpoints.
type ScoringService struct {
	db               *database.DB
	logger           *slog.Logger
	mu               sync.RWMutex
	criticalityMap   map[string]CriticalityConfig // key: checkType|endpointName
	latencyThreshold float64
}

// NewScoringService creates a new scoring service.
func NewScoringService(db *database.DB, logger *slog.Logger) *ScoringService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ScoringService{
		db:               db,
		logger:           logger,
		criticalityMap:   make(map[string]CriticalityConfig),
		latencyThreshold: DefaultLatencyThresholdMs,
	}
}

// SetCriticality configures the criticality level for an endpoint.
func (s *ScoringService) SetCriticality(config CriticalityConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clamp criticality to valid range
	if config.Criticality < minCriticality {
		config.Criticality = minCriticality
	}
	if config.Criticality > maxCriticality {
		config.Criticality = maxCriticality
	}

	key := s.criticalityKey(config.CheckType, config.EndpointName)
	s.criticalityMap[key] = config
}

// SetCriticalityBatch configures criticality for multiple endpoints.
func (s *ScoringService) SetCriticalityBatch(configs []CriticalityConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, config := range configs {
		// Clamp criticality to valid range
		if config.Criticality < minCriticality {
			config.Criticality = minCriticality
		}
		if config.Criticality > maxCriticality {
			config.Criticality = maxCriticality
		}

		key := s.criticalityKey(config.CheckType, config.EndpointName)
		s.criticalityMap[key] = config
	}
}

// SetLatencyThreshold sets the default latency threshold for scoring.
func (s *ScoringService) SetLatencyThreshold(thresholdMs float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latencyThreshold = thresholdMs
}

// GetCriticality returns the criticality config for an endpoint.
func (s *ScoringService) GetCriticality(checkType, endpointName string) CriticalityConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.criticalityKey(checkType, endpointName)
	if config, ok := s.criticalityMap[key]; ok {
		return config
	}

	// Try with empty check type (endpoint-level config)
	key = s.criticalityKey("", endpointName)
	if config, ok := s.criticalityMap[key]; ok {
		return config
	}

	// Return default
	return CriticalityConfig{
		EndpointName: endpointName,
		CheckType:    checkType,
		Criticality:  DefaultCriticality,
	}
}

// GetAllCriticalities returns all configured criticality settings.
func (s *ScoringService) GetAllCriticalities() []CriticalityConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs := make([]CriticalityConfig, 0, len(s.criticalityMap))
	for _, config := range s.criticalityMap {
		configs = append(configs, config)
	}
	return configs
}

// CalculateScore computes the health score for a single endpoint.
func (s *ScoringService) CalculateScore(
	ctx context.Context,
	checkType, endpointName string,
) (*EndpointHealthScore, error) {
	repo := s.db.HealthChecks()
	now := time.Now().UTC()
	timeRange := database.TimeRange{
		Start: now.Add(-AvailabilityTimeRange),
		End:   now,
	}

	// Get availability
	availability, err := repo.GetAvailability(ctx, checkType, endpointName, timeRange)
	if err != nil {
		return nil, fmt.Errorf("getting availability: %w", err)
	}

	// Get latency stats
	stats, err := repo.GetLatencyStats(ctx, checkType, endpointName, timeRange)
	if err != nil {
		return nil, fmt.Errorf("getting latency stats: %w", err)
	}

	// Get latest check time
	latest, err := repo.GetLatest(ctx, checkType, endpointName)
	if err != nil {
		return nil, fmt.Errorf("getting latest check: %w", err)
	}

	var lastCheck time.Time
	if latest != nil {
		lastCheck = latest.RecordedAt
	}

	// Get criticality config
	critConfig := s.GetCriticality(checkType, endpointName)

	// Calculate latency score
	latencyThreshold := s.latencyThreshold
	if critConfig.LatencyThreshold > 0 {
		latencyThreshold = float64(critConfig.LatencyThreshold)
	}
	latencyScore := s.calculateLatencyScore(stats.P95Ms, latencyThreshold)

	// Convert criticality (1-10) to score (0-100)
	criticalityScore := float64(critConfig.Criticality) * criticalityToScoreMul

	// Calculate composite score
	compositeScore := (availability * AvailabilityWeight) +
		(latencyScore * LatencyWeight) +
		(criticalityScore * CriticalityWeight)

	// Determine status
	status := s.determineStatus(compositeScore, availability, stats.Count)

	return &EndpointHealthScore{
		EndpointName:     endpointName,
		CheckType:        checkType,
		AvailabilityPct:  availability,
		LatencyScore:     latencyScore,
		CriticalityScore: criticalityScore,
		CompositeScore:   compositeScore,
		Status:           status,
		LastCheck:        lastCheck,
		P95LatencyMs:     stats.P95Ms,
		TotalChecks:      stats.Count,
		SuccessfulChecks: int64(float64(stats.Count) * availability / percentageMul),
	}, nil
}

// CalculateAllScores computes health scores for all endpoints.
func (s *ScoringService) CalculateAllScores(ctx context.Context) ([]*EndpointHealthScore, error) {
	repo := s.db.HealthChecks()

	// Get all distinct check types
	checkTypes, err := repo.GetDistinctCheckTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting check types: %w", err)
	}

	var scores []*EndpointHealthScore
	for _, checkType := range checkTypes {
		endpoints, epErr := repo.GetDistinctEndpoints(ctx, checkType)
		if epErr != nil {
			s.logger.WarnContext(ctx, "failed to get endpoints",
				"check_type", checkType,
				"error", epErr)
			continue
		}

		for _, endpoint := range endpoints {
			score, scoreErr := s.CalculateScore(ctx, checkType, endpoint)
			if scoreErr != nil {
				s.logger.WarnContext(ctx, "failed to calculate score",
					"check_type", checkType,
					"endpoint", endpoint,
					"error", scoreErr)
				continue
			}
			scores = append(scores, score)
		}
	}

	return scores, nil
}

// GetOverallHealth returns an aggregate health assessment across all endpoints.
func (s *ScoringService) GetOverallHealth(ctx context.Context) (*OverallHealthSummary, error) {
	scores, err := s.CalculateAllScores(ctx)
	if err != nil {
		return nil, err
	}

	if len(scores) == 0 {
		return &OverallHealthSummary{
			Status:        StatusUnknown,
			EndpointCount: 0,
			HealthyCount:  0,
			DegradedCount: 0,
			CriticalCount: 0,
			OverallScore:  0,
			CalculatedAt:  time.Now().UTC(),
		}, nil
	}

	var (
		healthyCount  int
		degradedCount int
		criticalCount int
		totalScore    float64
	)

	for _, score := range scores {
		totalScore += score.CompositeScore
		switch score.Status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusCritical, StatusUnknown:
			criticalCount++
		}
	}

	avgScore := totalScore / float64(len(scores))

	// Determine overall status based on aggregate score and critical count
	var status string
	switch {
	case criticalCount > 0:
		status = StatusCritical
	case degradedCount > len(scores)/4: // More than 25% degraded
		status = StatusDegraded
	case avgScore >= HealthyThreshold:
		status = StatusHealthy
	case avgScore >= DegradedThreshold:
		status = StatusDegraded
	default:
		status = StatusCritical
	}

	return &OverallHealthSummary{
		Status:         status,
		EndpointCount:  len(scores),
		HealthyCount:   healthyCount,
		DegradedCount:  degradedCount,
		CriticalCount:  criticalCount,
		OverallScore:   avgScore,
		CalculatedAt:   time.Now().UTC(),
		EndpointScores: scores,
	}, nil
}

// OverallHealthSummary provides aggregate health metrics.
type OverallHealthSummary struct {
	Status         string                 `json:"status"`
	EndpointCount  int                    `json:"endpointCount"`
	HealthyCount   int                    `json:"healthyCount"`
	DegradedCount  int                    `json:"degradedCount"`
	CriticalCount  int                    `json:"criticalCount"`
	OverallScore   float64                `json:"overallScore"`
	CalculatedAt   time.Time              `json:"calculatedAt"`
	EndpointScores []*EndpointHealthScore `json:"endpointScores,omitempty"`
}

// calculateLatencyScore converts P95 latency to a 0-100 score.
// Lower latency = higher score.
func (s *ScoringService) calculateLatencyScore(p95Ms, threshold float64) float64 {
	if threshold <= 0 {
		threshold = DefaultLatencyThresholdMs
	}

	ratio := p95Ms / threshold

	switch {
	case ratio <= ExcellentLatencyRatio:
		// Excellent: 90-100
		return scoreMax - (ratio/ExcellentLatencyRatio)*scoreSmallGap
	case ratio <= GoodLatencyRatio:
		// Good: 70-90
		normalized := (ratio - ExcellentLatencyRatio) / (GoodLatencyRatio - ExcellentLatencyRatio)
		return scoreExcellent - normalized*scoreTierGap
	case ratio <= AcceptableLatencyRatio:
		// Acceptable: 50-70
		normalized := (ratio - GoodLatencyRatio) / (AcceptableLatencyRatio - GoodLatencyRatio)
		return scoreGood - normalized*scoreTierGap
	case ratio <= PoorLatencyRatio:
		// Poor: 25-50
		normalized := (ratio - AcceptableLatencyRatio)
		return scoreAccept - normalized*scorePoorGap
	default:
		// Very poor: 0-25
		if ratio >= VeryPoorLatencyRatio {
			return 0.0
		}
		normalized := (ratio - PoorLatencyRatio) / PoorLatencyRatio
		return scorePoor - normalized*scorePoorGap
	}
}

// determineStatus converts composite score to health status.
func (s *ScoringService) determineStatus(compositeScore, availability float64, checkCount int64) string {
	// No data = unknown
	if checkCount == 0 {
		return StatusUnknown
	}

	// Zero availability = critical regardless of score
	if availability == 0 {
		return StatusCritical
	}

	// Use composite score for status
	switch {
	case compositeScore >= HealthyThreshold:
		return StatusHealthy
	case compositeScore >= DegradedThreshold:
		return StatusDegraded
	default:
		return StatusCritical
	}
}

// criticalityKey generates a map key for criticality lookup.
func (s *ScoringService) criticalityKey(checkType, endpointName string) string {
	return checkType + "|" + endpointName
}

// GetScoresByStatus returns endpoints grouped by their health status.
func (s *ScoringService) GetScoresByStatus(ctx context.Context) (map[string][]*EndpointHealthScore, error) {
	scores, err := s.CalculateAllScores(ctx)
	if err != nil {
		return nil, err
	}

	result := map[string][]*EndpointHealthScore{
		StatusHealthy:  {},
		StatusDegraded: {},
		StatusCritical: {},
		StatusUnknown:  {},
	}

	for _, score := range scores {
		result[score.Status] = append(result[score.Status], score)
	}

	return result, nil
}

// GetCriticalEndpoints returns only endpoints with critical status.
func (s *ScoringService) GetCriticalEndpoints(ctx context.Context) ([]*EndpointHealthScore, error) {
	scores, err := s.CalculateAllScores(ctx)
	if err != nil {
		return nil, err
	}

	var critical []*EndpointHealthScore
	for _, score := range scores {
		if score.Status == StatusCritical {
			critical = append(critical, score)
		}
	}

	return critical, nil
}
