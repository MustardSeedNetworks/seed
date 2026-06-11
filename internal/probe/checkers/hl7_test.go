package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// hl7FakeConn delivers canned MLLP-framed response bytes and records
// what the checker wrote.
type hl7FakeConn struct {
	mu       sync.Mutex
	response []byte
	written  []byte
	pos      int
}

func (c *hl7FakeConn) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pos >= len(c.response) {
		return 0, errors.New("hl7FakeConn: response exhausted")
	}
	n := copy(b, c.response[c.pos:])
	c.pos += n
	return n, nil
}

func (c *hl7FakeConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.written = append(c.written, b...)
	return len(b), nil
}

func (*hl7FakeConn) Close() error                     { return nil }
func (*hl7FakeConn) LocalAddr() net.Addr              { return nil }
func (*hl7FakeConn) RemoteAddr() net.Addr             { return nil }
func (*hl7FakeConn) SetDeadline(time.Time) error      { return nil }
func (*hl7FakeConn) SetReadDeadline(time.Time) error  { return nil }
func (*hl7FakeConn) SetWriteDeadline(time.Time) error { return nil }

// hl7Dialer is a test seam that returns either a fixed conn or error.
type hl7Dialer struct {
	conn net.Conn
	err  error
}

func (d *hl7Dialer) Dial(_ context.Context, _, _ string) (net.Conn, error) {
	return d.conn, d.err
}

// makeHL7MLLPResponse builds an MLLP-framed HL7 ACK payload containing
// the given MSA acknowledgment code. The structure mirrors what a real
// HL7 engine would return: 0x0B + HL7 segments + 0x1C 0x0D.
func makeHL7MLLPResponse(ackCode string) []byte {
	ts := time.Now().Format("20060102150405")
	hl7 := fmt.Sprintf(
		"MSH|^~\\&|TARGET|TARGET_FAC|SEED|SEED_FAC|%s||ACK^A01|%s|P|2.5\r"+
			"MSA|%s|%s\r",
		ts, ts, ackCode, ts,
	)
	// MLLP frame: 0x0B + payload + 0x1C 0x0D
	out := make([]byte, 0, 1+len(hl7)+2)
	out = append(out, 0x0B)
	out = append(out, []byte(hl7)...)
	out = append(out, 0x1C, 0x0D)
	return out
}

func TestHL7Checker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewHL7Checker().Kind() != "hl7" {
		t.Errorf("Kind() != hl7")
	}
}

func TestHL7Checker_Run_SuccessAA(t *testing.T) {
	t.Parallel()
	conn := &hl7FakeConn{response: makeHL7MLLPResponse("AA")}
	c := checkers.NewHL7Checker().WithHL7Dialer(&hl7Dialer{conn: conn})

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "hl7",
		Target: "hl7.example.com",
	})

	if !r.Success {
		t.Fatalf("Success = false on AA ACK: %s", r.Error)
	}

	// The first byte written must be the MLLP start byte 0x0B.
	conn.mu.Lock()
	written := conn.written
	conn.mu.Unlock()
	if len(written) == 0 || written[0] != 0x0B {
		t.Errorf("first written byte = 0x%02X, want 0x0B (MLLP start)", written[0])
	}

	// Metadata must carry ack_code.
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata not JSON: %v", err)
	}
	if got, ok := meta["ack_code"].(string); !ok || got != "AA" {
		t.Errorf("ack_code = %v, want AA", meta["ack_code"])
	}
}

func TestHL7Checker_Run_FailAR(t *testing.T) {
	t.Parallel()
	conn := &hl7FakeConn{response: makeHL7MLLPResponse("AR")}
	c := checkers.NewHL7Checker().WithHL7Dialer(&hl7Dialer{conn: conn})

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "hl7",
		Target: "hl7.example.com",
	})

	if r.Success {
		t.Error("Success = true on AR ACK; want false")
	}
	if r.Error == "" {
		t.Error("expected a non-empty Error on AR ACK")
	}
}

func TestHL7Checker_Run_DialError(t *testing.T) {
	t.Parallel()
	c := checkers.NewHL7Checker().WithHL7Dialer(&hl7Dialer{err: errors.New("connection refused")})

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "hl7",
		Target: "down.example.com",
	})

	if r.Success {
		t.Error("Success = true on dial error; want false")
	}
	if r.Error == "" {
		t.Error("expected a non-empty Error on dial error")
	}
}

func TestHL7Checker_Run_EmptyACKSucceeds(t *testing.T) {
	t.Parallel()
	// A response with no MSA segment produces an empty ACK code, which the
	// checker maps to success with synthesised ACK code "OK".
	noMSA := []byte{0x0B}
	noMSA = append(noMSA, []byte("MSH|^~\\&|TARGET||SEED||20240101||ACK|1|P|2.5\r")...)
	noMSA = append(noMSA, 0x1C, 0x0D)

	conn := &hl7FakeConn{response: noMSA}
	c := checkers.NewHL7Checker().WithHL7Dialer(&hl7Dialer{conn: conn})

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "hl7",
		Target: "hl7.example.com",
	})

	if !r.Success {
		t.Fatalf("Success = false on no-MSA response: %s", r.Error)
	}
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata not JSON: %v", err)
	}
	if got, ok := meta["ack_code"].(string); !ok || got != "OK" {
		t.Errorf("ack_code = %v, want OK", meta["ack_code"])
	}
}
