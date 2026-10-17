package deviceconfig_test

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

type memPlans struct {
	mu     sync.Mutex
	plans  []deviceconfig.Plan
	nextID int
}

func (m *memPlans) InsertPlan(_ context.Context, p *deviceconfig.Plan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	p.ID = fmt.Sprintf("dcp-%012d", m.nextID)
	m.plans = append(m.plans, *p)
	return nil
}

func (m *memPlans) GetPlan(_ context.Context, clientID, id string) (deviceconfig.Plan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.plans {
		if p.ID == id && p.ClientID == clientID {
			return p, nil
		}
	}
	return deviceconfig.Plan{}, deviceconfig.ErrNotFound
}

func (m *memPlans) ListPlans(_ context.Context, _, targetID string, limit int) ([]deviceconfig.Plan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []deviceconfig.Plan
	for _, p := range slices.Backward(m.plans) {
		if p.TargetID == targetID && len(out) < limit {
			p.CandidateCT, p.DiffCT = "", ""
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *memPlans) PrunePlans(context.Context, string, string, int) error { return nil }

func (m *memPlans) ClaimApply(_ context.Context, clientID, id, user string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.plans {
		if p.ID == id && p.ClientID == clientID && p.AppliedAt == nil && now.Before(p.ExpiresAt) {
			m.plans[i].AppliedAt, m.plans[i].AppliedBy = &now, user
			return nil
		}
	}
	return deviceconfig.ErrPlanSpent
}

func (m *memPlans) FinishApply(_ context.Context, clientID, id string, outcome deviceconfig.ApplyOutcome) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.plans {
		if p.ID == id && p.ClientID == clientID {
			m.plans[i].ApplyOutcome = outcome
		}
	}
	return nil
}

// memDevice is the device side of a push. A dry run answers res or err, or
// after a commit afterRes or afterErr, which is the post-check. Every login
// is recorded.
type memDevice struct {
	mu       sync.Mutex
	res      deviceconfig.DryRunResult
	err      error
	afterRes deviceconfig.DryRunResult
	afterErr error
	// commitErr, confirmErr and rollbackErr fail that step.
	commitErr, confirmErr, rollbackErr error
	// facts is what ImageFacts reads; factsErr, copyErr, verifyErr and
	// bootErr fail that staging step. copied is what CopyImage received.
	facts                                 deviceconfig.ImageFacts
	factsErr, copyErr, verifyErr, bootErr error
	copied                                string
	requests                              []deviceconfig.DeviceRequest
	steps                                 []string
	// gate, when set, holds each dry run and image copy until it is closed.
	gate chan struct{}
}

func (m *memDevice) DryRun(_ context.Context, req deviceconfig.DeviceRequest) (deviceconfig.DryRunResult, error) {
	m.mu.Lock()
	m.requests = append(m.requests, req)
	m.steps = append(m.steps, "dry-run")
	gate := m.gate
	committed := slices.Contains(m.steps, "commit")
	m.mu.Unlock()
	if gate != nil {
		<-gate
	}
	if committed {
		return m.afterRes, m.afterErr
	}
	return m.res, m.err
}

func (m *memDevice) step(name string, req deviceconfig.DeviceRequest, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, req)
	m.steps = append(m.steps, name)
	return err
}

func (m *memDevice) Commit(_ context.Context, req deviceconfig.DeviceRequest) error {
	return m.step("commit", req, m.commitErr)
}

func (m *memDevice) Confirm(_ context.Context, req deviceconfig.DeviceRequest) error {
	return m.step("confirm", req, m.confirmErr)
}

func (m *memDevice) Rollback(_ context.Context, req deviceconfig.DeviceRequest) error {
	return m.step("rollback", req, m.rollbackErr)
}

func (m *memDevice) ImageFacts(_ context.Context, req deviceconfig.DeviceRequest) (deviceconfig.ImageFacts, error) {
	return m.facts, m.step("facts", req, m.factsErr)
}

