package qos

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
)

const (
	// settleTime is how long a single-host check keeps capturing after its
	// last probe left: long enough for an access point's uplink queue to
	// drain, short enough that a lost class does not hold the job.
	settleTime = time.Second

	// arpTimeout bounds the next-hop lookup when the capture interface sits
	// behind a router; arpRetry repeats the request inside it.
	arpTimeout = 2 * time.Second
	arpRetry   = 250 * time.Millisecond

	// readTimeout is how long a capture read waits before returning
	// capture.ErrTimeout, which bounds how long closing a quiet handle takes.
	readTimeout = 100 * time.Millisecond

	// ephemeralPortBase and ephemeralPorts are the IANA dynamic range a
	// check draws its port from.
	ephemeralPortBase = 49152
	ephemeralPorts    = 16384
)

var (
	// ErrInterfaces rejects a single-host check that does not name two
	// different interfaces.
	ErrInterfaces = errors.New("sendInterface and captureInterface must name two different interfaces")
	// ErrNoIPv4 rejects an interface with no routable IPv4 address.
	ErrNoIPv4 = errors.New("interface has no IPv4 address")
	// ErrLinkType rejects an interface that does not frame Ethernet, which
	// is how both wired and Wi-Fi interfaces present to capture.
	ErrLinkType = errors.New("interface is not an Ethernet or Wi-Fi interface")
	// ErrNoRoute means the send interface has no route to the capture
	// interface's address.
	ErrNoRoute = errors.New("no route to the capture interface over the send interface")
	// ErrNextHop means the router on the send interface did not answer ARP.
	ErrNextHop = errors.New("next hop did not answer ARP")
)

// SingleHostRequest is one single-host check, as a caller asks for it: the
// probes leave SendInterface and are read as they arrive on
// CaptureInterface, so one host with a Wi-Fi and a wired interface checks
// what its access point and the wired path do to each marking.
type SingleHostRequest struct {
	SendInterface    string `json:"sendInterface"`
	CaptureInterface string `json:"captureInterface"`
	// DSCP lists the classes to send, each 0-63; the defaults when empty.
	DSCP []int `json:"dscp,omitempty"`
	// Count is the probes per class; DefaultCount when zero.
	Count int `json:"count,omitempty"`
}

// SingleHostResult is what reached the capture interface.
type SingleHostResult struct {
	SendInterface    string `json:"sendInterface"`
	CaptureInterface string `json:"captureInterface"`
	// Source and Target are the two interfaces' IPv4 addresses.
	Source string `json:"source"`
	Target string `json:"target"`
	// NextHop is the router the probes were handed to, or Target when the
	// two interfaces share a network; Routed says which.
	NextHop string `json:"nextHop"`
	Routed  bool   `json:"routed"`
	Port    int    `json:"port"`
	RunID   string `json:"runId"`
	Count   int    `json:"count"`
	// Probes counts the distinct probes that arrived.
	Probes  uint64        `json:"probes"`
	Classes []ClassResult `json:"classes"`
	// Preserved is true when every class arrived with the marking it left
	// with.
	Preserved bool `json:"preserved"`
}

// SingleHost runs a single-host check through opener.
//
// A host cannot put a datagram addressed to one of its own interfaces on the
// wire through a socket: the kernel delivers it over loopback. So the probes
// are built as Ethernet frames and injected on the send interface, and read
// off the capture interface by the same capture port, before the receiving
// kernel (which may drop a packet from one of its own addresses as a martian)
// sees them. IPv4 only.
func SingleHost(ctx context.Context, opener capture.Opener, req SingleHostRequest) (*SingleHostResult, error) {
	h := singleHost{
		opener:     opener,
		lookup:     lookupInterface,
		routes:     gateway.GetAllRoutes,
		pace:       pace,
		settle:     settleTime,
		arpTimeout: arpTimeout,
	}
	return h.run(ctx, req)
}

// hostInterface is what a single-host check needs of one interface.
type hostInterface struct {
	name   string
	mac    net.HardwareAddr
	prefix netip.Prefix
}

// singleHost is a check's dependencies, behind seams so the orchestration is
// testable without two interfaces and a network between them.
type singleHost struct {
	opener     capture.Opener
	lookup     func(name string) (hostInterface, error)
	routes     func() ([]gateway.RouteInfo, error)
	pace       func(context.Context) error
	settle     time.Duration
	arpTimeout time.Duration
}

