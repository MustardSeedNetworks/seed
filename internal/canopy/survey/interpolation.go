// Package survey provides WiFi site survey functionality.
package survey

import (
	"math"
)

// Point2D represents a 2D coordinate.
type Point2D struct {
	X float64
	Y float64
}

// SampleValue represents a sample point with its measured value.
type SampleValue struct {
	Point Point2D
	Value float64
}

// InterpolationMethod defines the interpolation algorithm to use.
type InterpolationMethod string

const (
	// MethodIDW uses Inverse Distance Weighting interpolation.
	MethodIDW InterpolationMethod = "idw"
	// MethodNearest uses nearest neighbor interpolation.
	MethodNearest InterpolationMethod = "nearest"
)

// Interpolator performs spatial interpolation of sample values.
type Interpolator struct {
	Samples  []SampleValue
	Method   InterpolationMethod
	Power    float64 // Power parameter for IDW (default: 2.0)
	MaxDist  float64 // Maximum distance to consider (0 = unlimited)
	MinCount int     // Minimum samples to use (default: 1)
}

// NewInterpolator creates a new interpolator with default settings.
func NewInterpolator(samples []SampleValue) *Interpolator {
	return &Interpolator{
		Samples:  samples,
		Method:   MethodIDW,
		Power:    2.0,
		MaxDist:  0,
		MinCount: 1,
	}
}

// Interpolate calculates the interpolated value at a given point.
func (i *Interpolator) Interpolate(x, y float64) float64 {
	if len(i.Samples) == 0 {
		return 0
	}

	switch i.Method {
	case MethodNearest:
		return i.nearestNeighbor(x, y)
	case MethodIDW:
		return i.inverseDistanceWeighting(x, y)
	default:
		return i.inverseDistanceWeighting(x, y)
	}
}

// inverseDistanceWeighting implements IDW interpolation.
// IDW formula: z = Σ(wi * zi) / Σ(wi) where wi = 1 / d^p.
func (i *Interpolator) inverseDistanceWeighting(x, y float64) float64 {
	var weightedSum, weightSum float64
	point := Point2D{X: x, Y: y}

	for _, sample := range i.Samples {
		dist := distance(point, sample.Point)

		// If we're exactly on a sample point, return its value
		if dist < 0.0001 {
			return sample.Value
		}

		// Skip samples beyond max distance (if set)
		if i.MaxDist > 0 && dist > i.MaxDist {
			continue
		}

		// Calculate weight: 1 / distance^power
		weight := 1.0 / math.Pow(dist, i.Power)
		weightedSum += weight * sample.Value
		weightSum += weight
	}

	if weightSum == 0 {
		// No samples within range, use nearest
		return i.nearestNeighbor(x, y)
	}

	return weightedSum / weightSum
}

// nearestNeighbor returns the value of the closest sample.
func (i *Interpolator) nearestNeighbor(x, y float64) float64 {
	if len(i.Samples) == 0 {
		return 0
	}

	point := Point2D{X: x, Y: y}
	minDist := math.MaxFloat64
	var nearestValue float64

	for _, sample := range i.Samples {
		dist := distance(point, sample.Point)
		if dist < minDist {
			minDist = dist
			nearestValue = sample.Value
		}
	}

	return nearestValue
}

// distance calculates Euclidean distance between two points.
func distance(p1, p2 Point2D) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// InterpolateGrid generates a 2D grid of interpolated values.
func (i *Interpolator) InterpolateGrid(width, height, cellSize int) [][]float64 {
	cols := (width + cellSize - 1) / cellSize
	rows := (height + cellSize - 1) / cellSize

	grid := make([][]float64, rows)
	for row := range rows {
		grid[row] = make([]float64, cols)
		for col := range cols {
			// Calculate center of cell
			x := float64(col*cellSize) + float64(cellSize)/2
			y := float64(row*cellSize) + float64(cellSize)/2
			grid[row][col] = i.Interpolate(x, y)
		}
	}

	return grid
}

// GridStats contains statistics about an interpolated grid.
type GridStats struct {
	Min     float64
	Max     float64
	Average float64
	Count   int
}

// CalculateGridStats computes statistics for an interpolated grid.
func CalculateGridStats(grid [][]float64) GridStats {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return GridStats{}
	}

	stats := GridStats{
		Min: math.MaxFloat64,
		Max: -math.MaxFloat64,
	}

	var sum float64
	for _, row := range grid {
		for _, val := range row {
			if val < stats.Min {
				stats.Min = val
			}
			if val > stats.Max {
				stats.Max = val
			}
			sum += val
			stats.Count++
		}
	}

	if stats.Count > 0 {
		stats.Average = sum / float64(stats.Count)
	}

	return stats
}

