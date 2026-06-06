package api

// handlers_jobs.go is the HTTP surface of the unified async job runner
// (ADR-0005, RE_ARCHITECTURE_BLUEPRINT.md §8): create / inspect / cancel a job
// and a single SSE stream of job state changes. It is a thin adapter over
// internal/platform/jobs — decode, delegate, map errors, encode — holding no
// business logic. Job kinds are registered on the runner separately; until the
// real long-ops are migrated into kinds, an unknown kind is a 400.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/platform/events"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

const (
	// jobsPathPrefix is the /jobs/{id} item-route prefix.
	jobsPathPrefix = APIVersionPrefix + "/jobs/"

	// jobSSEBuffer bounds a connection's pending event backlog. A consumer that
	// falls this far behind drops events (it can re-sync with GET /jobs/{id});
	// the stream favours liveness over guaranteed delivery, like the card hub.
	jobSSEBuffer = 64

	// jobIdempotencyCapacity bounds the in-memory Idempotency-Key cache. The
	// oldest key is evicted past this; idempotency is best-effort within that
	// window, which covers the realistic client-retry case.
	jobIdempotencyCapacity = 256

	// jobsRetention is how long terminal jobs are kept for inspection before a
	// retention sweep may remove them.
	jobsRetention = time.Hour

	// jobsShutdownTimeout bounds the graceful drain of the runner and event bus.
	jobsShutdownTimeout = 5 * time.Second
)

// CreateJobRequest is the POST /jobs body: a job kind plus opaque,
// kind-specific parameters passed through verbatim to the kind's handler.
type CreateJobRequest struct {
	Kind   string          `json:"kind"`
	Params json.RawMessage `json:"params,omitempty"`
}

// JobResponse is the transport view of a [jobs.Job].
type JobResponse struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"`
	State    string  `json:"state"`
	Progress float64 `json:"progress"`
	Result   any     `json:"result,omitempty"`
	Err      string  `json:"error,omitempty"`
}

// toJobResponse maps a domain job snapshot to its transport view.
func toJobResponse(j jobs.Job) JobResponse {
	return JobResponse{
		ID:       j.ID,
		Kind:     j.Kind,
		State:    string(j.State),
		Progress: j.Progress,
		Result:   j.Result,
		Err:      j.Err,
	}
}

// handleJobs is the /jobs collection handler: POST creates a job. The mutating
// method is operator-gated by writeGated at registration.
func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, "Only POST is supported on /jobs", "")
		return
	}

	var req CreateJobRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}
	if req.Kind == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "Job kind is required", "")
		return
	}

	// Idempotency: a repeated Idempotency-Key replays the original job rather
	// than creating a duplicate; the same key with a different body conflicts.
	key := r.Header.Get("Idempotency-Key")
	if key != "" {
		switch res := s.jobIdempotency().check(r.Context(), key, req); res.kind {
		case idemConflict:
			sendErrorResponseWithDetails(w, logger, http.StatusConflict, ErrCodeConflict,
				"Idempotency-Key already used with different parameters", "")
			return
		case idemHit:
			if j, ok := s.jobsRunner().Get(res.id); ok {
				sendJSONResponse(w, logger, http.StatusOK, toJobResponse(j))
				return
			}
			// Original job was retained-out; fall through and create afresh.
		case idemMiss:
		}
	}

	id, err := s.jobsRunner().Submit(req.Kind, req.Params)
	if err != nil {
		writeJobError(w, logger, err)
		return
	}
	if key != "" {
		s.jobIdempotency().store(r.Context(), key, req, id)
	}

	j, _ := s.jobsRunner().Get(id)
	w.Header().Set("Location", jobsPathPrefix+id)
	sendJSONResponse(w, logger, http.StatusCreated, toJobResponse(j))
}

