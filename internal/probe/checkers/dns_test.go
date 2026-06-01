package checkers_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/diagnostics/dns"
	"github.com/krisarmstrong/seed/internal/probe"
	"github.com/krisarmstrong/seed/internal/probe/checkers"
)

// fakeDNSResolver is the test seam for DNSChecker.
type fakeDNSResolver struct {
	ipv4Result *dns.LookupResult
	ipv6Result *dns.LookupResult
	ipv4Calls  int
	ipv6Calls  int
}

func (f *fakeDNSResolver) ForwardLookupIPv4(_ context.Context, _ string) *dns.LookupResult {
	f.ipv4Calls++
	return f.ipv4Result
}

func (f *fakeDNSResolver) ForwardLookupIPv6(_ context.Context, _ string) *dns.LookupResult {
	f.ipv6Calls++
	return f.ipv6Result
}

func TestDNSChecker_Kind(t *testing.T) {
	t.Parallel()
	c := checkers.NewDNSChecker()
	if c.Kind() != "dns" {
		t.Errorf("Kind() = %q, want %q", c.Kind(), "dns")
	}
}

func TestDNSChecker_RequiredCapabilities(t *testing.T) {
	t.Parallel()
	c := checkers.NewDNSChecker()
	if caps := c.RequiredCapabilities(); len(caps) != 0 {
		t.Errorf("RequiredCapabilities() = %v, want empty", caps)
	}
}

func TestDNSChecker_Run_IPv4Success(t *testing.T) {
	t.Parallel()
	fake := &fakeDNSResolver{
		ipv4Result: &dns.LookupResult{
			Result:   "ok",
			Time:     12 * time.Millisecond,
			TimeMs:   12,
			Status:   "success",
			Resolved: []string{"142.250.80.46"},
		},
	}
	c := checkers.NewDNSChecker().WithDNSResolverFactory(func(_, _ string) checkers.DNSResolver {
		return fake
	})

	p := probe.Probe{
		ID:       "p-1",
		ClientID: "default",
		Kind:     "dns",
		Target:   "google.com",
	}
	r := c.Run(context.Background(), p)

	if !r.Success {
		t.Errorf("Result.Success = false, want true; error=%q", r.Error)
	}
	if r.LatencyMs != 12 {
		t.Errorf("Result.LatencyMs = %v, want 12", r.LatencyMs)
	}
	if fake.ipv4Calls != 1 {
		t.Errorf("ipv4Calls = %d, want 1", fake.ipv4Calls)
	}
	if fake.ipv6Calls != 0 {
		t.Errorf("ipv6Calls = %d, want 0", fake.ipv6Calls)
	}

	// Metadata round-trips through JSON.
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["record_type"] != "A" {
		t.Errorf("metadata.record_type = %v, want %q", meta["record_type"], "A")
	}
}

func TestDNSChecker_Run_IPv6FromParams(t *testing.T) {
	t.Parallel()
	fake := &fakeDNSResolver{
		ipv6Result: &dns.LookupResult{
			Result:   "ok",
			Time:     8 * time.Millisecond,
			TimeMs:   8,
			Status:   "success",
			Resolved: []string{"2607:f8b0:4004:c1b::71"},
		},
	}
	c := checkers.NewDNSChecker().WithDNSResolverFactory(func(_, _ string) checkers.DNSResolver {
		return fake
	})

	p := probe.Probe{
		ID:     "p-1",
		Kind:   "dns",
		Target: "google.com",
		Params: json.RawMessage(`{"record_type":"AAAA"}`),
	}
	r := c.Run(context.Background(), p)

	if !r.Success {
		t.Errorf("Result.Success = false, want true")
	}
	if fake.ipv6Calls != 1 {
		t.Errorf("ipv6Calls = %d, want 1", fake.ipv6Calls)
	}
	if fake.ipv4Calls != 0 {
		t.Errorf("ipv4Calls = %d, want 0", fake.ipv4Calls)
	}
}

func TestDNSChecker_Run_LookupFailure(t *testing.T) {
	t.Parallel()
	fake := &fakeDNSResolver{
		ipv4Result: &dns.LookupResult{
			Status: "error",
			Error:  "NXDOMAIN",
			Time:   0,
			TimeMs: 0,
		},
	}
	c := checkers.NewDNSChecker().WithDNSResolverFactory(func(_, _ string) checkers.DNSResolver {
		return fake
	})

	p := probe.Probe{Kind: "dns", Target: "nonexistent.invalid"}
	r := c.Run(context.Background(), p)

	if r.Success {
		t.Error("Result.Success = true, want false on NXDOMAIN")
	}
	if r.Error != "NXDOMAIN" {
		t.Errorf("Result.Error = %q, want %q", r.Error, "NXDOMAIN")
	}
}

func TestDNSChecker_Run_UnsupportedRecordType(t *testing.T) {
	t.Parallel()
	fake := &fakeDNSResolver{}
	c := checkers.NewDNSChecker().WithDNSResolverFactory(func(_, _ string) checkers.DNSResolver {
		return fake
	})

	p := probe.Probe{
		Kind:   "dns",
		Target: "google.com",
		Params: json.RawMessage(`{"record_type":"CNAME"}`),
	}
	r := c.Run(context.Background(), p)

	if r.Success {
		t.Error("Result.Success = true, want false for unsupported record_type")
	}
	if fake.ipv4Calls != 0 || fake.ipv6Calls != 0 {
		t.Errorf("resolver should not be called for unsupported type; v4=%d v6=%d",
			fake.ipv4Calls, fake.ipv6Calls)
	}
}

func TestDNSChecker_Run_NilResolverResult(t *testing.T) {
	t.Parallel()
	// Defensive path: resolver returns nil. Should surface as
	// Success=false with an Error, not panic.
	fake := &fakeDNSResolver{ipv4Result: nil}
	c := checkers.NewDNSChecker().WithDNSResolverFactory(func(_, _ string) checkers.DNSResolver {
		return fake
	})

	p := probe.Probe{Kind: "dns", Target: "google.com"}
	r := c.Run(context.Background(), p)

	if r.Success {
		t.Error("Result.Success = true, want false")
	}
	if r.Error == "" {
		t.Error("Result.Error should describe nil resolver result")
	}
}

func TestDNSChecker_Run_ServerParamPassedToFactory(t *testing.T) {
	t.Parallel()
	var capturedServer string
	fake := &fakeDNSResolver{
		ipv4Result: &dns.LookupResult{Status: "success"},
	}
	c := checkers.NewDNSChecker().WithDNSResolverFactory(func(server, _ string) checkers.DNSResolver {
		capturedServer = server
		return fake
	})

	p := probe.Probe{
		Kind:   "dns",
		Target: "google.com",
		Params: json.RawMessage(`{"server":"8.8.8.8:53"}`),
	}
	_ = c.Run(context.Background(), p)

	if capturedServer != "8.8.8.8:53" {
		t.Errorf("factory got server=%q, want %q", capturedServer, "8.8.8.8:53")
	}
}
