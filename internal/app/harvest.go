// Package app is the composition root: it wires domain modules
// (internal/modules/*) to their infrastructure adapters (internal/adapters/*)
// and hands the assembled modules to the API layer. It is the one place that
// may depend on both a module and its adapters, keeping that knowledge out of
// the pure module cores.
package app

import (
	"github.com/krisarmstrong/seed/internal/adapters/store"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/modules/harvest"
)

// NewHarvest builds the Harvest module backed by the SQLite store adapters.
func NewHarvest(cfg *config.Config, db *database.DB) *harvest.Module {
	return harvest.New(cfg, harvest.Deps{
		Reports:  store.NewReportRepo(db),
		Schedule: store.NewScheduleRepo(db),
		Metrics:  store.NewMetricsRepo(db),
		Export:   store.NewExportRepo(db),
	})
}
