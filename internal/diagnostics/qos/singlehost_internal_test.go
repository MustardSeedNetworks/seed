package qos

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net"
	"net/netip"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
)

// The test network's addresses.
func wifiMAC() net.HardwareAddr   { return net.HardwareAddr{0x02, 0, 0, 0, 0, 0x01} }
func wiredMAC() net.HardwareAddr  { return net.HardwareAddr{0x02, 0, 0, 0, 0, 0x02} }
func routerMAC() net.HardwareAddr { return net.HardwareAddr{0x02, 0, 0, 0, 0, 0xfe} }
func routerIP() netip.Addr        { return netip.MustParseAddr("10.1.0.1") }

// fakeLAN is the network between a host's two interfaces. A frame injected
// on one interface is handed to forward, which returns the interface that
// receives it and the frame as it arrives there ("" drops it); every handle
// open on that interface reads it.
type fakeLAN struct {
	mu       sync.Mutex
	open     map[string][]*fakeHandle
	filters  map[string][]string
	injected map[string]int
	linkType map[string]layers.LinkType
	openErr  error
	forward  func(iface string, frame []byte) (string, []byte)
}

func newFakeLAN(forward func(string, []byte) (string, []byte)) *fakeLAN {
	return &fakeLAN{
		open:     map[string][]*fakeHandle{},
		filters:  map[string][]string{},
		injected: map[string]int{},
		linkType: map[string]layers.LinkType{},
		forward:  forward,
	}
}

func (l *fakeLAN) OpenLive(iface string, _ int32, _ bool, _ time.Duration) (capture.Handle, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.openErr != nil {
		return nil, l.openErr
	}
	lt, ok := l.linkType[iface]
	if !ok {
		lt = layers.LinkTypeEthernet
	}
	h := &fakeHandle{lan: l, iface: iface, linkType: lt, in: make(chan []byte, 1024), closed: make(chan struct{})}
	l.open[iface] = append(l.open[iface], h)
	return h, nil
}

func (l *fakeLAN) inject(iface string, frame []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.injected[iface]++
	to, out := l.forward(iface, bytes.Clone(frame))
	for _, h := range l.open[to] {
		h.in <- out
	}
}

func (l *fakeLAN) remove(h *fakeHandle) {
	l.mu.Lock()
	defer l.mu.Unlock()
	hs := l.open[h.iface]
	for i := range hs {
		if hs[i] == h {
			l.open[h.iface] = append(hs[:i], hs[i+1:]...)
			return
		}
	}
}

func (l *fakeLAN) stillOpen() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, hs := range l.open {
		n += len(hs)
	}
	return n
}

type fakeHandle struct {
	lan      *fakeLAN
	iface    string
	linkType layers.LinkType
	in       chan []byte
	closed   chan struct{}
	once     sync.Once
}

// ReadPacketData times out as libpcap does when a handle opened with a
// timeout sees nothing, so every idle wait exercises the read-past-timeout
// path.
func (h *fakeHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	select {
	case d := <-h.in:
		return d, gopacket.CaptureInfo{}, nil
	case <-h.closed:
		return nil, gopacket.CaptureInfo{}, io.EOF
	case <-time.After(5 * time.Millisecond):
		return nil, gopacket.CaptureInfo{}, capture.ErrTimeout
	}
}

func (h *fakeHandle) SetBPFFilter(f string) error {
	h.lan.mu.Lock()
	defer h.lan.mu.Unlock()
	h.lan.filters[h.iface] = append(h.lan.filters[h.iface], f)
	return nil
}

func (h *fakeHandle) LinkType() layers.LinkType { return h.linkType }

func (h *fakeHandle) WritePacketData(d []byte) error {
	h.lan.inject(h.iface, d)
	return nil
}

func (h *fakeHandle) Close() {
	h.once.Do(func() {
		close(h.closed)
		h.lan.remove(h)
	})
}

// hostPair is a Wi-Fi interface on 10.1.0.0/24 and a wired one on wiredNet.
func hostPair(wiredNet string) map[string]hostInterface {
	return map[string]hostInterface{
		"wlan0": {name: "wlan0", mac: wifiMAC(), prefix: netip.MustParsePrefix("10.1.0.10/24")},
		"eth0":  {name: "eth0", mac: wiredMAC(), prefix: netip.MustParsePrefix(wiredNet)},
	}
}

