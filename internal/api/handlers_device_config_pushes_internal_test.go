package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

const dcAdmin = "admin1"

// pushDiff is what the stand-in device says the candidate would change; it
// carries a secret, as a real diff can.
const pushDiff = "--- running\n+++ candidate\n-<community>n0t-public</community>\n+<community>s3cr3t-ro</community>\n"

// recordingDevice stands in for the device side of a push and records each
// step. A dry run before a commit answers pushDiff; after it, the post-check,
// it finds the change made.
type recordingDevice struct {
	mu       sync.Mutex
	requests []deviceconfig.DeviceRequest
	steps    []string
	// boot is the image the device boots next and running the one it runs.
	boot, running string
}

func (d *recordingDevice) record(step string, req deviceconfig.DeviceRequest) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	committed := slices.Contains(d.steps, "commit")
	d.requests = append(d.requests, req)
	d.steps = append(d.steps, step)
	return committed
}

func (d *recordingDevice) DryRun(_ context.Context, req deviceconfig.DeviceRequest) (deviceconfig.DryRunResult, error) {
	if d.record("dry-run", req) {
		return deviceconfig.DryRunResult{}, nil
	}
	return deviceconfig.DryRunResult{Diff: pushDiff, Added: 1, Removed: 1}, nil
}

func (d *recordingDevice) Commit(_ context.Context, req deviceconfig.DeviceRequest) error {
	d.record("commit", req)
	return nil
}

func (d *recordingDevice) Confirm(_ context.Context, req deviceconfig.DeviceRequest) error {
	d.record("confirm", req)
	return nil
}

func (d *recordingDevice) Rollback(_ context.Context, req deviceconfig.DeviceRequest) error {
	d.record("rollback", req)
	return nil
}

func (d *recordingDevice) ImageFacts(
	_ context.Context, req deviceconfig.DeviceRequest,
) (deviceconfig.ImageFacts, error) {
	d.record("facts", req)
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running == "" {
		d.running, d.boot = "flash:cat9k-17.09.04a.bin", "flash:cat9k-17.09.04a.bin"
	}
	return deviceconfig.ImageFacts{
		Version: "17.09.04a", RunningImage: d.running, Dir: "flash:", FreeBytes: 1 << 30, BootImage: d.boot,
	}, nil
}

func (d *recordingDevice) CopyImage(
	_ context.Context, req deviceconfig.DeviceRequest, img deviceconfig.DeviceImage, body io.Reader,
) error {
	if _, err := io.Copy(io.Discard, body); err != nil {
		return err
	}
	d.record("copy "+img.Path(), req)
	return nil
}

func (d *recordingDevice) VerifyImage(
	_ context.Context,
	req deviceconfig.DeviceRequest,
	_ deviceconfig.DeviceImage,
) error {
	d.record("verify", req)
	return nil
}

func (d *recordingDevice) SetBoot(
	_ context.Context, req deviceconfig.DeviceRequest, img deviceconfig.DeviceImage,
) error {
	d.record("boot", req)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.boot = img.Path()
	return nil
}

func (d *recordingDevice) Reload(_ context.Context, req deviceconfig.DeviceRequest) error {
	d.record("reload", req)
	return nil
}

// newDeviceConfigPushTestServer is newDeviceConfigTestServer with the push
// use-case wired over a recording device and an admin seeded.
func newDeviceConfigPushTestServer(t *testing.T) (*Server, *database.DB, *recordingDevice) {
	t.Helper()
	s, db := newDeviceConfigTestServer(t)
	seedRoledUser(t, db, dcAdmin, roles.Admin)
	device := &recordingDevice{}
	images, err := app.NewDeviceConfigImages(s.persistence)
	if err != nil {
		t.Fatalf("build image use-case: %v", err)
	}
	s.deviceImages = images
	pushes, err := app.NewDeviceConfigPushes(s.persistence, s.deviceConfigs, images, device)
	if err != nil {
		t.Fatalf("build push use-case: %v", err)
	}
	s.deviceConfigPushes = pushes
	s.registerDeviceConfigPushKind()
	return s, db, device
}

// pushReadyTarget creates a NETCONF target on the switch at 10.0.0.1, pins its
// key and enables pushing with a second credential, returning the target and
// the push credential's id.
func pushReadyTarget(t *testing.T, s *Server) (deviceconfig.Target, string) {
	t.Helper()
	return pushReadyTargetOn(t, s, deviceconfig.PlatformNETCONF)
}

