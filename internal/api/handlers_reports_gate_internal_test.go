// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/license"
)

// prodSeedStarterVector is the production-signed Starter key the licence
// package pins its keygen contract against. Reused here because a Starter
// licence is the only way to reach the PDF check: the route itself is gated on
// export_csv_json, so a Free caller is refused one layer earlier and never
// exercises what this test is about.
const prodSeedStarterVector = "MSN1.eyJjb2RlIjoiNDAwMSIsImlhdCI6MTc4MDg3NjgwMCwibWF4RGV2aWNlcyI6MywicHJvZHVjdCI6InNlZWQiLCJzZXJpYWwiOiIxMjM0NTY3IiwidGllciI6MSwidiI6MX0.KEv70KrphG0Y7ATG_OPJhf4I0YJNcF7KNAVY4GPSj_Mdvxkhi4aEi6_h4Ux2EV-vkiA3lV0l_Bo7yTN9zI29CA"

// The PDF audit report is sold as Pro (audit_pdf) and the route it is reached
// through is gated on export_csv_json, which is Starter. PDF is a value of
// `format`, not a path, so the route gate cannot express the difference and a
// Starter licence could generate one (#2327).
func TestReportGenerateGatesPDFOnAuditPDF(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	// The reports routes are rate-limited; the shared fixture leaves the
	// endpoint limiter nil, which panics before any handler runs.
	s.endpointLimiter = NewEndpointRateLimiter(DefaultEndpointRateLimitConfig())
	s.setupRoutes()

	res := mgr.Activate(prodSeedStarterVector)
	if !res.Success {
		t.Fatalf("Starter vector did not activate: %s", res.Message)
	}
	if license.Tier(res.Tier) != license.TierStarter {
		t.Fatalf("tier = %s, want Starter", license.Tier(res.Tier))
	}
	if !mgr.HasFeature("export_csv_json") {
		t.Fatal("Starter does not grant export_csv_json; the fixture is wrong")
	}

	post := func(format string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(GenerateReportRequest{Type: "executive", Format: format})
		req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/reports/generate", body, "alice")
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		return w
	}

	pdf := post("pdf")
	if pdf.Code != http.StatusPaymentRequired {
		t.Fatalf("pdf on Starter: status = %d, want 402; body=%s", pdf.Code, pdf.Body.String())
	}
	var gate FeatureGateResponse
	if err := json.NewDecoder(pdf.Body).Decode(&gate); err != nil {
		t.Fatalf("decode 402 body: %v", err)
	}
	if gate.RequiredFeature != "audit_pdf" {
		t.Errorf("requiredFeature = %q, want %q — the route gate answered instead of the format gate",
			gate.RequiredFeature, "audit_pdf")
	}

	// CSV is what Starter bought. It must not be collateral damage: the
	// generator is not wired in this fixture, so it fails further down, and the
	// only thing asserted is that it is not refused for payment.
	if csv := post("csv"); csv.Code == http.StatusPaymentRequired {
		t.Errorf("csv on Starter: status = 402, want anything else; body=%s", csv.Body.String())
	}
}
