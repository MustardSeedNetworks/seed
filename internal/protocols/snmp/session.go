package snmp

// session.go holds the credential set an SNMP exchange runs with.
//
// These fields used to live on config.SNMPConfig, which meant the community
// strings and v3 passwords a walk put on the wire were also the ones written
// to seed.json in plaintext and handed back to any client that read the
// settings API (#1799). They are two different things: the transport settings
// (port, timeout, retries, bulk size) are operator configuration and belong in
// the file, while the credentials are resolved from the encrypted vault at use
// time and never persisted. Splitting the type is what stops the second from
// following the first into the config file again.

import (
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// Session is the resolved credentials plus transport settings for one SNMP
// exchange. It is built per exchange by a credential resolver, carries
// plaintext secrets, and must not be stored.
type Session struct {
	// SNMPConfig supplies port, timeout, retries and bulk size from the
	// operator's file configuration.
	config.SNMPConfig

	// Communities are the v1/v2c community strings to try, in order.
	Communities []string

	// V3Credentials are the v3 credentials to try, in order. They are tried
	// before Communities.
	V3Credentials []V3Credential
}

// V3Credential is one decrypted SNMPv3 identity.
type V3Credential struct {
	// Name identifies the credential in logs and errors. It never carries a secret.
	Name string
	// Username is the USM security name.
	Username string
	// AuthProtocol is "SHA", "SHA224", "SHA256", "SHA384", "SHA512", the
	// deprecated "MD5", or "" for noAuth.
	AuthProtocol string
	// AuthPassword is the authentication passphrase in plaintext.
	AuthPassword string
	// PrivProtocol is "DES", "AES", "AES192", "AES256", or "" for noPriv.
	PrivProtocol string
	// PrivPassword is the privacy passphrase in plaintext.
	PrivPassword string
	// ContextName is the optional SNMP context.
	ContextName string
	// SecurityLevel is "noAuthNoPriv", "authNoPriv" or "authPriv".
	SecurityLevel string
}

// NewSession builds a session over the operator's transport settings. A nil
// transport yields the zero settings, which the client builders fill from
// their own defaults.
func NewSession(transport *config.SNMPConfig) *Session {
	s := &Session{}
	if transport != nil {
		s.SNMPConfig = config.SNMPConfig{
			Timeout:        transport.Timeout,
			Retries:        transport.Retries,
			Port:           transport.Port,
			MaxRepetitions: transport.MaxRepetitions,
		}
	}
	return s
}
