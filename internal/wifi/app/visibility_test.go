package wifiapp_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/anomaly"
	"github.com/krisarmstrong/seed/internal/wifi/airspace"
	wifiapp "github.com/krisarmstrong/seed/internal/wifi/app"
	"github.com/krisarmstrong/seed/internal/wifi/visibility"
)

type fakeSource struct {
	tree      []airspace.SSIDGroup
	anomalies []anomaly.Anomaly
	status    visibility.Status
}

func (f fakeSource) Tree() []airspace.SSIDGroup   { return f.tree }
func (f fakeSource) Anomalies() []anomaly.Anomaly { return f.anomalies }
func (f fakeSource) Status() visibility.Status    { return f.status }

func TestQueriesNilSourceDegradesEmpty(t *testing.T) {
	q := wifiapp.NewQueries(nil)
	air := q.Airspace()
	if air.SSIDs == nil || len(air.SSIDs) != 0 || air.Status.CaptureActive {
		t.Errorf("nil source should yield empty inactive airspace, got %+v", air)
	}
	an := q.Anomalies()
	if an.Anomalies == nil || len(an.Anomalies) != 0 {
		t.Errorf("nil source should yield empty anomaly stream, got %+v", an)
	}
}

func TestQueriesDelegatesToSource(t *testing.T) {
	src := fakeSource{
		tree:      []airspace.SSIDGroup{{SSID: "corp"}},
		anomalies: []anomaly.Anomaly{{DefKey: "wifi-open-network"}},
		status:    visibility.Status{CaptureActive: true, Source: "mon0", SSIDs: 1},
	}
	q := wifiapp.NewQueries(src)

	air := q.Airspace()
	if len(air.SSIDs) != 1 || air.SSIDs[0].SSID != "corp" || !air.Status.CaptureActive {
		t.Errorf("Airspace did not delegate to the source: %+v", air)
	}
	an := q.Anomalies()
	if len(an.Anomalies) != 1 || an.Anomalies[0].DefKey != "wifi-open-network" || an.Status.Source != "mon0" {
		t.Errorf("Anomalies did not delegate to the source: %+v", an)
	}
}
