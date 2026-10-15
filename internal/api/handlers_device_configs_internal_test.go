package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// coreConfig is what the device at 10.0.0.1 answers; it carries a secret, as
// a real configuration does.
const coreConfig = "hostname core\nsnmp-server community n0t-public RO\n"

// hostFetcher stands in for SSHFetcher: every device presents
// "SHA256:" + host, an unpinned login stops before authenticating, and only
// 10.0.0.1 with the right password returns a configuration.
type hostFetcher struct{}

func (hostFetcher) Fetch(_ context.Context, req deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
	presented := "SHA256:" + req.Host
	switch {
	case req.PinnedHostKey == "":
		return deviceconfig.FetchResult{}, &deviceconfig.HostKeyUnconfirmedError{Fingerprint: presented}
	case req.PinnedHostKey != presented:
		return deviceconfig.FetchResult{}, fmt.Errorf("%w: presented %s", deviceconfig.ErrHostKeyMismatch, presented)
	case req.Host == "10.0.0.1" && req.Password == "s3cret":
		return deviceconfig.FetchResult{Output: coreConfig}, nil
	}
	return deviceconfig.FetchResult{}, fmt.Errorf("%w: ssh: unable to authenticate", deviceconfig.ErrAuth)
}

// The users newDeviceConfigTestServer seeds, one per role the routes tell
// apart.
const (
	dcOperator = "operator1"
	dcViewer   = "viewer1"
)

// newDeviceConfigTestServer wires the vault and backups over a real keyring,
// so stored ciphertext is real ciphertext, and seeds an operator and a viewer.
func newDeviceConfigTestServer(t *testing.T) *Server {
	t.Helper()
	s, _ := newJobsTestServer(t, jobs.Config{})
	s.dbConn = newTestDB(t)
	s.initIdentityUseCases()
	seedRoledUser(t, s, dcOperator, roles.Operator)
	seedRoledUser(t, s, dcViewer, roles.Viewer)

	cfg := &config.Config{}
	if err := cfg.InitCredentialKeyring(t.TempDir()); err != nil {
		t.Fatalf("init keyring: %v", err)
	}
	keyring, err := cfg.CredentialKeyring()
	if err != nil {
		t.Fatalf("keyring: %v", err)
	}
	creds, err := app.NewDeviceCredentials(s.db, keyring)
	if err != nil {
		t.Fatalf("build credentials use-case: %v", err)
	}
	s.deviceCredentials = creds
	backups, err := app.NewDeviceConfigBackups(s.db, keyring, hostFetcher{})
	if err != nil {
		t.Fatalf("build device config use-case: %v", err)
	}
	s.deviceConfigs = backups
	s.registerDeviceConfigKind()
	return s
}

func callDeviceConfig(t *testing.T, h http.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return callDeviceConfigAs(t, h, dcOperator, method, path, body)
}

