// Package ai provides AI-assisted WiFi planning capabilities.
// This package offers signal prediction, AP placement optimization,
// and coverage analysis for WiFi network planning.
package ai

import (
	"errors"
	"math"
	"sort"
)

// Common errors returned by AI functions.
var (
	// ErrNoData is returned when there is insufficient data for analysis.
	ErrNoData = errors.New("no data provided for analysis")

	// ErrInvalidInput is returned when input parameters are invalid.
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrNoFloorPlan is returned when floor plan is required but not provided.
	ErrNoFloorPlan = errors.New("floor plan required for analysis")
)

// Point represents a 2D coordinate.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// SignalSample represents a WiFi signal measurement at a location.
type SignalSample struct {
	Location Point   `json:"location"`
	RSSI     int     `json:"rssiDbm"`   // Signal strength in dBm
	SSID     string  `json:"ssid"`      // Network SSID
	BSSID    string  `json:"bssid"`     // Access point MAC
	Channel  int     `json:"channel"`   // WiFi channel
	Band     string  `json:"band"`      // "2.4GHz", "5GHz", or "6GHz"
	Distance float64 `json:"distanceM"` // Distance from AP in meters (if known)
}

// FloorPlan represents the physical space being analyzed.
type FloorPlan struct {
	Width  float64 `json:"widthM"`  // Width in meters
	Height float64 `json:"heightM"` // Height in meters
}

// AccessPoint represents a WiFi access point for planning.
type AccessPoint struct {
	Location    Point   `json:"location"`
	TxPower     int     `json:"txPowerDbm"`  // Transmit power in dBm
	Band        string  `json:"band"`        // "2.4GHz", "5GHz", or "6GHz"
	AntennaGain float64 `json:"antennaGain"` // Antenna gain in dBi
}

// CoverageResult contains the results of coverage analysis.
type CoverageResult struct {
	TotalArea        float64        `json:"totalAreaSqM"`
	CoveredArea      float64        `json:"coveredAreaSqM"`
	CoveragePercent  float64        `json:"coveragePercent"`
	AverageRSSI      float64        `json:"averageRssiDbm"`
	MinRSSI          int            `json:"minRssiDbm"`
	MaxRSSI          int            `json:"maxRssiDbm"`
	DeadZoneCount    int            `json:"deadZoneCount"`
	Recommendations  []string       `json:"recommendations"`
	PredictedHeatmap []HeatmapPoint `json:"predictedHeatmap,omitempty"`
}

// HeatmapPoint represents signal strength at a specific location.
type HeatmapPoint struct {
	Location Point `json:"location"`
	RSSI     int   `json:"rssiDbm"`
}

// PlacementSuggestion represents a recommended AP placement.
type PlacementSuggestion struct {
	Location     Point   `json:"location"`
	Priority     int     `json:"priority"` // 1 = highest priority
	Reason       string  `json:"reason"`
	CoverageGain float64 `json:"coverageGainPercent"` // Estimated coverage improvement
}

// PathLossModel defines parameters for signal propagation.
type PathLossModel struct {
	ReferenceDistance float64 `json:"referenceDistanceM"` // Reference distance (typically 1m)
	ReferenceLoss     float64 `json:"referenceLossDb"`    // Loss at reference distance
	PathLossExponent  float64 `json:"pathLossExponent"`   // Environment-specific exponent
	WallAttenuation   float64 `json:"wallAttenuationDb"`  // Attenuation per wall
}

// Signal threshold constants in dBm.
const (
	// ThresholdExcellent is the RSSI threshold for excellent signal (-50 dBm).
	ThresholdExcellent = -50

	// ThresholdGood is the RSSI threshold for good signal (-65 dBm).
	ThresholdGood = -65

	// ThresholdFair is the RSSI threshold for fair/usable signal (-75 dBm).
	ThresholdFair = -75

	// ThresholdPoor is the RSSI threshold for poor signal (-85 dBm).
	ThresholdPoor = -85

	// ThresholdMinimum is the minimum usable signal threshold (-90 dBm).
	ThresholdMinimum = -90
)

