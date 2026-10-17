package delivery_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
)

// The SNMP trap channel (#3037). Its acceptance: an alert arrives at a trap
// receiver with the documented structure (docs/ALERT_TRAP.md and
// docs/mibs/SEED-ALERT-MIB.mib).

const trapCommunity = "seed-test-community"

// The SEED-ALERT-MIB OIDs, written out rather than shared with the package so
// a renumbering there fails here.
const (
	seedAlertRaised    = ".1.3.6.1.4.1.67249.1.1.1.0.1"
	seedAlertObjectArc = ".1.3.6.1.4.1.67249.1.1.1.1."
)

// trapReceiver is an SNMP trap receiver on loopback that keeps every datagram.
type trapReceiver struct {
	mu        sync.Mutex
	datagrams [][]byte
	port      int
}

func newTrapReceiver(t *testing.T) *trapReceiver {
	t.Helper()
	pc, err := (&net.ListenConfig{}).ListenPacket(context.Background(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	r := &trapReceiver{port: pc.LocalAddr().(*net.UDPAddr).Port}
	go func() {
		buf := make([]byte, 65535)
		for {
			n, _, readErr := pc.ReadFrom(buf)
			if readErr != nil {
				return
			}
			r.mu.Lock()
			r.datagrams = append(r.datagrams, append([]byte(nil), buf[:n]...))
			r.mu.Unlock()
		}
	}()
	return r
}

func (r *trapReceiver) received() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]byte(nil), r.datagrams...)
}

// decode parses the i-th datagram the way a receiver does, and fails unless it
// is an SNMPv2c trap under the test community.
func (r *trapReceiver) decode(t *testing.T, i int) *gosnmp.SnmpPacket {
	t.Helper()
	if !waitFor(t, func() bool { return len(r.received()) > i }) {
		t.Fatalf("receiver has %d datagrams, want %d", len(r.received()), i+1)
	}
	packet, err := (&gosnmp.GoSNMP{Version: gosnmp.Version2c}).UnmarshalTrap(r.received()[i], false)
	if err != nil {
		t.Fatalf("decode trap: %v", err)
	}
	if packet.Version != gosnmp.Version2c || packet.PDUType != gosnmp.SNMPv2Trap {
		t.Fatalf("got version %v PDU %v, want an SNMPv2c SNMPv2-Trap-PDU", packet.Version, packet.PDUType)
	}
	if packet.Community != trapCommunity {
		t.Fatalf("community = %q, want the configured one", packet.Community)
	}
	return packet
}

func trapConfig(r *trapReceiver, recorder delivery.Recorder) delivery.TrapConfig {
	return delivery.TrapConfig{
		Host:        "127.0.0.1",
		Port:        uint16(r.port),
		Community:   trapCommunity,
		Timeout:     5 * time.Second,
		Recorder:    recorder,
		MaxAttempts: 3,
		Backoff:     time.Millisecond,
		Logger:      quietLogger(),
	}
}

// object is the varbind value for one SEED-ALERT-MIB object.
type object struct {
	arc   string
	typ   gosnmp.Asn1BER
	value any
}

