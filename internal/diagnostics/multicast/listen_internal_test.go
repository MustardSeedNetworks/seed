package multicast

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"slices"
	"testing"
	"time"
)

// datagram is one canned read: who sent it, where it was addressed, how big.
type datagram struct {
	src netip.Addr
	dst netip.Addr
	n   int
}

// fakeSource replays datagrams, then blocks until the collector's deadline or
// context ends it — which is what a quiet socket does.
type fakeSource struct {
	datagrams []datagram
	next      int
	filters   bool
	readErr   error
	deadline  time.Time
	now       func() time.Time
	advance   func(time.Duration)
	perRead   time.Duration
}

func (f *fakeSource) setDeadline(t time.Time) { f.deadline = t }

func (f *fakeSource) read(buf []byte) (int, netip.Addr, netip.Addr, error) {
	if f.next < len(f.datagrams) {
		d := f.datagrams[f.next]
		f.next++
		if f.advance != nil {
			f.advance(f.perRead)
		}
		return min(d.n, len(buf)), d.src, d.dst, nil
	}
	if f.readErr != nil {
		return 0, netip.Addr{}, netip.Addr{}, f.readErr
	}
	if f.advance != nil && f.now().Before(f.deadline) {
		f.advance(f.deadline.Sub(f.now()))
	}
	return 0, netip.Addr{}, netip.Addr{}, os.ErrDeadlineExceeded
}

func (f *fakeSource) filtersByDestination() bool { return f.filters }

// interrupt is a no-op: the fake never blocks, and collect re-checks the
// context after every read.
func (f *fakeSource) interrupt() {}

// clock is a manual clock the fake source advances as it "receives".
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newClock() *clock                   { return &clock{t: time.Unix(1_800_000_000, 0)} }
func addr(s string) netip.Addr           { return netip.MustParseAddr(s) }
func testSpec(window time.Duration) spec {
	return spec{group: addr("239.1.1.1"), port: 5004, iface: "en0", window: window}
}

func mustCollect(t *testing.T, src packetSource, s spec, now func() time.Time) *ListenResult {
	t.Helper()
	res, err := collect(t.Context(), src, s, now)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return res
}

func TestParseRejectsWhatCannotBeJoined(t *testing.T) {
	t.Parallel()

	up := func(string) (ifaceInfo, error) { return ifaceInfo{up: true, multicast: true}, nil }
	cases := []struct {
		name   string
		req    ListenRequest
		lookup func(string) (ifaceInfo, error)
		want   error
	}{
		{"unicast group", ListenRequest{Group: "10.0.0.1", Port: 5004, Interface: "en0"}, up, ErrNotMulticast},
		{"not an address", ListenRequest{Group: "iptv", Port: 5004, Interface: "en0"}, up, ErrNotMulticast},
		{"port zero", ListenRequest{Group: "239.1.1.1", Port: 0, Interface: "en0"}, up, ErrPort},
		{"port too large", ListenRequest{Group: "239.1.1.1", Port: 65536, Interface: "en0"}, up, ErrPort},
		{"no interface", ListenRequest{Group: "239.1.1.1", Port: 5004}, up, ErrInterfaceRequired},
		{
			"unknown interface",
			ListenRequest{Group: "239.1.1.1", Port: 5004, Interface: "nope0"},
			func(string) (ifaceInfo, error) { return ifaceInfo{}, errors.New("no such network interface") },
			ErrInterface,
		},
		{
			"interface down",
			ListenRequest{Group: "239.1.1.1", Port: 5004, Interface: "en0"},
			func(string) (ifaceInfo, error) { return ifaceInfo{multicast: true}, nil }, ErrInterface,
		},
		{
			"interface without multicast",
			ListenRequest{Group: "239.1.1.1", Port: 5004, Interface: "en0"},
			func(string) (ifaceInfo, error) { return ifaceInfo{up: true}, nil }, ErrInterface,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parse(tc.req, tc.lookup); !errors.Is(err, tc.want) {
				t.Fatalf("parse(%+v) error = %v, want %v", tc.req, err, tc.want)
			}
		})
	}
}

func TestParseAcceptsBothFamiliesAndBoundsTheWindow(t *testing.T) {
	t.Parallel()

	up := func(string) (ifaceInfo, error) { return ifaceInfo{up: true, multicast: true}, nil }
	cases := []struct {
		name    string
		req     ListenRequest
		family  string
		wantWin time.Duration
	}{
		{"default window", ListenRequest{Group: "239.1.1.1", Port: 5004, Interface: "en0"}, "udp4", DefaultWindow},
		{
			"clamped to max",
			ListenRequest{Group: "239.1.1.1", Port: 5004, Interface: "en0", DurationSeconds: 3600},
			"udp4",
			MaxWindow,
		},
		{
			"explicit",
			ListenRequest{Group: "ff15::1234", Port: 5004, Interface: "en0", DurationSeconds: 3},
			"udp6",
			3 * time.Second,
		},
		{
			"v4-mapped v6 is v4",
			ListenRequest{Group: "::ffff:239.1.1.1", Port: 5004, Interface: "en0"},
			"udp4",
			DefaultWindow,
		},
		{"zoned v6", ListenRequest{Group: "ff02::fb%en0", Port: 5353, Interface: "en0"}, "udp6", DefaultWindow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, err := parse(tc.req, up)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if s.network() != tc.family || s.window != tc.wantWin {
				t.Fatalf("network %s window %s, want %s %s", s.network(), s.window, tc.family, tc.wantWin)
			}
			if s.group.Zone() != "" || s.group.Is4In6() {
				t.Fatalf("group %s kept a zone or a v4-mapped form; wire destinations carry neither", s.group)
			}
		})
	}
}

