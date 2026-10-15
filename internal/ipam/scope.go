package ipam

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"slices"
	"sync"
)

// ErrInvalidScope is returned for a scope whose ends are not two IPv4
// addresses in ascending order.
var ErrInvalidScope = errors.New("ipam: a DHCP scope is two IPv4 addresses, start first")

// Scope is a DHCP server's address pool: the inclusive range it leases from.
// Seed cannot read a pool out of an arbitrary DHCP server, so an operator
// enters it.
type Scope struct {
	Name  string
	Start netip.Addr
	End   netip.Addr
}

// ParseScope validates an operator's scope.
func ParseScope(name, start, end string) (Scope, error) {
	first, err := netip.ParseAddr(start)
	if err != nil {
		return Scope{}, ErrInvalidScope
	}
	last, err := netip.ParseAddr(end)
	if err != nil || !first.Is4() || !last.Is4() || last.Less(first) {
		return Scope{}, ErrInvalidScope
	}
	return Scope{Name: name, Start: first, End: last}, nil
}

// Size is the number of addresses the scope leases.
func (s Scope) Size() uint64 {
	return uint64(toUint32(s.End)-toUint32(s.Start)) + 1
}

// Contains reports whether addr is one of the scope's addresses.
func (s Scope) Contains(addr netip.Addr) bool {
	return addr.Is4() && !addr.Less(s.Start) && !s.End.Less(addr)
}

// Overlaps reports whether the two scopes share an address. Two pools leasing
// the same address would count it twice.
func (s Scope) Overlaps(other Scope) bool {
	return !s.End.Less(other.Start) && !other.End.Less(s.Start)
}

func toUint32(addr netip.Addr) uint32 {
	b := addr.As4()
	return binary.BigEndian.Uint32(b[:])
}

// Exhaustion is how close a scope is to having nothing left to lease.
type Exhaustion string

// The exhaustion levels, in rising order.
const (
	ExhaustionNone Exhaustion = "none"
	// ExhaustionNear is a scope with NearPercent or more of it in use.
	ExhaustionNear Exhaustion = "near"
	// ExhaustionFull is a scope with no address left.
	ExhaustionFull Exhaustion = "full"
)

// NearPercent is the utilization at which a scope is near exhaustion. One
// address in ten left is room for the clients a busy morning brings, and
// time to widen the pool or shorten its leases before they are refused.
const NearPercent = 90

// rank orders the levels from least to most exhausted; an unknown level,
// such as a scope not seen before, ranks below them all.
func (e Exhaustion) rank() int {
	return slices.Index([]Exhaustion{ExhaustionNone, ExhaustionNear, ExhaustionFull}, e)
}

// ScopeUsage is one scope's utilization.
type ScopeUsage struct {
	Scope

	// InUse is the scope's addresses seen in use, in address order.
	InUse []netip.Addr
}

// Free is the number of the scope's addresses nobody has been seen on.
func (u ScopeUsage) Free() uint64 {
	return u.Size() - uint64(len(u.InUse))
}

// Percent is the share of the scope in use, 0 to 100.
func (u ScopeUsage) Percent() float64 {
	return float64(len(u.InUse)) * 100 / float64(u.Size())
}

// Exhaustion is the scope's level.
func (u ScopeUsage) Exhaustion() Exhaustion {
	switch {
	case u.Free() == 0:
		return ExhaustionFull
	case u.Percent() >= NearPercent:
		return ExhaustionNear
	default:
		return ExhaustionNone
	}
}

// ScopeUtilization returns the usage of each scope, in the order given.
func ScopeUtilization(scopes []Scope, inUse []netip.Addr) []ScopeUsage {
	addrs := slices.Clone(inUse)
	slices.SortFunc(addrs, netip.Addr.Compare)
	addrs = slices.Compact(addrs)

	out := make([]ScopeUsage, 0, len(scopes))
	for _, scope := range scopes {
		usage := ScopeUsage{Scope: scope}
		for _, addr := range addrs {
			if scope.Contains(addr) {
				usage.InUse = append(usage.InUse, addr)
			}
		}
		out = append(out, usage)
	}
	return out
}

// Watch remembers each scope's last exhaustion level, so a scope that stays
// near exhaustion is reported once, not after every sweep. The zero value is
// ready to use.
type Watch struct {
	mu     sync.Mutex
	levels map[Scope]Exhaustion
}

// Rising records the usages and returns those whose level climbed since the
// last call. A scope that falls back is reported again when it next climbs,
// and one no longer listed is forgotten.
func (w *Watch) Rising(usages []ScopeUsage) []ScopeUsage {
	w.mu.Lock()
	defer w.mu.Unlock()

	levels := make(map[Scope]Exhaustion, len(usages))
	var rising []ScopeUsage
	for _, usage := range usages {
		level := usage.Exhaustion()
		levels[usage.Scope] = level
		if level != ExhaustionNone && level.rank() > w.levels[usage.Scope].rank() {
			rising = append(rising, usage)
		}
	}
	w.levels = levels
	return rising
}
