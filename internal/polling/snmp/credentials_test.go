package snmp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

type fakeCredStore struct {
	creds *polling.Credentials
	err   error
}

func (f *fakeCredStore) Get(context.Context, string) (*polling.Credentials, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.creds, nil
}

// fakeDecrypter reverses the "enc:" prefix convention used by the real keyring
// closely enough to prove the resolver passes ciphertext through and returns
// plaintext, without depending on a real DEK.
type fakeDecrypter struct{ err error }

func (f fakeDecrypter) DecryptValue(encrypted string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "plain(" + encrypted + ")", nil
}

func newResolver(t *testing.T, store snmp.CredentialStore, dec snmp.SecretDecrypter) *snmp.CredentialResolver {
	t.Helper()
	r, err := snmp.NewCredentialResolver(store, dec)
	if err != nil {
		t.Fatalf("NewCredentialResolver: %v", err)
	}
	return r
}

func TestResolveDecryptsStoredSecrets(t *testing.T) {
	t.Parallel()

	store := &fakeCredStore{creds: &polling.Credentials{
		ID:              "cred-1",
		SNMPCommunityCT: "enc:v1:community",
		SNMPv3User:      "operator",
		SNMPv3AuthCT:    "enc:v1:auth",
		SNMPv3PrivCT:    "enc:v1:priv",
		SNMPv3AuthProto: "SHA",
		SNMPv3PrivProto: "AES",
	}}

	got, err := newResolver(t, store, fakeDecrypter{}).
		Resolve(context.Background(), &polling.Target{ID: "t-1", CredentialsID: "cred-1"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got.SNMPCommunity != "plain(enc:v1:community)" {
		t.Errorf("SNMPCommunity = %q, want the decrypted value", got.SNMPCommunity)
	}
	if got.SNMPv3AuthSecret != "plain(enc:v1:auth)" {
		t.Errorf("SNMPv3AuthSecret = %q, want the decrypted value", got.SNMPv3AuthSecret)
	}
	if got.SNMPv3PrivSecret != "plain(enc:v1:priv)" {
		t.Errorf("SNMPv3PrivSecret = %q, want the decrypted value", got.SNMPv3PrivSecret)
	}
	if got.SNMPv3User != "operator" || got.SNMPv3AuthProto != "SHA" || got.SNMPv3PrivProto != "AES" {
		t.Errorf("non-secret fields not carried through: %+v", got)
	}
}

// TestResolveEmptyColumnsStayEmpty covers the mixed-version case: a v2c
// credential has no v3 secrets, and an empty column is absent rather than
// ciphertext, so it must not be handed to the decrypter.
func TestResolveEmptyColumnsStayEmpty(t *testing.T) {
	t.Parallel()

	store := &fakeCredStore{creds: &polling.Credentials{
		ID:              "cred-2",
		SNMPCommunityCT: "enc:v1:public",
	}}

	got, err := newResolver(t, store, fakeDecrypter{}).
		Resolve(context.Background(), &polling.Target{ID: "t-2", CredentialsID: "cred-2"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.SNMPv3AuthSecret != "" || got.SNMPv3PrivSecret != "" {
		t.Errorf("absent v3 secrets were decrypted: %+v", got)
	}
}

// TestResolveFailsClosed is the point of the whole type: every failure mode must
// return an error, never empty credentials. Returning ResolvedCredentials{} with
// a nil error is what let the poller run unauthenticated.
func TestResolveFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		store  snmp.CredentialStore
		dec    snmp.SecretDecrypter
		target *polling.Target
	}{
		{
			name:   "target references no credentials",
			store:  &fakeCredStore{creds: &polling.Credentials{}},
			dec:    fakeDecrypter{},
			target: &polling.Target{ID: "t-1"},
		},
		{
			name:   "credentials row missing",
			store:  &fakeCredStore{err: polling.ErrCredentialsNotFound},
			dec:    fakeDecrypter{},
			target: &polling.Target{ID: "t-1", CredentialsID: "gone"},
		},
		{
			name:   "decryption fails",
			store:  &fakeCredStore{creds: &polling.Credentials{SNMPCommunityCT: "enc:v1:x"}},
			dec:    fakeDecrypter{err: errors.New("bad key")},
			target: &polling.Target{ID: "t-1", CredentialsID: "cred-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newResolver(t, tt.store, tt.dec).Resolve(context.Background(), tt.target)
			if err == nil {
				t.Fatal("Resolve returned nil error; polling would proceed unauthenticated")
			}
			if !errors.Is(err, snmp.ErrCredentialsUnresolved) {
				t.Errorf("error = %v, want it to wrap ErrCredentialsUnresolved", err)
			}
			if got != (snmp.ResolvedCredentials{}) {
				t.Errorf("credentials leaked on failure: %+v", got)
			}
		})
	}
}

func TestNewCredentialResolverRejectsNilDependencies(t *testing.T) {
	t.Parallel()

	if _, err := snmp.NewCredentialResolver(nil, fakeDecrypter{}); err == nil {
		t.Error("nil store accepted")
	}
	if _, err := snmp.NewCredentialResolver(&fakeCredStore{}, nil); err == nil {
		t.Error("nil decrypter accepted")
	}
}
