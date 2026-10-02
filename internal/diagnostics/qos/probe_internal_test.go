package qos

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	t.Parallel()

	cases := map[int]string{
		0: "CS0", 8: "CS1", 56: "CS7", 46: "EF", 44: "VOICE-ADMIT",
		10: "AF11", 14: "AF13", 18: "AF21", 34: "AF41", 38: "AF43",
		1: "", 40: "CS5", 42: "", 63: "", -8: "",
	}
	for dscp, want := range cases {
		if got := Name(dscp); got != want {
			t.Errorf("Name(%d) = %q, want %q", dscp, got, want)
		}
	}
}

func TestParseSend(t *testing.T) {
	t.Parallel()

	s, err := parseSend(SendRequest{Target: "192.0.2.7", Port: 5004})
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	if got := s.classes(); !reflect.DeepEqual(got, []uint8{0, 10, 18, 26, 34, 46}) || s.count != DefaultCount {
		t.Fatalf("defaults gave classes %v count %d", got, s.count)
	}
	if s.target != netip.MustParseAddrPort("192.0.2.7:5004") {
		t.Fatalf("target = %s", s.target)
	}

	// A repeated class is one class: the listener keys on the value.
	s, err = parseSend(SendRequest{Target: "::ffff:192.0.2.7", Port: 1, DSCP: []int{46, 0, 46}, Count: MaxCount})
	if err != nil || !reflect.DeepEqual(s.classes(), []uint8{0, 46}) || s.target.Addr().Is4In6() {
		t.Fatalf("got %+v, %v", s, err)
	}

	refused := map[string]struct {
		req  SendRequest
		want error
	}{
		"hostname":    {SendRequest{Target: "listener.example", Port: 5004}, ErrTarget},
		"empty":       {SendRequest{Port: 5004}, ErrTarget},
		"multicast":   {SendRequest{Target: "239.1.1.1", Port: 5004}, ErrTarget},
		"unspecified": {SendRequest{Target: "::", Port: 5004}, ErrTarget},
		"broadcast":   {SendRequest{Target: "255.255.255.255", Port: 5004}, ErrTarget},
		"port 0":      {SendRequest{Target: "192.0.2.7"}, ErrPort},
		"port high":   {SendRequest{Target: "192.0.2.7", Port: 65536}, ErrPort},
		"dscp high":   {SendRequest{Target: "192.0.2.7", Port: 5004, DSCP: []int{64}}, ErrDSCP},
		"dscp neg":    {SendRequest{Target: "192.0.2.7", Port: 5004, DSCP: []int{-1}}, ErrDSCP},
		"count high":  {SendRequest{Target: "192.0.2.7", Port: 5004, Count: MaxCount + 1}, ErrCount},
		"count neg":   {SendRequest{Target: "192.0.2.7", Port: 5004, Count: -1}, ErrCount},
	}
	for name, tc := range refused {
		if _, refusal := parseSend(tc.req); !errors.Is(refusal, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, refusal, tc.want)
		}
	}
}

func TestParseListen(t *testing.T) {
	t.Parallel()

	s, err := parseListen(ListenRequest{Port: 5004})
	if err != nil || s.v6 || s.window != DefaultWindow || s.network() != "udp4" {
		t.Fatalf("defaults gave %+v, %v", s, err)
	}
	s, err = parseListen(ListenRequest{Port: 5004, Family: "ipv6", DurationSeconds: 600})
	if err != nil || !s.v6 || s.window != MaxWindow || s.family() != "ipv6" {
		t.Fatalf("ipv6 clamp gave %+v, %v", s, err)
	}
	if _, err = parseListen(ListenRequest{Port: 0}); !errors.Is(err, ErrPort) {
		t.Fatalf("port 0: %v", err)
	}
	if _, err = parseListen(ListenRequest{Port: 5004, Family: "udp"}); !errors.Is(err, ErrFamily) {
		t.Fatalf("bad family: %v", err)
	}
}

