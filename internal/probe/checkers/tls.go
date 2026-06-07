package checkers

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// hoursPerDay is used to convert duration → days for cert expiry.
const hoursPerDay = 24

// defaultTLSDialTimeout caps one TLS handshake attempt.
const defaultTLSDialTimeout = 10 * time.Second

// defaultTLSPort is used when Probe.Target omits a port.
const defaultTLSPort = "443"

// TLSParams is the kind-specific params shape for TLS-handshake
// probes. Empty SNI uses the host from Target.
type TLSParams struct {
	// SNI overrides the Server Name Indication sent during the
	// handshake. Useful when probing a server that hosts multiple
	// virtual hosts behind one IP.
	SNI string `json:"sni,omitempty"`

	// Port override; default 443. If Target already includes ":port"
	// this field is ignored.
	Port string `json:"port,omitempty"`
}

// TLSCertInfo is the cert-leaf data published to Result.Metadata.
type TLSCertInfo struct {
	Subject           string `json:"subject"`
	Issuer            string `json:"issuer"`
	NotBefore         string `json:"not_before"`
	NotAfter          string `json:"not_after"`
	DaysRemaining     int    `json:"days_remaining"`
	SHA256Fingerprint string `json:"sha256_fingerprint"`
	TLSVersion        string `json:"tls_version"`
	SNI               string `json:"sni"`
}

// TLSDialer captures the minimum surface a TLSChecker needs.
// Production wires this to [net.Dialer] + crypto/tls; tests inject a
// fake that returns predetermined certificates.
type TLSDialer interface {
	Dial(ctx context.Context, addr, sni string) (TLSConnState, error)
}

// TLSConnState abstracts the post-handshake state. Implementations
// expose the peer certificate chain + negotiated TLS version.
type TLSConnState struct {
	PeerCertificates []*x509.Certificate
	Version          uint16
}

// TLSChecker implements probe.Checker for Kind="tls". Performs a TLS
// handshake against the configured target and captures cert
// metadata. Days-remaining computed but not threshold-evaluated
// here — alerts pipeline reads Metadata.days_remaining and applies
// kind-specific rules (V1.0+).
type TLSChecker struct {
	dialer TLSDialer
}

// NewTLSChecker returns a TLSChecker wired to a production [net.Dialer]
// + crypto/tls. Tests inject a fake via WithTLSDialer.
func NewTLSChecker() *TLSChecker {
	return &TLSChecker{dialer: realTLSDialer{}}
}

// WithTLSDialer swaps the dialer; used by tests.
func (c *TLSChecker) WithTLSDialer(d TLSDialer) *TLSChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindTLS.
func (c *TLSChecker) Kind() string { return probe.KindTLS }

// RequiredCapabilities returns nil — TLS handshake needs no special
// capability.
func (c *TLSChecker) RequiredCapabilities() []string { return nil }

// Run performs a TLS handshake against Probe.Target and returns a
// Result containing handshake latency + cert metadata in JSON.
// Probe.Target shape: "host" or "host:port"; port defaults to 443.
func (c *TLSChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseTLSParams(p.Params)

	addr, host := resolveTLSAddr(p.Target, params.Port)
	sni := params.SNI
	if sni == "" {
		sni = host
	}

	start := time.Now()
	state, err := c.dialer.Dial(ctx, addr, sni)
	latencyMs := float64(time.Since(start).Milliseconds())

	if err != nil {
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			LatencyMs: latencyMs,
			Error:     err.Error(),
		}
	}

	if len(state.PeerCertificates) == 0 {
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			LatencyMs: latencyMs,
			Error:     "TLS handshake succeeded but server presented no certificate",
		}
	}

	info := extractCertInfo(state, sni)
	metaBytes, _ := json.Marshal(info)

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  metaBytes,
	}
}

// resolveTLSAddr returns the dial address (host:port) and the bare
// host (for SNI fallback) from the Probe.Target + optional Params.
func resolveTLSAddr(target, paramPort string) (string, string) {
	if h, _, splitErr := net.SplitHostPort(target); splitErr == nil {
		// Target already includes ":port" — honor it.
		return target, h
	}
	port := paramPort
	if port == "" {
		port = defaultTLSPort
	}
	return net.JoinHostPort(target, port), target
}

// extractCertInfo populates TLSCertInfo from the post-handshake state.
func extractCertInfo(state TLSConnState, sni string) TLSCertInfo {
	cert := state.PeerCertificates[0]

	issuer := ""
	switch {
	case len(cert.Issuer.Organization) > 0:
		issuer = cert.Issuer.Organization[0]
	case cert.Issuer.CommonName != "":
		issuer = cert.Issuer.CommonName
	}

	daysLeft := int(time.Until(cert.NotAfter).Hours() / hoursPerDay)

	fpBytes := sha256.Sum256(cert.Raw)
	fp := hex.EncodeToString(fpBytes[:])

	return TLSCertInfo{
		Subject:           cert.Subject.CommonName,
		Issuer:            issuer,
		NotBefore:         cert.NotBefore.UTC().Format(time.RFC3339),
		NotAfter:          cert.NotAfter.UTC().Format(time.RFC3339),
		DaysRemaining:     daysLeft,
		SHA256Fingerprint: fp,
		TLSVersion:        tlsVersionString(state.Version),
		SNI:               sni,
	}
}

// tlsVersionString renders a uint16 TLS version constant as a label
// suitable for Metadata + UI.
func tlsVersionString(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", v)
	}
}

// parseTLSParams decodes the params JSON; returns zero on empty input.
func parseTLSParams(raw json.RawMessage) TLSParams {
	if len(raw) == 0 {
		return TLSParams{}
	}
	var p TLSParams
	_ = json.Unmarshal(raw, &p)
	return p
}

// realTLSDialer is the production dialer — [net.Dialer] + crypto/tls
// with InsecureSkipVerify=true so the checker can inspect expired or
// self-signed certs. Verification is the operator's choice; the
// probe surfaces cert metadata regardless.
type realTLSDialer struct{}

// Dial connects, performs the TLS handshake, and returns the
// resulting state. Honors ctx for both dial and handshake timeouts.
// Uses InsecureSkipVerify=true so the checker can inspect expired
// or self-signed certs — verification is the operator's choice.
//
//nolint:gosec // InsecureSkipVerify intentional; see method doc.
func (realTLSDialer) Dial(ctx context.Context, addr, sni string) (TLSConnState, error) {
	if !strings.Contains(addr, ":") {
		return TLSConnState{}, errors.New("addr must include port")
	}

	dialer := &net.Dialer{Timeout: defaultTLSDialTimeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return TLSConnState{}, fmt.Errorf("tcp dial: %w", err)
	}

	tlsConfig := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true, // see method docstring
	}
	conn := tls.Client(rawConn, tlsConfig)
	if hsErr := conn.HandshakeContext(ctx); hsErr != nil {
		_ = rawConn.Close()
		return TLSConnState{}, fmt.Errorf("tls handshake: %w", hsErr)
	}
	defer func() { _ = conn.Close() }()

	state := conn.ConnectionState()
	return TLSConnState{
		PeerCertificates: state.PeerCertificates,
		Version:          state.Version,
	}, nil
}
