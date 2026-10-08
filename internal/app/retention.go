package app

// retention.go applies the operator's data-retention window (#755) to the
// database, so the api's maintenance loop never builds a persistence policy
// itself (ADR-0020).

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// Alerts, audit logs and inactive device records outlive the operator's
// retention window by these factors.
const (
	retentionAlertsMultiplier         = 2
	retentionAuditLogMultiplier       = 3
	retentionInactiveDeviceMultiplier = 4
)

// RunDataRetention applies one pass of the retention policy derived from
// retentionDays, which must be positive: a zero window means retention is off,
// never "delete all". rollupDailyDays is the anomaly daily-rollup horizon for
// the current tier; the caller resolves it per pass so an in-place license
// upgrade takes effect on the next one (ADR-0028 §4).
func RunDataRetention(
	ctx context.Context, db *database.DB, retentionDays, rollupDailyDays int,
) (*database.CleanupResult, error) {
	return db.RunCleanup(ctx, database.RetentionPolicy{
		MetricsDays:        retentionDays,
		AlertsDays:         retentionDays * retentionAlertsMultiplier,
		SpeedTestDays:      retentionDays,
		AuditLogDays:       retentionDays * retentionAuditLogMultiplier,
		InactiveDeviceDays: retentionDays * retentionInactiveDeviceMultiplier,
		// Resolved anomalies age out on their own fixed 90d window (ADR-0021);
		// active anomalies are never purged, so this is independent of the
		// operator's general retention window.
		AnomalyResolvedDays:    database.DefaultRetentionPolicy().AnomalyResolvedDays,
		AnomalyRollupDailyDays: rollupDailyDays,
		MicroburstDays:         retentionDays,
		VoIPDays:               retentionDays,
	})
}
