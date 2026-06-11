package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// --- harness --------------------------------------------------------------

// newJobsTestServer builds a Server populated with only the fields the jobs
// handlers touch (runner, bus, idempotency cache). The middleware chain — auth,
// CSRF, the operator role gate — is applied at registration and exercised by
// the authchain golden harness, not here; these tests drive the handlers
// directly to characterize their own logic.
func newJobsTestServer(t *testing.T, cfg jobs.Config) (*Server, *jobs.Runner) {
	t.Helper()
	bus := events.New(slog.New(slog.DiscardHandler))
	runner := jobs.New(bus, slog.New(slog.DiscardHandler), cfg)
	srv := &Server{
		bus:       bus,
		jobRunner: runner,
		jobIdemp:  newJobIdempotencyCache(16),
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runner.Close(ctx)
		_ = bus.Close(ctx)
	})
	return srv, runner
}

func okKind(result any) jobs.Handler {
	return func(context.Context, any, func(float64)) (any, error) { return result, nil }
}

// blockKind blocks until release is closed or the job ctx is cancelled.
func blockKind(release <-chan struct{}) jobs.Handler {
	return func(ctx context.Context, _ any, _ func(float64)) (any, error) {
		select {
		case <-release:
			return "released", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func decodeJob(t *testing.T, body *bytes.Buffer) JobResponse {
	t.Helper()
	var jr JobResponse
	if err := json.Unmarshal(body.Bytes(), &jr); err != nil {
		t.Fatalf("decode JobResponse: %v (body=%q)", err, body.String())
	}
	return jr
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within deadline: %s", what)
}

// --- create (POST /jobs) --------------------------------------------------

func TestHandleJobsCreate(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	if err := runner.Register("echo", okKind(map[string]any{"ok": true})); err != nil {
		t.Fatalf("Register: %v", err)
	}

	body := `{"kind":"echo","params":{"x":1}}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.handleJobs(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", w.Code, w.Body.String())
	}
	jr := decodeJob(t, w.Body)
	if jr.ID == "" || jr.Kind != "echo" {
		t.Fatalf("JobResponse = %+v, want id set + kind echo", jr)
	}
	if loc := w.Header().Get("Location"); loc != "/api/v1/jobs/"+jr.ID {
		t.Fatalf("Location = %q, want /api/v1/jobs/%s", loc, jr.ID)
	}
}

func TestHandleJobsCreateUnknownKind(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"kind":"nope"}`))
	w := httptest.NewRecorder()
	srv.handleJobs(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for unknown kind", w.Code)
	}
}

func TestHandleJobsCreateMissingKind(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"params":{}}`))
	w := httptest.NewRecorder()
	srv.handleJobs(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for missing kind", w.Code)
	}
}

func TestHandleJobsCreateBadJSON(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{not json`))
	w := httptest.NewRecorder()
	srv.handleJobs(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for malformed JSON", w.Code)
	}
}

func TestHandleJobsMethodNotAllowed(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()
	srv.handleJobs(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 for GET /jobs", w.Code)
	}
}

func TestHandleJobsAtCapacity(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{MaxConcurrent: 1})
	release := make(chan struct{})
	defer close(release)
	_ = runner.Register("slow", blockKind(release))

	// Occupy the only slot.
	first := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"kind":"slow"}`))
	fw := httptest.NewRecorder()
	srv.handleJobs(fw, first)
	if fw.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want 201", fw.Code)
	}

	// A second create must be rejected with 503 (appliance backpressure).
	second := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"kind":"slow"}`))
	sw := httptest.NewRecorder()
	srv.handleJobs(sw, second)
	if sw.Code != http.StatusServiceUnavailable {
		t.Fatalf("second create status = %d, want 503 at capacity", sw.Code)
	}
}

// --- idempotency ----------------------------------------------------------

