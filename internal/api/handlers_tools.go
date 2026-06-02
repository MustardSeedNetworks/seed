package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/services/discovery"
	"github.com/krisarmstrong/seed/internal/validation"
)

// Protocol constant for network tools (protoTCP and protoUDP defined in handlers_tests.go).
const protoICMP = "icmp"

// Network tools constants.
const (
	// dnsResolveTimeoutSec is the timeout in seconds for DNS hostname resolution.
	dnsResolveTimeoutSec = 5

	// maxTCPProbePorts is the maximum number of ports allowed in a TCP probe request.
	maxTCPProbePorts = 100

	// tcpProbeConcurrency is the number of concurrent TCP probes.
	tcpProbeConcurrency = 10

	// tracerouteDefaultTimeoutSec is the default per-hop timeout for traceroute in seconds.
	tracerouteDefaultTimeoutSec = 3

	// maxPortNumber is the maximum valid port number (65535).
	maxPortNumber = 65535

	// portScanTimeoutSec is the timeout in seconds for port scan operations.
	portScanTimeoutSec = 3

	// portScanMaxDurationMin is the maximum duration for port scan operations in minutes.
	portScanMaxDurationMin = 5

	// fingerprintDefaultTimeoutSec is the default timeout for device fingerprinting in seconds.
	fingerprintDefaultTimeoutSec = 10
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
	Target  string           `json:"target"`
	Results []TCPProbeResult `json:"results"`
}

// TCPProbeResult is the flat transport view of a single TCP probe outcome. It
// mirrors discovery.TCPProbeResult's wire shape so the published schema stays
// self-contained instead of reaching into the discovery domain package. State
// is a plain string (the domain's PortState is a string enum); Error carries
// the error message (the domain's error field serialized as an opaque "{}").
type TCPProbeResult struct {
	IP    string        `json:"ip"`
	Port  int           `json:"port"`
	State string        `json:"state"`
	TTL   int           `json:"ttl"`
	RTT   time.Duration `json:"rtt"`
	Flags uint8         `json:"flags,omitempty"`
	Error string        `json:"error,omitempty"`
}

// toTCPProbeResults maps discovery probe results onto their flat transport view.
func toTCPProbeResults(results []discovery.TCPProbeResult) []TCPProbeResult {
	out := make([]TCPProbeResult, len(results))
	for i, r := range results {
		out[i] = TCPProbeResult{
			IP:    r.IP,
			Port:  r.Port,
			State: string(r.State),
			TTL:   r.TTL,
			RTT:   r.RTT,
			Flags: r.Flags,
		}
		if r.Error != nil {
			out[i].Error = r.Error.Error()
		}
	}
	return out
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
	ctx, cancel := context.WithTimeout(context.Background(), dnsResolveTimeoutSec*time.Second)
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
	if len(ports) > maxTCPProbePorts {
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

	var req TCPProbeRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	ip, err := resolveTargetIP(req.Target)
	if err != nil {
		logger.WarnContext(r.Context(), "Invalid target", "error", err, "target", req.Target)
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
		logger.WarnContext(r.Context(), "Port validation failed", "error", err)
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
		logger.ErrorContext(r.Context(), "Failed to create TCP prober", "error", err)
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

	results := prober.ScanPorts(ctx, ip.String(), ports, tcpProbeConcurrency)

	resp := TCPProbeResponse{
		Target:  req.Target,
		Results: toTCPProbeResults(results),
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

	var req TracerouteRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
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
		timeout = tracerouteDefaultTimeoutSec * time.Second
	}

	port := req.Port
	if port <= 0 {
		if protocol == protoTCP {
			port = 80
		} else {
			port = 33434
		}
	} else if port > maxPortNumber {
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

	var req PortScanRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	// Validate target
	if err := validation.ValidateServerAddress(req.Target); err != nil {
		logger.WarnContext(r.Context(), "Invalid target", "error", err, "target", req.Target)
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
	scanner, err := discovery.NewPortScanner(portScanTimeoutSec * time.Second)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to create port scanner", "error", err)
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
	ctx, cancel := context.WithTimeout(r.Context(), portScanMaxDurationMin*time.Minute)
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

	var req struct {
		IP string `json:"ip"`
	}
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
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

	// Get existing device profile if available (discoveryService may be nil in test server)
	var existingProfile *discovery.DeviceProfile
	if svc := s.discoveryService(); svc != nil {
		if device := svc.GetDeviceByIP(req.IP); device != nil {
			existingProfile = device.Profile
		}
	}

	// Create fingerprinter with config timeout
	timeout := s.config.NetworkDiscovery.ScanTimeout
	if timeout == 0 {
		timeout = fingerprintDefaultTimeoutSec * time.Second
	}
	fingerprinter := discovery.NewFingerprinter(timeout)

	// Perform advanced probing
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	result := fingerprinter.ProbeDevice(ctx, req.IP, existingProfile)

	sendJSONResponse(w, logger, http.StatusOK, result)
}
