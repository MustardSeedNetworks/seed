package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/netif"
)

// These pin the bodies of /api/v1/interface and /api/v1/interfaces to the
// DTOs the OpenAPI document names for them (S6-4, #196).

func interfaceTestServer(current string, mutate func(map[string]*netif.InterfaceInfo)) *Server {
	cfg := netif.DefaultMockConfig()
	cfg.CurrentInterface = current
	if mutate != nil {
		mutate(cfg.Interfaces)
	}
	return &Server{netMgr: netif.NewMockManager(cfg)}
}

func TestInterfaceSelectionResponseShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		current string
		mutate  func(map[string]*netif.InterfaceInfo)
		want    string
	}{
		{
			name:    "selected and up",
			current: "eth0",
			want: `{"interface":"eth0","isWireless":false,"available":true,"type":"ethernet",` +
				`"up":true,"friendlyName":"Ethernet 0"}`,
		},
		{
			name:    "selected and down",
			current: "eth0",
			mutate:  func(m map[string]*netif.InterfaceInfo) { m["eth0"].Up = false },
			want: `{"interface":"eth0","isWireless":false,"available":false,"type":"ethernet",` +
				`"up":false,"friendlyName":"Ethernet 0",` +
				`"warning":"Selected interface is no longer available","suggestedInterface":"wlan0"}`,
		},
		{
			name:    "selected interface gone",
			current: "eth9",
			want: `{"interface":"eth9","isWireless":false,"available":false,"up":false,` +
				`"warning":"Selected interface is no longer available","suggestedInterface":"eth0"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := interfaceTestServer(tt.current, tt.mutate)
			w := serve(t, s.handleInterface, "/api/v1/interface")
			require.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

func TestSetInterfaceResponseShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(map[string]*netif.InterfaceInfo)
		want   string
	}{
		{
			name: "up",
			want: `{"status":"ok","interface":"wlan0","isWireless":false,"type":"wifi"}`,
		},
		{
			name:   "down",
			mutate: func(m map[string]*netif.InterfaceInfo) { m["wlan0"].Up = false },
			want: `{"status":"ok","interface":"wlan0","isWireless":false,"type":"wifi",` +
				`"warning":"Selected interface is currently down"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := interfaceTestServer("eth0", tt.mutate)
			w := httptest.NewRecorder()
			s.handleInterface(w, httptest.NewRequest(http.MethodPut, "/api/v1/interface",
				strings.NewReader(`{"interface":"wlan0"}`)))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

// TestInterfacesAnswersTheCategorizedListing: the flat array the route served
// without ?categorized=true had no consumer, and two shapes on one route
// cannot be one documented body.
func TestInterfacesAnswersTheCategorizedListing(t *testing.T) {
	t.Parallel()
	w := serve(t, interfaceTestServer("eth0", nil).handleInterfaces, "/api/v1/interfaces")
	require.Equal(t,
		[]string{"currentInterface", "currentType", "ethernet", "recommendedEthernet", "recommendedWifi", "wifi"},
		topLevelKeys(t, w.Body.Bytes()))
}
