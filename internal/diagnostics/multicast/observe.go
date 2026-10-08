package multicast

// observe.go is #399's passive half: watch the IGMP and MLD on a segment for a
// bounded window and report which groups have members, who reported them and
// who is querying. It joins nothing and sends nothing. A listen says whether
// a stream arrives; an observation says whether anyone on the segment asked
// for it, and whether a querier is there to keep snooping switches forwarding.
//
// Reports and leaves are addressed to the group or to the all-routers groups,
// so the capture is promiscuous. An IGMP- or MLD-snooping switch forwards
// them only towards its router ports: on a snooped segment the probe sees
// other hosts' reports from a mirror port, or from the querier's side.

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

const (
	// DefaultObserveWindow covers one IGMPv2/v3 and MLD general-query cycle:
	// the 125 s query interval plus the 10 s maximum response time. A
	// member that joined before the window reports only when it is queried.
	DefaultObserveWindow = 135 * time.Second

	// MaxObserveWindow bounds an observation.
	MaxObserveWindow = 10 * time.Minute

	// observeFilter keeps IGMP and MLD, tagged or not. MLD always follows a
	// hop-by-hop header carrying the router alert (RFC 2710 §3, RFC 3810
	// §5), which libpcap's icmp6 primitive does not look past, so the filter
	// matches ICMPv6 behind that header.
	observeFilter = "igmp or (ip6 and ip6[6] == 0 and ip6[40] == 58) or " +
		"(vlan and (igmp or (ip6 and ip6[6] == 0 and ip6[40] == 58)))"

	// observeSnaplen holds an IGMPv3 or MLDv2 report of a few hundred group
	// records; a longer one is decoded as far as it was captured.
	observeSnaplen = 4096

	// observeReadTimeout bounds each read so a quiet segment still sees the
	// stop (see capture.ErrTimeout).
	observeReadTimeout = 100 * time.Millisecond

	// observeProgressInterval throttles progress reports.
	observeProgressInterval = time.Second

	// maxObservedGroups, maxObservedReporters (per group) and
	// maxObservedQueriers bound what a flood of forged reports can grow.
	maxObservedGroups    = 512
	maxObservedReporters = 256
	maxObservedQueriers  = 64
)

// The protocol versions a report or query is attributed to.
const (
	igmpV1 = 1
	igmpV2 = 2
	igmpV3 = 3
	mldV1  = 1
	mldV2  = 2
)

// ObserveRequest is one observation, as a caller asks for it.
type ObserveRequest struct {
	Interface string `json:"interface"`
	// DurationSeconds is how long to observe, at most ten minutes; one
	// query cycle (135 s) when zero.
	DurationSeconds int `json:"durationSeconds,omitempty"`
}

// ObserveResult is what an observation saw.
type ObserveResult struct {
	Interface  string `json:"interface"`
	ObservedMs int64  `json:"observedMs"`
	// Messages counts the IGMP and MLD messages decoded.
	Messages uint64 `json:"messages"`
	// Groups are the groups reported, IPv4 then IPv6, in address order.
	Groups []ObservedGroup `json:"groups"`
	// Queriers are the hosts that sent a query, in address order.
	Queriers []ObservedQuerier `json:"queriers"`
	// Truncated reports that groups, reporters or queriers beyond the
	// bounds were seen and are not named.
	Truncated bool `json:"truncated"`
}

// ObservedGroup is one group and the hosts that reported it.
type ObservedGroup struct {
	Group     string             `json:"group"`
	Reporters []ObservedReporter `json:"reporters"`
}

// ObservedReporter is one host's reports for a group.
type ObservedReporter struct {
	Address string `json:"address"`
	// Version is the IGMP (1-3) or MLD (1-2) version the host reported in.
	Version int    `json:"version"`
	Reports uint64 `json:"reports"`
	// Left is true when the host's last message for the group was a leave
	// (IGMPv2 Leave, MLDv1 Done, or a v3/v2 record leaving the group).
	Left bool `json:"left"`
}

// ObservedQuerier is one host that sent queries.
type ObservedQuerier struct {
	Address string `json:"address"`
	Version int    `json:"version"`
	Queries uint64 `json:"queries"`
	// Elected marks the lowest address of its family: the querier the
	// election leaves in charge when several query (RFC 2236 §3, RFC 3376
	// §6.6.2, RFC 3810 §7.6.2). A snooping switch's proxy query from the
	// unspecified address takes no part in the election (RFC 4541 §2.1.1).
	Elected bool `json:"elected"`
}

