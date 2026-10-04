package delivery

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
)

// The email channel (#2997): one message per alert through the operator's own
// mail relay.
//
// TLS is not optional. The message carries device names, addresses and the
// alert text, and AUTH carries a password; a relay that cannot encrypt is
// refused by name rather than sent to in the clear.

// TLSMode is how the connection to the mail server is encrypted.
type TLSMode string

// The supported modes. There is deliberately no plaintext mode.
const (
	// TLSStartTLS connects in plaintext and upgrades with STARTTLS before
	// anything else is said (RFC 3207), conventionally on port 587. A server
	// that does not offer STARTTLS is refused.
	TLSStartTLS TLSMode = "starttls"
	// TLSImplicit speaks TLS from the first byte (RFC 8314), conventionally on
	// port 465.
	TLSImplicit TLSMode = "tls"
)

// The submission ports RFC 8314 names for each mode.
const (
	portSubmission  = 587
	portSubmissions = 465
)

// replyPermanent is the first permanent-failure reply code (RFC 5321 §4.2.1):
// 5yz will be answered the same way again, 4yz may not.
const replyPermanent = 500

// DefaultPort is the conventional submission port for mode.
func (m TLSMode) DefaultPort() int {
	if m == TLSImplicit {
		return portSubmissions
	}
	return portSubmission
}

// Named failures an operator can act on without reading the protocol trace.
var (
	// ErrNoStartTLS means the server did not offer STARTTLS, so the message
	// and any credentials would have crossed the network in the clear.
	ErrNoStartTLS = errors.New("mail server does not offer STARTTLS; refusing to send unencrypted")
	// ErrNoAuth means a username is configured but the server offers no AUTH
	// mechanism, so it cannot be the relay the credentials were meant for.
	ErrNoAuth = errors.New("mail server does not offer AUTH but a username is configured")
)

// EmailConfig wires an email Notifier. Host, From and at least one To are
// required; the rest default.
type EmailConfig struct {
	Options

	// Host is the mail server's name. It is also the name its certificate is
	// verified against, so an IP literal works only with a certificate that
	// names it.
	Host string
	// Port defaults to TLS.DefaultPort().
	Port int
	// TLS defaults to TLSStartTLS.
	TLS TLSMode
	// Username and Password authenticate with AUTH PLAIN, over TLS only.
	// Empty Username means no AUTH, for a relay that admits by address.
	Username string
	Password string
	// From is the sender, an RFC 5322 address ("Seed <seed@example.com>").
	From string
	// To is every recipient.
	To []string
	// RootCAs verifies the server's certificate; nil means the system roots.
	RootCAs *x509.CertPool
	// Timeout bounds one attempt, connect to QUIT.
	Timeout time.Duration
}

// ValidateEmail rejects a configuration that could never deliver. Like
// ValidateURL, it is exported so the settings service refuses at the API what
// NewEmail would refuse at startup, by the same rule.
func ValidateEmail(cfg EmailConfig) error {
	_, _, err := parseEmail(cfg)
	return err
}

// parseEmail validates cfg and returns the parsed sender and recipients.
func parseEmail(cfg EmailConfig) (*mail.Address, []*mail.Address, error) {
	if cfg.Host == "" {
		return nil, nil, fmt.Errorf("%w: mail server host is required", ErrInvalidConfig)
	}
	if strings.ContainsAny(cfg.Host, " /:@") {
		return nil, nil, fmt.Errorf("%w: mail server host %q must be a bare host name, without scheme, port or user",
			ErrInvalidConfig, cfg.Host)
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		return nil, nil, fmt.Errorf("%w: mail server port %d is out of range", ErrInvalidConfig, cfg.Port)
	}
	switch cfg.TLS {
	case "", TLSStartTLS, TLSImplicit:
	default:
		return nil, nil, fmt.Errorf("%w: tls mode %q is not %q or %q",
			ErrInvalidConfig, cfg.TLS, TLSStartTLS, TLSImplicit)
	}
	if cfg.Username == "" && cfg.Password != "" {
		return nil, nil, fmt.Errorf("%w: a password is set without a username", ErrInvalidConfig)
	}
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: sender %q is not an email address: %w", ErrInvalidConfig, cfg.From, err)
	}
	if len(cfg.To) == 0 {
		return nil, nil, fmt.Errorf("%w: at least one recipient is required", ErrInvalidConfig)
	}
	to := make([]*mail.Address, 0, len(cfg.To))
	for _, raw := range cfg.To {
		addr, parseErr := mail.ParseAddress(raw)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("%w: recipient %q is not an email address: %w", ErrInvalidConfig, raw, parseErr)
		}
		to = append(to, addr)
	}
	return from, to, nil
}

