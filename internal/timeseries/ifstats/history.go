package ifstats

import (
	"context"
	"time"
)

// HistoryBuckets is how many points a history is drawn from at most. A rated
// poll lands in the bucket its time falls in; a bucket narrower than the poll
// interval holds one poll, so a short range is served unbinned.
const HistoryBuckets = 240

// HistoryRange is a window a history can be read over. The rates are read
// from the raw rows, which every tier keeps for seven days, so the longest
// range needs no tier clamp.
type HistoryRange string

// The ranges a history can be read over.
const (
	RangeHour HistoryRange = "1h"
	RangeDay  HistoryRange = "24h"
	RangeWeek HistoryRange = "7d"
)

const (
	day  = 24 * time.Hour
	week = 7 * day
)

// Duration is the span the range covers; zero for a range not listed above.
func (r HistoryRange) Duration() time.Duration {
	switch r {
	case RangeHour:
		return time.Hour
	case RangeDay:
		return day
	case RangeWeek:
		return week
	}
	return 0
}

// History is one interface's rates over a window, oldest first. Each point is
// one bucket of Bucket width: octets and utilization are the bucket's mean,
// errors and discards its peak, so a single errored poll is not averaged
// away. A point's At is the time of the bucket's last poll. Buckets with no
// poll are absent.
type History struct {
	From   time.Time
	To     time.Time
	Bucket time.Duration
	Points []Rates
}

// History returns the interface's rates over the range ending at to, binned
// to at most HistoryBuckets points. rng must be one of the listed ranges.
func (s *Service) History(
	ctx context.Context, targetID string, ifIndex uint32, rng HistoryRange, to time.Time,
) (History, error) {
	span := rng.Duration()
	h := History{From: to.Add(-span), To: to, Bucket: span / HistoryBuckets}
	polls, err := s.store.InterfaceRates(ctx, targetID, ifIndex, h.From, h.To)
	if err != nil {
		return History{}, err
	}
	h.Points = bin(polls, h.From, h.Bucket)
	return h, nil
}

// bucket accumulates the polls of one bucket.
type bucket struct {
	index               int
	rates               Rates
	inOctets, outOctets mean
	inUtil, outUtil     mean
}

// bin folds polls, oldest first, into buckets of width starting at from.
func bin(polls []Rates, from time.Time, width time.Duration) []Rates {
	var buckets []*bucket
	for _, p := range polls {
		idx := min(int(p.At.Sub(from)/width), HistoryBuckets-1)
		if n := len(buckets); n == 0 || buckets[n-1].index != idx {
			buckets = append(buckets, &bucket{index: idx})
		}
		b := buckets[len(buckets)-1]
		b.rates.At = p.At
		b.rates.InErrors = max(b.rates.InErrors, p.InErrors)
		b.rates.OutErrors = max(b.rates.OutErrors, p.OutErrors)
		b.rates.InDiscards = max(b.rates.InDiscards, p.InDiscards)
		b.rates.OutDiscards = max(b.rates.OutDiscards, p.OutDiscards)
		b.inOctets.add(p.InOctets)
		b.outOctets.add(p.OutOctets)
		b.inUtil.add(p.InUtilization)
		b.outUtil.add(p.OutUtilization)
	}
	out := make([]Rates, len(buckets))
	for i, b := range buckets {
		out[i] = b.rates
		out[i].InOctets = b.inOctets.value()
		out[i].OutOctets = b.outOctets.value()
		out[i].InUtilization = b.inUtil.value()
		out[i].OutUtilization = b.outUtil.value()
	}
	return out
}

// mean is the running mean of the values present; a poll that could not rate
// a value does not pull the mean towards zero.
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
