package flow

import (
	"encoding/binary"
	"math"
	"net/netip"
	"testing"
	"time"
)

// TestTemplateCacheBounded proves an exporter cannot grow the cache past
// maxTemplates, and that the least recently refreshed template goes first.
func TestTemplateCacheBounded(t *testing.T) {
	t.Parallel()
	dec := NewDecoder()
	start := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	learn := func(exporter netip.Addr, id uint16, at time.Time) {
		t.Helper()
		body := binary.BigEndian.AppendUint16(nil, id)
		body = binary.BigEndian.AppendUint16(body, 1)
		body = binary.BigEndian.AppendUint16(body, ieSourceIPv4Address)
		body = binary.BigEndian.AppendUint16(body, 4)
		hdr := exportHeader{exporter: exporter, version: VersionNetFlow9}
		if err := dec.learnTemplates(&hdr, body, false, at); err != nil {
			t.Fatal(err)
		}
	}

	first := netip.MustParseAddr("192.0.2.1")
	learn(first, minDataSetID, start)
	for i := range maxTemplates + 10 {
		addr := netip.AddrFrom4([4]byte{10, 0, byte(i >> 8), byte(i)})
		learn(addr, minDataSetID, start.Add(time.Duration(i+1)*time.Millisecond))
	}
	if got := len(dec.templates); got != maxTemplates {
		t.Errorf("templates = %d, want %d", got, maxTemplates)
	}
	hdr := exportHeader{exporter: first, version: VersionNetFlow9}
	if dec.lookup(&hdr, minDataSetID, start) != nil {
		t.Error("the oldest template survived eviction")
	}
}

func TestRateScaleSaturates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		r    rate
		n    uint64
		want uint64
	}{
		{r: rate{num: 1, den: 1}, n: math.MaxUint64, want: math.MaxUint64},
		{r: rate{num: 10, den: 1}, n: 7, want: 70},
		{r: rate{num: 5, den: 2}, n: 3, want: 7},
		{r: rate{num: math.MaxUint32, den: 1}, n: math.MaxUint64 / 2, want: math.MaxUint64},
	} {
		if got := tc.r.scale(tc.n); got != tc.want {
			t.Errorf("%d/%d scale(%d) = %d, want %d", tc.r.num, tc.r.den, tc.n, got, tc.want)
		}
	}
}

// TestRateCacheBounded proves options records cannot grow the rate cache
// past maxTemplates.
func TestRateCacheBounded(t *testing.T) {
	t.Parallel()
	dec := NewDecoder()
	start := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	first := rateKey{exporter: netip.MustParseAddr("192.0.2.1"), scope: scopeDomain}
	dec.learnRate(first, rate{num: 10, den: 1}, start)
	for i := range maxTemplates + 10 {
		key := rateKey{exporter: first.exporter, scope: scopeInterface, id: uint64(i)}
		dec.learnRate(key, rate{num: 10, den: 1}, start.Add(time.Duration(i+1)*time.Millisecond))
	}
	if got := len(dec.rates); got != maxTemplates {
		t.Errorf("rates = %d, want %d", got, maxTemplates)
	}
	if _, ok := dec.rates[first]; ok {
		t.Error("the oldest rate survived eviction")
	}
}
