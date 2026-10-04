package flow

import (
	"math"
	"math/bits"
	"net/netip"
	"time"
)

const (
	// v9ScopeInterface is the v9 options scope type naming an interface
	// (RFC 3954 §6.1). v9 scope types are their own numbering, not field
	// types.
	v9ScopeInterface = 2

	// Packet sampling elements. 34 and 50 give a 1-in-N rate; 305 and 306
	// give RFC 5476's selected-interval and skipped-space counts. 48 and
	// 302 name the sampler a data record went through, and an options
	// record carrying one gives that sampler's rate.
	ieSamplingInterval       = 34
	ieSamplerID              = 48
	ieSamplerRandomInterval  = 50
	ieSelectorID             = 302
	ieSamplingPacketInterval = 305
	ieSamplingPacketSpace    = 306
)

// rateScope is what an announced sampling rate covers, most specific
// first: one sampler, one ingress interface, or the whole domain.
type rateScope uint8

const (
	scopeSampler rateScope = iota
	scopeInterface
	scopeDomain
)

type rateKey struct {
	exporter netip.Addr
	domain   uint32
	scope    rateScope
	// id is the sampler or selector ID, or the ifIndex; zero for a domain.
	id uint64
}

// rate is packets observed per packet sampled, as the fraction num/den:
// 1-in-N sampling is N/1, and RFC 5476's interval i and space s is
// (i+s)/i.
type rate struct {
	num, den uint64
}

// scale estimates the observed count from a sampled one, saturating
// rather than wrapping.
func (r rate) scale(n uint64) uint64 {
	hi, lo := bits.Mul64(n, r.num)
	if hi >= r.den {
		return math.MaxUint64
	}
	q, _ := bits.Div64(hi, lo, r.den)
	return q
}

type learnedRate struct {
	rate    rate
	learned time.Time
}

// recordRate finds the sampling rate covering rec: a rate the record
// carries itself, else the one its exporter announced for the record's
// sampler, its ingress interface or its domain, most specific first. A
// flow decoded before its exporter's options record arrives keeps its
// sampled counts.
func (d *Decoder) recordRate(hdr *exportHeader, rec *Record, smp *sampling, now time.Time) rate {
	if r, ok := smp.rate(); ok {
		return r
	}
	keys := []rateKey{
		{scope: scopeInterface, id: uint64(rec.InputIf)},
		{scope: scopeDomain},
	}
	if smp.hasSampler {
		keys = append([]rateKey{{scope: scopeSampler, id: smp.sampler}}, keys...)
	}
	for _, k := range keys {
		k.exporter, k.domain = hdr.exporter, hdr.domain
		lr, ok := d.rates[k]
		if !ok {
			continue
		}
		if now.Sub(lr.learned) > templateLifetime {
			delete(d.rates, k)
			continue
		}
		return lr.rate
	}
	return rate{num: 1, den: 1}
}

// sampling collects the sampling elements of one record.
type sampling struct {
	oneIn, interval, space, sampler uint64
	hasOneIn, hasInterval, hasSpace bool
	hasSampler                      bool
}

// apply records v when id is a sampling element and reports whether it
// was one.
func (s *sampling) apply(id uint16, v []byte) bool {
	switch id {
	case ieSamplingInterval, ieSamplerRandomInterval:
		n, ok := readUint[uint32](v)
		s.oneIn, s.hasOneIn = uint64(n), ok && n > 0
	case ieSamplingPacketInterval:
		n, ok := readUint[uint32](v)
		s.interval, s.hasInterval = uint64(n), ok && n > 0
	case ieSamplingPacketSpace:
		n, ok := readUint[uint32](v)
		s.space, s.hasSpace = uint64(n), ok
	case ieSamplerID, ieSelectorID:
		s.sampler, s.hasSampler = readUint[uint64](v)
	default:
		return false
	}
	return true
}

func (s *sampling) rate() (rate, bool) {
	switch {
	case s.hasOneIn:
		return rate{num: s.oneIn, den: 1}, true
	case s.hasInterval && s.hasSpace:
		return rate{num: s.interval + s.space, den: s.interval}, true
	}
	return rate{}, false
}

func (d *Decoder) learnRate(key rateKey, r rate, now time.Time) {
	if _, ok := d.rates[key]; !ok && len(d.rates) >= maxTemplates {
		evictOldest(d.rates, func(lr learnedRate) time.Time { return lr.learned })
	}
	d.rates[key] = learnedRate{rate: r, learned: now}
}
