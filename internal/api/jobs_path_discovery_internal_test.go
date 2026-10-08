package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// fakeFlowTrace alternates between two routes by flow identifier, the shape an
// ECMP pair hashing on it produces. When hold is set, the third trace blocks
// until the job is cancelled.
type fakeFlowTrace struct {
	mu       sync.Mutex
	calls    int
	maxHops  int
	timeout  time.Duration
	hold     bool
	started  chan struct{}
	traceErr error
}

func (f *fakeFlowTrace) trace(
	ctx context.Context,
	target string,
	maxHops, flowID int,
	timeout time.Duration,
) (*discovery.TracerouteResult, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	f.maxHops, f.timeout = maxHops, timeout
	f.mu.Unlock()

	if f.traceErr != nil {
		return nil, f.traceErr
	}
	if f.hold && call == 3 {
		close(f.started)
		<-ctx.Done()
		return flowRoute(target, "10.0.0.1"), nil
	}
	if flowID%2 == 0 {
		return flowRoute(target, "10.0.0.1", "10.0.1.1", "203.0.113.1"), nil
	}
	return flowRoute(target, "10.0.0.1", "10.0.2.1", "203.0.113.1"), nil
}

func flowRoute(target string, hops ...string) *discovery.TracerouteResult {
	result := &discovery.TracerouteResult{Target: target, TargetIP: "203.0.113.1", Protocol: "icmp"}
	for i, ip := range hops {
		result.Hops = append(result.Hops, discovery.TracerouteHop{TTL: i + 1, IP: ip, State: "reply"})
	}
	result.Completed = hops[len(hops)-1] == "203.0.113.1"
	return result
}

// fakePathMTU records what it was asked and returns a fixed measurement, or
// blocks until cancelled and reports an incomplete search, as the engine does.
type fakePathMTU struct {
	target  string
	timeout time.Duration
	hold    bool
	started chan struct{}
	err     error
}

func (f *fakePathMTU) measure(
	ctx context.Context,
	target string,
	timeout time.Duration,
) (*discovery.PMTUDResult, error) {
	f.target, f.timeout = target, timeout
	if f.err != nil {
		return nil, f.err
	}
	if f.hold {
		close(f.started)
		<-ctx.Done()
		return &discovery.PMTUDResult{Target: target, Status: discovery.PMTUDStatusIncomplete, PathMTU: 1280}, nil
	}
	return &discovery.PMTUDResult{Target: target, Status: discovery.PMTUDStatusOK, PathMTU: 1400, LocalMTU: 1500}, nil
}

func submitPathDiscovery(
	t *testing.T,
	trace *fakeFlowTrace,
	measure *fakePathMTU,
	kind, params string,
) (*jobs.Runner, string) {
	t.Helper()
	srv, runner := newJobsTestServer(t, jobs.Config{})
	srv.registerPathDiscoveryKinds(trace.trace, measure.measure)
	id, err := runner.Submit(kind, json.RawMessage(params))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return runner, id
}

func TestMultiPathKindReportsBothRoutesOfAnECMPPair(t *testing.T) {
	t.Parallel()

	trace := &fakeFlowTrace{}
	runner, id := submitPathDiscovery(t, trace, &fakePathMTU{}, multiPathJobKind, `{"destination":"203.0.113.1"}`)
	j := waitForState(t, runner, id, jobs.StateSucceeded)

	res, ok := j.Result.(*discovery.MultiPathResult)
	if !ok {
		t.Fatalf("result = %#v, want a multi-path result", j.Result)
	}
	if res.Attempts != discovery.MultiPathAttempts || len(res.Paths) != 2 || res.DivergesAtTTL != 2 {
		t.Fatalf("result = %+v, want %d attempts over two routes splitting at hop 2",
			res, discovery.MultiPathAttempts)
	}
	if trace.maxHops != multiPathMaxHops || trace.timeout != multiPathHopTimeout {
		t.Errorf("traced with maxHops=%d timeout=%v, want the probe budget %d/%v",
			trace.maxHops, trace.timeout, multiPathMaxHops, multiPathHopTimeout)
	}
	if j.Progress != 1 {
		t.Errorf("progress = %v, want 1", j.Progress)
	}
}

func TestMultiPathKindClampsMaxHopsToTheBudget(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		params string
		want   int
	}{
		{"within", `{"destination":"203.0.113.1","maxHops":8}`, 8},
		{"over", `{"destination":"203.0.113.1","maxHops":64}`, multiPathMaxHops},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			trace := &fakeFlowTrace{}
			runner, id := submitPathDiscovery(t, trace, &fakePathMTU{}, multiPathJobKind, tc.params)
			waitForState(t, runner, id, jobs.StateSucceeded)
			if trace.maxHops != tc.want {
				t.Fatalf("maxHops = %d, want %d", trace.maxHops, tc.want)
			}
		})
	}
}

// Stopping a run keeps the routes already traced; the trace the cancel cut
// short is not one of them.
func TestCancelledMultiPathKeepsTheTracesThatFinished(t *testing.T) {
	t.Parallel()

	trace := &fakeFlowTrace{hold: true, started: make(chan struct{})}
	runner, id := submitPathDiscovery(t, trace, &fakePathMTU{}, multiPathJobKind, `{"destination":"203.0.113.1"}`)
	<-trace.started
	if err := runner.Cancel(id); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	j := waitForState(t, runner, id, jobs.StateSucceeded)

	res, ok := j.Result.(*discovery.MultiPathResult)
	if !ok || res.Attempts != 2 || len(res.Paths) != 2 {
		t.Fatalf("result = %#v, want the two traces that finished", j.Result)
	}
}

