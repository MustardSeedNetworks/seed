package reporting

// services_summary.go contains the daily network summary: the window it
// covers, the shaping of the store reads into a NetworkSummary, and its PDF
// and JSON renderers. Unlike the other report types it reads its window
// exactly and fails rather than print a zero for a store it could not read,
// because its numbers are meant to be checked against those stores.

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"
)

const (
	// summaryWindow is how far back a summary reaches from the minute it is
	// generated in.
	summaryWindow = 24 * time.Hour
	// summaryListLimit caps each ranked list, in the JSON and in the PDF.
	summaryListLimit = 10
	// summaryTimeLayout is how the PDF prints a time. The window is UTC.
	summaryTimeLayout = "2006-01-02 15:04 UTC"
	fullPercent       = 100
)

// NetworkSummary is the daily network summary: how the interfaces ran, what
// alerted, what appeared on the network and how the probes did, all over one
// window.
type NetworkSummary struct {
	Window     DateRange        `json:"window"`
	Interfaces InterfaceSummary `json:"interfaces"`
	Alerts     AlertVolume      `json:"alerts"`
	Topology   TopologyChanges  `json:"topology"`
	Probes     ProbeSummary     `json:"probes"`
}

// InterfaceHealth is one interface's rates over the window. Utilization is a
// percentage of line rate and is zero for an interface whose line rate is
// unknown; errors and discards are the mean of the per-second rates.
type InterfaceHealth struct {
	Target            string  `json:"target"`
	IfIndex           uint32  `json:"ifIndex"`
	IfName            string  `json:"ifName,omitempty"`
	PeakUtilization   float64 `json:"peakUtilization"`
	AvgInUtilization  float64 `json:"avgInUtilization"`
	AvgOutUtilization float64 `json:"avgOutUtilization"`
	InErrors          float64 `json:"inErrors"`
	OutErrors         float64 `json:"outErrors"`
	InDiscards        float64 `json:"inDiscards"`
	OutDiscards       float64 `json:"outDiscards"`
}

func (h *InterfaceHealth) faultRate() float64 {
	return h.InErrors + h.OutErrors + h.InDiscards + h.OutDiscards
}

func (h *InterfaceHealth) label() string {
	if h.IfName != "" {
		return h.Target + " " + h.IfName
	}
	return h.Target + " ifIndex " + strconv.FormatUint(uint64(h.IfIndex), 10)
}

// InterfaceSummary ranks the rated interfaces. Busiest holds those with a
// known line rate, by peak utilization; Faulted holds those that counted any
// error or discard, by their combined rate.
type InterfaceSummary struct {
	Rated   int               `json:"rated"`
	Busiest []InterfaceHealth `json:"busiest"`
	Faulty  int               `json:"faulty"`
	Faulted []InterfaceHealth `json:"faulted"`
}

// AlertVolume counts the alerts raised in the window by severity. Open is how
// many of them are still unresolved when the summary is generated.
type AlertVolume struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Error    int `json:"error"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
	Open     int `json:"open"`
}

// TopologyChanges lists the devices and links first seen in the window.
type TopologyChanges struct {
	NewNodes []TopologyNode `json:"newNodes"`
	NewLinks []TopologyLink `json:"newLinks"`
}

// TopologyNode is a device the topology first saw in the window.
type TopologyNode struct {
	Name      string    `json:"name"`
	Address   string    `json:"address,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
}

// TopologyLink is a link the topology first saw in the window.
type TopologyLink struct {
	Source          string    `json:"source"`
	SourceInterface string    `json:"sourceInterface,omitempty"`
	Target          string    `json:"target"`
	TargetInterface string    `json:"targetInterface,omitempty"`
	Type            string    `json:"type"`
	FirstSeen       time.Time `json:"firstSeen"`
}

// ProbeOutcome is one probe's checks in the window. AvgLatencyMs averages the
// successful checks only.
type ProbeOutcome struct {
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`
	Checks       int     `json:"checks"`
	Failures     int     `json:"failures"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
}

func (p *ProbeOutcome) availability() float64 {
	return float64(p.Checks-p.Failures) / float64(p.Checks) * fullPercent
}

// ProbeSummary totals the probes' checks; Outcomes lists every probe that
// ran, the least available first.
type ProbeSummary struct {
	Probes   int            `json:"probes"`
	Checks   int            `json:"checks"`
	Failures int            `json:"failures"`
	Outcomes []ProbeOutcome `json:"outcomes"`
}

