// hibp.go — Have-I-Been-Pwned breach-corpus check for package auth.
//
// CheckPasswordBreached uses HIBP's k-anonymity range API: the client
// SHA-1s the password, sends the first five hex characters of the
// digest to https://api.pwnedpasswords.com/range/{prefix}, and scans
// the response for the remaining 35 characters. The plaintext password
// is never transmitted.
//
// Per the audit notes for task #86 this check is opt-out: when the
// environment variable SEED_DISABLE_HIBP=1 is set the function skips
// the network call and returns (false, 0, nil), preserving the
// air-gapped/offline deployment story.
//
// Network failures (timeout, DNS, closed port, non-2xx) are
// intentionally non-fatal: they log a WARN and return (false, 0, nil)
// so a temporary HIBP outage cannot lock operators out of password
// rotation.

package auth

import (
	"bufio"
	"context"
	"crypto/sha1" // #nosec G505 -- HIBP API contractually requires SHA-1 prefix lookup; not used as a security primitive.
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// HIBP integration constants.
const (
	hibpEnvDisable     = "SEED_DISABLE_HIBP"
	hibpRangeURL       = "https://api.pwnedpasswords.com/range/"
	hibpUserAgent      = "seed/security-check"
	hibpRequestTimeout = 5 * time.Second
	hibpPrefixLen      = 5
	hibpSuffixLen      = 35 // SHA-1 hex = 40 chars, minus 5-char prefix = 35

	// padding-aware bucket — many implementations send a random count of
	// padding lines; we just ignore lines that don't parse.
	hibpResponseMaxBytes = int64(2 << 20) // 2 MiB hard cap
)

// hibpEnvEndpoint is the environment variable name tests use to point
// CheckPasswordBreached at a [httptest.Server] instead of the public
// HIBP endpoint. Empty / unset means "use the public endpoint".
const hibpEnvEndpoint = "SEED_HIBP_ENDPOINT_TEST"

// hibpEndpoint returns the active HIBP range URL — the env override
// when present (test seam), otherwise the public endpoint.
func hibpEndpoint() string {
	if override := strings.TrimSpace(os.Getenv(hibpEnvEndpoint)); override != "" {
		return override
	}
	return hibpRangeURL
}

// hibpClient returns the HTTP client used to call HIBP. The client is
// created fresh per invocation; the [http.Client] internals are cheap
// to construct (no connection pool persistence required for a one-shot
// k-anonymity lookup against a single host).
func hibpClient() *http.Client {
	return &http.Client{Timeout: hibpRequestTimeout}
}

// CheckPasswordBreached returns whether the given password appears in
// the HIBP breach corpus, along with the number of times it has been
// seen (0 when the suffix is not in the response or when the check is
// disabled).
//
// Returns (false, 0, nil) — never an error — when SEED_DISABLE_HIBP=1
// or when the network call fails for any reason. This is deliberate:
// HIBP unreachability must not block legitimate password rotations on
// air-gapped or restricted-egress deployments.
func CheckPasswordBreached(ctx context.Context, password string) (bool, int, error) {
	if hibpDisabled() {
		return false, 0, nil
	}

	prefix, suffix := sha1PrefixAndSuffix(password)
	body, fetchErr := fetchHIBPRange(ctx, prefix)
	if fetchErr != nil {
		logging.FromContext(ctx).
			WarnContext(ctx, "HIBP breach check failed; treating as not-breached",
				"error", fetchErr,
				"event", "auth.password.hibp_unreachable")
		return false, 0, nil
	}

	count := scanHIBPResponse(body, suffix)
	return count > 0, count, nil
}

// hibpDisabled returns true when the env-var opt-out is set to "1".
func hibpDisabled() bool {
	return strings.TrimSpace(os.Getenv(hibpEnvDisable)) == "1"
}

// sha1PrefixAndSuffix returns the (5-char prefix, 35-char suffix)
// uppercase hex of SHA-1(password) as required by the HIBP API.
func sha1PrefixAndSuffix(password string) (string, string) {
	sum := sha1.Sum([]byte(password)) // #nosec G401 -- HIBP API contract; see file header.
	full := strings.ToUpper(hex.EncodeToString(sum[:]))
	return full[:hibpPrefixLen], full[hibpPrefixLen:]
}

// fetchHIBPRange performs the k-anonymity range request and returns the
// response body bounded to hibpResponseMaxBytes.
func fetchHIBPRange(ctx context.Context, prefix string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, hibpRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, hibpEndpoint()+prefix, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build hibp request: %w", err)
	}
	req.Header.Set("User-Agent", hibpUserAgent)
	req.Header.Set("Add-Padding", "true")

	resp, err := hibpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("hibp request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hibp unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, hibpResponseMaxBytes))
	if err != nil {
		return "", fmt.Errorf("read hibp body: %w", err)
	}
	return string(body), nil
}

// scanHIBPResponse parses the API response, which is a CRLF-delimited
// list of "SUFFIX:COUNT" entries (with optional padding lines that
// have a count of 0), and returns the count for the given suffix or
// zero if absent.
func scanHIBPResponse(body, suffix string) int {
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon != hibpSuffixLen {
			// not a well-formed suffix:count line
			continue
		}
		if !strings.EqualFold(line[:colon], suffix) {
			continue
		}
		count, parseErr := strconv.Atoi(line[colon+1:])
		if parseErr != nil {
			return 0
		}
		return count
	}
	return 0
}
