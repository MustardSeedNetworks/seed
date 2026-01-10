package database

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Data retention period defaults (in days).
const (
	// defaultMetricsDays is the default retention period for metric data (3 months).
	defaultMetricsDays = 90

	// defaultAlertsDays is the default retention period for alerts (1 year).
	defaultAlertsDays = 365

	// defaultInactiveDeviceDays is the default period before removing inactive devices (1 month).
	defaultInactiveDeviceDays = 30

	// defaultAuditLogDays is the default retention period for audit logs (1 year).
	defaultAuditLogDays = 365

	// defaultSpeedTestDays is the default retention period for speed test results (3 months).
	defaultSpeedTestDays = 90

	// defaultDNSResultDays is the default retention period for DNS test results (1 month).
	defaultDNSResultDays = 30

	// defaultGatewayResultDays is the default retention period for gateway results (1 month).
	defaultGatewayResultDays = 30

	// defaultHealthCheckRawDays is the default retention for raw health check data (7 days).
	defaultHealthCheckRawDays = 7

	// defaultHealthCheckHourlyDays is the default retention for hourly rollups (90 days).
	defaultHealthCheckHourlyDays = 90

	// defaultHealthCheckDailyDays is the default retention for daily rollups (365 days).
	defaultHealthCheckDailyDays = 365
)

// SQL query fragment constants.
const (
	sqlAndTimestampGte  = " AND timestamp >= ?"
	sqlAndTimestampLte  = " AND timestamp <= ?"
	sqlAndType          = " AND type = ?"
	sqlAndSeverity      = " AND severity = ?"
	sqlOrderByTimestamp = " ORDER BY timestamp DESC"
	sqlLimit            = " LIMIT ?"
	sqlOffset           = " OFFSET ?"
)

// Error variables for common database errors.
var errDatabaseClosed = errors.New("database is closed")

// RetentionPolicy defines how long to keep different types of data.
type RetentionPolicy struct {
	// MetricsDays is how many days to keep metric data (0 = forever)
	MetricsDays int

	// AlertsDays is how many days to keep alerts (0 = forever)
	AlertsDays int

	// InactiveDeviceDays is how many days before removing inactive devices (0 = forever)
	InactiveDeviceDays int

	// AuditLogDays is how many days to keep audit logs (0 = forever)
	AuditLogDays int

	// SpeedTestDays is how many days to keep speed test results (0 = forever)
	SpeedTestDays int

	// DNSResultDays is how many days to keep DNS results (0 = forever)
	DNSResultDays int

	// GatewayResultDays is how many days to keep gateway results (0 = forever)
	GatewayResultDays int

	// HealthCheckRawDays is how many days to keep raw health check data (0 = forever)
	HealthCheckRawDays int

	// HealthCheckHourlyDays is how many days to keep hourly rollups (0 = forever)
	HealthCheckHourlyDays int

	// HealthCheckDailyDays is how many days to keep daily rollups (0 = forever)
	HealthCheckDailyDays int
}

// DefaultRetentionPolicy returns the default retention policy.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		MetricsDays:           defaultMetricsDays,
		AlertsDays:            defaultAlertsDays,
		InactiveDeviceDays:    defaultInactiveDeviceDays,
		AuditLogDays:          defaultAuditLogDays,
		SpeedTestDays:         defaultSpeedTestDays,
		DNSResultDays:         defaultDNSResultDays,
		GatewayResultDays:     defaultGatewayResultDays,
		HealthCheckRawDays:    defaultHealthCheckRawDays,
		HealthCheckHourlyDays: defaultHealthCheckHourlyDays,
		HealthCheckDailyDays:  defaultHealthCheckDailyDays,
	}
}

// CleanupResult holds the results of a cleanup operation.
type CleanupResult struct {
	MetricsDeleted           int64
	AlertsDeleted            int64
	DevicesDeleted           int64
	AuditLogsDeleted         int64
	SpeedTestsDeleted        int64
	DNSResultsDeleted        int64
	GatewayResultsDeleted    int64
	HealthCheckRawDeleted    int64
	HealthCheckHourlyDeleted int64
	HealthCheckDailyDeleted  int64
	Duration                 time.Duration
}

// cleanupFunc is a function type for cleanup operations that delete records older than a cutoff time.
type cleanupFunc func(ctx context.Context, cutoff time.Time) (int64, error)

