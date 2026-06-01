package harvest

import "context"

// ports.go declares the infrastructure interfaces the harvest module depends
// on. The module is pure: it owns these ports, and the concrete I/O lives in
// internal/adapters/* (inward-only — an adapter imports harvest, never the
// reverse). See docs/architecture/PHASE3_EXTRACTION_PLAN.md §4.7.

// ReportRepo persists report records. The SQLite implementation lives in
// internal/adapters/store. The service depends on this interface so report
// orchestration can be unit-tested with a fake repo (no database).
type ReportRepo interface {
	// GetReport returns the report row with the given id.
	GetReport(ctx context.Context, id string) (*Report, error)
	// ListReports returns all report rows, newest first.
	ListReports(ctx context.Context) ([]Report, error)
	// SaveReport upserts a report row.
	SaveReport(ctx context.Context, r *Report) error
	// DeleteReport removes the report row only; file cleanup is the service's
	// orchestration concern.
	DeleteReport(ctx context.Context, id string) error
}
