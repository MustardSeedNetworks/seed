package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// sweepJobs evicts terminal jobs past the retention window (ADR-0005, Phase 5c).
// It runs both halves of jobs retention: the in-memory runner's Cleanup (frees
// the snapshot map) and the durable store's DeleteCompletedBefore (frees disk).
// repo may be nil (no database) — then only the in-memory sweep runs. It is the
// periodic jobs-retention task, driven by the maintenance loop; the two
// primitives it calls are independently safe for terminal-only, age-bounded
// eviction.
func sweepJobs(
	ctx context.Context,
	runner *jobs.Runner,
	repo *database.JobRepository,
	cutoff time.Time,
	logger *slog.Logger,
) {
	if runner != nil {
		if n := runner.Cleanup(); n > 0 {
			logger.DebugContext(ctx, "evicted terminal jobs from memory", "count", n)
		}
	}
	if repo != nil {
		if n, err := repo.DeleteCompletedBefore(ctx, cutoff); err != nil {
			logger.ErrorContext(ctx, "jobs retention cleanup failed", "error", err)
		} else if n > 0 {
			logger.InfoContext(ctx, "jobs retention cleanup completed", "jobs_deleted", n)
		}
	}
}
