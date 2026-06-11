package monitoring_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
)

// --- fakes --------------------------------------------------------------------

type fakeAnomaly struct {
	available bool
	anomalies []anomaly.Anomaly
	err       error
}

func (f *fakeAnomaly) Available() bool { return f.available }
func (f *fakeAnomaly) ActiveAnomalies(context.Context) ([]anomaly.Anomaly, error) {
	return f.anomalies, f.err
}

func newService(an *fakeAnomaly) *monitoring.Service {
	return monitoring.NewService(an)
}

// --- tests --------------------------------------------------------------------

func anomalyForSubject(id string) anomaly.Anomaly {
	return anomaly.Anomaly{Subject: anomaly.SubjectRef{Kind: anomaly.SubjectDevice, ID: id}}
}

func TestAnomaliesFilterAndCount(t *testing.T) {
	t.Parallel()
	an := &fakeAnomaly{
		available: true,
		anomalies: []anomaly.Anomaly{
			anomalyForSubject("host-a"), anomalyForSubject("host-b"), anomalyForSubject("host-a"),
		},
	}
	svc := newService(an)

	t.Run("filtered list keeps the total active count", func(t *testing.T) {
		t.Parallel()
		got, err := svc.Anomalies(context.Background(), "host-a")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got.Anomalies) != 2 {
			t.Fatalf("want 2 filtered anomalies, got %d", len(got.Anomalies))
		}
		if got.ActiveCount != 3 {
			t.Fatalf("ActiveCount must be total (3), got %d", got.ActiveCount)
		}
	})
	t.Run("no filter returns all", func(t *testing.T) {
		t.Parallel()
		got, err := svc.Anomalies(context.Background(), "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got.Anomalies) != 3 || got.ActiveCount != 3 {
			t.Fatalf("want all 3, got %d (count %d)", len(got.Anomalies), got.ActiveCount)
		}
	})
}

func TestAnomaliesStoreError(t *testing.T) {
	t.Parallel()
	an := &fakeAnomaly{available: true, err: errors.New("query failed")}
	svc := newService(an)
	if _, err := svc.Anomalies(context.Background(), ""); err == nil {
		t.Fatal("want the store error to propagate, got nil")
	}
}

func TestAnomaliesUnavailable(t *testing.T) {
	t.Parallel()
	svc := newService(&fakeAnomaly{})
	if _, err := svc.Anomalies(context.Background(), ""); !errors.Is(err, monitoring.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}
