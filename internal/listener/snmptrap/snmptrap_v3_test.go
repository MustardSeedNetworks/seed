package snmptrap_test

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/listener/snmptrap"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// senderEngine is the trap sender's snmpEngineID. For a trap the sender is
// the authoritative engine, so its keys are localized against this.
const senderEngine = "\x80\x00\x1f\x88\x80seedtest"

// fakeCredentials serves a fixed vault and counts reads.
type fakeCredentials struct {
	mu    sync.Mutex
	creds []snmp.V3Credential
	reads int
}

func (f *fakeCredentials) SNMPSession(context.Context) (*snmp.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	return &snmp.Session{V3Credentials: append([]snmp.V3Credential(nil), f.creds...)}, nil
}

func (f *fakeCredentials) set(creds ...snmp.V3Credential) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creds = creds
}

// manualClock is a settable Now for the listener.
type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func alicePriv() snmp.V3Credential {
	return snmp.V3Credential{
		Name: "alice", Username: "alice", SecurityLevel: "authPriv",
		AuthProtocol: "SHA256", AuthPassword: "alice-auth-pass",
		PrivProtocol: "AES", PrivPassword: "alice-priv-pass",
	}
}

func bobAuth() snmp.V3Credential {
	return snmp.V3Credential{
		Name: "bob", Username: "bob", SecurityLevel: "authNoPriv",
		AuthProtocol: "SHA512", AuthPassword: "bob-auth-pass",
	}
}

// v3Trap describes one SNMPv3 trap as the sender puts it on the wire.
type v3Trap struct {
	cred   snmp.V3Credential
	flags  gosnmp.SnmpV3MsgFlags
	boots  uint32
	time   uint32
	inform bool
}

// send fires the trap at addr. An inform waits up to one second for the
// acknowledgement and returns the error when none arrives.
func (v v3Trap) send(t *testing.T, addr string) error {
	t.Helper()
	usm := v.cred.USM()
	usm.AuthoritativeEngineID = senderEngine
	usm.AuthoritativeEngineBoots = v.boots
	usm.AuthoritativeEngineTime = v.time
	g := newSender(t, addr, gosnmp.Version3, func(g *gosnmp.GoSNMP) {
		g.SecurityModel = gosnmp.UserSecurityModel
		g.MsgFlags = v.flags
		g.SecurityParameters = usm
	})
	defer func() { _ = g.Conn.Close() }()
	_, err := g.SendTrap(gosnmp.SnmpTrap{IsInform: v.inform, Variables: linkDownVarbinds()})
	return err
}

func newSender(t *testing.T, addr string, version gosnmp.SnmpVersion, configure func(*gosnmp.GoSNMP)) *gosnmp.GoSNMP {
	t.Helper()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	g := &gosnmp.GoSNMP{
		Target:    host,
		Port:      mustAtoi(t, port),
		Community: "public",
		Version:   version,
		Timeout:   time.Second,
		Retries:   0,
	}
	if configure != nil {
		configure(g)
	}
	if connErr := g.Connect(); connErr != nil {
		t.Fatalf("connect: %v", connErr)
	}
	return g
}

func linkDownVarbinds() []gosnmp.SnmpPDU {
	return []gosnmp.SnmpPDU{
		{Name: "1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: uint32(54321)},
		{Name: "1.3.6.1.6.3.1.1.4.1.0", Type: gosnmp.ObjectIdentifier, Value: "1.3.6.1.6.3.1.1.5.3"},
		{Name: "1.3.6.1.2.1.2.2.1.1.7", Type: gosnmp.Integer, Value: 7},
	}
}

// startListener binds a listener over creds (nil for none) and stops it at
// cleanup.
func startListener(t *testing.T, creds snmptrap.CredentialSource, clock *manualClock) (string, *fakeSink) {
	t.Helper()
	addr := pickPort(t)
	sink := &fakeSink{}
	cfg := snmptrap.Config{BindAddr: addr, Sink: sink, Credentials: creds, Logger: silentLogger()}
	if clock != nil {
		cfg.Now = clock.Now
	}
	l, err := snmptrap.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if startErr := l.Start(context.Background()); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = l.Stop(ctx)
	})
	return addr, sink
}

