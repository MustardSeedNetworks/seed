package pipeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	alertmodel "github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// A route is one ipCidrRouteTable row as "destination mask next-hop".
const (
	lan     = "10.1.0.0 255.255.0.0 192.0.2.1"
	wan     = "203.0.113.0 255.255.255.0 192.0.2.1"
	wanAlt  = "203.0.113.0 255.255.255.0 192.0.2.9"
	ecmpA   = "198.51.100.0 255.255.255.0 192.0.2.1"
	ecmpB   = "198.51.100.0 255.255.255.0 192.0.2.2"
	oddMask = "10.9.0.0 255.0.255.0 192.0.2.1"
)

func routingObs(observed time.Time, truncated bool, routes ...string) *observation.SNMPObservation {
	rows := make([]map[string]any, 0, len(routes))
	for _, r := range routes {
		f := strings.Fields(r)
		rows = append(rows, map[string]any{
			"Destination": f[0], "Mask": f[1], "NextHop": f[2],
			"Type": 4, "Proto": 14,
		})
	}
	b, _ := json.Marshal(map[string]any{"Routes": rows, "Truncated": truncated})
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: "t-1", Kind: "routing",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

// decodeEvidence decodes an alert's Metadata strictly into T.
func decodeEvidence[T any](t *testing.T, a *alertmodel.Alert) T {
	t.Helper()
	var got T
	dec := json.NewDecoder(strings.NewReader(a.Metadata))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("metadata %s: %v", a.Metadata, err)
	}
	return got
}

func scanRouting(t *testing.T, polls ...*observation.SNMPObservation) []*alertmodel.Alert {
	t.Helper()
	alerts := &fakeAlerts{}
	p, err := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: &fakeObservations{rows: polls}, Alerts: alerts,
		Settings: newFakeSettings(), Logger: silentLogger(), Now: at,
	})
	if err != nil {
		t.Fatalf("NewObservationPipeline: %v", err)
	}
	if err = p.ScanOnce(context.Background()); err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	return alerts.created
}

func TestObservationScanOnce_Routing(t *testing.T) {
	t.Parallel()
	before, after := at().Add(-time.Minute), at()
	tests := []struct {
		name  string
		polls []*observation.SNMPObservation
		rules []string
	}{
		{"first poll", []*observation.SNMPObservation{routingObs(after, false, lan)}, nil},
		{"steady table", []*observation.SNMPObservation{
			routingObs(before, false, lan, wan), routingObs(after, false, lan, wan),
		}, nil},
		{"new prefix", []*observation.SNMPObservation{
			routingObs(before, false, lan), routingObs(after, false, lan, wan),
		}, nil},
		{"ECMP next hops listed in another order", []*observation.SNMPObservation{
			routingObs(before, false, ecmpA, ecmpB), routingObs(after, false, ecmpB, ecmpA),
		}, nil},
		{"withdrawal", []*observation.SNMPObservation{
			routingObs(before, false, lan, wan), routingObs(after, false, lan),
		}, []string{alertmodel.RuleRouteWithdrawn}},
		{"truncated poll hides withdrawals", []*observation.SNMPObservation{
			routingObs(before, false, lan, wan), routingObs(after, true, lan),
		}, nil},
		{"truncated poll still reports next-hop changes", []*observation.SNMPObservation{
			routingObs(before, false, lan, wan), routingObs(after, true, wanAlt),
		}, []string{alertmodel.RuleRouteChange}},
		{"one ECMP leg lost", []*observation.SNMPObservation{
			routingObs(before, false, ecmpA, ecmpB), routingObs(after, false, ecmpA),
		}, []string{alertmodel.RuleRouteChange}},
		{"withdrawal and change in one poll", []*observation.SNMPObservation{
			routingObs(before, false, lan, wan), routingObs(after, false, wanAlt),
		}, []string{alertmodel.RuleRouteWithdrawn, alertmodel.RuleRouteChange}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, a := range scanRouting(t, tt.polls...) {
				got = append(got, a.Rule)
			}
			if !reflect.DeepEqual(got, tt.rules) {
				t.Errorf("rules = %v, want %v", got, tt.rules)
			}
		})
	}
}

