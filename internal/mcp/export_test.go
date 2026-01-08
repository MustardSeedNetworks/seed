package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// This file is only compiled during testing (due to _test.go suffix)
// and provides access to internal implementation details.

// ExportParsePorts exposes parsePorts for testing.
func ExportParsePorts(spec string) ([]int, error) {
	return parsePorts(spec)
}

// ExportParsePortRange exposes parsePortRange for testing.
func ExportParsePortRange(part string) (int, int, error) {
	return parsePortRange(part)
}

// ExportParseSinglePort exposes parseSinglePort for testing.
func ExportParseSinglePort(part string) (int, error) {
	return parseSinglePort(part)
}

// ExportAddPortIfUnique exposes addPortIfUnique for testing.
func ExportAddPortIfUnique(ports []int, seen map[int]bool, port int) []int {
	return addPortIfUnique(ports, seen, port)
}

// ExportAddPortRange exposes addPortRange for testing.
func ExportAddPortRange(ports []int, seen map[int]bool, start, end int) []int {
	return addPortRange(ports, seen, start, end)
}

// ExportParsePortPart exposes parsePortPart for testing.
func ExportParsePortPart(ports []int, seen map[int]bool, part string) ([]int, error) {
	return parsePortPart(ports, seen, part)
}

// ExportGetArguments exposes getArguments for testing.
func ExportGetArguments(request any) map[string]any {
	// Need to use the mcp.CallToolRequest type here
	// Since we can't directly use the external type, we handle this differently
	return make(map[string]any)
}

// ExportFormatJSONResult exposes formatJSONResult for testing.
func ExportFormatJSONResult(data any) (any, error) {
	return formatJSONResult(data)
}

// ExportParseIperfConfig exposes parseIperfConfig for testing.
func ExportParseIperfConfig(serverAddr string, args map[string]any) (any, error) {
	return parseIperfConfig(serverAddr, args)
}

// Exported constants for testing.
const (
	ExportDefaultScanTimeoutSeconds       = defaultScanTimeoutSeconds
	ExportMaxScanTimeoutSeconds           = maxScanTimeoutSeconds
	ExportMaxTracerouteHops               = maxTracerouteHops
	ExportDefaultTracerouteTimeoutSeconds = defaultTracerouteTimeoutSeconds
	ExportDefaultTCPProbeTimeoutSeconds   = defaultTCPProbeTimeoutSeconds
	ExportDefaultPortScanTimeoutSeconds   = defaultPortScanTimeoutSeconds
	ExportPortScanConcurrency             = portScanConcurrency
	ExportPortRangeSplitParts             = portRangeSplitParts
	ExportMaxPortRangeSize                = maxPortRangeSize
	ExportDefaultTracerouteMaxHops        = defaultTracerouteMaxHops
	ExportSpeedtestTimeoutMinutes         = SpeedtestTimeoutMinutes
	ExportDefaultIperfPort                = DefaultIperfPort
	ExportDefaultIperfDurationSeconds     = DefaultIperfDurationSeconds
	ExportMaxIperfDurationSeconds         = MaxIperfDurationSeconds
	ExportIperfTimeoutBufferSeconds       = IperfTimeoutBufferSeconds
	ExportBytesPerKilobyte                = BytesPerKilobyte
	ExportBytesPerMegabyte                = BytesPerMegabyte
)

// ServerTestAccessor provides access to Server's private fields for testing.
type ServerTestAccessor struct {
	Server *Server
}

// GetMCPServer returns the server's underlying MCP server.
func (s *ServerTestAccessor) GetMCPServer() any {
	return s.Server.mcpServer
}

// GetConfig returns the server's config.
func (s *ServerTestAccessor) GetConfig() any {
	return s.Server.config
}

// GetServices returns the server's service provider.
func (s *ServerTestAccessor) GetServices() ServiceProvider {
	return s.Server.services
}

// ExportHandleNetworkScan exposes handleNetworkScan for testing.
func (s *Server) ExportHandleNetworkScan(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleNetworkScan(ctx, request)
}

// ExportHandleGetDevices exposes handleGetDevices for testing.
func (s *Server) ExportHandleGetDevices(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleGetDevices(ctx, request)
}

// ExportHandleDeviceFingerprint exposes handleDeviceFingerprint for testing.
func (s *Server) ExportHandleDeviceFingerprint(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	return s.handleDeviceFingerprint(ctx, request)
}

