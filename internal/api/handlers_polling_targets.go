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
// only operator+ roles can add/edit/remove devices to poll.
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

// pollingTargetInput is the request body for POST + PUT. Mirrors the
// repo struct minus the audit columns the server fills in.
type pollingTargetInput struct {
	Name            string   `json:"name"`
	IPAddress       string   `json:"ipAddress"`
	SNMPVersion     string   `json:"snmpVersion,omitempty"`
	CredentialsID   string   `json:"credentialsId,omitempty"`
	PollIntervalSec int      `json:"pollIntervalSeconds,omitempty"`
	Enabled         bool     `json:"enabled"`
	CollectorChain  []string `json:"collectorChain,omitempty"`
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePollingTargetByID routes the resource-level endpoint
// (GET / PUT / DELETE).
func (s *Server) handlePollingTargetByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, pollingTargetsPathPrefix)
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "Missing or invalid target id", http.StatusBadRequest)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// callerClient resolves the session's client, or writes the denial and reports
// false. A request that cannot be attributed to a tenant is unauthenticated,
// not a request belonging to the default tenant.
func (s *Server) callerClient(w http.ResponseWriter, r *http.Request) (string, bool) {
	clientID, err := auth.ClientIDFromContext(r.Context())
	if err != nil {
		logging.FromContext(r.Context()).WarnContext(r.Context(),
			"Request carries no client claim",
			"event", "auth.unauthorized",
			"client_ip", GetClientIP(r),
			"path", r.URL.Path,
			"method", r.Method,
		)
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
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
		writePollingError(w, err, "Failed to list polling targets")
		return
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount: len(list),
		"targets":    encodePollingTargets(list),
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
		writePollingError(w, err, "Failed to load target")
		return
	}
	writeJSON(w, r, encodePollingTarget(target))
}

func (s *Server) createPollingTarget(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	in, err := decodePollingTargetInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	target := inputToTarget(in, "", clientID)
	if createErr := s.pollingTargets.Create(r.Context(), target); createErr != nil {
		logger.ErrorContext(r.Context(), "create polling_target failed", "error", createErr)
		writePollingError(w, createErr, "Failed to create target")
		return
	}
	w.Header().Set("Location", pollingTargetsPathPrefix+target.ID)
	writeJSON(w, r, encodePollingTarget(target))
}

func (s *Server) updatePollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	in, err := decodePollingTargetInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	current, updErr := s.pollingTargets.Update(r.Context(), clientID, inputToTarget(in, id, clientID))
	if updErr != nil {
		logger.ErrorContext(r.Context(), "update polling_target failed", "id", id, "error", updErr)
		writePollingError(w, updErr, "Failed to update target")
		return
	}
	writeJSON(w, r, encodePollingTarget(current))
}

func (s *Server) deletePollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	if err := s.pollingTargets.Delete(r.Context(), clientID, id); err != nil {
		logger.ErrorContext(r.Context(), "delete polling_target failed", "id", id, "error", err)
		writePollingError(w, err, "Failed to delete target")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writePollingError maps a polling-targets use-case error to its HTTP status: the
// store unwired → 503 (the prior "Database not initialized"), a missing target →
// 404, a repo validation error → 400 with its message, anything else → 500.
func writePollingError(w http.ResponseWriter, err error, genericMsg string) {
	var ve targets.ValidationError
	switch {
	case errors.Is(err, targets.ErrUnavailable):
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
	case errors.Is(err, targets.ErrNotFound):
		http.Error(w, "Target not found", http.StatusNotFound)
	case errors.As(err, &ve):
		http.Error(w, ve.Msg, http.StatusBadRequest)
	default:
		http.Error(w, genericMsg, http.StatusInternalServerError)
	}
}

// decodePollingTargetInput parses the JSON body. Returns a 400-
// shaped error for malformed payloads.
func decodePollingTargetInput(r *http.Request) (*pollingTargetInput, error) {
	var in pollingTargetInput
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
// from the body — pollingTargetInput has no field for it, and the decoder
// rejects unknown fields, so a request that tries to name a tenant gets a 400.
func inputToTarget(in *pollingTargetInput, id, clientID string) *polling.Target {
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

// encodePollingTarget shapes the domain row into JSON. Done
// explicitly so the wire format stays stable when DB columns
// evolve.
func encodePollingTarget(t *polling.Target) map[string]any {
	row := map[string]any{
		"id":                  t.ID,
		"clientId":            t.ClientID,
		jsonKeyName:           t.Name,
		"ipAddress":           t.IPAddress,
		"snmpVersion":         t.SNMPVersion,
		"credentialsId":       t.CredentialsID,
		"pollIntervalSeconds": t.PollIntervalSec,
		jsonKeyEnabled:        t.Enabled,
		"collectorChain":      t.CollectorChain,
		"lastStatus":          t.LastStatus,
		"lastError":           t.LastError,
		"createdAt":           formatTime(t.CreatedAt),
		"updatedAt":           formatTime(t.UpdatedAt),
	}
	if !t.LastPolledAt.IsZero() {
		row["lastPolledAt"] = formatTime(t.LastPolledAt)
	}
	return row
}

func encodePollingTargets(targets []*polling.Target) []map[string]any {
	out := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		out = append(out, encodePollingTarget(t))
	}
	return out
}
