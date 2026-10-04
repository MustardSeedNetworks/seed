package flow

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/netip"
	"time"
)

// sFlow v5 (sflow.org/sflow_version_5.txt) is XDR: every field is a
// four-byte big-endian word or an opaque padded to four bytes. A datagram
// carries samples; a flow sample is one packet the agent sampled 1-in-N,
// described by one or more flow records. Counter samples are interface
// statistics, not traffic, and are skipped.
const (
	sflowAddrIPv4 = 1
	sflowAddrIPv6 = 2

	// Sample and record formats in enterprise 0, which is all a format
	// word with a zero enterprise field can name.
	sflowFlowSample         = 1
	sflowExpandedFlowSample = 3
	sflowSampledHeader      = 1
	sflowSampledIPv4        = 3
	sflowSampledIPv6        = 4

	// sampled_header protocols.
	sflowHeaderEthernet = 1
	sflowHeaderIPv4     = 11
	sflowHeaderIPv6     = 12

	// A compact interface word keeps its format in the top two bits;
	// format 0 is an ifIndex, and the all-ones index means unknown.
	sflowIfFormatShift = 30
	sflowIfIndexMask   = 0x3FFFFFFF

	sflowEnterpriseShift = 12
	sflowFormatMask      = 0xFFF
	xdrWord              = 4
	net4Len              = 4
	net16Len             = 16
)

// xdr reads XDR words. The first short read sets err; later reads
// return zero values, so a decoder checks err once per structure.
type xdr struct {
	b   []byte
	err error
}

func (x *xdr) u32() uint32 {
	b := x.take(xdrWord)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint32(b)
}

// u8 and u16 read a word that holds a narrower value; a larger value is
// malformed.
func (x *xdr) u8() uint8 {
	v := x.u32()
	if v > math.MaxUint8 {
		x.fail(v)
		return 0
	}
	return uint8(v)
}

func (x *xdr) u16() uint16 {
	v := x.u32()
	if v > math.MaxUint16 {
		x.fail(v)
		return 0
	}
	return uint16(v)
}

func (x *xdr) fail(v uint32) {
	if x.err == nil {
		x.err = fmt.Errorf("%w: sflow value %d out of range", ErrMalformed, v)
	}
}

// take returns the next n bytes and skips the padding to a word.
func (x *xdr) take(n int) []byte {
	if x.err != nil {
		return nil
	}
	padded := (n + xdrWord - 1) &^ (xdrWord - 1)
	if n < 0 || padded > len(x.b) {
		x.err = ErrTruncated
		return nil
	}
	b := x.b[:n:n]
	x.b = x.b[padded:]
	return b
}

// opaque reads a length-prefixed variable-length opaque.
func (x *xdr) opaque() []byte {
	return x.take(int(x.u32()))
}

// addr reads an sFlow address: a type word, then four or sixteen bytes.
func (x *xdr) addr() netip.Addr {
	switch t := x.u32(); t {
	case 0:
		return netip.Addr{}
	case sflowAddrIPv4:
		if b := x.take(net4Len); b != nil {
			return netip.AddrFrom4([4]byte(b))
		}
	case sflowAddrIPv6:
		if b := x.take(net16Len); b != nil {
			return netip.AddrFrom16([16]byte(b)).Unmap()
		}
	default:
		x.fail(t)
	}
	return netip.Addr{}
}

// decodeSFlow decodes an sFlow v5 datagram received at now. Each flow
// sample of an IP packet becomes one record standing for the packets
// the sample represents: Packets is the sampling rate and Bytes the IP
// length times the rate, the same layer-3 octets NetFlow counts.
func decodeSFlow(from netip.Addr, pkt []byte, now time.Time) (Result, error) {
	x := xdr{b: pkt}
	if v := x.u32(); v != VersionSFlow5 && x.err == nil {
		return Result{}, fmt.Errorf("%w: sflow %d", ErrUnsupportedVersion, v)
	}
	agent := x.addr()
	subAgent := x.u32()
	x.u32() // sequence number
	x.u32() // agent uptime
	count := x.u32()
	if x.err != nil {
		return Result{}, x.err
	}
	if !agent.IsValid() {
		agent = from
	}

	var res Result
	for range count {
		format := x.u32()
		body := x.opaque()
		if x.err != nil {
			return res, x.err
		}
		kind := format & sflowFormatMask
		if format>>sflowEnterpriseShift != 0 || (kind != sflowFlowSample && kind != sflowExpandedFlowSample) {
			continue
		}
		rec, ok, err := decodeFlowSample(&xdr{b: body}, kind == sflowExpandedFlowSample)
		if err != nil {
			return res, err
		}
		if !ok {
			continue
		}
		rec.Exporter, rec.Format, rec.ObservationDomain = agent, FormatSFlow5, subAgent
		rec.Start, rec.End = now, now
		res.Records = append(res.Records, rec)
	}
	return res, nil
}

