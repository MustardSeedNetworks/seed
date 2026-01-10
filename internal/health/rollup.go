package health

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

// Rollup timing constants.
const (
	// HourlyRollupInterval is how often to run hourly rollups.
	HourlyRollupInterval = time.Hour

	// DailyRollupHour is the hour of day (UTC) to run daily rollups.
	DailyRollupHour = 2 // 2 AM UTC

	// RollupCheckInterval is how often to check if rollups are needed.
	RollupCheckInterval = 15 * time.Minute

	// MaxRollupBatchSize limits how many hours/days to process in one batch.
	MaxRollupBatchSize = 24

	// MinDataPointsForRollup is the minimum data points needed to create a rollup.
	MinDataPointsForRollup = 1

	// HoursPerDay is used for day truncation calculations.
	HoursPerDay = 24
)

// RollupService manages background rollup jobs for health check data.
// It aggregates raw health check results into hourly and daily rollups
// to support efficient querying of historical data.
type RollupService struct {
	db     *database.DB
	logger *slog.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Last rollup times for tracking progress
	lastHourlyRollup time.Time
	lastDailyRollup  time.Time
}

// NewRollupService creates a new rollup service.
func NewRollupService(db *database.DB, logger *slog.Logger) *RollupService {
	if logger == nil {
		logger = slog.Default()
	}
	return &RollupService{
		db:     db,
		logger: logger,
	}
}

// Start begins the rollup service background loop.
func (s *RollupService) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)

	// Initialize last rollup times from database
	if err := s.initializeRollupState(ctx); err != nil {
		s.logger.WarnContext(ctx, "failed to initialize rollup state, starting fresh", "error", err)
	}

	// Start the background rollup loop
	s.wg.Add(1)
	go s.runRollupLoop(ctx)

	s.logger.InfoContext(ctx, "health check rollup service started")
	return nil
}

// Stop gracefully shuts down the rollup service.
func (s *RollupService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.logger.Info("health check rollup service stopped")
}

// initializeRollupState loads the last rollup timestamps from the database.
//
//nolint:gocognit // initialization logic with nested loops is acceptable
func (s *RollupService) initializeRollupState(ctx context.Context) error {
	repo := s.db.HealthChecks()

	// Get all distinct check types and endpoints
	checkTypes, err := repo.GetDistinctCheckTypes(ctx)
	if err != nil || len(checkTypes) == 0 {
		return err
	}

	// Find the latest hourly rollup across all endpoints
	for _, checkType := range checkTypes {
		endpoints, endpointsErr := repo.GetDistinctEndpoints(ctx, checkType)
		if endpointsErr != nil {
			continue
		}

		for _, endpoint := range endpoints {
			rollups, rollupsErr := repo.GetHourlyRollups(ctx, checkType, endpoint, database.TimeRange{})
			if rollupsErr == nil && len(rollups) > 0 {
				lastRollup := rollups[len(rollups)-1]
				if lastRollup.HourBucket.After(s.lastHourlyRollup) {
					s.lastHourlyRollup = lastRollup.HourBucket
				}
			}

			dailyRollups, dailyErr := repo.GetDailyRollups(ctx, checkType, endpoint, database.TimeRange{})
			if dailyErr == nil && len(dailyRollups) > 0 {
				lastDaily := dailyRollups[len(dailyRollups)-1]
				if lastDaily.DayBucket.After(s.lastDailyRollup) {
					s.lastDailyRollup = lastDaily.DayBucket
				}
			}
		}
	}

	return nil
}

// runRollupLoop is the main background loop that checks and performs rollups.
func (s *RollupService) runRollupLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(RollupCheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.performRollups(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.performRollups(ctx)
		}
	}
}

// performRollups checks if rollups are needed and executes them.
func (s *RollupService) performRollups(ctx context.Context) {
	now := time.Now().UTC()

	// Check if hourly rollup is needed
	if s.shouldRunHourlyRollup(now) {
		if err := s.runHourlyRollups(ctx, now); err != nil {
			s.logger.ErrorContext(ctx, "hourly rollup failed", "error", err)
		}
	}

	// Check if daily rollup is needed (run at DailyRollupHour)
	if s.shouldRunDailyRollup(now) {
		if err := s.runDailyRollups(ctx, now); err != nil {
			s.logger.ErrorContext(ctx, "daily rollup failed", "error", err)
		}
	}
}

// shouldRunHourlyRollup determines if hourly rollup should run.
func (s *RollupService) shouldRunHourlyRollup(now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If never run, always run
	if s.lastHourlyRollup.IsZero() {
		return true
	}

	// Run if more than an hour since last rollup
	return now.Sub(s.lastHourlyRollup) >= HourlyRollupInterval
}

// shouldRunDailyRollup determines if daily rollup should run.
func (s *RollupService) shouldRunDailyRollup(now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Only run during the designated hour
	if now.Hour() != DailyRollupHour {
		return false
	}

	// If never run, always run
	if s.lastDailyRollup.IsZero() {
		return true
	}

	// Run if we haven't run today
	lastDay := s.lastDailyRollup.Truncate(HoursPerDay * time.Hour)
	today := now.Truncate(HoursPerDay * time.Hour)
	return today.After(lastDay)
}

