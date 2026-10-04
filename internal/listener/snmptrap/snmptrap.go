// Package snmptrap is the passive SNMP trap receiver.
//
// It binds UDP/162 (the operator may remap it) and accepts SNMPv1 and SNMPv2c
// traps and informs, and SNMPv3 traps authenticated by a user in the
// credential vault. Each notification is normalized into a [Parsed] struct and
// published as a [listener.Event] of kind [Name] through the configured
// [listener.Sink].
//
// gosnmp decodes the messages and performs the USM authentication and
// decryption, which agree with net-snmp. Its TrapListener does not enforce the
// rest of RFC 3414 for a receiver, so this package owns the socket instead
// (ADR-0022, amended for P-B4): a v3 trap is accepted only at the security
// level its user is stored with, and only inside the sender's time window.
// SNMPv3 informs need Seed to act as an authoritative engine and are dropped
// until that lands (seed#1376).
package snmptrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// Name is the listener key used in observability, the engine registry and
// the persisted event kind. It covers every SNMP version the listener takes.
const Name = "snmp-trap"

const (
	defaultBindAddr = ":162"

	// maxDatagram is the largest UDP payload; a trap is never split.
	maxDatagram = 65535

	// credentialRefresh bounds how stale the v3 user table may be. A
	// credential added in the vault is honoured from the first datagram
	// after this interval, and a burst of traps costs one vault read.
	credentialRefresh = 30 * time.Second
)

var (
	errSecurityLevel   = errors.New("security level differs from the stored credential")
	errNotInTimeWindow = errors.New("outside the sender's time window (RFC 3414 3.2.7)")
	errV3Inform        = errors.New("SNMPv3 informs are not accepted yet")
	errNotNotification = errors.New("not a trap or inform PDU")
)

// CredentialSource resolves the vault's SNMP credentials. Its V3Credentials
// are the users whose traps are accepted.
type CredentialSource interface {
	SNMPSession(ctx context.Context) (*snmp.Session, error)
}

// Listener is one bound UDP socket decoding incoming traps.
type Listener struct {
	bindAddr    string
	sink        listener.Sink
	credentials CredentialSource
	logger      *slog.Logger
	now         func() time.Time

	mu      sync.Mutex
	conn    *net.UDPConn
	started bool
	wg      sync.WaitGroup

	// Owned by the serve goroutine.
	decoder     *gosnmp.GoSNMP
	userLevels  map[string][]gosnmp.SnmpV3MsgFlags
	refreshedAt time.Time
	engines     engineClocks
}

// Config wires the listener. BindAddr defaults to ":162". Sink is required.
// Credentials may be nil, in which case every SNMPv3 message is dropped.
// Logger / Now default to [slog.Default] + [time.Now] UTC.
type Config struct {
	BindAddr    string
	Sink        listener.Sink
	Credentials CredentialSource
	Logger      *slog.Logger
	Now         func() time.Time
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
	l := &Listener{
		bindAddr:    cfg.BindAddr,
		sink:        cfg.Sink,
		credentials: cfg.Credentials,
		logger:      cfg.Logger,
		now:         cfg.Now,
		engines:     engineClocks{},
	}
	l.setUsers(nil)
	return l, nil
}

// Name returns the listener key. Implements [listener.Listener] +
// [engine.Engine].
func (*Listener) Name() string { return Name }

// Start binds the socket and spawns the receive goroutine. A bind failure is
// returned synchronously. Idempotent.
func (l *Listener) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		return nil
	}
	addr, err := net.ResolveUDPAddr("udp", l.bindAddr)
	if err != nil {
		return fmt.Errorf("snmptrap: resolve %s: %w", l.bindAddr, err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("snmptrap: bind %s: %w", l.bindAddr, err)
	}
	l.conn = conn
	l.started = true
	l.wg.Add(1)
	go l.serve(context.WithoutCancel(ctx), conn)
	l.logger.InfoContext(ctx, "snmp trap listener started", "addr", conn.LocalAddr().String())
	return nil
}

