// Package history is the bounded read side of the tiered time-series store
// (#175): a probe's trend and the per-day anomaly count, each over a fixed
// window. Deliberately not a query surface: seed stays handheld-class and
// exposes a fixed window over what it recorded.
//
// Each series has a raw form and a rollup form. The raw form is identical on
// every tier and includes the bucket in progress; the rollup form reaches back
// past the raw horizon. Which one answers a request is decided by the caller
// from the licence tier's horizons (internal/api's resolveHistoryWindow); the
// Store behind the Service (internal/database's HistoryRepository, wired in
// internal/app) runs the SQL.
package history

import (
	"context"
	"time"
)

// ProbeTrendPoint is one bucket of a probe's history. Latencies are over the
// successful runs in the bucket; SampleCount counts every run, so
// SampleCount-SuccessCount is the failure count the UI draws as availability.
type ProbeTrendPoint struct {
	Bucket       time.Time `json:"bucket"`
	SampleCount  int       `json:"sampleCount"`
	SuccessCount int       `json:"successCount"`
	AvgLatencyMs float64   `json:"avgLatencyMs"`
	MinLatencyMs float64   `json:"minLatencyMs"`
	MaxLatencyMs float64   `json:"maxLatencyMs"`
}

// AnomalyDayCount is one day's anomaly census: how many distinct (def,
// subject) anomalies were open on that day.
//
// Deliberately no severity field. The obvious SQL for one — MAX over the
// severity column — is a lexical maximum, and "warning" sorts above
// "critical". The ordering that means anything is anomaly.Severity.rank(),
// which is unexported and lives a package away; restating it in SQL would be
// a second source of truth for the escalation order. A client that needs the
// worst severity on a day asks the anomaly surface for that day.
type AnomalyDayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// Store reads the time-series tables. Every window is [from, to) for a probe
// trend and an inclusive day range for anomaly counts.
type Store interface {
	ProbeTrendRaw(
		ctx context.Context,
		clientID, probeID string,
		from, to time.Time,
		daily bool,
	) ([]ProbeTrendPoint, error)
	ProbeTrendRollup(
		ctx context.Context,
		clientID, probeID string,
		from, to time.Time,
		daily bool,
	) ([]ProbeTrendPoint, error)
	AnomalyCountsByDayLive(ctx context.Context, from, to time.Time) ([]AnomalyDayCount, error)
	AnomalyCountsByDayRollup(ctx context.Context, from, to time.Time) ([]AnomalyDayCount, error)
}

// Service answers history reads from a Store.
type Service struct {
	store Store
}

// NewService builds the history read use-case over store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// ProbeTrendQuery names one probe's series and the window it is read over.
type ProbeTrendQuery struct {
	ClientID string
	ProbeID  string
	From, To time.Time
	// Daily selects day buckets; hour buckets otherwise.
	Daily bool
	// Raw reads probe_results on the fly instead of the rollup tables.
	Raw bool
}

// ProbeTrend reads one probe's bucketed trend.
func (s *Service) ProbeTrend(ctx context.Context, q ProbeTrendQuery) ([]ProbeTrendPoint, error) {
	if q.Raw {
		return s.store.ProbeTrendRaw(ctx, q.ClientID, q.ProbeID, q.From, q.To, q.Daily)
	}
	return s.store.ProbeTrendRollup(ctx, q.ClientID, q.ProbeID, q.From, q.To, q.Daily)
}

// AnomalyCounts reads the per-day anomaly count over the inclusive day range,
// from the live anomalies table when raw, from the daily census otherwise.
func (s *Service) AnomalyCounts(ctx context.Context, from, to time.Time, raw bool) ([]AnomalyDayCount, error) {
	if raw {
		return s.store.AnomalyCountsByDayLive(ctx, from, to)
	}
	return s.store.AnomalyCountsByDayRollup(ctx, from, to)
}
