package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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

// Path discovery timing constants.
const (
	// pathDiscoveryTimeoutMin is the timeout in minutes for path discovery operations.
	pathDiscoveryTimeoutMin = 2

	// tracerouteMaxHops is the maximum number of hops for traceroute operations.
	tracerouteMaxHops = 30
)

// PathRequest represents a path discovery request.
type PathRequest struct {
	Source      string `json:"source"      validate:"required"`                     // IP address or "self" for local machine
	Destination string `json:"destination" validate:"required"`                     // IP address or hostname
	Method      string `json:"method"      validate:"required,oneof=l3 l2 both"`    // route resolution method
	Protocol    string `json:"protocol"    validate:"omitempty,oneof=icmp udp tcp"` // L3 traceroute protocol
	Port        int    `json:"port"        validate:"omitempty,gte=1,lte=65535"`    // TCP/UDP traceroute port
}

// PathResponse contains both L2 and L3 path information.
type PathResponse struct {
	L3Path *TracerouteResult `json:"l3Path,omitempty"`
	L2Path *L2PathResult     `json:"l2Path,omitempty"`
}

// TracerouteResult is the flat transport view of discovery.TracerouteResult
// (an L3 path), mirroring its wire shape so the published schema does not
// depend on the discovery domain package.
type TracerouteResult struct {
	Target    string          `json:"target"`
	TargetIP  string          `json:"targetIp"`
	Protocol  string          `json:"protocol"`
	Port      int             `json:"port,omitempty"`
	Hops      []TracerouteHop `json:"hops"`
	Completed bool            `json:"completed"`
	Error     string          `json:"error,omitempty"`
}

// TracerouteHop is the flat transport view of a single L3 traceroute hop.
type TracerouteHop struct {
	TTL      int           `json:"ttl"`
	IP       string        `json:"ip,omitempty"`
	Hostname string        `json:"hostname,omitempty"`
	RTT      time.Duration `json:"rtt"`
	State    string        `json:"state"`
}

// L2PathResult is the flat transport view of discovery.L2PathResult (an L2
// switch path).
type L2PathResult struct {
	Hops []L2Hop `json:"hops"`
}

// L2Hop is the flat transport view of a single L2 hop. IngressPort/EgressPort
// keep pointer semantics (no omitempty) so an unknown port stays null on the
// wire, matching the domain shape.
type L2Hop struct {
	Device      string    `json:"device"`
	DeviceIP    string    `json:"deviceIp"`
	IngressPort *PortInfo `json:"ingressPort"`
	EgressPort  *PortInfo `json:"egressPort"`
	Source      string    `json:"source"`
}

// PortInfo is the flat transport view of a switch port on an L2 hop.
type PortInfo struct {
	Name        string `json:"name"`
	Index       int    `json:"index"`
	Speed       string `json:"speed"`
	Duplex      string `json:"duplex"`
	VLANs       []int  `json:"vlans"`
	IsTrunk     bool   `json:"isTrunk"`
	ConnectedTo string `json:"connectedTo"`
}

// toTracerouteResult maps an L3 traceroute result onto its flat transport view,
// preserving nil so an absent L3 path stays omitted.
func toTracerouteResult(result *discovery.TracerouteResult) *TracerouteResult {
	if result == nil {
		return nil
	}
	hops := make([]TracerouteHop, 0, len(result.Hops))
	for _, h := range result.Hops {
		hops = append(hops, TracerouteHop{
			TTL:      h.TTL,
			IP:       h.IP,
			Hostname: h.Hostname,
			RTT:      h.RTT,
			State:    h.State,
		})
	}
	return &TracerouteResult{
		Target:    result.Target,
		TargetIP:  result.TargetIP,
		Protocol:  result.Protocol,
		Port:      result.Port,
		Hops:      hops,
		Completed: result.Completed,
		Error:     result.Error,
	}
}

// toL2PathResult maps an L2 switch path onto its flat transport view,
// preserving nil so an absent L2 path stays omitted.
func toL2PathResult(result *discovery.L2PathResult) *L2PathResult {
	if result == nil {
		return nil
	}
	hops := make([]L2Hop, 0, len(result.Hops))
	for _, h := range result.Hops {
		hops = append(hops, L2Hop{
			Device:      h.Device,
			DeviceIP:    h.DeviceIP,
			IngressPort: toPortInfo(h.IngressPort),
			EgressPort:  toPortInfo(h.EgressPort),
			Source:      h.Source,
		})
	}
	return &L2PathResult{Hops: hops}
}

// toPortInfo maps a switch port onto its flat transport view, preserving nil.
func toPortInfo(port *discovery.PortInfo) *PortInfo {
	if port == nil {
		return nil
	}
	return &PortInfo{
		Name:        port.Name,
		Index:       port.Index,
		Speed:       port.Speed,
		Duplex:      port.Duplex,
		VLANs:       port.VLANs,
		IsTrunk:     port.IsTrunk,
		ConnectedTo: port.ConnectedTo,
	}
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

	var req PathRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	// Validate request
	if req.Source == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Source is required",
			"",
		)
		return
	}
	if req.Destination == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Destination is required",
			"",
		)
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
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Method must be 'l2', 'l3', or 'both'",
			"",
		)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), pathDiscoveryTimeoutMin*time.Minute)
	defer cancel()

	response := s.performPathDiscovery(ctx, w, req, logger)
	if response == nil {
		// Error already sent
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, response)
}

