package flow_test

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

// opaque frames an XDR variable-length opaque: a length word, then the
// bytes padded to four.
func opaque(body pkt) pkt {
	p := pkt{}.u32(uint32(len(body))).cat(body)
	for len(p)%4 != 0 {
		p = p.u8(0)
	}
	return p
}

// sflow builds an sFlow v5 datagram from an IPv4 agent.
func sflow(agent string, subAgent uint32, samples ...pkt) pkt {
	p := pkt{}.u32(5)
	if agent == "" {
		p = p.u32(0)
	} else {
		p = p.u32(1).addr(agent)
	}
	p = p.u32(subAgent).u32(1).u32(123_456).u32(uint32(len(samples)))
	return p.cat(samples...)
}

// sample frames a sample or flow record with its enterprise-0 format.
func sample(format uint32, body pkt) pkt { return pkt{}.u32(format).cat(opaque(body)) }

func flowSample(rate, input, output uint32, records ...pkt) pkt {
	body := pkt{}.u32(1).u32(7).u32(rate).u32(rate * 10).u32(0).
		u32(input).u32(output).u32(uint32(len(records))).cat(records...)
	return sample(1, body)
}

func expandedSample(rate, inFormat, input, outFormat, output uint32, records ...pkt) pkt {
	body := pkt{}.u32(1).u32(0).u32(7).u32(rate).u32(rate * 10).u32(0).
		u32(inFormat).u32(input).u32(outFormat).u32(output).
		u32(uint32(len(records))).cat(records...)
	return sample(3, body)
}

// headerRecord is a sampled_header record of protocol proto.
func headerRecord(proto uint32, header pkt) pkt {
	return sample(1, pkt{}.u32(proto).u32(uint32(len(header))+4).u32(4).cat(opaque(header)))
}

func ipv4Record(length uint32, proto uint8, src, dst string, sport, dport uint16, flags uint8) pkt {
	return sample(3, pkt{}.u32(length).u32(uint32(proto)).addr(src).addr(dst).
		u32(uint32(sport)).u32(uint32(dport)).u32(uint32(flags)).u32(0))
}

func ipv6Record(length uint32, proto uint8, src, dst string, sport, dport uint16) pkt {
	return sample(4, pkt{}.u32(length).u32(uint32(proto)).addr(src).addr(dst).
		u32(uint32(sport)).u32(uint32(dport)).u32(0).u32(0))
}

func ether(etherType uint16, payload pkt) pkt {
	return pkt{}.cat(make(pkt, 12)).u16(etherType).cat(payload)
}

func vlanTag(id, etherType uint16) pkt { return pkt{}.u16(id).u16(etherType) }

// ipv4Packet is an option-less IPv4 header claiming totalLen, followed
// by l4. frag is the flags and fragment-offset word.
func ipv4Packet(proto uint8, src, dst string, totalLen, frag uint16, l4 pkt) pkt {
	return pkt{}.u8(0x45).u8(0).u16(totalLen).u16(1).u16(frag).u8(64).u8(proto).u16(0).
		addr(src).addr(dst).cat(l4)
}

func ipv6Packet(next uint8, src, dst string, payloadLen uint16, rest pkt) pkt {
	return pkt{}.u32(6 << 28).u16(payloadLen).u8(next).u8(64).addr(src).addr(dst).cat(rest)
}

func tcpHeader(sport, dport uint16, flags uint8) pkt {
	return pkt{}.u16(sport).u16(dport).u32(1).u32(0).u8(0x50).u8(flags).u16(0xFFFF).u32(0)
}

func udpHeader(sport, dport uint16) pkt { return pkt{}.u16(sport).u16(dport).u16(8).u16(0) }

