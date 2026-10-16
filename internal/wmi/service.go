package wmi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// DefaultPort is the WinRM HTTPS listener's port.
const DefaultPort = 5986

const (
	// hostTimeout bounds one host's seven queries. Installed software is the
	// slow one, a few seconds on a busy server.
	hostTimeout = 2 * time.Minute
	// runConcurrency is how many hosts one run signs in to at once. Small on
	// purpose: domain controllers rate-limit NTLM, and inventory is not urgent.
	runConcurrency = 4
)

// Status is the outcome of one collection attempt.
type Status string

// Attempt outcomes. Everything but StatusOK carries an error message.
const (
	StatusOK           Status = "ok"
	StatusAuthFailed   Status = "auth_failed"
	StatusCertMismatch Status = "cert_mismatch"
	// StatusCertUnconfirmed: the target has no pinned certificate; the
	// presented one is pending until an operator confirms it, and no
	// credential was sent.
	StatusCertUnconfirmed Status = "cert_unconfirmed"
	StatusUnreachable     Status = "unreachable"
	// StatusQueryFailed: the host answered a query with a fault or with
	// something that is not WS-Management.
	StatusQueryFailed Status = "query_failed"
	StatusError       Status = "error"
)

// Service errors the handler maps onto status codes.
var (
	ErrNotFound    = errors.New("wmi: not found")
	ErrUnavailable = errors.New("wmi: store unavailable")
)

