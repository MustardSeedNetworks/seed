package reporting_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/reporting/store"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/forecast"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

const (
	trendDays = 28
	busyHour  = 14
	// trendTolerance is how far the acceptance lets the estimate fall from
	// the true crossing.
	trendTolerance = 24 * time.Hour
)

// trendPeak is the synthetic interface's busy-hour inbound utilization on day
// d: 1 point a day from 40, with a fixed zero-mean scatter of 1.5 points. The
// true line reaches 80 on day 40, at the busy hour.
func trendPeak(d int) float64 {
	return 40 + float64(d) + 1.5*[]float64{1, -1, 0.5, -0.5, 0}[d%5]
}

// seedTrend writes trendDays of rates through the real writer, ending
// yesterday, for three interfaces of core-sw1: ifIndex 1 trends up as
// trendPeak, ifIndex 2 holds at 30%, and ifIndex 3 was rated on three days
// only. Every hour gets four samples. Outbound runs at a quarter of inbound,
// and off the busy hour inbound runs at half its peak. It returns day 0 and
// the polling target's ID.
func seedTrend(t *testing.T, db *database.DB) (time.Time, string) {
	t.Helper()
	ctx := t.Context()
	target := &polling.Target{ClientID: "default", Name: "core-sw1", IPAddress: "10.0.0.1"}
	require.NoError(t, db.PollingTargets().Create(ctx, target))

	day0 := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -trendDays)
	rate := func(ifIndex uint32, at time.Time, in float64) ifrate.Rate {
		return ifrate.Rate{
			ClientID: "default", TargetID: target.ID, IfIndex: ifIndex, At: at,
			Octets: &ifrate.Octets{In: 1, Out: 1, Utilization: &ifrate.Utilization{In: in, Out: in / 4}},
		}
	}
	var rates []ifrate.Rate
	for d := range trendDays {
		for h := range 24 {
			for q := range 4 {
				at := day0.AddDate(0, 0, d).Add(time.Duration(h)*time.Hour + time.Duration(q)*15*time.Minute)
				peak := trendPeak(d)
				if h != busyHour {
					peak /= 2
				}
				rates = append(rates, rate(1, at, peak), rate(2, at, 30))
				if d >= trendDays-3 {
					rates = append(rates, rate(3, at, 90))
				}
			}
		}
	}
	require.NoError(t, db.Metrics().RecordInterfaceRates(ctx, rates))
	return day0, target.ID
}

func trueCrossing(day0 time.Time) time.Time {
	return day0.AddDate(0, 0, 40).Add(busyHour*time.Hour + 30*time.Minute)
}

func newCapacityGenerator(t *testing.T, db *database.DB) (*reporting.GeneratorService, *reporting.AggregatorService) {
	t.Helper()
	cfg := testConfig()
	templates := reporting.NewTemplateService(cfg)
	require.NoError(t, templates.Load())
	aggregator := reporting.NewAggregatorService(cfg, store.NewMetricsRepo(db))
	gen := reporting.NewGeneratorService(cfg, store.NewReportRepo(db), store.NewExportRepo(db), templates, aggregator)
	gen.SetReportsPath(t.TempDir())
	return gen, aggregator
}

func generateCapacity(t *testing.T, gen *reporting.GeneratorService, format reporting.ExportFormat) *reporting.Report {
	t.Helper()
	created, err := gen.GenerateFromTemplate(t.Context(), "capacity", format, nil)
	require.NoError(t, err)
	var report *reporting.Report
	eventually(t, func() bool {
		report, err = gen.GetReport(context.Background(), created.ID)
		return err == nil && report.Status != reporting.StatusPending && report.Status != reporting.StatusGenerating
	})
	require.Equal(t, reporting.StatusComplete, report.Status, report.Error)
	return report
}

