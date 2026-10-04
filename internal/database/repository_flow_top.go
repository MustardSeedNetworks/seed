package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// repository_flow_top.go is the read side of the flow store: top talkers and
// top conversations over a window. The caller (internal/api) picks the tier
// from the licence horizons, as it does for history: raw for any window the
// raw horizon covers, a rollup table past it.

// FlowTier names the table a top-N read aggregates.
type FlowTier int

// Flow tiers, finest first.
const (
	// FlowTierRaw aggregates flow_records, including the hour in progress.
	FlowTierRaw FlowTier = iota
	// FlowTierHourly reads flow_conversations_hourly; closed hours only.
	FlowTierHourly
	// FlowTierDaily reads flow_conversations_daily; closed days only.
	FlowTierDaily
)

// FlowRank is the counter a top-N list is ordered by.
type FlowRank string

// Flow rankings, as the wire spells them.
const (
	FlowRankBytes   FlowRank = "bytes"
	FlowRankPackets FlowRank = "packets"
)

// FlowTalker is one host's traffic over the window, sent and received.
type FlowTalker struct {
	Addr    string `json:"addr"`
	Bytes   int64  `json:"bytes"`
	Packets int64  `json:"packets"`
}

// FlowConversation is the traffic between two hosts over one protocol, both
// directions summed. AddrA and AddrB are ordered as text so each pair has one
// row; the order carries no meaning.
type FlowConversation struct {
	AddrA    string `json:"addrA"`
	AddrB    string `json:"addrB"`
	Protocol int    `json:"protocol"`
	Bytes    int64  `json:"bytes"`
	Packets  int64  `json:"packets"`
}

// flowWindowSQL returns the CTE body that selects the window's directional
// rows from the tier's table, and its bounds in that table's time format.
// A rollup bucket is in the window when it overlaps it, so the partial buckets
// at either end count whole. The end bucket matters: the retention pass
// re-writes the day in progress, and a window ending now must include it.
func flowWindowSQL(tier FlowTier, from, to time.Time) (string, string, string, error) {
	switch tier {
	case FlowTierRaw:
		return `SELECT src_addr, dst_addr, protocol, bytes, packets FROM flow_records
			WHERE client_id = ? AND flow_end >= ? AND flow_end < ?`,
			from.UTC().Format(flowTimeFormat), to.UTC().Format(flowTimeFormat), nil
	case FlowTierHourly:
		return `SELECT src_addr, dst_addr, protocol, bytes, packets FROM flow_conversations_hourly
			WHERE client_id = ? AND hour_bucket >= ? AND hour_bucket <= ?`,
			from.UTC().Format(hourFormat), to.UTC().Format(hourFormat), nil
	case FlowTierDaily:
		return `SELECT src_addr, dst_addr, protocol, bytes, packets FROM flow_conversations_daily
			WHERE client_id = ? AND day_bucket >= ? AND day_bucket <= ?`,
			from.UTC().Format(dayFormat), to.UTC().Format(dayFormat), nil
	}
	return "", "", "", fmt.Errorf("unknown flow tier %d", tier)
}

// flowOrderSQL orders by the ranked counter, then the other, then the key, so
// equal rows come back in a stable order.
func flowOrderSQL(by FlowRank, key string) (string, error) {
	switch by {
	case FlowRankBytes:
		return "ORDER BY bytes DESC, packets DESC, " + key, nil
	case FlowRankPackets:
		return "ORDER BY packets DESC, bytes DESC, " + key, nil
	}
	return "", fmt.Errorf("unknown flow rank %q", by)
}

// flowTopRead is one top-N read over a window.
type flowTopRead struct {
	clientID string
	tier     FlowTier
	from, to time.Time
	by       FlowRank
	limit    int
}

// queryFlowTop runs a top-N read and scans each row into a T. windowSQL
// picks the tier's table, which body selects from as the CTE f; key breaks
// ties in the ranking. The window and order fragments are constants, never
// caller input; every value is bound.
func queryFlowTop[T any](
	ctx context.Context, db *DB, op string, q flowTopRead,
	windowSQL func(FlowTier, time.Time, time.Time) (string, string, string, error),
	body, key string, scan func(*sql.Rows, *T) error,
) ([]T, error) {
	window, lo, hi, err := windowSQL(q.tier, q.from, q.to)
	if err != nil {
		return nil, err
	}
	order, err := flowOrderSQL(q.by, key)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, "WITH f AS ("+window+")\n"+body+" "+order+" LIMIT ?",
		q.clientID, lo, hi, q.limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = rows.Close() }()

	var out []T
	for rows.Next() {
		var v T
		if scanErr := scan(rows, &v); scanErr != nil {
			return nil, fmt.Errorf("scan %s: %w", op, scanErr)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// TopTalkers returns the limit hosts that sent and received the most over
// [from, to). A flow counts towards both of its ends, once each.
func (r *FlowRecordsRepository) TopTalkers(
	ctx context.Context, clientID string, tier FlowTier, from, to time.Time, by FlowRank, limit int,
) ([]FlowTalker, error) {
	return queryFlowTop(ctx, r.db, "top talkers", flowTopRead{clientID, tier, from, to, by, limit},
		flowWindowSQL, `
		SELECT addr, CAST(TOTAL(b) AS INTEGER) AS bytes, CAST(TOTAL(p) AS INTEGER) AS packets
		FROM (
		  SELECT src_addr AS addr, bytes AS b, packets AS p FROM f
		  UNION ALL
		  SELECT dst_addr, bytes, packets FROM f WHERE dst_addr <> src_addr
		)
		GROUP BY addr`, "addr",
		func(rows *sql.Rows, t *FlowTalker) error { return rows.Scan(&t.Addr, &t.Bytes, &t.Packets) })
}

// TopConversations returns the limit host pairs, per protocol, that carried
// the most over [from, to), both directions together.
func (r *FlowRecordsRepository) TopConversations(
	ctx context.Context, clientID string, tier FlowTier, from, to time.Time, by FlowRank, limit int,
) ([]FlowConversation, error) {
	return queryFlowTop(ctx, r.db, "top conversations", flowTopRead{clientID, tier, from, to, by, limit},
		flowWindowSQL, `
		SELECT MIN(src_addr, dst_addr) AS addr_a, MAX(src_addr, dst_addr) AS addr_b, protocol,
		       CAST(TOTAL(bytes) AS INTEGER) AS bytes, CAST(TOTAL(packets) AS INTEGER) AS packets
		FROM f
		GROUP BY addr_a, addr_b, protocol`, "addr_a, addr_b, protocol",
		func(rows *sql.Rows, c *FlowConversation) error {
			return rows.Scan(&c.AddrA, &c.AddrB, &c.Protocol, &c.Bytes, &c.Packets)
		})
}
