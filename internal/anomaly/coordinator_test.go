package anomaly_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
)

// syncStore is a thread-safe anomaly.Store for the concurrent-producers test
// (the bare fakeStore is only exercised single-threaded).
type syncStore struct {
	mu       sync.Mutex
	rows     map[string]anomaly.Record
	resolved map[string]time.Time
}

func newSyncStore() *syncStore {
	return &syncStore{rows: map[string]anomaly.Record{}, resolved: map[string]time.Time{}}
}

func (s *syncStore) Upsert(_ context.Context, recs []anomaly.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range recs {
		s.rows[r.ID] = r
		delete(s.resolved, r.ID)
	}
	return nil
}

func (s *syncStore) MarkResolved(_ context.Context, ids []string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		s.resolved[id] = at
		delete(s.rows, id)
	}
	return nil
}

func (s *syncStore) LoadActive(_ context.Context) ([]anomaly.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]anomaly.Record, 0, len(s.rows))
	for _, r := range s.rows {
		out = append(out, r)
	}
	return out, nil
}

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
	return anomaly.NewCoordinator(anomaly.NewEngine(coordTestCatalog(t), opts...), store)
}

func openSSID(id string) anomaly.Detection {
	return anomaly.Detection{
		DefKey:  "open-ssid",
		Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: id},
		Source:  anomaly.SourceWiFi,
	}
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

	n, err := c.Prune(ctx, anomaly.SourceWiFi, time.Unix(5000, 0))
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

// TestCoordinatorPruneIsSourceScoped asserts that pruning one source leaves
// another source's idle instances untouched (ADR-0029 §3): a single shared engine
// must honor per-source resolution windows. Both instances are equally idle, but
// only the pruned source's instance resolves.
func TestCoordinatorPruneIsSourceScoped(t *testing.T) {
	store := newFakeStore()
	c := newCoord(t, store)
	ctx := context.Background()
	old := time.Unix(100, 0)

	wifi := anomaly.Detection{
		DefKey:  "open-ssid",
		Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "wifi-subj"},
		Source:  anomaly.SourceWiFi,
	}
	probe := anomaly.Detection{
		DefKey:  "open-ssid",
		Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "probe-subj"},
		Source:  anomaly.SourceProbe,
	}
	if err := c.Observe(ctx, wifi, old); err != nil {
		t.Fatalf("Observe wifi: %v", err)
	}
	if err := c.Observe(ctx, probe, old); err != nil {
		t.Fatalf("Observe probe: %v", err)
	}

	// Prune the probe source past both instances' idle time.
	n, err := c.Prune(ctx, anomaly.SourceProbe, time.Unix(5000, 0))
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if n != 1 {
		t.Fatalf("Prune cleared %d, want 1 (only the probe instance)", n)
	}
	if _, resolved := store.resolved["open-ssid|bssid|probe-subj"]; !resolved {
		t.Errorf("probe instance not resolved; resolved=%v", store.resolved)
	}
	if _, stillResolved := store.resolved["open-ssid|bssid|wifi-subj"]; stillResolved {
		t.Errorf("wifi instance was wrongly resolved by a probe-scoped prune")
	}
	if _, live := store.rows["open-ssid|bssid|wifi-subj"]; !live {
		t.Errorf("wifi instance should remain live; rows=%v", store.rows)
	}
}

