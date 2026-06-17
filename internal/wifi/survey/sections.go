package survey

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// addCoverPage creates the report cover page.
func (g *ReportGenerator) addCoverPage() {
	g.pdf.AddPage()

	// Header with company name if provided
	if g.options.CompanyName != "" {
		g.pdf.SetFont("Arial", "", pdfFontSizeBody)
		g.pdf.SetTextColor(pdfColorGrayLight, pdfColorGrayLight, pdfColorGrayLight)
		g.pdf.CellFormat(0, pdfSpacingSection, g.options.CompanyName, "", 1, "C", false, 0, "")
	}

	// Main title
	g.pdf.Ln(pdfSpacingCover)
	g.pdf.SetFont("Arial", "B", pdfFontSizeTitle)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, pdfSpacingTitle, "WiFi Site Survey Report", "", 1, "C", false, 0, "")

	// Survey name
	g.pdf.Ln(pdfSpacingSection)
	g.pdf.SetFont("Arial", "", pdfFontSizeSubtitle)
	g.pdf.SetTextColor(pdfColorGrayDark, pdfColorGrayDark, pdfColorGrayDark)
	g.pdf.CellFormat(0, pdfSpacingSection, g.survey.Name, "", 1, "C", false, 0, "")

	// Description if provided
	if g.survey.Description != "" {
		g.pdf.Ln(pdfSpacingSmall)
		g.pdf.SetFont("Arial", "I", pdfFontSizeBody)
		g.pdf.SetTextColor(pdfColorGrayLight, pdfColorGrayLight, pdfColorGrayLight)
		g.pdf.MultiCell(0, pdfSpacingMedium, g.survey.Description, "", "C", false)
	}

	// Date info
	g.pdf.Ln(pdfSpacingHuge)
	g.pdf.SetFont("Arial", "", pdfFontSizeBody)
	g.pdf.SetTextColor(pdfColorGrayMedium, pdfColorGrayMedium, pdfColorGrayMedium)

	dateStr := time.Now().Format("January 2, 2006")
	g.pdf.CellFormat(0, pdfSpacingLarge, fmt.Sprintf("Report Generated: %s", dateStr), "", 1, "C", false, 0, "")

	surveyDate := g.survey.CreatedAt.Format("January 2, 2006")
	g.pdf.CellFormat(0, pdfSpacingLarge, fmt.Sprintf("Survey Date: %s", surveyDate), "", 1, "C", false, 0, "")

	// Status
	g.pdf.Ln(pdfSpacingSmall)
	g.pdf.SetFont("Arial", "B", pdfFontSizeBody)
	statusColor := getStatusColor(g.survey.Status)
	g.pdf.SetTextColor(statusColor[0], statusColor[1], statusColor[2])
	g.pdf.CellFormat(0, pdfSpacingLarge, fmt.Sprintf("Status: %s", g.survey.Status), "", 1, "C", false, 0, "")

	// Building info
	g.pdf.Ln(pdfSpacingMajor)
	g.pdf.SetFont("Arial", "", pdfFontSizeNormal)
	g.pdf.SetTextColor(pdfColorGrayMedium, pdfColorGrayMedium, pdfColorGrayMedium)

	floorCount := len(g.survey.Floors)
	sampleCount := len(g.survey.GetAllSamples())
	g.pdf.CellFormat(
		0,
		pdfSpacingNormal,
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
	g.pdf.Ln(pdfSpacingSmall)
	g.addStatCard(
		"Overall Coverage Score",
		fmt.Sprintf("%.0f%%", stats.CoverageScore),
		getCoverageGrade(stats.CoverageScore),
	)

	// Key metrics table
	g.pdf.Ln(pdfSpacingSection)
	g.pdf.SetFont("Arial", "B", pdfFontSizeBody)
	g.pdf.CellFormat(0, pdfSpacingLarge, "Key Metrics", "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
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
	g.pdf.Ln(pdfSpacingSection)
	g.pdf.SetFont("Arial", "B", pdfFontSizeBody)
	g.pdf.CellFormat(0, pdfSpacingLarge, "Signal Quality Distribution", "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
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
	g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
	g.pdf.SetTextColor(pdfColorGrayMedium, pdfColorGrayMedium, pdfColorGrayMedium)
	g.pdf.CellFormat(
		0,
		pdfSpacingMedium,
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
			pdfSpacingMedium,
			fmt.Sprintf("Dimensions: %d x %d px", floor.FloorPlan.Width, floor.FloorPlan.Height),
			"",
			1,
			"L",
			false,
			0,
			"",
		)
		if floor.FloorPlan.ScaleM > 0 {
			g.pdf.CellFormat(
				0,
				pdfSpacingMedium,
				fmt.Sprintf("Scale: %.2f m/px", floor.FloorPlan.ScaleM),
				"",
				1,
				"L",
				false,
				0,
				"",
			)
		}
	}

	// Floor statistics
	if len(floor.Samples) > 0 {
		g.pdf.Ln(pdfSpacingSmall)
		stats := calculateFloorStats(floor.Samples)

		g.pdf.SetFont("Arial", "B", pdfFontSizeNormal)
		g.pdf.SetTextColor(0, 0, 0)
		g.pdf.CellFormat(0, pdfSpacingLarge, "Coverage Statistics", "", 1, "L", false, 0, "")

		g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
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
			g.pdf.Ln(pdfSpacingSmall)
			g.pdf.SetFont("Arial", "B", pdfFontSizeNormal)
			g.pdf.CellFormat(0, pdfSpacingLarge, "WiFi Channels Detected", "", 1, "L", false, 0, "")

			g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
			for _, ch := range channels {
				g.pdf.CellFormat(
					0,
					pdfSpacingMedium,
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
		g.pdf.Ln(pdfSpacingSmall)
		g.pdf.SetFont("Arial", "I", pdfFontSizeSmall)
		g.pdf.SetTextColor(pdfColorGrayMuted, pdfColorGrayMuted, pdfColorGrayMuted)
		g.pdf.CellFormat(0, pdfSpacingLarge, "No samples collected for this floor", "", 1, "L", false, 0, "")
	}

	// Add heatmap if requested and floor has data
	if g.options.IncludeHeatmaps && len(floor.Samples) > 0 && floor.FloorPlan != nil {
		g.addFloorHeatmapNote(floor)
	}
}

// addFloorHeatmapNote adds a note about heatmap availability.
func (g *ReportGenerator) addFloorHeatmapNote(_ *Floor) {
	g.pdf.Ln(pdfSpacingSection)
	g.pdf.SetFont("Arial", "I", pdfFontSizeSmall)
	g.pdf.SetTextColor(pdfColorGrayMedium, pdfColorGrayMedium, pdfColorGrayMedium)
	g.pdf.CellFormat(
		0,
		pdfSpacingMedium,
		"Heatmap visualization available in the web interface.",
		"",
		1,
		"L",
		false,
		0,
		"",
	)
}

// addRecommendations adds the recommendations section.
func (g *ReportGenerator) addRecommendations() {
	g.pdf.AddPage()
	g.addSectionHeader("Recommendations")

	allSamples := g.survey.GetAllSamples()
	stats := calculateSurveyStats(allSamples)

	recommendations := generateSurveyRecommendations(&stats)

	if len(recommendations) == 0 {
		g.pdf.SetFont("Arial", "I", pdfFontSizeSmall)
		g.pdf.SetTextColor(pdfColorGrayLight, pdfColorGrayLight, pdfColorGrayLight)
		g.pdf.CellFormat(
			0,
			pdfSpacingLarge,
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

	g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
	g.pdf.SetTextColor(0, 0, 0)

	for i, rec := range recommendations {
		g.pdf.Ln(pdfSpacingTiny)
		priority := getPriorityLabel(rec.Priority)

		// Priority badge
		g.pdf.SetFont("Arial", "B", pdfFontSizePriority)
		switch rec.Priority {
		case PriorityHigh:
			g.pdf.SetTextColor(priorityHighColorR, priorityHighColorG, priorityHighColorB)
		case PriorityMedium:
			g.pdf.SetTextColor(priorityMediumColorR, priorityMediumColorG, priorityMediumColorB)
		case PriorityLow:
			g.pdf.SetTextColor(priorityLowColorR, priorityLowColorG, priorityLowColorB)
		}
		g.pdf.CellFormat(pdfSpacingPriority, pdfSpacingMedium, priority, "1", 0, "C", false, 0, "")

		// Recommendation text
		g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
		g.pdf.SetTextColor(0, 0, 0)
		g.pdf.MultiCell(0, pdfSpacingMedium, fmt.Sprintf(" %d. %s", i+1, rec.Text), "", "L", false)
	}

	// Implementation notes
	g.pdf.Ln(pdfSpacingSection)
	g.pdf.SetFont("Arial", "B", pdfFontSizeNormal)
	g.pdf.CellFormat(0, pdfSpacingLarge, "Implementation Notes", "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "", pdfFontSizeSmall)
	notes := []string{
		"High priority items should be addressed within 1-2 weeks",
		"Consider WiFi 6/6E access points for improved capacity",
		"Verify power levels and channel assignments after changes",
		"Re-survey affected areas after implementing changes",
	}

	for _, note := range notes {
		g.pdf.SetTextColor(pdfColorGrayMedium, pdfColorGrayMedium, pdfColorGrayMedium)
		g.pdf.CellFormat(pdfSpacingSmall, pdfSpacingMedium, "-", "", 0, "L", false, 0, "")
		g.pdf.CellFormat(0, pdfSpacingMedium, note, "", 1, "L", false, 0, "")
	}
}

// addRawDataAppendix adds the raw data appendix section.
func (g *ReportGenerator) addRawDataAppendix() {
	g.pdf.AddPage()
	g.addSectionHeader("Appendix: Raw Sample Data")

	allSamples := g.survey.GetAllSamples()
	if len(allSamples) == 0 {
		g.pdf.SetFont("Arial", "I", pdfFontSizeSmall)
		g.pdf.CellFormat(0, pdfSpacingLarge, "No sample data collected", "", 1, "L", false, 0, "")
		return
	}

	// Table header
	g.pdf.SetFont("Arial", "B", pdfFontSizeTableHeader)
	g.pdf.SetFillColor(pdfColorGrayTableBg, pdfColorGrayTableBg, pdfColorGrayTableBg)
	headers := []struct {
		text  string
		width float64
	}{
		{"#", tableColIndexWidth},
		{"X", tableColCoordWidth},
		{"Y", tableColCoordWidth},
		{"RSSI", tableColRSSIWidth},
		{"SNR", tableColSNRWidth},
		{"SSID", tableColSSIDWidth},
		{"Channel", tableColChannelWidth},
		{"Time", tableColTimeWidth},
	}

	for _, h := range headers {
		g.pdf.CellFormat(h.width, pdfSpacingNormal, h.text, "1", 0, "C", true, 0, "")
	}
	g.pdf.Ln(-1)

	// Table rows (limit samples per page section)
	g.pdf.SetFont("Arial", "", pdfFontSizeTiny)
	maxSamples := min(len(allSamples), pdfMaxSamplesInTable)

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
			ssid = truncateString(net.SSID, pdfSSIDMaxLength)
			channel = strconv.Itoa(net.Channel)
		}

		g.pdf.CellFormat(tableColIndexWidth, pdfSpacingMedium, strconv.Itoa(i+1), "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(tableColCoordWidth, pdfSpacingMedium, strconv.Itoa(sample.X), "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(tableColCoordWidth, pdfSpacingMedium, strconv.Itoa(sample.Y), "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(tableColRSSIWidth, pdfSpacingMedium, rssi, "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(tableColSNRWidth, pdfSpacingMedium, snr, "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(tableColSSIDWidth, pdfSpacingMedium, ssid, "1", 0, "L", false, 0, "")
		g.pdf.CellFormat(tableColChannelWidth, pdfSpacingMedium, channel, "1", 0, "C", false, 0, "")
		g.pdf.CellFormat(
			tableColTimeWidth,
			pdfSpacingMedium,
			sample.Timestamp.Format("15:04:05"),
			"1",
			0,
			"C",
			false,
			0,
			"",
		)
		g.pdf.Ln(-1)
	}

	if len(allSamples) > maxSamples {
		g.pdf.Ln(pdfSpacingSmall)
		g.pdf.SetFont("Arial", "I", pdfFontSizePriority)
		g.pdf.CellFormat(
			0,
			pdfSpacingMedium,
			fmt.Sprintf(
				"... and %d more samples (truncated for readability)",
				len(allSamples)-maxSamples,
			),
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
	g.pdf.SetFont("Arial", "B", pdfFontSizeSectionHead)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, pdfSpacingSectionTitle, title, "", 1, "L", false, 0, "")

	// Underline
	g.pdf.SetDrawColor(pdfColorGrayLine, pdfColorGrayLine, pdfColorGrayLine)
	g.pdf.Line(pdfLineStartX, g.pdf.GetY(), pdfLineEndX, g.pdf.GetY())
	g.pdf.Ln(pdfSpacingSmall)
}

func (g *ReportGenerator) addStatCard(label, value, grade string) {
	g.pdf.SetFont("Arial", "B", pdfFontSizeStatLabel)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, pdfSpacingSection, label, "", 1, "L", false, 0, "")

	g.pdf.SetFont("Arial", "B", pdfFontSizeStatValue)
	gradeColor := getGradeColor(grade)
	g.pdf.SetTextColor(gradeColor[0], gradeColor[1], gradeColor[2])
	g.pdf.CellFormat(pdfStatValueWidth, pdfSpacingMajor, value, "", 0, "L", false, 0, "")

	g.pdf.SetFont("Arial", "B", pdfFontSizeStatGrade)
	g.pdf.CellFormat(0, pdfSpacingMajor, grade, "", 1, "L", false, 0, "")
}

func (g *ReportGenerator) addMetricRow(label, value string) {
	g.pdf.SetTextColor(pdfColorGrayMedium, pdfColorGrayMedium, pdfColorGrayMedium)
	g.pdf.CellFormat(pdfMetricLabelWidth, pdfSpacingMedium, label+":", "", 0, "L", false, 0, "")
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(0, pdfSpacingMedium, value, "", 1, "L", false, 0, "")
}

func (g *ReportGenerator) addDistributionBar(label string, percent float64, col []int) {
	g.pdf.SetTextColor(pdfColorGrayMedium, pdfColorGrayMedium, pdfColorGrayMedium)
	g.pdf.CellFormat(pdfDistLabelWidth, pdfSpacingMedium, label, "", 0, "L", false, 0, "")

	// Draw bar background
	barX := g.pdf.GetX()
	barY := g.pdf.GetY()
	barWidth := float64(pdfDistBarWidth)
	barHeight := float64(pdfDistBarHeight)

	g.pdf.SetFillColor(pdfDistBarBgColor, pdfDistBarBgColor, pdfDistBarBgColor)
	g.pdf.Rect(barX, barY, barWidth, barHeight, "F")

	// Draw filled portion
	filledWidth := barWidth * (percent / percentMultiplier)
	g.pdf.SetFillColor(col[0], col[1], col[2])
	g.pdf.Rect(barX, barY, filledWidth, barHeight, "F")

	// Percentage label
	g.pdf.SetX(barX + barWidth + pdfDistBarGap)
	g.pdf.SetTextColor(0, 0, 0)
	g.pdf.CellFormat(pdfDistPercentWidth, pdfSpacingMedium, fmt.Sprintf("%.1f%%", percent), "", 1, "L", false, 0, "")
}
