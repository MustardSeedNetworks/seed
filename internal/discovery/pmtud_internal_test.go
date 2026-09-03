package discovery

import (
	"context"
	"errors"
	"testing"
)

// pathProbe simulates a path that carries packets up to a limit. advertise
// controls whether routers on it obey RFC 1191's SHOULD and report their
// next-hop MTU; blackhole makes oversized packets vanish instead of provoking
// the fragmentation-needed error the sender is entitled to.
type pathProbe struct {
	limit     int
	advertise bool
	blackhole bool
	sizes     []int
}

func (p *pathProbe) probe(_ context.Context, size int) (ProbeOutcome, int, error) {
	p.sizes = append(p.sizes, size)

	if size <= p.limit {
		return ProbeArrived, 0, nil
	}
	if p.blackhole {
		return ProbeLost, 0, nil
	}
	if p.advertise {
		return ProbeTooBig, p.limit, nil
	}
	return ProbeTooBig, 0, nil
}

func TestPathWithNoBottleneckReportsTheLocalMTU(t *testing.T) {
	path := &pathProbe{limit: JumboMTU}

	result := DiscoverPathMTU(t.Context(), "example.test", 1500, path.probe)

	if result.Status != PMTUDStatusOK || result.PathMTU != 1500 {
		t.Fatalf("result = %+v, want ok at 1500", result)
	}
	// The local link was the constraint, so one probe settles it.
	if result.Probes != 1 {
		t.Errorf("sent %d probes for an unconstrained path, want 1", result.Probes)
	}
}

// Probing above the local interface MTU measures the local card, not the path:
// the kernel rejects the packet before it reaches the wire.
func TestSearchStartsAtTheLocalMTUNotTheJumboCeiling(t *testing.T) {
	path := &pathProbe{limit: JumboMTU}

	DiscoverPathMTU(t.Context(), "example.test", 1500, path.probe)

	if path.sizes[0] != 1500 {
		t.Errorf("first probe was %d bytes on a 1500-byte link, want 1500", path.sizes[0])
	}
}

func TestJumboPathIsDiscovered(t *testing.T) {
	path := &pathProbe{limit: JumboMTU}

	result := DiscoverPathMTU(t.Context(), "example.test", JumboMTU, path.probe)

	if result.Status != PMTUDStatusOK || result.PathMTU != JumboMTU {
		t.Fatalf("result = %+v, want ok at %d", result, JumboMTU)
	}
}

// A router that advertises its next-hop MTU is what RFC 1191 exists for: the
// search should land on it directly rather than bisecting toward it.
func TestAdvertisedNextHopMTUIsUsedDirectly(t *testing.T) {
	path := &pathProbe{limit: 1400, advertise: true}

	result := DiscoverPathMTU(t.Context(), "example.test", JumboMTU, path.probe)

	if result.Status != PMTUDStatusOK || result.PathMTU != 1400 {
		t.Fatalf("result = %+v, want ok at 1400", result)
	}
	if result.Probes > 3 {
		t.Errorf("sent %d probes to confirm an advertised MTU, want a direct jump", result.Probes)
	}
}

// Plenty of routers advertise nothing, and the search has to work anyway.
func TestSilentRouterIsBisected(t *testing.T) {
	path := &pathProbe{limit: 1400}

	result := DiscoverPathMTU(t.Context(), "example.test", JumboMTU, path.probe)

	if result.Status != PMTUDStatusOK || result.PathMTU != 1400 {
		t.Fatalf("result = %+v, want ok at 1400", result)
	}
	if result.Probes < 4 {
		t.Errorf("found the answer in %d probes without advertisement; suspiciously few", result.Probes)
	}
}

// A router that lies about its next-hop MTU must not be taken at its word --
// the claim is confirmed, and a failed confirmation becomes an upper bound.
func TestOverstatedNextHopMTUIsNotBelieved(t *testing.T) {
	// Advertises 1400 but actually only carries 1200.
	path := &pathProbe{limit: 1200}
	liar := func(ctx context.Context, size int) (ProbeOutcome, int, error) {
		outcome, _, err := path.probe(ctx, size)
		if outcome == ProbeTooBig {
			return ProbeTooBig, 1400, err
		}
		return outcome, 0, err
	}

	result := DiscoverPathMTU(t.Context(), "example.test", JumboMTU, liar)

	if result.PathMTU != 1200 {
		t.Fatalf("path MTU = %d, want 1200 -- the advertisement was wrong", result.PathMTU)
	}
	if result.Status != PMTUDStatusOK {
		t.Errorf("status = %q, want ok", result.Status)
	}
}

