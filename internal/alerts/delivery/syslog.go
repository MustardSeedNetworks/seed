package delivery

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// The syslog channel (#3037): one RFC 5424 message per alert to the
// operator's collector, which is how a SIEM expects to be fed.
//
// The message carries no structured data. RFC 5424 §6.3.2 reserves SD-IDs
// without an "@" for IANA registration and requires every other one to name
// the sender's Private Enterprise Number, which Mustard Seed Networks does not
// have; an SD-ID under someone else's number would be a lie a collector
// believes. The alert's facts travel in MSG as logfmt key=value pairs instead,
// which every SIEM's field extraction already parses.

// SyslogTransport is how messages reach the collector.
type SyslogTransport string

// The supported transports.
const (
	// SyslogUDP sends one datagram per alert (RFC 5426). Nothing acknowledges
	// it, so a collector that is down loses the alert without an error.
	SyslogUDP SyslogTransport = "udp"
	// SyslogTCP frames each message with its octet count (RFC 6587 §3.4.1).
	SyslogTCP SyslogTransport = "tcp"
	// SyslogTLS is SyslogTCP inside TLS (RFC 5425), the certificate verified
	// against Host.
	SyslogTLS SyslogTransport = "tls"
)

// The ports IANA assigns to each transport.
const (
	portSyslog    = 514
	portSyslogTLS = 6514
)

// DefaultPort is the transport's IANA port.
func (t SyslogTransport) DefaultPort() int {
	if t == SyslogTLS {
		return portSyslogTLS
	}
	return portSyslog
}

const (
	// facilityLocal0 is the facility every alert is sent with. A collector
	// routes Seed's alerts by APP-NAME "seed"; local0 keeps them out of the
	// facilities the collector's own host logs to.
	facilityLocal0 = 16
	// syslogVersion is the RFC 5424 VERSION field.
	syslogVersion = 1
	// maxDatagram is the largest message sent over UDP: the size RFC 5426
	// §3.2 says every receiver SHOULD accept.
	maxDatagram = 2048
	// maxHostname is RFC 5424's limit on the HOSTNAME field.
	maxHostname = 255
	// syslogAppName and syslogMsgID let a collector select Seed's alerts
	// without parsing MSG.
	syslogAppName = "seed"
	syslogMsgID   = "alert"
	nilValue      = "-"
)

// The RFC 5424 severities alerts map onto.
const (
	syslogCritical      = 2
	syslogError         = 3
	syslogWarning       = 4
	syslogInformational = 6
)

// network is the net package network the transport dials.
func (t SyslogTransport) network() string {
	if t == SyslogUDP {
		return "udp"
	}
	return "tcp"
}

// SyslogConfig wires a syslog Notifier. Host is required; the rest default.
type SyslogConfig struct {
	Options

	// Host is the collector's name or address. Under SyslogTLS it is also the
	// name the certificate is verified against.
	Host string
	// Port defaults to Transport.DefaultPort().
	Port int
	// Transport defaults to SyslogUDP, what nearly every collector listens
	// on out of the box.
	Transport SyslogTransport
	// RootCAs verifies the collector's certificate; nil means the system
	// roots.
	RootCAs *x509.CertPool
	// Timeout bounds one attempt.
	Timeout time.Duration
}

// ValidateSyslog rejects a configuration that could never deliver, by the rule
// NewSyslog applies, so the settings service refuses it at the API.
func ValidateSyslog(cfg SyslogConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("%w: syslog collector host is required", ErrInvalidConfig)
	}
	if strings.ContainsAny(cfg.Host, " /@") || (strings.Contains(cfg.Host, ":") && net.ParseIP(cfg.Host) == nil) {
		return fmt.Errorf("%w: syslog collector host %q must be a bare host name or address, without scheme or port",
			ErrInvalidConfig, cfg.Host)
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		return fmt.Errorf("%w: syslog collector port %d is out of range", ErrInvalidConfig, cfg.Port)
	}
	switch cfg.Transport {
	case "", SyslogUDP, SyslogTCP, SyslogTLS:
		return nil
	default:
		return fmt.Errorf("%w: syslog transport %q is not %q, %q or %q",
			ErrInvalidConfig, cfg.Transport, SyslogUDP, SyslogTCP, SyslogTLS)
	}
}

// syslogSender sends one message per alert to one collector.
type syslogSender struct {
	addr      string
	transport SyslogTransport
	tls       *tls.Config
	hostname  string
	timeout   time.Duration
	now       func() time.Time
}