func lookupIn(ifaces map[string]hostInterface) func(string) (hostInterface, error) {
	return func(name string) (hostInterface, error) {
		i, ok := ifaces[name]
		if !ok {
			return hostInterface{}, errors.New("no such interface")
		}
		return i, nil
	}
}

// wifiRoutes is the Wi-Fi interface's table: its own network and a default
// route through the router.
func wifiRoutes() ([]gateway.RouteInfo, error) {
	return []gateway.RouteInfo{
		{Destination: "10.1.0.0", Prefix: 24, Interface: "wlan0", Family: "inet"},
		{Destination: "0.0.0.0", Prefix: 0, Gateway: routerIP().String(), Interface: "wlan0", Family: "inet"},
		{Destination: "10.2.0.0", Prefix: 24, Interface: "eth0", Family: "inet"},
	}, nil
}

func testSingleHost(lan *fakeLAN, ifaces map[string]hostInterface) singleHost {
	return singleHost{
		opener:     lan,
		lookup:     lookupIn(ifaces),
		routes:     wifiRoutes,
		pace:       func(ctx context.Context) error { return ctx.Err() },
		settle:     500 * time.Millisecond,
		arpTimeout: 200 * time.Millisecond,
	}
}

// accessPoint forwards the Wi-Fi interface's probes to the wired interface
// when they are addressed to nextMAC, rewriting each DSCP through remark
// (a negative value drops the probe), and answers ARP for the router.
func accessPoint(nextMAC net.HardwareAddr, remark func(dscp, seq int) int) func(string, []byte) (string, []byte) {
	seen := map[int]int{}
	return func(iface string, frame []byte) (string, []byte) {
		pkt := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)
		if arp, ok := pkt.Layer(layers.LayerTypeARP).(*layers.ARP); ok {
			if iface != "wlan0" || arp.Operation != layers.ARPRequest ||
				!bytes.Equal(arp.DstProtAddress, routerIP().AsSlice()) {
				return "", nil
			}
			return "wlan0", arpReply(arp)
		}
		eth, _ := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		ip, _ := pkt.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if iface != "wlan0" || eth == nil || ip == nil || !bytes.Equal(eth.DstMAC, nextMAC) {
			return "", nil
		}
		dscp := int(ip.TOS >> ecnBits)
		out := remark(dscp, seen[dscp])
		seen[dscp]++
		if out < 0 {
			return "", nil
		}
		frame[14+1] = byte(out << ecnBits) // the TOS byte, behind a 14-byte Ethernet header
		return "eth0", frame
	}
}

func arpReply(req *layers.ARP) []byte {
	eth := &layers.Ethernet{SrcMAC: routerMAC(), DstMAC: req.SourceHwAddress, EthernetType: layers.EthernetTypeARP}
	arp := &layers.ARP{
		AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
		HwAddressSize: 6, ProtAddressSize: 4, Operation: layers.ARPReply,
		SourceHwAddress: routerMAC(), SourceProtAddress: routerIP().AsSlice(),
		DstHwAddress: req.SourceHwAddress, DstProtAddress: req.SourceProtAddress,
	}
	buf := gopacket.NewSerializeBuffer()
	_ = gopacket.SerializeLayers(buf, serializeOptions(), eth, arp)
	return buf.Bytes()
}

func preserve(dscp, _ int) int { return dscp }

// On one network the probes go straight to the wired interface's own
// hardware address: no router, no ARP.
func TestSingleHostOnLinkPreserved(t *testing.T) {
	t.Parallel()

	lan := newFakeLAN(accessPoint(wiredMAC(), preserve))
	res, err := testSingleHost(lan, hostPair("10.1.0.20/24")).run(t.Context(),
		SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0", DSCP: []int{46, 0}, Count: 3})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Routed || res.NextHop != "10.1.0.20" || res.Source != "10.1.0.10" || res.Target != "10.1.0.20" {
		t.Fatalf("path = %+v", res)
	}
	if !res.Preserved || res.Probes != 6 || len(res.Classes) != 2 {
		t.Fatalf("result = %+v", res)
	}
	for _, c := range res.Classes {
		if c.Verdict != VerdictPreserved || c.Received != 3 || c.Expected != 3 {
			t.Errorf("class %d = %+v", c.SentDSCP, c)
		}
	}
	if lan.injected["wlan0"] != 6 {
		t.Errorf("injected %d frames on wlan0, want the 6 probes and no ARP", lan.injected["wlan0"])
	}
	want := "udp and src host 10.1.0.10 and dst host 10.1.0.20 and dst port " + strconv.Itoa(res.Port)
	if f := lan.filters["eth0"]; len(f) != 1 || f[0] != want {
		t.Errorf("capture filter = %q, want %q", f, want)
	}
	if n := lan.stillOpen(); n != 0 {
		t.Errorf("%d handles left open", n)
	}
}