// ValidationError carries a user-input message the handler maps to 400.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Target is a Windows host to inventory, with the outcome of its last
// attempt. LastStatus is empty until the first attempt.
type Target struct {
	ID            string
	ClientID      string
	Name          string
	Host          string
	Port          int
	CredentialsID string
	// CertSHA256 is the listener certificate's Fingerprint an operator
	// confirmed; empty means none is pinned.
	CertSHA256 string
	// PendingCertSHA256 is the certificate the host last presented while
	// nothing was pinned, waiting for an operator to confirm it.
	PendingCertSHA256 string
	LastAttemptAt     time.Time
	LastStatus        Status
	LastError         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TargetInput is a target write request.
type TargetInput struct {
	ID            string
	ClientID      string
	Name          string
	Host          string
	Port          int
	CredentialsID string
}

// Attempt is one collection's outcome. Inventory is set exactly when Status is
// StatusOK.
type Attempt struct {
	ClientID  string
	TargetID  string
	At        time.Time
	Status    Status
	Error     string
	Inventory *Inventory
}

// StoredInventory is a target's last successful inventory and when it was
// collected.
type StoredInventory struct {
	TargetID    string
	CollectedAt time.Time
	Inventory   Inventory
}

// Repository is the persistence surface the service needs.
type Repository interface {
	ListTargets(ctx context.Context, clientID string) ([]Target, error)
	GetTarget(ctx context.Context, clientID, id string) (Target, error)
	// SaveTarget creates the target when ID is blank (filling in ID) and
	// otherwise replaces its definition and certificate pins.
	SaveTarget(ctx context.Context, t *Target) error
	// DeleteTarget removes the target and its inventory.
	DeleteTarget(ctx context.Context, clientID, id string) error
	// SetPendingCert records the certificate an unpinned target presented.
	// It does nothing once a certificate is pinned.
	SetPendingCert(ctx context.Context, clientID, id, fingerprint string) error
	// ConfirmCert pins the pending certificate, provided it is still
	// fingerprint.
	ConfirmCert(ctx context.Context, clientID, id, fingerprint string) error
	// ClearCert forgets both the pinned and the pending certificate.
	ClearCert(ctx context.Context, clientID, id string) error
	// RecordAttempt stores an attempt's outcome on its target and, for a
	// successful one, replaces the target's inventory in the same
	// transaction.
	RecordAttempt(ctx context.Context, a Attempt) error
	// GetInventory returns the target's stored inventory, or ErrNotFound
	// when no collection has succeeded.
	GetInventory(ctx context.Context, clientID, targetID string) (StoredInventory, error)
}

// CredentialStore reads vault rows; satisfied by the device-credential
// repository. Secrets come back as ciphertext.
type CredentialStore interface {
	Get(ctx context.Context, id, clientID string) (*polling.Credentials, error)
}

// Decrypter decrypts vault passwords; satisfied by config.Keyring.
type Decrypter interface {
	DecryptValue(encrypted string) (string, error)
}

// Collector reads one host's inventory.
type Collector interface {
	Collect(ctx context.Context, e Endpoint) (Inventory, error)
}

// WSManCollector collects over WS-Management: the production Collector.
type WSManCollector struct{}

// Collect signs in to e and reads its inventory.
func (WSManCollector) Collect(ctx context.Context, e Endpoint) (Inventory, error) {
	c := NewClient(e)
	defer c.Close()
	return c.Inventory(ctx)
}

// HostResult is one host's outcome in a run.
type HostResult struct {
	TargetID string `json:"targetId"`
	Name     string `json:"name"`
	Status   Status `json:"status"`
	Error    string `json:"error,omitempty"`
}

// RunResult is a whole run: every host's outcome plus the tallies.
type RunResult struct {
	Results   []HostResult `json:"results"`
	Succeeded int          `json:"succeeded"`
	Failed    int          `json:"failed"`
}

// Service is the Windows inventory use-case.
type Service struct {
	repo    Repository
	creds   CredentialStore
	decrypt Decrypter
	collect Collector
	now     func() time.Time
}

// NewService builds the use-case. Every dependency is required.
func NewService(repo Repository, creds CredentialStore, decrypt Decrypter, collect Collector) (*Service, error) {
	if repo == nil || creds == nil || decrypt == nil || collect == nil {
		return nil, ErrUnavailable
	}
	return &Service{repo: repo, creds: creds, decrypt: decrypt, collect: collect, now: time.Now}, nil
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
// port drops its pinned and pending certificates, since they belonged to the
// old address; anything else keeps them.
func (s *Service) SaveTarget(ctx context.Context, in TargetInput) (Target, error) {
	t := Target{
		ID:            in.ID,
		ClientID:      in.ClientID,
		Name:          strings.TrimSpace(in.Name),
		Host:          strings.TrimSpace(in.Host),
		Port:          in.Port,
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
			t.CertSHA256 = prev.CertSHA256
			t.PendingCertSHA256 = prev.PendingCertSHA256
		}
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
	cred, err := s.creds.Get(ctx, t.CredentialsID, t.ClientID)
	if errors.Is(err, polling.ErrCredentialsNotFound) {
		return ValidationError{Msg: "credentialsId does not name a stored credential"}
	}
	if err != nil {
		return err
	}
	if cred.Kind != polling.CredentialKindWinRM {
		return ValidationError{Msg: "credentialsId must name a WinRM credential"}
	}
	return nil
}

// DeleteTarget removes a target and its inventory.
func (s *Service) DeleteTarget(ctx context.Context, clientID, id string) error {
	return s.repo.DeleteTarget(ctx, clientID, id)
}

// ClearCert forgets a target's pinned certificate; the next collection
// records the one the host presents as pending, for an operator to confirm.
// This is the operator's answer to a planned certificate renewal.
func (s *Service) ClearCert(ctx context.Context, clientID, id string) error {
	return s.repo.ClearCert(ctx, clientID, id)
}

// ConfirmCert pins the target's pending certificate. fingerprint is the one
// the operator was shown; it must still be the pending one, so a certificate
// that changed after the operator looked is not the one that gets trusted.
func (s *Service) ConfirmCert(ctx context.Context, clientID, id, fingerprint string) error {
	t, err := s.repo.GetTarget(ctx, clientID, id)
	if err != nil {
		return err
	}
	if t.PendingCertSHA256 == "" || t.PendingCertSHA256 != fingerprint {
		return ValidationError{Msg: "fingerprint does not match the target's pending certificate"}
	}
	return s.repo.ConfirmCert(ctx, clientID, id, fingerprint)
}

// GetInventory returns a target's last successful inventory.
func (s *Service) GetInventory(ctx context.Context, clientID, targetID string) (StoredInventory, error) {
	if _, err := s.repo.GetTarget(ctx, clientID, targetID); err != nil {
		return StoredInventory{}, err
	}
	return s.repo.GetInventory(ctx, clientID, targetID)
}

// Run collects the named targets, or every target of the client when ids is
// empty. report receives the fraction of hosts done.
func (s *Service) Run(ctx context.Context, clientID string, ids []string, report func(float64)) (RunResult, error) {
	targets, err := s.runTargets(ctx, clientID, ids)
	if err != nil {
		return RunResult{}, err
	}

	results := make([]HostResult, len(targets))
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
			results[i], errs[i] = s.collectOne(ctx, t)
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
			return nil, ValidationError{Msg: fmt.Sprintf("no inventory target %q", id)}
		}
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// collectOne attempts one host and records the attempt. The returned error is
// a cancelled run or a failure to record the attempt; the host's own failure
// is in the result.
func (s *Service) collectOne(ctx context.Context, t Target) (HostResult, error) {
	inv, collectErr := s.signIn(ctx, t)
	// A cancelled run records nothing for the hosts it interrupted: the
	// attempt did not fail, it was abandoned.
	if err := ctx.Err(); err != nil {
		return HostResult{}, err
	}
	a := Attempt{ClientID: t.ClientID, TargetID: t.ID, At: s.now().UTC(), Status: StatusOK}
	if collectErr != nil {
		a.Status = statusOf(collectErr)
		a.Error = collectErr.Error()
	} else {
		a.Inventory = &inv
	}

	if unconfirmed, ok := errors.AsType[*CertUnconfirmedError](collectErr); ok {
		if err := s.repo.SetPendingCert(ctx, t.ClientID, t.ID, unconfirmed.Fingerprint); err != nil {
			return HostResult{}, fmt.Errorf("record pending certificate for %s: %w", t.ID, err)
		}
	}
	if err := s.repo.RecordAttempt(ctx, a); err != nil {
		return HostResult{}, fmt.Errorf("record inventory of %s: %w", t.ID, err)
	}
	return HostResult{TargetID: t.ID, Name: t.Name, Status: a.Status, Error: a.Error}, nil
}

func (s *Service) signIn(ctx context.Context, t Target) (Inventory, error) {
	cred, err := s.creds.Get(ctx, t.CredentialsID, t.ClientID)
	if err != nil {
		return Inventory{}, fmt.Errorf("read credential: %w", err)
	}
	// The target was validated against a WinRM credential, but the
	// credential can be edited into another kind afterwards.
	if cred.Kind != polling.CredentialKindWinRM {
		return Inventory{}, fmt.Errorf("credential %s is %s, not WinRM", cred.ID, cred.Kind)
	}
	password, err := s.decrypt.DecryptValue(cred.WinRMPasswordCT)
	if err != nil {
		return Inventory{}, fmt.Errorf("decrypt WinRM password: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, hostTimeout)
	defer cancel()
	return s.collect.Collect(ctx, Endpoint{
		Host:       t.Host,
		Port:       t.Port,
		User:       cred.WinRMUser,
		Password:   password,
		PinnedCert: t.CertSHA256,
	})
}

func statusOf(err error) Status {
	switch {
	case errors.Is(err, ErrAuth):
		return StatusAuthFailed
	case errors.Is(err, ErrCertMismatch):
		return StatusCertMismatch
	case errors.As(err, new(*CertUnconfirmedError)):
		return StatusCertUnconfirmed
	case errors.Is(err, ErrUnreachable):
		return StatusUnreachable
	case errors.Is(err, ErrFault), errors.Is(err, ErrProtocol):
		return StatusQueryFailed
	}
	return StatusError
}
