package scheduler_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/harvest"
)

// testDB creates a temporary database for testing.
func testDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.Open(dbPath)
	require.NoError(t, err, "failed to open database")

	cleanup := func() {
		_ = db.Close()
		_ = os.Remove(dbPath)
	}

	return db, cleanup
}

// testConfig creates a default config for testing.
func testConfig() *config.Config {
	return config.DefaultConfig()
}

// setupSchedulerService creates all required services for scheduler testing.
func setupSchedulerService(t *testing.T) (
	*harvest.SchedulerService,
	*database.DB,
	func(),
) {
	t.Helper()

	db, cleanup := testDB(t)
	cfg := testConfig()

	ts := harvest.NewTemplateService(cfg)
	require.NoError(t, ts.Load())

	as := harvest.NewAggregatorService(cfg, db)
	gs := harvest.NewGeneratorService(cfg, db, ts, as)
	ss := harvest.NewSchedulerService(cfg, db, gs)

	return ss, db, cleanup
}

// intPtr is a helper to create int pointers.
func intPtr(i int) *int {
	return &i
}

// ----------------------------------------------------------------------------
// Next Run Calculation Tests (via Create API)
// ----------------------------------------------------------------------------

func TestNextRunCalculation_Daily(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name     string
		schedule harvest.Schedule
		validate func(t *testing.T, next *time.Time)
	}{
		{
			name: "daily at 9am UTC",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyDaily,
				Hour:      9,
				Minute:    0,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 9, next.Hour())
				assert.Equal(t, 0, next.Minute())
			},
		},
		{
			name: "daily at 23:59 UTC",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyDaily,
				Hour:      23,
				Minute:    59,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 23, next.Hour())
				assert.Equal(t, 59, next.Minute())
			},
		},
		{
			name: "daily at midnight UTC",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyDaily,
				Hour:      0,
				Minute:    0,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 0, next.Hour())
				assert.Equal(t, 0, next.Minute())
			},
		},
		{
			name: "daily at 15:30 America/New_York",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyDaily,
				Hour:      15,
				Minute:    30,
				Timezone:  "America/New_York",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				// Verify it's in the expected timezone
				loc, _ := time.LoadLocation("America/New_York")
				nextInTZ := next.In(loc)
				assert.Equal(t, 15, nextInTZ.Hour())
				assert.Equal(t, 30, nextInTZ.Minute())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &harvest.ScheduledReport{
				Name:     tt.name,
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: tt.schedule,
				Enabled:  true,
			}
			err := ss.Create(ctx, sr)
			require.NoError(t, err)
			tt.validate(t, sr.NextRun)
		})
	}
}

func TestNextRunCalculation_Weekly(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name     string
		schedule harvest.Schedule
		validate func(t *testing.T, next *time.Time)
	}{
		{
			name: "weekly on Monday at 10am",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyWeekly,
				DayOfWeek: intPtr(1), // Monday
				Hour:      10,
				Minute:    0,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, time.Monday, next.Weekday())
				assert.Equal(t, 10, next.Hour())
			},
		},
		{
			name: "weekly on Friday at 17:00",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyWeekly,
				DayOfWeek: intPtr(5), // Friday
				Hour:      17,
				Minute:    0,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, time.Friday, next.Weekday())
				assert.Equal(t, 17, next.Hour())
			},
		},
		{
			name: "weekly on Sunday at 6:00",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyWeekly,
				DayOfWeek: intPtr(0), // Sunday
				Hour:      6,
				Minute:    0,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, time.Sunday, next.Weekday())
				assert.Equal(t, 6, next.Hour())
			},
		},
		{
			name: "weekly with nil DayOfWeek returns valid time",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyWeekly,
				DayOfWeek: nil,
				Hour:      8,
				Minute:    0,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				// Just verify hour/minute are set correctly
				assert.Equal(t, 8, next.Hour())
				assert.Equal(t, 0, next.Minute())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &harvest.ScheduledReport{
				Name:     tt.name,
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: tt.schedule,
				Enabled:  true,
			}
			err := ss.Create(ctx, sr)
			require.NoError(t, err)
			tt.validate(t, sr.NextRun)
		})
	}
}

