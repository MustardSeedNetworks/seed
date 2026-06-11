package anomaly_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
)

// fakeStore is an in-memory anomaly.Store recording the calls a Coordinator
// makes, so tests can assert the write cadence (write-through vs batched) without
// a database.
type fakeStore struct {
	rows      map[string]anomaly.Record
	upserts   int // number of Upsert CALLS (not rows) — distinguishes batching
	upsertRow int // number of records passed across all Upsert calls
	resolved  map[string]time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{rows: map[string]anomaly.Record{}, resolved: map[string]time.Time{}}
}

func (f *fakeStore) Upsert(_ context.Context, recs []anomaly.Record) error {
	f.upserts++
	for _, r := range recs {
		f.upsertRow++
		f.rows[r.ID] = r
		delete(f.resolved, r.ID) // re-detection revives
	}
	return nil
}

func (f *fakeStore) MarkResolved(_ context.Context, ids []string, at time.Time) error {
	for _, id := range ids {
		f.resolved[id] = at
		delete(f.rows, id)
	}
	return nil
}

func (f *fakeStore) LoadActive(_ context.Context) ([]anomaly.Record, error) {
	out := make([]anomaly.Record, 0, len(f.rows))
	for _, r := range f.rows {
		out = append(out, r)
	}
	return out, nil
}

func coordTestCatalog(t *testing.T) *anomaly.Catalog {
	t.Helper()
	c, err := anomaly.NewCatalog(
		anomaly.Def{
			ID: "open-ssid", Category: anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Title:           "Open network", Description: "No encryption.",
			Recommendation: "Enable WPA2/WPA3.",
		},
	)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return c
}

func newCoord(t *testing.T, store anomaly.Store, opts ...anomaly.Option) *anomaly.Coordinator {
	t.Helper()
	return anomaly.NewCoordinator(anomaly.NewEngine(coordTestCatalog(t), opts...), store, anomaly.SourceWiFi)
}

func openSSID(id string) anomaly.Detection {
	return anomaly.Detection{DefKey: "open-ssid", Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: id}}
}

// TestCoordinatorWritesThroughOnCreate asserts a brand-new anomaly is persisted
// immediately (write-through), tagged with the coordinator's source.
func TestCoordinatorWritesThroughOnCreate(t *testing.T) {
	store := newFakeStore()
	c := newCoord(t, store)
	at := time.Unix(1000, 0)

	if err := c.Observe(context.Background(), openSSID("aa"), at); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if store.upserts != 1 || store.upsertRow != 1 {
		t.Fatalf("create: upsert calls/rows = %d/%d, want 1/1 (write-through)", store.upserts, store.upsertRow)
	}
	got, ok := store.rows["open-ssid|bssid|aa"]
	if !ok {
		t.Fatalf("instance not persisted; rows=%v", store.rows)
	}
	if got.Source != anomaly.SourceWiFi {
		t.Errorf("source = %q, want wifi", got.Source)
	}
}

// TestCoordinatorBatchesRecurrence asserts pure recurrence does NOT write per
// observation — it accumulates and is persisted by a single Flush.
func TestCoordinatorBatchesRecurrence(t *testing.T) {
	store := newFakeStore()
	c := newCoord(t, store) // default escalateAfter=5, so recurrences below stay non-material
	ctx := context.Background()
	at := time.Unix(1000, 0)

	// First observation = create (write-through).
	if err := c.Observe(ctx, openSSID("aa"), at); err != nil {
		t.Fatalf("create Observe: %v", err)
	}
	// Three more recurrences (counts 2,3,4 — below the escalation threshold).
	for i := range 3 {
		if err := c.Observe(ctx, openSSID("aa"), at.Add(time.Duration(i+1)*time.Second)); err != nil {
			t.Fatalf("recurrence Observe: %v", err)
		}
	}
	if store.upserts != 1 {
		t.Fatalf("recurrence wrote through: upsert calls = %d, want 1 (only the create)", store.upserts)
	}

	if err := c.Flush(ctx); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if store.upserts != 2 {
		t.Fatalf("after Flush: upsert calls = %d, want 2 (create + one batched flush)", store.upserts)
	}
	if store.rows["open-ssid|bssid|aa"].Anomaly.Count != 4 {
		t.Errorf("flushed count = %d, want 4", store.rows["open-ssid|bssid|aa"].Anomaly.Count)
	}
}

