// SPDX-License-Identifier: BUSL-1.1

package dns_test

import (
	"context"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

// The three states the card has to tell apart.
func TestResolversForInterface(t *testing.T) {
	system := []string{"1.1.1.1"}
	cases := []struct {
		name      string
		iface     string
		scoped    []string
		canScope  bool
		want      []string
		wantScope dns.Scope
	}{
		{
			name:      "interface with resolvers of its own",
			iface:     "feth0",
			scoped:    []string{"10.0.0.1"},
			canScope:  true,
			want:      []string{"10.0.0.1"},
			wantScope: dns.ScopeInterface,
		},
		{
			name:      "interface with none of its own is an honest absence",
			iface:     "feth0",
			canScope:  true,
			want:      nil,
			wantScope: dns.ScopeInterface,
		},
		{
			name:      "host that cannot attribute resolvers says so",
			iface:     "feth0",
			canScope:  false,
			want:      system,
			wantScope: dns.ScopeSystem,
		},
		{
			name:      "no interface selected",
			iface:     "",
			canScope:  true,
			scoped:    []string{"10.0.0.1"},
			want:      system,
			wantScope: dns.ScopeSystem,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := dns.ExportResolverSource(
				func() []string { return system },
				func(string) ([]string, bool) { return tc.scoped, tc.canScope },
			)
			got, scope := dns.ExportResolversFor(src, tc.iface)
			if !slices.Equal(got, tc.want) {
				t.Errorf("servers = %v, want %v", got, tc.want)
			}
			if scope != tc.wantScope {
				t.Errorf("scope = %q, want %q", scope, tc.wantScope)
			}
		})
	}
}

func TestTestReportsTheInterfaceScope(t *testing.T) {
	tester := dns.NewTesterForInterface("", "localhost", dns.DefaultThresholds(), "feth0")
	dns.ExportSetResolverSource(tester, dns.ExportResolverSource(
		func() []string { return []string{"1.1.1.1"} },
		func(string) ([]string, bool) { return []string{"127.0.0.1"}, true },
	))

	result := tester.Test(context.Background())
	if !slices.Equal(result.Servers, []string{"127.0.0.1"}) {
		t.Errorf("servers = %v, want the interface's own [127.0.0.1]", result.Servers)
	}
	if result.ServerScope != dns.ScopeInterface {
		t.Errorf("serverScope = %q, want %q", result.ServerScope, dns.ScopeInterface)
	}
}

// An interface with no resolvers of its own must not be measured through
// another link's: reporting a success the selected interface did not earn is
// the shape of the defect this row exists to remove.
func TestTestDoesNotLookUpThroughAnotherLink(t *testing.T) {
	tester := dns.NewTesterForInterface("", "localhost", dns.DefaultThresholds(), "feth0")
	dns.ExportSetResolverSource(tester, dns.ExportResolverSource(
		func() []string { return []string{"1.1.1.1"} },
		func(string) ([]string, bool) { return nil, true },
	))

	result := tester.Test(context.Background())
	if len(result.Servers) != 0 {
		t.Errorf("servers = %v, want none", result.Servers)
	}
	if result.ServerScope != dns.ScopeInterface {
		t.Errorf("serverScope = %q, want %q", result.ServerScope, dns.ScopeInterface)
	}
	if result.Forward != nil {
		t.Errorf("forward = %+v, want no lookup at all", result.Forward)
	}
	if len(result.PerServerResults) != 0 {
		t.Errorf("perServerResults = %v, want none", result.PerServerResults)
	}
}

func TestSetInterfaceRescopes(t *testing.T) {
	tester := dns.NewTester("", "localhost", dns.DefaultThresholds())
	if got := tester.GetInterface(); got != "" {
		t.Fatalf("interface = %q, want empty on a system-wide tester", got)
	}
	tester.SetInterface("feth0")
	if got := tester.GetInterface(); got != "feth0" {
		t.Errorf("interface = %q, want feth0", got)
	}
}

// A server the operator configured is explicit and is still offered, but it
// does not turn an interface-scoped answer into a system-wide one.
func TestConfiguredServersJoinTheInterfaceList(t *testing.T) {
	tester := dns.NewTesterForInterface("", "localhost", dns.DefaultThresholds(), "feth0")
	dns.ExportSetResolverSource(tester, dns.ExportResolverSource(
		func() []string { return []string{"1.1.1.1"} },
		func(string) ([]string, bool) { return []string{"127.0.0.1"}, true },
	))
	tester.SetConfiguredServers([]dns.ConfiguredServer{{Address: "127.0.0.2", Enabled: true}})

	result := tester.Test(context.Background())
	if !slices.Equal(result.Servers, []string{"127.0.0.1", "127.0.0.2"}) {
		t.Errorf("servers = %v, want [127.0.0.1 127.0.0.2]", result.Servers)
	}
	if result.ServerScope != dns.ScopeInterface {
		t.Errorf("serverScope = %q, want %q", result.ServerScope, dns.ScopeInterface)
	}
}
