package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/indicators"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// handlers_flow_indicators.go serves the threat indicator list the flow
// collector checks each flow against (P-C5). Seed ships no list and never
// fetches one: the operator supplies it, which keeps an air-gapped install
// silent.

// FlowIndicatorsRequest replaces the indicator list. An empty list turns
// matching off.
type FlowIndicatorsRequest struct {
	Indicators []string `json:"indicators"`
}

// FlowIndicatorsResponse is the indicator list in effect.
type FlowIndicatorsResponse struct {
	Indicators []string `json:"indicators"`
}

// handleFlowIndicators serves /api/v1/flows/threat-indicators: GET reads the
// list and PUT replaces it. Flows collected after a change are checked
// against the new list; nothing already stored is re-checked.
func (s *Server) handleFlowIndicators(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	repo := s.db().FlowRecords()

	if r.Method == http.MethodPut {
		var req FlowIndicatorsRequest
		if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
			return
		}
		list, err := indicators.New(req.Indicators)
		if err != nil {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation,
				localizer.T("errors.flows.invalidIndicators"), err.Error())
			return
		}
		if setErr := repo.SetFlowIndicators(r.Context(), list); setErr != nil {
			flowIndicatorsFailed(w, r, setErr)
			return
		}
		logger.InfoContext(r.Context(), "Threat indicator list replaced", "indicators", list.Len())
	}

	list, err := repo.FlowIndicators(r.Context())
	if err != nil {
		flowIndicatorsFailed(w, r, err)
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, FlowIndicatorsResponse{Indicators: list.Entries()})
}

func flowIndicatorsFailed(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	logger.ErrorContext(r.Context(), "Failed to access threat indicators", "error", err)
	sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
		ErrCodeInternal, i18n.FromRequest(r).T("errors.flows.indicatorsFailed"), "")
}
