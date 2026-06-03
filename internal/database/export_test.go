package database

import (
	"context"
	"io/fs"
	"strings"
	"time"
)

// ExportMigrationsCount returns the number of embedded goose migrations, for
// testing. After the Phase-5 collapse this is the count of .sql baselines.
func ExportMigrationsCount() int {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			count++
		}
	}
	return count
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
