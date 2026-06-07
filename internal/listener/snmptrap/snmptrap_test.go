package snmptrap_test

// Tests in this file deliberately avoid t.Parallel(). gosnmp v1.43.2's
// (*GoSNMP).validateParameters writes to a package-level map without
// synchronisation (see gosnmp@v1.43.2/gosnmp.go:405-406), so two
// parallel listener startups race on that write. The race is in the
// upstream library — fixing it from here would require monkey-patching
// or upstream contribution. Serialising the tests instead keeps the
// race detector clean and costs nothing meaningful (these tests
// already bind sockets and aren't latency-sensitive).

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/listener/snmptrap"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

type fakeSink struct {
	mu  sync.Mutex
	got []listener.Event
}

func (f *fakeSink) Publish(_ context.Context, evt listener.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, evt)
	return nil
}

func (f *fakeSink) wait(t *testing.T, want int, dur time.Duration) []listener.Event {
	t.Helper()
	deadline := time.Now().Add(dur)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		if len(f.got) >= want {
			out := append([]listener.Event{}, f.got...)
			f.mu.Unlock()
			return out
		}
		f.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]listener.Event{}, f.got...)
}

// pickPort returns a free UDP loopback address so multiple tests
// don't collide on a fixed bind.
func pickPort(t *testing.T) string {
	t.Helper()
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pickPort: %v", err)
	}
	addr := c.LocalAddr().String()
	_ = c.Close()
	return addr
}

