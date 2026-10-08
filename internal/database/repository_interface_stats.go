package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
)

// InterfaceStats lists every interface a polling target of clientID has
// reported, joined to the points of its most recent rated poll. The latest
// poll is found by its error rate, which RecordInterfaceRates stores on every
// rated poll; octets and utilization may be absent from it.
func (r *MetricsRepository) InterfaceStats(ctx context.Context, clientID string) ([]ifstats.Interface, error) {
	rows, err := r.db.Query(ctx, `
		WITH ifs AS (
			SELECT pt.id AS target_id, pt.name AS target_name, ti.if_index,
			       COALESCE(NULLIF(ti.if_name, ''), ti.if_descr, '') AS name,
			       COALESCE(ti.if_alias, '') AS alias,
			       COALESCE(ti.if_oper_status, 0) AS oper,
			       COALESCE(ti.speed_bps, 0) AS speed,
			       pt.id || '/' || ti.if_index AS rate_key
			FROM polling_targets pt
			JOIN topology_target_nodes ttn ON ttn.client_id = pt.client_id AND ttn.target_id = pt.id
			JOIN topology_interfaces ti ON ti.node_id = ttn.node_id
			WHERE pt.client_id = ?
		), latest AS (
			SELECT ifs.target_id, ifs.target_name, ifs.if_index, ifs.name, ifs.alias,
			       ifs.oper, ifs.speed, ifs.rate_key, (
				SELECT MAX(m.timestamp) FROM metrics m
				WHERE m.interface_name = ifs.rate_key AND m.metric_type = ?
				  AND m.client_id = ? AND m.target_kind = ?
			) AS sampled_at
			FROM ifs
		)
		SELECT l.target_id, l.target_name, l.if_index, l.name, l.alias, l.oper, l.speed,
		       l.sampled_at, m.metric_type, m.value
		FROM latest l
		LEFT JOIN metrics m ON m.interface_name = l.rate_key AND m.timestamp = l.sampled_at
		  AND m.client_id = ? AND m.target_kind = ?
		ORDER BY l.target_id, l.if_index
	`, clientID, ifrate.MetricInErrors, clientID, ifrate.TargetKind, clientID, ifrate.TargetKind)
	if err != nil {
		return nil, fmt.Errorf("query interface stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []ifstats.Interface
	for rows.Next() {
		var (
			iface     ifstats.Interface
			oper      int
			sampledAt sql.NullString
			metric    sql.NullString
			value     sql.NullFloat64
		)
		if scanErr := rows.Scan(&iface.TargetID, &iface.TargetName, &iface.IfIndex, &iface.Name,
			&iface.Alias, &oper, &iface.SpeedBps, &sampledAt, &metric, &value); scanErr != nil {
			return nil, fmt.Errorf("scan interface stats: %w", scanErr)
		}
		// Rows arrive grouped by interface, one per stored point.
		if n := len(out); n == 0 || out[n-1].TargetID != iface.TargetID || out[n-1].IfIndex != iface.IfIndex {
			iface.OperStatus = ifstats.OperStatusFromMIB(oper)
			if sampledAt.Valid {
				at, parseErr := time.Parse(time.RFC3339, sampledAt.String)
				if parseErr != nil {
					return nil, fmt.Errorf("parse interface rate time %q: %w", sampledAt.String, parseErr)
				}
				iface.Rates = &ifstats.Rates{At: at}
			}
			out = append(out, iface)
		}
		if rates := out[len(out)-1].Rates; rates != nil && metric.Valid {
			setRate(rates, metric.String, value.Float64)
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("interface stats rows: %w", rowsErr)
	}
	return out, nil
}

// setRate files one stored point under its field. Points the list does not
// show (the EtherLike-MIB rates) are left out.
func setRate(r *ifstats.Rates, metric string, v float64) {
	switch metric {
	case ifrate.MetricInOctets:
		r.InOctets = &v
	case ifrate.MetricOutOctets:
		r.OutOctets = &v
	case ifrate.MetricInUtilization:
		r.InUtilization = &v
	case ifrate.MetricOutUtilization:
		r.OutUtilization = &v
	case ifrate.MetricInErrors:
		r.InErrors = v
	case ifrate.MetricOutErrors:
		r.OutErrors = v
	case ifrate.MetricInDiscards:
		r.InDiscards = v
	case ifrate.MetricOutDiscards:
		r.OutDiscards = v
	}
}

// InterfaceRates reads the rated polls RecordInterfaceRates stored for one
// interface of clientID in (from, to], oldest first.
func (r *MetricsRepository) InterfaceRates(
	ctx context.Context, clientID, targetID string, ifIndex uint32, from, to time.Time,
) ([]ifstats.Rates, error) {
	rows, err := r.db.Query(ctx, `
		SELECT timestamp, metric_type, value FROM metrics
		WHERE interface_name = ? AND client_id = ? AND target_kind = ?
		  AND timestamp > ? AND timestamp <= ?
		ORDER BY timestamp
	`, rateKey(targetID, ifIndex), clientID, ifrate.TargetKind,
		from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query interface rates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []ifstats.Rates
	for rows.Next() {
		var (
			stamp, metric string
			value         float64
		)
		if scanErr := rows.Scan(&stamp, &metric, &value); scanErr != nil {
			return nil, fmt.Errorf("scan interface rate: %w", scanErr)
		}
		at, parseErr := time.Parse(time.RFC3339, stamp)
		if parseErr != nil {
			return nil, fmt.Errorf("parse interface rate time %q: %w", stamp, parseErr)
		}
		// Rows arrive grouped by poll, one per stored point.
		if n := len(out); n == 0 || !out[n-1].At.Equal(at) {
			out = append(out, ifstats.Rates{At: at})
		}
		setRate(&out[len(out)-1], metric, value)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("interface rates rows: %w", rowsErr)
	}
	return out, nil
}

// rateKey is the target_id RecordInterfaceRates keys an interface's points by.
func rateKey(targetID string, ifIndex uint32) string {
	return targetID + "/" + strconv.FormatUint(uint64(ifIndex), 10)
}
