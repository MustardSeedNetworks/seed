package checkers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultFHIRTimeout is the default request timeout for FHIR metadata probes.
const defaultFHIRTimeout = 30 * time.Second

// fhirMaxResponseBody caps the body read to 1 MiB, matching the legacy handler.
const fhirMaxResponseBody = 1 << 20

// FHIRParams is the kind-specific params shape for Kind="fhir".
// All fields are optional; AuthType defaults to "none".
type FHIRParams struct {
	AuthType    string `json:"auth_type,omitempty"`    // none|basic|bearer|oauth2, default none
	Username    string `json:"username,omitempty"`     // required for basic
	Password    string `json:"password,omitempty"`     // used with basic
	BearerToken string `json:"bearer_token,omitempty"` // required for bearer/oauth2
	TimeoutMs   int    `json:"timeout_ms,omitempty"`   // default 30000
}

// fhirCapabilityStatement represents the relevant fields of a FHIR
// CapabilityStatement resource (HL7 FHIR R4 §2.22).
type fhirCapabilityStatement struct {
	ResourceType string `json:"resourceType"`
	FHIRVersion  string `json:"fhirVersion"`
	Software     struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"software"`
	Rest []struct {
		Mode     string `json:"mode"`
		Resource []struct {
			Type string `json:"type"`
		} `json:"resource"`
	} `json:"rest"`
}

// FHIRChecker implements probe.Checker for Kind="fhir". It GETs the
// FHIR /metadata endpoint and validates the returned CapabilityStatement.
type FHIRChecker struct {
	doer  HTTPDoer
	clock func() time.Time
}

// NewFHIRChecker returns a FHIRChecker wired to a real HTTP client
// built per-Run from the probe params (timeout, auth).
func NewFHIRChecker() *FHIRChecker {
	return &FHIRChecker{clock: time.Now}
}

// WithFHIRDoer overrides the default per-request client; used by tests
// to inject a fake HTTPDoer.
func (c *FHIRChecker) WithFHIRDoer(d HTTPDoer) *FHIRChecker {
	c.doer = d
	return c
}

// Kind returns probe.KindFHIR ("fhir").
func (c *FHIRChecker) Kind() string { return probe.KindFHIR }

// RequiredCapabilities returns nil; FHIR probes need no special hardware.
func (c *FHIRChecker) RequiredCapabilities() []string { return nil }

// Run GETs <Target>/metadata, parses the FHIR CapabilityStatement, and
// reports Success=true when the response is 200 and the document is valid.
// Metadata includes status_code, fhir_version, server_name, and resource_count.
func (c *FHIRChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseFHIRParams(p.Params)

	timeout := defaultFHIRTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build the FHIR metadata URL from the probe Target.
	metadataURL := strings.TrimSuffix(p.Target, "/") + "/metadata"

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, metadataURL, http.NoBody)
	if err != nil {
		return fhirFailResult(p, c.clock().UTC(), 0, fmt.Sprintf("invalid request: %v", err))
	}
	req.Header.Set("Accept", "application/fhir+json")

	// Apply authentication before sending.
	if authErr := fhirApplyAuth(req, params); authErr != nil {
		return fhirFailResult(p, c.clock().UTC(), 0, fmt.Sprintf("authentication failed: %v", authErr))
	}

	doer := c.doer
	if doer == nil {
		doer = &http.Client{Timeout: timeout}
	}

	start := c.clock()
	resp, err := doer.Do(req)
	latencyMs := float64(c.clock().Sub(start).Milliseconds())
	if err != nil {
		return fhirFailResult(p, c.clock().UTC(), latencyMs, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fhirFailResult(p, c.clock().UTC(), latencyMs,
			fmt.Sprintf("Unexpected status: %d", resp.StatusCode))
	}

	// Read and parse the CapabilityStatement (body capped at 1 MiB).
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, fhirMaxResponseBody))
	if readErr != nil {
		return fhirFailResult(p, c.clock().UTC(), latencyMs,
			fmt.Sprintf("failed to read response: %v", readErr))
	}

	var capStmt fhirCapabilityStatement
	if parseErr := json.Unmarshal(body, &capStmt); parseErr != nil {
		return fhirFailResult(p, c.clock().UTC(), latencyMs,
			fmt.Sprintf("failed to parse CapabilityStatement: %v", parseErr))
	}

	// Build a composite server_name from software.name + software.version.
	serverName := capStmt.Software.Name
	if capStmt.Software.Version != "" {
		serverName += " " + capStmt.Software.Version
	}

	// Count resource types across all REST conformance groups.
	resourceCount := 0
	for _, rest := range capStmt.Rest {
		resourceCount += len(rest.Resource)
	}

	meta, _ := json.Marshal(map[string]any{
		"status_code":    resp.StatusCode,
		"fhir_version":   capStmt.FHIRVersion,
		"server_name":    serverName,
		"resource_count": resourceCount,
	})

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: c.clock().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// fhirFailResult builds a failed Result with a timestamp and latency.
func fhirFailResult(p probe.Probe, ts time.Time, latencyMs float64, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: ts,
		Success:   false,
		LatencyMs: latencyMs,
		Error:     msg,
	}
}

// fhirApplyAuth sets the appropriate Authorization header on req based
// on params.AuthType. Mirrors the legacy applyFHIRAuth logic exactly.
func fhirApplyAuth(req *http.Request, params FHIRParams) error {
	switch strings.ToLower(params.AuthType) {
	case "", "none":
		// No authentication required.
		return nil

	case "basic":
		if params.Username == "" {
			return errors.New("basic auth requires username")
		}
		encoded := base64.StdEncoding.EncodeToString(
			[]byte(params.Username + ":" + params.Password),
		)
		req.Header.Set("Authorization", "Basic "+encoded)

	case "bearer":
		if params.BearerToken == "" {
			return errors.New("bearer auth requires token")
		}
		req.Header.Set("Authorization", "Bearer "+params.BearerToken)

	case "oauth2":
		// Simplified OAuth2: expects a pre-obtained bearer token.
		if params.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+params.BearerToken)
		} else {
			return errors.New("oauth2 auth requires token_url and credentials (not yet implemented)")
		}

	default:
		return fmt.Errorf("unknown auth type: %s", params.AuthType)
	}

	return nil
}

// parseFHIRParams deserialises Probe.Params into FHIRParams. An empty
// or nil message returns the zero value (all defaults).
func parseFHIRParams(raw json.RawMessage) FHIRParams {
	if len(raw) == 0 {
		return FHIRParams{}
	}
	var p FHIRParams
	_ = json.Unmarshal(raw, &p)
	return p
}
