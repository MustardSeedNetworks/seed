// Package database exports internal functions for testing.
package database

import (
	"context"
	"time"
)

// ExportMigrationsCount returns the count of migrations for testing.
func ExportMigrationsCount() int {
	return len(migrations)
}

// DeleteAuditLogsOlderThan exports deleteAuditLogsOlderThan for testing.
func (db *DB) DeleteAuditLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.deleteAuditLogsOlderThan(ctx, cutoff)
}

// DeleteSpeedTestsOlderThan exports deleteSpeedTestsOlderThan for testing.
func (db *DB) DeleteSpeedTestsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.deleteSpeedTestsOlderThan(ctx, cutoff)
}

// DeleteDNSResultsOlderThan exports deleteDNSResultsOlderThan for testing.
func (db *DB) DeleteDNSResultsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.deleteDNSResultsOlderThan(ctx, cutoff)
}

// DeleteGatewayResultsOlderThan exports deleteGatewayResultsOlderThan for testing.
func (db *DB) DeleteGatewayResultsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.deleteGatewayResultsOlderThan(ctx, cutoff)
}
