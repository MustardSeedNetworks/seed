package performance_test

import (
	"testing"

	// Import the package to verify it compiles correctly.
	_ "github.com/krisarmstrong/seed/internal/sap/performance"
)

func TestPackageCompiles(t *testing.T) {
	// This test verifies that the performance package compiles correctly.
	// The package currently only contains doc.go with package documentation.
	// The actual implementation is in the parent sap.PerformanceService.
	t.Log("performance package compiles successfully")
}

func TestPackageImportable(t *testing.T) {
	// Verify the package can be imported without errors.
	// This is a compile-time check - if the import fails, the test won't run.
	t.Log("performance package is importable")
}
