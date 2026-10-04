package flow_test

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

// exportAt is the export time every test datagram carries.
const exportUnix = 1_791_115_200 // 2026-10-04T12:00:00Z

func exportAt() time.Time   { return time.Unix(exportUnix, 0).UTC() }
func exporterA() netip.Addr { return netip.MustParseAddr("192.0.2.1") }
func exporterB() netip.Addr { return netip.MustParseAddr("192.0.2.2") }

// pkt assembles big-endian wire bytes.
type pkt []byte

func (p pkt) u8(v uint8) pkt   { return append(p, v) }
func (p pkt) u16(v uint16) pkt { return binary.BigEndian.AppendUint16(p, v) }
func (p pkt) u32(v uint32) pkt { return binary.BigEndian.AppendUint32(p, v) }
func (p pkt) u64(v uint64) pkt { return binary.BigEndian.AppendUint64(p, v) }
func (p pkt) addr(s string) pkt {
	return append(p, netip.MustParseAddr(s).AsSlice()...)
}

// set frames body as a v9 FlowSet or IPFIX set, padded to four bytes.
func set(id uint16, body pkt) pkt {
	for len(body)%4 != 0 {
		body = body.u8(0)
	}
	return pkt{}.u16(id).u16(uint16(4 + len(body))).cat(body)
}

func (p pkt) cat(more ...pkt) pkt {
	for _, m := range more {
		p = append(p, m...)
	}
	return p
}

// v9 builds a NetFlow v9 datagram with the given uptime and sets.
func v9(sourceID, uptime uint32, sets ...pkt) pkt {
	return pkt{}.u16(9).u16(uint16(len(sets))).u32(uptime).
		u32(uint32(exportAt().Unix())).u32(1).u32(sourceID).cat(sets...)
}

// ipfix builds an IPFIX message with the given sets.
func ipfix(domain uint32, sets ...pkt) pkt {
	body := pkt{}.cat(sets...)
	return pkt{}.u16(10).u16(uint16(16 + len(body))).
		u32(uint32(exportAt().Unix())).u32(1).u32(domain).cat(body)
}

// v9Template() is template 256: IPv4 5-tuple, counters, interfaces and
// uptime timestamps, in v9 field types.
func v9Template() pkt {
	return set(0, pkt{}.u16(256).u16(10).
		u16(8).u16(4).  // IPV4_SRC_ADDR
		u16(12).u16(4). // IPV4_DST_ADDR
		u16(7).u16(2).  // L4_SRC_PORT
		u16(11).u16(2). // L4_DST_PORT
		u16(4).u16(1).  // PROTOCOL
		u16(1).u16(4).  // IN_BYTES
		u16(2).u16(4).  // IN_PKTS
		u16(10).u16(2). // INPUT_SNMP
		u16(22).u16(4). // FIRST_SWITCHED
		u16(21).u16(4), // LAST_SWITCHED
	)
}

func v9Data(src, dst string, sport, dport uint16, bytes, packets uint32, first, last uint32) pkt {
	return pkt{}.addr(src).addr(dst).u16(sport).u16(dport).u8(6).
		u32(bytes).u32(packets).u16(3).u32(first).u32(last)
}

