// Package canopy provides WiFi scanning, planning, and site survey capabilities.
// Color: Green #2d7a3e
package canopy

import (
	"context"
	"sync"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/iperf"
)

// Module is the main Canopy module providing WiFi services.
type Module struct {
	mu      sync.RWMutex
	cfg     *config.Config
	db      *database.DB
	wifi    *WiFiService
	survey  *SurveyService
	channel *ChannelService
	ai      *AIService
}

// New creates a new Canopy module instance.
func New(cfg *config.Config, db *database.DB) *Module {
	m := &Module{
		cfg: cfg,
		db:  db,
	}

	// Create WiFi service first - other services depend on its scanner/manager
	m.wifi = NewWiFiService(cfg)

	// Create iperf manager for throughput testing in surveys
	iperfMgr := iperf.NewManager()

	// Create survey service with WiFi and iperf dependencies
	m.survey = NewSurveyService(cfg, db, m.wifi.Scanner(), m.wifi.Manager(), iperfMgr)

	// Create channel service with WiFi scanner
	m.channel = NewChannelService(cfg, m.wifi.Scanner())

	m.ai = NewAIService(cfg)

	return m
}

// WiFi returns the WiFi scanning/connection service.
func (m *Module) WiFi() *WiFiService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.wifi
}

// Survey returns the site survey service.
func (m *Module) Survey() *SurveyService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.survey
}

// Channel returns the channel analysis service.
func (m *Module) Channel() *ChannelService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channel
}

// AI returns the AI-assisted WiFi planning service.
func (m *Module) AI() *AIService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ai
}

// Start initializes and starts the Canopy module services.
func (m *Module) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize WiFi scanning if available
	if err := m.wifi.Init(); err != nil {
		// WiFi may not be available on all platforms
		// Log but don't fail
		_ = err
	}

	return nil
}

// Stop gracefully shuts down all Canopy module services.
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.survey != nil {
		m.survey.Stop()
	}

	return nil
}
