// Package mdnswire reads and writes the mDNS messages seed's discovery uses:
// PTR, SRV, TXT, A and AAAA records, and single-question queries.
//
// It exists because golang.org/x/net/dns/dnsmessage refuses a whole message
// when any label holds a literal dot. DNS-SD instance names are free-form
// UTF-8 chosen by a device's owner ("Office No. 5"), and the wire keeps such a
// name inside one length-prefixed label, so one dotted neighbour made every
// record in its announcement invisible (#2626). Here a name is its labels, and
// a label may hold any byte.
package mdnswire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// Type is a DNS resource record type.
type Type uint16

// The record types discovery reads. Others are skipped when parsing.
const (
	TypeA    Type = 1
	TypePTR  Type = 12
	TypeTXT  Type = 16
	TypeAAAA Type = 28
	TypeSRV  Type = 33
)

// Section is the message section a record arrived in.
type Section uint8

// Message sections, in wire order.
const (
	Answer Section = iota + 1
	Authority
	Additional
)

// Wire limits from RFC 1035 §2.3.4 and §4.1.4.
const (
	headerLen    = 12
	maxLabelLen  = 63
	maxNameLen   = 255
	pointerFlags = 0xC0
	pointerMask  = 0x3FFF
	pointerLen   = 2
	// maxPointers bounds compression-pointer hops. A legitimate name needs
	// one; the bound is what stops a pointer loop.
	maxPointers = 16
	classINET   = 1
	flagQR      = 0x8000
	// questionTail is QTYPE and QCLASS, after a question's name.
	questionTail = 4
)

var (
	errTruncated = errors.New("mdnswire: message truncated")
	errLabel     = errors.New("mdnswire: bad label")
	errPointer   = errors.New("mdnswire: bad compression pointer")
	errNameLen   = errors.New("mdnswire: name too long")
	errRData     = errors.New("mdnswire: record data does not match its type")
)

// Name is a domain name held as its labels, root excluded.
type Name []string

// String returns the presentation form: labels joined by dots, with a dot or
// backslash inside a label escaped by a backslash (RFC 1035 §5.1), so the
// result splits back into the same labels with ParseName.
func (n Name) String() string {
	var b strings.Builder
	for i, label := range n {
		if i > 0 {
			b.WriteByte('.')
		}
		for j := range len(label) {
			if c := label[j]; c == '.' || c == '\\' {
				b.WriteByte('\\')
			}
			b.WriteByte(label[j])
		}
	}
	return b.String()
}

// ParseName reverses String. A trailing root dot is accepted and dropped.
func ParseName(s string) (Name, error) {
	var (
		name  Name
		label []byte
	)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			if i+1 == len(s) {
				return nil, fmt.Errorf("%w: dangling escape in %q", errLabel, s)
			}
			i++
			label = append(label, s[i])
		case '.':
			if len(label) == 0 {
				return nil, fmt.Errorf("%w: empty label in %q", errLabel, s)
			}
			name = append(name, string(label))
			label = label[:0]
		default:
			label = append(label, c)
		}
	}
	if len(label) > 0 {
		name = append(name, string(label))
	}
	if len(name) == 0 {
		return nil, fmt.Errorf("%w: empty name", errLabel)
	}
	return name, nil
}

// Record is one resource record of a type discovery reads.
type Record struct {
	Section Section
	Name    Name
	Type    Type
	// Target is the PTR target or the SRV target host.
	Target Name
	Port   uint16
	TXT    []string
	// Addr is the A or AAAA address.
	Addr netip.Addr
}

// Message is a parsed mDNS message.
type Message struct {
	Response bool
	Records  []Record
}

// Parse decodes a message. Any malformed byte fails the whole message: the
// segment is untrusted, and a half-read packet is not evidence of anything.
func Parse(data []byte) (*Message, error) {
	if len(data) < headerLen {
		return nil, errTruncated
	}
	flags := binary.BigEndian.Uint16(data[2:])
	counts := [4]int{}
	for i := range counts {
		counts[i] = int(binary.BigEndian.Uint16(data[4+2*i:]))
	}

	msg := &Message{Response: flags&flagQR != 0}
	off := headerLen
	for range counts[0] {
		_, next, err := readName(data, off)
		if err != nil {
			return nil, err
		}
		if next+questionTail > len(data) {
			return nil, errTruncated
		}
		off = next + questionTail
	}
	for i, section := range []Section{Answer, Authority, Additional} {
		for range counts[i+1] {
			rec, next, keep, err := readRecord(data, off)
			if err != nil {
				return nil, err
			}
			off = next
			if keep {
				rec.Section = section
				msg.Records = append(msg.Records, rec)
			}
		}
	}
	return msg, nil
}

