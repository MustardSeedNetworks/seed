package narrative_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
)

func base() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC) }

func evidence(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The fixtures' Source is the polling target id; the narrative must name the
// device by the name passed in, which is what the operator knows it as.
func ifaceDown(t *testing.T, oper int) *alerts.Alert {
	t.Helper()
	return &alerts.Alert{
		ID: 1, Rule: alerts.RuleInterfaceDown, Source: "tgt-1", CreatedAt: base(),
		Metadata: evidence(t, alerts.InterfaceDownEvidence{IfIndex: 3, IfName: "Gi0/3", IfOperStatus: oper}),
	}
}

func bgpFlap(t *testing.T, id int64, state int, at time.Time, cause *int64) *alerts.Alert {
	t.Helper()
	return &alerts.Alert{
		ID: id, Rule: alerts.RuleBGPFlap, Source: "tgt-1", CreatedAt: at, RootCauseID: cause,
		Metadata: evidence(t, alerts.BGPPeerEvidence{RemoteAddr: "192.0.2.2", RemoteAS: 65001, State: state}),
	}
}

func TestExplainInterfaceDownNamesDeviceInterfaceEvidenceAndNextCheck(t *testing.T) {
	cause := ifaceDown(t, 2)
	id := cause.ID
	effects := []*alerts.Alert{
		bgpFlap(t, 2, 1, base().Add(41600*time.Millisecond), &id),
		// An effect from a rule this narrative does not describe is left out
		// rather than misreported as a BGP peer.
		{ID: 3, Rule: "db.7", RootCauseID: &id, Metadata: `{"message":"x"}`},
	}

	n, ok := narrative.Explain(cause, effects, "core-sw1")
	if !ok {
		t.Fatal("Explain = false, want a narrative")
	}
	if n.Summary.Key != "api.narrative.interfaceDown.summaryWithBgp" {
		t.Errorf("summary key = %q", n.Summary.Key)
	}
	wantSummary := map[string]any{"interface": "Gi0/3", "device": "core-sw1", "peers": 1}
	assertData(t, "summary", n.Summary.Data, wantSummary)

	if len(n.Evidence) != 2 {
		t.Fatalf("evidence = %d messages, want oper status + 1 peer: %+v", len(n.Evidence), n.Evidence)
	}
	assertData(t, "oper evidence", n.Evidence[0].Data, map[string]any{
		"status": "down", "ifIndex": uint32(3), "time": "2026-10-04T12:00:00Z",
	})
	assertData(t, "peer evidence", n.Evidence[1].Data, map[string]any{
		"peer": "192.0.2.2", "as": uint32(65001), "seconds": int64(42),
	})
	if n.NextCheck.Key != "api.narrative.interfaceDown.next.down" {
		t.Errorf("next check key = %q", n.NextCheck.Key)
	}
}

func TestExplainInterfaceDownNextCheckFollowsOperStatus(t *testing.T) {
	tests := []struct {
		oper int
		want string
	}{
		{2, "down"},
		{3, "testing"},
		{4, "unknown"},
		{5, "dormant"},
		{6, "notPresent"},
		{7, "lowerLayerDown"},
		{99, "unknown"},
	}
	for _, tt := range tests {
		n, ok := narrative.Explain(ifaceDown(t, tt.oper), nil, "core-sw1")
		if !ok {
			t.Fatalf("oper %d: Explain = false", tt.oper)
		}
		if got := n.NextCheck.Key; got != "api.narrative.interfaceDown.next."+tt.want {
			t.Errorf("oper %d: next check = %q, want suffix %q", tt.oper, got, tt.want)
		}
		if n.Summary.Key != "api.narrative.interfaceDown.summary" {
			t.Errorf("oper %d: summary = %q, want the no-BGP summary", tt.oper, n.Summary.Key)
		}
	}
}

func TestExplainBGPFlapWithoutCauseNextCheckFollowsState(t *testing.T) {
	tests := []struct {
		state int
		want  string
	}{
		{1, "idle"},
		{2, "connect"},
		{3, "active"},
		{4, "openSent"},
		{5, "openConfirm"},
		{0, "unknown"},
	}
	for _, tt := range tests {
		n, ok := narrative.Explain(bgpFlap(t, 1, tt.state, base(), nil), nil, "core-sw1")
		if !ok {
			t.Fatalf("state %d: Explain = false", tt.state)
		}
		if got := n.NextCheck.Key; got != "api.narrative.bgpFlap.next."+tt.want {
			t.Errorf("state %d: next check = %q", tt.state, got)
		}
		assertData(t, "bgp summary", n.Summary.Data, map[string]any{
			"peer": "192.0.2.2", "as": uint32(65001), "device": "core-sw1", "minutes": 5,
		})
	}
}

func TestExplainStorage(t *testing.T) {
	for _, rule := range []string{alerts.RuleStorageHigh, alerts.RuleStorageCritical} {
		a := &alerts.Alert{
			ID: 1, Rule: rule, Source: "tgt-9", CreatedAt: base(),
			Metadata: evidence(t, alerts.StorageEvidence{
				Index: 31, Description: "/var", SizeBytes: 1000, UsedBytes: 962, UsedPercent: 96.2,
			}),
		}
		n, ok := narrative.Explain(a, nil, "nas1")
		if !ok {
			t.Fatalf("%s: Explain = false", rule)
		}
		assertData(t, rule+" summary", n.Summary.Data, map[string]any{
			"filesystem": "/var", "device": "nas1", "percent": "96.2",
		})
	}
}

func TestExplainDeclines(t *testing.T) {
	cause := int64(9)
	tests := []struct {
		name  string
		alert *alerts.Alert
	}{
		{"explained by another alert", bgpFlap(t, 1, 3, base(), &cause)},
		{"operator rule", &alerts.Alert{ID: 1, Rule: "db.4", Metadata: `{"message":"x"}`}},
		{"unreadable evidence", &alerts.Alert{ID: 1, Rule: alerts.RuleInterfaceDown, Metadata: "{"}},
	}
	for _, tt := range tests {
		if n, ok := narrative.Explain(tt.alert, nil, "core-sw1"); ok {
			t.Errorf("%s: Explain = %+v, want none", tt.name, n)
		}
	}
}

// Every key a narrative can produce must resolve, fully interpolated, in every
// shipped language: an unresolved key renders as the key itself.
func TestEveryNarrativeRendersInEveryLanguage(t *testing.T) {
	cause := ifaceDown(t, 2)
	id := cause.ID
	var all []narrative.Narrative
	add := func(a *alerts.Alert, effects ...*alerts.Alert) {
		n, ok := narrative.Explain(a, effects, a.Source)
		if !ok {
			t.Fatalf("Explain(%s) = false", a.Rule)
		}
		all = append(all, n)
	}
	add(cause, bgpFlap(t, 2, 1, base().Add(time.Second), &id))
	for _, oper := range []int{2, 3, 4, 5, 6, 7} {
		add(ifaceDown(t, oper))
	}
	for _, state := range []int{0, 1, 2, 3, 4, 5} {
		add(bgpFlap(t, 1, state, base(), nil))
	}
	add(&alerts.Alert{
		ID: 1, Rule: alerts.RuleStorageHigh, Source: "nas1",
		Metadata: evidence(t, alerts.StorageEvidence{Description: "/var", UsedPercent: 90}),
	})

	for _, lang := range i18n.GetSupportedLanguages() {
		l := i18n.NewLocalizer(lang)
		for _, n := range all {
			msgs := append([]narrative.Message{n.Summary, n.NextCheck}, n.Evidence...)
			for _, m := range msgs {
				got := l.TWithData(m.Key, m.Data)
				if got == m.Key || strings.Contains(got, "{{") {
					t.Errorf("%s: %s rendered as %q", lang, m.Key, got)
				}
			}
		}
	}
}

func assertData(t *testing.T, what string, got, want map[string]any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s data = %v, want %v", what, got, want)
		return
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s data[%q] = %#v, want %#v", what, k, got[k], v)
		}
	}
}
