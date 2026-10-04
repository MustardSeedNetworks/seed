package indicators_test

import (
	"encoding/json"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/indicators"
)

func TestNewRefuses(t *testing.T) {
	t.Parallel()
	tooMany := make([]string, 10001)
	for i := range tooMany {
		tooMany[i] = "198.51." + strconv.Itoa(i/256) + "." + strconv.Itoa(i%256)
	}
	cases := []struct {
		name    string
		entries []string
		want    string
	}{
		{"not an address", []string{"198.51.100.7", "evil.example"}, "entry 1: must be an IPv4 or IPv6 address"},
		{"zoned address", []string{"2001:db8::1%eth0"}, "entry 0: must be an IPv4 or IPv6 address"},
		{"host bits set", []string{"203.0.113.9/24"}, "entry 0: prefix has host bits set"},
		{"mapped IPv4", []string{"::ffff:198.51.100.7"}, "entry 0: must be written as IPv4"},
		{"private", []string{"10.1.2.3"}, "entry 0: reaches into a private"},
		{"CGNAT", []string{"100.64.0.1"}, "entry 0: reaches into"},
		{"loopback v6", []string{"::1"}, "entry 0: reaches into"},
		{"unique local", []string{"fd00::/8"}, "entry 0: reaches into"},
		{"multicast", []string{"239.1.1.1"}, "entry 0: reaches into"},
		{"broadcast", []string{"255.255.255.255"}, "entry 0: reaches into"},
		{"prefix covering private space", []string{"0.0.0.0/0"}, "entry 0: reaches into"},
		{"duplicate", []string{"198.51.100.7", "198.51.100.7/32"}, "entry 1 repeats an earlier entry"},
		{"too many", tooMany, "list has 10001 entries, limit 10000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := indicators.New(tc.entries)
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("error = %v, want prefix %q", err, tc.want)
			}
			for _, e := range tc.entries[:min(len(tc.entries), 2)] {
				if strings.Contains(err.Error(), e) {
					t.Errorf("error quotes the entry %q", e)
				}
			}
		})
	}
}

func TestMatch(t *testing.T) {
	t.Parallel()
	list, err := indicators.New([]string{
		"198.51.100.7", "198.51.0.0/16", "203.0.113.0/24", "2001:db8:bad::/48", "2001:db8:bad:1::5",
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		addr string
		want string // "" means no match
	}{
		{"198.51.100.7", "198.51.100.7/32"},
		{"198.51.100.8", "198.51.0.0/16"},
		{"203.0.113.250", "203.0.113.0/24"},
		{"::ffff:203.0.113.1", "203.0.113.0/24"},
		{"203.0.114.1", ""},
		{"2001:db8:bad:1::5", "2001:db8:bad:1::5/128"},
		{"2001:db8:bad:2::1", "2001:db8:bad::/48"},
		{"2001:db8:bae::1", ""},
		{"10.0.0.5", ""},
		{"fe80::1", ""},
	}
	for _, tc := range cases {
		got, ok := list.Match(netip.MustParseAddr(tc.addr))
		if tc.want == "" {
			if ok {
				t.Errorf("Match(%s) = %s, want no match", tc.addr, got)
			}
			continue
		}
		if !ok || got.String() != tc.want {
			t.Errorf("Match(%s) = %s, %v; want %s", tc.addr, got, ok, tc.want)
		}
	}
	if _, ok := list.Match(netip.Addr{}); ok {
		t.Error("the zero address matched")
	}
}

func TestEmptyListMatchesNothing(t *testing.T) {
	t.Parallel()
	list, err := indicators.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := list.Match(netip.MustParseAddr("198.51.100.7")); ok {
		t.Error("an empty list matched")
	}
	if got := list.Entries(); got == nil || len(got) != 0 {
		t.Errorf("Entries() = %#v, want an empty slice", got)
	}
}

func TestJSONRoundTripIsCanonical(t *testing.T) {
	t.Parallel()
	list, err := indicators.New([]string{"198.51.100.7/32", "2001:DB8:BAD::/48", "203.0.113.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"indicators":["198.51.100.7","2001:db8:bad::/48","203.0.113.0/24"]}`; string(data) != want {
		t.Errorf("marshal = %s, want %s", data, want)
	}
	back, err := indicators.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(back.Entries(), list.Entries()) || back.Len() != 3 {
		t.Errorf("round trip = %v, want %v", back.Entries(), list.Entries())
	}
	if _, parseErr := indicators.Parse([]byte(`{"indicators":"198.51.100.7"}`)); parseErr == nil {
		t.Error("a malformed document parsed")
	}
}

// The refused ranges end exactly where internal space ends: public entries
// right beside them are accepted and match, and the internal addresses next
// door do not.
func TestEntriesBesideInternalSpace(t *testing.T) {
	t.Parallel()
	list, err := indicators.New([]string{"8.0.0.0/7", "172.32.0.0/11", "100.128.0.0/9", "2001:4860::/32"})
	if err != nil {
		t.Fatal(err)
	}
	for addr, want := range map[string]bool{
		"10.9.9.9": false, "172.31.0.1": false, "192.168.0.1": false, "100.127.255.255": false,
		"127.0.0.1": false, "169.254.1.1": false, "224.0.0.251": false, "0.0.0.0": false,
		"::": false, "::1": false, "fe80::1": false, "fd12::1": false, "ff02::fb": false,
		"::ffff:10.0.0.1": false,
		"8.8.8.8":         true, "172.32.0.1": true, "100.128.0.1": true, "2001:4860::8888": true,
	} {
		if _, got := list.Match(netip.MustParseAddr(addr)); got != want {
			t.Errorf("Match(%s) = %v, want %v", addr, got, want)
		}
	}
}
