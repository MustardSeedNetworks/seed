package reporting

// services_scheduler.go contains SchedulerService: a persisted scheduler that
// loads ScheduledReport rows from the DB, ticks once per minute, and fires
// GenerateFromTemplate when a schedule's NextRun is past.

import (
	"context"
	"fmt"
	"slices"
	"strings"
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
	logger := logging.GetLogger()
	if _, err := s.generator.GenerateFromTemplate(
		ctx,
		schedule.Template,
		schedule.Format,
		&schedule.Parameters,
	); err != nil {
		logger.ErrorContext(ctx, "scheduled report failed",
			"event", "report.schedule.failed", "schedule_id", schedule.ID, "error", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stamp the entry that is live now, not the one that was due: an edit made
	// while the report generated replaced it, and saving the old copy would
	// revert the edit. A deleted schedule is not saved at all, because the
	// upsert would bring its row back.
	live, ok := s.schedules[schedule.ID]
	if !ok {
		return
	}
	now := time.Now()
	live.LastRun = &now
	live.NextRun = calculateNextRun(&live.Schedule, now)
	live.UpdatedAt = now

	if err := s.saveSchedule(ctx, live); err != nil {
		logger.ErrorContext(ctx, "saving scheduled report failed",
			"event", "report.schedule.save_failed", "schedule_id", schedule.ID, "error", err)
	}
}

// calculateNextRun returns the first occurrence of schedule strictly after now,
// in the schedule's timezone. A time still ahead today counts: a daily report
// set for 17:00 at 09:00 runs this evening, not tomorrow.
func calculateNextRun(schedule *Schedule, now time.Time) *time.Time {
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.Local
	}
	now = now.In(loc)
	at := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, schedule.Hour, schedule.Minute, 0, 0, loc)
	}

	var next time.Time
	switch schedule.Frequency {
	case FrequencyDaily:
		next = at(now.Year(), now.Month(), now.Day())
		if !next.After(now) {
			next = at(now.Year(), now.Month(), now.Day()+1)
		}
	case FrequencyWeekly:
		days := 0
		if schedule.DayOfWeek != nil {
			days = (*schedule.DayOfWeek - int(now.Weekday()) + daysInWeek) % daysInWeek
		}
		next = at(now.Year(), now.Month(), now.Day()+days)
		if !next.After(now) {
			next = at(now.Year(), now.Month(), now.Day()+days+daysInWeek)
		}
	case FrequencyMonthly:
		day := 1
		if schedule.DayOfMonth != nil {
			day = *schedule.DayOfMonth
		}
		next = at(now.Year(), now.Month(), day)
		if !next.After(now) {
			next = at(now.Year(), now.Month()+1, day)
		}
	}

	return &next
}

// Schedule field bounds. Day of month stops at 28 so that every month has the
// day: [time.Date] would carry the 31st of a short month into the next one.
const (
	maxHour       = 23
	maxMinute     = 59
	maxDayOfWeek  = 6
	maxDayOfMonth = 28
)

// validate rejects a schedule the tick loop could not run as written: an
// unknown template, a format the template does not produce, or a time that
// does not exist. An unknown frequency would leave NextRun at the zero time,
// which is always due, so the report would fire every tick.
//
// Reasons name the field, never its value: the caller sent the value, and a
// request-derived string in the error would reach a log line.
func (s *SchedulerService) validate(sr *ScheduledReport) error {
	if sr == nil {
		return fmt.Errorf("%w: scheduled report is nil", ErrInvalidSchedule)
	}
	if strings.TrimSpace(sr.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidSchedule)
	}
	tmpl, ok := s.generator.templates.Get(sr.Template)
	if !ok {
		return fmt.Errorf("%w: unknown template", ErrInvalidSchedule)
	}
	if !slices.Contains(tmpl.Formats, sr.Format) {
		return fmt.Errorf("%w: the template does not produce this format", ErrInvalidSchedule)
	}
	return sr.Schedule.validate()
}

