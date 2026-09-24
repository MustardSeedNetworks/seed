// Package multicast answers the first question of an IPTV or media-streaming
// fault: is the stream arriving on this port at all, and from whom (#399).
//
// A listen joins one group on one named interface for a bounded window and
// counts what arrives. It sends nothing. Joining is what makes an IGMP- or
// MLD-snooping switch forward the group, so a listen that hears nothing
// separates "the sender or the multicast routing is broken" from "the
// receiving application is broken", which is the split an operator cannot
// make from the application alone.
package multicast

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"slices"
	"time"
)

const (
	// DefaultWindow is how long a listen runs when the caller does not say.
	// An MPEG-TS stream sends hundreds of packets a second and a
	// service-announcement group sends one every few seconds; ten seconds
	// catches both.
	DefaultWindow = 10 * time.Second

	// MaxWindow bounds a listen: this is a diagnostic, and a membership held
	// for longer keeps the switch forwarding a group nobody is watching.
	MaxWindow = 60 * time.Second

	// maxSources is how many senders the result names, busiest first.
	maxSources = 64

	// maxTrackedSources bounds the per-sender table so a flood of spoofed
	// source addresses cannot grow it without limit. It is larger than
	// maxSources so the busiest senders are ranked over everything heard,
	// not over whoever happened to arrive first.
	maxTrackedSources = 4096

	// packetBufferSize holds a jumbo-frame datagram.
	packetBufferSize = 9000
)

var (
	// ErrNotMulticast rejects a group that is not a multicast address.
	ErrNotMulticast = errors.New("group is not a multicast address")
	// ErrPort rejects a port outside 1-65535.
	ErrPort = errors.New("port must be between 1 and 65535")
	// ErrInterfaceRequired rejects a listen with no interface. The kernel's
	// choice is the default route's interface, which on a multi-homed probe
	// is the wrong segment as often as not.
	ErrInterfaceRequired = errors.New("interface is required")
	// ErrInterface rejects an interface that is missing, down, or cannot
	// join a group.
	ErrInterface = errors.New("interface cannot join a multicast group")
)

