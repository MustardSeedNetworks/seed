// Package database provides data retention and cleanup functionality.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"
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
}

// DefaultRetentionPolicy returns the default retention policy.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		MetricsDays:        90,  // 3 months
		AlertsDays:         365, // 1 year
		InactiveDeviceDays: 30,  // 1 month
		AuditLogDays:       365, // 1 year
		SpeedTestDays:      90,  // 3 months
		DNSResultDays:      30,  // 1 month
		GatewayResultDays:  30,  // 1 month
	}
}

// CleanupResult holds the results of a cleanup operation.
type CleanupResult struct {
	MetricsDeleted        int64
	AlertsDeleted         int64
	DevicesDeleted        int64
	AuditLogsDeleted      int64
	SpeedTestsDeleted     int64
	DNSResultsDeleted     int64
	GatewayResultsDeleted int64
	Duration              time.Duration
}

// RunCleanup executes data cleanup based on the retention policy.
func (db *DB) RunCleanup(ctx context.Context, policy RetentionPolicy) (*CleanupResult, error) {
	start := time.Now()
	result := &CleanupResult{}
	now := time.Now().UTC()

	// Cleanup metrics
	if policy.MetricsDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.MetricsDays)
		deleted, err := db.Metrics().DeleteOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup metrics: %w", err)
		}
		result.MetricsDeleted = deleted
	}

	// Cleanup alerts
	if policy.AlertsDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.AlertsDays)
		deleted, err := db.Alerts().DeleteOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup alerts: %w", err)
		}
		result.AlertsDeleted = deleted
	}

	// Cleanup inactive devices
	if policy.InactiveDeviceDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.InactiveDeviceDays)
		deleted, err := db.Devices().DeleteInactive(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup devices: %w", err)
		}
		result.DevicesDeleted = deleted
	}

	// Cleanup audit logs
	if policy.AuditLogDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.AuditLogDays)
		deleted, err := db.deleteAuditLogsOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup audit logs: %w", err)
		}
		result.AuditLogsDeleted = deleted
	}

	// Cleanup speed tests
	if policy.SpeedTestDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.SpeedTestDays)
		deleted, err := db.deleteSpeedTestsOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup speed tests: %w", err)
		}
		result.SpeedTestsDeleted = deleted
	}

	// Cleanup DNS results
	if policy.DNSResultDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.DNSResultDays)
		deleted, err := db.deleteDNSResultsOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup DNS results: %w", err)
		}
		result.DNSResultsDeleted = deleted
	}

	// Cleanup gateway results
	if policy.GatewayResultDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.GatewayResultDays)
		deleted, err := db.deleteGatewayResultsOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup gateway results: %w", err)
		}
		result.GatewayResultsDeleted = deleted
	}

	result.Duration = time.Since(start)
	return result, nil
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
		return err
	}
	return db.Analyze(ctx)
}

// Helper functions for cleanup

func (db *DB) deleteAuditLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM audit_log WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) deleteSpeedTestsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM speedtest_results WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) deleteDNSResultsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM dns_results WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) deleteGatewayResultsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM gateway_results WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
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
//nolint:goconst,gocritic // SQL fragments are more readable inline; opts passed by value for API stability
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
		query += " AND timestamp >= ?"
		args = append(args, opts.Since.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var timestamp string
		var user, resType, resID, oldVal, newVal, ip, ua string

		err := rows.Scan(&e.ID, &e.Action, &user, &resType, &resID, &oldVal, &newVal,
			&ip, &ua, &timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
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

	return entries, rows.Err()
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