func TestProbeRoundTrip(t *testing.T) {
	t.Parallel()

	p := probe{run: 0xfeedface01020304, mask: 1<<46 | 1<<0, dscp: 46, count: 5, seq: 4}
	b := p.encode()
	if len(b) != probeSize {
		t.Fatalf("probe is %d bytes", len(b))
	}
	got, ok := decodeProbe(b)
	if !ok || got != p {
		t.Fatalf("decoded %+v %v, want %+v", got, ok, p)
	}

	refused := map[string]func([]byte){
		"short":            func([]byte) {},
		"magic":            func(b []byte) { b[0] = 'X' },
		"dscp not in mask": func(b []byte) { b[24] = 34 },
		"dscp over 63":     func(b []byte) { b[24] = 64 },
		"count zero":       func(b []byte) { b[25], b[26] = 0, 0 },
		"count over max":   func(b []byte) { b[25], b[26] = 0, MaxCount+1 },
		"seq past count":   func(b []byte) { b[27], b[28] = 0, 5 },
	}
	for name, mutate := range refused {
		bad := p.encode()
		mutate(bad)
		if name == "short" {
			bad = bad[:28]
		}
		if _, decoded := decodeProbe(bad); decoded {
			t.Errorf("%s: decoded", name)
		}
	}
}

// fakeConn records the marking in force when each datagram was written.
type fakeConn struct {
	dscp     uint8
	written  []probe
	marks    []uint8
	failAt   int
	setDSCPE error
}

func (c *fakeConn) setDSCP(d uint8) error {
	c.dscp = d
	return c.setDSCPE
}

func (c *fakeConn) write(b []byte) error {
	if c.failAt > 0 && len(c.written) == c.failAt {
		return errors.New("network is unreachable")
	}
	p, ok := decodeProbe(b)
	if !ok {
		return errors.New("wrote a datagram that is not a probe")
	}
	c.written = append(c.written, p)
	c.marks = append(c.marks, c.dscp)
	return nil
}

func noWait(context.Context) error { return nil }

func TestSendMarksEachProbeWithItsOwnClass(t *testing.T) {
	t.Parallel()

	s, err := parseSend(SendRequest{Target: "192.0.2.7", Port: 5004, DSCP: []int{46, 0, 26}, Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	conn := &fakeConn{}
	res, err := send(t.Context(), conn, s, 0xabc, noWait)
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// Classes interleave, so a congested queue costs each class alike.
	wantOrder := []uint8{0, 26, 46, 0, 26, 46}
	if !reflect.DeepEqual(conn.marks, wantOrder) {
		t.Fatalf("marks in write order %v, want %v", conn.marks, wantOrder)
	}
	for i, p := range conn.written {
		if p.dscp != conn.marks[i] || p.run != 0xabc || p.mask != s.mask || p.count != 2 || int(p.seq) != i/3 {
			t.Fatalf("probe %d = %+v written under DSCP %d", i, p, conn.marks[i])
		}
	}
	want := &SendResult{
		Target: "192.0.2.7", Port: 5004, RunID: "0000000000000abc", Count: 2,
		Classes: []SentClass{{0, "CS0", 2}, {26, "AF31", 2}, {46, "EF", 2}},
	}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("result %+v, want %+v", res, want)
	}
}

// Stopping a send reports what already went out, so the listener's counts
// can be read against it.
func TestCancelledSendReportsWhatWentOut(t *testing.T) {
	t.Parallel()

	s, _ := parseSend(SendRequest{Target: "192.0.2.7", Port: 5004, DSCP: []int{46, 0}, Count: 3})
	ctx, cancel := context.WithCancel(t.Context())
	writes := 0
	wait := func(ctx context.Context) error {
		writes++
		if writes == 3 {
			cancel()
		}
		return ctx.Err()
	}
	res, err := send(ctx, &fakeConn{}, s, 1, wait)
	if err != nil {
		t.Fatalf("cancelled send: %v", err)
	}
	if res.Classes[0].Sent != 2 || res.Classes[1].Sent != 1 {
		t.Fatalf("classes %+v, want 2 CS0 and 1 EF", res.Classes)
	}
}

func TestSendFailsOnASocketError(t *testing.T) {
	t.Parallel()

	s, _ := parseSend(SendRequest{Target: "192.0.2.7", Port: 5004})
	if _, err := send(t.Context(), &fakeConn{failAt: 2}, s, 1, noWait); err == nil ||
		!strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("write error: %v", err)
	}
	if _, err := send(t.Context(), &fakeConn{setDSCPE: errors.New("EPERM")}, s, 1, noWait); err == nil ||
		!strings.Contains(err.Error(), "set DSCP") {
		t.Fatalf("mark error: %v", err)
	}
}

