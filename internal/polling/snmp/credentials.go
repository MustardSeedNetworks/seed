package snmp

import (
	"context"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// ErrCredentialsUnresolved is returned when a target's credentials cannot be
// produced. Polling must stop on it rather than fall back to empty
// credentials: an unauthenticated SNMP attempt either fails obscurely or, on a
// device with a permissive default community, succeeds against a device the
// operator never authorised.
var ErrCredentialsUnresolved = errors.New("snmp credentials unresolved")

// CredentialStore reads stored device credentials. Implemented by
// database.DeviceCredentialRepository.
type CredentialStore interface {
	Get(ctx context.Context, id string) (*polling.Credentials, error)
}

// SecretDecrypter turns versioned ciphertext back into plaintext. Implemented
// by config.Keyring, which owns the DEK; this package holds only the seam so
// plaintext never reaches storage or the domain types.
type SecretDecrypter interface {
	DecryptValue(encrypted string) (string, error)
}

// CredentialResolver produces the plaintext credentials for a target at poll
// time.
type CredentialResolver struct {
	store     CredentialStore
	decrypter SecretDecrypter
}

// NewCredentialResolver builds a resolver over a credential store and the
// keyring. Both are required — a nil dependency would silently degrade every
// poll to unauthenticated, which is the defect this type exists to remove.
func NewCredentialResolver(store CredentialStore, decrypter SecretDecrypter) (*CredentialResolver, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: nil credential store", ErrCredentialsUnresolved)
	}
	if decrypter == nil {
		return nil, fmt.Errorf("%w: nil decrypter", ErrCredentialsUnresolved)
	}
	return &CredentialResolver{store: store, decrypter: decrypter}, nil
}

// Resolve decrypts the credentials a target references.
//
// A target with no CredentialsID is a configuration error, not a request for
// anonymous polling: v1/v2c needs a community and v3 needs a user, so there is
// no version for which empty credentials are valid.
func (r *CredentialResolver) Resolve(ctx context.Context, target *polling.Target) (ResolvedCredentials, error) {
	if target.CredentialsID == "" {
		return ResolvedCredentials{}, fmt.Errorf(
			"%w: target %s references no credentials", ErrCredentialsUnresolved, target.ID)
	}

	stored, err := r.store.Get(ctx, target.CredentialsID)
	if err != nil {
		return ResolvedCredentials{}, fmt.Errorf("%w: %w", ErrCredentialsUnresolved, err)
	}

	community, err := r.decrypt(stored.SNMPCommunityCT)
	if err != nil {
		return ResolvedCredentials{}, fmt.Errorf("%w: community: %w", ErrCredentialsUnresolved, err)
	}
	authSecret, err := r.decrypt(stored.SNMPv3AuthCT)
	if err != nil {
		return ResolvedCredentials{}, fmt.Errorf("%w: v3 auth secret: %w", ErrCredentialsUnresolved, err)
	}
	privSecret, err := r.decrypt(stored.SNMPv3PrivCT)
	if err != nil {
		return ResolvedCredentials{}, fmt.Errorf("%w: v3 priv secret: %w", ErrCredentialsUnresolved, err)
	}

	return ResolvedCredentials{
		SNMPCommunity:    community,
		SNMPv3User:       stored.SNMPv3User,
		SNMPv3AuthSecret: authSecret,
		SNMPv3PrivSecret: privSecret,
		SNMPv3AuthProto:  stored.SNMPv3AuthProto,
		SNMPv3PrivProto:  stored.SNMPv3PrivProto,
	}, nil
}

// decrypt treats an empty column as absent rather than as ciphertext: a v2c
// credential has no v3 secrets and a v3 credential has no community.
func (r *CredentialResolver) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return r.decrypter.DecryptValue(ciphertext)
}