// ExportHandleGetNeighbors exposes handleGetNeighbors for testing.
func (s *Server) ExportHandleGetNeighbors(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	return s.handleGetNeighbors(ctx, request)
}

// ExportHandleTraceroute exposes handleTraceroute for testing.
func (s *Server) ExportHandleTraceroute(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleTraceroute(ctx, request)
}

// ExportHandleTCPProbe exposes handleTCPProbe for testing.
func (s *Server) ExportHandleTCPProbe(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleTCPProbe(ctx, request)
}

// ExportHandlePortScan exposes handlePortScan for testing.
func (s *Server) ExportHandlePortScan(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handlePortScan(ctx, request)
}

// ExportHandleDNSTest exposes handleDNSTest for testing.
func (s *Server) ExportHandleDNSTest(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleDNSTest(ctx, request)
}

// ExportHandleGatewayPing exposes handleGatewayPing for testing.
func (s *Server) ExportHandleGatewayPing(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleGatewayPing(ctx, request)
}

// ExportHandleSpeedtest exposes handleSpeedtest for testing.
func (s *Server) ExportHandleSpeedtest(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleSpeedtest(ctx, request)
}

// ExportHandleIperfTest exposes handleIperfTest for testing.
func (s *Server) ExportHandleIperfTest(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleIperfTest(ctx, request)
}

// ExportHandleWiFiScan exposes handleWiFiScan for testing.
func (s *Server) ExportHandleWiFiScan(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleWiFiScan(ctx, request)
}

// ExportHandleWiFiInfo exposes handleWiFiInfo for testing.
func (s *Server) ExportHandleWiFiInfo(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleWiFiInfo(ctx, request)
}

// ExportHandleGetInterfaces exposes handleGetInterfaces for testing.
func (s *Server) ExportHandleGetInterfaces(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	return s.handleGetInterfaces(ctx, request)
}

// ExportHandleGetLinkStatus exposes handleGetLinkStatus for testing.
func (s *Server) ExportHandleGetLinkStatus(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	return s.handleGetLinkStatus(ctx, request)
}

// ExportHandleGetIPConfig exposes handleGetIPConfig for testing.
func (s *Server) ExportHandleGetIPConfig(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleGetIPConfig(ctx, request)
}

// ExportHandleGetPublicIP exposes handleGetPublicIP for testing.
func (s *Server) ExportHandleGetPublicIP(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleGetPublicIP(ctx, request)
}

// ExportHandleGetVLANInfo exposes handleGetVLANInfo for testing.
func (s *Server) ExportHandleGetVLANInfo(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleGetVLANInfo(ctx, request)
}

// ExportHandleSystemHealth exposes handleSystemHealth for testing.
func (s *Server) ExportHandleSystemHealth(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleSystemHealth(ctx, request)
}

// ExportHandleGetDiscoveryStatus exposes handleGetDiscoveryStatus for testing.
func (s *Server) ExportHandleGetDiscoveryStatus(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	return s.handleGetDiscoveryStatus(ctx, request)
}

// ExportHandleVulnerabilityScan exposes handleVulnerabilityScan for testing.
func (s *Server) ExportHandleVulnerabilityScan(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	return s.handleVulnerabilityScan(ctx, request)
}

// ExportHandleRogueDHCPCheck exposes handleRogueDHCPCheck for testing.
func (s *Server) ExportHandleRogueDHCPCheck(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	return s.handleRogueDHCPCheck(ctx, request)
}

// ExportHandleSNMPQuery exposes handleSNMPQuery for testing.
func (s *Server) ExportHandleSNMPQuery(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return s.handleSNMPQuery(ctx, request)
}

// CallToolRequest is a type alias for external mcp.CallToolRequest.
type CallToolRequest = mcp.CallToolRequest

// CallToolResult is a type alias for external mcp.CallToolResult.
type CallToolResult = mcp.CallToolResult

// NewCallToolRequest creates a new CallToolRequest for testing purposes.
func NewCallToolRequest(toolName string, arguments map[string]any) CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	}
}

// ExportGetArguments2 exposes getArguments properly for testing with mcp.CallToolRequest.
func ExportGetArguments2(request mcp.CallToolRequest) map[string]any {
	return getArguments(request)
}
