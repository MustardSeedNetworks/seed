// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Path Discovery Handlers (Sprint 3 - L2/L3 path tracing)
// ============================================================================

// Path discovery method constants.
const (
	PathMethodL2   = "l2"
	PathMethodL3   = "l3"
	PathMethodBoth = "both"
)

// PathRequest represents a path discovery request.
type PathRequest struct {
	Source      string `json:"source"`      // IP address or "self" for local machine
	Destination string `json:"destination"` // IP address or hostname
	Method      string `json:"method"`      // "l3", "l2", "both"
	Protocol    string `json:"protocol"`    // "icmp", "udp", "tcp" (for L3 traceroute)
	Port        int    `json:"port"`        // Port for tcp/udp traceroute
}

// PathResponse contains both L2 and L3 path information.
type PathResponse struct {
	L3Path *discovery.TracerouteResult `json:"l3Path,omitempty"`
	L2Path *discovery.L2PathResult     `json:"l2Path,omitempty"`
}

// handlePath performs L2 and/or L3 path discovery between two endpoints.
//
// POST /api/discovery/path
//
// This endpoint traces the network path between a source and destination:
//   - L3 path: Uses traceroute (ICMP/UDP/TCP) to find Layer 3 (IP) hops
//   - L2 path: Uses LLDP/CDP neighbor data to find Layer 2 (switch) hops
//   - Both: Provides complete network path visibility
//
// Request body:
//
//	{
//	  "source": "192.168.1.100",      // Source IP or "self"
//	  "destination": "192.168.1.200", // Destination IP or hostname
//	  "method": "both",               // "l3", "l2", or "both"
//	  "protocol": "icmp",             // "icmp", "udp", "tcp" (for L3)
//	  "port": 80                      // Port for tcp/udp (optional)
//	}
//
// Response contains L3Path and/or L2Path based on the requested method.
//
// L3 Path (traceroute):
//   - Shows IP-level hops (routers, gateways)
//   - Includes RTT measurements and hop addresses
//   - Useful for Internet routing and WAN connectivity
//
// L2 Path (switch path):
//   - Shows switch-level hops using LLDP/CDP data
//   - Includes ingress/egress port information
//   - Enriched with SNMP data when available
//   - Useful for LAN troubleshooting and VLAN tracing
//
// Authentication: Required
// Rate limiting: Recommended (can be resource-intensive)
//
// Error responses:
//   - 400: Invalid request (missing required fields)
//   - 404: Source or destination device not found
//   - 500: Path discovery failed
//   - 503: Required services not available
func (s *Server) handlePath(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req PathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), err.Error())
		return
	}

	// Validate request
	if req.Source == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Source is required", "")
		return
	}
	if req.Destination == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Destination is required", "")
		return
	}
	if req.Method == "" {
		req.Method = PathMethodBoth // Default to both L2 and L3
	}
	if req.Protocol == "" {
		req.Protocol = "icmp" // Default to ICMP traceroute
	}

	// Validate method
	if req.Method != PathMethodL2 && req.Method != PathMethodL3 && req.Method != PathMethodBoth {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Method must be 'l2', 'l3', or 'both'", "")
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	response := s.performPathDiscovery(ctx, w, req, logger)
	if response == nil {
		// Error already sent
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, response)
}

// performPathDiscovery executes the path discovery based on the requested method.
func (s *Server) performPathDiscovery(ctx context.Context, w http.ResponseWriter, req PathRequest, logger *slog.Logger) *PathResponse {
	response := &PathResponse{}

	// Perform L3 traceroute if requested
	if req.Method == PathMethodL3 || req.Method == PathMethodBoth {
		l3Path := s.performL3Trace(ctx, req)
		if l3Path.Error != "" {
			logger.Warn("L3 traceroute failed", "error", l3Path.Error)
			// Don't fail the entire request if only L3 fails and L2 was also requested
			if req.Method == PathMethodL3 {
				sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "L3 traceroute failed", l3Path.Error)
				return nil
			}
		}
		response.L3Path = l3Path
	}

	// Perform L2 path discovery if requested
	if req.Method == PathMethodL2 || req.Method == PathMethodBoth {
		l2Path, err := s.performL2Trace(ctx, req)
		if err != nil {
			logger.Warn("L2 path discovery failed", "error", err)
			// Don't fail the entire request if only L2 fails and L3 was also requested
			if req.Method == PathMethodL2 {
				sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "L2 path discovery failed", err.Error())
				return nil
			}
		} else {
			response.L2Path = l2Path
		}
	}

	// Check if we got any results
	if response.L3Path == nil && response.L2Path == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Path discovery failed for all methods", "")
		return nil
	}

	return response
}

// performL3Trace performs Layer 3 traceroute.
func (s *Server) performL3Trace(ctx context.Context, req PathRequest) *discovery.TracerouteResult {
	// Create tracer with reasonable defaults
	tracer := discovery.NewTracer(3*time.Second, 30)

	switch req.Protocol {
	case "icmp":
		return tracer.TraceICMP(ctx, req.Destination)
	case "udp":
		port := req.Port
		if port == 0 {
			port = 33434 // Traditional traceroute port
		}
		return tracer.TraceUDP(ctx, req.Destination, port)
	case "tcp":
		port := req.Port
		if port == 0 {
			port = 80 // Default to HTTP
		}
		return tracer.TraceTCP(ctx, req.Destination, port)
	default:
		return tracer.TraceICMP(ctx, req.Destination) // Fallback to ICMP
	}
}

// performL2Trace performs Layer 2 path discovery using LLDP/CDP.
func (s *Server) performL2Trace(ctx context.Context, req PathRequest) (*discovery.L2PathResult, error) {
	if s.deviceDiscovery == nil {
		return nil, fmt.Errorf("device discovery not available")
	}

	// Resolve "self" to local IP if specified
	sourceIP := req.Source
	if sourceIP == "self" {
		_, localIP := s.deviceDiscovery.GetSubnetInfo()
		if localIP == "" {
			return nil, fmt.Errorf("cannot determine local IP address")
		}
		sourceIP = localIP
	}

	// Build L2 path
	builder := discovery.NewL2PathBuilder(s.deviceDiscovery, &s.config.SNMP)
	result, err := builder.BuildPath(ctx, sourceIP, req.Destination)
	if err != nil {
		return nil, err
	}

	return result, nil
}