// summaryWindowEnding is the window a summary generated at now covers: the
// day up to the start of the current minute, so a run the scheduler fires at
// 06:00:20 covers 06:00 to 06:00.
func summaryWindowEnding(now time.Time) DateRange {
	end := now.UTC().Truncate(time.Minute)
	return DateRange{Start: end.Add(-summaryWindow), End: end}
}

// Summarize reads the network summary for window.
func (s *AggregatorService) Summarize(ctx context.Context, window DateRange) (*NetworkSummary, error) {
	interfaces, err := s.metrics.InterfaceHealth(ctx, window)
	if err != nil {
		return nil, fmt.Errorf("reading interface health: %w", err)
	}
	severities, open, err := s.metrics.AlertSeverityCounts(ctx, window)
	if err != nil {
		return nil, fmt.Errorf("reading alerts: %w", err)
	}
	topology, err := s.metrics.TopologyChanges(ctx, window)
	if err != nil {
		return nil, fmt.Errorf("reading topology changes: %w", err)
	}
	probes, err := s.metrics.ProbeOutcomes(ctx, window)
	if err != nil {
		return nil, fmt.Errorf("reading probe results: %w", err)
	}

	return &NetworkSummary{
		Window:     window,
		Interfaces: rankInterfaces(interfaces),
		Alerts:     alertVolume(severities, open),
		Topology:   topology,
		Probes:     summarizeProbes(probes),
	}, nil
}

func rankInterfaces(all []InterfaceHealth) InterfaceSummary {
	byName := func(a, b InterfaceHealth) int {
		return cmp.Or(cmp.Compare(a.Target, b.Target), cmp.Compare(a.IfIndex, b.IfIndex))
	}

	var busiest, faulted []InterfaceHealth
	for _, h := range all {
		if h.PeakUtilization > 0 {
			busiest = append(busiest, h)
		}
		if h.faultRate() > 0 {
			faulted = append(faulted, h)
		}
	}
	slices.SortFunc(busiest, func(a, b InterfaceHealth) int {
		return cmp.Or(cmp.Compare(b.PeakUtilization, a.PeakUtilization), byName(a, b))
	})
	slices.SortFunc(faulted, func(a, b InterfaceHealth) int {
		return cmp.Or(cmp.Compare(b.faultRate(), a.faultRate()), byName(a, b))
	})

	return InterfaceSummary{
		Rated:   len(all),
		Busiest: busiest[:min(len(busiest), summaryListLimit)],
		Faulty:  len(faulted),
		Faulted: faulted[:min(len(faulted), summaryListLimit)],
	}
}

func alertVolume(severities map[string]int, open int) AlertVolume {
	v := AlertVolume{Open: open}
	for severity, count := range severities {
		switch severity {
		case statusCritical:
			v.Critical = count
		case "error":
			v.Error = count
		case "warning":
			v.Warning = count
		case "info":
			v.Info = count
		}
		v.Total += count
	}
	return v
}

func summarizeProbes(outcomes []ProbeOutcome) ProbeSummary {
	sum := ProbeSummary{Probes: len(outcomes), Outcomes: outcomes}
	for _, o := range outcomes {
		sum.Checks += o.Checks
		sum.Failures += o.Failures
	}
	slices.SortFunc(sum.Outcomes, func(a, b ProbeOutcome) int {
		return cmp.Or(cmp.Compare(a.availability(), b.availability()), cmp.Compare(a.Name, b.Name))
	})
	return sum
}

// generateSummary renders the summary for the window generateReport stamped
// on report.
func (s *GeneratorService) generateSummary(ctx context.Context, report *Report) ([]byte, error) {
	summary, err := s.aggregator.Summarize(ctx, *report.Parameters.DateRange)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %w", err)
	}

	switch report.Format {
	case FormatPDF:
		return s.generateSummaryPDF(report, summary)
	case FormatJSON:
		out, marshalErr := json.MarshalIndent(map[string]any{
			"report": map[string]any{
				"id":        report.ID,
				"name":      report.Name,
				"type":      report.Type,
				"generated": time.Now().Format(time.RFC3339),
			},
			"summary": summary,
		}, "", "  ")
		if marshalErr != nil {
			return nil, fmt.Errorf("marshaling JSON report: %w", marshalErr)
		}
		return out, nil
	case FormatHTML, FormatCSV, FormatExcel, FormatMarkdown:
	}
	return nil, fmt.Errorf("unsupported format for a summary: %s", report.Format)
}

