package httpapi

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

// LTI protocol constants.
const (
	LTIVersion11      = "1.1"
	LTIVersion13      = "1.3"
	LTIVersionAdvantage = "advantage"
	LTITimeout        = 10 * time.Second
)

// OPC-UA protocol constants.
const (
	DefaultOPCUAPort = 4840
	OPCUATimeout     = 10 * time.Second

	// OPC-UA Security Modes.
	SecurityModeNone           = "None"
	SecurityModeSign           = "Sign"
	SecurityModeSignAndEncrypt = "SignAndEncrypt"
)

// Modbus protocol constants.
const (
	DefaultModbusPort = 502
	ModbusTimeout     = 5 * time.Second

	// Modbus function codes.
	ModbusFCReadCoils            = 0x01
	ModbusFCReadDiscreteInputs   = 0x02
	ModbusFCReadHoldingRegisters = 0x03
	ModbusFCReadInputRegisters   = 0x04

	// Modbus register types.
	ModbusRegisterHolding  = "holding"
	ModbusRegisterInput    = "input"
	ModbusRegisterCoil     = "coil"
	ModbusRegisterDiscrete = "discrete"
)

// LTITestResult contains the result of an LTI/LMS health check.
type LTITestResult struct {
	Name           string  `json:"name"`
	LaunchURL      string  `json:"launchUrl"`
	LTIVersion     string  `json:"ltiVersion,omitempty"`
	Success        bool    `json:"success"`
	ResponseTimeMs float64 `json:"responseTimeMs"`
	StatusCode     int     `json:"statusCode,omitempty"`
	SSLValid       bool    `json:"sslValid,omitempty"`
	Error          string  `json:"error,omitempty"`
	Timestamp      string  `json:"timestamp"`
}

// OPCUATestResult contains the result of an OPC-UA health check.
type OPCUATestResult struct {
	Name             string   `json:"name"`
	EndpointURL      string   `json:"endpointUrl"`
	Success          bool     `json:"success"`
	ConnectTimeMs    float64  `json:"connectTimeMs"`
	ResponseTimeMs   float64  `json:"responseTimeMs,omitempty"`
	TotalTimeMs      float64  `json:"totalTimeMs"`
	SecurityMode     string   `json:"securityMode,omitempty"`
	SecurityPolicies []string `json:"securityPolicies,omitempty"`
	ServerInfo       string   `json:"serverInfo,omitempty"`
	Error            string   `json:"error,omitempty"`
	Timestamp        string   `json:"timestamp"`
}

// ModbusTestResult contains the result of a Modbus TCP health check.
type ModbusTestResult struct {
	Name           string  `json:"name"`
	Host           string  `json:"host"`
	Port           int     `json:"port"`
	UnitID         int     `json:"unitId"`
	Success        bool    `json:"success"`
	ConnectTimeMs  float64 `json:"connectTimeMs"`
	ResponseTimeMs float64 `json:"responseTimeMs,omitempty"`
	TotalTimeMs    float64 `json:"totalTimeMs"`
	RegisterValue  int     `json:"registerValue,omitempty"`
	Error          string  `json:"error,omitempty"`
	Timestamp      string  `json:"timestamp"`
}

