package api

// Windows inventory endpoints (P-D11, #3506, ADR-0034):
//
//   GET    /api/v1/wmi-targets                          list
//   POST   /api/v1/wmi-targets                          create
//   GET    /api/v1/wmi-targets/{targetId}               fetch one
//   PUT    /api/v1/wmi-targets/{targetId}               full update
//   DELETE /api/v1/wmi-targets/{targetId}               delete, inventory included
//   PUT    /api/v1/wmi-targets/{targetId}/certificate   pin the pending certificate
//   DELETE /api/v1/wmi-targets/{targetId}/certificate   forget the pinned certificate
//   GET    /api/v1/wmi-targets/{targetId}/inventory     the last inventory collected
//   POST   /api/v1/wmi-runs                             start a collection (a job)
//
// Every route is Pro (server_monitoring, which already gates host monitoring
// over HOST-RESOURCES-MIB), operator+ on its mutating methods, and scoped to
// the client on the caller's session claim, as the credential vault is.
//
// A run is a job so the UI can follow it on /jobs/events. It is started here
// rather than through POST /jobs because /jobs is tenant-blind: the job's
// client comes from this request's session.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

const (
	wmiTargetsPath = APIVersionPrefix + "/wmi-targets"
	wmiRunsPath    = APIVersionPrefix + "/wmi-runs"
	wmiJobKind     = "wmi-inventory"
	wmiFeature     = "server_monitoring"
)

var wmiTargetIDPattern = regexp.MustCompile(`^wmt-[0-9a-f]{12}$`)

// WMITargetRequest is the POST and PUT body. Port defaults to 5986, the WinRM
// HTTPS listener; credentialsId names a winrm credential in the vault.
type WMITargetRequest struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          int    `json:"port,omitempty"`
	CredentialsID string `json:"credentialsId"`
}

