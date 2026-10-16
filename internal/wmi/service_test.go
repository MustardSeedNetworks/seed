package wmi_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

const client = "default"

// memRepo is an in-memory wmi.Repository.
type memRepo struct {
	mu          sync.Mutex
	targets     map[string]wmi.Target
	inventories map[string]wmi.StoredInventory
	attempts    []wmi.Attempt
	next        int
}

func newMemRepo() *memRepo {
	return &memRepo{targets: map[string]wmi.Target{}, inventories: map[string]wmi.StoredInventory{}}
}

func (m *memRepo) ListTargets(_ context.Context, clientID string) ([]wmi.Target, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []wmi.Target
	for _, t := range m.targets {
		if t.ClientID == clientID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memRepo) GetTarget(_ context.Context, clientID, id string) (wmi.Target, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.targets[id]
	if !ok || t.ClientID != clientID {
		return wmi.Target{}, wmi.ErrNotFound
	}
	return t, nil
}

func (m *memRepo) SaveTarget(_ context.Context, t *wmi.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.ID == "" {
		m.next++
		t.ID = fmt.Sprintf("wmt-%012x", m.next)
	} else {
		prev := m.targets[t.ID]
		t.LastStatus, t.LastAttemptAt, t.LastError = prev.LastStatus, prev.LastAttemptAt, prev.LastError
	}
	m.targets[t.ID] = *t
	return nil
}

func (m *memRepo) DeleteTarget(_ context.Context, _, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.targets, id)
	delete(m.inventories, id)
	return nil
}

func (m *memRepo) SetPendingCert(_ context.Context, _, id, fingerprint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.targets[id]
	if t.CertSHA256 == "" {
		t.PendingCertSHA256 = fingerprint
	}
	m.targets[id] = t
	return nil
}

func (m *memRepo) ConfirmCert(_ context.Context, _, id, fingerprint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.targets[id]
	if t.PendingCertSHA256 != fingerprint {
		return wmi.ErrNotFound
	}
	t.CertSHA256, t.PendingCertSHA256 = fingerprint, ""
	m.targets[id] = t
	return nil
}

func (m *memRepo) ClearCert(_ context.Context, _, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.targets[id]
	t.CertSHA256, t.PendingCertSHA256 = "", ""
	m.targets[id] = t
	return nil
}

func (m *memRepo) RecordAttempt(_ context.Context, a wmi.Attempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts = append(m.attempts, a)
	t := m.targets[a.TargetID]
	t.LastAttemptAt, t.LastStatus, t.LastError = a.At, a.Status, a.Error
	m.targets[a.TargetID] = t
	if a.Inventory != nil {
		m.inventories[a.TargetID] = wmi.StoredInventory{
			TargetID:    a.TargetID,
			CollectedAt: a.At,
			Inventory:   *a.Inventory,
		}
	}
	return nil
}

func (m *memRepo) GetInventory(_ context.Context, _, targetID string) (wmi.StoredInventory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.inventories[targetID]
	if !ok {
		return wmi.StoredInventory{}, wmi.ErrNotFound
	}
	return inv, nil
}

type memCreds map[string]*polling.Credentials

func (m memCreds) Get(_ context.Context, id, clientID string) (*polling.Credentials, error) {
	c, ok := m[id]
	if !ok || c.ClientID != clientID {
		return nil, polling.ErrCredentialsNotFound
	}
	return c, nil
}

// prefixCipher "decrypts" by stripping a fixed prefix.
type prefixCipher struct{}

func (prefixCipher) DecryptValue(s string) (string, error) {
	plain, ok := strings.CutPrefix(s, "enc:v1:")
	if !ok {
		return "", errors.New("not ciphertext")
	}
	return plain, nil
}

// fakeCollector answers each host from a table and records what it was sent.
type fakeCollector struct {
	mu     sync.Mutex
	byHost map[string]func(wmi.Endpoint) (wmi.Inventory, error)
	seen   []wmi.Endpoint
}

func (f *fakeCollector) Collect(_ context.Context, e wmi.Endpoint) (wmi.Inventory, error) {
	f.mu.Lock()
	f.seen = append(f.seen, e)
	answer := f.byHost[e.Host]
	f.mu.Unlock()
	return answer(e)
}

