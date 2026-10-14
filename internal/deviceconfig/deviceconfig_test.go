package deviceconfig_test

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

type memRepo struct {
	mu      sync.Mutex
	targets map[string]deviceconfig.Target
	backups []deviceconfig.Backup
	nextID  int
}

func newMemRepo(targets ...deviceconfig.Target) *memRepo {
	r := &memRepo{targets: map[string]deviceconfig.Target{}}
	for _, t := range targets {
		r.targets[t.ID] = t
	}
	return r
}

func (r *memRepo) ListTargets(_ context.Context, clientID string) ([]deviceconfig.Target, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []deviceconfig.Target
	for _, t := range r.targets {
		if t.ClientID == clientID {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b deviceconfig.Target) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (r *memRepo) GetTarget(_ context.Context, clientID, id string) (deviceconfig.Target, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.targets[id]
	if !ok || t.ClientID != clientID {
		return deviceconfig.Target{}, deviceconfig.ErrNotFound
	}
	return t, nil
}

func (r *memRepo) SaveTarget(_ context.Context, t *deviceconfig.Target) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.ID == "" {
		r.nextID++
		t.ID = fmt.Sprintf("dct-%012d", r.nextID)
	}
	r.targets[t.ID] = *t
	return nil
}

func (r *memRepo) DeleteTarget(context.Context, string, string) error { return nil }

func (r *memRepo) SetPendingHostKey(_ context.Context, _, id, fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.targets[id]
	if t.HostKeySHA256 == "" {
		t.PendingHostKeySHA256 = fingerprint
		r.targets[id] = t
	}
	return nil
}

func (r *memRepo) ConfirmHostKey(_ context.Context, _, id, fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.targets[id]
	if t.PendingHostKeySHA256 != fingerprint {
		return deviceconfig.ErrNotFound
	}
	t.HostKeySHA256, t.PendingHostKeySHA256 = fingerprint, ""
	r.targets[id] = t
	return nil
}

func (r *memRepo) ClearHostKey(context.Context, string, string) error { return nil }

func (r *memRepo) InsertBackup(_ context.Context, b *deviceconfig.Backup) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	b.ID = fmt.Sprintf("dcb-%d", r.nextID)
	r.backups = append(r.backups, *b)
	return nil
}

// targetBackups returns a target's attempts, newest (last inserted) first.
func (r *memRepo) targetBackups(targetID string) []deviceconfig.Backup {
	var out []deviceconfig.Backup
	for _, b := range slices.Backward(r.backups) {
		if b.TargetID == targetID {
			out = append(out, b)
		}
	}
	return out
}

func (r *memRepo) PruneBackups(_ context.Context, _, targetID string, keep int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.targetBackups(targetID)
	if len(kept) > keep {
		kept = kept[:keep]
	}
	r.backups = slices.DeleteFunc(r.backups, func(b deviceconfig.Backup) bool {
		return b.TargetID == targetID && !slices.ContainsFunc(kept, func(k deviceconfig.Backup) bool {
			return k.ID == b.ID
		})
	})
	return nil
}

func (r *memRepo) ListBackups(_ context.Context, _, targetID string, limit int) ([]deviceconfig.Backup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.targetBackups(targetID)
	if len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (r *memRepo) GetBackup(_ context.Context, _, id string) (deviceconfig.Backup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.backups {
		if b.ID == id {
			return b, nil
		}
	}
	return deviceconfig.Backup{}, deviceconfig.ErrNotFound
}

type memCreds map[string]*polling.Credentials

func (m memCreds) Get(_ context.Context, id, clientID string) (*polling.Credentials, error) {
	c, ok := m[id]
	if !ok || c.ClientID != clientID {
		return nil, polling.ErrCredentialsNotFound
	}
	return c, nil
}

// prefixCipher stands in for the keyring, passing already-encrypted input
// through as the keyring does.
type prefixCipher struct{}

const ctPrefix = "enc:v1:"

func (prefixCipher) EncryptValue(plaintext string) (string, error) {
	if strings.HasPrefix(plaintext, ctPrefix) {
		return plaintext, nil
	}
	return ctPrefix + plaintext, nil
}

func (prefixCipher) DecryptValue(ct string) (string, error) {
	return strings.TrimPrefix(ct, ctPrefix), nil
}

// scriptedFetcher answers per host and records what it was asked.
type scriptedFetcher struct {
	mu       sync.Mutex
	answers  map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error)
	requests []deviceconfig.FetchRequest
}

