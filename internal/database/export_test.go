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

// RollupDailyRow is one decoded anomaly_rollups_daily row, for testing the
// census output without exposing a production read API before a consumer needs
// one (ADR-0028).
type RollupDailyRow struct {
	DayBucket   string
	DefKey      string
	Source      string
	Category    string
	SubjectKind string
	SubjectID   string
	MaxSeverity string
	CountCumul  int64
	IsResolved  bool
	ResolvedAt  string
}

// ExportRollupDailyRows returns every anomaly_rollups_daily row ordered by
// (day_bucket, def_key, subject_id), for testing.
func (db *DB) ExportRollupDailyRows(ctx context.Context) ([]RollupDailyRow, error) {
	rows, err := db.Query(ctx, `
		SELECT day_bucket, def_key, source, category, subject_kind, subject_id,
		       max_severity, count_cumulative, is_resolved, COALESCE(resolved_at, '')
		FROM anomaly_rollups_daily
		ORDER BY day_bucket, def_key, subject_id
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []RollupDailyRow
	for rows.Next() {
		var r RollupDailyRow
		var resolved int
		if scanErr := rows.Scan(&r.DayBucket, &r.DefKey, &r.Source, &r.Category,
			&r.SubjectKind, &r.SubjectID, &r.MaxSeverity, &r.CountCumul,
			&resolved, &r.ResolvedAt); scanErr != nil {
			return nil, scanErr
		}
		r.IsResolved = resolved != 0
		out = append(out, r)
	}
	return out, rows.Err()
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