// A platform that cannot hold the flow identifier fixed fails the job with
// that reason, rather than reporting a single route it never compared.
func TestMultiPathKindReportsAnUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	trace := &fakeFlowTrace{traceErr: discovery.ErrMultiPathUnsupported}
	runner, id := submitPathDiscovery(t, trace, &fakePathMTU{}, multiPathJobKind, `{"destination":"203.0.113.1"}`)
	j := waitForState(t, runner, id, jobs.StateFailed)
	if j.Err != discovery.ErrMultiPathUnsupported.Error() {
		t.Fatalf("Err = %q, want %q", j.Err, discovery.ErrMultiPathUnsupported)
	}
	if trace.calls != 1 {
		t.Errorf("traced %d times after the platform refused, want 1", trace.calls)
	}
}

func TestPathMTUKindPassesTheDestinationAndBudget(t *testing.T) {
	t.Parallel()

	measure := &fakePathMTU{}
	runner, id := submitPathDiscovery(t, &fakeFlowTrace{}, measure, pathMTUJobKind, `{"destination":"203.0.113.1"}`)
	j := waitForState(t, runner, id, jobs.StateSucceeded)

	if measure.target != "203.0.113.1" || measure.timeout != pathMTUProbeTimeout {
		t.Fatalf("measured %q with timeout %v, want 203.0.113.1 with %v",
			measure.target, measure.timeout, pathMTUProbeTimeout)
	}
	res, ok := j.Result.(*discovery.PMTUDResult)
	if !ok || res.Status != discovery.PMTUDStatusOK || res.PathMTU != 1400 {
		t.Fatalf("result = %#v, want the measurement", j.Result)
	}
}

func TestCancelledPathMTUKeepsTheLowerBound(t *testing.T) {
	t.Parallel()

	measure := &fakePathMTU{hold: true, started: make(chan struct{})}
	runner, id := submitPathDiscovery(t, &fakeFlowTrace{}, measure, pathMTUJobKind, `{"destination":"203.0.113.1"}`)
	<-measure.started
	if err := runner.Cancel(id); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	j := waitForState(t, runner, id, jobs.StateSucceeded)

	res, ok := j.Result.(*discovery.PMTUDResult)
	if !ok || res.Status != discovery.PMTUDStatusIncomplete || res.PathMTU != 1280 {
		t.Fatalf("result = %#v, want the incomplete search", j.Result)
	}
}

func TestPathMTUKindReportsTheSetupError(t *testing.T) {
	t.Parallel()

	measure := &fakePathMTU{err: discovery.ErrPMTUDUnsupported}
	runner, id := submitPathDiscovery(t, &fakeFlowTrace{}, measure, pathMTUJobKind, `{"destination":"203.0.113.1"}`)
	if j := waitForState(t, runner, id, jobs.StateFailed); j.Err != discovery.ErrPMTUDUnsupported.Error() {
		t.Fatalf("Err = %q, want %q", j.Err, discovery.ErrPMTUDUnsupported)
	}
}

func TestPathDiscoveryKindsRejectBadParams(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"missing":             ``,
		"not json":            `{`,
		"no destination":      `{}`,
		"unknown field":       `{"destination":"203.0.113.1","ttl":4}`,
		"wrong type for hops": `{"destination":"203.0.113.1","maxHops":"8"}`,
	}
	for _, kind := range []string{multiPathJobKind, pathMTUJobKind} {
		for name, params := range cases {
			t.Run(kind+"/"+name, func(t *testing.T) {
				t.Parallel()
				runner, id := submitPathDiscovery(t, &fakeFlowTrace{}, &fakePathMTU{}, kind, params)
				if j := waitForState(t, runner, id, jobs.StateFailed); j.Err == "" {
					t.Fatal("failed job has an empty error")
				}
			})
		}
	}
}

// Both kinds are path analysis, sold at Pro with /path and the monitor, and
// POST /jobs is reachable from every tier, so the kind is the boundary.
func TestPathDiscoveryKindsRequirePathAnalysis(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{multiPathJobKind, pathMTUJobKind} {
		for _, tc := range []struct {
			name, key string
			gated     bool
		}{
			{"free", "", true},
			{"starter", prodSeedStarterVector, true},
			{"pro", prodSeedProVector, false},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				srv := newLicensedJobsServer(t, tc.key)
				srv.registerPathDiscoveryKinds((&fakeFlowTrace{}).trace, (&fakePathMTU{}).measure)
				body, _ := json.Marshal(CreateJobRequest{
					Kind:   kind,
					Params: json.RawMessage(`{"destination":"203.0.113.1"}`),
				})
				w := httptest.NewRecorder()
				srv.handleJobs(w, httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/jobs", bytes.NewReader(body)))

				if tc.gated {
					assertPathAnalysisGate(t, w)
					return
				}
				if w.Code != http.StatusCreated {
					t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
				}
			})
		}
	}
}

// Until #395/#435 the engines had no caller; registerJobKinds is what puts
// them on POST /jobs.
func TestPathDiscoveryKindsAreRegisteredAtStartup(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	srv.registerJobKinds()
	for _, kind := range []string{multiPathJobKind, pathMTUJobKind} {
		if _, err := runner.Submit(kind, json.RawMessage(`{`)); errors.Is(err, jobs.ErrUnknownKind) {
			t.Errorf("%s is not registered", kind)
		}
	}
}
