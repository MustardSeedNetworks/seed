// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/config"
	"github.com/krisarmstrong/luminetiq/internal/survey"
)

// ============================================================================
// Request/Response Types and Handlers (fixes #544 - split from handlers.go)
// ============================================================================

// DiscoveryResponse contains the status and results of network discovery operations.
type DiscoveryResponse struct {
	Running   bool                    `json:"running"`   // True if discovery managers are actively capturing
	Neighbors []DiscoveryNeighborInfo `json:"neighbors"` // All discovered neighbors (deduplicated by ChassisID + PortID)
}

// DiscoveryNeighborInfo represents a single discovered neighbor device in the API response.
//
// Information is gathered from link-layer discovery protocol advertisements (LLDP/CDP/EDP).
// Not all fields are present in every protocol - optional fields are omitted when empty.
type DiscoveryNeighborInfo struct {
	Protocol          string   `json:"protocol"`                    // "LLDP", "CDP", or "EDP"
	ChassisID         string   `json:"chassisId"`                   // Unique identifier for the neighbor device
	PortID            string   `json:"portId"`                      // Port/interface identifier on the neighbor
	PortDescription   string   `json:"portDescription,omitempty"`   // Human-readable port description
	SystemName        string   `json:"systemName,omitempty"`        // Hostname or system name of the neighbor
	SystemDescription string   `json:"systemDescription,omitempty"` // Device type, model, or software version
	Capabilities      []string `json:"capabilities,omitempty"`      // Device capabilities (bridge, router, etc.)
	ManagementAddress string   `json:"managementAddress,omitempty"` // IP address for device management
	TTL               int      `json:"ttl"`                         // Time-to-live in seconds until entry expires
	LastSeen          string   `json:"lastSeen"`                    // ISO 8601 timestamp of most recent advertisement
	SourceMAC         string   `json:"sourceMAC"`                   // MAC address of the advertising interface
}

// handleDiscovery returns discovery protocol neighbors from LLDP, CDP, and EDP.
//
// GET /api/discovery aggregates neighbors discovered from all active link-layer
// discovery protocols. The endpoint:
//   - Returns currently cached neighbors (no active scanning)
//   - Includes neighbors from LLDP (IEEE standard), CDP (Cisco), and EDP (Extreme)
//   - Deduplicates entries based on ChassisID + PortID combination
//   - Provides protocol-specific information when available
//   - Indicates whether discovery capture is currently running
//
// Neighbor information includes:
//   - Device identification (chassis ID, system name, management IP)
//   - Port information (port ID, description)
//   - Device capabilities (router, bridge, access point, etc.)
//   - Advertisement metadata (protocol, TTL, last seen timestamp)
//
// Use cases:
//   - Network topology mapping
//   - Switch/router neighbor discovery
//   - VLAN and network troubleshooting
//   - Automatic network documentation
//
// Discovery must be started separately via the discovery service.
// This endpoint only returns already-captured information.
//
// Authentication: Required
// Rate limiting: None (read-only operation)
//
// Response: 200 OK with DiscoveryResponse containing neighbors array.
func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.discoveryManager == nil {
		http.Error(w, "Discovery manager not available", http.StatusServiceUnavailable)
		return
	}

	neighbors := s.discoveryManager.GetNeighbors()
	resp := DiscoveryResponse{
		Running:   s.discoveryManager.IsRunning(),
		Neighbors: make([]DiscoveryNeighborInfo, 0, len(neighbors)),
	}

	for _, n := range neighbors {
		resp.Neighbors = append(resp.Neighbors, DiscoveryNeighborInfo{
			Protocol:          string(n.Protocol),
			ChassisID:         n.ChassisID,
			PortID:            n.PortID,
			PortDescription:   n.PortDescription,
			SystemName:        n.SystemName,
			SystemDescription: n.SystemDescription,
			Capabilities:      n.Capabilities,
			ManagementAddress: n.ManagementAddress,
			TTL:               n.TTL,
		})
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	devices := s.deviceDiscovery.GetDevices()
	status := s.deviceDiscovery.GetStatus()

	resp := map[string]interface{}{
		"devices": devices,
		"status":  status,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleDevicesScan triggers a network device scan.
func (s *Server) handleDevicesScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	// Check if scan is already in progress
	if s.deviceDiscovery.IsScanning() {
		sendJSONResponse(w, http.StatusOK, map[string]interface{}{
			"message":  "Scan already in progress",
			"scanning": true,
		})
		return
	}

	// Start scan in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := s.deviceDiscovery.Scan(ctx); err != nil {
			log.Printf("Device scan error: %v", err)
		}

		// Auto-scan for vulnerabilities if enabled
		if s.vulnScanner != nil && s.config.Security.VulnerabilityScanning.Enabled && s.config.Security.VulnerabilityScanning.AutoScan {
			log.Printf("Auto-scan: triggering vulnerability scan for %d discovered devices", s.deviceDiscovery.Count())
			devices := s.deviceDiscovery.GetDevices()

			vulnCtx, vulnCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer vulnCancel()

			for _, device := range devices {
				if _, err := s.vulnScanner.ScanDevice(vulnCtx, device); err != nil {
					log.Printf("Auto vulnerability scan failed for %s: %v", device.IP, err)
				}
			}

			// Broadcast vulnerability results
			results := s.vulnScanner.GetAllVulnerabilities()
			s.wsHub.BroadcastCardUpdate("vulnerabilities", map[string]interface{}{
				"results": results,
				"count":   len(results),
			})
			log.Printf("Auto-scan: completed vulnerability scan, found %d devices with vulnerabilities", len(results))
		}

		// Notify WebSocket clients when scan completes
		s.wsHub.Broadcast(Message{
			Type: "deviceScanComplete",
			Payload: map[string]interface{}{
				"deviceCount": s.deviceDiscovery.Count(),
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		})
	}()

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":  "Scan started",
		"scanning": true,
	})
}

