package snmptrap_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/listener/snmptrap"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// memEngine is an in-memory snmp_engine row.
type memEngine struct {
	mu    sync.Mutex
	id    []byte
	boots uint32
}

func (m *memEngine) BootSNMPEngine(_ context.Context, newID []byte) ([]byte, uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.id == nil {
		m.id = newID
	}
	m.boots++
	return m.id, m.boots, nil
}

func (m *memEngine) identity() (string, uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(m.id), m.boots
}

// informSender is an SNMPv3 notification originator that, like net-snmp's
// snmpinform, starts with no knowledge of Seed's engine and discovers it.
func informSender(t *testing.T, addr string, cred snmp.V3Credential, flags gosnmp.SnmpV3MsgFlags) *gosnmp.GoSNMP {
	t.Helper()
	g := newSender(t, addr, gosnmp.Version3, func(g *gosnmp.GoSNMP) {
		g.SecurityModel = gosnmp.UserSecurityModel
		g.MsgFlags = flags
		g.SecurityParameters = cred.USM()
		g.ContextEngineID = senderEngine
	})
	t.Cleanup(func() { _ = g.Conn.Close() })
	return g
}

func sendInform(g *gosnmp.GoSNMP) (*gosnmp.SnmpPacket, error) {
	return g.SendTrap(gosnmp.SnmpTrap{IsInform: true, Variables: linkDownVarbinds()})
}

// TestV3Inform_Acknowledged is P-B4 slice 2's acceptance in process: the
// sender discovers Seed's persisted engine, the inform is published once and
// answered at its own security level.
func TestV3Inform_Acknowledged(t *testing.T) {
	des := snmp.V3Credential{
		Name: "carol", Username: "carol", SecurityLevel: "authPriv",
		AuthProtocol: "SHA", AuthPassword: "carol-auth-pass",
		PrivProtocol: "DES", PrivPassword: "carol-priv-pass",
	}
	tests := []struct {
		name  string
		cred  snmp.V3Credential
		flags gosnmp.SnmpV3MsgFlags
	}{
		{"authPriv SHA-256/AES", alicePriv(), gosnmp.AuthPriv},
		{"authNoPriv SHA-512", bobAuth(), gosnmp.AuthNoPriv},
		{"authPriv SHA/DES", des, gosnmp.AuthPriv},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vault := &fakeCredentials{}
			vault.set(alicePriv(), bobAuth(), des)
			engine := &memEngine{}
			addr, sink := startListenerWith(t, snmptrap.Config{Credentials: vault, Engine: engine})

			g := informSender(t, addr, tc.cred, tc.flags)
			resp, err := sendInform(g)
			if err != nil {
				t.Fatalf("inform not acknowledged: %v", err)
			}
			if resp.PDUType != gosnmp.GetResponse || len(resp.Variables) != 3 || resp.MsgFlags != tc.flags {
				t.Errorf("response = %v flags %v with %d varbinds, want GetResponse at %v echoing 3",
					resp.PDUType, resp.MsgFlags, len(resp.Variables), tc.flags)
			}
			assertLearnedEngine(t, g, engine)
			assertInformEvent(t, sink, tc.cred.Username)
		})
	}
}

// assertLearnedEngine checks discovery taught the sender Seed's stored
// engine, an ID under PEN 67249 in the octets format.
func assertLearnedEngine(t *testing.T, g *gosnmp.GoSNMP, engine *memEngine) {
	t.Helper()
	id, boots := engine.identity()
	learned, ok := g.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok || learned.AuthoritativeEngineID != id || learned.AuthoritativeEngineBoots != boots {
		t.Errorf("sender learned %+v, want engine %x boots %d", g.SecurityParameters, id, boots)
	}
	if !bytes.HasPrefix([]byte(id), []byte{0x80, 0x01, 0x06, 0xb1, 0x05}) {
		t.Errorf("engine ID %x is not under PEN 67249 in the octets format", id)
	}
}