func (m *memDevice) CopyImage(
	_ context.Context, req deviceconfig.DeviceRequest, img deviceconfig.DeviceImage, body io.Reader,
) error {
	err := m.step("copy "+img.Path(), req, m.copyErr)
	if m.gate != nil {
		<-m.gate
	}
	b, readErr := io.ReadAll(body)
	m.mu.Lock()
	m.copied = string(b)
	m.mu.Unlock()
	return errors.Join(err, readErr)
}

func (m *memDevice) VerifyImage(_ context.Context, req deviceconfig.DeviceRequest, img deviceconfig.DeviceImage) error {
	return m.step("verify "+img.SHA512[:8], req, m.verifyErr)
}

func (m *memDevice) SetBoot(_ context.Context, req deviceconfig.DeviceRequest, img deviceconfig.DeviceImage) error {
	return m.step("boot "+img.Path()+" then "+img.Fallback, req, m.bootErr)
}

type memAudit struct {
	mu     sync.Mutex
	events []deviceconfig.PushAuditEvent
}

func (m *memAudit) record(_ context.Context, e deviceconfig.PushAuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return nil
}

func (m *memAudit) actions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, e := range m.events {
		out = append(out, e.Action)
	}
	return out
}

const iosRunning = "hostname edge\n!\narchive\n path flash:seed-archive\n!\n" +
	"interface Gi1/0/1\n description old\n switchport mode access\n!\nend\n"

type pushFixture struct {
	repo   *memRepo
	plans  *memPlans
	images *deviceconfig.Images
	store  *memImages
	device *memDevice
	audit  *memAudit
	fetch  *scriptedFetcher
	pushes *deviceconfig.Pushes
}

// pushCreds adds the push login to the vault the backup uses.
func pushCreds() memCreds {
	creds := sshCreds()
	creds["cred-push"] = &polling.Credentials{
		ID: "cred-push", ClientID: client, Kind: polling.CredentialKindSSH,
		SSHUser: "pusher", SSHPasswordCT: "enc:v1:wr1te",
	}
	return creds
}

func newPushFixture(t *testing.T, running string, targets ...deviceconfig.Target) pushFixture {
	t.Helper()
	answers := map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){}
	for _, tg := range targets {
		answers[tg.Host] = answer(running)
	}
	f := pushFixture{
		repo:   newMemRepo(targets...),
		plans:  &memPlans{},
		device: &memDevice{},
		audit:  &memAudit{},
		fetch:  &scriptedFetcher{answers: answers},
	}
	svc, err := deviceconfig.NewService(f.repo, pushCreds(), prefixCipher{}, f.fetch, &memAlerts{})
	require.NoError(t, err)
	dir := t.TempDir()
	f.store = &memImages{}
	f.images, err = deviceconfig.NewImages(
		f.store,
		func() (*os.Root, error) { return os.OpenRoot(dir) },
		f.audit.record,
	)
	require.NoError(t, err)
	f.pushes, err = deviceconfig.NewPushes(svc, f.plans, f.images, f.device, f.audit.record)
	require.NoError(t, err)
	return f
}

func pushTarget(id, host string, platform deviceconfig.Platform) deviceconfig.Target {
	tg := target(id, host, platform)
	tg.PushCredentialsID = "cred-push"
	return tg
}

func planRequest(targetID string, mode deviceconfig.PushMode, candidate string) deviceconfig.PlanRequest {
	return deviceconfig.PlanRequest{
		ClientID: client, TargetID: targetID, Mode: mode, Candidate: candidate,
		User: "admin", Remote: "192.0.2.10", UserAgent: "test",
	}
}

