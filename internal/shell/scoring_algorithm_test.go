// Package shell_test provides comprehensive tests for the posture scoring algorithm.
// These tests exercise all severity levels and edge cases in vulnerability scoring.
package shell_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/shell"
)

// ========== Posture Scoring Algorithm Tests ==========

// TestAssessWithMockVulnsCritical tests scoring with critical vulnerabilities.
func TestAssessWithMockVulnsCritical(t *testing.T) {
	t.Parallel()

	vulns := []shell.Vulnerability{
		{ID: "v1", CVEID: "CVE-2024-0001", Severity: shell.SeverityCritical, Remediation: "Patch immediately"},
	}

	score := shell.AssessWithMockVulns(vulns)

	// Critical deducts 20 points
	expectedOverall := shell.ExportPerfectSecurityScore - 20
	if score.Overall != expectedOverall {
		t.Errorf("Overall = %d, want %d", score.Overall, expectedOverall)
	}

	// Should have 1 issue
	if len(score.Issues) != 1 {
		t.Errorf("Issues count = %d, want 1", len(score.Issues))
	}

	if len(score.Issues) > 0 {
		if score.Issues[0].Severity != "critical" {
			t.Errorf("Issue severity = %s, want critical", score.Issues[0].Severity)
		}
		if score.Issues[0].Category != "vulnerabilities" {
			t.Errorf("Issue category = %s, want vulnerabilities", score.Issues[0].Category)
		}
	}
}

// TestAssessWithMockVulnsHigh tests scoring with high severity vulnerabilities.
func TestAssessWithMockVulnsHigh(t *testing.T) {
	t.Parallel()

	vulns := []shell.Vulnerability{
		{ID: "v1", CVEID: "CVE-2024-0002", Severity: shell.SeverityHigh},
	}

	score := shell.AssessWithMockVulns(vulns)

	// High deducts 10 points
	expectedOverall := shell.ExportPerfectSecurityScore - 10
	if score.Overall != expectedOverall {
		t.Errorf("Overall = %d, want %d", score.Overall, expectedOverall)
	}

	// High doesn't create issues in current implementation
	if len(score.Issues) != 0 {
		t.Errorf("Issues count = %d, want 0", len(score.Issues))
	}
}

// TestAssessWithMockVulnsMedium tests scoring with medium severity vulnerabilities.
func TestAssessWithMockVulnsMedium(t *testing.T) {
	t.Parallel()

	vulns := []shell.Vulnerability{
		{ID: "v1", CVEID: "CVE-2024-0003", Severity: shell.SeverityMedium},
	}

	score := shell.AssessWithMockVulns(vulns)

	// Medium deducts 5 points
	expectedOverall := shell.ExportPerfectSecurityScore - 5
	if score.Overall != expectedOverall {
		t.Errorf("Overall = %d, want %d", score.Overall, expectedOverall)
	}
}

// TestAssessWithMockVulnsLow tests scoring with low severity vulnerabilities.
func TestAssessWithMockVulnsLow(t *testing.T) {
	t.Parallel()

	vulns := []shell.Vulnerability{
		{ID: "v1", CVEID: "CVE-2024-0004", Severity: shell.SeverityLow},
	}

	score := shell.AssessWithMockVulns(vulns)

	// Low deducts 2 points
	expectedOverall := shell.ExportPerfectSecurityScore - 2
	if score.Overall != expectedOverall {
		t.Errorf("Overall = %d, want %d", score.Overall, expectedOverall)
	}
}

// TestAssessWithMockVulnsInfo tests scoring with info severity vulnerabilities.
func TestAssessWithMockVulnsInfo(t *testing.T) {
	t.Parallel()

	vulns := []shell.Vulnerability{
		{ID: "v1", CVEID: "CVE-2024-0005", Severity: shell.SeverityInfo},
	}

	score := shell.AssessWithMockVulns(vulns)

	// Info doesn't deduct any points
	if score.Overall != shell.ExportPerfectSecurityScore {
		t.Errorf("Overall = %d, want %d", score.Overall, shell.ExportPerfectSecurityScore)
	}
}

