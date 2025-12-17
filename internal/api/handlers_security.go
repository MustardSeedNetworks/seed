// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/gateway"
)

// passwordPlaceholder is used to mask sensitive values in API responses.
const passwordPlaceholder = "*****"

// ============================================================================
// Request/Response Types and Handlers (fixes #544 - split from handlers.go)
// ============================================================================

// RogueDHCPResponse represents rogue DHCP detection status.
type RogueDHCPResponse struct {
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// RogueServersResponse contains the list of detected DHCP servers.
type RogueServersResponse struct {
	Servers         []*dhcp.RogueServer `json:"servers"`
	RogueCount      int                 `json:"rogueCount"`
	AuthorizedCount int                 `json:"authorizedCount"`
}

// RogueDHCPConfigResponse contains the rogue DHCP detector configuration.
type RogueDHCPConfigResponse struct {
	Enabled          bool     `json:"enabled"`
	KnownServers     []string `json:"knownServers"`
	AlertOnDetection bool     `json:"alertOnDetection"`
	Interface        string   `json:"interface"`
}

// handleRogueDHCP starts/stops rogue DHCP detection (POST) or gets status (GET).
func (s *Server) handleRogueDHCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get current status
		resp := RogueDHCPResponse{
			Enabled: s.config.DHCP.RogueDetection.Enabled,
			Running: s.rogueDetector.IsRunning(),
		}
		sendJSONResponse(w, http.StatusOK, resp)

	case http.MethodPost:
		// Limit request body size to prevent DoS attacks (fixes #682)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

		// Start/stop detection
		var req struct {
			Action string `json:"action"` // "start" or "stop"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		resp := RogueDHCPResponse{
			Enabled: s.config.DHCP.RogueDetection.Enabled,
		}

		switch strings.ToLower(req.Action) {
		case "start":
			if !s.config.DHCP.RogueDetection.Enabled {
				resp.Error = "Rogue DHCP detection is disabled in configuration"
				sendJSONResponse(w, http.StatusBadRequest, resp)
				return
			}
			if s.rogueDetector.IsRunning() {
				resp.Running = true
				resp.Message = "Rogue DHCP detector already running"
				sendJSONResponse(w, http.StatusOK, resp)
				return
			}
			if err := s.rogueDetector.Start(); err != nil {
				slog.Error("Failed to start rogue DHCP detector", "error", err)
				resp.Error = "internal server error"
				sendJSONResponse(w, http.StatusInternalServerError, resp)
				return
			}
			resp.Running = true
			resp.Message = "Rogue DHCP detector started"
			sendJSONResponse(w, http.StatusOK, resp)

		case "stop":
			if !s.rogueDetector.IsRunning() {
				resp.Running = false
				resp.Message = "Rogue DHCP detector not running"
				sendJSONResponse(w, http.StatusOK, resp)
				return
			}
			if err := s.rogueDetector.Stop(); err != nil {
				resp.Error = err.Error()
				sendJSONResponse(w, http.StatusInternalServerError, resp)
				return
			}
			resp.Running = false
			resp.Message = "Rogue DHCP detector stopped"
			sendJSONResponse(w, http.StatusOK, resp)

		default:
			http.Error(w, "Invalid action. Use 'start' or 'stop'", http.StatusBadRequest)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRogueDHCPServers returns detected DHCP servers (GET) or clears the list (DELETE).
func (s *Server) handleRogueDHCPServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get all detected servers
		servers := s.rogueDetector.GetDetectedServers()
		rogues := s.rogueDetector.GetRogueServers()

		resp := RogueServersResponse{
			Servers:         servers,
			RogueCount:      len(rogues),
			AuthorizedCount: len(servers) - len(rogues),
		}
		sendJSONResponse(w, http.StatusOK, resp)

	case http.MethodDelete:
		// Clear detected servers list
		s.rogueDetector.ClearDetectedServers()
		sendJSONResponse(w, http.StatusOK, map[string]string{
			"message": "Detected servers list cleared",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRogueDHCPConfig gets (GET) or updates (PUT) the rogue DHCP detector configuration.
func (s *Server) handleRogueDHCPConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get current configuration
		rogueConfig := s.rogueDetector.GetConfig()
		resp := RogueDHCPConfigResponse{
			Enabled:          s.config.DHCP.RogueDetection.Enabled,
			KnownServers:     rogueConfig.KnownServers,
			AlertOnDetection: rogueConfig.AlertOnDetection,
			Interface:        rogueConfig.Interface,
		}
		sendJSONResponse(w, http.StatusOK, resp)

	case http.MethodPut:
		// Limit request body size to prevent DoS attacks (fixes #682)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

		// Update configuration
		var req struct {
			Enabled          *bool    `json:"enabled,omitempty"`
			KnownServers     []string `json:"knownServers,omitempty"`
			AlertOnDetection *bool    `json:"alertOnDetection,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update config
		s.config.Lock()
		if req.Enabled != nil {
			s.config.DHCP.RogueDetection.Enabled = *req.Enabled
		}
		if req.KnownServers != nil {
			s.config.DHCP.RogueDetection.KnownServers = req.KnownServers
			// Update detector's known servers
			s.rogueDetector.UpdateKnownServers(req.KnownServers)
		}
		if req.AlertOnDetection != nil {
			s.config.DHCP.RogueDetection.AlertOnDetection = *req.AlertOnDetection
		}
		s.config.Unlock()

		// Save config
		if err := s.config.Save(s.configPath); err != nil {
			slog.Warn("Failed to save config", "error", err)
		}

		// Return updated config
		rogueConfig := s.rogueDetector.GetConfig()
		resp := RogueDHCPConfigResponse{
			Enabled:          s.config.DHCP.RogueDetection.Enabled,
			KnownServers:     rogueConfig.KnownServers,
			AlertOnDetection: rogueConfig.AlertOnDetection,
			Interface:        rogueConfig.Interface,
		}
		sendJSONResponse(w, http.StatusOK, resp)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GatewayResponse represents the gateway ping test results for the API.
type GatewayResponse struct {
	Gateway     string           `json:"gateway"`
	Reachable   bool             `json:"reachable"`
	Sent        int              `json:"sent"`
	Received    int              `json:"received"`
	LossPercent float64          `json:"lossPercent"`
	MinTime     float64          `json:"minTime"`
	MaxTime     float64          `json:"maxTime"`
	AvgTime     float64          `json:"avgTime"`
	LastTime    float64          `json:"lastTime"`
	Status      string           `json:"status"`
	IPv6        *GatewayResponse `json:"ipv6,omitempty"`
}

// handleGateway performs gateway ping testing and returns results.
func (s *Server) handleGateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.gatewayTester == nil {
		http.Error(w, "Gateway tester not available", http.StatusServiceUnavailable)
		return
	}

	// Perform IPv4 gateway ping test
	stats := s.gatewayTester.Test()

	resp := GatewayResponse{
		Gateway:     stats.Gateway,
		Reachable:   stats.Reachable,
		Sent:        stats.Sent,
		Received:    stats.Received,
		LossPercent: stats.LossPercent,
		MinTime:     stats.MinTime,
		MaxTime:     stats.MaxTime,
		AvgTime:     stats.AvgTime,
		LastTime:    stats.LastTime,
		Status:      string(stats.Status),
	}

	// Detect and ping IPv6 gateway if available
	ipv6Gateway, err := gateway.DetectGatewayIPv6()
	if err == nil && ipv6Gateway != "" {
		// Create a temporary tester for IPv6
		ipv6Tester := gateway.NewTester(gateway.DefaultThresholds())
		ipv6Tester.SetGateway(ipv6Gateway)
		ipv6Stats := ipv6Tester.Test()

		resp.IPv6 = &GatewayResponse{
			Gateway:     ipv6Stats.Gateway,
			Reachable:   ipv6Stats.Reachable,
			Sent:        ipv6Stats.Sent,
			Received:    ipv6Stats.Received,
			LossPercent: ipv6Stats.LossPercent,
			MinTime:     ipv6Stats.MinTime,
			MaxTime:     ipv6Stats.MaxTime,
			AvgTime:     ipv6Stats.AvgTime,
			LastTime:    ipv6Stats.LastTime,
			Status:      string(ipv6Stats.Status),
		}
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// VLANResponse represents the VLAN information for the API.

// SNMPSettingsResponse represents the SNMP configuration settings.
type SNMPSettingsResponse struct {
	Communities   []string                   `json:"communities"`
	V3Credentials []SNMPv3CredentialResponse `json:"v3Credentials"`
	Timeout       int                        `json:"timeout"` // milliseconds
	Retries       int                        `json:"retries"`
	Port          int                        `json:"port"`
}

// SNMPv3CredentialResponse represents an SNMPv3 credential for API responses.
type SNMPv3CredentialResponse struct {
	Name          string `json:"name"`
	Username      string `json:"username"`
	AuthProtocol  string `json:"authProtocol"`
	AuthPassword  string `json:"authPassword"`
	PrivProtocol  string `json:"privProtocol"`
	PrivPassword  string `json:"privPassword"`
	ContextName   string `json:"contextName"`
	SecurityLevel string `json:"securityLevel"`
}

// handleSNMPSettings handles GET/PUT for SNMP settings.
func (s *Server) handleSNMPSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getSNMPSettings(w, r)
	case http.MethodPut:
		s.updateSNMPSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getSNMPSettings(w http.ResponseWriter, _ *http.Request) {
	s.config.RLock()
	defer s.config.RUnlock()

	// Convert v3 credentials to response format (fixes #518)
	// NEVER return actual passwords in GET responses - use placeholder instead
	v3Creds := make([]SNMPv3CredentialResponse, len(s.config.SNMP.V3Credentials))
	for i := range s.config.SNMP.V3Credentials {
		cred := &s.config.SNMP.V3Credentials[i]
		// Use passwordPlaceholder for passwords (never expose actual values)
		authPass := ""
		if cred.AuthPassword != "" {
			authPass = passwordPlaceholder
		}
		privPass := ""
		if cred.PrivPassword != "" {
			privPass = passwordPlaceholder
		}

		v3Creds[i] = SNMPv3CredentialResponse{
			Name:          cred.Name,
			Username:      cred.Username,
			AuthProtocol:  cred.AuthProtocol,
			AuthPassword:  authPass, // Never return actual password
			PrivProtocol:  cred.PrivProtocol,
			PrivPassword:  privPass, // Never return actual password
			ContextName:   cred.ContextName,
			SecurityLevel: cred.SecurityLevel,
		}
	}

	resp := SNMPSettingsResponse{
		Communities:   s.config.SNMP.Communities,
		V3Credentials: v3Creds,
		Timeout:       int(s.config.SNMP.Timeout.Milliseconds()),
		Retries:       s.config.SNMP.Retries,
		Port:          s.config.SNMP.Port,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

func (s *Server) updateSNMPSettings(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var req SNMPSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Lock config for write access
	s.config.Lock()
	defer s.config.Unlock()

	// Convert request v3 credentials to config format (fixes #518)
	v3Creds := make([]config.SNMPv3Credential, len(req.V3Credentials))
	for i := range req.V3Credentials {
		cred := &req.V3Credentials[i]
		newCred := config.SNMPv3Credential{
			Name:          cred.Name,
			Username:      cred.Username,
			AuthProtocol:  cred.AuthProtocol,
			PrivProtocol:  cred.PrivProtocol,
			ContextName:   cred.ContextName,
			SecurityLevel: cred.SecurityLevel,
		}

		// Handle AuthPassword: If placeholder, keep existing; otherwise encrypt new value
		if cred.AuthPassword != "" && cred.AuthPassword != passwordPlaceholder {
			// New password provided - encrypt it
			encrypted, err := config.EncryptCredential(cred.AuthPassword, s.config.Auth.JWTSecret)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to encrypt auth password: %v", err), http.StatusInternalServerError)
				return
			}
			newCred.AuthPassword = encrypted
		} else if i < len(s.config.SNMP.V3Credentials) {
			// Keep existing password if placeholder or empty
			newCred.AuthPassword = s.config.SNMP.V3Credentials[i].AuthPassword
		}

		// Handle PrivPassword: If placeholder, keep existing; otherwise encrypt new value
		if cred.PrivPassword != "" && cred.PrivPassword != passwordPlaceholder {
			// New password provided - encrypt it
			encrypted, err := config.EncryptCredential(cred.PrivPassword, s.config.Auth.JWTSecret)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to encrypt priv password: %v", err), http.StatusInternalServerError)
				return
			}
			newCred.PrivPassword = encrypted
		} else if i < len(s.config.SNMP.V3Credentials) {
			// Keep existing password if placeholder or empty
			newCred.PrivPassword = s.config.SNMP.V3Credentials[i].PrivPassword
		}

		v3Creds[i] = newCred
	}

	// Update SNMP settings
	s.config.SNMP.Communities = req.Communities
	s.config.SNMP.V3Credentials = v3Creds
	s.config.SNMP.Timeout = time.Duration(req.Timeout) * time.Millisecond
	s.config.SNMP.Retries = req.Retries
	s.config.SNMP.Port = req.Port

	// Save config (passwords are now encrypted)
	if err := s.config.Save(s.configPath); err != nil {
		slog.Warn("Failed to save config", "error", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "SNMP settings updated",
	})
}