// runHourlyRollups creates hourly aggregations for all pending hours.
func (s *RollupService) runHourlyRollups(ctx context.Context, now time.Time) error {
	repo := s.db.HealthChecks()
	currentHour := now.Truncate(time.Hour)

	s.mu.RLock()
	startHour := s.lastHourlyRollup
	s.mu.RUnlock()

	// If never run, start from 24 hours ago
	if startHour.IsZero() {
		startHour = currentHour.Add(-24 * time.Hour)
	} else {
		// Start from the next hour after last rollup
		startHour = startHour.Add(time.Hour)
	}

	// Don't process the current hour (incomplete data)
	if !startHour.Before(currentHour) {
		return nil
	}

	// Get all distinct endpoints
	endpoints, err := s.getDistinctEndpoints(ctx, repo)
	if err != nil {
		return fmt.Errorf("getting endpoints: %w", err)
	}

	hoursProcessed := 0
	for hour := startHour; hour.Before(currentHour) && hoursProcessed < MaxRollupBatchSize; hour = hour.Add(time.Hour) {
		for _, ep := range endpoints {
			if createErr := repo.CreateHourlyRollup(ctx, ep.CheckType, ep.EndpointName, hour); createErr != nil {
				s.logger.WarnContext(ctx, "failed to create hourly rollup",
					"endpoint", ep.EndpointName,
					"check_type", ep.CheckType,
					"hour", hour,
					"error", createErr)
			}
		}
		hoursProcessed++

		s.mu.Lock()
		s.lastHourlyRollup = hour
		s.mu.Unlock()
	}

	if hoursProcessed > 0 {
		s.logger.InfoContext(ctx, "hourly rollups completed", "hours_processed", hoursProcessed)
	}

	return nil
}

// runDailyRollups creates daily aggregations for all pending days.
func (s *RollupService) runDailyRollups(ctx context.Context, now time.Time) error {
	repo := s.db.HealthChecks()
	today := now.Truncate(HoursPerDay * time.Hour)

	s.mu.RLock()
	startDay := s.lastDailyRollup
	s.mu.RUnlock()

	// If never run, start from 7 days ago
	if startDay.IsZero() {
		startDay = today.AddDate(0, 0, -7)
	} else {
		// Start from the next day after last rollup
		startDay = startDay.AddDate(0, 0, 1)
	}

	// Don't process today (incomplete data)
	if !startDay.Before(today) {
		return nil
	}

	// Get all distinct endpoints
	endpoints, err := s.getDistinctEndpoints(ctx, repo)
	if err != nil {
		return fmt.Errorf("getting endpoints: %w", err)
	}

	daysProcessed := 0
	for day := startDay; day.Before(today) && daysProcessed < MaxRollupBatchSize; day = day.AddDate(0, 0, 1) {
		for _, ep := range endpoints {
			if createErr := repo.CreateDailyRollup(ctx, ep.CheckType, ep.EndpointName, day); createErr != nil {
				s.logger.WarnContext(ctx, "failed to create daily rollup",
					"endpoint", ep.EndpointName,
					"check_type", ep.CheckType,
					"day", day,
					"error", createErr)
			}
		}
		daysProcessed++

		s.mu.Lock()
		s.lastDailyRollup = day
		s.mu.Unlock()
	}

	if daysProcessed > 0 {
		s.logger.InfoContext(ctx, "daily rollups completed", "days_processed", daysProcessed)
	}

	return nil
}

// endpointInfo holds endpoint identification for rollup processing.
type endpointInfo struct {
	CheckType    string
	EndpointName string
}

// getDistinctEndpoints returns unique endpoint/check type combinations.
func (s *RollupService) getDistinctEndpoints(
	ctx context.Context,
	repo *database.HealthCheckRepository,
) ([]endpointInfo, error) {
	// Get all check types
	checkTypes, err := repo.GetDistinctCheckTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting check types: %w", err)
	}

	var endpoints []endpointInfo
	for _, checkType := range checkTypes {
		names, namesErr := repo.GetDistinctEndpoints(ctx, checkType)
		if namesErr != nil {
			continue
		}
		for _, name := range names {
			endpoints = append(endpoints, endpointInfo{
				CheckType:    checkType,
				EndpointName: name,
			})
		}
	}

	return endpoints, nil
}

// RunManualRollup allows triggering rollups on-demand (useful for testing).
func (s *RollupService) RunManualRollup(ctx context.Context) error {
	now := time.Now().UTC()

	if err := s.runHourlyRollups(ctx, now); err != nil {
		return fmt.Errorf("hourly rollups: %w", err)
	}

	if err := s.runDailyRollups(ctx, now); err != nil {
		return fmt.Errorf("daily rollups: %w", err)
	}

	return nil
}

// GetRollupStatus returns the current rollup state for monitoring.
func (s *RollupService) GetRollupStatus() RollupStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return RollupStatus{
		LastHourlyRollup: s.lastHourlyRollup,
		LastDailyRollup:  s.lastDailyRollup,
	}
}

// RollupStatus represents the current state of rollup processing.
type RollupStatus struct {
	LastHourlyRollup time.Time `json:"lastHourlyRollup"`
	LastDailyRollup  time.Time `json:"lastDailyRollup"`
}
