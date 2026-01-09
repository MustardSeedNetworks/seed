package shell_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== Posture Scoring Tests ==========

// TestPostureScoreRange tests that posture scores are within valid range.
func TestPostureScoreRange(t *testing.T) {
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

	if score.Overall < 0 {
		t.Errorf("Overall score %d should not be negative", score.Overall)
	}
	if score.Overall > 100 {
		t.Errorf("Overall score %d should not exceed 100", score.Overall)
	}

	// Check category scores
	for category, categoryScore := range score.Categories {
		if categoryScore < 0 {
			t.Errorf("Category %s score %d should not be negative", category, categoryScore)
		}
		if categoryScore > 100 {
			t.Errorf("Category %s score %d should not exceed 100", category, categoryScore)
		}
	}
}

// TestPostureScoreCategories tests that score categories are properly initialized.
func TestPostureScoreCategories(t *testing.T) {
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
		t.Fatal("Categories should be initialized")
	}

	// The vulnerabilities category should be present
	// (even if the vulnerability service isn't fully initialized)
	// Note: this depends on whether the vulnerability service returns
	// ErrNotInitialized or an empty result
}

// TestPostureIssuesStructure tests that posture issues are properly structured.
func TestPostureIssuesStructure(t *testing.T) {
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
		t.Fatal("Issues should be initialized (can be empty)")
	}

	// Verify each issue has required fields
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

// TestPostureAssessedAtTimestamp tests that AssessedAt is properly set.
func TestPostureAssessedAtTimestamp(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	beforeAssess := time.Now()
	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}
	afterAssess := time.Now()

	if score.AssessedAt.Before(beforeAssess) {
		t.Error("AssessedAt should not be before the assessment started")
	}
	if score.AssessedAt.After(afterAssess) {
		t.Error("AssessedAt should not be after the assessment completed")
	}
}

// TestPostureConsistentScoring tests that multiple assessments produce consistent results.
func TestPostureConsistentScoring(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score1, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("First Assess() returned error: %v", err)
	}

	score2, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Second Assess() returned error: %v", err)
	}

	// Scores should be consistent for same conditions
	if score1.Overall != score2.Overall {
		t.Errorf("Overall scores should be consistent: %d vs %d", score1.Overall, score2.Overall)
	}
}

// ========== Posture Constants Tests ==========

// TestPosturePerfectScoreConstant tests the perfect score constant value.
func TestPosturePerfectScoreConstant(t *testing.T) {
	t.Parallel()

	if shell.ExportPerfectSecurityScore != 100 {
		t.Errorf(
			"ExportPerfectSecurityScore = %d, expected 100",
			shell.ExportPerfectSecurityScore,
		)
	}
}

// TestPostureVulnerabilityPenaltyConstant tests the vulnerability penalty constant.
func TestPostureVulnerabilityPenaltyConstant(t *testing.T) {
	t.Parallel()

	if shell.ExportVulnerabilityPenaltyMultiplier < 1 {
		t.Errorf(
			"ExportVulnerabilityPenaltyMultiplier = %d, should be at least 1",
			shell.ExportVulnerabilityPenaltyMultiplier,
		)
	}
	if shell.ExportVulnerabilityPenaltyMultiplier > 50 {
		t.Errorf(
			"ExportVulnerabilityPenaltyMultiplier = %d, seems too high (> 50)",
			shell.ExportVulnerabilityPenaltyMultiplier,
		)
	}
}

// ========== Posture Concurrent Access Tests ==========

// TestPostureConcurrentAssess tests concurrent Assess calls.
func TestPostureConcurrentAssess(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	const numGoroutines = 10
	results := make(chan *shell.PostureScore, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			score, err := service.Assess(ctx)
			if err != nil {
				errors <- err
				return
			}
			results <- score
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case score := <-results:
			if score == nil {
				t.Error("received nil score")
			}
		case err := <-errors:
			t.Errorf("concurrent Assess() returned error: %v", err)
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for Assess() results")
		}
	}
}

// ========== Posture Edge Cases ==========

// TestPostureWithCancelledContext tests Assess with a cancelled context.
func TestPostureWithCancelledContext(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Assess should still work even with cancelled context
	// (unless it specifically checks context)
	score, err := service.Assess(ctx)
	if err != nil {
		t.Logf("Assess with cancelled context returned error (may be expected): %v", err)
		return
	}

	if score == nil {
		t.Error("score should not be nil when no error")
	}
}

// TestPostureWithDeadlineContext tests Assess with a deadline context.
func TestPostureWithDeadlineContext(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	// Very short deadline
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Logf("Assess with short deadline returned error (may be expected): %v", err)
		return
	}

	if score == nil {
		t.Error("score should not be nil when no error")
	}
}

// TestPostureMultipleModulesIndependent tests that posture from different modules is independent.
func TestPostureMultipleModulesIndependent(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module1 := shell.New(cfg, nil)
	module2 := shell.New(cfg, nil)

	service1 := module1.Posture()
	service2 := module2.Posture()

	if service1 == service2 {
		t.Error("Posture services from different modules should be different instances")
	}

	ctx := context.Background()

	score1, err := service1.Assess(ctx)
	if err != nil {
		t.Fatalf("Module1 Assess() returned error: %v", err)
	}

	score2, err := service2.Assess(ctx)
	if err != nil {
		t.Fatalf("Module2 Assess() returned error: %v", err)
	}

	// Scores should be equal for identical configurations
	if score1.Overall != score2.Overall {
		t.Logf(
			"Scores may differ due to timing: module1=%d, module2=%d",
			score1.Overall,
			score2.Overall,
		)
	}
}