// handleJobByID is the /jobs/{id} item handler: GET inspects, DELETE cancels.
// DELETE is operator-gated by writeGated; GET is a safe read.
func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	id := strings.TrimPrefix(r.URL.Path, jobsPathPrefix)
	if id == "" || strings.Contains(id, "/") {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "Missing or invalid job id", "")
		return
	}

	switch r.Method {
	case http.MethodGet:
		j, ok := s.jobsRunner().Get(id)
		if !ok {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, "Job not found", "")
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, toJobResponse(j))
	case http.MethodDelete:
		if err := s.jobsRunner().Cancel(id); err != nil {
			writeJobError(w, logger, err)
			return
		}
		// Cancellation is asynchronous; return the current snapshot with 202.
		j, _ := s.jobsRunner().Get(id)
		sendJSONResponse(w, logger, http.StatusAccepted, toJobResponse(j))
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, "Only GET and DELETE are supported on /jobs/{id}", "")
	}
}

// handleJobsEvents streams every job state change to the client as SSE, bridged
// from the events bus. One stream covers all jobs (clients filter by id),
// replacing the per-operation status-poll endpoints.
func (s *Server) handleJobsEvents(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, "Only GET is supported on /jobs/events", "")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Streaming unsupported", "")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	// Per-connection backlog. The bus delivers on its own goroutine; the push
	// handler must never block it, so a full buffer drops (re-syncable via GET).
	ch := make(chan []byte, jobSSEBuffer)
	unsubscribe := s.subscribeJobEvents(ch)
	defer unsubscribe()

	// Signal the stream is live: subscriptions are now active, so no state
	// change from this point is missed.
	if _, err := w.Write([]byte(": ready\n\n")); err != nil {
		return
	}
	flusher.Flush()

	s.pumpJobSSE(r.Context(), w, flusher, ch, logger)
}

// subscribeJobEvents fans every job state change into ch and returns a single
// unsubscribe func. The push never blocks the bus: a full backlog is dropped
// (the client re-syncs with GET /jobs/{id}).
func (s *Server) subscribeJobEvents(ch chan<- []byte) func() {
	push := func(_ context.Context, ev events.Event) {
		je, isJob := ev.(jobs.JobEvent)
		if !isJob {
			return
		}
		data, err := json.Marshal(toJobResponse(je.Job))
		if err != nil {
			return
		}
		select {
		case ch <- data:
		default: // consumer too slow; drop this event
		}
	}

	bus := s.eventBus()
	cancels := make([]func(), 0, len(jobs.States()))
	for _, st := range jobs.States() {
		cancels = append(cancels, bus.Subscribe(jobs.Topic(st), push))
	}
	return func() {
		for _, cancel := range cancels {
			cancel()
		}
	}
}

// pumpJobSSE writes queued job frames and periodic heartbeats until the client
// disconnects (ctx cancelled) or a write fails.
func (s *Server) pumpJobSSE(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	ch <-chan []byte,
	logger *slog.Logger,
) {
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-ch:
			if err := writeJobSSE(w, data); err != nil {
				logger.DebugContext(ctx, "jobs SSE write error", "error", err)
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// writeJobSSE writes one `event: job` SSE frame. It uses w.Write (not a
// value-interpolating Fprintf) so the data — already JSON-marshalled bytes —
// is emitted verbatim and the output-escaping gate stays satisfied.
func writeJobSSE(w http.ResponseWriter, data []byte) error {
	var b bytes.Buffer
	b.WriteString("event: job\ndata: ")
	b.Write(data)
	b.WriteString("\n\n")
	_, err := w.Write(b.Bytes())
	return err
}

// writeJobError maps a runner sentinel error to its HTTP status + ErrCode.
func writeJobError(w http.ResponseWriter, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, jobs.ErrUnknownKind):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "Unknown job kind", "")
	case errors.Is(err, jobs.ErrAtCapacity):
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "Job runner at capacity, retry later", "")
	case errors.Is(err, jobs.ErrNotFound):
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, "Job not found", "")
	case errors.Is(err, jobs.ErrClosed):
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "Job runner unavailable", "")
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to process job", "")
	}
}

