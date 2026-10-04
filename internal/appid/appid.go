// Package appid names the application a flow carries from its IP protocol
// and transport ports, against a signature table the operator can edit.
//
// This is a heuristic and says so: a flow on TCP 443 is reported as https
// because that is what the port conventionally carries, not because any
// payload was inspected. A flow that matches no signature is [Unknown];
// the classifier never guesses from proximity or traffic shape.
//
// When both ports of a flow match different signatures, the lower port
// wins. Servers listen on the lower, registered ports and clients send
// from the higher, ephemeral range, so the lower port is the service.
package appid

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Unknown is the application of a flow no signature matches.
const Unknown = "unknown"

// Table bounds. A signature table is operator input stored in the database
// and walked for every flow, so it is kept small enough to review by eye.
const (
	maxSignatures        = 256
	maxPortsPerSignature = 64
)

// Transport protocols whose flows carry ports. Every other protocol is
// matched on its number alone.
const (
	protoTCP  = 6
	protoUDP  = 17
	protoSCTP = 132
)

//go:embed signatures.json
var builtinJSON []byte

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// Signature names an application by protocol and, for TCP, UDP and SCTP,
// by port. Ports are single ports ("443") or inclusive ranges
// ("6000-6063"). A signature for any other protocol has no ports and
// matches the whole protocol.
type Signature struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Protocol    uint8    `json:"protocol"`
	Ports       []string `json:"ports,omitempty"`
}

// span is one parsed port range and the signature it belongs to.
type span struct {
	first, last uint16
	sig         int
}

// Table is a validated signature table. It is immutable once built.
type Table struct {
	signatures []Signature
	// ports holds each port-carrying protocol's spans sorted by first
	// port; validation guarantees they do not overlap.
	ports map[uint8][]span
	// whole maps a portless protocol to its signature.
	whole map[uint8]int
}

// Builtin returns the signature table Seed ships with.
func Builtin() *Table {
	t, err := Parse(builtinJSON)
	if err != nil {
		panic("appid: builtin signatures invalid: " + err.Error())
	}
	return t
}

// document is the JSON shape of a stored or submitted table.
type document struct {
	Signatures []Signature `json:"signatures"`
}

// Parse decodes and validates a table in its JSON form.
func Parse(data []byte) (*Table, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var doc document
	if err := dec.Decode(&doc); err != nil {
		return nil, errors.New("signature table is not valid JSON of the documented shape")
	}
	return New(doc.Signatures)
}

// New validates signatures and builds a table from them. Two signatures
// may share a name (DNS over UDP and over TCP), but no port of a protocol
// may belong to two signatures: the table has one answer for every flow.
//
// Errors identify a signature by its position and never quote what the
// caller sent: the table is operator input, and its text must not reach a
// log line or a response verbatim.
func New(signatures []Signature) (*Table, error) {
	if len(signatures) == 0 {
		return nil, errors.New("signature table is empty")
	}
	if len(signatures) > maxSignatures {
		return nil, fmt.Errorf("signature table has %d signatures, limit %d", len(signatures), maxSignatures)
	}
	t := &Table{
		signatures: slices.Clone(signatures),
		ports:      map[uint8][]span{},
		whole:      map[uint8]int{},
	}
	for i, sig := range t.signatures {
		if err := t.add(i, sig); err != nil {
			return nil, fmt.Errorf("signature %d: %w", i, err)
		}
	}
	for proto, spans := range t.ports {
		if _, ok := t.whole[proto]; ok {
			return nil, fmt.Errorf("protocol %d has both a whole-protocol signature and port signatures", proto)
		}
		slices.SortFunc(spans, func(a, b span) int { return int(a.first) - int(b.first) })
		for i := 1; i < len(spans); i++ {
			if spans[i].first <= spans[i-1].last {
				return nil, fmt.Errorf("protocol %d port %d is claimed by both signature %d and signature %d",
					proto, spans[i].first, spans[i-1].sig, spans[i].sig)
			}
		}
	}
	return t, nil
}

func (t *Table) add(i int, sig Signature) error {
	if !namePattern.MatchString(sig.Name) {
		return errors.New("name must be 1-32 lowercase letters, digits or hyphens")
	}
	if sig.Name == Unknown {
		return fmt.Errorf("name %q is reserved", Unknown)
	}
	if sig.Protocol == 0 {
		return errors.New("protocol must be an IP protocol number from 1 to 255")
	}
	if !carriesPorts(sig.Protocol) {
		if len(sig.Ports) > 0 {
			return fmt.Errorf("protocol %d has no ports", sig.Protocol)
		}
		if _, dup := t.whole[sig.Protocol]; dup {
			return fmt.Errorf("protocol %d already has a signature", sig.Protocol)
		}
		t.whole[sig.Protocol] = i
		return nil
	}
	if len(sig.Ports) == 0 {
		return fmt.Errorf("protocol %d needs at least one port", sig.Protocol)
	}
	if len(sig.Ports) > maxPortsPerSignature {
		return fmt.Errorf("%d port entries, limit %d", len(sig.Ports), maxPortsPerSignature)
	}
	for j, p := range sig.Ports {
		first, last, err := parsePorts(p)
		if err != nil {
			return fmt.Errorf("port entry %d: %w", j, err)
		}
		t.ports[sig.Protocol] = append(t.ports[sig.Protocol], span{first: first, last: last, sig: i})
	}
	return nil
}

func carriesPorts(proto uint8) bool {
	return proto == protoTCP || proto == protoUDP || proto == protoSCTP
}

// parsePorts reads "443" or "6000-6063". Port 0 is not a service port.
func parsePorts(s string) (uint16, uint16, error) {
	lo, hi, isRange := strings.Cut(s, "-")
	first, err := parsePort(lo)
	if err != nil {
		return 0, 0, err
	}
	if !isRange {
		return first, first, nil
	}
	last, err := parsePort(hi)
	if err != nil {
		return 0, 0, err
	}
	if last < first {
		return 0, 0, fmt.Errorf("range %d-%d runs backwards", first, last)
	}
	return first, last, nil
}

func parsePort(s string) (uint16, error) {
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil || n == 0 {
		return 0, errors.New("must be a number from 1 to 65535")
	}
	return uint16(n), nil
}

// Signatures returns the table's signatures in the order they were given.
func (t *Table) Signatures() []Signature {
	return slices.Clone(t.signatures)
}

// MarshalJSON writes the table in the form [Parse] reads.
func (t *Table) MarshalJSON() ([]byte, error) {
	return json.Marshal(document{Signatures: t.signatures})
}

// Classify names the application of a flow, or returns [Unknown].
func (t *Table) Classify(protocol uint8, srcPort, dstPort uint16) string {
	if sig, ok := t.whole[protocol]; ok {
		return t.signatures[sig].Name
	}
	spans := t.ports[protocol]
	if spans == nil {
		return Unknown
	}
	lo, hi := min(srcPort, dstPort), max(srcPort, dstPort)
	if sig, ok := lookup(spans, lo); ok {
		return t.signatures[sig].Name
	}
	if sig, ok := lookup(spans, hi); ok {
		return t.signatures[sig].Name
	}
	return Unknown
}

func lookup(spans []span, port uint16) (int, bool) {
	if port == 0 {
		return 0, false
	}
	i, found := slices.BinarySearchFunc(spans, port, func(s span, p uint16) int {
		switch {
		case s.last < p:
			return -1
		case s.first > p:
			return 1
		}
		return 0
	})
	if !found {
		return 0, false
	}
	return spans[i].sig, true
}
