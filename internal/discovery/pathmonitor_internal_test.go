package discovery

import (
	"testing"
	"time"
)

// round builds a traceroute result from a compact hop spec, so a test reads as
// the sequence of network conditions it is describing.
func round(hops ...TracerouteHop) *TracerouteResult {
	return &TracerouteResult{Target: "example.test", Hops: hops}
}

func reply(ttl int, ip string, rtt time.Duration) TracerouteHop {
	return TracerouteHop{TTL: ttl, IP: ip, RTT: rtt, State: hopStateReply}
}

func timeout(ttl int) TracerouteHop {
	return TracerouteHop{TTL: ttl, State: hopStateTimeout}
}

// hopByTTL finds a hop in a snapshot, failing the test when it is absent.
func hopByTTL(t *testing.T, snap PathMonitorSnapshot, ttl int) HopStats {
	t.Helper()
	for _, h := range snap.Hops {
		if h.TTL == ttl {
			return h
		}
	}
	t.Fatalf("no hop with ttl %d in snapshot %+v", ttl, snap.Hops)
	return HopStats{}
}

func TestLossIsCountedPerHop(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond), reply(2, "10.0.0.2", 2*time.Millisecond)))
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond), timeout(2)))
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond), timeout(2)))
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond), reply(2, "10.0.0.2", 2*time.Millisecond)))

	snap := m.Snapshot()
	if snap.Rounds != 4 {
		t.Fatalf("rounds = %d, want 4", snap.Rounds)
	}
	if got := hopByTTL(t, snap, 1).LossPct; got != 0 {
		t.Errorf("hop 1 loss = %v%%, want 0 -- it answered every round", got)
	}
	if got := hopByTTL(t, snap, 2).LossPct; got != 50 {
		t.Errorf("hop 2 loss = %v%%, want 50 -- it answered 2 of 4", got)
	}
}

// A hop that stops appearing at all has gone silent, which is loss. Counting
// only the rounds it appeared in would report a failing hop as perfectly
// healthy -- the fault this feature exists to catch.
func TestHopThatDisappearsCountsAsLoss(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond), reply(2, "10.0.0.2", 2*time.Millisecond)))
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond), timeout(2), reply(3, "10.0.0.3", 3*time.Millisecond)))

	hop2 := hopByTTL(t, m.Snapshot(), 2)
	if hop2.Sent != 2 {
		t.Fatalf("hop 2 sent = %d, want 2", hop2.Sent)
	}
	if hop2.LossPct != 50 {
		t.Errorf("hop 2 loss = %v%%, want 50", hop2.LossPct)
	}
}

// A round that terminates early has not probed the hops beyond its depth, so
// those must not be charged for packets that were never sent.
func TestShorterRoundDoesNotChargeDeeperHops(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond), reply(2, "10.0.0.2", 2*time.Millisecond)))
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond)))

	hop2 := hopByTTL(t, m.Snapshot(), 2)
	if hop2.Sent != 1 {
		t.Errorf("hop 2 sent = %d, want 1 -- round 2 never reached ttl 2", hop2.Sent)
	}
	if hop2.LossPct != 0 {
		t.Errorf("hop 2 loss = %v%%, want 0", hop2.LossPct)
	}
}

func TestLatencyExtremesAndAverage(t *testing.T) {
	m := NewPathMonitor("example.test")
	for _, rtt := range []time.Duration{10 * time.Millisecond, 30 * time.Millisecond, 20 * time.Millisecond} {
		m.Observe(round(reply(1, "10.0.0.1", rtt)))
	}

	hop := hopByTTL(t, m.Snapshot(), 1)
	if hop.BestRTT != 10*time.Millisecond {
		t.Errorf("best = %v, want 10ms", hop.BestRTT)
	}
	if hop.WorstRTT != 30*time.Millisecond {
		t.Errorf("worst = %v, want 30ms", hop.WorstRTT)
	}
	if hop.AvgRTT != 20*time.Millisecond {
		t.Errorf("avg = %v, want 20ms", hop.AvgRTT)
	}
	if hop.LastRTT != 20*time.Millisecond {
		t.Errorf("last = %v, want 20ms", hop.LastRTT)
	}
}

