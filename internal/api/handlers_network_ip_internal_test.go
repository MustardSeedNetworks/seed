// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// seed#123: the IP config card's Details group names the manufacturer behind
// the interface MAC, from the same OUI registry discovery labels devices with.
func TestIPConfigReportsInterfaceVendor(t *testing.T) {
	tests := []struct {
		name       string
		iface      string
		withOUI    bool
		wantVendor string
	}{
		// 00:11:22 is registered to CIMSYS Inc in the embedded IEEE registry.
		{name: "registered OUI", iface: "eth0", withOUI: true, wantVendor: "CIMSYS Inc"},
		// AA:BB:CC has the locally administered bit set: no manufacturer exists.
		{name: "locally administered MAC", iface: "wlan0", withOUI: true},
		{name: "no OUI registry", iface: "eth0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewTestServer()
			t.Cleanup(s.Close)
			if tc.withOUI {
				s.deviceDisc = enumerate.NewDeviceDiscoveryWithOUI(tc.iface, "", 0)
			}

			body := getIPConfig(t, s, tc.iface)
			if body["interface"] != tc.iface {
				t.Errorf("interface = %v, want %s", body["interface"], tc.iface)
			}
			vendor, present := body["vendor"]
			if tc.wantVendor == "" {
				if present {
					t.Errorf("vendor = %v, want the key omitted", vendor)
				}
				return
			}
			if vendor != tc.wantVendor {
				t.Errorf("vendor = %v, want %q", vendor, tc.wantVendor)
			}
		})
	}
}

func getIPConfig(t *testing.T, s *Server, iface string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/ipconfig?interface="+iface, http.NoBody)
	w := httptest.NewRecorder()
	s.Mux().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body
}