// email sends one message per alert through one mail server.
type email struct {
	host     string
	addr     string
	mode     TLSMode
	tls      *tls.Config
	username string
	password string
	from     *mail.Address
	to       []*mail.Address
	helo     string
	timeout  time.Duration
	now      func() time.Time
}

// NewEmail validates cfg and builds an email Notifier. It makes no connection.
func NewEmail(cfg EmailConfig) (*Notifier, error) {
	from, to, err := parseEmail(cfg)
	if err != nil {
		return nil, err
	}
	mode := cfg.TLS
	if mode == "" {
		mode = TLSStartTLS
	}
	port := cfg.Port
	if port == 0 {
		port = mode.DefaultPort()
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	// The name Seed introduces itself by. Relays commonly refuse the
	// "localhost" net/smtp would otherwise send.
	helo, err := os.Hostname()
	if err != nil || helo == "" {
		helo = "localhost"
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	n := newNotifier(alerts.ChannelEmail, "smtp://"+addr, cfg.Options)
	n.transport = &email{
		host: cfg.Host,
		addr: addr,
		mode: mode,
		tls: &tls.Config{
			ServerName: cfg.Host,
			RootCAs:    cfg.RootCAs,
			MinVersion: tls.VersionTLS12,
		},
		username: cfg.Username,
		password: cfg.Password,
		from:     from,
		to:       to,
		helo:     helo,
		timeout:  timeout,
		now:      n.now,
	}
	return n, nil
}

// send makes one SMTP transaction. A 5xx reply, a certificate the system
// cannot verify, or a server missing STARTTLS or AUTH is permanent: the same
// server will answer the same way. A 4xx reply or a network failure is not.
func (e *email) send(ctx context.Context, alert *alerts.Alert, story *narrative.Text) (bool, error) {
	msg, err := e.compose(alert, story)
	if err != nil {
		return true, err
	}

	dialer := &net.Dialer{Timeout: e.timeout}
	var conn net.Conn
	if e.mode == TLSImplicit {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: e.tls}).DialContext(ctx, "tcp", e.addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", e.addr)
	}
	if err != nil {
		return isPermanentSMTP(err), fmt.Errorf("connect to %s: %w", e.addr, err)
	}
	// net/smtp takes no context: the deadline bounds a server that stops
	// answering, and closing the connection on cancel bounds shutdown.
	_ = conn.SetDeadline(time.Now().Add(e.timeout))
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	c, err := smtp.NewClient(conn, e.host)
	if err != nil {
		return isPermanentSMTP(err), fmt.Errorf("greeting from %s: %w", e.addr, err)
	}
	defer func() { _ = c.Close() }()

	if err = e.converse(c, msg); err != nil {
		return isPermanentSMTP(err), err
	}
	return false, nil
}

// converse runs the SMTP dialogue after the greeting. Each error names the
// step, so "AUTH: 535 5.7.8 ..." reads as a credentials problem without the
// protocol trace.
func (e *email) converse(c *smtp.Client, msg []byte) error {
	if err := c.Hello(e.helo); err != nil {
		return &stepError{"EHLO", err}
	}
	if e.mode == TLSStartTLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return ErrNoStartTLS
		}
		if err := c.StartTLS(e.tls); err != nil {
			return &stepError{"STARTTLS", err}
		}
	}
	if e.username != "" {
		if ok, _ := c.Extension("AUTH"); !ok {
			return ErrNoAuth
		}
		if err := c.Auth(smtp.PlainAuth("", e.username, e.password, e.host)); err != nil {
			return &stepError{"AUTH", err}
		}
	}
	if err := c.Mail(e.from.Address); err != nil {
		return &stepError{"MAIL FROM " + e.from.Address, err}
	}
	for _, rcpt := range e.to {
		if err := c.Rcpt(rcpt.Address); err != nil {
			return &stepError{"RCPT TO " + rcpt.Address, err}
		}
	}
	w, err := c.Data()
	if err != nil {
		return &stepError{"DATA", err}
	}
	if _, err = w.Write(msg); err != nil {
		return &stepError{"DATA", err}
	}
	if err = w.Close(); err != nil {
		return &stepError{"DATA", err}
	}
	// The message was accepted at the end of DATA; a server that drops the
	// connection instead of answering QUIT has still delivered it.
	_ = c.Quit()
	return nil
}

// stepError names the SMTP step that failed and prints a server reply the way
// the server sent it ("535 5.7.8 ..."), which is what an operator searches
// their relay's documentation for.
type stepError struct {
	step string
	err  error
}