// cleanupTask represents a single cleanup operation with its configuration.
type cleanupTask struct {
	retentionDays int
	deleteFunc    cleanupFunc
	name          string
}

// runCleanupTask executes a single cleanup task if retention days > 0.
func (db *DB) runCleanupTask(ctx context.Context, task cleanupTask, now time.Time) (int64, error) {
	if task.retentionDays <= 0 {
		return 0, nil
	}

	cutoff := now.AddDate(0, 0, -task.retentionDays)
	deleted, err := task.deleteFunc(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup %s: %w", task.name, err)
	}

	return deleted, nil
}

// RunCleanup executes data cleanup based on the retention policy.
func (db *DB) RunCleanup(ctx context.Context, policy RetentionPolicy) (*CleanupResult, error) {
	start := time.Now()
	result := &CleanupResult{}
	now := time.Now().UTC()

	tasks := []cleanupTask{
		{policy.MetricsDays, db.cleanupMetrics, "metrics"},
		{policy.AlertsDays, db.cleanupAlerts, "alerts"},
		{policy.InactiveDeviceDays, db.cleanupDevices, "devices"},
		{policy.AuditLogDays, db.deleteAuditLogsOlderThan, "audit logs"},
		{policy.SpeedTestDays, db.deleteSpeedTestsOlderThan, "speed tests"},
		{policy.DNSResultDays, db.deleteDNSResultsOlderThan, "DNS results"},
		{policy.GatewayResultDays, db.deleteGatewayResultsOlderThan, "gateway results"},
		{policy.HealthCheckRawDays, db.cleanupHealthCheckRaw, "health check raw"},
		{policy.HealthCheckHourlyDays, db.cleanupHealthCheckHourly, "health check hourly"},
		{policy.HealthCheckDailyDays, db.cleanupHealthCheckDaily, "health check daily"},
	}

	results := make([]int64, len(tasks))
	for i, task := range tasks {
		deleted, err := db.runCleanupTask(ctx, task, now)
		if err != nil {
			return nil, err
		}
		results[i] = deleted
	}

	result.MetricsDeleted = results[0]
	result.AlertsDeleted = results[1]
	result.DevicesDeleted = results[2]
	result.AuditLogsDeleted = results[3]
	result.SpeedTestsDeleted = results[4]
	result.DNSResultsDeleted = results[5]
	result.GatewayResultsDeleted = results[6]
	result.HealthCheckRawDeleted = results[7]
	result.HealthCheckHourlyDeleted = results[8]
	result.HealthCheckDailyDeleted = results[9]
	result.Duration = time.Since(start)

	return result, nil
}

// cleanupMetrics wraps the Metrics().DeleteOlderThan call to match cleanupFunc signature.
func (db *DB) cleanupMetrics(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.Metrics().DeleteOlderThan(ctx, cutoff)
}

// cleanupAlerts wraps the Alerts().DeleteOlderThan call to match cleanupFunc signature.
func (db *DB) cleanupAlerts(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.Alerts().DeleteOlderThan(ctx, cutoff)
}

// cleanupDevices wraps the Devices().DeleteInactive call to match cleanupFunc signature.
func (db *DB) cleanupDevices(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.Devices().DeleteInactive(ctx, cutoff)
}

// Vacuum optimizes the database file by reclaiming unused space.
func (db *DB) Vacuum(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errDatabaseClosed
	}

	_, err := db.conn.ExecContext(ctx, "VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	return nil
}

// Analyze updates database statistics for query optimization.
func (db *DB) Analyze(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errDatabaseClosed
	}

	_, err := db.conn.ExecContext(ctx, "ANALYZE")
	if err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}

	return nil
}

// Optimize runs both vacuum and analyze for database maintenance.
func (db *DB) Optimize(ctx context.Context) error {
	if err := db.Vacuum(ctx); err != nil {
		return fmt.Errorf("optimize vacuum: %w", err)
	}
	if err := db.Analyze(ctx); err != nil {
		return fmt.Errorf("optimize analyze: %w", err)
	}
	return nil
}

// Helper functions for cleanup

func (db *DB) deleteAuditLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM audit_log WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete audit logs exec: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete audit logs rows affected: %w", err)
	}
	return rowsAffected, nil
}