func TestDecodeSFlow(t *testing.T) {
	t.Parallel()
	now := exportAt()
	agent := netip.MustParseAddr("192.0.2.9")
	src4, dst4 := netip.MustParseAddr("10.1.1.1"), netip.MustParseAddr("10.2.2.2")
	src6, dst6 := netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("2001:db8::2")
	tcp := ipv4Packet(6, "10.1.1.1", "10.2.2.2", 1500, 0x4000, tcpHeader(51000, 443, 0x18))
	base := flow.Record{
		Exporter: agent, Format: flow.FormatSFlow5, ObservationDomain: 3, Start: now, End: now,
		SrcAddr: src4, DstAddr: dst4, SrcPort: 51000, DstPort: 443, Protocol: 6, TCPFlags: 0x18,
		Packets: 400, Bytes: 1500 * 400, InputIf: 12, OutputIf: 14,
	}
	with := func(edit func(*flow.Record)) flow.Record {
		r := base
		edit(&r)
		return r
	}

	tests := []struct {
		name     string
		datagram pkt
		want     []flow.Record
	}{
		{
			name:     "ethernet header, tcp",
			datagram: sflow("192.0.2.9", 3, flowSample(400, 12, 14, headerRecord(1, ether(0x0800, tcp)))),
			want:     []flow.Record{base},
		},
		{
			name: "802.1ad and 802.1Q tags",
			datagram: sflow("192.0.2.9", 3, flowSample(400, 12, 14, headerRecord(1,
				ether(0x88A8, vlanTag(100, 0x8100).cat(vlanTag(200, 0x0800), tcp))))),
			want: []flow.Record{base},
		},
		{
			name: "pre-standard 0x9100 outer tag",
			datagram: sflow("192.0.2.9", 3, flowSample(400, 12, 14, headerRecord(1,
				ether(0x9100, vlanTag(100, 0x8100).cat(vlanTag(200, 0x0800), tcp))))),
			want: []flow.Record{base},
		},
		{
			name:     "raw ipv4 header",
			datagram: sflow("192.0.2.9", 3, flowSample(400, 12, 14, headerRecord(11, tcp))),
			want:     []flow.Record{base},
		},
		{
			// The header ends inside the IP header's options; the
			// addresses are still known, the ports are not.
			name: "header cut before the ports",
			datagram: sflow("192.0.2.9", 3, flowSample(400, 12, 14, headerRecord(11,
				ipv4Packet(6, "10.1.1.1", "10.2.2.2", 1500, 0, nil)))),
			want: []flow.Record{with(func(r *flow.Record) { r.SrcPort, r.DstPort, r.TCPFlags = 0, 0, 0 })},
		},
		{
			name: "non-first fragment has no ports",
			datagram: sflow("192.0.2.9", 3, flowSample(400, 12, 14, headerRecord(11,
				ipv4Packet(6, "10.1.1.1", "10.2.2.2", 1500, 185, tcpHeader(51000, 443, 0x18))))),
			want: []flow.Record{with(func(r *flow.Record) { r.SrcPort, r.DstPort, r.TCPFlags = 0, 0, 0 })},
		},
		{
			// The UDP payload is long enough to hold a TCP flags byte;
			// UDP has none.
			name: "ipv6 udp behind a hop-by-hop header",
			datagram: sflow("192.0.2.9", 3, flowSample(400, 12, 14, headerRecord(1, ether(
				0x86DD,
				ipv6Packet(
					0,
					"2001:db8::1",
					"2001:db8::2",
					16,
					pkt{}.u8(17).
						u8(0).
						cat(make(pkt, 6), udpHeader(5353, 53), pkt{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}),
				),
			)))),
			want: []flow.Record{with(func(r *flow.Record) {
				r.SrcAddr, r.DstAddr, r.SrcPort, r.DstPort = src6, dst6, 5353, 53
				r.Protocol, r.TCPFlags, r.Bytes = 17, 0, 56*400
			})},
		},
		{
			// A decoded sampled_ipv4 record outranks the header that
			// describes the same packet, in either order.
			name: "sampled_ipv4 outranks the header",
			datagram: sflow(
				"192.0.2.9",
				3,
				flowSample(400, 12, 14, ipv4Record(1500, 6, "10.1.1.1", "10.2.2.2", 51000, 443, 0x18),
					headerRecord(11, ipv4Packet(17, "10.9.9.9", "10.9.9.8", 60, 0, udpHeader(1, 2)))),
				flowSample(
					400,
					12,
					14,
					headerRecord(11, ipv4Packet(17, "10.9.9.9", "10.9.9.8", 60, 0, udpHeader(1, 2))),
					ipv4Record(1500, 6, "10.1.1.1", "10.2.2.2", 51000, 443, 0x18),
				),
			),
			want: []flow.Record{base, base},
		},
		{
			name: "sampled_ipv6",
			datagram: sflow(
				"192.0.2.9",
				3,
				flowSample(400, 12, 14, ipv6Record(1280, 17, "2001:db8::1", "2001:db8::2", 5353, 53)),
			),
			want: []flow.Record{with(func(r *flow.Record) {
				r.SrcAddr, r.DstAddr, r.SrcPort, r.DstPort = src6, dst6, 5353, 53
				r.Protocol, r.TCPFlags, r.Bytes = 17, 0, 1280*400
			})},
		},
		{
			name: "expanded sample with large ifIndex",
			datagram: sflow("192.0.2.9", 3, expandedSample(400, 0, 1<<30+5, 0, 14,
				headerRecord(11, tcp))),
			want: []flow.Record{with(func(r *flow.Record) { r.InputIf = 1<<30 + 5 })},
		},
		{
			// Format 1 is a discarded packet, format 2 several output
			// interfaces, and the all-ones index an unknown one; none is
			// an ifIndex.
			name: "interfaces that are not an ifIndex",
			datagram: sflow("192.0.2.9", 3,
				flowSample(400, 0x3FFFFFFF, 1<<30|3, headerRecord(11, tcp)),
				expandedSample(400, 0, 0x3FFFFFFF, 2, 4, headerRecord(11, tcp))),
			want: []flow.Record{
				with(func(r *flow.Record) { r.InputIf, r.OutputIf = 0, 0 }),
				with(func(r *flow.Record) { r.InputIf, r.OutputIf = 0, 0 }),
			},
		},
		{
			name:     "rate zero counts the sample once",
			datagram: sflow("192.0.2.9", 3, flowSample(0, 12, 14, headerRecord(11, tcp))),
			want:     []flow.Record{with(func(r *flow.Record) { r.Packets, r.Bytes = 1, 1500 })},
		},
		{
			name:     "no agent address falls back to the sender",
			datagram: sflow("", 3, flowSample(400, 12, 14, headerRecord(11, tcp))),
			want:     []flow.Record{with(func(r *flow.Record) { r.Exporter = exporterA() })},
		},
		{
			// ARP is not an IP flow; counter samples and another
			// enterprise's samples are not flows at all.
			name: "samples that are not IP flows",
			datagram: sflow("192.0.2.9", 3,
				flowSample(400, 12, 14, headerRecord(1, ether(0x0806, make(pkt, 28)))),
				sample(2, pkt{}.u32(1).u32(12).u32(0)),
				sample(4413<<12|1, make(pkt, 8)),
				flowSample(400, 12, 14, sample(4413<<12|1, make(pkt, 8))),
				flowSample(400, 12, 14, headerRecord(11, tcp))),
			want: []flow.Record{base},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := flow.NewDecoder().Decode(exporterA(), tc.datagram, now)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if len(res.Records) != len(tc.want) {
				t.Fatalf("records = %d, want %d: %+v", len(res.Records), len(tc.want), res.Records)
			}
			for i := range tc.want {
				if res.Records[i] != tc.want[i] {
					t.Errorf("record %d =\n%+v\nwant\n%+v", i, res.Records[i], tc.want[i])
				}
			}
		})
	}
}

