package api

//
// This file contains handlers for CVE vulnerability detection and reporting for discovered
// network devices. It integrates with the NVD (National Vulnerability Database) to identify
// known vulnerabilities based on device profiles.
//
// Key features:
//   - Retrieve vulnerability reports for devices
//
// Dependencies:
//   - internal/discovery: Device profile and CVE scanner

import (
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	securitysettings "github.com/MustardSeedNetworks/seed/internal/security/settings"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// vulnerabilityRoutes returns the scanner's routes. A scan is the vuln-scan
// job kind, gated at compliance_advanced (Pro, LICENSE_STRATEGY §2); these
// reads stay open so prior scan output remains visible to lower tiers, and
// the writes are gated by role.
func (s *Server) vulnerabilityRoutes() []route.Route {
	op := roles.Operator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	getPost := []string{http.MethodGet, http.MethodPost}
	getPut := []string{http.MethodGet, http.MethodPut}
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/security/vulnerabilities/status",
			Handler: s.handleVulnerabilityStatus,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/vulnerabilities/results",
			Handler: s.handleVulnerabilityResults,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/vulnerabilities/device",
			Handler: s.handleDeviceVulnerabilities,
			Methods: get,
			Auth:    true,
		},
		// Literals, not the vulnFindingsPath consts, so the route-consumer gate
		// sees them.
		{
			Path:    APIVersionPrefix + "/security/vulnerabilities/findings",
			Handler: s.handleVulnFindings,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/vulnerabilities/findings/",
			Handler: s.handleVulnFindingAction,
			Methods: getPost,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/vulnerabilities/settings",
			Handler: s.handleVulnerabilitySettings,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/vulnerabilities/validate-api-key",
			Handler: s.handleNVDAPIKeyValidate,
			Methods: post,
			Auth:    true,
			CSRF:    true,
		},
	}
}

// handleVulnerabilityStatus returns scanner status and statistics
// GET /api/vulnerabilities/status (fixes #703).
func (s *Server) handleVulnerabilityStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.vulnScanner() == nil {
		sendJSONResponse(w, logger, http.StatusServiceUnavailable, map[string]any{
			jsonKeyEnabled: false,
		})
		return
	}

	stats := s.vulnScanner().GetStats()

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		jsonKeyEnabled:   true,
		"scanning":       s.vulnScanner().IsRunning(),
		"stats":          stats,
		"severityFilter": s.securitySettings.VulnSeverity(),
	})
}

// handleVulnerabilityResults returns all vulnerability scan results
// GET /api/vulnerabilities/results?severity=high (optional filter) (fixes #703).
func (s *Server) handleVulnerabilityResults(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.vulnScanner() == nil {
		sendJSONResponse(w, logger, http.StatusServiceUnavailable, map[string]string{
			"error": "Vulnerability scanner not enabled",
		})
		return
	}

	results := s.vulnScanner().GetAllVulnerabilities()

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

	if s.vulnScanner() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.vulnerability.scannerNotEnabled"),
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
			localizer.T("errors.vulnerability.missingIpParam"),
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
			localizer.T("errors.vulnerability.invalidIp"),
			ip,
		) // fixes #694
		return
	}

	result := s.vulnScanner().GetDeviceVulnerabilities(ip)
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
		sendJSONResponse(w, logger, http.StatusOK, s.securitySettings.Vuln())

	case http.MethodPut:
		var settings vuln.VulnerabilityScannerConfig
		if !decodeJSONStrictLocalized(w, r, &settings, MaxBodySizeJSON,
			logger, localizer) {
			return
		}

		err := s.securitySettings.UpdateVuln(securitysettings.VulnUpdate{
			Enabled:           settings.Enabled,
			CVEDatabase:       settings.CVEDatabase,
			NVDAPIKey:         settings.NVDAPIKey,
			UpdateInterval:    settings.UpdateInterval,
			SeverityThreshold: settings.SeverityThreshold,
			MaxConcurrent:     settings.MaxConcurrent,
		})
		if err != nil {
			logger.ErrorContext(r.Context(), "Failed to save vulnerability config", "error", err)
			sendErrorResponseWithDetails(
				w, logger, http.StatusInternalServerError, ErrCodeInternal,
				localizer.T("errors.config.failedToSave"), "",
			)
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

	var req NVDAPIKeyValidateRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON,
		logger, localizer) {
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
	valid, err := vuln.ValidateNVDAPIKey(r.Context(), req.APIKey)
	if err != nil {
		logger.WarnContext(r.Context(), "NVD API key validation failed", "error", err)
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
