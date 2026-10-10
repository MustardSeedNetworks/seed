package api

// /api/v1/polling-targets endpoints — Stage A5.3. Operators add
// devices to poll here; the SNMP poller picks them up on the next
// tick (Enabled=true) and the chain reconcilers fold the resulting
// observations into the topology graph.
//
//   GET    /api/v1/polling-targets         list (this session's client only)
//   POST   /api/v1/polling-targets         create
//   GET    /api/v1/polling-targets/{id}    fetch one
//   PUT    /api/v1/polling-targets/{id}    full update
//   DELETE /api/v1/polling-targets/{id}    delete
//
// List is read-only; the mutating routes go through writeGated so
// only operator+ roles can add/edit/remove devices to poll. A create past
// the licence's target limit, or a chain naming a Pro collector below Pro,
// is a 402 (polling_entitlements.go).
//
// Every route is scoped to the client on the caller's session claim
// (internal/auth/client_context.go). There is no ?client_id filter and no
// clientId body field: the tenant is not something a caller gets to name, and
// a request that carries no claim is refused rather than defaulted. A target
// owned by another client is indistinguishable from one that does not exist.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/targets"
)

const (
	pollingTargetsPath       = APIVersionPrefix + "/polling-targets"
	pollingTargetsPathPrefix = pollingTargetsPath + "/"
)

// PollingTargetRequest is the request body for POST + PUT. Mirrors the
// repo struct minus the audit columns the server fills in.
type PollingTargetRequest struct {
	Name            string   `json:"name"`
	IPAddress       string   `json:"ipAddress"`
	SNMPVersion     string   `json:"snmpVersion,omitempty"`
	CredentialsID   string   `json:"credentialsId,omitempty"`
	PollIntervalSec int      `json:"pollIntervalSeconds,omitempty"`
	Enabled         bool     `json:"enabled"`
	CollectorChain  []string `json:"collectorChain,omitempty"`
}

