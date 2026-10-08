package multicast

import (
	"context"
	"encoding/binary"
	"errors"
	"net/netip"
	"reflect"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/capture/capturetest"
)

// igmpFrame is an Ethernet frame carrying msg in an IPv4 packet with the
// router alert option, as hosts and queriers send IGMP (RFC 2113).
func igmpFrame(src, dst string, msg []byte) []byte {
	ip := make([]byte, 24, 24+len(msg))
	ip[0] = 0x46 // version 4, 6-word header
	binary.BigEndian.PutUint16(ip[2:4], uint16(24+len(msg)))
	ip[8] = 1 // TTL
	ip[9] = byte(layers.IPProtocolIGMP)
	s, d := netip.MustParseAddr(src).As4(), netip.MustParseAddr(dst).As4()
	copy(ip[12:16], s[:])
	copy(ip[16:20], d[:])
	copy(ip[20:24], []byte{0x94, 0x04, 0x00, 0x00}) // router alert
	return ethernet(layers.EthernetTypeIPv4, append(ip, msg...))
}

// mldFrame is an Ethernet frame carrying the ICMPv6 message msg behind the
// hop-by-hop router alert every MLD message carries (RFC 2711).
func mldFrame(src, dst string, msg []byte) []byte {
	ip := make([]byte, 48, 48+len(msg))
	ip[0] = 0x60
	binary.BigEndian.PutUint16(ip[4:6], uint16(8+len(msg)))
	ip[6] = byte(layers.IPProtocolIPv6HopByHop)
	ip[7] = 1
	s, d := netip.MustParseAddr(src).As16(), netip.MustParseAddr(dst).As16()
	copy(ip[8:24], s[:])
	copy(ip[24:40], d[:])
	copy(ip[40:48], []byte{byte(layers.IPProtocolICMPv6), 0, 0x05, 0x02, 0, 0, 0x01, 0x00})
	return ethernet(layers.EthernetTypeIPv6, append(ip, msg...))
}

func ethernet(t layers.EthernetType, payload []byte) []byte {
	f := make([]byte, 14, 14+len(payload))
	copy(f[0:6], []byte{0x01, 0x00, 0x5e, 0x00, 0x00, 0x01})
	copy(f[6:12], []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x01})
	binary.BigEndian.PutUint16(f[12:14], uint16(t))
	return append(f, payload...)
}

func ip4(s string) []byte {
	a := netip.MustParseAddr(s).As4()
	return a[:]
}

func ip6(s string) []byte {
	a := netip.MustParseAddr(s).As16()
	return a[:]
}

// igmpV2Msg is an IGMPv1/v2 message: a query, report or leave for group.
func igmpV2Msg(typ layers.IGMPType, maxResp byte, group string) []byte {
	return append([]byte{byte(typ), maxResp, 0, 0}, ip4(group)...)
}

func igmpV3QueryMsg() []byte {
	return append(append([]byte{0x11, 100, 0, 0}, ip4("0.0.0.0")...), 0x02, 125, 0, 0)
}

type groupRecord struct {
	typ     byte
	group   string
	sources []string
}

func igmpV3ReportMsg(records ...groupRecord) []byte {
	msg := []byte{0x22, 0, 0, 0, 0, 0, 0, byte(len(records))}
	for _, r := range records {
		msg = append(msg, r.typ, 0, 0, byte(len(r.sources)))
		msg = append(msg, ip4(r.group)...)
		for _, s := range r.sources {
			msg = append(msg, ip4(s)...)
		}
	}
	return msg
}

// mldV1Msg is an MLDv1 query, report or done for group.
func mldV1Msg(typ uint8, group string) []byte {
	return append([]byte{typ, 0, 0, 0, 0x27, 0x10, 0, 0}, ip6(group)...)
}

func mldV2QueryMsg() []byte {
	return append(append([]byte{130, 0, 0, 0, 0x27, 0x10, 0, 0}, ip6("::")...), 0x02, 125, 0, 0)
}

func mldV2ReportMsg(records ...groupRecord) []byte {
	msg := []byte{143, 0, 0, 0, 0, 0, 0, byte(len(records))}
	for _, r := range records {
		msg = append(msg, r.typ, 0, 0, byte(len(r.sources)))
		msg = append(msg, ip6(r.group)...)
		for _, s := range r.sources {
			msg = append(msg, ip6(s)...)
		}
	}
	return msg
}

func replay(frames ...[]byte) *capturetest.ReplayOpener {
	o := &capturetest.ReplayOpener{LinkType: layers.LinkTypeEthernet}
	for _, f := range frames {
		o.Frames = append(o.Frames, capturetest.Frame{Data: f, Info: gopacket.CaptureInfo{Timestamp: time.Now()}})
	}
	return o
}

