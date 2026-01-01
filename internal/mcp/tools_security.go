package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/krisarmstrong/seed/internal/discovery"
)

// registerSecurityTools registers all security-related MCP tools.
func (s *Server) registerSecurityTools(isAllowed func(string) bool) {
	// vulnerability_scan - Scan device for CVEs
	s.addTool("vulnerability_scan", isAllowed,
		mcp.NewTool(
			"vulnerability_scan",
			mcp.WithDescription(
				"Scan a device for known vulnerabilities (CVEs) based on detected services and versions. Requires the device to be in the discovered devices list.",
			),
			mcp.WithString("ip",
				mcp.Required(),
				mcp.Description("IP address of the device to scan for vulnerabilities"),
			),
		),
		s.handleVulnerabilityScan,
	)

	// rogue_dhcp_check - Check for rogue DHCP servers
	s.addTool("rogue_dhcp_check", isAllowed,
		mcp.NewTool(
			"rogue_dhcp_check",
			mcp.WithDescription(
				"Check for rogue DHCP servers on the network. Returns a list of detected DHCP servers and whether they are known/authorized or potentially rogue.",
			),
		),
		s.handleRogueDHCPCheck,
	)

	// snmp_query - Query SNMP OID on device
	s.addTool("snmp_query", isAllowed,
		mcp.NewTool(
			"snmp_query",
			mcp.WithDescription(
				"Query a device via SNMP to get system information. Uses configured community strings or SNMPv3 credentials.",
			),
			mcp.WithString("host",
				mcp.Required(),
				mcp.Description("Target hostname or IP address"),
			),
			mcp.WithString("oid",
				mcp.Description("SNMP OID to query (default: system info OIDs like sysDescr, sysName, sysUpTime)"),
			),
			mcp.WithString("community",
				mcp.Description("SNMP community string (uses configured communities if not specified)"),
			),
		),
		s.handleSNMPQuery,
	)
}

// handleVulnerabilityScan handles the vulnerability_scan tool.
func (s *Server) handleVulnerabilityScan(
	ctx context.Context,
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

	var targetDevice *discovery.DiscoveredDevice
	for _, d := range svc.GetDevices() {
		if d.IP == ip {
			targetDevice = d
			break
		}
	}

	if targetDevice == nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Device %s not found in discovered devices. Run network_scan first.", ip),
		), nil
	}

	scanner := s.services.GetVulnScanner()
	if scanner == nil {
		return mcp.NewToolResultError(
			"Vulnerability scanner not available. Enable vulnerability scanning in configuration.",
		), nil
	}

	result, err := scanner.ScanDevice(ctx, targetDevice)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Vulnerability scan failed: %v", err)), nil
	}

	return formatJSONResult(result)
}

// handleRogueDHCPCheck handles the rogue_dhcp_check tool.
func (s *Server) handleRogueDHCPCheck(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	detector := s.services.GetRogueDetector()
	if detector == nil {
		return mcp.NewToolResultError("Rogue DHCP detector not available"), nil
	}

	servers := detector.GetDetectedServers()
	if servers == nil {
		return mcp.NewToolResultText(
			"No DHCP servers detected. The detector may need more time to capture DHCP traffic.",
		), nil
	}

	return formatJSONResult(map[string]any{
		"running": detector.IsRunning(),
		"servers": servers,
	})
}

// handleSNMPQuery handles the snmp_query tool.
func (s *Server) handleSNMPQuery(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host, _ := request.RequireString("host")
	if host == "" {
		return mcp.NewToolResultError(
			"host parameter is required",
		), nil
	}

	// Get SNMP configuration
	cfg := s.services.GetConfig()
	if cfg == nil {
		return mcp.NewToolResultError("Configuration not available"), nil
	}

	args := getArguments(request)
	// Use provided community or fall back to configured communities
	var community string
	if c, ok := args["community"].(string); ok && c != "" {
		community = c
	} else if len(cfg.SNMP.Communities) > 0 {
		community = cfg.SNMP.Communities[0]
	} else {
		return mcp.NewToolResultError("No SNMP community string provided and none configured"), nil
	}

	// For now, return a simplified response
	// A full implementation would use the snmp package
	result := map[string]any{
		"host":      host,
		"community": community,
		"status":    "SNMP query functionality requires snmp package integration",
	}

	return formatJSONResult(result)
}
