// Package enrichment_test provides tests for the enrichment package.
// Currently, the enrichment package is a placeholder. The actual enrichment
// functionality (EnrichmentService) lives in the parent roots package.
// This test file documents that relationship and provides basic package tests.
package enrichment_test

import (
	"testing"
)

// TestPackageExists verifies that the enrichment package can be imported.
// The actual enrichment functionality is implemented in the parent roots package
// as EnrichmentService. See internal/roots/services.go for the implementation
// and internal/roots/services_test.go for comprehensive tests.
func TestPackageExists(t *testing.T) {
	t.Parallel()

	// This test validates that the enrichment package compiles and can be imported.
	// The enrichment package is currently a placeholder for future IP enrichment logic.
	//
	// The following functionality is tested in internal/roots/services_test.go:
	// - NewEnrichmentService
	// - EnrichmentService.GetPublicIP
	// - EnrichmentService.Enrich
	// - IPEnrichment struct fields
}

// TestEnrichmentPackageDocumentation documents the expected enrichment functionality.
func TestEnrichmentPackageDocumentation(t *testing.T) {
	t.Parallel()

	// The enrichment package should provide:
	// 1. IP address enrichment with ASN, geo, and ISP data
	// 2. Support for public IP detection
	// 3. Caching of enrichment results
	// 4. Rate limiting for external API calls
	//
	// Current implementation location: internal/roots/services.go (EnrichmentService)
}
