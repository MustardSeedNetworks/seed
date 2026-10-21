package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/foundation/pkg/passkey"
)

// TestWebAuthnOptionsMatchLibraryWire pins the published option types to what
// go-webauthn serializes under foundation's passkey policy: the browser must
// receive every member the library fills, so a member the transport types
// drop (a library upgrade, a policy change in foundation) fails here.
func TestWebAuthnOptionsMatchLibraryWire(t *testing.T) {
	service, err := passkey.New(passkey.Config{
		RPID: "seed.example.com", Origin: "https://seed.example.com:8443", DisplayName: "Seed",
	})
	require.NoError(t, err)
	user := &passkeyUser{handle: userHandleFromID(42), name: "admin"}

	creation, _, err := service.BeginRegistration(user)
	require.NoError(t, err)
	require.JSONEq(t,
		string(mustMarshal(t, creation)),
		string(mustMarshal(t, newWebAuthnCreationOptions(creation, user.handle))))

	assertion, _, err := service.BeginLogin()
	require.NoError(t, err)
	require.JSONEq(t,
		string(mustMarshal(t, assertion)),
		string(mustMarshal(t, newWebAuthnRequestOptions(assertion))))
}
