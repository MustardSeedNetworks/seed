package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// shutdownDrainTimeout bounds the drain in these tests. It is long enough that
// a correct shutdown never reaches it and short enough that a wrong ordering
// fails the test in seconds rather than at the Go test deadline.
const shutdownDrainTimeout = 3 * time.Second

// serveOnLoopback puts handler behind the server's own httpServer field on a
// loopback listener, which is what Shutdown drains. Tests drive real requests
// against the returned base URL.
func serveOnLoopback(t *testing.T, s *Server, handler http.Handler) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s.httpServer = &http.Server{Handler: handler, ReadHeaderTimeout: shutdownDrainTimeout}

	served := make(chan struct{})
	go func() {
		defer close(served)
		if serveErr := s.httpServer.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			t.Errorf("Serve = %v, want http.ErrServerClosed", serveErr)
		}
	}()
	t.Cleanup(func() {
		_ = s.httpServer.Close()
		<-served
	})

	return "http://" + ln.Addr().String()
}

// get issues a plain GET bound to the test's context. The caller owns the body.
func get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

// TestShutdownCompletesInFlightRequestBeforeClosingTheDatabase is #2748's
// discriminating case. Shutdown used to stop every service and close the
// database *before* it drained the HTTP server, so a request still in a
// handler when SIGTERM arrived ran on torn-down dependencies. The handler here
// parks until the shutdown is under way and only then touches the database:
// on the old ordering that read fails with "sql: database is closed".
func TestShutdownCompletesInFlightRequestBeforeClosingTheDatabase(t *testing.T) {
	s := NewTestServer()
	t.Cleanup(s.Close)
	SetTestDB(s, newShutdownTestDB(t))

	entered := make(chan struct{})
	release := make(chan struct{})
	dbErr := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/inflight", func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		dbErr <- s.db().Ping(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	baseURL := serveOnLoopback(t, s, mux)

	responses := make(chan *http.Response, 1)
	go func() {
		resp, err := get(t.Context(), baseURL+"/inflight")
		if err != nil {
			t.Errorf("in-flight GET: %v", err)
			responses <- nil
			return
		}
		responses <- resp
	}()

	<-entered // the request is in the handler, exactly where SIGTERM catches one

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownDrainTimeout)
		defer cancel()
		shutdownDone <- s.Shutdown(ctx)
	}()

	// Give Shutdown a moment to get past the point where the old ordering had
	// already closed the database, then let the handler use it.
	time.Sleep(100 * time.Millisecond)
	close(release)

	if err := <-dbErr; err != nil {
		t.Errorf("in-flight handler reached the database: %v, want nil (the drain must precede the teardown)", err)
	}

	resp := <-responses
	if resp == nil {
		t.Fatal("in-flight request did not return a response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("in-flight response = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("Shutdown = %v, want nil", err)
		}
	case <-time.After(shutdownDrainTimeout):
		t.Fatal("Shutdown did not return")
	}
}

// TestShutdownReleasesSSEClientsSoTheDrainCompletes pins the constraint that
// makes "drain first" workable at all: an SSE handler parks on its client
// channel, so unless the hub is shut down before the drain — and unless the
// handler's unregister can give up once the hub's Run loop has gone — a single
// connected browser tab holds [http.Server.Shutdown] open until its deadline.
func TestShutdownReleasesSSEClientsSoTheDrainCompletes(t *testing.T) {
	s := NewTestServer()
	t.Cleanup(s.Close)
	go s.sseHub().Run()

	token, err := s.authManager().GenerateAccessToken(t.Context(), "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	streaming := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		close(streaming)
		s.handleSSE(w, r)
	})
	baseURL := serveOnLoopback(t, s, mux)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, baseURL+"/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE response = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	<-streaming

	// Read the initial state so the handler is parked in its event loop rather
	// than still writing.
	buf := make([]byte, 1)
	if _, err = resp.Body.Read(buf); err != nil {
		t.Fatalf("read SSE stream: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownDrainTimeout)
	defer cancel()

	start := time.Now()
	if err = s.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown with an SSE client connected = %v, want nil", err)
	}
	if elapsed := time.Since(start); elapsed >= shutdownDrainTimeout {
		t.Errorf("Shutdown took %s: it waited out the drain deadline instead of releasing the SSE client", elapsed)
	}
}

// TestUnregisterClientGivesUpAfterTheHubStops covers the unregister half on its
// own. The hub's Run loop is the only reader of the unregister channel and it
// returns on shutdown, so a plain send would park the caller for ever.
func TestUnregisterClientGivesUpAfterTheHubStops(t *testing.T) {
	hub := NewSSEHub()
	go hub.Run()
	client := hub.newClient()
	hub.register <- client

	hub.Shutdown()

	returned := make(chan struct{})
	go func() {
		defer close(returned)
		hub.unregisterClient(client)
	}()

	select {
	case <-returned:
	case <-time.After(shutdownDrainTimeout):
		t.Fatal("unregisterClient blocked after the hub stopped")
	}
}

// newShutdownTestDB opens a throwaway SQLite database for the drain tests. It
// is deliberately NOT registered for cleanup close: Shutdown closes it, which
// is the behaviour under test.
func newShutdownTestDB(t *testing.T) *database.DB {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "shutdown-drain.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return db
}

// TestShutdownTearsDownEvenWhenTheDrainTimesOut is the other half of the
// ordering: a handler that never returns must not keep the teardown from
// running, or a wedged request would leak every background goroutine and the
// database handle. The drain's failure is reported, not swallowed.
func TestShutdownTearsDownEvenWhenTheDrainTimesOut(t *testing.T) {
	s := NewTestServer()
	t.Cleanup(s.Close)
	db := newShutdownTestDB(t)
	SetTestDB(s, db)

	entered := make(chan struct{})
	wedged := make(chan struct{})
	t.Cleanup(func() { close(wedged) })

	mux := http.NewServeMux()
	mux.HandleFunc("/wedged", func(_ http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-wedged
	})
	baseURL := serveOnLoopback(t, s, mux)

	go func() {
		resp, getErr := get(t.Context(), baseURL+"/wedged")
		if getErr == nil {
			_ = resp.Body.Close()
		}
	}()
	<-entered

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := s.Shutdown(ctx)
	if err == nil {
		t.Error("Shutdown = nil, want the drain timeout reported")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Shutdown = %v, want a context.DeadlineExceeded", err)
	}
	if pingErr := db.Ping(context.Background()); pingErr == nil {
		t.Error("database is still open after a timed-out drain: the teardown was skipped")
	}
}
