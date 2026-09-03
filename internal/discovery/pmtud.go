package discovery

// Path MTU discovery (#435).
//
// A path that fragments is a path that performs badly for reasons nothing in a
// latency graph explains, and a path that black-holes oversized packets fails
// only for large transfers -- the fault that gets reported as "the network is
// fine but the backup won't finish".
//
// The search here is the algorithm only. It calls an injected probe, so the
// binary search, the RFC 1191 shortcut and the black-hole fallback are all
// testable without a socket; the platform-specific probes that actually set the
// don't-fragment bit live beside it in pmtud_{linux,darwin,windows}.go.

import (
	"context"
	"errors"
	"fmt"
)

// MTU bounds for the search.
const (
	// JumboMTU is the largest size probed. #435 requires jumbo support to be
	// useful on a modern LAN.
	JumboMTU = 9000

	// MinProbeMTU is the floor. RFC 791 obliges every IPv4 host to accept 68
	// bytes, but a path that cannot carry 576 is broken in a way this tool
	// cannot usefully characterise, so the search stops there.
	MinProbeMTU = 576

	// pmtudHeaderOverhead is the IPv4 plus ICMP header the payload sits
	// inside. A "1500-byte MTU" carries 1472 bytes of echo payload.
	pmtudHeaderOverhead = 28
)

// PMTUDStatus says how much the result can be trusted.
type PMTUDStatus string

const (
	// PMTUDStatusOK means the path MTU was measured: a size that arrives and
	// a size that does not, adjacent.
	PMTUDStatusOK PMTUDStatus = "ok"

	// PMTUDStatusICMPFiltered means oversized packets vanished without the
	// fragmentation-needed error the sender is entitled to. The reported
	// value is a lower bound -- the largest size seen to arrive -- and the
	// path is black-holing, which is itself the finding.
	PMTUDStatusICMPFiltered PMTUDStatus = "icmp_filtered"

	// PMTUDStatusUnreachable means nothing arrived at any size, so the
	// destination says nothing about MTU.
	PMTUDStatusUnreachable PMTUDStatus = "unreachable"
)

// PMTUDResult is the outcome of a path MTU discovery run.
type PMTUDResult struct {
	Target   string      `json:"target"`
	TargetIP string      `json:"targetIp"`
	Status   PMTUDStatus `json:"status"`

	// PathMTU is the largest size that traverses the path. Under
	// PMTUDStatusICMPFiltered it is a lower bound rather than the answer.
	PathMTU int `json:"pathMtu"`

	// LocalMTU is the interface MTU the search started from. A path MTU equal
	// to it means the local link was the constraint and the path was never
	// the limit -- a different conclusion from having measured a bottleneck.
	LocalMTU int `json:"localMtu"`

	// Probes is how many packets the search sent, so the cost is visible.
	Probes int `json:"probes"`

	Error string `json:"error,omitempty"`
}

// ProbeOutcome is what one sized probe learned.
type ProbeOutcome int

const (
	// ProbeArrived means the packet traversed the path and was answered.
	ProbeArrived ProbeOutcome = iota

	// ProbeTooBig means a router answered fragmentation-needed.
	ProbeTooBig

	// ProbeLost means nothing came back. Indistinguishable, from one probe,
	// between a black hole and ordinary loss -- which is why the search
	// retries before concluding.
	ProbeLost
)

// ProbeFunc sends one probe of a given total packet size and reports what
// happened. nextHop is the MTU a router advertised in its fragmentation-needed
// error, or zero when it advertised none (RFC 1191 made it a SHOULD, and
// pre-1990 routers omit it).
type ProbeFunc func(ctx context.Context, size int) (outcome ProbeOutcome, nextHop int, err error)

// errNoLocalMTU is returned when the starting size cannot be established.
var errNoLocalMTU = errors.New("local interface MTU is unknown")

// ErrPMTUDUnsupported is returned when the platform cannot run a path MTU
// probe. Callers turn it into a 501 rather than a failure: nothing is broken,
// the operator is on an OS that cannot answer the question. Declared here
// rather than beside the platform that returns it so every caller can compare
// against it without a build tag.
var ErrPMTUDUnsupported = errors.New("path MTU discovery is not supported on this platform")