// TestPlanPredictsAnIOSMerge is the slice's acceptance at the unit level: a
// plan takes a fresh backup, predicts the diff, stores the candidate and the
// diff encrypted, binds the diff by its hash, and writes nothing to a device.
func TestPlanPredictsAnIOSMerge(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t, iosRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))

	res, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge,
		"interface Gi1/0/1\n description uplink to core\n"))
	require.NoError(t, err)
	require.Equal(t, deviceconfig.PlanReady, res.Outcome)
	require.Equal(t, 1, res.Added)
	require.Equal(t, 1, res.Removed)

	require.Len(t, f.repo.backups, 1, "the plan starts from a fresh backup")
	require.Empty(t, f.device.requests, "IOS has no dry run to ask for")
	for _, req := range f.fetch.requests {
		require.Equal(t, "show running-config", req.Command, "the only command sent is the backup's")
	}

	require.Len(t, f.plans.plans, 1)
	stored := f.plans.plans[0]
	require.Equal(t, res.PlanID, stored.ID)
	require.True(t, strings.HasPrefix(stored.CandidateCT, ctPrefix), "candidate stored encrypted")
	require.True(t, strings.HasPrefix(stored.DiffCT, ctPrefix), "diff stored encrypted")
	require.Equal(t, f.repo.backups[0].ID, stored.BackupID)
	require.Equal(t, f.repo.backups[0].SHA256, stored.RunningSHA256)
	require.Equal(t, deviceconfig.DiffPredicted, stored.DiffSource)
	require.Equal(t, "admin", stored.CreatedBy)
	require.Equal(t, deviceconfig.PlanLifetime, stored.ExpiresAt.Sub(stored.CreatedAt))

	plan, err := f.pushes.GetPlan(context.Background(), client, res.PlanID)
	require.NoError(t, err)
	require.Contains(t, plan.Diff, "- description old")
	require.Contains(t, plan.Diff, "+ description uplink to core")
	sum := sha256.Sum256([]byte(plan.Diff))
	require.Equal(t, hex.EncodeToString(sum[:]), plan.DiffSHA256)
	require.Empty(t, plan.CandidateCT)

	require.Equal(t, []string{"device_config.push.plan_created"}, f.audit.actions())
	detail := f.audit.events[0].Detail
	require.Equal(t, res.PlanID, detail["plan_id"])
	for _, v := range detail {
		require.NotContains(t, fmt.Sprint(v), "uplink", "configuration text never enters the audit log")
	}
}

// TestPlanPredictsAnIOSReplace: the replacement is the predicted
// configuration, compared without what "show running-config" prints around
// it, so a candidate saved from the device is not a change.
func TestPlanPredictsAnIOSReplace(t *testing.T) {
	t.Parallel()
	const saved = "Building configuration...\n\nCurrent configuration : 120 bytes\n!\n" +
		"! Last configuration change at 09:00:00 UTC Fri Oct 16 2026\n!\n"
	f := newPushFixture(t, saved+iosRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))

	res, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushReplace,
		"hostname edge\narchive\n path flash:seed-archive\ninterface Gi1/0/1\n description old\nend\n"))
	require.NoError(t, err)
	require.Equal(t, deviceconfig.PlanReady, res.Outcome)
	require.Equal(t, 0, res.Added)
	require.Equal(t, 1, res.Removed, "a line the replacement lacks is removed")
	plan, err := f.pushes.GetPlan(context.Background(), client, res.PlanID)
	require.NoError(t, err)
	require.Contains(t, plan.Diff, "- switchport mode access")
	require.Equal(t, deviceconfig.DiffPredicted, plan.DiffSource)
	require.Empty(t, f.device.requests, "IOS has no dry run to ask for")

	res, err = f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushReplace, saved+iosRunning))
	require.NoError(t, err)
	require.Equal(t, deviceconfig.PlanNoChange, res.Outcome)
}

