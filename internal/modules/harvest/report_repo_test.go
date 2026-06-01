package harvest_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/modules/harvest"
)

// fakeReportRepo is an in-memory harvest.ReportRepo. It is the payoff of the
// ReportRepo port (PHASE3_EXTRACTION_PLAN.md §4.5): GeneratorService's
// report-record orchestration can now be exercised with no database and no
// filesystem.
type fakeReportRepo struct {
	reports map[string]*harvest.Report
	saveErr error
}

func newFakeReportRepo() *fakeReportRepo {
	return &fakeReportRepo{reports: make(map[string]*harvest.Report)}
}

func (f *fakeReportRepo) GetReport(_ context.Context, id string) (*harvest.Report, error) {
	r, ok := f.reports[id]
	if !ok {
		return nil, fmt.Errorf("report not found: %s", id)
	}
	clone := *r
	return &clone, nil
}

func (f *fakeReportRepo) ListReports(_ context.Context) ([]harvest.Report, error) {
	out := make([]harvest.Report, 0, len(f.reports))
	for _, r := range f.reports {
		out = append(out, *r)
	}
	return out, nil
}

func (f *fakeReportRepo) SaveReport(_ context.Context, r *harvest.Report) error {
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
	gs := harvest.NewGeneratorService(testConfig(), repo, nil, nil, nil)
	ctx := context.Background()

	rep := &harvest.Report{
		ID:        "r1",
		Name:      "Quarterly",
		Type:      harvest.ReportTypeExecutive,
		Format:    harvest.FormatJSON,
		Status:    harvest.StatusComplete,
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
	gs := harvest.NewGeneratorService(testConfig(), repo, nil, nil, nil)

	err := gs.ExportSaveReport(context.Background(), &harvest.Report{ID: "r1", CreatedAt: time.Now()})
	require.ErrorContains(t, err, "disk full")
}
