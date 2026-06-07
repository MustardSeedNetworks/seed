// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// captureLogs swaps the global logger to a buffer-backed JSON sink for
// the duration of fn so assertions can read the structured records that
// requireRole emits on a denial. The audit tests don't run in parallel
// because the swap is process-global.
func captureLogs(t *testing.T, fn func()) []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	if err := logging.InitLogger(&logging.LoggingConfig{
		Format: "json",
		Level:  "debug",
		Writer: &buf,
	}); err != nil {
		t.Fatalf("init test logger: %v", err)
	}
	fn()

	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("unparseable log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// TestRequireRole_EmitsForbiddenEvent covers the #1257 structured audit
// log for the 403 path: a viewer denied operator+ access produces a
// record with event=auth.forbidden plus the SIEM-essential fields
// (required_role, actual_role, username, path, method, client_ip).
func TestRequireRole_EmitsForbiddenEvent(t *testing.T) {
	s, _ := usersTestSetup(t)
	seedRoledUser(t, s, "viewer1", database.RoleViewer)

	records := captureLogs(t, func() {
		req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/profiles", []byte(`{}`), "viewer1")
		w := httptest.NewRecorder()
		if got := s.requireRole(w, req, database.RoleOperator); got {
			t.Fatal("requireRole should reject viewer asking for operator+")
		}
	})

	var forbidden map[string]any
	for _, r := range records {
		if r["event"] == "auth.forbidden" {
			forbidden = r
			break
		}
	}
	if forbidden == nil {
		t.Fatalf("expected an event=auth.forbidden record, got %v", records)
	}
	for _, field := range []string{"required_role", "actual_role", "username", "path", "method", "client_ip"} {
		if _, ok := forbidden[field]; !ok {
			t.Errorf("event=auth.forbidden missing field %q: %v", field, forbidden)
		}
	}
	if forbidden["required_role"] != database.RoleOperator {
		t.Errorf("required_role = %v, want operator", forbidden["required_role"])
	}
	if forbidden["actual_role"] != database.RoleViewer {
		t.Errorf("actual_role = %v, want viewer", forbidden["actual_role"])
	}
	if forbidden["username"] != "viewer1" {
		t.Errorf("username = %v, want viewer1", forbidden["username"])
	}
	if forbidden["method"] != http.MethodPost {
		t.Errorf("method = %v, want POST", forbidden["method"])
	}
}

// TestRequireRole_EmitsUnauthorizedEvent covers the 401 (no caller)
// path so SIEM can also alarm on unauthenticated probes of protected
// endpoints — distinct event name so the two failure modes are
// filterable independently.
func TestRequireRole_EmitsUnauthorizedEvent(t *testing.T) {
	s, _ := usersTestSetup(t)

	records := captureLogs(t, func() {
		req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/profiles", []byte(`{}`), "")
		w := httptest.NewRecorder()
		if got := s.requireRole(w, req, database.RoleOperator); got {
			t.Fatal("requireRole should reject empty caller")
		}
	})

	var unauthz map[string]any
	for _, r := range records {
		if r["event"] == "auth.unauthorized" {
			unauthz = r
			break
		}
	}
	if unauthz == nil {
		t.Fatalf("expected an event=auth.unauthorized record, got %v", records)
	}
	if unauthz["required_role"] != database.RoleOperator {
		t.Errorf("required_role = %v, want operator", unauthz["required_role"])
	}
	if unauthz["path"] != APIVersionPrefix+"/profiles" {
		t.Errorf("path = %v, want %s/profiles", unauthz["path"], APIVersionPrefix)
	}
}
