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
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/discovery"
)

// handleVulnerabilityScan triggers vulnerability scan for all or specific devices
// POST /api/vulnerabilities/scan?ip=x.x.x.x (optional IP filter).
func (s *Server) handleVulnerabilityScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.vulnScanner == nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Vulnerability scanner not enabled",
		})
		return
	}

	targetIP := r.URL.Query().Get("ip")

	// Run scan in background
	go func() {
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
				log.Printf("Vulnerability scan failed for %s: %v", device.IP, err)
			}
		}

		// Broadcast results via WebSocket
		results := s.vulnScanner.GetAllVulnerabilities()
		s.wsHub.BroadcastCardUpdate("vulnerabilities", map[string]interface{}{
			"results": results,
			"count":   len(results),
		})
	}()

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status": "scan started",
	})
}

// handleVulnerabilityStatus returns scanner status and statistics
// GET /api/vulnerabilities/status.
func (s *Server) handleVulnerabilityStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.vulnScanner == nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	stats := s.vulnScanner.GetStats()

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"enabled":        true,
		"scanning":       s.vulnScanner.IsRunning(),
		"stats":          stats,
		"severityFilter": s.config.Security.VulnerabilityScanning.SeverityThreshold,
	})
}

// handleVulnerabilityResults returns all vulnerability scan results
// GET /api/vulnerabilities/results?severity=high (optional filter).
func (s *Server) handleVulnerabilityResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.vulnScanner == nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, map[string]string{
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

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

// handleDeviceVulnerabilities returns vulnerabilities for a specific device
// GET /api/vulnerabilities/device?ip=x.x.x.x.
func (s *Server) handleDeviceVulnerabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.vulnScanner == nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Vulnerability scanner not enabled",
		})
		return
	}

	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "Missing 'ip' parameter", http.StatusBadRequest)
		return
	}

	result := s.vulnScanner.GetDeviceVulnerabilities(ip)
	if result == nil {
		sendJSONResponse(w, http.StatusNotFound, map[string]string{
			"error": "No vulnerability data for device",
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// handleVulnerabilitySettings returns or updates vulnerability scanner settings
// GET/PUT /api/vulnerabilities/settings.
func (s *Server) handleVulnerabilitySettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, http.StatusOK, s.config.Security.VulnerabilityScanning)

	case http.MethodPut:
		var settings discovery.VulnerabilityScannerConfig
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Update config
		s.config.Lock()
		s.config.Security.VulnerabilityScanning.Enabled = settings.Enabled
		s.config.Security.VulnerabilityScanning.CVEDatabase = settings.CVEDatabase
		s.config.Security.VulnerabilityScanning.NVDAPIKey = settings.NVDAPIKey
		s.config.Security.VulnerabilityScanning.UpdateInterval = settings.UpdateInterval
		s.config.Security.VulnerabilityScanning.SeverityThreshold = settings.SeverityThreshold
		s.config.Security.VulnerabilityScanning.MaxConcurrent = settings.MaxConcurrent
		s.config.Unlock()

		// Save config
		if err := s.config.Save(s.configPath); err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, map[string]string{
				"error": "Failed to save config",
			})
			return
		}

		sendJSONResponse(w, http.StatusOK, map[string]string{
			"status": "updated",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
