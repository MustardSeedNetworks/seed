package enumerate

import (
	"cmp"
	"net"
	"net/netip"
	"slices"
)

// probeHostsPerBlock is how many hosts at the bottom of a /24 a probe sweeps
// when nothing names a /24 inside a target: .1 to .4, where a site's gateway
// and core switches sit (seed#2832, owner decision 2026-10-01).
const probeHostsPerBlock = 4

// rotation is where a target network wider than one sweep left off, so the
// next sweep moves on rather than repeating the same /24s.
type rotation struct {
	round uint64
	// swept is the round each evidenced /24 was last swept whole; a /24 never
	// swept is absent and so sorts first.
	swept map[netip.Prefix]uint64
	// probeNext is the index, from the bottom of the target, of the next /24
	// a probe sweep starts at.
	probeNext int
}

// sweepPlan is what one sweep of a wide target probes.
type sweepPlan struct {
	blocks []netip.Prefix
	// probe means only the first probeHostsPerBlock hosts of each block.
	probe bool
}

// hosts lists the addresses the plan probes, in block order.
func (p sweepPlan) hosts() []netip.Addr {
	per := hostsPerSubnet24
	if p.probe {
		per = probeHostsPerBlock
	}
	out := make([]netip.Addr, 0, len(p.blocks)*per)
	for _, block := range p.blocks {
		addr := block.Addr()
		for range per {
			addr = addr.Next()
			out = append(out, addr)
		}
	}
	return out
}

// planSweep picks the /24s one sweep of target probes under a budget of
// maxHosts (seed#2832). target must be an IPv4 network wider than a /24.
//
// A target whose every /24 fits the budget is swept whole. Otherwise the /24s
// with evidence of devices are swept whole, as many as the budget allows, the
// least recently swept first and then the most evidence first; so a /16 with
// six live /24s and a one-/24 budget covers them in six sweeps. With no
// evidence inside the target at all, the sweep probes .1 to .4 of the next
// /24s in address order, as many as the budget allows, so a found gateway or
// core switch can turn into evidence without ever blind-sweeping 65k hosts.
//
// The plan is not recorded here: commit does that once the sweep has run, so
// a sweep that could not open a socket does not move the rotation on.
func (r *rotation) planSweep(target netip.Prefix, evidence []netip.Addr, maxHosts int) sweepPlan {
	blocks := 1 << (cidrMask24 - target.Bits())
	wholeBlocks := max(1, (maxHosts+roundUpAdjust)/hostsPerSubnet24)
	if blocks <= wholeBlocks {
		return sweepPlan{blocks: blockRange(target, 0, blocks, blocks)}
	}

	seen := make(map[netip.Prefix]int)
	for _, addr := range evidence {
		if target.Contains(addr) {
			block, _ := addr.Prefix(cidrMask24)
			seen[block]++
		}
	}
	if len(seen) == 0 {
		count := min(blocks, max(1, maxHosts/probeHostsPerBlock))
		return sweepPlan{blocks: blockRange(target, r.probeNext, count, blocks), probe: true}
	}

	evidenced := make([]netip.Prefix, 0, len(seen))
	for block := range seen {
		evidenced = append(evidenced, block)
	}
	slices.SortFunc(evidenced, func(a, b netip.Prefix) int {
		return cmp.Or(
			cmp.Compare(r.swept[a], r.swept[b]),
			cmp.Compare(seen[b], seen[a]),
			a.Addr().Compare(b.Addr()),
		)
	})
	return sweepPlan{blocks: evidenced[:min(len(evidenced), wholeBlocks)]}
}

// commit records a plan that was swept.
func (r *rotation) commit(target netip.Prefix, plan sweepPlan) {
	if plan.probe {
		blocks := 1 << (cidrMask24 - target.Bits())
		r.probeNext = (r.probeNext + len(plan.blocks)) % blocks
		return
	}
	r.round++
	if r.swept == nil {
		r.swept = make(map[netip.Prefix]uint64)
	}
	for _, block := range plan.blocks {
		r.swept[block] = r.round
	}
}

// blockRange returns count consecutive /24s of target starting at index
// first, wrapping past the top, where target holds total /24s.
func blockRange(target netip.Prefix, first, count, total int) []netip.Prefix {
	base := target.Masked().Addr().As4()
	start := int(base[0])<<byteShift24 | int(base[1])<<byteShift16 | int(base[2])<<byteShift8
	out := make([]netip.Prefix, 0, count)
	for i := range count {
		value := start + (first+i)%total<<byteShift8
		addr := netip.AddrFrom4([4]byte{
			byte(value >> byteShift24 & byteMask),
			byte(value >> byteShift16 & byteMask),
			byte(value >> byteShift8 & byteMask),
			0,
		})
		out = append(out, netip.PrefixFrom(addr, cidrMask24))
	}
	return out
}

// ipv4Prefix converts a scanner subnet to the masked netip form, reporting
// false for a nil or non-IPv4 one.
func ipv4Prefix(subnet *net.IPNet) (netip.Prefix, bool) {
	if subnet == nil {
		return netip.Prefix{}, false
	}
	ip := subnet.IP.To4()
	ones, bits := subnet.Mask.Size()
	if ip == nil || bits != cidrBits32 {
		return netip.Prefix{}, false
	}
	return netip.PrefixFrom(netip.AddrFrom4([4]byte(ip)), ones).Masked(), true
}
