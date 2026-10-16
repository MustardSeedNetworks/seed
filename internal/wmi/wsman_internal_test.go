package wmi

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckPinRejectsEmptyOrMissingPin is the guard behind InsecureSkipVerify
// (ADR-0034). An empty pin and a missing one (the zero Endpoint field) must
// both refuse the leaf, or the handshake would accept whatever certificate
// answered.
func TestCheckPinRejectsEmptyOrMissingPin(t *testing.T) {
	leaf := &x509.Certificate{Raw: []byte("presented-leaf")}
	cs := tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}}
	want := Fingerprint(leaf.Raw)

	t.Run("empty", func(t *testing.T) {
		var unconfirmed *CertUnconfirmedError
		require.ErrorAs(t, checkPin(cs, ""), &unconfirmed)
		require.Equal(t, want, unconfirmed.Fingerprint)
	})

	t.Run("missing", func(t *testing.T) {
		var endpoint Endpoint // PinnedCert was never set
		var unconfirmed *CertUnconfirmedError
		require.ErrorAs(t, checkPin(cs, endpoint.PinnedCert), &unconfirmed)
		require.Equal(t, want, unconfirmed.Fingerprint)
	})

	t.Run("pinned leaf", func(t *testing.T) {
		require.NoError(t, checkPin(cs, want))
	})
}
