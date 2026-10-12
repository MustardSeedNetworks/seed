package api

import (
	"encoding/base64"

	"github.com/go-webauthn/webauthn/protocol"
)

// The passkey ceremony options in the WebAuthn JSON serialization: every
// binary field is base64url without padding, and the browser decodes them to
// ArrayBuffers before calling navigator.credentials. go-webauthn builds the
// options; these transport types publish exactly the members it fills under
// Seed's relying-party configuration, so the schema carries no `any` user
// handle or open extension map. TestWebAuthnOptionsMatchLibraryWire fails if
// the library starts sending a member these types drop.

// WebAuthnCreationOptionsResponse is the body of POST
// /auth/webauthn/register/begin, passed to navigator.credentials.create().
type WebAuthnCreationOptionsResponse struct {
	PublicKey WebAuthnCreationOptions `json:"publicKey"`
}

// WebAuthnCreationOptions is PublicKeyCredentialCreationOptions.
type WebAuthnCreationOptions struct {
	RP                 WebAuthnRelyingParty           `json:"rp"`
	User               WebAuthnUserEntity             `json:"user"`
	Challenge          string                         `json:"challenge"`
	PubKeyCredParams   []WebAuthnCredentialParameter  `json:"pubKeyCredParams"`
	TimeoutMs          int                            `json:"timeout"`
	ExcludeCredentials []WebAuthnCredentialDescriptor `json:"excludeCredentials"`
}

// WebAuthnRelyingParty names the relying party; id is the RP ID (a hostname).
type WebAuthnRelyingParty struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// WebAuthnUserEntity is the account the credential is created for; id is the
// stable user handle, not the username.
type WebAuthnUserEntity struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
}

// WebAuthnCredentialParameter is one acceptable key type, by COSE algorithm.
type WebAuthnCredentialParameter struct {
	Type string `json:"type" jsonschema:"enum=public-key"`
	Alg  int    `json:"alg"`
}

// WebAuthnCredentialDescriptor names an enrolled credential.
type WebAuthnCredentialDescriptor struct {
	Type       string   `json:"type"                 jsonschema:"enum=public-key"`
	ID         string   `json:"id"`
	Transports []string `json:"transports,omitempty" jsonschema:"enum=usb,enum=nfc,enum=ble,enum=smart-card,enum=hybrid,enum=internal"`
}

// WebAuthnRequestOptionsResponse is the body of POST
// /auth/webauthn/login/begin, passed to navigator.credentials.get().
type WebAuthnRequestOptionsResponse struct {
	PublicKey WebAuthnRequestOptions `json:"publicKey"`
}

// WebAuthnRequestOptions is PublicKeyCredentialRequestOptions.
type WebAuthnRequestOptions struct {
	Challenge        string                         `json:"challenge"`
	TimeoutMs        int                            `json:"timeout"`
	RPID             string                         `json:"rpId"`
	AllowCredentials []WebAuthnCredentialDescriptor `json:"allowCredentials"`
}

// WebAuthnLoginBeginRequest is the body of POST /auth/webauthn/login/begin.
// It names the account because, unlike registration, nobody is signed in yet.
type WebAuthnLoginBeginRequest struct {
	Username string `json:"username" validate:"required"`
}

// newWebAuthnCreationOptions publishes go-webauthn's creation options. The
// user handle is passed in rather than read back from the library's untyped
// UserEntity.ID.
func newWebAuthnCreationOptions(c *protocol.CredentialCreation, userHandle []byte) WebAuthnCreationOptionsResponse {
	o := c.Response
	params := make([]WebAuthnCredentialParameter, 0, len(o.Parameters))
	for _, p := range o.Parameters {
		params = append(params, WebAuthnCredentialParameter{Type: string(p.Type), Alg: int(p.Algorithm)})
	}
	return WebAuthnCreationOptionsResponse{PublicKey: WebAuthnCreationOptions{
		RP: WebAuthnRelyingParty{Name: o.RelyingParty.Name, ID: o.RelyingParty.ID},
		User: WebAuthnUserEntity{
			Name:        o.User.Name,
			DisplayName: o.User.DisplayName,
			ID:          base64.RawURLEncoding.EncodeToString(userHandle),
		},
		Challenge:          o.Challenge.String(),
		PubKeyCredParams:   params,
		TimeoutMs:          o.Timeout,
		ExcludeCredentials: newWebAuthnDescriptors(o.CredentialExcludeList),
	}}
}

// newWebAuthnRequestOptions publishes go-webauthn's assertion options.
func newWebAuthnRequestOptions(a *protocol.CredentialAssertion) WebAuthnRequestOptionsResponse {
	o := a.Response
	return WebAuthnRequestOptionsResponse{PublicKey: WebAuthnRequestOptions{
		Challenge:        o.Challenge.String(),
		TimeoutMs:        o.Timeout,
		RPID:             o.RelyingPartyID,
		AllowCredentials: newWebAuthnDescriptors(o.AllowedCredentials),
	}}
}

func newWebAuthnDescriptors(list []protocol.CredentialDescriptor) []WebAuthnCredentialDescriptor {
	out := make([]WebAuthnCredentialDescriptor, 0, len(list))
	for _, d := range list {
		var transports []string
		for _, t := range d.Transport {
			transports = append(transports, string(t))
		}
		out = append(out, WebAuthnCredentialDescriptor{
			Type:       string(d.Type),
			ID:         d.CredentialID.String(),
			Transports: transports,
		})
	}
	return out
}
