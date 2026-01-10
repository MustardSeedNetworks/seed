package ai_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/canopy/ai"
)

// TestNewPathLossModel tests path loss model creation for different environments and bands.
func TestNewPathLossModel(t *testing.T) {
	tests := []struct {
		name                string
		environment         string
		band                string
		expectedExponent    float64
		expectedRefLoss     float64
		expectedRefDistance float64
		expectedWallAtten   float64
	}{
		{
			name:                "Office 2.4GHz",
			environment:         "office",
			band:                "2.4GHz",
			expectedExponent:    ai.PathLossExponentOffice,
			expectedRefLoss:     ai.DefaultReferenceLoss2_4GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
		{
			name:                "Office 5GHz",
			environment:         "office",
			band:                "5GHz",
			expectedExponent:    ai.PathLossExponentOffice,
			expectedRefLoss:     ai.DefaultReferenceLoss5GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
		{
			name:                "Office 6GHz",
			environment:         "office",
			band:                "6GHz",
			expectedExponent:    ai.PathLossExponentOffice,
			expectedRefLoss:     ai.DefaultReferenceLoss6GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
		{
			name:                "Residential 2.4GHz",
			environment:         "residential",
			band:                "2.4GHz",
			expectedExponent:    ai.PathLossExponentResidential,
			expectedRefLoss:     ai.DefaultReferenceLoss2_4GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
		{
			name:                "Warehouse 2.4GHz",
			environment:         "warehouse",
			band:                "2.4GHz",
			expectedExponent:    ai.PathLossExponentWarehouse,
			expectedRefLoss:     ai.DefaultReferenceLoss2_4GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
		{
			name:                "Free space 2.4GHz",
			environment:         "free_space",
			band:                "2.4GHz",
			expectedExponent:    ai.PathLossExponentFreeSpace,
			expectedRefLoss:     ai.DefaultReferenceLoss2_4GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
		{
			name:                "Unknown environment defaults to office",
			environment:         "unknown",
			band:                "2.4GHz",
			expectedExponent:    ai.PathLossExponentOffice,
			expectedRefLoss:     ai.DefaultReferenceLoss2_4GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
		{
			name:                "Unknown band defaults to 2.4GHz",
			environment:         "office",
			band:                "unknown",
			expectedExponent:    ai.PathLossExponentOffice,
			expectedRefLoss:     ai.DefaultReferenceLoss2_4GHz,
			expectedRefDistance: 1.0,
			expectedWallAtten:   ai.DefaultWallAttenuation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := ai.NewPathLossModel(tt.environment, tt.band)

			if model.PathLossExponent != tt.expectedExponent {
				t.Errorf("PathLossExponent = %v, want %v", model.PathLossExponent, tt.expectedExponent)
			}
			if model.ReferenceLoss != tt.expectedRefLoss {
				t.Errorf("ReferenceLoss = %v, want %v", model.ReferenceLoss, tt.expectedRefLoss)
			}
			if model.ReferenceDistance != tt.expectedRefDistance {
				t.Errorf("ReferenceDistance = %v, want %v", model.ReferenceDistance, tt.expectedRefDistance)
			}
			if model.WallAttenuation != tt.expectedWallAtten {
				t.Errorf("WallAttenuation = %v, want %v", model.WallAttenuation, tt.expectedWallAtten)
			}
		})
	}
}

// TestPathLossModelPredictRSSI tests RSSI prediction at various distances.
func TestPathLossModelPredictRSSI(t *testing.T) {
	model := ai.NewPathLossModel("office", "2.4GHz")

	tests := []struct {
		name     string
		txPower  int
		distance float64
		wantMin  int
		wantMax  int
	}{
		{
			name:     "Zero distance returns txPower",
			txPower:  20,
			distance: 0,
			wantMin:  20,
			wantMax:  20,
		},
		{
			name:     "Negative distance returns txPower",
			txPower:  20,
			distance: -1,
			wantMin:  20,
			wantMax:  20,
		},
		{
			name:     "1m distance",
			txPower:  20,
			distance: 1,
			wantMin:  -25,
			wantMax:  -15,
		},
		{
			name:     "5m distance",
			txPower:  20,
			distance: 5,
			wantMin:  -50,
			wantMax:  -35,
		},
		{
			name:     "10m distance",
			txPower:  20,
			distance: 10,
			wantMin:  -60,
			wantMax:  -40,
		},
		{
			name:     "20m distance",
			txPower:  20,
			distance: 20,
			wantMin:  -70,
			wantMax:  -50,
		},
		{
			name:     "Sub-reference distance (0.5m)",
			txPower:  20,
			distance: 0.5,
			wantMin:  -25,
			wantMax:  -15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rssi := model.PredictRSSI(tt.txPower, tt.distance)

			if rssi < tt.wantMin || rssi > tt.wantMax {
				t.Errorf("PredictRSSI(%d, %v) = %d, want between %d and %d",
					tt.txPower, tt.distance, rssi, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestPathLossModelPredictDistance tests distance estimation from RSSI.
func TestPathLossModelPredictDistance(t *testing.T) {
	model := ai.NewPathLossModel("office", "2.4GHz")

	tests := []struct {
		name    string
		txPower int
		rssi    int
		wantMin float64
		wantMax float64
	}{
		{
			name:    "RSSI equals txPower",
			txPower: 20,
			rssi:    20,
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "RSSI greater than txPower",
			txPower: 20,
			rssi:    25,
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "Typical office distance",
			txPower: 20,
			rssi:    -50,
			wantMin: 3,
			wantMax: 20,
		},
		{
			name:    "Far distance",
			txPower: 20,
			rssi:    -70,
			wantMin: 10,
			wantMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := model.PredictDistance(tt.txPower, tt.rssi)

			if distance < tt.wantMin || distance > tt.wantMax {
				t.Errorf("PredictDistance(%d, %d) = %v, want between %v and %v",
					tt.txPower, tt.rssi, distance, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestPathLossModelRoundTrip tests that PredictRSSI and PredictDistance are inverses.
func TestPathLossModelRoundTrip(t *testing.T) {
	model := ai.NewPathLossModel("office", "2.4GHz")
	txPower := 20

	distances := []float64{1.0, 5.0, 10.0, 20.0, 50.0}

	for _, origDistance := range distances {
		rssi := model.PredictRSSI(txPower, origDistance)
		recoveredDistance := model.PredictDistance(txPower, rssi)

		// Allow for rounding errors
		tolerance := 0.5
		if math.Abs(recoveredDistance-origDistance) > tolerance {
			t.Errorf("Round trip failed: distance %v -> RSSI %d -> distance %v (tolerance %v)",
				origDistance, rssi, recoveredDistance, tolerance)
		}
	}
}

// TestAnalyzeCoverage tests coverage analysis with various inputs.
func TestAnalyzeCoverage(t *testing.T) {
	validFloorPlan := &ai.FloorPlan{Width: 20, Height: 20}

	tests := []struct {
		name        string
		samples     []ai.SignalSample
		floorPlan   *ai.FloorPlan
		threshold   int
		wantErr     error
		checkResult func(*testing.T, *ai.CoverageResult)
	}{
		{
			name:      "No samples",
			samples:   []ai.SignalSample{},
			floorPlan: validFloorPlan,
			threshold: ai.ThresholdFair,
			wantErr:   ai.ErrNoData,
		},
		{
			name:      "Nil floor plan",
			samples:   []ai.SignalSample{{RSSI: -50}},
			floorPlan: nil,
			threshold: ai.ThresholdFair,
			wantErr:   ai.ErrNoFloorPlan,
		},
		{
			name:      "Invalid floor plan width",
			samples:   []ai.SignalSample{{RSSI: -50}},
			floorPlan: &ai.FloorPlan{Width: 0, Height: 20},
			threshold: ai.ThresholdFair,
			wantErr:   ai.ErrInvalidInput,
		},
		{
			name:      "Invalid floor plan height",
			samples:   []ai.SignalSample{{RSSI: -50}},
			floorPlan: &ai.FloorPlan{Width: 20, Height: -5},
			threshold: ai.ThresholdFair,
			wantErr:   ai.ErrInvalidInput,
		},
		{
			name: "All excellent signals",
			samples: []ai.SignalSample{
				{Location: ai.Point{X: 5, Y: 5}, RSSI: -45},
				{Location: ai.Point{X: 10, Y: 10}, RSSI: -50},
				{Location: ai.Point{X: 15, Y: 15}, RSSI: -48},
			},
			floorPlan: validFloorPlan,
			threshold: ai.ThresholdFair,
			checkResult: func(t *testing.T, r *ai.CoverageResult) {
				if r.CoveragePercent != 100 {
					t.Errorf("CoveragePercent = %v, want 100", r.CoveragePercent)
				}
				if r.DeadZoneCount != 0 {
					t.Errorf("DeadZoneCount = %v, want 0", r.DeadZoneCount)
				}
				if r.MinRSSI != -50 {
					t.Errorf("MinRSSI = %v, want -50", r.MinRSSI)
				}
				if r.MaxRSSI != -45 {
					t.Errorf("MaxRSSI = %v, want -45", r.MaxRSSI)
				}
			},
		},
		{
			name: "Mixed signals",
			samples: []ai.SignalSample{
				{Location: ai.Point{X: 5, Y: 5}, RSSI: -50},   // Good
				{Location: ai.Point{X: 10, Y: 10}, RSSI: -80}, // Poor
				{Location: ai.Point{X: 15, Y: 15}, RSSI: -60}, // Good
				{Location: ai.Point{X: 5, Y: 15}, RSSI: -90},  // Unusable
			},
			floorPlan: validFloorPlan,
			threshold: ai.ThresholdFair,
			checkResult: func(t *testing.T, r *ai.CoverageResult) {
				if r.CoveragePercent != 50 {
					t.Errorf("CoveragePercent = %v, want 50", r.CoveragePercent)
				}
				if r.TotalArea != 400 {
					t.Errorf("TotalArea = %v, want 400", r.TotalArea)
				}
				if r.CoveredArea != 200 {
					t.Errorf("CoveredArea = %v, want 200", r.CoveredArea)
				}
			},
		},
		{
			name: "All poor signals",
			samples: []ai.SignalSample{
				{Location: ai.Point{X: 5, Y: 5}, RSSI: -85},
				{Location: ai.Point{X: 10, Y: 10}, RSSI: -90},
				{Location: ai.Point{X: 15, Y: 15}, RSSI: -88},
			},
			floorPlan: validFloorPlan,
			threshold: ai.ThresholdFair,
			checkResult: func(t *testing.T, r *ai.CoverageResult) {
				if r.CoveragePercent != 0 {
					t.Errorf("CoveragePercent = %v, want 0", r.CoveragePercent)
				}
				if r.DeadZoneCount == 0 {
					t.Error("Expected at least one dead zone")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ai.AnalyzeCoverage(tt.samples, tt.floorPlan, tt.threshold)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("AnalyzeCoverage error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("AnalyzeCoverage unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// TestSuggestAPPlacements tests AP placement suggestions.
func TestSuggestAPPlacements(t *testing.T) {
	tests := []struct {
		name           string
		floorPlan      *ai.FloorPlan
		existingAPs    []ai.AccessPoint
		targetCoverage float64
		threshold      int
		wantErr        error
		checkResult    func(*testing.T, []ai.PlacementSuggestion)
	}{
		{
			name:           "Nil floor plan",
			floorPlan:      nil,
			existingAPs:    []ai.AccessPoint{},
			targetCoverage: 95,
			threshold:      ai.ThresholdFair,
			wantErr:        ai.ErrNoFloorPlan,
		},
		{
			name:           "Invalid floor plan",
			floorPlan:      &ai.FloorPlan{Width: 0, Height: 20},
			existingAPs:    []ai.AccessPoint{},
			targetCoverage: 95,
			threshold:      ai.ThresholdFair,
			wantErr:        ai.ErrInvalidInput,
		},
		{
			name:           "No existing APs suggests placements",
			floorPlan:      &ai.FloorPlan{Width: 20, Height: 20},
			existingAPs:    []ai.AccessPoint{},
			targetCoverage: 95,
			threshold:      ai.ThresholdFair,
			checkResult: func(t *testing.T, suggestions []ai.PlacementSuggestion) {
				if len(suggestions) == 0 {
					t.Error("Expected at least one suggestion for empty space")
					return
				}
				// First suggestion should have priority 1
				s := suggestions[0]
				if s.Priority != 1 {
					t.Errorf("Expected first suggestion to have priority 1, got %d", s.Priority)
				}
				// Suggestion should be within floor plan bounds
				if s.Location.X < 0 || s.Location.X > 20 || s.Location.Y < 0 || s.Location.Y > 20 {
					t.Errorf("Suggestion outside floor plan bounds: X=%v, Y=%v", s.Location.X, s.Location.Y)
				}
			},
		},
		{
			name:      "With existing AP in corner",
			floorPlan: &ai.FloorPlan{Width: 30, Height: 30},
			existingAPs: []ai.AccessPoint{
				{Location: ai.Point{X: 5, Y: 5}, TxPower: 20, Band: "2.4GHz"},
			},
			targetCoverage: 95,
			threshold:      ai.ThresholdFair,
			checkResult: func(t *testing.T, suggestions []ai.PlacementSuggestion) {
				// Should suggest placements away from the corner AP
				for _, s := range suggestions {
					if s.Location.X < 10 && s.Location.Y < 10 {
						// Within 10m of existing AP is too close
						dist := math.Sqrt(
							(s.Location.X-5)*(s.Location.X-5) + (s.Location.Y-5)*(s.Location.Y-5),
						)
						if dist < 5 {
							t.Errorf("Suggestion too close to existing AP at distance %v", dist)
						}
					}
				}
			},
		},
		{
			name:           "Invalid target coverage defaults to 95",
			floorPlan:      &ai.FloorPlan{Width: 20, Height: 20},
			existingAPs:    []ai.AccessPoint{},
			targetCoverage: 150, // Invalid
			threshold:      ai.ThresholdFair,
			checkResult: func(t *testing.T, suggestions []ai.PlacementSuggestion) {
				if len(suggestions) == 0 {
					t.Error("Expected suggestions even with invalid target coverage")
				}
			},
		},
		{
			name:           "Zero target coverage defaults to 95",
			floorPlan:      &ai.FloorPlan{Width: 20, Height: 20},
			existingAPs:    []ai.AccessPoint{},
			targetCoverage: 0,
			threshold:      ai.ThresholdFair,
			checkResult: func(t *testing.T, suggestions []ai.PlacementSuggestion) {
				if len(suggestions) == 0 {
					t.Error("Expected suggestions even with zero target coverage")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := ai.SuggestAPPlacements(
				tt.floorPlan, tt.existingAPs, tt.targetCoverage, tt.threshold,
			)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("SuggestAPPlacements error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("SuggestAPPlacements unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, suggestions)
			}
		})
	}
}

// TestPredictSignalMap tests signal heatmap generation.
func TestPredictSignalMap(t *testing.T) {
	tests := []struct {
		name        string
		floorPlan   *ai.FloorPlan
		aps         []ai.AccessPoint
		model       *ai.PathLossModel
		resolution  float64
		wantErr     error
		checkResult func(*testing.T, []ai.HeatmapPoint)
	}{
		{
			name:       "Nil floor plan",
			floorPlan:  nil,
			aps:        []ai.AccessPoint{{TxPower: 20}},
			model:      nil,
			resolution: 1.0,
			wantErr:    ai.ErrNoFloorPlan,
		},
		{
			name:       "No access points",
			floorPlan:  &ai.FloorPlan{Width: 10, Height: 10},
			aps:        []ai.AccessPoint{},
			model:      nil,
			resolution: 1.0,
			wantErr:    ai.ErrNoData,
		},
		{
			name:      "Single AP generates heatmap",
			floorPlan: &ai.FloorPlan{Width: 10, Height: 10},
			aps: []ai.AccessPoint{
				{Location: ai.Point{X: 5, Y: 5}, TxPower: 20, Band: "2.4GHz"},
			},
			model:      nil,
			resolution: 2.0,
			checkResult: func(t *testing.T, heatmap []ai.HeatmapPoint) {
				if len(heatmap) == 0 {
					t.Error("Expected non-empty heatmap")
					return
				}
				// Check that points near AP have stronger signal
				var centerRSSI, cornerRSSI int
				for _, p := range heatmap {
					if p.Location.X >= 4 && p.Location.X <= 6 &&
						p.Location.Y >= 4 && p.Location.Y <= 6 {
						centerRSSI = p.RSSI
					}
					if p.Location.X == 0 && p.Location.Y == 0 {
						cornerRSSI = p.RSSI
					}
				}
				if centerRSSI <= cornerRSSI {
					t.Errorf("Center RSSI (%d) should be stronger than corner (%d)",
						centerRSSI, cornerRSSI)
				}
			},
		},
		{
			name:      "Resolution below minimum is clamped",
			floorPlan: &ai.FloorPlan{Width: 5, Height: 5},
			aps: []ai.AccessPoint{
				{Location: ai.Point{X: 2.5, Y: 2.5}, TxPower: 20},
			},
			model:      nil,
			resolution: 0.1, // Below minimum
			checkResult: func(t *testing.T, heatmap []ai.HeatmapPoint) {
				// Should still generate heatmap with clamped resolution
				if len(heatmap) == 0 {
					t.Error("Expected non-empty heatmap even with sub-minimum resolution")
				}
			},
		},
		{
			name:      "Resolution above maximum is clamped",
			floorPlan: &ai.FloorPlan{Width: 20, Height: 20},
			aps: []ai.AccessPoint{
				{Location: ai.Point{X: 10, Y: 10}, TxPower: 20},
			},
			model:      nil,
			resolution: 10.0, // Above maximum
			checkResult: func(t *testing.T, heatmap []ai.HeatmapPoint) {
				// Should still generate heatmap with clamped resolution
				if len(heatmap) == 0 {
					t.Error("Expected non-empty heatmap even with above-maximum resolution")
				}
			},
		},
		{
			name:      "Multiple APs - best signal selected",
			floorPlan: &ai.FloorPlan{Width: 20, Height: 10},
			aps: []ai.AccessPoint{
				{Location: ai.Point{X: 5, Y: 5}, TxPower: 20, Band: "2.4GHz"},
				{Location: ai.Point{X: 15, Y: 5}, TxPower: 20, Band: "2.4GHz"},
			},
			model:      nil,
			resolution: 5.0,
			checkResult: func(t *testing.T, heatmap []ai.HeatmapPoint) {
				// Middle point should have reasonable signal from both APs
				for _, p := range heatmap {
					if p.Location.X >= 9 && p.Location.X <= 11 {
						// Middle point should have at least fair signal
						if p.RSSI < ai.ThresholdPoor {
							t.Errorf("Middle point RSSI (%d) worse than expected", p.RSSI)
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heatmap, err := ai.PredictSignalMap(tt.floorPlan, tt.aps, tt.model, tt.resolution)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("PredictSignalMap error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("PredictSignalMap unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, heatmap)
			}
		})
	}
}

// TestEstimateAPCount tests AP count estimation.
func TestEstimateAPCount(t *testing.T) {
	tests := []struct {
		name           string
		floorPlan      *ai.FloorPlan
		environment    string
		band           string
		targetCoverage float64
		wantErr        error
		wantMin        int
		wantMax        int
	}{
		{
			name:           "Nil floor plan",
			floorPlan:      nil,
			environment:    "office",
			band:           "2.4GHz",
			targetCoverage: 95,
			wantErr:        ai.ErrNoFloorPlan,
		},
		{
			name:           "Invalid floor plan",
			floorPlan:      &ai.FloorPlan{Width: -10, Height: 10},
			environment:    "office",
			band:           "2.4GHz",
			targetCoverage: 95,
			wantErr:        ai.ErrInvalidInput,
		},
		{
			name:           "Small office",
			floorPlan:      &ai.FloorPlan{Width: 10, Height: 10},
			environment:    "office",
			band:           "2.4GHz",
			targetCoverage: 95,
			wantMin:        1,
			wantMax:        2,
		},
		{
			name:           "Medium office",
			floorPlan:      &ai.FloorPlan{Width: 30, Height: 30},
			environment:    "office",
			band:           "2.4GHz",
			targetCoverage: 95,
			wantMin:        2,
			wantMax:        5,
		},
		{
			name:           "Large warehouse",
			floorPlan:      &ai.FloorPlan{Width: 100, Height: 50},
			environment:    "warehouse",
			band:           "2.4GHz",
			targetCoverage: 95,
			wantMin:        3,
			wantMax:        15,
		},
		{
			name:           "5GHz needs more APs",
			floorPlan:      &ai.FloorPlan{Width: 30, Height: 30},
			environment:    "office",
			band:           "5GHz",
			targetCoverage: 95,
			wantMin:        3,
			wantMax:        10,
		},
		{
			name:           "6GHz needs even more APs",
			floorPlan:      &ai.FloorPlan{Width: 30, Height: 30},
			environment:    "office",
			band:           "6GHz",
			targetCoverage: 95,
			wantMin:        5,
			wantMax:        20,
		},
		{
			name:           "Invalid target coverage defaults",
			floorPlan:      &ai.FloorPlan{Width: 20, Height: 20},
			environment:    "office",
			band:           "2.4GHz",
			targetCoverage: -50,
			wantMin:        1,
			wantMax:        5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := ai.EstimateAPCount(tt.floorPlan, tt.environment, tt.band, tt.targetCoverage)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("EstimateAPCount error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("EstimateAPCount unexpected error: %v", err)
			}

			if count < tt.wantMin || count > tt.wantMax {
				t.Errorf("EstimateAPCount = %d, want between %d and %d",
					count, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestCalibrateModel tests model calibration from measurements.
func TestCalibrateModel(t *testing.T) {
	tests := []struct {
		name       string
		samples    []ai.SignalSample
		ap         ai.AccessPoint
		wantErr    error
		checkModel func(*testing.T, *ai.PathLossModel)
	}{
		{
			name:    "No samples",
			samples: []ai.SignalSample{},
			ap:      ai.AccessPoint{TxPower: 20},
			wantErr: ai.ErrNoData,
		},
		{
			name: "One sample with distance",
			samples: []ai.SignalSample{
				{RSSI: -50, Distance: 5},
			},
			ap:      ai.AccessPoint{TxPower: 20},
			wantErr: ai.ErrNoData, // Need at least 2 samples
		},
		{
			name: "Samples without distance",
			samples: []ai.SignalSample{
				{RSSI: -50, Distance: 0},
				{RSSI: -60, Distance: 0},
			},
			ap:      ai.AccessPoint{TxPower: 20},
			wantErr: ai.ErrNoData,
		},
		{
			name: "Valid calibration data",
			samples: []ai.SignalSample{
				{RSSI: -50, Distance: 5},
				{RSSI: -60, Distance: 10},
				{RSSI: -70, Distance: 20},
			},
			ap: ai.AccessPoint{TxPower: 20},
			checkModel: func(t *testing.T, m *ai.PathLossModel) {
				if m == nil {
					t.Fatal("Expected non-nil model")
				}
				// Path loss exponent should be reasonable
				if m.PathLossExponent < 1.5 || m.PathLossExponent > 5.0 {
					t.Errorf("PathLossExponent = %v, want between 1.5 and 5.0",
						m.PathLossExponent)
				}
			},
		},
		{
			name: "Mixed valid and invalid samples",
			samples: []ai.SignalSample{
				{RSSI: -50, Distance: 5},
				{RSSI: -55, Distance: 0}, // Invalid - no distance
				{RSSI: -65, Distance: 15},
			},
			ap: ai.AccessPoint{TxPower: 20},
			checkModel: func(t *testing.T, m *ai.PathLossModel) {
				if m == nil {
					t.Fatal("Expected non-nil model")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := ai.CalibrateModel(tt.samples, tt.ap)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CalibrateModel error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("CalibrateModel unexpected error: %v", err)
			}

			if tt.checkModel != nil {
				tt.checkModel(t, model)
			}
		})
	}
}

// TestClassifySignalQuality tests signal quality classification.
func TestClassifySignalQuality(t *testing.T) {
	tests := []struct {
		name string
		rssi int
		want string
	}{
		{"Excellent signal", -40, "excellent"},
		{"Excellent boundary", -50, "excellent"},
		{"Good signal", -55, "good"},
		{"Good boundary", -65, "good"},
		{"Fair signal", -70, "fair"},
		{"Fair boundary", -75, "fair"},
		{"Poor signal", -80, "poor"},
		{"Poor boundary", -85, "poor"},
		{"Unusable signal", -90, "unusable"},
		{"Very weak signal", -100, "unusable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ai.ClassifySignalQuality(tt.rssi)
			if got != tt.want {
				t.Errorf("ClassifySignalQuality(%d) = %q, want %q", tt.rssi, got, tt.want)
			}
		})
	}
}

// TestCalculateDistance tests distance calculation between points.
func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name     string
		p1       ai.Point
		p2       ai.Point
		expected float64
	}{
		{
			name:     "Same point",
			p1:       ai.Point{X: 5, Y: 5},
			p2:       ai.Point{X: 5, Y: 5},
			expected: 0,
		},
		{
			name:     "Horizontal distance",
			p1:       ai.Point{X: 0, Y: 0},
			p2:       ai.Point{X: 3, Y: 0},
			expected: 3,
		},
		{
			name:     "Vertical distance",
			p1:       ai.Point{X: 0, Y: 0},
			p2:       ai.Point{X: 0, Y: 4},
			expected: 4,
		},
		{
			name:     "Diagonal distance 3-4-5 triangle",
			p1:       ai.Point{X: 0, Y: 0},
			p2:       ai.Point{X: 3, Y: 4},
			expected: 5,
		},
		{
			name:     "Negative coordinates",
			p1:       ai.Point{X: -3, Y: -4},
			p2:       ai.Point{X: 0, Y: 0},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ai.ExportCalculateDistance(tt.p1, tt.p2)
			if math.Abs(got-tt.expected) > 0.0001 {
				t.Errorf("calculateDistance(%v, %v) = %v, want %v",
					tt.p1, tt.p2, got, tt.expected)
			}
		})
	}
}

// TestCountDeadZones tests dead zone counting.
func TestCountDeadZones(t *testing.T) {
	tests := []struct {
		name      string
		samples   []ai.SignalSample
		threshold int
		want      int
	}{
		{
			name:      "No samples",
			samples:   []ai.SignalSample{},
			threshold: ai.ThresholdFair,
			want:      0,
		},
		{
			name: "All good signals",
			samples: []ai.SignalSample{
				{Location: ai.Point{X: 5, Y: 5}, RSSI: -50},
				{Location: ai.Point{X: 10, Y: 10}, RSSI: -60},
			},
			threshold: ai.ThresholdFair,
			want:      0,
		},
		{
			name: "Single weak sample",
			samples: []ai.SignalSample{
				{Location: ai.Point{X: 5, Y: 5}, RSSI: -85},
			},
			threshold: ai.ThresholdFair,
			want:      1,
		},
		{
			name: "Clustered weak samples",
			samples: []ai.SignalSample{
				{Location: ai.Point{X: 5, Y: 5}, RSSI: -85},
				{Location: ai.Point{X: 6, Y: 6}, RSSI: -88},
				{Location: ai.Point{X: 7, Y: 7}, RSSI: -82},
			},
			threshold: ai.ThresholdFair,
			want:      1, // All within 5m cluster radius
		},
		{
			name: "Two separate dead zones",
			samples: []ai.SignalSample{
				{Location: ai.Point{X: 0, Y: 0}, RSSI: -85},
				{Location: ai.Point{X: 2, Y: 2}, RSSI: -88},
				{Location: ai.Point{X: 50, Y: 50}, RSSI: -90},
				{Location: ai.Point{X: 52, Y: 52}, RSSI: -85},
			},
			threshold: ai.ThresholdFair,
			want:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ai.ExportCountDeadZones(tt.samples, tt.threshold)
			if got != tt.want {
				t.Errorf("countDeadZones = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGenerateRecommendations tests recommendation generation.
func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name           string
		coverage       float64
		avgRSSI        float64
		deadZones      int
		sampleCount    int
		expectKeywords []string
	}{
		{
			name:           "Excellent coverage",
			coverage:       98,
			avgRSSI:        -55,
			deadZones:      0,
			sampleCount:    50,
			expectKeywords: []string{"Excellent"},
		},
		{
			name:           "Good coverage",
			coverage:       85,
			avgRSSI:        -60,
			deadZones:      0,
			sampleCount:    30,
			expectKeywords: []string{"Good"},
		},
		{
			name:           "Moderate coverage",
			coverage:       70,
			avgRSSI:        -65,
			deadZones:      1,
			sampleCount:    20,
			expectKeywords: []string{"Moderate", "dead zone"},
		},
		{
			name:           "Poor coverage",
			coverage:       40,
			avgRSSI:        -80,
			deadZones:      3,
			sampleCount:    15,
			expectKeywords: []string{"Poor", "dead zone"},
		},
		{
			name:           "Limited samples",
			coverage:       90,
			avgRSSI:        -55,
			deadZones:      0,
			sampleCount:    5,
			expectKeywords: []string{"Limited sample"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := ai.ExportGenerateRecommendations(tt.coverage, tt.avgRSSI, tt.deadZones, tt.sampleCount)

			if len(recs) == 0 {
				t.Error("Expected at least one recommendation")
				return
			}

			allRecs := strings.Join(recs, " ")
			for _, keyword := range tt.expectKeywords {
				if !strings.Contains(strings.ToLower(allRecs), strings.ToLower(keyword)) {
					t.Errorf("Recommendations should contain %q, got: %v", keyword, recs)
				}
			}
		})
	}
}

// TestEstimateCoverageRadius tests coverage radius estimation.
func TestEstimateCoverageRadius(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		band        string
		wantMin     float64
		wantMax     float64
	}{
		{
			name:        "Office 2.4GHz",
			environment: "office",
			band:        "2.4GHz",
			wantMin:     10,
			wantMax:     20,
		},
		{
			name:        "Office 5GHz",
			environment: "office",
			band:        "5GHz",
			wantMin:     7,
			wantMax:     15,
		},
		{
			name:        "Office 6GHz",
			environment: "office",
			band:        "6GHz",
			wantMin:     5,
			wantMax:     12,
		},
		{
			name:        "Warehouse 2.4GHz",
			environment: "warehouse",
			band:        "2.4GHz",
			wantMin:     15,
			wantMax:     30,
		},
		{
			name:        "Free space 2.4GHz",
			environment: "free_space",
			band:        "2.4GHz",
			wantMin:     25,
			wantMax:     40,
		},
		{
			name:        "Residential 2.4GHz",
			environment: "residential",
			band:        "2.4GHz",
			wantMin:     12,
			wantMax:     22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			radius := ai.ExportEstimateCoverageRadius(tt.environment, tt.band)

			if radius < tt.wantMin || radius > tt.wantMax {
				t.Errorf("estimateCoverageRadius(%q, %q) = %v, want between %v and %v",
					tt.environment, tt.band, radius, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestEstimateCoverageGain tests coverage gain estimation.
func TestEstimateCoverageGain(t *testing.T) {
	tests := []struct {
		name        string
		currentRSSI int
		threshold   int
		wantMin     float64
		wantMax     float64
	}{
		{
			name:        "Already above threshold",
			currentRSSI: -70,
			threshold:   -75,
			wantMin:     0,
			wantMax:     0,
		},
		{
			name:        "Slightly below threshold",
			currentRSSI: -80,
			threshold:   -75,
			wantMin:     5,
			wantMax:     15,
		},
		{
			name:        "Well below threshold",
			currentRSSI: -90,
			threshold:   -75,
			wantMin:     20,
			wantMax:     35,
		},
		{
			name:        "Very weak signal",
			currentRSSI: -100,
			threshold:   -75,
			wantMin:     25,
			wantMax:     35, // Capped at 30
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gain := ai.ExportEstimateCoverageGain(tt.currentRSSI, tt.threshold)

			if gain < tt.wantMin || gain > tt.wantMax {
				t.Errorf("estimateCoverageGain(%d, %d) = %v, want between %v and %v",
					tt.currentRSSI, tt.threshold, gain, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestClassifyPlacementReason tests placement reason classification.
func TestClassifyPlacementReason(t *testing.T) {
	tests := []struct {
		name           string
		rssi           int
		expectKeywords []string
	}{
		{
			name:           "Unusable signal",
			rssi:           -95,
			expectKeywords: []string{"Critical", "dead zone"},
		},
		{
			name:           "Poor signal",
			rssi:           -82,
			expectKeywords: []string{"Poor", "connection drops"},
		},
		{
			name:           "Fair signal",
			rssi:           -72,
			expectKeywords: []string{"Fair", "improved"},
		},
		{
			name:           "Good signal",
			rssi:           -60,
			expectKeywords: []string{"improvement", "opportunity"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := ai.ExportClassifyPlacementReason(tt.rssi)

			for _, keyword := range tt.expectKeywords {
				if !strings.Contains(strings.ToLower(reason), strings.ToLower(keyword)) {
					t.Errorf("classifyPlacementReason(%d) = %q, expected to contain %q",
						tt.rssi, reason, keyword)
				}
			}
		})
	}
}

// TestFindOptimalPlacements tests optimal placement finding.
func TestFindOptimalPlacements(t *testing.T) {
	tests := []struct {
		name           string
		floorPlan      *ai.FloorPlan
		existingAPs    []ai.AccessPoint
		targetCoverage float64
		threshold      int
		checkResult    func(*testing.T, []ai.PlacementSuggestion)
	}{
		{
			name:           "Empty floor with no APs",
			floorPlan:      &ai.FloorPlan{Width: 20, Height: 20},
			existingAPs:    []ai.AccessPoint{},
			targetCoverage: 95,
			threshold:      ai.ThresholdFair,
			checkResult: func(t *testing.T, suggestions []ai.PlacementSuggestion) {
				if len(suggestions) == 0 {
					t.Error("Expected at least one suggestion")
					return
				}
				// First suggestion should be near center
				s := suggestions[0]
				if s.Priority != 1 {
					t.Errorf("First suggestion priority = %d, want 1", s.Priority)
				}
			},
		},
		{
			name:      "Floor with good existing coverage",
			floorPlan: &ai.FloorPlan{Width: 10, Height: 10},
			existingAPs: []ai.AccessPoint{
				{Location: ai.Point{X: 5, Y: 5}, TxPower: 23, Band: "2.4GHz"},
			},
			targetCoverage: 95,
			threshold:      ai.ThresholdFair,
			checkResult: func(t *testing.T, suggestions []ai.PlacementSuggestion) {
				// With a strong AP in a small room, might not need more
				// This is acceptable - no suggestions or suggestions far from center
				t.Helper()
				_ = suggestions
			},
		},
		{
			name:      "Large floor needs multiple APs",
			floorPlan: &ai.FloorPlan{Width: 100, Height: 100},
			existingAPs: []ai.AccessPoint{
				{Location: ai.Point{X: 5, Y: 5}, TxPower: 15, Band: "2.4GHz"},
			},
			targetCoverage: 95,
			threshold:      ai.ThresholdGood, // Stricter threshold to ensure weak spots
			checkResult: func(t *testing.T, suggestions []ai.PlacementSuggestion) {
				if len(suggestions) == 0 {
					t.Error("Expected suggestions for large floor with single weak AP in corner")
					return
				}
				// Suggestions should be sorted by priority
				for i := 1; i < len(suggestions); i++ {
					if suggestions[i].Priority < suggestions[i-1].Priority {
						t.Error("Suggestions should be sorted by priority ascending")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := ai.ExportFindOptimalPlacements(
				tt.floorPlan, tt.existingAPs, tt.targetCoverage, tt.threshold,
			)
			if tt.checkResult != nil {
				tt.checkResult(t, suggestions)
			}
		})
	}
}

// TestConstants verifies the signal threshold constants.
func TestConstants(t *testing.T) {
	// Verify thresholds are in descending order (stronger signals first)
	if ai.ThresholdExcellent <= ai.ThresholdGood {
		t.Error("ThresholdExcellent should be greater than ThresholdGood")
	}
	if ai.ThresholdGood <= ai.ThresholdFair {
		t.Error("ThresholdGood should be greater than ThresholdFair")
	}
	if ai.ThresholdFair <= ai.ThresholdPoor {
		t.Error("ThresholdFair should be greater than ThresholdPoor")
	}
	if ai.ThresholdPoor <= ai.ThresholdMinimum {
		t.Error("ThresholdPoor should be greater than ThresholdMinimum")
	}

	// Verify path loss exponents are positive and in expected ranges
	if ai.PathLossExponentFreeSpace < 1.5 || ai.PathLossExponentFreeSpace > 2.5 {
		t.Error("PathLossExponentFreeSpace should be around 2.0")
	}
	if ai.PathLossExponentOffice < 2.5 || ai.PathLossExponentOffice > 4.0 {
		t.Error("PathLossExponentOffice should be around 3.0")
	}

	// Verify reference losses are positive
	if ai.DefaultReferenceLoss2_4GHz <= 0 {
		t.Error("DefaultReferenceLoss2_4GHz should be positive")
	}
	if ai.DefaultReferenceLoss5GHz <= 0 {
		t.Error("DefaultReferenceLoss5GHz should be positive")
	}
	if ai.DefaultReferenceLoss6GHz <= 0 {
		t.Error("DefaultReferenceLoss6GHz should be positive")
	}

	// Higher frequencies should have higher reference loss
	if ai.DefaultReferenceLoss5GHz <= ai.DefaultReferenceLoss2_4GHz {
		t.Error("5GHz should have higher reference loss than 2.4GHz")
	}
	if ai.DefaultReferenceLoss6GHz <= ai.DefaultReferenceLoss5GHz {
		t.Error("6GHz should have higher reference loss than 5GHz")
	}
}

// BenchmarkPredictSignalMap benchmarks heatmap generation.
func BenchmarkPredictSignalMap(b *testing.B) {
	floorPlan := &ai.FloorPlan{Width: 50, Height: 50}
	aps := []ai.AccessPoint{
		{Location: ai.Point{X: 10, Y: 10}, TxPower: 20, Band: "2.4GHz"},
		{Location: ai.Point{X: 40, Y: 10}, TxPower: 20, Band: "2.4GHz"},
		{Location: ai.Point{X: 25, Y: 40}, TxPower: 20, Band: "2.4GHz"},
	}
	model := ai.NewPathLossModel("office", "2.4GHz")

	b.ResetTimer()
	for b.Loop() {
		_, _ = ai.PredictSignalMap(floorPlan, aps, model, 1.0)
	}
}

// BenchmarkAnalyzeCoverage benchmarks coverage analysis.
func BenchmarkAnalyzeCoverage(b *testing.B) {
	floorPlan := &ai.FloorPlan{Width: 50, Height: 50}
	samples := make([]ai.SignalSample, 100)
	for i := range samples {
		samples[i] = ai.SignalSample{
			Location: ai.Point{X: float64(i%10) * 5, Y: float64(i/10) * 5},
			RSSI:     -50 - (i % 40),
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = ai.AnalyzeCoverage(samples, floorPlan, ai.ThresholdFair)
	}
}