// TestAssessWithMockVulnsMixed tests scoring with mixed severity vulnerabilities.
func TestAssessWithMockVulnsMixed(t *testing.T) {
	t.Parallel()

	vulns := []shell.Vulnerability{
		{ID: "v1", CVEID: "CVE-2024-0001", Severity: shell.SeverityCritical, Remediation: "Patch"},
		{ID: "v2", CVEID: "CVE-2024-0002", Severity: shell.SeverityHigh},
		{ID: "v3", CVEID: "CVE-2024-0003", Severity: shell.SeverityMedium},
		{ID: "v4", CVEID: "CVE-2024-0004", Severity: shell.SeverityLow},
		{ID: "v5", CVEID: "CVE-2024-0005", Severity: shell.SeverityInfo},
	}

	score := shell.AssessWithMockVulns(vulns)

	// Total deduction: 20 + 10 + 5 + 2 + 0 = 37
	expectedOverall := shell.ExportPerfectSecurityScore - 37
	if score.Overall != expectedOverall {
		t.Errorf("Overall = %d, want %d", score.Overall, expectedOverall)
	}

	// Only critical creates issues
	if len(score.Issues) != 1 {
		t.Errorf("Issues count = %d, want 1", len(score.Issues))
	}
}

// TestAssessWithMockVulnsEmpty tests scoring with no vulnerabilities.
func TestAssessWithMockVulnsEmpty(t *testing.T) {
	t.Parallel()

	vulns := []shell.Vulnerability{}

	score := shell.AssessWithMockVulns(vulns)

	// No vulnerabilities = perfect score
	if score.Overall != shell.ExportPerfectSecurityScore {
		t.Errorf("Overall = %d, want %d", score.Overall, shell.ExportPerfectSecurityScore)
	}

	if len(score.Issues) != 0 {
		t.Errorf("Issues count = %d, want 0", len(score.Issues))
	}

	// Categories should include vulnerabilities with perfect score
	if catScore, ok := score.Categories["vulnerabilities"]; ok {
		if catScore != shell.ExportPerfectSecurityScore {
			t.Errorf("vulnerabilities category = %d, want %d", catScore, shell.ExportPerfectSecurityScore)
		}
	}
}

// TestAssessWithMockVulnsNil tests scoring with nil vulnerabilities.
func TestAssessWithMockVulnsNil(t *testing.T) {
	t.Parallel()

	score := shell.AssessWithMockVulns(nil)

	// Nil vulnerabilities = perfect score
	if score.Overall != shell.ExportPerfectSecurityScore {
		t.Errorf("Overall = %d, want %d", score.Overall, shell.ExportPerfectSecurityScore)
	}
}

// TestAssessWithMockVulnsScoreFloor tests that score doesn't go below 0.
func TestAssessWithMockVulnsScoreFloor(t *testing.T) {
	t.Parallel()

	// Create many critical vulnerabilities to exceed 100 points
	vulns := make([]shell.Vulnerability, 10)
	for i := range vulns {
		vulns[i] = shell.Vulnerability{
			ID:       "v" + string(rune('0'+i)),
			CVEID:    "CVE-2024-000" + string(rune('0'+i)),
			Severity: shell.SeverityCritical,
		}
	}

	score := shell.AssessWithMockVulns(vulns)

	// Total deduction: 10 * 20 = 200, but floor is 0
	if score.Overall != 0 {
		t.Errorf("Overall = %d, want 0 (floor)", score.Overall)
	}

	// Should have 10 issues
	if len(score.Issues) != 10 {
		t.Errorf("Issues count = %d, want 10", len(score.Issues))
	}
}

// TestAssessWithMockVulnsCategoryScore tests category score calculation.
func TestAssessWithMockVulnsCategoryScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		vulnCount    int
		wantCatScore int
	}{
		{"no_vulns", 0, 100},
		{"one_vuln", 1, 95},
		{"two_vulns", 2, 90},
		{"twenty_vulns", 20, 0},
		{"hundred_vulns", 100, 0}, // Floor at 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vulns := make([]shell.Vulnerability, tt.vulnCount)
			for i := range vulns {
				vulns[i] = shell.Vulnerability{
					ID:       "v" + string(rune('0'+i%10)),
					Severity: shell.SeverityMedium,
				}
			}

			score := shell.AssessWithMockVulns(vulns)

			catScore, ok := score.Categories["vulnerabilities"]
			if !ok {
				t.Fatal("vulnerabilities category should exist")
			}
			if catScore != tt.wantCatScore {
				t.Errorf("vulnerabilities category = %d, want %d", catScore, tt.wantCatScore)
			}
		})
	}
}

// ========== Vulnerability Counting Tests ==========

