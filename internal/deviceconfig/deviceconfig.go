// Package deviceconfig retrieves and keeps network device configurations
// (P-D1, #348): log in to a switch or router over SSH with a credential from
// the vault, run the platform's "show the running configuration" command, and
// store what comes back.
//
// A run backs up many devices and reports each one. One device refusing the
// credential, changing its host key or being unreachable is that device's
// result, not the run's failure; the run fails only when it cannot record
// results at all.
//
// Host keys are pinned on first contact. A later login that presents a
// different key is refused and recorded as host_key_mismatch until an operator
// clears the pin: a backup job that silently re-trusted a new key would hand
// the device password to whoever answered on that address.
//
// This is not internal/config/backups, which is Seed backing up its own
// configuration.
package deviceconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// Platform names the device's CLI dialect, which decides the command run.
type Platform string

// Supported platforms. Each answers its configuration command on an SSH exec
// channel, which is never paged, so no "terminal length 0" dance is needed.
const (
	PlatformCiscoIOS     Platform = "cisco-ios"
	PlatformCiscoNXOS    Platform = "cisco-nxos"
	PlatformAristaEOS    Platform = "arista-eos"
	PlatformJuniperJunos Platform = "juniper-junos"
)

// Command returns the platform's configuration command, or false for an
// unsupported platform.
func (p Platform) Command() (string, bool) {
	switch p {
	case PlatformCiscoIOS, PlatformCiscoNXOS, PlatformAristaEOS:
		return "show running-config", true
	case PlatformJuniperJunos:
		return "show configuration", true
	}
	return "", false
}

// Status is the outcome of one backup attempt.
type Status string

// Attempt outcomes. Everything but StatusOK carries an error message.
const (
	StatusOK              Status = "ok"
	StatusAuthFailed      Status = "auth_failed"
	StatusHostKeyMismatch Status = "host_key_mismatch"
	StatusUnreachable     Status = "unreachable"
	StatusCommandFailed   Status = "command_failed"
	StatusError           Status = "error"
)

// DefaultPort is the SSH port a target uses when none is given.
const DefaultPort = 22

const (
	// deviceTimeout bounds one device's login and command. A large chassis
	// configuration takes seconds; a minute means the device is not answering.
	deviceTimeout = 60 * time.Second
	// runConcurrency is how many devices one run logs in to at once. Small on
	// purpose: AAA servers rate-limit, and a backup is not urgent.
	runConcurrency = 4
)

// Fetch failure classes. A Fetcher wraps its errors in one of these so the
// service can name the outcome without knowing the transport.
var (
	ErrAuth            = errors.New("authentication failed")
	ErrHostKeyMismatch = errors.New("host key does not match the pinned key")
	ErrUnreachable     = errors.New("device unreachable")
	ErrCommand         = errors.New("configuration command failed")
)

// Service errors the handler maps onto status codes.
var (
	ErrNotFound    = errors.New("deviceconfig: not found")
	ErrUnavailable = errors.New("deviceconfig: store unavailable")
)

