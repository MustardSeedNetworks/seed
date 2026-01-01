// Package api provides the HTTP/WebSocket server.
// handlers_iperf.go contains iPerf3 client and server handlers.
// Split from handlers_health_checks.go for code organization (Plan F).
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
)

// ============================================================================
// iPerf Types
// ============================================================================

// IperfInfoResponse contains iperf3 installation info.
type IperfInfoResponse struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

// IperfClientRequest is the request body for running an iperf3 client test.
type IperfClientRequest struct {
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`  // "tcp" or "udp"
	Reverse   bool   `json:"reverse"`   // true = download, false = upload (legacy)
	Direction string `json:"direction"` // "upload", "download", "bidirectional"
	Duration  int    `json:"duration"`  // seconds
	Parallel  int    `json:"parallel"`  // number of streams
}

// IperfResultResponse is the response for an iperf3 test result.
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

// IperfClientStatusResponse is the status of an iperf3 client test.
type IperfClientStatusResponse struct {
	Running  bool                 `json:"running"`
	Phase    string               `json:"phase"`
	Progress float64              `json:"progress"`
	Last     *IperfResultResponse `json:"last,omitempty"`
}

// IperfServerRequest is the request body for starting/stopping the iperf3 server.
type IperfServerRequest struct {
	Action string `json:"action"` // "start" or "stop"
	Port   int    `json:"port"`
}

// IperfSuggestion represents a discovered host that responds on the iperf port.
type IperfSuggestion struct {
	Host      string  `json:"host"`
	Hostname  string  `json:"hostname,omitempty"`
	Source    string  `json:"source,omitempty"`
	LatencyMs float64 `json:"latencyMs,omitempty"`
}

// ============================================================================
// iPerf Validation
// ============================================================================

// validateIperfClientRequest validates and normalizes an iperf client request.
func validateIperfClientRequest(req *IperfClientRequest) error {
	if req.Server == "" {
		return errors.New("server address required")
	}

	req.Protocol = strings.ToLower(req.Protocol)
	if req.Protocol == "" {
		req.Protocol = protoTCP
	}
	if req.Protocol != protoTCP && req.Protocol != protoUDP {
		return errors.New("protocol must be tcp or udp")
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
		return errors.New("direction must be upload, download, or bidirectional")
	}

	// Validate numeric parameters
	if req.Port != 0 {
		if err := validation.ValidatePort(req.Port); err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}
	}
	if err := validation.ValidatePositiveInt(req.Duration, "duration"); err != nil {
		return err
	}
	return validation.ValidatePositiveInt(req.Parallel, "parallel streams")
}

// ============================================================================
// iPerf Handlers
// ============================================================================

// handleIperfInfo returns iperf3 installation status and version.
func (s *Server) handleIperfInfo(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	resp := IperfInfoResponse{}
	iperfVersion, err := iperf.GetVersion()
	if err != nil {
		logger.Warn("Failed to get iperf version", "error", err)
		resp.Installed = false
		resp.Error = "iperf3 not found or not accessible"
	} else {
		resp.Installed = true
		resp.Version = iperfVersion
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleIperfClient runs an iperf3 client test.
func (s *Server) handleIperfClient(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req IperfClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	if err := validateIperfClientRequest(&req); err != nil {
		logger.Warn("iPerf validation failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.health.iperfValidationFailed"),
			"",
		)
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
	go func(logger *slog.Logger) {
		// Add timeout protection for iperf operations
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Duration+30)*time.Second)
		defer cancel()
		if _, err := s.iperfManager.RunClient(ctx, &iperfConfig); err != nil {
			logger.Error("iperf client failed", "error", err)
		}
	}(logger)

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"message": "iperf3 test started. Poll /api/iperf/client/status for results.",
	})
}

// handleIperfClientStatus returns the status of the iperf3 client test.
func (s *Server) handleIperfClientStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
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

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleIperfServer starts or stops the iperf3 server.
func (s *Server) handleIperfServer(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req IperfServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	switch req.Action {
	case "start":
		port := req.Port
		if port == 0 {
			port = 5201
		}
		if err := s.iperfManager.StartServer(port); err != nil {
			logger.Error("Failed to start iPerf server", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusInternalServerError,
				ErrCodeInternal,
				localizer.T("errors.health.iperfServerStartFailed"),
				"",
			)
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, map[string]any{
			"message": fmt.Sprintf("iperf3 server started on port %d", port),
			"port":    port,
		})
	case "stop":
		if err := s.iperfManager.StopServer(); err != nil {
			logger.Error("Failed to stop iPerf server", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusInternalServerError,
				ErrCodeInternal,
				localizer.T("errors.health.iperfServerStopFailed"),
				"",
			)
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"message": "iperf3 server stopped",
		})
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.health.iperfInvalidAction"),
			"",
		)
	}
}

// handleIperfServerStatus returns the iperf3 server status.
func (s *Server) handleIperfServerStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	status := s.iperfManager.GetServerStatus()
	sendJSONResponse(w, logger, http.StatusOK, status)
}

// handleIperfSuggestions returns discovered devices that respond on the iperf port.
func (s *Server) handleIperfSuggestions(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	if s.deviceDiscovery == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.health.deviceDiscoveryNotAvailable"),
			"",
		)
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

	sendJSONResponse(w, logger, http.StatusOK, suggestions)
}