// Behind a router the probes are addressed to the router's hardware address,
// learned by ARP, and what the access point does to each marking is read off
// the wired interface.
func TestSingleHostRoutedReadsWhatThePathDid(t *testing.T) {
	t.Parallel()

	remark := func(dscp, seq int) int {
		switch dscp {
		case 46:
			return 0 // EF to best effort, the fault #400 names
		case 10:
			return -1 // AF11 dropped
		case 34:
			if seq%2 == 1 {
				return 32 // AF41 rewritten on every other probe
			}
		}
		return dscp
	}
	lan := newFakeLAN(accessPoint(routerMAC(), remark))
	res, err := testSingleHost(lan, hostPair("10.2.0.20/24")).run(t.Context(),
		SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0", DSCP: []int{46, 34, 10, 0}, Count: 4})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.Routed || res.NextHop != routerIP().String() {
		t.Fatalf("path = %+v", res)
	}
	want := map[int]Verdict{46: VerdictRemarked, 34: VerdictMixed, 10: VerdictLost, 0: VerdictPreserved}
	if got := verdicts(RunResult{Classes: res.Classes}); !maps.Equal(got, want) {
		t.Fatalf("verdicts = %v, want %v", got, want)
	}
	if res.Preserved || res.Probes != 12 {
		t.Fatalf("preserved=%v probes=%d, want false and 12", res.Preserved, res.Probes)
	}
	if n := lan.stillOpen(); n != 0 {
		t.Errorf("%d handles left open", n)
	}
}

