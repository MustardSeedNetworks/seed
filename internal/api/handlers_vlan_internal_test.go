package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
)

var errVLANGenericFailure = errors.New("netlink: exit status 1")

// serverWithStubbedVLAN wires the VLAN create/delete seam. The real functions
// would create or remove an 802.1Q subinterface on the machine running the
// tests.
func serverWithStubbedVLAN(supported bool, createErr, deleteErr error) *Server {
	return &Server{
		vlanSeam: vlanSeam{
			createSupported: func() bool { return supported },
			create: func(_ string, _ int) error {
				return createErr
			},
			delete: func(_ string, _ int) error {
				return deleteErr
			},
		},
	}
}

func vlanRequest(t *testing.T, s *Server, method, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, APIVersionPrefix+"/network/vlan/interface", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleVLANInterface(w, req)

	return w
}

func TestHandleVLANInterface_RefusesUnsupportedPlatformsUpFront(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"create", http.MethodPost},
		{"delete", http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := serverWithStubbedVLAN(false, nil, nil)

			w := vlanRequest(t, s, tt.method, `{"interface":"eth0","vlanId":100}`)

			if w.Code != http.StatusNotImplemented {
				t.Errorf("status = %d, want 501: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleVLANInterface_MapsUnsupportedFromTheAttempt(t *testing.T) {
	// The capability check and the attempt are two different calls; a platform
	// can disagree with itself between them (or the check can be wrong). That
	// is still a 501, not a 500 -- belt and braces on top of the up-front gate.
	tests := []struct {
		name   string
		method string
	}{
		{"create", http.MethodPost},
		{"delete", http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := serverWithStubbedVLAN(true, vlan.ErrUnsupported, vlan.ErrUnsupported)

			w := vlanRequest(t, s, tt.method, `{"interface":"eth0","vlanId":100}`)

			if w.Code != http.StatusNotImplemented {
				t.Errorf("status = %d, want 501: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleVLANInterface_ReportsARealFailureAs500(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"create", http.MethodPost},
		{"delete", http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := serverWithStubbedVLAN(true, errVLANGenericFailure, errVLANGenericFailure)

			w := vlanRequest(t, s, tt.method, `{"interface":"eth0","vlanId":100}`)

			if w.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500: %s", w.Code, w.Body.String())
			}
		})
	}
}
