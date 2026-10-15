package api

// IP-SLA operations (P-D8, #3463): the Cisco IP-SLA operations the polling
// targets run, with their latest run, and one operation's trend.
//
//   GET /api/v1/ip-sla/operations           list, failing operations first
//   GET /api/v1/ip-sla/operations/history   one operation's runs over a
//                                           recent window
//
// No feature gate: the ip_sla collector runs only for polling targets, which
// the tier already caps.

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

// IPSLAJitterResponse is the latest run of a jitter-type operation. MOS is 0
// when the operation has no codec configured.
type IPSLAJitterResponse struct {
	AvgJitterMs uint32  `json:"avgJitterMs"`
	Received    uint32  `json:"received"`
	Lost        uint32  `json:"lost"`
	MOS         float64 `json:"mos"`
}

// IPSLAResultResponse is an operation's most recent run. Sense is the
// CISCO-RTTMON-MIB rttMonLatestRttOperSense; rttMs is meaningful only when
// completed is true.
type IPSLAResultResponse struct {
	Completed bool                 `json:"completed"`
	Sense     int                  `json:"sense"`
	RTTMs     uint32               `json:"rttMs"`
	Jitter    *IPSLAJitterResponse `json:"jitter,omitempty"`
}

// IPSLAOperationResponse is one operation of one polling target, as its latest
// poll observed it. rttType is the RttMonRttType (1 echo, 9 jitter, ...);
// status is the rttMonCtrlAdminStatus row status (1 active). Latest is absent
// until the operation has run.
type IPSLAOperationResponse struct {
	TargetID    string               `json:"targetId"`
	TargetName  string               `json:"targetName"`
	ObservedAt  time.Time            `json:"observedAt"`
	Index       int                  `json:"index"`
	Tag         string               `json:"tag,omitempty"`
	RttType     int                  `json:"rttType"`
	Target      string               `json:"target,omitempty"`
	ThresholdMs int                  `json:"thresholdMs"`
	FrequencyS  int                  `json:"frequencySeconds"`
	Status      int                  `json:"status"`
	Latest      *IPSLAResultResponse `json:"latest,omitempty"`
}

// IPSLAOperationsResponse is the GET /api/v1/ip-sla/operations body.
type IPSLAOperationsResponse struct {
	Operations []IPSLAOperationResponse `json:"operations"`
	Count      int                      `json:"count"`
}

// IPSLARunResponse is one bucket of an operation's runs: how many ran, the
// share that completed, and the mean of the values they measured. RTT is
// absent when no run completed; jitter, loss and MOS for an operation that is
// not of a jitter type.
type IPSLARunResponse struct {
	SampledAt    time.Time `json:"sampledAt"`
	Runs         int       `json:"runs"`
	CompletedPct float64   `json:"completedPct"`
	RTTMs        *float64  `json:"rttMs,omitempty"`
	JitterMs     *float64  `json:"jitterMs,omitempty"`
	LossPct      *float64  `json:"lossPct,omitempty"`
	MOS          *float64  `json:"mos,omitempty"`
}

// IPSLAHistoryResponse is the GET /api/v1/ip-sla/operations/history body: the
// operation's runs over (from, to], oldest first, one point per bucket of
// bucketSeconds that holds a run; sampledAt is the bucket's last run.
type IPSLAHistoryResponse struct {
	TargetID      string               `json:"targetId"`
	Index         int                  `json:"index"`
	Range         ifstats.HistoryRange `json:"range"         jsonschema:"enum=1h,enum=24h,enum=7d"`
	From          time.Time            `json:"from"`
	To            time.Time            `json:"to"`
	BucketSeconds float64              `json:"bucketSeconds"`
	Points        []IPSLARunResponse   `json:"points"`
}

// ipSLARoutes registers the IP-SLA reads. The paths are literals, not consts,
// so the route-consumer gate sees them.
func (s *Server) ipSLARoutes() []route.Route {
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/ip-sla/operations",
			Handler: s.handleIPSLAOperations,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/ip-sla/operations/history",
			Handler: s.handleIPSLAHistory,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
	}
}

// handleIPSLAOperations serves GET /api/v1/ip-sla/operations.
func (s *Server) handleIPSLAOperations(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	ops, err := s.ipSLA.List(r.Context())
	if errors.Is(err, sla.ErrUnavailable) {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.ipSla.storeUnavailable"), "")
		return
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "list ip-sla operations failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.ipSla.readFailed"), "")
		return
	}

	out := make([]IPSLAOperationResponse, 0, len(ops))
	for _, op := range ops {
		resp := IPSLAOperationResponse{
			TargetID:    op.TargetID,
			TargetName:  op.TargetName,
			ObservedAt:  op.ObservedAt,
			Index:       op.Index,
			Tag:         op.Tag,
			RttType:     op.RttType,
			Target:      op.Target,
			ThresholdMs: op.ThresholdMs,
			FrequencyS:  op.FrequencyS,
			Status:      op.Status,
		}
		if l := op.Latest; l != nil {
			resp.Latest = &IPSLAResultResponse{Completed: l.Completed(), Sense: l.Sense, RTTMs: l.RTTMs}
			if j := l.Jitter; j != nil {
				resp.Latest.Jitter = &IPSLAJitterResponse{
					AvgJitterMs: j.AvgJitterMs, Received: j.Received, Lost: j.Lost, MOS: j.MOS,
				}
			}
		}
		out = append(out, resp)
	}
	sendJSONResponse(w, logger, http.StatusOK, IPSLAOperationsResponse{Operations: out, Count: len(out)})
}

// handleIPSLAHistory serves GET /api/v1/ip-sla/operations/history
// ?target=<polling target ID>&index=<rttMonCtrlAdminIndex>[&range=1h|24h|7d],
// 24h by default. An operation with no run in the window answers with no
// points.
func (s *Server) handleIPSLAHistory(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	params := r.URL.Query()

	target := params.Get("target")
	index, indexErr := strconv.Atoi(params.Get("index"))
	rng := ifstats.HistoryRange(params.Get("range"))
	if rng == "" {
		rng = ifstats.RangeDay
	}
	if target == "" || indexErr != nil || index < 1 || rng.Duration() == 0 {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.ipSla.invalidHistoryQuery"), "")
		return
	}

	h, err := s.ipSLA.History(r.Context(), target, index, rng, time.Now().UTC())
	if errors.Is(err, sla.ErrUnavailable) {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.ipSla.storeUnavailable"), "")
		return
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "read ip-sla history failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.ipSla.readFailed"), "")
		return
	}

	points := make([]IPSLARunResponse, len(h.Points))
	for i, p := range h.Points {
		points[i] = IPSLARunResponse{
			SampledAt:    p.At,
			Runs:         p.Runs,
			CompletedPct: p.CompletedPct,
			RTTMs:        p.RTTMs,
			JitterMs:     p.JitterMs,
			LossPct:      p.LossPct,
			MOS:          p.MOS,
		}
	}
	sendJSONResponse(w, logger, http.StatusOK, IPSLAHistoryResponse{
		TargetID:      target,
		Index:         index,
		Range:         rng,
		From:          h.From,
		To:            h.To,
		BucketSeconds: h.Bucket.Seconds(),
		Points:        points,
	})
}
