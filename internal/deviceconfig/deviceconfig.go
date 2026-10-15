// Package deviceconfig retrieves and keeps network device configurations
// (P-D1, #348): log in to a switch or router over SSH with a credential from
// the vault, run the platform's "show the running configuration" command, or
// read the running datastore over NETCONF, and store what comes back.
//
// A run backs up many devices and reports each one. One device refusing the
// credential, changing its host key or being unreachable is that device's
// result, not the run's failure; the run fails only when it cannot record
// results at all.
//
// Host keys are never trusted on first use. A target with no pinned key is
// contacted only far enough to learn the key it presents: the handshake stops
// before authentication, so the password is not sent, and the key is recorded
// as pending (host_key_unconfirmed) until an operator confirms it. A pinned
// target that presents a different key is refused as host_key_mismatch: a
// backup job that silently re-trusted a new key would hand the device password
// to whoever answered on that address.
//
// Configurations hold secrets of their own (enable hashes, SNMP communities,
// RADIUS keys), so they are stored encrypted with the credential vault's key.
//
// Each stored configuration is compared with the target's previous one (P-D2).
// A change is stored and raises a config.change alert; an identical
// configuration is neither stored again nor alerted on. The alert carries line
// counts, never configuration text: alerts leave Seed by webhook, mail and
// syslog, and the diff itself is read from the API by an operator.
//
// This is not internal/config/backups, which is Seed backing up its own
// configuration.
package deviceconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// Platform names how a device's configuration is read: the CLI dialect, which
// decides the command run, or NETCONF.
type Platform string

// Supported platforms. The CLI platforms answer their configuration command on
// an SSH exec channel, which is never paged, so no "terminal length 0" dance is
// needed. PlatformNETCONF is any vendor's NETCONF server: the configuration is
// the running datastore read with <get-config>.
const (
	PlatformCiscoIOS     Platform = "cisco-ios"
	PlatformCiscoNXOS    Platform = "cisco-nxos"
	PlatformAristaEOS    Platform = "arista-eos"
	PlatformJuniperJunos Platform = "juniper-junos"
	PlatformNETCONF      Platform = "netconf"
)

// Supported reports whether p is one of the platforms above.
func (p Platform) Supported() bool {
	switch p {
	case PlatformCiscoIOS, PlatformCiscoNXOS, PlatformAristaEOS, PlatformJuniperJunos, PlatformNETCONF:
		return true
	}
	return false
}

// Command returns a CLI platform's configuration command, or false for
// PlatformNETCONF and unsupported platforms.
func (p Platform) Command() (string, bool) {
	switch p {
	case PlatformCiscoIOS, PlatformCiscoNXOS, PlatformAristaEOS:
		return "show running-config", true
	case PlatformJuniperJunos:
		return "show configuration", true
	case PlatformNETCONF:
		return "", false
	}
	return "", false
}

// Well-known ports: SSH, and NETCONF over SSH (RFC 6242).
const (
	sshPort     = 22
	netconfPort = 830
)

// DefaultPort is the port a target uses when none is given.
func (p Platform) DefaultPort() int {
	if p == PlatformNETCONF {
		return netconfPort
	}
	return sshPort
}

// Status is the outcome of one backup attempt.
type Status string

// Attempt outcomes. Everything but StatusOK carries an error message.
const (
	StatusOK              Status = "ok"
	StatusAuthFailed      Status = "auth_failed"
	StatusHostKeyMismatch Status = "host_key_mismatch"
	// StatusHostKeyUnconfirmed: the target has no pinned key; the presented
	// key is pending until an operator confirms it, and nothing was sent.
	StatusHostKeyUnconfirmed Status = "host_key_unconfirmed"
	StatusUnreachable        Status = "unreachable"
	StatusCommandFailed      Status = "command_failed"
	StatusError              Status = "error"
)

const (
	// deviceTimeout bounds one device's login and command. A large chassis
	// configuration takes seconds; a minute means the device is not answering.
	deviceTimeout = 60 * time.Second
	// runConcurrency is how many devices one run logs in to at once. Small on
	// purpose: AAA servers rate-limit, and a backup is not urgent.
	runConcurrency = 4
	// backupRetention is how many attempts, and separately how many stored
	// configurations, a target keeps; older ones are pruned after each new
	// attempt is recorded, so an outage cannot push out the history.
	backupRetention = 50
	// diffContext is the unchanged lines shown around each change.
	diffContext = 3
)