func TestDecodeV5(t *testing.T) {
	t.Parallel()
	const uptime = 100_000
	rec := func(src string, bytes, packets, first, last uint32) pkt {
		return pkt{}.addr(src).addr("198.51.100.7").addr("0.0.0.0").
			u16(2).u16(5).u32(packets).u32(bytes).u32(first).u32(last).
			u16(51000).u16(443).u8(0).u8(0x12).u8(6).u8(0).
			u16(0).u16(0).u8(24).u8(24).u16(0)
	}
	datagram := func(sampling uint16) pkt {
		return pkt{}.u16(5).u16(2).u32(uptime).u32(uint32(exportAt().Unix())).u32(0).
			u32(7).u8(1).u8(4).u16(sampling).
			cat(rec("10.0.0.1", 1500, 3, 40_000, 90_000), rec("10.0.0.2", 60, 1, 99_000, 99_500))
	}

	tests := []struct {
		name        string
		sampling    uint16
		wantBytes   uint64
		wantPackets uint64
	}{
		{name: "unsampled", sampling: 0, wantBytes: 1500, wantPackets: 3},
		// Mode bits 01 (deterministic), interval 100.
		{name: "sampled one in 100", sampling: 1<<14 | 100, wantBytes: 150_000, wantPackets: 300},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := flow.NewDecoder().Decode(exporterA(), datagram(tc.sampling), exportAt())
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if len(res.Records) != 2 {
				t.Fatalf("records = %d, want 2", len(res.Records))
			}
			got := res.Records[0]
			want := flow.Record{
				Exporter: exporterA(), Version: 5, ObservationDomain: 1<<8 | 4,
				Start:   exportAt().Add(-60 * time.Second),
				End:     exportAt().Add(-10 * time.Second),
				SrcAddr: netip.MustParseAddr("10.0.0.1"), DstAddr: netip.MustParseAddr("198.51.100.7"),
				SrcPort: 51000, DstPort: 443, Protocol: 6, TCPFlags: 0x12,
				Bytes: tc.wantBytes, Packets: tc.wantPackets, InputIf: 2, OutputIf: 5,
			}
			if got != want {
				t.Errorf("record =\n%+v\nwant\n%+v", got, want)
			}
		})
	}
}

func TestDecodeV9TemplateLossAndRelearn(t *testing.T) {
	t.Parallel()
	const uptime = 500_000
	data := set(256, v9Data("10.1.1.1", "10.2.2.2", 40000, 53, 900, 9, 470_000, 499_000))
	dec := flow.NewDecoder()

	// The collector started after the exporter sent its template: the
	// data set is counted, not decoded and not held.
	res, err := dec.Decode(exporterA(), v9(7, uptime, data), exportAt())
	if err != nil || len(res.Records) != 0 || res.MissingTemplate != 1 {
		t.Fatalf("before template: records=%d missing=%d err=%v; want 0, 1, nil",
			len(res.Records), res.MissingTemplate, err)
	}

	// The exporter's next refresh carries the template ahead of data in
	// the same datagram.
	res, err = dec.Decode(exporterA(), v9(7, uptime, v9Template(), data), exportAt())
	if err != nil || len(res.Records) != 1 || res.MissingTemplate != 0 {
		t.Fatalf("with template: records=%d missing=%d err=%v; want 1, 0, nil",
			len(res.Records), res.MissingTemplate, err)
	}
	want := flow.Record{
		Exporter: exporterA(), Version: 9, ObservationDomain: 7,
		Start: exportAt().Add(-30 * time.Second), End: exportAt().Add(-time.Second),
		SrcAddr: netip.MustParseAddr("10.1.1.1"), DstAddr: netip.MustParseAddr("10.2.2.2"),
		SrcPort: 40000, DstPort: 53, Protocol: 6, Bytes: 900, Packets: 9, InputIf: 3,
	}
	if res.Records[0] != want {
		t.Errorf("record =\n%+v\nwant\n%+v", res.Records[0], want)
	}

	// Later data-only datagrams decode against the learned template.
	res, _ = dec.Decode(exporterA(), v9(7, uptime, data), exportAt().Add(time.Minute))
	if len(res.Records) != 1 {
		t.Errorf("data after learning: records = %d, want 1", len(res.Records))
	}

	// A template is scoped to its exporter and source ID.
	for _, tc := range []struct {
		name     string
		exporter netip.Addr
		sourceID uint32
	}{
		{name: "other exporter", exporter: exporterB(), sourceID: 7},
		{name: "other source ID", exporter: exporterA(), sourceID: 8},
	} {
		res, _ = dec.Decode(tc.exporter, v9(tc.sourceID, uptime, data), exportAt())
		if res.MissingTemplate != 1 || len(res.Records) != 0 {
			t.Errorf("%s: missing=%d records=%d, want 1, 0", tc.name, res.MissingTemplate, len(res.Records))
		}
	}

	// An exporter that stops refreshing its template loses it.
	res, _ = dec.Decode(exporterA(), v9(7, uptime, data), exportAt().Add(2*time.Hour))
	if res.MissingTemplate != 1 {
		t.Errorf("expired template: missing = %d, want 1", res.MissingTemplate)
	}
}

