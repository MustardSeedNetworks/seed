package api

// /api/v1/polling-targets endpoints — Stage A5.3. Operators add
// devices to poll here; the SNMP poller picks them up on the next
// tick (Enabled=true) and the chain reconcilers fold the resulting
// observations into the topology graph.
//
//   GET    /api/v1/polling-targets         list (filter by ?client_id)
//   POST   /api/v1/polling-targets         create
//   GET    /api/v1/polling-targets/{id}    fetch one
//   PUT    /api/v1/polling-targets/{id}    full update
//   DELETE /api/v1/polling-targets/{id}    delete
//
// List is read-only; the mutating routes go through writeGated so
// only operator+ roles can add/edit/remove devices to poll.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	ClientID        string   `json:"clientId,omitempty"`
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

func (s *Server) listPollingTargets(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	list, err := s.pollingTargets.List(r.Context(), r.URL.Query().Get("client_id"))
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
	target, err := s.pollingTargets.Get(r.Context(), id)
	if err != nil {
		logger.ErrorContext(r.Context(), "get polling_target failed", "id", id, "error", err)
		writePollingError(w, err, "Failed to load target")
		return
	}
	writeJSON(w, r, encodePollingTarget(target))
}

func (s *Server) createPollingTarget(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	in, err := decodePollingTargetInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	target := inputToTarget(in, "")
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
	in, err := decodePollingTargetInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	current, updErr := s.pollingTargets.Update(r.Context(), inputToTarget(in, id))
	if updErr != nil {
		logger.ErrorContext(r.Context(), "update polling_target failed", "id", id, "error", updErr)
		writePollingError(w, updErr, "Failed to update target")
		return
	}
	writeJSON(w, r, encodePollingTarget(current))
}

func (s *Server) deletePollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	if err := s.pollingTargets.Delete(r.Context(), id); err != nil {
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
// used for Update.
func inputToTarget(in *pollingTargetInput, id string) *polling.Target {
	return &polling.Target{
		ID:              id,
		ClientID:        in.ClientID,
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
