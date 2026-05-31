// Package syslog implements a passive UDP syslog listener (RFC 3164
// BSD-style + RFC 5424). Bind defaults to ":514" but operators can
// remap to a non-privileged port for unprivileged deployments.
//
// The listener parses inbound packets, extracts the PRI byte's
// facility + severity, the timestamp (if present), the hostname/
// tag, and the message body, then publishes a typed [listener.Event]
// via the configured [listener.Sink].
//
// Out of scope for V1.0:
//   - TLS over TCP (RFC 5425, port 6514) — separate Listener type.
//   - Forwarding events to a sibling Seed in the cluster — Stage B.
//   - Structured-data parsing for RFC 5424 (SD-ELEMENT). The raw
//     payload is preserved so a Stage A4 enrichment pass can decode
//     it later without re-receiving the packet.
//
// Internally the listener uses [net.PacketConn] for UDP framing.
package syslog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/listener"
)

// Name is the listener key used in observability + the engine
// registry.
const Name = "syslog-udp"

// Tunables. Values picked for V1.0 default deployment; the
// integration test seam (NewWithConn) lets advanced operators
// inject a pre-tuned [net.PacketConn].
const (
	defaultBindAddr     = ":514"
	defaultReadBufBytes = 65535           // syslog frames over UDP cap at the 64KiB datagram size
	defaultReadTimeout  = 1 * time.Second // re-arms the read loop so Stop responds promptly
	maxPriorityValue    = 191             // RFC 3164 §4.1.1
	severityMask        = 0x07            // PRI = facility<<3 | severity
	severityBits        = 3               // PRI = facility<<3 | severity

	// SeverityInformational is the fallback severity when a frame
	// has no PRI byte (RFC 3164 §4.1.1 advice). Operators reading
	// the event log get a stable "this had no PRI" value instead
	// of severity 0 ("emergency") spuriously firing alerts.
	SeverityInformational = severityInformationalValue
)

// Severity values from the PRI byte's low 3 bits (RFC 3164 §4.1.1).
// Exported so alert rules can compare against the named values
// rather than magic numbers.
const (
	SeverityEmergency = iota
	SeverityAlert
	SeverityCritical
	SeverityError
	SeverityWarning
	SeverityNotice
	// SeverityInformational is also the no-PRI fallback.
	severityInformationalValue
	SeverityDebug
)

// severityNameByValue returns the canonical RFC 3164 / 5424 string
// for one severity bits value (0..7). Out-of-range falls back to
// the no-PRI default (informational).
func severityNameByValue(s int) string {
	switch s {
	case SeverityEmergency:
		return "emergency"
	case SeverityAlert:
		return "alert"
	case SeverityCritical:
		return "critical"
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityNotice:
		return "notice"
	case severityInformationalValue:
		return "informational"
	case SeverityDebug:
		return "debug"
	default:
		return "informational"
	}
}

// Listener is one bound UDP socket parsing syslog frames.
type Listener struct {
	bindAddr string
	sink     listener.Sink
	logger   *slog.Logger
	now      func() time.Time

	mu      sync.Mutex
	conn    net.PacketConn
	started bool
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

// Config configures one Listener. BindAddr defaults to ":514"
// (operator may remap via SEED_SYSLOG_BIND). Logger + Now default
// to [slog.Default] / [time.Now] in UTC.
type Config struct {
	BindAddr string
	Sink     listener.Sink
	Logger   *slog.Logger
	Now      func() time.Time
}

// New returns an unstarted syslog Listener. Logger defaults to
// [slog.Default] when nil; Now defaults to [time.Now] in UTC.
func New(cfg Config) (*Listener, error) {
	if cfg.Sink == nil {
		return nil, errors.New("syslog: Sink required")
	}
	if cfg.BindAddr == "" {
		cfg.BindAddr = defaultBindAddr
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Listener{
		bindAddr: cfg.BindAddr,
		sink:     cfg.Sink,
		logger:   cfg.Logger,
		now:      cfg.Now,
	}, nil
}

// Name implements [listener.Listener].
func (*Listener) Name() string { return Name }

// Start binds the UDP socket and dispatches the read loop. A second
// Start without an intervening Stop is a no-op.
func (l *Listener) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		return nil
	}

	lc := &net.ListenConfig{}
	conn, err := lc.ListenPacket(ctx, "udp", l.bindAddr)
	if err != nil {
		return fmt.Errorf("syslog: bind %s: %w", l.bindAddr, err)
	}
	loopCtx, cancel := context.WithCancel(ctx)
	l.conn = conn
	l.cancel = cancel
	l.started = true

	l.wg.Add(1)
	go l.run(loopCtx)
	l.logger.InfoContext(ctx, "syslog listener started", "addr", l.bindAddr)
	return nil
}

