package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

const (
	wmiTargetPattern      = wmiTargetsPath + "/{targetId}"
	wmiCertificatePattern = wmiTargetPattern + "/certificate"
	wmiInventoryPattern   = wmiTargetPattern + "/inventory"
)

// hostCollector stands in for WSManCollector: every host presents the
// fingerprint of its own address, an unpinned host is refused before any
// credential is sent, and only the right password returns an inventory.
type hostCollector struct{}

func (hostCollector) Collect(_ context.Context, e wmi.Endpoint) (wmi.Inventory, error) {
	presented := wmi.Fingerprint([]byte(e.Host))
	switch {
	case e.PinnedCert == "":
		return wmi.Inventory{}, &wmi.CertUnconfirmedError{Fingerprint: presented}
	case e.PinnedCert != presented:
		return wmi.Inventory{}, fmt.Errorf("%w: host presented %s", wmi.ErrCertMismatch, presented)
	case e.Password != "s3cret":
		return wmi.Inventory{}, fmt.Errorf("%w: host refused the credential", wmi.ErrAuth)
	}
	return wmi.Inventory{
		OS:       wmi.OperatingSystem{Caption: "Microsoft Windows Server 2025 Standard", BuildNumber: "26100"},
		System:   wmi.ComputerSystem{Manufacturer: "Dell Inc.", Model: "PowerEdge R660", MemoryBytes: 68719476736},
		Software: []wmi.Program{{Name: "Microsoft Edge", Version: "141.0.3537.71", Vendor: "Microsoft Corporation"}},
	}, nil
}

func newWMITestServer(t *testing.T) *Server {
	t.Helper()
	s, _ := newJobsTestServer(t, jobs.Config{})
	db := newTestDB(t)
	s.persist = app.NewPersistence(db)
	s.initIdentityUseCases()
	seedRoledUser(t, db, dcOperator, roles.Operator)

	cfg := &config.Config{}
	require.NoError(t, cfg.InitCredentialKeyring(t.TempDir()))
	keyring, err := cfg.CredentialKeyring()
	require.NoError(t, err)
	creds, err := app.NewDeviceCredentials(s.persistence, keyring)
	require.NoError(t, err)
	s.deviceCredentials = creds
	inventory, err := app.NewWMIInventory(s.persistence, keyring, hostCollector{})
	require.NoError(t, err)
	s.wmiInventory = inventory
	s.registerWMIKind()
	return s
}

func callWMI(t *testing.T, pattern string, h http.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(newAuthedRequest(method, path, []byte(body), dcOperator))
	rec := httptest.NewRecorder()
	routed(pattern, h)(rec, req)
	return rec
}

