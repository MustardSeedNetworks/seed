package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

	// mlllpWrapperOverhead is the number of bytes added by MLLP framing (start + end bytes).
	mllpWrapperOverhead = 3

	// mllpReadBufferSize is the size of the buffer for reading MLLP messages.
	mllpReadBufferSize = 4096

	// minHL7ACKFieldsForError is the minimum field count to access error code (index 3).
	minHL7ACKFieldsForError = 4

	// fhirHTTPTimeout is the timeout for FHIR HTTP requests.
	fhirHTTPTimeout = 30 * time.Second

	// fhirMaxResponseBody is the maximum response body size for FHIR requests (1MB).
	fhirMaxResponseBody = 1 << 20
)

// HL7TestResult contains the result of an HL7 MLLP health check.
type HL7TestResult struct {
	Name           string  `json:"name"`
	Host           string  `json:"host"`
	Port           int     `json:"port"`
	Success        bool    `json:"success"`
	ConnectTimeMs  float64 `json:"connectTimeMs"`
	ResponseTimeMs float64 `json:"responseTimeMs,omitempty"`
	TotalTimeMs    float64 `json:"totalTimeMs"`
	ACKCode        string  `json:"ackCode,omitempty"`   // AA (Accept), AE (Error), AR (Reject)
	ErrorCode      string  `json:"errorCode,omitempty"` // HL7 error code if any
	Error          string  `json:"error,omitempty"`
	Timestamp      string  `json:"timestamp"`
}

// FHIRTestResult contains the result of a FHIR R4 health check.
type FHIRTestResult struct {
	Name           string   `json:"name"`
	BaseURL        string   `json:"baseUrl"`
	Success        bool     `json:"success"`
	ResponseTimeMs float64  `json:"responseTimeMs"`
	StatusCode     int      `json:"statusCode,omitempty"`
	FHIRVersion    string   `json:"fhirVersion,omitempty"` // e.g., "4.0.1"
	ServerName     string   `json:"serverName,omitempty"`  // From CapabilityStatement
	Resources      []string `json:"resources,omitempty"`   // Supported resource types
	ResourceCount  int      `json:"resourceCount,omitempty"`
	Error          string   `json:"error,omitempty"`
	Timestamp      string   `json:"timestamp"`
}

// testHL7Endpoint tests an HL7 MLLP endpoint.
//
//nolint:funlen // HL7 endpoint testing requires multiple steps: connection, message building, MLLP framing, and ACK parsing.
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
	if deadlineErr := conn.SetDeadline(time.Now().Add(HL7Timeout)); deadlineErr != nil {
		result.Error = fmt.Sprintf("Failed to set deadline: %v", deadlineErr)
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
	switch ackCode {
	case "AA", "CA":
		result.Success = true
	case "AE", "CE":
		result.Error = "Application error"
	case "AR", "CR":
		result.Error = "Application reject"
	case "":
		// No ACK code found but we got a response - connection works
		result.Success = true
		result.ACKCode = "OK"
	default:
		result.Error = fmt.Sprintf("Unexpected ACK code: %s", ackCode)
	}

	return result
}

// wrapMLLP wraps a message in MLLP framing.
func wrapMLLP(msg []byte) []byte {
	result := make([]byte, 0, len(msg)+mllpWrapperOverhead)
	result = append(result, MLLPStartByte)
	result = append(result, msg...)
	result = append(result, MLLPEndByte1, MLLPEndByte2)
	return result
}

// readMLLPMessage reads an MLLP-framed message from a connection.
//
//nolint:gocognit // MLLP framing requires byte-by-byte parsing with state machine logic.
func readMLLPMessage(conn net.Conn) ([]byte, error) {
	var buf bytes.Buffer
	readBuf := make([]byte, mllpReadBufferSize)
	foundStart := false

	for {
		n, err := conn.Read(readBuf)
		if err != nil {
			if errors.Is(err, io.EOF) && buf.Len() > 0 {
				break
			}
			return nil, err
		}

		for i := range n {
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
//
//nolint:nonamedreturns // Named returns improve readability for this HL7 parsing function.
func parseHL7ACK(msg string) (ackCode, errorCode string) {
	// Look for MSA segment: MSA|AA|original_msg_id|...
	for line := range strings.SplitSeq(msg, "\r") {
		if strings.HasPrefix(line, "MSA|") {
			fields := strings.Split(line, "|")
			if len(fields) > 1 {
				ackCode = fields[1]
			}
			if len(fields) >= minHL7ACKFieldsForError {
				errorCode = fields[3]
			}
			return ackCode, errorCode
		}
	}
	return "", ""
}

// testFHIREndpoint tests a FHIR R4 endpoint by checking its metadata.
func (s *Server) testFHIREndpoint(
	ctx context.Context,
	endpoint config.FHIREndpoint,
) FHIRTestResult {
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
	if authErr := applyFHIRAuth(req, endpoint); authErr != nil {
		result.Error = fmt.Sprintf("Authentication failed: %v", authErr)
		return result
	}

	// Make request
	client := &http.Client{Timeout: fhirHTTPTimeout}
	start := time.Now()
	resp, reqErr := client.Do(req)
	if reqErr != nil {
		result.Error = fmt.Sprintf("Request failed: %v", reqErr)
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
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, fhirMaxResponseBody))
	if readErr != nil {
		result.Error = fmt.Sprintf("Failed to read response: %v", readErr)
		return result
	}

	// Parse CapabilityStatement
	var capStmt FHIRCapabilityStatement
	if parseErr := json.Unmarshal(body, &capStmt); parseErr != nil {
		result.Error = fmt.Sprintf("Failed to parse CapabilityStatement: %v", parseErr)
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
			return errors.New("basic auth requires username")
		}
		auth := base64.StdEncoding.EncodeToString(
			[]byte(endpoint.Username + ":" + endpoint.Password),
		)
		req.Header.Set("Authorization", "Basic "+auth)

	case "bearer":
		if endpoint.BearerToken == "" {
			return errors.New("bearer auth requires token")
		}
		req.Header.Set("Authorization", "Bearer "+endpoint.BearerToken)

	case "oauth2":
		// For OAuth2, we would need to implement token fetching
		// This is a simplified version that expects a pre-obtained token
		if endpoint.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+endpoint.BearerToken)
		} else {
			return errors.New("oauth2 auth requires token_url and credentials (not yet implemented)")
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
