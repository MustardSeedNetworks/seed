package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
)

// TestRunDataRetentionScalesTheWindow pins the policy the maintenance loop
// applies: speed tests keep the operator's window, audit logs three times it.
func TestRunDataRetentionScalesTheWindow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	now := time.Now().UTC()
	ago := func(days int) string { return now.AddDate(0, 0, -days).Format(time.RFC3339) }

	for _, days := range []int{9, 11} {
		_, err := db.Exec(ctx,
			`INSERT INTO speedtest_results (interface_name, timestamp) VALUES ('eth0', ?)`, ago(days))
		require.NoError(t, err)
	}
	for _, days := range []int{29, 31} {
		_, err := db.Exec(ctx,
			`INSERT INTO audit_log (action, timestamp) VALUES ('login', ?)`, ago(days))
		require.NoError(t, err)
	}

	result, err := app.RunDataRetention(ctx, db, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.SpeedTestsDeleted)
	require.Equal(t, int64(1), result.AuditLogsDeleted)

	var speedTests, auditLogs int
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM speedtest_results`).Scan(&speedTests))
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_log`).Scan(&auditLogs))
	require.Equal(t, 1, speedTests)
	require.Equal(t, 1, auditLogs)
}
