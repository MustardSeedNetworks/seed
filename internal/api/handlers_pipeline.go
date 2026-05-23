package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/services/discovery"
)

// Pipeline request body size limits.
const (
	// maxPipelineRequestBodyBytes is the maximum request body size for pipeline operations (1MB).
	maxPipelineRequestBodyBytes = 1 << 20
)

// Pipeline timing profile constants.
const (
	// politeProbeDelayMs is the probe delay for polite scanning profile.
	politeProbeDelayMs = 200
	// politeHostDelayMs is the host delay for polite scanning profile.
	politeHostDelayMs = 100
	// politeMaxConcurrentHosts is max concurrent hosts for polite scanning profile.
	politeMaxConcurrentHosts = 5
	// politePhaseTimeoutMins is phase timeout in minutes for polite scanning profile.
	politePhaseTimeoutMins = 30

	// normalProbeDelayMs is the probe delay for normal scanning profile.
	normalProbeDelayMs = 50
	// normalHostDelayMs is the host delay for normal scanning profile.
	normalHostDelayMs = 20
	// normalMaxConcurrentHosts is max concurrent hosts for normal scanning profile.
	normalMaxConcurrentHosts = 20
	// normalPhaseTimeoutMins is phase timeout in minutes for normal scanning profile.
	normalPhaseTimeoutMins = 10

	// aggressiveProbeDelayMs is the probe delay for aggressive scanning profile.
	aggressiveProbeDelayMs = 10
	// aggressiveHostDelayMs is the host delay for aggressive scanning profile.
	aggressiveHostDelayMs = 5
	// aggressiveMaxConcurrentHosts is max concurrent hosts for aggressive scanning profile.
	aggressiveMaxConcurrentHosts = 100
	// aggressivePhaseTimeoutMins is phase timeout in minutes for aggressive scanning profile.
	aggressivePhaseTimeoutMins = 5
)

// handlePipelineStatus returns the current pipeline status (GET /api/pipeline/status).
func (s *Server) handlePipelineStatus(w http.ResponseWriter, _ *http.Request) {
	if s.pipeline() == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	status := s.pipeline().GetStatus()
	sendJSONResponse(w, nil, http.StatusOK, status)
}

// handlePipelineStart starts a new pipeline run (POST /api/pipeline/start).
func (s *Server) handlePipelineStart(w http.ResponseWriter, r *http.Request) {
	if s.pipeline() == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	// Fixes #925: Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, maxPipelineRequestBodyBytes)

	// Parse optional config override from request body
	var req struct {
		Config *discovery.PipelineConfig `json:"config,omitempty"`
	}

	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logging.GetLogger().WarnContext(r.Context(), "Failed to parse pipeline start request", "error", err)
			// Continue with existing config
		}
	}

	// Update config if provided
	if req.Config != nil {
		if err := s.pipeline().UpdateConfig(req.Config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Fixes #908: Use background context - pipeline should outlive the HTTP request.
	// The request context is cancelled when the HTTP response is sent, but the
	// pipeline runs asynchronously and should continue until complete or cancelled.
	run, err := s.pipeline().Start(context.Background(), "api")
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	logging.GetLogger().InfoContext(r.Context(), "Pipeline started via API", "runId", run.ID)
	sendJSONResponse(w, nil, http.StatusOK, run)
}

// handlePipelineCancel cancels the current pipeline run (POST /api/pipeline/cancel).
func (s *Server) handlePipelineCancel(w http.ResponseWriter, r *http.Request) {
	if s.pipeline() == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	if err := s.pipeline().Cancel(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logging.GetLogger().InfoContext(r.Context(), "Pipeline canceled via API")
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
	if s.pipeline() == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	config := s.pipeline().GetConfig()
	sendJSONResponse(w, nil, http.StatusOK, config)
}

// handlePipelineConfigUpdate updates the pipeline configuration (PUT /api/pipeline/config).
// Fixes #883: Wrap pipeline update and config save in atomic transaction to prevent race.
func (s *Server) handlePipelineConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if s.pipeline() == nil {
		http.Error(w, "Pipeline not initialized", http.StatusServiceUnavailable)
		return
	}

	// Fixes #925: Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, maxPipelineRequestBodyBytes)

	var config discovery.PipelineConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate port scan intensity requires acknowledgment for comprehensive
	if config.PortScan.Intensity == discovery.PortScanComprehensive {
		// Check for acknowledgment header
		if r.Header.Get("X-Acknowledge-Ids-Risk") != "true" {
			http.Error(w, "Comprehensive port scanning may trigger IDS/IPS alerts. "+
				"Set X-Acknowledge-IDS-Risk: true header to proceed.", http.StatusPreconditionRequired)
			return
		}
	}

	// Fixes #883: Hold config lock during both pipeline update and config file save
	// to ensure atomicity and prevent race conditions
	s.config.Lock()

	// Update pipeline with new config while holding the lock
	if err := s.pipeline().UpdateConfig(&config); err != nil {
		s.config.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Now update the config file (both operations are atomic)
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

	logging.GetLogger().InfoContext(r.Context(), "Pipeline config updated via API")
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
			PortCount:   len(discovery.GetQuickPorts()),
			Description: "Minimal ports for basic device identification (SSH, HTTP/S, Telnet)",
			IDSRisk:     "very_low",
		},
		{
			Level:       "standard",
			PortCount:   len(discovery.GetStandardPorts()),
			Description: "Common enterprise services (databases, email, file sharing, etc.)",
			IDSRisk:     "low",
		},
		{
			Level:       "comprehensive",
			PortCount:   len(discovery.GetComprehensivePorts()),
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
			ProbeDelayMs:       politeProbeDelayMs,
			HostDelayMs:        politeHostDelayMs,
			MaxConcurrentHosts: politeMaxConcurrentHosts,
			PhaseTimeoutMins:   politePhaseTimeoutMins,
			Description:        "Slow, deliberate scanning to avoid detection",
			UseCase:            "Production networks, IDS-sensitive environments",
		},
		{
			Profile:            "normal",
			ProbeDelayMs:       normalProbeDelayMs,
			HostDelayMs:        normalHostDelayMs,
			MaxConcurrentHosts: normalMaxConcurrentHosts,
			PhaseTimeoutMins:   normalPhaseTimeoutMins,
			Description:        "Balanced speed and stealth",
			UseCase:            "Most enterprise environments",
		},
		{
			Profile:            "aggressive",
			ProbeDelayMs:       aggressiveProbeDelayMs,
			HostDelayMs:        aggressiveHostDelayMs,
			MaxConcurrentHosts: aggressiveMaxConcurrentHosts,
			PhaseTimeoutMins:   aggressivePhaseTimeoutMins,
			Description:        "Maximum speed, may trigger alerts",
			UseCase:            "Lab environments, isolated networks",
		},
	}

	sendJSONResponse(w, nil, http.StatusOK, profiles)
}
