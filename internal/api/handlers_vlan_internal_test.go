package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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

// vlanBody is the create/delete request every test here sends.
const vlanBody = `{"interface":"eth0","vlanId":100}`

func vlanRequest(t *testing.T, s *Server, method string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, APIVersionPrefix+"/network/vlan/interface", strings.NewReader(vlanBody))
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

			w := vlanRequest(t, s, tt.method)

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

			w := vlanRequest(t, s, tt.method)

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

			w := vlanRequest(t, s, tt.method)

			if w.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleVLANInterface_SuccessBody(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			s := serverWithStubbedVLAN(true, nil, nil)

			w := vlanRequest(t, s, method)

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, []string{"interface", "message", "status", "vlanId"}, topLevelKeys(t, w.Body.Bytes()))
			var resp VLANInterfaceResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			require.Equal(t, statusSuccess, resp.Status)
			require.Equal(t, "eth0", resp.Interface)
			require.Equal(t, 100, resp.VlanID)
		})
	}
}

// TestVLANInterfaceRouteServesDelete pins #3261: the route declared GET, POST
// and PUT while the handler serves POST and DELETE, so the method gate refused
// every DELETE before it reached the handler.
func TestVLANInterfaceRouteServesDelete(t *testing.T) {
	s := ssoGateServer(t)
	s.vlanSeam = serverWithStubbedVLAN(true, nil, nil).vlanSeam
	const path = APIVersionPrefix + "/telemetry/vlan/interface"

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authedRequest(t, s, http.MethodDelete, path, "admin", vlanBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authedRequest(t, s, http.MethodGet, path, "admin", ""))
	require.Equal(t, http.StatusMethodNotAllowed, w.Code, w.Body.String())
	require.Equal(t, "POST, DELETE", w.Header().Get("Allow"))
}