// handleDevicesStatus returns the current device discovery status.
func (s *Server) handleDevicesStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	status := s.deviceDiscovery.GetStatus()
	sendJSONResponse(w, http.StatusOK, status)
}

// NetworkDiscoverySettingsResponse represents network discovery settings.
type NetworkDiscoverySettingsResponse struct {
	Enabled        bool   `json:"enabled"`
	ARPScanWorkers int    `json:"arpScanWorkers"`
	PingTimeoutMs  int64  `json:"pingTimeoutMs"`
	ScanTimeoutMs  int64  `json:"scanTimeoutMs"`
	AutoScan       bool   `json:"autoScan"`
	ScanIntervalMs int64  `json:"scanIntervalMs"`
	OUIFilePath    string `json:"ouiFilePath"`
}

// handleDevicesSettings handles GET/PUT for network discovery settings.
func (s *Server) handleDevicesSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getDevicesSettings(w, r)
	case http.MethodPut:
		s.updateDevicesSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getDevicesSettings(w http.ResponseWriter, _ *http.Request) {
	resp := NetworkDiscoverySettingsResponse{
		Enabled:        s.config.NetworkDiscovery.Enabled,
		ARPScanWorkers: s.config.NetworkDiscovery.ARPScanWorkers,
		PingTimeoutMs:  s.config.NetworkDiscovery.PingTimeout.Milliseconds(),
		ScanTimeoutMs:  s.config.NetworkDiscovery.ScanTimeout.Milliseconds(),
		AutoScan:       s.config.NetworkDiscovery.AutoScan,
		ScanIntervalMs: s.config.NetworkDiscovery.ScanInterval.Milliseconds(),
		OUIFilePath:    s.config.NetworkDiscovery.OUIFilePath,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

func (s *Server) updateDevicesSettings(w http.ResponseWriter, r *http.Request) {
	var req NetworkDiscoverySettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update config
	s.config.NetworkDiscovery.Enabled = req.Enabled
	if req.ARPScanWorkers > 0 {
		s.config.NetworkDiscovery.ARPScanWorkers = req.ARPScanWorkers
	}
	if req.PingTimeoutMs > 0 {
		s.config.NetworkDiscovery.PingTimeout = time.Duration(req.PingTimeoutMs) * time.Millisecond
	}
	if req.ScanTimeoutMs > 0 {
		s.config.NetworkDiscovery.ScanTimeout = time.Duration(req.ScanTimeoutMs) * time.Millisecond
	}
	s.config.NetworkDiscovery.AutoScan = req.AutoScan
	s.config.NetworkDiscovery.ScanInterval = time.Duration(req.ScanIntervalMs) * time.Millisecond
	if req.OUIFilePath != "" {
		s.config.NetworkDiscovery.OUIFilePath = req.OUIFilePath
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Network discovery settings updated",
	})
}

// SubnetRequest represents a subnet configuration request.
type SubnetRequest struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// SubnetResponse represents a subnet in API responses.
type SubnetResponse struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// handleDevicesSubnets handles GET/POST/DELETE for additional subnets.
func (s *Server) handleDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getDevicesSubnets(w, r)
	case http.MethodPost:
		s.addDevicesSubnet(w, r)
	case http.MethodPut:
		s.updateDevicesSubnet(w, r)
	case http.MethodDelete:
		s.deleteDevicesSubnet(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getDevicesSubnets(w http.ResponseWriter, _ *http.Request) {
	subnets := make([]SubnetResponse, 0, len(s.config.NetworkDiscovery.AdditionalSubnets))
	for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
		subnets = append(subnets, SubnetResponse{
			CIDR:    subnet.CIDR,
			Name:    subnet.Name,
			Enabled: subnet.Enabled,
		})
	}

	sendJSONResponse(w, http.StatusOK, subnets)
}

func (s *Server) addDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	var req SubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate CIDR format
	_, _, err := net.ParseCIDR(req.CIDR)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid CIDR format: %v", err), http.StatusBadRequest)
		return
	}

	// Check for duplicates
	for _, existing := range s.config.NetworkDiscovery.AdditionalSubnets {
		if existing.CIDR == req.CIDR {
			http.Error(w, "Subnet already exists", http.StatusConflict)
			return
		}
	}

	// Add the new subnet
	newSubnet := config.SubnetConfig{
		CIDR:    req.CIDR,
		Name:    req.Name,
		Enabled: req.Enabled,
	}
	s.config.NetworkDiscovery.AdditionalSubnets = append(
		s.config.NetworkDiscovery.AdditionalSubnets,
		newSubnet,
	)

	// Update the device discovery scanner
	if s.deviceDiscovery != nil {
		enabledCIDRs := make([]string, 0)
		for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled {
				enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
			log.Printf("Warning: Failed to update scanner subnets: %v", err)
		}
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet added",
	})
}