// TestCapacityForecastOfATrendingInterface is P-B8's acceptance: an interface
// whose busy hour rises a point a day from 40% over four weeks gets an
// estimate within a day of the true 80% crossing, and the 95% range contains
// it. A flat interface gets none, and one rated on three days is counted as
// too short to forecast.
func TestCapacityForecastOfATrendingInterface(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	day0, _ := seedTrend(t, db)
	truth := trueCrossing(day0)
	gen, _ := newCapacityGenerator(t, db)

	report := generateCapacity(t, gen, reporting.FormatJSON)
	window := *report.Parameters.DateRange
	assert.Equal(t, 90*24*time.Hour, window.End.Sub(window.Start))
	assert.Equal(t, day0.AddDate(0, 0, trendDays), window.End, "the history ends at the start of today")

	raw, err := os.ReadFile(report.FilePath)
	require.NoError(t, err)
	var got struct {
		Forecast reporting.CapacityForecast `json:"forecast"`
	}
	require.NoError(t, json.Unmarshal(raw, &got))
	fc := got.Forecast
	assert.InDelta(t, 80.0, fc.Threshold, 0)
	assert.Equal(t, 1, fc.ShortHistory)
	require.Len(t, fc.Interfaces, 2)

	rising := fc.Interfaces[0]
	assert.Equal(t, "core-sw1", rising.Target)
	assert.Equal(t, uint32(1), rising.IfIndex)
	assert.Equal(t, trendDays, rising.Days)
	assert.InDelta(t, trendPeak(trendDays-1), rising.BusyHour, 1e-9, "the last day's busy hour, inbound")
	assert.InDelta(t, 1.0, rising.TrendPerDay, 0.1)
	require.Equal(t, forecast.Rising, rising.Outlook)
	require.NotNil(t, rising.Latest)
	assert.WithinDuration(t, truth, *rising.Estimate, trendTolerance)
	assert.False(t, rising.Earliest.After(truth), "earliest %v after %v", *rising.Earliest, truth)
	assert.False(t, rising.Latest.Before(truth), "latest %v before %v", *rising.Latest, truth)
	t.Logf("true crossing %v; estimate %v, range %v to %v",
		truth, *rising.Estimate, *rising.Earliest, *rising.Latest)

	flat := fc.Interfaces[1]
	assert.Equal(t, uint32(2), flat.IfIndex)
	assert.Equal(t, forecast.Steady, flat.Outlook)
	assert.Nil(t, flat.Estimate)
}

// TestCapacityPDFShowsTheEstimate: the PDF prints the same forecast as the
// JSON, the trending interface first with its estimated day.
func TestCapacityPDFShowsTheEstimate(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	seedTrend(t, db)
	gen, aggregator := newCapacityGenerator(t, db)

	report := generateCapacity(t, gen, reporting.FormatPDF)
	want, err := aggregator.Forecast(t.Context(), *report.Parameters.DateRange)
	require.NoError(t, err)
	rising := want.Interfaces[0]
	require.Equal(t, forecast.Rising, rising.Outlook)

	text := pdfText(t, report.FilePath)
	at := slices.Index(text, "core-sw1 ifIndex 1")
	require.GreaterOrEqual(t, at, 0, "no row for the trending interface in %q", text)
	require.Less(t, at+4, len(text))
	day := func(t time.Time) string { return t.UTC().Format("2006-01-02") }
	assert.Equal(t, []string{
		"core-sw1 ifIndex 1",
		fmt.Sprintf("%.1f%%", rising.BusyHour),
		fmt.Sprintf("%+.2f", rising.TrendPerDay),
		day(*rising.Estimate),
		day(*rising.Earliest) + " to " + day(*rising.Latest),
	}, text[at:at+5])
	assert.Contains(t, text, "not rising")
	assert.Contains(t, text, "Reaches 80.0%")
}