// The black-hole case is the one worth getting right: oversized packets vanish
// with no error, so the number found is a lower bound, and saying "ok" would
// point the operator at the wrong problem.
func TestBlackHolePathIsReportedAsFiltered(t *testing.T) {
	path := &pathProbe{limit: 1400, blackhole: true}

	result := DiscoverPathMTU(t.Context(), "example.test", JumboMTU, path.probe)

	if result.Status != PMTUDStatusICMPFiltered {
		t.Fatalf("status = %q, want %q", result.Status, PMTUDStatusICMPFiltered)
	}
	if result.PathMTU != 1400 {
		t.Errorf("lower bound = %d, want 1400", result.PathMTU)
	}
}

// One lost packet is ordinary. Concluding "black hole" from a single silence
// would report a filtered path on any lossy link.
func TestSingleLostProbeIsRetriedNotBelieved(t *testing.T) {
	var calls int
	flaky := func(_ context.Context, _ int) (ProbeOutcome, int, error) {
		calls++
		if calls == 1 {
			return ProbeLost, 0, nil
		}
		return ProbeArrived, 0, nil
	}

	result := DiscoverPathMTU(t.Context(), "example.test", 1500, flaky)

	if result.Status != PMTUDStatusOK || result.PathMTU != 1500 {
		t.Fatalf("result = %+v, want ok at 1500 after one dropped probe", result)
	}
}

func TestDestinationThatNeverAnswersIsUnreachable(t *testing.T) {
	silent := func(_ context.Context, _ int) (ProbeOutcome, int, error) {
		return ProbeLost, 0, nil
	}

	result := DiscoverPathMTU(t.Context(), "example.test", 1500, silent)

	if result.Status != PMTUDStatusUnreachable {
		t.Fatalf("status = %q, want %q", result.Status, PMTUDStatusUnreachable)
	}
	if result.PathMTU != 0 {
		t.Errorf("path MTU = %d, want 0 -- nothing was learned", result.PathMTU)
	}
}

func TestUnknownLocalMTUIsRefusedRatherThanGuessed(t *testing.T) {
	result := DiscoverPathMTU(t.Context(), "example.test", 0, nil)

	if result.Status != PMTUDStatusUnreachable || result.Error == "" {
		t.Fatalf("result = %+v, want unreachable with an explanation", result)
	}
}

func TestProbeErrorStopsTheSearch(t *testing.T) {
	failing := func(_ context.Context, _ int) (ProbeOutcome, int, error) {
		return ProbeLost, 0, errors.New("socket closed")
	}

	result := DiscoverPathMTU(t.Context(), "example.test", 1500, failing)

	if result.Error == "" {
		t.Fatal("a failing socket produced no error on the result")
	}
}

func TestCancellationStopsTheSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	path := &pathProbe{limit: 1400}
	cancelling := func(c context.Context, size int) (ProbeOutcome, int, error) {
		cancel()
		return path.probe(c, size)
	}

	result := DiscoverPathMTU(ctx, "example.test", JumboMTU, cancelling)

	if result.Probes > 2 {
		t.Errorf("sent %d probes after cancellation", result.Probes)
	}
}

// A link below the search floor is legal -- a tunnel can be tiny -- and
// bisecting under 576 tells an operator nothing they can act on.
func TestLinkBelowTheFloorIsReportedNotSearched(t *testing.T) {
	path := &pathProbe{limit: 1500}

	result := DiscoverPathMTU(t.Context(), "example.test", 296, path.probe)

	if result.Status != PMTUDStatusOK || result.PathMTU != 296 {
		t.Fatalf("result = %+v, want ok at 296", result)
	}
	if result.Probes != 0 {
		t.Errorf("sent %d probes on a sub-floor link, want none", result.Probes)
	}
}

func TestPayloadAccountsForHeaders(t *testing.T) {
	if got := PayloadForMTU(1500); got != 1472 {
		t.Errorf("PayloadForMTU(1500) = %d, want 1472", got)
	}
	if got := PayloadForMTU(20); got != 0 {
		t.Errorf("PayloadForMTU(20) = %d, want 0", got)
	}
}