// TestCoordinatorResolveSubject asserts the explicit recovery fast-path clears
// every live instance for a subject (across all defs) and marks exactly those
// resolved as of the supplied time, leaving other subjects untouched.
func TestCoordinatorResolveSubject(t *testing.T) {
	cat, err := anomaly.NewCatalog(
		anomaly.Def{
			ID: "open-ssid", Category: anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning, Title: "Open network",
			Description: "No encryption.", Recommendation: "Enable WPA2/WPA3.",
		},
		anomaly.Def{
			ID: "weak-cipher", Category: anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning, Title: "Weak cipher",
			Description: "Deprecated cipher suite.", Recommendation: "Use AES-CCMP.",
		},
	)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	store := newFakeStore()
	c := anomaly.NewCoordinator(anomaly.NewEngine(cat), store)
	ctx := context.Background()
	at := time.Unix(1000, 0)

	det := func(def, id string) anomaly.Detection {
		return anomaly.Detection{
			DefKey:  def,
			Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: id},
			Source:  anomaly.SourceWiFi,
		}
	}
	// Two defs on subject A, one on subject B.
	for _, d := range []anomaly.Detection{det("open-ssid", "A"), det("weak-cipher", "A"), det("open-ssid", "B")} {
		if obsErr := c.Observe(ctx, d, at); obsErr != nil {
			t.Fatalf("Observe %s/%s: %v", d.DefKey, d.Subject.ID, obsErr)
		}
	}

	resolveAt := time.Unix(2000, 0)
	n, err := c.ResolveSubject(ctx, anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "A"}, resolveAt)
	if err != nil {
		t.Fatalf("ResolveSubject: %v", err)
	}
	if n != 2 {
		t.Fatalf("ResolveSubject cleared %d, want 2 (both defs on A)", n)
	}
	for _, id := range []string{"open-ssid|bssid|A", "weak-cipher|bssid|A"} {
		if got, ok := store.resolved[id]; !ok {
			t.Errorf("%s not marked resolved; resolved=%v", id, store.resolved)
		} else if !got.Equal(resolveAt) {
			t.Errorf("%s resolved at %v, want %v", id, got, resolveAt)
		}
	}
	if _, live := store.rows["open-ssid|bssid|B"]; !live {
		t.Errorf("subject B should remain live; rows=%v", store.rows)
	}

	// A Flush must not resurrect a resolved row.
	if flushErr := c.Flush(ctx); flushErr != nil {
		t.Fatalf("Flush: %v", flushErr)
	}
	if _, revived := store.rows["open-ssid|bssid|A"]; revived {
		t.Error("Flush resurrected a resolved instance")
	}
}

// TestCoordinatorResolveSubjectUnknownIsNoop asserts resolving a subject with no
// active instances neither errors nor touches the store.
func TestCoordinatorResolveSubjectUnknownIsNoop(t *testing.T) {
	store := newFakeStore()
	c := newCoord(t, store)

	n, err := c.ResolveSubject(
		context.Background(),
		anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "ghost"},
		time.Unix(1, 0),
	)
	if err != nil {
		t.Fatalf("ResolveSubject: %v", err)
	}
	if n != 0 || len(store.resolved) != 0 {
		t.Fatalf("unknown subject: cleared=%d resolved=%v, want 0 / empty", n, store.resolved)
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

// TestCoordinatorConcurrentProducers asserts the shared Coordinator (ADR-0029)
// is safe under two producers observing and pruning it concurrently, and that
// source-scoped prune only resolves the named source's instances — the property
// that lets Wi-Fi (5 m) and probe (15 m) share one engine without one source
// resolving the other's still-live instances.
func TestCoordinatorConcurrentProducers(t *testing.T) {
	c := newCoord(t, newSyncStore())
	ctx := context.Background()
	base := time.Unix(10000, 0).UTC()

	const n = 200
	var wg sync.WaitGroup
	wg.Add(2)
	observe := func(src anomaly.Source, prefix string) {
		defer wg.Done()
		for i := range n {
			at := base.Add(time.Duration(i) * time.Millisecond)
			id := prefix + string(rune('a'+i%26))
			_ = c.Observe(ctx, anomaly.Detection{
				DefKey:  "open-ssid",
				Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: id},
				Source:  src,
			}, at)
			if i%10 == 0 {
				_ = c.Flush(ctx)
			}
		}
	}
	go observe(anomaly.SourceWiFi, "w-")
	go observe(anomaly.SourceProbe, "p-")
	wg.Wait()

	// Both sources coalesced onto 26 distinct subjects each (no lost updates, no
	// races under -race).
	if got := c.Engine().LenBySource(anomaly.SourceWiFi); got != 26 {
		t.Errorf("wifi instances = %d, want 26", got)
	}
	if got := c.Engine().LenBySource(anomaly.SourceProbe); got != 26 {
		t.Errorf("probe instances = %d, want 26", got)
	}

	// Source-scoped prune past every observation resolves only Wi-Fi; probe stays.
	cutoff := base.Add(time.Hour)
	cleared, err := c.Prune(ctx, anomaly.SourceWiFi, cutoff)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if cleared != 26 {
		t.Errorf("pruned wifi = %d, want 26", cleared)
	}
	if got := c.Engine().LenBySource(anomaly.SourceProbe); got != 26 {
		t.Errorf("probe instances after wifi prune = %d, want 26 (untouched)", got)
	}
}
