package testutil

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// StaticSNMPCredentials satisfies discovery.SNMPCredentialProvider from a
// fixed config. Production resolves discovery's credentials from the encrypted
// vault, scoped to the deployment's one client (#2118); a test that only needs
// a profiler to exist should not have to stand up a database and a keyring to
// get one.
type StaticSNMPCredentials struct {
	Config *config.SNMPConfig
	Err    error
}

// SNMPConfig returns the fixed config, or the fixed error when one is set.
func (s StaticSNMPCredentials) SNMPConfig(context.Context) (*config.SNMPConfig, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Config, nil
}
