package delivery_test

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
)

// The email channel (#2997). Its acceptance: an alert reaches a local SMTP
// sink with correct headers and body, and a misconfigured server surfaces a
// named error rather than silence.

const (
	sinkUser = "seed-alerts"
	sinkPass = "relay-password"
)

// emailConfig points at sink with the credentials it accepts and fast retry.
func emailConfig(sink *smtpSink, recorder delivery.Recorder) delivery.EmailConfig {
	mode := delivery.TLSStartTLS
	if sink.implicit {
		mode = delivery.TLSImplicit
	}
	return delivery.EmailConfig{
		Host:        "127.0.0.1",
		Port:        sink.port(),
		TLS:         mode,
		Username:    sinkUser,
		Password:    sinkPass,
		From:        "Seed <seed@example.test>",
		To:          []string{"noc@example.test", "On Call <oncall@example.test>"},
		RootCAs:     sink.roots,
		Timeout:     5 * time.Second,
		Recorder:    recorder,
		MaxAttempts: 3,
		Backoff:     time.Millisecond,
		Logger:      quietLogger(),
	}
}

func TestEmailReachesTheSinkWithHeadersAndBody(t *testing.T) {
	for _, implicit := range []bool{false, true} {
		t.Run("implicit="+strconv.FormatBool(implicit), func(t *testing.T) {
			sink := &smtpSink{implicit: implicit, user: sinkUser, pass: sinkPass}
			sink.start(t)
			recorder := &recordingRecorder{}
			n, err := delivery.NewEmail(emailConfig(sink, recorder))
			if err != nil {
				t.Fatalf("NewEmail: %v", err)
			}
			n.Start()
			defer n.Stop(context.Background())

			alert := testAlert()
			device := "core-sw-01"
			alert.DeviceID = &device
			alert.Rule = "gateway.latency"
			n.Deliver(context.Background(), alert)

			if !waitFor(t, func() bool { w, ok := recorder.last(); return ok && w.status != alerts.DeliveryPending }) {
				t.Fatal("no delivery outcome was recorded")
			}
			if w, _ := recorder.last(); w.status != alerts.DeliveryDelivered || w.channel != alerts.ChannelEmail {
				t.Fatalf("recorded %+v, want delivered on the email channel", w)
			}

			got := sink.received()
			if len(got) != 1 {
				t.Fatalf("sink received %d messages, want 1", len(got))
			}
			assertAlertMessage(t, got[0])
		})
	}
}

// assertAlertMessage checks one received alert end to end: the envelope it
// travelled in, the headers a mail client and a mail filter read, and the
// body an on-call engineer reads.
func assertAlertMessage(t *testing.T, m sunkMessage) {
	t.Helper()
	if !m.tls {
		t.Error("the message crossed the network unencrypted")
	}
	if m.authUser != sinkUser {
		t.Errorf("authenticated as %q, want %q", m.authUser, sinkUser)
	}
	if m.from != "seed@example.test" {
		t.Errorf("envelope sender %q", m.from)
	}
	if strings.Join(m.to, ",") != "noc@example.test,oncall@example.test" {
		t.Errorf("envelope recipients %v", m.to)
	}

	header, rawBody := readMessage(t, m.data)
	for name, want := range map[string]string{
		"From":                      `"Seed" <seed@example.test>`,
		"To":                        `<noc@example.test>, "On Call" <oncall@example.test>`,
		"Subject":                   "[Seed] CRITICAL: Gateway latency threshold breached",
		"X-Seed-Alert-Id":           "42",
		"Auto-Submitted":            "auto-generated",
		"Content-Type":              "text/plain; charset=utf-8",
		"Content-Transfer-Encoding": "quoted-printable",
	} {
		if got := header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if id := header.Get("Message-Id"); !strings.HasPrefix(id, "<seed-alert-42.") ||
		!strings.HasSuffix(id, "@example.test>") {
		t.Errorf("Message-ID = %q", id)
	}
	if _, err := time.Parse(time.RFC1123Z, header.Get("Date")); err != nil {
		t.Errorf("Date %q is not RFC 1123: %v", header.Get("Date"), err)
	}

	body, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(rawBody)))
	if err != nil {
		t.Fatalf("decode body: %v", err)
	}
	for _, want := range []string{
		"Gateway latency threshold breached",
		"Severity: critical",
		"Device: core-sw-01",
		"Rule: gateway.latency",
		"Raised: 2023-11-14T22:13:20Z",
		"Alert ID: 42",
		"gateway latency 812ms over the 200ms threshold",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("body lacks %q:\n%s", want, body)
		}
	}
}

