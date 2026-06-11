package probe_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// fakeChecker is a programmable Checker for engine tests.
type fakeChecker struct {
	kind   string
	result probe.Result
	calls  int
}

func (f *fakeChecker) Kind() string                   { return f.kind }
func (f *fakeChecker) RequiredCapabilities() []string { return nil }
func (f *fakeChecker) Run(_ context.Context, _ probe.Probe) probe.Result {
	f.calls++
	return f.result
}

// silentLogger discards output for tests.
func silentLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestEngine_RegisterAndKinds(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	e.RegisterChecker(&fakeChecker{kind: "tls"})
	e.RegisterChecker(&fakeChecker{kind: "dns"})
	e.RegisterChecker(&fakeChecker{kind: "ping"})

	kinds := e.Kinds()
	want := []string{"dns", "ping", "tls"}
	if len(kinds) != len(want) {
		t.Fatalf("Kinds returned %d, want %d", len(kinds), len(want))
	}
	for i, k := range want {
		if kinds[i] != k {
			t.Errorf("Kinds[%d] = %q, want %q", i, kinds[i], k)
		}
	}
}

func TestEngine_RunDefinition_DispatchesToChecker(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	fc := &fakeChecker{
		kind: "dns",
		result: probe.Result{
			Success:   true,
			LatencyMs: 12.5,
		},
	}
	e.RegisterChecker(fc)

	p := probe.Probe{
		ID:       "p-1",
		ClientID: "default",
		Kind:     "dns",
		Target:   "google.com",
	}
	r := e.RunDefinition(context.Background(), p)

	if fc.calls != 1 {
		t.Errorf("checker called %d times, want 1", fc.calls)
	}
	if !r.Success {
		t.Error("Result.Success = false, want true")
	}
	if r.LatencyMs != 12.5 {
		t.Errorf("Result.LatencyMs = %v, want 12.5", r.LatencyMs)
	}
	if r.ProbeID != "p-1" {
		t.Errorf("Result.ProbeID = %q, want %q", r.ProbeID, "p-1")
	}
	if r.ClientID != "default" {
		t.Errorf("Result.ClientID = %q, want %q", r.ClientID, "default")
	}
	if r.Kind != "dns" {
		t.Errorf("Result.Kind = %q, want %q", r.Kind, "dns")
	}
	if r.Timestamp.IsZero() {
		t.Error("Result.Timestamp should be set")
	}
}

func TestEngine_RunDefinition_NoCheckerRegistered(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	p := probe.Probe{ID: "p-1", Kind: "nonexistent"}
	r := e.RunDefinition(context.Background(), p)

	if r.Success {
		t.Error("Result.Success should be false when no checker registered")
	}
	if r.Error == "" {
		t.Error("Result.Error should describe missing checker")
	}
}

