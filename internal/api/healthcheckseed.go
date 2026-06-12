package api

// healthcheckseed.go restores the pre-ADR-0027 out-of-box behavior: a fresh
// install ships with the factory health-check targets visible. Since P2 the
// probes table is the store of record (see healthcheckmapping.go), and the
// only writer is the settings-PUT path — so nothing populated it on first
// run and the health-check card came up empty. This seeds the factory
// defaults once, gated by a persistent marker so an operator who later
// deletes every probe stays empty across restarts.

import (
	"context"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/health/probemap"
)

// settingKeyHealthChecksSeeded marks that the factory health-check probes
// have been seeded. Its presence — not the probe count — is the gate, so
// deleting every health-check probe does not trigger a re-seed.
const settingKeyHealthChecksSeeded = "health_checks.seeded"

// seedDefaultHealthCheckProbes seeds the factory health-check targets into
// the probes table on a genuine first run. It is a no-op once the seed
// marker is set, and it never overwrites an install that already holds
// health-check probes (the upgrade path, where the marker predates this
// code). The factory set comes from config.DefaultConfig().HealthChecks and
// is mapped through the same ProbesFromConfig used by the
// settings-PUT save, so the two paths can never diverge.
func (s *Server) seedDefaultHealthCheckProbes(ctx context.Context, db *database.DB) error {
	settings := db.Settings()

	marker, err := settings.GetValue(ctx, settingKeyHealthChecksSeeded)
	if err != nil {
		return fmt.Errorf("read health-check seed marker: %w", err)
	}
	if marker != "" {
		return nil // already seeded — honor an operator's later delete-all.
	}

	// Upgrade guard: an install that already configured health-check probes
	// (saved before this seeding existed, so no marker) must keep its set.
	// Mark it seeded and leave the probes untouched.
	existing, err := probemap.CountProbes(ctx, db.Probes())
	if err != nil {
		return fmt.Errorf("count existing health-check probes: %w", err)
	}
	if existing > 0 {
		return s.markHealthChecksSeeded(ctx, settings)
	}

	hc := config.DefaultConfig().HealthChecks
	probes, err := probemap.ProbesFromConfig(&hc)
	if err != nil {
		return fmt.Errorf("build default health-check probes: %w", err)
	}
	if err = db.Probes().ReplaceProbesByKinds(
		ctx, database.DefaultClientID, probemap.Kinds(), probes,
	); err != nil {
		return fmt.Errorf("seed default health-check probes: %w", err)
	}
	return s.markHealthChecksSeeded(ctx, settings)
}

// markHealthChecksSeeded records the seed marker with a UTC timestamp value.
// The value is informational; only its presence gates re-seeding.
func (s *Server) markHealthChecksSeeded(ctx context.Context, settings *database.SettingsRepository) error {
	if err := settings.Set(ctx, settingKeyHealthChecksSeeded, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set health-check seed marker: %w", err)
	}
	return nil
}