// NewSyslog validates cfg and builds a syslog Notifier. It makes no
// connection.
func NewSyslog(cfg SyslogConfig) (*Notifier, error) {
	if err := ValidateSyslog(cfg); err != nil {
		return nil, err
	}
	transport := cfg.Transport
	if transport == "" {
		transport = SyslogUDP
	}
	port := cfg.Port
	if port == 0 {
		port = transport.DefaultPort()
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	// An unknown hostname is sent as the nil value, which RFC 5424 allows.
	hostname, _ := os.Hostname()
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	n := newNotifier(alerts.ChannelSyslog, string(transport)+"://"+addr, cfg.Options)
	n.transport = &syslogSender{
		addr:      addr,
		transport: transport,
		tls: &tls.Config{
			ServerName: cfg.Host,
			RootCAs:    cfg.RootCAs,
			MinVersion: tls.VersionTLS12,
		},
		hostname: headerField(hostname, maxHostname),
		timeout:  timeout,
		now:      n.now,
	}
	return n, nil
}

// send makes one delivery. Only a certificate the collector's name cannot be
// verified against is permanent; a refused connection or a collector that
// stops reading may answer differently on the next try.
func (s *syslogSender) send(ctx context.Context, alert *alerts.Alert) (bool, error) {
	msg := s.format(alert)
	dialer := &net.Dialer{Timeout: s.timeout}
	var conn net.Conn
	var err error
	if s.transport == SyslogTLS {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: s.tls}).DialContext(ctx, "tcp", s.addr)
	} else {
		conn, err = dialer.DialContext(ctx, s.transport.network(), s.addr)
	}
	if err != nil {
		return isCertificateError(err), fmt.Errorf("connect to %s: %w", s.addr, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(s.timeout))

	var frame []byte
	if s.transport == SyslogUDP {
		frame = truncateUTF8(msg, maxDatagram)
	} else {
		frame = append([]byte(strconv.Itoa(len(msg))+" "), msg...)
	}
	if _, err = conn.Write(frame); err != nil {
		return false, fmt.Errorf("send to %s: %w", s.addr, err)
	}
	return false, nil
}

// format renders alert as an RFC 5424 message:
//
//	<130>1 2023-11-14T22:13:20.000000Z probe-01 seed - alert - id=42 severity=critical ...
func (s *syslogSender) format(alert *alerts.Alert) []byte {
	created := alert.CreatedAt
	if created.IsZero() {
		created = s.now()
	}
	header := fmt.Sprintf("<%d>%d %s %s %s %s %s %s ",
		facilityLocal0*8+syslogSeverity(alert.Severity), syslogVersion,
		created.UTC().Format("2006-01-02T15:04:05.000000Z07:00"),
		s.hostname, syslogAppName, nilValue, syslogMsgID, nilValue)

	var pairs []string
	pair := func(key, value string) {
		if value != "" {
			pairs = append(pairs, key+"="+logfmtValue(value))
		}
	}
	if alert.ID != 0 {
		pair("id", strconv.FormatInt(alert.ID, 10))
	}
	pair("severity", alert.Severity)
	pair("type", alert.Type)
	pair("rule", alert.Rule)
	pair("source", alert.Source)
	if alert.DeviceID != nil {
		pair("device", *alert.DeviceID)
	}
	if alert.RootCauseID != nil {
		pair("cause", strconv.FormatInt(*alert.RootCauseID, 10))
	}
	pair("title", alert.Title)
	pair("message", alert.Message)
	return []byte(header + strings.Join(pairs, " "))
}

// syslogSeverity maps an alert severity onto RFC 5424's scale. An unknown
// severity is sent as warning: it was raised as an alert, so it is not
// informational, and nothing says it is worse.
func syslogSeverity(severity string) int {
	switch severity {
	case alerts.SeverityCritical:
		return syslogCritical
	case alerts.SeverityError:
		return syslogError
	case alerts.SeverityInfo:
		return syslogInformational
	default:
		return syslogWarning
	}
}

// logfmtValue writes value bare when it is one plain token and quoted
// otherwise. Quoting escapes CR and LF, so text a device controls cannot end
// the line and start a forged message on a collector that splits on newlines.
func logfmtValue(value string) string {
	if strings.ContainsFunc(value, func(r rune) bool {
		return r <= ' ' || r == 0x7f || r == '"' || r == '=' || r == '\\'
	}) {
		return strconv.Quote(value)
	}
	return value
}

// headerField reduces s to the printable ASCII an RFC 5424 header field
// allows, at most limit octets, or the nil value when nothing is left.
func headerField(s string, limit int) string {
	field := strings.Map(func(r rune) rune {
		if r > ' ' && r <= '~' {
			return r
		}
		return -1
	}, s)
	if len(field) > limit {
		field = field[:limit]
	}
	if field == "" {
		return nilValue
	}
	return field
}

// truncateUTF8 cuts msg to at most limit octets without splitting a rune.
func truncateUTF8(msg []byte, limit int) []byte {
	if len(msg) <= limit {
		return msg
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(msg[cut]) {
		cut--
	}
	return msg[:cut]
}