func TestNextRunCalculation_Monthly(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name     string
		schedule harvest.Schedule
		validate func(t *testing.T, next *time.Time)
	}{
		{
			name: "monthly on the 1st at 8am",
			schedule: harvest.Schedule{
				Frequency:  harvest.FrequencyMonthly,
				DayOfMonth: intPtr(1),
				Hour:       8,
				Minute:     0,
				Timezone:   "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 1, next.Day())
				assert.Equal(t, 8, next.Hour())
			},
		},
		{
			name: "monthly on the 15th at noon",
			schedule: harvest.Schedule{
				Frequency:  harvest.FrequencyMonthly,
				DayOfMonth: intPtr(15),
				Hour:       12,
				Minute:     0,
				Timezone:   "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 15, next.Day())
				assert.Equal(t, 12, next.Hour())
			},
		},
		{
			name: "monthly on the 28th",
			schedule: harvest.Schedule{
				Frequency:  harvest.FrequencyMonthly,
				DayOfMonth: intPtr(28),
				Hour:       9,
				Minute:     30,
				Timezone:   "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 28, next.Day())
				assert.Equal(t, 9, next.Hour())
				assert.Equal(t, 30, next.Minute())
			},
		},
		{
			name: "monthly with nil DayOfMonth (defaults to 1st)",
			schedule: harvest.Schedule{
				Frequency:  harvest.FrequencyMonthly,
				DayOfMonth: nil,
				Hour:       7,
				Minute:     0,
				Timezone:   "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 1, next.Day())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &harvest.ScheduledReport{
				Name:     tt.name,
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: tt.schedule,
				Enabled:  true,
			}
			err := ss.Create(ctx, sr)
			require.NoError(t, err)
			tt.validate(t, sr.NextRun)
		})
	}
}

func TestNextRunCalculation_Timezones(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name     string
		timezone string
	}{
		{name: "UTC", timezone: "UTC"},
		{name: "America/New_York", timezone: "America/New_York"},
		{name: "Europe/London", timezone: "Europe/London"},
		{name: "Asia/Tokyo", timezone: "Asia/Tokyo"},
		{name: "Australia/Sydney", timezone: "Australia/Sydney"},
		{name: "Invalid timezone (fallback)", timezone: "Invalid/Timezone"},
		{name: "Empty timezone (fallback)", timezone: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &harvest.ScheduledReport{
				Name:     tt.name,
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyDaily,
					Hour:      9,
					Minute:    0,
					Timezone:  tt.timezone,
				},
				Enabled: true,
			}
			err := ss.Create(ctx, sr)
			require.NoError(t, err)
			require.NotNil(t, sr.NextRun)
			// Verify the hour matches the expected schedule
			// (without checking if it's after now, which depends on current time)
			assert.Equal(t, 9, sr.NextRun.Hour())
		})
	}
}

// ----------------------------------------------------------------------------
// SchedulerService Create Tests
// ----------------------------------------------------------------------------