func (f *scriptedFetcher) Fetch(_ context.Context, req deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	answer := f.answers[req.Host]
	f.mu.Unlock()
	return answer(req)
}

const client = "default"

func sshCreds() memCreds {
	return memCreds{
		"cred-ssh": {
			ID: "cred-ssh", ClientID: client, Kind: polling.CredentialKindSSH,
			SSHUser: "backup", SSHPasswordCT: "enc:v1:s3cret",
		},
		"cred-snmp": {
			ID: "cred-snmp", ClientID: client, Kind: polling.CredentialKindV2c,
			SNMPCommunityCT: "enc:v1:public",
		},
	}
}

// target returns a target whose host key is pinned to "SHA256:" + host.
func target(id, host string, platform deviceconfig.Platform) deviceconfig.Target {
	return deviceconfig.Target{
		ID: id, ClientID: client, Name: id, Host: host, Port: 22, Platform: platform, CredentialsID: "cred-ssh",
		HostKeySHA256: "SHA256:" + host,
	}
}

// answer returns output to a login that presents host's pinned key, and
// otherwise refuses it as an SSHFetcher would.
func answer(output string) func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
	return func(req deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
		presented := "SHA256:" + req.Host
		switch req.PinnedHostKey {
		case "":
			return deviceconfig.FetchResult{}, &deviceconfig.HostKeyUnconfirmedError{Fingerprint: presented}
		case presented:
			return deviceconfig.FetchResult{Output: output}, nil
		}
		return deviceconfig.FetchResult{}, fmt.Errorf("%w: presented %s", deviceconfig.ErrHostKeyMismatch, presented)
	}
}

// TestRunReportsEachDeviceWithoutFailingTheRun is the row's acceptance at the
// unit level: a configuration is stored, and a refused credential is that
// device's result rather than the run's failure.
func TestRunReportsEachDeviceWithoutFailingTheRun(t *testing.T) {
	t.Parallel()
	repo := newMemRepo(
		target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS),
		target("dct-2", "10.0.0.2", deviceconfig.PlatformJuniperJunos),
		target("dct-3", "10.0.0.3", deviceconfig.PlatformAristaEOS),
	)
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": answer("hostname core\r\n!\r\n"),
		"10.0.0.2": func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return deviceconfig.FetchResult{}, fmt.Errorf("%w: ssh: unable to authenticate", deviceconfig.ErrAuth)
		},
		"10.0.0.3": func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return deviceconfig.FetchResult{}, fmt.Errorf("%w: connection refused", deviceconfig.ErrUnreachable)
		},
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, fetch)
	require.NoError(t, err)

	var progress []float64
	var mu sync.Mutex
	res, err := svc.Run(context.Background(), client, nil, func(p float64) {
		mu.Lock()
		progress = append(progress, p)
		mu.Unlock()
	})
	require.NoError(t, err)
	require.Equal(t, 1, res.Succeeded)
	require.Equal(t, 2, res.Failed)

	byTarget := map[string]deviceconfig.DeviceResult{}
	for _, r := range res.Results {
		byTarget[r.TargetID] = r
	}
	require.Equal(t, deviceconfig.StatusOK, byTarget["dct-1"].Status)
	require.Equal(t, deviceconfig.StatusAuthFailed, byTarget["dct-2"].Status)
	require.Equal(t, deviceconfig.StatusUnreachable, byTarget["dct-3"].Status)
	require.NotEmpty(t, byTarget["dct-2"].BackupID, "a failed attempt is recorded too")
	require.Contains(t, progress, 1.0)

	require.Len(t, repo.backups, 3)
	for _, b := range repo.backups {
		if b.TargetID == "dct-1" {
			require.Equal(t, ctPrefix+"hostname core\n!\n", b.ConfigCT, "stored encrypted, CRLF as LF")
			require.Empty(t, b.Config, "plaintext never reaches the repository")
			require.Len(t, b.SHA256, 64)
		} else {
			require.Empty(t, b.ConfigCT)
			require.NotEmpty(t, b.Error)
		}
	}
	got, err := svc.GetBackup(context.Background(), client, byTarget["dct-1"].BackupID)
	require.NoError(t, err)
	require.Equal(t, "hostname core\n!\n", got.Config, "reads decrypt")
	require.Empty(t, got.ConfigCT)

	commands := map[string]string{}
	for _, r := range fetch.requests {
		require.Equal(t, "backup", r.User)
		require.Equal(t, "s3cret", r.Password, "the vault password is decrypted for the login")
		commands[r.Host] = r.Command
	}
	require.Equal(t, "show running-config", commands["10.0.0.1"])
	require.Equal(t, "show configuration", commands["10.0.0.2"])
}

