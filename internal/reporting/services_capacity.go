package reporting

// services_capacity.go contains the capacity forecast: each interface's daily
// busy hour, the straight-line trend through it, and when that trend reaches
// the planning threshold, with its PDF and JSON renderers. The method is
// internal/timeseries/forecast: least squares and a 95% confidence band, with
// nothing learned or tuned.

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/forecast"
)

const (
	// capacityHistory is how far back a forecast reads: the longest the
	// retention engine keeps hourly rollups (Pro). A shorter retention
	// leaves less history, and the forecast's horizon shrinks with it.
	capacityHistory = 90 * hoursPerDay * time.Hour
	// CapacityThreshold is the busy-hour utilization, in percent of line
	// rate, that a forecast estimates the arrival of.
	CapacityThreshold  = 80.0
	capacityDateLayout = "2006-01-02"
)

// InterfaceUtilization is one interface's hourly utilization. Each point is
// an hour's mean in whichever direction ran busier, timed at the middle of
// the hour.
type InterfaceUtilization struct {
	Target  string
	IfIndex uint32
	IfName  string
	Hours   []forecast.Point
}

// CapacityForecast is the forecast for every interface with enough history.
// Interfaces are ordered most pressing first: over the threshold, then
// reaching it soonest, then rising past the horizon, then the rest by trend.
type CapacityForecast struct {
	Window DateRange `json:"window"`
	// Threshold is CapacityThreshold, carried so the JSON reads alone.
	Threshold  float64             `json:"threshold"`
	Interfaces []InterfaceForecast `json:"interfaces"`
	// ShortHistory counts the interfaces rated in the window on fewer than
	// forecast.MinPoints days, which get no forecast.
	ShortHistory int `json:"shortHistory"`
}

// InterfaceForecast is one interface's trend and when it reaches the
// threshold. BusyHour is its last day's busiest hour; Level is where the
// trend line stands on that day. The crossing is forecast no later than
// Horizon, which is as far past the last day as the history reaches back.
type InterfaceForecast struct {
	forecast.Crossing

	Target      string    `json:"target"`
	IfIndex     uint32    `json:"ifIndex"`
	IfName      string    `json:"ifName,omitempty"`
	Days        int       `json:"days"`
	BusyHour    float64   `json:"busyHour"`
	Level       float64   `json:"level"`
	TrendPerDay float64   `json:"trendPerDay"`
	Horizon     time.Time `json:"horizon"`
}

func (f *InterfaceForecast) label() string {
	return interfaceLabel(f.Target, f.IfName, f.IfIndex)
}

// capacityWindowEnding is the history a forecast generated at now reads:
// whole UTC days up to the start of today, so a day still in progress does
// not pull its busy hour down.
func capacityWindowEnding(now time.Time) DateRange {
	end := now.UTC().Truncate(hoursPerDay * time.Hour)
	return DateRange{Start: end.Add(-capacityHistory), End: end}
}

// Forecast reads the window's interface utilization and forecasts each
// interface.
func (s *AggregatorService) Forecast(ctx context.Context, window DateRange) (*CapacityForecast, error) {
	history, err := s.metrics.InterfaceUtilizationHistory(ctx, window)
	if err != nil {
		return nil, fmt.Errorf("reading interface utilization: %w", err)
	}

	out := &CapacityForecast{Window: window, Threshold: CapacityThreshold, Interfaces: []InterfaceForecast{}}
	for _, u := range history {
		days := busyHours(u.Hours)
		trend, fitErr := forecast.Fit(days)
		if fitErr != nil {
			out.ShortHistory++
			continue
		}
		out.Interfaces = append(out.Interfaces, InterfaceForecast{
			Target:      u.Target,
			IfIndex:     u.IfIndex,
			IfName:      u.IfName,
			Days:        len(days),
			BusyHour:    days[len(days)-1].Value,
			Level:       trend.Level,
			TrendPerDay: trend.PerDay,
			Horizon:     trend.Horizon(),
			Crossing:    trend.Crossing(CapacityThreshold),
		})
	}
	slices.SortFunc(out.Interfaces, comparePressing)
	return out, nil
}

// busyHours returns each UTC day's busiest hour from hours, which are in time
// order, in day order.
func busyHours(hours []forecast.Point) []forecast.Point {
	var days []forecast.Point
	for _, h := range hours {
		last := len(days) - 1
		switch {
		case last < 0 || !sameUTCDay(days[last].At, h.At):
			days = append(days, h)
		case h.Value > days[last].Value:
			days[last] = h
		}
	}
	return days
}

func sameUTCDay(a, b time.Time) bool {
	return a.UTC().Truncate(hoursPerDay * time.Hour).Equal(b.UTC().Truncate(hoursPerDay * time.Hour))
}

