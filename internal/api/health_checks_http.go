package api

// health_checks_http.go contains HTTP endpoint tests (both the lightweight
// timing-only path and the enhanced path with body matching and redirect
// following), plus the helpers and types they share.

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// Maximum bytes to read from response body for pattern matching.
const maxBodyReadBytes = 64 * 1024 // 64KB

// Default max redirects if not specified.
const defaultMaxRedirects = 10

// httpTimings is the per-phase HTTP timing breakdown captured via httptrace.
type httpTimings struct {
	DNS     float64
	Connect float64
	TLS     float64
	TTFB    float64 // Time to first byte (from request sent to first response byte)
	Total   float64
}

// httpResponse contains extended HTTP response data for enhanced checks.
type httpResponse struct {
	StatusCode   int
	Headers      http.Header
	Body         []byte // Limited to maxBodyReadBytes
	BodySize     int64  // Total content length (may be larger than Body)
	HTTPVersion  string // HTTP/1.1, HTTP/2, etc.
	Timings      httpTimings
	RedirectHops []RedirectHop
}

// runHTTPTests runs all configured HTTP endpoint tests and returns results.
func (s *Server) runHTTPTests(ctx context.Context, logger *slog.Logger) []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.HTTPEndpoints))

	for _, endpoint := range s.config.HealthChecks.HTTPEndpoints {
		if !endpoint.Enabled {
			continue
		}

		if err := validation.ValidateURL(endpoint.URL); err != nil {
			logger.WarnContext(ctx, "Skipping invalid HTTP endpoint URL", "url", endpoint.URL, "error", err)
			continue
		}

		result := s.runSingleHTTPTest(ctx, endpoint)
		results = append(results, result)
	}
	return results
}

// populateEnhancedTimings copies timing data from enhanced response to test result.
func populateEnhancedTimings(result *CustomTestResult, resp *httpResponse) {
	result.Status = resp.StatusCode
	result.Latency = resp.Timings.Total
	result.DNSLatency = resp.Timings.DNS
	result.TCPConnect = resp.Timings.Connect
	result.TLSLatency = resp.Timings.TLS
	result.TTFBLatency = resp.Timings.TTFB
	result.ResponseSize = resp.BodySize
	result.HTTPVersion = resp.HTTPVersion
	if len(resp.RedirectHops) > 0 {
		result.RedirectChain = resp.RedirectHops
	}
}

// evaluateBodyMatch checks body content against expected pattern and updates test result.
func evaluateBodyMatch(result *CustomTestResult, body []byte, pattern string, isRegex bool) {
	matched, matchErr := checkBodyMatch(body, pattern, isRegex)
	result.BodyMatchSuccess = matched

	switch {
	case matchErr != nil:
		result.BodyMatchStatus = statusError
		result.Error = matchErr.Error()
		result.Success = false
		result.TestStatus = statusError
	case !matched:
		result.BodyMatchStatus = statusError
		result.Success = false
		result.TestStatus = statusError
		result.Error = "Body content did not match expected pattern"
	default:
		result.BodyMatchStatus = statusSuccess
	}
}

// runEnhancedHTTPPath runs the enhanced HTTP test path with body reading and redirect following.
// Returns the final URL (after any fallback) for certificate checking.
func (s *Server) runEnhancedHTTPPath(
	ctx context.Context,
	endpoint config.HTTPEndpoint,
	url string,
	tryHTTPFallback bool,
	result *CustomTestResult,
	thresholds *config.CustomThresholds,
) string {
	resp, err := runHTTPTestEnhanced(
		ctx, url, endpoint.ExpectedStatus, endpoint.FollowRedirects, endpoint.MaxRedirects,
	)

	// Try HTTP fallback if HTTPS failed
	if err != nil && tryHTTPFallback {
		httpURL := "http://" + endpoint.URL
		httpResp, httpErr := runHTTPTestEnhanced(
			ctx, httpURL, endpoint.ExpectedStatus, endpoint.FollowRedirects, endpoint.MaxRedirects,
		)
		if httpErr == nil || (httpResp != nil && httpResp.StatusCode > 0) {
			url = httpURL
			result.URL = httpURL
			resp, err = httpResp, httpErr
		}
	}

	// Handle nil response with error
	if resp == nil {
		if err != nil {
			result.Success = false
			result.Error = errHTTPReqFailed
			result.TestStatus = statusError
		}
		return url
	}

	// Populate timing data
	populateEnhancedTimings(result, resp)

	// Handle request error
	if err != nil {
		result.Success = false
		result.Error = errHTTPReqFailed
		result.TestStatus = statusError
		return url
	}

	// Success path
	result.Success = true
	s.evaluateHTTPTimings(result, resp.Timings, thresholds)

	// Check body match if configured
	if endpoint.BodyMatch != "" {
		evaluateBodyMatch(result, resp.Body, endpoint.BodyMatch, endpoint.BodyMatchIsRegex)
	}

	// Check security headers if configured
	if endpoint.CheckSecurityHeaders && result.Success {
		isHTTPS := strings.HasPrefix(url, "https://")
		result.SecurityHeaders = checkSecurityHeaders(resp.Headers, isHTTPS)
	}

	return url
}

