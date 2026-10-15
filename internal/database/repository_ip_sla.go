package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

// percentScale reads a 0..1 share as a percentage.
const percentScale = 100

// IPSLAOperations lists the IP-SLA operations of every polling target of
// clientID, from that target's most recent ip_sla observation. A target that
// was deleted, or never ran the ip_sla collector, contributes nothing.
func (r *MetricsRepository) IPSLAOperations(ctx context.Context, clientID string) ([]sla.Operation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.target_id, pt.name, o.observed_at, o.payload_json
		FROM snmp_observations o
		JOIN polling_targets pt ON pt.client_id = o.client_id AND pt.id = o.target_id
		WHERE o.client_id = ? AND o.kind = ?
		  AND o.id = (
			SELECT latest.id FROM snmp_observations latest
			WHERE latest.client_id = o.client_id AND latest.target_id = o.target_id
			  AND latest.kind = o.kind
			ORDER BY latest.observed_at DESC, latest.id DESC LIMIT 1
		  )
		ORDER BY o.target_id
	`, clientID, sla.TargetKind)
	if err != nil {
		return nil, fmt.Errorf("query ip-sla operations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []sla.Operation
	for rows.Next() {
		var targetID, targetName, stamp, payload string
		if scanErr := rows.Scan(&targetID, &targetName, &stamp, &payload); scanErr != nil {
			return nil, fmt.Errorf("scan ip-sla operations: %w", scanErr)
		}
		observedAt, parseErr := time.Parse(time.RFC3339Nano, stamp)
		if parseErr != nil {
			return nil, fmt.Errorf("parse ip-sla observation time %q: %w", stamp, parseErr)
		}
		var obs ipsla.Observation
		if jsonErr := json.Unmarshal([]byte(payload), &obs); jsonErr != nil {
			return nil, fmt.Errorf("decode ip-sla observation of %s: %w", targetID, jsonErr)
		}
		for _, op := range obs.Operations {
			out = append(out, sla.Operation{
				TargetID: targetID, TargetName: targetName, ObservedAt: observedAt, Operation: op,
			})
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("ip-sla operations rows: %w", rowsErr)
	}
	return out, nil
}

// IPSLARuns reads the runs RecordSLAResults stored for one operation of
// clientID in (from, to], oldest first.
func (r *MetricsRepository) IPSLARuns(
	ctx context.Context, clientID, targetID string, index int, from, to time.Time,
) ([]sla.Run, error) {
	rows, err := r.db.Query(ctx, `
		SELECT timestamp, metric_type, value FROM metrics
		WHERE interface_name = ? AND client_id = ? AND target_kind = ?
		  AND timestamp > ? AND timestamp <= ?
		ORDER BY timestamp
	`, targetID+"/"+strconv.Itoa(index), clientID, sla.TargetKind,
		from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query ip-sla runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []sla.Run
	for rows.Next() {
		var (
			stamp, metric string
			value         float64
		)
		if scanErr := rows.Scan(&stamp, &metric, &value); scanErr != nil {
			return nil, fmt.Errorf("scan ip-sla run: %w", scanErr)
		}
		at, parseErr := time.Parse(time.RFC3339, stamp)
		if parseErr != nil {
			return nil, fmt.Errorf("parse ip-sla run time %q: %w", stamp, parseErr)
		}
		// Rows arrive grouped by run, one per stored point.
		if n := len(out); n == 0 || !out[n-1].At.Equal(at) {
			out = append(out, sla.Run{At: at})
		}
		setRun(&out[len(out)-1], metric, value)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("ip-sla runs rows: %w", rowsErr)
	}
	return out, nil
}

// setRun files one stored point under its field. ip_sla_completed is stored
// as 0 or 1 and read as a percentage.
func setRun(r *sla.Run, metric string, v float64) {
	switch metric {
	case sla.MetricCompleted:
		r.CompletedPct = v * percentScale
	case sla.MetricRTT:
		r.RTTMs = &v
	case sla.MetricJitter:
		r.JitterMs = &v
	case sla.MetricLoss:
		r.LossPct = &v
	case sla.MetricMOS:
		r.MOS = &v
	}
}
