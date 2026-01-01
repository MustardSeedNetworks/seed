// Package database provides settings repository for key-value settings.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrSettingNotFound is returned when a setting is not found.
var ErrSettingNotFound = errors.New("setting not found")

// SettingsRepository provides operations for key-value settings.
type SettingsRepository struct {
	db *DB
}

// Get retrieves a setting by key.
func (r *SettingsRepository) Get(ctx context.Context, key string) (*Setting, error) {
	row := r.db.QueryRow(ctx, `
		SELECT key, value, updated_at FROM settings WHERE key = ?
	`, key)

	var s Setting
	var updatedAt string

	err := row.Scan(&s.Key, &s.Value, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSettingNotFound
		}
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	if t, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil {
		s.UpdatedAt = t
	}
	return &s, nil
}

// GetValue retrieves the value for a setting key, returning empty string if not found.
func (r *SettingsRepository) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return "", nil
		}
		return "", err
	}
	return setting.Value, nil
}

// GetWithDefault retrieves the value for a key, returning the default if not found.
func (r *SettingsRepository) GetWithDefault(ctx context.Context, key, defaultValue string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaultValue, nil
		}
		return "", err
	}
	return setting.Value, nil
}

// Set creates or updates a setting.
func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	now := time.Now().UTC()

	_, err := r.db.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, now.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}

	return nil
}

// SetIfNotExists creates a setting only if it doesn't already exist.
func (r *SettingsRepository) SetIfNotExists(ctx context.Context, key, value string) (bool, error) {
	_, err := r.Get(ctx, key)
	if err == nil {
		// Already exists
		return false, nil
	}
	if !errors.Is(err, ErrSettingNotFound) {
		return false, err
	}

	// Doesn't exist, create it
	err = r.Set(ctx, key, value)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes a setting by key.
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM settings WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSettingNotFound
	}

	return nil
}

// List retrieves all settings.
func (r *SettingsRepository) List(ctx context.Context) ([]*Setting, error) {
	rows, err := r.db.Query(ctx, `
		SELECT key, value, updated_at FROM settings ORDER BY key
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var settings []*Setting
	for rows.Next() {
		var s Setting
		var updatedAt string

		if scanErr := rows.Scan(&s.Key, &s.Value, &updatedAt); scanErr != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", scanErr)
		}

		if t, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil {
			s.UpdatedAt = t
		}
		settings = append(settings, &s)
	}

	return settings, rows.Err()
}

// GetByPrefix retrieves all settings with keys starting with the given prefix.
func (r *SettingsRepository) GetByPrefix(ctx context.Context, prefix string) ([]*Setting, error) {
	rows, err := r.db.Query(ctx, `
		SELECT key, value, updated_at FROM settings WHERE key LIKE ? ORDER BY key
	`, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to get settings by prefix: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var settings []*Setting
	for rows.Next() {
		var s Setting
		var updatedAt string

		if scanErr := rows.Scan(&s.Key, &s.Value, &updatedAt); scanErr != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", scanErr)
		}

		if t, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil {
			s.UpdatedAt = t
		}
		settings = append(settings, &s)
	}

	return settings, rows.Err()
}

// DeleteByPrefix removes all settings with keys starting with the given prefix.
func (r *SettingsRepository) DeleteByPrefix(ctx context.Context, prefix string) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM settings WHERE key LIKE ?`, prefix+"%")
	if err != nil {
		return 0, fmt.Errorf("failed to delete settings by prefix: %w", err)
	}

	return result.RowsAffected()
}

// Count returns the total number of settings.
func (r *SettingsRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	row := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM settings`)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count settings: %w", err)
	}
	return count, nil
}

// Common setting keys.
const (
	SettingKeyActiveProfile     = "active_profile_id"
	SettingKeyLastScanTime      = "last_scan_time"
	SettingKeyRetentionDays     = "retention_days"
	SettingKeyAutoScanEnabled   = "auto_scan_enabled"
	SettingKeyAutoScanInterval  = "auto_scan_interval"
	SettingKeySNMPCommunity     = "snmp_community"
	SettingKeyDefaultInterface  = "default_interface"
	SettingKeyAlertEmailEnabled = "alert_email_enabled"
	SettingKeyAlertEmailTo      = "alert_email_to"
)
