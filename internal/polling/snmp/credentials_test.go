package snmp_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// The plaintext values the fake decryptor yields. Every error-path test
// asserts these never appear in a log line or an error string.
const (
	secretCommunity = "s3cr3t-community"
	secretAuth      = "s3cr3t-auth-pass"
	secretPriv      = "s3cr3t-priv-pass"
)

// fakeCredentialStore serves one credential row keyed by id.
type fakeCredentialStore struct {
	byID map[string]*polling.Credential
	err  error
}

func (f *fakeCredentialStore) Get(_ context.Context, id, clientID string) (*polling.Credential, error) {
	if f.err != nil {
		return nil, f.err
	}
	c, ok := f.byID[id]
	if !ok || c.ClientID != clientID {
		return nil, polling.ErrCredentialNotFound
	}
	return c, nil
}

// fakeDecryptor maps ciphertext to plaintext; unknown input is an error,
// mirroring Config.DecryptSNMPPassword's rejection of non-DEK values.
type fakeDecryptor struct{ plain map[string]string }

func (f *fakeDecryptor) DecryptSNMPPassword(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	p, ok := f.plain[encrypted]
	if !ok {
		return "", errors.New("invalid ciphertext: authentication failed")
	}
	return p, nil
}

func testDecryptor() *fakeDecryptor {
	return &fakeDecryptor{plain: map[string]string{
		"enc:v1:community": secretCommunity,
		"enc:v1:auth":      secretAuth,
		"enc:v1:priv":      secretPriv,
	}}
}

func fullCredential() *polling.Credential {
	return &polling.Credential{
		ID:           "cred-1",
		ClientID:     "default",
		Name:         "lab-v3",
		CommunityEnc: "enc:v1:community",
		V3User:       "snmpuser",
		V3AuthEnc:    "enc:v1:auth",
		V3PrivEnc:    "enc:v1:priv",
		V3AuthProto:  "SHA",
		V3PrivProto:  "AES",
	}
}

func TestResolver_DecryptsEveryEncryptedField(t *testing.T) {
	t.Parallel()
	store := &fakeCredentialStore{byID: map[string]*polling.Credential{"cred-1": fullCredential()}}
	r := snmp.NewCredentialResolver(store, testDecryptor())

	got, err := r.Resolve(context.Background(), &polling.Target{
		ID: "t-1", ClientID: "default", CredentialsID: "cred-1",
	})
	require.NoError(t, err)
	require.Equal(t, snmp.ResolvedCredentials{
		SNMPCommunity:    secretCommunity,
		SNMPv3User:       "snmpuser",
		SNMPv3AuthSecret: secretAuth,
		SNMPv3PrivSecret: secretPriv,
		SNMPv3AuthProto:  "SHA",
		SNMPv3PrivProto:  "AES",
	}, got)
}

func TestResolver_NoCredentialsIDResolvesEmpty(t *testing.T) {
	t.Parallel()
	store := &fakeCredentialStore{byID: map[string]*polling.Credential{}}
	r := snmp.NewCredentialResolver(store, testDecryptor())

	got, err := r.Resolve(context.Background(), &polling.Target{ID: "t-1", ClientID: "default"})
	require.NoError(t, err, "a nil credential is legitimate — the FK is ON DELETE SET NULL")
	require.Equal(t, snmp.ResolvedCredentials{}, got)
}

func TestResolver_ErrorPaths(t *testing.T) {
	t.Parallel()
	badCipher := fullCredential()
	badCipher.CommunityEnc = "plaintext-community"

	otherClient := fullCredential()
	otherClient.ClientID = "other"

	tests := []struct {
		name    string
		store   *fakeCredentialStore
		wantErr error
	}{
		{
			name:    "row missing",
			store:   &fakeCredentialStore{byID: map[string]*polling.Credential{}},
			wantErr: polling.ErrCredentialNotFound,
		},
		{
			name:    "row belongs to another client",
			store:   &fakeCredentialStore{byID: map[string]*polling.Credential{"cred-1": otherClient}},
			wantErr: polling.ErrCredentialNotFound,
		},
		{
			name:    "store failure",
			store:   &fakeCredentialStore{err: errors.New("database is locked")},
			wantErr: nil,
		},
		{
			name:    "undecryptable value",
			store:   &fakeCredentialStore{byID: map[string]*polling.Credential{"cred-1": badCipher}},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := snmp.NewCredentialResolver(tt.store, testDecryptor())
			got, err := r.Resolve(context.Background(), &polling.Target{
				ID: "t-1", ClientID: "default", CredentialsID: "cred-1",
			})
			require.Error(t, err)
			require.Equal(t, snmp.ResolvedCredentials{}, got,
				"a failed resolution must not hand back partial credentials")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			requireNoSecrets(t, err.Error())
			require.Contains(t, err.Error(), "cred-1", "errors identify the credential by id")
		})
	}
}

