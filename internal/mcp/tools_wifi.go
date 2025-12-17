package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerWiFiTools registers all WiFi-related MCP tools.
func (s *Server) registerWiFiTools(isAllowed func(string) bool) {
	// wifi_scan - Scan for WiFi networks
	s.addTool("wifi_scan", isAllowed,
		mcp.NewTool("wifi_scan",
			mcp.WithDescription("Scan for available WiFi networks. Returns a list of networks with SSID, BSSID, signal strength, channel, frequency, and security type. Networks are sorted by signal strength (strongest first)."),
		),
		s.handleWiFiScan,
	)

	// wifi_info - Get current WiFi connection info
	s.addTool("wifi_info", isAllowed,
		mcp.NewTool("wifi_info",
			mcp.WithDescription("Get information about the current WiFi connection including SSID, BSSID, signal strength, channel, and security type. Returns null if not connected to WiFi."),
		),
		s.handleWiFiInfo,
	)
}

// handleWiFiScan handles the wifi_scan tool.
func (s *Server) handleWiFiScan(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	scanner := s.services.GetWiFiScanner()
	if scanner == nil {
		return mcp.NewToolResultError("WiFi scanner not available"), nil
	}

	networks, err := scanner.Scan(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("WiFi scan failed: %v", err)), nil
	}

	if len(networks) == 0 {
		return mcp.NewToolResultText("No WiFi networks found. Ensure WiFi interface is available and enabled."), nil
	}

	return formatJSONResult(networks)
}

// handleWiFiInfo handles the wifi_info tool.
func (s *Server) handleWiFiInfo(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager := s.services.GetWiFiManager()
	if manager == nil {
		return mcp.NewToolResultError("WiFi manager not available"), nil
	}

	info, err := manager.GetCurrentNetwork()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get WiFi info: %v", err)), nil
	}

	if info == nil {
		return mcp.NewToolResultText("Not currently connected to a WiFi network."), nil
	}

	return formatJSONResult(info)
}
