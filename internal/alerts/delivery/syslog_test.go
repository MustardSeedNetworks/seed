package delivery_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"maps"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
)

// The syslog channel (#3037). Its acceptance: an alert arrives at a syslog
// collector with the documented structure (docs/ALERT_SYSLOG.md).

// collector is a syslog receiver on loopback for one transport: datagrams
// for udp, RFC 6587 octet-counted frames for tcp and tls.
type collector struct {
	transport delivery.SyslogTransport
	tlsConfig *tls.Config
	roots     *x509.CertPool

	mu       sync.Mutex
	messages []string
	accepts  int
	addr     net.Addr
}

func (c *collector) start(t *testing.T) {
	t.Helper()
	if c.transport == delivery.SyslogUDP {
		pc, err := (&net.ListenConfig{}).ListenPacket(context.Background(), "udp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen udp: %v", err)
		}
		t.Cleanup(func() { _ = pc.Close() })
		c.addr = pc.LocalAddr()
		go func() {
			buf := make([]byte, 65535)
			for {
				n, _, readErr := pc.ReadFrom(buf)
				if readErr != nil {
					return
				}
				c.add(string(buf[:n]))
			}
		}()
		return
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	if c.transport == delivery.SyslogTLS {
		ln = tls.NewListener(ln, c.tlsConfig)
	}
	t.Cleanup(func() { _ = ln.Close() })
	c.addr = ln.Addr()
	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			c.mu.Lock()
			c.accepts++
			c.mu.Unlock()
			go c.readFrames(conn)
		}
	}()
}

// readFrames reads "LEN SP MSG" frames until the sender closes.
func (c *collector) readFrames(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	for {
		prefix, err := r.ReadString(' ')
		if err != nil {
			return
		}
		size, err := strconv.Atoi(strings.TrimSuffix(prefix, " "))
		if err != nil {
			c.add("BAD FRAME " + prefix)
			return
		}
		msg := make([]byte, size)
		if _, err = io.ReadFull(r, msg); err != nil {
			return
		}
		c.add(string(msg))
	}
}

func (c *collector) add(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, msg)
}

func (c *collector) received() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.messages...)
}

func (c *collector) port() int {
	if udp, ok := c.addr.(*net.UDPAddr); ok {
		return udp.Port
	}
	return c.addr.(*net.TCPAddr).Port
}

func syslogConfig(c *collector, recorder delivery.Recorder) delivery.SyslogConfig {
	return delivery.SyslogConfig{
		Host:        "127.0.0.1",
		Port:        c.port(),
		Transport:   c.transport,
		Timeout:     5 * time.Second,
		Recorder:    recorder,
		MaxAttempts: 3,
		Backoff:     time.Millisecond,
		Logger:      quietLogger(),
		RootCAs:     c.roots,
	}
}

// newCollector starts a collector for transport, with a certificate that
// names 127.0.0.1 when it speaks TLS.
func newCollector(t *testing.T, transport delivery.SyslogTransport) *collector {
	t.Helper()
	c := &collector{transport: transport}
	if transport == delivery.SyslogTLS {
		c.tlsConfig, c.roots = sinkCertificate(t)
	}
	c.start(t)
	return c
}

// rfc5424 is the header the channel writes: PRI, VERSION 1, TIMESTAMP,
// HOSTNAME, APP-NAME seed, PROCID nil, MSGID alert, no structured data.
var rfc5424 = regexp.MustCompile(`^<(\d{1,3})>1 (\S+) (\S+) seed - alert - (.*)$`)

// parsed is one received message, split into its header and MSG pairs.
type parsed struct {
	pri       int
	timestamp string
	hostname  string
	pairs     map[string]string
}

func parseSyslog(t *testing.T, msg string) parsed {
	t.Helper()
	m := rfc5424.FindStringSubmatch(msg)
	if m == nil {
		t.Fatalf("not an RFC 5424 alert message: %q", msg)
	}
	pri, _ := strconv.Atoi(m[1])
	return parsed{pri: pri, timestamp: m[2], hostname: m[3], pairs: parseLogfmt(t, m[4])}
}

// parseLogfmt splits key=value pairs, a value either bare or a Go-quoted
// string, which is what a SIEM's logfmt extraction reads.
func parseLogfmt(t *testing.T, s string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for s != "" {
		eq := strings.IndexByte(s, '=')
		if eq <= 0 {
			t.Fatalf("no key=value at %q", s)
		}
		key := s[:eq]
		s = s[eq+1:]
		var value string
		if strings.HasPrefix(s, `"`) {
			quoted, err := strconv.QuotedPrefix(s)
			if err != nil {
				t.Fatalf("bad quoted value at %q: %v", s, err)
			}
			value, _ = strconv.Unquote(quoted)
			s = s[len(quoted):]
		} else {
			end := strings.IndexByte(s, ' ')
			if end < 0 {
				end = len(s)
			}
			value = s[:end]
			s = s[end:]
		}
		if _, dup := out[key]; dup {
			t.Fatalf("key %q appears twice", key)
		}
		out[key] = value
		s = strings.TrimPrefix(s, " ")
	}
	return out
}