// Fetch failure classes. A Fetcher wraps its errors in one of these so the
// service can name the outcome without knowing the transport.
var (
	ErrAuth            = errors.New("authentication failed")
	ErrHostKeyMismatch = errors.New("host key does not match the pinned key")
	ErrUnreachable     = errors.New("device unreachable")
	ErrCommand         = errors.New("configuration command failed")
)

// HostKeyUnconfirmedError is a Fetcher's answer for a target with no pinned
// key: the device presented Fingerprint and the login went no further.
type HostKeyUnconfirmedError struct {
	Fingerprint string
}

func (e *HostKeyUnconfirmedError) Error() string {
	return "host key " + e.Fingerprint + " is not confirmed"
}

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
	ID            string   `json:"id"`
	ClientID      string   `json:"clientId"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Platform      Platform `json:"platform"`
	CredentialsID string   `json:"credentialsId"`
	HostKeySHA256 string   `json:"hostKeySha256,omitempty"`
	// PendingHostKeySHA256 is the key the device last presented while nothing
	// was pinned, waiting for an operator to confirm it.
	PendingHostKeySHA256 string    `json:"pendingHostKeySha256,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// Backup is one attempt to back up a target. Config is set only when a single
// backup is read; lists leave it out. ConfigCT is the stored ciphertext and
// never leaves the service.
type Backup struct {
	ID       string    `json:"id"`
	ClientID string    `json:"clientId"`
	TargetID string    `json:"targetId"`
	TakenAt  time.Time `json:"takenAt"`
	Status   Status    `json:"status"`
	Error    string    `json:"error,omitempty"`
	SHA256   string    `json:"sha256,omitempty"`
	Config   string    `json:"config,omitempty"`
	ConfigCT string    `json:"-"`
}

// Repository is the persistence surface the service needs.
type Repository interface {
	TargetStore
	BackupStore
}

// TargetStore keeps backup targets and their host keys.
type TargetStore interface {
	ListTargets(ctx context.Context, clientID string) ([]Target, error)
	GetTarget(ctx context.Context, clientID, id string) (Target, error)
	// SaveTarget creates the target when ID is blank (filling in ID) and
	// otherwise replaces it, host key included.
	SaveTarget(ctx context.Context, t *Target) error
	DeleteTarget(ctx context.Context, clientID, id string) error
	// SetPendingHostKey records the key an unpinned target presented. It
	// does nothing once a key is pinned.
	SetPendingHostKey(ctx context.Context, clientID, id, fingerprint string) error
	// ConfirmHostKey pins the pending key, provided it is still fingerprint.
	ConfirmHostKey(ctx context.Context, clientID, id, fingerprint string) error
	// ClearHostKey forgets both the pinned and the pending key.
	ClearHostKey(ctx context.Context, clientID, id string) error
}

// BackupStore keeps backup attempts.
type BackupStore interface {
	// InsertBackup records an attempt; ConfigCT is the configuration's
	// ciphertext.
	InsertBackup(ctx context.Context, b *Backup) error
	// PruneBackups keeps the target's newest keep attempts and its newest
	// keep successful ones.
	PruneBackups(ctx context.Context, clientID, targetID string, keep int) error
	// ListBackups returns a target's newest attempts first, without their
	// configurations.
	ListBackups(ctx context.Context, clientID, targetID string, limit int) ([]Backup, error)
	// GetBackup returns one attempt with ConfigCT set.
	GetBackup(ctx context.Context, clientID, id string) (Backup, error)
	// LatestConfig returns the target's newest successful attempt taken
	// before the given time, with ConfigCT set, or ErrNotFound.
	LatestConfig(ctx context.Context, clientID, targetID string, before time.Time) (Backup, error)
}

// AlertWriter raises the configuration change alert; satisfied by the alert
// write chain the pipelines emit through.
type AlertWriter interface {
	Create(ctx context.Context, alert *alerts.Alert) error
}

// CredentialStore reads vault rows; satisfied by the device-credential
// repository. Secrets come back as ciphertext.
type CredentialStore interface {
	Get(ctx context.Context, id, clientID string) (*polling.Credentials, error)
}

// Cipher decrypts vault passwords for a login and encrypts the configurations
// it retrieves; satisfied by config.Keyring.
type Cipher interface {
	EncryptValue(plaintext string) (string, error)
	DecryptValue(encrypted string) (string, error)
}