func (db *DB) deleteSpeedTestsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM speedtest_results WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete speed tests exec: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete speed tests rows affected: %w", err)
	}
	return rowsAffected, nil
}

func (db *DB) deleteDNSResultsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM dns_results WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete DNS results exec: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete DNS results rows affected: %w", err)
	}
	return rowsAffected, nil
}

func (db *DB) deleteGatewayResultsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM gateway_results WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete gateway results exec: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete gateway results rows affected: %w", err)
	}
	return rowsAffected, nil
}

// cleanupHealthCheckRaw wraps HealthChecks().DeleteOlderThan for raw health check data.
func (db *DB) cleanupHealthCheckRaw(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.HealthChecks().DeleteOlderThan(ctx, cutoff)
}

// cleanupHealthCheckHourly wraps HealthChecks().DeleteHourlyRollupsOlderThan.
func (db *DB) cleanupHealthCheckHourly(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.HealthChecks().DeleteHourlyRollupsOlderThan(ctx, cutoff)
}

// cleanupHealthCheckDaily wraps HealthChecks().DeleteDailyRollupsOlderThan.
func (db *DB) cleanupHealthCheckDaily(ctx context.Context, cutoff time.Time) (int64, error) {
	return db.HealthChecks().DeleteDailyRollupsOlderThan(ctx, cutoff)
}

// RecordAuditLog records an audit log entry.
func (db *DB) RecordAuditLog(ctx context.Context, entry *AuditLogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	result, err := db.Exec(ctx, `
		INSERT INTO audit_log
		(action, user, resource_type, resource_id, old_value_json, new_value_json,
		 ip_address, user_agent, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.Action, entry.User, entry.ResourceType, entry.ResourceID,
		entry.OldValueJSON, entry.NewValueJSON, entry.IPAddress, entry.UserAgent,
		entry.Timestamp.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to record audit log: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		entry.ID = id
	}

	return nil
}

// GetAuditLogs retrieves audit log entries.
//

func (db *DB) GetAuditLogs(ctx context.Context, opts AuditLogOptions) ([]*AuditLogEntry, error) {
	query := `
		SELECT id, action, user, resource_type, resource_id, old_value_json,
		       new_value_json, ip_address, user_agent, timestamp
		FROM audit_log
		WHERE 1=1
	`
	var args []any

	if opts.Action != "" {
		query += " AND action = ?"
		args = append(args, opts.Action)
	}

	if opts.User != "" {
		query += " AND user = ?"
		args = append(args, opts.User)
	}

	if opts.ResourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, opts.ResourceType)
	}

	if opts.ResourceID != "" {
		query += " AND resource_id = ?"
		args = append(args, opts.ResourceID)
	}

	if !opts.Since.IsZero() {
		query += sqlAndTimestampGte
		args = append(args, opts.Since.UTC().Format(time.RFC3339))
	}

	query += sqlOrderByTimestamp

	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += sqlOffset
		args = append(args, opts.Offset)
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []*AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var timestamp string
		var user, resType, resID, oldVal, newVal, ip, ua string

		scanErr := rows.Scan(&e.ID, &e.Action, &user, &resType, &resID, &oldVal, &newVal,
			&ip, &ua, &timestamp)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", scanErr)
		}

		e.User = user
		e.ResourceType = resType
		e.ResourceID = resID
		e.OldValueJSON = oldVal
		e.NewValueJSON = newVal
		e.IPAddress = ip
		e.UserAgent = ua
		if t, parseErr := time.Parse(time.RFC3339, timestamp); parseErr == nil {
			e.Timestamp = t
		}

		entries = append(entries, &e)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("audit logs rows iteration: %w", rowsErr)
	}
	return entries, nil
}

// AuditLogOptions specifies criteria for querying audit logs.
type AuditLogOptions struct {
	Action       string
	User         string
	ResourceType string
	ResourceID   string
	Since        time.Time
	Limit        int
	Offset       int
}

// Common audit log actions.
const (
	AuditActionCreate   = "create"
	AuditActionUpdate   = "update"
	AuditActionDelete   = "delete"
	AuditActionLogin    = "login"
	AuditActionLogout   = "logout"
	AuditActionScan     = "scan"
	AuditActionExport   = "export"
	AuditActionSettings = "settings"
)