// PollingTargetResponse is one polling target on the wire. It is built
// explicitly from the domain row so the wire format stays stable when DB
// columns evolve. Timestamps are RFC 3339 UTC; lastPolledAt is absent until
// the first poll.
type PollingTargetResponse struct {
	ID              string   `json:"id"`
	ClientID        string   `json:"clientId"`
	Name            string   `json:"name"`
	IPAddress       string   `json:"ipAddress"`
	SNMPVersion     string   `json:"snmpVersion"`
	CredentialsID   string   `json:"credentialsId"`
	PollIntervalSec int      `json:"pollIntervalSeconds"`
	Enabled         bool     `json:"enabled"`
	CollectorChain  []string `json:"collectorChain"`
	LastStatus      string   `json:"lastStatus"`
	LastError       string   `json:"lastError"`
	LastPolledAt    string   `json:"lastPolledAt,omitempty"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

// PollingTargetListResponse is the GET /polling-targets envelope.
type PollingTargetListResponse struct {
	Count   int                     `json:"count"`
	Targets []PollingTargetResponse `json:"targets"`
}

// handlePollingTargets routes the collection-level endpoint
// (GET list / POST create) — both share the same path so the mux
// dispatches by method here.
func (s *Server) handlePollingTargets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPollingTargets(w, r)
	case http.MethodPost:
		s.createPollingTarget(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
	}
}

// handlePollingTargetByID routes the resource-level endpoint
// (GET / PUT / DELETE).
func (s *Server) handlePollingTargetByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, pollingTargetsPathPrefix)
	if id == "" || strings.Contains(id, "/") {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid target id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getPollingTarget(w, r, id)
	case http.MethodPut:
		s.updatePollingTarget(w, r, id)
	case http.MethodDelete:
		s.deletePollingTarget(w, r, id)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
	}
}

// callerClient resolves the session's client, or writes the denial and reports
// false. A request that cannot be attributed to a tenant is unauthenticated,
// not a request belonging to the default tenant.
func (s *Server) callerClient(w http.ResponseWriter, r *http.Request) (string, bool) {
	clientID, err := auth.ClientIDFromContext(r.Context())
	if err != nil {
		// method, path, client_ip and user_agent are already on the "http
		// request" line for this request id (internal/logging/middleware.go),
		// so this one carries only what that line does not: why it was denied.
		logging.FromContext(r.Context()).WarnContext(r.Context(),
			"Request carries no client claim",
			"event", "auth.unauthorized",
		)
		writeError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
			"Authentication required")
		return "", false
	}
	return clientID, true
}

func (s *Server) listPollingTargets(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	list, err := s.pollingTargets.ListAll(r.Context(), clientID)
	if err != nil {
		logger.ErrorContext(r.Context(), "list polling_targets failed", "error", err)
		writePollingError(w, r, err, "Failed to list polling targets")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, PollingTargetListResponse{
		Count:   len(list),
		Targets: encodePollingTargets(list),
	})
}

func (s *Server) getPollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	target, err := s.pollingTargets.Get(r.Context(), clientID, id)
	if err != nil {
		logger.ErrorContext(r.Context(), "get polling_target failed", "id", id, "error", err)
		writePollingError(w, r, err, "Failed to load target")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, encodePollingTarget(target))
}

func (s *Server) createPollingTarget(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	in, err := decodePollingTargetInput(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, err.Error())
		return
	}
	if feature, unlicensed := s.unlicensedCollectorFeature(in.CollectorChain); unlicensed {
		s.sendFeatureGate(w, r, feature)
		return
	}
	target := inputToTarget(in, "", clientID)
	if createErr := s.pollingTargets.Create(r.Context(), target); createErr != nil {
		if errors.As(createErr, new(targets.LimitError)) {
			s.sendFeatureGate(w, r, "estate_polling")
			return
		}
		logger.ErrorContext(r.Context(), "create polling_target failed", "error", createErr)
		writePollingError(w, r, createErr, "Failed to create target")
		return
	}
	s.reloadSNMPPoller(r.Context())
	w.Header().Set("Location", pollingTargetsPathPrefix+target.ID)
	sendJSONResponse(w, logger, http.StatusOK, encodePollingTarget(target))
}

func (s *Server) updatePollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	in, err := decodePollingTargetInput(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, err.Error())
		return
	}
	if feature, unlicensed := s.unlicensedCollectorFeature(in.CollectorChain); unlicensed {
		s.sendFeatureGate(w, r, feature)
		return
	}
	current, updErr := s.pollingTargets.Update(r.Context(), clientID, inputToTarget(in, id, clientID))
	if updErr != nil {
		logger.ErrorContext(r.Context(), "update polling_target failed", "id", id, "error", updErr)
		writePollingError(w, r, updErr, "Failed to update target")
		return
	}
	s.reloadSNMPPoller(r.Context())
	sendJSONResponse(w, logger, http.StatusOK, encodePollingTarget(current))
}

func (s *Server) deletePollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	if err := s.pollingTargets.Delete(r.Context(), clientID, id); err != nil {
		logger.ErrorContext(r.Context(), "delete polling_target failed", "id", id, "error", err)
		writePollingError(w, r, err, "Failed to delete target")
		return
	}
	s.reloadSNMPPoller(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// writePollingError maps a polling-targets use-case error to its HTTP status: the
// store unwired → 503 (the prior "Database not initialized"), a missing target →
// 404, a repo validation error → 400 with its message, anything else → 500.
func writePollingError(w http.ResponseWriter, r *http.Request, err error, genericMsg string) {
	var ve targets.ValidationError
	switch {
	case errors.Is(err, targets.ErrUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
	case errors.Is(err, targets.ErrNotFound):
		writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Target not found")
	case errors.As(err, &ve):
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, ve.Msg)
	default:
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, genericMsg)
	}
}

// decodePollingTargetInput parses the JSON body. Returns a 400-
// shaped error for malformed payloads.
func decodePollingTargetInput(r *http.Request) (*PollingTargetRequest, error) {
	var in PollingTargetRequest
	if r.Body == nil {
		return nil, errors.New("body required")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return nil, errors.New("invalid JSON body: " + err.Error())
	}
	if in.Name == "" {
		return nil, errors.New("'name' required")
	}
	if in.IPAddress == "" {
		return nil, errors.New("'ipAddress' required")
	}
	return &in, nil
}

// inputToTarget maps the wire shape into the domain struct. id ==
// "" means "let the repo generate one" (Create); a non-empty id is
// used for Update. The owning client comes from the caller's session, never
// from the body — PollingTargetRequest has no field for it, and the decoder
// rejects unknown fields, so a request that tries to name a tenant gets a 400.
func inputToTarget(in *PollingTargetRequest, id, clientID string) *polling.Target {
	return &polling.Target{
		ID:              id,
		ClientID:        clientID,
		Name:            in.Name,
		IPAddress:       in.IPAddress,
		SNMPVersion:     in.SNMPVersion,
		CredentialsID:   in.CredentialsID,
		PollIntervalSec: in.PollIntervalSec,
		Enabled:         in.Enabled,
		CollectorChain:  in.CollectorChain,
	}
}

// encodePollingTarget shapes the domain row into its wire form. A row whose
// stored chain is empty answers [] rather than null.
func encodePollingTarget(t *polling.Target) PollingTargetResponse {
	chain := t.CollectorChain
	if chain == nil {
		chain = []string{}
	}
	return PollingTargetResponse{
		ID:              t.ID,
		ClientID:        t.ClientID,
		Name:            t.Name,
		IPAddress:       t.IPAddress,
		SNMPVersion:     t.SNMPVersion,
		CredentialsID:   t.CredentialsID,
		PollIntervalSec: t.PollIntervalSec,
		Enabled:         t.Enabled,
		CollectorChain:  chain,
		LastStatus:      t.LastStatus,
		LastError:       t.LastError,
		LastPolledAt:    formatTime(t.LastPolledAt),
		CreatedAt:       formatTime(t.CreatedAt),
		UpdatedAt:       formatTime(t.UpdatedAt),
	}
}

func encodePollingTargets(targets []*polling.Target) []PollingTargetResponse {
	out := make([]PollingTargetResponse, 0, len(targets))
	for _, t := range targets {
		out = append(out, encodePollingTarget(t))
	}
	return out
}
