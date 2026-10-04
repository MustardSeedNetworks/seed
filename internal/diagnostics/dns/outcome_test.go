// SPDX-License-Identifier: BUSL-1.1

package dns_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"slices"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

// stubHost has an A record, 192.0.2.1, and nothing else; that address's PTR
// names it. Every other question gets an empty answer.
const stubHost = "host.test."

// serveStubDNS answers queries on a loopback UDP socket until the test ends.
func serveStubDNS(t *testing.T) string {
	t.Helper()
	conn, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ptr := dnsmessage.MustNewName("1.2.0.192.in-addr.arpa.")
	go func() {
		buf := make([]byte, 512)
		for {
			n, from, readErr := conn.ReadFrom(buf)
			if readErr != nil {
				return
			}
			var query dnsmessage.Message
			if query.Unpack(buf[:n]) != nil || len(query.Questions) != 1 {
				continue
			}
			q := query.Questions[0]
			reply := dnsmessage.Message{
				ID:            query.ID,
				Response:      true,
				Authoritative: true,
				Questions:     query.Questions,
			}
			hdr := dnsmessage.ResourceHeader{Name: q.Name, Type: q.Type, Class: q.Class, TTL: 60}
			switch {
			case q.Name.String() == stubHost && q.Type == dnsmessage.TypeA:
				reply.Answers = []dnsmessage.Resource{
					{Header: hdr, Body: &dnsmessage.AResource{A: [4]byte{192, 0, 2, 1}}},
				}
			case q.Name == ptr && q.Type == dnsmessage.TypePTR:
				reply.Answers = []dnsmessage.Resource{{
					Header: hdr,
					Body:   &dnsmessage.PTRResource{PTR: dnsmessage.MustNewName(stubHost)},
				}}
			}
			packed, packErr := reply.Pack()
			if packErr != nil {
				continue
			}
			_, _ = conn.WriteTo(packed, from)
		}
	}()
	return conn.LocalAddr().String()
}

func resolverAt(addr string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "udp", addr)
		},
	}
}

// unreachableResolver fails every query before it is sent.
func unreachableResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("unreachable")
		},
	}
}

// A lookup reports what it came back with as a code the UI words, never as
// English prose (#2844).
func TestLookupOutcome(t *testing.T) {
	stub := resolverAt(serveStubDNS(t))
	tests := []struct {
		name     string
		resolver *net.Resolver
		lookup   func(context.Context, *dns.Tester) *dns.LookupResult
		want     dns.Outcome
		status   dns.Status
		resolved []string
	}{
		{
			name:     "A record",
			resolver: stub,
			lookup:   func(ctx context.Context, d *dns.Tester) *dns.LookupResult { return d.ForwardLookupIPv4(ctx, stubHost) },
			want:     dns.OutcomeResolved,
			status:   dns.StatusSuccess,
			resolved: []string{"192.0.2.1"},
		},
		{
			name:     "no AAAA record",
			resolver: stub,
			lookup:   func(ctx context.Context, d *dns.Tester) *dns.LookupResult { return d.ForwardLookupIPv6(ctx, stubHost) },
			want:     dns.OutcomeNoRecord,
			status:   dns.StatusWarning,
		},
		{
			name:     "no A record from an unreachable resolver",
			resolver: unreachableResolver(),
			lookup:   func(ctx context.Context, d *dns.Tester) *dns.LookupResult { return d.ForwardLookupIPv4(ctx, stubHost) },
			want:     dns.OutcomeNoRecord,
			status:   dns.StatusWarning,
		},
		{
			name:     "host lookup",
			resolver: stub,
			lookup:   func(ctx context.Context, d *dns.Tester) *dns.LookupResult { return d.ForwardLookup(ctx, stubHost) },
			want:     dns.OutcomeResolved,
			status:   dns.StatusSuccess,
			resolved: []string{"192.0.2.1"},
		},
		{
			name:     "host lookup failed",
			resolver: unreachableResolver(),
			lookup:   func(ctx context.Context, d *dns.Tester) *dns.LookupResult { return d.ForwardLookup(ctx, stubHost) },
			want:     dns.OutcomeFailed,
			status:   dns.StatusError,
		},
		{
			name:     "PTR record",
			resolver: stub,
			lookup:   func(ctx context.Context, d *dns.Tester) *dns.LookupResult { return d.ReverseLookup(ctx, "192.0.2.1") },
			want:     dns.OutcomeResolved,
			status:   dns.StatusSuccess,
			resolved: []string{stubHost},
		},
		{
			name:     "reverse lookup failed",
			resolver: stub,
			lookup:   func(ctx context.Context, d *dns.Tester) *dns.LookupResult { return d.ReverseLookup(ctx, "192.0.2.2") },
			want:     dns.OutcomeFailed,
			status:   dns.StatusError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := dns.NewTester("", stubHost, dns.Thresholds{Warning: time.Minute, Critical: time.Hour})
			dns.ExportSetResolver(tester, tt.resolver)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			got := tt.lookup(ctx, tester)
			if got.Outcome != tt.want || got.Status != tt.status || !slices.Equal(got.Resolved, tt.resolved) {
				t.Errorf("got outcome %q status %q resolved %v, want %q %q %v",
					got.Outcome, got.Status, got.Resolved, tt.want, tt.status, tt.resolved)
			}

			wire, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var fields map[string]any
			if unmarshalErr := json.Unmarshal(wire, &fields); unmarshalErr != nil {
				t.Fatalf("unmarshal: %v", unmarshalErr)
			}
			if fields["outcome"] != string(tt.want) {
				t.Errorf("wire outcome = %v, want %q", fields["outcome"], tt.want)
			}
			if _, ok := fields["result"]; ok {
				t.Errorf("wire still carries a result field: %s", wire)
			}
		})
	}
}