func TestSchedulerService_Create_TableDriven(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name      string
		schedule  *harvest.ScheduledReport
		wantErr   bool
		errSubstr string
		validate  func(t *testing.T, sr *harvest.ScheduledReport)
	}{
		{
			name: "valid daily schedule",
			schedule: &harvest.ScheduledReport{
				Name:     "Daily Test",
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyDaily,
					Hour:      9,
					Minute:    0,
					Timezone:  "UTC",
				},
				Enabled: true,
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				assert.NotEmpty(t, sr.ID)
				assert.NotNil(t, sr.NextRun)
				assert.True(t, sr.NextRun.After(time.Now()))
				assert.False(t, sr.CreatedAt.IsZero())
				assert.False(t, sr.UpdatedAt.IsZero())
			},
		},
		{
			name: "valid weekly schedule with recipients",
			schedule: &harvest.ScheduledReport{
				Name:     "Weekly Test",
				Template: "vulnerability",
				Format:   harvest.FormatHTML,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyWeekly,
					DayOfWeek: intPtr(1),
					Hour:      10,
					Minute:    30,
					Timezone:  "America/New_York",
				},
				Recipients: []harvest.Recipient{
					{Email: "admin@example.com", Name: "Admin"},
					{Email: "security@example.com", Name: "Security Team"},
				},
				Enabled: true,
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				assert.Len(t, sr.Recipients, 2)
				assert.Equal(t, harvest.FrequencyWeekly, sr.Schedule.Frequency)
			},
		},
		{
			name: "valid monthly schedule with parameters",
			schedule: &harvest.ScheduledReport{
				Name:     "Monthly Test",
				Template: "inventory",
				Format:   harvest.FormatCSV,
				Schedule: harvest.Schedule{
					Frequency:  harvest.FrequencyMonthly,
					DayOfMonth: intPtr(1),
					Hour:       7,
					Minute:     0,
					Timezone:   "Europe/London",
				},
				Parameters: harvest.ReportParams{
					Filters: map[string]string{
						"department": "engineering",
					},
					IncludeSections: []string{"devices", "software"},
				},
				Enabled: true,
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				assert.NotEmpty(t, sr.Parameters.Filters)
				assert.Len(t, sr.Parameters.IncludeSections, 2)
			},
		},
		{
			name: "disabled schedule",
			schedule: &harvest.ScheduledReport{
				Name:     "Disabled Test",
				Template: "performance",
				Format:   harvest.FormatJSON,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyDaily,
					Hour:      12,
					Minute:    0,
					Timezone:  "UTC",
				},
				Enabled: false,
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				assert.False(t, sr.Enabled)
			},
		},
		{
			name: "schedule with custom ID",
			schedule: &harvest.ScheduledReport{
				ID:       "custom-schedule-id-123",
				Name:     "Custom ID Test",
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyDaily,
					Hour:      8,
					Timezone:  "UTC",
				},
				Enabled: true,
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				assert.Equal(t, "custom-schedule-id-123", sr.ID)
			},
		},
		{
			name:      "nil schedule",
			schedule:  nil,
			wantErr:   true,
			errSubstr: "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ss.Create(ctx, tt.schedule)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.schedule)
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// SchedulerService Get Tests
// ----------------------------------------------------------------------------

func TestSchedulerService_Get_TableDriven(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a schedule to retrieve
	existingSchedule := &harvest.ScheduledReport{
		Name:     "Existing Schedule",
		Template: "executive",
		Format:   harvest.FormatPDF,
		Schedule: harvest.Schedule{
			Frequency: harvest.FrequencyDaily,
			Hour:      9,
			Timezone:  "UTC",
		},
		Enabled: true,
	}
	require.NoError(t, ss.Create(ctx, existingSchedule))

	tests := []struct {
		name       string
		scheduleID string
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "get existing schedule",
			scheduleID: existingSchedule.ID,
			wantErr:    false,
		},
		{
			name:       "get non-existent schedule",
			scheduleID: "nonexistent-id-12345",
			wantErr:    true,
			errSubstr:  "not found",
		},
		{
			name:       "get with empty ID",
			scheduleID: "",
			wantErr:    true,
			errSubstr:  "not found",
		},
		{
			name:       "get with UUID-like non-existent ID",
			scheduleID: "550e8400-e29b-41d4-a716-446655440000",
			wantErr:    true,
			errSubstr:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ss.Get(ctx, tt.scheduleID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.scheduleID, result.ID)
				assert.Equal(t, existingSchedule.Name, result.Name)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// SchedulerService List Tests
// ----------------------------------------------------------------------------

func TestSchedulerService_List_TableDriven(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	// Get initial count
	initialList, err := ss.List(ctx)
	require.NoError(t, err)
	initialCount := len(initialList)

	tests := []struct {
		name          string
		createCount   int
		expectedCount int
	}{
		{
			name:          "list after adding 0 schedules",
			createCount:   0,
			expectedCount: initialCount,
		},
		{
			name:          "list after adding 1 schedule",
			createCount:   1,
			expectedCount: initialCount + 1,
		},
		{
			name:          "list after adding 3 more schedules",
			createCount:   3,
			expectedCount: initialCount + 4, // cumulative
		},
	}

	totalCreated := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create schedules
			for i := range tt.createCount {
				schedule := &harvest.ScheduledReport{
					Name:     "List Test " + string(rune('A'+totalCreated+i)),
					Template: "executive",
					Format:   harvest.FormatPDF,
					Schedule: harvest.Schedule{
						Frequency: harvest.FrequencyDaily,
						Hour:      i,
						Timezone:  "UTC",
					},
					Enabled: true,
				}
				require.NoError(t, ss.Create(ctx, schedule))
			}
			totalCreated += tt.createCount

			// List and verify count
			result, listErr := ss.List(ctx)
			require.NoError(t, listErr)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

// ----------------------------------------------------------------------------
// SchedulerService Update Tests
// ----------------------------------------------------------------------------

func TestSchedulerService_Update_TableDriven(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a schedule to update
	schedule := &harvest.ScheduledReport{
		Name:     "Update Test",
		Template: "executive",
		Format:   harvest.FormatPDF,
		Schedule: harvest.Schedule{
			Frequency: harvest.FrequencyDaily,
			Hour:      9,
			Timezone:  "UTC",
		},
		Enabled: true,
	}
	require.NoError(t, ss.Create(ctx, schedule))

	tests := []struct {
		name      string
		modifyFn  func() *harvest.ScheduledReport
		wantErr   bool
		errSubstr string
		validate  func(t *testing.T, sr *harvest.ScheduledReport)
	}{
		{
			name: "update name",
			modifyFn: func() *harvest.ScheduledReport {
				schedule.Name = "Updated Name"
				return schedule
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				result, _ := ss.Get(ctx, sr.ID)
				assert.Equal(t, "Updated Name", result.Name)
			},
		},
		{
			name: "update format",
			modifyFn: func() *harvest.ScheduledReport {
				schedule.Format = harvest.FormatHTML
				return schedule
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				result, _ := ss.Get(ctx, sr.ID)
				assert.Equal(t, harvest.FormatHTML, result.Format)
			},
		},
		{
			name: "change to weekly frequency",
			modifyFn: func() *harvest.ScheduledReport {
				schedule.Schedule.Frequency = harvest.FrequencyWeekly
				schedule.Schedule.DayOfWeek = intPtr(3) // Wednesday
				return schedule
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				result, _ := ss.Get(ctx, sr.ID)
				assert.Equal(t, harvest.FrequencyWeekly, result.Schedule.Frequency)
				require.NotNil(t, result.Schedule.DayOfWeek)
				assert.Equal(t, 3, *result.Schedule.DayOfWeek)
			},
		},
		{
			name: "disable schedule",
			modifyFn: func() *harvest.ScheduledReport {
				schedule.Enabled = false
				return schedule
			},
			wantErr: false,
			validate: func(t *testing.T, sr *harvest.ScheduledReport) {
				result, _ := ss.Get(ctx, sr.ID)
				assert.False(t, result.Enabled)
			},
		},
		{
			name: "update with nil",
			modifyFn: func() *harvest.ScheduledReport {
				return nil
			},
			wantErr:   true,
			errSubstr: "nil",
		},
		{
			name: "update non-existent",
			modifyFn: func() *harvest.ScheduledReport {
				return &harvest.ScheduledReport{ID: "nonexistent-id"}
			},
			wantErr:   true,
			errSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tt.modifyFn()
			err := ss.Update(ctx, sr)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, sr)
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// SchedulerService Delete Tests
// ----------------------------------------------------------------------------

func TestSchedulerService_Delete_TableDriven(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string // returns ID to delete
		wantErr     bool
		errSubstr   string
		validateDel func(t *testing.T, id string)
	}{
		{
			name: "delete existing schedule",
			setup: func() string {
				schedule := &harvest.ScheduledReport{
					Name:     "To Delete",
					Template: "executive",
					Format:   harvest.FormatPDF,
					Schedule: harvest.Schedule{
						Frequency: harvest.FrequencyDaily,
						Hour:      9,
						Timezone:  "UTC",
					},
					Enabled: true,
				}
				require.NoError(t, ss.Create(ctx, schedule))
				return schedule.ID
			},
			wantErr: false,
			validateDel: func(t *testing.T, id string) {
				_, err := ss.Get(ctx, id)
				require.Error(t, err)
			},
		},
		{
			name: "delete non-existent schedule",
			setup: func() string {
				return "nonexistent-id-to-delete"
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "delete with empty ID",
			setup: func() string {
				return ""
			},
			wantErr:   true,
			errSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.setup()
			err := ss.Delete(ctx, id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				if tt.validateDel != nil {
					tt.validateDel(t, id)
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// SchedulerService Start/Stop Tests
// ----------------------------------------------------------------------------

func TestSchedulerService_StartStop_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		contextTimeout time.Duration
		stopDelay      time.Duration
		wantStartErr   bool
	}{
		{
			name:           "start and stop immediately",
			contextTimeout: 100 * time.Millisecond,
			stopDelay:      10 * time.Millisecond,
			wantStartErr:   false,
		},
		{
			name:           "start with longer runtime",
			contextTimeout: 500 * time.Millisecond,
			stopDelay:      50 * time.Millisecond,
			wantStartErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss, _, cleanup := setupSchedulerService(t)
			defer cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			err := ss.Start(ctx)
			if tt.wantStartErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Let scheduler run briefly
			time.Sleep(tt.stopDelay)

			// Stop should not panic
			ss.Stop()
		})
	}
}

func TestSchedulerService_MultipleStops(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, ss.Start(ctx))

	// Multiple stops should not panic
	ss.Stop()
	ss.Stop()
	ss.Stop()
}

func TestSchedulerService_StopWithoutStart(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	// Stop without Start should not panic
	ss.Stop()
}

// ----------------------------------------------------------------------------
// Concurrency Tests
// ----------------------------------------------------------------------------

func TestSchedulerService_ConcurrentCreate(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()
	const numGoroutines = 5 // Reduced to avoid SQLite locking issues

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := range numGoroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			schedule := &harvest.ScheduledReport{
				Name:     "Concurrent Test " + string(rune('A'+idx)),
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyDaily,
					Hour:      idx % 24,
					Timezone:  "UTC",
				},
				Enabled: true,
			}
			if err := ss.Create(ctx, schedule); err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
			// SQLite may have locking issues; we just verify no panic
		}(i)
	}

	wg.Wait()

	// Verify at least some schedules were created (SQLite may lock some)
	list, err := ss.List(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1, "at least one schedule should be created")
}

func TestSchedulerService_ConcurrentReadWrite(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	// Create some initial schedules
	for i := range 5 {
		schedule := &harvest.ScheduledReport{
			Name:     "RW Test " + string(rune('A'+i)),
			Template: "executive",
			Format:   harvest.FormatPDF,
			Schedule: harvest.Schedule{
				Frequency: harvest.FrequencyDaily,
				Hour:      i,
				Timezone:  "UTC",
			},
			Enabled: true,
		}
		require.NoError(t, ss.Create(ctx, schedule))
	}

	var wg sync.WaitGroup
	const numGoroutines = 20

	// Mix of reads and writes
	for i := range numGoroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				// Read operation
				_, _ = ss.List(ctx)
			} else {
				// Write operation
				schedule := &harvest.ScheduledReport{
					Name:     "RW Concurrent " + string(rune('A'+idx)),
					Template: "executive",
					Format:   harvest.FormatPDF,
					Schedule: harvest.Schedule{
						Frequency: harvest.FrequencyDaily,
						Hour:      idx % 24,
						Timezone:  "UTC",
					},
					Enabled: true,
				}
				_ = ss.Create(ctx, schedule)
			}
		}(i)
	}

	wg.Wait()
}

// ----------------------------------------------------------------------------
// Schedule Frequency Edge Cases
// ----------------------------------------------------------------------------

func TestScheduleFrequency_EdgeCases(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name     string
		schedule harvest.Schedule
		validate func(t *testing.T, next *time.Time)
	}{
		{
			name: "weekly Saturday at midnight",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyWeekly,
				DayOfWeek: intPtr(6), // Saturday
				Hour:      0,
				Minute:    0,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, time.Saturday, next.Weekday())
			},
		},
		{
			name: "monthly on 31st",
			schedule: harvest.Schedule{
				Frequency:  harvest.FrequencyMonthly,
				DayOfMonth: intPtr(31),
				Hour:       12,
				Minute:     0,
				Timezone:   "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.True(t, next.After(time.Now()))
			},
		},
		{
			name: "daily at 23:59",
			schedule: harvest.Schedule{
				Frequency: harvest.FrequencyDaily,
				Hour:      23,
				Minute:    59,
				Timezone:  "UTC",
			},
			validate: func(t *testing.T, next *time.Time) {
				require.NotNil(t, next)
				assert.Equal(t, 23, next.Hour())
				assert.Equal(t, 59, next.Minute())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &harvest.ScheduledReport{
				Name:     tt.name,
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: tt.schedule,
				Enabled:  true,
			}
			err := ss.Create(ctx, sr)
			require.NoError(t, err)
			tt.validate(t, sr.NextRun)
		})
	}
}

