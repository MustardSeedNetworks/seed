package vuln

import (
	"strings"
	"testing"
)

// The cases below are the examples in each function's own docstring, which is
// the closest thing to a specification this package has. Where the code did not
// match its documented example, the code was wrong — this package feeds CPE
// lookups, so a wrong product or a truncated version means the CVE search is
// run against the wrong thing and finds nothing.

func TestParseCDPVersion(t *testing.T) {
	for _, tc := range []struct {
		name        string
		platform    string
		version     string
		wantProduct string
		wantVersion string
	}{
		{
			name:        "IOS named in the software version",
			platform:    "cisco WS-C2960-24TT-L",
			version:     "Cisco IOS Software, C2960 Software, Version 12.2(58)SE2",
			wantProduct: "cisco-ios",
			wantVersion: "12.2(58)SE2",
		},
		{
			// XE must win over the generic IOS branch, since both strings
			// contain "IOS".
			name:        "IOS-XE is distinguished from IOS",
			platform:    "cisco ISR4331",
			version:     "Cisco IOS XE Software, Version 16.9.4",
			wantProduct: "cisco-ios-xe",
			wantVersion: "16.9.4",
		},
		{
			name:        "NX-OS is distinguished from IOS",
			platform:    "cisco Nexus9000",
			version:     "Cisco NX-OS Software IOS, Version 9.3.5",
			wantProduct: "cisco-nx-os",
			wantVersion: "9.3.5",
		},
		{
			name:        "a Cisco device with no OS named stays generic",
			platform:    "Cisco 2960",
			version:     "15.2(4)E",
			wantProduct: "cisco-device",
			wantVersion: "15.2(4)E",
		},
		{
			name:     "an empty version yields nothing",
			platform: "cisco WS-C2960",
			version:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product, version := parseCDPVersion(tc.platform, tc.version)
			if product != tc.wantProduct || version != tc.wantVersion {
				t.Errorf("got (%q, %q), want (%q, %q)",
					product, version, tc.wantProduct, tc.wantVersion)
			}
		})
	}
}

func TestParseEDPVersion(t *testing.T) {
	for _, tc := range []struct {
		name, in, wantProduct, wantVersion string
	}{
		{"the documented example", "ExtremeXOS 22.5.1.7", "extremexos", "22.5.1.7"},
		{"an unparseable string falls back", "no version here", "extreme-device", "no version here"},
		{"empty input yields nothing", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product, version := parseEDPVersion(tc.in)
			if product != tc.wantProduct || version != tc.wantVersion {
				t.Errorf("got (%q, %q), want (%q, %q)",
					product, version, tc.wantProduct, tc.wantVersion)
			}
		})
	}
}

func TestParseSNMPSysDescr(t *testing.T) {
	for _, tc := range []struct {
		name, in, wantProduct, wantVersion string
	}{
		{
			name:        "Cisco IOS",
			in:          "Cisco IOS Software, Version 15.2(4)M",
			wantProduct: "cisco-ios",
			wantVersion: "15.2(4)m",
		},
		{
			// Regression: (\d+\w*) backtracked past the hyphen and captured
			// "48g", the port count, as the model number.
			name:        "an HP switch names its model, not its port count",
			in:          "HP J9729A 2920-48G Switch, revision WB.16.04.0008",
			wantProduct: "hp-2920",
			wantVersion: "wb.16.04.0008",
		},
		{
			// Regression: the class held an uppercase X while the input is
			// lowercased first, so the version was cut at "12.1".
			name:        "a JunOS version survives in full",
			in:          "Juniper Networks, Inc. srx240h2 internet router, kernel JUNOS 12.1X47-D15.4",
			wantProduct: "juniper-junos",
			wantVersion: "12.1x47-d15.4",
		},
		{
			name:        "an Aruba switch uses the HP matcher",
			in:          "Aruba J9773A 2530-24G Switch, revision YA.16.10.0003",
			wantProduct: "hp-2530",
			wantVersion: "ya.16.10.0003",
		},
		{
			name:        "an unmatched vendor falls through to the generic path",
			in:          "Arista Networks EOS version 4.24.2F",
			wantProduct: "arista",
			wantVersion: "4.24.2f",
		},
		{"empty input yields nothing", "", "", ""},
		{"an unrecognised device yields nothing", "Some Unknown Appliance", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product, version := parseSNMPSysDescr(tc.in)
			if product != tc.wantProduct || version != tc.wantVersion {
				t.Errorf("got (%q, %q), want (%q, %q)",
					product, version, tc.wantProduct, tc.wantVersion)
			}
		})
	}
}

