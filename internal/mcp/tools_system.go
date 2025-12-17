package mcp

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/krisarmstrong/seed/internal/version"
)

// registerSystemTools registers all system-related MCP tools.
func (s *Server) registerSystemTools(isAllowed func(string) bool) {
	// get_interfaces - List network interfaces
	s.addTool("get_interfaces", isAllowed,
		mcp.NewTool("get_interfaces",
			mcp.WithDescription("List all network interfaces with their type, status, speed, and capabilities."),
		),
		s.handleGetInterfaces,
	)

	// get_link_status - Get interface link status
	s.addTool("get_link_status", isAllowed,
		mcp.NewTool("get_link_status",
			mcp.WithDescription("Get the current link status of the active network interface including carrier state, speed, duplex, and auto-negotiation status."),
		),
		s.handleGetLinkStatus,
	)

	// get_ip_config - Get IP configuration
	s.addTool("get_ip_config", isAllowed,
		mcp.NewTool("get_ip_config",
			mcp.WithDescription("Get the current IP configuration including IPv4/IPv6 addresses, netmask, gateway, and DNS servers."),
		),
		s.handleGetIPConfig,
	)

	// get_public_ip - Get public IP address
	s.addTool("get_public_ip", isAllowed,
		mcp.NewTool("get_public_ip",
			mcp.WithDescription("Get the public IP address as seen from the internet. Useful for determining NAT status and external connectivity."),
		),
		s.handleGetPublicIP,
	)

	// get_vlan_info - Get VLAN information
	s.addTool("get_vlan_info", isAllowed,
		mcp.NewTool("get_vlan_info",
			mcp.WithDescription("Get VLAN information for the current interface including tagged and untagged VLANs detected."),
		),
		s.handleGetVLANInfo,
	)

	// system_health - Get system health metrics
	s.addTool("system_health", isAllowed,
		mcp.NewTool("system_health",
			mcp.WithDescription("Get system health metrics including CPU usage, memory usage, disk space, and uptime."),
		),
		s.handleSystemHealth,
	)

	// get_discovery_status - Get discovery service status
	s.addTool("get_discovery_status", isAllowed,
		mcp.NewTool("get_discovery_status",
			mcp.WithDescription("Get the current status of the discovery service including active profile, running state, and device count."),
		),
		s.handleGetDiscoveryStatus,
	)
}

// handleGetInterfaces handles the get_interfaces tool.
func (s *Server) handleGetInterfaces(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager := s.services.GetNetManager()
	if manager == nil {
		return mcp.NewToolResultError("Network manager not available"), nil
	}

	interfaces := manager.GetInterfaces()
	return formatJSONResult(interfaces)
}

// handleGetLinkStatus handles the get_link_status tool.
func (s *Server) handleGetLinkStatus(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	monitor := s.services.GetLinkMonitor()
	if monitor == nil {
		return mcp.NewToolResultError("Link monitor not available"), nil
	}

	state := monitor.GetState()
	return formatJSONResult(map[string]interface{}{
		"state": state,
		"isUp":  monitor.IsUp(),
	})
}

// handleGetIPConfig handles the get_ip_config tool.
func (s *Server) handleGetIPConfig(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager := s.services.GetNetManager()
	if manager == nil {
		return mcp.NewToolResultError("Network manager not available"), nil
	}

	// Get the current interface and its info
	currentIface := manager.GetCurrentInterface()
	if currentIface == "" {
		return mcp.NewToolResultError("No current interface configured"), nil
	}

	ifaceInfo, err := manager.GetInterface(currentIface)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get interface info: %v", err)), nil
	}

	return formatJSONResult(ifaceInfo)
}

// handleGetPublicIP handles the get_public_ip tool.
func (s *Server) handleGetPublicIP(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checker := s.services.GetPublicIPChecker()
	if checker == nil {
		return mcp.NewToolResultError("Public IP checker not available"), nil
	}

	result := checker.GetPublicIP(ctx)
	return formatJSONResult(result)
}

// handleGetVLANInfo handles the get_vlan_info tool.
func (s *Server) handleGetVLANInfo(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager := s.services.GetVLANManager()
	if manager == nil {
		return mcp.NewToolResultError("VLAN manager not available"), nil
	}

	info := manager.GetInfo()
	return formatJSONResult(info)
}

// handleSystemHealth handles the system_health tool.
func (s *Server) handleSystemHealth(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	health := map[string]interface{}{
		"version":    version.Version,
		"goVersion":  runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"numCPU":     runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"allocMB":      float64(memStats.Alloc) / 1024 / 1024,
			"totalAllocMB": float64(memStats.TotalAlloc) / 1024 / 1024,
			"sysMB":        float64(memStats.Sys) / 1024 / 1024,
			"numGC":        memStats.NumGC,
		},
		"timestamp": time.Now().UTC(),
	}

	// Add config info
	cfg := s.services.GetConfig()
	if cfg != nil {
		health["interface"] = cfg.Interface.Default
		health["mcpEnabled"] = cfg.MCP.Enabled
	}

	// Add ICMP capability
	health["icmpAvailable"] = s.services.IsICMPAvailable()

	return formatJSONResult(health)
}

// handleGetDiscoveryStatus handles the get_discovery_status tool.
func (s *Server) handleGetDiscoveryStatus(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc := s.services.GetDiscoveryService()
	if svc == nil {
		return mcp.NewToolResultError("Discovery service not available"), nil
	}

	status := svc.GetStatus()
	return formatJSONResult(status)
}