// performPathDiscovery executes the path discovery based on the requested method.
func (s *Server) performPathDiscovery(
	ctx context.Context,
	w http.ResponseWriter,
	req PathRequest,
	logger *slog.Logger,
) *PathResponse {
	response := &PathResponse{}

	// Perform L3 traceroute if requested
	if req.Method == PathMethodL3 || req.Method == PathMethodBoth {
		l3Path := s.performL3Trace(ctx, req)
		if l3Path.Error != "" {
			logger.WarnContext(ctx, "L3 traceroute failed", "error", l3Path.Error)
			// Don't fail the entire request if only L3 fails and L2 was also requested
			if req.Method == PathMethodL3 {
				logger.ErrorContext(ctx, "L3 traceroute failed", "error_details", l3Path.Error)
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusInternalServerError,
					ErrCodeInternal,
					"L3 traceroute failed",
					"",
				)
				return nil
			}
		}
		response.L3Path = toTracerouteResult(l3Path)
	}

	// Perform L2 path discovery if requested
	if req.Method == PathMethodL2 || req.Method == PathMethodBoth {
		l2Path, err := s.performL2Trace(ctx, req)
		if err != nil {
			logger.WarnContext(ctx, "L2 path discovery failed", "error", err)
			// Don't fail the entire request if only L2 fails and L3 was also requested
			if req.Method == PathMethodL2 {
				logger.ErrorContext(ctx, "L2 path discovery failed", "error", err)
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusInternalServerError,
					ErrCodeInternal,
					"L2 path discovery failed",
					"",
				)
				return nil
			}
		} else {
			response.L2Path = toL2PathResult(l2Path)
		}
	}

	// Check if we got any results
	if response.L3Path == nil && response.L2Path == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Path discovery failed for all methods",
			"",
		)
		return nil
	}

	return response
}

// TraceHopMessage is the WebSocket message for streaming traceroute hops.
type TraceHopMessage struct {
	Target    string                  `json:"target"`
	TargetIP  string                  `json:"targetIp"`
	Protocol  string                  `json:"protocol"`
	Hop       discovery.TracerouteHop `json:"hop"`
	Completed bool                    `json:"completed"`
}

// performL3Trace performs Layer 3 traceroute with streaming WebSocket updates.
func (s *Server) performL3Trace(ctx context.Context, req PathRequest) *discovery.TracerouteResult {
	// Create tracer with faster settings for responsive UI
	// 1s per-hop timeout (was 3s), max hops - most traces complete in 10-15 hops
	tracer := discovery.NewTracer(1*time.Second, tracerouteMaxHops)

	// Callback to broadcast each hop via SSE
	onHop := func(hop discovery.TracerouteHop, result *discovery.TracerouteResult) bool {
		if s.sseHub() != nil {
			s.sseHub().Broadcast(Message{
				Type: "traceHop",
				Payload: TraceHopMessage{
					Target:    result.Target,
					TargetIP:  result.TargetIP,
					Protocol:  result.Protocol,
					Hop:       hop,
					Completed: result.Completed,
				},
			})
		}
		return true // Continue tracing
	}

	switch req.Protocol {
	case "icmp":
		return tracer.TraceICMPStreaming(ctx, req.Destination, onHop)
	case "udp":
		port := req.Port
		if port == 0 {
			port = 33434 // Traditional traceroute port
		}
		// UDP and TCP still use non-streaming for now
		return tracer.TraceUDP(ctx, req.Destination, port)
	case "tcp":
		port := req.Port
		if port == 0 {
			port = 80 // Default to HTTP
		}
		return tracer.TraceTCP(ctx, req.Destination, port)
	default:
		return tracer.TraceICMPStreaming(ctx, req.Destination, onHop)
	}
}

// performL2Trace performs Layer 2 path discovery using LLDP/CDP.
func (s *Server) performL2Trace(
	ctx context.Context,
	req PathRequest,
) (*discovery.L2PathResult, error) {
	if s.deviceDiscovery() == nil {
		return nil, errors.New("device discovery not available")
	}

	// Resolve "self" to local IP if specified
	sourceIP := req.Source
	if sourceIP == "self" {
		_, localIP := s.deviceDiscovery().GetSubnetInfo()
		if localIP == "" {
			return nil, errors.New("cannot determine local IP address")
		}
		sourceIP = localIP
	}

	// Build L2 path
	builder := discovery.NewL2PathBuilder(s.deviceDiscovery(), s.snmpConfig())
	result, err := builder.BuildPath(ctx, sourceIP, req.Destination)
	if err != nil {
		return nil, err
	}

	return result, nil
}