func (e *stepError) Error() string {
	if reply, ok := errors.AsType[*textproto.Error](e.err); ok {
		return fmt.Sprintf("%s: %d %s", e.step, reply.Code, reply.Msg)
	}
	return e.step + ": " + e.err.Error()
}

func (e *stepError) Unwrap() error { return e.err }

// isPermanentSMTP reports whether retrying err would get the same answer.
func isPermanentSMTP(err error) bool {
	if errors.Is(err, ErrNoStartTLS) || errors.Is(err, ErrNoAuth) {
		return true
	}
	if reply, ok := errors.AsType[*textproto.Error](err); ok {
		return reply.Code >= replyPermanent
	}
	return isCertificateError(err)
}

// compose renders alert as an RFC 5322 message. Every header value that came
// from the alert is reduced to one line before it is written, so text a
// device controls (a syslog message, a hostname) cannot add a header.
func (e *email) compose(alert *alerts.Alert, story *narrative.Text) ([]byte, error) {
	recipients := make([]string, len(e.to))
	for i, addr := range e.to {
		recipients[i] = addr.String()
	}
	subject := fmt.Sprintf("[Seed] %s: %s", strings.ToUpper(alert.Severity), alert.Title)
	// An escalation is the same alert sent again because nobody acknowledged
	// it; the subject says so, or it reads as a duplicate.
	if alert.EscalationStage > 0 {
		subject = fmt.Sprintf("[Seed] %s, unacknowledged, escalation %d: %s",
			strings.ToUpper(alert.Severity), alert.EscalationStage, alert.Title)
	}
	created := alert.CreatedAt
	if created.IsZero() {
		created = e.now()
	}

	var msg bytes.Buffer
	header := func(name, value string) {
		msg.WriteString(name + ": " + value + "\r\n")
	}
	header("From", e.from.String())
	header("To", strings.Join(recipients, ", "))
	header("Subject", mime.QEncoding.Encode("utf-8", oneLine(subject)))
	header("Date", e.now().Format(time.RFC1123Z))
	header("Message-ID", fmt.Sprintf("<seed-alert-%d.%d@%s>", alert.ID, e.now().UnixNano(), domainOf(e.from.Address)))
	// A test-send alert was never stored, so it has no id to name.
	if alert.ID != 0 {
		header("X-Seed-Alert-Id", strconv.FormatInt(alert.ID, 10))
	}
	// RFC 3834: tells the recipient's mail system not to auto-reply, so an
	// out-of-office answer does not bounce back at the relay for every alert.
	header("Auto-Submitted", "auto-generated")
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=utf-8")
	header("Content-Transfer-Encoding", "quoted-printable")
	msg.WriteString("\r\n")

	qp := quotedprintable.NewWriter(&msg)
	if _, err := qp.Write([]byte(body(alert, story, created))); err != nil {
		return nil, fmt.Errorf("encode alert %d: %w", alert.ID, err)
	}
	if err := qp.Close(); err != nil {
		return nil, fmt.Errorf("encode alert %d: %w", alert.ID, err)
	}
	return msg.Bytes(), nil
}

// body is the plain-text message: the narrative, when the alert has one, then
// the facts an on-call engineer needs to decide whether to get up, then the
// alert's own text.
func body(alert *alerts.Alert, story *narrative.Text, created time.Time) string {
	var b strings.Builder
	b.WriteString(alert.Title + "\n\n")
	if story != nil {
		b.WriteString(story.Summary + "\n\nEvidence:\n")
		for _, line := range story.Evidence {
			b.WriteString("- " + line + "\n")
		}
		b.WriteString("\nNext check: " + story.NextCheck + "\n\n")
	}
	line := func(label, value string) {
		if value != "" {
			b.WriteString(label + ": " + value + "\n")
		}
	}
	line("Severity", alert.Severity)
	line("Type", alert.Type)
	line("Source", alert.Source)
	if alert.DeviceID != nil {
		line("Device", *alert.DeviceID)
	}
	line("Rule", alert.Rule)
	line("Raised", created.UTC().Format(time.RFC3339))
	if alert.EscalationStage > 0 {
		line("Escalation", "stage "+strconv.Itoa(alert.EscalationStage)+", not acknowledged")
	}
	if alert.RootCauseID != nil {
		line("Probable cause", "alert "+strconv.FormatInt(*alert.RootCauseID, 10))
	}
	if alert.ID != 0 {
		line("Alert ID", strconv.FormatInt(alert.ID, 10))
	}
	if alert.Message != "" {
		b.WriteString("\n" + alert.Message + "\n")
	}
	return b.String()
}

// oneLine collapses every run of whitespace, CR and LF included, to a space.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// domainOf is the part of an address after the last @.
func domainOf(address string) string {
	return address[strings.LastIndexByte(address, '@')+1:]
}