// A device controls the text of many alerts (a syslog line, a hostname), so
// that text must never be able to add a header — a Bcc would copy every
// alert to an outsider.
func TestEmailAlertTextCannotAddAHeader(t *testing.T) {
	sink := &smtpSink{user: sinkUser, pass: sinkPass}
	sink.start(t)
	n, err := delivery.NewEmail(emailConfig(sink, nil))
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}

	alert := testAlert()
	alert.Title = "link down\r\nBcc: outsider@example.net\r\n\r\nforged body"
	alert.Severity = "warning\nX-Injected: yes"
	if err = n.SendNow(context.Background(), alert); err != nil {
		t.Fatalf("SendNow: %v", err)
	}

	header, _ := readMessage(t, sink.received()[0].data)
	if header.Get("Bcc") != "" || header.Get("X-Injected") != "" {
		t.Fatalf("alert text added a header: %v", header)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	if want := "[Seed] WARNING X-INJECTED: YES: link down Bcc: outsider@example.net forged body"; subject != want {
		t.Errorf("Subject = %q, want %q", subject, want)
	}
	if got := sink.received()[0].to; len(got) != 2 {
		t.Errorf("envelope recipients %v; the alert text must not reach RCPT", got)
	}
}

// namedFailure is a misbehaving mail server and what the operator should read.
type namedFailure struct {
	script    smtpScript
	configure func(*delivery.EmailConfig)
	want      string
	wantErr   error
	// sessions is how many connections the bounded retry should make:
	// one for a failure the server will repeat, MaxAttempts for one it
	// may not.
	sessions int64
}

func TestEmailMisconfiguredServerIsNamed(t *testing.T) {
	for name, tc := range map[string]namedFailure{
		"no STARTTLS offered": {
			script:   smtpScript{noStartTLS: true},
			wantErr:  delivery.ErrNoStartTLS,
			sessions: 1,
		},
		"wrong password": {
			script:    smtpScript{user: sinkUser, pass: sinkPass},
			configure: func(c *delivery.EmailConfig) { c.Password = "not-it" },
			want:      "AUTH: 535 5.7.8",
			sessions:  1,
		},
		"username for a relay without AUTH": {
			script:   smtpScript{},
			wantErr:  delivery.ErrNoAuth,
			sessions: 1,
		},
		"recipient refused": {
			script: smtpScript{
				user: sinkUser, pass: sinkPass,
				replies: map[string]string{"RCPT": "550 5.1.1 no such mailbox"},
			},
			want:     "RCPT TO noc@example.test: 550 5.1.1 no such mailbox",
			sessions: 1,
		},
		"certificate not trusted": {
			script:    smtpScript{user: sinkUser, pass: sinkPass},
			configure: func(c *delivery.EmailConfig) { c.RootCAs = nil },
			want:      "STARTTLS: tls: failed to verify certificate",
			sessions:  1,
		},
		"temporary refusal is retried": {
			script: smtpScript{
				user: sinkUser, pass: sinkPass,
				replies: map[string]string{"MAIL": "451 4.3.0 try again later"},
			},
			want:     "MAIL FROM seed@example.test: 451 4.3.0 try again later",
			sessions: 3,
		},
	} {
		t.Run(name, func(t *testing.T) { checkNamedFailure(t, tc) })
	}
}

