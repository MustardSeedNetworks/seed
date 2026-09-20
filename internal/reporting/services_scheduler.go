package reporting

// services_scheduler.go contains SchedulerService: a persisted scheduler that
// loads ScheduledReport rows from the DB, ticks once per minute, and fires
// GenerateFromTemplate when a schedule's NextRun is past.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/supervise"
	"github.com/google/uuid"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// SchedulerService manages scheduled reports. It keeps schedules in memory and
// ticks once per minute; row persistence goes through the ScheduleRepo port.
type SchedulerService struct {
	cfg       *config.Config
	repo      ScheduleRepo
	generator *GeneratorService
	tick      time.Duration
	mu        sync.RWMutex
	schedules map[string]*ScheduledReport
}

// SchedulerOption tunes a SchedulerService.
type SchedulerOption func(*SchedulerService)

// WithTickInterval sets how often Run looks for due schedules (<= 0 keeps the
// default minute). The same seam visibility.WithEvalInterval provides: a
// minute-long tick makes the fan-out untestable in anything but a slow test.
func WithTickInterval(d time.Duration) SchedulerOption {
	return func(s *SchedulerService) {
		if d > 0 {
			s.tick = d
		}
	}
}

// defaultSchedulerTick is how often the scheduler looks for due reports. A
// schedule's resolution is an hour at finest, so a minute is ample.
const defaultSchedulerTick = time.Minute

// NewSchedulerService creates a new scheduler service.
func NewSchedulerService(
	cfg *config.Config,
	repo ScheduleRepo,
	generator *GeneratorService,
	opts ...SchedulerOption,
) *SchedulerService {
	s := &SchedulerService{
		cfg:       cfg,
		repo:      repo,
		generator: generator,
		tick:      defaultSchedulerTick,
		schedules: make(map[string]*ScheduledReport),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Load reads the persisted schedules into memory. Called once, synchronously,
// before Run: a database that cannot be read is a startup failure, not a
// worker fault (#2748).
func (s *SchedulerService) Load(ctx context.Context) error {
	if err := s.loadSchedules(ctx); err != nil {
		return fmt.Errorf("loading schedules: %w", err)
	}
	return nil
}

// Run ticks once a minute until ctx is cancelled, firing every schedule whose
// NextRun has passed. It blocks: the scheduler is a supervised worker (#2748),
// and the `go s.runScheduler(ctx)` it replaced put the tick loop outside the
// supervisor's recover, so a panic while checking schedules took the daemon
// down.
func (s *SchedulerService) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.checkSchedules(ctx)
		}
	}
}

func (s *SchedulerService) loadSchedules(ctx context.Context) error {
	schedules, err := s.repo.ListSchedules(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range schedules {
		s.schedules[schedules[i].ID] = &schedules[i]
	}

	return nil
}

// checkSchedules fires every schedule whose NextRun has passed. Each firing
// runs under its own supervisor rather than a bare `go`: generation is the
// deepest call in the component (templates, aggregation, export) and a panic
// there used to kill the daemon even once the tick loop was supervised (#2748).
//
// One group PER report, not one for the cycle: a group stops its remaining
// workers when one fails, so a single bad template would cancel the other
// reports due in the same minute. supervise.Fatal for the same reason the
// restart count is absent — the ticker is the retry. A panic unwinds before
// NextRun is stamped, so the schedule stays due and fires again next minute,
// once a minute, visibly.
func (s *SchedulerService) checkSchedules(ctx context.Context) {
	for _, schedule := range s.dueSchedules(time.Now()) {
		group := supervise.New(logging.GetLogger())
		group.Add("scheduled-report:"+schedule.ID, supervise.Fatal,
			func(reportCtx context.Context) error {
				s.runScheduledReport(reportCtx, schedule)
				return nil
			})
		group.Start(ctx)
	}
}

// dueSchedules returns the enabled schedules whose NextRun is in the past. It
// takes and releases the read lock before anything is generated: runScheduledReport
// takes the write lock to stamp LastRun, which would deadlock under a held RLock.
func (s *SchedulerService) dueSchedules(now time.Time) []*ScheduledReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var due []*ScheduledReport
	for _, schedule := range s.schedules {
		if !schedule.Enabled {
			continue
		}
		if schedule.NextRun != nil && now.After(*schedule.NextRun) {
			due = append(due, schedule)
		}
	}
	return due
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
	return s.repo.SaveSchedule(ctx, sr)
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
	return s.repo.DeleteSchedule(ctx, id)
}
