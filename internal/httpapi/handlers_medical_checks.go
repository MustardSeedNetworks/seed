package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

// HL7 MLLP Protocol Constants.
const (
	// MLLPStartByte is the MLLP message start byte (vertical tab).
	MLLPStartByte = 0x0B
	// MLLPEndByte1 is the first MLLP message end byte (file separator).
	MLLPEndByte1 = 0x1C
	// MLLPEndByte2 is the second MLLP message end byte (carriage return).
	MLLPEndByte2 = 0x0D

	// DefaultHL7Port is the default HL7 MLLP port.
	DefaultHL7Port = 2575

	// HL7Timeout is the default timeout for HL7 connections.
	HL7Timeout = 10 * time.Second
)

// HL7TestResult contains the result of an HL7 MLLP health check.
type HL7TestResult struct {
	Name          string  `json:"name"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	Success       bool    `json:"success"`
	ConnectTimeMs float64 `json:"connectTimeMs"`
	ResponseTimeMs float64 `json:"responseTimeMs,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	ACKCode       string  `json:"ackCode,omitempty"`    // AA (Accept), AE (Error), AR (Reject)
	ErrorCode     string  `json:"errorCode,omitempty"`  // HL7 error code if any
	Error         string  `json:"error,omitempty"`
	Timestamp     string  `json:"timestamp"`
}

// FHIRTestResult contains the result of a FHIR R4 health check.
type FHIRTestResult struct {
	Name           string   `json:"name"`
	BaseURL        string   `json:"baseUrl"`
	Success        bool     `json:"success"`
	ResponseTimeMs float64  `json:"responseTimeMs"`
	StatusCode     int      `json:"statusCode,omitempty"`
	FHIRVersion    string   `json:"fhirVersion,omitempty"`    // e.g., "4.0.1"
	ServerName     string   `json:"serverName,omitempty"`     // From CapabilityStatement
	Resources      []string `json:"resources,omitempty"`      // Supported resource types
	ResourceCount  int      `json:"resourceCount,omitempty"`
	Error          string   `json:"error,omitempty"`
	Timestamp      string   `json:"timestamp"`
}

// testHL7Endpoint tests an HL7 MLLP endpoint.
func (s *Server) testHL7Endpoint(ctx context.Context, endpoint config.HL7Endpoint) HL7TestResult {
	result := HL7TestResult{
		Name:      endpoint.Name,
		Host:      endpoint.Host,
		Port:      endpoint.Port,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if result.Port == 0 {
		result.Port = DefaultHL7Port
	}

	// Create connection with timeout
	addr := fmt.Sprintf("%s:%d", endpoint.Host, result.Port)
	connectStart := time.Now()

	dialer := net.Dialer{Timeout: HL7Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", err)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}
	defer func() { _ = conn.Close() }()

	result.ConnectTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Set read/write deadline
	if err := conn.SetDeadline(time.Now().Add(HL7Timeout)); err != nil {
		result.Error = fmt.Sprintf("Failed to set deadline: %v", err)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Build minimal HL7 ADT^A01 message for ping
	msgID := time.Now().Format("20060102150405")
	sendingApp := endpoint.SendingApp
	if sendingApp == "" {
		sendingApp = "SEED"
	}
	sendingFac := endpoint.SendingFac
	if sendingFac == "" {
		sendingFac = "SEED_FAC"
	}
	receivingApp := endpoint.ReceivingApp
	if receivingApp == "" {
		receivingApp = "TARGET"
	}
	receivingFac := endpoint.ReceivingFac
	if receivingFac == "" {
		receivingFac = "TARGET_FAC"
	}

	// Create minimal HL7 2.x message (ADT^A01 admission notification as health check)
	// Using QBP^Q21 (query by parameter) would be better but ADT is more universally supported
	timestamp := time.Now().Format("20060102150405")
	hl7Msg := fmt.Sprintf(
		"MSH|^~\\&|%s|%s|%s|%s|%s||ADT^A01|%s|P|2.5\r"+
			"EVN|A01|%s\r"+
			"PID|1||HEALTH_CHECK||SEED^PROBE|||U\r",
		sendingApp, sendingFac, receivingApp, receivingFac, timestamp, msgID, timestamp,
	)

	// Wrap in MLLP framing
	mllpMsg := wrapMLLP([]byte(hl7Msg))

	// Send message
	requestStart := time.Now()
	_, err = conn.Write(mllpMsg)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to send message: %v", err)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Read response with MLLP framing
	response, err := readMLLPMessage(conn)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to read response: %v", err)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	result.ResponseTimeMs = float64(time.Since(requestStart).Milliseconds())
	result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Parse ACK response
	ackCode, errorCode := parseHL7ACK(string(response))
	result.ACKCode = ackCode
	result.ErrorCode = errorCode

	// AA = Application Accept, CA = Commit Accept (success codes)
	if ackCode == "AA" || ackCode == "CA" {
		result.Success = true
	} else if ackCode == "AE" || ackCode == "CE" {
		result.Error = "Application error"
	} else if ackCode == "AR" || ackCode == "CR" {
		result.Error = "Application reject"
	} else if ackCode != "" {
		result.Error = fmt.Sprintf("Unexpected ACK code: %s", ackCode)
	} else {
		// No ACK code found but we got a response - connection works
		result.Success = true
		result.ACKCode = "OK"
	}

	return result
}

// wrapMLLP wraps a message in MLLP framing.
func wrapMLLP(msg []byte) []byte {
	result := make([]byte, 0, len(msg)+3)
	result = append(result, MLLPStartByte)
	result = append(result, msg...)
	result = append(result, MLLPEndByte1, MLLPEndByte2)
	return result
}

// readMLLPMessage reads an MLLP-framed message from a connection.
func readMLLPMessage(conn net.Conn) ([]byte, error) {
	var buf bytes.Buffer
	readBuf := make([]byte, 4096)
	foundStart := false

	for {
		n, err := conn.Read(readBuf)
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				break
			}
			return nil, err
		}

		for i := 0; i < n; i++ {
			b := readBuf[i]
			if !foundStart {
				if b == MLLPStartByte {
					foundStart = true
				}
				continue
			}

			// Check for end sequence
			if b == MLLPEndByte1 && i+1 < n && readBuf[i+1] == MLLPEndByte2 {
				return buf.Bytes(), nil
			}
			if b == MLLPEndByte1 && buf.Len() > 0 {
				// End byte 1 found, next read should have end byte 2
				return buf.Bytes(), nil
			}

			buf.WriteByte(b)
		}
	}

	return buf.Bytes(), nil
}

// parseHL7ACK extracts the acknowledgment code from an HL7 ACK message.
func parseHL7ACK(msg string) (ackCode, errorCode string) {
	// Look for MSA segment: MSA|AA|original_msg_id|...
	lines := strings.Split(msg, "\r")
	for _, line := range lines {
		if strings.HasPrefix(line, "MSA|") {
			fields := strings.Split(line, "|")
			if len(fields) > 1 {
				ackCode = fields[1]
			}
			if len(fields) > 3 {
				errorCode = fields[3]
			}
			return
		}
	}
	return "", ""
}

// testFHIREndpoint tests a FHIR R4 endpoint by checking its metadata.
func (s *Server) testFHIREndpoint(ctx context.Context, endpoint config.FHIREndpoint) FHIRTestResult {
	result := FHIRTestResult{
		Name:      endpoint.Name,
		BaseURL:   endpoint.BaseURL,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Build metadata URL
	metadataURL := strings.TrimSuffix(endpoint.BaseURL, "/") + "/metadata"

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	// Set Accept header for FHIR
	req.Header.Set("Accept", "application/fhir+json")

	// Apply authentication
	if err := applyFHIRAuth(req, endpoint); err != nil {
		result.Error = fmt.Sprintf("Authentication failed: %v", err)
		return result
	}

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("Request failed: %v", err)
		result.ResponseTimeMs = float64(time.Since(start).Milliseconds())
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.ResponseTimeMs = float64(time.Since(start).Milliseconds())
	result.StatusCode = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
		return result
	}

	// Read and parse CapabilityStatement
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		result.Error = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}

	// Parse CapabilityStatement
	var capStmt FHIRCapabilityStatement
	if err := json.Unmarshal(body, &capStmt); err != nil {
		result.Error = fmt.Sprintf("Failed to parse CapabilityStatement: %v", err)
		return result
	}

	// Extract relevant information
	result.FHIRVersion = capStmt.FHIRVersion
	if capStmt.Software.Name != "" {
		result.ServerName = capStmt.Software.Name
		if capStmt.Software.Version != "" {
			result.ServerName += " " + capStmt.Software.Version
		}
	}

	// Extract supported resource types
	for _, rest := range capStmt.Rest {
		for _, res := range rest.Resource {
			result.Resources = append(result.Resources, res.Type)
		}
	}
	result.ResourceCount = len(result.Resources)

	result.Success = true
	return result
}

