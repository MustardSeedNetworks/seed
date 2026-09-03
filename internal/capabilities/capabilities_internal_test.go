package capabilities

import (
	"strings"
	"testing"
)

// The matrix drifted because nothing could check it. These are the checks.

// A capability added for one platform and forgotten on the others reads as
// "unsupported there" in the generated document, which is a silent wrong
// answer rather than a build failure. This makes it a build failure.
func TestEveryPlatformCoversEveryCapability(t *testing.T) {
	t.Parallel()

	for _, platform := range Platforms() {
		levels := levelsByPlatform()[platform]
		if levels == nil {
			t.Errorf("%s has no table at all", platform)

			continue
		}
		for _, capability := range order() {
			if _, ok := levels[capability]; !ok {
				t.Errorf("%s has no level for %q", platform, capability)
			}
		}
		for capability := range levels {
			if Title(capability) == string(capability) {
				t.Errorf("%s lists %q, which is not a known capability", platform, capability)
			}
		}
	}
}

// A level that is not Full has to say why. "Partial" on its own tells an
// operator nothing they can act on, and it is how a row stays wrong quietly.
func TestEveryDegradedCapabilityIsExplained(t *testing.T) {
	t.Parallel()

	for _, platform := range Platforms() {
		notes := notesByPlatform()[platform]
		for capability, level := range levelsByPlatform()[platform] {
			if level == LevelFull {
				continue
			}
			if strings.TrimSpace(notes[capability]) == "" {
				t.Errorf("%s reports %q as %q with no explanation", platform, capability, level)
			}
		}
	}
}

// The inverse: a note on a capability that is Full is stale, left behind when a
// gap was closed.
func TestNoNoteOnAFullySupportedCapability(t *testing.T) {
	t.Parallel()

	for _, platform := range Platforms() {
		for capability := range notesByPlatform()[platform] {
			if levelsByPlatform()[platform][capability] == LevelFull {
				t.Errorf("%s explains %q but reports it as full support", platform, capability)
			}
		}
	}
}

// Every capability must appear in the row order, or it silently vanishes from
// both the report and the document.
func TestOrderCoversEveryCapability(t *testing.T) {
	t.Parallel()

	seen := map[Capability]bool{}
	for _, capability := range order() {
		if seen[capability] {
			t.Errorf("%q appears twice in the row order", capability)
		}
		seen[capability] = true
	}
	for capability := range titles() {
		if !seen[capability] {
			t.Errorf("%q has a title but no place in the row order", capability)
		}
	}
	if len(order()) != len(titles()) {
		t.Errorf("row order covers %d capabilities, %d have titles",
			len(order()), len(titles()))
	}
}

// An unknown GOOS must report nothing usable rather than an empty table, which
// a caller would read as "everything works".
func TestUnknownPlatformSupportsNothing(t *testing.T) {
	t.Parallel()

	levels := LevelsFor("plan9")
	if len(levels) != len(order()) {
		t.Fatalf("LevelsFor(unknown) returned %d rows, want %d", len(levels), len(order()))
	}
	for capability, level := range levels {
		if level != LevelNone {
			t.Errorf("LevelsFor(unknown)[%q] = %q, want %q", capability, level, LevelNone)
		}
	}
}

// The correction this package exists to record. If someone restores the old
// optimistic value, this fails.
func TestKnownPlatformGapsAreRecorded(t *testing.T) {
	t.Parallel()

	if got := LevelsFor("windows")[VLANManagement]; got != LevelNone {
		t.Errorf("windows VLAN management = %q, want %q — no API outside Hyper-V (#2104)",
			got, LevelNone)
	}
}

// Degraded is what the UI warns about, so it must not include healthy rows.
func TestDegradedListsOnlyLessThanFull(t *testing.T) {
	t.Parallel()

	for _, entry := range Degraded() {
		if entry.Level == LevelFull {
			t.Errorf("Degraded() included %q at full support", entry.Capability)
		}
		if entry.Title == "" {
			t.Errorf("Degraded() entry %q has no title", entry.Capability)
		}
	}
}