func (s *Server) updateDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	var req SubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find and update the subnet
	found := false
	for i, existing := range s.config.NetworkDiscovery.AdditionalSubnets {
		if existing.CIDR == req.CIDR {
			s.config.NetworkDiscovery.AdditionalSubnets[i].Name = req.Name
			s.config.NetworkDiscovery.AdditionalSubnets[i].Enabled = req.Enabled
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Subnet not found", http.StatusNotFound)
		return
	}

	// Update the device discovery scanner
	if s.deviceDiscovery != nil {
		enabledCIDRs := make([]string, 0)
		for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled {
				enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
			log.Printf("Warning: Failed to update scanner subnets: %v", err)
		}
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet updated",
	})
}

func (s *Server) deleteDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	cidr := r.URL.Query().Get("cidr")
	if cidr == "" {
		http.Error(w, "CIDR parameter required", http.StatusBadRequest)
		return
	}

	// Find and remove the subnet
	found := false
	newSubnets := make([]config.SubnetConfig, 0, len(s.config.NetworkDiscovery.AdditionalSubnets))
	for _, existing := range s.config.NetworkDiscovery.AdditionalSubnets {
		if existing.CIDR == cidr {
			found = true
			continue
		}
		newSubnets = append(newSubnets, existing)
	}

	if !found {
		http.Error(w, "Subnet not found", http.StatusNotFound)
		return
	}

	s.config.NetworkDiscovery.AdditionalSubnets = newSubnets

	// Update the device discovery scanner
	if s.deviceDiscovery != nil {
		enabledCIDRs := make([]string, 0)
		for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled {
				enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
			log.Printf("Warning: Failed to update scanner subnets: %v", err)
		}
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet deleted",
	})
}