func TestPlanOutcomesThatStoreNothing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mode      deviceconfig.PushMode
		running   string
		candidate string
		outcome   deviceconfig.PlanOutcome
		reason    string
		action    string
	}{
		{
			name:      "the change is already there",
			running:   iosRunning,
			candidate: "interface Gi1/0/1\n description old\n",
			outcome:   deviceconfig.PlanNoChange,
			action:    "device_config.push.plan_created",
		},
		{
			name:      "no archive path for the revert timer",
			running:   "hostname edge\n!\nend\n",
			candidate: "hostname core\n",
			outcome:   deviceconfig.PlanRefused,
			reason:    "archive path",
			action:    "device_config.push.plan_refused",
		},
		{
			name:      "a replacement without the archive path",
			mode:      deviceconfig.PushReplace,
			running:   iosRunning,
			candidate: "hostname core\n",
			outcome:   deviceconfig.PlanRefused,
			reason:    "in the replacement too",
			action:    "device_config.push.plan_refused",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newPushFixture(t, tc.running, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
			mode := cmp.Or(tc.mode, deviceconfig.PushMerge)
			res, err := f.pushes.Plan(context.Background(), planRequest("dct-1", mode, tc.candidate))
			require.NoError(t, err)
			require.Equal(t, tc.outcome, res.Outcome)
			require.Contains(t, res.Reason, tc.reason)
			require.Empty(t, res.PlanID)
			require.Empty(t, f.plans.plans)
			require.Equal(t, []string{tc.action}, f.audit.actions())
		})
	}
}

// TestPlanDryRunsNETCONFWithThePushCredential: the device's dry run logs in
// with the push credential, never the backup's, and its diff is labelled as
// the device's.
func TestPlanDryRunsNETCONFWithThePushCredential(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t, "<system><host-name>edge</host-name></system>\n",
		pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformNETCONF))
	f.device.res = deviceconfig.DryRunResult{Diff: "--- running\n+++ candidate\n-a\n+b\n", Added: 1, Removed: 1}

	candidate := `<system><host-name>core</host-name></system>`
	res, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushReplace, candidate))
	require.NoError(t, err)
	require.Equal(t, deviceconfig.PlanReady, res.Outcome)

	require.Len(t, f.device.requests, 1)
	req := f.device.requests[0]
	require.Equal(t, "pusher", req.User)
	require.Equal(t, "wr1te", req.Password)
	require.Equal(t, "SHA256:10.0.0.1", req.PinnedHostKey)
	require.Equal(t, deviceconfig.PushReplace, req.Mode)
	require.Equal(t, candidate, req.Candidate)
	for _, backup := range f.fetch.requests {
		require.Equal(t, "backup", backup.User, "the backup keeps its own login")
	}
	require.Equal(t, deviceconfig.DiffFromDevice, f.plans.plans[0].DiffSource)
}

func TestPlanDeviceFailures(t *testing.T) {
	t.Parallel()
	t.Run("the device refuses the dry run", func(t *testing.T) {
		t.Parallel()
		f := newPushFixture(t, "<a/>\n", pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformNETCONF))
		f.device.err = &deviceconfig.PlanRefusedError{Reason: "no confirmed-commit"}
		res, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "<a/>"))
		require.NoError(t, err)
		require.Equal(t, deviceconfig.PlanRefused, res.Outcome)
		require.Equal(t, "no confirmed-commit", res.Reason)
	})
	t.Run("the dry run fails", func(t *testing.T) {
		t.Parallel()
		f := newPushFixture(t, "<a/>\n", pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformNETCONF))
		f.device.err = fmt.Errorf("%w: lock: lock-denied", deviceconfig.ErrCommand)
		_, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "<a/>"))
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.Equal(t, []string{"device_config.push.plan_failed"}, f.audit.actions())
		require.Empty(t, f.plans.plans)
	})
	t.Run("the backup fails", func(t *testing.T) {
		t.Parallel()
		f := newPushFixture(t, iosRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
		f.fetch.answers["10.0.0.1"] = func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return deviceconfig.FetchResult{}, fmt.Errorf("%w: connection refused", deviceconfig.ErrUnreachable)
		}
		_, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "hostname x\n"))
		require.ErrorContains(t, err, "unreachable")
		require.Equal(t, []string{"device_config.push.plan_failed"}, f.audit.actions())
	})
}

