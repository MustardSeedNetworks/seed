package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/krisarmstrong/seed/internal/discovery"
)

// handlePipelineStatus returns the current pipeline status (GET /api/pipeline/status).
func (s *Server) handlePipelineStatus(w http.ResponseWriter, _ *http.Request) {
	if s.pipeline == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	status := s.pipeline.GetStatus()
	sendJSONResponse(w, nil, http.StatusOK, status)
}

// handlePipelineStart starts a new pipeline run (POST /api/pipeline/start).
func (s *Server) handlePipelineStart(w http.ResponseWriter, r *http.Request) {
	if s.pipeline == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	// Parse optional config override from request body
	var req struct {
		Config *discovery.PipelineConfig `json:"config,omitempty"`
	}

	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			slog.Warn("Failed to parse pipeline start request", "error", err)
			// Continue with existing config
		}
	}

	// Update config if provided
	if req.Config != nil {
		if err := s.pipeline.UpdateConfig(req.Config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	run, err := s.pipeline.Start(r.Context(), "api")
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	slog.Info("Pipeline started via API", "runId", run.ID)
	sendJSONResponse(w, nil, http.StatusOK, run)
}

// handlePipelineCancel cancels the current pipeline run (POST /api/pipeline/cancel).
func (s *Server) handlePipelineCancel(w http.ResponseWriter, _ *http.Request) {
	if s.pipeline == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	if err := s.pipeline.Cancel(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("Pipeline canceled via API")
	sendJSONResponse(w, nil, http.StatusOK, map[string]string{"status": "canceled"})
}

// handlePipelineConfigRoute routes /api/pipeline/config to GET or PUT handlers.
func (s *Server) handlePipelineConfigRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handlePipelineConfig(w, r)
	case http.MethodPut:
		s.handlePipelineConfigUpdate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePipelineConfig returns the current pipeline configuration (GET /api/pipeline/config).
func (s *Server) handlePipelineConfig(w http.ResponseWriter, _ *http.Request) {
	if s.pipeline == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	config := s.pipeline.GetConfig()
	sendJSONResponse(w, nil, http.StatusOK, config)
}

// handlePipelineConfigUpdate updates the pipeline configuration (PUT /api/pipeline/config).
func (s *Server) handlePipelineConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if s.pipeline == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	var config discovery.PipelineConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate port scan intensity requires acknowledgment for comprehensive
	if config.PortScan.Intensity == discovery.PortScanComprehensive {
		// Check for acknowledgment header
		if r.Header.Get("X-Acknowledge-IDS-Risk") != "true" {
			http.Error(w, "Comprehensive port scanning may trigger IDS/IPS alerts. "+
				"Set X-Acknowledge-IDS-Risk: true header to proceed.", http.StatusPreconditionRequired)
			return
		}
	}

	if err := s.pipeline.UpdateConfig(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Also update the config file
	s.config.Lock()
	s.config.Pipeline.Phases.Enumeration = config.Phases.Enumeration
	s.config.Pipeline.Phases.NameResolution = config.Phases.NameResolution
	s.config.Pipeline.Phases.ServiceDiscovery = config.Phases.ServiceDiscovery
	s.config.Pipeline.Phases.VulnAssessment = config.Phases.VulnAssessment
	s.config.Pipeline.Timing.Profile = string(config.Timing.Profile)
	s.config.Pipeline.Timing.ProbeDelay = config.Timing.ProbeDelay
	s.config.Pipeline.Timing.HostDelay = config.Timing.HostDelay
	s.config.Pipeline.Timing.MaxConcurrentHosts = config.Timing.MaxConcurrentHosts
	s.config.Pipeline.Timing.PhaseTimeout = config.Timing.PhaseTimeout
	s.config.Pipeline.PortScan.Intensity = string(config.PortScan.Intensity)
	s.config.Pipeline.PortScan.CustomPorts = config.PortScan.CustomPorts
	s.config.Pipeline.PortScan.BannerGrab = config.PortScan.BannerGrab
	s.config.Pipeline.PortScan.ConnectTimeout = config.PortScan.ConnectTimeout
	s.config.Pipeline.SNMPCollection.Enabled = config.SNMPCollection.Enabled
	s.config.Pipeline.SNMPCollection.MIBs.System = config.SNMPCollection.MIBs.System
	s.config.Pipeline.SNMPCollection.MIBs.Interfaces = config.SNMPCollection.MIBs.Interfaces
	s.config.Pipeline.SNMPCollection.MIBs.IPAddresses = config.SNMPCollection.MIBs.IPAddresses
	s.config.Pipeline.SNMPCollection.MIBs.Routing = config.SNMPCollection.MIBs.Routing
	s.config.Pipeline.SNMPCollection.MIBs.Bridge = config.SNMPCollection.MIBs.Bridge
	s.config.Pipeline.SNMPCollection.MIBs.Entity = config.SNMPCollection.MIBs.Entity
	s.config.Pipeline.SNMPCollection.MIBs.LLDP = config.SNMPCollection.MIBs.LLDP
	s.config.Pipeline.SNMPCollection.MIBs.VLAN = config.SNMPCollection.MIBs.VLAN
	s.config.Pipeline.SNMPCollection.WalkTimeout = config.SNMPCollection.WalkTimeout
	s.config.Pipeline.SNMPCollection.MaxOIDsPerRequest = config.SNMPCollection.MaxOIDsPerRequest
	s.config.Pipeline.Persistence.StoreHistory = config.Persistence.StoreHistory
	s.config.Pipeline.Persistence.StalenessThreshold = config.Persistence.StalenessThreshold
	s.config.Pipeline.Persistence.PurgeAfter = config.Persistence.PurgeAfter
	s.config.Unlock()

	slog.Info("Pipeline config updated via API")
	sendJSONResponse(w, nil, http.StatusOK, config)
}

// handlePipelinePortIntensityInfo returns information about port scan intensity levels (GET /api/pipeline/port-intensity).
func (s *Server) handlePipelinePortIntensityInfo(w http.ResponseWriter, _ *http.Request) {
	type PortIntensityInfo struct {
		Level       string `json:"level"`
		PortCount   int    `json:"portCount"`
		Description string `json:"description"`
		IDSRisk     string `json:"idsRisk"`
		Warning     string `json:"warning,omitempty"`
	}

	info := []PortIntensityInfo{
		{
			Level:       "off",
			PortCount:   0,
			Description: "No port scanning - passive discovery only",
			IDSRisk:     "none",
		},
		{
			Level:       "quick",
			PortCount:   len(discovery.QuickPorts),
			Description: "Minimal ports for basic device identification (SSH, HTTP/S, Telnet)",
			IDSRisk:     "very_low",
		},
		{
			Level:       "standard",
			PortCount:   len(discovery.StandardPorts),
			Description: "Common enterprise services (databases, email, file sharing, etc.)",
			IDSRisk:     "low",
		},
		{
			Level:       "comprehensive",
			PortCount:   len(discovery.ComprehensivePorts),
			Description: "Top 1000+ most common ports for thorough service enumeration",
			IDSRisk:     "medium_high",
			Warning: "WARNING: Comprehensive port scanning may trigger Intrusion Detection Systems (IDS) " +
				"or Intrusion Prevention Systems (IPS). This scan mode probes 1000+ ports per host, " +
				"may generate alerts in security monitoring systems, and could be blocked by firewalls " +
				"with rate limiting. Only use on networks you are authorized to scan.",
		},
		{
			Level:       "custom",
			PortCount:   0, // Varies
			Description: "User-defined port list",
			IDSRisk:     "varies",
		},
	}

	sendJSONResponse(w, nil, http.StatusOK, info)
}

// handlePipelineTimingProfiles returns information about timing profiles (GET /api/pipeline/timing-profiles).
func (s *Server) handlePipelineTimingProfiles(w http.ResponseWriter, _ *http.Request) {
	type TimingProfileInfo struct {
		Profile            string `json:"profile"`
		ProbeDelayMs       int64  `json:"probeDelayMs"`
		HostDelayMs        int64  `json:"hostDelayMs"`
		MaxConcurrentHosts int    `json:"maxConcurrentHosts"`
		PhaseTimeoutMins   int    `json:"phaseTimeoutMins"`
		Description        string `json:"description"`
		UseCase            string `json:"useCase"`
	}

	profiles := []TimingProfileInfo{
		{
			Profile:            "polite",
			ProbeDelayMs:       200,
			HostDelayMs:        100,
			MaxConcurrentHosts: 5,
			PhaseTimeoutMins:   30,
			Description:        "Slow, deliberate scanning to avoid detection",
			UseCase:            "Production networks, IDS-sensitive environments",
		},
		{
			Profile:            "normal",
			ProbeDelayMs:       50,
			HostDelayMs:        20,
			MaxConcurrentHosts: 20,
			PhaseTimeoutMins:   10,
			Description:        "Balanced speed and stealth",
			UseCase:            "Most enterprise environments",
		},
		{
			Profile:            "aggressive",
			ProbeDelayMs:       10,
			HostDelayMs:        5,
			MaxConcurrentHosts: 100,
			PhaseTimeoutMins:   5,
			Description:        "Maximum speed, may trigger alerts",
			UseCase:            "Lab environments, isolated networks",
		},
	}

	sendJSONResponse(w, nil, http.StatusOK, profiles)
}
