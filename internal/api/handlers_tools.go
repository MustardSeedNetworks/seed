// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/discovery"
	"github.com/krisarmstrong/luminetiq/internal/validation"
)

// ============================================================================
// Request/Response Types (fixes #544 - split from handlers.go)
// ============================================================================

// TCPProbeRequest represents a TCP probe request.
type TCPProbeRequest struct {
	Target  string `json:"target"`  // IP or hostname
	Port    int    `json:"port"`    // Single port
	Ports   []int  `json:"ports"`   // Multiple ports
	Timeout int    `json:"timeout"` // Timeout in ms (default 1000)
}

// TCPProbeResponse represents TCP probe results.
type TCPProbeResponse struct {
	Target  string                     `json:"target"`
	Results []discovery.TCPProbeResult `json:"results"`
}

// handleTCPProbe handles TCP port probe requests.
func (s *Server) handleTCPProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TCPProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate target
	if req.Target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}

	// Resolve hostname if needed
	ip := net.ParseIP(req.Target)
	if ip == nil {
		// Try to resolve hostname
		ips, err := net.LookupIP(req.Target)
		if err != nil || len(ips) == 0 {
			http.Error(w, "Unable to resolve hostname", http.StatusBadRequest)
			return
		}
		// Use first IPv4 address
		for _, resolvedIP := range ips {
			if resolvedIP.To4() != nil {
				ip = resolvedIP
				break
			}
		}
		if ip == nil {
			ip = ips[0]
		}
	}

	// Build port list
	var ports []int
	if req.Port > 0 {
		ports = append(ports, req.Port)
	}
	ports = append(ports, req.Ports...)
	if len(ports) == 0 {
		http.Error(w, "At least one port is required", http.StatusBadRequest)
		return
	}

	// Limit ports to prevent abuse
	if len(ports) > 100 {
		http.Error(w, "Maximum 100 ports allowed", http.StatusBadRequest)
		return
	}

	// Set timeout
	timeout := time.Second
	if req.Timeout > 0 && req.Timeout <= 10000 {
		timeout = time.Duration(req.Timeout) * time.Millisecond
	}

	// Create prober
	prober, err := discovery.NewTCPProber(timeout)
	if err != nil {
		http.Error(w, "Failed to create prober: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer prober.Close()

	// Run probes
	ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Duration(len(ports))+5*time.Second)
	defer cancel()

	results := prober.ScanPorts(ctx, ip.String(), ports, 10)

	resp := TCPProbeResponse{
		Target:  req.Target,
		Results: results,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// TracerouteRequest represents a traceroute request.
type TracerouteRequest struct {
	Target   string `json:"target"`   // IP or hostname
	Protocol string `json:"protocol"` // "icmp", "udp", "tcp" (default: icmp)
	Port     int    `json:"port"`     // Port for TCP/UDP (default: 80 for TCP, 33434 for UDP)
	MaxHops  int    `json:"maxHops"`  // Max TTL (default: 30)
	Timeout  int    `json:"timeout"`  // Per-hop timeout in ms (default: 3000)
}

// handleTraceroute handles traceroute requests.
func (s *Server) handleTraceroute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TracerouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate target
	if req.Target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}

	// Validate target for security (must be valid IP or resolvable hostname)
	ip := net.ParseIP(req.Target)
	if ip == nil {
		// Not an IP, check if it looks like a valid hostname
		if err := validation.ValidateServerAddress(req.Target); err != nil {
			http.Error(w, "Invalid target: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Set defaults
	protocol := req.Protocol
	if protocol == "" {
		protocol = "icmp"
	}
	if protocol != "icmp" && protocol != "udp" && protocol != "tcp" {
		http.Error(w, "Protocol must be icmp, udp, or tcp", http.StatusBadRequest)
		return
	}

	maxHops := req.MaxHops
	if maxHops <= 0 || maxHops > 64 {
		maxHops = 30
	}

	timeout := time.Duration(req.Timeout) * time.Millisecond
	if timeout <= 0 || timeout > 10*time.Second {
		timeout = 3 * time.Second
	}

	port := req.Port
	if port <= 0 {
		if protocol == "tcp" {
			port = 80
		} else {
			port = 33434
		}
	}

	// Create tracer
	tracer := discovery.NewTracer(timeout, maxHops)

	// Set overall timeout for the operation
	ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Duration(maxHops)+10*time.Second)
	defer cancel()

	// Run traceroute based on protocol
	var result *discovery.TracerouteResult
	switch protocol {
	case "icmp":
		result = tracer.TraceICMP(ctx, req.Target)
	case "udp":
		result = tracer.TraceUDP(ctx, req.Target, port)
	case "tcp":
		result = tracer.TraceTCP(ctx, req.Target, port)
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// PortScanRequest represents a port scan request.
type PortScanRequest struct {
	Target  string `json:"target"`            // IP or hostname
	Ports   []int  `json:"ports,omitempty"`   // Specific ports (optional, defaults to common ports)
	Profile string `json:"profile,omitempty"` // "quick", "web", "full" (default: quick)
	Workers int    `json:"workers,omitempty"` // Concurrent workers (default: 20)
}

// handlePortScan handles port scanning with service detection.
func (s *Server) handlePortScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PortScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate target
	if err := validation.ValidateServerAddress(req.Target); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid target: %v", err)})
		return
	}

	// Create scanner
	scanner, err := discovery.NewPortScanner(3 * time.Second)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create scanner: %v", err)})
		return
	}
	defer scanner.Close()

	// Set timeout for operation
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	var result *discovery.PortScanResult

	// Determine scan type
	if len(req.Ports) > 0 {
		workers := req.Workers
		if workers <= 0 {
			workers = 20
		}
		result = scanner.ScanWithBanners(ctx, req.Target, req.Ports, workers)
	} else {
		switch req.Profile {
		case "web":
			result = scanner.WebScan(ctx, req.Target)
		case "full":
			result = scanner.FullScan(ctx, req.Target)
		default: // "quick" or unspecified
			result = scanner.QuickScan(ctx, req.Target)
		}
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// POST /api/discovery/fingerprint with JSON body: {"ip": "192.168.1.1"}
func (s *Server) handleAdvancedFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.IP == "" {
		http.Error(w, "IP address is required", http.StatusBadRequest)
		return
	}

	// Get existing device profile if available
	var existingProfile *discovery.DeviceProfile
	if device := s.discoveryService.GetDeviceByIP(req.IP); device != nil {
		existingProfile = device.Profile
	}

	// Create fingerprinter with config timeout
	timeout := s.config.NetworkDiscovery.ScanTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	fingerprinter := discovery.NewFingerprinter(timeout)

	// Perform advanced probing
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	result := fingerprinter.ProbeDevice(ctx, req.IP, existingProfile)

	sendJSONResponse(w, http.StatusOK, result)
}
