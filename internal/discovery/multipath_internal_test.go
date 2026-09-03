package discovery

import (
	"context"
	"testing"
)

// tracePath builds a traceroute result from a list of hop addresses, where an
// empty string is a hop that did not answer.
func tracePath(addresses ...string) *TracerouteResult {
	result := &TracerouteResult{Target: "example.test", TargetIP: "203.0.113.1"}
	for i, address := range addresses {
		hop := TracerouteHop{TTL: i + 1, IP: address, State: hopStateTimeout}
		if address != "" {
			hop.State = hopStateReply
		}
		result.Hops = append(result.Hops, hop)
	}
	result.Completed = true
	return result
}

// alternating returns a trace function that cycles through routes, so a test
// says which routes exist rather than how the flow identifier maps onto them.
func alternating(routes ...*TracerouteResult) (pathTraceFunc, *[]int) {
	var flowIDs []int
	call := 0
	return func(_ context.Context, flowID int) *TracerouteResult {
		flowIDs = append(flowIDs, flowID)
		route := routes[call%len(routes)]
		call++
		return route
	}, &flowIDs
}

func TestSinglePathIsNotReportedAsLoadBalanced(t *testing.T) {
	trace, _ := alternating(tracePath("10.0.0.1", "10.0.0.2", "203.0.113.1"))

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if len(result.Paths) != 1 {
		t.Fatalf("found %d paths on a single-path network, want 1", len(result.Paths))
	}
	if result.LoadBalanced() {
		t.Error("reported load balancing where every attempt took one route")
	}
	if result.DivergesAtTTL != 0 {
		t.Errorf("divergence at ttl %d, want 0 -- nothing diverged", result.DivergesAtTTL)
	}
	if result.Paths[0].Seen != multiPathAttempts {
		t.Errorf("route seen %d times, want all %d attempts", result.Paths[0].Seen, multiPathAttempts)
	}
}

// The whole point: a two-way split that a single traceroute would show as one
// healthy path.
func TestTwoRoutesAreBothReported(t *testing.T) {
	trace, _ := alternating(
		tracePath("10.0.0.1", "10.0.0.2", "203.0.113.1"),
		tracePath("10.0.0.1", "10.0.0.3", "203.0.113.1"),
	)

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if len(result.Paths) != 2 {
		t.Fatalf("found %d paths, want 2", len(result.Paths))
	}
	if !result.LoadBalanced() {
		t.Error("two distinct routes were not reported as load balanced")
	}
	if result.DivergesAtTTL != 2 {
		t.Errorf("divergence at ttl %d, want 2 -- the routes agree on hop 1", result.DivergesAtTTL)
	}
}

// "There are three paths" is a curiosity; "they split at your second-hop
// router" is where to look.
func TestDivergencePointIsTheFirstDisagreement(t *testing.T) {
	trace, _ := alternating(
		tracePath("10.0.0.1", "10.0.0.2", "10.0.0.4", "203.0.113.1"),
		tracePath("10.0.0.1", "10.0.0.2", "10.0.0.5", "203.0.113.1"),
	)

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if result.DivergesAtTTL != 3 {
		t.Errorf("divergence at ttl %d, want 3", result.DivergesAtTTL)
	}
}

// A hop silent on one attempt and answering on another is one route with a
// lost packet. Calling it a split would put a load balancer where there is
// only loss.
func TestSilentHopIsNotADivergence(t *testing.T) {
	trace, _ := alternating(
		tracePath("10.0.0.1", "10.0.0.2", "203.0.113.1"),
		tracePath("10.0.0.1", "", "203.0.113.1"),
	)

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if result.DivergesAtTTL != 0 {
		t.Errorf("divergence at ttl %d, want 0 -- one hop merely went quiet", result.DivergesAtTTL)
	}
}

