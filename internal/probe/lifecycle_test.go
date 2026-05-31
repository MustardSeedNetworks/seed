package probe_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/probe"
	"github.com/krisarmstrong/seed/internal/scheduler"
)

// fakeStorage implements probe.storage's behavior for tests.
type fakeStorage struct {
	mu           sync.Mutex
	probes       map[string]*database.Probe
	results      []*database.ProbeResult
	getErr       error
	recordErr    error
	listFilter   string // captures the kind filter passed to ListProbes
	clientFilter string
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{probes: make(map[string]*database.Probe)}
}

func (f *fakeStorage) GetProbe(_ context.Context, id string) (*database.Probe, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	p, ok := f.probes[id]
	if !ok {
		return nil, database.ErrProbeNotFound
	}
	return p, nil
}

func (f *fakeStorage) ListProbes(_ context.Context, clientID, kind string) ([]*database.Probe, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clientFilter = clientID
	f.listFilter = kind
	out := make([]*database.Probe, 0, len(f.probes))
	for _, p := range f.probes {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeStorage) RecordResult(_ context.Context, pr *database.ProbeResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.recordErr != nil {
		return f.recordErr
	}
	f.results = append(f.results, pr)
	return nil
}

func (f *fakeStorage) addProbe(p *database.Probe) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.probes[p.ID] = p
}

func (f *fakeStorage) recordedResults() []*database.ProbeResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*database.ProbeResult, len(f.results))
	copy(out, f.results)
	return out
}

// fakeScheduler implements probe.probeScheduler for tests. Tracks
// registered jobs without actually ticking.
type fakeScheduler struct {
	mu         sync.Mutex
	registered map[string]scheduler.Job
	started    bool
	stopped    bool
}

func newFakeScheduler() *fakeScheduler {
	return &fakeScheduler{registered: make(map[string]scheduler.Job)}
}

func (f *fakeScheduler) Register(j scheduler.Job) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registered[j.ID()] = j
}

func (f *fakeScheduler) Unregister(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, present := f.registered[id]
	delete(f.registered, id)
	return present
}

func (f *fakeScheduler) Start(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = true
	return nil
}

func (f *fakeScheduler) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = true
}

func (f *fakeScheduler) jobCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.registered)
}

func TestEngine_WithStorage_ReturnsReceiver(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	storage := newFakeStorage()
	sched := newFakeScheduler()
	got := e.WithStorage(storage, sched)
	if got != e {
		t.Error("WithStorage should return the receiver for chaining")
	}
}

func TestEngine_Start_WithoutStorage_ReturnsError(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	err := e.Start(context.Background())
	if !errors.Is(err, probe.ErrStorageNotConfigured) {
		t.Errorf("Start without storage = %v, want ErrStorageNotConfigured", err)
	}
}

func TestEngine_RunNow_WithoutStorage_ReturnsError(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	_, err := e.RunNow(context.Background(), "p-1")
	if !errors.Is(err, probe.ErrStorageNotConfigured) {
		t.Errorf("RunNow without storage = %v, want ErrStorageNotConfigured", err)
	}
}

func TestEngine_Stop_IdempotentAndNoStorage(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	// Stop before Start should be a no-op (no error).
	if err := e.Stop(context.Background()); err != nil {
		t.Errorf("Stop before Start = %v, want nil", err)
	}
}

func TestEngine_Start_LoadsEnabledProbes_SkipsDisabled(t *testing.T) {
	t.Parallel()
	storage := newFakeStorage()
	storage.addProbe(&database.Probe{ID: "p-1", Kind: "dns", IntervalSeconds: 60, Enabled: true})
	storage.addProbe(&database.Probe{ID: "p-2", Kind: "tls", IntervalSeconds: 300, Enabled: false})
	storage.addProbe(&database.Probe{ID: "p-3", Kind: "dns", IntervalSeconds: 30, Enabled: true})
	sched := newFakeScheduler()
	e := probe.NewEngine(silentLogger()).WithStorage(storage, sched)

	if err := e.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if sched.jobCount() != 2 {
		t.Errorf("scheduler.jobCount = %d, want 2 (enabled probes only)", sched.jobCount())
	}
	if !sched.started {
		t.Error("scheduler.Start was not called")
	}
}

