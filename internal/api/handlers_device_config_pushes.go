package api

// Configuration push endpoints, plan half (P-D3, #3499, ADR-0035):
//
//   PUT    /api/v1/device-config-targets/{targetId}/push  enable pushing with a credential
//   DELETE /api/v1/device-config-targets/{targetId}/push  turn pushing off
//   POST   /api/v1/device-config-pushes                   plan a push (a job; a dry run)
//   GET    /api/v1/device-config-pushes?targetId=&limit=  a target's plans
//   GET    /api/v1/device-config-pushes/{planId}          one plan, with candidate and diff
//   POST   /api/v1/device-config-pushes/{planId}/apply    apply a plan (a job; changes the device)
//
// Every route is Pro (config_push). Enabling pushing, planning and applying
// are admin only. A personal access token may plan, since a plan changes
// nothing on a device, but never apply, so a leaked token can prepare a change
// and not make one. Reading plans is operator+, as reading backups is: a diff
// carries configuration text.
//
// A plan and an apply are jobs, started here for the reason POST
// /device-configs/run is: the job's client comes from this request's session.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

const (
	deviceConfigPushesPath = APIVersionPrefix + "/device-config-pushes"

	deviceConfigPushPlanJobKind  = "device-config-push-plan"
	deviceConfigPushApplyJobKind = "device-config-push-apply"
	configPushFeature            = "config_push"
)

var deviceConfigPlanIDPattern = regexp.MustCompile(`^dcp-[0-9a-f]{12}$`)

func (s *Server) deviceConfigPushRoutes() []route.Route {
	admin := roles.Admin
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/device-config-targets/{targetId}/push",
			Handler: s.handleDeviceConfigTargetPush,
			Methods: []string{http.MethodPut, http.MethodDelete},
			Scope:   admin,
			Feature: configPushFeature,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-config-pushes",
			Handler: s.handleDeviceConfigPushes,
			Methods: []string{http.MethodGet, http.MethodPost},
			Scope:   admin,
			Feature: configPushFeature,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-config-pushes/{planId}",
			Handler: s.handleDeviceConfigPushByID,
			Methods: []string{http.MethodGet},
			Feature: configPushFeature,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-config-pushes/{planId}/apply",
			Handler: s.handleDeviceConfigPushApply,
			Methods: []string{http.MethodPost},
			Scope:   admin,
			Feature: configPushFeature,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
	}
}

// pushCredentialsInput is the push PUT body: the SSH credential a push logs
// in with, which must not be the target's backup credential.
type pushCredentialsInput struct {
	CredentialsID string `json:"credentialsId"`
}

// pushPlanInput is the plan POST body.
type pushPlanInput struct {
	TargetID  string                `json:"targetId"`
	Mode      deviceconfig.PushMode `json:"mode"`
	Candidate string                `json:"candidate"`
}

// pushApplyInput is the apply POST body: the hash of the diff the admin was
// shown, so what is applied is what was read.
type pushApplyInput struct {
	PlanSHA256 string `json:"planSha256"`
}

// pushApplyJob is the apply's job params, a Go value like pushPlanJob.
type pushApplyJob struct {
	Request deviceconfig.ApplyRequest
}

// pushPlanJob is the plan's job params, a Go value for the reason
// deviceConfigJob is one.
type pushPlanJob struct {
	Request deviceconfig.PlanRequest
}

func (s *Server) handleDeviceConfigTargetPush(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("targetId")
	if !deviceConfigTargetIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid backup target id")
		return
	}
	clientID, ok := s.deviceConfigPushCaller(w, r)
	if !ok {
		return
	}
	var credentialsID string
	if r.Method == http.MethodPut {
		var in pushCredentialsInput
		if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
			return
		}
		if in.CredentialsID == "" {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "credentialsId is required")
			return
		}
		credentialsID = in.CredentialsID
	}
	if err := s.deviceConfigs.SetPushCredentials(r.Context(), clientID, id, credentialsID); err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	if credentialsID == "" {
		s.auditDeviceConfig(r, "device_config.push.disable", "Configuration push disabled", "target_id", id)
	} else {
		s.auditDeviceConfig(r, "device_config.push.enable", "Configuration push enabled",
			"target_id", id, "credentials_id", credentialsID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeviceConfigPushes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.listDeviceConfigPushPlans(w, r)
		return
	}
	clientID, ok := s.deviceConfigPushCaller(w, r)
	if !ok {
		return
	}
	var in pushPlanInput
	if !decodeJSONStrict(w, r, &in, MaxBodySizeConfigPush) {
		return
	}
	if !deviceConfigTargetIDPattern.MatchString(in.TargetID) {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "targetId is required")
		return
	}
	req := deviceconfig.PlanRequest{
		ClientID:  clientID,
		TargetID:  in.TargetID,
		Mode:      in.Mode,
		Candidate: in.Candidate,
		User:      usernameFromContext(r),
		Remote:    s.getClientIP(r),
		UserAgent: r.UserAgent(),
	}
	if err := s.deviceConfigPushes.Check(r.Context(), req); err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	s.submitClientJob(w, r, clientID, deviceConfigPushPlanJobKind, pushPlanJob{Request: req}, func(jobID string) {
		s.auditDeviceConfig(r, "device_config.push.plan", "Configuration push plan started",
			"job_id", jobID, "target_id", in.TargetID, "mode", in.Mode)
	})
}