// checkNamedFailure proves the failure reaches the operator on both paths:
// the test-send answer and the reason written onto the alert.
func checkNamedFailure(t *testing.T, tc namedFailure) {
	t.Helper()
	sink := &smtpSink{smtpScript: tc.script}
	sink.start(t)
	recorder := &recordingRecorder{}
	cfg := emailConfig(sink, recorder)
	if tc.configure != nil {
		tc.configure(&cfg)
	}
	n, err := delivery.NewEmail(cfg)
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}

	// The test-send path returns the error to the operator directly.
	sendErr := n.SendNow(context.Background(), testAlert())
	if sendErr == nil {
		t.Fatal("SendNow succeeded against a misconfigured server")
	}
	if tc.wantErr != nil && !errors.Is(sendErr, tc.wantErr) {
		t.Errorf("SendNow error = %v, want %v", sendErr, tc.wantErr)
	}
	if tc.want != "" && !strings.Contains(sendErr.Error(), tc.want) {
		t.Errorf("SendNow error = %q, want it to contain %q", sendErr, tc.want)
	}

	// The delivery path writes the same reason onto the alert.
	sink.sessions.Store(0)
	n.Start()
	defer n.Stop(context.Background())
	n.Deliver(context.Background(), testAlert())
	if !waitFor(t, func() bool { w, ok := recorder.last(); return ok && w.status == alerts.DeliveryFailed }) {
		t.Fatal("the failure was never recorded on the alert")
	}
	if w, _ := recorder.last(); w.errText != sendErr.Error() {
		t.Errorf("recorded reason %q, want %q", w.errText, sendErr.Error())
	}
	if got := sink.sessions.Load(); got != tc.sessions {
		t.Errorf("made %d connections, want %d", got, tc.sessions)
	}
	if len(sink.received()) != 0 {
		t.Error("a message was accepted despite the failure")
	}
}

func TestEmailUnreachableServerIsNamed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	n, err := delivery.NewEmail(delivery.EmailConfig{
		Host: "127.0.0.1", Port: port, From: "seed@example.test", To: []string{"noc@example.test"},
	})
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	sendErr := n.SendNow(context.Background(), testAlert())
	if sendErr == nil || !strings.Contains(sendErr.Error(), "connect to 127.0.0.1:"+strconv.Itoa(port)) {
		t.Fatalf("SendNow error = %v, want a named connect failure", sendErr)
	}
}

func TestValidateEmailRefusesWhatCouldNeverDeliver(t *testing.T) {
	valid := delivery.EmailConfig{
		Host: "mail.example.test",
		From: "seed@example.test",
		To:   []string{"noc@example.test"},
	}
	for name, mutate := range map[string]func(*delivery.EmailConfig){
		"no host":               func(c *delivery.EmailConfig) { c.Host = "" },
		"host with scheme":      func(c *delivery.EmailConfig) { c.Host = "smtp://mail.example.test" },
		"host with port":        func(c *delivery.EmailConfig) { c.Host = "mail.example.test:587" },
		"port out of range":     func(c *delivery.EmailConfig) { c.Port = 70000 },
		"plaintext mode":        func(c *delivery.EmailConfig) { c.TLS = "none" },
		"password, no user":     func(c *delivery.EmailConfig) { c.Password = "x" },
		"sender not an addr":    func(c *delivery.EmailConfig) { c.From = "seed" },
		"no recipient":          func(c *delivery.EmailConfig) { c.To = nil },
		"recipient not an addr": func(c *delivery.EmailConfig) { c.To = []string{"noc@example.test", "oncall"} },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := delivery.ValidateEmail(cfg); !errors.Is(err, delivery.ErrInvalidConfig) {
				t.Errorf("ValidateEmail = %v, want ErrInvalidConfig", err)
			}
		})
	}
	if err := delivery.ValidateEmail(valid); err != nil {
		t.Errorf("ValidateEmail(valid) = %v", err)
	}
}