// ValidationError carries a user-input message the handler maps to 400.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Target is a device to back up.
type Target struct {
	ID            string    `json:"id"`
	ClientID      string    `json:"clientId"`
	Name          string    `json:"name"`
	Host          string    `json:"host"`
	Port          int       `json:"port"`
	Platform      Platform  `json:"platform"`
	CredentialsID string    `json:"credentialsId"`
	HostKeySHA256 string    `json:"hostKeySha256,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Backup is one attempt to back up a target. Config is set only when a single
// backup is read; lists leave it out.
type Backup struct {
	ID       string    `json:"id"`
	ClientID string    `json:"clientId"`
	TargetID string    `json:"targetId"`
	TakenAt  time.Time `json:"takenAt"`
	Status   Status    `json:"status"`
	Error    string    `json:"error,omitempty"`
	SHA256   string    `json:"sha256,omitempty"`
	Config   string    `json:"config,omitempty"`
}

// Repository is the persistence surface the service needs.
type Repository interface {
	ListTargets(ctx context.Context, clientID string) ([]Target, error)
	GetTarget(ctx context.Context, clientID, id string) (Target, error)
	// SaveTarget creates the target when ID is blank (filling in ID) and
	// otherwise replaces it, host key included.
	SaveTarget(ctx context.Context, t *Target) error
	DeleteTarget(ctx context.Context, clientID, id string) error
	// PinHostKey records a fingerprint only where none is pinned, so two
	// runs racing on a new target cannot overwrite each other's pin.
	PinHostKey(ctx context.Context, clientID, id, fingerprint string) error
	ClearHostKey(ctx context.Context, clientID, id string) error
	InsertBackup(ctx context.Context, b *Backup) error
	ListBackups(ctx context.Context, clientID, targetID string, limit int) ([]Backup, error)
	GetBackup(ctx context.Context, clientID, id string) (Backup, error)
}

// CredentialStore reads vault rows; satisfied by the device-credential
// repository. Secrets come back as ciphertext.
type CredentialStore interface {
	Get(ctx context.Context, id, clientID string) (*polling.Credentials, error)
}

// Decrypter turns vault ciphertext into the plaintext a login needs; satisfied
// by config.Keyring.
type Decrypter interface {
	DecryptValue(encrypted string) (string, error)
}

// FetchRequest is one device login.
type FetchRequest struct {
	Host     string
	Port     int
	User     string
	Password string
	Command  string
	// PinnedHostKey is the expected SHA256 fingerprint; empty trusts and
	// reports whatever key the device presents.
	PinnedHostKey string
}

// FetchResult is what a successful login returned.
type FetchResult struct {
	Output  string
	HostKey string // SHA256 fingerprint the device presented
}

// Fetcher logs in to one device and runs one command.
type Fetcher interface {
	Fetch(ctx context.Context, req FetchRequest) (FetchResult, error)
}

// TargetInput is a target write request.
type TargetInput struct {
	ID            string
	ClientID      string
	Name          string
	Host          string
	Port          int
	Platform      Platform
	CredentialsID string
}

// DeviceResult is one device's outcome in a run.
type DeviceResult struct {
	TargetID string `json:"targetId"`
	Name     string `json:"name"`
	BackupID string `json:"backupId"`
	Status   Status `json:"status"`
	Error    string `json:"error,omitempty"`
}

// RunResult is a whole run: every device's outcome plus the tallies.
type RunResult struct {
	Results   []DeviceResult `json:"results"`
	Succeeded int            `json:"succeeded"`
	Failed    int            `json:"failed"`
}

// Service is the configuration backup use-case.
type Service struct {
	repo    Repository
	creds   CredentialStore
	decrypt Decrypter
	fetch   Fetcher
	now     func() time.Time
}

// NewService builds the use-case. Every dependency is required.
func NewService(repo Repository, creds CredentialStore, decrypt Decrypter, fetch Fetcher) (*Service, error) {
	if repo == nil || creds == nil || decrypt == nil || fetch == nil {
		return nil, ErrUnavailable
	}
	return &Service{repo: repo, creds: creds, decrypt: decrypt, fetch: fetch, now: time.Now}, nil
}

// ListTargets returns the client's targets.
func (s *Service) ListTargets(ctx context.Context, clientID string) ([]Target, error) {
	return s.repo.ListTargets(ctx, clientID)
}

// GetTarget returns one target.
func (s *Service) GetTarget(ctx context.Context, clientID, id string) (Target, error) {
	return s.repo.GetTarget(ctx, clientID, id)
}

// SaveTarget validates and writes a target. Moving a target to another host or
// port drops its pinned key, since the key belonged to the old address;
// anything else keeps it.
func (s *Service) SaveTarget(ctx context.Context, in TargetInput) (Target, error) {
	t := Target{
		ID:            in.ID,
		ClientID:      in.ClientID,
		Name:          strings.TrimSpace(in.Name),
		Host:          strings.TrimSpace(in.Host),
		Port:          in.Port,
		Platform:      in.Platform,
		CredentialsID: in.CredentialsID,
	}
	if t.Port == 0 {
		t.Port = DefaultPort
	}
	if err := s.validateTarget(ctx, t); err != nil {
		return Target{}, err
	}

	if t.ID != "" {
		prev, err := s.repo.GetTarget(ctx, t.ClientID, t.ID)
		if err != nil {
			return Target{}, err
		}
		if prev.Host == t.Host && prev.Port == t.Port {
			t.HostKeySHA256 = prev.HostKeySHA256
		}
		t.CreatedAt = prev.CreatedAt
	}
	if err := s.repo.SaveTarget(ctx, &t); err != nil {
		return Target{}, err
	}
	return s.repo.GetTarget(ctx, t.ClientID, t.ID)
}

func (s *Service) validateTarget(ctx context.Context, t Target) error {
	switch {
	case t.Name == "":
		return ValidationError{Msg: "name is required"}
	case !validation.IsValidHostOrIP(t.Host):
		return ValidationError{Msg: "host must be an IP address or a hostname"}
	case validation.ValidatePort(t.Port) != nil:
		return ValidationError{Msg: "port must be between 1 and 65535"}
	}
	if _, ok := t.Platform.Command(); !ok {
		return ValidationError{
			Msg: "platform must be one of cisco-ios, cisco-nxos, arista-eos, juniper-junos",
		}
	}
	cred, err := s.creds.Get(ctx, t.CredentialsID, t.ClientID)
	if errors.Is(err, polling.ErrCredentialsNotFound) {
		return ValidationError{Msg: "credentialsId does not name a stored credential"}
	}
	if err != nil {
		return err
	}
	if cred.Kind != polling.CredentialKindSSH {
		return ValidationError{Msg: "credentialsId must name an SSH credential"}
	}
	return nil
}

// DeleteTarget removes a target and its backup history.
func (s *Service) DeleteTarget(ctx context.Context, clientID, id string) error {
	return s.repo.DeleteTarget(ctx, clientID, id)
}

// ClearHostKey forgets a target's pinned key; the next backup pins whatever
// the device presents. This is the operator's answer to a legitimate key
// change, such as a replaced supervisor.
func (s *Service) ClearHostKey(ctx context.Context, clientID, id string) error {
	return s.repo.ClearHostKey(ctx, clientID, id)
}

// ListBackups returns a target's attempts, newest first, without the
// configurations.
func (s *Service) ListBackups(ctx context.Context, clientID, targetID string, limit int) ([]Backup, error) {
	return s.repo.ListBackups(ctx, clientID, targetID, limit)
}

// GetBackup returns one attempt with its configuration.
func (s *Service) GetBackup(ctx context.Context, clientID, id string) (Backup, error) {
	return s.repo.GetBackup(ctx, clientID, id)
}

// Run backs up the named targets, or every target of the client when ids is
// empty. report receives the fraction of devices done.
func (s *Service) Run(ctx context.Context, clientID string, ids []string, report func(float64)) (RunResult, error) {
	targets, err := s.runTargets(ctx, clientID, ids)
	if err != nil {
		return RunResult{}, err
	}

	results := make([]DeviceResult, len(targets))
	errs := make([]error, len(targets))
	sem := make(chan struct{}, runConcurrency)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		done int
	)
	for i, t := range targets {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i], errs[i] = s.backupOne(ctx, t)
			mu.Lock()
			done++
			report(float64(done) / float64(len(targets)))
			mu.Unlock()
		})
	}
	wg.Wait()

	if cerr := ctx.Err(); cerr != nil {
		return RunResult{}, cerr
	}
	if joined := errors.Join(errs...); joined != nil {
		return RunResult{}, joined
	}
	out := RunResult{Results: results}
	for _, r := range results {
		if r.Status == StatusOK {
			out.Succeeded++
		} else {
			out.Failed++
		}
	}
	return out, nil
}

func (s *Service) runTargets(ctx context.Context, clientID string, ids []string) ([]Target, error) {
	if len(ids) == 0 {
		return s.repo.ListTargets(ctx, clientID)
	}
	targets := make([]Target, 0, len(ids))
	for _, id := range ids {
		t, err := s.repo.GetTarget(ctx, clientID, id)
		if errors.Is(err, ErrNotFound) {
			return nil, ValidationError{Msg: fmt.Sprintf("no backup target %q", id)}
		}
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// backupOne attempts one device and records the attempt. The returned error is
// a cancelled run or a failure to record the attempt; the device's own
// failure is in the result.
func (s *Service) backupOne(ctx context.Context, t Target) (DeviceResult, error) {
	b := Backup{ClientID: t.ClientID, TargetID: t.ID, TakenAt: s.now().UTC()}
	output, fpr, fetchErr := s.login(ctx, t)
	if fetchErr != nil {
		b.Status = statusOf(fetchErr)
		b.Error = fetchErr.Error()
	} else {
		b.Status = StatusOK
		b.Config = output
		sum := sha256.Sum256([]byte(output))
		b.SHA256 = hex.EncodeToString(sum[:])
	}
	// A cancelled run records nothing for the devices it interrupted: the
	// attempt did not fail, it was abandoned.
	if err := ctx.Err(); err != nil {
		return DeviceResult{}, err
	}
	if fetchErr == nil && t.HostKeySHA256 == "" {
		if err := s.repo.PinHostKey(ctx, t.ClientID, t.ID, fpr); err != nil {
			return DeviceResult{}, fmt.Errorf("pin host key for %s: %w", t.ID, err)
		}
	}
	if err := s.repo.InsertBackup(ctx, &b); err != nil {
		return DeviceResult{}, fmt.Errorf("record backup of %s: %w", t.ID, err)
	}
	return DeviceResult{TargetID: t.ID, Name: t.Name, BackupID: b.ID, Status: b.Status, Error: b.Error}, nil
}

func (s *Service) login(ctx context.Context, t Target) (string, string, error) {
	cred, err := s.creds.Get(ctx, t.CredentialsID, t.ClientID)
	if err != nil {
		return "", "", fmt.Errorf("read credential: %w", err)
	}
	// The target was validated against an SSH credential, but the credential
	// can be edited into another kind afterwards.
	if cred.Kind != polling.CredentialKindSSH {
		return "", "", fmt.Errorf("credential %s is %s, not SSH", cred.ID, cred.Kind)
	}
	password, err := s.decrypt.DecryptValue(cred.SSHPasswordCT)
	if err != nil {
		return "", "", fmt.Errorf("decrypt SSH password: %w", err)
	}
	command, _ := t.Platform.Command()

	ctx, cancel := context.WithTimeout(ctx, deviceTimeout)
	defer cancel()
	res, err := s.fetch.Fetch(ctx, FetchRequest{
		Host:          t.Host,
		Port:          t.Port,
		User:          cred.SSHUser,
		Password:      password,
		Command:       command,
		PinnedHostKey: t.HostKeySHA256,
	})
	if err != nil {
		return "", "", err
	}
	// Devices answer in CRLF; storing LF keeps the history diffable whatever
	// the platform.
	output := strings.ReplaceAll(res.Output, "\r\n", "\n")
	if strings.TrimSpace(output) == "" {
		return "", "", fmt.Errorf("%w: the device returned no configuration", ErrCommand)
	}
	return output, res.HostKey, nil
}

func statusOf(err error) Status {
	switch {
	case errors.Is(err, ErrAuth):
		return StatusAuthFailed
	case errors.Is(err, ErrHostKeyMismatch):
		return StatusHostKeyMismatch
	case errors.Is(err, ErrUnreachable):
		return StatusUnreachable
	case errors.Is(err, ErrCommand):
		return StatusCommandFailed
	}
	return StatusError
}
