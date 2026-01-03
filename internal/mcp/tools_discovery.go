package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/krisarmstrong/seed/internal/discovery"
)

// registerDiscoveryTools registers all discovery-related MCP tools.
func (s *Server) registerDiscoveryTools(isAllowed func(string) bool) {
	s.registerNetworkScanTools(isAllowed)
	s.registerDeviceTools(isAllowed)
	s.registerProbeTools(isAllowed)
}

// registerNetworkScanTools registers network scanning tools.
func (s *Server) registerNetworkScanTools(isAllowed func(string) bool) {
	s.addTool("network_scan", isAllowed,
		mcp.NewTool(
			"network_scan",
			mcp.WithDescription(
				"Scan the local network for devices using ARP/ICMP/NDP protocols. Returns a list of discovered devices with their IP, MAC, hostname, and vendor information.",
			),
			mcp.WithNumber(
				"timeout",
				mcp.Description("Scan timeout in seconds (default: 30, max: 300)"),
			),
		), s.handleNetworkScan)

	s.addTool("get_devices", isAllowed,
		mcp.NewTool(
			"get_devices",
			mcp.WithDescription(
				"Get all previously discovered devices on the network. Returns devices found from previous scans without initiating a new scan.",
			),
		), s.handleGetDevices)
}

// registerDeviceTools registers device fingerprinting and neighbor tools.
func (s *Server) registerDeviceTools(isAllowed func(string) bool) {
	s.addTool("device_fingerprint", isAllowed,
		mcp.NewTool(
			"device_fingerprint",
			mcp.WithDescription(
				"Perform OS fingerprinting and service detection on a specific device. Attempts to identify the operating system, open services, and device type.",
			),
			mcp.WithString(
				"ip",
				mcp.Required(),
				mcp.Description("IP address of the device to fingerprint"),
			),
		), s.handleDeviceFingerprint)

	s.addTool("get_neighbors", isAllowed,
		mcp.NewTool(
			"get_neighbors",
			mcp.WithDescription(
				"Get network neighbors discovered via Layer 2 protocols (LLDP, CDP, EDP). Shows connected switches, routers, and other network devices.",
			),
			mcp.WithString(
				"protocol",
				mcp.Description("Filter by protocol: lldp, cdp, edp, or all (default: all)"),
			),
		), s.handleGetNeighbors)

	s.addTool("traceroute", isAllowed,
		mcp.NewTool(
			"traceroute",
			mcp.WithDescription(
				"Trace the network path to a target host, showing each hop along the route with latency information.",
			),
			mcp.WithString(
				"target",
				mcp.Required(),
				mcp.Description("Target hostname or IP address to trace"),
			),
			mcp.WithNumber(
				"max_hops",
				mcp.Description("Maximum number of hops (default: 30, max: 64)"),
			),
			mcp.WithNumber("timeout", mcp.Description("Timeout per hop in seconds (default: 3)")),
		), s.handleTraceroute)
}

// registerProbeTools registers TCP probe and port scan tools.
func (s *Server) registerProbeTools(isAllowed func(string) bool) {
	s.addTool("tcp_probe", isAllowed,
		mcp.NewTool(
			"tcp_probe",
			mcp.WithDescription(
				"Probe a specific TCP port on a host to check if it's open and measure connection latency.",
			),
			mcp.WithString(
				"host",
				mcp.Required(),
				mcp.Description("Target hostname or IP address"),
			),
			mcp.WithNumber(
				"port",
				mcp.Required(),
				mcp.Description("TCP port number to probe (1-65535)"),
			),
			mcp.WithNumber(
				"timeout",
				mcp.Description("Connection timeout in seconds (default: 5)"),
			),
		), s.handleTCPProbe)

	s.addTool("port_scan", isAllowed,
		mcp.NewTool(
			"port_scan",
			mcp.WithDescription(
				"Scan for open ports on a host with optional service banner detection. Returns a list of open ports with service information.",
			),
			mcp.WithString(
				"host",
				mcp.Required(),
				mcp.Description("Target hostname or IP address"),
			),
			mcp.WithString(
				"ports",
				mcp.Description(
					"Ports to scan: comma-separated list (e.g., '22,80,443') or range (e.g., '1-1024'). Default: common ports",
				),
			),
			mcp.WithBoolean(
				"banners",
				mcp.Description("Attempt to grab service banners (default: true)"),
			),
		), s.handlePortScan)
}

