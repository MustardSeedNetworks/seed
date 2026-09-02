package enumerate

import (
	"net"
	"testing"
)

// #328 moved the subnet filter out of the three platform readers and into
// readARPTable, so discovery keeps its scope while the neighbour-cache endpoint
// gets the whole table. These pin both halves of that split.

// The filter has to still apply on the discovery path, or a scan starts
// reporting devices outside its configured scope.
func TestReadARPTableStillNarrowsToScope(t *testing.T) {
	t.Parallel()

	_, subnet, err := net.ParseCIDR("192.0.2.0/24")
	if err != nil {
		t.Fatalf("parsing the fixture subnet: %v", err)
	}
	scanner := &ARPScanner{subnet: subnet}

	inScope := scanner.isInSubnet("192.0.2.10")
	outOfScope := scanner.isInSubnet("198.51.100.10")

	if !inScope {
		t.Error("an address inside the configured subnet was rejected")
	}
	if outOfScope {
		t.Error("an address outside the configured subnet was accepted; discovery would widen silently")
	}
}

// With no subnet configured the scanner accepts everything, which is the
// fallback the discovery path has always relied on.
func TestUnconfiguredScannerAcceptsEverything(t *testing.T) {
	t.Parallel()

	scanner := &ARPScanner{}
	if !scanner.isInSubnet("198.51.100.10") {
		t.Error("an unconfigured scanner rejected an address; discovery would find nothing")
	}
}

// vendorFor is the single implementation both the discovery enrichment and the
// neighbour-cache read now use, so its behaviour is worth pinning.
func TestVendorForNamesTheManufacturerOrSaysWhyNot(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		mac  string
		want string
	}{
		{
			// Bit 1 of the first octet set: assigned by software, so there is
			// no manufacturer to look up. A randomised client MAC looks like
			// this, and reporting it as an unknown vendor would be wrong.
			name: "a locally administered address is labelled, not looked up",
			mac:  "02:00:5e:10:00:00",
			want: "LAA",
		},
		{
			name: "an empty MAC has no vendor at all",
			mac:  "",
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := vendorFor(nil, tc.mac); got != tc.want {
				t.Errorf("vendorFor(nil, %q) = %q, want %q", tc.mac, got, tc.want)
			}
		})
	}
}
