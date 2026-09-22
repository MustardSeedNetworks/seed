package snmp

// credential_sweep.go collapses the credential-trying loop that every SNMP
// collector in this package repeated.
//
// Fourteen call sites carried the identical shape: try each v3 credential in
// turn, fall back to each v2c community, return the first success, and
// otherwise report that everything configured had been tried. Fourteen copies
// of a loop is fourteen chances for one of them to drift — to skip the v3 pass,
// to return the wrong error, or to keep trying after a context cancellation.
//
// It also puts the credential source behind one function. #1799 replaces
// Session's plaintext Communities and V3Credentials with the
// encrypted vault; with the sweep in one place, that becomes a change to what
// sweepCredentials iterates rather than an edit to every collector.

import (
	"context"
	"errors"
	"fmt"
)

// ErrNoCredentialSucceeded reports that every configured credential was tried
// and none produced a result. It deliberately does not carry the individual
// failures: they are per-credential authentication errors, and joining them
// risks a message that names which community strings exist.
var ErrNoCredentialSucceeded = errors.New("snmp: no configured credential succeeded")

// sweepCredentials tries v3 credentials then v2c communities, returning the
// first success.
//
// v3 goes first because it is the stronger of the two: a device that answers
// both should be talked to over the authenticated, optionally encrypted
// transport rather than a community string in cleartext on the wire.
//
// A cancelled context stops the sweep. Without that check a collector against
// an unreachable host works through every credential it has, turning one
// timeout into as many timeouts as there are credentials.
func sweepCredentials[T any](
	ctx context.Context,
	cfg *Session,
	what string,
	v3 func(cred *V3Credential) (T, error),
	v2c func(community string) (T, error),
) (T, error) {
	out, _, err := sweepCredentialsNaming(ctx, cfg, what, v3, v2c)
	return out, err
}

// sweepCredentialsNaming is the sweep itself, reporting which credential
// answered.
//
// Discovery promotes an SNMP-answering device to a polling target, and a
// target references a vault row rather than a secret (seed#2692): knowing that
// *some* credential worked is not enough to write one. Only the caller that
// needs the identity pays for it; sweepCredentials drops it.
func sweepCredentialsNaming[T any](
	ctx context.Context,
	cfg *Session,
	what string,
	v3 func(cred *V3Credential) (T, error),
	v2c func(community string) (T, error),
) (T, CredentialRef, error) {
	var zero T
	if cfg == nil {
		return zero, CredentialRef{}, errors.New("SNMP config is nil")
	}

	for i := range cfg.V3Credentials {
		if err := ctx.Err(); err != nil {
			return zero, CredentialRef{}, err
		}
		if out, err := v3(&cfg.V3Credentials[i]); err == nil {
			return out, CredentialRef{ID: cfg.V3Credentials[i].ID, Version: VersionV3}, nil
		}
	}

	for _, community := range cfg.Communities {
		if err := ctx.Err(); err != nil {
			return zero, CredentialRef{}, err
		}
		if out, err := v2c(community.String); err == nil {
			return out, CredentialRef{ID: community.ID, Version: VersionV2c}, nil
		}
	}

	return zero, CredentialRef{}, fmt.Errorf("%w: %s", ErrNoCredentialSucceeded, what)
}
