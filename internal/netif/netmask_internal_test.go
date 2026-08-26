package netif

import "testing"

// TestParseNetmask covers all three forms an operator may type. Validation and
// every platform's apply path share this parser, so a mask that validates is a
// mask that applies — previously they used two different broken parsers and a
// dotted mask passed neither.
func TestParseNetmask(t *testing.T) {
	for _, tc := range []struct {
		name       string
		in         string
		wantPrefix int
		wantOK     bool
	}{
		// Bare prefix.
		{"bare prefix", "24", 24, true},
		{"bare prefix /8", "8", 8, true},
		{"bare prefix /32", "32", 32, true},
		{"bare prefix /0", "0", 0, true},

		// Prefix with the CIDR separator, which is how most people write it.
		{"slash prefix", "/24", 24, true},
		{"slash prefix /16", "/16", 16, true},
		{"slash prefix /32", "/32", 32, true},

		// Dotted decimal — the form validateIPConfig has always documented and
		// never accepted.
		{"dotted /24", "255.255.255.0", 24, true},
		{"dotted /16", "255.255.0.0", 16, true},
		{"dotted /8", "255.0.0.0", 8, true},
		{"dotted /25", "255.255.255.128", 25, true},
		{"dotted /30", "255.255.255.252", 30, true},
		{"dotted /0", "0.0.0.0", 0, true},

		// Surrounding whitespace is an operator typo, not a different mask.
		{"whitespace is trimmed", "  255.255.255.0  ", 24, true},
		{"whitespace around a slash prefix", " /24 ", 24, true},

		{"a non-contiguous mask is not a mask", "255.0.255.0", 0, false},
		{"prefix above 32", "33", 0, false},
		{"a wildly large prefix", "999", 0, false},
		{"a negative prefix", "-1", 0, false},
		{"an IPv6 address", "ffff:ffff::", 0, false},
		{"not an address at all", "not-a-mask", 0, false},
		{"empty", "", 0, false},
		{"a lone slash", "/", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefix, ok := parseNetmask(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("parseNetmask(%q) ok = %v, want %v", tc.in, ok, tc.wantOK)
			}
			if ok && prefix != tc.wantPrefix {
				t.Errorf("parseNetmask(%q) = %d, want %d", tc.in, prefix, tc.wantPrefix)
			}
		})
	}
}

// TestValidationAndApplyAgree is the property that matters more than either
// half: anything validateIPConfig accepts, the platform apply path must be able
// to convert. They used to disagree — a dotted mask failed validation, and on
// Linux would also have failed netmaskToCIDR with "invalid CIDR prefix: 255".
func TestValidationAndApplyAgree(t *testing.T) {
	for _, mask := range []string{
		"24", "/24", "255.255.255.0",
		"16", "/16", "255.255.0.0",
		"8", "/8", "255.0.0.0",
		"30", "/30", "255.255.255.252",
	} {
		t.Run(mask, func(t *testing.T) {
			cfg := &StaticIPConfig{
				Address: "192.168.1.10",
				Netmask: mask,
				Gateway: "192.168.1.1",
			}
			if err := validateIPConfig(cfg); err != nil {
				t.Fatalf("validateIPConfig rejected %q: %v", mask, err)
			}
			if _, ok := parseNetmask(mask); !ok {
				t.Errorf("validation accepted %q but the apply path cannot parse it",
					mask)
			}
		})
	}
}

// TestEquivalentFormsAgree pins that the three spellings of one mask produce
// one prefix. A device configured with "/24" and one configured with
// "255.255.255.0" must end up on the same network.
func TestEquivalentFormsAgree(t *testing.T) {
	for _, forms := range [][3]string{
		{"24", "/24", "255.255.255.0"},
		{"16", "/16", "255.255.0.0"},
		{"8", "/8", "255.0.0.0"},
		{"25", "/25", "255.255.255.128"},
	} {
		t.Run(forms[0], func(t *testing.T) {
			var prefixes []int
			for _, form := range forms {
				prefix, ok := parseNetmask(form)
				if !ok {
					t.Fatalf("parseNetmask(%q) failed", form)
				}
				prefixes = append(prefixes, prefix)
			}
			if prefixes[0] != prefixes[1] || prefixes[1] != prefixes[2] {
				t.Errorf("%v parse to %v; the three forms disagree", forms, prefixes)
			}
		})
	}
}
