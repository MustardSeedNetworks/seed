package reporting_test

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/reporting/store"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// seedSummaryStores writes, through each store's own writer, rows on both
// sides of [start, end): at start itself and half a second into it (in), and
// a second before start and at end (out). It returns the polling target ID.
func seedSummaryStores(t *testing.T, db *database.DB, start, end time.Time) string {
	t.Helper()
	ctx := t.Context()
	in := start.Add

	target := &polling.Target{ClientID: "default", Name: "core-sw1", IPAddress: "10.0.0.1"}
	require.NoError(t, db.PollingTargets().Create(ctx, target))

	topo := db.Topology()
	node := func(id, name, ip string, firstSeen time.Time) {
		_, err := topo.Upsert(ctx, &topology.Node{
			ID: id, IdentityHash: "hash-" + id, DisplayName: name, PrimaryIP: ip,
			FirstSeen: firstSeen, LastSeen: end.Add(time.Hour),
		})
		require.NoError(t, err)
	}
	node("n1", "core-sw1", "10.0.0.1", start.Add(-time.Hour))
	node("n2", "access-sw2", "10.0.0.2", in(500*time.Millisecond))
	node("n3", "access-sw3", "10.0.0.3", end)
	node("n4", "access-sw4", "10.0.0.4", start.Add(-time.Second))
	require.NoError(t, topo.UpsertTargetNode(ctx, "default", target.ID, "n1", end))
	require.NoError(t, topo.UpsertInterface(ctx, &topology.Interface{
		NodeID: "n1", IfIndex: 3, IfName: "Gi0/3", LastSeen: end,
	}))
	link := func(id, from, to string, firstSeen time.Time) {
		require.NoError(t, topo.UpsertLink(ctx, &topology.Link{
			ID: id, SourceNodeID: from, TargetNodeID: to, SourceInterface: "Gi0/3", TargetInterface: "Gi0/1",
			LinkType: "lldp", Status: "up", FirstSeen: firstSeen, LastSeen: end,
		}))
	}
	link("l12", "n1", "n2", in(2*time.Hour))
	link("l13", "n1", "n3", end)

	rate := func(targetID string, ifIndex uint32, at time.Time, util *ifrate.Utilization, inErrors, inDiscards float64) ifrate.Rate {
		return ifrate.Rate{
			ClientID: "default", TargetID: targetID, IfIndex: ifIndex, At: at,
			Octets:   &ifrate.Octets{In: 1, Out: 1, Utilization: util},
			InErrors: inErrors, InDiscards: inDiscards,
		}
	}
	require.NoError(t, db.Metrics().RecordInterfaceRates(ctx, []ifrate.Rate{
		rate(target.ID, 3, start, &ifrate.Utilization{In: 40, Out: 10}, 2, 0),
		rate(target.ID, 3, in(time.Hour), &ifrate.Utilization{In: 80, Out: 20}, 4, 0),
		rate(target.ID, 3, end, &ifrate.Utilization{In: 99, Out: 99}, 100, 0),
		rate(target.ID, 3, start.Add(-time.Second), &ifrate.Utilization{In: 99, Out: 99}, 100, 0),
		rate(target.ID, 4, in(time.Hour), nil, 0, 1),
		rate("ghost", 1, in(time.Hour), &ifrate.Utilization{In: 5, Out: 1}, 0, 0),
	}))

	alert := func(severity string, resolved bool, at time.Time) {
		require.NoError(t, db.Alerts().Create(ctx, &alerts.Alert{
			Type: "connectivity", Severity: severity, Title: "t", Message: "m", Resolved: resolved, CreatedAt: at,
		}))
	}
	alert(alerts.SeverityCritical, false, start)
	alert(alerts.SeverityWarning, true, in(time.Hour))
	alert(alerts.SeverityWarning, false, in(2*time.Hour))
	alert(alerts.SeverityInfo, false, start.Add(-time.Second))
	alert(alerts.SeverityCritical, false, end)

	probes := db.Probes()
	for _, p := range []*database.Probe{
		{ID: "p1", ClientID: "default", Kind: "icmp", DisplayName: "gateway ping", Target: "10.0.0.1"},
		{ID: "p2", ClientID: "default", Kind: "dns", DisplayName: "resolver", Target: "10.0.0.53"},
	} {
		require.NoError(t, probes.CreateProbe(ctx, p))
	}
	result := func(probe, kind string, ok bool, latency float64, at time.Time) {
		require.NoError(t, probes.RecordResult(ctx, &database.ProbeResult{
			ProbeID: probe, ClientID: "default", Kind: kind, Success: ok, LatencyMs: latency, Timestamp: at,
		}))
	}
	result("p1", "icmp", true, 10, in(500*time.Millisecond))
	result("p1", "icmp", true, 20, in(time.Hour))
	result("p1", "icmp", false, 0, in(2*time.Hour))
	result("p1", "icmp", true, 1000, end)
	result("p2", "dns", true, 5, in(time.Hour))
	result("p2", "dns", false, 0, start.Add(-time.Second))

	return target.ID
}

func openSummaryDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(dbtest.Path(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestSummaryReadsCountOnlyTheWindow pins each store read to [start, end):
// a row at start counts, including one stored with fractional seconds, and
// rows a second before start or at end do not.
func TestSummaryReadsCountOnlyTheWindow(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	start := time.Date(2026, 10, 3, 6, 0, 0, 0, time.UTC)
	window := reporting.DateRange{Start: start, End: start.Add(24 * time.Hour)}
	targetID := seedSummaryStores(t, db, window.Start, window.End)
	repo := store.NewMetricsRepo(db)
	ctx := t.Context()

	health, err := repo.InterfaceHealth(ctx, window)
	require.NoError(t, err)
	// Ordered by target_id, so the unnamed "ghost" target sorts before "tgt-…".
	assert.Equal(t, []reporting.InterfaceHealth{
		{Target: "ghost", IfIndex: 1, PeakUtilization: 5, AvgInUtilization: 5, AvgOutUtilization: 1},
		{
			Target: "core-sw1", IfIndex: 3, IfName: "Gi0/3", PeakUtilization: 80,
			AvgInUtilization: 60, AvgOutUtilization: 15, InErrors: 3,
		},
		{Target: "core-sw1", IfIndex: 4, InDiscards: 1},
	}, health, "target %s", targetID)

	severities, open, err := repo.AlertSeverityCounts(ctx, window)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"critical": 1, "warning": 2}, severities)
	assert.Equal(t, 2, open)

	changes, err := repo.TopologyChanges(ctx, window)
	require.NoError(t, err)
	require.Len(t, changes.NewNodes, 1)
	assert.Equal(t, "access-sw2", changes.NewNodes[0].Name)
	assert.Equal(t, "10.0.0.2", changes.NewNodes[0].Address)
	require.Len(t, changes.NewLinks, 1)
	assert.Equal(t, reporting.TopologyLink{
		Source: "core-sw1", SourceInterface: "Gi0/3", Target: "access-sw2", TargetInterface: "Gi0/1",
		Type: "lldp", FirstSeen: start.Add(2 * time.Hour),
	}, changes.NewLinks[0])

	outcomes, err := repo.ProbeOutcomes(ctx, window)
	require.NoError(t, err)
	assert.Equal(t, []reporting.ProbeOutcome{
		{Name: "gateway ping", Kind: "icmp", Checks: 3, Failures: 1, AvgLatencyMs: 15},
		{Name: "resolver", Kind: "dns", Checks: 1, AvgLatencyMs: 5},
	}, outcomes)
}

// TestSummarizeRanksTheStores checks the shaping with no database: busiest
// leaves out interfaces with no known line rate and keeps ten, faulted keeps
// only those with an error or discard, unknown severities still count toward
// the total, and the least available probe comes first.
func TestSummarizeRanksTheStores(t *testing.T) {
	t.Parallel()
	var interfaces []reporting.InterfaceHealth
	for i := range 12 {
		interfaces = append(interfaces, reporting.InterfaceHealth{
			Target: "sw", IfIndex: uint32(i), PeakUtilization: float64(i * 5),
		})
	}
	interfaces[1].OutDiscards = 0.5
	interfaces[7].InErrors = 2

	as := reporting.NewAggregatorService(testConfig(), &fakeMetricsRepo{
		interfaces: interfaces,
		alerts:     map[string]int{"critical": 1, "error": 2, "warning": 3, "info": 4, "debug": 5},
		openAlerts: 6,
		probes: []reporting.ProbeOutcome{
			{Name: "a", Checks: 10, Failures: 0},
			{Name: "b", Checks: 10, Failures: 5},
			{Name: "c", Checks: 4, Failures: 1},
		},
	})
	sum, err := as.Summarize(t.Context(), reporting.DateRange{})
	require.NoError(t, err)

	assert.Equal(t, 12, sum.Interfaces.Rated)
	require.Len(t, sum.Interfaces.Busiest, 10)
	assert.InDelta(t, 55.0, sum.Interfaces.Busiest[0].PeakUtilization, 0)
	assert.InDelta(t, 10.0, sum.Interfaces.Busiest[9].PeakUtilization, 0)
	assert.Equal(t, 2, sum.Interfaces.Faulty)
	assert.Equal(t, []uint32{7, 1}, []uint32{sum.Interfaces.Faulted[0].IfIndex, sum.Interfaces.Faulted[1].IfIndex})

	assert.Equal(t, reporting.AlertVolume{Total: 15, Critical: 1, Error: 2, Warning: 3, Info: 4, Open: 6}, sum.Alerts)

	assert.Equal(t, reporting.ProbeSummary{Probes: 3, Checks: 24, Failures: 6, Outcomes: []reporting.ProbeOutcome{
		{Name: "b", Checks: 10, Failures: 5},
		{Name: "c", Checks: 4, Failures: 1},
		{Name: "a", Checks: 10, Failures: 0},
	}}, sum.Probes)
}

