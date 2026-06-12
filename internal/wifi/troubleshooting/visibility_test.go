package troubleshooting_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// fakeSource is the tree/status half of the read use-case (the anomaly list now
// comes from the store, not here — ADR-0029 §4).
type fakeSource struct {
	tree   []airspace.SSIDGroup
	status visibility.Status
}

func (f fakeSource) Tree() []airspace.SSIDGroup { return f.tree }
func (f fakeSource) Status() visibility.Status  { return f.status }

// fakeAnomalyStore is the AnomalyStore half: the source=wifi anomaly list.
type fakeAnomalyStore struct {
	available bool
	anomalies []anomaly.Anomaly
	err       error
}

func (f fakeAnomalyStore) Available() bool { return f.available }
func (f fakeAnomalyStore) ActiveWiFi(context.Context) ([]anomaly.Anomaly, error) {
	return f.anomalies, f.err
}

func TestQueriesNilSourceDegradesEmpty(t *testing.T) {
	q := troubleshooting.NewQueries(nil, nil)
	air := q.Airspace()
	if air.SSIDs == nil || len(air.SSIDs) != 0 || air.Status.CaptureActive {
		t.Errorf("nil source should yield empty inactive airspace, got %+v", air)
	}
	an, err := q.Anomalies(context.Background())
	if err != nil {
		t.Fatalf("nil store should not error: %v", err)
	}
	if an.Anomalies == nil || len(an.Anomalies) != 0 {
		t.Errorf("nil store should yield empty anomaly stream, got %+v", an)
	}
}

// TestQueriesUnavailableStoreDegradesEmpty asserts an unwired store still yields
// the live status with an empty list (Wi-Fi's historic graceful contract).
func TestQueriesUnavailableStoreDegradesEmpty(t *testing.T) {
	src := fakeSource{status: visibility.Status{CaptureActive: true, Source: "mon0"}}
	q := troubleshooting.NewQueries(src, fakeAnomalyStore{available: false})

	an, err := q.Anomalies(context.Background())
	if err != nil {
		t.Fatalf("unavailable store should not error: %v", err)
	}
	if len(an.Anomalies) != 0 || an.Status.Source != "mon0" {
		t.Errorf("want empty list with live status, got %+v", an)
	}
}

func TestQueriesReadsListFromStoreStatusFromSource(t *testing.T) {
	src := fakeSource{
		tree:   []airspace.SSIDGroup{{SSID: "corp"}},
		status: visibility.Status{CaptureActive: true, Source: "mon0", SSIDs: 1},
	}
	store := fakeAnomalyStore{
		available: true,
		anomalies: []anomaly.Anomaly{{DefKey: "wifi-open-network"}},
	}
	q := troubleshooting.NewQueries(src, store)

	air := q.Airspace()
	if len(air.SSIDs) != 1 || air.SSIDs[0].SSID != "corp" || !air.Status.CaptureActive {
		t.Errorf("Airspace did not delegate to the source: %+v", air)
	}
	an, err := q.Anomalies(context.Background())
	if err != nil {
		t.Fatalf("Anomalies: %v", err)
	}
	// List comes from the store; status comes from the visibility source.
	if len(an.Anomalies) != 1 || an.Anomalies[0].DefKey != "wifi-open-network" || an.Status.Source != "mon0" {
		t.Errorf("Anomalies did not compose store list + source status: %+v", an)
	}
}

// TestQueriesSurfacesStoreError asserts a genuine store error propagates (the
// handler maps it to 500) rather than being silently swallowed.
func TestQueriesSurfacesStoreError(t *testing.T) {
	wantErr := errors.New("db down")
	q := troubleshooting.NewQueries(
		fakeSource{}, fakeAnomalyStore{available: true, err: wantErr})

	if _, err := q.Anomalies(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("store error not surfaced: got %v", err)
	}
}