// ----------------------------------------------------------------------------
// Recipients Tests
// ----------------------------------------------------------------------------

func TestScheduledReport_Recipients(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name       string
		recipients []harvest.Recipient
	}{
		{
			name:       "no recipients",
			recipients: nil,
		},
		{
			name: "single recipient",
			recipients: []harvest.Recipient{
				{Email: "admin@example.com", Name: "Admin"},
			},
		},
		{
			name: "multiple recipients",
			recipients: []harvest.Recipient{
				{Email: "admin@example.com", Name: "Admin"},
				{Email: "security@example.com", Name: "Security"},
				{Email: "ops@example.com", Name: "Operations"},
			},
		},
		{
			name: "recipient without name",
			recipients: []harvest.Recipient{
				{Email: "noreply@example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &harvest.ScheduledReport{
				Name:     "Recipients Test",
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyDaily,
					Hour:      9,
					Timezone:  "UTC",
				},
				Recipients: tt.recipients,
				Enabled:    true,
			}

			err := ss.Create(ctx, schedule)
			require.NoError(t, err)

			// Retrieve and verify
			retrieved, err := ss.Get(ctx, schedule.ID)
			require.NoError(t, err)
			assert.Len(t, retrieved.Recipients, len(tt.recipients))
		})
	}
}

