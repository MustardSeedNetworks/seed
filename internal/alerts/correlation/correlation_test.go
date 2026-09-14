package correlation_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
)

// recordingStore is the alert store the decorator wraps: it assigns the
// ids SQLite would and keeps what it was handed, so a test can assert on
// the RootCauseID the decorator set *before* the write.
type recordingStore struct {
	next    int64
	written []alerts.Alert
	err     error
}

func (s *recordingStore) Create(_ context.Context, alert *alerts.Alert) error {
	if s.err != nil {
		return s.err
	}
	s.next++
	alert.ID = s.next
	s.written = append(s.written, *alert)
	return nil
}

// clock is a hand-advanced time source; correlation is entirely about
// what happened inside a window, so the test must own the clock.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

// testWindow is short enough to step past in one clock advance and long
// enough that the "two seconds later" cases sit well inside it.
const testWindow = 5 * time.Minute

// device is the switch under test. Correlation is per-device, so a second
// device only appears in the test that needs one.
const device = "10.51.200.1"

func newFixture() (*recordingStore, *clock, correlation.Writer) {
	store := &recordingStore{}
	c := &clock{t: time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)}
	w := correlation.WrapWriter(store, correlation.Config{
		Window: testWindow,
		Now:    c.now,
	})
	return store, c, w
}

func ifaceDown() *alerts.Alert {
	return &alerts.Alert{
		Rule: "iface.down", Type: alerts.TypeConnectivity,
		Severity: alerts.SeverityWarning, Title: "Interface down", Source: device,
	}
}

func bgpFlap(source string) *alerts.Alert {
	return &alerts.Alert{
		Rule: "bgp.flap", Type: alerts.TypeConnectivity,
		Severity: alerts.SeverityError, Title: "BGP peer left Established", Source: source,
	}
}

// The whole point of the slice: a BGP session that drops seconds after the
// interface carrying it went down names that interface as its probable cause.
func TestBGPFlapNamesTheInterfaceDownThatCausedIt(t *testing.T) {
	store, c, w := newFixture()
	ctx := context.Background()

	cause := ifaceDown()
	if err := w.Create(ctx, cause); err != nil {
		t.Fatalf("create cause: %v", err)
	}
	c.t = c.t.Add(2 * time.Second)

	effect := bgpFlap(device)
	if err := w.Create(ctx, effect); err != nil {
		t.Fatalf("create effect: %v", err)
	}

	if effect.RootCauseID == nil {
		t.Fatal("bgp.flap did not name a root cause; want the iface.down alert")
	}
	if *effect.RootCauseID != cause.ID {
		t.Errorf("root cause = %d, want %d (the iface.down alert)", *effect.RootCauseID, cause.ID)
	}
	// The annotation must be on the record the store persisted, not only on
	// the caller's struct — the inbox and the webhook both read the store.
	if got := store.written[1].RootCauseID; got == nil || *got != cause.ID {
		t.Errorf("persisted root cause = %v, want %d", got, cause.ID)
	}
}

// A cause that is not an alert Seed raised must not be invented.
func TestBGPFlapAloneNamesNoCause(t *testing.T) {
	_, _, w := newFixture()
	effect := bgpFlap(device)
	if err := w.Create(context.Background(), effect); err != nil {
		t.Fatalf("create: %v", err)
	}
	if effect.RootCauseID != nil {
		t.Errorf("root cause = %d, want none", *effect.RootCauseID)
	}
}

// Two devices are two stories. An interface down on one switch does not
// explain a BGP session dropping on another.
func TestCauseOnAnotherDeviceIsNotACause(t *testing.T) {
	_, c, w := newFixture()
	ctx := context.Background()

	if err := w.Create(ctx, ifaceDown()); err != nil {
		t.Fatalf("create cause: %v", err)
	}
	c.t = c.t.Add(2 * time.Second)

	effect := bgpFlap("10.51.200.99")
	if err := w.Create(ctx, effect); err != nil {
		t.Fatalf("create effect: %v", err)
	}
	if effect.RootCauseID != nil {
		t.Errorf("root cause = %d, want none (different device)", *effect.RootCauseID)
	}
}

// This is the case that fails when the window check is deleted: an interface
// that went down an hour ago explains nothing about a BGP session now.
func TestCauseOutsideTheWindowIsNotACause(t *testing.T) {
	_, c, w := newFixture()
	ctx := context.Background()

	if err := w.Create(ctx, ifaceDown()); err != nil {
		t.Fatalf("create cause: %v", err)
	}
	c.t = c.t.Add(time.Hour)

	effect := bgpFlap(device)
	if err := w.Create(ctx, effect); err != nil {
		t.Fatalf("create effect: %v", err)
	}
	if effect.RootCauseID != nil {
		t.Errorf("root cause = %d, want none (an hour is not a window)", *effect.RootCauseID)
	}
}

// Causation is directional. An interface going down after a BGP flap is not
// explained by it, and the pair must not correlate backwards.
func TestCorrelationIsDirectional(t *testing.T) {
	_, c, w := newFixture()
	ctx := context.Background()

	if err := w.Create(ctx, bgpFlap(device)); err != nil {
		t.Fatalf("create: %v", err)
	}
	c.t = c.t.Add(2 * time.Second)

	effect := ifaceDown()
	if err := w.Create(ctx, effect); err != nil {
		t.Fatalf("create: %v", err)
	}
	if effect.RootCauseID != nil {
		t.Errorf("root cause = %d, want none (iface.down has no cause rule)", *effect.RootCauseID)
	}
}

// An alert the store rejected never happened, so it must not become the
// named cause of the next one.
func TestRejectedAlertIsNeverACause(t *testing.T) {
	store, c, w := newFixture()
	ctx := context.Background()

	store.err = context.DeadlineExceeded
	if err := w.Create(ctx, ifaceDown()); err == nil {
		t.Fatal("want the store error to propagate")
	}
	store.err = nil
	c.t = c.t.Add(2 * time.Second)

	effect := bgpFlap(device)
	if err := w.Create(ctx, effect); err != nil {
		t.Fatalf("create effect: %v", err)
	}
	if effect.RootCauseID != nil {
		t.Errorf("root cause = %d, want none (the cause was never stored)", *effect.RootCauseID)
	}
}

// A nil config must not silently disable correlation with a zero window.
func TestZeroWindowFallsBackToTheDefault(t *testing.T) {
	store := &recordingStore{}
	c := &clock{t: time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)}
	w := correlation.WrapWriter(store, correlation.Config{Now: c.now})
	ctx := context.Background()

	cause := ifaceDown()
	if err := w.Create(ctx, cause); err != nil {
		t.Fatalf("create cause: %v", err)
	}
	c.t = c.t.Add(time.Second)

	effect := bgpFlap(device)
	if err := w.Create(ctx, effect); err != nil {
		t.Fatalf("create effect: %v", err)
	}
	if effect.RootCauseID == nil || *effect.RootCauseID != cause.ID {
		t.Errorf("root cause = %v, want %d under the default window", effect.RootCauseID, cause.ID)
	}
}