func TestEngine_Start_Idempotent(t *testing.T) {
	t.Parallel()
	storage := newFakeStorage()
	storage.addProbe(&database.Probe{ID: "p-1", Kind: "dns", IntervalSeconds: 60, Enabled: true})
	sched := newFakeScheduler()
	e := probe.NewEngine(silentLogger()).WithStorage(storage, sched)

	if err := e.Start(context.Background()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if err := e.Start(context.Background()); err != nil {
		t.Errorf("second Start = %v, want nil (idempotent)", err)
	}
	if sched.jobCount() != 1 {
		t.Errorf("scheduler.jobCount after 2 Starts = %d, want 1", sched.jobCount())
	}
}

func TestEngine_Stop_UnregistersJobs(t *testing.T) {
	t.Parallel()
	storage := newFakeStorage()
	storage.addProbe(&database.Probe{ID: "p-1", Kind: "dns", IntervalSeconds: 60, Enabled: true})
	storage.addProbe(&database.Probe{ID: "p-2", Kind: "tls", IntervalSeconds: 300, Enabled: true})
	sched := newFakeScheduler()
	e := probe.NewEngine(silentLogger()).WithStorage(storage, sched)

	if err := e.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := e.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if sched.jobCount() != 0 {
		t.Errorf("scheduler.jobCount after Stop = %d, want 0", sched.jobCount())
	}
	if !sched.stopped {
		t.Error("scheduler.Stop was not called")
	}
}

func TestEngine_RunNow_DispatchesAndPersists(t *testing.T) {
	t.Parallel()
	storage := newFakeStorage()
	storage.addProbe(&database.Probe{
		ID:          "p-1",
		ClientID:    "default",
		Kind:        "dns",
		DisplayName: "google.com",
		Target:      "google.com",
		ParamsJSON:  `{"record_type":"A"}`,
		Enabled:     true,
	})
	sched := newFakeScheduler()

	e := probe.NewEngine(silentLogger()).WithStorage(storage, sched)
	e.RegisterChecker(&fakeChecker{
		kind: "dns",
		result: probe.Result{
			Success:   true,
			LatencyMs: 15,
		},
	})

	r, err := e.RunNow(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	if !r.Success {
		t.Error("Result.Success = false, want true")
	}

	results := storage.recordedResults()
	if len(results) != 1 {
		t.Fatalf("recorded results = %d, want 1", len(results))
	}
	if results[0].ProbeID != "p-1" {
		t.Errorf("recorded ProbeID = %q, want p-1", results[0].ProbeID)
	}
	if results[0].ClientID != "default" {
		t.Errorf("recorded ClientID = %q, want default", results[0].ClientID)
	}
	if results[0].LatencyMs != 15 {
		t.Errorf("recorded LatencyMs = %v, want 15", results[0].LatencyMs)
	}
}

func TestEngine_RunNow_PropagatesProbeNotFound(t *testing.T) {
	t.Parallel()
	storage := newFakeStorage()
	sched := newFakeScheduler()
	e := probe.NewEngine(silentLogger()).WithStorage(storage, sched)

	_, err := e.RunNow(context.Background(), "missing")
	if err == nil {
		t.Error("RunNow on missing probe should error")
	}
}

func TestEngine_RunNow_PropagatesPersistError(t *testing.T) {
	t.Parallel()
	storage := newFakeStorage()
	storage.addProbe(&database.Probe{ID: "p-1", Kind: "dns", Enabled: true})
	storage.recordErr = errors.New("disk full")
	sched := newFakeScheduler()
	e := probe.NewEngine(silentLogger()).WithStorage(storage, sched)
	e.RegisterChecker(&fakeChecker{kind: "dns", result: probe.Result{Success: true}})

	_, err := e.RunNow(context.Background(), "p-1")
	if err == nil {
		t.Error("RunNow with persist error should fail")
	}
}

func TestEngine_RunNow_ProbeFieldsRoundTripToResult(t *testing.T) {
	t.Parallel()
	storage := newFakeStorage()
	storage.addProbe(&database.Probe{
		ID:          "p-1",
		ClientID:    "tenant-x",
		Kind:        "dns",
		DisplayName: "internal",
		Target:      "internal.example.com",
		ParamsJSON:  `{"server":"10.0.0.5"}`,
		WarningJSON: `{"latency_ms":50}`,
		Enabled:     true,
	})
	sched := newFakeScheduler()

	// Checker echoes the probe back as metadata so we can verify
	// the round-trip from DB row through Probe model into the
	// Checker's input.
	echoer := &echoChecker{}
	e := probe.NewEngine(silentLogger()).WithStorage(storage, sched)
	e.RegisterChecker(echoer)

	_, err := e.RunNow(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("RunNow: %v", err)
	}

	if echoer.got.ClientID != "tenant-x" {
		t.Errorf("Checker saw ClientID=%q, want tenant-x", echoer.got.ClientID)
	}
	if string(echoer.got.Params) != `{"server":"10.0.0.5"}` {
		t.Errorf("Checker saw Params=%q, want round-trip", string(echoer.got.Params))
	}
	if string(echoer.got.Warning) != `{"latency_ms":50}` {
		t.Errorf("Checker saw Warning=%q, want round-trip", string(echoer.got.Warning))
	}
}

// echoChecker captures the Probe it receives for round-trip
// assertions.
type echoChecker struct {
	got probe.Probe
}

func (c *echoChecker) Kind() string                   { return "dns" }
func (c *echoChecker) RequiredCapabilities() []string { return nil }
func (c *echoChecker) Run(_ context.Context, p probe.Probe) probe.Result {
	c.got = p
	return probe.Result{Success: true, Metadata: json.RawMessage(`{"echoed":true}`)}
}