// Stop closes the socket and waits up to ctx deadline for the receive
// goroutine to drain.
func (l *Listener) Stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return nil
	}
	l.started = false
	conn := l.conn
	l.conn = nil
	l.mu.Unlock()

	_ = conn.Close()
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

// serve reads datagrams until the socket is closed.
func (l *Listener) serve(ctx context.Context, conn *net.UDPConn) {
	defer l.wg.Done()
	buf := make([]byte, maxDatagram)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.logger.WarnContext(ctx, "snmptrap: read failed", "error", err)
			continue
		}
		l.receive(ctx, conn, buf[:n], src)
	}
}

// receive decodes, admits and publishes one datagram, and acknowledges an
// inform once it is published.
func (l *Listener) receive(ctx context.Context, conn *net.UDPConn, msg []byte, src *net.UDPAddr) {
	l.refreshUsers(ctx)
	pkt, err := l.decoder.UnmarshalTrap(msg, true)
	if err == nil {
		err = l.admit(pkt)
	}
	if err != nil {
		l.logger.DebugContext(ctx, "snmptrap: datagram dropped", "source", src.String(), "error", err)
		return
	}
	l.publish(ctx, pkt, src)
	if pkt.PDUType == gosnmp.InformRequest {
		l.acknowledge(ctx, conn, pkt, src)
	}
}

// admit applies the checks gosnmp's decoder leaves to the receiver.
func (l *Listener) admit(pkt *gosnmp.SnmpPacket) error {
	inform := pkt.PDUType == gosnmp.InformRequest
	if pkt.PDUType != gosnmp.Trap && pkt.PDUType != gosnmp.SNMPv2Trap && !inform {
		return fmt.Errorf("%w: %v", errNotNotification, pkt.PDUType)
	}
	if pkt.Version != gosnmp.Version3 {
		return nil
	}
	if inform {
		return errV3Inform
	}

	usm, ok := pkt.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok {
		return fmt.Errorf("security parameters %T", pkt.SecurityParameters)
	}
	// gosnmp authenticates at the level the message claims, so without this
	// a noAuthNoPriv message naming an authPriv user would be accepted
	// unauthenticated.
	level := pkt.MsgFlags &^ gosnmp.Reportable
	if !slices.Contains(l.userLevels[usm.UserName], level) {
		return fmt.Errorf("%w: user %q sent %v", errSecurityLevel, usm.UserName, level)
	}
	if level == gosnmp.NoAuthNoPriv {
		// The time fields of an unauthenticated message prove nothing.
		return nil
	}
	return l.engines.admit(usm.AuthoritativeEngineID,
		usm.AuthoritativeEngineBoots, usm.AuthoritativeEngineTime, l.now())
}

// publish hands the notification to the sink.
func (l *Listener) publish(ctx context.Context, pkt *gosnmp.SnmpPacket, src *net.UDPAddr) {
	parsed := Parse(pkt)
	payload, err := json.Marshal(parsed)
	if err != nil {
		l.logger.WarnContext(ctx, "snmptrap: marshal payload failed", "error", err)
		return
	}
	evt := listener.Event{
		Kind:       Name,
		SourceAddr: src.String(),
		Severity:   trapSeverityHeuristic(parsed),
		Timestamp:  l.now(),
		Payload:    json.RawMessage(payload),
	}
	if pubErr := l.sink.Publish(ctx, evt); pubErr != nil {
		l.logger.WarnContext(ctx, "snmptrap: sink publish failed",
			"source", evt.SourceAddr, "error", pubErr)
	}
}

// acknowledge answers a v1/v2c inform with the same varbinds, as RFC 3416
// 4.2.7 requires.
func (l *Listener) acknowledge(ctx context.Context, conn *net.UDPConn, pkt *gosnmp.SnmpPacket, src *net.UDPAddr) {
	pkt.PDUType = gosnmp.GetResponse
	pkt.Error = gosnmp.NoError
	pkt.ErrorIndex = 0
	out, err := pkt.MarshalMsg()
	if err == nil {
		_, err = conn.WriteToUDP(out, src)
	}
	if err != nil {
		l.logger.WarnContext(ctx, "snmptrap: inform response failed", "source", src.String(), "error", err)
	}
}

