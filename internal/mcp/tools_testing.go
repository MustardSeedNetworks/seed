package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// registerTestingTools registers all testing-related MCP tools.
func (s *Server) registerTestingTools(isAllowed func(string) bool) {
	// dns_test - Test DNS resolution
	s.addTool("dns_test", isAllowed,
		mcp.NewTool("dns_test",
			mcp.WithDescription("Test DNS resolution by looking up a hostname. Returns resolution time, resolved addresses, and status for configured DNS servers."),
			mcp.WithString("hostname",
				mcp.Description("Hostname to resolve (default: google.com)"),
			),
		),
		s.handleDNSTest,
	)

	// gateway_ping - Ping the default gateway
	s.addTool("gateway_ping", isAllowed,
		mcp.NewTool("gateway_ping",
			mcp.WithDescription("Ping the default gateway to check network connectivity. Returns latency statistics and packet loss information."),
			mcp.WithNumber("count",
				mcp.Description("Number of pings to send (default: 5, max: 20)"),
			),
		),
		s.handleGatewayPing,
	)

	// speedtest - Run internet speed test
	s.addTool("speedtest", isAllowed,
		mcp.NewTool("speedtest",
			mcp.WithDescription("Run an internet speed test to measure download speed, upload speed, and latency. Uses speedtest.net infrastructure. Takes 10-30 seconds to complete."),
			mcp.WithString("server_id",
				mcp.Description("Specific speedtest server ID (optional, auto-selects nearest if not specified)"),
			),
		),
		s.handleSpeedtest,
	)

	// iperf_test - Run iPerf3 throughput test
	s.addTool("iperf_test", isAllowed,
		mcp.NewTool("iperf_test",
			mcp.WithDescription("Run an iPerf3 throughput test against a server. Measures actual network bandwidth between this device and an iPerf3 server."),
			mcp.WithString("server",
				mcp.Required(),
				mcp.Description("iPerf3 server address (hostname or IP)"),
			),
			mcp.WithNumber("port",
				mcp.Description("iPerf3 server port (default: 5201)"),
			),
			mcp.WithNumber("duration",
				mcp.Description("Test duration in seconds (default: 10, max: 60)"),
			),
			mcp.WithString("protocol",
				mcp.Description("Protocol to use: tcp or udp (default: tcp)"),
			),
			mcp.WithString("direction",
				mcp.Description("Test direction: download, upload, or bidirectional (default: download)"),
			),
		),
		s.handleIperfTest,
	)
}

// handleDNSTest handles the dns_test tool.
func (s *Server) handleDNSTest(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tester := s.services.GetDNSTester()
	if tester == nil {
		return mcp.NewToolResultError("DNS tester not available"), nil
	}

	result := tester.Test(ctx)
	if result == nil {
		return mcp.NewToolResultError("DNS test failed: no result"), nil
	}

	return formatJSONResult(result)
}

// handleGatewayPing handles the gateway_ping tool.
func (s *Server) handleGatewayPing(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tester := s.services.GetGatewayTester()
	if tester == nil {
		return mcp.NewToolResultError("Gateway tester not available"), nil
	}

	// Check if ICMP is available
	if !s.services.IsICMPAvailable() {
		return mcp.NewToolResultError("Gateway ping requires raw socket capabilities (CAP_NET_RAW). Run with elevated privileges."), nil
	}

	result := tester.Test()
	if result == nil {
		return mcp.NewToolResultError("Gateway ping failed: no result"), nil
	}

	return formatJSONResult(result)
}

// handleSpeedtest handles the speedtest tool.
func (s *Server) handleSpeedtest(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tester := s.services.GetSpeedtestTester()
	if tester == nil {
		return mcp.NewToolResultError("Speedtest tester not available"), nil
	}

	// Check if already running
	status := tester.GetStatus()
	if status.Running {
		return mcp.NewToolResultError("A speedtest is already in progress"), nil
	}

	// Run with a 2-minute timeout
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	result, err := tester.Run(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Speedtest failed: %v", err)), nil
	}

	return formatJSONResult(result)
}

// parseIperfConfig parses iPerf arguments from the request into a ClientConfig.
func parseIperfConfig(serverAddr string, args map[string]interface{}) (*iperf.ClientConfig, error) {
	opts := &iperf.ClientConfig{
		Server:   serverAddr,
		Port:     5201,
		Duration: 10,
		Protocol: "tcp",
	}

	if port, ok := args["port"].(float64); ok && port > 0 {
		opts.Port = int(port)
	}

	if duration, ok := args["duration"].(float64); ok && duration > 0 {
		if duration > 60 {
			duration = 60
		}
		opts.Duration = int(duration)
	}

	if protocol, ok := args["protocol"].(string); ok && protocol != "" {
		if protocol != "tcp" && protocol != "udp" {
			return nil, fmt.Errorf("protocol must be 'tcp' or 'udp'")
		}
		opts.Protocol = protocol
	}

	if direction, ok := args["direction"].(string); ok && direction != "" {
		switch direction {
		case "download", "upload", "bidirectional":
			opts.Direction = direction
		default:
			return nil, fmt.Errorf("direction must be 'download', 'upload', or 'bidirectional'")
		}
	}

	return opts, nil
}

// handleIperfTest handles the iperf_test tool.
func (s *Server) handleIperfTest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverAddr, err := request.RequireString("server")
	if err != nil {
		return mcp.NewToolResultError("server parameter is required"), nil
	}

	args := getArguments(request)
	opts, err := parseIperfConfig(serverAddr, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	manager := s.services.GetIperfManager()
	if manager == nil {
		return mcp.NewToolResultError("iPerf manager not available"), nil
	}

	// Check if already running
	status := manager.GetClientStatus()
	if status.Running {
		return mcp.NewToolResultError("An iPerf test is already in progress"), nil
	}

	// Run with timeout based on duration + buffer
	timeout := time.Duration(opts.Duration+30) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := manager.RunClient(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("iPerf test failed: %v", err)), nil
	}

	return formatJSONResult(result)
}