// A router that misses the first ARP request is asked again.
func TestSingleHostRetriesARP(t *testing.T) {
	t.Parallel()

	answer := accessPoint(routerMAC(), preserve)
	asked := 0
	lan := newFakeLAN(func(iface string, frame []byte) (string, []byte) {
		if frame[12] == 0x08 && frame[13] == 0x06 { // EtherType ARP
			if asked++; asked == 1 {
				return "", nil
			}
		}
		return answer(iface, frame)
	})
	h := testSingleHost(lan, hostPair("10.2.0.20/24"))
	h.arpTimeout = 2 * arpRetry
	res, err := h.run(t.Context(), SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0", DSCP: []int{0}})
	if err != nil || !res.Preserved || asked < 2 {
		t.Fatalf("run = %+v, %v after %d ARP requests", res, err, asked)
	}
}

// A path that delivers nothing still answers for every class it was asked
// about, rather than returning no classes and reading as a pass.
func TestSingleHostNothingArrives(t *testing.T) {
	t.Parallel()

	lan := newFakeLAN(accessPoint(routerMAC(), func(int, int) int { return -1 }))
	res, err := testSingleHost(lan, hostPair("10.2.0.20/24")).run(t.Context(),
		SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0", DSCP: []int{46, 0}})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if want := (map[int]Verdict{46: VerdictLost, 0: VerdictLost}); !maps.Equal(
		verdicts(RunResult{Classes: res.Classes}),
		want,
	) ||
		res.Preserved {
		t.Fatalf("result = %+v", res)
	}
}

// Cancelling mid-burst returns what was sent and heard so far.
func TestSingleHostCancelKeepsWhatArrived(t *testing.T) {
	t.Parallel()

	lan := newFakeLAN(accessPoint(wiredMAC(), preserve))
	h := testSingleHost(lan, hostPair("10.1.0.20/24"))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sent := 0
	h.pace = func(ctx context.Context) error {
		if sent++; sent == 3 {
			cancel()
		}
		return ctx.Err()
	}
	res, err := h.run(
		ctx,
		SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0", DSCP: []int{46, 0}, Count: 5},
	)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Probes > 3 || lan.injected["wlan0"] != 3 {
		t.Fatalf("probes=%d injected=%d, want at most the 3 sent", res.Probes, lan.injected["wlan0"])
	}
	if n := lan.stillOpen(); n != 0 {
		t.Errorf("%d handles left open", n)
	}
}

func TestSingleHostRefuses(t *testing.T) {
	t.Parallel()

	onLink, routed := hostPair("10.1.0.20/24"), hostPair("10.2.0.20/24")
	pair := SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0"}
	openErr := errors.New("permission denied")
	cases := map[string]struct {
		req    SingleHostRequest
		ifaces map[string]hostInterface
		setup  func(*fakeLAN, *singleHost)
		want   error
	}{
		"same interface": {
			req:    SingleHostRequest{SendInterface: "eth0", CaptureInterface: "eth0"},
			ifaces: onLink,
			want:   ErrInterfaces,
		},
		"no send":    {req: SingleHostRequest{CaptureInterface: "eth0"}, ifaces: onLink, want: ErrInterfaces},
		"no capture": {req: SingleHostRequest{SendInterface: "wlan0"}, ifaces: onLink, want: ErrInterfaces},
		"dscp high": {
			req:    SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0", DSCP: []int{64}},
			ifaces: onLink,
			want:   ErrDSCP,
		},
		"count high": {
			req:    SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth0", Count: MaxCount + 1},
			ifaces: onLink,
			want:   ErrCount,
		},
		"unknown capture": {req: SingleHostRequest{SendInterface: "wlan0", CaptureInterface: "eth9"}, ifaces: onLink},
		"no route": {req: pair, ifaces: routed, setup: func(_ *fakeLAN, h *singleHost) {
			h.routes = func() ([]gateway.RouteInfo, error) { return nil, nil }
		}, want: ErrNoRoute},
		"router silent": {req: pair, ifaces: routed, setup: func(l *fakeLAN, _ *singleHost) {
			l.forward = func(string, []byte) (string, []byte) { return "", nil }
		}, want: ErrNextHop},
		"not ethernet": {req: pair, ifaces: onLink, setup: func(l *fakeLAN, _ *singleHost) {
			l.linkType["eth0"] = layers.LinkTypeIEEE80211Radio
		}, want: ErrLinkType},
		"capture refused": {req: pair, ifaces: onLink, setup: func(l *fakeLAN, _ *singleHost) {
			l.openErr = openErr
		}, want: openErr},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			lan := newFakeLAN(accessPoint(wiredMAC(), preserve))
			h := testSingleHost(lan, tc.ifaces)
			if tc.setup != nil {
				tc.setup(lan, &h)
			}
			res, err := h.run(t.Context(), tc.req)
			if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
				t.Fatalf("run = %+v, %v; want %v", res, err, tc.want)
			}
			if lan.injected["wlan0"] > 0 && !errors.Is(err, ErrNextHop) {
				t.Errorf("injected %d frames before refusing", lan.injected["wlan0"])
			}
			if n := lan.stillOpen(); n != 0 {
				t.Errorf("%d handles left open", n)
			}
		})
	}
}

func TestNextHop(t *testing.T) {
	t.Parallel()

	wifi := hostInterface{name: "wlan0", mac: wifiMAC(), prefix: netip.MustParsePrefix("10.1.0.10/24")}
	route := func(dst string, prefix int, gw, iface string) gateway.RouteInfo {
		return gateway.RouteInfo{Destination: dst, Prefix: prefix, Gateway: gw, Interface: iface, Family: "inet"}
	}
	cases := map[string]struct {
		routes     []gateway.RouteInfo
		target     string
		next       string
		routed     bool
		wantNoPath bool
	}{
		"own network": {target: "10.1.0.20", next: "10.1.0.20"},
		"default router": {
			routes: []gateway.RouteInfo{route("0.0.0.0", 0, "10.1.0.1", "wlan0")},
			target: "10.2.0.20",
			next:   "10.1.0.1",
			routed: true,
		},
		"most specific wins": {routes: []gateway.RouteInfo{
			route("10.2.0.0", 16, "10.1.0.2", "wlan0"),
			route("0.0.0.0", 0, "10.1.0.1", "wlan0"),
			route("10.2.0.0", 24, "10.1.0.3", "wlan0"),
		}, target: "10.2.0.20", next: "10.1.0.3", routed: true},
		"on-link route": {
			routes: []gateway.RouteInfo{route("10.2.0.0", 24, "", "wlan0"), route("0.0.0.0", 0, "10.1.0.1", "wlan0")},
			target: "10.2.0.20",
			next:   "10.2.0.20",
		},
		"macOS link gateway": {
			routes: []gateway.RouteInfo{route("10.2.0.0", 24, "link#5", "wlan0")},
			target: "10.2.0.20",
			next:   "10.2.0.20",
		},
		"other interface": {
			routes:     []gateway.RouteInfo{route("0.0.0.0", 0, "10.2.0.1", "eth0")},
			target:     "10.2.0.20",
			wantNoPath: true,
		},
		"ipv6 route ignored": {
			routes: []gateway.RouteInfo{
				{Destination: "::", Gateway: "fe80::1", Interface: "wlan0", Family: "inet6"},
			},
			target:     "10.2.0.20",
			wantNoPath: true,
		},
		"route misses target": {
			routes:     []gateway.RouteInfo{route("10.3.0.0", 16, "10.1.0.1", "wlan0")},
			target:     "10.2.0.20",
			wantNoPath: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := singleHost{routes: func() ([]gateway.RouteInfo, error) { return tc.routes, nil }}
			next, routed, err := h.nextHop(wifi, netip.MustParseAddr(tc.target))
			if tc.wantNoPath {
				if !errors.Is(err, ErrNoRoute) {
					t.Fatalf("nextHop = %s, %v, %v; want ErrNoRoute", next, routed, err)
				}
				return
			}
			if err != nil || next.String() != tc.next || routed != tc.routed {
				t.Fatalf("nextHop = %s, %v, %v; want %s, %v", next, routed, err, tc.next, tc.routed)
			}
		})
	}
}