// DiscoverPathMTU finds the largest packet the path to a destination carries.
//
// It starts at the smaller of the jumbo ceiling and the local interface MTU,
// because a packet larger than the local link is rejected by the kernel before
// it reaches the wire -- probing 9000 on a 1500-byte interface measures the
// local card, not the path.
//
// A router that advertises its next-hop MTU is believed and jumped to directly;
// that is the whole point of RFC 1191 and it converges in one round trip where
// a binary search would take several. The search is the fallback for routers
// that advertise nothing.
func DiscoverPathMTU(ctx context.Context, target string, localMTU int, probe ProbeFunc) *PMTUDResult {
	result := &PMTUDResult{Target: target, LocalMTU: localMTU}

	if localMTU <= 0 {
		result.Status = PMTUDStatusUnreachable
		result.Error = errNoLocalMTU.Error()
		return result
	}

	high := min(localMTU, JumboMTU)
	if high < MinProbeMTU {
		// A link below the floor is legal but not something to binary-search.
		result.Status = PMTUDStatusOK
		result.PathMTU = high
		return result
	}

	search := &pmtudSearch{ctx: ctx, probe: probe, result: result}
	search.run(high)
	return result
}

// pmtudSearch carries the state of one discovery run.
type pmtudSearch struct {
	ctx    context.Context
	probe  ProbeFunc
	result *PMTUDResult

	// low is the largest size seen to arrive; high the smallest seen to fail.
	low  int
	high int
}

// bisect is the divisor that halves the remaining range each round.
const bisect = 2

// pmtudLostRetries is how many times a silent probe is repeated before it is
// believed. One lost packet is ordinary; three in a row on the same size is a
// property of the path.
const pmtudLostRetries = 3

// run drives the search from a starting ceiling.
func (s *pmtudSearch) run(ceiling int) {
	// The ceiling itself first: on a path with no bottleneck this is the whole
	// search, and it is the common case on a LAN.
	outcome, nextHop := s.send(ceiling)
	switch outcome {
	case ProbeArrived:
		s.result.Status = PMTUDStatusOK
		s.result.PathMTU = ceiling
		return
	case ProbeTooBig:
		s.low, s.high = 0, ceiling
		if nextHop >= MinProbeMTU && nextHop < ceiling && s.confirmAdvertised(nextHop) {
			s.result.Status = PMTUDStatusOK
			s.result.PathMTU = nextHop
			return
		}
	case ProbeLost:
		s.low, s.high = 0, ceiling
	}

	s.narrow()
	s.conclude(outcome)
}

// confirmAdvertised checks a router's claimed next-hop MTU by sending a packet
// of exactly that size.
//
// The advertisement is a claim, not a measurement, and a router that overstates
// it -- or a second, smaller bottleneck further along -- would otherwise be
// reported as the answer. A confirmed value is accepted as the path MTU: that
// is what RFC 1191 is for, and it converges in one round trip where a search
// would take several. A rejected one becomes an upper bound for the search.
func (s *pmtudSearch) confirmAdvertised(nextHop int) bool {
	outcome, _ := s.send(nextHop)
	if outcome == ProbeArrived {
		return true
	}
	s.high = nextHop
	return false
}

// narrow binary-searches between a size known to arrive and one known to fail.
func (s *pmtudSearch) narrow() {
	if s.low == 0 {
		if outcome, _ := s.send(MinProbeMTU); outcome != ProbeArrived {
			return
		}
		s.low = MinProbeMTU
	}

	for s.high-s.low > 1 {
		if s.ctx.Err() != nil {
			return
		}
		mid := s.low + (s.high-s.low)/bisect
		if outcome, _ := s.send(mid); outcome == ProbeArrived {
			s.low = mid
		} else {
			s.high = mid
		}
	}
}

// conclude turns the bounds into a status. The distinction that matters is
// between a measured MTU and a black hole: both report a number, but only one
// of them is an answer, and telling an operator their path MTU is 1400 when in
// fact oversized packets vanish silently would send them looking in the wrong
// place.
func (s *pmtudSearch) conclude(first ProbeOutcome) {
	switch {
	case s.low == 0:
		s.result.Status = PMTUDStatusUnreachable
	case first == ProbeLost:
		s.result.Status = PMTUDStatusICMPFiltered
		s.result.PathMTU = s.low
	default:
		s.result.Status = PMTUDStatusOK
		s.result.PathMTU = s.low
	}
}

// send probes one size, retrying a silent probe before believing it, and
// counts every packet so the caller can see what the search cost.
func (s *pmtudSearch) send(size int) (ProbeOutcome, int) {
	last := ProbeLost

	for range pmtudLostRetries {
		if s.ctx.Err() != nil {
			return ProbeLost, 0
		}

		s.result.Probes++
		outcome, nextHop, err := s.probe(s.ctx, size)
		if err != nil {
			s.result.Error = fmt.Sprintf("probe at %d bytes failed: %v", size, err)
			return ProbeLost, 0
		}
		if outcome != ProbeLost {
			return outcome, nextHop
		}
		last = outcome
	}

	return last, 0
}

// PayloadForMTU returns the echo payload that fills a packet of a given total
// size, which is what a probe actually has to construct.
func PayloadForMTU(mtu int) int {
	if mtu <= pmtudHeaderOverhead {
		return 0
	}
	return mtu - pmtudHeaderOverhead
}
