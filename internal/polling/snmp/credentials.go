package snmp

// credentials.go resolves a polling target's SNMP auth material at poll time:
// it reads the target's device_credentials row and decrypts the three secret
// columns with the credential DEK (ADR-0015). Decrypted material lives only in
// the ResolvedCredentials value handed to the collector chain — it is never
// logged, never wrapped into an error, and never persisted.

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// CredentialStore reads one device_credentials row. Scoping by clientID as
// well as id is deliberate: polling_targets.credentials_id only guarantees the
// row exists, not that it belongs to the target's client. A miss — including a
// cross-client read — returns [polling.ErrCredentialNotFound].
type CredentialStore interface {
	Get(ctx context.Context, id, clientID string) (*polling.Credential, error)
}

// Decryptor turns versioned DEK ciphertext back into a secret. Satisfied by
// [config.Config.DecryptSNMPPassword], which rejects plaintext and legacy
// unversioned ciphertext so credentials cannot be hand-edited into the store.
type Decryptor interface {
	DecryptSNMPPassword(encrypted string) (string, error)
}

// CredentialResolver resolves per-target credentials on every poll. There is
// no cache: a poll costs one indexed read plus at most three AES-GCM opens,
// and resolving fresh means a rotated credential takes effect on the next
// poll rather than at the next restart.
type CredentialResolver struct {
	store     CredentialStore
	decryptor Decryptor
}

// NewCredentialResolver builds a resolver over its store and decryptor.
func NewCredentialResolver(store CredentialStore, decryptor Decryptor) *CredentialResolver {
	return &CredentialResolver{store: store, decryptor: decryptor}
}

// Resolve returns the decrypted credentials for a target.
//
// A target with no credentials_id resolves to empty credentials and no error:
// the column is ON DELETE SET NULL, so an unset credential is a legitimate
// state (SNMP v1/v2c targets may also carry their community elsewhere). A
// non-empty id that misses, belongs to another client, or fails to decrypt is
// an error — polling on with a silently empty community would surface as an
// unexplained SNMP timeout instead of the real cause.
//
// Errors name the credential by id only. No decrypted value, and no
// ciphertext, ever reaches the returned error.
func (r *CredentialResolver) Resolve(
	ctx context.Context, target *polling.Target,
) (ResolvedCredentials, error) {
	if target.CredentialsID == "" {
		return ResolvedCredentials{}, nil
	}

	cred, err := r.store.Get(ctx, target.CredentialsID, target.ClientID)
	if err != nil {
		return ResolvedCredentials{}, fmt.Errorf(
			"resolve credential %q for target %q: %w", target.CredentialsID, target.ID, err)
	}

	community, err := r.decrypt(cred, "snmp_community_enc", cred.CommunityEnc)
	if err != nil {
		return ResolvedCredentials{}, err
	}
	auth, err := r.decrypt(cred, "snmp_v3_auth_enc", cred.V3AuthEnc)
	if err != nil {
		return ResolvedCredentials{}, err
	}
	priv, err := r.decrypt(cred, "snmp_v3_priv_enc", cred.V3PrivEnc)
	if err != nil {
		return ResolvedCredentials{}, err
	}

	return ResolvedCredentials{
		SNMPCommunity:    community,
		SNMPv3User:       cred.V3User,
		SNMPv3AuthSecret: auth,
		SNMPv3PrivSecret: priv,
		SNMPv3AuthProto:  cred.V3AuthProto,
		SNMPv3PrivProto:  cred.V3PrivProto,
	}, nil
}

// decrypt unwraps one encrypted column. The failure is reported by credential
// id and column name; the underlying decrypt error is deliberately dropped
// rather than wrapped, because it is the one error in this path whose text is
// derived from the credential material.
func (r *CredentialResolver) decrypt(cred *polling.Credential, column, encrypted string) (string, error) {
	plaintext, err := r.decryptor.DecryptSNMPPassword(encrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt %s of credential %q: value is not readable with the "+
			"current credential key; re-set it via the API/CLI", column, cred.ID)
	}
	return plaintext, nil
}
