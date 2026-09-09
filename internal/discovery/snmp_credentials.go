package discovery

// snmp_credentials.go answers "whose credentials does discovery use" (#2118).
//
// Polling knows: a target names the credential it references and the client
// that owns it. Discovery does not — it sweeps hosts it has never seen, and it
// runs as a singleton built at server init, before any request exists. Until
// this file there was no client to scope the vault lookup to, so discovery was
// the last consumer still reading plaintext community strings out of the file
// config (#1799).
//
// The rule here is deliberately narrow: discovery serves the deployment's one
// client and refuses to scan when there is more than one. Silently picking the
// default client would mean an MSP scanning every tenant's network with one
// tenant's community strings — the isolation failure the vault exists to
// prevent, reappearing in a new place. Refusing is honest about what
// per-tenant discovery would need (a per-scan profiler lifecycle, and an
// answer for scheduled scans that belong to no request) without committing to
// it before anyone decides multi-tenant discovery is a product goal.

import (
	"context"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// ErrDiscoveryTenantAmbiguous reports that the deployment has no single client
// for discovery to act as. It is returned before any network I/O: a sweep that
// cannot name whose credentials it is about to put on the wire does not run.
var ErrDiscoveryTenantAmbiguous = errors.New(
	"discovery has no single client to resolve credentials for (seed#2118)")

// ClientLister names the deployment's clients. The composition root adapts
// database.ClientRepository to it; discovery takes ids rather than client
// records because the id is all a credential lookup needs.
type ClientLister interface {
	ListClientIDs(ctx context.Context) ([]string, error)
}

// CredentialLister reads one client's stored device credentials. Implemented
// by database.DeviceCredentialRepository.
type CredentialLister interface {
	List(ctx context.Context, clientID string) ([]*polling.Credentials, error)
}

// SecretDecrypter turns versioned ciphertext back into plaintext. Implemented
// by config.Keyring, which owns the DEK.
type SecretDecrypter interface {
	DecryptValue(encrypted string) (string, error)
}

// SNMPCredentialProvider yields the credentials and transport settings one
// SNMP exchange should use. It is resolved per scan rather than held from
// startup so a credential added after the daemon booted is used by the next
// sweep.
type SNMPCredentialProvider interface {
	SNMPConfig(ctx context.Context) (*config.SNMPConfig, error)
}

// VaultSNMPCredentials resolves discovery's SNMP credentials from the
// encrypted vault.
type VaultSNMPCredentials struct {
	clients   ClientLister
	vault     CredentialLister
	decrypter SecretDecrypter
	transport config.SNMPConfig
}

// NewVaultSNMPCredentials builds the provider over the client list, the
// credential vault and the keyring.
//
// transport supplies timeout, retries, port and MaxRepetitions only; its
// Communities and V3Credentials are ignored, because the whole point of #1799
// is that those no longer reach the wire.
//
// Every dependency is required. A nil one would degrade discovery to
// unauthenticated SNMP, which either fails obscurely or succeeds against a
// device with a permissive default community that nobody authorised.
func NewVaultSNMPCredentials(
	clients ClientLister,
	vault CredentialLister,
	decrypter SecretDecrypter,
	transport *config.SNMPConfig,
) (*VaultSNMPCredentials, error) {
	switch {
	case clients == nil:
		return nil, fmt.Errorf("%w: nil client lister", ErrDiscoveryTenantAmbiguous)
	case vault == nil:
		return nil, fmt.Errorf("%w: nil credential vault", ErrDiscoveryTenantAmbiguous)
	case decrypter == nil:
		return nil, fmt.Errorf("%w: nil decrypter", ErrDiscoveryTenantAmbiguous)
	}

	v := &VaultSNMPCredentials{clients: clients, vault: vault, decrypter: decrypter}
	if transport != nil {
		v.transport = config.SNMPConfig{
			Timeout:        transport.Timeout,
			Retries:        transport.Retries,
			Port:           transport.Port,
			MaxRepetitions: transport.MaxRepetitions,
		}
	}
	return v, nil
}

// SNMPConfig resolves the client, reads its credentials and decrypts them.
//
// The returned config carries plaintext secrets and is built fresh per call,
// so it lives for one exchange and is never stored.
func (v *VaultSNMPCredentials) SNMPConfig(ctx context.Context) (*config.SNMPConfig, error) {
	clientID, err := v.clientID(ctx)
	if err != nil {
		return nil, err
	}

	stored, err := v.vault.List(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("list discovery credentials: %w", err)
	}

	out := v.transport
	for _, cred := range stored {
		if credErr := v.append(&out, cred); credErr != nil {
			return nil, credErr
		}
	}
	return &out, nil
}

// clientID returns the id of the deployment's single client.
func (v *VaultSNMPCredentials) clientID(ctx context.Context) (string, error) {
	ids, err := v.clients.ListClientIDs(ctx)
	if err != nil {
		return "", fmt.Errorf("list clients: %w", err)
	}
	if len(ids) != 1 {
		return "", fmt.Errorf("%w: %d clients exist", ErrDiscoveryTenantAmbiguous, len(ids))
	}
	return ids[0], nil
}

// append decrypts one stored credential onto the sweep list.
//
// Kind decides which columns are read rather than inferring from which ones
// are populated: the column is NOT NULL under a CHECK of ('v2c','v3'), so it
// is the row's own answer, and a v3 row whose user is still blank would
// otherwise be silently read as a v2c row with no community.
func (v *VaultSNMPCredentials) append(out *config.SNMPConfig, cred *polling.Credentials) error {
	if cred == nil {
		return nil
	}

	if cred.Kind == polling.CredentialKindV3 {
		authSecret, err := v.decrypt(cred.SNMPv3AuthCT)
		if err != nil {
			return fmt.Errorf("credential %s: v3 auth secret: %w", cred.ID, err)
		}
		privSecret, err := v.decrypt(cred.SNMPv3PrivCT)
		if err != nil {
			return fmt.Errorf("credential %s: v3 priv secret: %w", cred.ID, err)
		}
		out.V3Credentials = append(out.V3Credentials, config.SNMPv3Credential{
			Name:          cred.Name,
			Username:      cred.SNMPv3User,
			AuthProtocol:  cred.SNMPv3AuthProto,
			AuthPassword:  authSecret,
			PrivProtocol:  cred.SNMPv3PrivProto,
			PrivPassword:  privSecret,
			SecurityLevel: cred.SecurityLevel,
		})
		return nil
	}

	community, err := v.decrypt(cred.SNMPCommunityCT)
	if err != nil {
		return fmt.Errorf("credential %s: community: %w", cred.ID, err)
	}
	if community != "" {
		out.Communities = append(out.Communities, community)
	}
	return nil
}

// decrypt treats an empty column as absent rather than as ciphertext: a v2c
// credential has no v3 secrets and a v3 credential has no community.
func (v *VaultSNMPCredentials) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return v.decrypter.DecryptValue(ciphertext)
}