// settle gives a dropped datagram time to have been processed, so a test
// asserting "nothing arrived" is not vacuous.
const settle = 300 * time.Millisecond

func TestV3Trap_Admission(t *testing.T) {
	wrongPass := alicePriv()
	wrongPass.AuthPassword = "not-alices-pass"
	unknown := alicePriv()
	unknown.Username = "mallory"
	both := []snmp.V3Credential{alicePriv(), bobAuth()}

	tests := []struct {
		name  string
		cred  snmp.V3Credential
		flags gosnmp.SnmpV3MsgFlags
		vault []snmp.V3Credential
		want  bool
	}{
		{"authPriv user at authPriv", alicePriv(), gosnmp.AuthPriv, both, true},
		{"authNoPriv user at authNoPriv", bobAuth(), gosnmp.AuthNoPriv, both, true},
		{"authPriv user sent unauthenticated", alicePriv(), gosnmp.NoAuthNoPriv, both, false},
		{"authPriv user sent without privacy", alicePriv(), gosnmp.AuthNoPriv, both, false},
		{"wrong auth passphrase", wrongPass, gosnmp.AuthPriv, both, false},
		{"user not in the vault", unknown, gosnmp.AuthPriv, both, false},
		{"no v3 users at all", alicePriv(), gosnmp.AuthPriv, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vault := &fakeCredentials{}
			vault.set(tc.vault...)
			addr, sink := startListener(t, vault, nil)
			trap := v3Trap{cred: tc.cred, flags: tc.flags, boots: 3, time: 100}
			if err := trap.send(t, addr); err != nil {
				t.Fatalf("send: %v", err)
			}
			if !tc.want {
				if got := sink.wait(t, 1, settle); len(got) != 0 {
					t.Fatalf("got %d events, want the trap dropped", len(got))
				}
				return
			}
			got := sink.wait(t, 1, 3*time.Second)
			if len(got) != 1 {
				t.Fatalf("got %d events, want 1", len(got))
			}
			assertLinkDownV3(t, got[0], tc.cred.Username)
		})
	}
}

// assertLinkDownV3 checks evt is linkDownVarbinds sent as user from
// senderEngine.
func assertLinkDownV3(t *testing.T, evt listener.Event, user string) {
	t.Helper()
	var parsed snmptrap.Parsed
	if err := json.Unmarshal(evt.Payload, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Version != "v3" || parsed.User != user ||
		parsed.EngineID != "80001f88807365656474657374" || parsed.Community != "" {
		t.Errorf("parsed identity = %+v", parsed)
	}
	if parsed.TrapOID != "1.3.6.1.6.3.1.1.5.3" || len(parsed.Varbinds) != 1 || parsed.Varbinds[0].Value != "7" {
		t.Errorf("parsed body = %+v", parsed)
	}
	if evt.Severity != "warning" {
		t.Errorf("Severity = %q, want warning", evt.Severity)
	}
}

// TestV3Trap_TimeWindow replays an authenticated trap whose clock is behind
// one the listener has already seen from the same engine.
func TestV3Trap_TimeWindow(t *testing.T) {
	tests := []struct {
		name        string
		boots, time uint32
		want        bool
	}{
		{"same boot, later time", 7, 5000, true},
		{"same boot, inside the window", 7, 1000 - 150, true},
		{"same boot, past the window", 7, 1000 - 151, false},
		{"earlier boot", 6, 99999, false},
		{"later boot, time reset", 8, 1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vault := &fakeCredentials{}
			vault.set(alicePriv())
			clock := &manualClock{now: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)}
			addr, sink := startListener(t, vault, clock)

			first := v3Trap{cred: alicePriv(), flags: gosnmp.AuthPriv, boots: 7, time: 1000}
			if err := first.send(t, addr); err != nil {
				t.Fatalf("send first: %v", err)
			}
			if got := sink.wait(t, 1, 3*time.Second); len(got) != 1 {
				t.Fatalf("first trap: got %d events", len(got))
			}

			second := v3Trap{cred: alicePriv(), flags: gosnmp.AuthPriv, boots: tc.boots, time: tc.time}
			if err := second.send(t, addr); err != nil {
				t.Fatalf("send second: %v", err)
			}
			wait := settle
			if tc.want {
				wait = 3 * time.Second
			}
			if got := sink.wait(t, 2, wait); (len(got) == 2) != tc.want {
				t.Fatalf("got %d events, want second accepted=%v", len(got), tc.want)
			}
		})
	}
}

