// Package survey provides WiFi site survey functionality.
package survey

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"
)

// ReportOptions configures what sections to include in the survey report.
type ReportOptions struct {
	IncludeHeatmaps         bool   `json:"includeHeatmaps"`
	IncludeRawData          bool   `json:"includeRawData"`
	IncludeRecommendations  bool   `json:"includeRecommendations"`
	IncludeExecutiveSummary bool   `json:"includeExecutiveSummary"`
	CompanyName             string `json:"companyName,omitempty"`
	CompanyLogo             []byte `json:"companyLogo,omitempty"`
}

// DefaultReportOptions returns sensible defaults for report generation.
func DefaultReportOptions() ReportOptions {
	return ReportOptions{
		IncludeHeatmaps:         true,
		IncludeRawData:          false,
		IncludeRecommendations:  true,
		IncludeExecutiveSummary: true,
	}
}

// ReportGenerator creates PDF reports from survey data.
type ReportGenerator struct {
	survey  *Survey
	options ReportOptions
	pdf     *fpdf.Fpdf
}

// NewReportGenerator creates a new report generator for the given survey.
func NewReportGenerator(survey *Survey, options ReportOptions) *ReportGenerator {
	return &ReportGenerator{
		survey:  survey,
		options: options,
	}
}

// Generate creates a PDF report and returns the bytes.
func (g *ReportGenerator) Generate() ([]byte, error) {
	if g.survey == nil {
		return nil, errors.New("survey is nil")
	}

	// Initialize PDF
	g.pdf = fpdf.New("P", "mm", "A4", "")
	g.pdf.SetAutoPageBreak(true, 15)

	// Add cover page
	g.addCoverPage()

	// Add executive summary if requested
	if g.options.IncludeExecutiveSummary {
		g.addExecutiveSummary()
	}

	// Add per-floor analysis
	g.addFloorAnalysis()

	// Add recommendations if requested
	if g.options.IncludeRecommendations {
		g.addRecommendations()
	}

	// Add raw data appendix if requested
	if g.options.IncludeRawData {
		g.addRawDataAppendix()
	}

	// Output to buffer
	var buf bytes.Buffer
	if err := g.pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// addCoverPage creates the report cover page.
func (g *ReportGenerator) addCoverPage() {
	g.pdf.AddPage()

	// Header with company name if provided
	if g.options.CompanyName != "" {
		g.pdf.SetFont("Arial", "", 12)
		g.pdf.SetTextColor(100, 100, 100)
		g.pdf.CellFormat(0, 10, g.options.CompanyName, "", 1, "C", false, 0, "")
	}

	// Main title
	g.pdf.Ln(50)
	g.pdf.SetFont("Arial", "B", 28)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, 15, "WiFi Site Survey Report", "", 1, "C", false, 0, "")

	// Survey name
	g.pdf.Ln(10)
	g.pdf.SetFont("Arial", "", 18)
	g.pdf.SetTextColor(60, 60, 60)
	g.pdf.CellFormat(0, 10, g.survey.Name, "", 1, "C", false, 0, "")

	// Description if provided
	if g.survey.Description != "" {
		g.pdf.Ln(5)
		g.pdf.SetFont("Arial", "I", 12)
		g.pdf.SetTextColor(100, 100, 100)
		g.pdf.MultiCell(0, 6, g.survey.Description, "", "C", false)
	}

	// Date info
	g.pdf.Ln(30)
	g.pdf.SetFont("Arial", "", 12)
	g.pdf.SetTextColor(80, 80, 80)

	dateStr := time.Now().Format("January 2, 2006")
	g.pdf.CellFormat(0, 8, fmt.Sprintf("Report Generated: %s", dateStr), "", 1, "C", false, 0, "")

	surveyDate := g.survey.CreatedAt.Format("January 2, 2006")
	g.pdf.CellFormat(0, 8, fmt.Sprintf("Survey Date: %s", surveyDate), "", 1, "C", false, 0, "")

	// Status
	g.pdf.Ln(5)
	g.pdf.SetFont("Arial", "B", 12)
	statusColor := getStatusColor(g.survey.Status)
	g.pdf.SetTextColor(statusColor[0], statusColor[1], statusColor[2])
	g.pdf.CellFormat(0, 8, fmt.Sprintf("Status: %s", g.survey.Status), "", 1, "C", false, 0, "")

	// Building info
	g.pdf.Ln(20)
	g.pdf.SetFont("Arial", "", 11)
	g.pdf.SetTextColor(80, 80, 80)

	floorCount := len(g.survey.Floors)
	sampleCount := len(g.survey.GetAllSamples())
	g.pdf.CellFormat(
		0,
		7,
		fmt.Sprintf("Floors: %d | Sample Points: %d", floorCount, sampleCount),
		"",
		1,
		"C",
		false,
		0,
		"",
	)
}