// handlePublicIP returns the public IPv4 and IPv6 addresses.
func (s *Server) handlePublicIP(w http.ResponseWriter, r *http.Request) {
	if s.publicipChecker == nil {
		http.Error(w, "Public IP checker not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return cached result or fetch if cache expired
		result := s.publicipChecker.GetPublicIP(r.Context())
		sendJSONResponse(w, http.StatusOK, result)

	case http.MethodPost:
		// Force refresh
		result := s.publicipChecker.Refresh(r.Context())
		sendJSONResponse(w, http.StatusOK, result)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDiscoveryProfile handles GET/PUT for the discovery profile.
func (s *Server) handleDiscoveryProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getDiscoveryProfile(w, r)
	case http.MethodPut:
		s.setDiscoveryProfile(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getDiscoveryProfile(w http.ResponseWriter, _ *http.Request) {
	profile := s.discoveryService.GetProfile()
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"profile": profile,
	})
}

func (s *Server) setDiscoveryProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Profile string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert string to DiscoveryProfile
	profile := config.DiscoveryProfile(req.Profile)

	// Validate profile
	switch profile {
	case config.ProfileStealth, config.ProfileStandard, config.ProfileFullScan, config.ProfileCustom:
		// Valid profile
	default:
		http.Error(w, "Invalid profile: must be stealth, standard, full_scan, or custom", http.StatusBadRequest)
		return
	}

	// Update the config
	s.config.NetworkDiscovery.Profile = profile

	// Apply the profile change to the running service
	if err := s.discoveryService.SetProfile(profile); err != nil {
		log.Printf("Failed to set discovery profile: %v", err)
		http.Error(w, "Failed to apply profile", http.StatusInternalServerError)
		return
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Profile updated to " + string(profile),
	})
}

// handleDiscoveryServiceStatus returns the current status of the discovery service.
func (s *Server) handleDiscoveryServiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.discoveryService.GetStatus()
	sendJSONResponse(w, http.StatusOK, status)
}

// handleAdvancedFingerprint performs advanced OS/service fingerprinting on a device.
// SetupStatusResponse represents the setup status response.
// WiFi Survey API Handlers

// CreateSurveyRequest contains parameters for creating a new WiFi survey.
type CreateSurveyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SurveyType  string `json:"surveyType"`
	Interface   string `json:"interface"`
}

func (s *Server) createSurvey(w http.ResponseWriter, r *http.Request) {
	var req CreateSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Survey name is required", http.StatusBadRequest)
		return
	}

	if req.Interface == "" {
		req.Interface = s.netManager.GetCurrentInterface()
	}

	newSurvey, err := s.surveyManager.CreateSurvey(req.Name, req.Description, req.Interface, survey.Type(req.SurveyType))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create survey: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, newSurvey)
}

func (s *Server) listSurveys(w http.ResponseWriter, _ *http.Request) {
	surveys := s.surveyManager.ListSurveys()
	sendJSONResponse(w, http.StatusOK, surveys)
}

func (s *Server) getSurvey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	surveyData, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, http.StatusOK, surveyData)
}

func (s *Server) deleteSurvey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.DeleteSurvey(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) startSurvey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.StartSurvey(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) pauseSurvey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.PauseSurvey(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (s *Server) completeSurvey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.CompleteSurvey(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "completed"})
}

// AddSampleRequest contains a WiFi signal sample measurement for a survey location.
type AddSampleRequest struct {
	X          int         `json:"x"`
	Y          int         `json:"y"`
	SampleData interface{} `json:"sampleData"`
}

func (s *Server) addSurveySample(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	var req AddSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.AddSample(id, req.X, req.Y, req.SampleData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "sample added"})
}

// UpdateFloorPlanRequest contains floor plan image and dimension parameters.
type UpdateFloorPlanRequest struct {
	ImageData string  `json:"imageData"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	ScaleM    float64 `json:"scaleM"`
}

func (s *Server) updateSurveyFloorPlan(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	var req UpdateFloorPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	floorPlan := &survey.FloorPlan{
		ImageData: req.ImageData,
		Width:     req.Width,
		Height:    req.Height,
		ScaleM:    req.ScaleM,
	}

	if err := s.surveyManager.UpdateFloorPlan(id, floorPlan); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "floor plan updated"})
}
