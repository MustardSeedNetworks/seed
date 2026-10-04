package store

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/forecast"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

// hourLayout is the hour prefix both metrics.timestamp and
// metrics_hourly.hour_bucket start with.
const hourLayout = "2006-01-02T15"

// hourMiddle times an hour's mean at the middle of the hour it stands for.
const hourMiddle = 30 * time.Minute

// InterfaceUtilizationHistory returns each rated interface's hourly
// utilization in the window. Hours the retention engine has rolled up come
// from metrics_hourly, which outlives the raw rows; the rest are averaged from
// the raw rates. Each hour's value is the busier direction's mean.
func (r *MetricsRepo) InterfaceUtilizationHistory(
	ctx context.Context,
	window reporting.DateRange,
) ([]reporting.InterfaceUtilization, error) {
	from, to := windowBounds(window)
	// Rolled-up hours sort first, so a raw average only fills an hour the
	// rollup has not reached. A rolled-up hour is preferred because the raw
	// purge may already have taken part of it.
	rows, err := r.db.Query(ctx, `
		SELECT client_id, target_id, hour, metric_type, value FROM (
			SELECT client_id, target_id, substr(hour_bucket, 1, 13) AS hour, metric_type,
				avg_value AS value, 0 AS raw
			FROM metrics_hourly
			WHERE target_kind = ? AND metric_type IN (?, ?) AND avg_value IS NOT NULL
				AND hour_bucket >= ? AND hour_bucket < ?
			UNION ALL
			SELECT client_id, target_id, substr(timestamp, 1, 13), metric_type, AVG(value), 1
			FROM metrics
			WHERE target_kind = ? AND metric_type IN (?, ?) AND timestamp >= ? AND timestamp < ?
			GROUP BY client_id, target_id, substr(timestamp, 1, 13), metric_type
		)
		ORDER BY raw
	`, ifrate.TargetKind, ifrate.MetricInUtilization, ifrate.MetricOutUtilization, from, to,
		ifrate.TargetKind, ifrate.MetricInUtilization, ifrate.MetricOutUtilization, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying interface utilization: %w", err)
	}
	defer rows.Close()

	type direction struct {
		k      ifKey
		hour   string
		metric string
	}
	seen := make(map[direction]bool)
	busiest := make(map[ifKey]map[string]float64)
	for rows.Next() {
		var client, targetID, hour, metric string
		var value float64
		if scanErr := rows.Scan(&client, &targetID, &hour, &metric, &value); scanErr != nil {
			return nil, fmt.Errorf("scanning interface utilization: %w", scanErr)
		}
		k, parseErr := parseIfKey(client, targetID)
		if parseErr != nil {
			return nil, parseErr
		}
		d := direction{k: k, hour: hour, metric: metric}
		if seen[d] {
			continue
		}
		seen[d] = true
		if busiest[k] == nil {
			busiest[k] = make(map[string]float64)
		}
		busiest[k][hour] = max(busiest[k][hour], value)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating interface utilization: %w", rowsErr)
	}
	if len(busiest) == 0 {
		return nil, nil
	}

	targets, err := r.targetNames(ctx)
	if err != nil {
		return nil, err
	}
	ifNames, err := r.interfaceNames(ctx)
	if err != nil {
		return nil, err
	}
	keys := slices.SortedFunc(maps.Keys(busiest), func(a, b ifKey) int {
		return cmp.Or(
			cmp.Compare(a.client, b.client),
			cmp.Compare(a.target, b.target),
			cmp.Compare(a.ifIndex, b.ifIndex),
		)
	})
	history := make([]reporting.InterfaceUtilization, 0, len(keys))
	for _, k := range keys {
		u := reporting.InterfaceUtilization{
			Target:  cmp.Or(targets[k.target], k.target),
			IfIndex: k.ifIndex,
			IfName:  ifNames[k],
		}
		for _, hour := range slices.Sorted(maps.Keys(busiest[k])) {
			start, parseErr := time.Parse(hourLayout, hour)
			if parseErr != nil {
				return nil, fmt.Errorf("parsing utilization hour %q: %w", hour, parseErr)
			}
			u.Hours = append(u.Hours, forecast.Point{At: start.Add(hourMiddle), Value: busiest[k][hour]})
		}
		history = append(history, u)
	}
	return history, nil
}