// ExtractSamplesFromSurvey extracts interpolation samples from survey data.
// Supports multi-floor surveys by using GetAllSamples().
func ExtractSamplesFromSurvey(survey *Survey, valueType string) []SampleValue {
	allSamples := survey.GetAllSamples()
	samples := make([]SampleValue, 0, len(allSamples))

	for _, sp := range allSamples {
		value := extractValue(sp.SampleData, valueType)
		if !math.IsNaN(value) {
			samples = append(samples, SampleValue{
				Point: Point2D{X: float64(sp.X), Y: float64(sp.Y)},
				Value: value,
			})
		}
	}

	return samples
}

// ExtractSamplesFromFloor extracts interpolation samples from a specific floor.
func ExtractSamplesFromFloor(floor *Floor, valueType string) []SampleValue {
	if floor == nil {
		return nil
	}
	samples := make([]SampleValue, 0, len(floor.Samples))

	for _, sp := range floor.Samples {
		value := extractValue(sp.SampleData, valueType)
		if !math.IsNaN(value) {
			samples = append(samples, SampleValue{
				Point: Point2D{X: float64(sp.X), Y: float64(sp.Y)},
				Value: value,
			})
		}
	}

	return samples
}

// extractValue extracts the requested value from sample data.
func extractValue(sampleData any, valueType string) float64 {
	switch data := sampleData.(type) {
	case *PassiveSample:
		return extractPassiveValue(data, valueType)
	case PassiveSample:
		return extractPassiveValue(&data, valueType)
	case *ActiveSample:
		return extractActiveValue(data, valueType)
	case ActiveSample:
		return extractActiveValue(&data, valueType)
	case *ThroughputSample:
		return extractThroughputValue(data, valueType)
	case ThroughputSample:
		return extractThroughputValue(&data, valueType)
	case map[string]any:
		// Handle JSON-decoded data
		return extractMapValue(data, valueType)
	default:
		return math.NaN()
	}
}

func extractPassiveValue(data *PassiveSample, valueType string) float64 {
	if data == nil || len(data.Networks) == 0 {
		return math.NaN()
	}

	switch valueType {
	case string(HeatmapRSSI), HeatmapAliasSignal:
		// Return strongest signal (first network, sorted by signal)
		return float64(data.Networks[0].Signal)
	case "snr":
		return float64(data.Networks[0].SNR)
	case "density", "ap_count":
		return float64(data.UniqueBSSIDs)
	case "interference", "cochannel":
		return float64(data.CoChannelAPs)
	case "ap_2_4":
		return float64(data.APCount2_4)
	case "ap_5":
		return float64(data.APCount5)
	case "ap_6":
		return float64(data.APCount6)
	default:
		return float64(data.Networks[0].Signal)
	}
}

func extractActiveValue(data *ActiveSample, valueType string) float64 {
	if data == nil {
		return math.NaN()
	}

	switch valueType {
	case string(HeatmapRSSI), HeatmapAliasSignal:
		return float64(data.RSSI)
	case "datarate", "speed":
		return data.DataRate
	default:
		return float64(data.RSSI)
	}
}

func extractThroughputValue(data *ThroughputSample, valueType string) float64 {
	if data == nil {
		return math.NaN()
	}

	switch valueType {
	case string(HeatmapRSSI), HeatmapAliasSignal:
		return float64(data.RSSI)
	case "download":
		return data.DownloadMbps
	case "upload":
		return data.UploadMbps
	case "latency":
		return data.Latency
	case "jitter":
		return data.Jitter
	default:
		return float64(data.RSSI)
	}
}

func extractMapValue(data map[string]any, valueType string) float64 {
	// Handle networks array for passive samples
	if networks, networksOK := data["networks"].([]any); networksOK && len(networks) > 0 {
		if first, firstOK := networks[0].(map[string]any); firstOK {
			switch valueType {
			case string(HeatmapRSSI), HeatmapAliasSignal:
				if rssi, rssiOK := first[string(HeatmapRSSI)].(float64); rssiOK {
					return rssi
				}
			}
		}
	}

	// Direct value lookup
	key := valueType
	switch valueType {
	case HeatmapAliasSignal:
		key = string(HeatmapRSSI)
	case "density":
		key = "uniqueBSSIDs"
	case "interference":
		key = "coChannelAPs"
	}

	if val, ok := data[key].(float64); ok {
		return val
	}
	if val, ok := data[key].(int); ok {
		return float64(val)
	}

	return math.NaN()
}
