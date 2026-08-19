package snmp_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// fakeStorage mirrors a tiny subset of database.PollingTargetRepository.
type fakeStorage struct {
	mu      sync.Mutex
	targets []*polling.Target
	listErr error

	updates []updateRecord
	updErr  error
}

type updateRecord struct {
	id     string
	status string
	errMsg string
}

func (f *fakeStorage) List(_ context.Context, _ string) ([]*polling.Target, error) {
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
		targets: []*polling.Target{
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
	p.SetCredentialResolver(testResolver(t))

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
		targets: []*polling.Target{
			{ID: "t-1", Enabled: true, PollIntervalSec: 60},
		},
	}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.SetCredentialResolver(testResolver(t))

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
	p.SetCredentialResolver(testResolver(t))
	if err := p.Start(context.Background()); err == nil {
		t.Error("Start should propagate List error")
	}
}

func TestPoller_Stop_UnregistersJobsAndStopsScheduler(t *testing.T) {
	t.Parallel()
	storage := &fakeStorage{
		targets: []*polling.Target{
			{ID: "t-1", Enabled: true, PollIntervalSec: 60},
		},
	}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.SetCredentialResolver(testResolver(t))

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
	target := &polling.Target{
		ID: "t-1", Name: "router-1", IPAddress: "10.0.0.1",
		Enabled: true, PollIntervalSec: 60, CredentialsID: "cred-1",
		CollectorChain: []string{"sys_info", "if_table"},
	}
	storage := &fakeStorage{targets: []*polling.Target{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.SetCredentialResolver(testResolver(t))

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
	target := &polling.Target{
		ID: "t-1", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 60, CredentialsID: "cred-1",
		CollectorChain: []string{"unknown_kind", "sys_info"},
	}
	storage := &fakeStorage{targets: []*polling.Target{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.SetCredentialResolver(testResolver(t))

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
	target := &polling.Target{
		ID: "t-1", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 60, CredentialsID: "cred-1",
		CollectorChain: []string{"sys_info"},
	}
	storage := &fakeStorage{targets: []*polling.Target{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.SetCredentialResolver(testResolver(t))

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
	target := &polling.Target{
		ID: "t-1", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 300,
		CollectorChain:  []string{"sys_info"},
	}
	storage := &fakeStorage{targets: []*polling.Target{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.SetCredentialResolver(testResolver(t))
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

// testResolver builds a resolver that always succeeds, so poller tests exercise
// the collector chain rather than credential resolution. Resolution itself —
// including its fail-closed behaviour — is covered in credentials_test.go.
func testResolver(t *testing.T) *snmp.CredentialResolver {
	t.Helper()
	r, err := snmp.NewCredentialResolver(
		&fakeCredStore{creds: &polling.Credentials{SNMPCommunityCT: "enc:v1:public"}},
		fakeDecrypter{},
	)
	if err != nil {
		t.Fatalf("NewCredentialResolver: %v", err)
	}
	return r
}

// secretPlaintext is what the decrypter below yields for the two columns that
// do decrypt. It must never appear in a log line, an error, or last_error.
const secretPlaintext = "s3cr3t-community"

// partialDecrypter decrypts the community but fails on the v3 priv secret, so
// the resolver holds recovered plaintext at the moment it returns an error —
// the state in which a careless wrap or log would leak it.
type partialDecrypter struct{}

func (partialDecrypter) DecryptValue(encrypted string) (string, error) {
	if encrypted == "enc:v1:priv" {
		return "", errors.New("invalid ciphertext: authentication failed")
	}
	return secretPlaintext, nil
}

// TestPoller_RunChain_CredentialFailureLeaksNoSecret pins the other half of the
// contract: when resolution fails partway, nothing the poller writes carries the
// plaintext it had already recovered — not the log, not the error, not the
// last_error the operator reads.
func TestPoller_RunChain_CredentialFailureLeaksNoSecret(t *testing.T) {
	t.Parallel()
	target := &polling.Target{
		ID: "t-1", ClientID: "default", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 60, CredentialsID: "cred-1",
		CollectorChain: []string{"sys_info"},
	}
	storage := &fakeStorage{targets: []*polling.Target{target}}
	sched := newFakeScheduler()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	resolver, err := snmp.NewCredentialResolver(
		&fakeCredStore{creds: &polling.Credentials{
			ID:              "cred-1",
			SNMPCommunityCT: "enc:v1:community",
			SNMPv3AuthCT:    "enc:v1:auth",
			SNMPv3PrivCT:    "enc:v1:priv",
		}},
		partialDecrypter{},
	)
	if err != nil {
		t.Fatalf("NewCredentialResolver: %v", err)
	}

	p := snmp.NewPoller(storage, sched, logger)
	p.SetCredentialResolver(resolver)
	p.RegisterCollector(&stubCollector{name: "sys_info"})

	if startErr := p.Start(context.Background()); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	_ = sched.firstJob().Run(context.Background())

	if strings.Contains(logBuf.String(), secretPlaintext) {
		t.Errorf("decrypted secret leaked into the poller log: %s", logBuf.String())
	}
	if len(storage.updates) != 1 {
		t.Fatalf("UpdateLastPoll called %d times, want 1", len(storage.updates))
	}
	if strings.Contains(storage.updates[0].errMsg, secretPlaintext) {
		t.Errorf("decrypted secret leaked into last_error: %s", storage.updates[0].errMsg)
	}
	if !strings.Contains(logBuf.String(), "t-1") {
		t.Error("the failure log should still name the target, so suppressing the " +
			"secret does not also suppress the diagnosis")
	}
}

// TestPoller_RunChain_SkipsTargetWhenCredentialsUnresolved pins the security
// contract: a target whose credentials cannot be resolved must not have any
// collector run against it. Before this, resolution returned empty credentials
// with no error and the whole chain ran unauthenticated.
func TestPoller_RunChain_SkipsTargetWhenCredentialsUnresolved(t *testing.T) {
	t.Parallel()
	target := &polling.Target{
		ID: "t-1", IPAddress: "10.0.0.1", Enabled: true,
		PollIntervalSec: 60, CredentialsID: "",
		CollectorChain: []string{"sys_info"},
	}
	storage := &fakeStorage{targets: []*polling.Target{target}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(storage, sched, silentLogger())
	p.SetCredentialResolver(testResolver(t))

	sys := &stubCollector{name: "sys_info"}
	p.RegisterCollector(sys)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	job := sched.firstJob()
	if job == nil {
		t.Fatal("expected job snmp:t-1")
	}
	_ = job.Run(context.Background())

	if sys.callCount() != 0 {
		t.Errorf("collector ran %d times against a target with unresolved credentials",
			sys.callCount())
	}
	if len(storage.updates) != 1 {
		t.Fatalf("UpdateLastPoll called %d times, want 1", len(storage.updates))
	}
	if storage.updates[0].status != "error" {
		t.Errorf("status = %q, want error so the operator sees why it is not polling",
			storage.updates[0].status)
	}
}