func TestDecodeV9TemplateRedefined(t *testing.T) {
	t.Parallel()
	dec := flow.NewDecoder()
	if _, err := dec.Decode(exporterA(), v9(1, 1000, v9Template()), exportAt()); err != nil {
		t.Fatal(err)
	}
	// After a restart the exporter reuses ID 256 for an IPv6 layout.
	v6Template := set(0, pkt{}.u16(256).u16(3).
		u16(27).u16(16). // IPV6_SRC_ADDR
		u16(28).u16(16). // IPV6_DST_ADDR
		u16(1).u16(8))   // IN_BYTES, 64-bit
	data := set(256, pkt{}.addr("2001:db8::1").addr("2001:db8::2").u64(1<<40))
	res, err := dec.Decode(exporterA(), v9(1, 1000, v6Template, data), exportAt())
	if err != nil || len(res.Records) != 1 {
		t.Fatalf("records=%d err=%v, want 1 record", len(res.Records), err)
	}
	got := res.Records[0]
	if got.SrcAddr != netip.MustParseAddr("2001:db8::1") || got.Bytes != 1<<40 {
		t.Errorf("record = %+v, want the IPv6 layout", got)
	}
	// No timestamps in the template: the record falls back to the export time.
	if !got.Start.Equal(exportAt()) || !got.End.Equal(exportAt()) {
		t.Errorf("start, end = %v, %v; want the export time", got.Start, got.End)
	}
}

func TestDecodeV9OptionsDataSkipped(t *testing.T) {
	t.Parallel()
	// Options template 300: scope System (4 bytes), option SAMPLING_INTERVAL.
	opts := set(1, pkt{}.u16(300).u16(4).u16(4).u16(1).u16(4).u16(34).u16(4))
	optsData := set(300, pkt{}.u32(1).u32(100))
	res, err := flow.NewDecoder().Decode(exporterA(), v9(1, 1000, opts, optsData), exportAt())
	if err != nil || len(res.Records) != 0 || res.MissingTemplate != 0 {
		t.Errorf("records=%d missing=%d err=%v; want 0, 0, nil", len(res.Records), res.MissingTemplate, err)
	}
}

func TestDecodeIPFIX(t *testing.T) {
	t.Parallel()
	// Template 400 exercises what v9 cannot carry: an enterprise-specific
	// field, a variable-length field, reduced-size counters and absolute
	// millisecond timestamps.
	tmpl := set(2, pkt{}.u16(400).u16(9).
		u16(8).u16(4).                     // sourceIPv4Address
		u16(12).u16(4).                    // destinationIPv4Address
		u16(0x8000|1000).u16(2).u32(9999). // enterprise field, skipped
		u16(82).u16(0xFFFF).               // interfaceName, variable length
		u16(1).u16(4).                     // octetDeltaCount, reduced to 4 bytes
		u16(86).u16(8).                    // packetTotalCount
		u16(6).u16(2).                     // tcpControlBits, 16-bit form
		u16(152).u16(8).                   // flowStartMilliseconds
		u16(153).u16(8))                   // flowEndMilliseconds
	startMs := uint64(exportAt().Add(-5 * time.Second).UnixMilli())
	endMs := uint64(exportAt().Add(-1500 * time.Millisecond).UnixMilli())
	rec := func(name string) pkt {
		r := pkt{}.addr("172.16.0.9").addr("172.16.0.10").u16(0xBEEF)
		if len(name) >= 255 {
			r = r.u8(255).u16(uint16(len(name)))
		} else {
			r = r.u8(uint8(len(name)))
		}
		return r.cat(pkt(name)).u32(4096).u64(12).u16(0x0118).u64(startMs).u64(endMs)
	}
	long := string(make([]byte, 300))
	data := set(400, rec("Gi0/1").cat(rec(long)))

	res, err := flow.NewDecoder().Decode(exporterA(), ipfix(42, tmpl, data), exportAt())
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(res.Records) != 2 {
		t.Fatalf("records = %d, want 2", len(res.Records))
	}
	want := flow.Record{
		Exporter: exporterA(), Version: 10, ObservationDomain: 42,
		Start: exportAt().Add(-5 * time.Second), End: exportAt().Add(-1500 * time.Millisecond),
		SrcAddr: netip.MustParseAddr("172.16.0.9"), DstAddr: netip.MustParseAddr("172.16.0.10"),
		TCPFlags: 0x18, Bytes: 4096, Packets: 12,
	}
	for i, got := range res.Records {
		if got != want {
			t.Errorf("record %d =\n%+v\nwant\n%+v", i, got, want)
		}
	}
}