// runStandardHTTPPath runs the standard HTTP test path (faster, no body reading).
// Returns the final URL (after any fallback) for certificate checking.
func (s *Server) runStandardHTTPPath(
	ctx context.Context,
	endpoint config.HTTPEndpoint,
	url string,
	tryHTTPFallback bool,
	result *CustomTestResult,
	thresholds *config.CustomThresholds,
) string {
	statusCode, timings, err := runHTTPTest(ctx, url, endpoint.ExpectedStatus)

	// Try HTTP fallback if HTTPS failed
	if err != nil && tryHTTPFallback {
		httpURL := "http://" + endpoint.URL
		httpStatus, fallbackTimings, httpErr := runHTTPTest(ctx, httpURL, endpoint.ExpectedStatus)
		if httpErr == nil || httpStatus > 0 {
			url = httpURL
			result.URL = httpURL
			statusCode, timings, err = httpStatus, fallbackTimings, httpErr
		}
	}

	// Populate timing data
	result.Status = statusCode
	result.Latency = timings.Total
	result.DNSLatency = timings.DNS
	result.TCPConnect = timings.Connect
	result.TLSLatency = timings.TLS
	result.TTFBLatency = timings.TTFB

	if err != nil {
		result.Success = false
		result.Error = errHTTPReqFailed
		result.TestStatus = statusError
	} else {
		result.Success = true
		s.evaluateHTTPTimings(result, timings, thresholds)
	}

	return url
}

// runSingleHTTPTest runs a single HTTP endpoint test.
func (s *Server) runSingleHTTPTest(
	ctx context.Context,
	endpoint config.HTTPEndpoint,
) CustomTestResult {
	thresholds := s.config.Thresholds.CustomTests

	url, tryHTTPFallback := normalizeHTTPURL(endpoint.URL)
	name := endpoint.Name
	if name == "" {
		name = endpoint.URL
	}

	testResult := CustomTestResult{Name: name, URL: url}

	// Determine if we need enhanced testing (body match, security headers, or redirects)
	needsEnhanced := endpoint.BodyMatch != "" || endpoint.CheckSecurityHeaders || endpoint.FollowRedirects

	if needsEnhanced {
		url = s.runEnhancedHTTPPath(ctx, endpoint, url, tryHTTPFallback, &testResult, &thresholds)
	} else {
		url = s.runStandardHTTPPath(ctx, endpoint, url, tryHTTPFallback, &testResult, &thresholds)
	}

	// Check certificate expiry for HTTPS URLs
	if strings.HasPrefix(url, "https://") && testResult.Success {
		s.evaluateCertExpiry(&testResult, url, thresholds.CertExpiry)
	}

	return testResult
}

// normalizeHTTPURL adds scheme if missing and returns whether HTTP fallback should be tried.
func normalizeHTTPURL(rawURL string) (string, bool) {
	if rawURL == "" {
		return rawURL, false
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL, false
	}
	return "https://" + rawURL, true
}

