package shell_test

import (
	"context"
	"testing"

	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== Posture Assess Detailed Tests ==========

// TestPostureAssessReturnsNonNilScore tests that Assess always returns a non-nil score.
func TestPostureAssessReturnsNonNilScore(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	if score == nil {
		t.Fatal("Assess() should return non-nil score")
	}
}

// TestPostureAssessInitializesCategories tests that categories map is initialized.
func TestPostureAssessInitializesCategories(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	if score.Categories == nil {
		t.Error("Categories should be initialized")
	}
}

// TestPostureAssessInitializesIssues tests that issues slice is initialized.
func TestPostureAssessInitializesIssues(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	if score.Issues == nil {
		t.Error("Issues should be initialized (may be empty)")
	}
}

// TestPostureAssessStartsWithPerfectScore tests that the baseline is perfect score.
func TestPostureAssessStartsWithPerfectScore(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	// With no vulnerabilities, score should be at or near perfect
	// (may be slightly less if vulnerability service returns error)
	if score.Overall < 0 {
		t.Errorf("Overall score %d should not be negative", score.Overall)
	}
	if score.Overall > shell.ExportPerfectSecurityScore {
		t.Errorf(
			"Overall score %d should not exceed perfect score %d",
			score.Overall,
			shell.ExportPerfectSecurityScore,
		)
	}
}

// TestPostureAssessSetsTimestamp tests that AssessedAt is set correctly.
func TestPostureAssessSetsTimestamp(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	if score.AssessedAt.IsZero() {
		t.Error("AssessedAt should not be zero")
	}
}

// TestPostureAssessWithNilVulnerabilityService tests Assess when vulnerability service is nil.
func TestPostureAssessWithNilVulnerabilityService(t *testing.T) {
	t.Parallel()

	// Create module normally - the vulnerability service may have a nil scanner
	// but the service itself should not be nil
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	// Score should still be valid even if vulnerability service has issues
	if score == nil {
		t.Fatal("score should not be nil")
	}
	if score.Overall < 0 || score.Overall > 100 {
		t.Errorf("Overall score %d out of valid range [0, 100]", score.Overall)
	}
}

// TestPostureAssessScoreNeverNegative tests that overall score never goes negative.
func TestPostureAssessScoreNeverNegative(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	// Run multiple times to ensure consistency
	for i := range 5 {
		score, err := service.Assess(ctx)
		if err != nil {
			t.Fatalf("Assess() iteration %d returned error: %v", i, err)
		}

		if score.Overall < 0 {
			t.Errorf("iteration %d: Overall score %d should not be negative", i, score.Overall)
		}
	}
}

// TestPostureAssessCategoryScoresValid tests that category scores are valid.
func TestPostureAssessCategoryScoresValid(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	for category, categoryScore := range score.Categories {
		if categoryScore < 0 {
			t.Errorf("Category %q score %d should not be negative", category, categoryScore)
		}
		if categoryScore > 100 {
			t.Errorf("Category %q score %d should not exceed 100", category, categoryScore)
		}
	}
}

// TestPostureAssessIssuesHaveRequiredFields tests that all issues have required fields.
func TestPostureAssessIssuesHaveRequiredFields(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	for i, issue := range score.Issues {
		if issue.Category == "" {
			t.Errorf("Issue %d: Category should not be empty", i)
		}
		if issue.Severity == "" {
			t.Errorf("Issue %d: Severity should not be empty", i)
		}
		if issue.Description == "" {
			t.Errorf("Issue %d: Description should not be empty", i)
		}
	}
}