// Stop closes the socket and waits up to ctx deadline for the read
// loop to drain.
func (l *Listener) Stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return nil
	}
	l.started = false
	if l.cancel != nil {
		l.cancel()
	}
	conn := l.conn
	l.conn = nil
	l.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	doneCh := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	l.logger.InfoContext(ctx, "syslog listener stopped", "addr", l.bindAddr)
	return nil
}

// run is the read loop. Honors ctx cancellation between reads (the
// SetReadDeadline call rearms every defaultReadTimeout so a cancel
// during a quiet period is observed promptly).
func (l *Listener) run(ctx context.Context) {
	defer l.wg.Done()
	buf := make([]byte, defaultReadBufBytes)
	for {
		if ctx.Err() != nil {
			return
		}
		conn := l.connSnapshot()
		if conn == nil {
			return
		}
		if !l.readOnce(ctx, conn, buf) {
			return
		}
	}
}

// readOnce performs one ReadFrom pass. Returns false when the loop
// should exit (conn closed, ctx cancelled); true to keep iterating
// (including on timeouts and transient read errors).
func (l *Listener) readOnce(ctx context.Context, conn net.PacketConn, buf []byte) bool {
	_ = conn.SetReadDeadline(time.Now().Add(defaultReadTimeout))
	n, addr, err := conn.ReadFrom(buf)
	if err != nil {
		if isTimeout(err) {
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		l.logger.WarnContext(ctx, "syslog read error", "error", err)
		return true
	}
	l.handle(ctx, addr, buf[:n])
	return true
}

func (l *Listener) connSnapshot() net.PacketConn {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.conn
}

// handle parses one UDP datagram and publishes the resulting Event.
func (l *Listener) handle(ctx context.Context, addr net.Addr, frame []byte) {
	parsed := Parse(frame)
	payload, err := json.Marshal(parsed)
	if err != nil {
		l.logger.WarnContext(ctx, "syslog: marshal payload failed", "error", err)
		return
	}
	evt := listener.Event{
		Kind:       Name,
		SourceAddr: addr.String(),
		Severity:   parsed.SeverityName,
		Timestamp:  l.now(),
		Payload:    json.RawMessage(payload),
	}
	if pubErr := l.sink.Publish(ctx, evt); pubErr != nil {
		l.logger.WarnContext(ctx, "syslog: sink publish failed",
			"source", evt.SourceAddr, "error", pubErr)
	}
}

func isTimeout(err error) bool {
	var nerr net.Error
	return errors.As(err, &nerr) && nerr.Timeout()
}

// Parsed carries the structured fields the syslog parser extracted
// from a raw datagram. Raw preserves the original message verbatim
// so downstream consumers can re-parse against richer schemas (e.g.
// RFC 5424 SD-ELEMENT, vendor-specific TAG patterns) without
// re-receiving the packet.
type Parsed struct {
	Facility     int    `json:"facility"`
	Severity     int    `json:"severity"`
	SeverityName string `json:"severityName"`
	Tag          string `json:"tag,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
	Message      string `json:"message"`
	Raw          string `json:"raw"`
}

// Parse decodes a syslog frame using the common BSD-style and
// RFC 5424 heuristics. Frames without a parseable PRI fall back to
// [SeverityInformational] (RFC 3164's recommendation) with the
// original message preserved in Raw + Message.
func Parse(frame []byte) Parsed {
	raw := string(frame)
	p := Parsed{Raw: raw, Severity: SeverityInformational}

	// PRI is "<NNN>" leading the frame. RFC 3164 §4.1.1: 1-3 digits,
	// max value 191.
	rest := raw
	hasPri := false
	if strings.HasPrefix(rest, "<") {
		if end := strings.IndexByte(rest, '>'); end > 1 && end <= 4 {
			if pri, err := strconv.Atoi(rest[1:end]); err == nil && pri >= 0 && pri <= maxPriorityValue {
				p.Facility = pri >> severityBits
				p.Severity = pri & severityMask
				rest = rest[end+1:]
				hasPri = true
			}
		}
	}
	if !hasPri {
		p.Severity = SeverityInformational
	}
	p.SeverityName = severityName(p.Severity)

	// Heuristic split: "HEADER TAG: MSG" -> tag = field before ":"
	// in the first line. Hostname is the first whitespace-delimited
	// token after the timestamp (or PRI). Pure BSD-style is fine
	// because RFC 5424 frames also start with this token pattern.
	rest = strings.TrimSpace(rest)
	if colon := strings.IndexByte(rest, ':'); colon > 0 && colon < 128 {
		header := rest[:colon]
		fields := strings.Fields(header)
		if len(fields) > 0 {
			p.Tag = fields[len(fields)-1]
			if len(fields) > 1 {
				p.Hostname = fields[len(fields)-2]
			}
		}
		p.Message = strings.TrimSpace(rest[colon+1:])
	} else {
		p.Message = rest
	}
	return p
}

func severityName(s int) string { return severityNameByValue(s) }
