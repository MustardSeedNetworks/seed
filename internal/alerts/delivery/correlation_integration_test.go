package delivery_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// The slice's acceptance, in the same shape slice 1 used: two real conditions
// on one device, through the real observation pipeline and the production
// decorator stack, and the receiver is told which one explains the other.
// Only the SNMP observations and the alert store are fakes.

func ifTableObservation(observed time.Time, target string, ifOper int) *observation.SNMPObservation {
	payload, err := json.Marshal(map[string]any{"Rows": []map[string]any{
		{"IfIndex": 5, "IfName": "eth0", "IfAdmin": 1, "IfOper": ifOper},
	}})
	if err != nil {
		panic(err)
	}
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: target, Kind: "if_table",
		ObservedAt: observed, PayloadJSON: string(payload),
	}
}

func bgpObservation(observed time.Time, target string, state int) *observation.SNMPObservation {
	payload, err := json.Marshal(map[string]any{"Peers": []map[string]any{
		{"RemoteAddr": "10.0.0.2", "State": state, "RemoteAS": 65001},
	}})
	if err != nil {
		panic(err)
	}
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: target, Kind: "bgp4_mib",
		ObservedAt: observed, PayloadJSON: string(payload),
	}
}

func TestBGPFlapReachesTheReceiverNamingTheInterfaceThatCausedIt(t *testing.T) {
	const (
		bgpEstablished = 6
		bgpIdle        = 1
		ifUp           = 1
		ifDown         = 2
	)

	got := make(chan received, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Alert alerts.Alert `json:"alert"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode envelope: %v", err)
		}
		got <- received{alert: payload.Alert}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	notifier, err := delivery.New(delivery.Config{
		URL: srv.URL, Secret: signingKey, Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	notifier.Start()
	defer notifier.Stop(context.Background())

	now := func() time.Time { return time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC) }
	store := &recordingStore{}

	// The production stack: correlate, then store, then deliver — the same
	// order server.alertStore wires.
	writer := delivery.WrapWriter(
		correlation.WrapWriter(store, correlation.Config{Now: now}),
		notifier,
	)

	p, err := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: &staticObservations{rows: []*observation.SNMPObservation{
			// The interface and the session are both healthy, then both fail:
			// the link drops and the session it carried drops with it.
			ifTableObservation(now().Add(-time.Minute), "t-1", ifUp),
			ifTableObservation(now(), "t-1", ifDown),
			bgpObservation(now().Add(-time.Minute), "t-1", bgpEstablished),
			bgpObservation(now(), "t-1", bgpIdle),
		}},
		Alerts:   writer,
		Settings: &memorySettings{values: map[string]string{}},
		Logger:   slog.New(slog.DiscardHandler),
		Now:      now,
	})
	if err != nil {
		t.Fatalf("NewObservationPipeline: %v", err)
	}
	if scanErr := p.ScanOnce(context.Background()); scanErr != nil {
		t.Fatalf("ScanOnce: %v", scanErr)
	}

	store.mu.Lock()
	created := append([]*alerts.Alert(nil), store.created...)
	store.mu.Unlock()
	if len(created) != 2 {
		t.Fatalf("pipeline stored %d alerts, want 2 (iface.down and bgp.flap)", len(created))
	}

	cause, effect := byRule(created, "iface.down"), byRule(created, "bgp.flap")
	if cause == nil || effect == nil {
		t.Fatalf("want one iface.down and one bgp.flap, got %q and %q",
			created[0].Rule, created[1].Rule)
	}
	if cause.ID == 0 {
		t.Fatal("the store assigned no id to the cause; a zero rootCauseId would " +
			"compare equal to an unset one and prove nothing")
	}
	if effect.RootCauseID == nil {
		t.Fatal("the BGP flap named no cause; want the interface-down alert")
	}
	if *effect.RootCauseID != cause.ID {
		t.Errorf("root cause = %d, want %d (the iface.down alert)", *effect.RootCauseID, cause.ID)
	}

	// Both alerts still reach the receiver — correlation annotates, it never
	// swallows the symptom.
	seen := collectByRule(t, got, 2)
	deliveredEffect, ok := seen["bgp.flap"]
	if !ok {
		t.Fatal("the bgp.flap alert never reached the receiver")
	}
	if deliveredEffect.RootCauseID == nil || *deliveredEffect.RootCauseID != cause.ID {
		t.Errorf("delivered rootCauseId = %v, want %d — the receiver was not told the cause",
			deliveredEffect.RootCauseID, cause.ID)
	}
	if _, sawSymptom := seen["iface.down"]; !sawSymptom {
		t.Error("the iface.down symptom did not reach the receiver; correlation must not swallow it")
	}
	t.Logf("receiver got bgp.flap %q with rootCauseId=%d (the iface.down alert %q)",
		deliveredEffect.Title, *deliveredEffect.RootCauseID, cause.Title)
}

// received is one alert as the httptest receiver saw it.
type received struct{ alert alerts.Alert }

// byRule picks the alert a given pipeline rule raised.
func byRule(list []*alerts.Alert, rule string) *alerts.Alert {
	for _, a := range list {
		if a.Rule == rule {
			return a
		}
	}
	return nil
}

// collectByRule drains want alerts from the receiver channel, keyed by the
// rule that raised them, and fails if they do not all arrive.
func collectByRule(t *testing.T, ch <-chan received, want int) map[string]alerts.Alert {
	t.Helper()
	seen := map[string]alerts.Alert{}
	deadline := time.After(5 * time.Second)
	for len(seen) < want {
		select {
		case d := <-ch:
			seen[d.alert.Rule] = d.alert
		case <-deadline:
			t.Fatalf("only %d of %d alerts reached the receiver: %v", len(seen), want, seen)
		}
	}
	return seen
}
