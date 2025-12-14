package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/config"
	"github.com/krisarmstrong/luminetiq/internal/dhcp"
	"github.com/krisarmstrong/luminetiq/internal/discovery"
	"github.com/krisarmstrong/luminetiq/internal/dns"
	"github.com/krisarmstrong/luminetiq/internal/gateway"
	"github.com/krisarmstrong/luminetiq/internal/iperf"
	"github.com/krisarmstrong/luminetiq/internal/survey"
	"github.com/krisarmstrong/luminetiq/internal/validation"
)


// handleSettings handles settings get/update.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r)
	case http.MethodPut:
		s.updateSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getSettings(w http.ResponseWriter, _ *http.Request) {
	// Lock config for read access
	s.config.RLock()
	defer s.config.RUnlock()

	settings := map[string]interface{}{
		"interface": map[string]interface{}{
			"current":   s.config.Interface.Default,
			"available": []string{}, // Will be populated by network module
		},
		"vlan": map[string]interface{}{
			"enabled": s.config.VLAN.Enabled,
			"id":      s.config.VLAN.ID,
		},
		"ip": map[string]interface{}{
			"mode": s.config.IP.Mode,
		},
		"thresholds": map[string]interface{}{
			"dns": map[string]int64{
				"good":    s.config.Thresholds.DNS.Warning.Milliseconds(),
				"warning": s.config.Thresholds.DNS.Critical.Milliseconds(),
			},
			"gateway": map[string]int64{
				"good":    s.config.Thresholds.Ping.Warning.Milliseconds(),
				"warning": s.config.Thresholds.Ping.Critical.Milliseconds(),
			},
			"wifi": map[string]int{
				"good":    s.config.Thresholds.WiFi.Signal.Warning,
				"warning": s.config.Thresholds.WiFi.Signal.Critical,
			},
			"customPing": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.Ping.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.Ping.Critical.Milliseconds(),
			},
			"customTcp": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.TCP.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.TCP.Critical.Milliseconds(),
			},
			"customHttp": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.HTTP.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.HTTP.Critical.Milliseconds(),
			},
			"httpTimings": map[string]map[string]int64{
				"dns": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
				},
				"tcp": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
				},
				"tls": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
				},
				"ttfb": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
				},
			},
		},
		"tests": map[string]interface{}{
			"runPerformance": s.config.Tests.RunPerformance,
			"runSpeedtest":   s.config.Tests.RunSpeedtest,
			"runIperf":       s.config.Tests.RunIperf,
			"runDiscovery":   s.config.Tests.RunDiscovery,
		},
		"speedtest": map[string]interface{}{
			"serverId":      s.config.Speedtest.ServerID,
			"autoRunOnLink": s.config.Speedtest.AutoRunOnLink,
		},
		"iperf": map[string]interface{}{
			"autoRunOnLink": s.config.Iperf.AutoRunOnLink,
			"server":        s.config.Iperf.Server,
			"port":          s.config.Iperf.Port,
			"protocol":      s.config.Iperf.Protocol,
			"direction":     s.config.Iperf.Direction,
			"duration":      s.config.Iperf.Duration,
			"serverPort":    s.config.Iperf.ServerPort,
			"enableServer":  s.config.Iperf.EnableServer,
		},
		"fabOptions": map[string]interface{}{
			"runLink":             s.config.FABOptions.RunLink,
			"runSwitch":           s.config.FABOptions.RunSwitch,
			"runVLAN":             s.config.FABOptions.RunVLAN,
			"runIPConfig":         s.config.FABOptions.RunIPConfig,
			"runGateway":          s.config.FABOptions.RunGateway,
			"runDNS":              s.config.FABOptions.RunDNS,
			"runHealthChecks":     s.config.FABOptions.RunHealthChecks,
			"runNetworkDiscovery": s.config.FABOptions.RunNetworkDiscovery,
			"runSpeedtest":        s.config.FABOptions.RunSpeedtest,
			"runIperf":            s.config.FABOptions.RunIperf,
			"runPerformance":      s.config.FABOptions.RunPerformance,
			"autoScanOnLink":      s.config.FABOptions.AutoScanOnLink,
		},
		"displayOptions": map[string]interface{}{
			"showPublicIP": s.config.DisplayOptions.ShowPublicIP,
		},
	}

	sendJSONResponse(w, http.StatusOK, settings)
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Lock config for write access
	s.config.Lock()
	defer s.config.Unlock()

	// Apply threshold updates
	if thresholds, ok := updates["thresholds"].(map[string]interface{}); ok {
		if dnsThresh, ok := thresholds["dns"].(map[string]interface{}); ok {
			if good, ok := dnsThresh["good"].(float64); ok {
				s.config.Thresholds.DNS.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := dnsThresh["warning"].(float64); ok {
				s.config.Thresholds.DNS.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if gwThresh, ok := thresholds["gateway"].(map[string]interface{}); ok {
			if good, ok := gwThresh["good"].(float64); ok {
				s.config.Thresholds.Ping.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := gwThresh["warning"].(float64); ok {
				s.config.Thresholds.Ping.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if wifi, ok := thresholds["wifi"].(map[string]interface{}); ok {
			if good, ok := wifi["good"].(float64); ok {
				s.config.Thresholds.WiFi.Signal.Warning = int(good)
			}
			if warning, ok := wifi["warning"].(float64); ok {
				s.config.Thresholds.WiFi.Signal.Critical = int(warning)
			}
		}
		if customPing, ok := thresholds["customPing"].(map[string]interface{}); ok {
			if good, ok := customPing["good"].(float64); ok {
				s.config.Thresholds.CustomTests.Ping.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := customPing["warning"].(float64); ok {
				s.config.Thresholds.CustomTests.Ping.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if customTcp, ok := thresholds["customTcp"].(map[string]interface{}); ok {
			if good, ok := customTcp["good"].(float64); ok {
				s.config.Thresholds.CustomTests.TCP.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := customTcp["warning"].(float64); ok {
				s.config.Thresholds.CustomTests.TCP.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if customHttp, ok := thresholds["customHttp"].(map[string]interface{}); ok {
			if good, ok := customHttp["good"].(float64); ok {
				s.config.Thresholds.CustomTests.HTTP.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := customHttp["warning"].(float64); ok {
				s.config.Thresholds.CustomTests.HTTP.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if httpTimings, ok := thresholds["httpTimings"].(map[string]interface{}); ok {
			if dnsT, ok := httpTimings["dns"].(map[string]interface{}); ok {
				if good, ok := dnsT["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.DNS.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := dnsT["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.DNS.Critical = time.Duration(warning) * time.Millisecond
				}
			}
			if tcpT, ok := httpTimings["tcp"].(map[string]interface{}); ok {
				if good, ok := tcpT["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TCP.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := tcpT["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TCP.Critical = time.Duration(warning) * time.Millisecond
				}
			}
			if tlsT, ok := httpTimings["tls"].(map[string]interface{}); ok {
				if good, ok := tlsT["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TLS.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := tlsT["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TLS.Critical = time.Duration(warning) * time.Millisecond
				}
			}
			if ttfb, ok := httpTimings["ttfb"].(map[string]interface{}); ok {
				if good, ok := ttfb["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := ttfb["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Critical = time.Duration(warning) * time.Millisecond
				}
			}
		}
	}

	// Apply tests updates
	if tests, ok := updates["tests"].(map[string]interface{}); ok {
		if runPerformance, ok := tests["runPerformance"].(bool); ok {
			s.config.Tests.RunPerformance = runPerformance
		}
		if runSpeedtest, ok := tests["runSpeedtest"].(bool); ok {
			s.config.Tests.RunSpeedtest = runSpeedtest
		}
		if runIperf, ok := tests["runIperf"].(bool); ok {
			s.config.Tests.RunIperf = runIperf
		}
		if runDiscovery, ok := tests["runDiscovery"].(bool); ok {
			s.config.Tests.RunDiscovery = runDiscovery
		}
	}

	// Apply speedtest updates
	if speedtest, ok := updates["speedtest"].(map[string]interface{}); ok {
		if serverId, ok := speedtest["serverId"].(string); ok {
			s.config.Speedtest.ServerID = serverId
		}
		if autoRunOnLink, ok := speedtest["autoRunOnLink"].(bool); ok {
			s.config.Speedtest.AutoRunOnLink = autoRunOnLink
		}
	}

	// Apply iperf updates
	if iperf, ok := updates["iperf"].(map[string]interface{}); ok {
		if autoRunOnLink, ok := iperf["autoRunOnLink"].(bool); ok {
			s.config.Iperf.AutoRunOnLink = autoRunOnLink
		}
		if server, ok := iperf["server"].(string); ok {
			s.config.Iperf.Server = server
		}
		if port, ok := iperf["port"].(float64); ok {
			p := int(port)
			if validation.ValidatePort(p) == nil {
				s.config.Iperf.Port = p
			}
		}
		if protocol, ok := iperf["protocol"].(string); ok {
			s.config.Iperf.Protocol = protocol
		}
		if direction, ok := iperf["direction"].(string); ok {
			s.config.Iperf.Direction = direction
		}
		if duration, ok := iperf["duration"].(float64); ok {
			s.config.Iperf.Duration = int(duration)
		}
		if serverPort, ok := iperf["serverPort"].(float64); ok {
			p := int(serverPort)
			if validation.ValidatePort(p) == nil {
				s.config.Iperf.ServerPort = p
			}
		}
		if enableServer, ok := iperf["enableServer"].(bool); ok {
			s.config.Iperf.EnableServer = enableServer
		}
	}

	// Apply fabOptions updates
	if fabOptions, ok := updates["fabOptions"].(map[string]interface{}); ok {
		if runLink, ok := fabOptions["runLink"].(bool); ok {
			s.config.FABOptions.RunLink = runLink
		}
		if runSwitch, ok := fabOptions["runSwitch"].(bool); ok {
			s.config.FABOptions.RunSwitch = runSwitch
		}
		if runVLAN, ok := fabOptions["runVLAN"].(bool); ok {
			s.config.FABOptions.RunVLAN = runVLAN
		}
		if runIPConfig, ok := fabOptions["runIPConfig"].(bool); ok {
			s.config.FABOptions.RunIPConfig = runIPConfig
		}
		if runGateway, ok := fabOptions["runGateway"].(bool); ok {
			s.config.FABOptions.RunGateway = runGateway
		}
		if runDNS, ok := fabOptions["runDNS"].(bool); ok {
			s.config.FABOptions.RunDNS = runDNS
		}
		if runHealthChecks, ok := fabOptions["runHealthChecks"].(bool); ok {
			s.config.FABOptions.RunHealthChecks = runHealthChecks
		}
		if runNetworkDiscovery, ok := fabOptions["runNetworkDiscovery"].(bool); ok {
			s.config.FABOptions.RunNetworkDiscovery = runNetworkDiscovery
		}
		if runSpeedtest, ok := fabOptions["runSpeedtest"].(bool); ok {
			s.config.FABOptions.RunSpeedtest = runSpeedtest
		}
		if runIperf, ok := fabOptions["runIperf"].(bool); ok {
			s.config.FABOptions.RunIperf = runIperf
		}
		if runPerformance, ok := fabOptions["runPerformance"].(bool); ok {
			s.config.FABOptions.RunPerformance = runPerformance
		}
		if autoScanOnLink, ok := fabOptions["autoScanOnLink"].(bool); ok {
			s.config.FABOptions.AutoScanOnLink = autoScanOnLink
		}
	}

	// Apply displayOptions updates
	if displayOptions, ok := updates["displayOptions"].(map[string]interface{}); ok {
		if showPublicIP, ok := displayOptions["showPublicIP"].(bool); ok {
			s.config.DisplayOptions.ShowPublicIP = showPublicIP
		}
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleInterfaces returns available network interfaces.



















// DiscoveryResponse contains all discovered neighbors.
type DiscoveryResponse struct {
	Running   bool                    `json:"running"`
	Neighbors []DiscoveryNeighborInfo `json:"neighbors"`
}

// DiscoveryNeighborInfo represents a discovered neighbor in the API.
type DiscoveryNeighborInfo struct {
	Protocol          string   `json:"protocol"`
	ChassisID         string   `json:"chassisId"`
	PortID            string   `json:"portId"`
	PortDescription   string   `json:"portDescription,omitempty"`
	SystemName        string   `json:"systemName,omitempty"`
	SystemDescription string   `json:"systemDescription,omitempty"`
	Capabilities      []string `json:"capabilities,omitempty"`
	ManagementAddress string   `json:"managementAddress,omitempty"`
	TTL               int      `json:"ttl"`
	LastSeen          string   `json:"lastSeen"`
	SourceMAC         string   `json:"sourceMAC"`
}

// handleDiscovery returns discovery protocol neighbors.
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

// DNSLookupResult represents a DNS lookup result for the API.
type DNSLookupResult struct {
	Result   string   `json:"result"`
	Time     int64    `json:"time"` // ms (deprecated, use timeMs)
	TimeMs   int64    `json:"timeMs"`
	Status   string   `json:"status"`
	Error    string   `json:"error,omitempty"`
	Resolved []string `json:"resolved,omitempty"`
}

// DNSServerTestResult represents per-server DNS test results for the API.
type DNSServerTestResult struct {
	Server      string           `json:"server"`
	Forward     *DNSLookupResult `json:"forward,omitempty"`
	ForwardIpv6 *DNSLookupResult `json:"forwardIpv6,omitempty"`
	Status      string           `json:"status"`
	AvgTimeMs   int64            `json:"avgTimeMs"`
}

// DNSResponse represents the DNS test results for the API.
type DNSResponse struct {
	Server           string                 `json:"server"`
	Servers          []string               `json:"servers"` // All configured DNS servers
	TestHostname     string                 `json:"testHostname"`
	Forward          *DNSLookupResult       `json:"forward,omitempty"`
	ForwardIpv6      *DNSLookupResult       `json:"forwardIpv6,omitempty"`
	Reverse          *DNSLookupResult       `json:"reverse,omitempty"`
	ReverseIpv6      *DNSLookupResult       `json:"reverseIpv6,omitempty"`
	PerServerResults []*DNSServerTestResult `json:"perServerResults,omitempty"`
}

// handleDNS performs DNS testing and returns results.
func (s *Server) handleDNS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.dnsTester == nil {
		http.Error(w, "DNS tester not available", http.StatusServiceUnavailable)
		return
	}

	// Perform DNS test
	result := s.dnsTester.Test(r.Context())

	resp := DNSResponse{
		Server:       result.Server,
		Servers:      result.Servers,
		TestHostname: result.TestHostname,
	}

	if result.Forward != nil {
		resp.Forward = &DNSLookupResult{
			Result:   result.Forward.Result,
			Time:     result.Forward.TimeMs,
			TimeMs:   result.Forward.TimeMs,
			Status:   string(result.Forward.Status),
			Error:    result.Forward.Error,
			Resolved: result.Forward.Resolved,
		}
	}

	if result.ForwardIPv6 != nil {
		resp.ForwardIpv6 = &DNSLookupResult{
			Result:   result.ForwardIPv6.Result,
			Time:     result.ForwardIPv6.TimeMs,
			TimeMs:   result.ForwardIPv6.TimeMs,
			Status:   string(result.ForwardIPv6.Status),
			Error:    result.ForwardIPv6.Error,
			Resolved: result.ForwardIPv6.Resolved,
		}
	}

	if result.Reverse != nil {
		resp.Reverse = &DNSLookupResult{
			Result:   result.Reverse.Result,
			Time:     result.Reverse.TimeMs,
			TimeMs:   result.Reverse.TimeMs,
			Status:   string(result.Reverse.Status),
			Error:    result.Reverse.Error,
			Resolved: result.Reverse.Resolved,
		}
	}

	if result.ReverseIPv6 != nil {
		resp.ReverseIpv6 = &DNSLookupResult{
			Result:   result.ReverseIPv6.Result,
			Time:     result.ReverseIPv6.TimeMs,
			TimeMs:   result.ReverseIPv6.TimeMs,
			Status:   string(result.ReverseIPv6.Status),
			Error:    result.ReverseIPv6.Error,
			Resolved: result.ReverseIPv6.Resolved,
		}
	}

	// Map per-server results
	if len(result.PerServerResults) > 0 {
		for _, serverResult := range result.PerServerResults {
			apiResult := &DNSServerTestResult{
				Server:    serverResult.Server,
				Status:    string(serverResult.Status),
				AvgTimeMs: serverResult.AvgTimeMs,
			}
			if serverResult.Forward != nil {
				apiResult.Forward = &DNSLookupResult{
					Result:   serverResult.Forward.Result,
					Time:     serverResult.Forward.TimeMs,
					TimeMs:   serverResult.Forward.TimeMs,
					Status:   string(serverResult.Forward.Status),
					Error:    serverResult.Forward.Error,
					Resolved: serverResult.Forward.Resolved,
				}
			}
			if serverResult.ForwardIPv6 != nil {
				apiResult.ForwardIpv6 = &DNSLookupResult{
					Result:   serverResult.ForwardIPv6.Result,
					Time:     serverResult.ForwardIPv6.TimeMs,
					TimeMs:   serverResult.ForwardIPv6.TimeMs,
					Status:   string(serverResult.ForwardIPv6.Status),
					Error:    serverResult.ForwardIPv6.Error,
					Resolved: serverResult.ForwardIPv6.Resolved,
				}
			}
			resp.PerServerResults = append(resp.PerServerResults, apiResult)
		}
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// RogueDHCPResponse represents rogue DHCP detection status.
type RogueDHCPResponse struct {
	Enabled  bool   `json:"enabled"`
	Running  bool   `json:"running"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
}

// RogueServersResponse contains the list of detected DHCP servers.
type RogueServersResponse struct {
	Servers        []*dhcp.RogueServer `json:"servers"`
	RogueCount     int                 `json:"rogueCount"`
	AuthorizedCount int                `json:"authorizedCount"`
}

// RogueDHCPConfigResponse contains the rogue DHCP detector configuration.
type RogueDHCPConfigResponse struct {
	Enabled         bool     `json:"enabled"`
	KnownServers    []string `json:"knownServers"`
	AlertOnDetection bool     `json:"alertOnDetection"`
	Interface       string   `json:"interface"`
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
				resp.Error = err.Error()
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
			Servers:        servers,
			RogueCount:     len(rogues),
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
		config := s.rogueDetector.GetConfig()
		resp := RogueDHCPConfigResponse{
			Enabled:         s.config.DHCP.RogueDetection.Enabled,
			KnownServers:    config.KnownServers,
			AlertOnDetection: config.AlertOnDetection,
			Interface:       config.Interface,
		}
		sendJSONResponse(w, http.StatusOK, resp)

	case http.MethodPut:
		// Update configuration
		var req struct {
			Enabled         *bool    `json:"enabled,omitempty"`
			KnownServers    []string `json:"knownServers,omitempty"`
			AlertOnDetection *bool     `json:"alertOnDetection,omitempty"`
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
			log.Printf("Failed to save config: %v", err)
		}

		// Return updated config
		config := s.rogueDetector.GetConfig()
		resp := RogueDHCPConfigResponse{
			Enabled:         s.config.DHCP.RogueDetection.Enabled,
			KnownServers:    config.KnownServers,
			AlertOnDetection: config.AlertOnDetection,
			Interface:       config.Interface,
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
	Communities   []string                 `json:"communities"`
	V3Credentials []SNMPv3CredentialResponse `json:"v3Credentials"`
	Timeout       int                      `json:"timeout"` // milliseconds
	Retries       int                      `json:"retries"`
	Port          int                      `json:"port"`
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
	for i, cred := range s.config.SNMP.V3Credentials {
		// Use "*****" placeholder for passwords (never expose actual values)
		authPass := ""
		if cred.AuthPassword != "" {
			authPass = "*****"
		}
		privPass := ""
		if cred.PrivPassword != "" {
			privPass = "*****"
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
	for i, cred := range req.V3Credentials {
		newCred := config.SNMPv3Credential{
			Name:          cred.Name,
			Username:      cred.Username,
			AuthProtocol:  cred.AuthProtocol,
			PrivProtocol:  cred.PrivProtocol,
			ContextName:   cred.ContextName,
			SecurityLevel: cred.SecurityLevel,
		}

		// Handle AuthPassword: If "*****" placeholder, keep existing; otherwise encrypt new value
		if cred.AuthPassword != "" && cred.AuthPassword != "*****" {
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

		// Handle PrivPassword: If "*****" placeholder, keep existing; otherwise encrypt new value
		if cred.PrivPassword != "" && cred.PrivPassword != "*****" {
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
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "SNMP settings updated",
	})
}







// SetMTURequest represents the request to set interface MTU.


// TestsSettingsResponse represents the custom tests configuration.
type TestsSettingsResponse struct {
	DNSHostname    string                    `json:"dnsHostname"`
	DNSServers     []DNSServerResponse       `json:"dnsServers"`
	PingTargets    []PingTargetResponse      `json:"pingTargets"`
	TCPPorts       []TCPPortResponse         `json:"tcpPorts"`
	UDPPorts       []UDPPortResponse         `json:"udpPorts"`
	HTTPEndpoints  []HTTPEndpointResponse    `json:"httpEndpoints"`
	Speedtest      SpeedtestSettingsResponse `json:"speedtest"`
	Iperf          IperfSettingsResponse     `json:"iperf"`
	RunPerformance bool                      `json:"runPerformance"`
	RunSpeedtest   bool                      `json:"runSpeedtest"`
	RunIperf       bool                      `json:"runIperf"`
	RunDiscovery   bool                      `json:"runDiscovery"`
}

// DNSServerResponse represents a DNS server for testing.
type DNSServerResponse struct {
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

type PingTargetResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
}

type TCPPortResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

type UDPPortResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

type HTTPEndpointResponse struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expectedStatus"`
	Enabled        bool   `json:"enabled"`
}

type SpeedtestSettingsResponse struct {
	ServerID      string `json:"serverId"`
	AutoRunOnLink bool   `json:"autoRunOnLink"`
}

type IperfSettingsResponse struct {
	AutoRunOnLink bool `json:"autoRunOnLink"`
}

// handleTestsSettings handles GET/PUT for custom tests settings.
func (s *Server) handleTestsSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getTestsSettings(w, r)
	case http.MethodPut:
		s.updateTestsSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getTestsSettings(w http.ResponseWriter, _ *http.Request) {
	resp := TestsSettingsResponse{
		DNSHostname:    s.config.DNS.TestHostname,
		DNSServers:     make([]DNSServerResponse, 0, len(s.config.DNS.Servers)),
		PingTargets:    make([]PingTargetResponse, 0, len(s.config.Tests.PingTargets)),
		TCPPorts:       make([]TCPPortResponse, 0, len(s.config.Tests.TCPPorts)),
		UDPPorts:       make([]UDPPortResponse, 0, len(s.config.Tests.UDPPorts)),
		HTTPEndpoints:  make([]HTTPEndpointResponse, 0, len(s.config.Tests.HTTPEndpoints)),
		RunPerformance: s.config.Tests.RunPerformance,
		RunSpeedtest:   s.config.Tests.RunSpeedtest,
		RunIperf:       s.config.Tests.RunIperf,
		RunDiscovery:   s.config.Tests.RunDiscovery,
		Speedtest: SpeedtestSettingsResponse{
			ServerID:      s.config.Speedtest.ServerID,
			AutoRunOnLink: s.config.Speedtest.AutoRunOnLink,
		},
		Iperf: IperfSettingsResponse{
			AutoRunOnLink: s.config.Iperf.AutoRunOnLink,
		},
	}

	// DNS servers
	for _, d := range s.config.DNS.Servers {
		resp.DNSServers = append(resp.DNSServers, DNSServerResponse{
			Address: d.Address,
			Enabled: d.Enabled,
		})
	}

	for _, p := range s.config.Tests.PingTargets {
		resp.PingTargets = append(resp.PingTargets, PingTargetResponse{
			Name:    p.Name,
			Host:    p.Host,
			Enabled: p.Enabled,
		})
	}

	for _, t := range s.config.Tests.TCPPorts {
		resp.TCPPorts = append(resp.TCPPorts, TCPPortResponse{
			Name:    t.Name,
			Host:    t.Host,
			Port:    t.Port,
			Enabled: t.Enabled,
		})
	}

	for _, u := range s.config.Tests.UDPPorts {
		resp.UDPPorts = append(resp.UDPPorts, UDPPortResponse{
			Name:    u.Name,
			Host:    u.Host,
			Port:    u.Port,
			Enabled: u.Enabled,
		})
	}

	for _, h := range s.config.Tests.HTTPEndpoints {
		resp.HTTPEndpoints = append(resp.HTTPEndpoints, HTTPEndpointResponse{
			Name:           h.Name,
			URL:            h.URL,
			ExpectedStatus: h.ExpectedStatus,
			Enabled:        h.Enabled,
		})
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

func (s *Server) updateTestsSettings(w http.ResponseWriter, r *http.Request) {
	var req TestsSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Lock config for write access
	s.config.Lock()
	defer s.config.Unlock()

	// Update DNS hostname
	if req.DNSHostname != "" {
		s.config.DNS.TestHostname = req.DNSHostname
		// Update the DNS tester with the new hostname
		if s.dnsTester != nil {
			s.dnsTester.SetTestHostname(req.DNSHostname)
		}
	}

	// Update DNS servers
	s.config.DNS.Servers = make([]config.DNSServer, 0, len(req.DNSServers))
	for _, d := range req.DNSServers {
		s.config.DNS.Servers = append(s.config.DNS.Servers, config.DNSServer{
			Address: d.Address,
			Enabled: d.Enabled,
		})
	}
	// Update the DNS tester with the configured servers
	if s.dnsTester != nil {
		configuredServers := make([]dns.ConfiguredServer, 0, len(s.config.DNS.Servers))
		for _, d := range s.config.DNS.Servers {
			configuredServers = append(configuredServers, dns.ConfiguredServer{
				Address: d.Address,
				Enabled: d.Enabled,
			})
		}
		s.dnsTester.SetConfiguredServers(configuredServers)
	}

	// Update ping targets
	s.config.Tests.PingTargets = make([]config.PingTarget, 0, len(req.PingTargets))
	for _, p := range req.PingTargets {
		s.config.Tests.PingTargets = append(s.config.Tests.PingTargets, config.PingTarget{
			Name:    p.Name,
			Host:    p.Host,
			Enabled: p.Enabled,
		})
	}

	// Update TCP ports
	s.config.Tests.TCPPorts = make([]config.TCPPortTest, 0, len(req.TCPPorts))
	for _, t := range req.TCPPorts {
		s.config.Tests.TCPPorts = append(s.config.Tests.TCPPorts, config.TCPPortTest{
			Name:    t.Name,
			Host:    t.Host,
			Port:    t.Port,
			Enabled: t.Enabled,
		})
	}

	// Update UDP ports
	s.config.Tests.UDPPorts = make([]config.UDPPortTest, 0, len(req.UDPPorts))
	for _, u := range req.UDPPorts {
		s.config.Tests.UDPPorts = append(s.config.Tests.UDPPorts, config.UDPPortTest{
			Name:    u.Name,
			Host:    u.Host,
			Port:    u.Port,
			Enabled: u.Enabled,
		})
	}

	// Update HTTP endpoints
	// Store URL as-is to preserve user intent - scheme-less URLs enable HTTPS->HTTP fallback at test time
	s.config.Tests.HTTPEndpoints = make([]config.HTTPEndpoint, 0, len(req.HTTPEndpoints))
	for _, h := range req.HTTPEndpoints {
		s.config.Tests.HTTPEndpoints = append(s.config.Tests.HTTPEndpoints, config.HTTPEndpoint{
			Name:           h.Name,
			URL:            h.URL,
			ExpectedStatus: h.ExpectedStatus,
			Enabled:        h.Enabled,
		})
	}

	// Update performance toggle
	s.config.Tests.RunPerformance = req.RunPerformance
	s.config.Tests.RunSpeedtest = req.RunSpeedtest
	s.config.Tests.RunIperf = req.RunIperf
	s.config.Tests.RunDiscovery = req.RunDiscovery

	// Update speedtest settings
	s.config.Speedtest.ServerID = req.Speedtest.ServerID
	s.config.Speedtest.AutoRunOnLink = req.Speedtest.AutoRunOnLink
	if s.speedtestTester != nil {
		s.speedtestTester.SetServerID(req.Speedtest.ServerID)
	}

	// Update iperf settings
	s.config.Iperf.AutoRunOnLink = req.Iperf.AutoRunOnLink

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Tests settings updated",
	})
}

// CustomTestResult represents the result of a single custom test.
type CustomTestResult struct {
	Name        string  `json:"name"`
	Host        string  `json:"host"`
	Port        int     `json:"port,omitempty"`
	URL         string  `json:"url,omitempty"`
	Success     bool    `json:"success"`
	Latency     float64 `json:"latency"` // ms
	DNSLatency  float64 `json:"dnsLatency,omitempty"`
	TCPConnect  float64 `json:"tcpConnect,omitempty"`
	TLSLatency  float64 `json:"tlsLatency,omitempty"`
	TTFBLatency float64 `json:"ttfbLatency,omitempty"` // Time to first byte (server processing + wait)
	Error       string  `json:"error,omitempty"`
	Status      int     `json:"status,omitempty"`     // HTTP status code
	TestStatus  string  `json:"testStatus,omitempty"` // success, warning, error
	// Per-phase status fields for HTTP timing breakdown
	DNSStatus  string `json:"dnsStatus,omitempty"`  // success, warning, error
	TCPStatus  string `json:"tcpStatus,omitempty"`  // success, warning, error
	TLSStatus  string `json:"tlsStatus,omitempty"`  // success, warning, error
	TTFBStatus string `json:"ttfbStatus,omitempty"` // success, warning, error
	// Extended ping fields
	PacketLoss float64 `json:"packetLoss,omitempty"` // Percentage
	Jitter     float64 `json:"jitter,omitempty"`     // ms
	MinLatency float64 `json:"minLatency,omitempty"` // ms
	MaxLatency float64 `json:"maxLatency,omitempty"` // ms
	// Certificate fields
	CertDaysLeft   int    `json:"certDaysLeft,omitempty"`   // Days until cert expires
	CertStatus     string `json:"certStatus,omitempty"`     // success, warning, error
	CertExpiry     string `json:"certExpiry,omitempty"`     // Expiry date string
	CertCommonName string `json:"certCommonName,omitempty"` // Certificate CN
	TLSVersion     string `json:"tlsVersion,omitempty"`     // TLS 1.2, TLS 1.3, etc.
	CertIssuer     string `json:"certIssuer,omitempty"`     // Certificate issuer
}

// CustomTestsResult represents results from all custom tests.
type CustomTestsResult struct {
	PingResults []CustomTestResult `json:"pingResults"`
	TCPResults  []CustomTestResult `json:"tcpResults"`
	UDPResults  []CustomTestResult `json:"udpResults"`
	HTTPResults []CustomTestResult `json:"httpResults"`
	HasTests    bool               `json:"hasTests"`
}

// handleCustomTests runs all configured custom tests and returns results.
func (s *Server) handleCustomTests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := CustomTestsResult{
		PingResults: make([]CustomTestResult, 0),
		TCPResults:  make([]CustomTestResult, 0),
		UDPResults:  make([]CustomTestResult, 0),
		HTTPResults: make([]CustomTestResult, 0),
		HasTests:    false,
	}

	// Check if there are any tests configured
	if len(s.config.Tests.PingTargets) > 0 || len(s.config.Tests.TCPPorts) > 0 ||
		len(s.config.Tests.UDPPorts) > 0 || len(s.config.Tests.HTTPEndpoints) > 0 {
		result.HasTests = true
	}

	// Get thresholds
	pingThreshold := s.config.Thresholds.CustomTests.Ping
	tcpThreshold := s.config.Thresholds.CustomTests.TCP
	udpThreshold := s.config.Thresholds.CustomTests.UDP
	httpThreshold := s.config.Thresholds.CustomTests.HTTP
	httpTimingThresholds := s.config.Thresholds.CustomTests.HTTPTimings
	certThreshold := s.config.Thresholds.CustomTests.CertExpiry

	// Run extended ping tests (with packet loss and jitter)
	for _, target := range s.config.Tests.PingTargets {
		if !target.Enabled {
			continue
		}
		name := target.Name
		if name == "" {
			name = target.Host
		}

		testResult := CustomTestResult{
			Name: name,
			Host: target.Host,
		}

		// Run extended ping (5 pings for stats)
		pingStats, err := runExtendedPing(target.Host, 5)
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = pingStats.PacketLoss < 100
			testResult.Latency = pingStats.AvgLatency
			testResult.MinLatency = pingStats.MinLatency
			testResult.MaxLatency = pingStats.MaxLatency
			testResult.PacketLoss = pingStats.PacketLoss
			testResult.Jitter = pingStats.Jitter

			// Determine status based on latency or packet loss
			switch {
			case pingStats.PacketLoss > 50:
				testResult.TestStatus = "error"
			case pingStats.PacketLoss > 10:
				testResult.TestStatus = "warning"
			default:
				testResult.TestStatus = getTestStatus(pingStats.AvgLatency, pingThreshold.Warning.Milliseconds(), pingThreshold.Critical.Milliseconds())
			}
		}
		result.PingResults = append(result.PingResults, testResult)
	}

	// Run TCP port tests
	for _, target := range s.config.Tests.TCPPorts {
		if !target.Enabled {
			continue
		}
		name := target.Name
		if name == "" {
			name = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
		}

		testResult := CustomTestResult{
			Name: name,
			Host: target.Host,
			Port: target.Port,
		}

		latency, err := runTCPTest(r.Context(), target.Host, target.Port)
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = true
			testResult.Latency = latency
			testResult.TestStatus = getTestStatus(latency, tcpThreshold.Warning.Milliseconds(), tcpThreshold.Critical.Milliseconds())
		}
		result.TCPResults = append(result.TCPResults, testResult)
	}

	// Run UDP port tests
	for _, target := range s.config.Tests.UDPPorts {
		if !target.Enabled {
			continue
		}
		name := target.Name
		if name == "" {
			name = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
		}

		testResult := CustomTestResult{
			Name: name,
			Host: target.Host,
			Port: target.Port,
		}

		latency, err := runUDPTest(target.Host, target.Port)
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = true
			testResult.Latency = latency
			testResult.TestStatus = getTestStatus(latency, udpThreshold.Warning.Milliseconds(), udpThreshold.Critical.Milliseconds())
		}
		result.UDPResults = append(result.UDPResults, testResult)
	}

	// Run HTTP endpoint tests with certificate expiry checking
	for _, endpoint := range s.config.Tests.HTTPEndpoints {
		if !endpoint.Enabled {
			continue
		}

		// Validate URL to prevent SSRF attacks
		if err := validation.ValidateURL(endpoint.URL); err != nil {
			log.Printf("Skipping invalid HTTP endpoint URL %q: %v", endpoint.URL, err)
			continue
		}

		// Determine URL and whether to try fallback
		url := endpoint.URL
		tryHTTPFallback := false

		if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			// No scheme provided - try HTTPS first, can fallback to HTTP
			url = "https://" + url
			tryHTTPFallback = true
		}

		name := endpoint.Name
		if name == "" {
			name = endpoint.URL // Show original URL in name
		}

		testResult := CustomTestResult{
			Name: name,
			URL:  url,
		}

		statusCode, timings, err := runHTTPTest(r.Context(), url, endpoint.ExpectedStatus)

		// If HTTPS failed and we can try HTTP fallback
		if err != nil && tryHTTPFallback {
			httpURL := "http://" + endpoint.URL
			httpStatus, httpTimings, httpErr := runHTTPTest(r.Context(), httpURL, endpoint.ExpectedStatus)
			if httpErr == nil || httpStatus > 0 {
				// HTTP worked (or at least connected) - use those results
				url = httpURL
				testResult.URL = httpURL
				statusCode = httpStatus
				timings = httpTimings
				err = httpErr
			}
		}

		testResult.Status = statusCode
		testResult.Latency = timings.Total
		testResult.DNSLatency = timings.DNS
		testResult.TCPConnect = timings.Connect
		testResult.TLSLatency = timings.TLS
		testResult.TTFBLatency = timings.TTFB
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = true
			// Evaluate each phase against its threshold
			testResult.DNSStatus = getTestStatus(timings.DNS, httpTimingThresholds.DNS.Warning.Milliseconds(), httpTimingThresholds.DNS.Critical.Milliseconds())
			testResult.TCPStatus = getTestStatus(timings.Connect, httpTimingThresholds.TCP.Warning.Milliseconds(), httpTimingThresholds.TCP.Critical.Milliseconds())
			testResult.TLSStatus = getTestStatus(timings.TLS, httpTimingThresholds.TLS.Warning.Milliseconds(), httpTimingThresholds.TLS.Critical.Milliseconds())
			testResult.TTFBStatus = getTestStatus(timings.TTFB, httpTimingThresholds.TTFB.Warning.Milliseconds(), httpTimingThresholds.TTFB.Critical.Milliseconds())

			// Overall test status: error if any phase is error, warning if any warning, else use total time
			switch {
			case testResult.DNSStatus == "error" || testResult.TCPStatus == "error" ||
				testResult.TLSStatus == "error" || testResult.TTFBStatus == "error":
				testResult.TestStatus = "error"
			case testResult.DNSStatus == "warning" || testResult.TCPStatus == "warning" ||
				testResult.TLSStatus == "warning" || testResult.TTFBStatus == "warning":
				testResult.TestStatus = "warning"
			default:
				testResult.TestStatus = getTestStatus(timings.Total, httpThreshold.Warning.Milliseconds(), httpThreshold.Critical.Milliseconds())
			}
		}

		// Check certificate expiry for HTTPS URLs only
		if strings.HasPrefix(url, "https://") && testResult.Success {
			certInfo := checkCertExpiry(url, certThreshold.Warning, certThreshold.Critical)
			testResult.CertDaysLeft = certInfo.DaysLeft
			testResult.CertStatus = certInfo.Status
			testResult.CertExpiry = certInfo.ExpiryDate
			testResult.CertCommonName = certInfo.CommonName
			testResult.TLSVersion = certInfo.TLSVersion
			testResult.CertIssuer = certInfo.Issuer

			// Upgrade test status if cert is in bad shape
			if certInfo.Status == "error" && testResult.TestStatus != "error" {
				testResult.TestStatus = "error"
			} else if certInfo.Status == "warning" && testResult.TestStatus == "success" {
				testResult.TestStatus = "warning"
			}
		}

		result.HTTPResults = append(result.HTTPResults, testResult)
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// runTCPTest runs a TCP port test and returns latency in ms (fixes #534).
func runTCPTest(ctx context.Context, host string, port int) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	latency := time.Since(start).Seconds() * 1000
	conn.Close()
	return latency, nil
}

type httpTimings struct {
	DNS     float64
	Connect float64
	TLS     float64
	TTFB    float64 // Time to first byte (from request sent to first response byte)
	Total   float64
}

// runHTTPTest runs an HTTP test and returns status code and timings in ms (fixes #534).
// Uses SafeTransport to prevent DNS rebinding SSRF attacks.
func runHTTPTest(ctx context.Context, url string, expectedStatus int) (status int, timing httpTimings, err error) {
	// Use SafeTransport to block connections to private IPs (prevents DNS rebinding)
	transport := validation.SafeTransport()
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, timing, err
	}

	var dnsStart, connStart, tlsStart, wroteRequest time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				timing.DNS += time.Since(dnsStart).Seconds() * 1000
			}
		},
		ConnectStart: func(_, _ string) {
			connStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			if !connStart.IsZero() {
				timing.Connect += time.Since(connStart).Seconds() * 1000
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStart.IsZero() {
				timing.TLS += time.Since(tlsStart).Seconds() * 1000
			}
		},
		WroteRequest: func(httptrace.WroteRequestInfo) {
			wroteRequest = time.Now()
		},
		GotFirstResponseByte: func() {
			if !wroteRequest.IsZero() {
				timing.TTFB = time.Since(wroteRequest).Seconds() * 1000
			}
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	start := time.Now()
	resp, err := client.Do(req)
	timing.Total = time.Since(start).Seconds() * 1000

	if err != nil {
		return 0, timing, err
	}
	defer resp.Body.Close()

	status = resp.StatusCode
	if expectedStatus > 0 && status != expectedStatus {
		return status, timing, fmt.Errorf("expected %d, got %d", expectedStatus, status)
	}

	return status, timing, nil
}

// getTestStatus returns status based on latency and thresholds.
func getTestStatus(latencyMs float64, warningMs, criticalMs int64) string {
	if latencyMs < float64(warningMs) {
		return "success"
	}
	if latencyMs < float64(criticalMs) {
		return "warning"
	}
	return "error"
}

// PingStats holds extended ping statistics.
type PingStats struct {
	AvgLatency float64 // ms
	MinLatency float64 // ms
	MaxLatency float64 // ms
	PacketLoss float64 // percentage
	Jitter     float64 // ms (standard deviation)
}

// runExtendedPing runs multiple pings and returns statistics.
func runExtendedPing(host string, count int) (*PingStats, error) {
	var latencies []float64
	sent := 0
	received := 0

	for i := 0; i < count; i++ {
		sent++
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		start := time.Now()
		// Try TCP 80/443 as ping alternative (actual ICMP requires root)
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", host+":80")
		if err != nil {
			conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", host+":443")
		}
		cancel()

		if err == nil {
			latency := time.Since(start).Seconds() * 1000
			latencies = append(latencies, latency)
			received++
			conn.Close()
		}

		// Small delay between pings
		if i < count-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if len(latencies) == 0 {
		return &PingStats{PacketLoss: 100}, fmt.Errorf("host unreachable")
	}

	// Calculate statistics
	stats := &PingStats{
		PacketLoss: float64(sent-received) / float64(sent) * 100,
	}

	// Min, max, avg
	stats.MinLatency = latencies[0]
	stats.MaxLatency = latencies[0]
	var sum float64
	for _, lat := range latencies {
		sum += lat
		if lat < stats.MinLatency {
			stats.MinLatency = lat
		}
		if lat > stats.MaxLatency {
			stats.MaxLatency = lat
		}
	}
	stats.AvgLatency = sum / float64(len(latencies))

	// Jitter (standard deviation)
	if len(latencies) > 1 {
		var variance float64
		for _, lat := range latencies {
			diff := lat - stats.AvgLatency
			variance += diff * diff
		}
		stats.Jitter = math.Sqrt(variance / float64(len(latencies)))
	}

	return stats, nil
}

// runUDPTest runs a UDP port test and returns latency in ms.
// Note: UDP is connectionless, so we send a packet and wait for ICMP unreachable
// or application response. For DNS (53), NTP (123), etc. we can get actual responses.
func runUDPTest(host string, port int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// For DNS port, try a simple DNS query
	if port == 53 {
		return testDNSPort(ctx, host)
	}

	// For other UDP ports, we try to connect (which on UDP just sets up local state)
	// and send a small probe packet
	start := time.Now()

	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	// Set deadline for response
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return 0, err
	}

	// Send a small probe packet
	_, err = conn.Write([]byte{0x00})
	if err != nil {
		return 0, err
	}

	// Try to read response (may timeout for non-responding services)
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)

	latency := time.Since(start).Seconds() * 1000

	// For UDP, no error on Write means the port is likely open
	// (no ICMP unreachable received)
	if err != nil {
		// Check if it's a timeout (which for UDP often means the port is open but not responding)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Port is likely open but service didn't respond - still count as success
			return latency, nil
		}
		// Connection refused or other error means port is closed
		return 0, fmt.Errorf("port closed or filtered")
	}

	return latency, nil
}

// testDNSPort tests DNS port by sending a simple query.
func testDNSPort(ctx context.Context, host string) (float64, error) {
	// Use Go's resolver to test DNS
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", host+":53")
		},
	}

	start := time.Now()
	_, err := resolver.LookupHost(ctx, "google.com")
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		return 0, err
	}
	return latency, nil
}

// CertInfo holds certificate expiry information.
type CertInfo struct {
	DaysLeft   int
	Status     string // success, warning, error
	ExpiryDate string
	CommonName string
	TLSVersion string // TLS 1.0, TLS 1.1, TLS 1.2, TLS 1.3
	Issuer     string // Certificate issuer (for context)
}

// checkCertExpiry checks the TLS certificate expiry for a URL.
func checkCertExpiry(url string, warningDays, criticalDays int) CertInfo {
	info := CertInfo{Status: "success"}

	// Extract host from URL
	host := strings.TrimPrefix(url, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx == -1 {
		host += ":443"
	}

	// Connect with TLS
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp",
		host,
		// #nosec G402 - certificate verification intentionally skipped to inspect expiry
		&tls.Config{InsecureSkipVerify: true}, // We want to check expiry even for self-signed
	)
	if err != nil {
		info.Status = "error"
		return info
	}
	defer conn.Close()

	// Get connection state for TLS info
	connState := conn.ConnectionState()

	// Get TLS version
	info.TLSVersion = getTLSVersionString(connState.Version)

	// Get certificate chain
	certs := connState.PeerCertificates
	if len(certs) == 0 {
		info.Status = "error"
		return info
	}

	// Check the leaf certificate
	cert := certs[0]
	info.CommonName = cert.Subject.CommonName
	info.ExpiryDate = cert.NotAfter.Format("2006-01-02")

	// Get issuer (org or CN)
	if len(cert.Issuer.Organization) > 0 {
		info.Issuer = cert.Issuer.Organization[0]
	} else if cert.Issuer.CommonName != "" {
		info.Issuer = cert.Issuer.CommonName
	}

	// Calculate days until expiry
	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
	info.DaysLeft = daysLeft

	// Determine status
	switch {
	case daysLeft <= 0:
		info.Status = "error" // Expired
	case daysLeft <= criticalDays:
		info.Status = "error" // Critical
	case daysLeft <= warningDays:
		info.Status = "warning" // Warning
	default:
		info.Status = "success" // OK
	}

	return info
}

// getTLSVersionString converts TLS version to human-readable string.
func getTLSVersionString(tlsVersion uint16) string {
	switch tlsVersion {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

// SpeedtestResponse represents the speedtest results for the API.
type SpeedtestResponse struct {
	Download     float64 `json:"download"` // Mbps
	Upload       float64 `json:"upload"`   // Mbps
	Latency      float64 `json:"latency"`  // ms
	Server       string  `json:"server"`   // Server name
	Location     string  `json:"location"` // Server location
	Host         string  `json:"host"`     // Server host
	Distance     float64 `json:"distance"` // km
	Timestamp    string  `json:"timestamp"`
	TestDuration float64 `json:"testDuration"` // seconds
}

// SpeedtestStatusResponse represents the current speedtest status.
type SpeedtestStatusResponse struct {
	Running  bool               `json:"running"`
	Phase    string             `json:"phase"`
	Progress float64            `json:"progress"`
	Last     *SpeedtestResponse `json:"last,omitempty"`
}

// handleSpeedtest starts a speedtest in the background and returns immediately.
// Use /api/speedtest/status to poll for results.
func (s *Server) handleSpeedtest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed - use POST to start a speedtest", http.StatusMethodNotAllowed)
		return
	}

	if s.speedtestTester == nil {
		http.Error(w, "Speedtest not available", http.StatusServiceUnavailable)
		return
	}

	// Check if already running
	status := s.speedtestTester.GetStatus()
	if status.Running {
		http.Error(w, "Speedtest already in progress", http.StatusConflict)
		return
	}

	// Run the test in the background (takes 30-60 seconds)
	go func() {
		ctx := context.Background()
		_, err := s.speedtestTester.RunTest(ctx)
		if err != nil {
			log.Printf("Speedtest failed: %v", err)
		}
	}()

	// Return immediately with "started" status
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "started",
		"message": "Speedtest started. Poll /api/speedtest/status for results.",
	})
}

// handleSpeedtestStatus returns the current speedtest status.
func (s *Server) handleSpeedtestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.speedtestTester == nil {
		http.Error(w, "Speedtest not available", http.StatusServiceUnavailable)
		return
	}

	status := s.speedtestTester.GetStatus()
	resp := SpeedtestStatusResponse{
		Running:  status.Running,
		Phase:    status.Phase,
		Progress: status.Progress,
	}

	// Include last result if available
	if lastResult := s.speedtestTester.GetLastResult(); lastResult != nil {
		resp.Last = &SpeedtestResponse{
			Download:     lastResult.Download,
			Upload:       lastResult.Upload,
			Latency:      lastResult.Latency,
			Server:       lastResult.Server,
			Location:     lastResult.Location,
			Host:         lastResult.Host,
			Distance:     lastResult.Distance,
			Timestamp:    lastResult.Timestamp.Format(time.RFC3339),
			TestDuration: lastResult.TestDuration,
		}
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// iperf3 handlers

// IperfInfoResponse contains iperf3 installation info
type IperfInfoResponse struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

// handleIperfInfo returns iperf3 installation status and version
func (s *Server) handleIperfInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := IperfInfoResponse{}
	iperfVersion, err := iperf.GetVersion()
	if err != nil {
		resp.Installed = false
		resp.Error = err.Error()
	} else {
		resp.Installed = true
		resp.Version = iperfVersion
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// IperfClientRequest is the request body for running an iperf3 client test
type IperfClientRequest struct {
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`  // "tcp" or "udp"
	Reverse   bool   `json:"reverse"`   // true = download, false = upload (legacy)
	Direction string `json:"direction"` // "upload", "download", "bidirectional"
	Duration  int    `json:"duration"`  // seconds
	Parallel  int    `json:"parallel"`  // number of streams
}

// IperfResultResponse is the response for an iperf3 test result
type IperfResultResponse struct {
	Bandwidth         float64 `json:"bandwidth"`   // Mbps
	Transfer          float64 `json:"transfer"`    // MB
	Retransmits       int     `json:"retransmits"` // TCP only
	Jitter            float64 `json:"jitter"`      // UDP only, ms
	LostPackets       int     `json:"lostPackets"` // UDP only
	LostPercent       float64 `json:"lostPercent"` // UDP only
	Protocol          string  `json:"protocol"`
	Direction         string  `json:"direction"`
	Duration          float64 `json:"duration"`
	Server            string  `json:"server"`
	Port              int     `json:"port"`
	Timestamp         string  `json:"timestamp"`
	DownloadBandwidth float64 `json:"downloadBandwidth,omitempty"`
	UploadBandwidth   float64 `json:"uploadBandwidth,omitempty"`
	DownloadTransfer  float64 `json:"downloadTransfer,omitempty"`
	UploadTransfer    float64 `json:"uploadTransfer,omitempty"`
}

// handleIperfClient runs an iperf3 client test
func (s *Server) handleIperfClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IperfClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Server == "" {
		http.Error(w, "Server address required", http.StatusBadRequest)
		return
	}

	req.Protocol = strings.ToLower(req.Protocol)
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	if req.Protocol != "tcp" && req.Protocol != "udp" {
		http.Error(w, "protocol must be tcp or udp", http.StatusBadRequest)
		return
	}

	req.Direction = strings.ToLower(req.Direction)
	if req.Direction == "" {
		if req.Reverse {
			req.Direction = "download"
		} else {
			req.Direction = "upload"
		}
	}
	if req.Direction != "upload" && req.Direction != "download" && req.Direction != "bidirectional" {
		http.Error(w, "direction must be upload, download, or bidirectional", http.StatusBadRequest)
		return
	}

	// Validate numeric parameters (fixes #522)
	if req.Port != 0 {
		if err := validation.ValidatePort(req.Port); err != nil {
			http.Error(w, fmt.Sprintf("Invalid port: %v", err), http.StatusBadRequest)
			return
		}
	}
	if err := validation.ValidatePositiveInt(req.Duration, "duration"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidatePositiveInt(req.Parallel, "parallel streams"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	iperfConfig := iperf.ClientConfig{
		Server:    req.Server,
		Port:      req.Port,
		Protocol:  req.Protocol,
		Reverse:   req.Reverse,
		Direction: req.Direction,
		Duration:  req.Duration,
		Parallel:  req.Parallel,
	}

	// Run test in background and return immediately
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Duration+30)*time.Second)
		defer cancel()
		if _, err := s.iperfManager.RunClient(ctx, &iperfConfig); err != nil {
			log.Printf("iperf client failed: %v", err)
		}
	}()

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"message": "iperf3 test started. Poll /api/iperf/client/status for results.",
	})
}

// IperfClientStatusResponse is the status of an iperf3 client test
type IperfClientStatusResponse struct {
	Running  bool                 `json:"running"`
	Phase    string               `json:"phase"`
	Progress float64              `json:"progress"`
	Last     *IperfResultResponse `json:"last,omitempty"`
}

// handleIperfClientStatus returns the status of the iperf3 client test
func (s *Server) handleIperfClientStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.iperfManager.GetClientStatus()
	resp := IperfClientStatusResponse{
		Running:  status.Running,
		Phase:    status.Phase,
		Progress: status.Progress,
	}

	if lastResult := s.iperfManager.GetLastResult(); lastResult != nil {
		resp.Last = &IperfResultResponse{
			Bandwidth:         lastResult.Bandwidth,
			Transfer:          lastResult.Transfer,
			Retransmits:       lastResult.Retransmits,
			Jitter:            lastResult.Jitter,
			LostPackets:       lastResult.LostPackets,
			LostPercent:       lastResult.LostPercent,
			Protocol:          lastResult.Protocol,
			Direction:         lastResult.Direction,
			Duration:          lastResult.Duration,
			Server:            lastResult.Server,
			Port:              lastResult.Port,
			Timestamp:         lastResult.Timestamp.Format(time.RFC3339),
			DownloadBandwidth: lastResult.DownloadBandwidth,
			UploadBandwidth:   lastResult.UploadBandwidth,
			DownloadTransfer:  lastResult.DownloadTransfer,
			UploadTransfer:    lastResult.UploadTransfer,
		}
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// IperfServerRequest is the request body for starting/stopping the iperf3 server
type IperfServerRequest struct {
	Action string `json:"action"` // "start" or "stop"
	Port   int    `json:"port"`
}

// handleIperfServer starts or stops the iperf3 server
func (s *Server) handleIperfServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IperfServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "start":
		port := req.Port
		if port == 0 {
			port = 5201
		}
		if err := s.iperfManager.StartServer(port); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("iperf3 server started on port %d", port),
			"port":    port,
		})
	case "stop":
		if err := s.iperfManager.StopServer(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, http.StatusOK, map[string]string{
			"message": "iperf3 server stopped",
		})
	default:
		http.Error(w, "Invalid action (use 'start' or 'stop')", http.StatusBadRequest)
	}
}

// handleIperfServerStatus returns the iperf3 server status
func (s *Server) handleIperfServerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.iperfManager.GetServerStatus()
	sendJSONResponse(w, http.StatusOK, status)
}

// IperfSuggestion represents a discovered host that responds on the iperf port.
type IperfSuggestion struct {
	Host      string  `json:"host"`
	Hostname  string  `json:"hostname,omitempty"`
	Source    string  `json:"source,omitempty"`
	LatencyMs float64 `json:"latencyMs,omitempty"`
}

// handleIperfSuggestions returns discovered devices that respond on the iperf port.
func (s *Server) handleIperfSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	port := 5201
	if p := r.URL.Query().Get("port"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			port = parsed
		}
	}

	devices := s.deviceDiscovery.GetDevices()
	suggestions := make([]IperfSuggestion, 0, len(devices))

	for _, d := range devices {
		if d.IP == "" {
			continue
		}

		addr := net.JoinHostPort(d.IP, strconv.Itoa(port))
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, 400*time.Millisecond)
		if err != nil {
			continue
		}
		latency := time.Since(start).Seconds() * 1000
		_ = conn.Close()

		var source string
		if len(d.DiscoveryMethod) > 0 {
			methods := make([]string, 0, len(d.DiscoveryMethod))
			for _, m := range d.DiscoveryMethod {
				methods = append(methods, string(m))
			}
			source = strings.Join(methods, ",")
		}

		suggestions = append(suggestions, IperfSuggestion{
			Host:      d.IP,
			Hostname:  d.Hostname,
			Source:    source,
			LatencyMs: latency,
		})

		if len(suggestions) >= 10 {
			break
		}
	}

	sendJSONResponse(w, http.StatusOK, suggestions)
}

// handleDevices returns all discovered network devices.
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

// SetupStatusResponse represents the setup status response.
// WiFi Survey API Handlers

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

	newSurvey, err := s.surveyManager.CreateSurvey(req.Name, req.Description, req.Interface, survey.SurveyType(req.SurveyType))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create survey: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, newSurvey)
}

func (s *Server) listSurveys(w http.ResponseWriter, r *http.Request) {
	surveys := s.surveyManager.ListSurveys()
	sendJSONResponse(w, http.StatusOK, surveys)
}

func (s *Server) getSurvey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Survey ID required", http.StatusBadRequest)
		return
	}

	survey, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, http.StatusOK, survey)
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
