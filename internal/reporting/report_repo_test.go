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