// addExecutiveSummary adds the executive summary section.
func (g *ReportGenerator) addExecutiveSummary() {
	g.pdf.AddPage()
	g.addSectionHeader("Executive Summary")

	// Calculate overall statistics
	allSamples := g.survey.GetAllSamples()
	stats := calculateSurveyStats(allSamples)

	// Coverage score card
	g.pdf.Ln(5)
	g.addStatCard(
		"Overall Coverage Score",
		fmt.Sprintf("%.0f%%", stats.CoverageScore),
		getCoverageGrade(stats.CoverageScore),
	)

	// Key metrics table
	g.pdf.Ln(10)
	g.pdf.SetFont("Arial", "B", 12)
	g.pdf.CellFormat(0, 8, "Key Metrics", "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "", 10)
	metrics := []struct {
		label string
		value string
	}{
		{"Total Sample Points", strconv.Itoa(stats.TotalSamples)},
		{"Average Signal Strength", fmt.Sprintf("%d dBm", stats.AvgRSSI)},
		{"Minimum Signal", fmt.Sprintf("%d dBm", stats.MinRSSI)},
		{"Maximum Signal", fmt.Sprintf("%d dBm", stats.MaxRSSI)},
		{"Weak Coverage Areas", strconv.Itoa(stats.WeakAreas)},
		{"Dead Zones Detected", strconv.Itoa(stats.DeadZones)},
	}

	for _, m := range metrics {
		g.addMetricRow(m.label, m.value)
	}

	// Signal distribution breakdown
	g.pdf.Ln(10)
	g.pdf.SetFont("Arial", "B", 12)
	g.pdf.CellFormat(0, 8, "Signal Quality Distribution", "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "", 10)
	signalDist := []struct {
		label   string
		percent float64
		color   []int
	}{
		{"Excellent (> -50 dBm)", stats.ExcellentPercent, []int{40, 167, 69}},
		{"Good (-50 to -65 dBm)", stats.GoodPercent, []int{144, 238, 144}},
		{"Fair (-65 to -75 dBm)", stats.FairPercent, []int{255, 193, 7}},
		{"Poor (-75 to -85 dBm)", stats.PoorPercent, []int{255, 128, 0}},
		{"Dead (< -85 dBm)", stats.DeadPercent, []int{220, 53, 69}},
	}

	for _, dist := range signalDist {
		g.addDistributionBar(dist.label, dist.percent, dist.color)
	}
}

// addFloorAnalysis adds per-floor analysis sections.
func (g *ReportGenerator) addFloorAnalysis() {
	floors := g.survey.Floors
	if len(floors) == 0 {
		return
	}

	// Sort floors by level
	sortedFloors := make([]*Floor, len(floors))
	copy(sortedFloors, floors)
	sort.Slice(sortedFloors, func(i, j int) bool {
		return sortedFloors[i].Level < sortedFloors[j].Level
	})

	for _, floor := range sortedFloors {
		g.addFloorSection(floor)
	}
}

// addFloorSection adds analysis for a single floor.
func (g *ReportGenerator) addFloorSection(floor *Floor) {
	g.pdf.AddPage()
	g.addSectionHeader(fmt.Sprintf("Floor Analysis: %s", floor.Name))

	// Floor info
	g.pdf.SetFont("Arial", "", 10)
	g.pdf.SetTextColor(80, 80, 80)
	g.pdf.CellFormat(
		0,
		6,
		fmt.Sprintf("Level: %d | Samples: %d", floor.Level, len(floor.Samples)),
		"",
		1,
		"L",
		false,
		0,
		"",
	)

	// Floor plan dimensions if available
	if floor.FloorPlan != nil {
		g.pdf.CellFormat(
			0,
			6,
			fmt.Sprintf("Dimensions: %d x %d px", floor.FloorPlan.Width, floor.FloorPlan.Height),
			"",
			1,
			"L",
			false,
			0,
			"",
		)
		if floor.FloorPlan.ScaleM > 0 {
			g.pdf.CellFormat(0, 6, fmt.Sprintf("Scale: %.2f m/px", floor.FloorPlan.ScaleM), "", 1, "L", false, 0, "")
		}
	}

	// Floor statistics
	if len(floor.Samples) > 0 {
		g.pdf.Ln(5)
		stats := calculateFloorStats(floor.Samples)

		g.pdf.SetFont("Arial", "B", 11)
		g.pdf.SetTextColor(0, 0, 0)
		g.pdf.CellFormat(0, 8, "Coverage Statistics", "", 1, "L", false, 0, "")

		g.pdf.SetFont("Arial", "", 10)
		floorMetrics := []struct {
			label string
			value string
		}{
			{"Coverage Score", fmt.Sprintf("%.0f%%", stats.CoverageScore)},
			{"Average Signal", fmt.Sprintf("%d dBm", stats.AvgRSSI)},
			{"Signal Range", fmt.Sprintf("%d to %d dBm", stats.MinRSSI, stats.MaxRSSI)},
			{"Weak Spots", strconv.Itoa(stats.WeakAreas)},
		}

		for _, m := range floorMetrics {
			g.addMetricRow(m.label, m.value)
		}

		// Channel usage summary
		channels := getChannelUsage(floor.Samples)
		if len(channels) > 0 {
			g.pdf.Ln(5)
			g.pdf.SetFont("Arial", "B", 11)
			g.pdf.CellFormat(0, 8, "WiFi Channels Detected", "", 1, "L", false, 0, "")

			g.pdf.SetFont("Arial", "", 10)
			for _, ch := range channels {
				g.pdf.CellFormat(
					0,
					6,
					fmt.Sprintf("Channel %d: %d APs", ch.Channel, ch.Count),
					"",
					1,
					"L",
					false,
					0,
					"",
				)
			}
		}
	} else {
		g.pdf.Ln(5)
		g.pdf.SetFont("Arial", "I", 10)
		g.pdf.SetTextColor(150, 150, 150)
		g.pdf.CellFormat(0, 8, "No samples collected for this floor", "", 1, "L", false, 0, "")
	}

	// Add heatmap if requested and floor has data
	if g.options.IncludeHeatmaps && len(floor.Samples) > 0 && floor.FloorPlan != nil {
		g.addFloorHeatmapNote(floor)
	}
}

// addFloorHeatmapNote adds a note about heatmap availability.
func (g *ReportGenerator) addFloorHeatmapNote(_ *Floor) {
	g.pdf.Ln(10)
	g.pdf.SetFont("Arial", "I", 10)
	g.pdf.SetTextColor(80, 80, 80)
	g.pdf.CellFormat(0, 6, "Heatmap visualization available in the web interface.", "", 1, "L", false, 0, "")
}

// addRecommendations adds the recommendations section.
func (g *ReportGenerator) addRecommendations() {
	g.pdf.AddPage()
	g.addSectionHeader("Recommendations")

	allSamples := g.survey.GetAllSamples()
	stats := calculateSurveyStats(allSamples)

	recommendations := generateSurveyRecommendations(&stats)

	if len(recommendations) == 0 {
		g.pdf.SetFont("Arial", "I", 10)
		g.pdf.SetTextColor(100, 100, 100)
		g.pdf.CellFormat(
			0,
			8,
			"No specific recommendations - WiFi coverage meets quality standards.",
			"",
			1,
			"L",
			false,
			0,
			"",
		)
		return
	}

	g.pdf.SetFont("Arial", "", 10)
	g.pdf.SetTextColor(0, 0, 0)

	for i, rec := range recommendations {
		g.pdf.Ln(3)
		priority := getPriorityLabel(rec.Priority)

		// Priority badge
		g.pdf.SetFont("Arial", "B", 9)
		switch rec.Priority {
		case PriorityHigh:
			g.pdf.SetTextColor(220, 53, 69)
		case PriorityMedium:
			g.pdf.SetTextColor(255, 128, 0)
		case PriorityLow:
			g.pdf.SetTextColor(40, 167, 69)
		}
		g.pdf.CellFormat(20, 6, priority, "1", 0, "C", false, 0, "")

		// Recommendation text
		g.pdf.SetFont("Arial", "", 10)
		g.pdf.SetTextColor(0, 0, 0)
		g.pdf.MultiCell(0, 6, fmt.Sprintf(" %d. %s", i+1, rec.Text), "", "L", false)
	}

	// Implementation notes
	g.pdf.Ln(10)
	g.pdf.SetFont("Arial", "B", 11)
	g.pdf.CellFormat(0, 8, "Implementation Notes", "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "", 10)
	notes := []string{
		"High priority items should be addressed within 1-2 weeks",
		"Consider WiFi 6/6E access points for improved capacity",
		"Verify power levels and channel assignments after changes",
		"Re-survey affected areas after implementing changes",
	}

	for _, note := range notes {
		g.pdf.SetTextColor(80, 80, 80)
		g.pdf.CellFormat(5, 6, "-", "", 0, "L", false, 0, "")
		g.pdf.CellFormat(0, 6, note, "", 1, "L", false, 0, "")
	}
}

// addRawDataAppendix adds the raw data appendix section.
func (g *ReportGenerator) addRawDataAppendix() {
	g.pdf.AddPage()
	g.addSectionHeader("Appendix: Raw Sample Data")

	allSamples := g.survey.GetAllSamples()
	if len(allSamples) == 0 {
		g.pdf.SetFont("Arial", "I", 10)
		g.pdf.CellFormat(0, 8, "No sample data collected", "", 1, "L", false, 0, "")
		return
	}

	// Table header
	g.pdf.SetFont("Arial", "B", 8)
	g.pdf.SetFillColor(240, 240, 240)
	headers := []struct {
		text  string
		width float64
	}{
		{"#", 8},
		{"X", 15},
		{"Y", 15},
		{"RSSI", 20},
		{"SNR", 20},
		{"SSID", 50},
		{"Channel", 20},
		{"Time", 35},
	}

	for _, h := range headers {
		g.pdf.CellFormat(h.width, 7, h.text, "1", 0, "C", true, 0, "")
	}
	g.pdf.Ln(-1)

	// Table rows (limit to 50 samples per page section)
	g.pdf.SetFont("Arial", "", 7)
	maxSamples := min(len(allSamples), 50)

	for i := range maxSamples {
		sample := allSamples[i]

		// Get first network from passive sample for display
		rssi := "-"
		snr := "-"
		ssid := "-"
		channel := "-"
		ps := getPassiveSampleFromPoint(sample)
		if ps != nil && len(ps.Networks) > 0 {
			net := ps.Networks[0]
			rssi = strconv.Itoa(net.Signal)
			snr = strconv.Itoa(net.SNR)
			ssid = truncateString(net.SSID, 15)
			channel = strconv.Itoa(net.Channel)
		}

		g.pdf.CellFormat(8, 6, strconv.Itoa(i+1), "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(15, 6, strconv.Itoa(sample.X), "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(15, 6, strconv.Itoa(sample.Y), "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(20, 6, rssi, "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(20, 6, snr, "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(50, 6, ssid, "1", 0, "L", false, 0, "")
		g.pdf.CellFormat(20, 6, channel, "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(35, 6, sample.Timestamp.Format("15:04:05"), "1", 0, "C", false, 0, "")
		g.pdf.Ln(-1)
	}

	if len(allSamples) > maxSamples {
		g.pdf.Ln(5)
		g.pdf.SetFont("Arial", "I", 9)
		g.pdf.CellFormat(
			0,
			6,
			fmt.Sprintf("... and %d more samples (truncated for readability)", len(allSamples)-maxSamples),
			"",
			1,
			"L",
			false,
			0,
			"",
		)
	}
}

// Helper methods

func (g *ReportGenerator) addSectionHeader(title string) {
	g.pdf.SetFont("Arial", "B", 16)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, 12, title, "", 1, "L", false, 0, "")

	// Underline
	g.pdf.SetDrawColor(200, 200, 200)
	g.pdf.Line(10, g.pdf.GetY(), 200, g.pdf.GetY())
	g.pdf.Ln(5)
}

func (g *ReportGenerator) addStatCard(label, value, grade string) {
	g.pdf.SetFont("Arial", "B", 14)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, 10, label, "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "B", 36)
	gradeColor := getGradeColor(grade)
	g.pdf.SetTextColor(gradeColor[0], gradeColor[1], gradeColor[2])
	g.pdf.CellFormat(60, 20, value, "", 0, "L", false, 0, "")

	g.pdf.SetFont("Arial", "B", 24)
	g.pdf.CellFormat(0, 20, grade, "", 1, "L", false, 0, "")
}

func (g *ReportGenerator) addMetricRow(label, value string) {
	g.pdf.SetTextColor(80, 80, 80)
	g.pdf.CellFormat(80, 6, label+":", "", 0, "L", false, 0, "")
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, 6, value, "", 1, "L", false, 0, "")
}

func (g *ReportGenerator) addDistributionBar(label string, percent float64, col []int) {
	g.pdf.SetTextColor(80, 80, 80)
	g.pdf.CellFormat(60, 6, label, "", 0, "L", false, 0, "")

	// Draw bar background
	barX := g.pdf.GetX()
	barY := g.pdf.GetY()
	barWidth := 80.0
	barHeight := 5.0

	g.pdf.SetFillColor(230, 230, 230)
	g.pdf.Rect(barX, barY, barWidth, barHeight, "F")

	// Draw filled portion
	filledWidth := barWidth * (percent / 100.0)
	g.pdf.SetFillColor(col[0], col[1], col[2])
	g.pdf.Rect(barX, barY, filledWidth, barHeight, "F")

	// Percentage label
	g.pdf.SetX(barX + barWidth + 2)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(20, 6, fmt.Sprintf("%.1f%%", percent), "", 1, "L", false, 0, "")
}

// SurveyStats holds calculated statistics for a survey.
//
//nolint:revive // Renaming to Stats would be a breaking API change
type SurveyStats struct {
	TotalSamples     int
	AvgRSSI          int
	MinRSSI          int
	MaxRSSI          int
	CoverageScore    float64
	WeakAreas        int
	DeadZones        int
	ExcellentPercent float64
	GoodPercent      float64
	FairPercent      float64
	PoorPercent      float64
	DeadPercent      float64
}

func calculateSurveyStats(samples []*SamplePoint) SurveyStats {
	stats := SurveyStats{
		TotalSamples: len(samples),
		MinRSSI:      0,
		MaxRSSI:      -100,
	}

	if len(samples) == 0 {
		return stats
	}

	var sumRSSI int
	var excellent, good, fair, poor, dead int

	for _, sample := range samples {
		ps := getPassiveSampleFromPoint(sample)
		if ps == nil || len(ps.Networks) == 0 {
			continue
		}

		// Use best RSSI from networks in the passive sample
		bestRSSI := -100
		for _, net := range ps.Networks {
			if net.Signal > bestRSSI {
				bestRSSI = net.Signal
			}
		}

		sumRSSI += bestRSSI
		if bestRSSI > stats.MaxRSSI {
			stats.MaxRSSI = bestRSSI
		}
		if bestRSSI < stats.MinRSSI || stats.MinRSSI == 0 {
			stats.MinRSSI = bestRSSI
		}

		switch {
		case bestRSSI > ExcellentSignal:
			excellent++
		case bestRSSI > GoodSignal:
			good++
		case bestRSSI > FairSignal:
			fair++
		case bestRSSI > PoorSignal:
			poor++
		default:
			dead++
		}
	}

	if len(samples) > 0 {
		stats.AvgRSSI = sumRSSI / len(samples)
		total := float64(len(samples))
		stats.ExcellentPercent = float64(excellent) / total * 100
		stats.GoodPercent = float64(good) / total * 100
		stats.FairPercent = float64(fair) / total * 100
		stats.PoorPercent = float64(poor) / total * 100
		stats.DeadPercent = float64(dead) / total * 100
		stats.CoverageScore = stats.ExcellentPercent + stats.GoodPercent + (stats.FairPercent * 0.5)
		stats.WeakAreas = poor
		stats.DeadZones = dead
	}

	return stats
}

func calculateFloorStats(samples []*SamplePoint) SurveyStats {
	return calculateSurveyStats(samples)
}

// ChannelInfo represents WiFi channel usage information.
type ChannelInfo struct {
	Channel int
	Count   int
}

func getChannelUsage(samples []*SamplePoint) []ChannelInfo {
	channelMap := make(map[int]int)

	for _, sample := range samples {
		ps := getPassiveSampleFromPoint(sample)
		if ps == nil {
			continue
		}
		for _, net := range ps.Networks {
			if net.Channel > 0 {
				channelMap[net.Channel]++
			}
		}
	}

	channels := make([]ChannelInfo, 0, len(channelMap))
	for ch, count := range channelMap {
		channels = append(channels, ChannelInfo{Channel: ch, Count: count})
	}

	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Count > channels[j].Count
	})

	// Limit to top 5 channels
	if len(channels) > 5 {
		channels = channels[:5]
	}

	return channels
}

// RecommendationPriority indicates the urgency of a recommendation.
type RecommendationPriority int

// Recommendation priority levels.
const (
	PriorityLow RecommendationPriority = iota
	PriorityMedium
	PriorityHigh
)

// Recommendation represents an improvement suggestion.
type Recommendation struct {
	Text     string
	Priority RecommendationPriority
}

func generateSurveyRecommendations(stats *SurveyStats) []Recommendation {
	var recommendations []Recommendation

	// Coverage-based recommendations
	switch {
	case stats.CoverageScore < 50:
		recommendations = append(recommendations, Recommendation{
			Text:     "Critical coverage issues detected. Consider a complete WiFi infrastructure redesign with additional access points.",
			Priority: PriorityHigh,
		})
	case stats.CoverageScore < 70:
		recommendations = append(recommendations, Recommendation{
			Text:     "Poor overall coverage. Add 2-3 additional access points in strategic locations to improve connectivity.",
			Priority: PriorityHigh,
		})
	case stats.CoverageScore < 85:
		recommendations = append(recommendations, Recommendation{
			Text:     "Moderate coverage detected. Consider adding 1-2 access points to strengthen weak areas.",
			Priority: PriorityMedium,
		})
	}

	// Dead zone recommendations
	if stats.DeadZones > 0 {
		recommendations = append(recommendations, Recommendation{
			Text: fmt.Sprintf(
				"Found %d dead zone(s) with signal below -85 dBm. Prioritize these areas for immediate AP placement.",
				stats.DeadZones,
			),
			Priority: PriorityHigh,
		})
	}

	// Weak area recommendations
	if stats.WeakAreas > 0 {
		recommendations = append(recommendations, Recommendation{
			Text: fmt.Sprintf(
				"Found %d weak area(s) with poor signal. Consider power adjustments or additional coverage.",
				stats.WeakAreas,
			),
			Priority: PriorityMedium,
		})
	}

	// Sample density recommendations
	if stats.TotalSamples < 20 {
		recommendations = append(recommendations, Recommendation{
			Text:     "Limited sample data. Collect more samples for accurate analysis, especially in edge areas and around obstacles.",
			Priority: PriorityLow,
		})
	}

	// No issues found
	if len(recommendations) == 0 && stats.CoverageScore >= 90 {
		recommendations = append(recommendations, Recommendation{
			Text:     "WiFi coverage meets quality standards. Continue monitoring for any future degradation.",
			Priority: PriorityLow,
		})
	}

	return recommendations
}

func getPriorityLabel(p RecommendationPriority) string {
	switch p {
	case PriorityHigh:
		return "HIGH"
	case PriorityMedium:
		return "MED"
	case PriorityLow:
		return "LOW"
	}
	return "LOW" // Default fallback for unknown priority
}

func getStatusColor(status Status) []int {
	switch status {
	case StatusCompleted:
		return []int{40, 167, 69} // Green
	case StatusInProgress:
		return []int{0, 123, 255} // Blue
	case StatusPaused:
		return []int{255, 193, 7} // Yellow
	case StatusCreated:
		return []int{108, 117, 125} // Gray
	}
	return []int{108, 117, 125} // Default fallback
}

func getCoverageGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

func getGradeColor(grade string) []int {
	switch grade {
	case "A":
		return []int{40, 167, 69}
	case "B":
		return []int{144, 238, 144}
	case "C":
		return []int{255, 193, 7}
	case "D":
		return []int{255, 128, 0}
	default:
		return []int{220, 53, 69}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// getPassiveSampleFromPoint extracts PassiveSample data from a SamplePoint.
// Returns nil if the SampleData is not a PassiveSample.
func getPassiveSampleFromPoint(sp *SamplePoint) *PassiveSample {
	if sp == nil || sp.SampleData == nil {
		return nil
	}
	switch data := sp.SampleData.(type) {
	case *PassiveSample:
		return data
	case PassiveSample:
		return &data
	default:
		return nil
	}
}

// GenerateReport creates a PDF report for the specified survey.
func (m *Manager) GenerateReport(surveyID string, options ReportOptions) ([]byte, error) {
	survey, err := m.GetSurvey(surveyID)
	if err != nil {
		return nil, err
	}

	generator := NewReportGenerator(survey, options)
	return generator.Generate()
}
