package harvest

// services_scheduler.go contains SchedulerService: a persisted scheduler that
// loads ScheduledReport rows from the DB, ticks once per minute, and fires
// GenerateFromTemplate when a schedule's NextRun is past.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
)

// SchedulerService manages scheduled reports.
type SchedulerService struct {
	cfg       *config.Config
	db        *database.DB
	generator *GeneratorService
	cancel    context.CancelFunc
	mu        sync.RWMutex
	schedules map[string]*ScheduledReport
}

// NewSchedulerService creates a new scheduler service.
func NewSchedulerService(
	cfg *config.Config,
	db *database.DB,
	generator *GeneratorService,
) *SchedulerService {
	return &SchedulerService{
		cfg:       cfg,
		db:        db,
		generator: generator,
		schedules: make(map[string]*ScheduledReport),
	}
}

// Start begins the scheduler.
func (s *SchedulerService) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)

	// Load existing schedules
	if err := s.loadSchedules(ctx); err != nil {
		return fmt.Errorf("loading schedules: %w", err)
	}

	// Start scheduler loop
	go s.runScheduler(ctx)

	return nil
}

func (s *SchedulerService) loadSchedules(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, template, format, schedule_json, parameters_json, recipients_json, enabled, last_run, next_run, created_at, updated_at
		FROM scheduled_reports
	`)
	if err != nil {
		return fmt.Errorf("querying scheduled reports: %w", err)
	}
	defer rows.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	for rows.Next() {
		var sr ScheduledReport
		var scheduleJSON, paramsJSON, recipientsJSON string
		var lastRun, nextRun *string

		scanErr := rows.Scan(
			&sr.ID,
			&sr.Name,
			&sr.Template,
			&sr.Format,
			&scheduleJSON,
			&paramsJSON,
			&recipientsJSON,
			&sr.Enabled,
			&lastRun,
			&nextRun,
			&sr.CreatedAt,
			&sr.UpdatedAt,
		)
		if scanErr != nil {
			continue
		}

		_ = json.Unmarshal([]byte(scheduleJSON), &sr.Schedule)
		_ = json.Unmarshal([]byte(paramsJSON), &sr.Parameters)
		_ = json.Unmarshal([]byte(recipientsJSON), &sr.Recipients)

		if lastRun != nil {
			t, _ := time.Parse(time.RFC3339, *lastRun)
			sr.LastRun = &t
		}
		if nextRun != nil {
			t, _ := time.Parse(time.RFC3339, *nextRun)
			sr.NextRun = &t
		}

		s.schedules[sr.ID] = &sr
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return fmt.Errorf("iterating scheduled reports: %w", rowsErr)
	}

	return nil
}

func (s *SchedulerService) runScheduler(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkSchedules(ctx)
		}
	}
}

func (s *SchedulerService) checkSchedules(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, schedule := range s.schedules {
		if !schedule.Enabled {
			continue
		}
		if schedule.NextRun != nil && now.After(*schedule.NextRun) {
			go s.runScheduledReport(ctx, schedule)
		}
	}
}

func (s *SchedulerService) runScheduledReport(ctx context.Context, schedule *ScheduledReport) {
	// Generate report
	_, _ = s.generator.GenerateFromTemplate(
		ctx,
		schedule.Template,
		schedule.Format,
		&schedule.Parameters,
	)

	// Update last run and calculate next run
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	schedule.LastRun = &now
	schedule.NextRun = calculateNextRun(&schedule.Schedule)
	schedule.UpdatedAt = now

	_ = s.saveSchedule(ctx, schedule)
}

func calculateNextRun(schedule *Schedule) *time.Time {
	now := time.Now()

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.Local
	}

	var next time.Time
	switch schedule.Frequency {
	case FrequencyDaily:
		next = time.Date(
			now.Year(),
			now.Month(),
			now.Day()+1,
			schedule.Hour,
			schedule.Minute,
			0,
			0,
			loc,
		)
	case FrequencyWeekly:
		next = now
		if schedule.DayOfWeek != nil {
			daysUntil := (*schedule.DayOfWeek - int(now.Weekday()) + daysInWeek) % daysInWeek
			if daysUntil == 0 {
				daysUntil = daysInWeek
			}
			next = next.AddDate(0, 0, daysUntil)
		}
		next = time.Date(
			next.Year(),
			next.Month(),
			next.Day(),
			schedule.Hour,
			schedule.Minute,
			0,
			0,
			loc,
		)
	case FrequencyMonthly:
		day := 1
		if schedule.DayOfMonth != nil {
			day = *schedule.DayOfMonth
		}
		next = time.Date(now.Year(), now.Month()+1, day, schedule.Hour, schedule.Minute, 0, 0, loc)
	}

	return &next
}

// Stop halts the scheduler.
func (s *SchedulerService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// Create adds a scheduled report.
func (s *SchedulerService) Create(ctx context.Context, sr *ScheduledReport) error {
	if sr == nil {
		return errors.New("scheduled report is nil")
	}
	if sr.ID == "" {
		sr.ID = uuid.New().String()
	}

	sr.CreatedAt = time.Now()
	sr.UpdatedAt = time.Now()
	sr.NextRun = calculateNextRun(&sr.Schedule)

	if err := s.saveSchedule(ctx, sr); err != nil {
		return err
	}

	s.mu.Lock()
	s.schedules[sr.ID] = sr
	s.mu.Unlock()

	return nil
}

func (s *SchedulerService) saveSchedule(ctx context.Context, sr *ScheduledReport) error {
	scheduleJSON, _ := json.Marshal(sr.Schedule)
	paramsJSON, _ := json.Marshal(sr.Parameters)
	recipientsJSON, _ := json.Marshal(sr.Recipients)

	var lastRun, nextRun *string
	if sr.LastRun != nil {
		t := sr.LastRun.Format(time.RFC3339)
		lastRun = &t
	}
	if sr.NextRun != nil {
		t := sr.NextRun.Format(time.RFC3339)
		nextRun = &t
	}

	_, err := s.db.Exec(
		ctx,
		`
		INSERT OR REPLACE INTO scheduled_reports (id, name, template, format, schedule_json, parameters_json, recipients_json, enabled, last_run, next_run, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sr.ID,
		sr.Name,
		sr.Template,
		sr.Format,
		string(scheduleJSON),
		string(paramsJSON),
		string(recipientsJSON),
		sr.Enabled,
		lastRun,
		nextRun,
		sr.CreatedAt.Format(time.RFC3339),
		sr.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("saving scheduled report: %w", err)
	}

	return nil
}