// pushReadyTargetOn is pushReadyTarget for a target of another platform.
func pushReadyTargetOn(t *testing.T, s *Server, platform deviceconfig.Platform) (deviceconfig.Target, string) {
	t.Helper()
	rec := callDeviceConfig(t, s.handleDeviceConfigTargets, http.MethodPost, deviceConfigTargetsPath,
		`{"name":"core","host":"10.0.0.1","platform":"`+string(platform)+`","credentialsId":"`+
			createSSHCredential(t, s, "s3cret")+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create target: %d %s", rec.Code, rec.Body.String())
	}
	tgt := decodeInto[deviceconfig.Target](t, rec)
	runDeviceConfigBackup(t, s)
	confirmHostKey(t, s, tgt.ID)

	pushCred := createSSHCredential(t, s, "wr1te")
	put := callPushTarget(t, s, dcAdmin, http.MethodPut, tgt.ID, `{"credentialsId":"`+pushCred+`"}`)
	if put.Code != http.StatusNoContent {
		t.Fatalf("enable push: %d %s", put.Code, put.Body.String())
	}
	return tgt, pushCred
}

func callPushTarget(t *testing.T, s *Server, user, method, targetID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(newAuthedRequest(method, deviceConfigTargetsPathPrefix+targetID+"/push", []byte(body), user))
	req.SetPathValue("targetId", targetID)
	rec := httptest.NewRecorder()
	s.handleDeviceConfigTargetPush(rec, req)
	return rec
}

func callPushPlan(t *testing.T, s *Server, user, planID string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(newAuthedRequest(http.MethodGet, deviceConfigPushesPath+"/"+planID, nil, user))
	req.SetPathValue("planId", planID)
	rec := httptest.NewRecorder()
	s.handleDeviceConfigPushByID(rec, req)
	return rec
}