// refreshUsers re-reads the vault's v3 users once the table is older than
// credentialRefresh. A failed read keeps the previous table.
func (l *Listener) refreshUsers(ctx context.Context) {
	now := l.now()
	if l.credentials == nil || (!l.refreshedAt.IsZero() && now.Sub(l.refreshedAt) < credentialRefresh) {
		return
	}
	l.refreshedAt = now
	session, err := l.credentials.SNMPSession(ctx)
	if err != nil {
		l.logger.WarnContext(ctx, "snmptrap: SNMPv3 users not refreshed", "error", err)
		return
	}
	l.setUsers(session.V3Credentials)
}

// setUsers rebuilds the decoder over creds. The decoder is configured for v3
// so it can authenticate; v1 and v2c messages decode through it unchanged.
func (l *Listener) setUsers(creds []snmp.V3Credential) {
	table := gosnmp.NewSnmpV3SecurityParametersTable(gosnmp.NewLogger(nil))
	levels := make(map[string][]gosnmp.SnmpV3MsgFlags, len(creds))
	for i := range creds {
		cred := &creds[i]
		if err := table.Add(cred.Username, cred.USM()); err != nil {
			l.logger.Warn("snmptrap: SNMPv3 user skipped", "credential", cred.Name, "error", err)
			continue
		}
		levels[cred.Username] = append(levels[cred.Username], cred.MsgFlags())
	}
	l.decoder = &gosnmp.GoSNMP{
		Version:                     gosnmp.Version3,
		SecurityModel:               gosnmp.UserSecurityModel,
		SecurityParameters:          &gosnmp.UsmSecurityParameters{},
		TrapSecurityParametersTable: table,
	}
	l.userLevels = levels
}

// timeWindow is RFC 3414 2.2.3's 150-second acceptance window.
const timeWindow = 150

// maxEngineBoots is snmpEngineBoots' latched maximum (RFC 3414 2.2.2).
const maxEngineBoots = 2147483647

// engineClock is the receiver's notion of one sender engine's clock.
type engineClock struct {
	boots      uint32
	time       uint32
	latestTime uint32
	at         time.Time
}

// engineClocks tracks every authoritative engine an authenticated trap has
// come from, keyed by snmpEngineID. For a trap the sender is authoritative,
// so this is the non-authoritative side of RFC 3414 3.2.7 b.
type engineClocks map[string]engineClock

// admit updates the sender's clock and reports whether the message's time
// fields fall inside its window. The first message from an engine sets the
// clock.
func (c engineClocks) admit(engineID string, boots, engineTime uint32, now time.Time) error {
	clock, known := c[engineID]
	if !known || boots > clock.boots || (boots == clock.boots && engineTime > clock.latestTime) {
		clock = engineClock{boots: boots, time: engineTime, latestTime: engineTime, at: now}
		c[engineID] = clock
	}
	local := int64(clock.time) + int64(now.Sub(clock.at)/time.Second)
	switch {
	case clock.boots == maxEngineBoots,
		boots < clock.boots,
		boots == clock.boots && int64(engineTime) < local-timeWindow:
		return fmt.Errorf("%w: engine %x boots %d time %d", errNotInTimeWindow,
			[]byte(engineID), boots, engineTime)
	}
	return nil
}

// Parsed carries the structured fields the trap parser extracted.
// Version is "v1", "v2c", or "v3"; Community is set for v1/v2c, User and the
// hex EngineID of the sending engine for v3. Varbinds preserves the
// original PDUs so downstream consumers can re-decode against
// vendor-specific schemas without re-receiving the trap.
type Parsed struct {
	Version     string    `json:"version"`
	Community   string    `json:"community,omitempty"`
	User        string    `json:"user,omitempty"`
	EngineID    string    `json:"engineId,omitempty"`
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
		if usm, ok := pkt.SecurityParameters.(*gosnmp.UsmSecurityParameters); ok {
			out.User = usm.UserName
			out.EngineID = hex.EncodeToString([]byte(usm.AuthoritativeEngineID))
		}
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
