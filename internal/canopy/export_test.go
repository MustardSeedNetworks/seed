package canopy

import (
	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// FrequencyToBand exports frequencyToBand for testing.
func FrequencyToBand(freq int) WiFiBand {
	return frequencyToBand(freq)
}

// ChannelToFrequency exports channelToFrequency for testing.
func ChannelToFrequency(channel int) int {
	return channelToFrequency(channel)
}

// IsDFSChannel exports isDFSChannel for testing.
func IsDFSChannel(channel int) bool {
	return isDFSChannel(channel)
}

// ConvertScannedNetwork exports convertScannedNetwork for testing.
func ConvertScannedNetwork(n *wifi.ScannedNetwork) WiFiNetwork {
	return convertScannedNetwork(n)
}

// ConvertSurvey exports convertSurvey for testing.
func ConvertSurvey(s *survey.Survey) *Survey {
	return convertSurvey(s)
}

// SetWiFiServiceScanner sets the scanner for testing (allows injecting mocks).
func (s *WiFiService) SetScanner(scanner *wifi.Scanner) {
	s.scanner = scanner
}

// SetWiFiServiceManager sets the manager for testing (allows injecting mocks).
func (s *WiFiService) SetManager(manager *wifi.Manager) {
	s.manager = manager
}

// SetWiFiServiceAvailable sets the available flag for testing.
func (s *WiFiService) SetAvailable(available bool) {
	s.available = available
}

// GetSurveyManager returns the survey manager for testing.
func (s *SurveyService) GetSurveyManagerInternal() *survey.Manager {
	return s.manager
}

// GetChannelScanner returns the scanner from channel service for testing.
func (s *ChannelService) GetScanner() *wifi.Scanner {
	return s.scanner
}

// NewModuleWithNilDB creates a Module with nil database for testing purposes.
// This is useful for testing module creation without database dependency.
func NewModuleWithNilDB() *Module {
	return &Module{}
}

// SetModuleWiFi sets the WiFi service on a module for testing.
func (m *Module) SetWiFi(wifi *WiFiService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wifi = wifi
}

// SetModuleSurvey sets the Survey service on a module for testing.
func (m *Module) SetSurvey(survey *SurveyService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.survey = survey
}

// SetModuleChannel sets the Channel service on a module for testing.
func (m *Module) SetChannel(channel *ChannelService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channel = channel
}

// SetModuleAI sets the AI service on a module for testing.
func (m *Module) SetAI(ai *AIService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ai = ai
}