// Observe captures IGMP and MLD on req.Interface through opener until the
// window closes or ctx ends. Ending early is not a failure: what was seen is
// the answer the operator stopped for. report receives the fraction of the
// window elapsed.
func Observe(
	ctx context.Context,
	opener capture.Opener,
	req ObserveRequest,
	report func(float64),
) (*ObserveResult, error) {
	if req.Interface == "" {
		return nil, ErrInterfaceRequired
	}
	window := observeWindow(req.DurationSeconds)
	handle, err := opener.OpenLive(req.Interface, observeSnaplen, true, observeReadTimeout)
	if err != nil {
		return nil, fmt.Errorf("open %s for capture: %w", req.Interface, err)
	}
	defer handle.Close()
	if filterErr := handle.SetBPFFilter(observeFilter); filterErr != nil {
		return nil, fmt.Errorf("set IGMP/MLD filter on %s: %w", req.Interface, filterErr)
	}

	obs := newObservation()
	started := time.Now()
	lastReport := started
	for ctx.Err() == nil {
		now := time.Now()
		elapsed := now.Sub(started)
		if elapsed >= window {
			break
		}
		if now.Sub(lastReport) >= observeProgressInterval {
			report(float64(elapsed) / float64(window))
			lastReport = now
		}
		data, _, readErr := handle.ReadPacketData()
		if errors.Is(readErr, capture.ErrTimeout) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("read frame: %w", readErr)
		}
		obs.decode(gopacket.NewPacket(data, handle.LinkType(), gopacket.DecodeOptions{Lazy: true, NoCopy: true}))
	}

	res := obs.result()
	res.Interface = req.Interface
	res.ObservedMs = time.Since(started).Milliseconds()
	return res, nil
}

// observeWindow is the requested window, one query cycle when none is asked
// for, and at most MaxObserveWindow.
func observeWindow(seconds int) time.Duration {
	if seconds <= 0 {
		return DefaultObserveWindow
	}
	return min(time.Duration(seconds)*time.Second, MaxObserveWindow)
}

// observation accumulates what the decoded messages say.
type observation struct {
	messages  uint64
	groups    map[netip.Addr]map[netip.Addr]*ObservedReporter
	queriers  map[netip.Addr]*ObservedQuerier
	truncated bool
}

func newObservation() *observation {
	return &observation{
		groups:   make(map[netip.Addr]map[netip.Addr]*ObservedReporter),
		queriers: make(map[netip.Addr]*ObservedQuerier),
	}
}

// decode records the one IGMP or MLD message a packet carries, if any.
func (o *observation) decode(p gopacket.Packet) {
	src, ok := senderAddress(p)
	if !ok {
		return
	}
	for _, l := range p.Layers() {
		switch m := l.(type) {
		case *layers.IGMPv1or2:
			o.messages++
			o.igmp(src, m.Type, int(m.Version), m.GroupAddress, nil)
			return
		case *layers.IGMP:
			o.messages++
			o.igmp(src, m.Type, igmpV3, nil, m.GroupRecords)
			return
		case *layers.MLDv1MulticastListenerQueryMessage:
			o.messages++
			o.query(src, mldV1)
			return
		case *layers.MLDv2MulticastListenerQueryMessage:
			o.messages++
			o.query(src, mldV2)
			return
		case *layers.MLDv1MulticastListenerReportMessage:
			o.messages++
			o.membership(m.MulticastAddress, src, mldV1, false)
			return
		case *layers.MLDv1MulticastListenerDoneMessage:
			o.messages++
			o.membership(m.MulticastAddress, src, mldV1, true)
			return
		case *layers.MLDv2MulticastListenerReportMessage:
			o.messages++
			for _, r := range m.MulticastAddressRecords {
				o.membership(r.MulticastAddress, src, mldV2, mldv2Leaves(r))
			}
			return
		}
	}
}