func TestEngine_Thresholds_WarningLatency(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	e.RegisterChecker(&fakeChecker{
		kind: "dns",
		result: probe.Result{
			Success:   true,
			LatencyMs: 150,
		},
	})
	sub := e.Subscribe()

	warningJSON := json.RawMessage(`{"latency_ms": 100}`)
	p := probe.Probe{ID: "p-1", Kind: "dns", Warning: warningJSON}
	e.RunDefinition(context.Background(), p)

	select {
	case evt := <-sub:
		if len(evt.Breaches) != 1 {
			t.Fatalf("got %d breaches, want 1: %+v", len(evt.Breaches), evt.Breaches)
		}
		if evt.Breaches[0].Severity != "warning" {
			t.Errorf("Breach.Severity = %q, want %q", evt.Breaches[0].Severity, "warning")
		}
		if evt.Breaches[0].Field != "latency_ms" {
			t.Errorf("Breach.Field = %q, want %q", evt.Breaches[0].Field, "latency_ms")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEngine_Thresholds_CriticalLatency(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	e.RegisterChecker(&fakeChecker{
		kind: "dns",
		result: probe.Result{
			Success:   true,
			LatencyMs: 600,
		},
	})
	sub := e.Subscribe()

	// Set both warning and critical; both should fire.
	p := probe.Probe{
		ID:       "p-1",
		Kind:     "dns",
		Warning:  json.RawMessage(`{"latency_ms": 100}`),
		Critical: json.RawMessage(`{"latency_ms": 500}`),
	}
	e.RunDefinition(context.Background(), p)

	select {
	case evt := <-sub:
		if len(evt.Breaches) != 2 {
			t.Fatalf("got %d breaches, want 2: %+v", len(evt.Breaches), evt.Breaches)
		}
		severities := map[string]bool{}
		for _, b := range evt.Breaches {
			severities[b.Severity] = true
		}
		if !severities["warning"] || !severities["critical"] {
			t.Errorf("expected both warning + critical breaches, got %v", severities)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

// TestEngine_Thresholds_CertExpiry covers the certificate days-remaining gate.
// Unlike latency (breach when actual exceeds the bound), cert-expiry breaches
// when the remaining days fall BELOW the bound — fewer days is worse. The actual
// value is read from Result.Metadata.days_remaining, which only TLS-family
// checkers publish, so a probe without that metadata skips the gate entirely.
func TestEngine_Thresholds_CertExpiry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		metadata     string // Result.Metadata JSON ("" = none, e.g. a non-TLS probe)
		warning      string // Probe.Warning JSON
		critical     string // Probe.Critical JSON
		wantBreaches int
		wantSevs     []string // severities expected, any order
	}{
		{
			name:         "below warning fires warning",
			metadata:     `{"days_remaining": 20}`,
			warning:      `{"cert_days_remaining": 30}`,
			wantBreaches: 1,
			wantSevs:     []string{"warning"},
		},
		{
			name:         "below both fires warning and critical",
			metadata:     `{"days_remaining": 3}`,
			warning:      `{"cert_days_remaining": 30}`,
			critical:     `{"cert_days_remaining": 7}`,
			wantBreaches: 2,
			wantSevs:     []string{"warning", "critical"},
		},
		{
			name:         "expired cert (negative days) fires critical",
			metadata:     `{"days_remaining": -5}`,
			critical:     `{"cert_days_remaining": 7}`,
			wantBreaches: 1,
			wantSevs:     []string{"critical"},
		},
		{
			name:         "healthy cert above threshold does not breach",
			metadata:     `{"days_remaining": 90}`,
			warning:      `{"cert_days_remaining": 30}`,
			wantBreaches: 0,
		},
		{
			name:         "no cert metadata skips the gate",
			metadata:     "",
			warning:      `{"cert_days_remaining": 30}`,
			wantBreaches: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := probe.NewEngine(silentLogger())
			e.RegisterChecker(&fakeChecker{
				kind: "tls",
				result: probe.Result{
					Success:  true, // a TLS handshake to an expiring cert still succeeds
					Metadata: rawOrNil(tc.metadata),
				},
			})
			sub := e.Subscribe()

			p := probe.Probe{
				ID:       "p-1",
				Kind:     "tls",
				Warning:  rawOrNil(tc.warning),
				Critical: rawOrNil(tc.critical),
			}
			e.RunDefinition(context.Background(), p)

			evt := waitForEvent(t, sub)
			if len(evt.Breaches) != tc.wantBreaches {
				t.Fatalf("got %d breaches, want %d: %+v", len(evt.Breaches), tc.wantBreaches, evt.Breaches)
			}
			gotSevs := certBreachSeverities(t, evt.Breaches)
			for _, want := range tc.wantSevs {
				if !gotSevs[want] {
					t.Errorf("missing %q breach; got %v", want, gotSevs)
				}
			}
		})
	}
}

// waitForEvent reads one ResultEvent from sub or fails after a timeout.
func waitForEvent(t *testing.T, sub <-chan probe.ResultEvent) probe.ResultEvent {
	t.Helper()
	select {
	case evt := <-sub:
		return evt
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return probe.ResultEvent{}
	}
}

// certBreachSeverities asserts every breach is on the cert field and returns the
// set of severities seen, so a test can check which thresholds fired.
func certBreachSeverities(t *testing.T, breaches []probe.Breach) map[string]bool {
	t.Helper()
	sevs := map[string]bool{}
	for _, b := range breaches {
		if b.Field != "cert_days_remaining" {
			t.Errorf("Breach.Field = %q, want %q", b.Field, "cert_days_remaining")
		}
		sevs[b.Severity] = true
	}
	return sevs
}

// rawOrNil returns nil for an empty string so a test case can express "no JSON
// configured" distinctly from an empty object.
func rawOrNil(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}

func TestEngine_FailedProbe_AlwaysBreaches(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	e.RegisterChecker(&fakeChecker{
		kind: "dns",
		result: probe.Result{
			Success: false,
			Error:   "NXDOMAIN",
		},
	})
	sub := e.Subscribe()

	// No thresholds set; failure alone should still produce a breach.
	p := probe.Probe{ID: "p-1", Kind: "dns"}
	e.RunDefinition(context.Background(), p)

	select {
	case evt := <-sub:
		if len(evt.Breaches) != 1 {
			t.Fatalf("got %d breaches, want 1: %+v", len(evt.Breaches), evt.Breaches)
		}
		if evt.Breaches[0].Field != "success" {
			t.Errorf("Breach.Field = %q, want %q", evt.Breaches[0].Field, "success")
		}
		if evt.Breaches[0].Severity != "critical" {
			t.Errorf("Breach.Severity = %q, want %q", evt.Breaches[0].Severity, "critical")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEngine_Subscribe_FanOut(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	e.RegisterChecker(&fakeChecker{kind: "dns", result: probe.Result{Success: true}})

	sub1 := e.Subscribe()
	sub2 := e.Subscribe()

	e.RunDefinition(context.Background(), probe.Probe{ID: "p-1", Kind: "dns"})

	for i, sub := range []<-chan probe.ResultEvent{sub1, sub2} {
		select {
		case <-sub:
			// Got event.
		case <-time.After(time.Second):
			t.Errorf("subscriber %d did not receive event", i)
		}
	}
}

func TestEngine_DropsWhenSubscriberBufferFull(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	e.RegisterChecker(&fakeChecker{kind: "dns", result: probe.Result{Success: true}})

	// Subscribe but never drain.
	_ = e.Subscribe()

	// Emit more events than the buffer can hold to trigger drops.
	const burst = 200 // > defaultSubscriberBufferSize (64)
	for range burst {
		e.RunDefinition(context.Background(), probe.Probe{ID: "p-1", Kind: "dns"})
	}

	if e.DroppedEvents() == 0 {
		t.Error("expected DroppedEvents > 0 when subscriber buffer overflows")
	}
}

func TestEngine_RunDefinition_HonorsCheckerError(t *testing.T) {
	t.Parallel()
	e := probe.NewEngine(silentLogger())
	e.RegisterChecker(&fakeChecker{
		kind: "dns",
		result: probe.Result{
			Success: false,
			Error:   "context deadline exceeded",
		},
	})
	p := probe.Probe{ID: "p-1", Kind: "dns"}
	r := e.RunDefinition(context.Background(), p)
	if r.Success {
		t.Error("Result.Success = true, want false")
	}
	if r.Error == "" {
		t.Error("Result.Error should be propagated from Checker")
	}
}

// Verify the ErrCheckerNotRegistered sentinel is what the package
// exports — defends against accidental rename in future refactors.
func TestEngine_ErrCheckerNotRegisteredIsSentinel(t *testing.T) {
	t.Parallel()
	if !errors.Is(probe.ErrCheckerNotRegistered, probe.ErrCheckerNotRegistered) {
		t.Fatal("ErrCheckerNotRegistered should be its own sentinel")
	}
}
