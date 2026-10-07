package api

// Interface statistics (UI-SEED-21, #3191): every interface the polling
// targets have reported, with the rates of its latest rated poll.
//
//   GET /api/v1/topology/interfaces   list, highest error rate first
//
// No feature gate: the rates exist only for polling targets, which the tier
// already caps.

import (
	"errors"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
)

// InterfaceRatesResponse is one rated poll. Octets are per second and absent
// when the counter could not be rated; utilization is a percentage of the
// line rate and absent when the line rate is unknown. Errors and discards are
// packets per second.
type InterfaceRatesResponse struct {
	SampledAt      time.Time `json:"sampledAt"`
	InOctets       *float64  `json:"inOctetsPerSec,omitempty"`
	OutOctets      *float64  `json:"outOctetsPerSec,omitempty"`
	InUtilization  *float64  `json:"inUtilizationPct,omitempty"`
	OutUtilization *float64  `json:"outUtilizationPct,omitempty"`
	InErrors       float64   `json:"inErrorsPerSec"`
	OutErrors      float64   `json:"outErrorsPerSec"`
	InDiscards     float64   `json:"inDiscardsPerSec"`
	OutDiscards    float64   `json:"outDiscardsPerSec"`
}

// InterfaceStatsResponse is one interface of one polling target. Rates is
// absent until the first rated poll; speedBps is 0 when the agent reports
// no line rate.
type InterfaceStatsResponse struct {
	TargetID   string                  `json:"targetId"`
	TargetName string                  `json:"targetName"`
	IfIndex    uint32                  `json:"ifIndex"`
	Name       string                  `json:"name"`
	Alias      string                  `json:"alias,omitempty"`
	OperStatus ifstats.OperStatus      `json:"operStatus"      jsonschema:"enum=up,enum=down,enum=testing,enum=unknown,enum=dormant,enum=notPresent,enum=lowerLayerDown"`
	SpeedBps   uint64                  `json:"speedBps"`
	Rates      *InterfaceRatesResponse `json:"rates,omitempty"`
}

// InterfaceStatsListResponse is the GET /api/v1/topology/interfaces body.
type InterfaceStatsListResponse struct {
	Interfaces []InterfaceStatsResponse `json:"interfaces"`
	Count      int                      `json:"count"`
}

// interfaceStatsRoutes registers the interface statistics read. The path is a
// literal, not a const, so the route-consumer gate sees it.
func (s *Server) interfaceStatsRoutes() []route {
	return []route{{
		path:    APIVersionPrefix + "/topology/interfaces",
		handler: s.handleInterfaceStats,
		methods: []string{http.MethodGet},
	}}
}

// handleInterfaceStats serves GET /api/v1/topology/interfaces.
func (s *Server) handleInterfaceStats(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	ifaces, err := s.interfaceStats.List(r.Context())
	if errors.Is(err, ifstats.ErrUnavailable) {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.interfaceStats.storeUnavailable"), "")
		return
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "list interface stats failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.interfaceStats.readFailed"), "")
		return
	}

	out := make([]InterfaceStatsResponse, 0, len(ifaces))
	for _, iface := range ifaces {
		resp := InterfaceStatsResponse{
			TargetID:   iface.TargetID,
			TargetName: iface.TargetName,
			IfIndex:    iface.IfIndex,
			Name:       iface.Name,
			Alias:      iface.Alias,
			OperStatus: iface.OperStatus,
			SpeedBps:   iface.SpeedBps,
		}
		if rt := iface.Rates; rt != nil {
			resp.Rates = &InterfaceRatesResponse{
				SampledAt:      rt.At,
				InOctets:       rt.InOctets,
				OutOctets:      rt.OutOctets,
				InUtilization:  rt.InUtilization,
				OutUtilization: rt.OutUtilization,
				InErrors:       rt.InErrors,
				OutErrors:      rt.OutErrors,
				InDiscards:     rt.InDiscards,
				OutDiscards:    rt.OutDiscards,
			}
		}
		out = append(out, resp)
	}
	sendJSONResponse(w, logger, http.StatusOK, InterfaceStatsListResponse{Interfaces: out, Count: len(out)})
}
