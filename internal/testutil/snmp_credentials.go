package testutil

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// StaticSNMPCredentials satisfies discovery.SNMPCredentialProvider from a
// fixed session. Production resolves discovery's credentials from the encrypted
// vault, scoped to the deployment's one client (#2118); a test that only needs
// a profiler to exist should not have to stand up a database and a keyring to
// get one.
type StaticSNMPCredentials struct {
	Session *snmp.Session
	Err     error
}

// SNMPSession returns the fixed session, or the fixed error when one is set.
func (s StaticSNMPCredentials) SNMPSession(context.Context) (*snmp.Session, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Session, nil
}
