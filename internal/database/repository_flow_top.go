package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/flows"
)

// repository_flow_top.go is the read side of the flow store: top talkers and
// top conversations over a window (flows.Store). The caller picks the tier.

// flowWindowSQL returns the CTE body that selects the window's directional
// rows from the tier's table, and its bounds in that table's time format.
// A rollup bucket is in the window when it overlaps it, so the partial buckets
// at either end count whole. The end bucket matters: the retention pass
// re-writes the day in progress, and a window ending now must include it.
func flowWindowSQL(tier flows.Tier, from, to time.Time) (string, string, string, error) {
	switch tier {
	case flows.TierRaw:
		return `SELECT src_addr, dst_addr, protocol, bytes, packets FROM flow_records
			WHERE client_id = ? AND flow_end >= ? AND flow_end < ?`,
			from.UTC().Format(flowTimeFormat), to.UTC().Format(flowTimeFormat), nil
	case flows.TierHourly:
		return `SELECT src_addr, dst_addr, protocol, bytes, packets FROM flow_conversations_hourly
			WHERE client_id = ? AND hour_bucket >= ? AND hour_bucket <= ?`,
			from.UTC().Format(hourFormat), to.UTC().Format(hourFormat), nil
	case flows.TierDaily:
		return `SELECT src_addr, dst_addr, protocol, bytes, packets FROM flow_conversations_daily
			WHERE client_id = ? AND day_bucket >= ? AND day_bucket <= ?`,
			from.UTC().Format(dayFormat), to.UTC().Format(dayFormat), nil
	}
	return "", "", "", fmt.Errorf("unknown flow tier %d", tier)
}

// flowOrderSQL orders by the ranked counter, then the other, then the key, so
// equal rows come back in a stable order.
func flowOrderSQL(by flows.Rank, key string) (string, error) {
	switch by {
	case flows.RankBytes:
		return "ORDER BY bytes DESC, packets DESC, " + key, nil
	case flows.RankPackets:
		return "ORDER BY packets DESC, bytes DESC, " + key, nil
	}
	return "", fmt.Errorf("unknown flow rank %q", by)
}

// queryFlowTop runs a top-N read and scans each row into a T. windowSQL
// picks the tier's table, which body selects from as the CTE f; key breaks
// ties in the ranking. The window and order fragments are constants, never
// caller input; every value is bound.
func queryFlowTop[T any](
	ctx context.Context, db *DB, op string, q flows.Query,
	windowSQL func(flows.Tier, time.Time, time.Time) (string, string, string, error),
	body, key string, scan func(*sql.Rows, *T) error,
) ([]T, error) {
	window, lo, hi, err := windowSQL(q.Tier, q.From, q.To)
	if err != nil {
		return nil, err
	}
	order, err := flowOrderSQL(q.By, key)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, "WITH f AS ("+window+")\n"+body+" "+order+" LIMIT ?",
		q.ClientID, lo, hi, q.Limit)
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

// TopTalkers returns the q.Limit hosts that sent and received the most over
// [q.From, q.To). A flow counts towards both of its ends, once each.
func (r *FlowRecordsRepository) TopTalkers(
	ctx context.Context, q flows.Query,
) ([]flows.Talker, error) {
	return queryFlowTop(ctx, r.db, "top talkers", q,
		flowWindowSQL, `
		SELECT addr, CAST(TOTAL(b) AS INTEGER) AS bytes, CAST(TOTAL(p) AS INTEGER) AS packets
		FROM (
		  SELECT src_addr AS addr, bytes AS b, packets AS p FROM f
		  UNION ALL
		  SELECT dst_addr, bytes, packets FROM f WHERE dst_addr <> src_addr
		)
		GROUP BY addr`, "addr",
		func(rows *sql.Rows, t *flows.Talker) error { return rows.Scan(&t.Addr, &t.Bytes, &t.Packets) })
}

// TopConversations returns the q.Limit host pairs, per protocol, that carried
// the most over [q.From, q.To), both directions together.
func (r *FlowRecordsRepository) TopConversations(
	ctx context.Context, q flows.Query,
) ([]flows.Conversation, error) {
	return queryFlowTop(ctx, r.db, "top conversations", q,
		flowWindowSQL, `
		SELECT MIN(src_addr, dst_addr) AS addr_a, MAX(src_addr, dst_addr) AS addr_b, protocol,
		       CAST(TOTAL(bytes) AS INTEGER) AS bytes, CAST(TOTAL(packets) AS INTEGER) AS packets
		FROM f
		GROUP BY addr_a, addr_b, protocol`, "addr_a, addr_b, protocol",
		func(rows *sql.Rows, c *flows.Conversation) error {
			return rows.Scan(&c.AddrA, &c.AddrB, &c.Protocol, &c.Bytes, &c.Packets)
		})
}