func TestTrapReachesTheReceiverWithTheDocumentedStructure(t *testing.T) {
	r := newTrapReceiver(t)
	recorder := &recordingRecorder{}
	cfg := trapConfig(r, recorder)
	cfg.Narrator = fakeNarrator{text: testStory(), ok: true}
	n, err := delivery.NewTrap(cfg)
	if err != nil {
		t.Fatalf("NewTrap: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	alert := testAlert()
	device := "core-sw-01"
	cause := int64(7)
	alert.DeviceID = &device
	alert.RootCauseID = &cause
	alert.Rule = "gateway.latency"
	n.Deliver(context.Background(), alert)

	if !waitFor(t, func() bool { w, ok := recorder.last(); return ok && w.status != alerts.DeliveryPending }) {
		t.Fatal("no delivery outcome was recorded")
	}
	if w, _ := recorder.last(); w.status != alerts.DeliveryDelivered || w.channel != alerts.ChannelTrap {
		t.Fatalf("recorded %+v, want delivered on the trap channel", w)
	}
	story := testStory()
	assertSeedAlertRaised(t, r.decode(t, 0), []object{
		{"1", gosnmp.Counter64, uint64(42)},
		{"2", gosnmp.Integer, 1},
		{"3", gosnmp.OctetString, alerts.TypePerformance},
		{"4", gosnmp.OctetString, "gateway.latency"},
		{"5", gosnmp.OctetString, "alert-observation-pipeline"},
		{"6", gosnmp.OctetString, "core-sw-01"},
		{"7", gosnmp.Counter64, uint64(7)},
		{"8", gosnmp.OctetString, "Gateway latency threshold breached"},
		{"9", gosnmp.OctetString, "gateway latency 812ms over the 200ms threshold"},
		{"10", gosnmp.OctetString, story.Summary},
		{"11", gosnmp.OctetString, story.NextCheck},
		{"12", gosnmp.OctetString, strings.Join(story.Evidence, "; ")},
	})
}

func TestTrapSendsEveryObjectForATestAlert(t *testing.T) {
	r := newTrapReceiver(t)
	n, err := delivery.NewTrap(trapConfig(r, nil))
	if err != nil {
		t.Fatalf("NewTrap: %v", err)
	}
	// The Settings test send: no ID, rule, device, cause or narrative. Every
	// object is still sent, so a receiver reading by position is not shifted.
	if err = n.SendNow(context.Background(), &alerts.Alert{
		Type: alerts.TypeSystem, Severity: alerts.SeverityInfo, Title: "Test alert", Source: "settings",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	assertSeedAlertRaised(t, r.decode(t, 0), []object{
		{"1", gosnmp.Counter64, uint64(0)},
		{"2", gosnmp.Integer, 4},
		{"3", gosnmp.OctetString, alerts.TypeSystem},
		{"4", gosnmp.OctetString, ""},
		{"5", gosnmp.OctetString, "settings"},
		{"6", gosnmp.OctetString, ""},
		{"7", gosnmp.Counter64, uint64(0)},
		{"8", gosnmp.OctetString, "Test alert"},
		{"9", gosnmp.OctetString, ""},
		{"10", gosnmp.OctetString, ""},
		{"11", gosnmp.OctetString, ""},
		{"12", gosnmp.OctetString, ""},
	})
}

// assertSeedAlertRaised checks the varbind list RFC 3416 §4.2.6 and
// SEED-ALERT-MIB require: sysUpTime.0, snmpTrapOID.0 naming seedAlertRaised,
// then each object at instance .0, in OBJECTS order.
func assertSeedAlertRaised(t *testing.T, packet *gosnmp.SnmpPacket, want []object) {
	t.Helper()
	vars := packet.Variables
	if len(vars) != 2+len(want) {
		t.Fatalf("%d varbinds, want %d", len(vars), 2+len(want))
	}
	if vars[0].Name != ".1.3.6.1.2.1.1.3.0" || vars[0].Type != gosnmp.TimeTicks {
		t.Errorf("varbind 1 = %s %v, want sysUpTime.0 TimeTicks", vars[0].Name, vars[0].Type)
	}
	if vars[1].Name != ".1.3.6.1.6.3.1.1.4.1.0" || vars[1].Value != seedAlertRaised {
		t.Errorf("varbind 2 = %s %v, want snmpTrapOID.0 = seedAlertRaised", vars[1].Name, vars[1].Value)
	}
	for i, w := range want {
		got := vars[i+2]
		value := got.Value
		if b, ok := value.([]byte); ok {
			value = string(b)
		}
		if got.Name != seedAlertObjectArc+w.arc+".0" || got.Type != w.typ || value != w.value {
			t.Errorf("varbind %d = %s %v %#v, want %s%s.0 %v %#v",
				i+3, got.Name, got.Type, value, seedAlertObjectArc, w.arc, w.typ, w.value)
		}
	}
}

func TestTrapSeverityFollowsTheMIB(t *testing.T) {
	r := newTrapReceiver(t)
	n, err := delivery.NewTrap(trapConfig(r, nil))
	if err != nil {
		t.Fatalf("NewTrap: %v", err)
	}
	for i, tc := range []struct {
		severity string
		want     int
	}{
		{alerts.SeverityCritical, 1},
		{alerts.SeverityError, 2},
		{alerts.SeverityWarning, 3},
		{alerts.SeverityInfo, 4},
		{"unheard-of", 3},
	} {
		alert := testAlert()
		alert.Severity = tc.severity
		if err = n.SendNow(context.Background(), alert); err != nil {
			t.Fatalf("send %s: %v", tc.severity, err)
		}
		if got := r.decode(t, i).Variables[3].Value; got != tc.want {
			t.Errorf("severity %q sent as %v, want %d", tc.severity, got, tc.want)
		}
	}
}

func TestTrapTextIsCutToAnAdminStringOnACharacterBoundary(t *testing.T) {
	r := newTrapReceiver(t)
	n, err := delivery.NewTrap(trapConfig(r, nil))
	if err != nil {
		t.Fatalf("NewTrap: %v", err)
	}
	alert := testAlert()
	// 254 ASCII octets and then a three-octet rune that would straddle 255.
	alert.Message = strings.Repeat("x", 254) + "€ and more"
	if err = n.SendNow(context.Background(), alert); err != nil {
		t.Fatalf("send: %v", err)
	}
	message, _ := r.decode(t, 0).Variables[10].Value.([]byte)
	if len(message) != 254 || !utf8.Valid(message) {
		t.Errorf("message is %d octets (valid UTF-8 %v), want the 254 before the rune that would pass 255",
			len(message), utf8.Valid(message))
	}
}

func TestTrapUnresolvableReceiverIsNamed(t *testing.T) {
	n, err := delivery.NewTrap(delivery.TrapConfig{
		Host: "receiver.invalid", Community: trapCommunity, Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewTrap: %v", err)
	}
	err = n.SendNow(context.Background(), testAlert())
	if err == nil || !strings.Contains(err.Error(), "receiver.invalid:162") {
		t.Fatalf("send error = %v, want one naming the receiver", err)
	}
}

func TestValidateTrapRefusesWhatCouldNeverDeliver(t *testing.T) {
	for name, tc := range map[string]struct {
		cfg  delivery.TrapConfig
		want string
	}{
		"no host":        {delivery.TrapConfig{Community: "c"}, "host is required"},
		"scheme":         {delivery.TrapConfig{Host: "udp://nms", Community: "c"}, "bare host name"},
		"port in host":   {delivery.TrapConfig{Host: "nms:162", Community: "c"}, "bare host name"},
		"no community":   {delivery.TrapConfig{Host: "nms"}, "community is required"},
		"valid ipv6":     {delivery.TrapConfig{Host: "2001:db8::1", Community: "c"}, ""},
		"valid and port": {delivery.TrapConfig{Host: "nms.example.com", Port: 1162, Community: "c"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			err := delivery.ValidateTrap(tc.cfg)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("ValidateTrap: %v", err)
				}
				return
			}
			if !errors.Is(err, delivery.ErrInvalidConfig) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateTrap = %v, want ErrInvalidConfig naming %q", err, tc.want)
			}
		})
	}
}

func TestTrapDefaultsToTheSnmptrapPort(t *testing.T) {
	n, err := delivery.NewTrap(delivery.TrapConfig{Host: "nms.example.com", Community: trapCommunity})
	if err != nil {
		t.Fatalf("NewTrap: %v", err)
	}
	if got := n.Endpoint(); got != "snmp://nms.example.com:162" {
		t.Errorf("endpoint = %q, want port 162", got)
	}
}