// TestScheduledDailySummaryMatchesTheStores is P-B6's acceptance: a daily
// schedule for the summary template fires, and the PDF it produces carries the
// numbers the stores hold for the window recorded on the report.
func TestScheduledDailySummaryMatchesTheStores(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	now := time.Now().UTC()
	seedSummaryStores(t, db, now.Add(-20*time.Hour), now.Add(-2*time.Hour))

	cfg := testConfig()
	templates := reporting.NewTemplateService(cfg)
	require.NoError(t, templates.Load())
	metrics := store.NewMetricsRepo(db)
	aggregator := reporting.NewAggregatorService(cfg, metrics)
	gen := reporting.NewGeneratorService(cfg, store.NewReportRepo(db), store.NewExportRepo(db), templates, aggregator)
	gen.SetReportsPath(t.TempDir())

	due := now.Add(-time.Minute)
	require.NoError(t, store.NewScheduleRepo(db).SaveSchedule(t.Context(), &reporting.ScheduledReport{
		ID: "daily", Name: "Daily summary", Template: "summary", Format: reporting.FormatPDF, Enabled: true,
		Schedule: reporting.Schedule{Frequency: reporting.FrequencyDaily, Hour: 6, Timezone: "UTC"},
		NextRun:  &due, CreatedAt: now, UpdatedAt: now,
	}))
	scheduler := reporting.NewSchedulerService(cfg, store.NewScheduleRepo(db), gen,
		reporting.WithTickInterval(5*time.Millisecond))
	require.NoError(t, scheduler.Load(t.Context()))
	stop := runUntilCancelled(t, scheduler.Run)

	var report reporting.Report
	eventually(t, func() bool {
		reports, err := gen.ListReports(context.Background())
		if err != nil || len(reports) == 0 || reports[0].Status == reporting.StatusGenerating ||
			reports[0].Status == reporting.StatusPending {
			return false
		}
		report = reports[0]
		return true
	})
	stop()
	require.Equal(t, reporting.StatusComplete, report.Status, report.Error)
	require.Equal(t, reporting.ReportTypeSummary, report.Type)
	window := *report.Parameters.DateRange
	assert.Equal(t, 24*time.Hour, window.End.Sub(window.Start))
	assert.WithinDuration(t, now, window.End, 2*time.Minute)

	want, err := aggregator.Summarize(t.Context(), window)
	require.NoError(t, err)
	require.NotZero(t, want.Interfaces.Rated, "the seed must land in the window")
	require.NotEmpty(t, want.Topology.NewNodes)
	require.NotZero(t, want.Alerts.Total)
	require.NotZero(t, want.Probes.Checks)

	text := pdfText(t, report.FilePath)
	follows := func(label, value string) {
		t.Helper()
		for i := range len(text) - 1 {
			if text[i] == label && text[i+1] == value {
				return
			}
		}
		t.Errorf("PDF has no %q followed by %q", label, value)
	}
	follows("Window:", window.Start.Format("2006-01-02 15:04 UTC")+" to "+window.End.Format("2006-01-02 15:04 UTC"))
	follows("Interfaces rated:", strconv.Itoa(want.Interfaces.Rated))
	follows("Interfaces with errors:", strconv.Itoa(want.Interfaces.Faulty))
	follows("Alerts raised:", fmt.Sprintf("%d (%d still open)", want.Alerts.Total, want.Alerts.Open))
	follows("New devices:", strconv.Itoa(len(want.Topology.NewNodes)))
	follows("New links:", strconv.Itoa(len(want.Topology.NewLinks)))
	follows("Probe checks failed:", fmt.Sprintf("%d of %d", want.Probes.Failures, want.Probes.Checks))
	follows("Critical:", strconv.Itoa(want.Alerts.Critical))
	follows("Warning:", strconv.Itoa(want.Alerts.Warning))
	busiest := want.Interfaces.Busiest[0]
	follows(busiest.Target+" "+busiest.IfName, fmt.Sprintf("%.1f%%", busiest.PeakUtilization))
	worst := want.Probes.Outcomes[0]
	follows(worst.Name, worst.Kind)
	follows(want.Topology.NewNodes[0].Name, want.Topology.NewNodes[0].Address)
}