// FetchRequest is one device login.
type FetchRequest struct {
	Host     string
	Port     int
	User     string
	Password string
	// Command is the CLI command run on an exec channel. It is ignored when
	// NETCONF is set: the running datastore is read on the netconf subsystem.
	Command string
	NETCONF bool
	// PinnedHostKey is the expected SHA256 fingerprint. Empty means none is
	// pinned: the Fetcher must stop before authenticating and return a
	// *HostKeyUnconfirmedError naming the key the device presented.
	PinnedHostKey string
}

// FetchResult is what a successful login returned.
type FetchResult struct {
	Output string
}

// Fetcher logs in to one device and retrieves its configuration.
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

// Diff is a stored configuration compared with the one stored before it.
type Diff struct {
	TargetID         string    `json:"targetId"`
	BackupID         string    `json:"backupId"`
	TakenAt          time.Time `json:"takenAt"`
	PreviousBackupID string    `json:"previousBackupId"`
	PreviousTakenAt  time.Time `json:"previousTakenAt"`
	Added            int       `json:"added"`
	Removed          int       `json:"removed"`
	// Unified is the change in unified diff format.
	Unified string `json:"unified"`
}

// Service is the configuration backup use-case.
type Service struct {
	repo   Repository
	creds  CredentialStore
	cipher Cipher
	fetch  Fetcher
	alerts AlertWriter
	now    func() time.Time
}