func TestRunPassesThePinAndTreatsEmptyOutputAsFailure(t *testing.T) {
	t.Parallel()
	repo := newMemRepo(target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoNXOS))
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": answer(" \r\n"),
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, fetch)
	require.NoError(t, err)

	res, err := svc.Run(context.Background(), client, []string{"dct-1"}, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StatusCommandFailed, res.Results[0].Status)
}

func TestRunRefusesAnUnknownTarget(t *testing.T) {
	t.Parallel()
	svc, err := deviceconfig.NewService(newMemRepo(), sshCreds(), prefixCipher{}, &scriptedFetcher{})
	require.NoError(t, err)

	_, err = svc.Run(context.Background(), client, []string{"dct-404"}, func(float64) {})
	var ve deviceconfig.ValidationError
	require.ErrorAs(t, err, &ve)
}

func TestSaveTargetValidates(t *testing.T) {
	t.Parallel()
	svc, err := deviceconfig.NewService(newMemRepo(), sshCreds(), prefixCipher{}, &scriptedFetcher{})
	require.NoError(t, err)

	valid := deviceconfig.TargetInput{
		ClientID: client, Name: "core", Host: "core.example.net",
		Platform: deviceconfig.PlatformCiscoIOS, CredentialsID: "cred-ssh",
	}
	for name, mutate := range map[string]func(*deviceconfig.TargetInput){
		"no name":          func(in *deviceconfig.TargetInput) { in.Name = " " },
		"bad host":         func(in *deviceconfig.TargetInput) { in.Host = "core switch" },
		"bad port":         func(in *deviceconfig.TargetInput) { in.Port = 70000 },
		"unknown platform": func(in *deviceconfig.TargetInput) { in.Platform = "cisco-asa" },
		"no credential":    func(in *deviceconfig.TargetInput) { in.CredentialsID = "cred-404" },
		"snmp credential":  func(in *deviceconfig.TargetInput) { in.CredentialsID = "cred-snmp" },
	} {
		in := valid
		mutate(&in)
		_, saveErr := svc.SaveTarget(context.Background(), in)
		var ve deviceconfig.ValidationError
		require.ErrorAs(t, saveErr, &ve, name)
	}

	got, err := svc.SaveTarget(context.Background(), valid)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.DefaultPort, got.Port)
}

func TestSaveTargetKeepsThePinUnlessTheAddressMoves(t *testing.T) {
	t.Parallel()
	pinned := target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS)
	pinned.PendingHostKeySHA256 = "SHA256:next"
	repo := newMemRepo(pinned)
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, &scriptedFetcher{})
	require.NoError(t, err)

	in := deviceconfig.TargetInput{
		ID: "dct-1", ClientID: client, Name: "renamed", Host: "10.0.0.1", Port: 22,
		Platform: deviceconfig.PlatformCiscoIOS, CredentialsID: "cred-ssh",
	}
	got, err := svc.SaveTarget(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, "SHA256:10.0.0.1", got.HostKeySHA256)
	require.Equal(t, "SHA256:next", got.PendingHostKeySHA256)

	in.Host = "10.0.0.9"
	got, err = svc.SaveTarget(context.Background(), in)
	require.NoError(t, err)
	require.Empty(t, got.HostKeySHA256, "a new address has a new key to learn")
	require.Empty(t, got.PendingHostKeySHA256)
}

