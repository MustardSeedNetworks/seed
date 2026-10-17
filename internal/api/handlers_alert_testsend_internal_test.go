package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

// The test-send action (#2997) is an outbound connection a caller can trigger
// on demand, so it is operator-gated and rate-limited like the other
// diagnostic actions that reach off the box.
func TestAlertTestSendRouteIsOperatorGatedAndRateLimited(t *testing.T) {
	s := NewTestServer()
	defer s.Close()

	for _, rt := range s.routes.Policies() {
		if rt.Path != APIVersionPrefix+"/settings/alerts/test" {
			continue
		}
		if rt.Scope != roles.Operator || !rt.RateLimited ||
			len(rt.Methods) != 1 || rt.Methods[0] != http.MethodPost {
			t.Errorf("route = %+v, want POST, operator-gated, rate-limited", rt)
		}
		return
	}
	t.Fatal("POST /api/v1/settings/alerts/test is not registered")
}

func TestAlertTestSendAnswersWithTheReceiversReason(t *testing.T) {
	status := http.StatusNoContent
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}))
	defer receiver.Close()

	manager := alertdelivery.NewManager(nil, nil, slog.New(slog.DiscardHandler))
	defer manager.Stop(context.Background())

	send := func(s *Server, body string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/alerts/test", strings.NewReader(body))
		w := httptest.NewRecorder()
		s.handleAlertTestSend(w, req)
		var decoded map[string]any
		if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return w.Code, decoded
	}

	s := &Server{alertDelivery: manager}

	if code, body := send(s, `{"channel":"webhook"}`); code != http.StatusConflict ||
		!strings.Contains(body["error"].(string), "no receiver is configured") {
		t.Errorf("no receiver: %d %v, want 409 naming the missing receiver", code, body)
	}

	manager.ApplyWebhook(alertdelivery.WebhookConfig{URL: receiver.URL, Secret: "s3cret"})
	if code, body := send(s, `{"channel":"webhook"}`); code != http.StatusOK || body["sent"] != true {
		t.Errorf("working receiver: %d %v, want 200 sent", code, body)
	}

	status = http.StatusUnauthorized
	if code, body := send(s, `{"channel":"webhook"}`); code != http.StatusBadGateway ||
		body["code"] != ErrCodeDeliveryFailed || !strings.Contains(body["error"].(string), "receiver answered 401") {
		t.Errorf("refusing receiver: %d %v, want 502 carrying the receiver's answer", code, body)
	}

	for _, bad := range []string{`{"channel":"pager"}`, `not json`} {
		if code, _ := send(s, bad); code != http.StatusBadRequest {
			t.Errorf("body %s: status %d, want 400", bad, code)
		}
	}

	if code, _ := send(&Server{}, `{"channel":"email"}`); code != http.StatusServiceUnavailable {
		t.Errorf("no delivery manager: status %d, want 503", code)
	}
}

func TestAlertTestSendReachesTheSyslogCollector(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	received := make(chan string, 1)
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		frame, _ := io.ReadAll(conn)
		received <- string(frame)
	}()

	manager := alertdelivery.NewManager(nil, nil, slog.New(slog.DiscardHandler))
	defer manager.Stop(context.Background())
	manager.ApplySyslog(alertdelivery.SyslogConfig{Host: "127.0.0.1", Port: port, Transport: alertdelivery.SyslogTCP})
	s := &Server{alertDelivery: manager}

	send := func() (int, string) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/alerts/test",
			strings.NewReader(`{"channel":"syslog"}`))
		w := httptest.NewRecorder()
		s.handleAlertTestSend(w, req)
		return w.Code, w.Body.String()
	}

	if code, body := send(); code != http.StatusOK {
		t.Fatalf("working collector: %d %s, want 200", code, body)
	}
	if frame := <-received; !strings.Contains(frame, " seed - alert - ") ||
		!strings.Contains(frame, `title="Test alert"`) {
		t.Errorf("collector received %q, want the test alert as RFC 5424", frame)
	}

	_ = ln.Close()
	if code, body := send(); code != http.StatusBadGateway || !strings.Contains(body, "connect to 127.0.0.1:") {
		t.Errorf("collector gone: %d %s, want 502 naming the connection", code, body)
	}
}

func TestAlertTestSendReachesTheTrapReceiver(t *testing.T) {
	pc, err := (&net.ListenConfig{}).ListenPacket(context.Background(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = pc.Close() }()

	manager := alertdelivery.NewManager(nil, nil, slog.New(slog.DiscardHandler))
	defer manager.Stop(context.Background())
	manager.ApplyTrap(alertdelivery.TrapConfig{
		Host: "127.0.0.1", Port: uint16(pc.LocalAddr().(*net.UDPAddr).Port), Community: "nms",
	})
	s := &Server{alertDelivery: manager}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/alerts/test",
		strings.NewReader(`{"channel":"trap"}`))
	w := httptest.NewRecorder()
	s.handleAlertTestSend(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("working receiver: %d %s, want 200", w.Code, w.Body)
	}

	_ = pc.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 65535)
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatalf("receiver read: %v", err)
	}
	packet, err := (&gosnmp.GoSNMP{Version: gosnmp.Version2c}).UnmarshalTrap(buf[:n], false)
	if err != nil {
		t.Fatalf("decode trap: %v", err)
	}
	if packet.Community != "nms" || len(packet.Variables) < 2 ||
		packet.Variables[1].Value != ".1.3.6.1.4.1.67249.1.1.1.0.1" {
		t.Errorf("receiver got community %q varbinds %+v, want seedAlertRaised", packet.Community, packet.Variables)
	}
}