func (s *GeneratorService) generateSummaryPDF(report *Report, sum *NetworkSummary) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, pdfPageMarginBottom)
	// Device, interface and probe names are the operator's and may be any
	// UTF-8; the core fonts take cp1252.
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AddPage()
	s.addPDFCover(pdf, report)

	pdf.AddPage()
	s.addPDFSectionHeader(pdf, "Network Summary")
	addPDFMetrics(pdf, []pdfMetric{
		{"Window", sum.Window.Start.Format(summaryTimeLayout) + " to " + sum.Window.End.Format(summaryTimeLayout)},
		{"Interfaces rated", strconv.Itoa(sum.Interfaces.Rated)},
		{"Interfaces with errors", strconv.Itoa(sum.Interfaces.Faulty)},
		{"Alerts raised", fmt.Sprintf("%d (%d still open)", sum.Alerts.Total, sum.Alerts.Open)},
		{"New devices", strconv.Itoa(len(sum.Topology.NewNodes))},
		{"New links", strconv.Itoa(len(sum.Topology.NewLinks))},
		{"Probe checks failed", fmt.Sprintf("%d of %d", sum.Probes.Failures, sum.Probes.Checks)},
	})

	pdf.Ln(pdfSectionSpacingMed)
	s.addPDFSectionHeader(pdf, "Interface Health")
	addSummaryInterfaces(pdf, tr, &sum.Interfaces)

	pdf.Ln(pdfSectionSpacingMed)
	s.addPDFSectionHeader(pdf, "Alerts")
	addPDFMetrics(pdf, []pdfMetric{
		{"Critical", strconv.Itoa(sum.Alerts.Critical)},
		{"Error", strconv.Itoa(sum.Alerts.Error)},
		{"Warning", strconv.Itoa(sum.Alerts.Warning)},
		{"Info", strconv.Itoa(sum.Alerts.Info)},
		{"Still open", strconv.Itoa(sum.Alerts.Open)},
	})

	pdf.Ln(pdfSectionSpacingMed)
	s.addPDFSectionHeader(pdf, "Topology Changes")
	addSummaryTopology(pdf, tr, &sum.Topology)

	pdf.Ln(pdfSectionSpacingMed)
	s.addPDFSectionHeader(pdf, "Probe Results")
	addSummaryProbes(pdf, tr, &sum.Probes)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

// The summary tables' column widths are in mm, and each table's add up to the
// 190 mm between the margins.

func addSummaryInterfaces(pdf *fpdf.Fpdf, tr func(string) string, sum *InterfaceSummary) {
	addPDFSubsection(pdf, "Busiest interfaces (utilization of line rate)")
	rows := make([][]string, 0, len(sum.Busiest))
	for _, h := range sum.Busiest {
		rows = append(rows, []string{
			tr(h.label()), percent(h.PeakUtilization), percent(h.AvgInUtilization), percent(h.AvgOutUtilization),
		})
	}
	addPDFTable(pdf, []float64{100, 30, 30, 30}, []string{"Interface", "Peak", "Mean in", "Mean out"}, rows,
		"No interface with a known line rate was rated in this window.")

	addPDFSubsection(pdf, "Interfaces with errors or discards (mean per second)")
	rows = make([][]string, 0, len(sum.Faulted))
	for _, h := range sum.Faulted {
		rows = append(rows, []string{
			tr(
				h.label(),
			),
			perSecond(h.InErrors),
			perSecond(h.OutErrors),
			perSecond(h.InDiscards),
			perSecond(h.OutDiscards),
		})
	}
	addPDFTable(pdf, []float64{90, 25, 25, 25, 25},
		[]string{"Interface", "In errors", "Out errors", "In discards", "Out discards"},
		rows, "No interface counted an error or discard in this window.")
}

