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

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
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
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	targets, err := db.PollingTargets().List(r.Context(), r.URL.Query().Get("client_id"))
	if err != nil {
		logger.ErrorContext(r.Context(), "list polling_targets failed", "error", err)
		http.Error(w, "Failed to list polling targets", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount: len(targets),
		"targets":    encodePollingTargets(targets),
	})
}

func (s *Server) getPollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	target, err := db.PollingTargets().Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrPollingTargetNotFound) {
			http.Error(w, "Target not found", http.StatusNotFound)
			return
		}
		logger.ErrorContext(r.Context(), "get polling_target failed", "id", id, "error", err)
		http.Error(w, "Failed to load target", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, encodePollingTarget(target))
}

func (s *Server) createPollingTarget(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	in, err := decodePollingTargetInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	target := inputToTarget(in, "")
	if createErr := db.PollingTargets().Create(r.Context(), target); createErr != nil {
		logger.ErrorContext(r.Context(), "create polling_target failed", "error", createErr)
		// Validation errors from the repo are user input; everything
		// else is a server problem. The repo signals validation via
		// errors that don't wrap an underlying sql error — pragmatic
		// classification by prefix keeps us from inventing a typed
		// error hierarchy for one endpoint.
		if strings.HasPrefix(createErr.Error(), "polling_targets:") {
			http.Error(w, createErr.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to create target", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", pollingTargetsPathPrefix+target.ID)
	writeJSON(w, r, encodePollingTarget(target))
}

func (s *Server) updatePollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	in, err := decodePollingTargetInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	target := inputToTarget(in, id)
	if updErr := db.PollingTargets().Update(r.Context(), target); updErr != nil {
		if errors.Is(updErr, database.ErrPollingTargetNotFound) {
			http.Error(w, "Target not found", http.StatusNotFound)
			return
		}
		logger.ErrorContext(r.Context(), "update polling_target failed", "id", id, "error", updErr)
		http.Error(w, "Failed to update target", http.StatusInternalServerError)
		return
	}
	// Echo the freshly-read row so the client sees the audit columns
	// (updated_at) the repo refreshed.
	current, _ := db.PollingTargets().Get(r.Context(), id)
	if current == nil {
		writeJSON(w, r, encodePollingTarget(target))
		return
	}
	writeJSON(w, r, encodePollingTarget(current))
}

func (s *Server) deletePollingTarget(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	if err := db.PollingTargets().Delete(r.Context(), id); err != nil {
		if errors.Is(err, database.ErrPollingTargetNotFound) {
			http.Error(w, "Target not found", http.StatusNotFound)
			return
		}
		logger.ErrorContext(r.Context(), "delete polling_target failed", "id", id, "error", err)
		http.Error(w, "Failed to delete target", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// inputToTarget maps the wire shape into the repo struct. id ==
// "" means "let the repo generate one" (Create); a non-empty id is
// used for Update.
func inputToTarget(in *pollingTargetInput, id string) *database.PollingTarget {
	return &database.PollingTarget{
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

// encodePollingTarget shapes the repo row into JSON. Done
// explicitly so the wire format stays stable when DB columns
// evolve.
func encodePollingTarget(t *database.PollingTarget) map[string]any {
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

func encodePollingTargets(targets []*database.PollingTarget) []map[string]any {
	out := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		out = append(out, encodePollingTarget(t))
	}
	return out
}
