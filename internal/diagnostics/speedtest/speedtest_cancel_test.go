package speedtest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
)

// TestRunTestHonorsCancelledContext verifies RunTest returns promptly with the
// context error when its context is already cancelled, instead of ignoring the
// context and proceeding to hit the network for the full run. The first
// phase-boundary check fires before any server lookup, so this is fast and
// hermetic (no network).
func TestRunTestHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	tester := speedtest.NewTester()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before the run starts

	done := make(chan error, 1)
	go func() {
		_, err := tester.RunTest(ctx)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RunTest err = %v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunTest did not honor the cancelled context (still running)")
	}

	// The guard must also release the running flag so a later run is possible.
	if tester.GetStatus().Running {
		t.Fatal("Running flag stuck true after a cancelled run")
	}
}
