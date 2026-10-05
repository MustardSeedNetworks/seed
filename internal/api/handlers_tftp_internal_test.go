package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/tftp"
)

// loopbackName is the host's IPv4 loopback interface: lo on Linux, lo0 on
// macOS.
func loopbackName(t *testing.T) string {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("interfaces: %v", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return iface.Name
		}
	}
	t.Skip("no loopback interface")
	return ""
}

// newTFTPTestServer registers the TFTP route through the real registry, so a
// request meets the same role and licence gates as in the daemon. The manager
// binds an ephemeral port: tests cannot bind 69.
func newTFTPTestServer(t *testing.T, trial bool) *Server {
	t.Helper()
	s, mgr := usersTestSetup(t)
	if trial {
		if r := mgr.StartTrial(); !r.Success {
			t.Fatalf("StartTrial: %s", r.Message)
		}
	}
	seedRoledUser(t, s, "operator1", roles.Operator)

	probe, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	_ = probe.Close()

	s.tftpAudit = app.NewTFTPAudit(s.db)
	s.tftpSessions = tftp.NewManager(tftp.Config{
		Dir:        t.TempDir(),
		Port:       port,
		OnTransfer: s.auditTFTPTransfer,
		OnStop:     s.auditTFTPStop,
	})
	t.Cleanup(s.tftpSessions.Close)
	s.registerAll(s.tftpRoutes())
	return s
}

func tftpCall(s *Server, method, user, body string) *httptest.ResponseRecorder {
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, newAuthedRequest(method, APIVersionPrefix+"/tftp/session", raw, user))
	return rec
}

func tftpStatusOf(t *testing.T, rec *httptest.ResponseRecorder) tftp.Status {
	t.Helper()
	var st tftp.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return st
}

// TestTFTPSessionGates: starting and stopping is admin only and Pro; reading
// the state is open to any role on Pro.
func TestTFTPSessionGates(t *testing.T) {
	t.Parallel()
	start := `{"interface":"` + loopbackName(t) + `"}`

	free := newTFTPTestServer(t, false)
	if rec := tftpCall(free, http.MethodPost, "admin", start); rec.Code != http.StatusPaymentRequired {
		t.Errorf("admin POST on Free: %d, want 402 (%s)", rec.Code, rec.Body.String())
	}

	pro := newTFTPTestServer(t, true)
	cases := []struct {
		method, user, body string
		want               int
	}{
		{http.MethodGet, "operator1", "", http.StatusOK},
		{http.MethodPost, "operator1", start, http.StatusForbidden},
		{http.MethodDelete, "operator1", "", http.StatusForbidden},
		{http.MethodPut, "admin", start, http.StatusMethodNotAllowed},
	}
	for _, c := range cases {
		if rec := tftpCall(pro, c.method, c.user, c.body); rec.Code != c.want {
			t.Errorf("%s by %s: %d, want %d (%s)", c.method, c.user, rec.Code, c.want, rec.Body.String())
		}
	}
	if pro.tftpSessions.Status().Running {
		t.Fatal("a refused request started a session")
	}
}

// TestTFTPSessionLifecycle is the row's acceptance at the API: nothing runs
// until an admin starts a session, it binds the chosen interface, and the
// start and stop are audited with the admin's name.
func TestTFTPSessionLifecycle(t *testing.T) {
	t.Parallel()
	s := newTFTPTestServer(t, true)

	if st := tftpStatusOf(t, tftpCall(s, http.MethodGet, "admin", "")); st.Running {
		t.Fatalf("a session is running before any start: %+v", st)
	}
	if rec := tftpCall(s, http.MethodDelete, "admin", ""); rec.Code != http.StatusConflict {
		t.Errorf("DELETE with nothing running: %d, want 409", rec.Code)
	}
	if rec := tftpCall(s, http.MethodPost, "admin", `{"interface":"no-such-if0"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("POST unknown interface: %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	iface := loopbackName(t)
	rec := tftpCall(s, http.MethodPost, "admin", `{"interface":"`+iface+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST start: %d %s", rec.Code, rec.Body.String())
	}
	st := tftpStatusOf(t, rec)
	if !st.Running || st.Interface != iface || st.AllowUpload || !net.ParseIP(st.Address).IsLoopback() {
		t.Fatalf("started session = %+v, want running read-only on %s's address", st, iface)
	}
	if again := tftpCall(s, http.MethodPost, "admin", `{"interface":"`+iface+`"}`); again.Code != http.StatusConflict {
		t.Errorf("second POST: %d, want 409", again.Code)
	}

	rec = tftpCall(s, http.MethodDelete, "admin", "")
	if rec.Code != http.StatusOK || tftpStatusOf(t, rec).Running {
		t.Fatalf("DELETE: %d %s, want 200 and not running", rec.Code, rec.Body.String())
	}

	logs, err := s.dbConn.GetAuditLogs(t.Context(), database.AuditLogOptions{ResourceType: tftp.AuditResource})
	if err != nil {
		t.Fatalf("audit logs: %v", err)
	}
	got := map[string]string{}
	for _, l := range logs {
		got[l.Action] = l.User
	}
	if got[tftpAuditStart] != "admin" || got[tftpAuditStop] != "admin" {
		t.Errorf("audit actions = %v, want start and stop by admin", got)
	}
}
