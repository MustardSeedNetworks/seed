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
// survives a restart. Starts, stops and every completed transfer go to the
// audit log; failed and refused requests are aggregated per device, so a host
// hammering the port cannot flood the log.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"path/filepath"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/paths"
	"github.com/MustardSeedNetworks/seed/internal/tftp"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

const (
	tftpFeature    = "compliance_advanced"
	tftpAuditStart = "start"
	tftpAuditStop  = "stop"
	// tftpAuditTimeout bounds an audit write made from a transfer goroutine,
	// which has no request context to inherit a deadline from.
	tftpAuditTimeout = 5 * time.Second
	// tftpFailureWindow is how often one device's failed or refused requests
	// may write an audit row, and tftpMaxFailureRows how many such rows all
	// devices together may write in it. The rest are counted and logged.
	tftpFailureWindow  = time.Minute
	tftpMaxFailureRows = 32
	// tftpMaxAllowFrom bounds the prefixes an admin may list.
	tftpMaxAllowFrom = 16
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
	Interface   string   `json:"interface"`
	AllowUpload bool     `json:"allowUpload"`
	AllowFrom   []string `json:"allowFrom"`
}

// initTFTP builds the session manager over <data dir>/tftp, with uploads
// received into <data dir>/tftp-incoming. It starts nothing.
func (s *Server) initTFTP() {
	dataDir := paths.Resolve(paths.ModeAuto).DataDir
	s.tftp.audit = app.NewTFTPAudit(s.persistence)
	s.tftp.sessions = tftp.NewManager(tftp.Config{
		Dir:        filepath.Join(dataDir, "tftp"),
		UploadDir:  filepath.Join(dataDir, "tftp-incoming"),
		OnTransfer: s.auditTFTPTransfer,
		OnStop:     s.auditTFTPStop,
		Logger:     logging.GetLogger(),
	})
}

func (s *Server) handleTFTPSession(w http.ResponseWriter, r *http.Request) {
	if s.tftp.sessions == nil {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "TFTP is unavailable")
		return
	}
	switch r.Method {
	case http.MethodPost:
		s.startTFTP(w, r)
	case http.MethodDelete:
		stopping := s.tftp.sessions.Status()
		if err := s.tftp.sessions.Stop(); err != nil {
			writeError(w, r, http.StatusConflict, ErrCodeConflict, err.Error())
			return
		}
		s.auditTFTP(r.Context(), tftp.AuditEvent{
			Action:    tftpAuditStop,
			User:      usernameFromContext(r),
			Resource:  stopping.Interface,
			Remote:    GetClientIP(r),
			UserAgent: r.UserAgent(),
			Detail:    map[string]string{"reason": string(tftp.StopOperator)},
		})
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, s.tftp.sessions.Status())
	default:
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, s.tftp.sessions.Status())
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
	allowFrom, err := parseTFTPAllowFrom(in.AllowFrom)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	status, err := s.tftp.sessions.Start(tftp.Options{
		Interface:   in.Interface,
		AllowUpload: in.AllowUpload,
		AllowFrom:   allowFrom,
	})
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
	s.auditTFTP(r.Context(), tftp.AuditEvent{
		Action:    tftpAuditStart,
		User:      usernameFromContext(r),
		Resource:  status.Interface,
		Remote:    GetClientIP(r),
		UserAgent: r.UserAgent(),
		Detail:    status,
	})
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusCreated, status)
}

// parseTFTPAllowFrom reads the admin's source prefixes. TFTP is served on
// IPv4 only, so an IPv6 prefix could never match and is refused.
func parseTFTPAllowFrom(in []string) ([]netip.Prefix, error) {
	if len(in) > tftpMaxAllowFrom {
		return nil, fmt.Errorf("allowFrom lists %d prefixes, at most %d are allowed", len(in), tftpMaxAllowFrom)
	}
	out := make([]netip.Prefix, 0, len(in))
	for _, raw := range in {
		p, err := netip.ParsePrefix(raw)
		if err != nil || !p.Addr().Is4() {
			return nil, fmt.Errorf("allowFrom %q is not an IPv4 prefix such as 192.0.2.0/24", raw)
		}
		out = append(out, p.Masked())
	}
	return out, nil
}

// tftpService holds the admin-started TFTP session and its audit trail.
type tftpService struct {
	sessions *tftp.Manager
	audit    func(context.Context, tftp.AuditEvent) error
	failures tftpFailureGate // one audit row per failing device per window
}

// tftpFailureGate decides which failed or refused transfers reach the audit
// log: in each window, the first from a device and at most
// tftpMaxFailureRows in all. A refused request's source may be spoofed, so the
// gate remembers only devices it wrote a row for, which bounds its memory by
// that cap however many sources a flood cycles through.
type tftpFailureGate struct {
	mu         sync.Mutex
	window     time.Time
	logged     map[string]struct{}
	suppressed int
}

// admit reports whether a failure from remote at now should be audited. When a
// window closes with failures suppressed, it also returns their count, for the
// caller to log.
func (g *tftpFailureGate) admit(remote string, now time.Time) (bool, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	closedSuppressed := 0
	if g.logged == nil || now.Sub(g.window) >= tftpFailureWindow {
		closedSuppressed = g.suppressed
		g.window, g.logged, g.suppressed = now, map[string]struct{}{}, 0
	}
	if _, seen := g.logged[remote]; seen || len(g.logged) >= tftpMaxFailureRows {
		g.suppressed++
		return false, closedSuppressed
	}
	g.logged[remote] = struct{}{}
	return true, closedSuppressed
}

// reset forgets the window, so a new session starts with a clean count.
func (g *tftpFailureGate) reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.logged, g.suppressed = nil, 0
}

// auditTFTPTransfer records one transfer. TFTP carries no identity, so the
// device's address stands in for the user. A completed transfer is always
// recorded; failures go through the per-device gate.
func (s *Server) auditTFTPTransfer(t tftp.Transfer) {
	detail := map[string]any{"bytes": t.Bytes}
	if t.Err != nil {
		audit, suppressed := s.tftp.failures.admit(t.Remote, time.Now())
		if suppressed > 0 {
			logging.GetLogger().Warn("tftp failures not audited", "event", "tftp.suppressed",
				"count", suppressed, "window", tftpFailureWindow.String())
		}
		if !audit {
			return
		}
		detail["error"] = t.Err.Error()
	}
	ctx, cancel := context.WithTimeout(context.Background(), tftpAuditTimeout)
	defer cancel()
	s.auditTFTP(ctx, tftp.AuditEvent{
		Action:   string(t.Direction),
		Resource: t.Filename,
		Remote:   t.Remote,
		Detail:   detail,
	})
}

// auditTFTPStop records the stops no admin asked for; the DELETE handler
// records an operator's stop with their name.
func (s *Server) auditTFTPStop(st tftp.Status, reason tftp.StopReason) {
	s.tftp.failures.reset()
	if reason == tftp.StopOperator {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), tftpAuditTimeout)
	defer cancel()
	s.auditTFTP(ctx, tftp.AuditEvent{
		Action:   tftpAuditStop,
		Resource: st.Interface,
		Detail:   map[string]string{"reason": string(reason)},
	})
}

func (s *Server) auditTFTP(ctx context.Context, e tftp.AuditEvent) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "tftp audit", "event", "tftp."+e.Action, "resource", e.Resource,
		"remote", e.Remote, "user", e.User)
	if s.tftp.audit == nil {
		return
	}
	if err := s.tftp.audit(ctx, e); err != nil {
		logger.WarnContext(ctx, "tftp audit write failed", "error", err)
	}
}
