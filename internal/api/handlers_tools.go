// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
)

// Protocol constant for network tools (protoTCP and protoUDP defined in handlers_tests.go).
const protoICMP = "icmp"

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

// resolveTargetIP resolves a target string to an IP address.
func resolveTargetIP(target string) (net.IP, error) {
	if target == "" {
		return nil, errors.New("target is required")
	}
	ip := net.ParseIP(target)
	if ip != nil {
		return ip, nil
	}
	// Try to resolve hostname with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, "ip", target)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("unable to resolve hostname")
	}
	// Use first IPv4 address
	for _, resolvedIP := range ips {
		if resolvedIP.To4() != nil {
			return resolvedIP, nil
		}
	}
	return ips[0], nil
}

// validateTCPProbePorts builds and validates the port list from a request.
func validateTCPProbePorts(req *TCPProbeRequest) ([]int, error) {
	var ports []int
	if req.Port > 0 {
		ports = append(ports, req.Port)
	}
	ports = append(ports, req.Ports...)
	if len(ports) == 0 {
		return nil, errors.New("at least one port is required")
	}
	if len(ports) > 100 {
		return nil, errors.New("maximum 100 ports allowed")
	}
	return ports, nil
}

// handleTCPProbe handles TCP port probe requests.
func (s *Server) handleTCPProbe(w http.ResponseWriter, r *http.Request) {
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

	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req TCPProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	ip, err := resolveTargetIP(req.Target)
	if err != nil {
		logger.Warn("Invalid target", "error", err, "target", req.Target)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.tools.invalidTarget"),
			"",
		)
		return
	}

	ports, err := validateTCPProbePorts(&req)
	if err != nil {
		logger.Warn("Port validation failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.tools.portRequired"),
			"",
		)
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
		logger.Error("Failed to create TCP prober", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.tools.failedToCreateProber"),
			"",
		)
		return
	}
	defer func() { _ = prober.Close() }()

	// Run probes
	ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Duration(len(ports))+5*time.Second)
	defer cancel()

	results := prober.ScanPorts(ctx, ip.String(), ports, 10)

	resp := TCPProbeResponse{
		Target:  req.Target,
		Results: results,
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
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

	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req TracerouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	if errMsg := validateTracerouteTarget(req.Target); errMsg != "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.tools.invalidTarget"),
			errMsg,
		) // fixes #694
		return
	}

	protocol, maxHops, timeout, port, errMsg := parseTracerouteParams(&req)
	if errMsg != "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.tools.invalidTarget"),
			errMsg,
		) // fixes #694
		return
	}

	tracer := discovery.NewTracer(timeout, maxHops)
	ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Duration(maxHops)+10*time.Second)
	defer cancel()

	var result *discovery.TracerouteResult
	switch protocol {
	case protoICMP:
		result = tracer.TraceICMP(ctx, req.Target)
	case protoUDP:
		result = tracer.TraceUDP(ctx, req.Target, port)
	case protoTCP:
		result = tracer.TraceTCP(ctx, req.Target, port)
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

func validateTracerouteTarget(target string) string {
	if target == "" {
		return "Target is required"
	}
	if ip := net.ParseIP(target); ip == nil {
		if err := validation.ValidateServerAddress(target); err != nil {
			return "Invalid target format. Must be a valid IP address or hostname."
		}
	}
	return ""
}

func parseTracerouteParams(
	req *TracerouteRequest,
) (string, int, time.Duration, int, string) {
	protocol := req.Protocol
	if protocol == "" {
		protocol = protoICMP
	}
	if protocol != protoICMP && protocol != protoUDP && protocol != protoTCP {
		return "", 0, 0, 0, "Protocol must be icmp, udp, or tcp"
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
		if protocol == protoTCP {
			port = 80
		} else {
			port = 33434
		}
	} else if port > 65535 {
		return "", 0, 0, 0, "Port must be between 1 and 65535"
	}
	return protocol, maxHops, timeout, port, ""
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

	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req PortScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	// Validate target
	if err := validation.ValidateServerAddress(req.Target); err != nil {
		logger.Warn("Invalid target", "error", err, "target", req.Target)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.tools.invalidTarget"),
			"",
		)
		return
	}

	// Create scanner
	scanner, err := discovery.NewPortScanner(3 * time.Second)
	if err != nil {
		logger.Error("Failed to create port scanner", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.tools.failedToCreateScanner"),
			"",
		)
		return
	}
	defer func() { _ = scanner.Close() }()

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

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// POST /api/discovery/fingerprint with JSON body: {"ip": "192.168.1.1"}.
func (s *Server) handleAdvancedFingerprint(w http.ResponseWriter, r *http.Request) {
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

	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	if req.IP == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.tools.ipRequired"),
			"",
		) // fixes #694
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

	sendJSONResponse(w, logger, http.StatusOK, result)
}
