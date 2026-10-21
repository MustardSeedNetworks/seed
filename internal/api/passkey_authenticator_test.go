package api_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/stretchr/testify/require"
)

// softAuthenticator is a platform authenticator in software: one P-256
// discoverable credential, user verification always performed, "none"
// attestation. It produces the JSON a browser posts to the finish routes.
type softAuthenticator struct {
	rpID   string
	origin string
	key    *ecdsa.PrivateKey
	id     []byte
	handle []byte
	count  uint32
}

func newSoftAuthenticator(t *testing.T, rpID, origin string) *softAuthenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	return &softAuthenticator{rpID: rpID, origin: origin, key: key, id: id}
}

const (
	flagUserPresent  = 0x01
	flagUserVerified = 0x04
	flagAttested     = 0x40
)

func (a *softAuthenticator) authenticatorData(t *testing.T, attested bool) []byte {
	t.Helper()
	rpHash := sha256.Sum256([]byte(a.rpID))
	flags := byte(flagUserPresent | flagUserVerified)
	if attested {
		flags |= flagAttested
	}
	data := append(rpHash[:], flags)
	data = binary.BigEndian.AppendUint32(data, a.count)
	if !attested {
		return data
	}
	point, err := a.key.PublicKey.Bytes() // 0x04 || X || Y
	require.NoError(t, err)
	cose, err := webauthncbor.Marshal(webauthncose.EC2PublicKeyData{
		KeyType: int64(webauthncose.EllipticKey), Algorithm: int64(webauthncose.AlgES256),
		Curve: int64(webauthncose.P256), XCoord: point[1:33], YCoord: point[33:],
	})
	require.NoError(t, err)
	data = append(data, make([]byte, 16)...) // zero AAGUID
	data = binary.BigEndian.AppendUint16(data, uint16(len(a.id)))
	data = append(data, a.id...)
	return append(data, cose...)
}

func (a *softAuthenticator) clientData(t *testing.T, ceremony, challenge string) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"type": ceremony, "challenge": challenge, "origin": a.origin, "crossOrigin": false,
	})
	require.NoError(t, err)
	return data
}

// register answers navigator.credentials.create for the given options.
func (a *softAuthenticator) register(t *testing.T, challenge string, userHandle []byte) []byte {
	t.Helper()
	a.handle = userHandle
	object, err := webauthncbor.Marshal(map[string]any{
		"fmt": "none", "attStmt": map[string]any{}, "authData": a.authenticatorData(t, true),
	})
	require.NoError(t, err)
	return a.credentialJSON(t, map[string]any{
		"clientDataJSON":    b64(a.clientData(t, "webauthn.create", challenge)),
		"attestationObject": b64(object),
	})
}

// assert answers navigator.credentials.get, signing under userHandle.
func (a *softAuthenticator) assert(t *testing.T, challenge string, userHandle []byte) []byte {
	t.Helper()
	a.count++
	authData := a.authenticatorData(t, false)
	clientData := a.clientData(t, "webauthn.get", challenge)
	clientHash := sha256.Sum256(clientData)
	digest := sha256.Sum256(append(authData, clientHash[:]...))
	signature, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	require.NoError(t, err)
	return a.credentialJSON(t, map[string]any{
		"clientDataJSON":    b64(clientData),
		"authenticatorData": b64(authData),
		"signature":         b64(signature),
		"userHandle":        b64(userHandle),
	})
}

func (a *softAuthenticator) credentialJSON(t *testing.T, response map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"id": b64(a.id), "rawId": b64(a.id), "type": "public-key", "response": response,
	})
	require.NoError(t, err)
	return body
}

func b64(data []byte) string { return base64.RawURLEncoding.EncodeToString(data) }