// The socket is bound to the port, not the group, so on a host where another
// application has joined a different group on the same port the kernel hands
// this socket that traffic too. Counting it would answer "yes, the stream is
// arriving" for a stream that is not.
func TestCollectCountsOnlyTheRequestedGroup(t *testing.T) {
	t.Parallel()

	c := newClock()
	src := &fakeSource{
		filters: true, now: c.now, advance: c.advance,
		datagrams: []datagram{
			{addr("10.0.0.5"), addr("239.1.1.1"), 1316},
			{addr("10.0.0.6"), addr("239.9.9.9"), 1316},
			{addr("10.0.0.5"), addr("10.0.0.2"), 100},
			{addr("10.0.0.5"), addr("239.1.1.1"), 1316},
		},
	}
	res := mustCollect(t, src, testSpec(2*time.Second), c.now)

	if res.Packets != 2 || res.Bytes != 2632 {
		t.Fatalf("packets %d bytes %d, want 2 and 2632", res.Packets, res.Bytes)
	}
	if len(res.Sources) != 1 || res.Sources[0].Address != "10.0.0.5" {
		t.Fatalf("sources = %+v, want only 10.0.0.5", res.Sources)
	}
	if !res.GroupFiltered {
		t.Fatal("GroupFiltered = false on a source that reports destinations")
	}
}

// Where the platform cannot report a datagram's destination (Windows), every
// datagram on the port is counted, and the result must say so rather than
// present the count as the group's.
func TestCollectSaysWhenItCouldNotFilterByGroup(t *testing.T) {
	t.Parallel()

	c := newClock()
	src := &fakeSource{
		now: c.now, advance: c.advance,
		datagrams: []datagram{{addr("10.0.0.5"), netip.Addr{}, 1316}},
	}
	res := mustCollect(t, src, testSpec(time.Second), c.now)

	if res.GroupFiltered || res.Packets != 1 {
		t.Fatalf("GroupFiltered %v packets %d, want false and 1", res.GroupFiltered, res.Packets)
	}
}

func TestCollectRanksSourcesAndCapsThem(t *testing.T) {
	t.Parallel()

	c := newClock()
	var dgs []datagram
	for i := range maxSources + 5 {
		dgs = append(dgs, datagram{netip.AddrFrom4([4]byte{10, 0, 1, byte(i)}), addr("239.1.1.1"), 100})
	}
	for range 3 {
		dgs = append(dgs, datagram{addr("10.0.0.9"), addr("239.1.1.1"), 1316})
	}
	src := &fakeSource{filters: true, now: c.now, advance: c.advance, datagrams: dgs}
	res := mustCollect(t, src, testSpec(time.Second), c.now)

	if len(res.Sources) != maxSources || !res.SourcesTruncated {
		t.Fatalf("%d sources truncated=%v, want %d and true", len(res.Sources), res.SourcesTruncated, maxSources)
	}
	if res.Sources[0].Address != "10.0.0.9" || res.Sources[0].Packets != 3 {
		t.Fatalf("top source = %+v, want 10.0.0.9 with 3 packets", res.Sources[0])
	}
	if want := uint64(maxSources + 5 + 3); res.Packets != want {
		t.Fatalf("packets %d, want %d: sources past the cap still count", res.Packets, want)
	}
	if !slices.IsSortedFunc(res.Sources, func(a, b ListenSource) int { return int(b.Packets) - int(a.Packets) }) {
		t.Fatalf("sources not ranked by packets: %+v", res.Sources)
	}
}

func TestCollectReportsRateOverTheTimeActuallyListened(t *testing.T) {
	t.Parallel()

	c := newClock()
	src := &fakeSource{
		filters: true, now: c.now, advance: c.advance, perRead: 10 * time.Millisecond,
		datagrams: []datagram{
			{addr("10.0.0.5"), addr("239.1.1.1"), 1316},
			{addr("10.0.0.5"), addr("239.1.1.1"), 1316},
			{addr("10.0.0.5"), addr("239.1.1.1"), 1316},
			{addr("10.0.0.5"), addr("239.1.1.1"), 1316},
		},
	}
	res := mustCollect(t, src, testSpec(2*time.Second), c.now)

	if res.ListenedMs != 2000 {
		t.Fatalf("ListenedMs = %d, want 2000", res.ListenedMs)
	}
	if res.PacketsPerSecond != 2 {
		t.Fatalf("PacketsPerSecond = %v, want 2", res.PacketsPerSecond)
	}
}

// Stopping a listen early is how an operator says "I have my answer"; the
// packets already counted are that answer.
func TestCancelledListenKeepsWhatItHeard(t *testing.T) {
	t.Parallel()

	c := newClock()
	ctx, cancel := context.WithCancel(t.Context())
	src := &fakeSource{
		filters: true, now: c.now, perRead: 500 * time.Millisecond,
		datagrams: []datagram{{addr("10.0.0.5"), addr("239.1.1.1"), 1316}},
	}
	src.advance = func(d time.Duration) {
		c.advance(d)
		cancel()
	}
	res, err := collect(ctx, src, testSpec(10*time.Second), c.now)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	if res.Packets != 1 || res.ListenedMs != 500 {
		t.Fatalf("packets %d listened %dms, want 1 and 500", res.Packets, res.ListenedMs)
	}
}

func TestCollectSurfacesAReadFailure(t *testing.T) {
	t.Parallel()

	c := newClock()
	boom := errors.New("socket closed underneath")
	src := &fakeSource{filters: true, now: c.now, advance: c.advance, readErr: boom}

	if _, err := collect(t.Context(), src, testSpec(time.Second), c.now); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}