// TestCountVulnsBySeverity tests the severity counting function.
func TestCountVulnsBySeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		vulns        []shell.Vulnerability
		wantCritical int
		wantHigh     int
		wantMedium   int
		wantLow      int
	}{
		{
			name:         "empty",
			vulns:        []shell.Vulnerability{},
			wantCritical: 0,
			wantHigh:     0,
			wantMedium:   0,
			wantLow:      0,
		},
		{
			name: "all_critical",
			vulns: []shell.Vulnerability{
				{Severity: shell.SeverityCritical},
				{Severity: shell.SeverityCritical},
			},
			wantCritical: 2,
			wantHigh:     0,
			wantMedium:   0,
			wantLow:      0,
		},
		{
			name: "all_high",
			vulns: []shell.Vulnerability{
				{Severity: shell.SeverityHigh},
				{Severity: shell.SeverityHigh},
				{Severity: shell.SeverityHigh},
			},
			wantCritical: 0,
			wantHigh:     3,
			wantMedium:   0,
			wantLow:      0,
		},
		{
			name: "all_medium",
			vulns: []shell.Vulnerability{
				{Severity: shell.SeverityMedium},
			},
			wantCritical: 0,
			wantHigh:     0,
			wantMedium:   1,
			wantLow:      0,
		},
		{
			name: "all_low",
			vulns: []shell.Vulnerability{
				{Severity: shell.SeverityLow},
				{Severity: shell.SeverityLow},
			},
			wantCritical: 0,
			wantHigh:     0,
			wantMedium:   0,
			wantLow:      2,
		},
		{
			name: "mixed",
			vulns: []shell.Vulnerability{
				{Severity: shell.SeverityCritical},
				{Severity: shell.SeverityHigh},
				{Severity: shell.SeverityHigh},
				{Severity: shell.SeverityMedium},
				{Severity: shell.SeverityMedium},
				{Severity: shell.SeverityMedium},
				{Severity: shell.SeverityLow},
				{Severity: shell.SeverityInfo}, // Info not counted
			},
			wantCritical: 1,
			wantHigh:     2,
			wantMedium:   3,
			wantLow:      1,
		},
		{
			name: "all_info",
			vulns: []shell.Vulnerability{
				{Severity: shell.SeverityInfo},
				{Severity: shell.SeverityInfo},
			},
			wantCritical: 0,
			wantHigh:     0,
			wantMedium:   0,
			wantLow:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			critical, high, medium, low := shell.CountVulnsBySeverity(tt.vulns)

			if critical != tt.wantCritical {
				t.Errorf("critical = %d, want %d", critical, tt.wantCritical)
			}
			if high != tt.wantHigh {
				t.Errorf("high = %d, want %d", high, tt.wantHigh)
			}
			if medium != tt.wantMedium {
				t.Errorf("medium = %d, want %d", medium, tt.wantMedium)
			}
			if low != tt.wantLow {
				t.Errorf("low = %d, want %d", low, tt.wantLow)
			}
		})
	}
}

// TestCountVulnsBySeverityNil tests counting with nil slice.
func TestCountVulnsBySeverityNil(t *testing.T) {
	t.Parallel()

	critical, high, medium, low := shell.CountVulnsBySeverity(nil)

	if critical != 0 || high != 0 || medium != 0 || low != 0 {
		t.Errorf("counts should all be 0 for nil slice, got c=%d h=%d m=%d l=%d",
			critical, high, medium, low)
	}
}

// ========== Posture Score Validation Tests ==========

// TestAssessWithMockVulnsAssessedAt tests that AssessedAt is set.
func TestAssessWithMockVulnsAssessedAt(t *testing.T) {
	t.Parallel()

	score := shell.AssessWithMockVulns([]shell.Vulnerability{})

	if score.AssessedAt.IsZero() {
		t.Error("AssessedAt should be set")
	}
}

// TestAssessWithMockVulnsCategoriesInitialized tests that categories is initialized.
func TestAssessWithMockVulnsCategoriesInitialized(t *testing.T) {
	t.Parallel()

	score := shell.AssessWithMockVulns([]shell.Vulnerability{})

	if score.Categories == nil {
		t.Error("Categories should be initialized")
	}
}

// TestAssessWithMockVulnsIssuesInitialized tests that issues is initialized.
func TestAssessWithMockVulnsIssuesInitialized(t *testing.T) {
	t.Parallel()

	score := shell.AssessWithMockVulns([]shell.Vulnerability{})

	if score.Issues == nil {
		t.Error("Issues should be initialized")
	}
}
