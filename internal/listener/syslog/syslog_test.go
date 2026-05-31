package syslog_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/listener"
	"github.com/krisarmstrong/seed/internal/listener/syslog"
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
		time.Sleep(10 * time.Millisecond)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]listener.Event{}, f.got...)
}

// pickPort returns a free UDP loopback address so parallel tests
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

func TestParse_RFC3164Frame(t *testing.T) {
	t.Parallel()
	// <30> = facility 3 (system), severity 6 (informational)
	frame := []byte("<30>Oct 11 22:14:15 router-1 sshd[1234]: Accepted publickey for admin")
	p := syslog.Parse(frame)
	if p.Facility != 3 || p.Severity != 6 {
		t.Errorf("facility/severity = %d/%d, want 3/6", p.Facility, p.Severity)
	}
	if p.SeverityName != "informational" {
		t.Errorf("severityName = %q", p.SeverityName)
	}
	if p.Tag == "" {
		t.Error("Tag should be parsed from header")
	}
	if p.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestParse_NoPriDefaultsToInformational(t *testing.T) {
	t.Parallel()
	p := syslog.Parse([]byte("plain message without pri"))
	if p.SeverityName != "informational" {
		t.Errorf("severityName = %q, want informational", p.SeverityName)
	}
	if p.Message != "plain message without pri" {
		t.Errorf("Message = %q", p.Message)
	}
}

func TestParse_OverlargePRIIgnored(t *testing.T) {
	t.Parallel()
	// PRI 999 is invalid (max 191); parser falls back to the
	// no-PRI default (informational, facility 0).
	p := syslog.Parse([]byte("<999>something"))
	if p.Facility != 0 || p.Severity != syslog.SeverityInformational {
		t.Errorf("overlarge PRI should fall through to informational, got %+v", p)
	}
}

func TestParse_AllSeverityNamesIndexed(t *testing.T) {
	t.Parallel()
	want := []string{
		"emergency", "alert", "critical", "error",
		"warning", "notice", "informational", "debug",
	}
	for i, w := range want {
		frame := []byte("<" + itoa(i) + ">test")
		p := syslog.Parse(frame)
		if p.SeverityName != w {
			t.Errorf("severity %d -> %q, want %q", i, p.SeverityName, w)
		}
	}
}

func TestNew_RejectsNilSink(t *testing.T) {
	t.Parallel()
	if _, err := syslog.New(syslog.Config{}); err == nil {
		t.Error("expected New with nil Sink to fail")
	}
}

func TestStartStop_BindAndUnbind(t *testing.T) {
	t.Parallel()
	addr := pickPort(t)
	sink := &fakeSink{}
	l, newErr := syslog.New(syslog.Config{
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
	if stopErr := l.Stop(context.Background()); stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}
	if stopErr2 := l.Stop(context.Background()); stopErr2 != nil {
		t.Fatalf("second Stop: %v", stopErr2)
	}
}

func TestRoundTrip_DatagramArrivesAtSink(t *testing.T) {
	t.Parallel()
	addr := pickPort(t)
	sink := &fakeSink{}
	l, newErr := syslog.New(syslog.Config{
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
	defer func() { _ = l.Stop(context.Background()) }()

	c, dialErr := net.Dial("udp", addr)
	if dialErr != nil {
		t.Fatalf("dial: %v", dialErr)
	}
	defer func() { _ = c.Close() }()

	if _, writeErr := c.Write([]byte("<30>Jan 1 00:00:00 host-1 app[1]: hello world")); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}

	got := sink.wait(t, 1, 2*time.Second)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	evt := got[0]
	if evt.Kind != syslog.Name {
		t.Errorf("Kind = %q", evt.Kind)
	}
	if evt.Severity != "informational" {
		t.Errorf("Severity = %q", evt.Severity)
	}
	var parsed syslog.Parsed
	if uErr := json.Unmarshal(evt.Payload, &parsed); uErr != nil {
		t.Fatalf("unmarshal payload: %v", uErr)
	}
	if parsed.Facility != 3 {
		t.Errorf("Parsed.Facility = %d, want 3", parsed.Facility)
	}
}

func TestStart_BindFailureBubblesError(t *testing.T) {
	t.Parallel()
	addr := pickPort(t)
	first, newFirstErr := syslog.New(syslog.Config{
		BindAddr: addr,
		Sink:     &fakeSink{},
		Logger:   silentLogger(),
	})
	if newFirstErr != nil {
		t.Fatalf("New first: %v", newFirstErr)
	}
	if startErr := first.Start(context.Background()); startErr != nil {
		t.Fatalf("first Start: %v", startErr)
	}
	defer func() { _ = first.Stop(context.Background()) }()

	second, newSecondErr := syslog.New(syslog.Config{
		BindAddr: addr,
		Sink:     &fakeSink{},
		Logger:   silentLogger(),
	})
	if newSecondErr != nil {
		t.Fatalf("New second: %v", newSecondErr)
	}
	if startErr := second.Start(context.Background()); startErr == nil {
		t.Error("expected second Start on same port to fail")
		_ = second.Stop(context.Background())
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := "0123456789"
	out := ""
	for n > 0 {
		out = string(digits[n%10]) + out
		n /= 10
	}
	return out
}
