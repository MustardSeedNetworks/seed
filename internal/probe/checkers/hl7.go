package checkers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultHL7Timeout is the per-attempt HL7 MLLP probe timeout.
const defaultHL7Timeout = 10 * time.Second

// defaultHL7Port is the default HL7 MLLP port (IANA 2575).
const defaultHL7Port = 2575

// MLLP framing bytes (HL7 v2 Minimum Lower Layer Protocol).
const (
	mllpStartByte = 0x0B // vertical tab — marks start of MLLP block
	mllpEndByte1  = 0x1C // file separator — first end byte
	mllpEndByte2  = 0x0D // carriage return — second end byte
)

// mllpWrapOverhead is the number of bytes MLLP framing adds per message
// (one start byte + two end bytes).
const mllpWrapOverhead = 3

// mllpReadBufSize bounds a single read of the raw TCP stream.
const mllpReadBufSize = 4096

// minHL7MSAFields is the minimum field count required to access MSA[1]
// (the ACK code).
const minHL7MSAFields = 2

// minHL7MSAErrorFields is the minimum field count to access MSA[3]
// (the error code).
const minHL7MSAErrorFields = 4

// HL7Params is the kind-specific params shape. All fields are optional;
// application / facility identifiers default to the values the legacy
// internal/api/handlers_medical_checks.go used.
type HL7Params struct {
	Port         int    `json:"port,omitempty"`               // default 2575
	SendingApp   string `json:"sending_app,omitempty"`        // default "SEED"
	SendingFac   string `json:"sending_facility,omitempty"`   // default "SEED_FAC"
	ReceivingApp string `json:"receiving_app,omitempty"`      // default "TARGET"
	ReceivingFac string `json:"receiving_facility,omitempty"` // default "TARGET_FAC"
	TimeoutMs    int    `json:"timeout_ms,omitempty"`         // default 10000
}

// HL7Checker implements probe.Checker for Kind="hl7". It opens a TCP
// connection to an HL7 MLLP endpoint, sends a minimal ADT^A01 admission
// message wrapped in MLLP framing, reads the ACK, and reports success
// when the MSA acknowledgment code is AA, CA, or absent (connection-only
// proof).
type HL7Checker struct {
	dialer PingDialer
}

// NewHL7Checker returns an HL7Checker wired to a real dialer.
func NewHL7Checker() *HL7Checker {
	return &HL7Checker{dialer: realPingDialer{}}
}

// WithHL7Dialer swaps the dialer (for tests).
func (c *HL7Checker) WithHL7Dialer(d PingDialer) *HL7Checker {
	c.dialer = d
	return c
}

// Kind returns probe.KindHL7.
func (c *HL7Checker) Kind() string { return probe.KindHL7 }

// RequiredCapabilities returns nil; HL7 MLLP needs no special hardware.
func (c *HL7Checker) RequiredCapabilities() []string { return nil }

