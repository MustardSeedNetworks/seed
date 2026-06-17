package survey

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-pdf/fpdf"
)

// PDF layout constants for spacing, margins, and positioning.
const (
	// Page margins and breaks.
	pdfPageBreakMargin = 15

	// Spacing constants (in mm).
	pdfSpacingTiny         = 3  // Tiny spacing between elements
	pdfSpacingSmall        = 5  // Small spacing between elements
	pdfSpacingMedium       = 6  // Medium spacing for multi-cell content
	pdfSpacingNormal       = 7  // Normal line spacing
	pdfSpacingLarge        = 8  // Large spacing for cells
	pdfSpacingSection      = 10 // Section separation
	pdfSpacingSectionTitle = 12 // Section header cell height
	pdfSpacingTitle        = 15 // Title cell height
	pdfSpacingPriority     = 20 // Priority badge width
	pdfSpacingMajor        = 20 // Major section separation
	pdfSpacingHuge         = 30 // Large vertical gap
	pdfSpacingCover        = 50 // Cover page title offset

	// Font sizes.
	pdfFontSizeTiny        = 7  // Very small text (table data)
	pdfFontSizeTableHeader = 8  // Table header text
	pdfFontSizePriority    = 9  // Priority badge text
	pdfFontSizeSmall       = 10 // Small body text
	pdfFontSizeNormal      = 11 // Normal body text
	pdfFontSizeBody        = 12 // Standard body text
	pdfFontSizeSectionHead = 16 // Section header text
	pdfFontSizeSubtitle    = 18 // Subtitle text
	pdfFontSizeTitle       = 28 // Main title text

	// Text colors (grayscale values).
	pdfColorGrayDark    = 60  // Dark gray text
	pdfColorGrayMedium  = 80  // Medium gray text
	pdfColorGrayLight   = 100 // Light gray text
	pdfColorGrayMuted   = 150 // Muted gray for disabled text
	pdfColorGrayLine    = 200 // Gray for line drawing
	pdfColorGrayTableBg = 240 // Light gray for table header background

	// Table column widths for raw data appendix.
	tableColIndexWidth   = 8
	tableColCoordWidth   = 15
	tableColRSSIWidth    = 20
	tableColSNRWidth     = 20
	tableColSSIDWidth    = 50
	tableColChannelWidth = 20
	tableColTimeWidth    = 35

	// Miscellaneous layout constants.
	pdfLineStartX        = 10  // Line drawing start X position
	pdfLineEndX          = 200 // Line drawing end X position
	pdfMaxSamplesInTable = 50  // Maximum samples to display in raw data table
	pdfSSIDMaxLength     = 15  // Maximum SSID length before truncation

	// Priority badge colors (RGB).
	priorityHighColorR   = 220
	priorityHighColorG   = 53
	priorityHighColorB   = 69
	priorityMediumColorR = 255
	priorityMediumColorG = 128
	priorityMediumColorB = 0
	priorityLowColorR    = 40
	priorityLowColorG    = 167
	priorityLowColorB    = 69

	// Stat card layout constants.
	pdfFontSizeStatLabel = 14 // Stat card label font size
	pdfFontSizeStatValue = 36 // Stat card value font size
	pdfFontSizeStatGrade = 24 // Stat card grade font size
	pdfStatValueWidth    = 60 // Stat card value cell width

	// Distribution bar constants.
	pdfDistLabelWidth   = 60  // Distribution bar label width
	pdfDistBarWidth     = 80  // Distribution bar width
	pdfDistBarHeight    = 5   // Distribution bar height
	pdfDistBarGap       = 2   // Gap after distribution bar
	pdfDistPercentWidth = 20  // Percentage label width
	pdfDistBarBgColor   = 230 // Distribution bar background color

	// Metric row constants.
	pdfMetricLabelWidth = 80 // Metric row label width

	// Percentage and score constants.
	percentMultiplier     = 100 // Multiplier for percentage calculation
	fairCoverageWeight    = 0.5 // Weight for fair coverage in score calculation
	topChannelsLimit      = 5   // Maximum number of top channels to show
	minSampleThreshold    = 20  // Minimum samples for accurate analysis
	coverageScoreCritical = 50  // Coverage score threshold: critical
	coverageScorePoor     = 70  // Coverage score threshold: poor
	coverageScoreModerate = 85  // Coverage score threshold: moderate
	coverageScoreGood     = 90  // Coverage score threshold: good (for grade)
	coverageGradeB        = 80  // Minimum score for grade B
	coverageGradeC        = 70  // Minimum score for grade C
	coverageGradeD        = 60  // Minimum score for grade D

	// Signal strength constants.
	minRSSIValue = -100 // Minimum RSSI value in dBm (worst possible signal)
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
	g.pdf.SetAutoPageBreak(true, pdfPageBreakMargin)

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

// GenerateReport creates a PDF report for the specified survey.
func (m *Manager) GenerateReport(surveyID string, options ReportOptions) ([]byte, error) {
	survey, err := m.GetSurvey(surveyID)
	if err != nil {
		return nil, err
	}

	generator := NewReportGenerator(survey, options)
	return generator.Generate()
}
