package api_test

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/outbox"
)

// panicOnceStore is an outbox.Store whose SECOND drain panics and whose later
// drains succeed. It is the forced fault in D-SEED-28's acceptance ("a forced
// panic in the outbox loop is logged and restarted and the daemon stays up").
//
// The second drain, not the first, because the first is the replay the relay
// performs on entry: a fault there is raised on the caller's goroutine, which a
// relay that still spawned its own poll loop would also survive. Only the
// ticker's drain proves the LOOP runs under the supervisor.
type panicOnceStore struct {
	fetches atomic.Int32
}

func (s *panicOnceStore) FetchUnpublished(context.Context, int) ([]outbox.Record, error) {
	if s.fetches.Add(1) == 2 {
		panic("forced outbox fault")
	}
	return nil, nil
}

func (s *panicOnceStore) MarkPublished(context.Context, []string) error { return nil }

func (s *panicOnceStore) DeletePublishedBefore(context.Context, time.Time) (int, error) {
	return 0, nil
}

// childEnv marks the re-executed child of the subprocess test below.
const childEnv = "SEED_TEST_BACKGROUND_PANIC_CHILD"

// TestBackgroundComponentsSurviveOutboxPanic runs the acceptance in a child
// process because the thing under test is whether the DAEMON stays up: an
// unrecovered panic in a background goroutine takes the whole process down, and
// no in-process assertion can observe that from inside the process it kills.
// The parent asserts the child exited 0 and logged the supervisor's restart
// line; before the relay loop ran under pkg/supervise the child died with the
// panic instead.
func TestBackgroundComponentsSurviveOutboxPanic(t *testing.T) {
	if os.Getenv(childEnv) == "1" {
		runBackgroundPanicChild()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestBackgroundComponentsSurviveOutboxPanic", "-test.timeout=60s")
	cmd.Env = append(os.Environ(), childEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child process died instead of surviving the outbox panic: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "CHILD-SURVIVED") {
		t.Fatalf("child did not reach the end of the run:\n%s", text)
	}
	if !strings.Contains(text, "supervised worker failed, restarting") {
		t.Fatalf("supervisor did not log the restart line:\n%s", text)
	}
	if !strings.Contains(text, "worker=outbox") {
		t.Fatalf("restart line does not name the outbox worker:\n%s", text)
	}
}

// runBackgroundPanicChild is the child half: start the background components
// with a relay whose first drain panics, and report that the process is still
// alive and draining afterwards.
func runBackgroundPanicChild() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(log) // what logging.GetLogger() falls back to, so the supervisor logs here

	store := &panicOnceStore{}
	bus := events.New(log)
	relay := outbox.NewRelay(store, bus, log, outbox.WithInterval(10*time.Millisecond))
	components := &api.BackgroundComponents{Outbox: relay}

	ctx := context.Background()
	if err := components.Start(ctx); err != nil {
		log.Error("child start failed", "error", err.Error())
		os.Exit(1)
	}

	// Two drains after the panicking one prove the loop was restarted rather
	// than merely recovered-and-abandoned: the restart's own replay, then a
	// tick of the restarted loop.
	deadline := time.Now().Add(20 * time.Second)
	for store.fetches.Load() < 4 {
		if time.Now().After(deadline) {
			log.Error("relay did not resume draining after the panic",
				"fetches", store.fetches.Load())
			os.Exit(1)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := components.Stop(); err != nil {
		log.Error("child stop failed", "error", err.Error())
		os.Exit(1)
	}
	os.Stdout.WriteString("CHILD-SURVIVED\n")
}
