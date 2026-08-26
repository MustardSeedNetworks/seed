package config_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestParsePortList(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []int
	}{
		{"a single port", "80", []int{80}},
		{"several ports", "21,23,80", []int{21, 23, 80}},
		{"an inclusive range", "6000-6003", []int{6000, 6001, 6002, 6003}},
		{"singles and a range", "21,80,6000-6002", []int{21, 80, 6000, 6001, 6002}},
		{"a one-port range", "443-443", []int{443}},
		{"whitespace is tolerated", " 21 , 80 , 6000-6001 ", []int{21, 80, 6000, 6001}},

		{"the result is sorted", "80,21,443", []int{21, 80, 443}},
		{"duplicates collapse", "80,80,21", []int{21, 80}},
		{"a range overlapping a single collapses", "80,79-81", []int{79, 80, 81}},

		// A malformed entry drops that entry rather than the whole list: the
		// presets are constants in this package, so a bad one is our typo, and
		// losing one port beats a scan that will not start.
		{"a non-numeric entry is skipped", "21,http,80", []int{21, 80}},
		{"port zero is skipped", "0,80", []int{80}},
		{"an out-of-range port is skipped", "80,65536", []int{80}},
		// "-5" splits on the comma first, so it reaches the range branch as an
		// empty low bound and is dropped; the valid entry beside it survives.
		{"a negative port is skipped, its neighbour is not", "-5,80", []int{80}},
		{"a reversed range is skipped", "80,100-90", []int{80}},
		{"an empty field is skipped", "21,,80", []int{21, 80}},

		{"empty input", "", nil},
		{"only separators", ",,,", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := config.ParsePortList(tc.in)
			if !slices.Equal(got, tc.want) {
				t.Errorf("ParsePortList(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestParsePortListOnTheRealPresets pins that every preset shipped in this
// package expands to something usable. A preset that parses to nothing would
// make its scan silently do nothing at all.
func TestParsePortListOnTheRealPresets(t *testing.T) {
	for name, list := range map[string]string{
		"PortsInsecureTCP": config.PortsInsecureTCP,
		"PortsInsecureUDP": config.PortsInsecureUDP,
		"PortsCommonTCP":   config.PortsCommonTCP,
		"PortsCommonUDP":   config.PortsCommonUDP,
		"PortsSecureTCP":   config.PortsSecureTCP,
		"PortsSecureUDP":   config.PortsSecureUDP,
	} {
		t.Run(name, func(t *testing.T) {
			ports := config.ParsePortList(list)
			if len(ports) == 0 {
				t.Fatalf("%s expands to no ports; a scan using it would do nothing", name)
			}
			for _, port := range ports {
				if port < 1 || port > 65535 {
					t.Errorf("%s yielded port %d, outside 1..65535", name, port)
				}
			}
			if !slices.IsSorted(ports) {
				t.Errorf("%s is not sorted", name)
			}
		})
	}
}

// TestInsecurePresetCoversTheNamedProtocols pins the ports #347 actually asks
// for. The list is allowed to grow; these must not fall out of it.
func TestInsecurePresetCoversTheNamedProtocols(t *testing.T) {
	ports := config.ParsePortList(config.PortsInsecureTCP)
	for _, tc := range []struct {
		port int
		what string
	}{
		{21, "FTP"},
		{23, "Telnet"},
		{80, "HTTP"},
		{514, "rsh/syslog"},
	} {
		if !slices.Contains(ports, tc.port) {
			t.Errorf("the insecure preset omits port %d (%s)", tc.port, tc.what)
		}
	}

	udp := config.ParsePortList(config.PortsInsecureUDP)
	for _, tc := range []struct {
		port int
		what string
	}{
		{69, "TFTP"},
		{161, "SNMPv1/v2c"},
	} {
		if !slices.Contains(udp, tc.port) {
			t.Errorf("the insecure UDP preset omits port %d (%s)", tc.port, tc.what)
		}
	}
}
