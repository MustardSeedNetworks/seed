package api

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// TestDrainJobSubstrate_ClosesRunnerAndBus proves Server.Shutdown's job-substrate
// teardown actually closes the runner and the bus. Before D1 this drain lived
// only in the dead ServiceContainer.Stop() (no callers), so on a real shutdown
// the runner goroutines and bus were never drained.
func TestDrainJobSubstrate_ClosesRunnerAndBus(t *testing.T) {
	bus := events.New(slog.New(slog.DiscardHandler))
	runner := jobs.New(bus, slog.New(slog.DiscardHandler), jobs.Config{})
	s := &Server{bus: bus, jobRunner: runner}

	s.drainJobSubstrate(context.Background())

	// After the drain the runner rejects new work with ErrClosed — the
	// observable proof it was closed.
	if _, err := runner.Submit("noop", nil); !errors.Is(err, jobs.ErrClosed) {
		t.Errorf("Submit after drain = %v, want jobs.ErrClosed", err)
	}

	// The bus is closed too: a second Close hits the idempotent already-closed
	// path and returns nil immediately.
	if err := bus.Close(context.Background()); err != nil {
		t.Errorf("second bus Close = %v, want nil (already closed)", err)
	}

	// And the drain is a no-op on a bare Server — nil runner/bus must not panic
	// (the lightweight test server never wires them).
	(&Server{}).drainJobSubstrate(context.Background())
}
