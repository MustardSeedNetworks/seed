// Package credentials is the device-credential CRUD use-case (ADR-0020): the
// /api/v1/device-credentials endpoints' application service over a narrow
// Repository port, so the transport layer depends on a use-case rather than
// reaching into the database.
//
// The secrets are write-only by construction. A caller submits plaintext, the
// service encrypts it with the keyring before it reaches storage, and nothing
// on the way back out carries a secret: polling.Credentials marks every
// ciphertext field `json:"-"`, so a handler cannot serialise one even by
// accident. That is the property #1799 exists to establish — settings used to
// store v2c communities in plaintext config and hand them back to clients.
package credentials

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

var (
	// ErrNotFound is returned when no credential matches the id.
	ErrNotFound = errors.New("credentials: not found")
	// ErrUnavailable is returned when the store is not wired (handler → 503).
	ErrUnavailable = errors.New("credentials: store unavailable")
	// ErrInUse is returned when a credential is still referenced by a polling
	// target (handler → 409). Deleting it would leave that target unable to
	// authenticate, which fails closed but silently.
	ErrInUse = errors.New("credentials: still referenced by a polling target")
)

// ValidationError carries a user-input message the handler maps to 400.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Repository is the persistence surface the use-case needs.
type Repository interface {
	List(ctx context.Context, clientID string) ([]*polling.Credentials, error)
	Get(ctx context.Context, id, clientID string) (*polling.Credentials, error)
	Upsert(ctx context.Context, c *polling.Credentials) error
	Delete(ctx context.Context, clientID, id string) error
}

// Encrypter turns plaintext into versioned ciphertext. Satisfied by
// config.Keyring, which owns the DEK; this package holds only the seam so
// plaintext never reaches storage.
type Encrypter interface {
	EncryptValue(plaintext string) (string, error)
}

// Input is a write request. Every secret is plaintext here and nowhere else:
// the service encrypts before handing anything to the repository.
type Input struct {
	ID              string
	ClientID        string
	Name            string
	Community       string // v2c
	V3User          string
	V3AuthSecret    string
	V3PrivSecret    string
	SNMPv3AuthProto string
	SNMPv3PrivProto string
}

// Service is the device-credential CRUD use-case.
type Service struct {
	repo    Repository
	encrypt Encrypter
}

// NewService builds the use-case. Both dependencies are required: a nil
// encrypter would persist plaintext, which is the defect this replaces.
func NewService(repo Repository, encrypt Encrypter) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("%w: nil repository", ErrUnavailable)
	}
	if encrypt == nil {
		return nil, fmt.Errorf("%w: nil encrypter", ErrUnavailable)
	}
	return &Service{repo: repo, encrypt: encrypt}, nil
}

// List returns every credential the client owns, secrets redacted by the
// domain type's json tags.
func (s *Service) List(ctx context.Context, clientID string) ([]*polling.Credentials, error) {
	return s.repo.List(ctx, clientID)
}

// Get returns one credential, mapping the repository's not-found.
func (s *Service) Get(ctx context.Context, clientID, id string) (*polling.Credentials, error) {
	c, err := s.repo.Get(ctx, id, clientID)
	return c, mapErr(err)
}

// Save encrypts the submitted secrets and writes the credential.
//
// A request carrying neither a community nor a v3 user is refused here rather
// than at the schema, so the caller gets a sentence instead of a constraint
// number. Which secrets are present is what names the kind, so the repository
// derives that; this only checks the request says something.
func (s *Service) Save(ctx context.Context, in Input) (*polling.Credentials, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ValidationError{Msg: "name is required"}
	}
	hasCommunity := strings.TrimSpace(in.Community) != ""
	hasUser := strings.TrimSpace(in.V3User) != ""
	switch {
	case hasCommunity && hasUser:
		return nil, ValidationError{
			Msg: "a credential is either v2c (community) or v3 (user), not both",
		}
	case !hasCommunity && !hasUser:
		return nil, ValidationError{
			Msg: "provide either a community string (v2c) or a user (v3)",
		}
	}

	c := &polling.Credentials{
		ID:              in.ID,
		ClientID:        in.ClientID,
		Name:            strings.TrimSpace(in.Name),
		SNMPv3User:      strings.TrimSpace(in.V3User),
		SNMPv3AuthProto: in.SNMPv3AuthProto,
		SNMPv3PrivProto: in.SNMPv3PrivProto,
	}

	for _, f := range []struct {
		plaintext string
		dst       *string
		what      string
	}{
		{in.Community, &c.SNMPCommunityCT, "community"},
		{in.V3AuthSecret, &c.SNMPv3AuthCT, "v3 auth secret"},
		{in.V3PrivSecret, &c.SNMPv3PrivCT, "v3 privacy secret"},
	} {
		if strings.TrimSpace(f.plaintext) == "" {
			continue
		}
		ct, err := s.encrypt.EncryptValue(f.plaintext)
		if err != nil {
			// The message deliberately names the field and not the value.
			return nil, fmt.Errorf("encrypt %s: %w", f.what, err)
		}
		*f.dst = ct
	}

	if err := s.repo.Upsert(ctx, c); err != nil {
		return nil, mapErr(err)
	}
	return s.repo.Get(ctx, c.ID, c.ClientID)
}

// Delete removes a credential, mapping the repository's in-use conflict.
func (s *Service) Delete(ctx context.Context, clientID, id string) error {
	return mapErr(s.repo.Delete(ctx, clientID, id))
}

func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, polling.ErrCredentialsNotFound):
		return ErrNotFound
	default:
		return err
	}
}