// newHTTPTimingTrace creates an [httptrace.ClientTrace] that records timings to the provided httpTimings struct.
func newHTTPTimingTrace(timing *httpTimings) *httptrace.ClientTrace {
	var dnsStart, connStart, tlsStart, wroteRequest time.Time
	return &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				timing.DNS += time.Since(dnsStart).Seconds() * millisecondsPerSecond
			}
		},
		ConnectStart: func(_, _ string) {
			connStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			if !connStart.IsZero() {
				timing.Connect += time.Since(connStart).Seconds() * millisecondsPerSecond
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStart.IsZero() {
				timing.TLS += time.Since(tlsStart).Seconds() * millisecondsPerSecond
			}
		},
		WroteRequest: func(httptrace.WroteRequestInfo) {
			wroteRequest = time.Now()
		},
		GotFirstResponseByte: func() {
			if !wroteRequest.IsZero() {
				timing.TTFB = time.Since(wroteRequest).Seconds() * millisecondsPerSecond
			}
		},
	}
}

// runHTTPTest runs an HTTP test and returns status code and timings in ms.
// Uses SafeTransport to prevent DNS rebinding SSRF attacks.
func runHTTPTest(ctx context.Context, url string, expectedStatus int) (int, httpTimings, error) {
	var timing httpTimings
	// Use SafeTransport to block connections to private IPs (prevents DNS rebinding)
	transport := validation.SafeTransport()
	client := &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeoutSec * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, timing, err
	}

	trace := newHTTPTimingTrace(&timing)
	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	start := time.Now()
	resp, err := client.Do(req)
	timing.Total = time.Since(start).Seconds() * millisecondsPerSecond

	if err != nil {
		return 0, timing, err
	}
	defer func() { _ = resp.Body.Close() }()

	statusCode := resp.StatusCode
	if expectedStatus > 0 && statusCode != expectedStatus {
		return statusCode, timing, fmt.Errorf("expected %d, got %d", expectedStatus, statusCode)
	}

	return statusCode, timing, nil
}

// runHTTPTestEnhanced runs an HTTP test with body reading and redirect following.
// Returns full response details for body matching and security header checks.
func runHTTPTestEnhanced(
	ctx context.Context,
	url string,
	expectedStatus int,
	followRedirects bool,
	maxRedirects int,
) (*httpResponse, error) {
	result := &httpResponse{
		RedirectHops: make([]RedirectHop, 0),
	}

	if maxRedirects <= 0 {
		maxRedirects = defaultMaxRedirects
	}

	transport := validation.SafeTransport()

	currentURL := url
	var lastTiming httpTimings

	for i := 0; i <= maxRedirects; i++ {
		hopStart := time.Now()
		resp, timing, err := runSingleHTTPRequest(ctx, currentURL, transport)
		if err != nil {
			return nil, err
		}

		lastTiming = timing

		// Check if this is a redirect
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			_ = resp.Body.Close()

			hop := RedirectHop{
				URL:        currentURL,
				StatusCode: resp.StatusCode,
				LatencyMs:  time.Since(hopStart).Seconds() * millisecondsPerSecond,
			}
			result.RedirectHops = append(result.RedirectHops, hop)

			if !followRedirects || location == "" {
				// Return the redirect response without following
				result.StatusCode = resp.StatusCode
				result.Headers = resp.Header
				result.HTTPVersion = formatHTTPVersion(resp.Proto)
				result.Timings = lastTiming
				return result, nil
			}

			// Resolve redirect URL (handles both absolute and relative URLs)
			currentURL = resolveRedirectURL(currentURL, location)
			continue
		}

		// Non-redirect response - read body and return
		result.StatusCode = resp.StatusCode
		result.Headers = resp.Header
		result.HTTPVersion = formatHTTPVersion(resp.Proto)
		result.Timings = lastTiming

		// Read body (limited)
		body, size := readLimitedBody(resp.Body)
		_ = resp.Body.Close()
		result.Body = body
		result.BodySize = size

		// Check expected status
		if expectedStatus > 0 && result.StatusCode != expectedStatus {
			return result, fmt.Errorf("expected %d, got %d", expectedStatus, result.StatusCode)
		}

		return result, nil
	}

	return nil, fmt.Errorf("too many redirects (max %d)", maxRedirects)
}

