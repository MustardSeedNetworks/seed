// Package snmpwalkfile replays recorded SNMP walks as an [snmp.Client].
//
// The collectors are otherwise tested against hand-written fakes, which return
// whatever the test author believed a device emits. A recorded walk returns
// what a device actually emitted — including the empty tables, the truncated
// strings, and the vendors who put a Counter32 where the MIB says Counter64.
// Real walks disagree with our assumptions, which is the entire point of using
// them.
//
// The format is `snmpwalk -On` output, one varbind per line:
//
//	.1.3.6.1.2.1.1.5.0 = STRING: switch-01
//	.1.3.6.1.2.1.2.2.1.2.1 = STRING: GigabitEthernet1/0/1
//	.1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000000000
//
// Lines starting with '#' are comments; blank lines are ignored.
package snmpwalkfile

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// maxWalkLineLen bounds a single line. Some agents return multi-kilobyte
// sysDescr values, and [bufio.Scanner]'s default would reject them.
const maxWalkLineLen = 1024 * 1024

// walkScanBufSize is the scanner's starting buffer. Most lines are well under
// it; maxWalkLineLen is what actually bounds growth.
const walkScanBufSize = 64 * 1024

// Walk is a recorded walk, held in OID order so [Client.Walk] can return a
// subtree without re-sorting on every call.
type Walk struct {
	oids     []string
	byOID    map[string]snmp.Varbind
	unparsed int
}

// Open reads a walk file from disk.
func Open(path string) (*Walk, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open walk %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	return Parse(file)
}

// Parse reads a walk from r.
//
// Unparseable lines are counted rather than rejected: real captures contain
// continuation lines from multi-line strings and the occasional agent quirk,
// and refusing to load a whole device because of one odd line would mean
// testing against only the tidy devices. [Walk.Unparsed] reports the count so a
// test can assert it stays small.
func Parse(r io.Reader) (*Walk, error) {
	w := &Walk{byOID: make(map[string]snmp.Varbind)}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, walkScanBufSize), maxWalkLineLen)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		oid, value, ok := parseLine(line)
		if !ok {
			w.unparsed++
			continue
		}
		if _, seen := w.byOID[oid]; !seen {
			w.oids = append(w.oids, oid)
		}
		w.byOID[oid] = snmp.Varbind{OID: oid, Value: value}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read walk: %w", err)
	}
	if len(w.oids) == 0 {
		return nil, fmt.Errorf("walk contains no varbinds (%d unparsed lines)", w.unparsed)
	}

	sort.Slice(w.oids, func(i, j int) bool {
		return compareOID(w.oids[i], w.oids[j]) < 0
	})
	return w, nil
}

// Len reports how many varbinds the walk holds.
func (w *Walk) Len() int { return len(w.oids) }

// Unparsed reports how many lines could not be read as varbinds.
func (w *Walk) Unparsed() int { return w.unparsed }

// Client returns an [snmp.Client] served entirely from this walk.
func (w *Walk) Client() snmp.Client { return &client{walk: w} }

// parseLine splits one `.OID = TYPE: value` line.
func parseLine(line string) (string, any, bool) {
	oid, rest, found := strings.Cut(line, " = ")
	if !found || !strings.HasPrefix(oid, ".") {
		return "", nil, false
	}
	oid = strings.TrimPrefix(strings.TrimSpace(oid), ".")

	typeName, raw, found := strings.Cut(rest, ": ")
	if !found {
		// "OID = \"\"" and bare end-of-mib markers have no type.
		return oid, "", true
	}
	return oid, decodeValue(strings.TrimSpace(typeName), raw), true
}

// decodeValue converts a walk's textual value into the Go type gosnmp would
// have produced, so a collector cannot pass here and fail against a live agent
// on a type assertion.
func decodeValue(typeName, raw string) any {
	raw = strings.TrimSpace(raw)
	switch typeName {
	case "INTEGER":
		// 32, not 64: SMIv2 INTEGER is Integer32, and parsing wider then
		// narrowing to int would truncate on a 32-bit build (CodeQL
		// go/incorrect-integer-conversion). gosnmp decodes this as int.
		if n, err := strconv.ParseInt(firstField(raw), 10, 32); err == nil {
			return int(n)
		}
	case "Gauge32", "Counter32", "UInteger32":
		if n, err := strconv.ParseUint(firstField(raw), 10, 32); err == nil {
			return uint(n)
		}
	case "Counter64":
		if n, err := strconv.ParseUint(firstField(raw), 10, 64); err == nil {
			return n
		}
	case "Timeticks":
		// "(123456789) 14 days, 6:56:07.89" — the parenthesised ticks are the
		// value; the rest is snmpwalk's rendering for humans.
		if open := strings.Index(raw, "("); open >= 0 {
			if shut := strings.Index(raw[open:], ")"); shut > 0 {
				if n, err := strconv.ParseUint(raw[open+1:open+shut], 10, 32); err == nil {
					return uint(n)
				}
			}
		}
	case "Hex-STRING":
		return decodeHexString(raw)
	case "OID":
		return strings.TrimPrefix(raw, ".")
	case "STRING":
		return strings.Trim(raw, `"`)
	}
	return strings.Trim(raw, `"`)
}

// firstField returns the leading whitespace-delimited token, which is where the
// numeric value sits when snmpwalk appends a unit or an enum label.
func firstField(s string) string {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

// decodeHexString converts "00 1A 2B ..." into bytes. MAC addresses and
// physical port IDs arrive this way.
func decodeHexString(raw string) []byte {
	fields := strings.Fields(raw)
	out := make([]byte, 0, len(fields))
	for _, f := range fields {
		b, err := strconv.ParseUint(f, 16, 8)
		if err != nil {
			return []byte(raw)
		}
		out = append(out, byte(b))
	}
	return out
}

// compareOID orders two dotted OIDs numerically. Lexical ordering is wrong —
// it puts ".10" before ".2" — and a Walk that returns rows out of order would
// hide exactly the ordering bugs these fixtures exist to catch.
func compareOID(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		an, aerr := strconv.Atoi(as[i])
		bn, berr := strconv.Atoi(bs[i])
		if aerr != nil || berr != nil {
			if c := strings.Compare(as[i], bs[i]); c != 0 {
				return c
			}
			continue
		}
		if an != bn {
			if an < bn {
				return -1
			}
			return 1
		}
	}
	return len(as) - len(bs)
}

// client serves an [snmp.Client] from a Walk.
type client struct{ walk *Walk }

// Get returns one varbind per requested OID, in request order. A missing OID
// yields a nil Value rather than an error, matching the interface contract and
// what a real agent does for noSuchObject.
func (c *client) Get(ctx context.Context, oids []string) ([]snmp.Varbind, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]snmp.Varbind, 0, len(oids))
	for _, oid := range oids {
		key := strings.TrimPrefix(oid, ".")
		if vb, ok := c.walk.byOID[key]; ok {
			out = append(out, snmp.Varbind{OID: oid, Value: vb.Value})
			continue
		}
		out = append(out, snmp.Varbind{OID: oid})
	}
	return out, nil
}

// Walk returns every varbind under prefix, in OID order. An unimplemented
// subtree returns no rows and no error, which is what a real agent does and
// what the collector chain relies on to survive a device lacking a MIB.
func (c *client) Walk(ctx context.Context, prefix string) ([]snmp.Varbind, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := strings.TrimPrefix(prefix, ".")
	var out []snmp.Varbind
	for _, oid := range c.walk.oids {
		if oid == root || strings.HasPrefix(oid, root+".") {
			out = append(out, c.walk.byOID[oid])
		}
	}
	return out, nil
}
