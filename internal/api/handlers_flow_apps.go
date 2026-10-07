package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/appid"
	"github.com/MustardSeedNetworks/seed/internal/flows"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// handlers_flow_apps.go serves application identification over collected
// flows (P-C4): the top applications over a window, and the signature table
// the collector names flows with. The table is a port-and-protocol
// heuristic (internal/appid) and is editable because no shipped table knows
// an operator's own services.

// Signature table sources, as the wire spells them.
const (
	appSignaturesBuiltin = "builtin"
	appSignaturesCustom  = "custom"
)

// FlowApplicationsResponse is the top applications by traffic.
type FlowApplicationsResponse struct {
	Window       HistoryWindowResponse `json:"window"`
	By           flows.Rank            `json:"by"`
	Applications []flows.Application   `json:"applications"`
}

// AppSignaturesRequest replaces the signature table.
type AppSignaturesRequest struct {
	Signatures []appid.Signature `json:"signatures"`
}

// AppSignaturesResponse is the signature table in effect. Source is
// "builtin" for the table Seed ships with and "custom" once an operator has
// replaced it.
type AppSignaturesResponse struct {
	Source     string            `json:"source"`
	Signatures []appid.Signature `json:"signatures"`
}

// handleFlowTopApplications serves GET /api/v1/flows/top-applications.
func (s *Server) handleFlowTopApplications(w http.ResponseWriter, r *http.Request) {
	q, ok := s.flowTopQuery(w, r)
	if !ok {
		return
	}
	apps, err := s.flows.TopApplications(r.Context(), q.read)
	if err != nil {
		flowReadFailed(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, FlowApplicationsResponse{
		Window:       windowResponse(q.window),
		By:           q.read.By,
		Applications: emptyIfNil(apps),
	})
}

// handleAppSignatures serves /api/v1/flows/application-signatures: GET reads
// the table, PUT replaces it and DELETE returns to the builtin one. A change
// names the flows stored after it; stored flows keep their names.
func (s *Server) handleAppSignatures(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodPut:
		var req AppSignaturesRequest
		if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeConfig, logger, localizer) {
			return
		}
		table, err := appid.New(req.Signatures)
		if err != nil {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation,
				localizer.T("errors.flows.invalidSignatures"), err.Error())
			return
		}
		if setErr := s.flows.SetAppSignatures(r.Context(), table); setErr != nil {
			appSignaturesFailed(w, r, setErr)
			return
		}
		logger.InfoContext(r.Context(), "Application signatures replaced", "signatures", len(req.Signatures))
	case http.MethodDelete:
		if err := s.flows.ResetAppSignatures(r.Context()); err != nil {
			appSignaturesFailed(w, r, err)
			return
		}
		logger.InfoContext(r.Context(), "Application signatures reset to builtin")
	}

	sigs, err := s.flows.AppSignatures(r.Context())
	if err != nil {
		appSignaturesFailed(w, r, err)
		return
	}
	source := appSignaturesBuiltin
	if sigs.Custom {
		source = appSignaturesCustom
	}
	sendJSONResponse(w, logger, http.StatusOK, AppSignaturesResponse{
		Source:     source,
		Signatures: sigs.Table.Signatures(),
	})
}

func appSignaturesFailed(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	logger.ErrorContext(r.Context(), "Failed to access application signatures", "error", err)
	sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
		ErrCodeInternal, i18n.FromRequest(r).T("errors.flows.signaturesFailed"), "")
}