// decodeFlowSample reads a flow_sample or expanded_flow_sample. ok is
// false for a sample of a packet that is not IP.
func decodeFlowSample(x *xdr, expanded bool) (Record, bool, error) {
	var rec Record
	x.u32() // sequence number
	if expanded {
		x.u32() // source ID type
	}
	x.u32() // source ID index, or the compact source ID
	rate := uint64(x.u32())
	x.u32() // sample pool
	x.u32() // drops
	if expanded {
		rec.InputIf = expandedInterface(x.u32(), x.u32())
		rec.OutputIf = expandedInterface(x.u32(), x.u32())
	} else {
		rec.InputIf = compactInterface(x.u32())
		rec.OutputIf = compactInterface(x.u32())
	}
	count := x.u32()
	if x.err != nil {
		return Record{}, false, x.err
	}
	if rate == 0 {
		rate = 1
	}

	ip, found, err := readFlowRecords(x, count)
	if err != nil || !found {
		return Record{}, false, err
	}
	rec.SrcAddr, rec.DstAddr = ip.src, ip.dst
	rec.SrcPort, rec.DstPort = ip.srcPort, ip.dstPort
	rec.Protocol, rec.TCPFlags = ip.protocol, ip.tcpFlags
	rec.Packets, rec.Bytes = rate, uint64(ip.length)*rate
	return rec, true, nil
}

// readFlowRecords reads a flow sample's records for the sampled packet's
// flow key. An agent may describe one packet twice, as a header and as a
// decoded sampled_ipv4/ipv6; the decoded record wins.
func readFlowRecords(x *xdr, count uint32) (packetInfo, bool, error) {
	var (
		ip       packetInfo
		found    bool
		explicit bool
	)
	for range count {
		format := x.u32()
		body := x.opaque()
		if x.err != nil {
			return packetInfo{}, false, x.err
		}
		if format>>sflowEnterpriseShift != 0 {
			continue
		}
		r := &xdr{b: body}
		switch format & sflowFormatMask {
		case sflowSampledHeader:
			if explicit {
				continue
			}
			if p, ok := sampledHeader(r); ok {
				ip, found = p, true
			}
		case sflowSampledIPv4:
			ip, found, explicit = sampledIP(r, net4Len), true, true
		case sflowSampledIPv6:
			ip, found, explicit = sampledIP(r, net16Len), true, true
		}
		if r.err != nil {
			return packetInfo{}, false, fmt.Errorf("%w: sflow flow record %d: %w", ErrMalformed, format, r.err)
		}
	}
	return ip, found, nil
}

func compactInterface(v uint32) uint32 {
	if v>>sflowIfFormatShift != 0 || v&sflowIfIndexMask == sflowIfIndexMask {
		return 0
	}
	return v
}

func expandedInterface(format, value uint32) uint32 {
	if format != 0 || value == sflowIfIndexMask {
		return 0
	}
	return value
}

// sampledIP reads a sampled_ipv4 or sampled_ipv6 record, whose addresses
// are addrLen bytes.
func sampledIP(x *xdr, addrLen int) packetInfo {
	p := packetInfo{length: x.u32(), protocol: x.u8()}
	src, dst := x.take(addrLen), x.take(addrLen)
	p.srcPort, p.dstPort = x.u16(), x.u16()
	p.tcpFlags = x.u8()
	if x.err != nil {
		return packetInfo{}
	}
	p.src, _ = netip.AddrFromSlice(src)
	p.dst, _ = netip.AddrFromSlice(dst)
	return p
}

// sampledHeader reads a sampled_header record and parses the packet
// bytes it carries. ok is false for a protocol other than Ethernet, IPv4
// or IPv6, or a header too short to hold the IP addresses.
func sampledHeader(x *xdr) (packetInfo, bool) {
	proto := x.u32()
	x.u32() // frame length
	x.u32() // octets stripped
	header := x.opaque()
	if x.err != nil {
		return packetInfo{}, false
	}
	switch proto {
	case sflowHeaderEthernet:
		return parseEthernet(header)
	case sflowHeaderIPv4, sflowHeaderIPv6:
		return parseIP(header)
	}
	return packetInfo{}, false
}
