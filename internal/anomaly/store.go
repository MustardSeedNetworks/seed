package anomaly

import (
	"context"
	"time"
)

// Source identifies the subsystem that produced an anomaly instance. The single
// shared store stays source-neutral while remaining filterable by producer
// (ADR-0021); new sources are added here as they come online.
type Source string

const (
	// SourceWiFi is the Wi-Fi visibility/troubleshooting stack (internal/wifi).
	SourceWiFi Source = "wifi"
	// SourceWired is wired/link health (duplex, errors, STP/VLAN).
	SourceWired Source = "wired"
	// SourceSNMP is SNMP polling (interface errors, utilization, device down).
	SourceSNMP Source = "snmp"
	// SourceBluetooth is Bluetooth live capture.
	SourceBluetooth Source = "bluetooth"
	// SourceProbe is the active-monitoring probe engine (internal/probe): the
	// recurring DNS/TLS/ping/HTTP/… observations whose threshold breaches become
	// anomalies (ADR-0025). Replaces the never-built "health" source name — probe
	// spans far more than endpoint latency.
	SourceProbe Source = "probe"
	// SourceSecurity is the security/authorization framework.
	SourceSecurity Source = "security"
	// SourceAutoTest is automated test outcomes (RFC/standards conformance).
	SourceAutoTest Source = "autotest"
)

// Record is the persistence-facing view of one live anomaly instance: the
// projected Anomaly plus the columns the in-memory model lacks — a stable id,
// the producing Source, and the resolve/acknowledge lifecycle (ADR-0021). The
// catalog-static Impact and FollowUps are intentionally NOT persisted; they are
// re-derived from the catalog by DefKey on read, so the store never duplicates
// static catalog copy.
type Record struct {
	ID      string
	Source  Source
	Anomaly Anomaly

	Resolved       bool
	ResolvedAt     time.Time
	AcknowledgedBy string
	AcknowledgedAt time.Time
}

// RecordID derives the stable persistence id for an instance from its coalescing
// identity (anomaly type + subject), so a restart re-loads the exact row a
// re-detection updates. It MUST match the engine's in-memory key (see
// Detection.key) — that equality is the restart-idempotency guarantee.
func RecordID(defKey string, subject SubjectRef) string {
	return defKey + "|" + string(subject.Kind) + "|" + subject.ID
}

// Store is the persistence port the engine's Coordinator writes anomaly
// instances through (ADR-0021), implemented by internal/database. It is the
// single SQL system of record for detected anomalies across every source.
type Store interface {
	// Upsert idempotently persists instances keyed by Record.ID: write-through
	// on a material change and in batches from the periodic Flush. A re-detected
	// instance clears any prior resolution (it is live again).
	Upsert(ctx context.Context, recs []Record) error
	// MarkResolved flags the given instance ids resolved as of at (their
	// condition cleared on Prune). Unknown or already-resolved ids are ignored.
	MarkResolved(ctx context.Context, ids []string, at time.Time) error
	// LoadActive returns all unresolved instances, to repopulate the engine on
	// start. (Wired by a later phase; defined now so the port is complete.)
	LoadActive(ctx context.Context) ([]Record, error)
}
