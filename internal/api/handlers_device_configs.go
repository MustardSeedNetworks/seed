package api

// Device configuration backup endpoints (P-D1, #348):
//
//   GET    /api/v1/device-config-targets                 list
//   POST   /api/v1/device-config-targets                 create
//   GET    /api/v1/device-config-targets/{id}            fetch one
//   PUT    /api/v1/device-config-targets/{id}            full update
//   DELETE /api/v1/device-config-targets/{id}            delete, history included
//   PUT    /api/v1/device-config-targets/{id}/host-key   pin the pending key
//   DELETE /api/v1/device-config-targets/{id}/host-key   forget the pinned key
//   POST   /api/v1/device-configs/run                    start a run (a job)
//   GET    /api/v1/device-configs?targetId=&limit=       a target's attempts
//   GET    /api/v1/device-configs/{id}                   one attempt, with config
//   GET    /api/v1/device-configs/{id}/diff              its change from the previous config (P-D2)
//
// Every route is Pro (compliance_advanced, the feature #348 was filed under),
// operator+ on its mutating methods, and scoped to the client on the caller's
// session claim, as the credential vault is. Reading backups is operator+ too:
// a running configuration carries the device's secrets, as a capture does.
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
	diffSuffix                    = "/diff"

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

// deviceConfigTargetInput is the POST/PUT body. Port defaults to the
// platform's: 830 for netconf, 22 otherwise.
type deviceConfigTargetInput struct {
	Name          string                `json:"name"`
	Host          string                `json:"host"`
	Port          int                   `json:"port,omitempty"`
	Platform      deviceconfig.Platform `json:"platform"`
	CredentialsID string                `json:"credentialsId"`
}

// hostKeyConfirmInput is the host-key PUT body: the pending fingerprint the
// operator was shown and is confirming.
type hostKeyConfirmInput struct {
	Fingerprint string `json:"fingerprint"`
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
	if !deviceConfigTargetIDPattern.MatchString(id) || (hostKey && r.Method == http.MethodGet) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid backup target id")
		return
	}
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	switch {
	case hostKey && r.Method == http.MethodPut:
		var in hostKeyConfirmInput
		if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
			return
		}
		if err := s.deviceConfigs.ConfirmHostKey(r.Context(), clientID, id, in.Fingerprint); err != nil {
			writeDeviceConfigError(w, r, err)
			return
		}
		s.auditDeviceConfig(r, "device_config.host_key.confirm", "Configuration backup host key confirmed",
			"target_id", id, "fingerprint", in.Fingerprint)
		w.WriteHeader(http.StatusNoContent)
	case hostKey:
		if err := s.deviceConfigs.ClearHostKey(r.Context(), clientID, id); err != nil {
			writeDeviceConfigError(w, r, err)
			return
		}
		s.auditDeviceConfig(r, "device_config.host_key.clear", "Configuration backup host key cleared",
			"target_id", id)
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
		s.auditDeviceConfig(r, "device_config.target.delete", "Configuration backup target deleted",
			"target_id", id)
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
	s.auditDeviceConfig(r, "device_config.target.save", "Configuration backup target saved",
		"target_id", t.ID, "host", t.Host, "port", t.Port)
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
	s.submitClientJob(w, r, clientID, deviceConfigJobKind,
		deviceConfigJob{ClientID: clientID, TargetIDs: in.TargetIDs}, func(jobID string) {
			s.auditDeviceConfig(r, "device_config.run", "Configuration backup run started",
				"job_id", jobID, "target_ids", in.TargetIDs)
		})
}

func (s *Server) handleDeviceConfigBackups(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, roles.Operator) {
		return
	}
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	targetID, limit, ok := deviceConfigListQuery(w, r)
	if !ok {
		return
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

// deviceConfigListQuery reads a target history's ?targetId=&limit=, answering
// 400 itself when either is malformed.
func deviceConfigListQuery(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	q := r.URL.Query()
	targetID := q.Get("targetId")
	if !deviceConfigTargetIDPattern.MatchString(targetID) {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "targetId is required")
		return "", 0, false
	}
	limit := deviceConfigListDefault
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > deviceConfigListMax {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation,
				fmt.Sprintf("limit must be between 1 and %d", deviceConfigListMax))
			return "", 0, false
		}
		limit = n
	}
	return targetID, limit, true
}

func (s *Server) handleDeviceConfigBackupByID(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, roles.Operator) {
		return
	}
	id, diff := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, deviceConfigsPathPrefix), diffSuffix)
	if !deviceConfigBackupIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid backup id")
		return
	}
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return
	}
	if diff {
		d, err := s.deviceConfigs.Diff(r.Context(), clientID, id)
		if err != nil {
			writeDeviceConfigError(w, r, err)
			return
		}
		s.auditDeviceConfig(r, "device_config.diff", "Configuration backup diff read",
			"backup_id", id, "previous_backup_id", d.PreviousBackupID, "target_id", d.TargetID)
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, d)
		return
	}
	b, err := s.deviceConfigs.GetBackup(r.Context(), clientID, id)
	if err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	s.auditDeviceConfig(r, "device_config.read", "Configuration backup read",
		"backup_id", id, "target_id", b.TargetID)
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, b)
}

// auditDeviceConfig writes the security audit line for a configuration backup
// action, in the shape the Seed configuration backup routes use.
func (s *Server) auditDeviceConfig(r *http.Request, event, msg string, attrs ...any) {
	attrs = append(attrs,
		"username", usernameFromContext(r),
		"client_ip", s.getClientIP(r),
		"event", event)
	logging.FromContext(r.Context()).InfoContext(r.Context(), msg, attrs...)
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
	case errors.Is(err, deviceconfig.ErrPlanSpent):
		writeError(w, r, http.StatusConflict, ErrCodeConflict,
			"The push plan was already applied or has expired; plan again")
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