// Path loss model presets.
const (
	// PathLossExponentFreeSpace is the path loss exponent for free space.
	PathLossExponentFreeSpace = 2.0

	// PathLossExponentOffice is the path loss exponent for typical office environments.
	PathLossExponentOffice = 3.0

	// PathLossExponentResidential is the path loss exponent for residential buildings.
	PathLossExponentResidential = 2.8

	// PathLossExponentWarehouse is the path loss exponent for open warehouse/industrial.
	PathLossExponentWarehouse = 2.2

	// DefaultWallAttenuation is the typical wall attenuation in dB.
	DefaultWallAttenuation = 3.0

	// DefaultReferenceLoss2_4GHz is the reference loss at 1m for 2.4 GHz.
	DefaultReferenceLoss2_4GHz = 40.0

	// DefaultReferenceLoss5GHz is the reference loss at 1m for 5 GHz.
	DefaultReferenceLoss5GHz = 47.0

	// DefaultReferenceLoss6GHz is the reference loss at 1m for 6 GHz.
	DefaultReferenceLoss6GHz = 49.0
)

// Grid resolution for analysis.
const (
	// DefaultGridResolution is the default grid cell size in meters.
	DefaultGridResolution = 1.0

	// MinGridResolution is the minimum grid cell size in meters.
	MinGridResolution = 0.5

	// MaxGridResolution is the maximum grid cell size in meters.
	MaxGridResolution = 5.0
)

// NewPathLossModel creates a path loss model for the specified environment.
func NewPathLossModel(environment, band string) *PathLossModel {
	model := &PathLossModel{
		ReferenceDistance: 1.0,
		WallAttenuation:   DefaultWallAttenuation,
	}

	// Set reference loss based on band
	switch band {
	case "5GHz":
		model.ReferenceLoss = DefaultReferenceLoss5GHz
	case "6GHz":
		model.ReferenceLoss = DefaultReferenceLoss6GHz
	default:
		model.ReferenceLoss = DefaultReferenceLoss2_4GHz
	}

	// Set path loss exponent based on environment
	switch environment {
	case "office":
		model.PathLossExponent = PathLossExponentOffice
	case "residential":
		model.PathLossExponent = PathLossExponentResidential
	case "warehouse":
		model.PathLossExponent = PathLossExponentWarehouse
	case "free_space":
		model.PathLossExponent = PathLossExponentFreeSpace
	default:
		model.PathLossExponent = PathLossExponentOffice
	}

	return model
}

// PredictRSSI estimates the RSSI at a given distance using the path loss model.
func (m *PathLossModel) PredictRSSI(txPower int, distance float64) int {
	if distance <= 0 {
		return txPower
	}

	// Path loss formula: PL = PL_0 + 10 * n * log10(d/d_0)
	// RSSI = TxPower - PathLoss
	if distance < m.ReferenceDistance {
		distance = m.ReferenceDistance
	}

	pathLoss := m.ReferenceLoss + 10*m.PathLossExponent*math.Log10(distance/m.ReferenceDistance)
	rssi := float64(txPower) - pathLoss

	return int(math.Round(rssi))
}

// PredictDistance estimates the distance for a given RSSI using the path loss model.
func (m *PathLossModel) PredictDistance(txPower, rssi int) float64 {
	if rssi >= txPower {
		return 0
	}

	// Rearranging: d = d_0 * 10^((PL - PL_0) / (10 * n))
	// Where PL = TxPower - RSSI
	pathLoss := float64(txPower - rssi)
	exponent := (pathLoss - m.ReferenceLoss) / (10 * m.PathLossExponent)
	distance := m.ReferenceDistance * math.Pow(10, exponent)

	return distance
}

