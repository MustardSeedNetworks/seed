// Package api provides the HTTP/WebSocket server.
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

	"github.com/krisarmstrong/luminetiq/internal/dns"
	"github.com/krisarmstrong/luminetiq/internal/iperf"

	"github.com/krisarmstrong/luminetiq/internal/config"
	"github.com/krisarmstrong/luminetiq/internal/validation"
)

// ============================================================================
// Request/Response Types and Handlers (fixes #544 - split from handlers.go)
// ============================================================================

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