func TestDecodeIPFIXWithdrawal(t *testing.T) {
	t.Parallel()
	tmpl := func(id uint16) pkt {
		return pkt{}.u16(id).u16(2).u16(8).u16(4).u16(12).u16(4)
	}
	data := func(id uint16) pkt { return set(id, pkt{}.addr("10.0.0.1").addr("10.0.0.2")) }

	tests := []struct {
		name        string
		withdrawal  pkt
		wantMissing int
	}{
		{name: "one template", withdrawal: set(2, pkt{}.u16(500).u16(0)), wantMissing: 1},
		{name: "every template", withdrawal: set(2, pkt{}.u16(2).u16(0)), wantMissing: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dec := flow.NewDecoder()
			if _, err := dec.Decode(exporterA(), ipfix(1, set(2, tmpl(500).cat(tmpl(501)))), exportAt()); err != nil {
				t.Fatal(err)
			}
			if _, err := dec.Decode(exporterA(), ipfix(1, tc.withdrawal), exportAt()); err != nil {
				t.Fatal(err)
			}
			res, _ := dec.Decode(exporterA(), ipfix(1, data(500), data(501)), exportAt())
			if res.MissingTemplate != tc.wantMissing {
				t.Errorf("missing = %d, want %d", res.MissingTemplate, tc.wantMissing)
			}
		})
	}
}

func TestDecodeMalformed(t *testing.T) {
	t.Parallel()
	good := set(256, v9Data("10.0.0.1", "10.0.0.2", 1, 2, 3, 4, 5, 6))
	tests := []struct {
		name        string
		datagram    pkt
		wantErr     error
		wantRecords int
	}{
		{name: "empty", datagram: pkt{}, wantErr: flow.ErrTruncated},
		{name: "unknown version", datagram: pkt{}.u16(7).u16(0), wantErr: flow.ErrUnsupportedVersion},
		{name: "v5 header only", datagram: pkt{}.u16(5).u16(1), wantErr: flow.ErrTruncated},
		{name: "v5 zero records", datagram: pkt{}.u16(5).u16(0).cat(make(pkt, 20)), wantErr: flow.ErrMalformed},
		{
			name:     "v5 short of its count",
			datagram: pkt{}.u16(5).u16(2).cat(make(pkt, 20+48)),
			wantErr:  flow.ErrTruncated,
		},
		{
			name:     "ipfix length past datagram",
			datagram: pkt{}.u16(10).u16(100).cat(make(pkt, 12)),
			wantErr:  flow.ErrTruncated,
		},
		{
			name:     "set length past datagram",
			datagram: v9(1, 1000, pkt{}.u16(256).u16(64)),
			wantErr:  flow.ErrMalformed,
		},
		{
			name:     "set length under its header",
			datagram: v9(1, 1000, pkt{}.u16(256).u16(2)),
			wantErr:  flow.ErrMalformed,
		},
		{
			name:     "template of zero-length fields",
			datagram: v9(1, 1000, set(0, pkt{}.u16(256).u16(1).u16(8).u16(0))),
			wantErr:  flow.ErrMalformed,
		},
		{
			name:     "template ID in the reserved range",
			datagram: v9(1, 1000, set(0, pkt{}.u16(255).u16(1).u16(8).u16(4))),
			wantErr:  flow.ErrMalformed,
		},
		{
			name:     "template over the field bound",
			datagram: v9(1, 1000, set(0, pkt{}.u16(256).u16(257))),
			wantErr:  flow.ErrMalformed,
		},
		{
			name:     "variable length in v9",
			datagram: v9(1, 1000, set(0, pkt{}.u16(256).u16(1).u16(82).u16(0xFFFF))),
			wantErr:  flow.ErrMalformed,
		},
		{
			// Records already decoded are kept when a later set is bad.
			name:        "bad set after good data",
			datagram:    v9(1, 1000, v9Template(), good, pkt{}.u16(256).u16(200)),
			wantErr:     flow.ErrMalformed,
			wantRecords: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := flow.NewDecoder().Decode(exporterA(), tc.datagram, exportAt())
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
			if len(res.Records) != tc.wantRecords {
				t.Errorf("records = %d, want %d", len(res.Records), tc.wantRecords)
			}
		})
	}
}

