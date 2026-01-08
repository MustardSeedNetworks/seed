package mcp

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