// readRecord decodes the record at off. keep is false for a type discovery
// does not read; its data is still bounds-checked and skipped.
func readRecord(data []byte, off int) (Record, int, bool, error) {
	owner, off, err := readName(data, off)
	if err != nil {
		return Record{}, 0, false, err
	}
	// TYPE, CLASS, TTL, RDLENGTH.
	const fixedLen = 10
	if off+fixedLen > len(data) {
		return Record{}, 0, false, errTruncated
	}
	rtype := Type(binary.BigEndian.Uint16(data[off:]))
	rdLen := int(binary.BigEndian.Uint16(data[off+8:]))
	start := off + fixedLen
	end := start + rdLen
	if end > len(data) {
		return Record{}, 0, false, errTruncated
	}

	rec := Record{Name: owner, Type: rtype}
	rdata := data[start:end]
	switch rtype {
	case TypeA:
		if rdLen != net.IPv4len {
			return Record{}, 0, false, errRData
		}
		rec.Addr = netip.AddrFrom4([4]byte(rdata))
	case TypeAAAA:
		if rdLen != net.IPv6len {
			return Record{}, 0, false, errRData
		}
		rec.Addr = netip.AddrFrom16([16]byte(rdata)).Unmap()
	case TypePTR:
		rec.Target, err = readNameIn(data, start, end)
	case TypeSRV:
		// Priority and weight precede the port; discovery reads neither.
		const srvFixedLen = 6
		if rdLen < srvFixedLen {
			return Record{}, 0, false, errRData
		}
		rec.Port = binary.BigEndian.Uint16(rdata[4:])
		rec.Target, err = readNameIn(data, start+srvFixedLen, end)
	case TypeTXT:
		rec.TXT, err = readStrings(rdata)
	default:
		return Record{}, end, false, nil
	}
	if err != nil {
		return Record{}, 0, false, err
	}
	return rec, end, true, nil
}

// readNameIn reads a name that must end exactly at end, the close of the
// record data holding it. Its pointers may still reach earlier in the message.
func readNameIn(data []byte, off, end int) (Name, error) {
	name, next, err := readName(data[:end], off)
	if err != nil {
		return nil, err
	}
	if next != end {
		return nil, errRData
	}
	return name, nil
}

// readName decodes the name at off, following compression pointers, and
// returns the offset just past it in the original position.
func readName(data []byte, off int) (Name, int, error) {
	var (
		name    Name
		wireLen int
		next    = -1
	)
	for hops := 0; hops <= maxPointers; {
		if off >= len(data) {
			return nil, 0, errTruncated
		}
		n := int(data[off])
		if n&pointerFlags != 0 {
			if next < 0 {
				next = off + pointerLen
			}
			target, err := readPointer(data, off)
			if err != nil {
				return nil, 0, err
			}
			off = target
			hops++
			continue
		}
		if n == 0 {
			if next < 0 {
				next = off + 1
			}
			return name, next, nil
		}
		// The terminating zero octet counts toward the limit too.
		if wireLen += n + 1; wireLen >= maxNameLen {
			return nil, 0, errNameLen
		}
		if off+1+n > len(data) {
			return nil, 0, errTruncated
		}
		name = append(name, string(data[off+1:off+1+n]))
		off += 1 + n
	}
	return nil, 0, errPointer
}

// readPointer returns the offset a compression pointer at off refers to.
func readPointer(data []byte, off int) (int, error) {
	if data[off]&pointerFlags != pointerFlags {
		// 0x40 and 0x80 are reserved label types.
		return 0, errLabel
	}
	if off+pointerLen > len(data) {
		return 0, errTruncated
	}
	return int(binary.BigEndian.Uint16(data[off:]) & pointerMask), nil
}

// readStrings splits TXT record data into its length-prefixed strings.
func readStrings(rdata []byte) ([]string, error) {
	var out []string
	for len(rdata) > 0 {
		n := int(rdata[0])
		if 1+n > len(rdata) {
			return nil, errRData
		}
		out = append(out, string(rdata[1:1+n]))
		rdata = rdata[1+n:]
	}
	return out, nil
}

// Query builds a one-question mDNS query: ID 0, no flags, class IN, the
// unicast-response bit clear (RFC 6762 §18).
func Query(name Name, qtype Type) ([]byte, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("%w: empty name", errLabel)
	}
	out := make([]byte, headerLen, headerLen+maxNameLen+questionTail)
	binary.BigEndian.PutUint16(out[4:], 1)
	wireLen := 1
	for _, label := range name {
		n := len(label)
		if n == 0 || n > maxLabelLen {
			return nil, fmt.Errorf("%w: %q", errLabel, label)
		}
		if wireLen += n + 1; wireLen > maxNameLen {
			return nil, errNameLen
		}
		out = append(out, uint8(n))
		out = append(out, label...)
	}
	out = append(out, 0)
	out = binary.BigEndian.AppendUint16(out, uint16(qtype))
	return binary.BigEndian.AppendUint16(out, classINET), nil
}