func TestSyslogReachesTheCollectorWithTheDocumentedStructure(t *testing.T) {
	for _, transport := range []delivery.SyslogTransport{delivery.SyslogUDP, delivery.SyslogTCP, delivery.SyslogTLS} {
		t.Run(string(transport), func(t *testing.T) {
			c := newCollector(t, transport)
			recorder := &recordingRecorder{}
			n, err := delivery.NewSyslog(syslogConfig(c, recorder))
			if err != nil {
				t.Fatalf("NewSyslog: %v", err)
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
			if w, _ := recorder.last(); w.status != alerts.DeliveryDelivered || w.channel != alerts.ChannelSyslog {
				t.Fatalf("recorded %+v, want delivered on the syslog channel", w)
			}
			if !waitFor(t, func() bool { return len(c.received()) == 1 }) {
				t.Fatalf("collector received %d messages, want 1", len(c.received()))
			}
			assertDocumentedStructure(t, parseSyslog(t, c.received()[0]))
		})
	}
}

// assertDocumentedStructure checks one received alert against the structure
// docs/ALERT_SYSLOG.md promises a SIEM.
func assertDocumentedStructure(t *testing.T, got parsed) {
	t.Helper()
	// local0 (16) * 8 + critical (2).
	if got.pri != 130 {
		t.Errorf("PRI = %d, want 130 (local0.crit)", got.pri)
	}
	if got.timestamp != "2023-11-14T22:13:20.000000Z" {
		t.Errorf("TIMESTAMP = %q, want the alert's creation time", got.timestamp)
	}
	if got.hostname == "" {
		t.Error("HOSTNAME is empty")
	}
	want := map[string]string{
		"id":       "42",
		"severity": "critical",
		"type":     "performance",
		"rule":     "gateway.latency",
		"source":   "alert-observation-pipeline",
		"device":   "core-sw-01",
		"cause":    "7",
		"title":    "Gateway latency threshold breached",
		"message":  "gateway latency 812ms over the 200ms threshold",
	}
	if !maps.Equal(got.pairs, want) {
		t.Errorf("MSG pairs %v, want %v", got.pairs, want)
	}
}

func TestSyslogSeverityMapsOntoRFC5424(t *testing.T) {
	c := newCollector(t, delivery.SyslogTCP)
	n := syslogNotifier(t, c)

	for i, tc := range []struct {
		severity string
		pri      int
	}{
		{alerts.SeverityCritical, 130},
		{alerts.SeverityError, 131},
		{alerts.SeverityWarning, 132},
		{alerts.SeverityInfo, 134},
		// An unrecognised severity was still raised as an alert.
		{"bogus", 132},
	} {
		alert := testAlert()
		alert.Severity = tc.severity
		if err := n.SendNow(context.Background(), alert); err != nil {
			t.Fatalf("send %s: %v", tc.severity, err)
		}
		if !waitFor(t, func() bool { return len(c.received()) == i+1 }) {
			t.Fatalf("collector has %d messages, want %d", len(c.received()), i+1)
		}
		if got := parseSyslog(t, c.received()[i]).pri; got != tc.pri {
			t.Errorf("severity %q sent as PRI %d, want %d", tc.severity, got, tc.pri)
		}
	}
}

// syslogNotifier is an unstarted Notifier for c, for synchronous SendNow.
func syslogNotifier(t *testing.T, c *collector) *delivery.Notifier {
	t.Helper()
	n, err := delivery.NewSyslog(syslogConfig(c, nil))
	if err != nil {
		t.Fatalf("NewSyslog: %v", err)
	}
	return n
}

func TestSyslogAlertTextCannotForgeAMessage(t *testing.T) {
	c := newCollector(t, delivery.SyslogUDP)
	n := syslogNotifier(t, c)

	// A device controls the text of a syslog-sourced alert. A newline in it
	// must not end this message and start a forged one on a collector that
	// splits on lines, and an "=" must not add a field.
	alert := testAlert()
	alert.Message = "link down\n<10>1 2023-11-14T00:00:00Z fw seed - alert - severity=info title=\"all clear\""
	alert.Title = `admin" severity="info`
	if err := n.SendNow(context.Background(), alert); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !waitFor(t, func() bool { return len(c.received()) == 1 }) {
		t.Fatal("collector received nothing")
	}
	raw := c.received()[0]
	if strings.ContainsAny(raw, "\r\n") {
		t.Fatalf("message carries a line break: %q", raw)
	}
	got := parseSyslog(t, raw)
	if got.pairs["severity"] != "critical" {
		t.Errorf("severity = %q, want critical: alert text added a field", got.pairs["severity"])
	}
	if got.pairs["message"] != alert.Message || got.pairs["title"] != alert.Title {
		t.Errorf("text did not round-trip: %v", got.pairs)
	}
}

func TestSyslogDatagramIsBounded(t *testing.T) {
	c := newCollector(t, delivery.SyslogUDP)
	n := syslogNotifier(t, c)

	// A long device message must not become a datagram a collector may
	// refuse; RFC 5426 says every receiver SHOULD accept 2048 octets. The cut
	// must not split a multi-byte rune, which would make the tail invalid
	// UTF-8.
	alert := testAlert()
	alert.Message = strings.Repeat("é", 4000)
	if err := n.SendNow(context.Background(), alert); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !waitFor(t, func() bool { return len(c.received()) == 1 }) {
		t.Fatal("collector received nothing")
	}
	raw := c.received()[0]
	if len(raw) > 2048 || len(raw) < 2040 {
		t.Errorf("datagram is %d octets, want just under 2048", len(raw))
	}
	if !strings.HasSuffix(raw, "é") {
		t.Errorf("datagram ends mid-rune: %q", raw[len(raw)-4:])
	}
}

func TestSyslogUntrustedCertificateIsNamedAndNotRetried(t *testing.T) {
	c := newCollector(t, delivery.SyslogTLS)
	recorder := &recordingRecorder{}
	cfg := syslogConfig(c, recorder)
	// The system roots do not trust the collector's self-signed certificate.
	cfg.RootCAs = nil
	n, err := delivery.NewSyslog(cfg)
	if err != nil {
		t.Fatalf("NewSyslog: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())
	n.Deliver(context.Background(), testAlert())

	if !waitFor(t, func() bool { w, ok := recorder.last(); return ok && w.status != alerts.DeliveryPending }) {
		t.Fatal("no delivery outcome was recorded")
	}
	w, _ := recorder.last()
	if w.status != alerts.DeliveryFailed || !strings.Contains(w.errText, "certificate") {
		t.Fatalf("recorded %+v, want failed naming the certificate", w)
	}
	c.mu.Lock()
	accepts := c.accepts
	c.mu.Unlock()
	if accepts != 1 {
		t.Errorf("collector saw %d connections, want 1: a certificate the system does not trust "+
			"will not be trusted on the next try", accepts)
	}
	if len(c.received()) != 0 {
		t.Error("the alert reached a collector whose certificate was not verified")
	}
}

func TestSyslogUnreachableCollectorIsNamed(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	n, err := delivery.NewSyslog(delivery.SyslogConfig{
		Host: "127.0.0.1", Port: port, Transport: delivery.SyslogTCP, Logger: quietLogger(),
	})
	if err != nil {
		t.Fatalf("NewSyslog: %v", err)
	}
	err = n.SendNow(context.Background(), testAlert())
	if err == nil || !strings.Contains(err.Error(), "connect to 127.0.0.1:"+strconv.Itoa(port)) {
		t.Fatalf("SendNow = %v, want a connect error naming the collector", err)
	}
}

func TestValidateSyslogRefusesWhatCouldNeverDeliver(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  delivery.SyslogConfig
		ok   bool
	}{
		{"host name", delivery.SyslogConfig{Host: "siem.example.com"}, true},
		{"ipv4", delivery.SyslogConfig{Host: "10.0.0.5", Transport: delivery.SyslogTCP}, true},
		{"ipv6", delivery.SyslogConfig{Host: "2001:db8::5", Transport: delivery.SyslogTLS, Port: 6514}, true},
		{"no host", delivery.SyslogConfig{}, false},
		{"host with port", delivery.SyslogConfig{Host: "siem.example.com:514"}, false},
		{"scheme", delivery.SyslogConfig{Host: "udp://siem.example.com"}, false},
		{"port out of range", delivery.SyslogConfig{Host: "siem", Port: 70000}, false},
		{"unknown transport", delivery.SyslogConfig{Host: "siem", Transport: "relp"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := delivery.ValidateSyslog(tc.cfg)
			if tc.ok && err != nil {
				t.Fatalf("ValidateSyslog = %v, want nil", err)
			}
			if !tc.ok && !errors.Is(err, delivery.ErrInvalidConfig) {
				t.Fatalf("ValidateSyslog = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func TestSyslogDefaultPorts(t *testing.T) {
	for transport, want := range map[delivery.SyslogTransport]int{
		delivery.SyslogUDP: 514, delivery.SyslogTCP: 514, delivery.SyslogTLS: 6514,
	} {
		if got := transport.DefaultPort(); got != want {
			t.Errorf("%s default port %d, want %d", transport, got, want)
		}
	}
}
