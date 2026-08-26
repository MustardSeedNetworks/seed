package credentials_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/credentials"
)

type fakeRepo struct {
	saved   *polling.Credentials
	listErr error
}

func (f *fakeRepo) List(_ context.Context, clientID string) ([]*polling.Credentials, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.saved == nil || f.saved.ClientID != clientID {
		return nil, nil
	}
	return []*polling.Credentials{f.saved}, nil
}

func (f *fakeRepo) Get(_ context.Context, id, clientID string) (*polling.Credentials, error) {
	if f.saved == nil || f.saved.ID != id || f.saved.ClientID != clientID {
		return nil, polling.ErrCredentialsNotFound
	}
	return f.saved, nil
}

func (f *fakeRepo) Upsert(_ context.Context, c *polling.Credentials) error {
	copied := *c
	f.saved = &copied
	return nil
}

func (f *fakeRepo) Delete(context.Context, string, string) error { return nil }

// reverseEncrypter stands in for the keyring. It is deliberately not a no-op:
// a no-op would let a test pass while plaintext reached storage.
type reverseEncrypter struct{ fail bool }

func (e reverseEncrypter) EncryptValue(plaintext string) (string, error) {
	if e.fail {
		return "", errors.New("keyring unavailable")
	}
	return "enc:v1:" + plaintext, nil
}

func newService(t *testing.T, repo *fakeRepo) *credentials.Service {
	t.Helper()
	svc, err := credentials.NewService(repo, reverseEncrypter{})
	require.NoError(t, err)
	return svc
}

// TestSaveEncryptsBeforeStorage is the property this use-case exists for:
// plaintext must not reach the repository. Settings used to store v2c
// communities in plaintext config (#1799).
func TestSaveEncryptsBeforeStorage(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(t, repo)

	_, err := svc.Save(context.Background(), credentials.Input{
		ID: "c1", ClientID: "client-1", Name: "core switches", Community: "s3cret-community",
	})
	require.NoError(t, err)

	require.NotNil(t, repo.saved)
	require.Equal(t, "enc:v1:s3cret-community", repo.saved.SNMPCommunityCT)
	require.NotContains(t, repo.saved.SNMPCommunityCT, "s3cret-community\x00")
	require.Equal(t, "enc:v1:s3cret-community", repo.saved.SNMPCommunityCT,
		"the community reached storage without being encrypted")
}

// TestSavedCredentialSerialisesWithoutSecrets is the redaction half. The
// domain type's json tags are what stop a handler leaking ciphertext, so this
// asserts on the wire form rather than on the struct.
func TestSavedCredentialSerialisesWithoutSecrets(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(t, repo)

	got, err := svc.Save(context.Background(), credentials.Input{
		ID: "c1", ClientID: "client-1", Name: "v3 admin",
		V3User: "netops", V3AuthSecret: "auth-plain", V3PrivSecret: "priv-plain",
	})
	require.NoError(t, err)

	encoded, err := json.Marshal(got)
	require.NoError(t, err)
	body := string(encoded)

	for _, secret := range []string{"auth-plain", "priv-plain", "enc:v1:"} {
		require.NotContains(t, body, secret,
			"serialised credential leaked %q", secret)
	}
	// The non-secret identity of the credential must still be there, or the
	// UI cannot show a list.
	require.Contains(t, body, "v3 admin")
	require.Contains(t, body, "netops")
}

func TestSaveRejectsAmbiguousAndEmptyCredentials(t *testing.T) {
	svc := newService(t, &fakeRepo{})

	for _, tc := range []struct {
		name string
		in   credentials.Input
		want string
	}{
		{"both", credentials.Input{Name: "x", Community: "c", V3User: "u"}, "not both"},
		{"neither", credentials.Input{Name: "x"}, "either a community"},
		{"no name", credentials.Input{Community: "c"}, "name is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Save(context.Background(), tc.in)
			require.Error(t, err)
			var ve credentials.ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, strings.ToLower(ve.Msg), strings.ToLower(tc.want))
		})
	}
}

// TestSaveFailsClosedWhenTheKeyringIsUnavailable — storing the plaintext
// because encryption failed is the one outcome that must never happen.
func TestSaveFailsClosedWhenTheKeyringIsUnavailable(t *testing.T) {
	repo := &fakeRepo{}
	svc, err := credentials.NewService(repo, reverseEncrypter{fail: true})
	require.NoError(t, err)

	_, err = svc.Save(context.Background(), credentials.Input{
		ID: "c1", ClientID: "client-1", Name: "n", Community: "plaintext",
	})
	require.Error(t, err)
	require.Nil(t, repo.saved, "a failed encryption still wrote to storage")
	require.NotContains(t, err.Error(), "plaintext",
		"the error message carried the secret it failed to encrypt")
}

// TestNewServiceRequiresBothDependencies — a nil encrypter would silently
// persist plaintext.
func TestNewServiceRequiresBothDependencies(t *testing.T) {
	_, err := credentials.NewService(nil, reverseEncrypter{})
	require.ErrorIs(t, err, credentials.ErrUnavailable)

	_, err = credentials.NewService(&fakeRepo{}, nil)
	require.ErrorIs(t, err, credentials.ErrUnavailable)
}

func TestGetMapsNotFound(t *testing.T) {
	svc := newService(t, &fakeRepo{})
	_, err := svc.Get(context.Background(), "client-1", "missing")
	require.ErrorIs(t, err, credentials.ErrNotFound)
}
