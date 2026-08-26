package resolve

import (
	"net"
	"strings"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

// packResponse builds a real mDNS response so the parsers are exercised against
// bytes rather than against a hand-rolled struct. Anything these tests accept, a
// responder on the wire could actually send.
func packResponse(t *testing.T, answers, additionals []dnsmessage.Resource) []byte {
	t.Helper()
	// Response and Authoritative are promoted from the embedded Header.
	msg := dnsmessage.Message{
		Response:      true,
		Authoritative: true,
		Answers:       answers,
		Additionals:   additionals,
	}
	data, err := msg.Pack()
	if err != nil {
		t.Fatalf("pack response: %v", err)
	}
	return data
}

func ptrResource(t *testing.T, owner, target string) dnsmessage.Resource {
	t.Helper()
	ownerName, err := dnsmessage.NewName(owner)
	if err != nil {
		t.Fatalf("owner name %q: %v", owner, err)
	}
	targetName, err := dnsmessage.NewName(target)
	if err != nil {
		t.Fatalf("target name %q: %v", target, err)
	}
	return dnsmessage.Resource{
		Header: dnsmessage.ResourceHeader{
			Name: ownerName, Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET,
		},
		Body: &dnsmessage.PTRResource{PTR: targetName},
	}
}

func aResource(t *testing.T, owner string, ip [4]byte) dnsmessage.Resource {
	t.Helper()
	ownerName, err := dnsmessage.NewName(owner)
	if err != nil {
		t.Fatalf("owner name %q: %v", owner, err)
	}
	return dnsmessage.Resource{
		Header: dnsmessage.ResourceHeader{
			Name: ownerName, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET,
		},
		Body: &dnsmessage.AResource{A: ip},
	}
}

func TestBuildMDNSQuery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		query   string
		qtype   dnsmessage.Type
		wantErr bool
	}{
		{"reverse lookup", "1.0.168.192.in-addr.arpa.", dnsmessage.TypePTR, false},
		{"forward lookup", "printer.local.", dnsmessage.TypeA, false},
		{"unterminated name is rejected", "printer.local", dnsmessage.TypeA, true},
		{"empty name is rejected", "", dnsmessage.TypeA, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := buildMDNSQuery(tc.query, tc.qtype)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("buildMDNSQuery(%q) succeeded, want an error", tc.query)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildMDNSQuery(%q): %v", tc.query, err)
			}

			assertQueryRoundTrips(t, data, tc.query, tc.qtype)
		})
	}
}

// assertQueryRoundTrips unpacks a query we packed. One the standard parser
// cannot read is one no responder will answer.
func assertQueryRoundTrips(t *testing.T, data []byte, query string, qtype dnsmessage.Type) {
	t.Helper()

	var msg dnsmessage.Message
	if err := msg.Unpack(data); err != nil {
		t.Fatalf("the query we built does not parse: %v", err)
	}
	if msg.ID != 0 {
		t.Errorf("ID = %d, want 0 — mDNS queries use 0", msg.ID)
	}
	if msg.RecursionDesired {
		t.Error("RecursionDesired is set; mDNS queries are not recursive")
	}
	if len(msg.Questions) != 1 {
		t.Fatalf("got %d questions, want 1", len(msg.Questions))
	}
	if got := msg.Questions[0].Type; got != qtype {
		t.Errorf("question type = %v, want %v", got, qtype)
	}
	if got := msg.Questions[0].Name.String(); got != query {
		t.Errorf("question name = %q, want %q", got, query)
	}
}

func TestParseMDNSResponse(t *testing.T) {
	ptr := ptrResource(t, "1.0.168.192.in-addr.arpa.", "printer.local.")
	a := aResource(t, "printer.local.", [4]byte{192, 168, 0, 1})

	for _, tc := range []struct {
		name    string
		data    []byte
		want    string
		wantErr bool
	}{
		{
			name: "PTR in the answer section",
			data: packResponse(t, []dnsmessage.Resource{ptr}, nil),
			want: "printer.local",
		},
		{
			// Real responders often put the useful record here, so a parser that
			// only reads Answers silently resolves nothing.
			name: "PTR in the additional section",
			data: packResponse(t, nil, []dnsmessage.Resource{ptr}),
			want: "printer.local",
		},
		{
			name:    "a response carrying only an A record",
			data:    packResponse(t, []dnsmessage.Resource{a}, nil),
			wantErr: true,
		},
		{
			name:    "an empty response",
			data:    packResponse(t, nil, nil),
			wantErr: true,
		},
		{
			name:    "bytes that are not a DNS message",
			data:    []byte{0xff, 0xff, 0xff},
			wantErr: true,
		},
		{
			name:    "no bytes at all",
			data:    nil,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMDNSResponse(tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseMDNSResponse returned %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseMDNSResponse: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if strings.HasSuffix(got, ".") {
				t.Errorf("got %q, which keeps its trailing dot", got)
			}
		})
	}
}

