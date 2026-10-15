package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
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

// members answers whether a polling target, an alert source or a probe target
// is one of a device group's members. A nil members admits everything, for a
// summary of the whole network.
type members struct {
	targets map[string]bool
	hosts   map[string]bool
}

func groupMembers(group *reporting.GroupScope) *members {
	if group == nil {
		return nil
	}
	m := &members{targets: make(map[string]bool), hosts: make(map[string]bool)}
	for _, id := range group.Targets {
		m.targets[id] = true
	}
	for _, addr := range group.Addresses {
		m.hosts[devicegroups.Host(addr)] = true
	}
	return m
}

func (m *members) target(id string) bool { return m == nil || m.targets[id] }

// source matches an alert's source the way the group dashboard does: the
// poller's own observations name the polling target id, syslog and traps the
// member's address and port, a flow indicator the bare address.
func (m *members) source(source string) bool {
	return m == nil || m.targets[source] || m.hosts[devicegroups.Host(source)]
}

// probeTarget matches a probe aimed at a member: a URL's host, or a host with
// or without a port. A target that names no host, such as a DSN, matches none.
func (m *members) probeTarget(target string) bool {
	if m == nil {
		return true
	}
	if u, err := url.Parse(target); err == nil && u.Host != "" {
		return m.hosts[devicegroups.Host(u.Hostname())]
	}
	return m.hosts[devicegroups.Host(target)]
}

// targetsJSON is the members' target ids as a JSON array for json_each, or
// NULL for the whole network.
func targetsJSON(group *reporting.GroupScope) (sql.NullString, error) {
	if group == nil {
		return sql.NullString{}, nil
	}
	raw, err := json.Marshal(group.Targets)
	if err != nil {
		return sql.NullString{}, fmt.Errorf("encoding group members: %w", err)
	}
	return sql.NullString{String: string(raw), Valid: true}, nil
}

// ifKey names one interface of one polling target.
type ifKey struct {
	client, target string
	ifIndex        uint32
}

// parseIfKey splits an interface rate's target_id, which is
// "<polling target ID>/<ifIndex>" (ifrate.TargetKind).
func parseIfKey(client, targetID string) (ifKey, error) {
	target, index, ok := strings.Cut(targetID, "/")
	ifIndex, err := strconv.ParseUint(index, 10, 32)
	if !ok || err != nil {
		return ifKey{}, fmt.Errorf("interface rate target %q is not <target>/<ifIndex>", targetID)
	}
	return ifKey{client: client, target: target, ifIndex: uint32(ifIndex)}, nil
}

// InterfaceHealth returns the rated interfaces' utilization and error and
// discard rates in the window, named by polling target and, once the
// topology has seen the interface, by ifName.
func (r *ReportMetricsRepository) InterfaceHealth(
	ctx context.Context,
	window reporting.DateRange,
	group *reporting.GroupScope,
) ([]reporting.InterfaceHealth, error) {
	scope := groupMembers(group)
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
		k, parseErr := parseIfKey(client, targetID)
		if parseErr != nil {
			return nil, parseErr
		}
		if !scope.target(k.target) {
			continue
		}
		keys = append(keys, k)
		health = append(health, reporting.InterfaceHealth{
			Target:            k.target,
			IfIndex:           k.ifIndex,
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
func (r *ReportMetricsRepository) targetNames(ctx context.Context) (map[string]string, error) {
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
func (r *ReportMetricsRepository) interfaceNames(ctx context.Context) (map[ifKey]string, error) {
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
// window, and how many of them are unresolved now. Rows are per source so a
// group's members' alerts can be told apart.
func (r *ReportMetricsRepository) AlertSeverityCounts(
	ctx context.Context,
	window reporting.DateRange,
	group *reporting.GroupScope,
) (map[string]int, int, error) {
	scope := groupMembers(group)
	from, to := windowBounds(window)
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(source, ''), severity, COUNT(*),
			SUM(CASE WHEN COALESCE(resolved, 0) = 0 THEN 1 ELSE 0 END)
		FROM alerts
		WHERE created_at >= ? AND created_at < ?
		GROUP BY source, severity
	`, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("querying alert counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	open := 0
	for rows.Next() {
		var source, severity string
		var count, unresolved int
		if scanErr := rows.Scan(&source, &severity, &count, &unresolved); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning alert count: %w", scanErr)
		}
		if !scope.source(source) {
			continue
		}
		counts[severity] += count
		open += unresolved
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating alert counts: %w", rowsErr)
	}
	return counts, open, nil
}

// TopologyChanges returns the nodes and links first seen in the window, in
// the order they appeared.
//
// For a group, the new devices are members first seen in the window and the
// new links those with a member at either end.
func (r *ReportMetricsRepository) TopologyChanges(
	ctx context.Context,
	window reporting.DateRange,
	group *reporting.GroupScope,
) (reporting.TopologyChanges, error) {
	var changes reporting.TopologyChanges
	from, to := windowBounds(window)
	targets, err := targetsJSON(group)
	if err != nil {
		return changes, err
	}

	rows, err := r.db.Query(ctx, `
		WITH member_nodes AS (
			SELECT tn.node_id FROM topology_target_nodes tn
			WHERE tn.target_id IN (SELECT value FROM json_each(?))
		)
		SELECT display_name, COALESCE(primary_ip, ''), first_seen
		FROM topology_nodes
		WHERE first_seen >= ? AND first_seen < ?
			AND (? IS NULL OR id IN (SELECT node_id FROM member_nodes))
		ORDER BY first_seen, display_name
	`, targets, from, to, targets)
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
		WITH member_nodes AS (
			SELECT tn.node_id FROM topology_target_nodes tn
			WHERE tn.target_id IN (SELECT value FROM json_each(?))
		)
		SELECT s.display_name, COALESCE(l.source_interface, ''),
			t.display_name, COALESCE(l.target_interface, ''),
			l.link_type, l.first_seen
		FROM topology_links l
		JOIN topology_nodes s ON s.id = l.source_node_id
		JOIN topology_nodes t ON t.id = l.target_node_id
		WHERE l.first_seen >= ? AND l.first_seen < ?
			AND (? IS NULL
				OR l.source_node_id IN (SELECT node_id FROM member_nodes)
				OR l.target_node_id IN (SELECT node_id FROM member_nodes))
		ORDER BY l.first_seen, l.id
	`, targets, from, to, targets)
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
//
// For a group, only the probes aimed at a member's address count.
func (r *ReportMetricsRepository) ProbeOutcomes(
	ctx context.Context,
	window reporting.DateRange,
	group *reporting.GroupScope,
) ([]reporting.ProbeOutcome, error) {
	scope := groupMembers(group)
	from, to := windowBounds(window)
	rows, err := r.db.Query(ctx, `
		SELECT p.display_name, p.kind, p.target, COUNT(*),
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
		var target string
		var latency sql.NullFloat64
		if scanErr := rows.Scan(&o.Name, &o.Kind, &target, &o.Checks, &o.Failures, &latency); scanErr != nil {
			return nil, fmt.Errorf("scanning probe result: %w", scanErr)
		}
		if !scope.probeTarget(target) {
			continue
		}
		o.AvgLatencyMs = latency.Float64
		outcomes = append(outcomes, o)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating probe results: %w", rowsErr)
	}
	return outcomes, nil
}