func (h singleHost) run(ctx context.Context, req SingleHostRequest) (*SingleHostResult, error) {
	if req.SendInterface == "" || req.CaptureInterface == "" || req.SendInterface == req.CaptureInterface {
		return nil, ErrInterfaces
	}
	mask, count, err := parseClasses(req.DSCP, req.Count)
	if err != nil {
		return nil, err
	}
	from, err := h.lookup(req.SendInterface)
	if err != nil {
		return nil, err
	}
	to, err := h.lookup(req.CaptureInterface)
	if err != nil {
		return nil, err
	}
	target := to.prefix.Addr()
	next, routed, err := h.nextHop(from, target)
	if err != nil {
		return nil, err
	}
	dstMAC := to.mac
	if routed {
		if dstMAC, err = h.resolveMAC(ctx, from, next); err != nil {
			return nil, err
		}
	}

	var random [10]byte
	_, _ = rand.Read(random[:])
	run := binary.BigEndian.Uint64(random[:8])
	port := uint16(ephemeralPortBase + int(binary.BigEndian.Uint16(random[8:]))%ephemeralPorts)
	path := framePath{srcMAC: from.mac, dstMAC: dstMAC, src: from.prefix.Addr(), dst: target, port: port}
	spec := sendSpec{target: netip.AddrPortFrom(target, port), mask: mask, count: count}

	heard, err := h.exchange(ctx, path, from.name, to.name, spec, run)
	if err != nil {
		return nil, err
	}
	res := &SingleHostResult{
		SendInterface:    from.name,
		CaptureInterface: to.name,
		Source:           path.src.String(),
		Target:           target.String(),
		NextHop:          next.String(),
		Routed:           routed,
		Port:             int(port),
		RunID:            fmt.Sprintf("%016x", run),
		Count:            int(count),
		Probes:           heard.Probes,
		Preserved:        true,
	}
	for _, r := range heard.Runs {
		if r.RunID == res.RunID {
			res.Classes = r.Classes
		}
	}
	if res.Classes == nil {
		// Nothing of the run arrived: every class it sent was lost.
		nothing := newRunTally(probe{mask: mask, count: count})
		for _, d := range spec.classes() {
			res.Classes = append(res.Classes, classResult(d, nothing))
		}
	}
	for _, c := range res.Classes {
		res.Preserved = res.Preserved && c.Verdict == VerdictPreserved
	}
	return res, nil
}

// exchange opens the capture before the first probe leaves, injects the
// burst, and captures for the settle time after the last one.
func (h singleHost) exchange(
	ctx context.Context,
	path framePath,
	sendIface, captureIface string,
	spec sendSpec,
	run uint64,
) (*ListenResult, error) {
	listen, err := h.openEthernet(captureIface)
	if err != nil {
		return nil, err
	}
	filter := fmt.Sprintf("udp and src host %s and dst host %s and dst port %d", path.src, path.dst, path.port)
	if err = listen.SetBPFFilter(filter); err != nil {
		listen.Close()
		return nil, fmt.Errorf("capture filter on %s: %w", captureIface, err)
	}
	src := &frameSource{handle: listen}
	defer src.interrupt()

	inject, err := h.openEthernet(sendIface)
	if err != nil {
		return nil, err
	}
	defer inject.Close()

	collectCtx, stopCollect := context.WithCancel(ctx)
	defer stopCollect()
	type collected struct {
		res *ListenResult
		err error
	}
	done := make(chan collected, 1)
	go func() {
		res, collectErr := collect(collectCtx, src, listenSpec{port: int(path.port), window: MaxWindow}, time.Now)
		done <- collected{res, collectErr}
	}()

	if _, err = send(ctx, &frameConn{handle: inject, path: path}, spec, run, h.pace); err != nil {
		return nil, err
	}
	settle := time.NewTimer(h.settle)
	select {
	case <-ctx.Done():
	case <-settle.C:
	}
	settle.Stop()
	stopCollect()
	c := <-done
	return c.res, c.err
}

// openEthernet opens iface for capture and injection and refuses one that
// does not frame Ethernet.
func (h singleHost) openEthernet(iface string) (capture.Handle, error) {
	handle, err := h.opener.OpenLive(iface, frameSnaplen, false, readTimeout)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", iface, err)
	}
	if lt := handle.LinkType(); lt != layers.LinkTypeEthernet {
		handle.Close()
		return nil, fmt.Errorf("%w: %s frames %s", ErrLinkType, iface, lt)
	}
	return handle, nil
}

// nextHop decides where the send interface hands a probe for target: to
// target itself when it is on the send interface's network, otherwise to the
// router of the most specific route out of the send interface.
func (h singleHost) nextHop(from hostInterface, target netip.Addr) (netip.Addr, bool, error) {
	if from.prefix.Contains(target) {
		return target, false, nil
	}
	routes, err := h.routes()
	if err != nil {
		return netip.Addr{}, false, fmt.Errorf("read routes: %w", err)
	}
	best := -1
	var next netip.Addr
	for _, r := range routes {
		if r.Interface != from.name || r.Family != "inet" || r.Prefix <= best {
			continue
		}
		dst, dstErr := netip.ParseAddr(r.Destination)
		if dstErr != nil || !netip.PrefixFrom(dst, r.Prefix).Contains(target) {
			continue
		}
		// An on-link route names no router: its gateway is empty, or a link
		// reference rather than an address on macOS.
		gw, gwErr := netip.ParseAddr(r.Gateway)
		if gwErr != nil || gw.IsUnspecified() {
			gw = target
		}
		best, next = r.Prefix, gw
	}
	if best < 0 {
		return netip.Addr{}, false, fmt.Errorf("%w: %s via %s", ErrNoRoute, target, from.name)
	}
	return next, next != target, nil
}