// TestCoordinatorWritesThroughOnEscalation asserts crossing the escalation
// threshold is treated as a material change and written through, even though the
// underlying observation is a recurrence.
func TestCoordinatorWritesThroughOnEscalation(t *testing.T) {
	store := newFakeStore()
	c := newCoord(t, store, anomaly.WithEscalateAfter(3))
	ctx := context.Background()
	at := time.Unix(1000, 0)

	if err := c.Observe(ctx, openSSID("aa"), at); err != nil { // count 1, create
		t.Fatalf("Observe 1: %v", err)
	}
	if err := c.Observe(ctx, openSSID("aa"), at); err != nil { // count 2, recurrence (deferred)
		t.Fatalf("Observe 2: %v", err)
	}
	if store.upserts != 1 {
		t.Fatalf("before threshold: upsert calls = %d, want 1", store.upserts)
	}
	if err := c.Observe(ctx, openSSID("aa"), at); err != nil { // count 3 == threshold: escalation crossing
		t.Fatalf("Observe 3: %v", err)
	}
	if store.upserts != 2 {
		t.Fatalf("escalation crossing should write through: upsert calls = %d, want 2", store.upserts)
	}
	if store.rows["open-ssid|bssid|aa"].Anomaly.Severity != anomaly.SeverityError {
		t.Errorf("persisted severity = %q, want error (one bump up the ladder from warning)",
			store.rows["open-ssid|bssid|aa"].Anomaly.Severity)
	}
}

// TestCoordinatorResolvesOnPrune asserts pruning idle instances marks exactly
// those resolved in the store and clears pending dirty marks.
func TestCoordinatorResolvesOnPrune(t *testing.T) {
	store := newFakeStore()
	c := newCoord(t, store)
	ctx := context.Background()

	old := time.Unix(100, 0)
	fresh := time.Unix(10000, 0)
	if err := c.Observe(ctx, openSSID("stale"), old); err != nil {
		t.Fatalf("Observe stale: %v", err)
	}
	if err := c.Observe(ctx, openSSID("live"), fresh); err != nil {
		t.Fatalf("Observe live: %v", err)
	}

	n, err := c.Prune(ctx, time.Unix(5000, 0))
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if n != 1 {
		t.Fatalf("Prune cleared %d, want 1", n)
	}
	if _, resolved := store.resolved["open-ssid|bssid|stale"]; !resolved {
		t.Errorf("stale instance not marked resolved; resolved=%v", store.resolved)
	}
	if _, live := store.rows["open-ssid|bssid|live"]; !live {
		t.Errorf("live instance should remain; rows=%v", store.rows)
	}

	// A Flush after prune must not resurrect the resolved row.
	if flushErr := c.Flush(ctx); flushErr != nil {
		t.Fatalf("Flush: %v", flushErr)
	}
	if _, revived := store.rows["open-ssid|bssid|stale"]; revived {
		t.Error("Flush resurrected a pruned instance")
	}
}

// TestEngineRestoreSeedsActiveInstances asserts Restore repopulates the live set
// from records (load-on-start) and skips records whose def is uncatalogued.
func TestEngineRestoreSeedsActiveInstances(t *testing.T) {
	e := anomaly.NewEngine(coordTestCatalog(t))
	t0 := time.Unix(1000, 0)
	recs := []anomaly.Record{
		{
			ID: "open-ssid|bssid|aa", Source: anomaly.SourceWiFi,
			Anomaly: anomaly.Anomaly{
				DefKey: "open-ssid", Category: anomaly.CategorySecurity,
				Severity:  anomaly.SeverityWarning,
				Subject:   anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "aa"},
				FirstSeen: t0, LastSeen: t0.Add(time.Minute), Count: 4,
			},
		},
		{ // uncatalogued def — must be skipped, not loaded
			ID: "ghost|bssid|zz", Source: anomaly.SourceWiFi,
			Anomaly: anomaly.Anomaly{
				DefKey: "ghost", Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "zz"},
				Severity: anomaly.SeverityInfo, FirstSeen: t0, LastSeen: t0, Count: 1,
			},
		},
	}
	if n := e.Restore(recs); n != 1 {
		t.Fatalf("Restore loaded %d, want 1 (ghost skipped)", n)
	}
	snap := e.Snapshot()
	if len(snap) != 1 || snap[0].Subject.ID != "aa" {
		t.Fatalf("snapshot = %+v, want one restored 'aa'", snap)
	}
	if snap[0].Count != 4 || !snap[0].FirstSeen.Equal(t0) {
		t.Errorf("restored lifecycle: count=%d firstSeen=%v, want 4/%v", snap[0].Count, snap[0].FirstSeen, t0)
	}
}

// TestCoordinatorLoad asserts Load pulls active instances from the store into the
// engine.
func TestCoordinatorLoad(t *testing.T) {
	store := newFakeStore()
	store.rows["open-ssid|bssid|aa"] = anomaly.Record{
		ID: "open-ssid|bssid|aa", Source: anomaly.SourceWiFi,
		Anomaly: anomaly.Anomaly{
			DefKey: "open-ssid", Category: anomaly.CategorySecurity,
			Severity:  anomaly.SeverityWarning,
			Subject:   anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "aa"},
			FirstSeen: time.Unix(1, 0), LastSeen: time.Unix(2, 0), Count: 3,
		},
	}
	c := newCoord(t, store)
	n, err := c.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if n != 1 || c.Engine().Len() != 1 {
		t.Fatalf("Load restored n=%d engineLen=%d, want 1/1", n, c.Engine().Len())
	}
}