// TestV3Trap_WindowFollowsTheClock checks the window is measured against the
// receiver's running estimate of the sender's clock, not the last value seen.
func TestV3Trap_WindowFollowsTheClock(t *testing.T) {
	vault := &fakeCredentials{}
	vault.set(alicePriv())
	clock := &manualClock{now: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)}
	addr, sink := startListener(t, vault, clock)

	if err := (v3Trap{cred: alicePriv(), flags: gosnmp.AuthPriv, boots: 2, time: 1000}).send(t, addr); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got := sink.wait(t, 1, 3*time.Second); len(got) != 1 {
		t.Fatalf("first trap: got %d events", len(got))
	}
	// Ten minutes on, the sender's clock reads about 1600. A trap stamped
	// 1000 is a capture being replayed.
	clock.advance(10 * time.Minute)
	if err := (v3Trap{cred: alicePriv(), flags: gosnmp.AuthPriv, boots: 2, time: 1000}).send(t, addr); err != nil {
		t.Fatalf("send replay: %v", err)
	}
	if got := sink.wait(t, 2, settle); len(got) != 1 {
		t.Fatalf("replayed trap accepted: got %d events", len(got))
	}
}

// TestV3Trap_CredentialRefresh adds a user to the vault after the listener
// has read it and checks the user is honoured once the table is stale.
func TestV3Trap_CredentialRefresh(t *testing.T) {
	vault := &fakeCredentials{}
	vault.set(alicePriv())
	clock := &manualClock{now: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)}
	addr, sink := startListener(t, vault, clock)

	bob := v3Trap{cred: bobAuth(), flags: gosnmp.AuthNoPriv, boots: 1, time: 10}
	if err := bob.send(t, addr); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got := sink.wait(t, 1, settle); len(got) != 0 {
		t.Fatalf("bob accepted before he was in the vault")
	}

	vault.set(alicePriv(), bobAuth())
	clock.advance(10 * time.Second)
	if err := bob.send(t, addr); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got := sink.wait(t, 1, settle); len(got) != 0 {
		t.Fatalf("vault re-read inside the refresh interval")
	}

	clock.advance(30 * time.Second)
	bob.time = 50
	if err := bob.send(t, addr); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got := sink.wait(t, 1, 3*time.Second); len(got) != 1 {
		t.Fatalf("bob not accepted after the refresh interval: got %d events", len(got))
	}
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if vault.reads != 2 {
		t.Errorf("vault reads = %d, want 2 (first datagram and one refresh)", vault.reads)
	}
}

// TestV3Inform_NotAcknowledged pins slice 1: a v3 inform needs Seed to be an
// authoritative engine, so it is neither published nor answered.
func TestV3Inform_NotAcknowledged(t *testing.T) {
	vault := &fakeCredentials{}
	vault.set(alicePriv())
	addr, sink := startListener(t, vault, nil)

	inform := v3Trap{cred: alicePriv(), flags: gosnmp.AuthPriv, boots: 1, time: 10, inform: true}
	if err := inform.send(t, addr); err == nil {
		t.Fatal("v3 inform was acknowledged")
	}
	if got := sink.wait(t, 1, settle); len(got) != 0 {
		t.Fatalf("v3 inform published: got %d events", len(got))
	}
}

func TestV2cInform_Acknowledged(t *testing.T) {
	addr, sink := startListener(t, nil, nil)

	g := newSender(t, addr, gosnmp.Version2c, nil)
	defer func() { _ = g.Conn.Close() }()
	resp, err := g.SendTrap(gosnmp.SnmpTrap{IsInform: true, Variables: linkDownVarbinds()})
	if err != nil {
		t.Fatalf("inform not acknowledged: %v", err)
	}
	if resp.PDUType != gosnmp.GetResponse || len(resp.Variables) != 3 {
		t.Errorf("response = %v with %d varbinds, want GetResponse echoing 3", resp.PDUType, len(resp.Variables))
	}
	if got := sink.wait(t, 1, 3*time.Second); len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
}
