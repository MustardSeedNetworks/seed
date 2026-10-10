package api

// Interface statistics (UI-SEED-21, #3191): every interface the polling
// targets have reported, with the rates of its latest rated poll.
//
//   GET /api/v1/topology/interfaces           list, highest error rate first
//   GET /api/v1/topology/interfaces/history   one interface's rates over a
//                                             recent window
//
// No feature gate: the rates exist only for polling targets, which the tier
// already caps.

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

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

// InterfaceHistoryResponse is the GET /api/v1/topology/interfaces/history
// body: the interface's rates over (from, to], oldest first, one point per
// bucket of bucketSeconds that holds a poll. A point's octets and utilization
// are the bucket's mean and its errors and discards the bucket's peak;
// sampledAt is the bucket's last poll.
type InterfaceHistoryResponse struct {
	TargetID      string                   `json:"targetId"`
	IfIndex       uint32                   `json:"ifIndex"`
	Range         ifstats.HistoryRange     `json:"range"         jsonschema:"enum=1h,enum=24h,enum=7d"`
	From          time.Time                `json:"from"`
	To            time.Time                `json:"to"`
	BucketSeconds float64                  `json:"bucketSeconds"`
	Points        []InterfaceRatesResponse `json:"points"`
}

// interfaceStatsRoutes registers the interface statistics reads. The paths
// are literals, not consts, so the route-consumer gate sees them.
func (s *Server) interfaceStatsRoutes() []route.Route {
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/topology/interfaces",
			Handler: s.handleInterfaceStats,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/topology/interfaces/history",
			Handler: s.handleInterfaceHistory,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
	}
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
			rates := ratesResponse(*rt)
			resp.Rates = &rates
		}
		out = append(out, resp)
	}
	sendJSONResponse(w, logger, http.StatusOK, InterfaceStatsListResponse{Interfaces: out, Count: len(out)})
}

// handleInterfaceHistory serves GET /api/v1/topology/interfaces/history
// ?target=<polling target ID>&ifIndex=<n>[&range=1h|24h|7d], 24h by default.
// An interface with no rated poll in the window answers with no points.
func (s *Server) handleInterfaceHistory(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	params := r.URL.Query()

	target := params.Get("target")
	ifIndex, indexErr := strconv.ParseUint(params.Get("ifIndex"), 10, 32)
	rng := ifstats.HistoryRange(params.Get("range"))
	if rng == "" {
		rng = ifstats.RangeDay
	}
	if target == "" || indexErr != nil || rng.Duration() == 0 {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.interfaceStats.invalidHistoryQuery"), "")
		return
	}

	h, err := s.interfaceStats.History(r.Context(), target, uint32(ifIndex), rng, time.Now().UTC())
	if errors.Is(err, ifstats.ErrUnavailable) {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.interfaceStats.storeUnavailable"), "")
		return
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "read interface history failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.interfaceStats.readFailed"), "")
		return
	}

	points := make([]InterfaceRatesResponse, len(h.Points))
	for i, p := range h.Points {
		points[i] = ratesResponse(p)
	}
	sendJSONResponse(w, logger, http.StatusOK, InterfaceHistoryResponse{
		TargetID:      target,
		IfIndex:       uint32(ifIndex),
		Range:         rng,
		From:          h.From,
		To:            h.To,
		BucketSeconds: h.Bucket.Seconds(),
		Points:        points,
	})
}

func ratesResponse(rt ifstats.Rates) InterfaceRatesResponse {
	return InterfaceRatesResponse{
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