// Run dials Target:Port, sends a minimal HL7 v2.5 ADT^A01 message in
// MLLP framing, reads the ACK, and maps the MSA acknowledgment code to
// success or failure.
//
// Success rule (mirrors legacy internal/api/handlers_medical_checks.go):
//   - "AA" or "CA" → success
//   - "" (no MSA segment / empty code) → success with synthesised ACK "OK"
//   - "AE" or "CE" → failure "Application error"
//   - "AR" or "CR" → failure "Application reject"
//   - anything else → failure "Unexpected ACK code: X"
//
//nolint:funlen // HL7 probing is one linear flow: validate, dial, build message, MLLP-frame, write, read, parse ACK.
func (c *HL7Checker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseHL7Params(p.Params)

	port := params.Port
	if port == 0 {
		port = defaultHL7Port
	}

	timeout := defaultHL7Timeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(p.Target, strconv.Itoa(port))
	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	if err != nil {
		return hl7Failure(p, time.Since(start), err.Error())
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	// Apply application / facility defaults used in the legacy stack.
	sendingApp := params.SendingApp
	if sendingApp == "" {
		sendingApp = "SEED"
	}
	sendingFac := params.SendingFac
	if sendingFac == "" {
		sendingFac = "SEED_FAC"
	}
	receivingApp := params.ReceivingApp
	if receivingApp == "" {
		receivingApp = "TARGET"
	}
	receivingFac := params.ReceivingFac
	if receivingFac == "" {
		receivingFac = "TARGET_FAC"
	}

	// Build a minimal HL7 v2.5 ADT^A01 admission notification for health
	// probing. ADT^A01 is more universally supported than QBP^Q21.
	ts := time.Now().Format("20060102150405")
	hl7Msg := fmt.Sprintf(
		"MSH|^~\\&|%s|%s|%s|%s|%s||ADT^A01|%s|P|2.5\r"+
			"EVN|A01|%s\r"+
			"PID|1||HEALTH_CHECK||SEED^PROBE|||U\r",
		sendingApp, sendingFac, receivingApp, receivingFac, ts, ts, ts,
	)

	mllpMsg := hl7WrapMLLP([]byte(hl7Msg))

	if _, writeErr := conn.Write(mllpMsg); writeErr != nil {
		return hl7Failure(p, time.Since(start), "write message: "+writeErr.Error())
	}

	response, readErr := hl7ReadMLLP(conn)
	if readErr != nil {
		return hl7Failure(p, time.Since(start), "read response: "+readErr.Error())
	}

	latencyMs := float64(time.Since(start).Milliseconds())

	ackCode, _ := hl7ParseACK(string(response))

	switch ackCode {
	case "AA", "CA":
		// Application Accept / Commit Accept — full success.
	case "AE", "CE":
		return hl7Failure(p, time.Duration(latencyMs)*time.Millisecond, "Application error")
	case "AR", "CR":
		return hl7Failure(p, time.Duration(latencyMs)*time.Millisecond, "Application reject")
	case "":
		// No MSA segment found but a response was received — connection
		// proof is sufficient; surface a synthetic ACK code.
		ackCode = "OK"
	default:
		return hl7Failure(p, time.Duration(latencyMs)*time.Millisecond,
			fmt.Sprintf("Unexpected ACK code: %s", ackCode))
	}

	meta, _ := json.Marshal(map[string]any{
		metaKeyAddr: addr,
		"ack_code":  ackCode,
	})

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// hl7Failure builds a failed Result with the measured latency so far.
func hl7Failure(p probe.Probe, elapsed time.Duration, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   false,
		LatencyMs: float64(elapsed.Milliseconds()),
		Error:     msg,
	}
}

// hl7WrapMLLP wraps msg in MLLP framing: 0x0B + payload + 0x1C 0x0D.
func hl7WrapMLLP(msg []byte) []byte {
	out := make([]byte, 0, len(msg)+mllpWrapOverhead)
	out = append(out, mllpStartByte)
	out = append(out, msg...)
	out = append(out, mllpEndByte1, mllpEndByte2)
	return out
}

// hl7ReadMLLP reads one MLLP-framed message from conn. It discards
// bytes before the start byte (0x0B) and stops at the end sequence
// (0x1C 0x0D). An EOF with accumulated data is treated as a complete
// message.
//
//nolint:gocognit // MLLP framing requires byte-by-byte state machine parsing.
func hl7ReadMLLP(conn net.Conn) ([]byte, error) {
	var buf bytes.Buffer
	readBuf := make([]byte, mllpReadBufSize)
	foundStart := false

	for {
		n, err := conn.Read(readBuf)
		if err != nil {
			if errors.Is(err, io.EOF) && buf.Len() > 0 {
				break
			}
			return nil, err
		}

		for i := range n {
			b := readBuf[i]
			if !foundStart {
				if b == mllpStartByte {
					foundStart = true
				}
				continue
			}

			// End sequence: 0x1C followed by 0x0D.
			if b == mllpEndByte1 && i+1 < n && readBuf[i+1] == mllpEndByte2 {
				return buf.Bytes(), nil
			}
			// End byte 1 without a lookahead — treat as end; the next read
			// would deliver end byte 2 (or the stream is over).
			if b == mllpEndByte1 && buf.Len() > 0 {
				return buf.Bytes(), nil
			}

			buf.WriteByte(b)
		}
	}

	return buf.Bytes(), nil
}

// hl7ParseACK extracts the acknowledgment code and error code from the
// MSA segment of an HL7 message. Returns empty strings when no MSA
// segment is found.
func hl7ParseACK(msg string) (string, string) {
	for line := range strings.SplitSeq(msg, "\r") {
		if !strings.HasPrefix(line, "MSA|") {
			continue
		}
		fields := strings.Split(line, "|")
		var ackCode, errorCode string
		if len(fields) >= minHL7MSAFields {
			ackCode = fields[1]
		}
		if len(fields) >= minHL7MSAErrorFields {
			errorCode = fields[3]
		}
		return ackCode, errorCode
	}
	return "", ""
}

// parseHL7Params decodes the params JSON; returns zero value on empty
// or unparseable input.
func parseHL7Params(raw json.RawMessage) HL7Params {
	if len(raw) == 0 {
		return HL7Params{}
	}
	var p HL7Params
	_ = json.Unmarshal(raw, &p)
	return p
}
