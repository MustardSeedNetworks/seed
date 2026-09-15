// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// sseServer builds a routed Server with one operator to authenticate as.
func sseServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "sse.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, createErr := db.CreateUser(t.Context(), "operator", "$2a$10$x",
		database.RoleOperator); createErr != nil {
		t.Fatalf("seed operator: %v", createErr)
	}

	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	netMgr := netif.NewMockManager(netif.DefaultMockConfig())
	s := NewServer(cfg, filepath.Join(dir, "seed.json"), "", netMgr, false, nil, db, nil)
	t.Cleanup(s.Close)
	return s
}

// TestJobsEventsStreams pins #2553 (reopened). The SSE handlers ask
// `w.([http.Flusher])`, and the logging middleware's responseWriter wrapped every
// request without implementing it, so the assertion failed and each stream
// answered 500 "Streaming unsupported" instead of opening.
//
// The test runs the whole chain through Handler() and over a real listener,
// because the defect lives in the chain rather than in the handler: a
// [httptest.ResponseRecorder] IS an [http.Flusher], so a handler-level test that
// calls handleJobsEvents directly passes with the defect fully present.
func TestJobsEventsStreams(t *testing.T) {
	s := sseServer(t)

	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)

	token, err := s.authManager().GenerateAccessToken(t.Context(), "operator")
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/jobs/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/jobs/events: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d — the stream refused to open", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	// The ready comment is written and flushed before the pump blocks, so
	// reading it proves the flush reached the wire rather than merely that the
	// assertion passed. A Flush that exists but forwards nothing fails this
	// test too: net/http holds the response in its own write buffer until the
	// handler returns, which for a stream is never, so the request above times
	// out before a single header arrives.
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil {
		t.Fatalf("read the first frame: %v", err)
	}
	if got := strings.TrimRight(line, "\r\n"); got != ": ready" {
		t.Errorf("first frame = %q, want %q", got, ": ready")
	}
}