// TestSummaryJSONCarriesTheSummary: the JSON format serves the same summary
// as the PDF, for the window recorded on the report.
func TestSummaryJSONCarriesTheSummary(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	now := time.Now().UTC()
	seedSummaryStores(t, db, now.Add(-20*time.Hour), now.Add(-2*time.Hour))

	cfg := testConfig()
	templates := reporting.NewTemplateService(cfg)
	require.NoError(t, templates.Load())
	aggregator := reporting.NewAggregatorService(cfg, store.NewMetricsRepo(db))
	gen := reporting.NewGeneratorService(cfg, store.NewReportRepo(db), store.NewExportRepo(db), templates, aggregator)
	gen.SetReportsPath(t.TempDir())

	created, err := gen.GenerateFromTemplate(t.Context(), "summary", reporting.FormatJSON, nil)
	require.NoError(t, err)
	var report *reporting.Report
	eventually(t, func() bool {
		report, err = gen.GetReport(context.Background(), created.ID)
		return err == nil && report.Status != reporting.StatusPending && report.Status != reporting.StatusGenerating
	})
	require.Equal(t, reporting.StatusComplete, report.Status, report.Error)

	want, err := aggregator.Summarize(t.Context(), *report.Parameters.DateRange)
	require.NoError(t, err)
	raw, err := os.ReadFile(report.FilePath)
	require.NoError(t, err)
	var got struct {
		Summary reporting.NetworkSummary `json:"summary"`
	}
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, want.Interfaces, got.Summary.Interfaces)
	assert.Equal(t, want.Alerts, got.Summary.Alerts)
	assert.Equal(t, want.Probes, got.Summary.Probes)
	assert.Len(t, got.Summary.Topology.NewNodes, len(want.Topology.NewNodes))
	assert.True(t, want.Window.End.Equal(got.Summary.Window.End))
}

// TestSummaryRejectsFormatsItDoesNotRender: the template offers PDF and JSON
// only, and an ad hoc request for another format fails rather than render the
// generic aggregate under the summary's name.
func TestSummaryRejectsFormatsItDoesNotRender(t *testing.T) {
	t.Parallel()
	db := openSummaryDB(t)
	cfg := testConfig()
	templates := reporting.NewTemplateService(cfg)
	require.NoError(t, templates.Load())
	gen := reporting.NewGeneratorService(cfg, store.NewReportRepo(db), store.NewExportRepo(db), templates,
		reporting.NewAggregatorService(cfg, store.NewMetricsRepo(db)))
	gen.SetReportsPath(t.TempDir())

	_, err := gen.GenerateFromTemplate(t.Context(), "summary", reporting.FormatHTML, nil)
	require.Error(t, err)

	created, err := gen.Generate(t.Context(), reporting.ReportTypeSummary, reporting.FormatCSV, nil)
	require.NoError(t, err)
	eventually(t, func() bool {
		r, getErr := gen.GetReport(context.Background(), created.ID)
		return getErr == nil && r.Status == reporting.StatusFailed &&
			strings.Contains(r.Error, "unsupported format for a summary")
	})
}

var (
	pdfStream = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	pdfShow   = regexp.MustCompile(`\(((?:\\.|[^\\)])*)\) ?Tj`)
)

// pdfText returns the strings the PDF at path shows, in drawing order.
func pdfText(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	unescape := strings.NewReplacer(`\(`, "(", `\)`, ")", `\\`, `\`)

	var text []string
	for _, m := range pdfStream.FindAllSubmatch(raw, -1) {
		content := m[1]
		if r, zErr := zlib.NewReader(bytes.NewReader(content)); zErr == nil {
			if inflated, readErr := io.ReadAll(r); readErr == nil {
				content = inflated
			}
		}
		for _, show := range pdfShow.FindAllSubmatch(content, -1) {
			text = append(text, unescape.Replace(string(show[1])))
		}
	}
	require.NotEmpty(t, text, "no text in %s", path)
	return text
}