func TestObservationScanOnce_RouteWithdrawnEvidence(t *testing.T) {
	t.Parallel()
	created := scanRouting(t,
		routingObs(at().Add(-time.Minute), false, lan, ecmpA, ecmpB, oddMask),
		routingObs(at(), false, lan),
	)
	if len(created) != 1 {
		t.Fatalf("alerts = %d, want 1", len(created))
	}
	a := created[0]
	if a.Severity != alertmodel.SeverityWarning || a.Type != alertmodel.TypeConnectivity || a.Source != "t-1" {
		t.Errorf("type/severity/source = %q/%q/%q", a.Type, a.Severity, a.Source)
	}
	if a.Title != "2 route(s) withdrawn on t-1" {
		t.Errorf("title = %q", a.Title)
	}
	got := decodeEvidence[alertmodel.RouteWithdrawnEvidence](t, a)
	want := alertmodel.RouteWithdrawnEvidence{Withdrawn: 2, Routes: []alertmodel.RouteWithdrawal{
		{Prefix: "10.9.0.0/255.0.255.0", NextHops: []string{"192.0.2.1"}, Proto: 14},
		{Prefix: "198.51.100.0/24", NextHops: []string{"192.0.2.1", "192.0.2.2"}, Proto: 14},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("evidence = %+v, want %+v", got, want)
	}
}

func TestObservationScanOnce_RouteChangeEvidence(t *testing.T) {
	t.Parallel()
	created := scanRouting(t,
		routingObs(at().Add(-time.Minute), false, wan),
		routingObs(at(), false, wanAlt),
	)
	if len(created) != 1 {
		t.Fatalf("alerts = %d, want 1", len(created))
	}
	if created[0].Severity != alertmodel.SeverityInfo {
		t.Errorf("severity = %q, want info", created[0].Severity)
	}
	got := decodeEvidence[alertmodel.RouteChangeEvidence](t, created[0])
	want := alertmodel.RouteChangeEvidence{Changed: 1, Routes: []alertmodel.RouteNextHopChange{
		{Prefix: "203.0.113.0/24", From: []string{"192.0.2.1"}, To: []string{"192.0.2.9"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("evidence = %+v, want %+v", got, want)
	}
}

// A dropped BGP session withdraws thousands of prefixes at once: one alert
// counts all of them and lists a bounded few.
func TestObservationScanOnce_MassWithdrawalIsOneBoundedAlert(t *testing.T) {
	t.Parallel()
	const withdrawn = 300
	routes := make([]string, 0, withdrawn+1)
	routes = append(routes, lan)
	for i := range withdrawn {
		routes = append(routes, fmt.Sprintf("172.%d.%d.0 255.255.255.0 192.0.2.7", 16+i/256, i%256))
	}
	created := scanRouting(t,
		routingObs(at().Add(-time.Minute), false, routes...),
		routingObs(at(), false, lan),
	)
	if len(created) != 1 {
		t.Fatalf("alerts = %d, want 1", len(created))
	}
	got := decodeEvidence[alertmodel.RouteWithdrawnEvidence](t, created[0])
	if got.Withdrawn != withdrawn || len(got.Routes) != 50 {
		t.Errorf("withdrawn = %d, listed = %d; want %d, 50", got.Withdrawn, len(got.Routes), withdrawn)
	}
	if !strings.HasSuffix(created[0].Message, "and 295 more") {
		t.Errorf("message = %q", created[0].Message)
	}
}

// sinceObservations honours ListOptions.Since and Limit as the repository does,
// newest first.
type sinceObservations struct {
	rows []*observation.SNMPObservation
}

func (f *sinceObservations) List(
	_ context.Context, opts observation.ListOptions,
) ([]*observation.SNMPObservation, error) {
	var out []*observation.SNMPObservation
	for i := len(f.rows) - 1; i >= 0 && (opts.Limit == 0 || len(out) < opts.Limit); i-- {
		if row := f.rows[i]; row.Kind == opts.Kind && !row.ObservedAt.Before(opts.Since) {
			out = append(out, row)
		}
	}
	return out, nil
}

// After a restart the scan resumes past its high-water mark, so the only
// record of the table before it is what Start primed from history. A
// withdrawal in the first new poll must still alert.
func TestObservationStart_PrimesRoutingFromHistory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	previous := at().Add(-time.Minute)
	o := &sinceObservations{rows: []*observation.SNMPObservation{routingObs(previous, false, lan, wan)}}
	settings := newFakeSettings()
	highWater := previous.Add(time.Second).Format(time.RFC3339Nano)
	if err := settings.Set(ctx, "alerts.observation.high_water", highWater); err != nil {
		t.Fatal(err)
	}
	alerts := &fakeAlerts{}
	p, err := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: settings,
		Logger: silentLogger(), Now: at, Interval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err = p.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	o.rows = append(o.rows, routingObs(at(), false, lan))
	if err = p.ScanOnce(ctx); err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	if len(alerts.created) != 1 || alerts.created[0].Rule != alertmodel.RuleRouteWithdrawn {
		t.Fatalf("alerts = %+v, want one %s", alerts.created, alertmodel.RuleRouteWithdrawn)
	}
}