// planPush posts a plan as the admin and waits for its job.
func planPush(t *testing.T, s *Server, targetID string) deviceconfig.PlanResult {
	t.Helper()
	rec := callDeviceConfigAs(t, s.handleDeviceConfigPushes, dcAdmin, http.MethodPost, deviceConfigPushesPath,
		`{"targetId":"`+targetID+`","mode":"merge","candidate":"<snmp><community>s3cr3t-ro</community></snmp>"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("plan: %d %s", rec.Code, rec.Body.String())
	}
	jobID := decodeJob(t, rec.Body).ID
	var job jobs.Job
	waitFor(t, "plan finishes", func() bool {
		job, _ = s.jobsRunner().Get(jobID)
		return job.State.Terminal()
	})
	if job.State != jobs.StateSucceeded {
		t.Fatalf("plan state %s (%s), want succeeded", job.State, job.Err)
	}
	res, ok := job.Result.(deviceconfig.PlanResult)
	if !ok {
		t.Fatalf("plan result is %T", job.Result)
	}
	return res
}

// TestDeviceConfigPushPlanThroughTheAPI is the slice's acceptance through the
// API: an admin enables pushing with a second credential and plans a change;
// the dry run logs in with that credential; the plan is stored encrypted,
// readable by an operator and not by a viewer, and audited without its text.
func TestDeviceConfigPushPlanThroughTheAPI(t *testing.T) {
	s, db, dryRun := newDeviceConfigPushTestServer(t)
	tgt, _ := pushReadyTarget(t, s)

	res := planPush(t, s, tgt.ID)
	if res.Outcome != deviceconfig.PlanReady || res.PlanID == "" || res.Added != 1 || res.Removed != 1 {
		t.Fatalf("plan result = %+v, want a ready plan of +1 -1", res)
	}
	if len(dryRun.requests) != 1 || dryRun.requests[0].Password != "wr1te" ||
		dryRun.requests[0].PinnedHostKey != "SHA256:10.0.0.1" {
		t.Errorf("dry run logins = %+v, want one with the push credential and the pinned key", dryRun.requests)
	}

	requirePlanReads(t, s, tgt.ID, res.PlanID)
	requirePlanCiphertext(t, db, res.PlanID)
	requirePlanAudit(t, db, tgt.ID, res.PlanID)
}

// requirePlanReads checks an operator reads the plan list without its text and
// the plan with it, and a viewer reads neither.
func requirePlanReads(t *testing.T, s *Server, targetID, planID string) {
	t.Helper()
	list := callDeviceConfigAs(t, s.handleDeviceConfigPushes, dcOperator, http.MethodGet,
		deviceConfigPushesPath+"?targetId="+targetID, "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), planID) ||
		strings.Contains(list.Body.String(), "s3cr3t-ro") {
		t.Errorf("plan list: %d %s, want the plan without its text", list.Code, list.Body.String())
	}
	got := callPushPlan(t, s, dcOperator, planID)
	if got.Code != http.StatusOK {
		t.Fatalf("read plan: %d %s", got.Code, got.Body.String())
	}
	if p := decodeInto[deviceconfig.Plan](t, got); p.Diff != pushDiff || p.DiffSource != deviceconfig.DiffFromDevice ||
		!strings.Contains(p.Candidate, "s3cr3t-ro") || p.CreatedBy != dcAdmin {
		t.Errorf("plan read back = %+v", p)
	}
	if viewer := callPushPlan(t, s, dcViewer, planID); viewer.Code != http.StatusForbidden ||
		strings.Contains(viewer.Body.String(), "s3cr3t-ro") {
		t.Errorf("plan as a viewer: %d %s, want 403 without the text", viewer.Code, viewer.Body.String())
	}
}

// requirePlanCiphertext checks the stored candidate and diff are ciphertext.
func requirePlanCiphertext(t *testing.T, db *database.DB, planID string) {
	t.Helper()
	var candidateCT, diffCT string
	if err := db.QueryRow(t.Context(), `SELECT CAST(candidate_enc AS TEXT), CAST(diff_enc AS TEXT)
		FROM device_config_push_plans WHERE id = ?`, planID).Scan(&candidateCT, &diffCT); err != nil {
		t.Fatalf("read stored plan: %v", err)
	}
	for _, ct := range []string{candidateCT, diffCT} {
		if !strings.HasPrefix(ct, "enc:v1:") || strings.Contains(ct, "s3cr3t-ro") {
			t.Errorf("plan text is not stored as ciphertext: %q", ct)
		}
	}
}

// requirePlanAudit checks the plan was audited, by the admin, without its text.
func requirePlanAudit(t *testing.T, db *database.DB, targetID, planID string) {
	t.Helper()
	var action, user, detail string
	if err := db.QueryRow(t.Context(), `SELECT action, user, new_value_json FROM audit_log
		WHERE resource_type = ? AND resource_id = ?`, deviceconfig.PushAuditResource, targetID).
		Scan(&action, &user, &detail); err != nil {
		t.Fatalf("read push audit: %v", err)
	}
	if action != "device_config.push.plan_created" || user != dcAdmin || !strings.Contains(detail, planID) ||
		strings.Contains(detail, "s3cr3t-ro") {
		t.Errorf("push audit = %s by %s: %s", action, user, detail)
	}
}

func TestDeviceConfigPushRefusalsAreA400(t *testing.T) {
	s, _, dryRun := newDeviceConfigPushTestServer(t)
	tgt, pushCred := pushReadyTarget(t, s)
	off := createDeviceConfigTarget(t, s, "10.0.0.2", createSSHCredential(t, s, "s3cret"))

	for _, tc := range []struct {
		name, body, want string
	}{
		{"pushing off", `{"targetId":"` + off.ID + `","mode":"merge","candidate":"hostname x"}`, "not enabled"},
		{"bad mode", `{"targetId":"` + tgt.ID + `","mode":"overwrite","candidate":"<a/>"}`, "mode"},
		{"not XML", `{"targetId":"` + tgt.ID + `","mode":"merge","candidate":"hostname x"}`, "outside its elements"},
		{"no target", `{"mode":"merge","candidate":"<a/>"}`, "targetId"},
	} {
		rec := callDeviceConfigAs(t, s.handleDeviceConfigPushes, dcAdmin, http.MethodPost, deviceConfigPushesPath,
			tc.body)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s: %d %s, want 400 naming %q", tc.name, rec.Code, rec.Body.String(), tc.want)
		}
	}
	if len(dryRun.requests) != 0 {
		t.Errorf("a refused plan reached the device: %+v", dryRun.requests)
	}

	backupCred := callDeviceConfig(t, s.handleDeviceConfigTargetByID, http.MethodGet,
		deviceConfigTargetsPathPrefix+tgt.ID, "")
	same := callPushTarget(t, s, dcAdmin, http.MethodPut, tgt.ID,
		`{"credentialsId":"`+decodeInto[deviceconfig.Target](t, backupCred).CredentialsID+`"}`)
	if same.Code != http.StatusBadRequest {
		t.Errorf("push with the backup credential: %d %s, want 400", same.Code, same.Body.String())
	}
	if rec := callPushTarget(t, s, dcAdmin, http.MethodDelete, tgt.ID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("disable push: %d %s", rec.Code, rec.Body.String())
	}
	got := decodeInto[deviceconfig.Target](t, callDeviceConfig(t, s.handleDeviceConfigTargetByID, http.MethodGet,
		deviceConfigTargetsPathPrefix+tgt.ID, ""))
	if got.PushCredentialsID != "" {
		t.Errorf("after disabling, push credential = %q (was %q)", got.PushCredentialsID, pushCred)
	}
}

// TestDeviceConfigPushNeedsAdminAndPro drives the push routes through the
// real chain: below Pro they answer 402, and at Pro an operator holding a
// valid CSRF token is refused by the role gate.
func TestDeviceConfigPushNeedsAdminAndPro(t *testing.T) {
	t.Parallel()
	s, _ := routedServer(t, dcOperator, roles.Operator)
	token, err := s.authManager().GenerateAccessToken(t.Context(), dcOperator)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	csrfToken := fetchCSRFToken(t, s, token)
	mgr, err := license.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("license manager: %v", err)
	}
	s.licenseMgr = mgr

	const target = deviceConfigTargetsPathPrefix + "dct-0123456789ab/push"
	routes := []struct{ method, path string }{
		{http.MethodPut, target},
		{http.MethodDelete, target},
		{http.MethodPost, deviceConfigPushesPath},
		{http.MethodPost, deviceConfigPushesPath + "/dcp-0123456789ab/apply"},
	}
	call := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Csrf-Token", csrfToken)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec
	}
	for _, tc := range routes {
		if rec := call(tc.method, tc.path); rec.Code != http.StatusPaymentRequired ||
			!strings.Contains(rec.Body.String(), configPushFeature) {
			t.Errorf("Free %s %s: %d %s, want 402 naming %s", tc.method, tc.path, rec.Code, rec.Body.String(),
				configPushFeature)
		}
	}
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}
	for _, tc := range routes {
		if rec := call(tc.method, tc.path); rec.Code != http.StatusForbidden ||
			!strings.Contains(rec.Body.String(), "admin") {
			t.Errorf("operator %s %s: %d %s, want 403 from the role gate", tc.method, tc.path, rec.Code,
				rec.Body.String())
		}
	}
	// The admin scope gates the plan, not reading plans.
	if rec := call(http.MethodGet, deviceConfigPushesPath+"?targetId=dct-0123456789ab"); rec.Code != http.StatusOK {
		t.Errorf("operator lists plans: %d %s, want 200", rec.Code, rec.Body.String())
	}
}

func TestJobsRouteRefusesPushJobs(t *testing.T) {
	s, _, _ := newDeviceConfigPushTestServer(t)
	for _, kind := range []string{deviceConfigPushPlanJobKind, deviceConfigPushApplyJobKind} {
		req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/jobs",
			strings.NewReader(`{"kind":"`+kind+`","params":{}}`))
		rec := httptest.NewRecorder()
		s.handleJobs(rec, req)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), deviceConfigPushesPath) {
			t.Errorf("POST /jobs %s: %d %s, want 400 naming the push route", kind, rec.Code, rec.Body.String())
		}
	}
}

func callPushApply(t *testing.T, s *Server, req *http.Request, planID string) *httptest.ResponseRecorder {
	t.Helper()
	req.SetPathValue("planId", planID)
	rec := httptest.NewRecorder()
	s.handleDeviceConfigPushApply(rec, req)
	return rec
}

// applyRequest is the admin's apply of planID by the diff hash sha.
func applyRequest(planID, sha string) *http.Request {
	return withClaim(newAuthedRequest(http.MethodPost, deviceConfigPushesPath+"/"+planID+"/apply",
		[]byte(`{"planSha256":"`+sha+`"}`), dcAdmin))
}

// TestDeviceConfigPushApplyThroughTheAPI is the slice's acceptance through
// the API: the admin applies the plan they read by its diff hash; the job
// commits with the push credential, post-checks, confirms and backs up; the
// plan then reads as confirmed, and a second apply is a 409.
func TestDeviceConfigPushApplyThroughTheAPI(t *testing.T) {
	s, db, device := newDeviceConfigPushTestServer(t)
	tgt, _ := pushReadyTarget(t, s)
	planID := planPush(t, s, tgt.ID).PlanID
	plan := decodeInto[deviceconfig.Plan](t, callPushPlan(t, s, dcAdmin, planID))

	rec := callPushApply(t, s, applyRequest(planID, plan.DiffSHA256), planID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("apply: %d %s", rec.Code, rec.Body.String())
	}
	jobID := decodeJob(t, rec.Body).ID
	var job jobs.Job
	waitFor(t, "apply finishes", func() bool {
		job, _ = s.jobsRunner().Get(jobID)
		return job.State.Terminal()
	})
	res, ok := job.Result.(deviceconfig.ApplyResult)
	if job.State != jobs.StateSucceeded || !ok || res.Outcome != deviceconfig.ApplyConfirmed {
		t.Fatalf("apply job %s (%s) result %+v, want confirmed", job.State, job.Err, job.Result)
	}
	if want := []string{"dry-run", "commit", "dry-run", "confirm"}; !slices.Equal(device.steps, want) {
		t.Errorf("device steps = %v, want %v", device.steps, want)
	}
	if commit := device.requests[1]; commit.Password != "wr1te" || commit.CommitID == "" ||
		commit.CommitID != device.requests[3].CommitID {
		t.Errorf("commit %+v / confirm %+v, want the push login and one commit id", commit, device.requests[3])
	}

	got := decodeInto[deviceconfig.Plan](t, callPushPlan(t, s, dcOperator, planID))
	if got.ApplyOutcome != deviceconfig.ApplyConfirmed || got.AppliedBy != dcAdmin || got.AppliedAt == nil {
		t.Errorf("plan after apply = %+v", got)
	}
	again := callPushApply(t, s, applyRequest(planID, plan.DiffSHA256), planID)
	if again.Code != http.StatusConflict {
		t.Errorf("second apply: %d %s, want 409", again.Code, again.Body.String())
	}

	var actions []string
	rows, err := db.Query(t.Context(), `SELECT action, new_value_json FROM audit_log WHERE resource_type = ?
		ORDER BY id`, deviceconfig.PushAuditResource)
	if err != nil {
		t.Fatalf("read push audit: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var action, detail string
		if err = rows.Scan(&action, &detail); err != nil {
			t.Fatalf("scan push audit: %v", err)
		}
		if strings.Contains(detail, "s3cr3t-ro") {
			t.Errorf("audit %s carries configuration text: %s", action, detail)
		}
		actions = append(actions, action)
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("iterate push audit: %v", err)
	}
	if want := []string{
		"device_config.push.plan_created", "device_config.push.apply_started", "device_config.push.confirmed",
	}; !slices.Equal(actions, want) {
		t.Errorf("push audit = %v, want %v", actions, want)
	}
}

// TestDeviceConfigPushApplyRefusals: a token never applies, and a hash that
// is not the plan's is a 400; neither reaches the device or spends the plan.
func TestDeviceConfigPushApplyRefusals(t *testing.T) {
	s, _, device := newDeviceConfigPushTestServer(t)
	tgt, _ := pushReadyTarget(t, s)
	planID := planPush(t, s, tgt.ID).PlanID
	plan := decodeInto[deviceconfig.Plan](t, callPushPlan(t, s, dcAdmin, planID))

	pat := applyRequest(planID, plan.DiffSHA256)
	pat = pat.WithContext(auth.WithAPITokenAuth(pat.Context()))
	if rec := callPushApply(t, s, pat, planID); rec.Code != http.StatusForbidden ||
		!strings.Contains(rec.Body.String(), "personal access token") {
		t.Errorf("apply by token: %d %s, want 403", rec.Code, rec.Body.String())
	}
	if rec := callPushApply(t, s, applyRequest(planID, strings.Repeat("0", 64)), planID); rec.Code !=
		http.StatusBadRequest || !strings.Contains(rec.Body.String(), "planSha256") {
		t.Errorf("apply with another hash: %d %s, want 400", rec.Code, rec.Body.String())
	}
	const unknown = "dcp-0123456789ab"
	if rec := callPushApply(t, s, applyRequest(unknown, plan.DiffSHA256), unknown); rec.Code != http.StatusNotFound {
		t.Errorf("apply of an unknown plan: %d %s, want 404", rec.Code, rec.Body.String())
	}
	if slices.Contains(device.steps, "commit") {
		t.Errorf("a refused apply reached the device: %v", device.steps)
	}
	if got := decodeInto[deviceconfig.Plan](t, callPushPlan(t, s, dcAdmin, planID)); got.AppliedAt != nil {
		t.Errorf("a refused apply spent the plan: %+v", got)
	}
}
