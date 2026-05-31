package database

import (
	"context"
	"fmt"
	"time"
)

// listQueryBuilder is the shared filter-and-pagination scaffold used
// by the snmp_observations + listener_events List methods. Both
// queries share the same shape — WHERE 1=1 + N optional filters +
// ORDER BY observed_at DESC + LIMIT — so the SQL string assembly is
// factored here. Adding a new filter is one Where call at the
// caller; the table-specific column list stays at the call site.
type listQueryBuilder struct {
	base string
	args []any
}

// newListQueryBuilder returns a builder for a SELECT that already
// includes a WHERE 1=1 clause so callers can append AND-clauses
// without tracking whether a WHERE was emitted yet.
func newListQueryBuilder(base string) *listQueryBuilder {
	return &listQueryBuilder{base: base}
}

// Where appends ` AND <clause>` when value is non-empty.
func (b *listQueryBuilder) Where(clause string, value string) *listQueryBuilder {
	if value == "" {
		return b
	}
	b.base += " " + clause
	b.args = append(b.args, value)
	return b
}

// WhereTime appends ` AND <clause>` when value is non-zero,
// formatting the time as RFC3339Nano UTC.
func (b *listQueryBuilder) WhereTime(clause string, value time.Time) *listQueryBuilder {
	if value.IsZero() {
		return b
	}
	b.base += " " + clause
	b.args = append(b.args, value.UTC().Format(time.RFC3339Nano))
	return b
}

// OrderLimit appends an ORDER BY + LIMIT clause. limit is clamped
// to (defaultLimit, maxLimit) when out of range.
func (b *listQueryBuilder) OrderLimit(orderBy string, limit, defaultLimit, maxLimit int) *listQueryBuilder {
	b.base += " " + orderBy + " LIMIT ?"
	switch {
	case limit <= 0:
		limit = defaultLimit
	case limit > maxLimit:
		limit = maxLimit
	}
	b.args = append(b.args, limit)
	return b
}

// Build returns the final query string and the args slice.
func (b *listQueryBuilder) Build() (string, []any) {
	return b.base, b.args
}

// queryRows runs a SELECT and scans each row via scanOne. The
// operation label is used to wrap query + iter errors. Eliminates
// the (query, defer rows.Close, for rows.Next, rows.Err) boilerplate
// every per-table List method would otherwise duplicate.
func queryRows[T any](
	ctx context.Context,
	db *DB,
	query string,
	args []any,
	scanOne func(scan func(...any) error) (*T, error),
	operation string,
) ([]*T, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	defer func() { _ = rows.Close() }()

	var out []*T
	for rows.Next() {
		item, scanErr := scanOne(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("%s iter: %w", operation, rowsErr)
	}
	return out, nil
}
