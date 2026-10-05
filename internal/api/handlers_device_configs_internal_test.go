package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

type testDecrypter struct{}

func (testDecrypter) DecryptValue(ct string) (string, error) {
	return strings.TrimPrefix(ct, "enc:v1:"), nil
}

// hostFetcher answers by host: "good" returns a configuration, anything else
// refuses the login.
type hostFetcher struct{}

func (hostFetcher) Fetch(_ context.Context, req deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
	if req.Host == "10.0.0.1" && req.Password == "s3cret" {
		return deviceconfig.FetchResult{Output: "hostname core\n", HostKey: "SHA256:core"}, nil
	}
	return deviceconfig.FetchResult{}, fmt.Errorf("%w: ssh: unable to authenticate", deviceconfig.ErrAuth)
}

func newDeviceConfigTestServer(t *testing.T) *Server {
	t.Helper()
	s, _ := newJobsTestServer(t, jobs.Config{})
	s.dbConn = newTestDB(t)
	creds, err := app.NewDeviceCredentials(s.db, testEncrypter{})
	if err != nil {
		t.Fatalf("build credentials use-case: %v", err)
	}
	s.deviceCredentials = creds
	backups, err := app.NewDeviceConfigBackups(s.db, testDecrypter{}, hostFetcher{})
	if err != nil {
		t.Fatalf("build device config use-case: %v", err)
	}
	s.deviceConfigs = backups
	s.registerDeviceConfigKind()
	return s
}

func callDeviceConfig(t *testing.T, h http.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(httptest.NewRequest(method, path, bytes.NewBufferString(body)))
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

// TestDeviceConfigRunStoresConfigAndReportsCredentialFailure is the row's
// acceptance through the API: one switch's configuration is retrieved and
// stored, and another switch refusing the credential is reported for that
// device while the run itself succeeds.
func TestDeviceConfigRunStoresConfigAndReportsCredentialFailure(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	good := createDeviceConfigTarget(t, s, "10.0.0.1", createSSHCredential(t, s, "s3cret"))
	bad := createDeviceConfigTarget(t, s, "10.0.0.2", createSSHCredential(t, s, "wrong"))
	if good.Port != deviceconfig.DefaultPort {
		t.Errorf("port defaulted to %d, want 22", good.Port)
	}

	rec := callDeviceConfig(t, s.handleDeviceConfigRun, http.MethodPost, deviceConfigRunPath, `{}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("run: %d %s", rec.Code, rec.Body.String())
	}
	jobID := decodeJob(t, rec.Body).ID
	var job jobs.Job
	waitFor(t, "run finishes", func() bool {
		job, _ = s.jobsRunner().Get(jobID)
		return job.State == jobs.StateSucceeded || job.State == jobs.StateFailed
	})
	if job.State != jobs.StateSucceeded {
		t.Fatalf("run state %s (%s), want succeeded", job.State, job.Err)
	}
	result, ok := job.Result.(deviceconfig.RunResult)
	if !ok || result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("run result = %#v, want 1 succeeded and 1 failed", job.Result)
	}

	statuses := map[string]deviceconfig.DeviceResult{}
	for _, r := range result.Results {
		statuses[r.TargetID] = r
	}
	if statuses[bad.ID].Status != deviceconfig.StatusAuthFailed {
		t.Errorf("bad target status %q, want auth_failed", statuses[bad.ID].Status)
	}

	got := callDeviceConfig(t, s.handleDeviceConfigBackupByID, http.MethodGet,
		deviceConfigsPathPrefix+statuses[good.ID].BackupID, "")
	if got.Code != http.StatusOK {
		t.Fatalf("get backup: %d %s", got.Code, got.Body.String())
	}
	if b := decodeInto[deviceconfig.Backup](
		t,
		got,
	); b.Config != "hostname core\n" ||
		b.Status != deviceconfig.StatusOK {
		t.Errorf("stored backup = %+v", b)
	}

	list := callDeviceConfig(t, s.handleDeviceConfigBackups, http.MethodGet,
		deviceConfigsPath+"?targetId="+bad.ID, "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"auth_failed"`) {
		t.Errorf("history of the refused target: %d %s", list.Code, list.Body.String())
	}

	pinned := callDeviceConfig(t, s.handleDeviceConfigTargetByID, http.MethodGet,
		deviceConfigTargetsPathPrefix+good.ID, "")
	if decodeInto[deviceconfig.Target](t, pinned).HostKeySHA256 != "SHA256:core" {
		t.Errorf("first successful login did not pin the host key: %s", pinned.Body.String())
	}
	cleared := callDeviceConfig(t, s.handleDeviceConfigTargetByID, http.MethodDelete,
		deviceConfigTargetsPathPrefix+good.ID+hostKeySuffix, "")
	if cleared.Code != http.StatusNoContent {
		t.Errorf("clear host key: %d %s", cleared.Code, cleared.Body.String())
	}
}

func TestDeviceConfigTargetsAreScopedToTheCallersClient(t *testing.T) {
	s := newDeviceConfigTestServer(t)
	tgt := createDeviceConfigTarget(t, s, "10.0.0.1", createSSHCredential(t, s, "s3cret"))

	other := httptest.NewRequest(http.MethodGet, deviceConfigTargetsPathPrefix+tgt.ID, nil)
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
