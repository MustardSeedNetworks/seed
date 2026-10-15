package sla

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
)

// ErrUnavailable is returned when the store cannot be reached (no database).
var ErrUnavailable = errors.New("ip-sla results unavailable")

// Operation is one IP-SLA operation of one polling target, as that target's
// most recent ip_sla poll reported it.
type Operation struct {
	ipsla.Operation

	TargetID   string
	TargetName string
	ObservedAt time.Time
}

// Run is one recorded run of an operation, or in a History the Runs runs of
// one bucket. CompletedPct is the share of them that reached the target, so 0
// or 100 for a single run. The other values are the mean over the runs that
// carry them, and absent when none does: RTT needs a completed run, and
// jitter, loss and MOS a jitter-type operation.
type Run struct {
	At           time.Time
	Runs         int
	CompletedPct float64
	RTTMs        *float64
	JitterMs     *float64
	LossPct      *float64
	MOS          *float64
}

// Reader reads every target's operations from its latest poll, and the runs
// recorded for one operation in (from, to], oldest first, one per run with
// Runs unset.
type Reader interface {
	Operations(ctx context.Context) ([]Operation, error)
	Runs(ctx context.Context, targetID string, index int, from, to time.Time) ([]Run, error)
}

// Service answers IP-SLA reads from a Reader.
type Service struct {
	store Reader
}

// NewService builds the IP-SLA read use-case over store.
func NewService(store Reader) *Service {
	return &Service{store: store}
}

// List returns every operation, the ones worth looking at first: a latest
// run that failed, then one over its threshold, then the healthy ones, then
// those that have not run. Ties go by target, then operation index.
func (s *Service) List(ctx context.Context) ([]Operation, error) {
	ops, err := s.store.Operations(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(ops, func(a, b Operation) int {
		return cmp.Or(
			cmp.Compare(health(a.Latest), health(b.Latest)),
			cmp.Compare(a.TargetName, b.TargetName),
			cmp.Compare(a.TargetID, b.TargetID),
			cmp.Compare(a.Index, b.Index),
		)
	})
	return ops, nil
}

// The ranks of a latest run in List, worst first.
const (
	rankFailed = iota
	rankOverThreshold
	rankHealthy
	rankNeverRun
)

func health(r *ipsla.Result) int {
	switch {
	case r == nil:
		return rankNeverRun
	case !r.Completed():
		return rankFailed
	case r.Sense == ipsla.SenseOverThreshold:
		return rankOverThreshold
	}
	return rankHealthy
}

// History is one operation's runs over a window, oldest first, binned into
// buckets of Bucket width. A point's At is the time of the bucket's last run;
// buckets with no run are absent.
type History struct {
	From   time.Time
	To     time.Time
	Bucket time.Duration
	Points []Run
}

// History returns the operation's runs over the range ending at to, binned
// to at most ifstats.HistoryBuckets points. The runs are read from the raw
// metrics rows, which every tier keeps for the longest range, as the
// interface history is. rng must be one of the ifstats ranges.
func (s *Service) History(
	ctx context.Context, targetID string, index int, rng ifstats.HistoryRange, to time.Time,
) (History, error) {
	span := rng.Duration()
	h := History{From: to.Add(-span), To: to, Bucket: span / ifstats.HistoryBuckets}
	runs, err := s.store.Runs(ctx, targetID, index, h.From, h.To)
	if err != nil {
		return History{}, err
	}
	h.Points = binRuns(runs, h.From, h.Bucket)
	return h, nil
}

// runBucket accumulates the runs of one bucket.
type runBucket struct {
	index                  int
	at                     time.Time
	runs                   int
	completed              float64
	rtt, jitter, loss, mos mean
}

func binRuns(runs []Run, from time.Time, width time.Duration) []Run {
	var buckets []*runBucket
	for _, r := range runs {
		idx := min(int(r.At.Sub(from)/width), ifstats.HistoryBuckets-1)
		if n := len(buckets); n == 0 || buckets[n-1].index != idx {
			buckets = append(buckets, &runBucket{index: idx})
		}
		b := buckets[len(buckets)-1]
		b.at = r.At
		b.runs++
		b.completed += r.CompletedPct
		b.rtt.add(r.RTTMs)
		b.jitter.add(r.JitterMs)
		b.loss.add(r.LossPct)
		b.mos.add(r.MOS)
	}
	out := make([]Run, len(buckets))
	for i, b := range buckets {
		out[i] = Run{
			At:           b.at,
			Runs:         b.runs,
			CompletedPct: b.completed / float64(b.runs),
			RTTMs:        b.rtt.value(),
			JitterMs:     b.jitter.value(),
			LossPct:      b.loss.value(),
			MOS:          b.mos.value(),
		}
	}
	return out
}

// mean is the running mean of the values present.
type mean struct {
	sum float64
	n   int
}

func (m *mean) add(v *float64) {
	if v != nil {
		m.sum += *v
		m.n++
	}
}

func (m *mean) value() *float64 {
	if m.n == 0 {
		return nil
	}
	v := m.sum / float64(m.n)
	return &v
}