func TestDecodeSFlowMalformed(t *testing.T) {
	t.Parallel()
	good := flowSample(1, 1, 2, headerRecord(11, ipv4Packet(17, "10.0.0.1", "10.0.0.2", 60, 0, udpHeader(1, 2))))
	tests := []struct {
		name        string
		datagram    pkt
		wantErr     error
		wantRecords int
	}{
		{name: "version 4", datagram: pkt{}.u32(4).cat(make(pkt, 24)), wantErr: flow.ErrUnsupportedVersion},
		{name: "header cut short", datagram: pkt{}.u32(5).u32(1).u32(0), wantErr: flow.ErrTruncated},
		{name: "address type 3", datagram: pkt{}.u32(5).u32(3).cat(make(pkt, 32)), wantErr: flow.ErrMalformed},
		{
			name:        "sample length past the datagram",
			datagram:    sflow("192.0.2.9", 1, good, pkt{}.u32(1).u32(400)),
			wantErr:     flow.ErrTruncated,
			wantRecords: 1,
		},
		{
			name:        "fewer samples than counted",
			datagram:    sflow("192.0.2.9", 1, good, good)[:len(sflow("192.0.2.9", 1, good))],
			wantErr:     flow.ErrTruncated,
			wantRecords: 1,
		},
		{
			name:     "flow sample cut before its record count",
			datagram: sflow("192.0.2.9", 1, sample(1, pkt{}.u32(1).u32(7))),
			wantErr:  flow.ErrTruncated,
		},
		{
			name: "sampled_ipv4 protocol past a byte",
			datagram: sflow("192.0.2.9", 1, flowSample(1, 1, 2, sample(3, pkt{}.u32(60).u32(300).
				addr("10.0.0.1").addr("10.0.0.2").u32(1).u32(2).u32(0).u32(0)))),
			wantErr: flow.ErrMalformed,
		},
		{
			name:     "sampled_ipv4 record too short",
			datagram: sflow("192.0.2.9", 1, flowSample(1, 1, 2, sample(3, pkt{}.u32(60).u32(6)))),
			wantErr:  flow.ErrMalformed,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := flow.NewDecoder().Decode(exporterA(), tc.datagram, exportAt())
			if tc.wantErr == nil && err != nil || tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
			if len(res.Records) != tc.wantRecords {
				t.Errorf("records = %d, want %d", len(res.Records), tc.wantRecords)
			}
		})
	}
}

// TestDecodeSFlowNeverStampsUptime pins that a sample's time is when it
// arrived: sFlow carries no wall clock, only agent uptime.
func TestDecodeSFlowNeverStampsUptime(t *testing.T) {
	t.Parallel()
	later := exportAt().Add(90 * time.Minute)
	res, err := flow.NewDecoder().Decode(exporterA(),
		sflow(
			"192.0.2.9",
			0,
			flowSample(1, 1, 2, headerRecord(11, ipv4Packet(17, "10.0.0.1", "10.0.0.2", 60, 0, udpHeader(1, 2)))),
		),
		later)
	if err != nil || len(res.Records) != 1 {
		t.Fatalf("Decode = %d records, %v", len(res.Records), err)
	}
	if r := res.Records[0]; !r.Start.Equal(later) || !r.End.Equal(later) {
		t.Errorf("times = %v..%v, want %v", r.Start, r.End, later)
	}
}