// igmp records an IGMP message: v1/v2 messages name one group, a v3 report
// carries a record per group.
func (o *observation) igmp(
	src netip.Addr,
	typ layers.IGMPType,
	version int,
	group net.IP,
	records []layers.IGMPv3GroupRecord,
) {
	switch typ {
	case layers.IGMPMembershipQuery:
		o.query(src, version)
	case layers.IGMPMembershipReportV1:
		o.membership(group, src, igmpV1, false)
	case layers.IGMPMembershipReportV2:
		o.membership(group, src, igmpV2, false)
	case layers.IGMPLeaveGroup:
		o.membership(group, src, igmpV2, true)
	case layers.IGMPMembershipReportV3:
		for _, r := range records {
			o.membership(r.MulticastAddress, src, igmpV3, igmpV3Leaves(r))
		}
	}
}

// igmpV3Leaves reports whether a group record leaves the group: an include
// list that is, or becomes, empty (RFC 3376 §5.1).
func igmpV3Leaves(r layers.IGMPv3GroupRecord) bool {
	return (r.Type == layers.IGMPIsIn || r.Type == layers.IGMPToIn) && len(r.SourceAddresses) == 0
}

// mldv2Leaves is igmpV3Leaves for MLDv2 (RFC 3810 §6.1).
func mldv2Leaves(r layers.MLDv2MulticastAddressRecord) bool {
	return (r.RecordType == layers.MLDv2MulticastAddressRecordTypeModeIsIncluded ||
		r.RecordType == layers.MLDv2MulticastAddressRecordTypeChangeToIncludeMode) &&
		len(r.SourceAddresses) == 0
}

func (o *observation) membership(groupIP net.IP, src netip.Addr, version int, leaves bool) {
	group, ok := netip.AddrFromSlice(groupIP)
	if !ok || !group.Unmap().IsMulticast() {
		return
	}
	group = group.Unmap()
	reporters, ok := o.groups[group]
	if !ok {
		if len(o.groups) >= maxObservedGroups {
			o.truncated = true
			return
		}
		reporters = make(map[netip.Addr]*ObservedReporter)
		o.groups[group] = reporters
	}
	r, ok := reporters[src]
	if !ok {
		if len(reporters) >= maxObservedReporters {
			o.truncated = true
			return
		}
		r = &ObservedReporter{Address: src.String()}
		reporters[src] = r
	}
	r.Version = version
	r.Reports++
	r.Left = leaves
}

func (o *observation) query(src netip.Addr, version int) {
	q, ok := o.queriers[src]
	if !ok {
		if len(o.queriers) >= maxObservedQueriers {
			o.truncated = true
			return
		}
		q = &ObservedQuerier{Address: src.String()}
		o.queriers[src] = q
	}
	q.Version = version
	q.Queries++
}

func (o *observation) result() *ObserveResult {
	res := &ObserveResult{
		Messages:  o.messages,
		Groups:    make([]ObservedGroup, 0, len(o.groups)),
		Queriers:  make([]ObservedQuerier, 0, len(o.queriers)),
		Truncated: o.truncated,
	}
	for _, group := range slices.SortedFunc(maps.Keys(o.groups), netip.Addr.Compare) {
		reporters := o.groups[group]
		g := ObservedGroup{Group: group.String(), Reporters: make([]ObservedReporter, 0, len(reporters))}
		for _, addr := range slices.SortedFunc(maps.Keys(reporters), netip.Addr.Compare) {
			g.Reporters = append(g.Reporters, *reporters[addr])
		}
		res.Groups = append(res.Groups, g)
	}
	var lowest4, lowest6 netip.Addr
	for _, addr := range slices.SortedFunc(maps.Keys(o.queriers), netip.Addr.Compare) {
		q := *o.queriers[addr]
		switch {
		case addr.IsUnspecified():
		case addr.Is4() && !lowest4.IsValid():
			lowest4, q.Elected = addr, true
		case addr.Is6() && !lowest6.IsValid():
			lowest6, q.Elected = addr, true
		}
		res.Queriers = append(res.Queriers, q)
	}
	return res
}

// senderAddress is the sender's address on the packet's network layer.
func senderAddress(p gopacket.Packet) (netip.Addr, bool) {
	switch n := p.NetworkLayer().(type) {
	case *layers.IPv4:
		a, ok := netip.AddrFromSlice(n.SrcIP)
		return a.Unmap(), ok
	case *layers.IPv6:
		a, ok := netip.AddrFromSlice(n.SrcIP)
		return a, ok
	}
	return netip.Addr{}, false
}