func TestParseMDNSResponseForIP(t *testing.T) {
	a := aResource(t, "printer.local.", [4]byte{192, 168, 0, 1})
	ptr := ptrResource(t, "1.0.168.192.in-addr.arpa.", "printer.local.")

	for _, tc := range []struct {
		name    string
		data    []byte
		want    string
		wantErr bool
	}{
		{
			name: "A in the answer section",
			data: packResponse(t, []dnsmessage.Resource{a}, nil),
			want: "192.168.0.1",
		},
		{
			name: "A in the additional section",
			data: packResponse(t, nil, []dnsmessage.Resource{a}),
			want: "192.168.0.1",
		},
		{
			name:    "a response carrying only a PTR record",
			data:    packResponse(t, []dnsmessage.Resource{ptr}, nil),
			wantErr: true,
		},
		{
			name:    "bytes that are not a DNS message",
			data:    []byte{0x00},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMDNSResponseForIP(tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseMDNSResponseForIP returned %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseMDNSResponseForIP: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if net.ParseIP(got) == nil {
				t.Errorf("got %q, which is not a parseable IP", got)
			}
		})
	}
}

// TestMDNSResolverCache covers the cache accessors, which are the part of the
// resolver reachable without a network.
func TestMDNSResolverCache(t *testing.T) {
	r := NewMDNSResolver("en0")

	if name, ok := r.GetCached("192.168.0.1"); ok {
		t.Errorf("a fresh resolver returned %q for an unseen IP", name)
	}

	r.SetName("192.168.0.1", "printer.local")
	name, ok := r.GetCached("192.168.0.1")
	if !ok {
		t.Fatal("SetName stored nothing that GetCached can see")
	}
	if name != "printer.local" {
		t.Errorf("GetCached = %q, want %q", name, "printer.local")
	}

	r.ClearCache()
	if leftover, stillThere := r.GetCached("192.168.0.1"); stillThere {
		t.Errorf("ClearCache left %q behind", leftover)
	}
}

// TestMDNSListenerProcessesAnswers drives the passive listener through
// processAnswer, the entry point the listen loop actually calls, rather than
// through storeName. That covers the record handlers and, more usefully, pins
// the precedence between them: an A record names the address it carries and
// overwrites, a PTR names the sender and only fills a gap.
func TestMDNSListenerProcessesAnswers(t *testing.T) {
	t.Run("an A record names the address it carries", func(t *testing.T) {
		l := NewMDNSListener("en0")
		answer := aResource(t, "camera.local.", [4]byte{192, 168, 0, 5})
		l.processAnswer(&answer, "192.168.0.99")

		got, ok := l.GetName("192.168.0.5")
		if !ok {
			t.Fatal("no name stored for the A record's own address")
		}
		if got != "camera.local" {
			t.Errorf("got %q, want %q — the trailing dot should be gone", got, "camera.local")
		}
		if _, wrong := l.GetName("192.168.0.99"); wrong {
			t.Error("the A record was filed under the sender's address")
		}
	})

	t.Run("a PTR record names the sender", func(t *testing.T) {
		l := NewMDNSListener("en0")
		answer := ptrResource(t, "5.0.168.192.in-addr.arpa.", "printer.local.")
		l.processAnswer(&answer, "192.168.0.5")

		got, ok := l.GetName("192.168.0.5")
		if !ok {
			t.Fatal("no name stored for the sender")
		}
		if got != "printer.local" {
			t.Errorf("got %q, want %q", got, "printer.local")
		}
	})

	t.Run("names outside .local are ignored", func(t *testing.T) {
		l := NewMDNSListener("en0")
		answer := aResource(t, "evil.example.com.", [4]byte{192, 168, 0, 7})
		l.processAnswer(&answer, "192.168.0.7")

		if got, ok := l.GetName("192.168.0.7"); ok {
			t.Errorf("stored %q; mDNS should only accept the .local namespace", got)
		}
	})

	t.Run("an A record overwrites, a PTR does not", func(t *testing.T) {
		l := NewMDNSListener("en0")

		first := aResource(t, "first.local.", [4]byte{192, 168, 0, 8})
		l.processAnswer(&first, "192.168.0.8")

		// A PTR must not displace a name an A record established.
		ptr := ptrResource(t, "8.0.168.192.in-addr.arpa.", "second.local.")
		l.processAnswer(&ptr, "192.168.0.8")
		if got, _ := l.GetName("192.168.0.8"); got != "first.local" {
			t.Errorf("after the PTR the name is %q, want %q — PTR should only "+
				"fill a gap", got, "first.local")
		}

		// A second A record is authoritative for its own address and replaces.
		second := aResource(t, "third.local.", [4]byte{192, 168, 0, 8})
		l.processAnswer(&second, "192.168.0.8")
		if got, _ := l.GetName("192.168.0.8"); got != "third.local" {
			t.Errorf("after the second A record the name is %q, want %q", got, "third.local")
		}
	})

	t.Run("GetNames hands out a copy", func(t *testing.T) {
		l := NewMDNSListener("en0")
		answer := aResource(t, "copy.local.", [4]byte{192, 168, 0, 9})
		l.processAnswer(&answer, "192.168.0.9")

		names := l.GetNames()
		names["192.168.0.9"] = "caller-mutated"
		if got, _ := l.GetName("192.168.0.9"); got == "caller-mutated" {
			t.Error("GetNames handed out the listener's live map, so a caller " +
				"can corrupt state the listen loop is writing")
		}
	})
}
