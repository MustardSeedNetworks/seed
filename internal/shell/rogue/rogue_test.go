package rogue_test

import (
	"testing"

	// Import the rogue package to ensure it compiles correctly.
	_ "github.com/krisarmstrong/seed/internal/shell/rogue"
)

// TestPackageExists verifies that the rogue package exists and can be imported.
// This is a placeholder test that ensures the package compiles correctly.
// The actual rogue detection tests are in the parent shell package.
func TestPackageExists(t *testing.T) {
	// The rogue subpackage currently only contains a doc.go file.
	// The actual RogueService implementation is in internal/shell/services.go.
	// This test verifies the package can be imported without issues.
	t.Log("rogue package exists and can be imported")
}