// assertInformEvent checks exactly one linkDown inform from user, named by
// the sender's contextEngineID, was published.
func assertInformEvent(t *testing.T, sink *fakeSink, user string) {
	t.Helper()
	got := sink.wait(t, 1, 3*time.Second)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	var parsed snmptrap.Parsed
	if err := json.Unmarshal(got[0].Payload, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Version != "v3" || parsed.User != user ||
		parsed.EngineID != hex.EncodeToString([]byte(senderEngine)) ||
		parsed.TrapOID != "1.3.6.1.6.3.1.1.5.3" {
		t.Errorf("parsed = %+v", parsed)
	}
}

// TestV3Inform_Rejected: an inform is held to the same admission rules as a
// trap, and a rejected one is neither published nor answered.
func TestV3Inform_Rejected(t *testing.T) {
	wrongPass := alicePriv()
	wrongPass.PrivPassword = "not-alices-priv"
	unknown := alicePriv()
	unknown.Username = "mallory"
	tests := []struct {
		name  string
		cred  snmp.V3Credential
		flags gosnmp.SnmpV3MsgFlags
	}{
		{"authPriv user sent unauthenticated", alicePriv(), gosnmp.NoAuthNoPriv},
		{"authPriv user sent without privacy", alicePriv(), gosnmp.AuthNoPriv},
		{"wrong privacy passphrase", wrongPass, gosnmp.AuthPriv},
		{"user not in the vault", unknown, gosnmp.AuthPriv},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vault := &fakeCredentials{}
			vault.set(alicePriv())
			addr, sink := startListenerWith(t, snmptrap.Config{Credentials: vault, Engine: &memEngine{}})

			if _, err := sendInform(informSender(t, addr, tc.cred, tc.flags)); err == nil {
				t.Error("rejected inform was acknowledged")
			}
			if got := sink.wait(t, 1, settle); len(got) != 0 {
				t.Fatalf("rejected inform published: got %d events", len(got))
			}
		})
	}
}

// rawInform puts one v3 inform on the wire with exactly the given engine
// fields, so no sender library resynchronizes it, and returns Seed's reply
// decoded under cred, or nil when none arrives.
func rawInform(
	t *testing.T,
	addr string,
	cred snmp.V3Credential,
	engineID string,
	boots, engineTime uint32,
) *gosnmp.SnmpPacket {
	t.Helper()
	usm := cred.USM()
	usm.AuthoritativeEngineID = engineID
	usm.AuthoritativeEngineBoots = boots
	usm.AuthoritativeEngineTime = engineTime
	if err := usm.InitSecurityKeys(); err != nil {
		t.Fatalf("localize keys: %v", err)
	}
	pkt := &gosnmp.SnmpPacket{
		Version: gosnmp.Version3, MsgFlags: cred.MsgFlags() | gosnmp.Reportable,
		SecurityModel: gosnmp.UserSecurityModel, SecurityParameters: usm,
		MsgID: 41, ContextEngineID: senderEngine, PDUType: gosnmp.InformRequest,
		RequestID: 42, Variables: linkDownVarbinds(), Logger: gosnmp.NewLogger(nil),
	}
	if err := usm.InitPacket(pkt); err != nil {
		t.Fatalf("salt: %v", err)
	}
	msg, err := pkt.MarshalMsg()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	conn, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err = conn.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return nil
	}
	table := gosnmp.NewSnmpV3SecurityParametersTable(gosnmp.NewLogger(nil))
	if err = table.Add(cred.Username, cred.USM()); err != nil {
		t.Fatalf("table: %v", err)
	}
	decoder := &gosnmp.GoSNMP{
		Version: gosnmp.Version3, SecurityModel: gosnmp.UserSecurityModel,
		SecurityParameters: &gosnmp.UsmSecurityParameters{}, TrapSecurityParametersTable: table,
	}
	reply, err := decoder.UnmarshalTrap(buf[:n], true)
	if err != nil {
		t.Fatalf("Seed's reply does not authenticate under %s: %v", cred.Username, err)
	}
	return reply
}

