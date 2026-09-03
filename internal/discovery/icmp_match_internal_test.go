//go:build !windows

package discovery

import (
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// quoteProbe builds the payload an ICMP error carries: the IP header of the
// datagram that provoked it, followed by that datagram's first eight bytes.
// This is what a real router sends back, and it is the only way to tell whose
// probe an error answers.
func quoteProbe(t *testing.T, flowID, seq int) []byte {
	t.Helper()

	probe, err := buildICMPEchoRequest(flowID, seq)
	if err != nil {
		t.Fatalf("buildICMPEchoRequest: %v", err)
	}

	header := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TotalLen: ipv4.HeaderLen + len(probe),
		TTL:      1,
		Protocol: protocolICMP,
		// Arbitrary: nothing routes these, they only have to make the
		// quoted IP header well-formed.
		Src: net.IPv4(192, 0, 2, 1),
		Dst: net.IPv4(198, 51, 100, 1),
	}
	raw, err := header.Marshal()
	if err != nil {
		t.Fatalf("marshal ip header: %v", err)
	}
	return append(raw, probe...)
}

func TestEchoReplyIdentifiesItsProbe(t *testing.T) {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEchoReply,
		Body: &icmp.Echo{ID: 4242, Seq: 7, Data: []byte("SEED")},
	}

	id, seq, ok := echoIDSeq(msg)
	if !ok || id != 4242 || seq != 7 {
		t.Fatalf("echoIDSeq = (%d, %d, %v), want (4242, 7, true)", id, seq, ok)
	}
}

// A time-exceeded is how every intermediate hop answers, so recovering the
// probe from its quotation is what makes traceroute work at all under
// concurrent ICMP traffic.
func TestTimeExceededIdentifiesTheQuotedProbe(t *testing.T) {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeTimeExceeded,
		Body: &icmp.TimeExceeded{Data: quoteProbe(t, 1234, 9)},
	}

	id, seq, ok := echoIDSeq(msg)
	if !ok || id != 1234 || seq != 9 {
		t.Fatalf("echoIDSeq = (%d, %d, %v), want (1234, 9, true)", id, seq, ok)
	}
}

func TestDestinationUnreachableIdentifiesTheQuotedProbe(t *testing.T) {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeDestinationUnreachable,
		Body: &icmp.DstUnreach{Data: quoteProbe(t, 555, 3)},
	}

	id, seq, ok := echoIDSeq(msg)
	if !ok || id != 555 || seq != 3 {
		t.Fatalf("echoIDSeq = (%d, %d, %v), want (555, 3, true)", id, seq, ok)
	}
}

// An error too short to carry the eight bytes it is obliged to quote cannot
// identify anything, and guessing would attribute it to whichever probe was
// outstanding.
func TestTruncatedQuotationIdentifiesNothing(t *testing.T) {
	full := quoteProbe(t, 77, 2)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeTimeExceeded,
		Body: &icmp.TimeExceeded{Data: full[:ipv4.HeaderLen+4]},
	}

	if _, _, ok := echoIDSeq(msg); ok {
		t.Error("echoIDSeq accepted a truncated quotation")
	}
}

func TestNonEchoMessageIdentifiesNothing(t *testing.T) {
	msg := &icmp.Message{Type: ipv4.ICMPTypeRedirect, Body: &icmp.RawBody{}}
	if _, _, ok := echoIDSeq(msg); ok {
		t.Error("echoIDSeq claimed a redirect identifies a probe")
	}
}

// Two tracers on one host share the raw ICMP socket, so distinct flow
// identifiers are what keep one from consuming the other's replies.
func TestTracersGetDistinctFlowIDs(t *testing.T) {
	const samples = 32
	seen := make(map[int]int, samples)
	for range samples {
		seen[NewTracer(0, 0).flowID]++
	}
	if len(seen) < samples/2 {
		t.Errorf("only %d distinct flow ids across %d tracers: %v", len(seen), samples, seen)
	}
	for id := range seen {
		if id <= 0 || id > 0xffff {
			t.Errorf("flow id %d outside the 16-bit echo identifier field", id)
		}
	}
}

func TestWithFlowIDPinsTheIdentifier(t *testing.T) {
	tracer := NewTracer(0, 0).WithFlowID(4096)
	if tracer.flowID != 4096 {
		t.Fatalf("flowID = %d, want 4096", tracer.flowID)
	}

	probe, err := buildICMPEchoRequest(tracer.flowID, 1)
	if err != nil {
		t.Fatalf("buildICMPEchoRequest: %v", err)
	}
	parsed, err := icmp.ParseMessage(protocolICMP, probe)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	id, _, ok := echoIDSeq(parsed)
	if !ok || id != 4096 {
		t.Fatalf("probe carries id %d (ok=%v), want 4096", id, ok)
	}
}
