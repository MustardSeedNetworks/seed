package discovery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

type fakeClients struct {
	ids []string
	err error
}

func (f fakeClients) ListClientIDs(context.Context) ([]string, error) { return f.ids, f.err }

type fakeVault struct {
	byClient map[string][]*polling.Credentials
	err      error
}

func (f fakeVault) List(_ context.Context, clientID string) ([]*polling.Credentials, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byClient[clientID], nil
}

// fakeDecrypter reverses the "enc:" prefix the fixtures use, so a test that
// forgets to decrypt shows up as a literal "enc:..." on the sweep list rather
// than passing.
type fakeDecrypter struct{ err error }

func (f fakeDecrypter) DecryptValue(encrypted string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "plain-" + encrypted, nil
}

func newProvider(t *testing.T, clients fakeClients, vault fakeVault) *discovery.VaultSNMPCredentials {
	t.Helper()
	p, err := discovery.NewVaultSNMPCredentials(clients, vault, fakeDecrypter{}, &config.SNMPConfig{
		Timeout: 3 * time.Second, Retries: 2, Port: 161, MaxRepetitions: 10,
	})
	require.NoError(t, err)
	return p
}

func TestVaultSNMPCredentialsResolvesTheOneClient(t *testing.T) {
	p := newProvider(t,
		fakeClients{ids: []string{"acme"}},
		fakeVault{byClient: map[string][]*polling.Credentials{
			"acme": {
				{ID: "c1", Name: "v2c", SNMPCommunityCT: "community"},
				{
					ID: "c2", Name: "v3", SNMPv3User: "operator",
					SNMPv3AuthCT: "auth", SNMPv3PrivCT: "priv",
					SNMPv3AuthProto: "SHA256", SNMPv3PrivProto: "AES256",
					SecurityLevel: "authPriv",
				},
			},
		}},
	)

	got, err := p.SNMPConfig(context.Background())
	require.NoError(t, err)

	require.Equal(t, []string{"plain-community"}, got.Communities)
	require.Len(t, got.V3Credentials, 1)
	require.Equal(t, config.SNMPv3Credential{
		Name:          "v3",
		Username:      "operator",
		AuthProtocol:  "SHA256",
		AuthPassword:  "plain-auth",
		PrivProtocol:  "AES256",
		PrivPassword:  "plain-priv",
		SecurityLevel: "authPriv",
	}, got.V3Credentials[0])

	// Transport settings come from the file config; credentials never do.
	require.Equal(t, 3*time.Second, got.Timeout)
	require.Equal(t, 161, got.Port)
	require.Equal(t, uint32(10), got.MaxRepetitions)
}

func TestVaultSNMPCredentialsRefusesMoreThanOneClient(t *testing.T) {
	p := newProvider(t,
		fakeClients{ids: []string{"acme", "globex"}},
		fakeVault{byClient: map[string][]*polling.Credentials{
			"acme": {{ID: "c1", SNMPCommunityCT: "community"}},
		}},
	)

	_, err := p.SNMPConfig(context.Background())
	require.ErrorIs(t, err, discovery.ErrDiscoveryTenantAmbiguous)
	require.Contains(t, err.Error(), "2 clients exist")
}

func TestVaultSNMPCredentialsRefusesNoClient(t *testing.T) {
	p := newProvider(t, fakeClients{}, fakeVault{})

	_, err := p.SNMPConfig(context.Background())
	require.ErrorIs(t, err, discovery.ErrDiscoveryTenantAmbiguous)
}

// An empty vault is not an error: the deployment simply has no SNMP credential
// yet. It must not produce a community either — the caller skips the probe.
func TestVaultSNMPCredentialsEmptyVaultYieldsNoCredential(t *testing.T) {
	p := newProvider(t, fakeClients{ids: []string{"acme"}}, fakeVault{})

	got, err := p.SNMPConfig(context.Background())
	require.NoError(t, err)
	require.Empty(t, got.Communities)
	require.Empty(t, got.V3Credentials)
}

func TestVaultSNMPCredentialsPropagatesDecryptFailure(t *testing.T) {
	p, err := discovery.NewVaultSNMPCredentials(
		fakeClients{ids: []string{"acme"}},
		fakeVault{byClient: map[string][]*polling.Credentials{
			"acme": {{ID: "c1", SNMPCommunityCT: "community"}},
		}},
		fakeDecrypter{err: errors.New("wrong key version")},
		&config.SNMPConfig{},
	)
	require.NoError(t, err)

	_, err = p.SNMPConfig(context.Background())
	require.ErrorContains(t, err, "credential c1: community")
}

func TestNewVaultSNMPCredentialsRequiresEveryDependency(t *testing.T) {
	clients := fakeClients{ids: []string{"acme"}}
	vault := fakeVault{}

	for name, build := range map[string]func() error{
		"no clients": func() error {
			_, err := discovery.NewVaultSNMPCredentials(nil, vault, fakeDecrypter{}, nil)
			return err
		},
		"no vault": func() error {
			_, err := discovery.NewVaultSNMPCredentials(clients, nil, fakeDecrypter{}, nil)
			return err
		},
		"no decrypter": func() error {
			_, err := discovery.NewVaultSNMPCredentials(clients, vault, nil, nil)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, build(), discovery.ErrDiscoveryTenantAmbiguous)
		})
	}
}