// getArguments safely extracts arguments as a map.
func getArguments(request mcp.CallToolRequest) map[string]any {
	if request.Params.Arguments == nil {
		return make(map[string]any)
	}
	if args, ok := request.Params.Arguments.(map[string]any); ok {
		return args
	}
	return make(map[string]any)
}

// handleNetworkScan handles the network_scan tool.
func (s *Server) handleNetworkScan(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := getArguments(request)
	timeout := 30 * time.Second
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		if t > 300 {
			t = 300 // Cap at 5 minutes
		}
		timeout = time.Duration(t) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	svc := s.services.GetDiscoveryService()
	if svc == nil {
		return mcp.NewToolResultError("Discovery service not available"), nil
	}

	if err := svc.Scan(ctx); err != nil {
		// ErrScanInProgress is not a failure - a scan is just already running
		// Return current devices instead of an error
		if errors.Is(err, discovery.ErrScanInProgress) {
			devices := svc.GetDevices()
			result := map[string]any{
				"status":  "scan_in_progress",
				"message": "A network scan is already in progress. Returning cached devices.",
				"devices": devices,
			}
			return formatJSONResult(result)
		}
		return mcp.NewToolResultError(fmt.Sprintf("Scan failed: %v", err)), nil
	}

	devices := svc.GetDevices()
	return formatJSONResult(devices)
}

// handleGetDevices handles the get_devices tool.
func (s *Server) handleGetDevices(
	_ context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	svc := s.services.GetDiscoveryService()
	if svc == nil {
		return mcp.NewToolResultError("Discovery service not available"), nil
	}

	devices := svc.GetDevices()
	return formatJSONResult(devices)
}

// handleDeviceFingerprint handles the device_fingerprint tool.
func (s *Server) handleDeviceFingerprint(
	_ context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	ip, _ := request.RequireString("ip")
	if ip == "" {
		return mcp.NewToolResultError(
			"ip parameter is required",
		), nil
	}

	// Find the device in discovered devices
	svc := s.services.GetDiscoveryService()
	if svc == nil {
		return mcp.NewToolResultError("Discovery service not available"), nil
	}

	devices := svc.GetDevices()
	for _, d := range devices {
		if d.IP == ip {
			// Return the profile if available
			if d.Profile != nil {
				return formatJSONResult(d.Profile)
			}
			// Return basic device info if no profile
			return formatJSONResult(d)
		}
	}

	return mcp.NewToolResultError(
		fmt.Sprintf("Device %s not found in discovered devices. Run network_scan first.", ip),
	), nil
}

// handleGetNeighbors handles the get_neighbors tool.
func (s *Server) handleGetNeighbors(
	_ context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := getArguments(request)
	protocol := "all"
	if p, ok := args["protocol"].(string); ok && p != "" {
		protocol = strings.ToLower(p)
	}

	svc := s.services.GetDiscoveryService()
	if svc == nil {
		return mcp.NewToolResultError("Discovery service not available"), nil
	}

	devices := svc.GetDevices()

	// Filter to only devices with protocol-specific info
	neighbors := make([]*discovery.DiscoveredDevice, 0)
	for _, d := range devices {
		switch protocol {
		case "lldp":
			if d.LLDPInfo != nil {
				neighbors = append(neighbors, d)
			}
		case "cdp":
			if d.CDPInfo != nil {
				neighbors = append(neighbors, d)
			}
		case "edp":
			if d.EDPInfo != nil {
				neighbors = append(neighbors, d)
			}
		default: // "all"
			if d.LLDPInfo != nil || d.CDPInfo != nil || d.EDPInfo != nil {
				neighbors = append(neighbors, d)
			}
		}
	}

	return formatJSONResult(neighbors)
}