func callDeviceConfigAs(
	t *testing.T, h http.HandlerFunc, username, method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(newAuthedRequest(method, path, []byte(body), username))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func decodeInto[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func createSSHCredential(t *testing.T, s *Server, password string) string {
	t.Helper()
	rec := postCredential(t, s, `{"name":"backup","sshUser":"backup","sshPassword":"`+password+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create ssh credential: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), password) {
		t.Fatalf("credential response leaked the password: %s", rec.Body.String())
	}
	return decodeInto[map[string]any](t, rec)["id"].(string)
}

func createDeviceConfigTarget(t *testing.T, s *Server, host, credID string) deviceconfig.Target {
	t.Helper()
	rec := callDeviceConfig(t, s.handleDeviceConfigTargets, http.MethodPost, deviceConfigTargetsPath,
		`{"name":"sw-`+host+`","host":"`+host+`","platform":"cisco-ios","credentialsId":"`+credID+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create target: %d %s", rec.Code, rec.Body.String())
	}
	return decodeInto[deviceconfig.Target](t, rec)
}

// runDeviceConfigBackup starts a run of every target and waits for it.
func runDeviceConfigBackup(t *testing.T, s *Server) deviceconfig.RunResult {
	t.Helper()
	rec := callDeviceConfig(t, s.handleDeviceConfigRun, http.MethodPost, deviceConfigRunPath, `{}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("run: %d %s", rec.Code, rec.Body.String())
	}
	jobID := decodeJob(t, rec.Body).ID
	var job jobs.Job
	waitFor(t, "run finishes", func() bool {
		job, _ = s.jobsRunner().Get(jobID)
		return job.State.Terminal()
	})
	if job.State != jobs.StateSucceeded {
		t.Fatalf("run state %s (%s), want succeeded", job.State, job.Err)
	}
	result, ok := job.Result.(deviceconfig.RunResult)
	if !ok {
		t.Fatalf("run result is %T", job.Result)
	}
	return result
}

func resultsByTarget(res deviceconfig.RunResult) map[string]deviceconfig.DeviceResult {
	out := map[string]deviceconfig.DeviceResult{}
	for _, r := range res.Results {
		out[r.TargetID] = r
	}
	return out
}

// confirmHostKey PUTs the target's pending fingerprint back, as an operator
// confirming the key they were shown.
func confirmHostKey(t *testing.T, s *Server, id string) {
	t.Helper()
	tgt := decodeInto[deviceconfig.Target](t, callDeviceConfig(t, s.handleDeviceConfigTargetByID,
		http.MethodGet, deviceConfigTargetsPathPrefix+id, ""))
	if tgt.PendingHostKeySHA256 == "" {
		t.Fatalf("target %s has no pending host key to confirm", id)
	}
	rec := callDeviceConfig(t, s.handleDeviceConfigTargetByID, http.MethodPut,
		deviceConfigTargetsPathPrefix+id+hostKeySuffix, `{"fingerprint":"`+tgt.PendingHostKeySHA256+`"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("confirm host key of %s: %d %s", id, rec.Code, rec.Body.String())
	}
}

// TestDeviceConfigHostKeyWaitsForAnOperator: a new switch's key is pending and
// nothing is sent to it until an operator confirms that exact key, and
// clearing the pin puts the target back to waiting.
func TestDeviceConfigHostKeyWaitsForAnOperator(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	tgt := createDeviceConfigTarget(t, s, "10.0.0.1", createSSHCredential(t, s, "s3cret"))
	path := deviceConfigTargetsPathPrefix + tgt.ID

	if got := resultsByTarget(runDeviceConfigBackup(t, s))[tgt.ID].Status; got != deviceconfig.StatusHostKeyUnconfirmed {
		t.Errorf("first contact: status %q, want host_key_unconfirmed", got)
	}
	mismatch := callDeviceConfig(t, s.handleDeviceConfigTargetByID, http.MethodPut,
		path+hostKeySuffix, `{"fingerprint":"SHA256:someoneelse"}`)
	if mismatch.Code != http.StatusBadRequest {
		t.Errorf("confirm a key that is not pending: %d %s, want 400", mismatch.Code, mismatch.Body.String())
	}
	confirmHostKey(t, s, tgt.ID)
	pinned := decodeInto[deviceconfig.Target](t, callDeviceConfig(t, s.handleDeviceConfigTargetByID,
		http.MethodGet, path, ""))
	if pinned.HostKeySHA256 != "SHA256:10.0.0.1" || pinned.PendingHostKeySHA256 != "" {
		t.Errorf("after confirming: pinned %q pending %q", pinned.HostKeySHA256, pinned.PendingHostKeySHA256)
	}
	if got := resultsByTarget(runDeviceConfigBackup(t, s))[tgt.ID].Status; got != deviceconfig.StatusOK {
		t.Errorf("after confirming: status %q, want ok", got)
	}

	cleared := callDeviceConfig(t, s.handleDeviceConfigTargetByID, http.MethodDelete, path+hostKeySuffix, "")
	if cleared.Code != http.StatusNoContent {
		t.Errorf("clear host key: %d %s", cleared.Code, cleared.Body.String())
	}
	if got := resultsByTarget(runDeviceConfigBackup(t, s))[tgt.ID].Status; got != deviceconfig.StatusHostKeyUnconfirmed {
		t.Errorf("after clearing the pin: status %q, want host_key_unconfirmed", got)
	}
}

// TestDeviceConfigRunStoresConfigAndReportsCredentialFailure is the row's
// acceptance through the API: one switch's configuration is retrieved and
// stored encrypted, and another refusing the credential is reported for that
// device while the run itself succeeds.
func TestDeviceConfigRunStoresConfigAndReportsCredentialFailure(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	good := createDeviceConfigTarget(t, s, "10.0.0.1", createSSHCredential(t, s, "s3cret"))
	bad := createDeviceConfigTarget(t, s, "10.0.0.2", createSSHCredential(t, s, "wrong"))
	if good.Port != deviceconfig.PlatformCiscoIOS.DefaultPort() {
		t.Errorf("port defaulted to %d, want 22", good.Port)
	}
	runDeviceConfigBackup(t, s)
	confirmHostKey(t, s, good.ID)
	confirmHostKey(t, s, bad.ID)

	result := runDeviceConfigBackup(t, s)
	if result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("run result = %#v, want 1 succeeded and 1 failed", result)
	}
	statuses := resultsByTarget(result)
	if statuses[bad.ID].Status != deviceconfig.StatusAuthFailed {
		t.Errorf("bad target status %q, want auth_failed", statuses[bad.ID].Status)
	}

	got := callDeviceConfig(t, s.handleDeviceConfigBackupByID, http.MethodGet,
		deviceConfigsPathPrefix+statuses[good.ID].BackupID, "")
	if got.Code != http.StatusOK {
		t.Fatalf("get backup: %d %s", got.Code, got.Body.String())
	}
	if b := decodeInto[deviceconfig.Backup](t, got); b.Config != coreConfig || b.Status != deviceconfig.StatusOK {
		t.Errorf("stored backup = %+v", b)
	}
	var raw string
	if err := s.dbConn.QueryRow(t.Context(), `SELECT CAST(config_enc AS TEXT) FROM device_config_backups
		WHERE id = ?`, statuses[good.ID].BackupID).Scan(&raw); err != nil {
		t.Fatalf("read stored configuration: %v", err)
	}
	if !strings.HasPrefix(raw, "enc:v1:") || strings.Contains(raw, "n0t-public") || strings.Contains(raw, "hostname") {
		t.Errorf("configuration is not stored as ciphertext: %q", raw)
	}

	again := resultsByTarget(runDeviceConfigBackup(t, s))
	if again[good.ID].BackupID != statuses[good.ID].BackupID {
		t.Errorf("an unchanged configuration was stored again: %s, then %s",
			statuses[good.ID].BackupID, again[good.ID].BackupID)
	}

	list := callDeviceConfig(t, s.handleDeviceConfigBackups, http.MethodGet,
		deviceConfigsPath+"?targetId="+bad.ID, "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"auth_failed"`) {
		t.Errorf("history of the refused target: %d %s", list.Code, list.Body.String())
	}
}

// TestDeviceConfigBackupReadsNeedOperator: a configuration carries the
// device's secrets, so a viewer can read neither a backup nor the history.
func TestDeviceConfigBackupReadsNeedOperator(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	tgt := createDeviceConfigTarget(t, s, "10.0.0.1", createSSHCredential(t, s, "s3cret"))
	runDeviceConfigBackup(t, s)
	confirmHostKey(t, s, tgt.ID)
	backupID := resultsByTarget(runDeviceConfigBackup(t, s))[tgt.ID].BackupID

	for _, tc := range []struct {
		name string
		h    http.HandlerFunc
		path string
	}{
		{"history", s.handleDeviceConfigBackups, deviceConfigsPath + "?targetId=" + tgt.ID},
		{"one backup", s.handleDeviceConfigBackupByID, deviceConfigsPathPrefix + backupID},
	} {
		for _, c := range []struct {
			user string
			want int
		}{
			{dcViewer, http.StatusForbidden},
			{dcOperator, http.StatusOK},
		} {
			rec := callDeviceConfigAs(t, tc.h, c.user, http.MethodGet, tc.path, "")
			if rec.Code != c.want {
				t.Errorf("%s as %s: status %d, want %d (%s)", tc.name, c.user, rec.Code, c.want, rec.Body.String())
			}
			if c.want == http.StatusForbidden && strings.Contains(rec.Body.String(), "n0t-public") {
				t.Errorf("%s as %s: a refused read leaked the configuration", tc.name, c.user)
			}
		}
	}
}

// TestViewerCannotChangeDeviceConfigs drives every mutating device-config
// route through the real chain as a viewer holding a valid CSRF token, so the
// refusal is the role gate's and not the CSRF wall's.
func TestViewerCannotChangeDeviceConfigs(t *testing.T) {
	t.Parallel()
	s := routedServer(t, dcViewer, roles.Viewer)
	// Pro, so the feature gate in front of the role gate lets the request by.
	mgr, err := license.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("license manager: %v", err)
	}
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}
	s.licenseMgr = mgr
	token, err := s.authManager().GenerateAccessToken(t.Context(), dcViewer)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	csrfToken := fetchCSRFToken(t, s, token)

	const target = deviceConfigTargetsPathPrefix + "dct-0123456789ab"
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, deviceConfigTargetsPath},
		{http.MethodPut, target},
		{http.MethodDelete, target},
		{http.MethodPut, target + hostKeySuffix},
		{http.MethodDelete, target + hostKeySuffix},
		{http.MethodPost, deviceConfigRunPath},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Csrf-Token", csrfToken)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "role") {
			t.Errorf("%s %s: status %d %s, want 403 from the role gate",
				tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestDeviceConfigTargetsAreScopedToTheCallersClient(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	tgt := createDeviceConfigTarget(t, s, "10.0.0.1", createSSHCredential(t, s, "s3cret"))

	other := newAuthedRequest(http.MethodGet, deviceConfigTargetsPathPrefix+tgt.ID, nil, dcOperator)
	other = other.WithContext(auth.WithClientID(other.Context(), "tenant-other"))
	rec := httptest.NewRecorder()
	s.handleDeviceConfigTargetByID(rec, other)
	if rec.Code != http.StatusNotFound {
		t.Errorf("another client's target: status %d, want 404", rec.Code)
	}
}

func TestDeviceConfigTargetValidationIsA400(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	snmp := decodeInto[map[string]any](t, postCredential(t, s, `{"name":"ro","community":"public"}`))["id"].(string)

	for name, body := range map[string]string{
		"snmp credential": `{"name":"sw","host":"10.0.0.1","platform":"cisco-ios","credentialsId":"` + snmp + `"}`,
		"bad platform":    `{"name":"sw","host":"10.0.0.1","platform":"hp","credentialsId":"` + snmp + `"}`,
		"unknown field":   `{"name":"sw","host":"10.0.0.1","password":"x"}`,
	} {
		rec := callDeviceConfig(t, s.handleDeviceConfigTargets, http.MethodPost, deviceConfigTargetsPath, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400 (%s)", name, rec.Code, rec.Body.String())
		}
	}
}

// TestJobsRouteRefusesDeviceConfigRuns — POST /jobs cannot say whose devices
// to back up, so it must not start a run.
func TestJobsRouteRefusesDeviceConfigRuns(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/jobs",
		strings.NewReader(`{"kind":"`+deviceConfigJobKind+`","params":{"ClientID":"tenant-other"}}`))
	rec := httptest.NewRecorder()
	s.handleJobs(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), deviceConfigRunPath) {
		t.Errorf("POST /jobs device-config-backup: %d %s, want 400 naming the run route", rec.Code, rec.Body.String())
	}
}