// Jitter is the mean absolute step between consecutive replies: |30-10| and
// |20-30| average to 15ms. A hop with the same average latency but no movement
// must report zero, or the number says nothing.
func TestJitterMeasuresMovementNotMagnitude(t *testing.T) {
	moving := NewPathMonitor("example.test")
	for _, rtt := range []time.Duration{10 * time.Millisecond, 30 * time.Millisecond, 20 * time.Millisecond} {
		moving.Observe(round(reply(1, "10.0.0.1", rtt)))
	}
	if got := hopByTTL(t, moving.Snapshot(), 1).Jitter; got != 15*time.Millisecond {
		t.Errorf("jitter = %v, want 15ms", got)
	}

	steady := NewPathMonitor("example.test")
	for range 3 {
		steady.Observe(round(reply(1, "10.0.0.1", 20*time.Millisecond)))
	}
	if got := hopByTTL(t, steady.Snapshot(), 1).Jitter; got != 0 {
		t.Errorf("steady jitter = %v, want 0", got)
	}
}

// Fewer than two replies gives nothing to difference, and a fabricated jitter
// would read as a real measurement.
func TestJitterIsZeroBelowTwoReplies(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(reply(1, "10.0.0.1", 10*time.Millisecond)))
	if got := hopByTTL(t, m.Snapshot(), 1).Jitter; got != 0 {
		t.Errorf("jitter = %v, want 0 with one sample", got)
	}
}

// Two addresses answering the same TTL is load balancing, not instability.
// Both are reported, most-seen first, so the operator sees the fan-out.
func TestLoadBalancedHopKeepsEveryAddress(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(reply(2, "10.0.0.9", time.Millisecond)))
	m.Observe(round(reply(2, "10.0.0.8", time.Millisecond)))
	m.Observe(round(reply(2, "10.0.0.8", time.Millisecond)))

	addrs := hopByTTL(t, m.Snapshot(), 2).Addresses
	if len(addrs) != 2 {
		t.Fatalf("addresses = %+v, want 2", addrs)
	}
	if addrs[0].IP != "10.0.0.8" || addrs[0].Count != 2 {
		t.Errorf("first address = %+v, want 10.0.0.8 seen twice", addrs[0])
	}
	if addrs[1].IP != "10.0.0.9" || addrs[1].Count != 1 {
		t.Errorf("second address = %+v, want 10.0.0.9 seen once", addrs[1])
	}
}

// A name resolved in a later round belongs to the address regardless of which
// round produced it.
func TestHostnameIsKeptOnceResolved(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(TracerouteHop{TTL: 1, IP: "10.0.0.1", State: hopStateReply}))
	m.Observe(round(TracerouteHop{TTL: 1, IP: "10.0.0.1", Hostname: "gw.example.test", State: hopStateReply}))

	if got := hopByTTL(t, m.Snapshot(), 1).Addresses[0].Hostname; got != "gw.example.test" {
		t.Errorf("hostname = %q, want gw.example.test", got)
	}
}

// An unreachable reply is an answer -- the hop responded, it just refused.
// Treating it as loss would blame a firewall on the network.
func TestUnreachableCountsAsAReply(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(TracerouteHop{
		TTL: 3, IP: "10.0.0.3", RTT: 5 * time.Millisecond, State: hopStateUnreachable,
	}))

	hop := hopByTTL(t, m.Snapshot(), 3)
	if hop.Received != 1 || hop.LossPct != 0 {
		t.Errorf("received = %d, loss = %v%%, want 1 and 0", hop.Received, hop.LossPct)
	}
}

func TestSnapshotIsOrderedByTTLAndDetached(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(round(reply(3, "10.0.0.3", time.Millisecond), reply(1, "10.0.0.1", time.Millisecond)))

	snap := m.Snapshot()
	if len(snap.Hops) != 2 || snap.Hops[0].TTL != 1 || snap.Hops[1].TTL != 3 {
		t.Fatalf("hops = %+v, want ttl 1 then 3", snap.Hops)
	}

	// Continuing to monitor must not mutate a snapshot already handed out.
	m.Observe(round(reply(1, "10.0.0.1", time.Millisecond)))
	if snap.Rounds != 1 || snap.Hops[0].Sent != 1 {
		t.Errorf("snapshot changed under the caller: %+v", snap)
	}
}

func TestObserveIgnoresNilRound(t *testing.T) {
	m := NewPathMonitor("example.test")
	m.Observe(nil)
	if snap := m.Snapshot(); snap.Rounds != 0 || len(snap.Hops) != 0 {
		t.Errorf("snapshot = %+v, want empty", snap)
	}
}