// Get retrieves a scheduled report.
func (s *SchedulerService) Get(_ context.Context, id string) (*ScheduledReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sr, ok := s.schedules[id]
	if !ok {
		return nil, fmt.Errorf("scheduled report not found: %s", id)
	}
	return sr, nil
}

// List returns all scheduled reports.
func (s *SchedulerService) List(_ context.Context) ([]ScheduledReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ScheduledReport, 0, len(s.schedules))
	for _, sr := range s.schedules {
		result = append(result, *sr)
	}
	return result, nil
}

// Update modifies a scheduled report.
func (s *SchedulerService) Update(ctx context.Context, sr *ScheduledReport) error {
	if sr == nil {
		return errors.New("scheduled report is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schedules[sr.ID]; !ok {
		return fmt.Errorf("scheduled report not found: %s", sr.ID)
	}

	sr.UpdatedAt = time.Now()
	sr.NextRun = calculateNextRun(&sr.Schedule)
	s.schedules[sr.ID] = sr

	return s.saveSchedule(ctx, sr)
}

// Delete removes a scheduled report.
func (s *SchedulerService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schedules[id]; !ok {
		return fmt.Errorf("scheduled report not found: %s", id)
	}

	delete(s.schedules, id)
	_, err := s.db.Exec(ctx, "DELETE FROM scheduled_reports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting scheduled report: %w", err)
	}
	return nil
}