// The routes are still distinct observations even though they do not diverge:
// collapsing them would lose the fact that a hop stops answering sometimes.
func TestSilentHopStillMakesADistinctObservation(t *testing.T) {
	trace, _ := alternating(
		tracePath("10.0.0.1", "10.0.0.2", "203.0.113.1"),
		tracePath("10.0.0.1", "", "203.0.113.1"),
	)

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if len(result.Paths) != 2 {
		t.Fatalf("found %d observations, want 2", len(result.Paths))
	}
}

// Seen counts are only readable as a ratio if the attempts are known.
func TestSeenCountsSumToTheAttempts(t *testing.T) {
	trace, _ := alternating(
		tracePath("10.0.0.1", "10.0.0.2"),
		tracePath("10.0.0.1", "10.0.0.3"),
		tracePath("10.0.0.1", "10.0.0.4"),
	)

	result := DiscoverPaths(t.Context(), "example.test", trace)

	total := 0
	for _, path := range result.Paths {
		total += path.Seen
	}
	if total != result.Attempts || result.Attempts != multiPathAttempts {
		t.Errorf("seen totals %d over %d attempts, want %d each", total, result.Attempts, multiPathAttempts)
	}
}

func TestMostTravelledRouteIsReportedFirst(t *testing.T) {
	// Three of every four attempts take the first route.
	busy := tracePath("10.0.0.1", "10.0.0.2")
	quiet := tracePath("10.0.0.1", "10.0.0.9")
	trace, _ := alternating(busy, busy, busy, quiet)

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if len(result.Paths) != 2 {
		t.Fatalf("found %d paths, want 2", len(result.Paths))
	}
	if result.Paths[0].Hops[1] != "10.0.0.2" || result.Paths[0].Seen <= result.Paths[1].Seen {
		t.Errorf("paths not ordered by traffic: %+v", result.Paths)
	}
}

// A router hashing the low bits of the identifier would map consecutive
// attempts onto very few buckets and hide a split that exists.
func TestFlowIDsAreSpreadNotSequential(t *testing.T) {
	trace, flowIDs := alternating(tracePath("10.0.0.1"))

	DiscoverPaths(t.Context(), "example.test", trace)

	seen := make(map[int]bool, len(*flowIDs))
	consecutive := 0
	for i, id := range *flowIDs {
		if id <= 0 || id > 0xffff {
			t.Errorf("flow id %d outside the 16-bit identifier field", id)
		}
		seen[id] = true
		if i > 0 && id == (*flowIDs)[i-1]+1 {
			consecutive++
		}
	}
	if len(seen) != multiPathAttempts {
		t.Errorf("%d distinct flow ids across %d attempts", len(seen), multiPathAttempts)
	}
	if consecutive > 0 {
		t.Errorf("%d flow ids were consecutive; they should be spread", consecutive)
	}
}

func TestCancellationStopsTracing(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	trace := func(_ context.Context, _ int) *TracerouteResult {
		calls++
		cancel()
		return tracePath("10.0.0.1")
	}

	DiscoverPaths(ctx, "example.test", trace)

	if calls > 1 {
		t.Errorf("ran %d traces after cancellation, want 1", calls)
	}
}

func TestFailedAttemptDoesNotCount(t *testing.T) {
	call := 0
	trace := func(_ context.Context, _ int) *TracerouteResult {
		call++
		if call%2 == 0 {
			return nil
		}
		return tracePath("10.0.0.1", "10.0.0.2")
	}

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if result.Attempts != multiPathAttempts/2 {
		t.Errorf("counted %d attempts, want %d -- half of them produced nothing", result.Attempts, multiPathAttempts/2)
	}
}

func TestTraceErrorIsSurfaced(t *testing.T) {
	trace := func(_ context.Context, _ int) *TracerouteResult {
		result := tracePath("10.0.0.1")
		result.Error = "no route to host"
		return result
	}

	result := DiscoverPaths(t.Context(), "example.test", trace)

	if result.Error != "no route to host" {
		t.Errorf("error = %q, want it surfaced from the trace", result.Error)
	}
}
