package vuln

import "testing"

func TestExtractVersion(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		// Each of these is a real string shape from the corpus. Between them
		// they cover every keyword and every version format it contains.
		{"Cisco after Version", "Cisco IOS Software, C3650 Software (C3650-UNIVERSALK9-M), Version 16.12.4, RELEASE SOFTWARE (fc5)", "16.12.4"},
		{"Cisco with a bracketed train", "Cisco IOS Software [Gibraltar], Catalyst L3 Switch Software, Version 17.6.3, RELEASE SOFTWARE (fc4)", "17.6.3"},
		{"Cisco parenthesised rebuild", "Cisco IOS Software, Version 15.2(4)M", "15.2(4)M"},
		{"HP after revision", "HP J4850A ProCurve Switch 5304XL, revision E.10.74, ROM E.05.05", "E.10.74"},
		{"Aruba after ArubaOS-CX", "Aruba CX 6300 48G 4SFP56 Switch, ArubaOS-CX FL.10.10.1010", "FL.10.10.1010"},
		{"HPE with a bare trailing version", "HPE Aruba 2930F 48G 4SFP+ Switch, FL.16.11.0009", "FL.16.11.0009"},
		{"Meraki after firmware", "Cisco Meraki MS390-48UXB Cloud-Managed Aggregation Switch, firmware 15.21", "15.21"},
		{"JunOS", "Juniper Networks, Inc. ex4300-48p Ethernet Switch, kernel JUNOS 20.4R3.8, Build date: 2021-10-15", "20.4R3.8"},
		{"JunOS with a letter-dot release", "Juniper Networks, Inc. srx240h2 internet router, kernel JUNOS 12.1X47-D15.4", "12.1X47-D15.4"},
		{"Brocade after IronWare Version", "Brocade Communications Systems, Inc. ICX6430-24-HPOE, IronWare Version 08.0.10dT311", "08.0.10dT311"},
		{"ExtremeXOS after version", "ExtremeXOS (X465-48W) version 32.3.1.4 by release-manager", "32.3.1.4"},
		{"VMware with no keyword", "VMware ESXi 5.5.0 build-1331820 VMware, Inc. x86_64", "5.5.0"},
		{"a Linux kernel release", "Linux Oracle Server 2.6.32-504.3.3.el6.x86_64 #1 SMP Wed Dec 17 01:55:02 UTC 2014 x86_64", "2.6.32-504.3.3.el6.x86_64"},

		// A build date sitting next to a version must not be mistaken for one.
		{"a build date is not a version", "Some Switch, Build date: 2021-10-15", ""},

		// Development images append build metadata the vendor still calls a
		// version; the release prefix is what a CPE lookup matches.
		{"a Cisco nightly keeps its release prefix", "Cisco IOS Software, Version 15.2(CML_NIGHTLY_20151103)FLO_DSGS7, EARLY DEPLOYMENT", "15.2"},
		{"a colon in the build metadata", "Cisco IOS Software, Experimental Version 15.1(20130726:213425) [dstivers 104]", "15.1"},

		{"no version present", "Cisco Controller", ""},
		{"no version present, model only", "RouterOS RB750", ""},
		{"empty input", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractVersion(tc.in); got != tc.want {
				t.Errorf("extractVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsVersionToken(t *testing.T) {
	for _, tc := range []struct {
		tok  string
		want bool
	}{
		{"16.12.4", true},
		{"FL.10.10.1010", true},
		{"12.1X47-D15.4", true},
		{"15.2(4)M", true},
		{"E.10.74", true},
		{"08.0.10dT311", true},

		{"", false},
		{"switch", false},        // no digit, no dot
		{"2960", false},          // a model number: digits but no dot
		{"48G", false},           // a port count
		{"2021-10-15", false},    // a build date
		{"build-1331820", false}, // too long an alphabetic prefix
		{"Software", false},      // no digit
		{"15:21", false},         // a clock time: colon is not a version char
	} {
		t.Run(tc.tok, func(t *testing.T) {
			if got := isVersionToken(tc.tok); got != tc.want {
				t.Errorf("isVersionToken(%q) = %v, want %v", tc.tok, got, tc.want)
			}
		})
	}
}

// TestContainsWord covers the substring bug directly: [strings.Contains] matched
// "hp" inside "ICX6430-24-HPOE", so every Brocade switch in the corpus was
// classified as HP and looked up against the wrong vendor's CVEs.
func TestContainsWord(t *testing.T) {
	for _, tc := range []struct {
		haystack, needle string
		want             bool
	}{
		{"hp j9729a 2920-48g switch", "hp", true},
		{"brocade icx6430-24-hpoe, ironware", "hp", false},
		{"extremexos (x465-48w) version 32.3.1.4", "extreme", false},
		{"extremexos (x465-48w) version 32.3.1.4", "extremexos", true},
		{"palo alto networks pa-440 firewall", "palo alto", true},
		{"cisco ios software", "cisco", true},
		{"ciscoish appliance", "cisco", false},
		{"aruba cx 6300 switch", "aruba", true},
		{"", "hp", false},
		{"anything", "", false},
	} {
		t.Run(tc.needle+"/"+tc.haystack[:min(20, len(tc.haystack))], func(t *testing.T) {
			if got := containsWord(tc.haystack, tc.needle); got != tc.want {
				t.Errorf("containsWord(%q, %q) = %v, want %v",
					tc.haystack, tc.needle, got, tc.want)
			}
		})
	}
}
