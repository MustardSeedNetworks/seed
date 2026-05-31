package snmp_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/polling/snmp"
	"github.com/krisarmstrong/seed/internal/scheduler"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// fakeStorage mirrors a tiny subset of database.PollingTargetRepository.
type fakeStorage struct {
	mu      sync.Mutex
	targets []*database.PollingTarget
	listErr error

	updates []updateRecord
	updErr  error
}

type updateRecord struct {
	id     string
	status string
	errMsg string
}

func (f *fakeStorage) List(_ context.Context, _ string) ([]*database.PollingTarget, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.targets, nil
}

func (f *fakeStorage) UpdateLastPoll(_ context.Context, id, status, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, updateRecord{id: id, status: status, errMsg: errMsg})
	return f.updErr
}

// fakeScheduler captures registered jobs without ticking.
type fakeScheduler struct {
	mu      sync.Mutex
	jobs    map[string]scheduler.Job
	started bool
	stopped bool
}

func newFakeScheduler() *fakeScheduler {
	return &fakeScheduler{jobs: make(map[string]scheduler.Job)}
}

func (f *fakeScheduler) Register(j scheduler.Job) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[j.ID()] = j
}

func (f *fakeScheduler) Unregister(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.jobs[id]
	delete(f.jobs, id)
	return ok
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
	return len(f.jobs)
}

// firstJob returns any one job in the scheduler — useful when the
// test created a single target and just wants to drive its Run().
func (f *fakeScheduler) firstJob() scheduler.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, j := range f.jobs {
		return j
	}
	return nil
}

// stubCollector records every Collect invocation; returns the
// configured err on call.
type stubCollector struct {
	mu    sync.Mutex
	name  string
	err   error
	calls []snmp.Target
}

func (s *stubCollector) Name() string { return s.name }

func (s *stubCollector) Collect(
	_ context.Context,
	t snmp.Target,
	_ snmp.ResolvedCredentials,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, t)
	return s.err
}

func (s *stubCollector) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func TestPoller_Start_RegistersJobsForEachEnabledTarget(t *testing.T) {
	t.Parallel()
	storage := &fakeStorage{
		targets: []*database.PollingTarget{
			{
				ID:              "t-1",
				Name:            "router-1",
				IPAddress:       "10.0.0.1",
				Enabled:         true,
				PollIntervalSec: 60,
				CollectorChain:  []string{"sys_info"},
			},
			{
				ID:              "t-2",
				Name:            "router-2",
				IPAddress:       "10.0.0.2",
				Enabled:         true,
				PollIntervalSec: 300,
				CollectorChain:  []string{"sys_info"},
			},
		},
	}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if sched.jobCount() != 2 {
		t.Errorf("scheduler.jobCount = %d, want 2", sched.jobCount())
	}
	if !sched.started {
		t.Error("scheduler.Start was not called")
	}
}

func TestPoller_Start_Idempotent(t *testing.T) {
	t.Parallel()
	storage := &fakeStorage{
		targets: []*database.PollingTarget{
			{ID: "t-1", Enabled: true, PollIntervalSec: 60},
		},
	}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Errorf("second Start = %v, want nil", err)
	}
	if sched.jobCount() != 1 {
		t.Errorf("scheduler.jobCount = %d, want 1 after 2 Starts", sched.jobCount())
	}
}

func TestPoller_Start_PropagatesListError(t *testing.T) {
	t.Parallel()
	storage := &fakeStorage{listErr: errors.New("DB down")}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	if err := p.Start(context.Background()); err == nil {
		t.Error("Start should propagate List error")
	}
}

func TestPoller_Stop_UnregistersJobsAndStopsScheduler(t *testing.T) {
	t.Parallel()
	storage := &fakeStorage{
		targets: []*database.PollingTarget{
			{ID: "t-1", Enabled: true, PollIntervalSec: 60},
		},
	}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if sched.jobCount() != 0 {
		t.Errorf("scheduler.jobCount = %d, want 0", sched.jobCount())
	}
	if !sched.stopped {
		t.Error("scheduler.Stop was not called")
	}
}