// resolveRedirectURL resolves a Location header value against the current URL.
// Handles both absolute URLs and relative paths.
func resolveRedirectURL(currentURL, location string) string {
	// Already absolute
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		return location
	}

	// Find the scheme separator
	schemeEnd := strings.Index(currentURL, "//")
	if schemeEnd < 0 {
		return location
	}

	// Absolute path (starts with /)
	if strings.HasPrefix(location, "/") {
		hostEnd := strings.Index(currentURL[schemeEnd+2:], "/")
		if hostEnd >= 0 {
			return currentURL[:schemeEnd+2+hostEnd] + location
		}
		return currentURL + location
	}

	// Relative path
	lastSlash := strings.LastIndex(currentURL, "/")
	if lastSlash > httpsSchemeLen { // After "https://"
		return currentURL[:lastSlash+1] + location
	}

	return location
}

// runSingleHTTPRequest performs a single HTTP request without following redirects.
func runSingleHTTPRequest(
	ctx context.Context,
	url string,
	transport http.RoundTripper,
) (*http.Response, httpTimings, error) {
	var timing httpTimings
	client := &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeoutSec * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, timing, err
	}

	trace := newHTTPTimingTrace(&timing)
	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	start := time.Now()
	resp, err := client.Do(req)
	timing.Total = time.Since(start).Seconds() * millisecondsPerSecond

	return resp, timing, err
}

// formatHTTPVersion formats the HTTP protocol version.
func formatHTTPVersion(proto string) string {
	switch proto {
	case "HTTP/2.0":
		return "HTTP/2"
	case "HTTP/3":
		return "HTTP/3"
	default:
		return proto
	}
}

// readLimitedBody reads up to maxBodyReadBytes from body and returns total size.
func readLimitedBody(body io.ReadCloser) ([]byte, int64) {
	limitedReader := io.LimitReader(body, maxBodyReadBytes)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, 0
	}

	// Try to determine total size
	size := int64(len(data))

	// Read and discard remaining bytes to get total size
	remaining, _ := io.Copy(io.Discard, body)
	size += remaining

	return data, size
}

// checkBodyMatch checks if response body matches the expected pattern.
func checkBodyMatch(body []byte, pattern string, isRegex bool) (bool, error) {
	if pattern == "" {
		return true, nil
	}

	bodyStr := string(body)

	if isRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return re.MatchString(bodyStr), nil
	}

	// Substring match
	return strings.Contains(bodyStr, pattern), nil
}

// evaluateHTTPTimings sets timing statuses and overall test status.
func (s *Server) evaluateHTTPTimings(
	result *CustomTestResult,
	timings httpTimings,
	thresholds *config.CustomThresholds,
) {
	httpTimingThresholds := thresholds.HTTPTimings

	result.DNSStatus = getTestStatus(
		timings.DNS,
		httpTimingThresholds.DNS.Warning.Milliseconds(),
		httpTimingThresholds.DNS.Critical.Milliseconds(),
	)
	result.TCPStatus = getTestStatus(
		timings.Connect,
		httpTimingThresholds.TCP.Warning.Milliseconds(),
		httpTimingThresholds.TCP.Critical.Milliseconds(),
	)
	result.TLSStatus = getTestStatus(
		timings.TLS,
		httpTimingThresholds.TLS.Warning.Milliseconds(),
		httpTimingThresholds.TLS.Critical.Milliseconds(),
	)
	result.TTFBStatus = getTestStatus(
		timings.TTFB,
		httpTimingThresholds.TTFB.Warning.Milliseconds(),
		httpTimingThresholds.TTFB.Critical.Milliseconds(),
	)

	switch {
	case result.DNSStatus == statusError || result.TCPStatus == statusError ||
		result.TLSStatus == statusError || result.TTFBStatus == statusError:
		result.TestStatus = statusError
	case result.DNSStatus == statusWarning || result.TCPStatus == statusWarning ||
		result.TLSStatus == statusWarning || result.TTFBStatus == statusWarning:
		result.TestStatus = statusWarning
	default:
		result.TestStatus = getTestStatus(
			timings.Total,
			thresholds.HTTP.Warning.Milliseconds(),
			thresholds.HTTP.Critical.Milliseconds(),
		)
	}
}
