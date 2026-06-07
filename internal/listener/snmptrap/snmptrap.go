// Package snmptrap is the passive SNMP trap + inform receiver.
// Binds UDP/162 (operator may remap) and decodes incoming SNMPv1
// + SNMPv2c trap PDUs via gosnmp's TrapListener; SNMPv3 traps with
// USM are explicitly out of V1.0 scope (gosnmp's own trap code is
// flagged unreliable for v3) and will land in a follow-up.
//
// Each trap is normalized into a [Parsed] struct (community,
// trap OID, varbinds, uptime) and published as a
// [listener.Event] kind="snmp-trap-v2c" through the configured
// [listener.Sink].
package snmptrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/listener"
)

// Name is the listener key used in observability + the engine
// registry. SNMPv3 traps land here under the same name once
// they're supported.
const Name = "snmp-trap-v2c"

const (
	defaultBindAddr = ":162"

	// startupListenWait is how long Start blocks on the
	// TrapListener.Listening channel before declaring a bind
	// failure. gosnmp signals readiness almost immediately on a
	// successful bind; 2s leaves headroom for slow CI hosts.
	startupListenWait = 2 * time.Second
)

// Listener is one bound UDP/162 socket decoding incoming traps.
type Listener struct {
	bindAddr string
	sink     listener.Sink
	logger   *slog.Logger
	now      func() time.Time

	mu      sync.Mutex
	tl      *gosnmp.TrapListener
	started bool
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

// Config wires the listener. BindAddr defaults to ":162". Sink is
// required. Logger / Now default to [slog.Default] + [time.Now] UTC.
type Config struct {
	BindAddr string
	Sink     listener.Sink
	Logger   *slog.Logger
	Now      func() time.Time
}

// New returns an unstarted trap Listener.
func New(cfg Config) (*Listener, error) {
	if cfg.Sink == nil {
		return nil, errors.New("snmptrap: Sink required")
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

// Name returns the listener key. Implements [listener.Listener] +
// [engine.Engine].
func (*Listener) Name() string { return Name }

// Start binds UDP/162 and spawns the gosnmp.TrapListener goroutine.
// Idempotent.
func (l *Listener) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		return nil
	}

	tl := gosnmp.NewTrapListener()
	tl.Params = gosnmp.Default
	tl.OnNewTrap = l.handle
	l.tl = tl
	l.started = true

	listenCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	l.wg.Add(1)
	go l.runListen(listenCtx)

	// Block briefly to surface bind failures synchronously — without
	// this, a port conflict would surface only in a background log.
	select {
	case <-tl.Listening():
		l.logger.InfoContext(ctx, "snmp trap listener started", "addr", l.bindAddr)
		return nil
	case <-time.After(startupListenWait):
		// Listening signal never arrived. Treat as bind failure;
		// Close the listener so the goroutine exits.
		tl.Close()
		l.started = false
		l.tl = nil
		cancel()
		l.wg.Wait()
		return fmt.Errorf("snmptrap: bind %s timed out", l.bindAddr)
	case <-listenCtx.Done():
		return listenCtx.Err()
	}
}

// runListen owns the blocking gosnmp.TrapListener.Listen call. Any
// error other than the expected close-during-shutdown surfaces in
// the logs because gosnmp.Listen blocks until Close is invoked.
func (l *Listener) runListen(ctx context.Context) {
	defer l.wg.Done()
	if err := l.tl.Listen(l.bindAddr); err != nil {
		if ctx.Err() == nil {
			l.logger.WarnContext(ctx, "snmp trap listen returned",
				"addr", l.bindAddr, "error", err)
		}
	}
}

// Stop closes the trap listener and waits up to ctx deadline for
// the listen goroutine to drain.
func (l *Listener) Stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return nil
	}
	l.started = false
	tl := l.tl
	l.tl = nil
	if l.cancel != nil {
		l.cancel()
	}
	l.mu.Unlock()

	if tl != nil {
		tl.Close()
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
	l.logger.InfoContext(ctx, "snmp trap listener stopped", "addr", l.bindAddr)
	return nil
}

// handle is the gosnmp.TrapHandlerFunc. Runs in gosnmp's read
// goroutine — keep it short, no blocking calls beyond sink.Publish.
func (l *Listener) handle(pkt *gosnmp.SnmpPacket, src *net.UDPAddr) {
	if pkt == nil || src == nil {
		return
	}
	parsed := Parse(pkt)
	payload, err := json.Marshal(parsed)
	if err != nil {
		l.logger.Warn("snmptrap: marshal payload failed", "error", err)
		return
	}
	evt := listener.Event{
		Kind:       Name,
		SourceAddr: src.String(),
		Severity:   trapSeverityHeuristic(parsed),
		Timestamp:  l.now(),
		Payload:    json.RawMessage(payload),
	}
	// Best-effort publish — sink errors are logged but cannot
	// back-pressure gosnmp's read goroutine.
	if pubErr := l.sink.Publish(context.Background(), evt); pubErr != nil {
		l.logger.Warn("snmptrap: sink publish failed",
			"source", evt.SourceAddr, "error", pubErr)
	}
}