type datagram struct {
	b    []byte
	from string
	dscp int
}

// fakeSource hands collect a fixed sequence of datagrams, then the window's
// deadline.
type fakeSource struct {
	grams    []datagram
	observes bool
}

func (f *fakeSource) read(buf []byte) (int, netip.Addr, int, error) {
	if len(f.grams) == 0 {
		return 0, netip.Addr{}, 0, os.ErrDeadlineExceeded
	}
	g := f.grams[0]
	f.grams = f.grams[1:]
	return copy(buf, g.b), netip.MustParseAddr(g.from), g.dscp, nil
}

func (*fakeSource) setDeadline(time.Time) {}
func (*fakeSource) interrupt()            {}
func (f *fakeSource) observesDSCP() bool  { return f.observes }

// burst is what one send of classes x count delivers, arriving as mark maps.
func burst(from string, run uint64, classes []int, count int, mark func(sent, seq int) int) []datagram {
	var mask uint64
	for _, d := range classes {
		mask |= 1 << d
	}
	var out []datagram
	for seq := range uint16(count) {
		for _, d := range classes {
			m := mark(d, int(seq))
			if m == dropped {
				continue
			}
			p := probe{run: run, mask: mask, dscp: uint8(d), count: uint16(count), seq: seq}
			out = append(out, datagram{p.encode(), from, m})
		}
	}
	return out
}

const dropped = -2

func listenOnce(t *testing.T, grams []datagram, observes bool) *ListenResult {
	t.Helper()
	s, _ := parseListen(ListenRequest{Port: 5004})
	res, err := collect(t.Context(), &fakeSource{grams: grams, observes: observes}, s, time.Now)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return res
}

func verdicts(r RunResult) map[int]Verdict {
	out := map[int]Verdict{}
	for _, c := range r.Classes {
		out[c.SentDSCP] = c.Verdict
	}
	return out
}

func TestListenVerdicts(t *testing.T) {
	t.Parallel()

	classes := []int{46, 34, 26, 18, 0}
	res := listenOnce(t, burst("192.0.2.1", 7, classes, 4, func(sent, seq int) int {
		switch sent {
		case 46: // a port that does not trust the marking
			return 0
		case 34: // rewritten on one queue and not another
			if seq%2 == 0 {
				return 34
			}
			return 32
		case 18: // a policy that drops the class
			return dropped
		}
		return sent
	}), true)

	if len(res.Runs) != 1 {
		t.Fatalf("runs %+v", res.Runs)
	}
	run := res.Runs[0]
	want := map[int]Verdict{
		46: VerdictRemarked, 34: VerdictMixed, 26: VerdictPreserved, 18: VerdictLost, 0: VerdictPreserved,
	}
	if got := verdicts(run); !reflect.DeepEqual(got, want) {
		t.Fatalf("verdicts %v, want %v", got, want)
	}
	if run.Preserved || run.Sender != "192.0.2.1" || run.RunID != "0000000000000007" {
		t.Fatalf("run %+v", run)
	}
	if res.Probes != 16 || res.Ignored != 0 {
		t.Fatalf("probes %d ignored %d, want 16 and 0", res.Probes, res.Ignored)
	}

	ef := run.Classes[len(run.Classes)-1]
	if ef.SentDSCP != 46 || ef.SentName != "EF" || ef.Expected != 4 || ef.Received != 4 ||
		!reflect.DeepEqual(ef.Observed, []ObservedDSCP{{0, "CS0", 4}}) {
		t.Fatalf("EF class %+v", ef)
	}
	lost := run.Classes[1]
	if lost.SentDSCP != 18 || lost.Received != 0 || lost.Observed == nil {
		t.Fatalf("lost class %+v: Observed must be an empty list, not null", lost)
	}
}