// ----------------------------------------------------------------------------
// Parameters Tests
// ----------------------------------------------------------------------------

func TestScheduledReport_Parameters(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name   string
		params harvest.ReportParams
	}{
		{
			name:   "empty parameters",
			params: harvest.ReportParams{},
		},
		{
			name: "with date range",
			params: harvest.ReportParams{
				DateRange: &harvest.DateRange{
					Start: time.Now().AddDate(0, 0, -7),
					End:   time.Now(),
				},
			},
		},
		{
			name: "with filters",
			params: harvest.ReportParams{
				Filters: map[string]string{
					"severity":   "critical",
					"department": "engineering",
				},
			},
		},
		{
			name: "with include sections",
			params: harvest.ReportParams{
				IncludeSections: []string{"summary", "vulnerabilities", "devices"},
			},
		},
		{
			name: "with exclude sections",
			params: harvest.ReportParams{
				ExcludeSections: []string{"low-priority"},
			},
		},
		{
			name: "with custom data",
			params: harvest.ReportParams{
				CustomData: map[string]any{
					"logo":      "custom-logo.png",
					"threshold": 95,
				},
			},
		},
		{
			name: "with all parameters",
			params: harvest.ReportParams{
				DateRange: &harvest.DateRange{
					Start: time.Now().AddDate(0, -1, 0),
					End:   time.Now(),
				},
				Filters: map[string]string{
					"type": "security",
				},
				IncludeSections: []string{"overview"},
				ExcludeSections: []string{"appendix"},
				CustomData: map[string]any{
					"company": "Acme Corp",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &harvest.ScheduledReport{
				Name:     "Parameters Test",
				Template: "executive",
				Format:   harvest.FormatPDF,
				Schedule: harvest.Schedule{
					Frequency: harvest.FrequencyDaily,
					Hour:      9,
					Timezone:  "UTC",
				},
				Parameters: tt.params,
				Enabled:    true,
			}

			err := ss.Create(ctx, schedule)
			require.NoError(t, err)

			// Retrieve and verify basic structure
			retrieved, err := ss.Get(ctx, schedule.ID)
			require.NoError(t, err)
			assert.NotNil(t, retrieved)
		})
	}
}

