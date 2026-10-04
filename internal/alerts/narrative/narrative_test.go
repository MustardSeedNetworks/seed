package narrative_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
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

	n, ok := narrative.Explain(narrative.Cluster{Cause: cause, Effects: effects, Device: "core-sw1"})
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
		n, ok := narrative.Explain(narrative.Cluster{Cause: ifaceDown(t, tt.oper), Device: "core-sw1"})
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

// The counters sit between the cause's oper status and its effects, and
// state only what moved: an idle counter is left out, not reported as zero.
func TestExplainInterfaceDownCitesCountersBeforeTheDrop(t *testing.T) {
	cause := ifaceDown(t, 2)
	id := cause.ID
	n, ok := narrative.Explain(narrative.Cluster{
		Cause:    cause,
		Effects:  []*alerts.Alert{bgpFlap(t, 2, 1, base().Add(time.Second), &id)},
		Device:   "core-sw1",
		Counters: ifrate.ErrorPeaks{Polls: 4, InErrors: 1.0 / 60, OutDiscards: 1234.5},
	})
	if !ok {
		t.Fatal("Explain = false")
	}
	keys := make([]string, len(n.Evidence))
	for i, m := range n.Evidence {
		keys[i] = strings.TrimPrefix(m.Key, "api.narrative.interfaceDown.evidence.")
	}
	if want := []string{"operStatus", "counterPeak", "counterPeak", "bgpPeer"}; !slices.Equal(keys, want) {
		t.Fatalf("evidence = %v, want %v", keys, want)
	}
	assertData(t, "in errors", n.Evidence[1].Data, map[string]any{
		"counter": "ifInErrors", "interface": "Gi0/3", "rate": "0.0167", "minutes": 15,
	})
	assertData(t, "out discards", n.Evidence[2].Data, map[string]any{
		"counter": "ifOutDiscards", "interface": "Gi0/3", "rate": "1234", "minutes": 15,
	})
	if n.NextCheck.Key != "api.narrative.interfaceDown.next.downAfterErrors" {
		t.Errorf("next check = %q, want the physical-layer check", n.NextCheck.Key)
	}
}

func TestExplainInterfaceDownCounters(t *testing.T) {
	tests := []struct {
		name     string
		oper     int
		counters ifrate.ErrorPeaks
		evidence []string
		next     string
	}{
		{"no rated poll says nothing", 2, ifrate.ErrorPeaks{}, []string{"operStatus"}, "down"},
		{"clean before the drop", 2, ifrate.ErrorPeaks{Polls: 5}, []string{"operStatus", "countersClean"}, "down"},
		{
			"output errors",
			2,
			ifrate.ErrorPeaks{Polls: 5, OutErrors: 3},
			[]string{"operStatus", "counterPeak"},
			"downAfterErrors",
		},
		// Discards are congestion, not a failing link.
		{"discards only", 2, ifrate.ErrorPeaks{Polls: 5, InDiscards: 9}, []string{"operStatus", "counterPeak"}, "down"},
		// A missing module is the check whatever it counted before.
		{
			"errors then not present",
			6,
			ifrate.ErrorPeaks{Polls: 5, InErrors: 2},
			[]string{"operStatus", "counterPeak"},
			"notPresent",
		},
	}
	for _, tt := range tests {
		n, ok := narrative.Explain(
			narrative.Cluster{Cause: ifaceDown(t, tt.oper), Device: "core-sw1", Counters: tt.counters},
		)
		if !ok {
			t.Fatalf("%s: Explain = false", tt.name)
		}
		var keys []string
		for _, m := range n.Evidence {
			keys = append(keys, strings.TrimPrefix(m.Key, "api.narrative.interfaceDown.evidence."))
		}
		if !slices.Equal(keys, tt.evidence) {
			t.Errorf("%s: evidence = %v, want %v", tt.name, keys, tt.evidence)
		}
		if got := n.NextCheck.Key; got != "api.narrative.interfaceDown.next."+tt.next {
			t.Errorf("%s: next check = %q, want suffix %q", tt.name, got, tt.next)
		}
	}
	n, _ := narrative.Explain(narrative.Cluster{Cause: ifaceDown(t, 2), Counters: ifrate.ErrorPeaks{Polls: 5}})
	assertData(t, "clean", n.Evidence[1].Data, map[string]any{"interface": "Gi0/3", "minutes": 15, "polls": 5})
}

func TestCountersFor(t *testing.T) {
	cause := int64(9)
	explained := ifaceDown(t, 2)
	explained.RootCauseID = &cause
	tests := []struct {
		name  string
		alert *alerts.Alert
		want  bool
	}{
		{"interface down", ifaceDown(t, 2), true},
		{"explained by another alert", explained, false},
		{"bgp flap", bgpFlap(t, 1, 3, base(), nil), false},
		{"unreadable evidence", &alerts.Alert{Rule: alerts.RuleInterfaceDown, Metadata: "{"}, false},
	}
	for _, tt := range tests {
		ifIndex, ok := narrative.CountersFor(tt.alert)
		if ok != tt.want || (ok && ifIndex != 3) {
			t.Errorf("%s: CountersFor = %d, %v; want ifIndex 3 only when %v", tt.name, ifIndex, ok, tt.want)
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
		n, ok := narrative.Explain(narrative.Cluster{Cause: bgpFlap(t, 1, tt.state, base(), nil), Device: "core-sw1"})
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
		n, ok := narrative.Explain(narrative.Cluster{Cause: a, Device: "nas1"})
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
		if n, ok := narrative.Explain(narrative.Cluster{Cause: tt.alert, Device: "core-sw1"}); ok {
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
	add := func(c narrative.Cluster) {
		c.Device = c.Cause.Source
		n, ok := narrative.Explain(c)
		if !ok {
			t.Fatalf("Explain(%s) = false", c.Cause.Rule)
		}
		all = append(all, n)
	}
	add(narrative.Cluster{
		Cause: cause, Effects: []*alerts.Alert{bgpFlap(t, 2, 1, base().Add(time.Second), &id)},
		Counters: ifrate.ErrorPeaks{Polls: 3, InErrors: 0.5, OutErrors: 1, InDiscards: 2, OutDiscards: 4},
	})
	add(narrative.Cluster{Cause: ifaceDown(t, 2), Counters: ifrate.ErrorPeaks{Polls: 3}})
	for _, oper := range []int{2, 3, 4, 5, 6, 7} {
		add(narrative.Cluster{Cause: ifaceDown(t, oper)})
	}
	for _, state := range []int{0, 1, 2, 3, 4, 5} {
		add(narrative.Cluster{Cause: bgpFlap(t, 1, state, base(), nil)})
	}
	add(narrative.Cluster{Cause: &alerts.Alert{
		ID: 1, Rule: alerts.RuleStorageHigh, Source: "nas1",
		Metadata: evidence(t, alerts.StorageEvidence{Description: "/var", UsedPercent: 90}),
	}})

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