// TestBackupJobsAreTheirClientsOnly: the shared job routes answer for a
// configuration backup run only to the client that started it; a job no
// client owns stays visible to everyone.
func TestBackupJobsAreTheirClientsOnly(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	createDeviceConfigTarget(t, s, "10.0.0.1", createSSHCredential(t, s, "s3cret"))
	rec := callDeviceConfig(t, s.handleDeviceConfigRun, http.MethodPost, deviceConfigRunPath, `{}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("run: %d %s", rec.Code, rec.Body.String())
	}
	owned := decodeJob(t, rec.Body).ID
	if err := s.jobsRunner().Register("shared-probe", okKind("ok")); err != nil {
		t.Fatalf("register kind: %v", err)
	}
	shared, err := s.jobsRunner().Submit("shared-probe", nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	for _, tc := range []struct {
		name, method, id, client string
		want                     int
	}{
		{"own client reads", http.MethodGet, owned, testClientID, http.StatusOK},
		{"other client reads", http.MethodGet, owned, "tenant-other", http.StatusNotFound},
		{"no client reads", http.MethodGet, owned, "", http.StatusNotFound},
		{"other client cancels", http.MethodDelete, owned, "tenant-other", http.StatusNotFound},
		{"other client reads an unowned job", http.MethodGet, shared, "tenant-other", http.StatusOK},
	} {
		req := newAuthedRequest(tc.method, jobsPath+"/"+tc.id, nil, dcOperator)
		if tc.client != "" {
			req = req.WithContext(auth.WithClientID(req.Context(), tc.client))
		}
		w := httptest.NewRecorder()
		routed(jobPath, s.handleJobByID)(w, req)
		if w.Code != tc.want {
			t.Errorf("%s: status %d, want %d", tc.name, w.Code, tc.want)
		}
	}

	ch := make(chan []byte, 8)
	unsubscribe := s.subscribeJobEvents(ch, "tenant-other")
	defer unsubscribe()
	s.eventBus().
		Publish(jobs.JobEvent{Job: jobs.Job{ID: owned, Kind: deviceConfigJobKind, ClientID: testClientID, State: jobs.StateRunning}})
	s.eventBus().Publish(jobs.JobEvent{Job: jobs.Job{ID: shared, Kind: "shared-probe", State: jobs.StateRunning}})
	select {
	case frame := <-ch:
		if strings.Contains(string(frame), owned) {
			t.Errorf("another client's job reached the stream: %s", frame)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the unowned job never reached the stream")
	}
}
