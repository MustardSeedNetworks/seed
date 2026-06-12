package app

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
)

// TestEnrichAnomalies asserts the composition-root helper fills the catalog-static
// Impact / FollowUps a store read omits (ADR-0029), and that a nil engine (no
// platform wired) leaves the list untouched.
func TestEnrichAnomalies(t *testing.T) {
	cat, err := anomaly.NewCatalog(anomaly.Def{
		ID: "probe-unreachable", Category: anomaly.CategoryNetHealth,
		DefaultSeverity: anomaly.SeverityCritical,
		Title:           "Unreachable", Description: "down.",
		Recommendation: "check it.", Impact: "service is down.",
	})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	engine := anomaly.NewEngine(cat)

	// A store-backed read: scalar fields present, Impact blank (unpersisted).
	list := []anomaly.Anomaly{{DefKey: "probe-unreachable", Title: "Unreachable"}}

	got := enrichAnomalies(engine, list)
	if len(got) != 1 || got[0].Impact != "service is down." {
		t.Errorf("enriched Impact = %q, want it re-derived from the catalog", got[0].Impact)
	}

	// Nil engine → passthrough, no enrichment.
	raw := []anomaly.Anomaly{{DefKey: "probe-unreachable"}}
	if passthrough := enrichAnomalies(nil, raw); passthrough[0].Impact != "" {
		t.Errorf("nil engine should not enrich, got Impact = %q", passthrough[0].Impact)
	}
}
