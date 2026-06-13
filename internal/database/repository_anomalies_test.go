// SPDX-License-Identifier: BUSL-1.1

package database_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

func setupAnomalyDB(t *testing.T) (*database.DB, context.Context) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-anomalies-*.db")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, openErr := database.Open(tmpPath)
	if openErr != nil {
		t.Fatalf("open db: %v", openErr)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, context.Background()
}

func setupAnomalyTest(t *testing.T) (*database.AnomalyRepository, context.Context) {
	t.Helper()
	db, ctx := setupAnomalyDB(t)
	return db.Anomalies(), ctx
}

func sampleRecord(id string) anomaly.Record {
	t0 := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	// Derive the subject from the id's last "|"-segment so distinct fixtures get
	// distinct census keys (day_bucket, def_key, subject_kind, subject_id). A fixed
	// subject would collide unrelated records (e.g. "A|bssid|aa:bb" and
	// "B|bssid|cc:dd") on that key, making the daily census nondeterministic.
	subject := id
	if i := strings.LastIndex(id, "|"); i >= 0 {
		subject = id[i+1:]
	}
	return anomaly.Record{
		ID:     id,
		Source: anomaly.SourceWiFi,
		Anomaly: anomaly.Anomaly{
			DefKey:         "open-ssid",
			Category:       anomaly.CategorySecurity,
			Severity:       anomaly.SeverityWarning,
			Subject:        anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: subject},
			Title:          "Open network",
			Description:    "No encryption.",
			Recommendation: "Enable WPA2/WPA3.",
			Standards:      []string{"IEEE 802.11i-2004"},
			Evidence:       map[string]string{"rssi": "-42"},
			FirstSeen:      t0,
			LastSeen:       t0,
			Count:          1,
		},
	}
}