// TestHostKeyIsConfirmedBeforeTheDeviceIsTrusted: an unpinned target is
// contacted without its password reaching the device, its key waits for an
// operator, and only the confirmed key is pinned.
func TestHostKeyIsConfirmedBeforeTheDeviceIsTrusted(t *testing.T) {
	t.Parallel()
	unpinned := target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS)
	unpinned.HostKeySHA256 = ""
	repo := newMemRepo(unpinned)
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": answer("hostname core\n"),
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, fetch)
	require.NoError(t, err)
	ctx := context.Background()

	res, err := svc.Run(ctx, client, nil, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StatusHostKeyUnconfirmed, res.Results[0].Status)
	require.Empty(t, fetch.requests[0].PinnedHostKey)
	tgt, err := svc.GetTarget(ctx, client, "dct-1")
	require.NoError(t, err)
	require.Empty(t, tgt.HostKeySHA256, "a presented key is not trusted")
	require.Equal(t, "SHA256:10.0.0.1", tgt.PendingHostKeySHA256)

	for name, fpr := range map[string]string{"empty": "", "different key": "SHA256:other"} {
		var ve deviceconfig.ValidationError
		require.ErrorAs(t, svc.ConfirmHostKey(ctx, client, "dct-1", fpr), &ve, name)
	}
	require.NoError(t, svc.ConfirmHostKey(ctx, client, "dct-1", "SHA256:10.0.0.1"))

	res, err = svc.Run(ctx, client, nil, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StatusOK, res.Results[0].Status)
	require.Equal(t, "SHA256:10.0.0.1", fetch.requests[1].PinnedHostKey)
}

// TestMismatchedHostKeyIsNotQueuedForConfirmation: a pinned target presenting
// another key is refused, and that key is not offered as pending.
func TestMismatchedHostKeyIsNotQueuedForConfirmation(t *testing.T) {
	t.Parallel()
	moved := target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS)
	moved.HostKeySHA256 = "SHA256:original"
	repo := newMemRepo(moved)
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": answer("hostname core\n"),
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, fetch)
	require.NoError(t, err)

	res, err := svc.Run(context.Background(), client, nil, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StatusHostKeyMismatch, res.Results[0].Status)
	require.Empty(t, repo.targets["dct-1"].PendingHostKeySHA256)
	require.Equal(t, "SHA256:original", repo.targets["dct-1"].HostKeySHA256)
}

func TestUnchangedConfigurationIsNotStoredAgain(t *testing.T) {
	t.Parallel()
	repo := newMemRepo(target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
	output := "hostname core\n"
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": func(req deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return answer(output)(req)
		},
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, fetch)
	require.NoError(t, err)
	run := func() deviceconfig.DeviceResult {
		res, runErr := svc.Run(context.Background(), client, nil, func(float64) {})
		require.NoError(t, runErr)
		return res.Results[0]
	}

	first := run()
	second := run()
	require.Equal(t, deviceconfig.StatusOK, second.Status)
	require.Equal(t, first.BackupID, second.BackupID, "an identical configuration names the backup holding it")
	require.Len(t, repo.backups, 1)

	output = "hostname core-2\n"
	third := run()
	require.NotEqual(t, first.BackupID, third.BackupID)
	require.Len(t, repo.backups, 2)
}

func TestBackupHistoryIsCapped(t *testing.T) {
	t.Parallel()
	repo := newMemRepo(target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return deviceconfig.FetchResult{}, fmt.Errorf("%w: refused", deviceconfig.ErrUnreachable)
		},
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, fetch)
	require.NoError(t, err)

	var last string
	for range 55 {
		res, runErr := svc.Run(context.Background(), client, nil, func(float64) {})
		require.NoError(t, runErr)
		last = res.Results[0].BackupID
	}
	require.Len(t, repo.backups, 50)
	require.Equal(t, last, repo.backups[len(repo.backups)-1].ID, "the newest attempts are the ones kept")
}

// TestOutputShapedLikeCiphertextIsRefused: the keyring passes such input
// through unencrypted, so storing it would store a configuration in the clear.
func TestOutputShapedLikeCiphertextIsRefused(t *testing.T) {
	t.Parallel()
	repo := newMemRepo(target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": answer(ctPrefix + "hostname core\n"),
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixCipher{}, fetch)
	require.NoError(t, err)

	res, err := svc.Run(context.Background(), client, nil, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StatusCommandFailed, res.Results[0].Status)
	require.Empty(t, repo.backups[0].ConfigCT)
}
