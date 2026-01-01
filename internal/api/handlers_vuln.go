// Package api provides HTTP handlers for vulnerability scanning and management.
//
// This file contains handlers for CVE vulnerability detection and reporting for discovered
// network devices. It integrates with the NVD (National Vulnerability Database) to identify
// known vulnerabilities based on device profiles.
//
// Key features:
//   - Trigger vulnerability scans for all or specific devices
//   - Retrieve vulnerability reports for devices
//   - Mark vulnerabilities as acknowledged
//   - Background scanning with timeout protection
//
// Dependencies:
//   - internal/discovery: Device profile and CVE scanner
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
)

// handleVulnerabilityScan triggers vulnerability scan for all or specific devices
// POST /api/vulnerabilities/scan?ip=x.x.x.x (optional IP filter).
func (s *Server) handleVulnerabilityScan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	if s.vulnScanner == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.vuln.scannerNotEnabled"),
			"",
		) // fixes #694
		return
	}

	targetIP := r.URL.Query().Get("ip")

	// Validate IP if provided
	if targetIP != "" && !validation.IsValidIP(targetIP) {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.vuln.invalidIp"),
			targetIP,
		) // fixes #694
		return
	}

	// Check if scan is already in progress
	if s.vulnScanner.IsRunning() {
		sendJSONResponse(w, logger, http.StatusOK, map[string]any{
			"status":  "scan already in progress",
			"running": true,
		})
		return
	}

	// Run scan in background (fixes #698 - timeout protection)
	go func(reqCtx context.Context) {
		bgLogger := logging.FromContext(reqCtx)
		// Add timeout protection for vulnerability scan operations (fixes #698)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var devices []*discovery.DiscoveredDevice

		if targetIP != "" {
			// Scan specific device
			device := s.deviceDiscovery.GetDeviceByIP(targetIP)
			if device != nil {
				devices = append(devices, device)
			}
		} else {
			// Scan all discovered devices
			devices = s.deviceDiscovery.GetDevices()
		}

		// Scan each device
		for _, device := range devices {
			if _, err := s.vulnScanner.ScanDevice(ctx, device); err != nil {
				bgLogger.Warn("Vulnerability scan failed", "device_ip", device.IP, "error", err)
			}
		}

		// Broadcast results via WebSocket
		results := s.vulnScanner.GetAllVulnerabilities()
		s.wsHub.BroadcastCardUpdate("vulnerabilities", map[string]any{
			"results": results,
			"count":   len(results),
		})
	}(r.Context())

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status": "scan started",
	})
}

// handleVulnerabilityStatus returns scanner status and statistics
// GET /api/vulnerabilities/status (fixes #703).
func (s *Server) handleVulnerabilityStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	if s.vulnScanner == nil {
		sendJSONResponse(w, logger, http.StatusServiceUnavailable, map[string]any{
			"enabled": false,
		})
		return
	}

	stats := s.vulnScanner.GetStats()

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"enabled":        true,
		"scanning":       s.vulnScanner.IsRunning(),
		"stats":          stats,
		"severityFilter": s.config.Security.VulnerabilityScanning.SeverityThreshold,
	})
}

// handleVulnerabilityResults returns all vulnerability scan results
// GET /api/vulnerabilities/results?severity=high (optional filter) (fixes #703).
func (s *Server) handleVulnerabilityResults(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	if s.vulnScanner == nil {
		sendJSONResponse(w, logger, http.StatusServiceUnavailable, map[string]string{
			"error": "Vulnerability scanner not enabled",
		})
		return
	}

	results := s.vulnScanner.GetAllVulnerabilities()

	// Optional severity filter
	if severityFilter := r.URL.Query().Get("severity"); severityFilter != "" {
		filtered := make([]*discovery.DeviceVulnerabilities, 0)
		for _, result := range results {
			for i := range result.Vulnerabilities {
				if strings.EqualFold(result.Vulnerabilities[i].Severity, severityFilter) {
					filtered = append(filtered, result)
					break
				}
			}
		}
		results = filtered
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"results": results,
		"count":   len(results),
	})
}

