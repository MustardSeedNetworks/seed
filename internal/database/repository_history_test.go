// SPDX-License-Identifier: BUSL-1.1

package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

func setupHistoryDB(t *testing.T) (*database.DB, context.Context) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-history-*.db")
	require.NoError(t, err)
	tmpPath := tmpFile.Name()
	require.NoError(t, tmpFile.Close())
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, openErr := database.Open(tmpPath)
	require.NoError(t, openErr)
	t.Cleanup(func() { _ = db.Close() })
	return db, context.Background()
}

// seedProbeWithResults creates a probe and appends one result per entry.
func seedProbeWithResults(
	ctx context.Context, t *testing.T, db *database.DB, probeID string,
	results []database.ProbeResult,
) {
	t.Helper()
	require.NoError(t, db.Probes().CreateProbe(ctx, &database.Probe{
		ID:       probeID,
		ClientID: database.DefaultClientID,
		Kind:     "ping",
		Target:   "10.0.0.1",
		Enabled:  true,
	}))
	for i := range results {
		results[i].ProbeID = probeID
		require.NoError(t, db.Probes().RecordResult(ctx, &results[i]))
	}
}

// The raw trend must include the bucket in progress — that is the whole reason
// a window the raw horizon covers is served from raw rather than from the
// hourly rollup, which is only written once its hour has closed.
func TestProbeTrendRawIncludesTheBucketInProgress(t *testing.T) {
	db, ctx := setupHistoryDB(t)

	now := time.Now().UTC()
	thisHour := now.Truncate(time.Hour)
	lastHour := thisHour.Add(-time.Hour)

	seedProbeWithResults(ctx, t, db, "probe-open-bucket", []database.ProbeResult{
		{Kind: "ping", Timestamp: lastHour.Add(5 * time.Minute), Success: true, LatencyMs: 10},
		{Kind: "ping", Timestamp: lastHour.Add(35 * time.Minute), Success: true, LatencyMs: 30},
		{Kind: "ping", Timestamp: thisHour.Add(time.Minute), Success: true, LatencyMs: 50},
	})

	points, err := db.History().ProbeTrendRaw(ctx,
		database.DefaultClientID, "probe-open-bucket",
		now.Add(-6*time.Hour), now, false)
	require.NoError(t, err)
	require.Len(t, points, 2, "the closed hour and the hour in progress")

	require.Equal(t, lastHour, points[0].Bucket)
	require.Equal(t, 2, points[0].SampleCount)
	require.InDelta(t, 20.0, points[0].AvgLatencyMs, 0.001)
	require.InDelta(t, 10.0, points[0].MinLatencyMs, 0.001)
	require.InDelta(t, 30.0, points[0].MaxLatencyMs, 0.001)

	require.Equal(t, thisHour, points[1].Bucket, "the open bucket is present")
	require.Equal(t, 1, points[1].SampleCount)
}

// A bucket in which every run failed has no latency to report. It must still
// appear, with its failure count, rather than being dropped from the series.
func TestProbeTrendRawKeepsFailedBuckets(t *testing.T) {
	db, ctx := setupHistoryDB(t)

	now := time.Now().UTC()
	hour := now.Truncate(time.Hour).Add(-2 * time.Hour)

	seedProbeWithResults(ctx, t, db, "probe-all-failed", []database.ProbeResult{
		{Kind: "ping", Timestamp: hour.Add(time.Minute), Success: false, Error: "timeout"},
		{Kind: "ping", Timestamp: hour.Add(2 * time.Minute), Success: false, Error: "timeout"},
	})

	points, err := db.History().ProbeTrendRaw(ctx,
		database.DefaultClientID, "probe-all-failed",
		now.Add(-6*time.Hour), now, false)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Equal(t, 2, points[0].SampleCount)
	require.Equal(t, 0, points[0].SuccessCount)
	require.Zero(t, points[0].AvgLatencyMs, "no successful run means no latency, not zero milliseconds")
}

// Another client's results must never appear in this client's trend.
func TestProbeTrendRawIsClientScoped(t *testing.T) {
	db, ctx := setupHistoryDB(t)

	now := time.Now().UTC()
	require.NoError(t, db.Clients().Create(ctx, &database.Client{ID: "other", Name: "Other site"}))
	seedProbeWithResults(ctx, t, db, "probe-scoped", []database.ProbeResult{
		{Kind: "ping", Timestamp: now.Add(-30 * time.Minute), Success: true, LatencyMs: 10},
		{
			Kind: "ping", ClientID: "other",
			Timestamp: now.Add(-20 * time.Minute), Success: true, LatencyMs: 99,
		},
	})

	points, err := db.History().ProbeTrendRaw(ctx,
		database.DefaultClientID, "probe-scoped", now.Add(-6*time.Hour), now, false)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Equal(t, 1, points[0].SampleCount, "the other client's run is excluded")
	require.InDelta(t, 10.0, points[0].AvgLatencyMs, 0.001)
}

// The live-table anomaly count answers the same question the daily census
// does, for the days the raw horizon still covers: how many distinct anomalies
// were open on each day, with no gap for the quiet days.
func TestAnomalyCountsByDayLive(t *testing.T) {
	db, ctx := setupHistoryDB(t)

	day := func(d int) time.Time {
		return time.Date(2026, time.September, d, 0, 0, 0, 0, time.UTC)
	}
	mk := func(defKey, subject, severity string, first, last time.Time, resolved *time.Time) anomaly.Record {
		rec := anomaly.Record{
			ID:     defKey + "|interface|" + subject,
			Source: anomaly.SourceWired,
			Anomaly: anomaly.Anomaly{
				DefKey:         defKey,
				Category:       anomaly.CategoryNetHealth,
				Severity:       anomaly.Severity(severity),
				Subject:        anomaly.SubjectRef{Kind: anomaly.SubjectInterface, ID: subject},
				Title:          "t",
				Description:    "d",
				Recommendation: "r",
				FirstSeen:      first,
				LastSeen:       last,
				Count:          1,
			},
		}
		if resolved != nil {
			rec.Resolved = true
			rec.ResolvedAt = *resolved
		}
		return rec
	}
	resolvedAt := day(12).Add(11 * time.Hour)
	require.NoError(t, db.Anomalies().Upsert(ctx, []anomaly.Record{
		mk("wired.fcs", "eth0", "warning", day(10).Add(9*time.Hour), day(12).Add(9*time.Hour), nil),
		mk("wired.duplex", "eth1", "critical", day(12).Add(9*time.Hour), day(12).Add(10*time.Hour), &resolvedAt),
	}))

	counts, err := db.History().AnomalyCountsByDayLive(ctx, day(9), day(13))
	require.NoError(t, err)
	require.Len(t, counts, 5, "every day in the range, quiet days included")

	byDay := map[string]database.AnomalyDayCount{}
	for _, c := range counts {
		byDay[c.Day] = c
	}
	require.Equal(t, 0, byDay["2026-09-09"].Count)
	require.Equal(t, 1, byDay["2026-09-10"].Count)
	require.Equal(t, 1, byDay["2026-09-11"].Count, "still open on the day between")
	require.Equal(t, 2, byDay["2026-09-12"].Count)
	require.Equal(t, 0, byDay["2026-09-13"].Count)
}