func TestListenPreservedRun(t *testing.T) {
	t.Parallel()

	res := listenOnce(t, burst("192.0.2.1", 1, defaultClasses(), 3, func(sent, _ int) int { return sent }), true)
	if len(res.Runs) != 1 || !res.Runs[0].Preserved || !res.DSCPObserved {
		t.Fatalf("result %+v", res)
	}
}

// A platform that cannot read the marking must not report a class preserved
// or remarked on no evidence.
func TestListenWithoutTOSReportsUnobserved(t *testing.T) {
	t.Parallel()

	res := listenOnce(t, burst("192.0.2.1", 1, []int{46, 0}, 2, func(int, int) int { return -1 }), false)
	run := res.Runs[0]
	if res.DSCPObserved || run.Preserved {
		t.Fatalf("result %+v", res)
	}
	for _, c := range run.Classes {
		if c.Verdict != VerdictUnobserved || c.Received != 2 || len(c.Observed) != 0 {
			t.Fatalf("class %+v", c)
		}
	}
}

func TestListenCountsEachProbeOnce(t *testing.T) {
	t.Parallel()

	grams := burst("192.0.2.1", 1, []int{46}, 2, func(sent, _ int) int { return sent })
	grams = append(grams, grams[0])
	res := listenOnce(t, grams, true)
	if c := res.Runs[0].Classes[0]; c.Received != 2 || c.Observed[0].Count != 2 {
		t.Fatalf("duplicate counted twice: %+v", c)
	}
	if res.Probes != 3 {
		t.Fatalf("probes %d, want 3 with the duplicate", res.Probes)
	}
}

func TestListenIgnoresWhatIsNotAProbe(t *testing.T) {
	t.Parallel()

	grams := burst("192.0.2.1", 1, []int{46}, 2, func(sent, _ int) int { return sent })
	// A stray datagram, and a forged probe of the same run naming a class
	// set the run's first probe did not.
	forged := probe{run: 1, mask: 1<<46 | 1<<0, dscp: 0, count: 2, seq: 0}.encode()
	grams = append(grams, datagram{[]byte("GET / HTTP/1.1\r\n"), "192.0.2.9", 0}, datagram{forged, "192.0.2.1", 0})
	res := listenOnce(t, grams, true)
	if res.Ignored != 2 || res.Probes != 2 || len(res.Runs) != 1 || len(res.Runs[0].Classes) != 1 {
		t.Fatalf("result %+v", res)
	}
}

func TestListenSeparatesSendersAndRuns(t *testing.T) {
	t.Parallel()

	same := func(sent, _ int) int { return sent }
	var grams []datagram
	grams = append(grams, burst("192.0.2.2", 5, []int{46}, 1, same)...)
	grams = append(grams, burst("192.0.2.1", 9, []int{46}, 1, func(int, int) int { return 0 })...)
	grams = append(grams, burst("192.0.2.1", 5, []int{46}, 1, same)...)
	res := listenOnce(t, grams, true)

	var got []string
	for _, r := range res.Runs {
		got = append(got, r.Sender+"/"+r.RunID[14:]+"/"+string(r.Classes[0].Verdict))
	}
	want := []string{"192.0.2.1/05/preserved", "192.0.2.1/09/remarked", "192.0.2.2/05/preserved"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runs %v, want %v", got, want)
	}
}

func TestListenBoundsTheRunTable(t *testing.T) {
	t.Parallel()

	var grams []datagram
	for run := range uint64(maxTrackedRuns + 3) {
		grams = append(grams, burst("192.0.2.1", run, []int{0}, 1, func(sent, _ int) int { return sent })...)
	}
	res := listenOnce(t, grams, true)
	if len(res.Runs) != maxTrackedRuns || !res.RunsTruncated || res.Ignored != 3 {
		t.Fatalf("%d runs, truncated %v, ignored %d", len(res.Runs), res.RunsTruncated, res.Ignored)
	}
}
