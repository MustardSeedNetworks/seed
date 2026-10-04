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

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// The test-send action (#2997) is an outbound connection a caller can trigger
// on demand, so it is operator-gated and rate-limited like the other
// diagnostic actions that reach off the box.
func TestAlertTestSendRouteIsOperatorGatedAndRateLimited(t *testing.T) {
	s := NewTestServer()
	defer s.Close()

	for _, rt := range s.manifest {
		if apiPath(rt.path) != APIVersionPrefix+"/settings/alerts/test" {
			continue
		}
		if rt.minRole != database.RoleOperator || !rt.rateLimited ||
			len(rt.methods) != 1 || rt.methods[0] != http.MethodPost {
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

	manager := alertdelivery.NewManager(nil, slog.New(slog.DiscardHandler))
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

	manager := alertdelivery.NewManager(nil, slog.New(slog.DiscardHandler))
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