func TestParse_V2CSysUptimeAndTrapOID(t *testing.T) {
	pkt := &gosnmp.SnmpPacket{
		Version:   gosnmp.Version2c,
		Community: "public",
		Variables: []gosnmp.SnmpPDU{
			{Name: "1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: uint32(12345)},
			{Name: "1.3.6.1.6.3.1.1.4.1.0", Type: gosnmp.ObjectIdentifier, Value: "1.3.6.1.6.3.1.1.5.3"},
			{Name: "1.3.6.1.2.1.2.2.1.1.5", Type: gosnmp.Integer, Value: 5},
		},
	}
	p := snmptrap.Parse(pkt)
	if p.Version != "v2c" {
		t.Errorf("Version = %q", p.Version)
	}
	if p.Community != "public" {
		t.Errorf("Community = %q", p.Community)
	}
	if p.UptimeTicks != 12345 {
		t.Errorf("UptimeTicks = %d", p.UptimeTicks)
	}
	if p.TrapOID != "1.3.6.1.6.3.1.1.5.3" {
		t.Errorf("TrapOID = %q", p.TrapOID)
	}
	if len(p.Varbinds) != 1 || p.Varbinds[0].OID != "1.3.6.1.2.1.2.2.1.1.5" {
		t.Errorf("Varbinds = %+v", p.Varbinds)
	}
}

func TestParse_MissingTrapOIDStillEmitsVarbinds(t *testing.T) {
	pkt := &gosnmp.SnmpPacket{
		Version:   gosnmp.Version1,
		Community: "private",
		Variables: []gosnmp.SnmpPDU{
			{Name: "1.3.6.1.4.1.9.9.41.1.2.3.1.5", Type: gosnmp.OctetString, Value: []byte("link state changed")},
		},
	}
	p := snmptrap.Parse(pkt)
	if p.Version != "v1" {
		t.Errorf("Version = %q", p.Version)
	}
	if p.TrapOID != "" {
		t.Errorf("Expected empty TrapOID without sysUpTime/snmpTrapOID; got %q", p.TrapOID)
	}
	if len(p.Varbinds) != 1 {
		t.Fatalf("Varbinds = %d, want 1", len(p.Varbinds))
	}
	if p.Varbinds[0].Value != "link state changed" {
		t.Errorf("Varbind value = %q", p.Varbinds[0].Value)
	}
}

func TestNew_RejectsNilSink(t *testing.T) {
	if _, err := snmptrap.New(snmptrap.Config{}); err == nil {
		t.Error("expected New with nil Sink to fail")
	}
}

func TestStartStop_BindAndUnbind(t *testing.T) {
	addr := pickPort(t)
	sink := &fakeSink{}
	l, newErr := snmptrap.New(snmptrap.Config{
		BindAddr: addr,
		Sink:     sink,
		Logger:   silentLogger(),
	})
	if newErr != nil {
		t.Fatalf("New: %v", newErr)
	}
	if startErr := l.Start(context.Background()); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	if startErr2 := l.Start(context.Background()); startErr2 != nil {
		t.Fatalf("second Start: %v", startErr2)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if stopErr := l.Stop(stopCtx); stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}
}

func TestStop_BeforeStartIsNoOp(t *testing.T) {
	l, _ := snmptrap.New(snmptrap.Config{
		BindAddr: pickPort(t),
		Sink:     &fakeSink{},
		Logger:   silentLogger(),
	})
	if err := l.Stop(context.Background()); err != nil {
		t.Errorf("Stop before Start should be nil, got %v", err)
	}
}

// TestRoundTrip_TrapArrivesAtSink fires a real SNMPv2c trap at the
// listener via gosnmp's SendTrap and verifies it lands in the sink.
func TestRoundTrip_TrapArrivesAtSink(t *testing.T) {
	addr := pickPort(t)
	sink := &fakeSink{}
	l, newErr := snmptrap.New(snmptrap.Config{
		BindAddr: addr,
		Sink:     sink,
		Logger:   silentLogger(),
	})
	if newErr != nil {
		t.Fatalf("New: %v", newErr)
	}
	if startErr := l.Start(context.Background()); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = l.Stop(stopCtx)
	}()

	host, port, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		t.Fatalf("split addr: %v", splitErr)
	}
	g := &gosnmp.GoSNMP{
		Target:    host,
		Port:      mustAtoi(t, port),
		Community: "public",
		Version:   gosnmp.Version2c,
		Timeout:   time.Second,
		Retries:   0,
	}
	if connErr := g.Connect(); connErr != nil {
		t.Fatalf("connect: %v", connErr)
	}
	defer func() { _ = g.Conn.Close() }()

	_, sendErr := g.SendTrap(gosnmp.SnmpTrap{
		Variables: []gosnmp.SnmpPDU{
			{Name: "1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: uint32(54321)},
			{Name: "1.3.6.1.6.3.1.1.4.1.0", Type: gosnmp.ObjectIdentifier, Value: "1.3.6.1.6.3.1.1.5.3"},
			{Name: "1.3.6.1.2.1.2.2.1.1.5", Type: gosnmp.Integer, Value: 5},
		},
	})
	if sendErr != nil {
		t.Fatalf("send trap: %v", sendErr)
	}

	got := sink.wait(t, 1, 3*time.Second)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	evt := got[0]
	if evt.Kind != snmptrap.Name {
		t.Errorf("Kind = %q", evt.Kind)
	}
	if evt.Severity != "warning" {
		t.Errorf("Severity for linkDown = %q, want warning", evt.Severity)
	}
	var parsed snmptrap.Parsed
	if uErr := json.Unmarshal(evt.Payload, &parsed); uErr != nil {
		t.Fatalf("unmarshal: %v", uErr)
	}
	if parsed.TrapOID != "1.3.6.1.6.3.1.1.5.3" {
		t.Errorf("Parsed.TrapOID = %q", parsed.TrapOID)
	}
	if parsed.UptimeTicks != 54321 {
		t.Errorf("Parsed.UptimeTicks = %d, want 54321", parsed.UptimeTicks)
	}
}

func mustAtoi(t *testing.T, s string) uint16 {
	t.Helper()
	var n uint16
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("bad port %q", s)
		}
		n = n*10 + uint16(c-'0')
	}
	return n
}
