package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ChannelUtilizationDB represents channel utilization data in the database.
type ChannelUtilizationDB struct {
	ID                 string    `json:"id"`
	Channel            int       `json:"channel"`
	Band               string    `json:"band"`
	FrequencyMHz       int       `json:"frequencyMhz"`
	UtilizationPercent float64   `json:"utilizationPercent"`
	NonWiFiPercent     float64   `json:"nonWifiPercent"`
	RetryPercent       float64   `json:"retryPercent"`
	APCount            int       `json:"apCount"`
	ClientCount        int       `json:"clientCount"`
	RecordedAt         time.Time `json:"recordedAt"`
}

// CreateChannelUtilization creates a new channel utilization record.
func (r *DiscoveryRepository) CreateChannelUtilization(ctx context.Context, util *ChannelUtilizationDB) error {
	if util.ID == "" {
		util.ID = uuid.New().String()
	}
	if util.RecordedAt.IsZero() {
		util.RecordedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO channel_utilization
		(id, channel, band, frequency_mhz, utilization_percent, non_wifi_percent, retry_percent,
		 ap_count, client_count, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, util.ID, util.Channel, util.Band, util.FrequencyMHz, util.UtilizationPercent,
		util.NonWiFiPercent, util.RetryPercent, util.APCount, util.ClientCount,
		util.RecordedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to create channel utilization: %w", err)
	}
	return nil
}

// GetChannelUtilization retrieves the latest utilization for a channel.
func (r *DiscoveryRepository) GetChannelUtilization(
	ctx context.Context,
	channel int,
	band string,
) (*ChannelUtilizationDB, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, channel, band, frequency_mhz, utilization_percent, non_wifi_percent, retry_percent,
		       ap_count, client_count, recorded_at
		FROM channel_utilization
		WHERE channel = ? AND band = ?
		ORDER BY recorded_at DESC
		LIMIT 1
	`, channel, band)

	var u ChannelUtilizationDB
	var recordedAt string

	err := row.Scan(&u.ID, &u.Channel, &u.Band, &u.FrequencyMHz, &u.UtilizationPercent,
		&u.NonWiFiPercent, &u.RetryPercent, &u.APCount, &u.ClientCount, &recordedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrChannelUtilizationNotFound
		}
		return nil, fmt.Errorf("failed to scan channel utilization: %w", err)
	}

	if t, parseErr := time.Parse(time.RFC3339, recordedAt); parseErr == nil {
		u.RecordedAt = t
	}
	return &u, nil
}

// ListChannelUtilization retrieves channel utilization for all channels.
func (r *DiscoveryRepository) ListChannelUtilization(
	ctx context.Context,
	band string,
) ([]*ChannelUtilizationDB, error) {
	query := `
		SELECT id, channel, band, frequency_mhz, utilization_percent, non_wifi_percent, retry_percent,
		       ap_count, client_count, recorded_at
		FROM channel_utilization
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY channel, band ORDER BY recorded_at DESC) as rn
				FROM channel_utilization
			) WHERE rn = 1
		)
	`
	var args []any

	if band != "" {
		query += " AND band = ?"
		args = append(args, band)
	}

	query += " ORDER BY channel"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list channel utilization: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var utils []*ChannelUtilizationDB
	for rows.Next() {
		var u ChannelUtilizationDB
		var recordedAt string

		if scanErr := rows.Scan(&u.ID, &u.Channel, &u.Band, &u.FrequencyMHz, &u.UtilizationPercent,
			&u.NonWiFiPercent, &u.RetryPercent, &u.APCount, &u.ClientCount, &recordedAt); scanErr != nil {
			return nil, fmt.Errorf("failed to scan channel utilization: %w", scanErr)
		}

		if t, parseErr := time.Parse(time.RFC3339, recordedAt); parseErr == nil {
			u.RecordedAt = t
		}
		utils = append(utils, &u)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating channel utilization: %w", rowsErr)
	}
	return utils, nil
}

// Discovery Statistics
