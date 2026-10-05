package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
)

// TestIperfInfoOnTheTestServerStartsNoProcess pins #2530: the route test drove
// this handler into `iperf3 --version`, and an exec from a loaded macOS test
// binary can wedge its child until the package times out. The test server
// answers from its stub, so on a host with iperf3 installed this reports
// not-installed rather than the host's version.
func TestIperfInfoOnTheTestServerStartsNoProcess(t *testing.T) {
	t.Parallel()

	server := api.NewTestServer()
	defer server.Close()

	w := httptest.NewRecorder()
	server.Mux().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/iperf/info", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	var resp api.IperfInfoResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Installed || resp.Version != "" {
		t.Errorf("info = %+v, want the stub's not-installed answer, not a probed binary", resp)
	}
}
