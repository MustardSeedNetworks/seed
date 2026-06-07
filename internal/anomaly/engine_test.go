package anomaly_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
)

func testCatalog(t *testing.T) *anomaly.Catalog {
	t.Helper()
	c, err := anomaly.NewCatalog(
		anomaly.Def{
			ID: "open-ssid", Category: anomaly.CategorySecurity,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"IEEE 802.11i-2004"},
			Title:           "Open network", Description: "No encryption.",
			Impact: "Traffic is cleartext.", Recommendation: "Enable WPA2/WPA3.",
			FollowUps: []anomaly.FollowUp{
				{
					Kind:       anomaly.FollowUpAuto,
					Label:      "Deauth-response test",
					Action:     "pmf-probe",
					Capability: "wifi-active-diag",
				},
				{Kind: anomaly.FollowUpAuto, Label: "Always-on check", Action: "recheck"},
				{Kind: anomaly.FollowUpPrompt, Label: "Move closer", Action: "rescan"},
			},
		},
		anomaly.Def{
			ID: "co-channel", Category: anomaly.CategoryRF,
			DefaultSeverity: anomaly.SeverityInfo,
			Title:           "Co-channel contention", Description: "Many BSSes share a channel.",
			Recommendation: "Re-plan channels.",
		},
	)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return c
}

func bssid(id string) anomaly.SubjectRef {
	return anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: id}
}

func TestCatalogValidation(t *testing.T) {
	good := anomaly.Def{
		ID: "x", Category: anomaly.CategorySecurity, DefaultSeverity: anomaly.SeverityInfo,
		Title: "t", Description: "d", Recommendation: "r",
	}
	if _, err := anomaly.NewCatalog(good); err != nil {
		t.Fatalf("valid def rejected: %v", err)
	}

	mut := func(f func(*anomaly.Def)) []anomaly.Def {
		d := good
		f(&d)
		return []anomaly.Def{d}
	}
	cases := map[string][]anomaly.Def{
		"missing id":    mut(func(d *anomaly.Def) { d.ID = "" }),
		"missing title": mut(func(d *anomaly.Def) { d.Title = "" }),
		"missing desc":  mut(func(d *anomaly.Def) { d.Description = "" }),
		"missing rec":   mut(func(d *anomaly.Def) { d.Recommendation = "" }),
		"bad category":  mut(func(d *anomaly.Def) { d.Category = "bogus" }),
		"bad severity":  mut(func(d *anomaly.Def) { d.DefaultSeverity = "huge" }),
		"bad followup": mut(
			func(d *anomaly.Def) { d.FollowUps = []anomaly.FollowUp{{Kind: "weird", Label: "x"}} },
		),
		"followup no label": mut(func(d *anomaly.Def) {
			d.FollowUps = []anomaly.FollowUp{{Kind: anomaly.FollowUpPrompt}}
		}),
	}
	for name, defs := range cases {
		if _, err := anomaly.NewCatalog(defs...); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}

	if _, err := anomaly.NewCatalog(good, good); err == nil {
		t.Error("duplicate id: expected error, got nil")
	}
}

func TestObserveRejectsUnknownDef(t *testing.T) {
	e := anomaly.NewEngine(testCatalog(t))
	err := e.Observe(anomaly.Detection{DefKey: "nope", Subject: bssid("aa")}, time.Unix(0, 0))
	if err == nil {
		t.Fatal("expected error for unknown def")
	}
	if e.Len() != 0 {
		t.Errorf("unknown def created an instance: Len=%d", e.Len())
	}
}

func TestCoalesceByTypeAndSubject(t *testing.T) {
	e := anomaly.NewEngine(testCatalog(t))
	t0 := time.Unix(1000, 0)
	t1 := time.Unix(2000, 0)

	must(t, e.Observe(anomaly.Detection{DefKey: "open-ssid", Subject: bssid("aa:bb")}, t0))
	must(t, e.Observe(anomaly.Detection{DefKey: "open-ssid", Subject: bssid("aa:bb")}, t1))
	// Different subject -> distinct instance.
	must(t, e.Observe(anomaly.Detection{DefKey: "open-ssid", Subject: bssid("cc:dd")}, t1))

	if e.Len() != 2 {
		t.Fatalf("Len = %d, want 2 (coalesced + distinct)", e.Len())
	}
	snap := e.Snapshot()
	var coalesced anomaly.Anomaly
	for _, a := range snap {
		if a.Subject.ID == "aa:bb" {
			coalesced = a
		}
	}
	if coalesced.Count != 2 {
		t.Errorf("count = %d, want 2", coalesced.Count)
	}
	if !coalesced.FirstSeen.Equal(t0) {
		t.Errorf("firstSeen = %v, want %v (earliest preserved)", coalesced.FirstSeen, t0)
	}
	if !coalesced.LastSeen.Equal(t1) {
		t.Errorf("lastSeen = %v, want %v", coalesced.LastSeen, t1)
	}
}