// ListenRequest is one listen, as a caller asks for it.
type ListenRequest struct {
	Group           string `json:"group"`
	Port            int    `json:"port"`
	Interface       string `json:"interface"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
}

// ListenResult is what a listen heard.
type ListenResult struct {
	Group      string `json:"group"`
	Port       int    `json:"port"`
	Interface  string `json:"interface"`
	ListenedMs int64  `json:"listenedMs"`
	Packets    uint64 `json:"packets"`
	Bytes      uint64 `json:"bytes"`
	// PacketsPerSecond is over the time actually listened, so a listen
	// stopped early does not understate the rate.
	PacketsPerSecond float64 `json:"packetsPerSecond"`
	// Sources are the senders heard, busiest first, at most maxSources.
	Sources []ListenSource `json:"sources"`
	// SourcesTruncated reports that more senders were heard than are named.
	SourcesTruncated bool `json:"sourcesTruncated"`
	// GroupFiltered is false where the platform cannot report a datagram's
	// destination (Windows): the counts then cover every datagram that
	// reached the port, whichever group it was sent to.
	GroupFiltered bool `json:"groupFiltered"`
}

// ListenSource is one sender and what it sent to the group.
type ListenSource struct {
	Address string `json:"address"`
	Packets uint64 `json:"packets"`
	Bytes   uint64 `json:"bytes"`
}

// ifaceInfo is the part of an interface a listen depends on.
type ifaceInfo struct {
	up        bool
	multicast bool
}

// spec is a validated ListenRequest.
type spec struct {
	group  netip.Addr
	port   int
	iface  string
	window time.Duration
}

func (s spec) network() string {
	if s.group.Is4() {
		return "udp4"
	}
	return "udp6"
}

// packetSource is the socket, behind a seam so the counting is testable
// without one.
type packetSource interface {
	// read returns one datagram's size, sender and destination. The
	// destination is the zero Addr where the platform does not report it.
	read(buf []byte) (n int, src, dst netip.Addr, err error)
	setDeadline(t time.Time)
	// interrupt ends a blocked read.
	interrupt()
	filtersByDestination() bool
}

// parse validates a request against the interface it names.
func parse(req ListenRequest, lookup func(string) (ifaceInfo, error)) (spec, error) {
	group, err := netip.ParseAddr(req.Group)
	if err != nil || !group.Unmap().IsMulticast() {
		return spec{}, fmt.Errorf("%w: %q", ErrNotMulticast, req.Group)
	}
	if req.Port < 1 || req.Port > 65535 {
		return spec{}, fmt.Errorf("%w: %d", ErrPort, req.Port)
	}
	if req.Interface == "" {
		return spec{}, ErrInterfaceRequired
	}
	info, err := lookup(req.Interface)
	switch {
	case err != nil:
		return spec{}, fmt.Errorf("%w: %s: %w", ErrInterface, req.Interface, err)
	case !info.up:
		return spec{}, fmt.Errorf("%w: %s is down", ErrInterface, req.Interface)
	case !info.multicast:
		return spec{}, fmt.Errorf("%w: %s does not support multicast", ErrInterface, req.Interface)
	}

	window := DefaultWindow
	if req.DurationSeconds > 0 {
		window = min(time.Duration(req.DurationSeconds)*time.Second, MaxWindow)
	}
	// The zone is the interface, which the request already names; a
	// destination read off the wire never carries one, so keeping it would
	// make every datagram look addressed elsewhere.
	return spec{group: group.Unmap().WithZone(""), port: req.Port, iface: req.Interface, window: window}, nil
}

// collect counts what arrives for the group until the window closes or ctx
// ends. Ending early is not a failure: what was heard is the answer the
// operator stopped for.
func collect(ctx context.Context, src packetSource, s spec, now func() time.Time) (*ListenResult, error) {
	started := now()
	src.setDeadline(started.Add(s.window))
	stop := context.AfterFunc(ctx, src.interrupt)
	defer stop()

	res := &ListenResult{
		Group:         s.group.String(),
		Port:          s.port,
		Interface:     s.iface,
		GroupFiltered: src.filtersByDestination(),
	}
	senders := make(map[netip.Addr]*ListenSource)
	buf := make([]byte, packetBufferSize)

	for ctx.Err() == nil {
		n, from, dst, err := src.read(buf)
		if err != nil {
			// The window closing and interrupt both surface as a deadline.
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}
			return nil, err
		}
		if dst.IsValid() && dst.Unmap() != s.group {
			continue
		}
		size := byteCount(n)
		res.Packets++
		res.Bytes += size
		tally(senders, from, size)
	}

	listened := now().Sub(started)
	res.ListenedMs = listened.Milliseconds()
	if listened > 0 {
		res.PacketsPerSecond = float64(res.Packets) / listened.Seconds()
	}
	res.Sources, res.SourcesTruncated = rank(senders)
	return res, nil
}

// byteCount converts a read's size; a read never returns a negative count.
func byteCount(n int) uint64 {
	if n <= 0 {
		return 0
	}
	return uint64(n)
}

func tally(senders map[netip.Addr]*ListenSource, from netip.Addr, size uint64) {
	from = from.Unmap()
	sender, ok := senders[from]
	if !ok {
		if len(senders) >= maxTrackedSources {
			return
		}
		sender = &ListenSource{Address: from.String()}
		senders[from] = sender
	}
	sender.Packets++
	sender.Bytes += size
}

func rank(senders map[netip.Addr]*ListenSource) ([]ListenSource, bool) {
	out := make([]ListenSource, 0, len(senders))
	for _, s := range senders {
		out = append(out, *s)
	}
	slices.SortFunc(out, func(a, b ListenSource) int {
		return cmp.Or(
			cmp.Compare(b.Packets, a.Packets),
			cmp.Compare(b.Bytes, a.Bytes),
			cmp.Compare(a.Address, b.Address),
		)
	})
	if len(out) > maxSources {
		return out[:maxSources], true
	}
	return out, false
}
