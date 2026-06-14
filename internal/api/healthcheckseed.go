package api

// healthcheckseed.go restores the pre-ADR-0027 out-of-box behavior: a fresh
// install ships with the factory health-check targets visible. The probes table
// is the store of record (ADR-0027 P2); seeding is owned by the health-checks
// settings use-case, gated by a persistent marker so an operator who later
// deletes every probe stays empty across restarts.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// seedDefaultHealthCheckProbes seeds the factory health-check targets on a
// genuine first run via the health-settings use-case. No-op once seeded.
func (s *Server) seedDefaultHealthCheckProbes(ctx context.Context) error {
	return s.healthSettings.SeedDefaults(ctx, config.DefaultConfig().HealthChecks)
}
