package discovery

// Multi-path egress detection (#395).
//
// ECMP and SD-WAN move traffic over several routes at once, and a single
// traceroute samples exactly one of them. That is why an operator sees a
// traceroute that looks healthy while some connections are slow: the trace and
// the complaint took different paths.
//
// Repeating the trace with a different flow identifier each time is what makes
// the alternatives visible -- a router hashing on that identifier sends each
// attempt down a different route. As with the monitor, the aggregation here
// touches no network so the set arithmetic is testable without one.

import (
	"context"
	"sort"
	"strings"
)

// PathVariant is one distinct route to the destination.
type PathVariant struct {
	// Hops is the address at each TTL, empty where the hop did not answer. A
	// silent hop is part of the path's shape: two routes that differ only in
	// which hop stayed quiet are still two observations, not one.
	Hops []string `json:"hops"`

	// Seen is how many attempts followed this route, which is the closest
	// thing to a load-balancing ratio that traceroute can offer.
	Seen int `json:"seen"`

	// Completed is whether an attempt on this route reached the destination.
	Completed bool `json:"completed"`
}

// MultiPathResult is what repeated tracing found.
type MultiPathResult struct {
	Target   string `json:"target"`
	TargetIP string `json:"targetIp"`

	// Attempts is how many traces ran, so Seen counts can be read as a ratio.
	Attempts int `json:"attempts"`

	// Paths are the distinct routes, most-travelled first.
	Paths []PathVariant `json:"paths"`

	// DivergesAtTTL is the first hop at which the routes differ, or zero when
	// there is only one route. It is the actionable part: "there are three
	// paths" is a curiosity, "they split at your second-hop router" is where
	// to look.
	DivergesAtTTL int `json:"divergesAtTtl"`

	Error string `json:"error,omitempty"`
}

// LoadBalanced reports whether more than one route was observed.
func (r *MultiPathResult) LoadBalanced() bool { return len(r.Paths) > 1 }

// pathTraceFunc runs one traceroute with a given flow identifier. Injected so
// the discovery loop is testable without a socket.
type pathTraceFunc func(ctx context.Context, flowID int) *TracerouteResult

// multiPathAttempts is how many traces are run. Enough to expose a two- or
// three-way split with reasonable confidence, few enough that the whole run
// stays within the patience of someone watching it: each trace is bounded by
// the per-hop timeout times the hop count.
const multiPathAttempts = 8

// DiscoverPaths traces a destination repeatedly with a different flow
// identifier each time and reports the distinct routes.
//
// The identifier varies per attempt because that is the field a load balancer
// hashes on that a caller controls; holding it fixed is what keeps a single
// trace on one route, and varying it is what enumerates the rest. Routers
// differ in what they hash -- many use only the address pair -- so a single
// path in the result means "no split was observed", not "no split exists".
func DiscoverPaths(ctx context.Context, target string, trace pathTraceFunc) *MultiPathResult {
	result := &MultiPathResult{Target: target, Attempts: 0}
	variants := make(map[string]*PathVariant)

	for attempt := range multiPathAttempts {
		if ctx.Err() != nil {
			break
		}

		// Deliberately spread rather than sequential: a router hashing the low
		// bits of the identifier would map 1..8 onto very few buckets and hide
		// a split that exists.
		round := trace(ctx, multiPathFlowID(attempt))
		if round == nil {
			continue
		}
		result.Attempts++

		if result.TargetIP == "" {
			result.TargetIP = round.TargetIP
		}
		if round.Error != "" && result.Error == "" {
			result.Error = round.Error
		}

		foldVariant(variants, round)
	}

	result.Paths = sortedVariants(variants)
	result.DivergesAtTTL = divergencePoint(result.Paths)
	return result
}

// multiPathFlowID spreads attempt numbers across the 16-bit identifier space.
func multiPathFlowID(attempt int) int {
	// An odd stride coprime with 65535 walks the space without repeating.
	const stride = 7919
	return 1 + (attempt*stride)%0xffff
}

// foldVariant records one traceroute against the set of known routes.
func foldVariant(variants map[string]*PathVariant, round *TracerouteResult) {
	hops := hopAddresses(round)
	key := strings.Join(hops, ">")

	variant, ok := variants[key]
	if !ok {
		variant = &PathVariant{Hops: hops}
		variants[key] = variant
	}
	variant.Seen++
	if round.Completed {
		variant.Completed = true
	}
}

// hopAddresses flattens a traceroute into the address at each TTL, keeping the
// position of a silent hop so path shape is preserved.
func hopAddresses(round *TracerouteResult) []string {
	depth := 0
	for _, hop := range round.Hops {
		if hop.TTL > depth {
			depth = hop.TTL
		}
	}

	addresses := make([]string, depth)
	for _, hop := range round.Hops {
		if hop.TTL >= 1 && hop.TTL <= depth {
			addresses[hop.TTL-1] = hop.IP
		}
	}
	return addresses
}

// sortedVariants orders routes by how often they were travelled.
func sortedVariants(variants map[string]*PathVariant) []PathVariant {
	out := make([]PathVariant, 0, len(variants))
	for _, variant := range variants {
		out = append(out, *variant)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Seen != out[j].Seen {
			return out[i].Seen > out[j].Seen
		}
		return strings.Join(out[i].Hops, ">") < strings.Join(out[j].Hops, ">")
	})
	return out
}

// divergencePoint returns the first TTL at which the routes stop agreeing.
//
// A hop that stayed silent on one attempt and answered on another is not a
// divergence -- that is one route with a packet lost on it, and reporting it as
// a split would put a load balancer where there is only loss.
func divergencePoint(paths []PathVariant) int {
	// One route cannot disagree with itself.
	const needTwoToDiverge = 2
	if len(paths) < needTwoToDiverge {
		return 0
	}

	for ttl := range longestPath(paths) {
		var seen string
		for _, path := range paths {
			if ttl >= len(path.Hops) || path.Hops[ttl] == "" {
				continue
			}
			if seen == "" {
				seen = path.Hops[ttl]
				continue
			}
			if path.Hops[ttl] != seen {
				return ttl + 1
			}
		}
	}
	return 0
}

// longestPath returns the hop count of the deepest route.
func longestPath(paths []PathVariant) int {
	longest := 0
	for _, path := range paths {
		if len(path.Hops) > longest {
			longest = len(path.Hops)
		}
	}
	return longest
}