// AnalyzeCoverage analyzes WiFi coverage from signal samples.
func AnalyzeCoverage(samples []SignalSample, floorPlan *FloorPlan, threshold int) (*CoverageResult, error) {
	if len(samples) == 0 {
		return nil, ErrNoData
	}

	if floorPlan == nil {
		return nil, ErrNoFloorPlan
	}

	if floorPlan.Width <= 0 || floorPlan.Height <= 0 {
		return nil, ErrInvalidInput
	}

	// Calculate statistics
	var totalRSSI float64
	minRSSI := 0
	maxRSSI := -200
	coveredCount := 0

	for i, sample := range samples {
		rssi := sample.RSSI

		if i == 0 || rssi < minRSSI {
			minRSSI = rssi
		}
		if rssi > maxRSSI {
			maxRSSI = rssi
		}

		totalRSSI += float64(rssi)

		if rssi >= threshold {
			coveredCount++
		}
	}

	avgRSSI := totalRSSI / float64(len(samples))
	coveragePercent := float64(coveredCount) / float64(len(samples)) * 100
	totalArea := floorPlan.Width * floorPlan.Height
	coveredArea := totalArea * coveragePercent / 100

	// Count dead zones (clusters of weak signal)
	deadZoneCount := countDeadZones(samples, threshold)

	// Generate recommendations
	recommendations := generateRecommendations(coveragePercent, avgRSSI, deadZoneCount, len(samples))

	return &CoverageResult{
		TotalArea:       totalArea,
		CoveredArea:     coveredArea,
		CoveragePercent: coveragePercent,
		AverageRSSI:     avgRSSI,
		MinRSSI:         minRSSI,
		MaxRSSI:         maxRSSI,
		DeadZoneCount:   deadZoneCount,
		Recommendations: recommendations,
	}, nil
}

// SuggestAPPlacements suggests optimal access point placements.
func SuggestAPPlacements(
	floorPlan *FloorPlan,
	existingAPs []AccessPoint,
	targetCoverage float64,
	threshold int,
) ([]PlacementSuggestion, error) {
	if floorPlan == nil {
		return nil, ErrNoFloorPlan
	}

	if floorPlan.Width <= 0 || floorPlan.Height <= 0 {
		return nil, ErrInvalidInput
	}

	if targetCoverage <= 0 || targetCoverage > 100 {
		targetCoverage = 95.0
	}

	// Use a grid-based approach to find optimal placements
	suggestions := findOptimalPlacements(floorPlan, existingAPs, targetCoverage, threshold)

	// Sort by priority
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Priority < suggestions[j].Priority
	})

	return suggestions, nil
}

// PredictSignalMap generates a predicted signal heatmap for given APs.
func PredictSignalMap(
	floorPlan *FloorPlan,
	aps []AccessPoint,
	model *PathLossModel,
	resolution float64,
) ([]HeatmapPoint, error) {
	if floorPlan == nil {
		return nil, ErrNoFloorPlan
	}

	if len(aps) == 0 {
		return nil, ErrNoData
	}

	if model == nil {
		model = NewPathLossModel("office", "2.4GHz")
	}

	if resolution < MinGridResolution {
		resolution = MinGridResolution
	}
	if resolution > MaxGridResolution {
		resolution = MaxGridResolution
	}

	var heatmap []HeatmapPoint

	for x := 0.0; x < floorPlan.Width; x += resolution {
		for y := 0.0; y < floorPlan.Height; y += resolution {
			point := Point{X: x, Y: y}

			// Find the strongest signal at this point from any AP
			bestRSSI := ThresholdMinimum - 10 // Start below minimum
			for _, ap := range aps {
				distance := calculateDistance(point, ap.Location)
				rssi := model.PredictRSSI(ap.TxPower, distance)
				if rssi > bestRSSI {
					bestRSSI = rssi
				}
			}

			heatmap = append(heatmap, HeatmapPoint{
				Location: point,
				RSSI:     bestRSSI,
			})
		}
	}

	return heatmap, nil
}

// EstimateAPCount estimates the number of APs needed for target coverage.
func EstimateAPCount(floorPlan *FloorPlan, environment, band string, targetCoverage float64) (int, error) {
	if floorPlan == nil {
		return 0, ErrNoFloorPlan
	}

	if floorPlan.Width <= 0 || floorPlan.Height <= 0 {
		return 0, ErrInvalidInput
	}

	if targetCoverage <= 0 || targetCoverage > 100 {
		targetCoverage = 95.0
	}

	totalArea := floorPlan.Width * floorPlan.Height

	// Estimate coverage radius based on environment and band
	coverageRadius := estimateCoverageRadius(environment, band)

	// Calculate area per AP (using circle area)
	areaPerAP := math.Pi * coverageRadius * coverageRadius

	// Account for overlap needed for good coverage
	overlapFactor := 0.7 // 30% overlap between APs

	effectiveAreaPerAP := areaPerAP * overlapFactor

	// Calculate number of APs needed
	apCount := math.Ceil(totalArea / effectiveAreaPerAP)

	// Adjust for target coverage (higher target = more APs)
	coverageFactor := targetCoverage / 95.0
	apCount = math.Ceil(apCount * coverageFactor)

	if apCount < 1 {
		apCount = 1
	}

	return int(apCount), nil
}