func FuzzDecode(f *testing.F) {
	f.Add([]byte(v9(1, 1000, v9Template(), set(256, v9Data("10.0.0.1", "10.0.0.2", 1, 2, 3, 4, 5, 6)))))
	f.Add([]byte(ipfix(1, set(2, pkt{}.u16(400).u16(1).u16(82).u16(0xFFFF)), set(400, pkt{}.u8(255).u16(9)))))
	f.Add([]byte(set(3, pkt{}.u16(300).u16(1).u16(1).u16(1).u16(4))))
	f.Fuzz(func(t *testing.T, b []byte) {
		dec := flow.NewDecoder()
		// Decode twice so templates learned from the input are exercised
		// against its own data sets.
		_, _ = dec.Decode(exporterA(), b, exportAt())
		res, _ := dec.Decode(exporterA(), b, exportAt())
		if len(res.Records) > len(b) {
			t.Fatalf("%d records from %d bytes", len(res.Records), len(b))
		}
	})
}

// TestDecodeIPFIXUptimeTimes follows softflowd's layout: flow times are
// uptime-relative and the exporter's boot time arrives in an options
// record, scoped by meteringProcessId, ahead of the data.
func TestDecodeIPFIXUptimeTimes(t *testing.T) {
	t.Parallel()
	boot := exportAt().Add(-time.Hour)
	options := set(3, pkt{}.u16(256).u16(2).u16(1).
		u16(143).u16(4). // meteringProcessId, the scope
		u16(160).u16(8)) // systemInitTimeMilliseconds
	optionsData := set(256, pkt{}.u32(1).u64(uint64(boot.UnixMilli())))
	tmpl := set(2, pkt{}.u16(1024).u16(4).
		u16(8).u16(4).  // sourceIPv4Address
		u16(12).u16(4). // destinationIPv4Address
		u16(22).u16(4). // flowStartSysUpTime
		u16(21).u16(4)) // flowEndSysUpTime
	data := set(1024, pkt{}.addr("10.0.0.1").addr("10.0.0.2").u32(10_000).u32(12_500))

	tests := []struct {
		name      string
		messages  []pkt
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "boot time in the same message",
			messages:  []pkt{ipfix(1, options, optionsData, tmpl, data)},
			wantStart: boot.Add(10 * time.Second),
			wantEnd:   boot.Add(12500 * time.Millisecond),
		},
		{
			name:      "boot time from an earlier message",
			messages:  []pkt{ipfix(1, options, optionsData), ipfix(1, tmpl, data)},
			wantStart: boot.Add(10 * time.Second),
			wantEnd:   boot.Add(12500 * time.Millisecond),
		},
		{
			// An exporter up 60 days has wrapped its 32-bit uptime once.
			name: "uptime past the 49.7-day wrap",
			messages: []pkt{ipfix(1, options,
				set(256, pkt{}.u32(1).u64(uint64(exportAt().Add(-60*24*time.Hour).UnixMilli()))),
				tmpl, set(1024, pkt{}.addr("10.0.0.1").addr("10.0.0.2").
					u32(wrapped(60*24*time.Hour-10*time.Second)).
					u32(wrapped(60*24*time.Hour-2*time.Second))))},
			wantStart: exportAt().Add(-10 * time.Second),
			wantEnd:   exportAt().Add(-2 * time.Second),
		},
		{
			// A flow that predates the boot time, as softflowd reports a
			// capture file read after it starts.
			name: "flow before the boot time",
			messages: []pkt{ipfix(1, options,
				set(256, pkt{}.u32(1).u64(uint64(exportAt().Add(-time.Minute).UnixMilli()))),
				tmpl, set(1024, pkt{}.addr("10.0.0.1").addr("10.0.0.2").
					u32(wrapped(-3*time.Minute)).u32(wrapped(-2*time.Minute))))},
			wantStart: exportAt().Add(-4 * time.Minute),
			wantEnd:   exportAt().Add(-3 * time.Minute),
		},
		{
			name:      "boot time for another domain",
			messages:  []pkt{ipfix(2, options, optionsData), ipfix(1, tmpl, data)},
			wantStart: exportAt(),
			wantEnd:   exportAt(),
		},
		{
			name:      "no boot time",
			messages:  []pkt{ipfix(1, tmpl, data)},
			wantStart: exportAt(),
			wantEnd:   exportAt(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dec := flow.NewDecoder()
			var records []flow.Record
			for _, m := range tc.messages {
				res, err := dec.Decode(exporterA(), m, exportAt())
				if err != nil {
					t.Fatalf("Decode: %v", err)
				}
				records = append(records, res.Records...)
			}
			if len(records) != 1 {
				t.Fatalf("records = %d, want 1", len(records))
			}
			if !records[0].Start.Equal(tc.wantStart) || !records[0].End.Equal(tc.wantEnd) {
				t.Errorf("start, end = %v, %v; want %v, %v",
					records[0].Start, records[0].End, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

// wrapped is an uptime offset as an exporter's 32-bit millisecond counter
// carries it.
func wrapped(d time.Duration) uint32 {
	return uint32(d.Milliseconds() & 0xFFFFFFFF)
}

// TestDecodeSamplingRate covers the sampling rates v9 and IPFIX exporters
// announce, each shaped after an exporter that sends it: softflowd scopes
// a v9 rate to an interface and an IPFIX one to its metering process,
// Cisco names a sampler in each data record, and some exporters carry the
// rate in the data record itself.
func TestDecodeSamplingRate(t *testing.T) {
	t.Parallel()
	const uptime = 500_000
	v9Flow := set(256, v9Data("10.1.1.1", "10.2.2.2", 40000, 53, 1000, 3, 470_000, 499_000))
	// softflowd -v 9 -s 10: scope Interface, SAMPLING_INTERVAL,
	// SAMPLING_ALGORITHM.
	v9IfOptions := set(1, pkt{}.u16(300).u16(4).u16(8).
		u16(2).u16(4).u16(34).u16(4).u16(35).u16(1))
	v9IfRate := func(ifIndex, interval uint32) pkt {
		return set(300, pkt{}.u32(ifIndex).u32(interval).u8(1))
	}
	// Cisco random sampler: scope System, FLOW_SAMPLER_ID,
	// FLOW_SAMPLER_MODE, FLOW_SAMPLER_RANDOM_INTERVAL.
	v9SamplerOptions := set(1, pkt{}.u16(301).u16(4).u16(12).
		u16(1).u16(4).u16(48).u16(1).u16(49).u16(1).u16(50).u16(4))
	v9SamplerRate := set(301, pkt{}.u32(0).u8(7).u8(2).u32(100))
	v9SamplerTemplate := set(0, pkt{}.u16(257).u16(5).
		u16(8).u16(4).u16(12).u16(4).u16(1).u16(4).u16(2).u16(4).u16(48).u16(1))
	v9SamplerFlow := func(sampler uint8) pkt {
		return set(257, pkt{}.addr("10.1.1.1").addr("10.2.2.2").u32(1000).u32(3).u8(sampler))
	}
	v9InRecordTemplate := set(0, pkt{}.u16(258).u16(5).
		u16(8).u16(4).u16(12).u16(4).u16(1).u16(4).u16(2).u16(4).u16(34).u16(4))
	v9InRecordFlow := set(258, pkt{}.addr("10.1.1.1").addr("10.2.2.2").u32(1000).u32(3).u32(4))
	// softflowd -v 10 -s 10: scope meteringProcessId,
	// samplingPacketInterval, samplingPacketSpace.
	ipfixOptions := set(3, pkt{}.u16(256).u16(3).u16(1).
		u16(143).u16(4).u16(305).u16(4).u16(306).u16(4))
	ipfixRate := func(interval, space uint32) pkt {
		return set(256, pkt{}.u32(1).u32(interval).u32(space))
	}
	ipfixTemplate := set(2, pkt{}.u16(1024).u16(5).
		u16(8).u16(4).u16(12).u16(4).u16(1).u16(4).u16(2).u16(4).u16(10).u16(4))
	ipfixFlow := set(1024, pkt{}.addr("10.0.0.1").addr("10.0.0.2").u32(1000).u32(3).u32(3))

	type message struct {
		p     pkt
		after time.Duration
	}
	tests := []struct {
		name string
		msgs []message
		// want is each decoded record's bytes and packets, in order.
		want [][2]uint64
	}{
		{
			name: "v9 rate for the flow's interface",
			msgs: []message{{p: v9(1, uptime, v9IfOptions, v9IfRate(3, 10), v9Template(), v9Flow)}},
			want: [][2]uint64{{10_000, 30}},
		},
		{
			name: "v9 rate for another interface",
			msgs: []message{{p: v9(1, uptime, v9IfOptions, v9IfRate(5, 10), v9Template(), v9Flow)}},
			want: [][2]uint64{{1000, 3}},
		},
		{
			name: "v9 system-wide rate",
			msgs: []message{{p: v9(1, uptime,
				set(1, pkt{}.u16(300).u16(4).u16(4).u16(1).u16(4).u16(34).u16(4)),
				set(300, pkt{}.u32(1).u32(100)), v9Template(), v9Flow)}},
			want: [][2]uint64{{100_000, 300}},
		},
		{
			name: "v9 rate for the flow's sampler",
			msgs: []message{{p: v9(1, uptime, v9SamplerOptions, v9SamplerRate,
				v9SamplerTemplate, v9SamplerFlow(7), v9SamplerFlow(8))}},
			want: [][2]uint64{{100_000, 300}, {1000, 3}},
		},
		{
			name: "v9 rate in the data record",
			msgs: []message{{p: v9(1, uptime, v9IfOptions, v9IfRate(0, 10), v9InRecordTemplate, v9InRecordFlow)}},
			want: [][2]uint64{{4000, 12}},
		},
		{
			name: "v9 rate from another domain",
			msgs: []message{
				{p: v9(2, uptime, v9IfOptions, v9IfRate(3, 10))},
				{p: v9(1, uptime, v9Template(), v9Flow)},
			},
			want: [][2]uint64{{1000, 3}},
		},
		{
			name: "IPFIX interval and space",
			msgs: []message{{p: ipfix(1, ipfixOptions, ipfixRate(1, 9), ipfixTemplate, ipfixFlow)}},
			want: [][2]uint64{{10_000, 30}},
		},
		{
			name: "IPFIX fractional rate",
			msgs: []message{{p: ipfix(1, ipfixOptions, ipfixRate(2, 3), ipfixTemplate, ipfixFlow)}},
			want: [][2]uint64{{2500, 7}},
		},
		{
			name: "IPFIX rate changed",
			msgs: []message{
				{p: ipfix(1, ipfixOptions, ipfixRate(1, 9), ipfixTemplate)},
				{p: ipfix(1, ipfixRate(1, 0), ipfixFlow)},
			},
			want: [][2]uint64{{1000, 3}},
		},
		{
			// The options record arrives after the flow: the flow keeps
			// its sampled counts and later flows are scaled.
			name: "flow before the rate",
			msgs: []message{
				{p: ipfix(1, ipfixOptions, ipfixTemplate, ipfixFlow)},
				{p: ipfix(1, ipfixRate(1, 9), ipfixFlow)},
			},
			want: [][2]uint64{{1000, 3}, {10_000, 30}},
		},
		{
			name: "rate not refreshed",
			msgs: []message{
				{p: ipfix(1, ipfixOptions, ipfixRate(1, 9))},
				{p: ipfix(1, ipfixTemplate, ipfixFlow), after: 2 * time.Hour},
			},
			want: [][2]uint64{{1000, 3}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dec := flow.NewDecoder()
			var got [][2]uint64
			for _, m := range tc.msgs {
				res, err := dec.Decode(exporterA(), m.p, exportAt().Add(m.after))
				if err != nil {
					t.Fatalf("Decode: %v", err)
				}
				for _, r := range res.Records {
					got = append(got, [2]uint64{r.Bytes, r.Packets})
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("bytes, packets = %v, want %v", got, tc.want)
			}
		})
	}
}
