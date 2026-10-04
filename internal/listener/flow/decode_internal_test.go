package flow

import (
	"encoding/binary"
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
