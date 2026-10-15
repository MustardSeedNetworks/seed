package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

func newSetupPendingServer(t *testing.T) *api.Server {
	t.Helper()
	cfg := testutil.NewConfigBuilder().
		WithAuth(testutil.GetTestDefaults().Auth.Username, auth.SetupModePlaceholder).
		Build()
	server := api.NewTestServerWithConfig(cfg)
	server.SetConfigPath(filepath.Join(t.TempDir(), "config.yaml"))
	t.Cleanup(server.Close)
	return server
}

func postSetupComplete(t *testing.T, server *api.Server, password, token string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(api.SetupCompleteRequest{Password: password, SetupToken: token})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.HandleSetupComplete(w, req)
	return w
}

// Setup status answers any network peer, so it must name the token's file
// and never hand out the token itself: whoever holds the token sets the admin
// password.
func TestSetupStatusNeverReturnsTheToken(t *testing.T) {
	server := newSetupPendingServer(t)
	token := api.IssueSetupToken(t, server)

	w := httptest.NewRecorder()
	server.HandleSetupStatus(w, httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", http.NoBody))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), token) {
		t.Fatalf("setup status returned the setup token: %s", w.Body.String())
	}
	var resp api.SetupStatusResponse
	dec := json.NewDecoder(w.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.NeedsSetup || resp.SetupTokenFile == "" {
		t.Errorf("response = %+v, want needsSetup and the token file", resp)
	}
}

func TestSetupCompleteRequiresTheFileToken(t *testing.T) {
	const password = "MySecure123!Pass"
	tests := []struct {
		name  string
		issue bool
		token func(issued string) string
		want  int
	}{
		{"no token issued", false, func(string) string { return "anything" }, http.StatusForbidden},
		{"wrong token", true, func(string) string { return "not-the-token" }, http.StatusForbidden},
		{"token from the file", true, func(issued string) string { return issued }, http.StatusOK},
		{"token pasted with its newline", true, func(issued string) string { return issued + "\n" }, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newSetupPendingServer(t)
			issued := ""
			if tt.issue {
				issued = api.IssueSetupToken(t, server)
			}
			if w := postSetupComplete(t, server, password, tt.token(issued)); w.Code != tt.want {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

// A password the policy rejects must not cost the operator the token, and a
// completed setup must remove the token's file.
func TestSetupTokenOutlivesAPolicyRejection(t *testing.T) {
	server := newSetupPendingServer(t)
	token := api.IssueSetupToken(t, server)

	if w := postSetupComplete(t, server, "password", token); w.Code != http.StatusBadRequest {
		t.Fatalf("weak password: status = %d, want 400 (body %s)", w.Code, w.Body.String())
	}
	if w := postSetupComplete(t, server, "MySecure123!Pass", token); w.Code != http.StatusOK {
		t.Fatalf("retry with the same token: status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}

	w := httptest.NewRecorder()
	server.HandleSetupStatus(w, httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", http.NoBody))
	var resp api.SetupStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.NeedsSetup {
		t.Fatal("setup still pending after a successful completion")
	}
}

func TestSetupTokenFileIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no POSIX mode bits; the data directory's ACL protects the file")
	}
	dir := t.TempDir()
	manager := api.NewSetupTokenManager(dir)
	// A leftover file with a looser mode must not keep it.
	if err := os.WriteFile(manager.FilePath(), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := manager.Issue(); err != nil {
		t.Fatalf("issue: %v", err)
	}
	info, err := os.Stat(manager.FilePath())
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("token file mode = %o, want 600", mode)
	}
	raw, err := os.ReadFile(manager.FilePath())
	if err != nil {
		t.Fatal(err)
	}
	if !manager.Validate(strings.TrimSpace(string(raw))) {
		t.Error("the token in the file does not validate")
	}

	if consumeErr := manager.Consume(); consumeErr != nil {
		t.Fatalf("consume: %v", consumeErr)
	}
	if _, statErr := os.Stat(manager.FilePath()); !os.IsNotExist(statErr) {
		t.Errorf("token file still present after Consume: %v", statErr)
	}
	if manager.Validate(strings.TrimSpace(string(raw))) {
		t.Error("a consumed token still validates")
	}
}