type fixture struct {
	svc     *wmi.Service
	repo    *memRepo
	collect *fakeCollector
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	creds := memCreds{
		"cred-w": {
			ID: "cred-w", ClientID: client, Kind: polling.CredentialKindWinRM,
			WinRMUser: `LAB\inventory`, WinRMPasswordCT: "enc:v1:s3cret",
		},
		"cred-s": {
			ID: "cred-s", ClientID: client, Kind: polling.CredentialKindSSH,
			SSHUser: "backup", SSHPasswordCT: "enc:v1:pw",
		},
	}
	repo := newMemRepo()
	collect := &fakeCollector{byHost: map[string]func(wmi.Endpoint) (wmi.Inventory, error){}}
	svc, err := wmi.NewService(repo, creds, prefixCipher{}, collect)
	require.NoError(t, err)
	return fixture{svc: svc, repo: repo, collect: collect}
}

func (f fixture) target(t *testing.T, host string) wmi.Target {
	t.Helper()
	tgt, err := f.svc.SaveTarget(t.Context(), wmi.TargetInput{
		ClientID: client, Name: "win-" + host, Host: host, CredentialsID: "cred-w",
	})
	require.NoError(t, err)
	return tgt
}

func TestSaveTargetValidates(t *testing.T) {
	f := newFixture(t)

	tgt := f.target(t, "10.0.0.5")
	require.Equal(t, wmi.DefaultPort, tgt.Port, "the port defaults to the WinRM HTTPS listener")

	tests := []struct {
		name string
		in   wmi.TargetInput
		want string
	}{
		{"no name", wmi.TargetInput{Host: "10.0.0.1", CredentialsID: "cred-w"}, "name is required"},
		{"bad host", wmi.TargetInput{Name: "x", Host: "not a host!", CredentialsID: "cred-w"}, "host must be"},
		{
			"bad port",
			wmi.TargetInput{Name: "x", Host: "10.0.0.1", Port: 70000, CredentialsID: "cred-w"},
			"port must be",
		},
		{"unknown credential", wmi.TargetInput{Name: "x", Host: "10.0.0.1", CredentialsID: "nope"}, "does not name"},
		{"ssh credential", wmi.TargetInput{Name: "x", Host: "10.0.0.1", CredentialsID: "cred-s"}, "WinRM credential"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.ClientID = client
			_, err := f.svc.SaveTarget(t.Context(), tt.in)
			var ve wmi.ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, ve.Msg, tt.want)
		})
	}
}

func TestRunPinsNothingUntilAnOperatorConfirms(t *testing.T) {
	f := newFixture(t)
	tgt := f.target(t, "10.0.0.5")
	presented := strings.Repeat("AB", 32)
	f.collect.byHost["10.0.0.5"] = func(e wmi.Endpoint) (wmi.Inventory, error) {
		if e.PinnedCert == "" {
			return wmi.Inventory{}, &wmi.CertUnconfirmedError{Fingerprint: presented}
		}
		return wmi.Inventory{OS: wmi.OperatingSystem{Caption: "Windows Server 2025"}}, nil
	}

	res, err := f.svc.Run(t.Context(), client, nil, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, wmi.StatusCertUnconfirmed, res.Results[0].Status)
	require.Equal(t, 1, res.Failed)

	got, err := f.svc.GetTarget(t.Context(), client, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, presented, got.PendingCertSHA256)
	require.Empty(t, got.CertSHA256)
	_, err = f.svc.GetInventory(t.Context(), client, tgt.ID)
	require.ErrorIs(t, err, wmi.ErrNotFound)

	var ve wmi.ValidationError
	_, err = f.svc.ConfirmCert(t.Context(), client, tgt.ID, strings.Repeat("CD", 32))
	require.ErrorAs(t, err, &ve, "only the certificate the operator was shown can be confirmed")
	confirmed, err := f.svc.ConfirmCert(t.Context(), client, tgt.ID, presented)
	require.NoError(t, err)
	require.Equal(t, presented, confirmed.CertSHA256)

	res, err = f.svc.Run(t.Context(), client, []string{tgt.ID}, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, wmi.StatusOK, res.Results[0].Status)
	inv, err := f.svc.GetInventory(t.Context(), client, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, "Windows Server 2025", inv.Inventory.OS.Caption)

	last := f.collect.seen[len(f.collect.seen)-1]
	require.Equal(t, wmi.Endpoint{
		Host: "10.0.0.5", Port: wmi.DefaultPort, User: `LAB\inventory`, Password: "s3cret", PinnedCert: presented,
	}, last, "the collector gets the decrypted password and the pin")
}

