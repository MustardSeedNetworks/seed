// Package survey exports internal functions for testing.
package survey

// InterpolateColor exports interpolateColor for testing.
var InterpolateColor = interpolateColor

// FilterWeakSamples exports filterWeakSamples for testing.
var FilterWeakSamples = filterWeakSamples

// CalculateCoverageScore exports calculateCoverageScore for testing.
var CalculateCoverageScore = calculateCoverageScore

// DetermineSeverity exports determineSeverity for testing.
var DetermineSeverity = determineSeverity

// GenerateRecommendations exports generateRecommendations for testing.
var GenerateRecommendations = generateRecommendations

// SetSurvey sets a survey in the manager for testing.
func (m *Manager) SetSurvey(s *Survey) {
	m.surveys[s.ID] = s
}
