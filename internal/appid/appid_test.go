package appid_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/appid"
)

func TestBuiltinClassifiesKnownFlows(t *testing.T) {
	t.Parallel()
	table := appid.Builtin()
	tests := []struct {
		name     string
		protocol uint8
		src, dst uint16
		want     string
	}{
		{"https request", 6, 51514, 443, "https"},
		{"https response", 6, 443, 51514, "https"},
		{"quic is udp 443", 17, 50000, 443, "quic"},
		{"dns over udp", 17, 40000, 53, "dns"},
		{"dns over tcp shares the name", 6, 40000, 53, "dns"},
		{"dhcp both ports in the range", 17, 68, 67, "dhcp"},
		{"ssh", 6, 60000, 22, "ssh"},
		{"range interior", 6, 50000, 5902, "vnc"},
		{"icmp has no ports", 1, 0, 0, "icmp"},
		{"icmpv6", 58, 0, 0, "icmpv6"},
		{"lower port wins when both match", 6, 3306, 443, "https"},
		{"lower port wins the other way round", 6, 443, 3306, "https"},
		{"server port is the higher one", 6, 40000, 3306, "mysql"},
		{"unlisted tcp port", 6, 51514, 9999, appid.Unknown},
		{"both ephemeral", 6, 50001, 50002, appid.Unknown},
		{"udp port that is only a tcp signature", 17, 50000, 22, appid.Unknown},
		{"tcp port that is only a udp signature", 6, 50000, 123, appid.Unknown},
		{"port zero is never a service", 6, 0, 0, appid.Unknown},
		{"unlisted protocol", 132, 2905, 2905, appid.Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := table.Classify(tt.protocol, tt.src, tt.dst); got != tt.want {
				t.Errorf("Classify(%d, %d, %d) = %q, want %q", tt.protocol, tt.src, tt.dst, got, tt.want)
			}
		})
	}
}

func TestNewRejectsAmbiguousOrMalformedTables(t *testing.T) {
	t.Parallel()
	tcp := func(name string, ports ...string) appid.Signature {
		return appid.Signature{Name: name, Protocol: 6, Ports: ports}
	}
	tests := []struct {
		name    string
		sigs    []appid.Signature
		wantErr string
	}{
		{"empty", nil, "empty"},
		{"uppercase name", []appid.Signature{tcp("HTTPS", "443")}, "lowercase"},
		{"reserved name", []appid.Signature{tcp("unknown", "443")}, "reserved"},
		{"protocol zero", []appid.Signature{{Name: "x", Protocol: 0}}, "1 to 255"},
		{"tcp without ports", []appid.Signature{tcp("web")}, "at least one port"},
		{"port zero", []appid.Signature{tcp("web", "0")}, "1 to 65535"},
		{"port too large", []appid.Signature{tcp("web", "65536")}, "1 to 65535"},
		{"port not a number", []appid.Signature{tcp("web", "http")}, "1 to 65535"},
		{"backwards range", []appid.Signature{tcp("web", "90-80")}, "backwards"},
		{"same port twice", []appid.Signature{tcp("a", "443"), tcp("b", "443")}, "signature 0 and signature 1"},
		{"overlapping ranges", []appid.Signature{tcp("a", "8000-8010"), tcp("b", "8010-8020")}, "port 8010"},
		{
			"ports on a portless protocol",
			[]appid.Signature{{Name: "icmp", Protocol: 1, Ports: []string{"1"}}},
			"no ports",
		},
		{"portless protocol twice", []appid.Signature{{Name: "a", Protocol: 47}, {Name: "b", Protocol: 47}}, "already"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := appid.New(tt.sigs)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("New() error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestOperatorTableReplacesBuiltin(t *testing.T) {
	t.Parallel()
	table, err := appid.Parse([]byte(`{"signatures":[
		{"name":"historian","description":"Plant historian","protocol":6,"ports":["5450","5460-5469"]}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := table.Classify(6, 50000, 5465); got != "historian" {
		t.Errorf("custom range: got %q", got)
	}
	if got := table.Classify(6, 50000, 443); got != appid.Unknown {
		t.Errorf("a replaced table must not fall back to the builtin: got %q", got)
	}
}

func TestTableRoundTripsThroughJSON(t *testing.T) {
	t.Parallel()
	want := appid.Builtin()
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := appid.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Signatures()) != len(want.Signatures()) {
		t.Fatalf("round trip kept %d of %d signatures", len(got.Signatures()), len(want.Signatures()))
	}
}

func TestParseRejectsUnknownShape(t *testing.T) {
	t.Parallel()
	if _, err := appid.Parse([]byte(`{"signatures":[{"name":"x","protocol":300}]}`)); err == nil {
		t.Fatal("protocol 300 does not fit a byte and must be rejected")
	}
	if _, err := appid.Parse([]byte(`not json`)); err == nil {
		t.Fatal("malformed JSON must be rejected")
	}
}