func TestRunRecordsEachHostsOutcome(t *testing.T) {
	f := newFixture(t)
	for host, err := range map[string]error{
		"10.0.0.1": nil,
		"10.0.0.2": fmt.Errorf("%w: host refused the credential", wmi.ErrAuth),
		"10.0.0.3": fmt.Errorf("%w: connection refused", wmi.ErrUnreachable),
		"10.0.0.4": fmt.Errorf("%w: host presented X, pinned Y", wmi.ErrCertMismatch),
		"10.0.0.5": fmt.Errorf("%w 2150859002: invalid class", wmi.ErrFault),
	} {
		f.target(t, host)
		f.collect.byHost[host] = func(wmi.Endpoint) (wmi.Inventory, error) { return wmi.Inventory{}, err }
	}

	var reports []float64
	res, err := f.svc.Run(t.Context(), client, nil, func(p float64) { reports = append(reports, p) })
	require.NoError(t, err)
	require.Equal(t, 1, res.Succeeded)
	require.Equal(t, 4, res.Failed)
	require.Len(t, reports, 5)
	require.InDelta(t, 1.0, reports[len(reports)-1], 1e-9)

	byHost := map[string]wmi.Status{}
	for _, tgt := range f.repo.targets {
		byHost[tgt.Host] = tgt.LastStatus
	}
	require.Equal(t, map[string]wmi.Status{
		"10.0.0.1": wmi.StatusOK,
		"10.0.0.2": wmi.StatusAuthFailed,
		"10.0.0.3": wmi.StatusUnreachable,
		"10.0.0.4": wmi.StatusCertMismatch,
		"10.0.0.5": wmi.StatusQueryFailed,
	}, byHost)
}

func TestRunRefusesACredentialEditedIntoAnotherKind(t *testing.T) {
	creds := memCreds{"cred-w": {
		ID: "cred-w", ClientID: client, Kind: polling.CredentialKindWinRM,
		WinRMUser: "u", WinRMPasswordCT: "enc:v1:p",
	}}
	repo := newMemRepo()
	collect := &fakeCollector{byHost: map[string]func(wmi.Endpoint) (wmi.Inventory, error){}}
	svc, err := wmi.NewService(repo, creds, prefixCipher{}, collect)
	require.NoError(t, err)
	tgt, err := svc.SaveTarget(t.Context(), wmi.TargetInput{
		ClientID: client, Name: "dc01", Host: "10.0.0.9", CredentialsID: "cred-w",
	})
	require.NoError(t, err)

	creds["cred-w"] = &polling.Credentials{
		ID: "cred-w", ClientID: client, Kind: polling.CredentialKindSSH,
		SSHUser: "u", SSHPasswordCT: "enc:v1:p",
	}
	res, err := svc.Run(t.Context(), client, []string{tgt.ID}, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, wmi.StatusError, res.Results[0].Status)
	require.Empty(t, collect.seen, "nothing is sent with a credential that is not a WinRM login")
}

func TestSaveTargetKeepsThePinOnlyAtTheSameAddress(t *testing.T) {
	f := newFixture(t)
	tgt := f.target(t, "10.0.0.5")
	pin := strings.Repeat("AB", 32)
	require.NoError(t, f.repo.SetPendingCert(t.Context(), client, tgt.ID, pin))
	_, err := f.svc.ConfirmCert(t.Context(), client, tgt.ID, pin)
	require.NoError(t, err)

	renamed, err := f.svc.SaveTarget(t.Context(), wmi.TargetInput{
		ID: tgt.ID, ClientID: client, Name: "renamed", Host: "10.0.0.5", CredentialsID: "cred-w",
	})
	require.NoError(t, err)
	require.Equal(t, pin, renamed.CertSHA256)

	moved, err := f.svc.SaveTarget(t.Context(), wmi.TargetInput{
		ID: tgt.ID, ClientID: client, Name: "renamed", Host: "10.0.0.6", CredentialsID: "cred-w",
	})
	require.NoError(t, err)
	require.Empty(t, moved.CertSHA256, "a certificate belongs to the address it was presented on")
}

func TestRunNamesAnUnknownTarget(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Run(t.Context(), client, []string{"wmt-000000000099"}, func(float64) {})
	var ve wmi.ValidationError
	require.ErrorAs(t, err, &ve)
}

func TestRunRecordsNothingWhenCancelled(t *testing.T) {
	f := newFixture(t)
	f.target(t, "10.0.0.5")
	ctx, cancel := context.WithCancel(t.Context())
	f.collect.byHost["10.0.0.5"] = func(wmi.Endpoint) (wmi.Inventory, error) {
		cancel()
		return wmi.Inventory{}, fmt.Errorf("%w: %w", wmi.ErrUnreachable, context.Canceled)
	}
	_, err := f.svc.Run(ctx, client, nil, func(float64) {})
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, f.repo.attempts, "an abandoned attempt is not a failed one")
}