// Each channel answers for itself: a webhook that delivers must not hide a
// mail relay that refuses, which is what one shared status slot would do.
func TestManagerRecordsEachChannelSeparately(t *testing.T) {
	srv, hits := countingReceiver(t)
	sink := &smtpSink{
		user: sinkUser, pass: sinkPass,
		replies: map[string]string{"RCPT": "550 5.1.1 no such mailbox"},
	}
	sink.start(t)

	recorder := &recordingRecorder{}
	m := delivery.NewManager(recorder, quietLogger())
	t.Cleanup(func() { m.Stop(context.Background()) })
	m.ApplyWebhook(delivery.WebhookConfig{URL: srv.URL, Secret: signingKey})
	m.ApplyEmail(emailConfig(sink, nil))

	store := &recordingStore{}
	alert := testAlert()
	alert.ID = 0
	if err := delivery.WrapWriter(store, m).Create(context.Background(), alert); err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, channel := range []alerts.Channel{alerts.ChannelEmail, alerts.ChannelWebhook} {
		if got := statusOn(alert, channel); got != alerts.DeliveryPending {
			t.Errorf("stored %s state %q, want pending", channel, got)
		}
	}

	outcome := func(channel alerts.Channel) (recordedDelivery, bool) {
		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		for _, w := range recorder.writes {
			if w.channel == channel {
				return w, true
			}
		}
		return recordedDelivery{}, false
	}
	if !waitFor(t, func() bool {
		_, web := outcome(alerts.ChannelWebhook)
		_, mail := outcome(alerts.ChannelEmail)
		return web && mail
	}) {
		t.Fatalf("both channels did not record an outcome: %+v", recorder.writes)
	}
	if w, _ := outcome(alerts.ChannelWebhook); w.status != alerts.DeliveryDelivered || hits.Load() != 1 {
		t.Errorf("webhook outcome %+v after %d posts, want delivered", w, hits.Load())
	}
	if w, _ := outcome(alerts.ChannelEmail); w.status != alerts.DeliveryFailed ||
		!strings.Contains(w.errText, "550 5.1.1") {
		t.Errorf("email outcome %+v, want failed with the relay's reason", w)
	}
}

func TestManagerSendTest(t *testing.T) {
	m := delivery.NewManager(nil, quietLogger())
	t.Cleanup(func() { m.Stop(context.Background()) })

	if err := m.SendTest(context.Background(), alerts.ChannelEmail); !errors.Is(err, delivery.ErrNotConfigured) {
		t.Fatalf("SendTest with no relay = %v, want ErrNotConfigured", err)
	}

	sink := &smtpSink{user: sinkUser, pass: sinkPass}
	sink.start(t)
	m.ApplyEmail(emailConfig(sink, nil))
	if err := m.SendTest(context.Background(), alerts.ChannelEmail); err != nil {
		t.Fatalf("SendTest: %v", err)
	}
	got := sink.received()
	if len(got) != 1 {
		t.Fatalf("sink received %d messages, want 1", len(got))
	}
	header, _ := readMessage(t, got[0].data)
	if subject := header.Get("Subject"); subject != "[Seed] INFO: Test alert" {
		t.Errorf("Subject = %q", subject)
	}
	// The test alert was never stored, so it must not claim an alert id.
	if id := header.Get("X-Seed-Alert-Id"); id != "" {
		t.Errorf("X-Seed-Alert-Id = %q on a test alert", id)
	}
}

// An escalation goes out only on the channels its stage names, and the email
// says it is an escalation, or it reads as a duplicate of the first send.
func TestManagerEscalatesOnTheStageChannelsOnly(t *testing.T) {
	srv, hits := countingReceiver(t)
	sink := &smtpSink{user: sinkUser, pass: sinkPass}
	sink.start(t)

	recorder := &recordingRecorder{}
	m := delivery.NewManager(recorder, quietLogger())
	t.Cleanup(func() { m.Stop(context.Background()) })
	m.ApplyWebhook(delivery.WebhookConfig{URL: srv.URL, Secret: signingKey})
	m.ApplyEmail(emailConfig(sink, nil))

	alert := testAlert()
	alert.EscalationStage = 2
	m.Escalate(context.Background(), alert, []alerts.Channel{alerts.ChannelEmail})

	if !waitFor(t, func() bool { return len(sink.received()) == 1 }) {
		t.Fatal("the escalation was not emailed")
	}
	header, body := readMessage(t, sink.received()[0].data)
	subject, err := new(mime.WordDecoder).DecodeHeader(header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	if want := "[Seed] CRITICAL, unacknowledged, escalation 2: Gateway latency threshold breached"; subject != want {
		t.Errorf("Subject = %q, want %q", subject, want)
	}
	if !strings.Contains(body, "Escalation: stage 2, not acknowledged") {
		t.Errorf("body does not name the escalation:\n%s", body)
	}
	if hits.Load() != 0 {
		t.Errorf("webhook got %d posts; the stage names email only", hits.Load())
	}
}
