package reporting_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// fakeReportRepo is an in-memory reporting.ReportRepo. It is the payoff of the
// ReportRepo port: GeneratorService's
// report-record orchestration can now be exercised with no database and no
// filesystem.
type fakeReportRepo struct {
	reports map[string]*reporting.Report
	saveErr error
}

func newFakeReportRepo() *fakeReportRepo {
	return &fakeReportRepo{reports: make(map[string]*reporting.Report)}
}

func (f *fakeReportRepo) GetReport(_ context.Context, id string) (*reporting.Report, error) {
	r, ok := f.reports[id]
	if !ok {
		return nil, fmt.Errorf("report not found: %s", id)
	}
	clone := *r
	return &clone, nil
}

func (f *fakeReportRepo) UpdateReport(_ context.Context, r *reporting.Report) (bool, error) {
	if f.saveErr != nil {
		return false, f.saveErr
	}
	if _, ok := f.reports[r.ID]; !ok {
		return false, nil
	}
	clone := *r
	f.reports[r.ID] = &clone
	return true, nil
}

func (f *fakeReportRepo) ListReports(_ context.Context) ([]reporting.Report, error) {
	out := make([]reporting.Report, 0, len(f.reports))
	for _, r := range f.reports {
		out = append(out, *r)
	}
	return out, nil
}

func (f *fakeReportRepo) SaveReport(_ context.Context, r *reporting.Report) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	clone := *r
	f.reports[r.ID] = &clone
	return nil
}

func (f *fakeReportRepo) DeleteReport(_ context.Context, id string) error {
	delete(f.reports, id)
	return nil
}

// TestGeneratorService_ReportCRUD_NoDB exercises save → get → list → delete
// against a fake repo. db/templates/aggregator are nil: report-record CRUD
// touches none of them, which is the whole point of the port.
func TestGeneratorService_ReportCRUD_NoDB(t *testing.T) {
	repo := newFakeReportRepo()
	gs := reporting.NewGeneratorService(testConfig(), repo, nil, nil, nil)
	ctx := context.Background()

	rep := &reporting.Report{
		ID:        "r1",
		Name:      "Quarterly",
		Type:      reporting.ReportTypeExecutive,
		Format:    reporting.FormatJSON,
		Status:    reporting.StatusComplete,
		CreatedAt: time.Now(),
	}
	require.NoError(t, gs.ExportSaveReport(ctx, rep))

	got, err := gs.GetReport(ctx, "r1")
	require.NoError(t, err)
	assert.Equal(t, "Quarterly", got.Name)

	list, err := gs.ListReports(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, gs.DeleteReport(ctx, "r1"))
	_, err = gs.GetReport(ctx, "r1")
	require.Error(t, err)
}

// TestGeneratorService_SaveReportError surfaces a repo write failure through
// the service unchanged.
func TestGeneratorService_SaveReportError(t *testing.T) {
	repo := newFakeReportRepo()
	repo.saveErr = errors.New("disk full")
	gs := reporting.NewGeneratorService(testConfig(), repo, nil, nil, nil)

	err := gs.ExportSaveReport(context.Background(), &reporting.Report{ID: "r1", CreatedAt: time.Now()})
	require.ErrorContains(t, err, "disk full")
}

// TestGeneratorService_DeletedReportIsNotResurrected pins the race that wiring
// the DELETE endpoint makes reachable (#2154).
//
// Generate returns as soon as the initial row is written and finishes the work
// in a goroutine that writes status again afterwards. Those writes used to go
// through SaveReport, an INSERT OR REPLACE, so a report deleted while it was
// still generating came back with whatever status the goroutine wrote next.
func TestGeneratorService_DeletedReportIsNotResurrected(t *testing.T) {
	repo := newFakeReportRepo()
	gs := reporting.NewGeneratorService(testConfig(), repo, nil, nil, nil)
	ctx := context.Background()

	rep := &reporting.Report{
		ID:        "generating-1",
		Name:      "In flight",
		Type:      reporting.ReportTypeExecutive,
		Format:    reporting.FormatJSON,
		Status:    reporting.StatusGenerating,
		CreatedAt: time.Now(),
	}
	require.NoError(t, gs.ExportSaveReport(ctx, rep))
	require.NoError(t, gs.DeleteReport(ctx, rep.ID))

	// The generation goroutine's next write lands here, after the delete.
	rep.Status = reporting.StatusComplete
	live, err := repo.UpdateReport(ctx, rep)
	require.NoError(t, err)
	assert.False(t, live, "update matched a row that was deleted")

	_, err = gs.GetReport(ctx, rep.ID)
	require.Error(t, err, "deleted report came back after the generator wrote to it")

	list, err := gs.ListReports(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}