// A segment with two IGMP queriers, an MLD querier, and hosts reporting in
// every protocol version: each group is named with its reporters, a host
// whose last word was a leave is marked as having left, and the lowest
// querier of each family is the elected one.
func TestObserveNamesGroupsReportersAndQueriers(t *testing.T) {
	t.Parallel()

	const (
		isEx = 2
		toIn = 3
		toEx = 4
	)
	opener := replay(
		igmpFrame("10.0.0.2", "224.0.0.1", igmpV3QueryMsg()),
		igmpFrame("10.0.0.1", "224.0.0.1", igmpV2Msg(layers.IGMPMembershipQuery, 100, "0.0.0.0")),
		igmpFrame("10.0.0.1", "224.0.0.1", igmpV2Msg(layers.IGMPMembershipQuery, 100, "0.0.0.0")),
		igmpFrame("10.0.0.10", "239.1.1.1", igmpV2Msg(layers.IGMPMembershipReportV2, 0, "239.1.1.1")),
		igmpFrame("10.0.0.11", "224.0.0.22", igmpV3ReportMsg(
			groupRecord{typ: isEx, group: "239.1.1.1"},
			groupRecord{typ: toEx, group: "239.2.2.2"},
		)),
		igmpFrame("10.0.0.10", "224.0.0.2", igmpV2Msg(layers.IGMPLeaveGroup, 0, "239.1.1.1")),
		igmpFrame("10.0.0.12", "224.0.0.22", igmpV3ReportMsg(
			groupRecord{typ: toIn, group: "239.3.3.3", sources: []string{"192.0.2.1"}},
		)),
		mldFrame("fe80::1", "ff02::1", mldV2QueryMsg()),
		mldFrame("fe80::10", "ff15::1", mldV1Msg(layers.ICMPv6TypeMLDv1MulticastListenerReportMessage, "ff15::1")),
		mldFrame("fe80::11", "ff02::16", mldV2ReportMsg(
			groupRecord{typ: toEx, group: "ff15::1"},
			groupRecord{typ: toIn, group: "ff15::2"},
		)),
		mldFrame("fe80::12", "ff02::2", mldV1Msg(layers.ICMPv6TypeMLDv1MulticastListenerDoneMessage, "ff15::2")),
	)

	got, err := Observe(t.Context(), opener, ObserveRequest{Interface: "eth0", DurationSeconds: 1}, func(float64) {})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if got.ObservedMs < 1000 {
		t.Errorf("ObservedMs = %d, want the 1 s window", got.ObservedMs)
	}
	got.ObservedMs = 0

	want := &ObserveResult{
		Interface: "eth0",
		Messages:  11,
		Groups: []ObservedGroup{
			{Group: "239.1.1.1", Reporters: []ObservedReporter{
				{Address: "10.0.0.10", Version: 2, Reports: 2, Left: true},
				{Address: "10.0.0.11", Version: 3, Reports: 1},
			}},
			{Group: "239.2.2.2", Reporters: []ObservedReporter{{Address: "10.0.0.11", Version: 3, Reports: 1}}},
			{Group: "239.3.3.3", Reporters: []ObservedReporter{{Address: "10.0.0.12", Version: 3, Reports: 1}}},
			{Group: "ff15::1", Reporters: []ObservedReporter{
				{Address: "fe80::10", Version: 1, Reports: 1},
				{Address: "fe80::11", Version: 2, Reports: 1},
			}},
			{Group: "ff15::2", Reporters: []ObservedReporter{
				{Address: "fe80::11", Version: 2, Reports: 1, Left: true},
				{Address: "fe80::12", Version: 1, Reports: 1, Left: true},
			}},
		},
		Queriers: []ObservedQuerier{
			{Address: "10.0.0.1", Version: 2, Queries: 2, Elected: true},
			{Address: "10.0.0.2", Version: 3, Queries: 1},
			{Address: "fe80::1", Version: 2, Queries: 1, Elected: true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Observe =\n%+v\nwant\n%+v", got, want)
	}
}

// A snooping switch's proxy query comes from 0.0.0.0. It is a querier the
// operator needs to see, but the election ignores it, so the lowest real
// address is the elected one.
func TestProxyQuerierIsNotElected(t *testing.T) {
	t.Parallel()

	opener := replay(
		igmpFrame("0.0.0.0", "224.0.0.1", igmpV3QueryMsg()),
		igmpFrame("10.0.0.5", "224.0.0.1", igmpV3QueryMsg()),
	)
	got, err := Observe(t.Context(), opener, ObserveRequest{Interface: "eth0", DurationSeconds: 1}, func(float64) {})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	want := []ObservedQuerier{
		{Address: "0.0.0.0", Version: 3, Queries: 1},
		{Address: "10.0.0.5", Version: 3, Queries: 1, Elected: true},
	}
	if !reflect.DeepEqual(got.Queriers, want) {
		t.Fatalf("Queriers = %+v, want %+v", got.Queriers, want)
	}
}

// Frames the filter would pass but that carry no usable message are skipped
// without failing the observation.
func TestObserveSkipsMalformedMessages(t *testing.T) {
	t.Parallel()

	opener := replay(
		igmpFrame("10.0.0.10", "239.1.1.1", []byte{0x16, 0}), // truncated report
		igmpFrame("10.0.0.10", "239.1.1.1", igmpV2Msg(layers.IGMPMembershipReportV2, 0, "10.0.0.1")),
		mldFrame("fe80::10", "ff02::1", []byte{131, 0, 0, 0}), // truncated report
		mldFrame("fe80::10", "ff02::1", []byte{135, 0, 0, 0, 0, 0, 0, 0}),
	)
	got, err := Observe(t.Context(), opener, ObserveRequest{Interface: "eth0", DurationSeconds: 1}, func(float64) {})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if len(got.Groups) != 0 || len(got.Queriers) != 0 {
		t.Fatalf("groups %+v, queriers %+v, want none", got.Groups, got.Queriers)
	}
}

// Stopping an observation is how an operator says "I have my answer": it
// returns promptly with what it saw.
func TestCancelledObserveKeepsWhatItSaw(t *testing.T) {
	t.Parallel()

	opener := replay(igmpFrame("10.0.0.10", "239.1.1.1", igmpV2Msg(layers.IGMPMembershipReportV2, 0, "239.1.1.1")))
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	got, err := Observe(ctx, opener, ObserveRequest{Interface: "eth0", DurationSeconds: 600}, func(float64) {})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("cancelled observation took %v", took)
	}
	if len(got.Groups) != 1 || got.Groups[0].Group != "239.1.1.1" {
		t.Fatalf("groups = %+v, want 239.1.1.1", got.Groups)
	}
}

func TestObserveWindowIsBounded(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		seconds int
		want    time.Duration
	}{
		{0, DefaultObserveWindow},
		{-5, DefaultObserveWindow},
		{30, 30 * time.Second},
		{600, MaxObserveWindow},
		{86400, MaxObserveWindow},
	} {
		if got := observeWindow(tc.seconds); got != tc.want {
			t.Errorf("observeWindow(%d) = %v, want %v", tc.seconds, got, tc.want)
		}
	}
}