func createWinRMCredential(t *testing.T, s *Server, password string) string {
	t.Helper()
	rec := postCredential(t, s, `{"name":"inventory","winrmUser":"LAB\\inventory","winrmPassword":"`+password+`"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), password, "the credential response leaked the password")
	cred := decodeInto[map[string]any](t, rec)
	require.Equal(t, "winrm", cred["kind"])
	require.Equal(t, `LAB\inventory`, cred["winrmUser"])
	return cred["id"].(string)
}

func createWMITarget(t *testing.T, s *Server, host, credID string) WMITargetResponse {
	t.Helper()
	rec := callWMI(t, wmiTargetsPath, s.handleWMITargets, http.MethodPost, wmiTargetsPath,
		`{"name":"win-`+host+`","host":"`+host+`","credentialsId":"`+credID+`"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	tgt := decodeInto[WMITargetResponse](t, rec)
	require.Equal(t, wmiTargetsPath+"/"+tgt.ID, rec.Header().Get("Location"))
	return tgt
}

// runWMI starts a run of every target and waits for it.
func runWMI(t *testing.T, s *Server) map[string]wmi.HostResult {
	t.Helper()
	rec := callWMI(t, wmiRunsPath, s.handleWMIRun, http.MethodPost, wmiRunsPath, `{}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	jobID := decodeJob(t, rec.Body).ID
	var job jobs.Job
	waitFor(t, "run finishes", func() bool {
		job, _ = s.jobsRunner().Get(jobID)
		return job.State.Terminal()
	})
	require.Equal(t, jobs.StateSucceeded, job.State, job.Err)
	res, ok := job.Result.(wmi.RunResult)
	require.True(t, ok, "run result is %T", job.Result)
	out := map[string]wmi.HostResult{}
	for _, r := range res.Results {
		out[r.TargetID] = r
	}
	return out
}

// TestWMIInventoryThroughTheAPI is the row's acceptance at the API: a Windows
// host's certificate waits for an operator, and once it is confirmed the host
// reports its OS, hardware and installed software.
func TestWMIInventoryThroughTheAPI(t *testing.T) {
	s := newWMITestServer(t)
	tgt := createWMITarget(t, s, "10.0.0.5", createWinRMCredential(t, s, "s3cret"))
	require.Equal(t, wmi.DefaultPort, tgt.Port)
	path := wmiTargetsPath + "/" + tgt.ID

	require.Equal(t, wmi.StatusCertUnconfirmed, runWMI(t, s)[tgt.ID].Status)
	got := decodeInto[WMITargetResponse](t, callWMI(t, wmiTargetPattern, s.handleWMITargetByID,
		http.MethodGet, path, ""))
	require.Equal(t, wmi.Fingerprint([]byte("10.0.0.5")), got.PendingCertSHA256)
	require.Equal(t, string(wmi.StatusCertUnconfirmed), got.LastStatus)
	require.NotEmpty(t, got.LastAttemptAt)

	rec := callWMI(t, wmiInventoryPattern, s.handleWMIInventory, http.MethodGet, path+"/inventory", "")
	require.Equal(t, http.StatusNotFound, rec.Code, "no collection has succeeded yet")

	rec = callWMI(t, wmiCertificatePattern, s.handleWMICertificate, http.MethodPut, path+"/certificate",
		`{"fingerprint":"`+strings.Repeat("AB", 32)+`"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, "only the pending certificate can be confirmed")
	rec = callWMI(t, wmiCertificatePattern, s.handleWMICertificate, http.MethodPut, path+"/certificate",
		`{"fingerprint":"`+got.PendingCertSHA256+`"}`)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

	require.Equal(t, wmi.StatusOK, runWMI(t, s)[tgt.ID].Status)
	rec = callWMI(t, wmiInventoryPattern, s.handleWMIInventory, http.MethodGet, path+"/inventory", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	inv := decodeInto[WMIInventoryResponse](t, rec)
	require.Equal(t, tgt.ID, inv.TargetID)
	require.NotEmpty(t, inv.CollectedAt)
	require.Equal(t, "Microsoft Windows Server 2025 Standard", inv.Inventory.OS.Caption)
	require.Equal(t, "PowerEdge R660", inv.Inventory.System.Model)
	require.Equal(t, "Microsoft Edge", inv.Inventory.Software[0].Name)

	rec = callWMI(t, wmiCertificatePattern, s.handleWMICertificate, http.MethodDelete, path+"/certificate", "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, wmi.StatusCertUnconfirmed, runWMI(t, s)[tgt.ID].Status,
		"a cleared pin waits for an operator again")
	rec = callWMI(t, wmiInventoryPattern, s.handleWMIInventory, http.MethodGet, path+"/inventory", "")
	require.Equal(t, http.StatusOK, rec.Code, "a failed collection keeps the last inventory")

	rec = callWMI(t, wmiTargetPattern, s.handleWMITargetByID, http.MethodDelete, path, "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	rec = callWMI(t, wmiInventoryPattern, s.handleWMIInventory, http.MethodGet, path+"/inventory", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWMIRunReportsARefusedCredential(t *testing.T) {
	s := newWMITestServer(t)
	tgt := createWMITarget(t, s, "10.0.0.6", createWinRMCredential(t, s, "wrong"))
	runWMI(t, s)
	pending := decodeInto[WMITargetResponse](t, callWMI(t, wmiTargetPattern, s.handleWMITargetByID,
		http.MethodGet, wmiTargetsPath+"/"+tgt.ID, "")).PendingCertSHA256
	rec := callWMI(t, wmiCertificatePattern, s.handleWMICertificate, http.MethodPut,
		wmiTargetsPath+"/"+tgt.ID+"/certificate", `{"fingerprint":"`+pending+`"}`)
	require.Equal(t, http.StatusNoContent, rec.Code)

	res := runWMI(t, s)[tgt.ID]
	require.Equal(t, wmi.StatusAuthFailed, res.Status)
	require.Contains(t, res.Error, "refused the credential")
}

func TestWMITargetsRefuseBadInput(t *testing.T) {
	s := newWMITestServer(t)
	winrm := createWinRMCredential(t, s, "s3cret")
	ssh := createSSHCredential(t, s, "pw")
	createWMITarget(t, s, "10.0.0.5", winrm)

	for name, body := range map[string]string{
		"ssh credential": `{"name":"x","host":"10.0.0.7","credentialsId":"` + ssh + `"}`,
		"same address":   `{"name":"x","host":"10.0.0.5","credentialsId":"` + winrm + `"}`,
		"no host":        `{"name":"x","credentialsId":"` + winrm + `"}`,
		"unknown field":  `{"name":"x","host":"10.0.0.8","credentialsId":"` + winrm + `","platform":"x"}`,
	} {
		rec := callWMI(t, wmiTargetsPath, s.handleWMITargets, http.MethodPost, wmiTargetsPath, body)
		require.Equal(t, http.StatusBadRequest, rec.Code, "%s: %s", name, rec.Body.String())
	}

	rec := callWMI(t, wmiTargetPattern, s.handleWMITargetByID, http.MethodGet, wmiTargetsPath+"/not-an-id", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	rec = callWMI(t, wmiRunsPath, s.handleWMIRun, http.MethodPost, wmiRunsPath, `{"targetIds":["../x"]}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWMITargetsAreTheirClientsOnly(t *testing.T) {
	s := newWMITestServer(t)
	tgt := createWMITarget(t, s, "10.0.0.5", createWinRMCredential(t, s, "s3cret"))

	req := newAuthedRequest(http.MethodGet, wmiTargetsPath+"/"+tgt.ID, nil, dcOperator)
	req = req.WithContext(auth.WithClientID(req.Context(), "tenant-other"))
	rec := httptest.NewRecorder()
	routed(wmiTargetPattern, s.handleWMITargetByID)(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code, "another client's target must not resolve")
}

// TestJobsRouteRefusesWMIRuns — POST /jobs cannot say whose hosts to
// inventory, so it must not start a run.
func TestJobsRouteRefusesWMIRuns(t *testing.T) {
	s := newWMITestServer(t)
	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/jobs",
		strings.NewReader(`{"kind":"`+wmiJobKind+`","params":{"ClientID":"tenant-other"}}`))
	rec := httptest.NewRecorder()
	s.handleJobs(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), wmiRunsPath)
}

func TestSingleLogLineStripsNewlines(t *testing.T) {
	got := singleLogLine("alice\nFAKE level=ERROR\rroot")
	require.NotContains(t, got, "\n")
	require.NotContains(t, got, "\r")
	require.Equal(t, "alice FAKE level=ERROR root", got)
}

func TestWMIRoutesAnswer503WithoutAKeyring(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	s.handleWMITargets(rec, withClaim(httptest.NewRequest(http.MethodGet, wmiTargetsPath, nil)))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
