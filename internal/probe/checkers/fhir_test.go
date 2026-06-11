package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// fhirDoer is a test double for HTTPDoer used exclusively by FHIR tests.
// It records the last outbound request so tests can assert on it.
type fhirDoer struct {
	resp *http.Response
	err  error
	got  *http.Request
}

func (f *fhirDoer) Do(req *http.Request) (*http.Response, error) {
	f.got = req
	return f.resp, f.err
}

// fhirMockResponse builds a minimal *[http.Response] with the given status
// and a JSON body — matching the mockResponse convention from http_test.go.
func fhirMockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/fhir+json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// validCapabilityStatement returns a minimal but parseable FHIR
// CapabilityStatement JSON body for success-path tests.
func validCapabilityStatement() string {
	return `{
		"resourceType": "CapabilityStatement",
		"fhirVersion": "4.0.1",
		"software": {"name": "TestFHIR", "version": "1.2.3"},
		"rest": [{"mode": "server", "resource": [{"type": "Patient"}, {"type": "Observation"}]}]
	}`
}

func TestFHIRChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewFHIRChecker().Kind() != "fhir" {
		t.Errorf("FHIRChecker.Kind() = %q, want \"fhir\"", checkers.NewFHIRChecker().Kind())
	}
}

func TestFHIRChecker_Run_Success(t *testing.T) {
	t.Parallel()

	doer := &fhirDoer{resp: fhirMockResponse(200, validCapabilityStatement())}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	r := c.Run(context.Background(), probe.Probe{
		ID:     "p1",
		Kind:   "fhir",
		Target: "https://fhir.example.com",
	})

	if !r.Success {
		t.Errorf("Success = false; want true: %s", r.Error)
	}

	// Verify the request was sent to the /metadata path.
	if doer.got == nil {
		t.Fatal("no request captured")
	}
	if !strings.HasSuffix(doer.got.URL.Path, "/metadata") {
		t.Errorf("request URL = %q; want path ending with /metadata", doer.got.URL.String())
	}

	// Verify Accept header.
	if got := doer.got.Header.Get("Accept"); got != "application/fhir+json" {
		t.Errorf("Accept = %q; want application/fhir+json", got)
	}

	// Verify fhir_version is surfaced in metadata.
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata unmarshal: %v", err)
	}
	if meta["fhir_version"] != "4.0.1" {
		t.Errorf("metadata fhir_version = %v; want 4.0.1", meta["fhir_version"])
	}
	// resource_count comes back as float64 from JSON unmarshal.
	if meta["resource_count"] != float64(2) {
		t.Errorf("metadata resource_count = %v; want 2", meta["resource_count"])
	}
}

func TestFHIRChecker_Run_NonOKStatus(t *testing.T) {
	t.Parallel()

	doer := &fhirDoer{resp: fhirMockResponse(404, "not found")}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fhir",
		Target: "https://fhir.example.com",
	})

	if r.Success {
		t.Error("Success = true for 404; want false")
	}
	if !strings.Contains(r.Error, "404") {
		t.Errorf("Error = %q; want status code 404 mentioned", r.Error)
	}
}

func TestFHIRChecker_Run_TransportError(t *testing.T) {
	t.Parallel()

	doer := &fhirDoer{err: errors.New("connection refused")}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fhir",
		Target: "https://fhir.example.com",
	})

	if r.Success {
		t.Error("Success = true on transport error; want false")
	}
	if r.Error == "" {
		t.Error("Error is empty; expected transport error message")
	}
}

func TestFHIRChecker_Run_InvalidJSON(t *testing.T) {
	t.Parallel()

	doer := &fhirDoer{resp: fhirMockResponse(200, "not json at all")}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fhir",
		Target: "https://fhir.example.com",
	})

	if r.Success {
		t.Error("Success = true on unparseable body; want false")
	}
}

func TestFHIRChecker_Run_BasicAuth(t *testing.T) {
	t.Parallel()

	doer := &fhirDoer{resp: fhirMockResponse(200, validCapabilityStatement())}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fhir",
		Target: "https://fhir.example.com",
		Params: json.RawMessage(`{"auth_type":"basic","username":"admin","password":"secret"}`),
	})

	if !r.Success {
		t.Errorf("Success = false with basic auth; error: %s", r.Error)
	}
	if doer.got == nil {
		t.Fatal("no request captured")
	}
	authHeader := doer.got.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("Authorization = %q; want Basic ... header", authHeader)
	}
}

func TestFHIRChecker_Run_BasicAuthMissingUsername(t *testing.T) {
	t.Parallel()

	doer := &fhirDoer{resp: fhirMockResponse(200, validCapabilityStatement())}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fhir",
		Target: "https://fhir.example.com",
		Params: json.RawMessage(`{"auth_type":"basic"}`),
	})

	if r.Success {
		t.Error("Success = true; basic auth with no username should fail")
	}
}

func TestFHIRChecker_Run_BearerAuth(t *testing.T) {
	t.Parallel()

	doer := &fhirDoer{resp: fhirMockResponse(200, validCapabilityStatement())}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fhir",
		Target: "https://fhir.example.com",
		Params: json.RawMessage(`{"auth_type":"bearer","bearer_token":"tok123"}`),
	})

	if !r.Success {
		t.Errorf("Success = false with bearer auth; error: %s", r.Error)
	}
	if got := doer.got.Header.Get("Authorization"); got != "Bearer tok123" {
		t.Errorf("Authorization = %q; want Bearer tok123", got)
	}
}

func TestFHIRChecker_Run_MetadataURLSuffix(t *testing.T) {
	t.Parallel()

	// Target with a trailing slash should still produce exactly one /metadata suffix.
	doer := &fhirDoer{resp: fhirMockResponse(200, validCapabilityStatement())}
	c := checkers.NewFHIRChecker().WithFHIRDoer(doer)

	_ = c.Run(context.Background(), probe.Probe{
		Kind:   "fhir",
		Target: "https://fhir.example.com/",
	})

	if doer.got == nil {
		t.Fatal("no request captured")
	}
	if !strings.HasSuffix(doer.got.URL.String(), "/metadata") {
		t.Errorf("URL = %q; want suffix /metadata (no double slash)", doer.got.URL.String())
	}
}