// TestV3Inform_TimeWindow: Seed holds an inform to its own clock (RFC 3414
// 3.2.7 a). Outside the window it answers with an authenticated
// usmStatsNotInTimeWindows report carrying its boots and time, and publishes
// nothing; inside it, the inform is answered and published.
func TestV3Inform_TimeWindow(t *testing.T) {
	const elapsed = 1000
	tests := []struct {
		name       string
		bootsDelta int
		engineTime uint32
		want       bool
	}{
		{"inside the window, behind", 0, elapsed - 150, true},
		{"inside the window, ahead", 0, elapsed + 150, true},
		{"behind past the window", 0, elapsed - 151, false},
		{"ahead past the window", 0, elapsed + 151, false},
		{"earlier boot", -1, elapsed, false},
		{"later boot", 1, elapsed, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vault := &fakeCredentials{}
			vault.set(alicePriv())
			engine := &memEngine{boots: 4}
			clock := &manualClock{now: time.Date(2026, 10, 17, 1, 0, 0, 0, time.UTC)}
			addr, sink := startListenerWith(t, snmptrap.Config{Credentials: vault, Engine: engine, Now: clock.Now})
			clock.advance(elapsed * time.Second)
			id, boots := engine.identity()

			reply := rawInform(t, addr, alicePriv(), id, uint32(int(boots)+tc.bootsDelta), tc.engineTime)
			if reply == nil {
				t.Fatal("no reply")
			}
			usm := reply.SecurityParameters.(*gosnmp.UsmSecurityParameters)
			if usm.AuthoritativeEngineID != id || usm.AuthoritativeEngineBoots != boots ||
				usm.AuthoritativeEngineTime != elapsed {
				t.Errorf("reply engine %x boots %d time %d, want %x boots %d time %d",
					usm.AuthoritativeEngineID, usm.AuthoritativeEngineBoots, usm.AuthoritativeEngineTime,
					id, boots, elapsed)
			}
			if tc.want {
				assertAnswered(t, reply, sink)
			} else {
				assertNotInTimeWindow(t, reply, sink)
			}
		})
	}
}

func assertAnswered(t *testing.T, reply *gosnmp.SnmpPacket, sink *fakeSink) {
	t.Helper()
	if reply.PDUType != gosnmp.GetResponse || reply.RequestID != 42 {
		t.Errorf("reply = %v request %d, want GetResponse to 42", reply.PDUType, reply.RequestID)
	}
	if got := sink.wait(t, 1, 3*time.Second); len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
}

func assertNotInTimeWindow(t *testing.T, reply *gosnmp.SnmpPacket, sink *fakeSink) {
	t.Helper()
	if reply.PDUType != gosnmp.Report || reply.MsgFlags != gosnmp.AuthNoPriv ||
		len(reply.Variables) != 1 || reply.Variables[0].Name != ".1.3.6.1.6.3.15.1.1.2.0" {
		t.Errorf("reply = %v flags %v %+v, want an authNoPriv usmStatsNotInTimeWindows report",
			reply.PDUType, reply.MsgFlags, reply.Variables)
	}
	if got := sink.wait(t, 1, settle); len(got) != 0 {
		t.Fatalf("stale inform published: got %d events", len(got))
	}
}

// TestV3Inform_EngineSurvivesRestart: a restart keeps the engine ID and
// counts a boot, so a sender's localized keys stay valid and it resumes after
// one time window report.
func TestV3Inform_EngineSurvivesRestart(t *testing.T) {
	vault := &fakeCredentials{}
	vault.set(alicePriv())
	engine := &memEngine{}
	cfg := snmptrap.Config{
		BindAddr:    pickPort(t),
		Sink:        &fakeSink{},
		Credentials: vault,
		Engine:      engine,
		Logger:      silentLogger(),
	}
	l, err := snmptrap.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	if err = l.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	firstID, _ := engine.identity()
	if err = l.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err = l.Start(ctx); err != nil {
		t.Fatalf("restart: %v", err)
	}
	t.Cleanup(func() { _ = l.Stop(ctx) })
	id, boots := engine.identity()
	if id != firstID || boots != 2 {
		t.Fatalf("after restart engine %x boots %d, want %x boots 2", id, boots, firstID)
	}
	if _, err = sendInform(informSender(t, cfg.BindAddr, alicePriv(), gosnmp.AuthPriv)); err != nil {
		t.Fatalf("inform after restart: %v", err)
	}
}

// TestV3Inform_ResponseSaltsDiffer: Seed's encrypted responses under one key
// never reuse a salt, which would repeat the AES keystream (RFC 3826 3.1.2.1).
func TestV3Inform_ResponseSaltsDiffer(t *testing.T) {
	vault := &fakeCredentials{}
	vault.set(alicePriv())
	addr, _ := startListenerWith(t, snmptrap.Config{Credentials: vault, Engine: &memEngine{}})

	g := informSender(t, addr, alicePriv(), gosnmp.AuthPriv)
	seen := map[string]bool{}
	for range 3 {
		resp, err := sendInform(g)
		if err != nil {
			t.Fatalf("inform: %v", err)
		}
		salt := resp.SecurityParameters.(*gosnmp.UsmSecurityParameters).PrivacyParameters
		if seen[string(salt)] {
			t.Fatalf("salt %x reused", salt)
		}
		seen[string(salt)] = true
	}
}