// jobsRoutes returns the unified job-runner routes (ADR-0005). POST (create)
// and DELETE (cancel) are mutating -> operator-gated via writeGated; GET
// (status) and the SSE stream are safe reads. The exact /jobs/events pattern
// out-ranks the /jobs/ subtree in net/http's ServeMux (longest match wins).
func (s *Server) jobsRoutes() []route {
	op := database.RoleOperator
	return []route{
		{
			path:        APIVersionPrefix + "/jobs",
			handler:     s.handleJobs,
			methods:     []string{http.MethodPost},
			minRole:     op,
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/jobs/events",
			handler: s.handleJobsEvents,
			methods: []string{http.MethodGet},
		},
		{
			path:    APIVersionPrefix + "/jobs/",
			handler: s.handleJobByID,
			methods: []string{http.MethodGet, http.MethodDelete},
			minRole: op,
		},
	}
}

// --- idempotency cache ----------------------------------------------------

// idemKind is the outcome of an idempotency-key lookup.
type idemKind int

const (
	idemMiss     idemKind = iota // unseen key — caller should create
	idemHit                      // same key + same request — replay the job
	idemConflict                 // same key, different request — reject
)

// idemResult is a lookup outcome plus the prior job id when it is a hit.
type idemResult struct {
	id   string
	kind idemKind
}

// idemEntry binds an Idempotency-Key to the job it created and a hash of the
// request, so reuse with a different body can be detected as a conflict.
type idemEntry struct {
	jobID string
	hash  string
}

// jobIdempotencyCache is a bounded, in-memory Idempotency-Key store. It is
// best-effort: keys evict in FIFO order past the capacity, and two genuinely
// simultaneous identical submits may both create (the realistic case is a
// sequential client retry, which is deduplicated). Durable idempotency arrives
// with persistence in Phase 5.
type jobIdempotencyCache struct {
	mu    sync.Mutex
	byKey map[string]idemEntry
	order []string
	cap   int
}

func newJobIdempotencyCache(capacity int) *jobIdempotencyCache {
	return &jobIdempotencyCache{byKey: make(map[string]idemEntry), cap: capacity}
}

// requestHash fingerprints the kind + params so a reused key with a changed
// body is detectable.
func requestHash(req CreateJobRequest) string {
	h := sha256.New()
	h.Write([]byte(req.Kind))
	h.Write([]byte{0})
	h.Write(req.Params)
	return hex.EncodeToString(h.Sum(nil))
}

// jobIdempotencyStore dedups POST /jobs by Idempotency-Key. The in-memory
// jobIdempotencyCache (no database) and the durable dbJobIdempotency both
// satisfy it; ctx is honored by the durable implementation and ignored by the
// cache. Both are best-effort: a backend error degrades to idemMiss (create)
// rather than failing the request.
type jobIdempotencyStore interface {
	check(ctx context.Context, key string, req CreateJobRequest) idemResult
	store(ctx context.Context, key string, req CreateJobRequest, jobID string)
}

// check classifies a key against a request without recording anything.
func (c *jobIdempotencyCache) check(
	_ context.Context,
	key string,
	req CreateJobRequest,
) idemResult {
	h := requestHash(req)
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.byKey[key]
	if !ok {
		return idemResult{kind: idemMiss}
	}
	if e.hash != h {
		return idemResult{kind: idemConflict}
	}
	return idemResult{id: e.jobID, kind: idemHit}
}

// store records the job a key created, evicting the oldest key past capacity.
func (c *jobIdempotencyCache) store(
	_ context.Context,
	key string,
	req CreateJobRequest,
	jobID string,
) {
	h := requestHash(req)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.byKey[key]; !exists {
		c.order = append(c.order, key)
		if len(c.order) > c.cap {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.byKey, oldest)
		}
	}
	c.byKey[key] = idemEntry{jobID: jobID, hash: h}
}