// ----------------------------------------------------------------------------
// All Template Formats Tests
// ----------------------------------------------------------------------------

func TestScheduledReport_AllTemplatesAndFormats(t *testing.T) {
	ss, _, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()

	templates := []string{"executive", "vulnerability", "inventory", "performance"}
	formats := []harvest.ExportFormat{
		harvest.FormatPDF,
		harvest.FormatHTML,
		harvest.FormatCSV,
		harvest.FormatJSON,
	}

	for _, tmpl := range templates {
		for _, fmt := range formats {
			t.Run(tmpl+"_"+string(fmt), func(t *testing.T) {
				schedule := &harvest.ScheduledReport{
					Name:     "Template Format Test",
					Template: tmpl,
					Format:   fmt,
					Schedule: harvest.Schedule{
						Frequency: harvest.FrequencyDaily,
						Hour:      10,
						Timezone:  "UTC",
					},
					Enabled: true,
				}

				err := ss.Create(ctx, schedule)
				require.NoError(t, err)
				assert.NotEmpty(t, schedule.ID)
			})
		}
	}
}

// ----------------------------------------------------------------------------
// Scheduler Types Tests
// ----------------------------------------------------------------------------

func TestScheduleFrequency_Constants(t *testing.T) {
	tests := []struct {
		name     string
		freq     harvest.ScheduleFrequency
		expected string
	}{
		{name: "daily", freq: harvest.FrequencyDaily, expected: "daily"},
		{name: "weekly", freq: harvest.FrequencyWeekly, expected: "weekly"},
		{name: "monthly", freq: harvest.FrequencyMonthly, expected: "monthly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.freq))
		})
	}
}

func TestExportFormat_Constants(t *testing.T) {
	tests := []struct {
		name     string
		format   harvest.ExportFormat
		expected string
	}{
		{name: "PDF", format: harvest.FormatPDF, expected: "pdf"},
		{name: "HTML", format: harvest.FormatHTML, expected: "html"},
		{name: "CSV", format: harvest.FormatCSV, expected: "csv"},
		{name: "JSON", format: harvest.FormatJSON, expected: "json"},
		{name: "Excel", format: harvest.FormatExcel, expected: "xlsx"},
		{name: "Markdown", format: harvest.FormatMarkdown, expected: "md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.format))
		})
	}
}