// validate checks the timing fields against the frequency they belong to.
func (sch *Schedule) validate() error {
	if sch.Hour < 0 || sch.Hour > maxHour || sch.Minute < 0 || sch.Minute > maxMinute {
		return fmt.Errorf("%w: hour must be 0-23 and minute 0-59", ErrInvalidSchedule)
	}
	if _, err := time.LoadLocation(sch.Timezone); err != nil {
		return fmt.Errorf("%w: unknown timezone", ErrInvalidSchedule)
	}
	weekly := sch.Frequency == FrequencyWeekly
	monthly := sch.Frequency == FrequencyMonthly
	switch {
	case sch.Frequency != FrequencyDaily && !weekly && !monthly:
		return fmt.Errorf("%w: frequency must be daily, weekly or monthly", ErrInvalidSchedule)
	case weekly != (sch.DayOfWeek != nil):
		return fmt.Errorf("%w: dayOfWeek is required for weekly and only for weekly", ErrInvalidSchedule)
	case monthly != (sch.DayOfMonth != nil):
		return fmt.Errorf("%w: dayOfMonth is required for monthly and only for monthly", ErrInvalidSchedule)
	case weekly && (*sch.DayOfWeek < 0 || *sch.DayOfWeek > maxDayOfWeek):
		return fmt.Errorf("%w: dayOfWeek must be 0-6", ErrInvalidSchedule)
	case monthly && (*sch.DayOfMonth < 1 || *sch.DayOfMonth > maxDayOfMonth):
		return fmt.Errorf("%w: dayOfMonth must be 1-28", ErrInvalidSchedule)
	}
	return nil
}

// Create validates and adds a scheduled report, filling in its ID, timestamps
// and first NextRun on sr. The scheduler keeps its own copy, so the caller may
// go on reading sr while the tick loop stamps runs.
func (s *SchedulerService) Create(ctx context.Context, sr *ScheduledReport) error {
	if err := s.validate(sr); err != nil {
		return err
	}
	if sr.ID == "" {
		sr.ID = uuid.New().String()
	}

	now := time.Now()
	sr.CreatedAt = now
	sr.UpdatedAt = now
	sr.LastRun = nil
	sr.NextRun = calculateNextRun(&sr.Schedule, now)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.saveSchedule(ctx, sr); err != nil {
		return err
	}
	stored := *sr
	s.schedules[sr.ID] = &stored

	return nil
}

func (s *SchedulerService) saveSchedule(ctx context.Context, sr *ScheduledReport) error {
	return s.repo.SaveSchedule(ctx, sr)
}

// Get returns a copy of a scheduled report.
func (s *SchedulerService) Get(_ context.Context, id string) (*ScheduledReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sr, ok := s.schedules[id]
	if !ok {
		return nil, ErrScheduleNotFound
	}
	snapshot := *sr
	return &snapshot, nil
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

// Update validates sr and replaces the scheduled report with its ID. CreatedAt
// and LastRun belong to the scheduler and are carried over from the stored
// entry; NextRun is recomputed from the new schedule.
func (s *SchedulerService) Update(ctx context.Context, sr *ScheduledReport) error {
	if err := s.validate(sr); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.schedules[sr.ID]
	if !ok {
		return ErrScheduleNotFound
	}

	now := time.Now()
	sr.CreatedAt = existing.CreatedAt
	sr.LastRun = existing.LastRun
	sr.UpdatedAt = now
	sr.NextRun = calculateNextRun(&sr.Schedule, now)

	if err := s.saveSchedule(ctx, sr); err != nil {
		return err
	}
	stored := *sr
	s.schedules[sr.ID] = &stored

	return nil
}

// Delete removes a scheduled report.
func (s *SchedulerService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schedules[id]; !ok {
		return ErrScheduleNotFound
	}

	if err := s.repo.DeleteSchedule(ctx, id); err != nil {
		return err
	}
	delete(s.schedules, id)
	return nil
}
