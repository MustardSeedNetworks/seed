// Package app is the composition root: it wires feature packages
// (internal/<feature>) to their persistence adapters and hands the assembled
// components to the API layer. It is the one place that may depend on both a
// component and its store adapter, keeping that knowledge out of the
// persistence-free feature cores.
package app

import (
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/reporting"
	"github.com/krisarmstrong/seed/internal/reporting/store"
)

// NewReporting builds the reporting component backed by the SQLite store adapters.
func NewReporting(cfg *config.Config, db *database.DB) *reporting.Service {
	return reporting.New(cfg, reporting.Deps{
		Reports:  store.NewReportRepo(db),
		Schedule: store.NewScheduleRepo(db),
		Metrics:  store.NewMetricsRepo(db),
		Export:   store.NewExportRepo(db),
	})
}