func addSummaryTopology(pdf *fpdf.Fpdf, tr func(string) string, changes *TopologyChanges) {
	addPDFSubsection(pdf, "New devices")
	rows := make([][]string, 0, summaryListLimit)
	for _, n := range changes.NewNodes[:min(len(changes.NewNodes), summaryListLimit)] {
		rows = append(rows, []string{tr(n.Name), n.Address, n.FirstSeen.UTC().Format(summaryTimeLayout)})
	}
	addPDFTable(pdf, []float64{90, 50, 50}, []string{"Device", "Address", "First seen"}, rows,
		"No new device appeared in this window.")
	addPDFMore(pdf, len(changes.NewNodes))

	addPDFSubsection(pdf, "New links")
	rows = make([][]string, 0, summaryListLimit)
	for _, l := range changes.NewLinks[:min(len(changes.NewLinks), summaryListLimit)] {
		rows = append(rows, []string{
			tr(endpoint(l.Source, l.SourceInterface)), tr(endpoint(l.Target, l.TargetInterface)),
			l.Type, l.FirstSeen.UTC().Format(summaryTimeLayout),
		})
	}
	addPDFTable(pdf, []float64{70, 70, 20, 30}, []string{"From", "To", "Type", "First seen"}, rows,
		"No new link appeared in this window.")
	addPDFMore(pdf, len(changes.NewLinks))
}

func addSummaryProbes(pdf *fpdf.Fpdf, tr func(string) string, probes *ProbeSummary) {
	rows := make([][]string, 0, summaryListLimit)
	for _, p := range probes.Outcomes[:min(len(probes.Outcomes), summaryListLimit)] {
		latency := "-"
		if p.Failures < p.Checks {
			latency = fmt.Sprintf("%.1f ms", p.AvgLatencyMs)
		}
		rows = append(rows, []string{
			tr(p.Name), p.Kind, strconv.Itoa(p.Checks), strconv.Itoa(p.Failures), percent(p.availability()), latency,
		})
	}
	addPDFTable(pdf, []float64{60, 30, 20, 20, 30, 30},
		[]string{"Probe", "Kind", "Checks", "Failed", "Available", "Mean latency"},
		rows, "No probe ran in this window.")
	addPDFMore(pdf, len(probes.Outcomes))
}

type pdfMetric struct {
	label string
	value string
}

func addPDFMetrics(pdf *fpdf.Fpdf, metrics []pdfMetric) {
	pdf.SetFont("Arial", "", pdfFontSizeBody)
	for _, m := range metrics {
		pdf.SetTextColor(pdfColorGrayDark, pdfColorGrayDark, pdfColorGrayDark)
		pdf.CellFormat(pdfLabelColumnWidth, pdfCellHeightMetric, m.label+":", "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, pdfCellHeightMetric, m.value, "", 1, "L", false, 0, "")
	}
}

func addPDFSubsection(pdf *fpdf.Fpdf, title string) {
	pdf.Ln(pdfSectionSpacingSmall)
	pdf.SetFont("Arial", "B", pdfFontSizeSubsection)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, pdfCellHeightBody, title, "", 1, "L", false, 0, "")
}

// addPDFTable prints a header row and rows, or empty when there are no rows.
// The first column is left-aligned and the rest right-aligned.
func addPDFTable(pdf *fpdf.Fpdf, widths []float64, header []string, rows [][]string, empty string) {
	if len(rows) == 0 {
		pdf.SetFont("Arial", "", pdfFontSizeBody)
		pdf.SetTextColor(pdfColorGrayMid, pdfColorGrayMid, pdfColorGrayMid)
		pdf.CellFormat(0, pdfCellHeightSeverity, empty, "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		return
	}
	row := func(cells []string) {
		for i, cell := range cells {
			align := "R"
			if i == 0 {
				align = "L"
			}
			pdf.CellFormat(widths[i], pdfCellHeightSeverity, cell, "", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.SetFont("Arial", "B", pdfFontSizeSmall)
	pdf.SetTextColor(pdfColorGrayDark, pdfColorGrayDark, pdfColorGrayDark)
	row(header)
	pdf.SetFont("Arial", "", pdfFontSizeSmall)
	pdf.SetTextColor(0, 0, 0)
	for _, cells := range rows {
		row(cells)
	}
}

// addPDFMore notes how many of total a capped list left out.
func addPDFMore(pdf *fpdf.Fpdf, total int) {
	if total <= summaryListLimit {
		return
	}
	pdf.SetFont("Arial", "I", pdfFontSizeSmall)
	pdf.SetTextColor(pdfColorGrayMid, pdfColorGrayMid, pdfColorGrayMid)
	pdf.CellFormat(0, pdfCellHeightSeverity,
		fmt.Sprintf("and %d more; the JSON summary lists every one.", total-summaryListLimit),
		"", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func endpoint(node, iface string) string {
	if iface == "" {
		return node
	}
	return node + " " + iface
}

func percent(v float64) string   { return fmt.Sprintf("%.1f%%", v) }
func perSecond(v float64) string { return fmt.Sprintf("%.2f", v) }