// A flood of forged reports cannot grow the tables past their bounds, and
// the result says something was left out.
func TestObservationIsBounded(t *testing.T) {
	t.Parallel()

	o := newObservation()
	src := netip.MustParseAddr("10.0.0.10")
	for i := range maxObservedGroups + 1 {
		o.membership(ip4(netip.AddrFrom4([4]byte{239, 9, byte(i >> 8), byte(i)}).String()), src, 2, false)
	}
	group := ip4("239.9.0.0") // tracked by the loop above
	for i := range maxObservedReporters + 1 {
		o.membership(group, netip.AddrFrom4([4]byte{10, 1, byte(i >> 8), byte(i)}), 2, false)
	}
	for i := range maxObservedQueriers + 1 {
		o.query(netip.AddrFrom4([4]byte{10, 2, 0, byte(i)}), 2)
	}

	res := o.result()
	if len(res.Groups) != maxObservedGroups || len(res.Queriers) != maxObservedQueriers || !res.Truncated {
		t.Fatalf("groups %d, queriers %d, truncated %v; want %d, %d, true",
			len(res.Groups), len(res.Queriers), res.Truncated, maxObservedGroups, maxObservedQueriers)
	}
	if n := len(res.Groups[0].Reporters); n != maxObservedReporters {
		t.Fatalf("239.9.0.0 has %d reporters, want %d", n, maxObservedReporters)
	}
}

func TestObserveRefusesMissingInterface(t *testing.T) {
	t.Parallel()

	_, err := Observe(t.Context(), replay(), ObserveRequest{}, func(float64) {})
	if !errors.Is(err, ErrInterfaceRequired) {
		t.Fatalf("err = %v, want ErrInterfaceRequired", err)
	}
}

type failingOpener struct{ err error }

func (f failingOpener) OpenLive(string, int32, bool, time.Duration) (capture.Handle, error) {
	return nil, f.err
}

func TestObserveReportsAnInterfaceThatCannotCapture(t *testing.T) {
	t.Parallel()

	refused := errors.New("permission denied")
	_, err := Observe(t.Context(), failingOpener{err: refused}, ObserveRequest{Interface: "eth9"}, func(float64) {})
	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want the opener's error", err)
	}
}
