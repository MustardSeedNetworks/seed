package ai

// ExportCalculateDistance exposes calculateDistance for testing.
func ExportCalculateDistance(p1, p2 Point) float64 {
	return calculateDistance(p1, p2)
}

// ExportCountDeadZones exposes countDeadZones for testing.
func ExportCountDeadZones(samples []SignalSample, threshold int) int {
	return countDeadZones(samples, threshold)
}

// ExportGenerateRecommendations exposes generateRecommendations for testing.
func ExportGenerateRecommendations(coverage, avgRSSI float64, deadZones, sampleCount int) []string {
	return generateRecommendations(coverage, avgRSSI, deadZones, sampleCount)
}

// ExportEstimateCoverageRadius exposes estimateCoverageRadius for testing.
func ExportEstimateCoverageRadius(environment, band string) float64 {
	return estimateCoverageRadius(environment, band)
}

// ExportEstimateCoverageGain exposes estimateCoverageGain for testing.
func ExportEstimateCoverageGain(currentRSSI, threshold int) float64 {
	return estimateCoverageGain(currentRSSI, threshold)
}

// ExportClassifyPlacementReason exposes classifyPlacementReason for testing.
func ExportClassifyPlacementReason(rssi int) string {
	return classifyPlacementReason(rssi)
}

// ExportFindOptimalPlacements exposes findOptimalPlacements for testing.
func ExportFindOptimalPlacements(
	floorPlan *FloorPlan,
	existingAPs []AccessPoint,
	targetCoverage float64,
	threshold int,
) []PlacementSuggestion {
	return findOptimalPlacements(floorPlan, existingAPs, targetCoverage, threshold)
}