func TestSeverityEscalationOnRecurrence(t *testing.T) {
	e := anomaly.NewEngine(testCatalog(t), anomaly.WithEscalateAfter(3))
	at := time.Unix(0, 0)
	det := anomaly.Detection{DefKey: "open-ssid", Subject: bssid("aa")} // base = warning
	for range 2 {
		must(t, e.Observe(det, at))
	}
	if got := e.Snapshot()[0].Severity; got != anomaly.SeverityWarning {
		t.Fatalf("before threshold: severity = %q, want warning", got)
	}
	must(t, e.Observe(det, at)) // count now 3 == threshold
	if got := e.Snapshot()[0].Severity; got != anomaly.SeverityCritical {
		t.Errorf("after threshold: severity = %q, want critical (escalated)", got)
	}
}

func TestDetectionSeverityOverride(t *testing.T) {
	e := anomaly.NewEngine(testCatalog(t))
	// co-channel default is info; override to warning.
	must(t, e.Observe(anomaly.Detection{
		DefKey: "co-channel", Subject: anomaly.SubjectRef{Kind: anomaly.SubjectChannel, ID: "6"},
		Severity: anomaly.SeverityWarning,
	}, time.Unix(0, 0)))
	if got := e.Snapshot()[0].Severity; got != anomaly.SeverityWarning {
		t.Errorf("severity = %q, want warning (override)", got)
	}

	badSev := anomaly.Detection{DefKey: "co-channel", Subject: bssid("z"), Severity: "nope"}
	if err := e.Observe(badSev, time.Unix(0, 0)); err == nil {
		t.Error("invalid severity override: expected error")
	}
}

func TestPruneClearsStale(t *testing.T) {
	e := anomaly.NewEngine(testCatalog(t))
	old := time.Unix(100, 0)
	fresh := time.Unix(10000, 0)
	must(t, e.Observe(anomaly.Detection{DefKey: "open-ssid", Subject: bssid("old")}, old))
	must(t, e.Observe(anomaly.Detection{DefKey: "open-ssid", Subject: bssid("fresh")}, fresh))

	cleared := e.Prune(time.Unix(5000, 0))
	if cleared != 1 {
		t.Fatalf("Prune cleared %d, want 1", cleared)
	}
	snap := e.Snapshot()
	if len(snap) != 1 || snap[0].Subject.ID != "fresh" {
		t.Errorf("after prune: %+v, want only 'fresh'", snap)
	}
}

func TestSnapshotOrdering(t *testing.T) {
	e := anomaly.NewEngine(testCatalog(t))
	at := time.Unix(0, 0)
	must(
		t,
		e.Observe(
			anomaly.Detection{
				DefKey:  "co-channel",
				Subject: anomaly.SubjectRef{Kind: anomaly.SubjectChannel, ID: "11"},
			},
			at,
		),
	) // info
	must(
		t,
		e.Observe(anomaly.Detection{DefKey: "open-ssid", Subject: bssid("bb")}, at),
	) // warning
	must(
		t,
		e.Observe(anomaly.Detection{DefKey: "open-ssid", Subject: bssid("aa")}, at),
	) // warning

	snap := e.Snapshot()
	// Most urgent first: both warnings before the info; warnings tie-break by subject id.
	if snap[0].Severity != anomaly.SeverityWarning || snap[0].Subject.ID != "aa" {
		t.Errorf("snap[0] = %q/%s, want warning/aa", snap[0].Severity, snap[0].Subject.ID)
	}
	if snap[1].Subject.ID != "bb" {
		t.Errorf("snap[1] subject = %s, want bb", snap[1].Subject.ID)
	}
	if snap[2].Severity != anomaly.SeverityInfo {
		t.Errorf("snap[2] severity = %q, want info (least urgent last)", snap[2].Severity)
	}
}

func TestFollowUpCapabilityDegradation(t *testing.T) {
	at := time.Unix(0, 0)
	det := anomaly.Detection{DefKey: "open-ssid", Subject: bssid("aa")}

	// Without the capability: the gated auto follow-up degrades to a prompt; the
	// always-on auto stays auto; the prompt stays prompt.
	e := anomaly.NewEngine(testCatalog(t))
	must(t, e.Observe(det, at))
	fus := e.Snapshot()[0].FollowUps
	if len(fus) != 3 {
		t.Fatalf("follow-ups = %d, want 3", len(fus))
	}
	if fus[0].Kind != anomaly.FollowUpPrompt {
		t.Errorf("gated auto without capability: kind = %q, want prompt (degraded)", fus[0].Kind)
	}
	if fus[1].Kind != anomaly.FollowUpAuto {
		t.Errorf("ungated auto: kind = %q, want auto", fus[1].Kind)
	}

	// With the capability registered: the gated auto stays auto.
	e2 := anomaly.NewEngine(testCatalog(t), anomaly.WithCapabilities("wifi-active-diag"))
	must(t, e2.Observe(det, at))
	if got := e2.Snapshot()[0].FollowUps[0].Kind; got != anomaly.FollowUpAuto {
		t.Errorf("gated auto with capability: kind = %q, want auto", got)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
