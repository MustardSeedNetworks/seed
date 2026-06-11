// SPDX-License-Identifier: BUSL-1.1

package database_test

import (
	"context"
	"os"
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
	return anomaly.Record{
		ID:     id,
		Source: anomaly.SourceWiFi,
		Anomaly: anomaly.Anomaly{
			DefKey:         "open-ssid",
			Category:       anomaly.CategorySecurity,
			Severity:       anomaly.SeverityWarning,
			Subject:        anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "aa:bb"},
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