// CalibrateModel adjusts the path loss model based on actual measurements.
func CalibrateModel(samples []SignalSample, ap AccessPoint) (*PathLossModel, error) {
	if len(samples) == 0 {
		return nil, ErrNoData
	}

	// Filter samples with known distances
	var validSamples []SignalSample
	for _, s := range samples {
		if s.Distance > 0 {
			validSamples = append(validSamples, s)
		}
	}

	if len(validSamples) < 2 {
		return nil, ErrNoData
	}

	// Use linear regression to estimate path loss exponent
	// PL = PL_0 + 10 * n * log10(d)
	// We solve for n using least squares
	var sumX, sumY, sumXY, sumX2 float64
	referenceLoss := DefaultReferenceLoss2_4GHz

	for _, s := range validSamples {
		pathLoss := float64(ap.TxPower - s.RSSI)
		x := 10 * math.Log10(s.Distance)
		y := pathLoss - referenceLoss

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	n := float64(len(validSamples))
	exponent := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Clamp exponent to reasonable range
	if exponent < 1.5 {
		exponent = 1.5
	}
	if exponent > 5.0 {
		exponent = 5.0
	}

	return &PathLossModel{
		ReferenceDistance: 1.0,
		ReferenceLoss:     referenceLoss,
		PathLossExponent:  exponent,
		WallAttenuation:   DefaultWallAttenuation,
	}, nil
}

// ClassifySignalQuality returns a quality classification for the given RSSI.
func ClassifySignalQuality(rssi int) string {
	switch {
	case rssi >= ThresholdExcellent:
		return "excellent"
	case rssi >= ThresholdGood:
		return "good"
	case rssi >= ThresholdFair:
		return "fair"
	case rssi >= ThresholdPoor:
		return "poor"
	default:
		return "unusable"
	}
}

// Internal helper functions

func calculateDistance(p1, p2 Point) float64 {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func countDeadZones(samples []SignalSample, threshold int) int {
	if len(samples) == 0 {
		return 0
	}

	// Find weak signal samples
	var weakSamples []SignalSample
	for _, s := range samples {
		if s.RSSI < threshold {
			weakSamples = append(weakSamples, s)
		}
	}

	if len(weakSamples) == 0 {
		return 0
	}

	// Simple clustering: count groups of weak samples within 5m of each other
	const clusterRadius = 5.0
	visited := make([]bool, len(weakSamples))
	zones := 0

	for i := range weakSamples {
		if visited[i] {
			continue
		}
		zones++
		// Mark all samples within cluster radius as visited
		for j := i; j < len(weakSamples); j++ {
			if !visited[j] {
				dist := calculateDistance(weakSamples[i].Location, weakSamples[j].Location)
				if dist <= clusterRadius {
					visited[j] = true
				}
			}
		}
	}

	return zones
}

func generateRecommendations(coverage, avgRSSI float64, deadZones, sampleCount int) []string {
	var recs []string

	switch {
	case coverage >= 95 && avgRSSI >= float64(ThresholdGood):
		recs = append(recs, "Excellent coverage achieved. Network is well-optimized.")
	case coverage >= 80:
		recs = append(recs, "Good coverage overall. Minor improvements possible.")
	case coverage >= 60:
		recs = append(recs, "Moderate coverage. Consider adding access points in weak areas.")
	default:
		recs = append(recs, "Poor coverage detected. Significant network improvements needed.")
	}

	if deadZones > 0 {
		recs = append(recs,
			"Dead zones detected. Consider repositioning APs or adding repeaters.")
	}

	if avgRSSI < float64(ThresholdFair) {
		recs = append(recs,
			"Average signal strength is below optimal. Consider increasing AP density.")
	}

	if sampleCount < 10 {
		recs = append(recs,
			"Limited sample data. Collect more measurements for accurate analysis.")
	}

	return recs
}

//nolint:gocognit // Complex scoring logic; keep in one place for clarity.
func findOptimalPlacements(
	floorPlan *FloorPlan,
	existingAPs []AccessPoint,
	_ float64,
	threshold int,
) []PlacementSuggestion {
	var suggestions []PlacementSuggestion

	// Grid-based analysis
	gridSize := 2.0 // 2m grid
	model := NewPathLossModel("office", "2.4GHz")

	// Find areas with weakest coverage
	type gridCell struct {
		point Point
		rssi  int
	}

	var cells []gridCell

	for x := gridSize / 2; x < floorPlan.Width; x += gridSize {
		for y := gridSize / 2; y < floorPlan.Height; y += gridSize {
			point := Point{X: x, Y: y}
			bestRSSI := ThresholdMinimum - 20

			for _, ap := range existingAPs {
				dist := calculateDistance(point, ap.Location)
				rssi := model.PredictRSSI(ap.TxPower, dist)
				if rssi > bestRSSI {
					bestRSSI = rssi
				}
			}

			cells = append(cells, gridCell{point: point, rssi: bestRSSI})
		}
	}

	// Sort cells by RSSI (weakest first)
	sort.Slice(cells, func(i, j int) bool {
		return cells[i].rssi < cells[j].rssi
	})

	// Suggest placements for weakest areas
	priority := 1
	for _, cell := range cells {
		if cell.rssi >= threshold {
			break
		}

		// Check if this is far enough from other suggestions
		tooClose := false
		minDistance := 10.0 // Minimum 10m between suggested APs

		for _, s := range suggestions {
			if calculateDistance(cell.point, s.Location) < minDistance {
				tooClose = true
				break
			}
		}

		if !tooClose {
			coverageGain := estimateCoverageGain(cell.rssi, threshold)
			suggestions = append(suggestions, PlacementSuggestion{
				Location:     cell.point,
				Priority:     priority,
				Reason:       classifyPlacementReason(cell.rssi),
				CoverageGain: coverageGain,
			})
			priority++
		}

		// Limit suggestions
		if len(suggestions) >= 5 {
			break
		}
	}

	// If no existing APs and no weak spots found, suggest center placement
	if len(existingAPs) == 0 && len(suggestions) == 0 {
		suggestions = append(suggestions, PlacementSuggestion{
			Location:     Point{X: floorPlan.Width / 2, Y: floorPlan.Height / 2},
			Priority:     1,
			Reason:       "Initial central placement for baseline coverage",
			CoverageGain: 50.0,
		})
	}

	return suggestions
}

func estimateCoverageRadius(environment, band string) float64 {
	baseRadius := 15.0 // Base radius in meters for 2.4 GHz office

	// Adjust for environment
	switch environment {
	case "free_space":
		baseRadius *= 2.0
	case "warehouse":
		baseRadius *= 1.5
	case "residential":
		baseRadius *= 1.1
	case "office":
		// Default
	default:
		// Use office as default
	}

	// Adjust for band (higher frequencies have shorter range)
	switch band {
	case "5GHz":
		baseRadius *= 0.7
	case "6GHz":
		baseRadius *= 0.5
	default:
		// 2.4 GHz - use base
	}

	return baseRadius
}

func estimateCoverageGain(currentRSSI, threshold int) float64 {
	if currentRSSI >= threshold {
		return 0
	}

	// Estimate gain based on how far below threshold
	gap := float64(threshold - currentRSSI)
	// Larger gap = bigger potential improvement
	gain := gap * 2.0

	if gain > 30 {
		gain = 30
	}

	return gain
}

func classifyPlacementReason(rssi int) string {
	quality := ClassifySignalQuality(rssi)
	switch quality {
	case "unusable":
		return "Critical dead zone - no usable signal detected"
	case "poor":
		return "Poor signal area - connection drops likely"
	case "fair":
		return "Fair signal area - improved coverage recommended"
	default:
		return "Coverage improvement opportunity"
	}
}
