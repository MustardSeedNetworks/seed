package api

// TFTP session endpoint (P-D4):
//
//   GET    /api/v1/tftp/session   the running session, or running false
//   POST   /api/v1/tftp/session   start one: {interface, allowUpload}
//   DELETE /api/v1/tftp/session   stop it
//
// TFTP has no authentication: whoever starts a session decides what every host
// on that segment may read, and with uploads ticked, write. So starting and
// stopping are admin only, the route is Pro with configuration backup
// (compliance_advanced), nothing starts with the daemon, and a session never
// survives a restart. Starts, stops and every transfer go to the audit log.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/paths"
	"github.com/MustardSeedNetworks/seed/internal/tftp"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

const (
	tftpFeature       = "compliance_advanced"
	tftpAuditResource = "tftp"
	tftpAuditStart    = "start"
	tftpAuditStop     = "stop"
	// tftpAuditTimeout bounds an audit write made from a transfer goroutine,
	// which has no request context to inherit a deadline from.
	tftpAuditTimeout = 5 * time.Second
)

func (s *Server) tftpRoutes() []route.Route {
	return []route.Route{{
		Path:    APIVersionPrefix + "/tftp/session",
		Handler: s.handleTFTPSession,
		Methods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
		Scope:   roles.Admin,
		Feature: tftpFeature,
		Auth:    true,
		CSRF:    true,
	}}
}

type tftpStartInput struct {
	Interface   string `json:"interface"`
	AllowUpload bool   `json:"allowUpload"`
}

// initTFTP builds the session manager over <data dir>/tftp. It starts nothing.
func (s *Server) initTFTP() {
	s.tftpSessions = tftp.NewManager(tftp.Config{
		Dir:        filepath.Join(paths.Resolve(paths.ModeAuto).DataDir, "tftp"),
		OnTransfer: s.auditTFTPTransfer,
		OnStop:     s.auditTFTPStop,
		Logger:     logging.GetLogger(),
	})
}

func (s *Server) handleTFTPSession(w http.ResponseWriter, r *http.Request) {
	if s.tftpSessions == nil {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "TFTP is unavailable")
		return
	}
	switch r.Method {
	case http.MethodPost:
		s.startTFTP(w, r)
	case http.MethodDelete:
		stopping := s.tftpSessions.Status()
		if err := s.tftpSessions.Stop(); err != nil {
			writeError(w, r, http.StatusConflict, ErrCodeConflict, err.Error())
			return
		}
		s.auditTFTP(r.Context(), &database.AuditLogEntry{
			Action:     tftpAuditStop,
			User:       usernameFromContext(r),
			ResourceID: stopping.Interface,
			IPAddress:  GetClientIP(r),
			UserAgent:  r.UserAgent(),
		}, map[string]string{"reason": string(tftp.StopOperator)})
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, s.tftpSessions.Status())
	default:
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, s.tftpSessions.Status())
	}
}

func (s *Server) startTFTP(w http.ResponseWriter, r *http.Request) {
	var in tftpStartInput
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	if err := validation.ValidateInterface(in.Interface); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	status, err := s.tftpSessions.Start(tftp.Options{Interface: in.Interface, AllowUpload: in.AllowUpload})
	switch {
	case errors.Is(err, tftp.ErrRunning):
		writeError(w, r, http.StatusConflict, ErrCodeConflict, err.Error())
		return
	case errors.Is(err, tftp.ErrNoInterface), errors.Is(err, tftp.ErrNoIPv4):
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	case errors.Is(err, tftp.ErrBind):
		logging.FromContext(r.Context()).WarnContext(r.Context(), "tftp bind failed", "error", err)
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			"UDP port 69 on that interface is in use or not permitted")
		return
	case err != nil:
		logging.FromContext(r.Context()).ErrorContext(r.Context(), "tftp start failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "TFTP could not start")
		return
	}
	s.auditTFTP(r.Context(), &database.AuditLogEntry{
		Action:     tftpAuditStart,
		User:       usernameFromContext(r),
		ResourceID: status.Interface,
		IPAddress:  GetClientIP(r),
		UserAgent:  r.UserAgent(),
	}, status)
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusCreated, status)
}

// auditTFTPTransfer records one transfer. TFTP carries no identity, so the
// device's address stands in for the user.
func (s *Server) auditTFTPTransfer(t tftp.Transfer) {
	detail := map[string]any{"bytes": t.Bytes}
	if t.Err != nil {
		detail["error"] = t.Err.Error()
	}
	ctx, cancel := context.WithTimeout(context.Background(), tftpAuditTimeout)
	defer cancel()
	s.auditTFTP(ctx, &database.AuditLogEntry{
		Action:     string(t.Direction),
		ResourceID: t.Filename,
		IPAddress:  t.Remote,
	}, detail)
}

// auditTFTPStop records the stops no admin asked for; the DELETE handler
// records an operator's stop with their name.
func (s *Server) auditTFTPStop(st tftp.Status, reason tftp.StopReason) {
	if reason == tftp.StopOperator {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), tftpAuditTimeout)
	defer cancel()
	s.auditTFTP(ctx, &database.AuditLogEntry{Action: tftpAuditStop, ResourceID: st.Interface},
		map[string]string{"reason": string(reason)})
}

func (s *Server) auditTFTP(ctx context.Context, entry *database.AuditLogEntry, detail any) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "tftp audit", "event", "tftp."+entry.Action, "resource", entry.ResourceID,
		"remote", entry.IPAddress, "user", entry.User)
	db := s.db()
	if db == nil {
		return
	}
	body, err := json.Marshal(detail)
	if err != nil {
		logger.WarnContext(ctx, "tftp audit encode failed", "error", err)
		return
	}
	entry.ResourceType = tftpAuditResource
	entry.NewValueJSON = string(body)
	if err = db.RecordAuditLog(ctx, entry); err != nil {
		logger.WarnContext(ctx, "tftp audit write failed", "error", err)
	}
}