func (s *Server) handleDeviceConfigPushApply(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("planId")
	if !deviceConfigPlanIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid push plan id")
		return
	}
	if auth.IsAPITokenAuth(r.Context()) {
		// A token's credential scope is what is short, hence reason=scope.
		logging.FromContext(r.Context()).WarnContext(r.Context(), "Forbidden: a token cannot apply a push",
			"event", "auth.forbidden",
			"reason", "scope",
			"username", usernameFromContext(r),
			"client_ip", GetClientIP(r),
			"path", singleLogLine(r.URL.Path),
			"method", r.Method,
		)
		writeError(w, r, http.StatusForbidden, ErrCodeForbidden,
			"A personal access token cannot apply a configuration push; sign in to apply")
		return
	}
	clientID, ok := s.deviceConfigPushCaller(w, r)
	if !ok {
		return
	}
	var in pushApplyInput
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	req := deviceconfig.ApplyRequest{
		ClientID:   clientID,
		PlanID:     id,
		DiffSHA256: in.PlanSHA256,
		User:       usernameFromContext(r),
		Remote:     s.getClientIP(r),
		UserAgent:  r.UserAgent(),
	}
	if err := s.deviceConfigPushes.CheckApply(r.Context(), req); err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	s.submitClientJob(w, r, clientID, deviceConfigPushApplyJobKind, pushApplyJob{Request: req}, func(jobID string) {
		s.auditDeviceConfig(r, "device_config.push.apply", "Configuration push apply started",
			"job_id", jobID, "plan_id", id)
	})
}

func (s *Server) listDeviceConfigPushPlans(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, roles.Operator) {
		return
	}
	clientID, ok := s.deviceConfigPushCaller(w, r)
	if !ok {
		return
	}
	targetID, limit, ok := deviceConfigListQuery(w, r)
	if !ok {
		return
	}
	list, err := s.deviceConfigPushes.ListPlans(r.Context(), clientID, targetID, limit)
	if err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, map[string]any{
		jsonKeyCount: len(list),
		"plans":      list,
	})
}

func (s *Server) handleDeviceConfigPushByID(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, roles.Operator) {
		return
	}
	id := r.PathValue("planId")
	if !deviceConfigPlanIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid push plan id")
		return
	}
	clientID, ok := s.deviceConfigPushCaller(w, r)
	if !ok {
		return
	}
	plan, err := s.deviceConfigPushes.GetPlan(r.Context(), clientID, id)
	if err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	s.auditDeviceConfig(r, "device_config.push.read", "Configuration push plan read",
		"plan_id", id, "target_id", plan.TargetID)
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, plan)
}

// deviceConfigPushCaller is deviceConfigCaller for the push routes, which also
// need the push use-case wired.
func (s *Server) deviceConfigPushCaller(w http.ResponseWriter, r *http.Request) (string, bool) {
	clientID, ok := s.deviceConfigCaller(w, r)
	if !ok {
		return "", false
	}
	if s.deviceConfigPushes == nil {
		writeDeviceConfigError(w, r, deviceconfig.ErrUnavailable)
		return "", false
	}
	return clientID, true
}

// newDeviceConfigPushPlanHandler returns the job handler for a plan. Params
// that are not a pushPlanJob came through POST /jobs.
func newDeviceConfigPushPlanHandler(svc func() *deviceconfig.Pushes) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		p, ok := params.(pushPlanJob)
		if !ok {
			return nil, fmt.Errorf("plan a configuration push with POST %s", deviceConfigPushesPath)
		}
		pushes := svc()
		if pushes == nil {
			return nil, deviceconfig.ErrUnavailable
		}
		return pushes.Plan(ctx, p.Request)
	}
}

// newDeviceConfigPushApplyHandler returns the job handler for an apply.
// Params that are not a pushApplyJob came through POST /jobs.
func newDeviceConfigPushApplyHandler(svc func() *deviceconfig.Pushes) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		p, ok := params.(pushApplyJob)
		if !ok {
			return nil, fmt.Errorf("apply a configuration push with POST %s/{planId}/apply", deviceConfigPushesPath)
		}
		pushes := svc()
		if pushes == nil {
			return nil, deviceconfig.ErrUnavailable
		}
		return pushes.Apply(ctx, p.Request)
	}
}

func (s *Server) registerDeviceConfigPushKind() {
	pushes := func() *deviceconfig.Pushes { return s.deviceConfigPushes }
	for kind, handler := range map[string]jobs.Handler{
		deviceConfigPushPlanJobKind:  newDeviceConfigPushPlanHandler(pushes),
		deviceConfigPushApplyJobKind: newDeviceConfigPushApplyHandler(pushes),
		deviceImageStageJobKind:      newDeviceImageStageHandler(pushes),
		deviceImageReloadJobKind:     newDeviceImageReloadHandler(pushes),
	} {
		if err := s.jobsRunner().Register(kind, handler); err != nil {
			logging.GetLogger().Error("failed to register config-push job kind", "kind", kind, "error", err)
		}
	}
}
