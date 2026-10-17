package probe_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

// errProbeNotFound is the fake's stand-in for a missing-probe lookup. The
// engine wraps whatever the storage port returns with %w, so the concrete
// sentinel value is not asserted — only that an error propagates.
var errProbeNotFound = errors.New("probe not found")

// fakeStorage implements the probe engine's probeStorage port for tests,
// speaking the probe package's own domain types (the DB-row translation now
// lives in the internal/app adapter).
type fakeStorage struct {
	mu           sync.Mutex
	probes       map[string]probe.Probe
	results      []probe.Result
	getErr       error
	recordErr    error
	listFilter   string // captures the kind filter passed to ListProbes
	clientFilter string
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{probes: make(map[string]probe.Probe)}
}

func (f *fakeStorage) GetProbe(_ context.Context, id string) (probe.Probe, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return probe.Probe{}, f.getErr
	}
	p, ok := f.probes[id]
	if !ok {
		return probe.Probe{}, errProbeNotFound
	}
	return p, nil
}

func (f *fakeStorage) ListProbes(_ context.Context, clientID, kind string) ([]probe.Probe, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clientFilter = clientID
	f.listFilter = kind
	out := make([]probe.Probe, 0, len(f.probes))
	for _, p := range f.probes {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeStorage) RecordResult(_ context.Context, r probe.Result) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.recordErr != nil {
		return f.recordErr
	}
	f.results = append(f.results, r)
	return nil
}

func (f *fakeStorage) addProbe(p probe.Probe) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.probes[p.ID] = p
}

func (f *fakeStorage) recordedResults() []probe.Result {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]probe.Result, len(f.results))
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

func (f *fakeScheduler) Start(_ context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = true
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
	storage.addProbe(probe.Probe{ID: "p-1", Kind: "dns", IntervalSeconds: 60, Enabled: true})
	storage.addProbe(probe.Probe{ID: "p-2", Kind: "tls", IntervalSeconds: 300, Enabled: false})
	storage.addProbe(probe.Probe{ID: "p-3", Kind: "dns", IntervalSeconds: 30, Enabled: true})
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
	storage.addProbe(probe.Probe{ID: "p-1", Kind: "dns", IntervalSeconds: 60, Enabled: true})
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
	storage.addProbe(probe.Probe{ID: "p-1", Kind: "dns", IntervalSeconds: 60, Enabled: true})
	storage.addProbe(probe.Probe{ID: "p-2", Kind: "tls", IntervalSeconds: 300, Enabled: true})
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
	storage.addProbe(probe.Probe{
		ID:          "p-1",
		ClientID:    "default",
		Kind:        "dns",
		DisplayName: "google.com",
		Target:      "google.com",
		Params:      json.RawMessage(`{"record_type":"A"}`),
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
	storage.addProbe(probe.Probe{ID: "p-1", Kind: "dns", Enabled: true})
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
	storage.addProbe(probe.Probe{
		ID:          "p-1",
		ClientID:    "tenant-x",
		Kind:        "dns",
		DisplayName: "internal",
		Target:      "internal.example.com",
		Params:      json.RawMessage(`{"server":"10.0.0.5"}`),
		Warning:     json.RawMessage(`{"latency_ms":50}`),
		Enabled:     true,
	})
	sched := newFakeScheduler()

	// Checker echoes the probe back as metadata so we can verify the
	// Probe definition's fields reach the Checker's input unchanged.
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

// A scheduled run that Stop cancels was abandoned and says nothing about the
// target: it records no result and is not a failed job. A deadline, or any
// other error, is still the run's outcome (seed#3586).
func TestProbeJob_Run_CancelledRunIsNotAFailure(t *testing.T) {
	t.Parallel()
	expired := func() (context.Context, context.CancelFunc) {
		return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	}
	cancelled := func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, cancel
	}
	live := func() (context.Context, context.CancelFunc) {
		return context.WithCancel(context.Background())
	}
	cases := []struct {
		name        string
		ctx         func() (context.Context, context.CancelFunc)
		recordErr   error
		wantErr     bool
		wantResults int
	}{
		{name: "cancelled", ctx: cancelled},
		{name: "deadline", ctx: expired, recordErr: context.DeadlineExceeded, wantErr: true},
		{name: "storage error", ctx: live, recordErr: errors.New("disk full"), wantErr: true},
		{name: "success", ctx: live, wantResults: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			storage := newFakeStorage()
			storage.addProbe(probe.Probe{ID: "p-1", Kind: "dns", IntervalSeconds: 60, Enabled: true})
			storage.recordErr = tc.recordErr
			sched := newFakeScheduler()
			var logs bytes.Buffer
			e := probe.NewEngine(slog.New(slog.NewTextHandler(&logs, nil))).WithStorage(storage, sched)
			e.RegisterChecker(&fakeChecker{kind: "dns", result: probe.Result{Success: true}})
			if err := e.Start(context.Background()); err != nil {
				t.Fatalf("Start: %v", err)
			}
			job, ok := sched.registered["probe:p-1"]
			if !ok {
				t.Fatal("probe:p-1 was not registered")
			}

			ctx, cancel := tc.ctx()
			defer cancel()
			err := job.Run(ctx)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("Run error = %v, wantErr %v", err, tc.wantErr)
			}
			if got := len(storage.recordedResults()); got != tc.wantResults {
				t.Errorf("recorded %d results, want %d", got, tc.wantResults)
			}
			if logged := strings.Contains(logs.String(), "probe dispatch failed"); logged != tc.wantErr {
				t.Errorf("logged dispatch failure = %v, want %v:\n%s", logged, tc.wantErr, logs.String())
			}
		})
	}
}