// WMITargetResponse is one inventory target. certSha256 is the listener
// certificate an operator confirmed and pendingCertSha256 the one the host
// last presented while none was; both are the uppercase hex SHA-256 that
// PowerShell prints for $cert.GetCertHashString('SHA256'). The last* fields
// are absent until the first collection. Timestamps are RFC 3339 UTC.
type WMITargetResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	CredentialsID     string `json:"credentialsId"`
	CertSHA256        string `json:"certSha256,omitempty"`
	PendingCertSHA256 string `json:"pendingCertSha256,omitempty"`
	LastAttemptAt     string `json:"lastAttemptAt,omitempty"`
	LastStatus        string `json:"lastStatus,omitempty"        jsonschema:"enum=ok,enum=auth_failed,enum=cert_mismatch,enum=cert_unconfirmed,enum=unreachable,enum=query_failed,enum=error"`
	LastError         string `json:"lastError,omitempty"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// WMITargetListResponse is the GET /wmi-targets envelope.
type WMITargetListResponse struct {
	Count   int                 `json:"count"`
	Targets []WMITargetResponse `json:"targets"`
}

// WMICertificateRequest is the certificate PUT body: the pending fingerprint
// the operator was shown and is confirming.
type WMICertificateRequest struct {
	Fingerprint string `json:"fingerprint"`
}

// WMIRunRequest is the run body; no targetIds collects every target.
type WMIRunRequest struct {
	TargetIDs []string `json:"targetIds,omitempty"`
}

// WMIInventoryResponse is a target's last successful inventory and when it was
// collected (RFC 3339 UTC).
type WMIInventoryResponse struct {
	TargetID    string        `json:"targetId"`
	CollectedAt string        `json:"collectedAt"`
	Inventory   wmi.Inventory `json:"inventory"`
}

// wmiJob is the run's job params. It is a Go value, not JSON, so only this
// package can construct one: see the file comment.
type wmiJob struct {
	ClientID  string
	TargetIDs []string
}

func (s *Server) wmiRoutes() []route.Route {
	op := roles.Operator
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/wmi-targets",
			Handler: s.handleWMITargets,
			Methods: []string{http.MethodGet, http.MethodPost},
			Scope:   op,
			Feature: wmiFeature,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/wmi-targets/{targetId}",
			Handler: s.handleWMITargetByID,
			Methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			Scope:   op,
			Feature: wmiFeature,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/wmi-targets/{targetId}/certificate",
			Handler: s.handleWMICertificate,
			Methods: []string{http.MethodPut, http.MethodDelete},
			Scope:   op,
			Feature: wmiFeature,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/wmi-targets/{targetId}/inventory",
			Handler: s.handleWMIInventory,
			Methods: []string{http.MethodGet},
			Feature: wmiFeature,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/wmi-runs",
			Handler: s.handleWMIRun,
			Methods: []string{http.MethodPost},
			Scope:   op,
			Feature: wmiFeature,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
	}
}

func (s *Server) handleWMITargets(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.wmiCaller(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		s.saveWMITarget(w, r, clientID, "")
		return
	}
	list, err := s.wmiInventory.ListTargets(r.Context(), clientID)
	if err != nil {
		writeWMIError(w, r, err)
		return
	}
	out := make([]WMITargetResponse, 0, len(list))
	for _, t := range list {
		out = append(out, encodeWMITarget(t))
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		WMITargetListResponse{Count: len(out), Targets: out})
}

func (s *Server) handleWMITargetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := wmiTargetID(w, r)
	if !ok {
		return
	}
	clientID, ok := s.wmiCaller(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodPut:
		s.saveWMITarget(w, r, clientID, id)
	case http.MethodDelete:
		if err := s.wmiInventory.DeleteTarget(r.Context(), clientID, id); err != nil {
			writeWMIError(w, r, err)
			return
		}
		s.auditWMI(r, "wmi.target.delete", "Windows inventory target deleted", "target_id", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		t, err := s.wmiInventory.GetTarget(r.Context(), clientID, id)
		if err != nil {
			writeWMIError(w, r, err)
			return
		}
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, encodeWMITarget(t))
	}
}

func (s *Server) saveWMITarget(w http.ResponseWriter, r *http.Request, clientID, id string) {
	var in WMITargetRequest
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	t, err := s.wmiInventory.SaveTarget(r.Context(), wmi.TargetInput{
		ID:            id,
		ClientID:      clientID,
		Name:          in.Name,
		Host:          in.Host,
		Port:          in.Port,
		CredentialsID: in.CredentialsID,
	})
	if err != nil {
		writeWMIError(w, r, err)
		return
	}
	s.auditWMI(r, "wmi.target.save", "Windows inventory target saved",
		"target_id", t.ID, "host", t.Host, "port", t.Port)
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
		w.Header().Set("Location", wmiTargetsPath+"/"+t.ID)
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), status, encodeWMITarget(t))
}

func (s *Server) handleWMICertificate(w http.ResponseWriter, r *http.Request) {
	id, ok := wmiTargetID(w, r)
	if !ok {
		return
	}
	clientID, ok := s.wmiCaller(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodDelete {
		if err := s.wmiInventory.ClearCert(r.Context(), clientID, id); err != nil {
			writeWMIError(w, r, err)
			return
		}
		s.auditWMI(r, "wmi.certificate.clear", "Windows inventory certificate cleared", "target_id", id)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var in WMICertificateRequest
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	if err := s.wmiInventory.ConfirmCert(r.Context(), clientID, id, in.Fingerprint); err != nil {
		writeWMIError(w, r, err)
		return
	}
	s.auditWMI(r, "wmi.certificate.confirm", "Windows inventory certificate confirmed",
		"target_id", id, "fingerprint", in.Fingerprint)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWMIInventory(w http.ResponseWriter, r *http.Request) {
	id, ok := wmiTargetID(w, r)
	if !ok {
		return
	}
	clientID, ok := s.wmiCaller(w, r)
	if !ok {
		return
	}
	inv, err := s.wmiInventory.GetInventory(r.Context(), clientID, id)
	if err != nil {
		writeWMIError(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, WMIInventoryResponse{
		TargetID:    inv.TargetID,
		CollectedAt: inv.CollectedAt.UTC().Format(time.RFC3339),
		Inventory:   inv.Inventory,
	})
}

func (s *Server) handleWMIRun(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.wmiCaller(w, r)
	if !ok {
		return
	}
	var in WMIRunRequest
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	for _, id := range in.TargetIDs {
		if !wmiTargetIDPattern.MatchString(id) {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "targetIds holds an invalid inventory target id")
			return
		}
	}
	s.submitClientJob(w, r, clientID, wmiJobKind, wmiJob{ClientID: clientID, TargetIDs: in.TargetIDs},
		func(jobID string) {
			s.auditWMI(r, "wmi.run", "Windows inventory run started", "job_id", jobID, "target_ids", in.TargetIDs)
		})
}

func encodeWMITarget(t wmi.Target) WMITargetResponse {
	out := WMITargetResponse{
		ID:                t.ID,
		Name:              t.Name,
		Host:              t.Host,
		Port:              t.Port,
		CredentialsID:     t.CredentialsID,
		CertSHA256:        t.CertSHA256,
		PendingCertSHA256: t.PendingCertSHA256,
		LastStatus:        string(t.LastStatus),
		LastError:         t.LastError,
		CreatedAt:         t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if !t.LastAttemptAt.IsZero() {
		out.LastAttemptAt = t.LastAttemptAt.UTC().Format(time.RFC3339)
	}
	return out
}

// wmiTargetID reads the path's target id. One that cannot be a target id is
// answered 404, as an unknown one is, and never reaches an audit line.
func wmiTargetID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("targetId")
	if !wmiTargetIDPattern.MatchString(id) {
		writeWMIError(w, r, wmi.ErrNotFound)
		return "", false
	}
	return id, true
}

// auditWMI writes the security audit line for a Windows inventory action.
func (s *Server) auditWMI(r *http.Request, event, msg string, attrs ...any) {
	attrs = append(attrs,
		"username", usernameFromContext(r),
		"client_ip", s.getClientIP(r),
		"event", event)
	logging.FromContext(r.Context()).InfoContext(r.Context(), msg, attrs...)
}

// wmiCaller resolves the caller's client and answers 503 when the use-case is
// not wired (no config, so no keyring to decrypt passwords with).
func (s *Server) wmiCaller(w http.ResponseWriter, r *http.Request) (string, bool) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return "", false
	}
	if s.wmiInventory == nil {
		writeWMIError(w, r, wmi.ErrUnavailable)
		return "", false
	}
	return clientID, true
}

func writeWMIError(w http.ResponseWriter, r *http.Request, err error) {
	var ve wmi.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, ve.Msg)
	case errors.Is(err, wmi.ErrNotFound):
		writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Not found")
	case errors.Is(err, wmi.ErrUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			"Windows inventory store unavailable")
	default:
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"Windows inventory request failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to process Windows inventory request")
	}
}

// newWMIHandler returns the job handler for a run. Params that are not a
// wmiJob came through POST /jobs, which cannot say whose hosts to inventory.
func newWMIHandler(svc func() *wmi.Service) jobs.Handler {
	return func(ctx context.Context, params any, report func(float64)) (any, error) {
		p, ok := params.(wmiJob)
		if !ok {
			return nil, fmt.Errorf("start a Windows inventory run with POST %s", wmiRunsPath)
		}
		inventory := svc()
		if inventory == nil {
			return nil, wmi.ErrUnavailable
		}
		return inventory.Run(ctx, p.ClientID, p.TargetIDs, report)
	}
}

func (s *Server) registerWMIKind() {
	if err := s.jobsRunner().Register(wmiJobKind,
		newWMIHandler(func() *wmi.Service { return s.wmiInventory })); err != nil {
		logging.GetLogger().Error("failed to register Windows inventory job kind", "error", err)
	}
}
