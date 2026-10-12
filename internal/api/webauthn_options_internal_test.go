package api

import (
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestWebAuthnOptionsMatchLibraryWire pins the published option types to what
// go-webauthn serializes for Seed's relying party: the browser must receive
// every member the library fills, so a member the transport types drop (a
// library upgrade, a new registration option) fails here.
func TestWebAuthnOptionsMatchLibraryWire(t *testing.T) {
	manager, err := auth.NewWebAuthnManager(auth.WebAuthnConfig{
		RPID: "seed.example.com", RPOrigins: []string{"https://seed.example.com:8443"},
	})
	require.NoError(t, err)
	user := &auth.WebAuthnUser{
		ID:   userHandleFromID(42),
		Name: "admin",
		Credentials: []webauthn.Credential{{
			ID:        []byte("enrolled-passkey"),
			Transport: []protocol.AuthenticatorTransport{protocol.USB, protocol.NFC},
		}},
	}

	creation, _, err := manager.BeginRegistration(user)
	require.NoError(t, err)
	require.JSONEq(t,
		string(mustMarshal(t, creation)),
		string(mustMarshal(t, newWebAuthnCreationOptions(creation, user.ID))))

	assertion, _, err := manager.BeginLogin(user)
	require.NoError(t, err)
	require.JSONEq(t,
		string(mustMarshal(t, assertion)),
		string(mustMarshal(t, newWebAuthnRequestOptions(assertion))))
}