// handleDeviceVulnerabilities returns vulnerabilities for a specific device
// GET /api/vulnerabilities/device?ip=x.x.x.x (fixes #703).
func (s *Server) handleDeviceVulnerabilities(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	if s.vulnScanner == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.vuln.scannerNotEnabled"),
			"",
		) // fixes #694
		return
	}

	ip := r.URL.Query().Get("ip")
	if ip == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.vuln.missingIpParam"),
			"",
		) // fixes #694
		return
	}

	// Validate IP address
	if !validation.IsValidIP(ip) {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.vuln.invalidIp"),
			ip,
		) // fixes #694
		return
	}

	result := s.vulnScanner.GetDeviceVulnerabilities(ip)
	if result == nil {
		sendJSONResponse(w, logger, http.StatusNotFound, map[string]string{
			"error": "No vulnerability data for device",
		})
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// handleVulnerabilitySettings returns or updates vulnerability scanner settings
// GET/PUT /api/vulnerabilities/settings (fixes #703).
func (s *Server) handleVulnerabilitySettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, logger, http.StatusOK, s.config.Security.VulnerabilityScanning)

	case http.MethodPut:
		var settings discovery.VulnerabilityScannerConfig
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			logger.Warn("Invalid JSON for vulnerability settings", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusBadRequest,
				ErrCodeBadRequest,
				localizer.T("errors.vuln.invalidJson"),
				"",
			) // fixes #694, #H7
			return
		}

		// Update config
		// NOTE: Must unlock before Save() - Save() acquires RLock internally (fixes #783)
		s.config.Lock()
		s.config.Security.VulnerabilityScanning.Enabled = settings.Enabled
		s.config.Security.VulnerabilityScanning.CVEDatabase = settings.CVEDatabase
		s.config.Security.VulnerabilityScanning.NVDAPIKey = settings.NVDAPIKey
		s.config.Security.VulnerabilityScanning.UpdateInterval = settings.UpdateInterval
		s.config.Security.VulnerabilityScanning.SeverityThreshold = settings.SeverityThreshold
		s.config.Security.VulnerabilityScanning.MaxConcurrent = settings.MaxConcurrent
		// Unlock before Save() to avoid deadlock
		s.config.Unlock()

		// Save config
		if err := s.config.Save(s.configPath); err != nil {
			logger.Error("Failed to save vulnerability config", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusInternalServerError,
				ErrCodeInternal,
				localizer.T("errors.config.failedToSave"),
				"",
			) // fixes #694, #H7
			return
		}

		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"status": "updated",
		})

	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
	}
}

// NVDAPIKeyValidateRequest represents a request to validate an NVD API key.
type NVDAPIKeyValidateRequest struct {
	APIKey string `json:"apiKey"`
}

// NVDAPIKeyValidateResponse represents the response for NVD API key validation.
type NVDAPIKeyValidateResponse struct {
	Valid       bool   `json:"valid"`
	Message     string `json:"message"`
	RateLimit   int    `json:"rateLimit"`   // Requests per 30 seconds
	ObtainURL   string `json:"obtainUrl"`   // URL to obtain an API key
	HelpMessage string `json:"helpMessage"` // Help text for obtaining a key
}

// handleNVDAPIKeyValidate validates an NVD API key by making a test request
// POST /api/vulnerabilities/validate-api-key.
func (s *Server) handleNVDAPIKeyValidate(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	var req NVDAPIKeyValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid JSON for NVD API key validation", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.vuln.invalidJson"),
			"",
		) // fixes #H7
		return
	}

	response := NVDAPIKeyValidateResponse{
		ObtainURL:   "https://nvd.nist.gov/developers/request-an-api-key",
		HelpMessage: "Get a free NVD API key from NIST to increase your rate limit from 10 to 100 requests per 30 seconds.",
	}

	// If no API key provided, return info about how to get one
	if req.APIKey == "" {
		response.Valid = false
		response.Message = "No API key provided. You can use vulnerability scanning without an API key (rate limited to 10 requests per 30 seconds)."
		response.RateLimit = 10
		sendJSONResponse(w, logger, http.StatusOK, response)
		return
	}

	// Validate the API key by making a test request to NVD
	valid, err := discovery.ValidateNVDAPIKey(r.Context(), req.APIKey)
	if err != nil {
		logger.Warn("NVD API key validation failed", "error", err)
		response.Valid = false
		response.Message = "Failed to validate API key. Please check that the key is correct and try again."
		response.RateLimit = 10
		sendJSONResponse(w, logger, http.StatusOK, response)
		return
	}

	if valid {
		response.Valid = true
		response.Message = "API key is valid. Rate limit increased to 100 requests per 30 seconds."
		response.RateLimit = 100
	} else {
		response.Valid = false
		response.Message = "API key is invalid. Please check your key or request a new one."
		response.RateLimit = 10
	}

	sendJSONResponse(w, logger, http.StatusOK, response)
}
