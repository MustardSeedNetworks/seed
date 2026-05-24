// SPDX-License-Identifier: BUSL-1.1

package license

import "testing"

// TestFromAlphanumeric verifies the digit/letter→int mapping used by
// the rotor cipher's checksum helpers. Kept in a package-internal
// test so the helper stays exercised even when callers don't.
func TestFromAlphanumeric(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   byte
		want int
	}{
		{'0', 0},
		{'9', 9},
		{'A', 10},
		{'Z', 35},
		// Lowercase is normalized to its uppercase value.
		{'a', 10},
		{'z', 35},
		// Anything outside [0-9A-Za-z] maps to 0.
		{'!', 0},
		{'/', 0},
	}
	for _, c := range cases {
		if got := fromAlphanumeric(c.in); got != c.want {
			t.Errorf("fromAlphanumeric(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
