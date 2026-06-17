package database

import (
	"context"
	"fmt"
)

// GetWiFiStats returns WiFi discovery statistics.
func (r *DiscoveryRepository) GetWiFiStats(ctx context.Context) (*WiFiStatsDB, error) {
	stats := &WiFiStatsDB{
		SecurityBreakdown: make(map[string]int),
		BandBreakdown:     make(map[string]int),
	}

	// Total networks
	row := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_networks`)
	if err := row.Scan(&stats.TotalNetworks); err != nil {
		return nil, fmt.Errorf("failed to count networks: %w", err)
	}

	// Hidden networks
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_networks WHERE is_hidden = 1`)
	if err := row.Scan(&stats.HiddenNetworks); err != nil {
		return nil, fmt.Errorf("failed to count hidden networks: %w", err)
	}

	// Total APs
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_access_points`)
	if err := row.Scan(&stats.TotalAPs); err != nil {
		return nil, fmt.Errorf("failed to count APs: %w", err)
	}

	// Authorized/Unauthorized APs
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_access_points WHERE is_authorized = 1`)
	if err := row.Scan(&stats.AuthorizedAPs); err != nil {
		return nil, fmt.Errorf("failed to count authorized APs: %w", err)
	}
	stats.UnauthorizedAPs = stats.TotalAPs - stats.AuthorizedAPs

	// Security breakdown
	rows, err := r.db.Query(ctx, `SELECT security_type, COUNT(*) FROM wifi_networks GROUP BY security_type`)
	if err != nil {
		return nil, fmt.Errorf("failed to get security breakdown: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var secType string
		var count int
		if scanErr := rows.Scan(&secType, &count); scanErr != nil {
			return nil, scanErr
		}
		stats.SecurityBreakdown[secType] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating security breakdown: %w", err)
	}

	// Band breakdown
	rowsBand, err := r.db.Query(ctx, `SELECT band, COUNT(*) FROM wifi_access_points GROUP BY band`)
	if err != nil {
		return nil, fmt.Errorf("failed to get band breakdown: %w", err)
	}
	defer func() { _ = rowsBand.Close() }()
	for rowsBand.Next() {
		var band string
		var count int
		if scanErr := rowsBand.Scan(&band, &count); scanErr != nil {
			return nil, scanErr
		}
		stats.BandBreakdown[band] = count
	}
	if err = rowsBand.Err(); err != nil {
		return nil, fmt.Errorf("iterating band breakdown: %w", err)
	}

	return stats, nil
}

// WiFiStatsDB represents WiFi discovery statistics.
type WiFiStatsDB struct {
	TotalNetworks     int            `json:"totalNetworks"`
	HiddenNetworks    int            `json:"hiddenNetworks"`
	TotalAPs          int            `json:"totalAPs"`
	AuthorizedAPs     int            `json:"authorizedAPs"`
	UnauthorizedAPs   int            `json:"unauthorizedAPs"`
	SecurityBreakdown map[string]int `json:"securityBreakdown"`
	BandBreakdown     map[string]int `json:"bandBreakdown"`
}