func TestPoller_Stop_NotStartedReturnsNil(t *testing.T) {
	t.Parallel()
	p := snmp.NewPoller(&fakeStorage{}, newFakeScheduler(), silentLogger())
	if err := p.Stop(context.Background()); err != nil {
		t.Errorf("Stop without Start = %v, want nil", err)
	}
}

func TestPoller_RunChain_InvokesEveryCollectorInOrder(t *testing.T) {
	t.Parallel()
	target := &database.PollingTarget{
		ID: "t-1", Name: "router-1", IPAddress: "10.0.0.1",
		Enabled: true, PollIntervalSec: 60,
		CollectorChain: []string{"sys_info", "if_table"},
	}
	storage := &fakeStorage{targets: []*database.PollingTarget{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())

	sys := &stubCollector{name: "sys_info"}
	ift := &stubCollector{name: "if_table"}
	p.RegisterCollector(sys)
	p.RegisterCollector(ift)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	job := sched.firstJob()
	if job == nil {
		t.Fatal("expected job snmp:t-1")
	}
	_ = job.Run(context.Background())

	if sys.callCount() != 1 {
		t.Errorf("sys_info called %d times, want 1", sys.callCount())
	}
	if ift.callCount() != 1 {
		t.Errorf("if_table called %d times, want 1", ift.callCount())
	}
	if len(storage.updates) != 1 {
		t.Fatalf("UpdateLastPoll called %d times, want 1", len(storage.updates))
	}
	if storage.updates[0].status != "ok" {
		t.Errorf("status = %q, want ok", storage.updates[0].status)
	}
}

func TestPoller_RunChain_UnknownCollectorIsSkipped(t *testing.T) {
	t.Parallel()
	target := &database.PollingTarget{
		ID: "t-1", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 60,
		CollectorChain:  []string{"unknown_kind", "sys_info"},
	}
	storage := &fakeStorage{targets: []*database.PollingTarget{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())

	sys := &stubCollector{name: "sys_info"}
	p.RegisterCollector(sys)

	_ = p.Start(context.Background())
	job := sched.firstJob()
	_ = job.Run(context.Background())

	// sys_info should still run even though unknown_kind was first.
	if sys.callCount() != 1 {
		t.Errorf("sys_info called %d times, want 1", sys.callCount())
	}
	if len(storage.updates) != 1 || storage.updates[0].status != "error" {
		t.Errorf("expected one update with status=error; got %+v", storage.updates)
	}
}

func TestPoller_RunChain_CollectorErrorCapturedInLastError(t *testing.T) {
	t.Parallel()
	target := &database.PollingTarget{
		ID: "t-1", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 60,
		CollectorChain:  []string{"sys_info"},
	}
	storage := &fakeStorage{targets: []*database.PollingTarget{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())

	sys := &stubCollector{name: "sys_info", err: errors.New("snmp timeout")}
	p.RegisterCollector(sys)

	_ = p.Start(context.Background())
	job := sched.firstJob()
	_ = job.Run(context.Background())

	if len(storage.updates) != 1 {
		t.Fatalf("UpdateLastPoll called %d times, want 1", len(storage.updates))
	}
	if storage.updates[0].status != "error" {
		t.Errorf("status = %q, want error", storage.updates[0].status)
	}
	if storage.updates[0].errMsg != "snmp timeout" {
		t.Errorf("errMsg = %q, want %q", storage.updates[0].errMsg, "snmp timeout")
	}
}

func TestPoller_TargetJob_NextRunCadence(t *testing.T) {
	t.Parallel()
	target := &database.PollingTarget{
		ID: "t-1", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 300,
		CollectorChain:  []string{"sys_info"},
	}
	storage := &fakeStorage{targets: []*database.PollingTarget{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.RegisterCollector(&stubCollector{name: "sys_info"})

	_ = p.Start(context.Background())
	job := sched.firstJob()
	if job == nil {
		t.Fatal("no job registered")
	}

	now := time.Now().UTC()
	first := job.NextRun(now)
	if !first.Equal(now) {
		t.Errorf("first NextRun = %v, want now (immediate first run)", first)
	}
	_ = job.Run(context.Background())
	second := job.NextRun(now)
	if !second.After(first) {
		t.Errorf("second NextRun = %v, want after first %v", second, first)
	}
}