func TestHandleJobsIdempotentReplay(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{Retention: time.Hour})
	_ = runner.Register("echo", okKind("done"))

	post := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"kind":"echo","params":{"a":1}}`))
		r.Header.Set("Idempotency-Key", "key-123")
		w := httptest.NewRecorder()
		srv.handleJobs(w, r)
		return w
	}

	first := post()
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want 201", first.Code)
	}
	id1 := decodeJob(t, first.Body).ID

	second := post()
	if second.Code != http.StatusOK {
		t.Fatalf("replay status = %d, want 200 (not a new 201)", second.Code)
	}
	if id2 := decodeJob(t, second.Body).ID; id2 != id1 {
		t.Fatalf("replay id = %s, want same job %s", id2, id1)
	}
}

func TestHandleJobsIdempotentConflict(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{Retention: time.Hour})
	_ = runner.Register("echo", okKind("done"))
	_ = runner.Register("other", okKind("done"))

	r1 := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"kind":"echo"}`))
	r1.Header.Set("Idempotency-Key", "dup")
	w1 := httptest.NewRecorder()
	srv.handleJobs(w1, r1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want 201", w1.Code)
	}

	// Same key, different request body -> conflict.
	r2 := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"kind":"other"}`))
	r2.Header.Set("Idempotency-Key", "dup")
	w2 := httptest.NewRecorder()
	srv.handleJobs(w2, r2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("conflicting reuse status = %d, want 409", w2.Code)
	}
}

// --- inspect / cancel (/jobs/{id}) ----------------------------------------

func TestHandleJobByIDGet(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{Retention: time.Hour})
	_ = runner.Register("echo", okKind("payload"))
	id, err := runner.Submit("echo", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job terminal", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+id, nil)
	w := httptest.NewRecorder()
	srv.handleJobByID(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if jr := decodeJob(t, w.Body); jr.ID != id || jr.State != string(jobs.StateSucceeded) {
		t.Fatalf("JobResponse = %+v, want id %s succeeded", jr, id)
	}
}

func TestHandleJobByIDNotFound(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing", nil)
	w := httptest.NewRecorder()
	srv.handleJobByID(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleJobByIDInvalidID(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	// A nested path segment is not a valid job id.
	r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/a/b", nil)
	w := httptest.NewRecorder()
	srv.handleJobByID(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for invalid id", w.Code)
	}
}

func TestHandleJobByIDCancel(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{Retention: time.Hour})
	release := make(chan struct{})
	defer close(release)
	_ = runner.Register("slow", blockKind(release))
	id, _ := runner.Submit("slow", nil)
	waitFor(t, "job running", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateRunning
	})

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+id, nil)
	w := httptest.NewRecorder()
	srv.handleJobByID(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("cancel status = %d, want 202", w.Code)
	}
	waitFor(t, "job cancelled", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateCancelled
	})
}

func TestHandleJobByIDCancelNotFound(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/ghost", nil)
	w := httptest.NewRecorder()
	srv.handleJobByID(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleJobByIDMethodNotAllowed(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodPut, "/api/v1/jobs/x", nil)
	w := httptest.NewRecorder()
	srv.handleJobByID(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// --- SSE (/jobs/events) ---------------------------------------------------

// syncRecorder is a flushable, mutex-guarded ResponseWriter so the test can
// read the streamed body while the handler writes it concurrently.
type syncRecorder struct {
	mu     sync.Mutex
	header http.Header
	buf    bytes.Buffer
}

func newSyncRecorder() *syncRecorder { return &syncRecorder{header: make(http.Header)} }

func (r *syncRecorder) Header() http.Header { return r.header }

func (r *syncRecorder) WriteHeader(int) {}

func (r *syncRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(p)
}

func (r *syncRecorder) Flush() {}

func (r *syncRecorder) snapshot() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

func TestHandleJobsEventsStreamsStateChanges(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{Retention: time.Hour})
	_ = runner.Register("echo", okKind("payload"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/events", nil).WithContext(ctx)
	rec := newSyncRecorder()

	done := make(chan struct{})
	go func() {
		srv.handleJobsEvents(rec, r)
		close(done)
	}()

	// Wait until the handler signals the stream is live (subscriptions active),
	// so the job we submit next cannot race ahead of them.
	waitFor(t, "stream ready", func() bool { return strings.Contains(rec.snapshot(), ": ready") })

	id, err := runner.Submit("echo", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	waitFor(t, "job event streamed", func() bool {
		s := rec.snapshot()
		return strings.Contains(s, "event: job") && strings.Contains(s, id) &&
			strings.Contains(s, string(jobs.StateSucceeded))
	})

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after context cancel")
	}
}

func TestHandleJobsEventsRejectsNonGet(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/events", nil)
	w := httptest.NewRecorder()
	srv.handleJobsEvents(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// plainWriter implements [http.ResponseWriter] but NOT [http.Flusher].
type plainWriter struct {
	header http.Header
	status int
	buf    bytes.Buffer
}

func (p *plainWriter) Header() http.Header {
	if p.header == nil {
		p.header = make(http.Header)
	}
	return p.header
}
func (p *plainWriter) Write(b []byte) (int, error) { return p.buf.Write(b) }
func (p *plainWriter) WriteHeader(s int)           { p.status = s }

func TestHandleJobsEventsRequiresFlusher(t *testing.T) {
	t.Parallel()

	srv, _ := newJobsTestServer(t, jobs.Config{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/events", nil)
	pw := &plainWriter{}
	srv.handleJobsEvents(pw, r)
	if pw.status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 when streaming unsupported", pw.status)
	}
}
