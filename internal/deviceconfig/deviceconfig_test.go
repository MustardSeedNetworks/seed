package deviceconfig_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
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

func (r *memRepo) PinHostKey(_ context.Context, _, id, fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.targets[id]
	if t.HostKeySHA256 == "" {
		t.HostKeySHA256 = fingerprint
		r.targets[id] = t
	}
	return nil
}

func (r *memRepo) ClearHostKey(context.Context, string, string) error { return nil }

func (r *memRepo) InsertBackup(_ context.Context, b *deviceconfig.Backup) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b.ID = fmt.Sprintf("dcb-%d", len(r.backups))
	r.backups = append(r.backups, *b)
	return nil
}

func (r *memRepo) ListBackups(context.Context, string, string, int) ([]deviceconfig.Backup, error) {
	return nil, nil
}

func (r *memRepo) GetBackup(context.Context, string, string) (deviceconfig.Backup, error) {
	return deviceconfig.Backup{}, nil
}

type memCreds map[string]*polling.Credentials

func (m memCreds) Get(_ context.Context, id, clientID string) (*polling.Credentials, error) {
	c, ok := m[id]
	if !ok || c.ClientID != clientID {
		return nil, polling.ErrCredentialsNotFound
	}
	return c, nil
}

type prefixDecrypter struct{}

func (prefixDecrypter) DecryptValue(ct string) (string, error) {
	return ct[len("enc:v1:"):], nil
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

func target(id, host string, platform deviceconfig.Platform) deviceconfig.Target {
	return deviceconfig.Target{
		ID: id, ClientID: client, Name: id, Host: host, Port: 22, Platform: platform, CredentialsID: "cred-ssh",
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
		"10.0.0.1": func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return deviceconfig.FetchResult{Output: "hostname core\r\n!\r\n", HostKey: "SHA256:core"}, nil
		},
		"10.0.0.2": func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return deviceconfig.FetchResult{}, fmt.Errorf("%w: ssh: unable to authenticate", deviceconfig.ErrAuth)
		},
		"10.0.0.3": func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			return deviceconfig.FetchResult{}, fmt.Errorf("%w: connection refused", deviceconfig.ErrUnreachable)
		},
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixDecrypter{}, fetch)
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
			require.Equal(t, "hostname core\n!\n", b.Config, "CRLF is stored as LF")
			require.Len(t, b.SHA256, 64)
		} else {
			require.Empty(t, b.Config)
			require.NotEmpty(t, b.Error)
		}
	}
	require.Equal(t, "SHA256:core", repo.targets["dct-1"].HostKeySHA256, "first contact pins the key")
	require.Empty(t, repo.targets["dct-2"].HostKeySHA256, "a failed login pins nothing")

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
	pinned := target("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoNXOS)
	pinned.HostKeySHA256 = "SHA256:old"
	repo := newMemRepo(pinned)
	fetch := &scriptedFetcher{answers: map[string]func(deviceconfig.FetchRequest) (deviceconfig.FetchResult, error){
		"10.0.0.1": func(req deviceconfig.FetchRequest) (deviceconfig.FetchResult, error) {
			if req.PinnedHostKey != "SHA256:old" {
				return deviceconfig.FetchResult{}, errors.New("pin not passed")
			}
			return deviceconfig.FetchResult{Output: " \r\n", HostKey: "SHA256:old"}, nil
		},
	}}
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixDecrypter{}, fetch)
	require.NoError(t, err)

	res, err := svc.Run(context.Background(), client, []string{"dct-1"}, func(float64) {})
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StatusCommandFailed, res.Results[0].Status)
}

func TestRunRefusesAnUnknownTarget(t *testing.T) {
	t.Parallel()
	svc, err := deviceconfig.NewService(newMemRepo(), sshCreds(), prefixDecrypter{}, &scriptedFetcher{})
	require.NoError(t, err)

	_, err = svc.Run(context.Background(), client, []string{"dct-404"}, func(float64) {})
	var ve deviceconfig.ValidationError
	require.ErrorAs(t, err, &ve)
}

func TestSaveTargetValidates(t *testing.T) {
	t.Parallel()
	svc, err := deviceconfig.NewService(newMemRepo(), sshCreds(), prefixDecrypter{}, &scriptedFetcher{})
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
	pinned.HostKeySHA256 = "SHA256:core"
	repo := newMemRepo(pinned)
	svc, err := deviceconfig.NewService(repo, sshCreds(), prefixDecrypter{}, &scriptedFetcher{})
	require.NoError(t, err)

	in := deviceconfig.TargetInput{
		ID: "dct-1", ClientID: client, Name: "renamed", Host: "10.0.0.1", Port: 22,
		Platform: deviceconfig.PlatformCiscoIOS, CredentialsID: "cred-ssh",
	}
	got, err := svc.SaveTarget(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, "SHA256:core", got.HostKeySHA256)

	in.Host = "10.0.0.9"
	got, err = svc.SaveTarget(context.Background(), in)
	require.NoError(t, err)
	require.Empty(t, got.HostKeySHA256, "a new address has a new key to learn")
}
