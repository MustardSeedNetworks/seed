package harvest

// services_pdf.go contains the PDF rendering pipeline for GeneratorService:
// generatePDF orchestrates a multi-page report; each addPDFXxx helper renders
// one section.

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"
)

// generatePDF creates a PDF report.
func (s *GeneratorService) generatePDF(report *Report, data *AggregatedData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, pdfPageMarginBottom)

	// Cover page
	pdf.AddPage()
	s.addPDFCover(pdf, report)

	// Executive summary
	pdf.AddPage()
	s.addPDFExecutiveSummary(pdf, data)

	// Device inventory section
	pdf.AddPage()
	s.addPDFDeviceSection(pdf, data)

	// Vulnerability section (if applicable)
	if report.Type == ReportTypeVulnerability || report.Type == ReportTypeExecutive ||
		report.Type == ReportTypeDetailed {
		pdf.AddPage()
		s.addPDFVulnerabilitySection(pdf, data)
	}

	// Performance section
	if report.Type == ReportTypePerformance || report.Type == ReportTypeExecutive ||
		report.Type == ReportTypeDetailed {
		pdf.AddPage()
		s.addPDFPerformanceSection(pdf, data)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *GeneratorService) addPDFCover(pdf *fpdf.Fpdf, report *Report) {
	pdf.SetFont("Arial", "B", pdfFontSizeCoverTitle)
	pdf.Ln(pdfCoverTopMargin)
	pdf.CellFormat(0, pdfCellHeightTitle, "Network Report", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", pdfFontSizeCoverName)
	pdf.Ln(pdfSectionSpacingMed)
	pdf.CellFormat(0, pdfCellHeightSubtitle, report.Name, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", pdfFontSizeCoverMeta)
	pdf.Ln(pdfSectionSpacingLarge)
	pdf.SetTextColor(pdfColorGrayMid, pdfColorGrayMid, pdfColorGrayMid)
	pdf.CellFormat(
		0,
		pdfCellHeightBody,
		fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 15:04")),
		"",
		1,
		"C",
		false,
		0,
		"",
	)
	pdf.CellFormat(0, pdfCellHeightBody, fmt.Sprintf("Type: %s", report.Type), "", 1, "C", false, 0, "")
}

func (s *GeneratorService) addPDFExecutiveSummary(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Executive Summary")

	pdf.SetFont("Arial", "", pdfFontSizeBody)
	pdf.SetTextColor(0, 0, 0)

	metrics := []struct {
		label string
		value string
	}{
		{
			"Report Period",
			fmt.Sprintf(
				"%s to %s",
				data.StartDate.Format("Jan 2"),
				data.EndDate.Format("Jan 2, 2006"),
			),
		},
		{"Total Devices", strconv.Itoa(data.DeviceCount)},
		{"Total Vulnerabilities", strconv.Itoa(data.VulnCount.Total)},
		{"Critical Issues", strconv.Itoa(data.VulnCount.Critical)},
		{"Average Latency", fmt.Sprintf("%.1f ms", data.Performance.AvgLatencyMs)},
		{"Uptime", fmt.Sprintf("%.1f%%", data.Performance.UptimePercent)},
	}

	for _, m := range metrics {
		pdf.SetTextColor(pdfColorGrayDark, pdfColorGrayDark, pdfColorGrayDark)
		pdf.CellFormat(pdfLabelColumnWidth, pdfCellHeightMetric, m.label+":", "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, pdfCellHeightMetric, m.value, "", 1, "L", false, 0, "")
	}
}

func (s *GeneratorService) addPDFDeviceSection(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Device Inventory")

	pdf.SetFont("Arial", "", pdfFontSizeBody)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(
		0,
		pdfCellHeightBody,
		fmt.Sprintf("Total devices discovered: %d", data.DeviceCount),
		"",
		1,
		"L",
		false,
		0,
		"",
	)
}

func (s *GeneratorService) addPDFVulnerabilitySection(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Vulnerability Assessment")

	// Severity breakdown
	pdf.SetFont("Arial", "B", pdfFontSizeSubsection)
	pdf.CellFormat(0, pdfCellHeightBody, "Severity Distribution", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", pdfFontSizeBody)
	severities := []struct {
		label string
		count int
		color []int
	}{
		{"Critical", data.VulnCount.Critical, []int{220, 53, 69}},
		{"High", data.VulnCount.High, []int{255, 128, 0}},
		{"Medium", data.VulnCount.Medium, []int{255, 193, 7}},
		{"Low", data.VulnCount.Low, []int{40, 167, 69}},
	}

	for _, sev := range severities {
		pdf.SetTextColor(sev.color[0], sev.color[1], sev.color[2])
		pdf.CellFormat(pdfSeverityLabelWidth, pdfCellHeightSeverity, sev.label+":", "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, pdfCellHeightSeverity, strconv.Itoa(sev.count), "", 1, "L", false, 0, "")
	}

	// Top issues
	if len(data.TopIssues) > 0 {
		pdf.Ln(pdfSectionSpacingSmall)
		pdf.SetFont("Arial", "B", pdfFontSizeSubsection)
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, pdfCellHeightBody, "Top Issues", "", 1, "L", false, 0, "")

		pdf.SetFont("Arial", "", pdfFontSizeSmall)
		for i, issue := range data.TopIssues {
			if i >= topIssuesDisplayLimit {
				break
			}
			pdf.CellFormat(
				0,
				pdfCellHeightSeverity,
				fmt.Sprintf("%d. %s (%d occurrences)", i+1, issue.Description, issue.Count),
				"",
				1,
				"L",
				false,
				0,
				"",
			)
		}
	}
}

func (s *GeneratorService) addPDFPerformanceSection(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Performance Metrics")

	pdf.SetFont("Arial", "", pdfFontSizeBody)
	pdf.SetTextColor(0, 0, 0)

	metrics := []struct {
		label string
		value string
	}{
		{"Average Latency", fmt.Sprintf("%.2f ms", data.Performance.AvgLatencyMs)},
		{"Packet Loss", fmt.Sprintf("%.2f%%", data.Performance.AvgPacketLoss)},
		{"Average Bandwidth", fmt.Sprintf("%.2f Mbps", data.Performance.AvgBandwidthMbps)},
		{"Uptime", fmt.Sprintf("%.2f%%", data.Performance.UptimePercent)},
	}

	for _, m := range metrics {
		pdf.SetTextColor(pdfColorGrayDark, pdfColorGrayDark, pdfColorGrayDark)
		pdf.CellFormat(pdfLabelColumnWidth, pdfCellHeightMetric, m.label+":", "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, pdfCellHeightMetric, m.value, "", 1, "L", false, 0, "")
	}
}

func (s *GeneratorService) addPDFSectionHeader(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Arial", "B", pdfFontSizeSection)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, pdfCellHeightSubtitle, title, "", 1, "L", false, 0, "")

	pdf.SetDrawColor(pdfColorGrayLight, pdfColorGrayLight, pdfColorGrayLight)
	pdf.Line(pdfLineStartX, pdf.GetY(), pdfLineEndX, pdf.GetY())
	pdf.Ln(pdfSectionSpacingSmall)
}
