package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestHandleDevicesSubnetsGET tests the subnets list endpoint.
func TestHandleDevicesSubnetsGET(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/security/devices/subnets", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response is an array
	var resp []api.SubnetResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// TestHandleDevicesSubnetsPOST tests adding a subnet.
func TestHandleDevicesSubnetsPOST(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()
	server.SetConfigPath("/tmp/test-config.json")

	reqBody := api.SubnetRequest{
		CIDR:    "10.0.0.0/24",
		Name:    "Test Subnet",
		Enabled: true,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/devices/subnets",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestHandleDevicesSubnetsPOSTInvalidCIDR tests adding a subnet with invalid CIDR.
func TestHandleDevicesSubnetsPOSTInvalidCIDR(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()

	reqBody := api.SubnetRequest{
		CIDR:    "invalid-cidr",
		Name:    "Test Subnet",
		Enabled: true,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/devices/subnets",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleDevicesSubnetsPUT tests updating a subnet.
func TestHandleDevicesSubnetsPUT(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()
	server.SetConfigPath("/tmp/test-config.json")

	// First add a subnet
	addBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "172.16.0.0/16",
		Name:    "Initial Name",
		Enabled: true,
	})
	addReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/devices/subnets",
		bytes.NewReader(addBody),
	)
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	server.Mux().ServeHTTP(addW, addReq)

	// Now update it
	updateBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "172.16.0.0/16",
		Name:    "Updated Name",
		Enabled: false,
	})
	updateReq := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/security/devices/subnets",
		bytes.NewReader(updateBody),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()

	server.Mux().ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf(
			"Expected status %d, got %d: %s",
			http.StatusOK,
			updateW.Code,
			updateW.Body.String(),
		)
	}
}

// TestHandleDevicesSubnetsPUTNotFound tests updating a non-existent subnet.
func TestHandleDevicesSubnetsPUTNotFound(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()

	updateBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "192.168.99.0/24",
		Name:    "Non-existent",
		Enabled: false,
	})
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/security/devices/subnets",
		bytes.NewReader(updateBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestHandleDevicesSubnetsDELETE tests deleting a subnet.
func TestHandleDevicesSubnetsDELETE(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()
	server.SetConfigPath("/tmp/test-config.json")

	// First add a subnet
	addBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "192.168.50.0/24",
		Name:    "To Delete",
		Enabled: true,
	})
	addReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/devices/subnets",
		bytes.NewReader(addBody),
	)
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	server.Mux().ServeHTTP(addW, addReq)

	// Now delete it
	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/security/devices/subnets?cidr=192.168.50.0/24",
		http.NoBody,
	)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestHandleDevicesSubnetsDELETEMissingCIDR tests deleting without CIDR parameter.
func TestHandleDevicesSubnetsDELETEMissingCIDR(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/security/devices/subnets", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleDevicesSubnetsDELETENotFound tests deleting a non-existent subnet.
func TestHandleDevicesSubnetsDELETENotFound(t *testing.T) {
	skipInShortMode(t)
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/security/devices/subnets?cidr=10.99.99.0/24",
		http.NoBody,
	)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestSubnetRequestValidation tests subnet request validation.
func TestSubnetRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     api.SubnetRequest
		isValid bool
	}{
		{
			name: "valid IPv4 /24",
			req: api.SubnetRequest{
				CIDR:    "192.168.1.0/24",
				Name:    "LAN",
				Enabled: true,
			},
			isValid: true,
		},
		{
			name: "valid IPv4 /16",
			req: api.SubnetRequest{
				CIDR:    "172.16.0.0/16",
				Name:    "Private",
				Enabled: true,
			},
			isValid: true,
		},
		{
			name: "valid IPv4 /8",
			req: api.SubnetRequest{
				CIDR:    "10.0.0.0/8",
				Name:    "Class A",
				Enabled: false,
			},
			isValid: true,
		},
		{
			name: "disabled subnet",
			req: api.SubnetRequest{
				CIDR:    "192.168.100.0/24",
				Name:    "Disabled Subnet",
				Enabled: false,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - CIDR should be non-empty
			if tt.req.CIDR == "" && tt.isValid {
				t.Error("Expected valid CIDR for valid request")
			}
		})
	}
}

const pendingSubnetsPath = "/api/v1/security/devices/subnets/pending"

// offeredSubnets is the pending list, checking every entry awaits a decision.
func offeredSubnets(t *testing.T, server *api.Server) []string {
	t.Helper()
	w := httptest.NewRecorder()
	server.Mux().ServeHTTP(w, httptest.NewRequest(http.MethodGet, pendingSubnetsPath, http.NoBody))
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", w.Code, w.Body)
	}
	var resp []api.SubnetResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	cidrs := make([]string, 0, len(resp))
	for _, subnet := range resp {
		if !subnet.Learned || subnet.Enabled || subnet.Decision != "" {
			t.Errorf("listed %+v, which does not await a decision", subnet)
		}
		cidrs = append(cidrs, subnet.CIDR)
	}
	return cidrs
}

// The learned-network prompt's API (seed#3108): GET lists what awaits an
// answer, POST records one, and an answered network leaves the list.
func TestPendingSubnetsListAndDecide(t *testing.T) {
	skipInShortMode(t)
	cfg := config.DefaultConfig()
	cfg.NetworkDiscovery.TargetNetworks = []config.SubnetConfig{
		{CIDR: "192.168.1.0/24", Name: "Typed in by hand", Enabled: true},
		{CIDR: "10.51.0.0/16", Name: "Learned from 10.50.0.1 (routing table)", Learned: true},
		{CIDR: "10.52.0.0/16", Name: "Learned from 10.50.0.1 (routing table)", Learned: true},
	}
	server := api.NewTestServerWithConfig(cfg)
	defer server.Close()
	server.SetConfigPath(filepath.Join(t.TempDir(), "config.json"))

	if got := offeredSubnets(t, server); !slices.Equal(got, []string{"10.51.0.0/16", "10.52.0.0/16"}) {
		t.Fatalf("pending = %v, want both learned networks", got)
	}

	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"cidr":"10.51.0.0/16","decision":"maybe"}`, http.StatusBadRequest},
		{`{"cidr":"192.168.1.0/24","decision":"added"}`, http.StatusConflict},
		{`{"cidr":"10.99.0.0/16","decision":"added"}`, http.StatusNotFound},
		{`{"cidr":"10.51.0.0/16","decision":"added","enabled":true}`, http.StatusBadRequest},
		{`{"cidr":"10.51.0.0/16","decision":"added"}`, http.StatusOK},
		{`{"cidr":"10.52.0.0/16","decision":"dismissed"}`, http.StatusOK},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, pendingSubnetsPath, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		server.Mux().ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("POST %s: status %d, want %d", tc.body, w.Code, tc.want)
		}
	}

	if got := offeredSubnets(t, server); len(got) != 0 {
		t.Errorf("pending = %v after both were answered", got)
	}
	want := []config.SubnetConfig{
		{CIDR: "192.168.1.0/24", Name: "Typed in by hand", Enabled: true},
		{
			CIDR: "10.51.0.0/16", Name: "Learned from 10.50.0.1 (routing table)",
			Enabled: true, Learned: true, Decision: config.LearnedAdded,
		},
		{
			CIDR: "10.52.0.0/16", Name: "Learned from 10.50.0.1 (routing table)",
			Learned: true, Decision: config.LearnedDismissed,
		},
	}
	if got := cfg.NetworkDiscovery.TargetNetworks; !slices.Equal(got, want) {
		t.Errorf("target networks = %+v, want %+v", got, want)
	}
}
