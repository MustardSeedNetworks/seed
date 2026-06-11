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

func setupAnomalyTest(t *testing.T) (*database.AnomalyRepository, context.Context) {
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
	return db.Anomalies(), context.Background()
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
