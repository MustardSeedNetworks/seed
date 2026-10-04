package delivery_test

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// smtpScript is how an smtpSink misbehaves. The zero value is a well-behaved
// STARTTLS server that offers no AUTH.
type smtpScript struct {
	// implicit speaks TLS from the first byte.
	implicit bool
	// noStartTLS leaves STARTTLS out of the EHLO reply.
	noStartTLS bool
	// user and pass, when user is set, are the only credentials AUTH accepts.
	user, pass string
	// replies overrides the reply to a command verb ("MAIL", "RCPT", "DATA").
	replies map[string]string
}

// smtpSink is a mail server just real enough to receive what the email
// channel sends: EHLO, STARTTLS, AUTH PLAIN, MAIL, RCPT, DATA, QUIT, over a
// certificate the test controls.
type smtpSink struct {
	smtpScript

	ln       net.Listener
	tls      *tls.Config
	roots    *x509.CertPool
	sessions atomic.Int64

	mu       sync.Mutex
	messages []sunkMessage
}

// sunkMessage is one accepted DATA and the envelope around it.
type sunkMessage struct {
	from     string
	to       []string
	data     string
	tls      bool
	authUser string
}

// start listens on loopback and serves until the test ends.
func (s *smtpSink) start(t *testing.T) {
	t.Helper()
	s.tls, s.roots = sinkCertificate(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s.ln = ln
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			s.sessions.Add(1)
			go s.serve(conn)
		}
	}()
}

func (s *smtpSink) port() int { return s.ln.Addr().(*net.TCPAddr).Port }

func (s *smtpSink) received() []sunkMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]sunkMessage(nil), s.messages...)
}

// session is one client connection's state.
type session struct {
	conn     net.Conn
	text     *textproto.Conn
	secure   bool
	authUser string
	msg      sunkMessage
}

func (ss *session) reply(line string) { _ = ss.text.PrintfLine("%s", line) }

func (s *smtpSink) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	ss := &session{conn: conn}
	if s.implicit {
		ss.conn = tls.Server(conn, s.tls)
		ss.secure = true
	}
	ss.text = textproto.NewConn(ss.conn)
	ss.reply("220 sink.test ESMTP")
	for {
		line, err := ss.text.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		verb = strings.ToUpper(verb)
		if override, ok := s.replies[verb]; ok {
			ss.reply(override)
			continue
		}
		if !s.command(ss, verb, arg) {
			return
		}
	}
}

// command answers one SMTP command and reports whether the session goes on.
func (s *smtpSink) command(ss *session, verb, arg string) bool {
	switch verb {
	case "EHLO":
		s.ehlo(ss)
	case "STARTTLS":
		ss.reply("220 2.0.0 ready")
		tlsConn := tls.Server(ss.conn, s.tls)
		if tlsConn.Handshake() != nil {
			return false
		}
		ss.conn = tlsConn
		ss.text = textproto.NewConn(tlsConn)
		ss.secure = true
	case "AUTH":
		s.auth(ss, arg)
	case "MAIL":
		ss.msg = sunkMessage{from: addrOf(arg), tls: ss.secure, authUser: ss.authUser}
		ss.reply("250 2.1.0 ok")
	case "RCPT":
		ss.msg.to = append(ss.msg.to, addrOf(arg))
		ss.reply("250 2.1.5 ok")
	case "DATA":
		ss.reply("354 go ahead")
		data, err := ss.text.ReadDotBytes()
		if err != nil {
			return false
		}
		ss.msg.data = string(data)
		s.mu.Lock()
		s.messages = append(s.messages, ss.msg)
		s.mu.Unlock()
		ss.reply("250 2.0.0 queued")
	case "QUIT":
		ss.reply("221 2.0.0 bye")
		return false
	default:
		ss.reply("250 ok")
	}
	return true
}

func (s *smtpSink) ehlo(ss *session) {
	lines := []string{"250-sink.test"}
	if !ss.secure && !s.noStartTLS {
		lines = append(lines, "250-STARTTLS")
	}
	if ss.secure && s.user != "" {
		lines = append(lines, "250-AUTH PLAIN")
	}
	for _, l := range append(lines, "250 8BITMIME") {
		ss.reply(l)
	}
}

func (s *smtpSink) auth(ss *session, arg string) {
	decoded, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(arg, "PLAIN "))
	parts := strings.Split(string(decoded), "\x00")
	if len(parts) == 3 && parts[1] == s.user && parts[2] == s.pass {
		ss.authUser = parts[1]
		ss.reply("235 2.7.0 accepted")
		return
	}
	ss.reply("535 5.7.8 authentication credentials invalid")
}

// addrOf pulls the address out of "FROM:<a@b>" / "TO:<a@b>".
func addrOf(arg string) string {
	_, rest, _ := strings.Cut(arg, "<")
	addr, _, _ := strings.Cut(rest, ">")
	return addr
}

// sinkCertificate is a fresh self-signed certificate for 127.0.0.1 and the
// pool that trusts it.
func sinkCertificate(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sink.test"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:         true,

		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
		MinVersion:   tls.VersionTLS12,
	}, pool
}

// readMessage splits a received DATA into its headers and body. ReadDotBytes
// has already turned every CRLF into LF.
func readMessage(t *testing.T, data string) (textproto.MIMEHeader, string) {
	t.Helper()
	r := textproto.NewReader(bufio.NewReader(strings.NewReader(data)))
	header, err := r.ReadMIMEHeader()
	if err != nil {
		t.Fatalf("parse message header: %v\n%s", err, data)
	}
	_, body, _ := strings.Cut(data, "\n\n")
	return header, body
}
