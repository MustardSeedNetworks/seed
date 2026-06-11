package checkers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultLDAPTimeout is the per-attempt LDAP probe timeout, matching the
// legacy internal/api/handlers_enterprise_checks.go LDAPTimeout.
const defaultLDAPTimeout = 10 * time.Second

// defaultLDAPPort is the standard LDAP port (IANA 389).
const defaultLDAPPort = 389

// defaultLDAPSPort is the standard LDAPS port (IANA 636) used when UseTLS=true.
const defaultLDAPSPort = 636

// LDAPParams is the kind-specific params shape for LDAP reachability
// probes. All fields are optional and fall back to safe defaults.
//
// V1.0 scope: TCP reachability + optional TLS handshake only.
// Authenticated bind and LDAP search require an LDAP library and are
// out of scope until V1.1+ (see ADR-0027 P1 notes).
type LDAPParams struct {
	// Port overrides the target port. Default: 636 when UseTLS=true, 389
	// otherwise.
	Port int `json:"port,omitempty"`

	// UseTLS opens an LDAPS (TLS-from-the-start) connection instead of
	// plain TCP. Validates server certificates; MinVersion TLS 1.2.
	UseTLS bool `json:"use_tls,omitempty"`

	// BindDN is recorded in metadata to indicate that bind credentials
	// are configured. No bind is attempted in V1.0 — an LDAP library is
	// required for that.
	BindDN string `json:"bind_dn,omitempty"`

	// TimeoutMs overrides the per-attempt dial timeout. Default 10000.
	TimeoutMs int `json:"timeout_ms,omitempty"`
}

// LDAPChecker implements probe.Checker for Kind="ldap". It verifies TCP
// reachability of an LDAP/LDAPS endpoint and, when UseTLS=true, performs
// a TLS 1.2+ handshake to validate the server certificate.
//
// V1.0 scope — reachability only: TCP connectivity succeeds on connect
// (plain) or TLS handshake (UseTLS). Authenticated bind and directory
// search are not performed; an LDAP library is required for those
// operations (V1.1+). BindDN presence is surfaced in metadata only.
//
// The plain-TCP path is injectable via PingDialer for unit testing.
// The TLS path constructs a [tls.Dialer] directly (mirrors tls.go).
type LDAPChecker struct {
	dialer PingDialer
}

// NewLDAPChecker returns an LDAPChecker wired to a real dialer.
func NewLDAPChecker() *LDAPChecker {
	return &LDAPChecker{dialer: realPingDialer{}}
}

// WithLDAPDialer swaps the plain-TCP dialer (for tests).
func (c *LDAPChecker) WithLDAPDialer(d PingDialer) *LDAPChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindLDAP.
func (c *LDAPChecker) Kind() string { return probe.KindLDAP }

// RequiredCapabilities returns nil — TCP/LDAPS reachability needs no
// special hardware capability.
func (c *LDAPChecker) RequiredCapabilities() []string { return nil }

// Run dials Target:Port (plain TCP or TLS-from-the-start) and returns a
// Result. Success indicates the TCP connection was established and, when
// UseTLS=true, that the TLS handshake completed against a valid cert.
// The Metadata JSON carries the dialed addr, tls (bool), and
// bind_dn_configured (bool).
func (c *LDAPChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := ldapParseParams(p.Params)

	port := params.Port
	if port == 0 {
		port = ldapDefaultPort(params.UseTLS)
	}

	timeout := defaultLDAPTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(p.Target, strconv.Itoa(port))
	start := time.Now()

	var dialErr error
	if params.UseTLS {
		dialErr = ldapDialTLS(dialCtx, addr, p.Target)
	} else {
		var conn net.Conn
		conn, dialErr = c.dialer.Dial(dialCtx, "tcp", addr)
		if dialErr == nil {
			_ = conn.Close()
		}
	}

	latencyMs := float64(time.Since(start).Milliseconds())

	if dialErr != nil {
		return ldapFailure(p, latencyMs, dialErr.Error())
	}

	meta, _ := json.Marshal(map[string]any{
		metaKeyAddr:          addr,
		"tls":                params.UseTLS,
		"bind_dn_configured": params.BindDN != "",
	})

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// ldapDefaultPort returns the canonical port for plain (389) or TLS (636)
// LDAP connections.
func ldapDefaultPort(useTLS bool) int {
	if useTLS {
		return defaultLDAPSPort
	}
	return defaultLDAPPort
}

// ldapDialTLS opens a TLS-from-the-start connection to addr, validating
// the server certificate against the system trust store. ServerName is
// set to target (bare host without port) for SNI. MinVersion is TLS 1.2.
func ldapDialTLS(ctx context.Context, addr, target string) error {
	cfg := &tls.Config{
		ServerName: target,
		MinVersion: tls.VersionTLS12,
		// InsecureSkipVerify defaults to false — validate certs.
	}
	td := &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    cfg,
	}
	conn, err := td.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("ldaps dial: %w", err)
	}
	_ = conn.Close()
	return nil
}

// ldapFailure builds a failed Result with the measured latency.
func ldapFailure(p probe.Probe, latencyMs float64, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   false,
		LatencyMs: latencyMs,
		Error:     msg,
	}
}

// ldapParseParams decodes the params JSON; returns zero value on empty input.
func ldapParseParams(raw json.RawMessage) LDAPParams {
	if len(raw) == 0 {
		return LDAPParams{}
	}
	var p LDAPParams
	_ = json.Unmarshal(raw, &p)
	return p
}
