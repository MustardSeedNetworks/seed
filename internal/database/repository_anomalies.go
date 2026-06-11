package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
)

// AnomalyRepository is the SQL system of record for detected anomalies across
// every source (ADR-0021). It implements anomaly.Store: the engine's persistence
// Coordinator writes instances through here on a material change and in batched
// flushes, marks them resolved on prune, and loads active instances to
// repopulate the engine on start. Catalog-static copy (impact, follow-ups) is not
// stored — it is re-derived from the embedded catalog by def_key on read.
type AnomalyRepository struct {
	db *DB
}

// Compile-time proof the repository satisfies the engine's persistence port.
var _ anomaly.Store = (*AnomalyRepository)(nil)

const anomalyUpsertSQL = `
	INSERT INTO anomalies
	(id, def_key, source, category, severity, subject_kind, subject_id,
	 title, description, recommendation, evidence, standards,
	 count, first_seen, last_seen, resolved_at, is_resolved,
	 acknowledged_by, acknowledged_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, 0, NULL, NULL)
	ON CONFLICT(id) DO UPDATE SET
		severity = excluded.severity,
		title = excluded.title,
		description = excluded.description,
		recommendation = excluded.recommendation,
		evidence = excluded.evidence,
		standards = excluded.standards,
		count = excluded.count,
		last_seen = excluded.last_seen,
		resolved_at = NULL,
		is_resolved = 0`

// Upsert idempotently persists instances by their stable id in one transaction.
// On conflict it refreshes the mutable columns and clears any prior resolution
// (a re-detected instance is live again); first_seen and operator
// acknowledgement are preserved.
func (r *AnomalyRepository) Upsert(ctx context.Context, recs []anomaly.Record) error {
	if len(recs) == 0 {
		return nil
	}
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, anomalyUpsertSQL)
		if err != nil {
			return fmt.Errorf("prepare anomaly upsert: %w", err)
		}
		defer func() { _ = stmt.Close() }()

		for _, rec := range recs {
			if execErr := upsertAnomaly(ctx, stmt, rec); execErr != nil {
				return execErr
			}
		}
		return nil
	})
}

// upsertAnomaly encodes one record's JSON columns and executes the prepared
// upsert.
func upsertAnomaly(ctx context.Context, stmt *sql.Stmt, rec anomaly.Record) error {
	evidence, err := marshalAnomalyJSON(rec.Anomaly.Evidence)
	if err != nil {
		return fmt.Errorf("marshal evidence for %q: %w", rec.ID, err)
	}
	standards, err := marshalAnomalyJSON(rec.Anomaly.Standards)
	if err != nil {
		return fmt.Errorf("marshal standards for %q: %w", rec.ID, err)
	}
	a := rec.Anomaly
	_, err = stmt.ExecContext(ctx,
		rec.ID, a.DefKey, string(rec.Source), string(a.Category), string(a.Severity),
		string(a.Subject.Kind), a.Subject.ID, a.Title, a.Description, a.Recommendation,
		evidence, standards, a.Count,
		a.FirstSeen.UTC().Format(time.RFC3339), a.LastSeen.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert anomaly %q: %w", rec.ID, err)
	}
	return nil
}

