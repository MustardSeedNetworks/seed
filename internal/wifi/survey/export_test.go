package survey

import (
	"image"
	"image/color"
)

// ExportInterpolateColor exports interpolateColor for testing.
func ExportInterpolateColor(stop1, stop2 ColorStop, value float64) color.RGBA {
	return interpolateColor(stop1, stop2, value)
}

// ExportFilterWeakSamples exports filterWeakSamples for testing.
func ExportFilterWeakSamples(samples []SampleValue, threshold float64) []SampleValue {
	return filterWeakSamples(samples, threshold)
}

// ExportCalculateCoverageScore exports calculateCoverageScore for testing.
func ExportCalculateCoverageScore(allSamples, weakSamples []SampleValue) float64 {
	return calculateCoverageScore(allSamples, weakSamples)
}

// ExportDetermineSeverity exports determineSeverity for testing.
func ExportDetermineSeverity(avgRSSI float64) string {
	return determineSeverity(avgRSSI)
}

// ExportGenerateRecommendations exports generateRecommendations for testing.
func ExportGenerateRecommendations(
	deadZones []DeadZone,
	coverageScore float64,
	totalSamples int,
) []string {
	return generateRecommendations(deadZones, coverageScore, totalSamples)
}

// SetSurvey sets a survey in the manager for testing.
func (m *Manager) SetSurvey(s *Survey) {
	m.surveys[s.ID] = s
}

// GetStoragePath returns the storage path for testing.
func (m *Manager) GetStoragePath() string {
	return m.storagePath
}

// GetSurveys returns the surveys map for testing.
func (m *Manager) GetSurveys() map[string]*Survey {
	return m.surveys
}

// GetWifiScanner returns the WiFi scanner for testing.
func (m *Manager) GetWifiScanner() any {
	return m.wifiScanner
}

// GetWifiManager returns the WiFi manager for testing.
func (m *Manager) GetWifiManager() any {
	return m.wifiManager
}

// GetIperfManager returns the iperf manager for testing.
func (m *Manager) GetIperfManager() any {
	return m.iperfManager
}

// ExportDistance exports distance for testing.
func ExportDistance(p1, p2 Point2D) float64 {
	return distance(p1, p2)
}

// ExportExtractValue exports extractValue for testing.
func ExportExtractValue(sampleData any, valueType string) float64 {
	return extractValue(sampleData, valueType)
}

// ExportExtractPassiveValue exports extractPassiveValue for testing.
func ExportExtractPassiveValue(data *PassiveSample, valueType string) float64 {
	return extractPassiveValue(data, valueType)
}

// ExportExtractActiveValue exports extractActiveValue for testing.
func ExportExtractActiveValue(data *ActiveSample, valueType string) float64 {
	return extractActiveValue(data, valueType)
}

// ExportExtractThroughputValue exports extractThroughputValue for testing.
func ExportExtractThroughputValue(data *ThroughputSample, valueType string) float64 {
	return extractThroughputValue(data, valueType)
}

// ExportExtractMapValue exports extractMapValue for testing.
func ExportExtractMapValue(data map[string]any, valueType string) float64 {
	return extractMapValue(data, valueType)
}

// ExportGetHeatmapDimensions exports getHeatmapDimensions for testing.
func ExportGetHeatmapDimensions(s *Survey) (int, int) {
	return getHeatmapDimensions(s)
}

// ExportMapHeatmapTypeToValueType exports mapHeatmapTypeToValueType for testing.
func ExportMapHeatmapTypeToValueType(ht HeatmapType) string {
	return mapHeatmapTypeToValueType(ht)
}

// ExportGetColorScaleForType exports getColorScaleForType for testing.
func ExportGetColorScaleForType(ht HeatmapType) ColorScale {
	return getColorScaleForType(ht)
}

// ExportRenderHeatmapToImage exports renderHeatmapToImage for testing.
func ExportRenderHeatmapToImage(
	img *image.RGBA,
	grid [][]float64,
	cellSize int,
	scale *ColorScale,
	opacity uint8,
) {
	renderHeatmapToImage(img, grid, cellSize, scale, opacity)
}

// ExportRenderSamplePoints exports renderSamplePoints for testing.
func ExportRenderSamplePoints(img *image.RGBA, samples []SampleValue) {
	renderSamplePoints(img, samples)
}

// ExportRenderGrid exports renderGrid for testing.
func ExportRenderGrid(img *image.RGBA, cellSize int) {
	renderGrid(img, cellSize)
}

// CreateTestImage creates a test image for testing.
func CreateTestImage(width, height int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, width, height))
}

// ExportTruncateString exports truncateString for testing.
func ExportTruncateString(s string, maxLen int) string {
	return truncateString(s, maxLen)
}

// ExportGetCoverageGrade exports getCoverageGrade for testing.
func ExportGetCoverageGrade(score float64) string {
	return getCoverageGrade(score)
}

// ExportGetGradeColor exports getGradeColor for testing.
func ExportGetGradeColor(grade string) []int {
	return getGradeColor(grade)
}

// ExportGetStatusColor exports getStatusColor for testing.
func ExportGetStatusColor(status Status) []int {
	return getStatusColor(status)
}

// ExportGetPriorityLabel exports getPriorityLabel for testing.
func ExportGetPriorityLabel(p RecommendationPriority) string {
	return getPriorityLabel(p)
}

// ExportGenerateSurveyRecommendations exports generateSurveyRecommendations for testing.
func ExportGenerateSurveyRecommendations(stats *SurveyStats) []Recommendation {
	return generateSurveyRecommendations(stats)
}

// ExportCalculateSurveyStats exports calculateSurveyStats for testing.
func ExportCalculateSurveyStats(samples []*SamplePoint) SurveyStats {
	return calculateSurveyStats(samples)
}

// ExportGetChannelUsage exports getChannelUsage for testing.
func ExportGetChannelUsage(samples []*SamplePoint) []ChannelInfo {
	return getChannelUsage(samples)
}

// ExportGetPassiveSampleFromPoint exports getPassiveSampleFromPoint for testing.
func ExportGetPassiveSampleFromPoint(sp *SamplePoint) *PassiveSample {
	return getPassiveSampleFromPoint(sp)
}