// TestResolver_ErrorsAndLogsNeverCarrySecrets forces the decrypt-failure
// path with every field decryptable except the last, and asserts that
// neither the error nor anything the poller logs leaks the plaintext that
// was already recovered.
func TestResolver_ErrorsAndLogsNeverCarrySecrets(t *testing.T) {
	t.Parallel()
	cred := fullCredential()
	cred.V3PrivEnc = "enc:v1:unknown"

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	storage := &fakeStorage{targets: []*polling.Target{{
		ID: "t-1", ClientID: "default", Name: "router-1", IPAddress: "10.0.0.1",
		CredentialsID: "cred-1", Enabled: true, PollIntervalSec: 60,
		CollectorChain: []string{"sys_info"},
	}}}
	sched := newFakeScheduler()
	collector := &stubCollector{name: "sys_info"}
	store := &fakeCredentialStore{byID: map[string]*polling.Credential{"cred-1": cred}}

	p := snmp.NewPoller(storage, sched, snmp.NewCredentialResolver(store, testDecryptor()), logger)
	p.RegisterCollector(collector)
	require.NoError(t, p.Start(context.Background()))
	require.NoError(t, sched.firstJob().Run(context.Background()))

	require.Zero(t, collector.callCount(), "the chain must not run with partial credentials")
	requireNoSecrets(t, buf.String())

	storage.mu.Lock()
	updates := append([]updateRecord(nil), storage.updates...)
	storage.mu.Unlock()
	require.Len(t, updates, 1, "a resolution failure still records the poll outcome")
	require.Equal(t, "error", updates[0].status)
	requireNoSecrets(t, updates[0].errMsg)
	require.Contains(t, updates[0].errMsg, "cred-1")
}

// TestPoller_PassesResolvedCredentialsToCollectors is the behaviour the
// empty-credential stub broke: what the resolver returns is what the
// collector receives.
func TestPoller_PassesResolvedCredentialsToCollectors(t *testing.T) {
	t.Parallel()
	storage := &fakeStorage{targets: []*polling.Target{{
		ID: "t-1", ClientID: "default", Name: "router-1", IPAddress: "10.0.0.1",
		CredentialsID: "cred-1", Enabled: true, PollIntervalSec: 60,
		CollectorChain: []string{"sys_info"},
	}}}
	sched := newFakeScheduler()
	collector := &stubCollector{name: "sys_info"}
	store := &fakeCredentialStore{byID: map[string]*polling.Credential{"cred-1": fullCredential()}}

	p := snmp.NewPoller(storage, sched, snmp.NewCredentialResolver(store, testDecryptor()), silentLogger())
	p.RegisterCollector(collector)
	require.NoError(t, p.Start(context.Background()))
	require.NoError(t, sched.firstJob().Run(context.Background()))

	collector.mu.Lock()
	defer collector.mu.Unlock()
	require.Len(t, collector.creds, 1)
	require.Equal(t, secretCommunity, collector.creds[0].SNMPCommunity)
	require.Equal(t, secretAuth, collector.creds[0].SNMPv3AuthSecret)
	require.Equal(t, secretPriv, collector.creds[0].SNMPv3PrivSecret)
	require.Equal(t, "snmpuser", collector.creds[0].SNMPv3User)
}

// requireNoSecrets fails the test if any plaintext secret appears in s.
func requireNoSecrets(t *testing.T, s string) {
	t.Helper()
	for _, secret := range []string{secretCommunity, secretAuth, secretPriv} {
		require.NotContains(t, s, secret, "decrypted secret leaked into %q", s)
	}
}