// handleTraceroute handles the traceroute tool.
func (s *Server) handleTraceroute(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	target, _ := request.RequireString("target")
	if target == "" {
		return mcp.NewToolResultError(
			"target parameter is required",
		), nil
	}

	args := getArguments(request)
	maxHops := 30
	if h, ok := args["max_hops"].(float64); ok && h > 0 {
		maxHops = min(int(h), 64)
	}

	timeout := 3 * time.Second
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	// Check if ICMP is available
	if !s.services.IsICMPAvailable() {
		return mcp.NewToolResultError(
			"Traceroute requires raw socket capabilities (CAP_NET_RAW). Run with elevated privileges.",
		), nil
	}

	tracer := discovery.NewTracer(timeout, maxHops)
	result := tracer.TraceICMP(ctx, target)

	return formatJSONResult(result)
}

// handleTCPProbe handles the tcp_probe tool.
func (s *Server) handleTCPProbe(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	host, err := request.RequireString("host")
	if err != nil {
		return mcp.NewToolResultError("host parameter is required"), nil
	}

	port, err := request.RequireFloat("port")
	if err != nil {
		return mcp.NewToolResultError("port parameter is required"), nil
	}

	if port < 1 || port > 65535 {
		return mcp.NewToolResultError("port must be between 1 and 65535"), nil
	}

	args := getArguments(request)
	timeout := 5 * time.Second
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	prober, err := discovery.NewTCPProber(timeout)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create TCP prober: %v", err)), nil
	}
	defer func() { _ = prober.Close() }()

	result := prober.ProbeTCP(ctx, host, int(port))
	return formatJSONResult(result)
}

// handlePortScan handles the port_scan tool.
func (s *Server) handlePortScan(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	host, err := request.RequireString("host")
	if err != nil {
		return mcp.NewToolResultError("host parameter is required"), nil
	}

	args := getArguments(request)
	// Parse ports
	ports := []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 8080}
	if portsStr, ok := args["ports"].(string); ok && portsStr != "" {
		ports, err = parsePorts(portsStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid ports: %v", err)), nil
		}
	}

	scanner, err := discovery.NewPortScanner(5 * time.Second)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create port scanner: %v", err)), nil
	}
	defer func() { _ = scanner.Close() }()

	result := scanner.ScanWithBanners(ctx, host, ports, 10)
	return formatJSONResult(result)
}

// parsePortRange parses a port range like "1-1024" and returns start, end.
func parsePortRange(part string) (int, int, error) {
	rangeParts := strings.SplitN(part, "-", 2)
	if len(rangeParts) != 2 {
		return 0, 0, fmt.Errorf("invalid range: %s", part)
	}
	start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid port in range: %s", rangeParts[0])
	}
	end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid port in range: %s", rangeParts[1])
	}
	if start < 1 || end > 65535 || start > end {
		return 0, 0, fmt.Errorf("invalid port range: %d-%d", start, end)
	}
	if end-start > 1000 {
		return 0, 0, fmt.Errorf("port range too large (max 1000 ports): %d-%d", start, end)
	}
	return start, end, nil
}

// parseSinglePort parses a single port number.
func parseSinglePort(part string) (int, error) {
	p, err := strconv.Atoi(part)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %s", part)
	}
	if p < 1 || p > 65535 {
		return 0, fmt.Errorf("port out of range: %d", p)
	}
	return p, nil
}

// parsePorts parses a port specification string into a list of ports.
// Supports formats like "22,80,443" or "1-1024" or "22,80,100-200".
func parsePorts(spec string) ([]int, error) {
	ports := make([]int, 0)
	seen := make(map[int]bool)

	parts := strings.SplitSeq(spec, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			start, end, err := parsePortRange(part)
			if err != nil {
				return nil, err
			}
			for p := start; p <= end; p++ {
				if !seen[p] {
					seen[p] = true
					ports = append(ports, p)
				}
			}
		} else {
			p, err := parseSinglePort(part)
			if err != nil {
				return nil, err
			}
			if !seen[p] {
				seen[p] = true
				ports = append(ports, p)
			}
		}
	}

	if len(ports) == 0 {
		return nil, errors.New("no valid ports specified")
	}

	return ports, nil
}

// formatJSONResult formats data as a JSON result.
func formatJSONResult(data any) (*mcp.CallToolResult, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}