// TestPlanOneJobPerTarget: a second plan on a target whose first is still
// running is refused rather than run beside it.
func TestPlanOneJobPerTarget(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t, "<a/>\n", pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformNETCONF))
	f.device.gate = make(chan struct{})
	f.device.res = deviceconfig.DryRunResult{Diff: "-a\n+b\n", Added: 1, Removed: 1}

	first := make(chan error, 1)
	go func() {
		_, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "<b/>"))
		first <- err
	}()
	require.Eventually(t, func() bool {
		f.device.mu.Lock()
		defer f.device.mu.Unlock()
		return len(f.device.requests) == 1
	}, 5*time.Second, 10*time.Millisecond)

	_, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "<b/>"))
	require.ErrorIs(t, err, deviceconfig.ErrPushBusy)
	close(f.device.gate)
	require.NoError(t, <-first)

	_, err = f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "<b/>"))
	require.NoError(t, err, "the target is free once the first job ends")
}

func TestPlanCheckRefusesBeforeALogin(t *testing.T) {
	t.Parallel()
	off := target("dct-off", "10.0.0.2", deviceconfig.PlatformCiscoIOS)
	unpinned := pushTarget("dct-unpinned", "10.0.0.3", deviceconfig.PlatformCiscoIOS)
	unpinned.HostKeySHA256 = ""
	f := newPushFixture(t, iosRunning,
		pushTarget("dct-ios", "10.0.0.1", deviceconfig.PlatformCiscoIOS), off, unpinned,
		pushTarget("dct-nxos", "10.0.0.4", deviceconfig.PlatformCiscoNXOS),
		pushTarget("dct-junos", "10.0.0.5", deviceconfig.PlatformJuniperJunos),
		pushTarget("dct-nc", "10.0.0.6", deviceconfig.PlatformNETCONF),
		pushTarget("dct-eos", "10.0.0.7", deviceconfig.PlatformAristaEOS),
	)
	tests := []struct {
		name string
		req  deviceconfig.PlanRequest
		msg  string
	}{
		{"unknown target", planRequest("dct-none", deviceconfig.PushMerge, "x"), "no backup target"},
		{"bad mode", planRequest("dct-ios", "overwrite", "x"), "mode must be"},
		{"empty candidate", planRequest("dct-ios", deviceconfig.PushMerge, " \n"), "candidate is required"},
		{
			"oversized candidate",
			planRequest("dct-ios", deviceconfig.PushMerge, strings.Repeat("x", deviceconfig.MaxCandidateBytes+1)),
			"exceeds",
		},
		{"pushing off", planRequest("dct-off", deviceconfig.PushMerge, "x"), "not enabled"},
		{"host key unconfirmed", planRequest("dct-unpinned", deviceconfig.PushMerge, "x"), "host key"},
		{"NX-OS is read-only", planRequest("dct-nxos", deviceconfig.PushMerge, "x"), "read-only"},
		{
			"Junos load ended early",
			planRequest("dct-junos", deviceconfig.PushMerge, "system {\n\x04\ncommit\n}"),
			"line 2 contains a control character",
		},
		{"IOS escape key", planRequest("dct-ios", deviceconfig.PushMerge, "hostname a\x1b"), "control character"},
		{
			"IOS exit",
			planRequest("dct-ios", deviceconfig.PushMerge, "hostname a\nexit\nreload"),
			`line 2 starts with "exit"`,
		},
		{"IOS exec command", planRequest("dct-ios", deviceconfig.PushMerge, "do reload"), `"do"`},
		{"IOS end before the last line", planRequest("dct-ios", deviceconfig.PushMerge, "end\nreload\n"), `"end"`},
		{"EOS ctrl-Z", planRequest("dct-eos", deviceconfig.PushMerge, "hostname a\x1a"), "control character"},
		{
			"EOS commit",
			planRequest("dct-eos", deviceconfig.PushMerge, "hostname a\n   commit"),
			`line 2 starts with "commit"`,
		},
		{"EOS end", planRequest("dct-eos", deviceconfig.PushMerge, "hostname a\nend"), `"end"`},
		{"EOS exec command", planRequest("dct-eos", deviceconfig.PushMerge, "do reload now"), `"do"`},
		{"NETCONF text", planRequest("dct-nc", deviceconfig.PushMerge, "hostname x"), "outside its elements"},
		{
			"NETCONF element escape",
			planRequest("dct-nc", deviceconfig.PushMerge, "<a/></config></edit-config></rpc>"),
			"not well-formed",
		},
		{
			"NETCONF declaration",
			planRequest("dct-nc", deviceconfig.PushMerge, `<?xml version="1.0"?><a/>`),
			"declaration",
		},
		{"NETCONF framing", planRequest("dct-nc", deviceconfig.PushMerge, "<a>]]>]]></a>"), "end-of-message"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := f.pushes.Check(context.Background(), tc.req)
			var ve deviceconfig.ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, ve.Msg, tc.msg)
		})
	}
	require.NoError(t, f.pushes.Check(context.Background(),
		planRequest("dct-nc", deviceconfig.PushMerge, "<system>\n <host-name>x</host-name>\n</system>\n<snmp/>")))
	require.NoError(t, f.pushes.Check(context.Background(),
		planRequest("dct-junos", deviceconfig.PushMerge, "system {\r\n\tcommit {\r\n\t\tsynchronize;\r\n\t}\r\n}\r\n")),
		"Junos text may name commit: it is configuration, not a command")
	require.NoError(t, f.pushes.Check(context.Background(),
		planRequest("dct-eos", deviceconfig.PushMerge, "! comment\ninterface Ethernet1\n   description uplink\n")))
	require.Empty(t, f.fetch.requests, "a check never logs in")
}