// resolveMAC asks the router on the send interface for its hardware address,
// fresh rather than from the host's cache, which may not hold it.
func (h singleHost) resolveMAC(ctx context.Context, from hostInterface, router netip.Addr) (net.HardwareAddr, error) {
	handle, err := h.openEthernet(from.name)
	if err != nil {
		return nil, err
	}
	if err = handle.SetBPFFilter("arp"); err != nil {
		handle.Close()
		return nil, fmt.Errorf("ARP filter on %s: %w", from.name, err)
	}
	request, err := arpRequest(from, router)
	if err != nil {
		handle.Close()
		return nil, err
	}

	answered := make(chan net.HardwareAddr, 1)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			data, _, readErr := handle.ReadPacketData()
			if errors.Is(readErr, capture.ErrTimeout) {
				continue
			}
			if readErr != nil {
				return
			}
			if mac, ok := arpReplyFrom(data, router); ok {
				answered <- mac
				return
			}
		}
	}()
	defer func() {
		handle.Close()
		<-readerDone
	}()

	deadline := time.NewTimer(h.arpTimeout)
	defer deadline.Stop()
	retry := time.NewTicker(arpRetry)
	defer retry.Stop()
	for {
		if err = handle.WritePacketData(request); err != nil {
			return nil, fmt.Errorf("send ARP request on %s: %w", from.name, err)
		}
		select {
		case mac := <-answered:
			return mac, nil
		case <-retry.C:
		case <-deadline.C:
			return nil, fmt.Errorf("%w: %s on %s", ErrNextHop, router, from.name)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// lookupInterface reads name's hardware address and first routable IPv4
// address from the host.
func lookupInterface(name string) (hostInterface, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return hostInterface{}, fmt.Errorf("interface %s: %w", name, err)
	}
	if len(ifi.HardwareAddr) != ethernetAddrLen {
		return hostInterface{}, fmt.Errorf("%w: %s", ErrLinkType, name)
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return hostInterface{}, fmt.Errorf("%s addresses: %w", name, err)
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		addr, ok := netip.AddrFromSlice(ipnet.IP)
		if !ok {
			continue
		}
		addr = addr.Unmap()
		if !addr.Is4() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
			continue
		}
		ones, _ := ipnet.Mask.Size()
		return hostInterface{name: name, mac: ifi.HardwareAddr, prefix: netip.PrefixFrom(addr, ones)}, nil
	}
	return hostInterface{}, fmt.Errorf("%w: %s", ErrNoIPv4, name)
}

// frameConn injects each probe as a frame marked through its IPv4 header.
type frameConn struct {
	handle capture.Handle
	path   framePath
	dscp   uint8
}

func (c *frameConn) setDSCP(dscp uint8) error {
	c.dscp = dscp
	return nil
}

func (c *frameConn) write(b []byte) error {
	f, err := c.path.frame(c.dscp, b)
	if err != nil {
		return err
	}
	return c.handle.WritePacketData(f)
}

// frameSource reads probes off a capture handle for collect. The marking is
// read from the captured IPv4 header, so it is observed on every platform.
// A deadline and an interrupt both close the handle, once; the read timeout
// is what lets a read notice.
type frameSource struct {
	handle capture.Handle
	once   sync.Once
	closed atomic.Bool
}

func (s *frameSource) read(buf []byte) (int, netip.Addr, int, error) {
	for {
		data, _, err := s.handle.ReadPacketData()
		if err != nil {
			if s.closed.Load() {
				return 0, netip.Addr{}, 0, os.ErrDeadlineExceeded
			}
			if errors.Is(err, capture.ErrTimeout) {
				continue
			}
			return 0, netip.Addr{}, 0, err
		}
		if from, dscp, payload, ok := decodeFrame(data); ok {
			return copy(buf, payload), from, dscp, nil
		}
	}
}

func (s *frameSource) setDeadline(t time.Time) { time.AfterFunc(time.Until(t), s.interrupt) }

func (s *frameSource) interrupt() {
	s.once.Do(func() {
		s.closed.Store(true)
		s.handle.Close()
	})
}

func (*frameSource) observesDSCP() bool { return true }
