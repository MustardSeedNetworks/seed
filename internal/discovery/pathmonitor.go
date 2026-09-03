package discovery

// Continuous path monitoring: the aggregation half of the mtr experience.
//
// A traceroute is a photograph. Repeating it and accumulating per-hop
// statistics is what turns "the path looks like this" into "hop 4 is dropping
// 12% of packets", which is the question an operator actually has.
//
// Nothing here touches the network. The engine consumes completed rounds, so
// the loss and jitter arithmetic is testable without a wire, and the same
// aggregation serves whatever produced the rounds.

import (
	"sort"
	"time"
)

// HopStats accumulates what repeated probing reveals about one hop.
type HopStats struct {
	TTL int `json:"ttl"`

	// Addresses that answered at this TTL, most-seen first. More than one
	// means the path is load-balanced across them rather than unstable, and
	// showing a single "current" address would hide that.
	Addresses []HopAddress `json:"addresses"`

	Sent     int     `json:"sent"`
	Received int     `json:"received"`
	LossPct  float64 `json:"lossPct"`

	LastRTT  time.Duration `json:"lastRtt"`
	BestRTT  time.Duration `json:"bestRtt"`
	WorstRTT time.Duration `json:"worstRtt"`
	AvgRTT   time.Duration `json:"avgRtt"`

	// Jitter is the mean absolute difference between consecutive replies --
	// how much the latency moves, which a caller cannot recover from the
	// average alone. Undefined below two replies, and zero there.
	Jitter time.Duration `json:"jitter"`

	// totals kept out of the wire shape; averages are derived on snapshot.
	sumRTT     time.Duration
	sumJitter  time.Duration
	jitterObs  int
	hasLast    bool
	addrCounts map[string]*HopAddress
}

// HopAddress is one address seen at a TTL, with how often it answered.
type HopAddress struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname,omitempty"`
	Count    int    `json:"count"`
}

// PathMonitor accumulates repeated traceroutes to one destination.
//
// It is not safe for concurrent use: the loop that produces rounds is the only
// thing that should write to it, and Snapshot copies out for readers.
type PathMonitor struct {
	target string
	rounds int
	hops   map[int]*HopStats
}

// NewPathMonitor returns an empty monitor for a destination.
func NewPathMonitor(target string) *PathMonitor {
	return &PathMonitor{target: target, hops: make(map[int]*HopStats)}
}

// PathMonitorSnapshot is the accumulated view at one instant.
type PathMonitorSnapshot struct {
	Target string     `json:"target"`
	Rounds int        `json:"rounds"`
	Hops   []HopStats `json:"hops"`
}

// Observe folds one completed traceroute into the accumulated statistics.
//
// A round that reached fewer hops than a previous one still counts as a probe
// against every TTL already known up to its own length: a hop that stopped
// answering has lost packets, and skipping it would report 0% loss for a hop
// that has gone silent -- the exact fault this feature exists to catch.
func (m *PathMonitor) Observe(result *TracerouteResult) {
	if result == nil {
		return
	}
	m.rounds++

	seen := make(map[int]bool, len(result.Hops))
	for _, hop := range result.Hops {
		seen[hop.TTL] = true
		m.hopAt(hop.TTL).observe(hop)
	}

	// Silence up to the depth this round reached.
	depth := 0
	for _, hop := range result.Hops {
		if hop.TTL > depth {
			depth = hop.TTL
		}
	}
	for ttl, stats := range m.hops {
		if ttl <= depth && !seen[ttl] {
			stats.Sent++
		}
	}
}

// hopAt returns the accumulator for a TTL, creating it on first sight.
func (m *PathMonitor) hopAt(ttl int) *HopStats {
	stats, ok := m.hops[ttl]
	if !ok {
		stats = &HopStats{TTL: ttl, addrCounts: make(map[string]*HopAddress)}
		m.hops[ttl] = stats
	}
	return stats
}

// observe folds a single hop observation into the accumulator.
func (h *HopStats) observe(hop TracerouteHop) {
	h.Sent++

	if hop.State != hopStateReply && hop.State != hopStateUnreachable {
		return
	}
	if hop.IP == "" {
		return
	}

	h.Received++
	h.noteAddress(hop)
	h.noteRTT(hop.RTT)
}

// noteAddress records which address answered.
func (h *HopStats) noteAddress(hop TracerouteHop) {
	addr, ok := h.addrCounts[hop.IP]
	if !ok {
		addr = &HopAddress{IP: hop.IP, Hostname: hop.Hostname}
		h.addrCounts[hop.IP] = addr
	}
	// A later round may resolve a name an earlier one did not.
	if addr.Hostname == "" && hop.Hostname != "" {
		addr.Hostname = hop.Hostname
	}
	addr.Count++
}

// noteRTT folds one latency sample into the running extremes and jitter.
func (h *HopStats) noteRTT(rtt time.Duration) {
	if h.hasLast {
		h.sumJitter += absDuration(rtt - h.LastRTT)
		h.jitterObs++
	}

	if !h.hasLast || rtt < h.BestRTT {
		h.BestRTT = rtt
	}
	if rtt > h.WorstRTT {
		h.WorstRTT = rtt
	}

	h.LastRTT = rtt
	h.sumRTT += rtt
	h.hasLast = true
}

// Snapshot returns the accumulated statistics, ordered by TTL, with the
// derived fields computed. The result shares nothing with the monitor, so a
// caller may hold it while monitoring continues.
func (m *PathMonitor) Snapshot() PathMonitorSnapshot {
	out := PathMonitorSnapshot{
		Target: m.target,
		Rounds: m.rounds,
		Hops:   make([]HopStats, 0, len(m.hops)),
	}

	for _, stats := range m.hops {
		out.Hops = append(out.Hops, stats.derive())
	}
	sort.Slice(out.Hops, func(i, j int) bool { return out.Hops[i].TTL < out.Hops[j].TTL })
	return out
}

// derive returns a copy with loss, average and jitter computed from the totals.
func (h *HopStats) derive() HopStats {
	out := *h
	out.addrCounts = nil
	out.sumRTT, out.sumJitter, out.jitterObs, out.hasLast = 0, 0, 0, false

	if h.Sent > 0 {
		out.LossPct = float64(h.Sent-h.Received) / float64(h.Sent) * percentScale
	}
	if h.Received > 0 {
		out.AvgRTT = h.sumRTT / time.Duration(h.Received)
	}
	if h.jitterObs > 0 {
		out.Jitter = h.sumJitter / time.Duration(h.jitterObs)
	}

	out.Addresses = make([]HopAddress, 0, len(h.addrCounts))
	for _, addr := range h.addrCounts {
		out.Addresses = append(out.Addresses, *addr)
	}
	sort.Slice(out.Addresses, func(i, j int) bool {
		if out.Addresses[i].Count != out.Addresses[j].Count {
			return out.Addresses[i].Count > out.Addresses[j].Count
		}
		return out.Addresses[i].IP < out.Addresses[j].IP
	})
	return out
}

// percentScale converts a ratio to a percentage.
const percentScale = 100

// absDuration returns the magnitude of a duration difference.
func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
