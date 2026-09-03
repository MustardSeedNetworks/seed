package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/capabilities"
)

// The interface name reaches both a log line and the ethtool call, and it comes
// straight off the query string (CodeQL CWE-117 on #2315). This asserts the
// rejection rather than the log text: the handler resolves its logger from the
// process-wide one, so capturing it would mean mutating global state in a test
// that otherwise runs in parallel. What matters is that a name carrying a
// newline never gets far enough to be written at all.
func TestDriverStatsRejectsAnInterfaceNameThatCouldForgeALogEntry(t *testing.T) {
	t.Parallel()

	if !capabilities.Supported(capabilities.DriverStatistics) {
		t.Skip("driver statistics are unsupported here; the platform gate answers first")
	}

	for _, name := range []string{
		"eth0\ninjected=true",
		"eth0 level=ERROR",
		"../../etc/passwd",
		"an-interface-name-far-longer-than-sixteen",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet,
				"/api/v1/telemetry/interface/driver-stats?interface="+url.QueryEscape(name), nil)

			(&Server{}).handleDriverStats(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d — %q reached the driver call",
					recorder.Code, http.StatusBadRequest, name)
			}
		})
	}
}