// TestCapacityHistoryOutlivesTheRawPurge: once the retention engine has
// rolled the hours up and purged raw rates older than a week, the forecast
// reads the same history from the hourly rollups.
func TestCapacityHistoryOutlivesTheRawPurge(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	day0, _ := seedTrend(t, db)
	_, aggregator := newCapacityGenerator(t, db)
	window := reporting.DateRange{Start: day0.AddDate(0, 0, -62), End: day0.AddDate(0, 0, trendDays)}

	before, err := aggregator.Forecast(t.Context(), window)
	require.NoError(t, err)

	rollup := database.NewMetricsRollupSource(db)
	for hour := day0; hour.Before(window.End); hour = hour.Add(time.Hour) {
		_, rollErr := rollup.RollupHour(t.Context(), hour)
		require.NoError(t, rollErr)
	}
	purged, err := rollup.PurgeRaw(t.Context(), window.End.AddDate(0, 0, -7))
	require.NoError(t, err)
	require.NotZero(t, purged)

	after, err := aggregator.Forecast(t.Context(), window)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

// TestCapacityPrefersTheRolledUpHour: a raw rate written into an hour after
// it was rolled up does not change that hour, since the raw purge may already
// have taken part of a rolled-up hour's samples.
func TestCapacityPrefersTheRolledUpHour(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	day0, targetID := seedTrend(t, db)
	hour := day0.Add(busyHour * time.Hour)
	_, err := database.NewMetricsRollupSource(db).RollupHour(t.Context(), hour)
	require.NoError(t, err)
	history, err := store.NewMetricsRepo(db).InterfaceUtilizationHistory(t.Context(),
		reporting.DateRange{Start: hour, End: hour.Add(2 * time.Hour)})
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Len(t, history[0].Hours, 2)
	assert.InDelta(t, trendPeak(0), history[0].Hours[0].Value, 1e-9, "the rolled-up busy hour")

	require.NoError(t, db.Metrics().RecordInterfaceRates(t.Context(), []ifrate.Rate{{
		ClientID: "default", TargetID: targetID, IfIndex: 1, At: hour.Add(time.Minute),
		Octets: &ifrate.Octets{In: 1, Out: 1, Utilization: &ifrate.Utilization{In: 100, Out: 100}},
	}}))

	again, err := store.NewMetricsRepo(db).InterfaceUtilizationHistory(t.Context(),
		reporting.DateRange{Start: hour, End: hour.Add(2 * time.Hour)})
	require.NoError(t, err)
	assert.Equal(t, history, again)
}

// TestForecastOrdersMostPressingFirst checks the shaping with no database:
// each day's busiest hour feeds the trend, and interfaces sort over the
// threshold first, then by soonest crossing, then rising past the horizon,
// then the rest.
func TestForecastOrdersMostPressingFirst(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)
	// hours gives 14 days of two hours each: a quiet 02:30 at 5% and a busy
	// hour that follows peak.
	hours := func(peak func(d int) float64) []forecast.Point {
		var points []forecast.Point
		for d := range 14 {
			at := start.AddDate(0, 0, d)
			points = append(points,
				forecast.Point{At: at.Add(2 * time.Hour), Value: 5},
				forecast.Point{At: at.Add(14 * time.Hour), Value: peak(d)})
		}
		return points
	}
	line := func(from, perDay float64) func(int) float64 {
		return func(d int) float64 { return from + perDay*float64(d) }
	}
	as := reporting.NewAggregatorService(testConfig(), &fakeMetricsRepo{history: []reporting.InterfaceUtilization{
		{Target: "steady", IfIndex: 1, Hours: hours(line(30, 0))},
		{Target: "slow", IfIndex: 1, Hours: hours(line(70, 0.5))},
		{Target: "beyond", IfIndex: 1, Hours: hours(line(10, 0.5))},
		{Target: "over", IfIndex: 1, Hours: hours(line(75, 1))},
		{Target: "fast", IfIndex: 1, Hours: hours(line(60, 1.2))},
		{Target: "short", IfIndex: 1, Hours: hours(line(60, 1))[:4]},
	}})
	fc, err := as.Forecast(t.Context(), reporting.DateRange{})
	require.NoError(t, err)

	var order []string
	for _, f := range fc.Interfaces {
		order = append(order, f.Target+":"+string(f.Outlook))
	}
	assert.Equal(t, []string{"over:above", "fast:rising", "slow:rising", "beyond:beyond", "steady:steady"}, order)
	assert.Equal(t, 1, fc.ShortHistory)
	assert.Equal(t, 14, fc.Interfaces[0].Days)
	assert.InDelta(t, 88.0, fc.Interfaces[0].BusyHour, 1e-9, "the last day's busy hour, not its quiet one")
}

// TestCapacityRejectsFormatsItDoesNotRender: like the summary, the forecast
// renders PDF and JSON only.
func TestCapacityRejectsFormatsItDoesNotRender(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	gen, _ := newCapacityGenerator(t, db)

	_, err := gen.GenerateFromTemplate(t.Context(), "capacity", reporting.FormatCSV, nil)
	require.Error(t, err)
}