// TestAnomalyUpsertRoundTrip stores a record and reads it back through
// LoadActive, asserting the JSON columns and lifecycle survive the round trip.
func TestAnomalyUpsertRoundTrip(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAnomalyTest(t)

	rec := sampleRecord("open-ssid|bssid|aa:bb")
	if err := repo.Upsert(ctx, []anomaly.Record{rec}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	active, err := repo.LoadActive(ctx)
	if err != nil {
		t.Fatalf("LoadActive: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("LoadActive returned %d records, want 1", len(active))
	}
	got := active[0]
	if got.ID != rec.ID || got.Source != anomaly.SourceWiFi {
		t.Errorf("id/source = %q/%q, want %q/wifi", got.ID, got.Source, rec.ID)
	}
	if got.Anomaly.Severity != anomaly.SeverityWarning || got.Anomaly.Category != anomaly.CategorySecurity {
		t.Errorf("severity/category = %q/%q", got.Anomaly.Severity, got.Anomaly.Category)
	}
	if got.Anomaly.Subject != rec.Anomaly.Subject {
		t.Errorf("subject = %+v, want %+v", got.Anomaly.Subject, rec.Anomaly.Subject)
	}
	if got.Anomaly.Evidence["rssi"] != "-42" {
		t.Errorf("evidence = %+v, want rssi=-42", got.Anomaly.Evidence)
	}
	if len(got.Anomaly.Standards) != 1 || got.Anomaly.Standards[0] != "IEEE 802.11i-2004" {
		t.Errorf("standards = %+v", got.Anomaly.Standards)
	}
	if !got.Anomaly.FirstSeen.Equal(rec.Anomaly.FirstSeen) {
		t.Errorf("firstSeen = %v, want %v", got.Anomaly.FirstSeen, rec.Anomaly.FirstSeen)
	}
	if got.Resolved {
		t.Error("new record should not be resolved")
	}
}

// TestAnomalyUpsertIsIdempotentByID re-upserts the same id with a higher count
// and later lastSeen, asserting one row is updated (not duplicated) and
// first_seen is preserved.
func TestAnomalyUpsertIsIdempotentByID(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAnomalyTest(t)

	rec := sampleRecord("open-ssid|bssid|aa:bb")
	if err := repo.Upsert(ctx, []anomaly.Record{rec}); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	later := rec
	later.Anomaly.Count = 7
	later.Anomaly.LastSeen = rec.Anomaly.LastSeen.Add(time.Hour)
	later.Anomaly.Severity = anomaly.SeverityCritical
	if err := repo.Upsert(ctx, []anomaly.Record{later}); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	active, err := repo.LoadActive(ctx)
	if err != nil {
		t.Fatalf("LoadActive: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("got %d rows, want 1 (upsert must not duplicate)", len(active))
	}
	got := active[0]
	if got.Anomaly.Count != 7 || got.Anomaly.Severity != anomaly.SeverityCritical {
		t.Errorf("count/severity = %d/%q, want 7/critical", got.Anomaly.Count, got.Anomaly.Severity)
	}
	if !got.Anomaly.FirstSeen.Equal(rec.Anomaly.FirstSeen) {
		t.Errorf("firstSeen = %v, want %v (preserved on conflict)", got.Anomaly.FirstSeen, rec.Anomaly.FirstSeen)
	}
	if !got.Anomaly.LastSeen.Equal(later.Anomaly.LastSeen) {
		t.Errorf("lastSeen = %v, want %v (refreshed)", got.Anomaly.LastSeen, later.Anomaly.LastSeen)
	}
}

// TestAnomalyMarkResolved removes a record from the active set and a re-upsert
// revives it (resolution cleared on re-detection).
func TestAnomalyMarkResolved(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAnomalyTest(t)

	rec := sampleRecord("open-ssid|bssid|aa:bb")
	if err := repo.Upsert(ctx, []anomaly.Record{rec}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	resolvedAt := time.Date(2026, 6, 11, 13, 0, 0, 0, time.UTC)
	if err := repo.MarkResolved(ctx, []string{rec.ID}, resolvedAt); err != nil {
		t.Fatalf("MarkResolved: %v", err)
	}
	active, err := repo.LoadActive(ctx)
	if err != nil {
		t.Fatalf("LoadActive: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("resolved record still active: %d rows", len(active))
	}

	// Re-detection revives it.
	if upErr := repo.Upsert(ctx, []anomaly.Record{rec}); upErr != nil {
		t.Fatalf("revive Upsert: %v", upErr)
	}
	active, err = repo.LoadActive(ctx)
	if err != nil {
		t.Fatalf("LoadActive after revive: %v", err)
	}
	if len(active) != 1 || active[0].Resolved {
		t.Fatalf("re-detected record should be active again: %+v", active)
	}
}

// TestAnomalyEmptyBatchesAreNoops guards the len==0 fast paths.
func TestAnomalyEmptyBatchesAreNoops(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAnomalyTest(t)

	if err := repo.Upsert(ctx, nil); err != nil {
		t.Errorf("Upsert(nil): %v", err)
	}
	if err := repo.MarkResolved(ctx, nil, time.Now()); err != nil {
		t.Errorf("MarkResolved(nil): %v", err)
	}
	active, err := repo.LoadActive(ctx)
	if err != nil {
		t.Errorf("LoadActive: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("empty store returned %d records", len(active))
	}
}

// TestAnomalyDeleteResolvedOlderThan asserts the TTL purge removes only resolved
// instances past the cutoff, never active ones (ADR-0021 retention).
func TestAnomalyDeleteResolvedOlderThan(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAnomalyTest(t)

	for _, id := range []string{"old-resolved", "recent-resolved", "active"} {
		if err := repo.Upsert(ctx, []anomaly.Record{sampleRecord(id)}); err != nil {
			t.Fatalf("Upsert %q: %v", id, err)
		}
	}
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	// old-resolved cleared 100d ago; recent-resolved cleared 1d ago; active stays open.
	if err := repo.MarkResolved(ctx, []string{"old-resolved"}, now.AddDate(0, 0, -100)); err != nil {
		t.Fatalf("MarkResolved old: %v", err)
	}
	if err := repo.MarkResolved(ctx, []string{"recent-resolved"}, now.AddDate(0, 0, -1)); err != nil {
		t.Fatalf("MarkResolved recent: %v", err)
	}

	// Purge resolved older than 90d: only old-resolved qualifies.
	cutoff := now.AddDate(0, 0, -90)
	deleted, err := repo.DeleteResolvedOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteResolvedOlderThan: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1 (only old-resolved)", deleted)
	}

	// active is untouched and still loadable.
	active, err := repo.LoadActive(ctx)
	if err != nil {
		t.Fatalf("LoadActive: %v", err)
	}
	if len(active) != 1 || active[0].ID != "active" {
		t.Fatalf("active set = %+v, want only 'active'", active)
	}

	// recent-resolved survived the first purge; a purge at `now` removes it,
	// proving it was still present (not collaterally deleted).
	deleted2, err := repo.DeleteResolvedOlderThan(ctx, now)
	if err != nil {
		t.Fatalf("second DeleteResolvedOlderThan: %v", err)
	}
	if deleted2 != 1 {
		t.Errorf("second purge deleted = %d, want 1 (recent-resolved)", deleted2)
	}
}

// TestRunCleanupPurgesResolvedAnomalies proves the retention task is wired into
// RunCleanup: a resolved anomaly past the window is reported deleted while an
// active one survives.
func TestRunCleanupPurgesResolvedAnomalies(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()

	for _, id := range []string{"stale-resolved", "open"} {
		if err := repo.Upsert(ctx, []anomaly.Record{sampleRecord(id)}); err != nil {
			t.Fatalf("Upsert %q: %v", id, err)
		}
	}
	// Resolve one well beyond the 90d window (relative to now, since RunCleanup
	// computes its cutoff from time.Now).
	if err := repo.MarkResolved(ctx, []string{"stale-resolved"}, time.Now().UTC().AddDate(0, 0, -120)); err != nil {
		t.Fatalf("MarkResolved: %v", err)
	}

	result, err := db.RunCleanup(ctx, database.RetentionPolicy{AnomalyResolvedDays: 90})
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}
	if result.AnomaliesResolvedDeleted != 1 {
		t.Errorf("AnomaliesResolvedDeleted = %d, want 1", result.AnomaliesResolvedDeleted)
	}
	active, err := repo.LoadActive(ctx)
	if err != nil {
		t.Fatalf("LoadActive: %v", err)
	}
	if len(active) != 1 || active[0].ID != "open" {
		t.Errorf("active set = %+v, want only 'open'", active)
	}
}

// recordForSource builds a minimal record tagged with the given source/id so the
// source-scoped read can be exercised across producers.
func recordForSource(id string, source anomaly.Source) anomaly.Record {
	rec := sampleRecord(id)
	rec.Source = source
	return rec
}

// TestActiveBySourceFiltersAndExcludesResolved asserts the source-scoped read
// returns only the requested producer's unresolved instances.
func TestActiveBySourceFiltersAndExcludesResolved(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAnomalyTest(t)

	recs := []anomaly.Record{
		recordForSource("p1", anomaly.SourceProbe),
		recordForSource("p2", anomaly.SourceProbe),
		recordForSource("w1", anomaly.SourceWiFi),
	}
	if err := repo.Upsert(ctx, recs); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	probe, err := repo.ActiveBySource(ctx, anomaly.SourceProbe)
	if err != nil {
		t.Fatalf("ActiveBySource(probe): %v", err)
	}
	if len(probe) != 2 {
		t.Fatalf("want 2 probe anomalies, got %d (%+v)", len(probe), probe)
	}

	// Resolving one probe instance drops it from the active read; the other
	// source is untouched.
	at := time.Date(2026, 6, 11, 13, 0, 0, 0, time.UTC)
	if err = repo.MarkResolved(ctx, []string{"p1"}, at); err != nil {
		t.Fatalf("MarkResolved: %v", err)
	}
	probe, err = repo.ActiveBySource(ctx, anomaly.SourceProbe)
	if err != nil {
		t.Fatalf("ActiveBySource(probe) after resolve: %v", err)
	}
	if len(probe) != 1 {
		t.Fatalf("want 1 active probe anomaly after resolve, got %d", len(probe))
	}
	wifi, err := repo.ActiveBySource(ctx, anomaly.SourceWiFi)
	if err != nil {
		t.Fatalf("ActiveBySource(wifi): %v", err)
	}
	if len(wifi) != 1 || wifi[0].Subject != recs[2].Anomaly.Subject {
		t.Fatalf("want the single wifi anomaly intact, got %+v", wifi)
	}
}

// recordSpanning builds a record whose lifecycle covers [first, last] with a
// known cumulative count, for exercising the daily census intersection (ADR-0028).
func recordSpanning(id string, first, last time.Time, count int) anomaly.Record {
	rec := sampleRecord(id)
	rec.Anomaly.FirstSeen = first
	rec.Anomaly.LastSeen = last
	rec.Anomaly.Count = count
	return rec
}

func dayKey(t time.Time) string { return t.UTC().Format("2006-01-02") }

// TestCensusThroughCurrentDayActive censuses an empty table for today and asserts
// one row per active anomaly intersecting today, carrying the live row's facts.
func TestCensusThroughCurrentDayActive(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()
	now := time.Now().UTC()

	if err := repo.Upsert(ctx, []anomaly.Record{
		recordSpanning("open-ssid|bssid|aa:bb", now.AddDate(0, 0, -2), now, 3),
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	written, err := repo.CensusThrough(ctx, now)
	if err != nil {
		t.Fatalf("CensusThrough: %v", err)
	}
	if written != 1 {
		t.Fatalf("written = %d, want 1", written)
	}

	rows, err := db.ExportRollupDailyRows(ctx)
	if err != nil {
		t.Fatalf("ExportRollupDailyRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rollup rows = %d, want 1 (%+v)", len(rows), rows)
	}
	r := rows[0]
	if r.DayBucket != dayKey(now) {
		t.Errorf("day_bucket = %q, want %q", r.DayBucket, dayKey(now))
	}
	if r.CountCumul != 3 || r.MaxSeverity != "warning" || r.Source != "wifi" || r.IsResolved {
		t.Errorf("row = %+v, want count=3 sev=warning source=wifi active", r)
	}
}

// TestCensusThroughExcludesNonIntersectingDay proves the intersection predicate:
// an anomaly whose whole lifecycle predates today is not in today's census.
func TestCensusThroughExcludesNonIntersectingDay(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()
	now := time.Now().UTC()

	old := now.AddDate(0, 0, -5)
	if err := repo.Upsert(ctx, []anomaly.Record{recordSpanning("old|bssid|aa:bb", old, old, 1)}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := repo.MarkResolved(ctx, []string{"old|bssid|aa:bb"}, old); err != nil {
		t.Fatalf("MarkResolved: %v", err)
	}

	written, err := repo.CensusThrough(ctx, now)
	if err != nil {
		t.Fatalf("CensusThrough: %v", err)
	}
	if written != 0 {
		t.Fatalf("written = %d, want 0 (lifecycle predates today)", written)
	}
}

// TestCensusThroughCatchUp proves the rollup table's MAX(day_bucket) acts as the
// high-water mark: after a census at an earlier day, a later census fills every
// missed day through today (the downtime catch-up, ADR-0028 §3).
func TestCensusThroughCatchUp(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()
	now := time.Now().UTC()

	// An anomaly active across the whole window.
	if err := repo.Upsert(ctx, []anomaly.Record{
		recordSpanning("flap|bssid|aa:bb", now.AddDate(0, 0, -10), now, 9),
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Establish the high-water two days back.
	if _, err := repo.CensusThrough(ctx, now.AddDate(0, 0, -2)); err != nil {
		t.Fatalf("CensusThrough(now-2d): %v", err)
	}
	// Catch up to today: re-census now-2d plus now-1d and now.
	if _, err := repo.CensusThrough(ctx, now); err != nil {
		t.Fatalf("CensusThrough(now): %v", err)
	}

	rows, err := db.ExportRollupDailyRows(ctx)
	if err != nil {
		t.Fatalf("ExportRollupDailyRows: %v", err)
	}
	want := map[string]bool{
		dayKey(now.AddDate(0, 0, -2)): true,
		dayKey(now.AddDate(0, 0, -1)): true,
		dayKey(now):                   true,
	}
	if len(rows) != len(want) {
		t.Fatalf("rollup rows = %d, want %d (%+v)", len(rows), len(want), rows)
	}
	for _, r := range rows {
		if !want[r.DayBucket] {
			t.Errorf("unexpected day_bucket %q", r.DayBucket)
		}
	}
}

// TestCensusThroughIdempotentReflectsLatestCount re-censuses the same day after
// the live count advances and asserts the row is replaced, not duplicated.
func TestCensusThroughIdempotentReflectsLatestCount(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()
	now := time.Now().UTC()

	id := "open-ssid|bssid|aa:bb"
	if err := repo.Upsert(ctx, []anomaly.Record{recordSpanning(id, now.AddDate(0, 0, -1), now, 3)}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := repo.CensusThrough(ctx, now); err != nil {
		t.Fatalf("first CensusThrough: %v", err)
	}
	// The anomaly recurs; cumulative count advances.
	if err := repo.Upsert(ctx, []anomaly.Record{recordSpanning(id, now.AddDate(0, 0, -1), now, 7)}); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if _, err := repo.CensusThrough(ctx, now); err != nil {
		t.Fatalf("second CensusThrough: %v", err)
	}

	rows, err := db.ExportRollupDailyRows(ctx)
	if err != nil {
		t.Fatalf("ExportRollupDailyRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rollup rows = %d, want 1 (idempotent replace)", len(rows))
	}
	if rows[0].CountCumul != 7 {
		t.Errorf("count_cumulative = %d, want 7 (latest snapshot)", rows[0].CountCumul)
	}
}

// TestPurgeRollupsDailyOlderThan asserts the bucket-cutoff boundary: a same-day
// rollup survives a cutoff at today, and is removed once the cutoff passes it.
func TestPurgeRollupsDailyOlderThan(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()
	now := time.Now().UTC()

	if err := repo.Upsert(ctx, []anomaly.Record{recordSpanning("open-ssid|bssid|aa:bb", now, now, 1)}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := repo.CensusThrough(ctx, now); err != nil {
		t.Fatalf("CensusThrough: %v", err)
	}

	// Cutoff at today: today's bucket is not strictly older, so it survives.
	deleted, err := repo.PurgeRollupsDailyOlderThan(ctx, now)
	if err != nil {
		t.Fatalf("PurgeRollupsDailyOlderThan(now): %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0 (today's bucket survives)", deleted)
	}

	// Cutoff tomorrow: today's bucket is now older and is removed.
	deleted, err = repo.PurgeRollupsDailyOlderThan(ctx, now.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("PurgeRollupsDailyOlderThan(now+1d): %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
}

// TestRunCleanupCensusesResolvedRowBeforePurge is the ADR-0028 §3 callsite guard:
// RunCleanup must census the live anomaly table BEFORE the resolved-anomaly TTL
// purge, so a row resolved within the census window is preserved in the rollups
// even as its live row is deleted in the same pass. The CreateDailyRollup "0
// callers" gap must be impossible to reintroduce.
func TestRunCleanupCensusesResolvedRowBeforePurge(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()
	now := time.Now().UTC()

	// A: resolved two days ago (lifecycle ends now-2d). B: still active.
	resolvedDay := now.AddDate(0, 0, -2)
	if err := repo.Upsert(ctx, []anomaly.Record{
		recordSpanning("A|bssid|aa:bb", now.AddDate(0, 0, -5), resolvedDay, 4),
		recordForSource("B|bssid|cc:dd", anomaly.SourceWiFi),
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	// Keep B's lifecycle current so it intersects every catch-up day.
	bSpan := recordSpanning("B|bssid|cc:dd", now.AddDate(0, 0, -5), now, 2)
	if err := repo.Upsert(ctx, []anomaly.Record{bSpan}); err != nil {
		t.Fatalf("Upsert B span: %v", err)
	}
	if err := repo.MarkResolved(ctx, []string{"A|bssid|aa:bb"}, resolvedDay); err != nil {
		t.Fatalf("MarkResolved A: %v", err)
	}
	// Prior census established the high-water at now-3d, so the catch-up covers
	// A's resolution day (now-2d).
	if _, err := repo.CensusThrough(ctx, now.AddDate(0, 0, -3)); err != nil {
		t.Fatalf("seed high-water: %v", err)
	}

	result, err := db.RunCleanup(ctx, database.RetentionPolicy{
		AnomalyResolvedDays:    1,   // cutoff now-1d → A (resolved now-2d) is purged
		AnomalyRollupDailyDays: 730, // Pro: census + retain
	})
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}
	if result.AnomaliesResolvedDeleted != 1 {
		t.Errorf("AnomaliesResolvedDeleted = %d, want 1 (A purged)", result.AnomaliesResolvedDeleted)
	}
	if result.AnomalyRollupsCensused == 0 {
		t.Error("AnomalyRollupsCensused = 0, want > 0 (census ran inside RunCleanup)")
	}

	// A is gone from the live table...
	active, err := repo.LoadActive(ctx)
	if err != nil {
		t.Fatalf("LoadActive: %v", err)
	}
	if len(active) != 1 || active[0].ID != "B|bssid|cc:dd" {
		t.Fatalf("active = %+v, want only B", active)
	}

	// ...but its resolution-day census survived (captured before the purge).
	rows, err := db.ExportRollupDailyRows(ctx)
	if err != nil {
		t.Fatalf("ExportRollupDailyRows: %v", err)
	}
	var foundA bool
	for _, r := range rows {
		if r.DefKey == "open-ssid" && r.SubjectID == "aa:bb" && r.DayBucket == dayKey(resolvedDay) {
			foundA = true
			if !r.IsResolved {
				t.Errorf("A's census row is_resolved = false, want true")
			}
		}
	}
	if !foundA {
		t.Errorf("A's resolution-day census (%s) missing — census did not run before purge; rows=%+v",
			dayKey(resolvedDay), rows)
	}
}

// TestRunCleanupSkipsCensusOnFreeTier asserts no census is written when the
// DailyDays horizon is zero (Free/Starter).
func TestRunCleanupSkipsCensusOnFreeTier(t *testing.T) {
	t.Parallel()
	db, ctx := setupAnomalyDB(t)
	repo := db.Anomalies()
	now := time.Now().UTC()

	active := recordSpanning("open-ssid|bssid|aa:bb", now.AddDate(0, 0, -1), now, 3)
	if err := repo.Upsert(ctx, []anomaly.Record{active}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	result, err := db.RunCleanup(ctx, database.RetentionPolicy{AnomalyRollupDailyDays: 0})
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}
	if result.AnomalyRollupsCensused != 0 {
		t.Errorf("AnomalyRollupsCensused = %d, want 0 (Free tier)", result.AnomalyRollupsCensused)
	}
	rows, err := db.ExportRollupDailyRows(ctx)
	if err != nil {
		t.Fatalf("ExportRollupDailyRows: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rollup rows = %d, want 0 (no census on Free)", len(rows))
	}
}