// NewService builds the use-case. Every dependency is required.
func NewService(
	repo Repository, creds CredentialStore, cipher Cipher, fetch Fetcher, alertWriter AlertWriter,
) (*Service, error) {
	if repo == nil || creds == nil || cipher == nil || fetch == nil || alertWriter == nil {
		return nil, ErrUnavailable
	}
	return &Service{repo: repo, creds: creds, cipher: cipher, fetch: fetch, alerts: alertWriter, now: time.Now}, nil
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
// port drops its pinned and pending keys, since they belonged to the old
// address; anything else keeps them.
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
		t.Port = t.Platform.DefaultPort()
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
			t.PendingHostKeySHA256 = prev.PendingHostKeySHA256
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
	if !t.Platform.Supported() {
		return ValidationError{
			Msg: "platform must be one of cisco-ios, cisco-nxos, arista-eos, juniper-junos, netconf",
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

// ClearHostKey forgets a target's pinned key; the next backup records the
// key the device presents as pending, for an operator to confirm. This is the
// operator's answer to a legitimate key change, such as a replaced supervisor.
func (s *Service) ClearHostKey(ctx context.Context, clientID, id string) error {
	return s.repo.ClearHostKey(ctx, clientID, id)
}

// ConfirmHostKey pins the target's pending key. fingerprint is the key the
// operator was shown; it must still be the pending one, so a key that changed
// after the operator looked is not the key that gets trusted.
func (s *Service) ConfirmHostKey(ctx context.Context, clientID, id, fingerprint string) error {
	t, err := s.repo.GetTarget(ctx, clientID, id)
	if err != nil {
		return err
	}
	if t.PendingHostKeySHA256 == "" || t.PendingHostKeySHA256 != fingerprint {
		return ValidationError{Msg: "fingerprint does not match the target's pending host key"}
	}
	return s.repo.ConfirmHostKey(ctx, clientID, id, fingerprint)
}

// ListBackups returns a target's attempts, newest first, without the
// configurations.
func (s *Service) ListBackups(ctx context.Context, clientID, targetID string, limit int) ([]Backup, error) {
	return s.repo.ListBackups(ctx, clientID, targetID, limit)
}

// GetBackup returns one attempt with its configuration decrypted.
func (s *Service) GetBackup(ctx context.Context, clientID, id string) (Backup, error) {
	b, err := s.repo.GetBackup(ctx, clientID, id)
	if err != nil {
		return Backup{}, err
	}
	if b.ConfigCT != "" {
		if b.Config, err = s.cipher.DecryptValue(b.ConfigCT); err != nil {
			return Backup{}, fmt.Errorf("decrypt configuration %s: %w", id, err)
		}
		b.ConfigCT = ""
	}
	return b, nil
}

// Diff compares a stored configuration with the target's previous one.
// ErrNotFound covers both an unknown backup and the target's first
// configuration, which has nothing before it.
func (s *Service) Diff(ctx context.Context, clientID, id string) (Diff, error) {
	cur, err := s.GetBackup(ctx, clientID, id)
	if err != nil {
		return Diff{}, err
	}
	if cur.Status != StatusOK {
		return Diff{}, ValidationError{Msg: "the backup attempt failed and holds no configuration"}
	}
	prev, err := s.repo.LatestConfig(ctx, clientID, cur.TargetID, cur.TakenAt)
	if err != nil {
		return Diff{}, err
	}
	prevConfig, err := s.cipher.DecryptValue(prev.ConfigCT)
	if err != nil {
		return Diff{}, fmt.Errorf("decrypt configuration %s: %w", prev.ID, err)
	}
	d := diffConfigs(prev, prevConfig, cur, cur.Config)
	d.Unified, err = unifiedDiff(prev, prevConfig, cur, cur.Config)
	return d, err
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
// failure is in the result. A configuration identical to the target's latest
// stored one is not stored again: the result names the backup that already
// holds it. One that differs is stored and alerted on.
func (s *Service) backupOne(ctx context.Context, t Target) (DeviceResult, error) {
	b := Backup{ClientID: t.ClientID, TargetID: t.ID, TakenAt: s.now().UTC()}
	output, fetchErr := s.login(ctx, t)
	// A cancelled run records nothing for the devices it interrupted: the
	// attempt did not fail, it was abandoned.
	if err := ctx.Err(); err != nil {
		return DeviceResult{}, err
	}
	if fetchErr == nil {
		sum := sha256.Sum256([]byte(output))
		b.SHA256 = hex.EncodeToString(sum[:])
		b.ConfigCT, fetchErr = s.encryptConfig(output)
	}
	if fetchErr != nil {
		b.SHA256 = ""
		b.Status = statusOf(fetchErr)
		b.Error = fetchErr.Error()
	} else {
		b.Status = StatusOK
	}

	if unconfirmed, ok := errors.AsType[*HostKeyUnconfirmedError](fetchErr); ok {
		if err := s.repo.SetPendingHostKey(ctx, t.ClientID, t.ID, unconfirmed.Fingerprint); err != nil {
			return DeviceResult{}, fmt.Errorf("record pending host key for %s: %w", t.ID, err)
		}
	}

	var prev Backup
	if b.Status == StatusOK {
		latest, err := s.repo.LatestConfig(ctx, t.ClientID, t.ID, b.TakenAt)
		switch {
		case errors.Is(err, ErrNotFound):
		case err != nil:
			return DeviceResult{}, fmt.Errorf("read latest configuration of %s: %w", t.ID, err)
		case latest.SHA256 == b.SHA256:
			return DeviceResult{TargetID: t.ID, Name: t.Name, BackupID: latest.ID, Status: StatusOK}, nil
		default:
			prev = latest
		}
	}
	if err := s.repo.InsertBackup(ctx, &b); err != nil {
		return DeviceResult{}, fmt.Errorf("record backup of %s: %w", t.ID, err)
	}
	if prev.ID != "" {
		if err := s.alertChange(ctx, t, prev, b, output); err != nil {
			return DeviceResult{}, err
		}
	}
	if err := s.repo.PruneBackups(ctx, t.ClientID, t.ID, backupRetention); err != nil {
		return DeviceResult{}, fmt.Errorf("prune backups of %s: %w", t.ID, err)
	}
	return DeviceResult{TargetID: t.ID, Name: t.Name, BackupID: b.ID, Status: b.Status, Error: b.Error}, nil
}

// alertChange raises the config.change alert for cur, which replaced prev as
// the target's configuration.
func (s *Service) alertChange(ctx context.Context, t Target, prev, cur Backup, config string) error {
	prevConfig, err := s.cipher.DecryptValue(prev.ConfigCT)
	if err != nil {
		return fmt.Errorf("decrypt configuration %s: %w", prev.ID, err)
	}
	d := diffConfigs(prev, prevConfig, cur, config)
	evidence, err := json.Marshal(alerts.ConfigChangeEvidence{
		TargetID:         d.TargetID,
		BackupID:         d.BackupID,
		PreviousBackupID: d.PreviousBackupID,
		Added:            d.Added,
		Removed:          d.Removed,
	})
	if err != nil {
		return fmt.Errorf("encode configuration change of %s: %w", t.ID, err)
	}
	err = s.alerts.Create(ctx, &alerts.Alert{
		Type:     alerts.TypeSecurity,
		Severity: alerts.SeverityWarning,
		Title:    "Configuration changed on " + t.Name,
		Message: fmt.Sprintf("The running configuration of %s (%s) changed since %s: %d lines added, %d removed.",
			t.Name, t.Host, prev.TakenAt.UTC().Format(time.RFC3339), d.Added, d.Removed),
		Source:   t.ID,
		Rule:     alerts.RuleConfigChange,
		Metadata: string(evidence),
	})
	if err != nil {
		return fmt.Errorf("alert configuration change of %s: %w", t.ID, err)
	}
	return nil
}

// diffConfigs counts the lines added and removed between two stored
// configurations, without the unified text.
func diffConfigs(prev Backup, prevConfig string, cur Backup, curConfig string) Diff {
	d := Diff{
		TargetID:         cur.TargetID,
		BackupID:         cur.ID,
		TakenAt:          cur.TakenAt,
		PreviousBackupID: prev.ID,
		PreviousTakenAt:  prev.TakenAt,
	}
	m := difflib.NewMatcher(difflib.SplitLines(prevConfig), difflib.SplitLines(curConfig))
	for _, op := range m.GetOpCodes() {
		switch op.Tag {
		case 'r':
			d.Removed += op.I2 - op.I1
			d.Added += op.J2 - op.J1
		case 'd':
			d.Removed += op.I2 - op.I1
		case 'i':
			d.Added += op.J2 - op.J1
		}
	}
	return d
}

// unifiedDiff renders the change between two stored configurations, each side
// named by its backup id and the time it was taken.
func unifiedDiff(prev Backup, prevConfig string, cur Backup, curConfig string) (string, error) {
	text, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(prevConfig),
		B:        difflib.SplitLines(curConfig),
		FromFile: prev.ID,
		FromDate: prev.TakenAt.UTC().Format(time.RFC3339),
		ToFile:   cur.ID,
		ToDate:   cur.TakenAt.UTC().Format(time.RFC3339),
		Context:  diffContext,
	})
	if err != nil {
		return "", fmt.Errorf("render configuration diff: %w", err)
	}
	return text, nil
}

// encryptConfig encrypts a configuration for storage. The keyring passes
// through anything already shaped like its ciphertext, so a device whose
// output begins that way would be stored as sent; that output is refused.
func (s *Service) encryptConfig(output string) (string, error) {
	ct, err := s.cipher.EncryptValue(output)
	if err != nil {
		return "", fmt.Errorf("encrypt configuration: %w", err)
	}
	if ct == output {
		return "", fmt.Errorf("%w: the output is not a configuration", ErrCommand)
	}
	return ct, nil
}

func (s *Service) login(ctx context.Context, t Target) (string, error) {
	cred, err := s.creds.Get(ctx, t.CredentialsID, t.ClientID)
	if err != nil {
		return "", fmt.Errorf("read credential: %w", err)
	}
	// The target was validated against an SSH credential, but the credential
	// can be edited into another kind afterwards.
	if cred.Kind != polling.CredentialKindSSH {
		return "", fmt.Errorf("credential %s is %s, not SSH", cred.ID, cred.Kind)
	}
	password, err := s.cipher.DecryptValue(cred.SSHPasswordCT)
	if err != nil {
		return "", fmt.Errorf("decrypt SSH password: %w", err)
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
		NETCONF:       t.Platform == PlatformNETCONF,
		PinnedHostKey: t.HostKeySHA256,
	})
	if err != nil {
		return "", err
	}
	// Devices answer in CRLF; storing LF keeps the history diffable whatever
	// the platform.
	output := strings.ReplaceAll(res.Output, "\r\n", "\n")
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("%w: the device returned no configuration", ErrCommand)
	}
	return output, nil
}

func statusOf(err error) Status {
	switch {
	case errors.Is(err, ErrAuth):
		return StatusAuthFailed
	case errors.Is(err, ErrHostKeyMismatch):
		return StatusHostKeyMismatch
	case errors.As(err, new(*HostKeyUnconfirmedError)):
		return StatusHostKeyUnconfirmed
	case errors.Is(err, ErrUnreachable):
		return StatusUnreachable
	case errors.Is(err, ErrCommand):
		return StatusCommandFailed
	}
	return StatusError
}