// FHIRCapabilityStatement represents relevant parts of a FHIR CapabilityStatement.
type FHIRCapabilityStatement struct {
	ResourceType string `json:"resourceType"`
	FHIRVersion  string `json:"fhirVersion"`
	Software     struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"software"`
	Rest []struct {
		Mode     string `json:"mode"`
		Resource []struct {
			Type string `json:"type"`
		} `json:"resource"`
	} `json:"rest"`
}

// applyFHIRAuth applies authentication to a FHIR request.
func applyFHIRAuth(req *http.Request, endpoint config.FHIREndpoint) error {
	switch strings.ToLower(endpoint.AuthType) {
	case "", "none":
		// No authentication
		return nil

	case "basic":
		if endpoint.Username == "" {
			return fmt.Errorf("basic auth requires username")
		}
		auth := base64.StdEncoding.EncodeToString([]byte(endpoint.Username + ":" + endpoint.Password))
		req.Header.Set("Authorization", "Basic "+auth)

	case "bearer":
		if endpoint.BearerToken == "" {
			return fmt.Errorf("bearer auth requires token")
		}
		req.Header.Set("Authorization", "Bearer "+endpoint.BearerToken)

	case "oauth2":
		// For OAuth2, we would need to implement token fetching
		// This is a simplified version that expects a pre-obtained token
		if endpoint.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+endpoint.BearerToken)
		} else {
			return fmt.Errorf("oauth2 auth requires token_url and credentials (not yet implemented)")
		}

	default:
		return fmt.Errorf("unknown auth type: %s", endpoint.AuthType)
	}

	return nil
}

// MedicalCheckResults contains results from all medical protocol checks.
type MedicalCheckResults struct {
	HL7Results  []HL7TestResult  `json:"hl7Results,omitempty"`
	FHIRResults []FHIRTestResult `json:"fhirResults,omitempty"`
}

// RunMedicalChecks runs all configured medical protocol health checks.
func (s *Server) RunMedicalChecks(ctx context.Context) *MedicalCheckResults {
	cfg := s.config
	results := &MedicalCheckResults{}

	// Run HL7 checks
	for _, endpoint := range cfg.HealthChecks.HL7Endpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testHL7Endpoint(ctx, endpoint)
		results.HL7Results = append(results.HL7Results, result)
	}

	// Run FHIR checks
	for _, endpoint := range cfg.HealthChecks.FHIREndpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testFHIREndpoint(ctx, endpoint)
		results.FHIRResults = append(results.FHIRResults, result)
	}

	return results
}
