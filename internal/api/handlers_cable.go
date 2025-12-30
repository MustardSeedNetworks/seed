// Package api provides the HTTP/WebSocket server.
// handlers_cable.go contains cable test handlers.
// Split from handlers_network.go for code organization (Plan F).
package api

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/sap/cable"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Cable Test Types
// ============================================================================

// CableResponse represents the cable test results for the API.
type CableResponse struct {
	Supported   bool              `json:"supported"`
	Status      string            `json:"status"`
	Length      *float64          `json:"length,omitempty"`   // meters (overall or to first fault)
	LengthFt    *float64          `json:"lengthFt,omitempty"` // feet
	Pairs       []CablePairResult `json:"pairs,omitempty"`    // Per-pair results
	Faults      []string          `json:"faults"`
	WiringStd   string            `json:"wiringStandard"`        // 568A or 568B
	Pinout      []CablePinout     `json:"pinout,omitempty"`      // Pin-to-color mapping
	IsCrossover bool              `json:"isCrossover,omitempty"` // True if crossover cable
	DriverName  string            `json:"driverName,omitempty"`  // NIC driver
}

// CablePairResult represents TDR test results for a single twisted pair.
type CablePairResult struct {
	Pair       string   `json:"pair"`               // "1-2", "3-6", "4-5", "7-8"
	PairLetter string   `json:"pairLetter"`         // "A", "B", "C", "D"
	Status     string   `json:"status"`             // ok, open, short, etc.
	LengthM    *float64 `json:"lengthM,omitempty"`  // Distance in meters
	LengthFt   *float64 `json:"lengthFt,omitempty"` // Distance in feet
}

// CablePinout represents a pin-to-color mapping for wiring standard display.
type CablePinout struct {
	Pin   int    `json:"pin"`
	Color string `json:"color"`
	Pair  string `json:"pair"` // Which pair this pin belongs to
}

// ============================================================================
// Cable Test Handler
// ============================================================================

// handleCable performs a cable test and returns results.
// Accepts optional query parameter: ?standard=568A (default: 568B).
func (s *Server) handleCable(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	if s.cableTester == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeServiceUnavail, localizer.TWithData("errors.service.notAvailable", map[string]interface{}{"service": "Cable tester"}), "")
		return
	}

	result := s.cableTester.Test()

	// Allow wiring standard override via query param
	wiringStd := cable.Wiring568B // Default to 568B (most common)
	if std := r.URL.Query().Get("standard"); std == "568A" {
		wiringStd = cable.Wiring568A
	}

	resp := CableResponse{
		Supported:   result.Supported,
		Status:      string(result.Status),
		Length:      result.Length,
		LengthFt:    result.LengthFt,
		Faults:      result.Faults,
		WiringStd:   string(wiringStd),
		IsCrossover: result.IsCrossover,
		DriverName:  result.DriverName,
	}

	// Convert per-pair results
	if len(result.Pairs) > 0 {
		resp.Pairs = make([]CablePairResult, len(result.Pairs))
		for i, pair := range result.Pairs {
			resp.Pairs[i] = CablePairResult{
				Pair:       pair.Pair,
				PairLetter: pair.PairLetter,
				Status:     string(pair.Status),
				LengthM:    pair.LengthM,
				LengthFt:   pair.LengthFt,
			}
		}
	}

	// Get pinout for requested wiring standard
	pinout := cable.GetPinout(wiringStd)
	resp.Pinout = make([]CablePinout, len(pinout))
	for i, p := range pinout {
		resp.Pinout[i] = CablePinout{
			Pin:   p.Pin,
			Color: p.Color,
			Pair:  p.Pair,
		}
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}
