package enumerate_test

import (
	"errors"
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// icmp.go had all 25 of its functions at 0.0% coverage (#211), because
// NewICMPPinger opens a raw socket and needs root or CAP_NET_RAW. These cover
// the parsing and bookkeeping that never reach the socket — the parts where a
// bug is silent rather than loud.

func TestSweepConfigDefaults(t *testing.T) {
	def := enumerate.DefaultSweepConfig()
	if def.Workers <= 0 {
		t.Errorf("default workers = %d, want > 0", def.Workers)
	}
	if def.JitterMin != 0 || def.JitterMax != 0 {
		t.Errorf("default jitter = %v..%v, want no jitter", def.JitterMin, def.JitterMax)
	}

	polite := enumerate.PoliteSweepConfig()
	// The polite profile exists to be gentler on an IDS, so it must actually
	// differ: fewer workers and non-zero jitter, with a usable range.
	if polite.Workers >= def.Workers {
		t.Errorf("polite workers = %d, want fewer than default %d", polite.Workers, def.Workers)
	}
	if polite.JitterMax <= 0 {
		t.Errorf("polite JitterMax = %v, want > 0", polite.JitterMax)
	}
	if polite.JitterMin > polite.JitterMax {
		t.Errorf("polite jitter range inverted: %v..%v", polite.JitterMin, polite.JitterMax)
	}
}

func TestNextSeqWrapsWithinMask(t *testing.T) {
	a := enumerate.NewICMPPingerWithoutSocket(1234)

	seen := make(map[int]bool)
	prev := -1
	for range 100 {
		seq := a.NextSeq()
		if seq < 0 || seq > 0xffff {
			t.Fatalf("seq %d outside the 16-bit ICMP field", seq)
		}
		if seen[seq] {
			t.Fatalf("seq %d repeated within 100 calls", seq)
		}
		seen[seq] = true
		if prev >= 0 && seq != prev+1 {
			t.Fatalf("seq jumped from %d to %d", prev, seq)
		}
		prev = seq
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestHandleReadErrorDistinguishesTimeoutFromClose(t *testing.T) {
	a := enumerate.NewICMPPingerWithoutSocket(1)

	// A read deadline expiring is the normal case — the receiver loop polls
	// stopCh between reads — and must not be mistaken for a closed socket.
	shouldExit, cont := a.HandleReadError(timeoutError{})
	if shouldExit || !cont {
		t.Errorf("timeout: shouldExit=%v continue=%v, want false/true", shouldExit, cont)
	}

	shouldExit, cont = a.HandleReadError(net.ErrClosed)
	if !shouldExit || cont {
		t.Errorf("closed socket: shouldExit=%v continue=%v, want true/false", shouldExit, cont)
	}

	shouldExit, cont = a.HandleReadError(errors.New("some other failure"))
	if !shouldExit || cont {
		t.Errorf("unknown error: shouldExit=%v continue=%v, want true/false", shouldExit, cont)
	}
}

func echoReply(t *testing.T, id, seq int, typ icmp.Type) []byte {
	t.Helper()
	msg := icmp.Message{
		Type: typ,
		Code: 0,
		Body: &icmp.Echo{ID: id, Seq: seq, Data: []byte("seed")},
	}
	b, err := msg.Marshal(nil)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestExtractEchoReplyAcceptsOnlyOurReplies(t *testing.T) {
	const ourID = 4242
	a := enumerate.NewICMPPingerWithoutSocket(ourID)

	seq, accepted := a.ExtractEchoReplySeq(echoReply(t, ourID, 7, ipv4.ICMPTypeEchoReply))
	if !accepted || seq != 7 {
		t.Errorf("our reply: seq=%d accepted=%v, want 7/true", seq, accepted)
	}

	rejects := []struct {
		name string
		data []byte
	}{
		// Another process's pings arrive on the same raw socket. Accepting one
		// would complete the wrong pending entry and report a bogus RTT.
		{"another pinger's reply", echoReply(t, ourID+1, 7, ipv4.ICMPTypeEchoReply)},
		// An echo *request* is not a reply — the socket sees both.
		{"an echo request", echoReply(t, ourID, 7, ipv4.ICMPTypeEcho)},
		{"an unparseable message", []byte{0xff, 0x00}},
	}
	for _, tc := range rejects {
		_, ok := a.ExtractEchoReplySeq(tc.data)
		if ok {
			t.Errorf("accepted %s", tc.name)
		}
	}
}

func TestCompletePendingPingRemovesExactlyOnce(t *testing.T) {
	a := enumerate.NewICMPPingerWithoutSocket(1)
	a.AddPending(1, "192.0.2.1")
	a.AddPending(2, "192.0.2.2")

	ip, completed := a.CompletePendingPing(1)
	if !completed || ip != "192.0.2.1" {
		t.Errorf("first completion: ip=%q completed=%v", ip, completed)
	}
	if got := a.PendingCount(); got != 1 {
		t.Errorf("pending after completion = %d, want 1", got)
	}

	// A duplicate reply — the network can deliver one — must not complete the
	// entry a second time.
	_, again := a.CompletePendingPing(1)
	if again {
		t.Error("completed the same sequence twice")
	}
	_, unknown := a.CompletePendingPing(999)
	if unknown {
		t.Error("completed a sequence that was never pending")
	}
	if got := a.PendingCount(); got != 1 {
		t.Errorf("pending after spurious completions = %d, want 1", got)
	}
}
