package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

// windowBounds returns the bounds of window as SQL parameters for
// `col >= ? AND col < ?` over a UTC timestamp column.
//
// The stores write RFC 3339 with a Z (metrics, alerts) or RFC 3339 with
// trailing fractional seconds (topology, probes), and those compare wrongly as
// strings at a whole second: "…00.5Z" sorts before "…00Z". A bound with no zone
// suffix is a prefix of both forms of its own second, so it sorts below both
// and the comparison is exact for either.
func windowBounds(window reporting.DateRange) (string, string) {
	const layout = "2006-01-02T15:04:05"
	return window.Start.UTC().Format(layout), window.End.UTC().Format(layout)
}

// ifKey names one interface of one polling target.
type ifKey struct {
	client, target string
	ifIndex        uint32
}

// InterfaceHealth returns the rated interfaces' utilization and error and
// discard rates in the window, named by polling target and, once the
// topology has seen the interface, by ifName.
func (r *MetricsRepo) InterfaceHealth(
	ctx context.Context,
	window reporting.DateRange,
) ([]reporting.InterfaceHealth, error) {
	from, to := windowBounds(window)
	rows, err := r.db.Query(ctx, `
		SELECT m.client_id, m.target_id,
			MAX(CASE WHEN m.metric_type IN (?, ?) THEN m.value END),
			AVG(CASE WHEN m.metric_type = ? THEN m.value END),
			AVG(CASE WHEN m.metric_type = ? THEN m.value END),
			AVG(CASE WHEN m.metric_type = ? THEN m.value END),
			AVG(CASE WHEN m.metric_type = ? THEN m.value END),
			AVG(CASE WHEN m.metric_type = ? THEN m.value END),
			AVG(CASE WHEN m.metric_type = ? THEN m.value END)
		FROM metrics m
		WHERE m.target_kind = ? AND m.timestamp >= ? AND m.timestamp < ?
		GROUP BY m.client_id, m.target_id
		ORDER BY m.target_id
	`, ifrate.MetricInUtilization, ifrate.MetricOutUtilization,
		ifrate.MetricInUtilization, ifrate.MetricOutUtilization,
		ifrate.MetricInErrors, ifrate.MetricOutErrors,
		ifrate.MetricInDiscards, ifrate.MetricOutDiscards,
		ifrate.TargetKind, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying interface rates: %w", err)
	}
	defer rows.Close()

	var keys []ifKey
	var health []reporting.InterfaceHealth
	for rows.Next() {
		var client, targetID string
		var peak, inUtil, outUtil, inErr, outErr, inDisc, outDisc sql.NullFloat64
		if scanErr := rows.Scan(&client, &targetID, &peak, &inUtil, &outUtil,
			&inErr, &outErr, &inDisc, &outDisc); scanErr != nil {
			return nil, fmt.Errorf("scanning interface rates: %w", scanErr)
		}
		// target_id is "<polling target ID>/<ifIndex>" (ifrate.TargetKind).
		target, index, ok := strings.Cut(targetID, "/")
		ifIndex, parseErr := strconv.ParseUint(index, 10, 32)
		if !ok || parseErr != nil {
			return nil, fmt.Errorf("interface rate target %q is not <target>/<ifIndex>", targetID)
		}
		keys = append(keys, ifKey{client: client, target: target, ifIndex: uint32(ifIndex)})
		health = append(health, reporting.InterfaceHealth{
			Target:            target,
			IfIndex:           uint32(ifIndex),
			PeakUtilization:   peak.Float64,
			AvgInUtilization:  inUtil.Float64,
			AvgOutUtilization: outUtil.Float64,
			InErrors:          inErr.Float64,
			OutErrors:         outErr.Float64,
			InDiscards:        inDisc.Float64,
			OutDiscards:       outDisc.Float64,
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating interface rates: %w", rowsErr)
	}
	if len(health) == 0 {
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
	for i, k := range keys {
		if name := targets[k.target]; name != "" {
			health[i].Target = name
		}
		health[i].IfName = ifNames[k]
	}
	return health, nil
}

// targetNames maps polling target ID to its name.
func (r *MetricsRepo) targetNames(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name FROM polling_targets`)
	if err != nil {
		return nil, fmt.Errorf("querying polling target names: %w", err)
	}
	defer rows.Close()

	names := make(map[string]string)
	for rows.Next() {
		var id, name string
		if scanErr := rows.Scan(&id, &name); scanErr != nil {
			return nil, fmt.Errorf("scanning polling target name: %w", scanErr)
		}
		names[id] = name
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating polling target names: %w", rowsErr)
	}
	return names, nil
}

// interfaceNames maps a polling target's interface to the ifName the
// topology recorded for it on the target's node.
func (r *MetricsRepo) interfaceNames(ctx context.Context) (map[ifKey]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tn.client_id, tn.target_id, ti.if_index, ti.if_name
		FROM topology_target_nodes tn
		JOIN topology_interfaces ti ON ti.node_id = tn.node_id
		WHERE ti.if_name IS NOT NULL AND ti.if_name != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("querying interface names: %w", err)
	}
	defer rows.Close()

	names := make(map[ifKey]string)
	for rows.Next() {
		var k ifKey
		var name string
		if scanErr := rows.Scan(&k.client, &k.target, &k.ifIndex, &name); scanErr != nil {
			return nil, fmt.Errorf("scanning interface name: %w", scanErr)
		}
		names[k] = name
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating interface names: %w", rowsErr)
	}
	return names, nil
}

// AlertSeverityCounts returns severity → count of the alerts raised in the
// window, and how many of them are unresolved now.
func (r *MetricsRepo) AlertSeverityCounts(
	ctx context.Context,
	window reporting.DateRange,
) (map[string]int, int, error) {
	from, to := windowBounds(window)
	rows, err := r.db.Query(ctx, `
		SELECT severity, COUNT(*), SUM(CASE WHEN COALESCE(resolved, 0) = 0 THEN 1 ELSE 0 END)
		FROM alerts
		WHERE created_at >= ? AND created_at < ?
		GROUP BY severity
	`, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("querying alert counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	open := 0
	for rows.Next() {
		var severity string
		var count, unresolved int
		if scanErr := rows.Scan(&severity, &count, &unresolved); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning alert count: %w", scanErr)
		}
		counts[severity] = count
		open += unresolved
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating alert counts: %w", rowsErr)
	}
	return counts, open, nil
}

// TopologyChanges returns the nodes and links first seen in the window, in
// the order they appeared.
func (r *MetricsRepo) TopologyChanges(
	ctx context.Context,
	window reporting.DateRange,
) (reporting.TopologyChanges, error) {
	var changes reporting.TopologyChanges
	from, to := windowBounds(window)

	rows, err := r.db.Query(ctx, `
		SELECT display_name, COALESCE(primary_ip, ''), first_seen
		FROM topology_nodes
		WHERE first_seen >= ? AND first_seen < ?
		ORDER BY first_seen, display_name
	`, from, to)
	if err != nil {
		return changes, fmt.Errorf("querying new topology nodes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n reporting.TopologyNode
		var firstSeen string
		if scanErr := rows.Scan(&n.Name, &n.Address, &firstSeen); scanErr != nil {
			return changes, fmt.Errorf("scanning new topology node: %w", scanErr)
		}
		if n.FirstSeen, err = time.Parse(time.RFC3339Nano, firstSeen); err != nil {
			return changes, fmt.Errorf("parsing topology node first_seen: %w", err)
		}
		changes.NewNodes = append(changes.NewNodes, n)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return changes, fmt.Errorf("iterating new topology nodes: %w", rowsErr)
	}

	links, err := r.db.Query(ctx, `
		SELECT s.display_name, COALESCE(l.source_interface, ''),
			t.display_name, COALESCE(l.target_interface, ''),
			l.link_type, l.first_seen
		FROM topology_links l
		JOIN topology_nodes s ON s.id = l.source_node_id
		JOIN topology_nodes t ON t.id = l.target_node_id
		WHERE l.first_seen >= ? AND l.first_seen < ?
		ORDER BY l.first_seen, l.id
	`, from, to)
	if err != nil {
		return changes, fmt.Errorf("querying new topology links: %w", err)
	}
	defer links.Close()
	for links.Next() {
		var l reporting.TopologyLink
		var firstSeen string
		if scanErr := links.Scan(&l.Source, &l.SourceInterface, &l.Target, &l.TargetInterface,
			&l.Type, &firstSeen); scanErr != nil {
			return changes, fmt.Errorf("scanning new topology link: %w", scanErr)
		}
		if l.FirstSeen, err = time.Parse(time.RFC3339Nano, firstSeen); err != nil {
			return changes, fmt.Errorf("parsing topology link first_seen: %w", err)
		}
		changes.NewLinks = append(changes.NewLinks, l)
	}
	if linksErr := links.Err(); linksErr != nil {
		return changes, fmt.Errorf("iterating new topology links: %w", linksErr)
	}
	return changes, nil
}

// ProbeOutcomes returns each probe's checks and failures in the window, and
// the mean latency of its successful checks.
func (r *MetricsRepo) ProbeOutcomes(
	ctx context.Context,
	window reporting.DateRange,
) ([]reporting.ProbeOutcome, error) {
	from, to := windowBounds(window)
	rows, err := r.db.Query(ctx, `
		SELECT p.display_name, p.kind, COUNT(*),
			SUM(CASE WHEN r.success = 0 THEN 1 ELSE 0 END),
			AVG(CASE WHEN r.success = 1 THEN r.latency_ms END)
		FROM probe_results r
		JOIN probes p ON p.id = r.probe_id
		WHERE r.timestamp >= ? AND r.timestamp < ?
		GROUP BY r.probe_id
		ORDER BY p.display_name
	`, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying probe results: %w", err)
	}
	defer rows.Close()

	var outcomes []reporting.ProbeOutcome
	for rows.Next() {
		var o reporting.ProbeOutcome
		var latency sql.NullFloat64
		if scanErr := rows.Scan(&o.Name, &o.Kind, &o.Checks, &o.Failures, &latency); scanErr != nil {
			return nil, fmt.Errorf("scanning probe result: %w", scanErr)
		}
		o.AvgLatencyMs = latency.Float64
		outcomes = append(outcomes, o)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating probe results: %w", rowsErr)
	}
	return outcomes, nil
}
