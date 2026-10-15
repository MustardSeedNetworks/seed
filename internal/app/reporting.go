// Package app is the composition root: it wires feature packages
// (internal/<feature>) to their persistence adapters and hands the assembled
// components to the API layer. It is the one place that may depend on both a
// component and its persistence adapter, keeping that knowledge out of the
// persistence-free feature cores.
package app

import (
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// NewReporting builds the reporting component backed by the database report repositories.
func NewReporting(cfg *config.Config, persist *Persistence) *reporting.Service {
	db := persist.handle()
	return reporting.New(cfg, reporting.Deps{
		Reports:  database.NewReportRepository(db),
		Schedule: database.NewReportScheduleRepository(db),
		Metrics:  database.NewReportMetricsRepository(db),
		Export:   database.NewReportExportRepository(db),
	})
}