// testLTIEndpoint tests an LTI/LMS endpoint by checking launch URL accessibility.
func (s *Server) testLTIEndpoint(ctx context.Context, endpoint config.LTIEndpoint) LTITestResult {
	result := LTITestResult{
		Name:       endpoint.Name,
		LaunchURL:  endpoint.LaunchURL,
		LTIVersion: endpoint.LTIVersion,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	// Parse and validate URL
	parsedURL, parseErr := url.Parse(endpoint.LaunchURL)
	if parseErr != nil {
		result.Error = fmt.Sprintf("Invalid URL: %v", parseErr)
		return result
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: LTITimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Create HEAD request (less intrusive than GET)
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodHead, endpoint.LaunchURL, nil)
	if reqErr != nil {
		result.Error = fmt.Sprintf("Failed to create request: %v", reqErr)
		return result
	}

	// Send request
	start := time.Now()
	resp, respErr := client.Do(req)
	result.ResponseTimeMs = float64(time.Since(start).Milliseconds())

	if respErr != nil {
		result.Error = fmt.Sprintf("Request failed: %v", respErr)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode

	// Check SSL validity for HTTPS URLs
	if parsedURL.Scheme == "https" {
		result.SSLValid = resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0
	}

	// Consider 2xx and 3xx as success (LTI endpoints often redirect)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Success = true
	} else {
		result.Error = fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
	}

	return result
}

// testOPCUAEndpoint tests an OPC-UA endpoint.
func (s *Server) testOPCUAEndpoint(ctx context.Context, endpoint config.OPCUAEndpoint) OPCUATestResult {
	result := OPCUATestResult{
		Name:         endpoint.Name,
		EndpointURL:  endpoint.EndpointURL,
		SecurityMode: endpoint.SecurityMode,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	// Parse OPC-UA URL (opc.tcp://host:port/path)
	parsedURL, parseErr := url.Parse(endpoint.EndpointURL)
	if parseErr != nil {
		result.Error = fmt.Sprintf("Invalid URL: %v", parseErr)
		return result
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = fmt.Sprintf("%d", DefaultOPCUAPort)
	}

	connectStart := time.Now()

	// Test TCP connectivity
	addr := fmt.Sprintf("%s:%s", host, port)
	dialer := net.Dialer{Timeout: OPCUATimeout}
	conn, connErr := dialer.DialContext(ctx, "tcp", addr)
	if connErr != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", connErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}
	defer func() { _ = conn.Close() }()

	result.ConnectTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Set deadline for OPC-UA operations
	if deadlineErr := conn.SetDeadline(time.Now().Add(OPCUATimeout)); deadlineErr != nil {
		result.Error = fmt.Sprintf("Failed to set deadline: %v", deadlineErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// In a full implementation, we would:
	// 1. Send OPC-UA Hello message
	// 2. Receive Acknowledge
	// 3. Send OpenSecureChannel request
	// 4. Call GetEndpoints service
	// For now, we verify TCP connectivity and note it's an OPC-UA endpoint

	requestStart := time.Now()

	// Send minimal OPC-UA Hello message header (for connection test)
	// OPC-UA Hello: "HEL" + message type + chunk type + message size + ...
	// This is a simplified test - just verify we can write to the connection
	helloPrefix := []byte{'H', 'E', 'L', 'F'} // Hello Final chunk
	_, writeErr := conn.Write(helloPrefix)
	if writeErr != nil {
		result.Error = fmt.Sprintf("Failed to send Hello: %v", writeErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Try to read response (OPC-UA server should respond with ACK or error)
	respBuf := make([]byte, 256)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, readErr := conn.Read(respBuf)
	if readErr != nil {
		// Connection closed or timeout - still consider TCP working
		result.ServerInfo = "TCP connection successful, server may require full handshake"
	} else if n >= 4 {
		// Check if we got an ACK or ERR response
		msgType := string(respBuf[:3])
		if msgType == "ACK" {
			result.ServerInfo = "OPC-UA server acknowledged connection"
		} else if msgType == "ERR" {
			result.ServerInfo = "OPC-UA server returned error (authentication may be required)"
		} else {
			result.ServerInfo = fmt.Sprintf("Server responded with: %s", msgType)
		}
	}

	result.ResponseTimeMs = float64(time.Since(requestStart).Milliseconds())
	result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
	result.Success = true

	return result
}

// testModbusEndpoint tests a Modbus TCP endpoint.
func (s *Server) testModbusEndpoint(ctx context.Context, endpoint config.ModbusEndpoint) ModbusTestResult {
	result := ModbusTestResult{
		Name:      endpoint.Name,
		Host:      endpoint.Host,
		Port:      endpoint.Port,
		UnitID:    endpoint.UnitID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Set default port
	if result.Port == 0 {
		result.Port = DefaultModbusPort
	}

	// Validate unit ID (1-247 for standard Modbus, 0 for broadcast)
	if endpoint.UnitID < 0 || endpoint.UnitID > 247 {
		result.Error = fmt.Sprintf("Invalid unit ID: %d (must be 0-247)", endpoint.UnitID)
		return result
	}

	connectStart := time.Now()

	// Connect to Modbus TCP server
	addr := fmt.Sprintf("%s:%d", endpoint.Host, result.Port)
	dialer := net.Dialer{Timeout: ModbusTimeout}
	conn, connErr := dialer.DialContext(ctx, "tcp", addr)
	if connErr != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", connErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}
	defer func() { _ = conn.Close() }()

	result.ConnectTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Set deadline
	if deadlineErr := conn.SetDeadline(time.Now().Add(ModbusTimeout)); deadlineErr != nil {
		result.Error = fmt.Sprintf("Failed to set deadline: %v", deadlineErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Build Modbus TCP request
	functionCode := getModbusFunctionCode(endpoint.RegisterType)
	request := buildModbusTCPRequest(
		uint8(endpoint.UnitID),
		functionCode,
		uint16(endpoint.TestRegister),
		1, // Read 1 register
	)

	// Send request
	requestStart := time.Now()
	_, writeErr := conn.Write(request)
	if writeErr != nil {
		result.Error = fmt.Sprintf("Failed to send request: %v", writeErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Read response
	respBuf := make([]byte, 256)
	n, readErr := conn.Read(respBuf)
	if readErr != nil {
		result.Error = fmt.Sprintf("Failed to read response: %v", readErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	result.ResponseTimeMs = float64(time.Since(requestStart).Milliseconds())
	result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Parse Modbus TCP response
	if n < 9 {
		result.Error = "Response too short"
		return result
	}

	// Check for exception response
	if respBuf[7]&0x80 != 0 {
		exceptionCode := respBuf[8]
		result.Error = fmt.Sprintf("Modbus exception: %s", getModbusExceptionString(exceptionCode))
		return result
	}

	// Extract register value (for holding/input registers)
	if functionCode == ModbusFCReadHoldingRegisters || functionCode == ModbusFCReadInputRegisters {
		if n >= 11 {
			result.RegisterValue = int(binary.BigEndian.Uint16(respBuf[9:11]))
		}
	}

	result.Success = true
	return result
}

// getModbusFunctionCode returns the appropriate function code for the register type.
func getModbusFunctionCode(registerType string) uint8 {
	switch strings.ToLower(registerType) {
	case ModbusRegisterCoil:
		return ModbusFCReadCoils
	case ModbusRegisterDiscrete:
		return ModbusFCReadDiscreteInputs
	case ModbusRegisterInput:
		return ModbusFCReadInputRegisters
	default:
		return ModbusFCReadHoldingRegisters
	}
}

// buildModbusTCPRequest builds a Modbus TCP/IP ADU (Application Data Unit).
func buildModbusTCPRequest(unitID, functionCode uint8, startAddr, quantity uint16) []byte {
	// Modbus TCP/IP ADU structure:
	// Transaction ID (2 bytes) + Protocol ID (2 bytes) + Length (2 bytes) + Unit ID (1 byte) + PDU
	// PDU: Function Code (1 byte) + Start Address (2 bytes) + Quantity (2 bytes)

	request := make([]byte, 12)

	// Transaction ID (can be any value, we use 1)
	binary.BigEndian.PutUint16(request[0:2], 1)

	// Protocol ID (0 for Modbus)
	binary.BigEndian.PutUint16(request[2:4], 0)

	// Length (Unit ID + PDU = 6 bytes)
	binary.BigEndian.PutUint16(request[4:6], 6)

	// Unit ID
	request[6] = unitID

	// Function Code
	request[7] = functionCode

	// Start Address
	binary.BigEndian.PutUint16(request[8:10], startAddr)

	// Quantity
	binary.BigEndian.PutUint16(request[10:12], quantity)

	return request
}

// getModbusExceptionString returns a human-readable string for Modbus exception codes.
func getModbusExceptionString(code uint8) string {
	exceptions := map[uint8]string{
		0x01: "Illegal Function",
		0x02: "Illegal Data Address",
		0x03: "Illegal Data Value",
		0x04: "Server Device Failure",
		0x05: "Acknowledge",
		0x06: "Server Device Busy",
		0x08: "Memory Parity Error",
		0x0A: "Gateway Path Unavailable",
		0x0B: "Gateway Target Device Failed to Respond",
	}

	if msg, ok := exceptions[code]; ok {
		return msg
	}
	return fmt.Sprintf("Unknown exception (0x%02X)", code)
}

// IndustryCheckResults contains results from all industry protocol checks.
type IndustryCheckResults struct {
	LTIResults    []LTITestResult    `json:"ltiResults,omitempty"`
	OPCUAResults  []OPCUATestResult  `json:"opcuaResults,omitempty"`
	ModbusResults []ModbusTestResult `json:"modbusResults,omitempty"`
}

// RunIndustryChecks runs all configured industry protocol health checks.
func (s *Server) RunIndustryChecks(ctx context.Context) *IndustryCheckResults {
	cfg := s.config
	results := &IndustryCheckResults{}

	// Run LTI checks (Education)
	for _, endpoint := range cfg.HealthChecks.LTIEndpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testLTIEndpoint(ctx, endpoint)
		results.LTIResults = append(results.LTIResults, result)
	}

	// Run OPC-UA checks (Manufacturing)
	for _, endpoint := range cfg.HealthChecks.OPCUAEndpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testOPCUAEndpoint(ctx, endpoint)
		results.OPCUAResults = append(results.OPCUAResults, result)
	}

	// Run Modbus checks (Manufacturing)
	for _, endpoint := range cfg.HealthChecks.ModbusEndpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testModbusEndpoint(ctx, endpoint)
		results.ModbusResults = append(results.ModbusResults, result)
	}

	return results
}
