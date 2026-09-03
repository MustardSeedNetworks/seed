package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/capabilities"
)

// #750's point is that the three refusals must not be confused: 403 means you
// may not, 402 means your licence does not include it, 501 means this operating
// system cannot and no amount of privilege or money changes that here. An
// operator sent to the wrong remedy loses an afternoon.
func TestRequirePlatformRefusesWithNotImplemented(t *testing.T) {
	t.Parallel()

	// Chosen because no platform in the matrix supports it, so this asserts the
	// same thing on every OS the suite runs on.
	unsupported := capabilities.Capability("a_capability_no_platform_has")
	if capabilities.Supported(unsupported) {
		t.Fatal("the fixture capability is supported somewhere; pick another")
	}

	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/cable", nil)

	if server.requirePlatform(recorder, request, unsupported) {
		t.Error("requirePlatform allowed a capability no platform supports")
	}
	if recorder.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d — a platform gap is not a 403 or a 402",
			recorder.Code, http.StatusNotImplemented)
	}
	if !strings.Contains(recorder.Body.String(), "NOT_IMPLEMENTED") {
		t.Errorf("body does not carry the not-implemented code: %s", recorder.Body.String())
	}
}

// A supported capability must pass straight through and write nothing, or every
// gated handler would emit two responses.
func TestRequirePlatformAllowsASupportedCapability(t *testing.T) {
	t.Parallel()

	// Interface listing is Full on all three platforms.
	if !capabilities.Supported(capabilities.InterfaceListing) {
		t.Skip("interface listing is unsupported here; nothing to assert")
	}

	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/interfaces", nil)

	if !server.requirePlatform(recorder, request, capabilities.InterfaceListing) {
		t.Error("requirePlatform refused a fully supported capability")
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("requirePlatform wrote a body when allowing: %s", recorder.Body.String())
	}
}