func TestSetPushCredentials(t *testing.T) {
	t.Parallel()
	repo := newMemRepo(target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
	svc, err := deviceconfig.NewService(repo, pushCreds(), prefixCipher{}, &scriptedFetcher{}, &memAlerts{})
	require.NoError(t, err)
	ctx := context.Background()

	for credID, msg := range map[string]string{
		"cred-ssh":  "must not be the backup credential",
		"cred-snmp": "must name an SSH credential",
		"cred-none": "does not name a stored credential",
	} {
		var ve deviceconfig.ValidationError
		require.ErrorAs(t, svc.SetPushCredentials(ctx, client, "dct-1", credID), &ve, credID)
		require.Contains(t, ve.Msg, msg)
	}
	require.ErrorIs(t, svc.SetPushCredentials(ctx, client, "dct-none", "cred-push"), deviceconfig.ErrNotFound)

	require.NoError(t, svc.SetPushCredentials(ctx, client, "dct-1", "cred-push"))
	saved, err := svc.SaveTarget(ctx, deviceconfig.TargetInput{
		ID: "dct-1", ClientID: client, Name: "renamed", Host: "10.0.0.1", Platform: deviceconfig.PlatformCiscoIOS,
		CredentialsID: "cred-ssh",
	})
	require.NoError(t, err)
	require.Equal(t, "cred-push", saved.PushCredentialsID, "an edit keeps the admin's push decision")

	_, err = svc.SaveTarget(ctx, deviceconfig.TargetInput{
		ID: "dct-1", ClientID: client, Name: "renamed", Host: "10.0.0.1", Platform: deviceconfig.PlatformCiscoIOS,
		CredentialsID: "cred-push",
	})
	var ve deviceconfig.ValidationError
	require.ErrorAs(t, err, &ve, "the backup cannot borrow the push login")

	require.NoError(t, svc.SetPushCredentials(ctx, client, "dct-1", ""))
	off, err := svc.GetTarget(ctx, client, "dct-1")
	require.NoError(t, err)
	require.Empty(t, off.PushCredentialsID)
}

func TestNewPushesNeedsEveryDependency(t *testing.T) {
	t.Parallel()
	_, err := deviceconfig.NewPushes(nil, &memPlans{}, nil, &memDevice{}, (&memAudit{}).record)
	require.ErrorIs(t, err, deviceconfig.ErrUnavailable)
}
