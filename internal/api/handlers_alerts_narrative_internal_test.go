package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	alertcorrelation "github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
	alertpipeline "github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

type scriptedObservations []*observation.SNMPObservation

func (o scriptedObservations) List(
	_ context.Context, opts observation.ListOptions,
) ([]*observation.SNMPObservation, error) {
	var out []*observation.SNMPObservation
	for _, obs := range o {
		if obs.Kind == opts.Kind && obs.ObservedAt.After(opts.Since) {
			out = append(out, obs)
		}
	}
	return out, nil
}

func scriptedObs(t *testing.T, target, kind string, at time.Time, payload any) *observation.SNMPObservation {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return &observation.SNMPObservation{
		TargetID: target, Kind: kind, ObservedAt: at, PayloadJSON: string(b),
	}
}

type narrativeRow struct {
	ID          int64  `json:"id"`
	Rule        string `json:"rule"`
	RootCauseID *int64 `json:"rootCauseId"`
	Narrative   *struct {
		Summary   string   `json:"summary"`
		Evidence  []string `json:"evidence"`
		NextCheck string   `json:"nextCheck"`
	} `json:"narrative"`
}

// seedLinkFault scans a link fault on core-sw1's Gi0/3 that takes the BGP
// session to 192.0.2.2 with it, through the real pipeline into alerts.
func seedLinkFault(t *testing.T, db *database.DB, alerts alertdelivery.Writer) {
	t.Helper()
	t0 := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	// The pipeline stamps the target id on the alert; the narrative names the
	// device as the operator does.
	target := &polling.Target{
		ClientID: database.DefaultClientID, Name: "core-sw1", IPAddress: "192.0.2.1",
		SNMPVersion: "v2c", PollIntervalSec: 60, Enabled: true, CollectorChain: []string{"if_table"},
	}
	if err := db.PollingTargets().Create(context.Background(), target); err != nil {
		t.Fatal(err)
	}

	up := map[string]any{"Rows": []map[string]any{{"IfIndex": 3, "IfName": "Gi0/3", "IfAdmin": 1, "IfOper": 1}}}
	down := map[string]any{"Rows": []map[string]any{{"IfIndex": 3, "IfName": "Gi0/3", "IfAdmin": 1, "IfOper": 2}}}
	established := map[string]any{"Peers": []map[string]any{{"RemoteAddr": "192.0.2.2", "RemoteAS": 65001, "State": 6}}}
	active := map[string]any{"Peers": []map[string]any{{"RemoteAddr": "192.0.2.2", "RemoteAS": 65001, "State": 3}}}
	obs := scriptedObservations{
		scriptedObs(t, target.ID, "if_table", t0, up),
		scriptedObs(t, target.ID, "bgp4_mib", t0, established),
		scriptedObs(t, target.ID, "if_table", t0.Add(time.Minute), down),
		scriptedObs(t, target.ID, "bgp4_mib", t0.Add(time.Minute), active),
	}
	p, err := alertpipeline.NewObservationPipeline(alertpipeline.ObservationConfig{
		Observations: obs,
		Alerts:       alerts,
		Settings:     db.Settings(),
		Logger:       logging.GetLogger(),
		Suppressions: alertpipeline.NewDBSuppressionStore(db.AlertSuppressions()),
		ReplayDepth:  -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// A link fault that takes a BGP session with it, raised by the real pipeline
// through the real correlation writer into the real store, reads back from
// the inbox as one narrative on the interface alert naming the device, the
// interface, the evidence and the next check, with the BGP alert pointing at
// it.
func TestHandleAlerts_SeededLinkFaultReadsAsOneNarrative(t *testing.T) {
	s := newAlertsTestServer(t)
	db := s.db()
	seedLinkFault(t, db, alertcorrelation.WrapWriter(db.Alerts(), alertcorrelation.Config{}))

	byRule := listNarrativeRows(t, s, "")
	iface, bgp := byRule["iface.down"], byRule["bgp.flap"]
	if iface.ID == 0 || bgp.ID == 0 {
		t.Fatalf("want one iface.down and one bgp.flap alert, got %+v", byRule)
	}
	if bgp.RootCauseID == nil || *bgp.RootCauseID != iface.ID {
		t.Errorf("bgp.flap rootCauseId = %v, want %d", bgp.RootCauseID, iface.ID)
	}
	if bgp.Narrative != nil {
		t.Errorf("caused alert carries its own narrative: %+v", bgp.Narrative)
	}
	if iface.Narrative == nil {
		t.Fatal("iface.down alert has no narrative")
	}
	n := iface.Narrative
	for _, want := range []string{"Gi0/3", "core-sw1", "BGP"} {
		if !strings.Contains(n.Summary, want) {
			t.Errorf("summary %q does not name %q", n.Summary, want)
		}
	}
	if len(n.Evidence) != 2 || !strings.Contains(n.Evidence[0], "ifOperStatus went from up to down") ||
		!strings.Contains(n.Evidence[1], "192.0.2.2 (AS65001)") {
		t.Errorf("evidence = %q, want the oper status change then the BGP peer", n.Evidence)
	}
	if !strings.Contains(n.NextCheck, "physical link on Gi0/3") {
		t.Errorf("next check = %q", n.NextCheck)
	}

	es := listNarrativeRows(t, s, "es")["iface.down"].Narrative
	if es == nil || !strings.HasPrefix(es.Summary, "La interfaz Gi0/3 de core-sw1") {
		t.Errorf("Spanish reader got %+v", es)
	}
}

func listNarrativeRows(t *testing.T, s *Server, lang string) map[string]narrativeRow {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alerts", http.NoBody)
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	w := httptest.NewRecorder()
	i18n.Middleware()(http.HandlerFunc(s.handleAlerts)).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Alerts []narrativeRow `json:"alerts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	out := make(map[string]narrativeRow, len(resp.Alerts))
	for _, a := range resp.Alerts {
		out[a.Rule] = a
	}
	return out
}

// The same fault, written through the daemon's own alert store chain, reaches
// a webhook receiver with the interface alert's narrative in English, and the
// BGP alert it caused without one of its own.
func TestAlertStore_DeliversTheNarrativeToAReceiver(t *testing.T) {
	var mu sync.Mutex
	var bodies [][]byte
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer receiver.Close()

	s := newAlertsTestServer(t)
	db := s.db()
	store := s.alertStore(db, logging.GetLogger())
	defer s.alertDelivery.Stop(context.Background())
	s.alertDelivery.ApplyWebhook(alertdelivery.WebhookConfig{URL: receiver.URL, Secret: "test-signing-material"})
	seedLinkFault(t, db, store)

	type envelope struct {
		Alert struct {
			Rule string `json:"rule"`
		} `json:"alert"`
		Narrative *narrative.Text `json:"narrative"`
	}
	byRule := map[string]envelope{}
	deadline := time.Now().Add(5 * time.Second)
	for len(byRule) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		for _, b := range bodies {
			var e envelope
			if err := json.Unmarshal(b, &e); err != nil {
				t.Fatalf("payload: %v", err)
			}
			byRule[e.Alert.Rule] = e
		}
		mu.Unlock()
	}
	iface, bgp := byRule["iface.down"], byRule["bgp.flap"]
	if iface.Alert.Rule == "" || bgp.Alert.Rule == "" {
		t.Fatalf("receiver got %+v, want an iface.down and a bgp.flap delivery", byRule)
	}
	if bgp.Narrative != nil {
		t.Errorf("caused alert was delivered with its own narrative: %+v", bgp.Narrative)
	}
	if iface.Narrative == nil {
		t.Fatal("iface.down was delivered without its narrative")
	}
	if !strings.Contains(iface.Narrative.Summary, "Gi0/3") || !strings.Contains(iface.Narrative.Summary, "core-sw1") {
		t.Errorf("summary = %q, want it to name Gi0/3 on core-sw1", iface.Narrative.Summary)
	}
	if !strings.Contains(iface.Narrative.NextCheck, "physical link on Gi0/3") {
		t.Errorf("next check = %q", iface.Narrative.NextCheck)
	}
}