func TestFrameRoundTrip(t *testing.T) {
	t.Parallel()

	path := framePath{
		srcMAC: wifiMAC(), dstMAC: routerMAC(),
		src: netip.MustParseAddr("10.1.0.10"), dst: netip.MustParseAddr("10.2.0.20"), port: 50000,
	}
	p := probe{run: 7, mask: 1<<46 | 1, dscp: 46, count: 2, seq: 1}
	frame, err := path.frame(46, p.encode())
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	pkt := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)
	ip, _ := pkt.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	udp, _ := pkt.Layer(layers.LayerTypeUDP).(*layers.UDP)
	eth, _ := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if ip == nil || udp == nil || eth == nil {
		t.Fatalf("frame does not decode: %v", pkt)
	}
	if ip.TOS != 46<<ecnBits || udp.DstPort != 50000 || !bytes.Equal(eth.DstMAC, routerMAC()) ||
		!bytes.Equal(eth.SrcMAC, wifiMAC()) {
		t.Fatalf("headers: tos=%d dport=%d eth=%s->%s", ip.TOS, udp.DstPort, eth.SrcMAC, eth.DstMAC)
	}

	from, dscp, payload, ok := decodeFrame(frame)
	if !ok || from != path.src || dscp != 46 {
		t.Fatalf("decodeFrame = %s, %d, ok=%v", from, dscp, ok)
	}
	if got, decoded := decodeProbe(payload); !decoded || got != p {
		t.Fatalf("probe = %+v, %v; want %+v", got, decoded, p)
	}
}

func TestARPExchange(t *testing.T) {
	t.Parallel()

	arp, err := arpRequest(hostInterface{mac: wifiMAC(), prefix: netip.MustParsePrefix("10.1.0.10/24")}, routerIP())
	if err != nil {
		t.Fatalf("arpRequest: %v", err)
	}
	if _, _, _, isDatagram := decodeFrame(arp); isDatagram {
		t.Fatal("an ARP frame decoded as a probe datagram")
	}
	if _, ok := arpReplyFrom(arp, routerIP()); ok {
		t.Fatal("a request read as the router's reply")
	}
	req, _ := gopacket.NewPacket(arp, layers.LayerTypeEthernet, gopacket.Default).Layer(layers.LayerTypeARP).(*layers.ARP)
	if mac, ok := arpReplyFrom(arpReply(req), routerIP()); !ok || !bytes.Equal(mac, routerMAC()) {
		t.Fatalf("reply = %s, %v", mac, ok)
	}
	if _, ok := arpReplyFrom(arpReply(req), netip.MustParseAddr("10.1.0.9")); ok {
		t.Fatal("another address's reply was accepted")
	}
}

func TestFrameSourceEndsOnInterrupt(t *testing.T) {
	t.Parallel()

	lan := newFakeLAN(nil)
	h, _ := lan.OpenLive("eth0", frameSnaplen, false, readTimeout)
	src := &frameSource{handle: h}
	src.setDeadline(time.Now().Add(20 * time.Millisecond))
	if _, _, _, err := src.read(make([]byte, readBufferSize)); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("read after the deadline = %v, want ErrDeadlineExceeded", err)
	}
	src.interrupt() // a second close is a no-op
}
