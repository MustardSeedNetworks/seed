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
// config.SNMPConfig's plaintext Communities and V3Credentials with the
// encrypted vault; with the sweep in one place, that becomes a change to what
// sweepCredentials iterates rather than an edit to every collector.

import (
	"context"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
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
	cfg *config.SNMPConfig,
	what string,
	v3 func(cred *config.SNMPv3Credential) (T, error),
	v2c func(community string) (T, error),
) (T, error) {
	var zero T
	if cfg == nil {
		return zero, errors.New("SNMP config is nil")
	}

	for i := range cfg.V3Credentials {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		if out, err := v3(&cfg.V3Credentials[i]); err == nil {
			return out, nil
		}
	}

	for _, community := range cfg.Communities {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		if out, err := v2c(community); err == nil {
			return out, nil
		}
	}

	return zero, fmt.Errorf("%w: %s", ErrNoCredentialSucceeded, what)
}