// MarkResolved flags the given ids resolved as of at, leaving rows that are
// already resolved untouched (idempotent).
func (r *AnomalyRepository) MarkResolved(ctx context.Context, ids []string, at time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	resolvedAt := at.UTC().Format(time.RFC3339)
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			UPDATE anomalies SET is_resolved = 1, resolved_at = ?
			WHERE id = ? AND is_resolved = 0
		`)
		if err != nil {
			return fmt.Errorf("prepare anomaly resolve: %w", err)
		}
		defer func() { _ = stmt.Close() }()

		for _, id := range ids {
			if _, execErr := stmt.ExecContext(ctx, resolvedAt, id); execErr != nil {
				return fmt.Errorf("resolve anomaly %q: %w", id, execErr)
			}
		}
		return nil
	})
}

// DeleteResolvedOlderThan removes resolved anomalies whose resolved_at predates
// cutoff, bounding table growth on appliances (ADR-0021 retention: TTL-age
// resolved). Active instances are NEVER deleted regardless of age — a long-idle
// but still-open anomaly is kept until it actually resolves. Returns the number
// of rows removed.
func (r *AnomalyRepository) DeleteResolvedOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM anomalies WHERE is_resolved = 1 AND resolved_at < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete resolved anomalies: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete resolved anomalies rows affected: %w", err)
	}
	return affected, nil
}

// LoadActive returns every unresolved instance, ordered by id for determinism,
// so the engine can be repopulated on start.
func (r *AnomalyRepository) LoadActive(ctx context.Context) ([]anomaly.Record, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, def_key, source, category, severity, subject_kind, subject_id,
		       title, description, recommendation, evidence, standards,
		       count, first_seen, last_seen, resolved_at, is_resolved,
		       acknowledged_by, acknowledged_at
		FROM anomalies
		WHERE is_resolved = 0
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query active anomalies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []anomaly.Record
	for rows.Next() {
		rec, scanErr := scanAnomaly(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, rec)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate active anomalies: %w", rowsErr)
	}
	return out, nil
}

// scanAnomaly maps one row back into a Record, decoding the JSON columns and
// nullable lifecycle timestamps.
func scanAnomaly(rows *sql.Rows) (anomaly.Record, error) {
	var (
		rec                        anomaly.Record
		a                          anomaly.Anomaly
		source, category, severity string
		subjectKind                string
		evidence, standards        sql.NullString
		firstSeen, lastSeen        string
		resolvedAt, ackBy, ackAt   sql.NullString
		isResolved                 int
	)
	if err := rows.Scan(
		&rec.ID, &a.DefKey, &source, &category, &severity, &subjectKind, &a.Subject.ID,
		&a.Title, &a.Description, &a.Recommendation, &evidence, &standards,
		&a.Count, &firstSeen, &lastSeen, &resolvedAt, &isResolved, &ackBy, &ackAt,
	); err != nil {
		return anomaly.Record{}, fmt.Errorf("scan anomaly: %w", err)
	}

	a.Category = anomaly.Category(category)
	a.Severity = anomaly.Severity(severity)
	a.Subject.Kind = anomaly.SubjectKind(subjectKind)
	if err := json.Unmarshal([]byte(evidence.String), &a.Evidence); evidence.Valid && err != nil {
		return anomaly.Record{}, fmt.Errorf("unmarshal evidence for %q: %w", rec.ID, err)
	}
	if err := json.Unmarshal([]byte(standards.String), &a.Standards); standards.Valid && err != nil {
		return anomaly.Record{}, fmt.Errorf("unmarshal standards for %q: %w", rec.ID, err)
	}
	a.FirstSeen = parseAnomalyTime(firstSeen)
	a.LastSeen = parseAnomalyTime(lastSeen)

	rec.Source = anomaly.Source(source)
	rec.Anomaly = a
	rec.Resolved = isResolved != 0
	if resolvedAt.Valid {
		rec.ResolvedAt = parseAnomalyTime(resolvedAt.String)
	}
	rec.AcknowledgedBy = ackBy.String
	if ackAt.Valid {
		rec.AcknowledgedAt = parseAnomalyTime(ackAt.String)
	}
	return rec, nil
}

// marshalAnomalyJSON encodes a JSON column, returning a NULL for an empty
// value so absent evidence/standards are stored as NULL rather than "null".
func marshalAnomalyJSON(v any) (sql.NullString, error) {
	switch t := v.(type) {
	case map[string]string:
		if len(t) == 0 {
			return sql.NullString{}, nil
		}
	case []string:
		if len(t) == 0 {
			return sql.NullString{}, nil
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

// parseAnomalyTime parses an RFC3339 timestamp, returning the zero time on a
// malformed value (defensive — the writer always formats RFC3339).
func parseAnomalyTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
