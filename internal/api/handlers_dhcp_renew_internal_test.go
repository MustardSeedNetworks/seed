package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
)

// stubRenew swaps the package's renewal seam for the duration of a test. The
// real one would renew the DHCP lease of the machine running the tests.
// serverWithStubbedRenew wires the renewal seam. The real one would restart
// the DHCP client on the machine running the tests.
func serverWithStubbedRenew(supported bool, err error, called *string) *Server {
	return &Server{
		renewSupportedFn: func() bool { return supported },
		renewLeaseFn: func(_ context.Context, iface string) error {
			*called = iface

			return err
		},
	}
}

func postRenew(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/telemetry/dhcp/renew",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleRenewDHCPLease(w, req)

	return w
}

func TestHandleRenewDHCPLease_RefusesUnsupportedPlatformsUpFront(t *testing.T) {
	var called string
	s := serverWithStubbedRenew(false, nil, &called)

	w := postRenew(t, s, `{"interface":"en0"}`)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501", w.Code)
	}
	// Refused before attempting: a platform that cannot renew must not be
	// asked to, so the failure is about capability rather than a runtime error.
	if called != "" {
		t.Errorf("renewal attempted on an unsupported platform (interface %q)", called)
	}
}

func TestHandleRenewDHCPLease_RequiresAnInterface(t *testing.T) {
	var called string
	s := serverWithStubbedRenew(true, nil, &called)

	w := postRenew(t, s, `{"interface":""}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	// The dangerous case: an empty name must never reach the renewal, where a
	// platform might interpret it as "every interface".
	if called != "" {
		t.Errorf("empty interface reached the renewal as %q", called)
	}
}

func TestHandleRenewDHCPLease_RenewsTheNamedInterface(t *testing.T) {
	var called string
	s := serverWithStubbedRenew(true, nil, &called)

	w := postRenew(t, s, `{"interface":"en0"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if called != "en0" {
		t.Errorf("renewed %q, want en0", called)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["interface"] != "en0" {
		t.Errorf("response interface = %v, want en0", body["interface"])
	}
}

func TestHandleRenewDHCPLease_MapsUnsupportedFromTheAttempt(t *testing.T) {
	// The managing client can disappear between the capability check and the
	// attempt; that is still a 501, not a 500.
	var called string
	s := serverWithStubbedRenew(true, dhcp.ErrRenewUnsupported, &called)

	w := postRenew(t, s, `{"interface":"en0"}`)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501", w.Code)
	}
}

func TestHandleRenewDHCPLease_ReportsARealFailureAs500(t *testing.T) {
	var called string
	s := serverWithStubbedRenew(true, errors.New("networkctl: exit status 1"), &called)

	w := postRenew(t, s, `{"interface":"en0"}`)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	// The underlying command's stderr must not be echoed to the caller.
	if strings.Contains(w.Body.String(), "networkctl") {
		t.Errorf("command detail leaked into the response: %s", w.Body.String())
	}
}

func TestHandleRenewDHCPLease_RejectsNamesThatCouldForgeALogEntry(t *testing.T) {
	// The handler logs the interface name on both its success and failure
	// paths, straight from the request body (CodeQL 402/403/404, CWE-117).
	// A name carrying a newline would let a caller append a second, invented
	// log line -- which matters here because the fleet's SIEM rules parse
	// those lines. The charset guard is what makes that unrepresentable.
	//
	// The names are JSON string escapes on purpose: that is how a real caller
	// would deliver a control character through the request body.
	cases := map[string]string{
		"newline":         `en0\nlevel=INFO msg="DHCP lease renewed" interface=forged`,
		"carriage return": `en0\rlevel=ERROR msg=forged`,
		"tab":             `en0\tforged`,
		"space":           `en0 forged`,
		"shell metachar":  `en0; reboot`,
		"too long":        `aaaaaaaaaaaaaaaaaaaaaaaa`,
	}

	for name, iface := range cases {
		t.Run(name, func(t *testing.T) {
			var called string
			s := serverWithStubbedRenew(true, nil, &called)

			w := postRenew(t, s, `{"interface":"`+iface+`"}`)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 for %q", w.Code, iface)
			}
			// The load-bearing assertion: a rejected name must not reach the
			// renewal, which is what runs a command against the interface.
			if called != "" {
				t.Errorf("rejected name still reached the renewal as %q", called)
			}
		})
	}
}