func TestParseLLDPDescription(t *testing.T) {
	for _, tc := range []struct {
		name, in, wantProduct, wantVersion string
	}{
		{
			// Regression: same backtracking bug as the HP SNMP matcher.
			name:        "a ProCurve switch names its model, not its port count",
			in:          "ProCurve J9850A Switch 2910al-48G, revision WB.16.04",
			wantProduct: "procurve-2910",
			wantVersion: "wb.16.04",
		},
		{
			// Regression: uppercase R in the class against lowercased input.
			name:        "a JunOS version survives in full",
			in:          "Juniper Networks, Inc. ex2200-48t-4g Ethernet Switch, kernel JUNOS 12.3R12.4",
			wantProduct: "junos",
			wantVersion: "12.3r12.4",
		},
		{
			name:        "a generic device falls back to the version pattern",
			in:          "Generic Switch, Version 1.2.3",
			wantProduct: "network-device",
			wantVersion: "1.2.3",
		},
		{"empty input yields nothing", "", "", ""},
		{"an undescribed device yields nothing", "just some text", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product, version := parseLLDPDescription(tc.in)
			if product != tc.wantProduct || version != tc.wantVersion {
				t.Errorf("got (%q, %q), want (%q, %q)",
					product, version, tc.wantProduct, tc.wantVersion)
			}
		})
	}
}

func TestParseServiceBanner(t *testing.T) {
	for _, tc := range []struct {
		name, in, wantProduct, wantVersion string
	}{
		{"the documented OpenSSH example", "OpenSSH_8.9p1 Ubuntu-3ubuntu0.1", "openssh", "8.9p1"},
		{"a slash separator", "ProFTPD/1.3.5", "proftpd", "1.3.5"},
		{"a space separator", "Postfix 3.4.13", "postfix", "3.4.13"},
		{"empty input yields nothing", "", "", ""},
		{"a banner with no version yields nothing", "SSH-2.0", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product, version := parseServiceBanner(tc.in)
			if product != tc.wantProduct || version != tc.wantVersion {
				t.Errorf("got (%q, %q), want (%q, %q)",
					product, version, tc.wantProduct, tc.wantVersion)
			}
		})
	}
}

func TestParseHTTPServer(t *testing.T) {
	for _, tc := range []struct {
		name, in, wantProduct, wantVersion string
	}{
		{"the documented nginx example", "nginx/1.18.0", "nginx", "1.18.0"},
		{"IIS", "Microsoft-IIS/10.0", "microsoft-iis", "10.0"},
		{"Apache with a distribution suffix", "Apache/2.4.41 (Ubuntu)", "apache", "2.4.41"},
		{"a hyphenated product name", "lighttpd-fork/1.4.55", "lighttpd-fork", "1.4.55"},
		{"empty input yields nothing", "", "", ""},
		{"a header with no version yields nothing", "cloudflare", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product, version := parseHTTPServer(tc.in)
			if product != tc.wantProduct || version != tc.wantVersion {
				t.Errorf("got (%q, %q), want (%q, %q)",
					product, version, tc.wantProduct, tc.wantVersion)
			}
			if strings.HasSuffix(version, ".") {
				t.Errorf("version %q ends in a dot", version)
			}
		})
	}
}

func TestParseOSGuess(t *testing.T) {
	for _, tc := range []struct {
		name, in, wantProduct, wantVersion string
	}{
		{
			// Regression: [\d.]+ consumed the trailing dot and stopped before
			// the x, so TrimSuffix(".x") never fired and the version was "2.6.".
			name:        "the documented Linux example",
			in:          "Linux 2.6.x",
			wantProduct: "linux-kernel",
			wantVersion: "2.6",
		},
		{"a precise Linux version", "Linux 5.15.0", "linux-kernel", "5.15.0"},
		{"Linux with no version", "Linux", "linux-kernel", ""},
		{"Windows with a release", "Windows Server 2019", "microsoft-windows", "server 2019"},
		{"Windows with no release", "Windows", "microsoft-windows", ""},
		{"empty input yields nothing", "", "", ""},
		{"an unrecognised OS yields nothing", "FreeBSD 13.0", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product, version := parseOSGuess(tc.in)
			if product != tc.wantProduct || version != tc.wantVersion {
				t.Errorf("got (%q, %q), want (%q, %q)",
					product, version, tc.wantProduct, tc.wantVersion)
			}
			if strings.HasSuffix(version, ".") {
				t.Errorf("version %q ends in a dot; a CPE lookup will not match it", version)
			}
		})
	}
}