// Parsed carries the structured fields the trap parser extracted.
// Version is "v1", "v2c", or "v3". Varbinds preserves the
// original PDUs so downstream consumers can re-decode against
// vendor-specific schemas without re-receiving the trap.
type Parsed struct {
	Version     string    `json:"version"`
	Community   string    `json:"community"`
	TrapOID     string    `json:"trapOid,omitempty"`
	UptimeTicks uint32    `json:"uptimeTicks,omitempty"`
	Varbinds    []Varbind `json:"varbinds"`
}

// Varbind is one OID/value pair from a trap PDU. Value is a string
// repr — full type fidelity (Counter32 vs Gauge32 vs etc.) is
// preserved in Type.
type Varbind struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Parse extracts the structured fields from a gosnmp SnmpPacket.
// Tolerant of malformed/partial PDUs: missing fields surface as
// empty strings rather than failing the trap.
func Parse(pkt *gosnmp.SnmpPacket) Parsed {
	out := Parsed{
		Community: pkt.Community,
		Varbinds:  make([]Varbind, 0, len(pkt.Variables)),
	}
	switch pkt.Version {
	case gosnmp.Version1:
		out.Version = "v1"
	case gosnmp.Version2c:
		out.Version = "v2c"
	case gosnmp.Version3:
		out.Version = "v3"
	}

	// SNMPv2c trap convention: varbinds[0] = sysUpTime.0,
	// varbinds[1] = snmpTrapOID.0. Extract those into top-level
	// fields and keep the rest in Varbinds for the listener pipeline
	// to dispatch on.
	for i, vb := range pkt.Variables {
		oid := strings.TrimPrefix(vb.Name, ".")
		switch {
		case i == 0 && oid == "1.3.6.1.2.1.1.3.0":
			out.UptimeTicks = uint32Value(vb.Value)
		case i == 1 && oid == "1.3.6.1.6.3.1.1.4.1.0":
			out.TrapOID = strings.TrimPrefix(stringValue(vb.Value), ".")
		default:
			out.Varbinds = append(out.Varbinds, Varbind{
				OID:   oid,
				Type:  pduTypeName(vb.Type),
				Value: stringValue(vb.Value),
			})
		}
	}
	return out
}

// trapSeverityHeuristic returns a syslog-aligned severity string
// derived from the trap OID. Cold/warm-start traps are "notice";
// link-down is "warning"; authentication-failure is "error";
// link-up is "informational". Everything else falls to
// "informational" until rule-driven mapping lands in Stage A4.
func trapSeverityHeuristic(p Parsed) string {
	switch p.TrapOID {
	case "1.3.6.1.6.3.1.1.5.3":
		return "warning" // linkDown
	case "1.3.6.1.6.3.1.1.5.5":
		return "error" // authenticationFailure
	case "1.3.6.1.6.3.1.1.5.4":
		return "informational" // linkUp
	case "1.3.6.1.6.3.1.1.5.1", "1.3.6.1.6.3.1.1.5.2":
		return "notice" // cold/warm start
	}
	return "informational"
}

// pduTypeName returns a stable string label for a gosnmp Asn1BER
// type used in Varbind.Type so alert rules can distinguish
// Counter32 from Gauge32 etc. without importing gosnmp downstream.
//
// The mapping is built each call rather than cached because Go's
// linter mix flags every alternative: a package var trips
// gochecknoglobals, a single-switch fn trips cyclop, two split
// switches each trip exhaustive. The map is small (21 entries) and
// only built once per trap-varbind — cheap enough to not justify
// fighting the linter.
func pduTypeName(t gosnmp.Asn1BER) string {
	names := map[gosnmp.Asn1BER]string{
		gosnmp.EndOfContents:     "EndOfContents",
		gosnmp.Boolean:           "Boolean",
		gosnmp.Integer:           "Integer",
		gosnmp.BitString:         "BitString",
		gosnmp.OctetString:       "OctetString",
		gosnmp.Null:              "Null",
		gosnmp.ObjectIdentifier:  "ObjectIdentifier",
		gosnmp.ObjectDescription: "ObjectDescription",
		gosnmp.IPAddress:         "IPAddress",
		gosnmp.Counter32:         "Counter32",
		gosnmp.Gauge32:           "Gauge32",
		gosnmp.TimeTicks:         "TimeTicks",
		gosnmp.Opaque:            "Opaque",
		gosnmp.NsapAddress:       "NsapAddress",
		gosnmp.Counter64:         "Counter64",
		gosnmp.Uinteger32:        "Uinteger32",
		gosnmp.OpaqueFloat:       "OpaqueFloat",
		gosnmp.OpaqueDouble:      "OpaqueDouble",
		gosnmp.NoSuchObject:      "NoSuchObject",
		gosnmp.NoSuchInstance:    "NoSuchInstance",
		gosnmp.EndOfMibView:      "EndOfMibView",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return "Unknown"
}

func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func uint32Value(v any) uint32 {
	const maxUint32 uint64 = 1<<32 - 1
	switch t := v.(type) {
	case nil:
		return 0
	case uint32:
		return t
	case uint:
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case uint64:
		if t > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case int:
		if t < 0 {
			return 0
		}
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case int32:
		if t < 0 {
			return 0
		}
		return uint32(t)
	case int64:
		if t < 0 {
			return 0
		}
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	}
	return 0
}
