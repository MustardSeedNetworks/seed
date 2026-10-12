//go:build iperf3

// The iperf3 routes' success answers, driven against a genuine iperf3. The
// default suite cannot reach them: the test server never forks (#2530), and
// both answers only exist once iperf3 has run. Run by the iperf3 job in
// .github/workflows/ci.yml; like the boundary suite beside the manager,
// nothing here skips.
package api_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
)

// postStrict posts body to path and decodes a 200 answer into out, refusing
// any field out does not declare.
func postStrict(t *testing.T, server *api.Server, path string, body, out any) {
	t.Helper()

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Mux().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST %s: status = %d, want 200: %s", path, w.Code, w.Body.String())
	}
	dec := json.NewDecoder(w.Body)
	dec.DisallowUnknownFields()
	if err = dec.Decode(out); err != nil {
		t.Fatalf("POST %s: decode: %v", path, err)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestReal_IperfServerAndClientAnswers(t *testing.T) {
	if os.Getenv("SEED_IPERF3_TESTS") != "1" {
		t.Fatal("SEED_IPERF3_TESTS must be 1 for this suite; see the iperf3 job in .github/workflows/ci.yml")
	}

	server := api.NewTestServer()
	defer server.Close()
	port := freeTCPPort(t)

	var started iperf.ServerStatus
	postStrict(t, server, "/api/v1/telemetry/iperf/server",
		api.IperfServerRequest{Action: "start", Port: port}, &started)
	if !started.Running || started.Port != port || started.PID == 0 {
		t.Fatalf("start answered %+v, want running on port %d with a PID", started, port)
	}

	var accepted api.StatusMessageResponse
	postStrict(t, server, "/api/v1/telemetry/iperf/client", api.IperfClientRequest{
		Server: "127.0.0.1", Port: port, Protocol: "tcp", Direction: "upload", Duration: 1, Parallel: 1,
	}, &accepted)
	want := api.StatusMessageResponse{
		Status:  "success",
		Message: "iperf3 test started. Poll /api/v1/telemetry/iperf/client/status for results.",
	}
	if accepted != want {
		t.Errorf("client answered %+v, want %+v", accepted, want)
	}

	// Let the background run finish before the listener goes away, so the
	// stop below is not racing a transfer.
	deadline := time.Now().Add(30 * time.Second)
	for {
		w := httptest.NewRecorder()
		server.Mux().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/iperf/client/status", nil))
		var status api.IperfClientStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatalf("client status: decode: %v", err)
		}
		if !status.Running && status.Last != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("client run did not finish: %+v", status)
		}
		time.Sleep(200 * time.Millisecond)
	}

	var stopped iperf.ServerStatus
	postStrict(t, server, "/api/v1/telemetry/iperf/server",
		api.IperfServerRequest{Action: "stop"}, &stopped)
	if stopped != (iperf.ServerStatus{}) {
		t.Errorf("stop answered %+v, want the zero (not running) status", stopped)
	}
}
