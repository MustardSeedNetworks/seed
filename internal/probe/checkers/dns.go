// Package checkers houses the probe.Checker implementations — one
// file per Kind. See msn-docs-internal/01-Strategy/SEED_ARCHITECTURE.md
// section 3.1 (probes engine) and section 5.2 (Stage A1 plan).
//
// V1.0 NMS expansion — Stage A1.4 onward (Checker port-in from
// the pre-refactor in-flight services).
package checkers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/diagnostics/dns"
	"github.com/krisarmstrong/seed/internal/probe"
)

// DNSParams is the kind-specific params shape for the DNS probe.
// Empty Server falls back to the OS resolver chain.
type DNSParams struct {
	// Server is the DNS server address (with optional :port). Empty
	// uses the system resolver.
	Server string `json:"server,omitempty"`

	// RecordType is the lookup record type. V1.0 supports "A" and
	// "AAAA"; default "A".
	RecordType string `json:"record_type,omitempty"`
}

// DNSResolver is the minimum surface a DNSChecker needs from the
// dns package. Tests inject a fake; production uses
// defaultResolverFactory which constructs dns.Tester instances.
type DNSResolver interface {
	ForwardLookupIPv4(ctx context.Context, hostname string) *dns.LookupResult
	ForwardLookupIPv6(ctx context.Context, hostname string) *dns.LookupResult
}

// DNSResolverFactory builds a DNSResolver for a given DNS server +
// query hostname. Empty server means "use system resolver." Returning
// a fresh Resolver per call lets callers vary the server per-probe.
type DNSResolverFactory func(server, hostname string) DNSResolver

// DNSChecker implements probe.Checker for Kind="dns". It performs
// forward lookups against a configured DNS server (or the system
// resolver) and reports latency + resolution status.
type DNSChecker struct {
	factory DNSResolverFactory
}

// NewDNSChecker returns a DNSChecker wired to the production
// dns.Tester. Pass a custom factory via WithDNSResolverFactory for
// tests.
func NewDNSChecker() *DNSChecker {
	return &DNSChecker{factory: defaultDNSResolverFactory}
}

// WithDNSResolverFactory swaps the factory — used in tests to inject
// a fake.
func (c *DNSChecker) WithDNSResolverFactory(f DNSResolverFactory) *DNSChecker {
	c.factory = f
	return c
}

// Kind returns probe.KindDNS.
func (c *DNSChecker) Kind() string { return probe.KindDNS }

// RequiredCapabilities returns nil — DNS lookup needs no special
// hardware capabilities.
func (c *DNSChecker) RequiredCapabilities() []string { return nil }

// Run performs the lookup configured by Probe.Target (hostname) and
// optional Probe.Params (server + record_type). Returns a Result
// with Success=true on lookup success, Success=false with an Error
// message otherwise. LatencyMs carries the round-trip time;
// Metadata carries the resolved IPs and the configured server +
// record type as JSON.
func (c *DNSChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseDNSParams(p.Params)
	server := params.Server
	hostname := p.Target

	resolver := c.factory(server, hostname)

	var lookup *dns.LookupResult
	switch params.RecordType {
	case "", "A":
		lookup = resolver.ForwardLookupIPv4(ctx, hostname)
	case "AAAA":
		lookup = resolver.ForwardLookupIPv6(ctx, hostname)
	default:
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error:     fmt.Sprintf("unsupported record_type %q (V1.0 supports A and AAAA)", params.RecordType),
		}
	}

	if lookup == nil {
		// Defensive: an injected resolver returning nil is a bug; surface
		// it rather than panic.
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error:     "dns resolver returned nil result",
		}
	}

	meta := map[string]any{
		"server":      server,
		"record_type": coalesce(params.RecordType, "A"),
		"status":      lookup.Status,
		"resolved":    lookup.Resolved,
	}
	metaBytes, _ := json.Marshal(meta)

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   lookup.Error == "",
		LatencyMs: float64(lookup.TimeMs),
		Error:     lookup.Error,
		Metadata:  metaBytes,
	}
}

// defaultDNSResolverFactory wires DNSChecker to dns.Tester in
// production. Tests pass their own factory.
func defaultDNSResolverFactory(server, hostname string) DNSResolver {
	return dns.NewTester(server, hostname, dns.DefaultThresholds())
}

// parseDNSParams decodes the params JSON; returns zero value on
// empty / unparseable input.
func parseDNSParams(raw json.RawMessage) DNSParams {
	if len(raw) == 0 {
		return DNSParams{}
	}
	var p DNSParams
	_ = json.Unmarshal(raw, &p)
	return p
}

// coalesce returns the first argument if non-empty, otherwise the
// second.
func coalesce(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}
