package main

import (
	"context"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestBackgroundComponentsStartWithoutDatabase pins the panic in #2380.
//
// internal/api treats a nil *database.DB as a supported degraded mode — it
// guards on it in a dozen places — but initializeBackgroundComponents built the
// reporting service unconditionally, and its scheduler dereferenced the nil
// handle on the first call. The daemon died before binding a listener, so the
// operator saw a stack trace instead of the ERROR line naming the database.
func TestBackgroundComponentsStartWithoutDatabase(t *testing.T) {
	components := initializeBackgroundComponents(config.DefaultConfig(), nil)

	if components.Reporting != nil {
		t.Error("reporting was constructed with no database; it cannot load its schedules")
	}

	// The panic was here, inside Start.
	if err := components.Start(context.Background()); err != nil {
		t.Fatalf("Start with no database: %v", err)
	}
	t.Cleanup(func() { _ = components.Stop() })
}
