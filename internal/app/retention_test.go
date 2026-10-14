package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
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

// TestRunDataRetentionPurgesRawIngest pins that the raw ingest tables age out
// on the operator's window (#3009): every SNMP poll and every syslog message or
// trap writes a row, so without a purge both tables grow without bound.
func TestRunDataRetentionPurgesRawIngest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	now := time.Now().UTC()

	for _, days := range []int{9, 11} {
		observedAt := now.AddDate(0, 0, -days)
		require.NoError(t, db.SNMPObservations().Insert(ctx, &observation.SNMPObservation{
			TargetID: "sw1", Kind: "if_table", ObservedAt: observedAt, PayloadJSON: "{}",
		}))
		require.NoError(t, db.ListenerEvents().Insert(ctx, &listener.EventRecord{
			Kind: "syslog", SourceAddr: "192.0.2.1", ObservedAt: observedAt, PayloadJSON: "{}",
		}))
	}

	result, err := app.RunDataRetention(ctx, db, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.SNMPObservationsDeleted)
	require.Equal(t, int64(1), result.ListenerEventsDeleted)

	var observations, events int
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM snmp_observations`).Scan(&observations))
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM listener_events`).Scan(&events))
	require.Equal(t, 1, observations)
	require.Equal(t, 1, events)
}
