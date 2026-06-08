package vuln_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
)

// TestNVDRateLimitConstants verifies the NVD provider's rate-limit constants
// (moved here with the provider, ADR-0018).
func TestNVDRateLimitConstants(t *testing.T) {
	t.Parallel()
	if vuln.NVDRateLimitNoKey <= 0 {
		t.Error("Expected NVDRateLimitNoKey to be positive")
	}
	if vuln.NVDRateLimitWithKey <= 0 {
		t.Error("Expected NVDRateLimitWithKey to be positive")
	}
	if vuln.NVDRateLimitWithKey <= vuln.NVDRateLimitNoKey {
		t.Error("Expected NVDRateLimitWithKey to be greater than NVDRateLimitNoKey")
	}
}
