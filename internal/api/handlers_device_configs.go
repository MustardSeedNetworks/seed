package api

// Device configuration backup endpoints (P-D1, #348):
//
//   GET    /api/v1/device-config-targets                 list
//   POST   /api/v1/device-config-targets                 create
//   GET    /api/v1/device-config-targets/{id}            fetch one
//   PUT    /api/v1/device-config-targets/{id}            full update
//   DELETE /api/v1/device-config-targets/{id}            delete, history included
//   DELETE /api/v1/device-config-targets/{id}/host-key   forget the pinned key
//   POST   /api/v1/device-configs/run                    start a run (a job)
//   GET    /api/v1/device-configs?targetId=&limit=       a target's attempts
//   GET    /api/v1/device-configs/{id}                   one attempt, with config
//
// Every route is Pro (compliance_advanced, the feature #348 was filed under),
// operator+ on its mutating methods, and scoped to the client on the caller's
// session claim, as the credential vault is.
//
// A run is a job so the UI can follow it on /jobs/events. It is started here
// rather than through POST /jobs because /jobs is tenant-blind: the job's
// client comes from this request's session, and a body field naming one would
// let a caller back up another client's devices.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

const (
	deviceConfigTargetsPath       = APIVersionPrefix + "/device-config-targets"
	deviceConfigTargetsPathPrefix = deviceConfigTargetsPath + "/"
	deviceConfigsPath             = APIVersionPrefix + "/device-configs"
	deviceConfigsPathPrefix       = deviceConfigsPath + "/"
	deviceConfigRunPath           = deviceConfigsPath + "/run"
	hostKeySuffix                 = "/host-key"

	deviceConfigJobKind = "device-config-backup"
	deviceConfigFeature = "compliance_advanced"

	deviceConfigListDefault = 50
	deviceConfigListMax     = 500
)

var (
	deviceConfigTargetIDPattern = regexp.MustCompile(`^dct-[0-9a-f]{12}$`)
	deviceConfigBackupIDPattern = regexp.MustCompile(`^dcb-[0-9a-f]{12}$`)
)

func (s *Server) deviceConfigRoutes() []route.Route {
	op := roles.Operator
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/device-config-targets",
			Handler: s.handleDeviceConfigTargets,
			Methods: []string{http.MethodGet, http.MethodPost},
			Scope:   op,
			Feature: deviceConfigFeature,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-config-targets/",
			Handler: s.handleDeviceConfigTargetByID,
			Methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			Scope:   op,
			Feature: deviceConfigFeature,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-configs/run",
			Handler: s.handleDeviceConfigRun,
			Methods: []string{http.MethodPost},
			Scope:   op,
			Feature: deviceConfigFeature,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-configs",
			Handler: s.handleDeviceConfigBackups,
			Methods: []string{http.MethodGet},
			Feature: deviceConfigFeature,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-configs/",
			Handler: s.handleDeviceConfigBackupByID,
			Methods: []string{http.MethodGet},
			Feature: deviceConfigFeature,
			Auth:    true,
		},
	}
}

// deviceConfigTargetInput is the POST/PUT body. Port defaults to 22.
type deviceConfigTargetInput struct {
	Name          string                `json:"name"`
	Host          string                `json:"host"`
	Port          int                   `json:"port,omitempty"`
	Platform      deviceconfig.Platform `json:"platform"`
	CredentialsID string                `json:"credentialsId"`
}

// deviceConfigRunInput is the run body; no targetIds backs up every target.
type deviceConfigRunInput struct {
	TargetIDs []string `json:"targetIds,omitempty"`
}

// deviceConfigJob is the run's job params. It is a Go value, not JSON, so only
// this package can construct one: see the package comment.
type deviceConfigJob struct {
	ClientID  string
	TargetIDs []string
}

func (s *Server) handleDeviceConfigTargets(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		s.saveDeviceConfigTarget(w, r, clientID, "")
		return
	}
	list, err := s.deviceConfigs.ListTargets(r.Context(), clientID)
	if err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, map[string]any{
		jsonKeyCount: len(list),
		"targets":    list,
	})
}

func (s *Server) handleDeviceConfigTargetByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, deviceConfigTargetsPathPrefix)
	id, hostKey := strings.CutSuffix(rest, hostKeySuffix)
	if !deviceConfigTargetIDPattern.MatchString(id) || (hostKey && r.Method != http.MethodDelete) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid backup target id")
		return
	}
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	switch {
	case hostKey:
		if err := s.deviceConfigs.ClearHostKey(r.Context(), clientID, id); err != nil {
			writeDeviceConfigError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodGet:
		t, err := s.deviceConfigs.GetTarget(r.Context(), clientID, id)
		if err != nil {
			writeDeviceConfigError(w, r, err)
			return
		}
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, t)
	case r.Method == http.MethodPut:
		s.saveDeviceConfigTarget(w, r, clientID, id)
	default:
		if err := s.deviceConfigs.DeleteTarget(r.Context(), clientID, id); err != nil {
			writeDeviceConfigError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) saveDeviceConfigTarget(w http.ResponseWriter, r *http.Request, clientID, id string) {
	var in deviceConfigTargetInput
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	t, err := s.deviceConfigs.SaveTarget(r.Context(), deviceconfig.TargetInput{
		ID:            id,
		ClientID:      clientID,
		Name:          in.Name,
		Host:          in.Host,
		Port:          in.Port,
		Platform:      in.Platform,
		CredentialsID: in.CredentialsID,
	})
	if err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), status, t)
}

func (s *Server) handleDeviceConfigRun(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	var in deviceConfigRunInput
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	for _, id := range in.TargetIDs {
		if !deviceConfigTargetIDPattern.MatchString(id) {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "targetIds holds an invalid backup target id")
			return
		}
	}
	logger := logging.FromContext(r.Context())
	id, err := s.jobsRunner().Submit(deviceConfigJobKind, deviceConfigJob{ClientID: clientID, TargetIDs: in.TargetIDs})
	if err != nil {
		writeJobError(w, logger, err)
		return
	}
	j, _ := s.jobsRunner().Get(id)
	w.Header().Set("Location", jobsPathPrefix+id)
	sendJSONResponse(w, logger, http.StatusCreated, toJobResponse(j))
}

func (s *Server) handleDeviceConfigBackups(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	targetID := q.Get("targetId")
	if !deviceConfigTargetIDPattern.MatchString(targetID) {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "targetId is required")
		return
	}
	limit := deviceConfigListDefault
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > deviceConfigListMax {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation,
				fmt.Sprintf("limit must be between 1 and %d", deviceConfigListMax))
			return
		}
		limit = n
	}
	list, err := s.deviceConfigs.ListBackups(r.Context(), clientID, targetID, limit)
	if err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, map[string]any{
		jsonKeyCount: len(list),
		"backups":    list,
	})
}

func (s *Server) handleDeviceConfigBackupByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, deviceConfigsPathPrefix)
	if !deviceConfigBackupIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid backup id")
		return
	}
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	b, err := s.deviceConfigs.GetBackup(r.Context(), clientID, id)
	if err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, b)
}

// deviceConfigCaller resolves the caller's client and answers 503 when the
// use-case is not wired (no config, so no keyring to decrypt passwords with).
func (s *Server) deviceConfigCaller(w http.ResponseWriter, r *http.Request) (string, bool) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return "", false
	}
	if s.deviceConfigs == nil {
		writeDeviceConfigError(w, r, deviceconfig.ErrUnavailable)
		return "", false
	}
	return clientID, true
}

func writeDeviceConfigError(w http.ResponseWriter, r *http.Request, err error) {
	var ve deviceconfig.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, ve.Msg)
	case errors.Is(err, deviceconfig.ErrNotFound):
		writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Not found")
	case errors.Is(err, deviceconfig.ErrUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			"Configuration backup store unavailable")
	default:
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"configuration backup request failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to process configuration backup request")
	}
}

// newDeviceConfigHandler returns the job handler for a run. Params that are
// not a deviceConfigJob came through POST /jobs, which cannot say whose
// devices to back up.
func newDeviceConfigHandler(svc func() *deviceconfig.Service) jobs.Handler {
	return func(ctx context.Context, params any, report func(float64)) (any, error) {
		p, ok := params.(deviceConfigJob)
		if !ok {
			return nil, fmt.Errorf("start a configuration backup with POST %s", deviceConfigRunPath)
		}
		backups := svc()
		if backups == nil {
			return nil, deviceconfig.ErrUnavailable
		}
		return backups.Run(ctx, p.ClientID, p.TargetIDs, report)
	}
}

func (s *Server) registerDeviceConfigKind() {
	if err := s.jobsRunner().Register(deviceConfigJobKind,
		newDeviceConfigHandler(func() *deviceconfig.Service { return s.deviceConfigs })); err != nil {
		logging.GetLogger().Error("failed to register config-backup job kind", "error", err)
	}
}
