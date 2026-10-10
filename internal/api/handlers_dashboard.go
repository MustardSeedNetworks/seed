package api

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// dashboardMaxBody bounds a layout PUT: MaxDashboardWidgets ids of at most 32
// bytes each fit with room to spare.
const dashboardMaxBody = 4 << 10

// DashboardLayout is the caller's own dashboard: GET /api/v1/users/me/dashboard
// returns it and PUT replaces it. Widgets are ids from the UI's catalog, in
// display order. Saved is false when the caller has never saved a layout, so
// the UI can show its default rather than an empty page.
type DashboardLayout struct {
	Widgets []string `json:"widgets"`
	Saved   bool     `json:"saved"`
}

// DashboardLayoutRequest is the body of PUT /api/v1/users/me/dashboard. An
// empty list is a deliberately empty dashboard; leaving widgets out is an
// error.
type DashboardLayoutRequest struct {
	Widgets []string `json:"widgets" validate:"required"`
}

// dashboardRoutes registers the caller's own layout. It is self-scoped, so
// it carries no role gate.
func (s *Server) dashboardRoutes() []route.Route {
	return []route.Route{{
		Path:         APIVersionPrefix + "/users/me/dashboard",
		Handler:      s.handleMyDashboard,
		Methods:      []string{http.MethodGet, http.MethodPut},
		MaxBodyBytes: dashboardMaxBody,
		Auth:         true,
		CSRF:         true,
	}}
}

// handleMyDashboard serves GET and PUT on /api/v1/users/me/dashboard. The path
// names no user: a caller reads and writes only their own layout, so it needs
// no role gate: a viewer arranging their own page changes nothing anyone else
// sees.
func (s *Server) handleMyDashboard(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	t := i18n.FromRequest(r)

	caller := usernameFromContext(r)
	if caller == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, t.T("errors.auth.unauthorized"), "")
		return
	}

	if r.Method == http.MethodPut {
		var req DashboardLayoutRequest
		if !decodeJSONStrictLocalized(w, r, &req, dashboardMaxBody, logger, t) || !validateStruct(w, r, &req, t) {
			return
		}
		if err := s.identityUsers.SetDashboard(r.Context(), caller, req.Widgets); err != nil {
			s.sendDashboardError(w, r, err, t.T("errors.dashboard.saveFailed"))
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, DashboardLayout{Widgets: req.Widgets, Saved: true})
		return
	}

	widgets, saved, err := s.identityUsers.GetDashboard(r.Context(), caller)
	if err != nil {
		s.sendDashboardError(w, r, err, t.T("errors.dashboard.readFailed"))
		return
	}
	if widgets == nil {
		widgets = []string{}
	}
	sendJSONResponse(w, logger, http.StatusOK, DashboardLayout{Widgets: widgets, Saved: saved})
}

// sendDashboardError maps a layout-store error onto the response. A refused
// layout carries its reason in details: it is our own text naming the widget.
func (s *Server) sendDashboardError(w http.ResponseWriter, r *http.Request, err error, failedMsg string) {
	logger := logging.FromContext(r.Context())
	t := i18n.FromRequest(r)

	switch {
	case errors.Is(err, users.ErrInvalidDashboard):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, t.T("errors.dashboard.invalidLayout"), err.Error())
	case errors.Is(err, users.ErrUnavailable):
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, t.T("errors.dashboard.storeUnavailable"), "")
	case errors.Is(err, users.ErrUserNotFound):
		// A valid token for a deleted account: the caller no longer exists.
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, t.T("errors.auth.unauthorized"), "")
	default:
		logger.ErrorContext(r.Context(), "dashboard store failed", "event", "dashboard.failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, failedMsg, "")
	}
}