func comparePressing(a, b InterfaceForecast) int {
	rank := func(o forecast.Outlook) int {
		return slices.Index([]forecast.Outlook{forecast.Above, forecast.Rising, forecast.Beyond, forecast.Steady}, o)
	}
	if c := cmp.Compare(rank(a.Outlook), rank(b.Outlook)); c != 0 {
		return c
	}
	var c int
	switch a.Outlook {
	case forecast.Above:
		c = cmp.Compare(b.Level, a.Level)
	case forecast.Rising:
		c = a.Estimate.Compare(*b.Estimate)
	case forecast.Beyond, forecast.Steady:
		c = cmp.Compare(b.TrendPerDay, a.TrendPerDay)
	}
	return cmp.Or(c, cmp.Compare(a.Target, b.Target), cmp.Compare(a.IfIndex, b.IfIndex))
}

// generateCapacity renders the forecast for the window generateReport stamped
// on report.
func (s *GeneratorService) generateCapacity(ctx context.Context, report *Report) ([]byte, error) {
	fc, err := s.aggregator.Forecast(ctx, *report.Parameters.DateRange)
	if err != nil {
		return nil, fmt.Errorf("forecast failed: %w", err)
	}

	switch report.Format {
	case FormatPDF:
		return s.generateCapacityPDF(report, fc)
	case FormatJSON:
		return marshalWindowedJSON(report, "forecast", fc)
	case FormatHTML, FormatCSV, FormatExcel, FormatMarkdown:
	}
	return nil, fmt.Errorf("unsupported format for a capacity forecast: %s", report.Format)
}

func (s *GeneratorService) generateCapacityPDF(report *Report, fc *CapacityForecast) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, pdfPageMarginBottom)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AddPage()
	s.addPDFCover(pdf, report)

	counts := make(map[forecast.Outlook]int)
	for _, f := range fc.Interfaces {
		counts[f.Outlook]++
	}
	pdf.AddPage()
	s.addPDFSectionHeader(pdf, "Capacity Forecast")
	addPDFMetrics(pdf, []pdfMetric{
		{"History", fc.Window.Start.Format(capacityDateLayout) + " to " + fc.Window.End.Format(capacityDateLayout)},
		{"Threshold", percent(fc.Threshold) + " of line rate in the busy hour"},
		{"Interfaces forecast", strconv.Itoa(len(fc.Interfaces))},
		{"Already over", strconv.Itoa(counts[forecast.Above])},
		{"Reaching it", strconv.Itoa(counts[forecast.Rising])},
		{"Too little history", strconv.Itoa(fc.ShortHistory)},
	})

	addPDFSubsection(pdf, "Interfaces, most pressing first")
	rows := make([][]string, 0, summaryListLimit)
	for _, f := range fc.Interfaces[:min(len(fc.Interfaces), summaryListLimit)] {
		reaches, between := capacityCells(&f)
		rows = append(rows, []string{
			tr(f.label()), percent(f.BusyHour), fmt.Sprintf("%+.2f", f.TrendPerDay), reaches, between,
		})
	}
	addPDFTable(pdf, []float64{70, 22, 22, 36, 40},
		[]string{"Interface", "Busy hour", "Trend/day", "Reaches " + percent(fc.Threshold), "95% range"},
		rows, fmt.Sprintf("No interface has %d days of utilization history yet.", forecast.MinPoints))
	addPDFMore(pdf, len(fc.Interfaces))

	pdf.Ln(pdfSectionSpacingSmall)
	pdf.SetFont("Arial", "I", pdfFontSizeSmall)
	pdf.SetTextColor(pdfColorGrayMid, pdfColorGrayMid, pdfColorGrayMid)
	pdf.MultiCell(0, pdfCellHeightSeverity, "Busy hour is a day's busiest hour, in the busier direction. "+
		"The trend is a straight line fitted to the daily busy hours; the range is where its 95% confidence band "+
		"reaches the threshold. A forecast reaches no further ahead than the interface's history reaches back.",
		"", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

// capacityCells returns the "Reaches" and "95% range" cells for f.
func capacityCells(f *InterfaceForecast) (string, string) {
	day := func(t time.Time) string { return t.UTC().Format(capacityDateLayout) }
	switch f.Outlook {
	case forecast.Above:
		return "already over", "-"
	case forecast.Rising:
		if f.Latest == nil {
			return day(*f.Estimate), "from " + day(*f.Earliest)
		}
		return day(*f.Estimate), day(*f.Earliest) + " to " + day(*f.Latest)
	case forecast.Beyond:
		return "after " + day(f.Horizon), "-"
	case forecast.Steady:
	}
	return "not rising", "-"
}
