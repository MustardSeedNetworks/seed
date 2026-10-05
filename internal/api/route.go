package api

// route.go wires seed's policy into the fleet's capability registry
// (foundation pkg/httpserver/route, ADR-0002). Routes are declared as data and
// the shared Registrar composes each one's policy in its one canonical order —
// limiter → auth → method gate → CSRF → feature → scope → body cap — wrapped in
// request ID, access log and panic recovery. This file supplies only what that
// order does not decide: seed's error envelope, its authentication chain, its
// CSRF session key, its role gate, its licence gate and its endpoint limiter.
// scripts/check-route-policy.sh keeps every route on the Registrar.

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"
	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// limitEndpoint names the shared endpoint rate limiter (Route.Limiter).
const limitEndpoint = "endpoint"

// newRegistrar builds the Registrar over this server's policy. It reads the
// auth manager, CSRF manager and endpoint limiter when called, so build it
// after they are set.
func (s *Server) newRegistrar() *route.Registrar {
	return route.New(route.Config{
		Error:        registrarError,
		MaxBodyBytes: MaxBodySizeJSON,
		Logger:       logging.GetLogger(),
		Auth:         s.authenticate,
		CSRF:         s.csrfManager(),
		SessionKey:   csrfSessionKey,
		Scope:        s.scopeGate,
		Feature:      s.featureGate,
		Limiters: map[string]route.Middleware{
			limitEndpoint: s.endpointRateLimiter().RateLimitMiddleware,
		},
	})
}

// RouteManifest returns every route the registry declares, in registration
// order, without starting a daemon: registration only composes closures, so a
// Server carrying just the auth manager, CSRF manager and endpoint limiter
// registers the full set. It is what cmd/seed-openapi documents.
func RouteManifest() []route.Policy {
	s := &Server{
		authMgr:         auth.NewManager("", time.Hour, "", ""),
		csrf:            csrf.NewManager(),
		endpointLimiter: NewEndpointRateLimiter(DefaultEndpointRateLimitConfig()),
	}
	defer s.authMgr.Stop()
	defer s.csrf.Stop()
	defer s.endpointLimiter.Stop()
	s.setupRoutes()
	return s.routes.Policies()
}

// authenticate admits a request carrying a personal access token or a valid
// session JWT. The PAT middleware runs first so an `sd_pat_…` bearer is
// resolved before the JWT middleware would reject it as a malformed JWT.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return apiTokenMiddleware(s.identityTokens, s.resolveClientID, s.authManager().Middleware(next))
}

// scopeGate is the Route.Scope hook. A scope is the minimum role for a
// route's state-changing methods, roles.Operator or roles.Admin: writeGated
// lets safe methods through and requires that role or above for the rest.
func (s *Server) scopeGate(scope string) route.Middleware {
	if scope != roles.Operator && scope != roles.Admin {
		panic(fmt.Sprintf("route scope %q: only %q and %q are supported", scope, roles.Operator, roles.Admin))
	}
	return func(next http.Handler) http.Handler {
		return s.writeGated(scope, next.ServeHTTP)
	}
}

// featureGate is the Route.Feature hook: the route answers 402 unless the
// active licence includes feature.
func (s *Server) featureGate(feature string) route.Middleware {
	return func(next http.Handler) http.Handler {
		return s.requireFeature(feature, next.ServeHTTP)
	}
}

// csrfSessionKey keys CSRF tokens by the session JWT, the key
// /api/v1/auth/csrf mints under. A personal access token is not an ambient
// credential — a cross-site page cannot set an Authorization header — so a
// PAT-authenticated request has no session to forge and passes (#2450). A
// request with no session passes too: behind Auth it has already been refused,
// and on a pre-session route there is nothing to forge (#2391).
func csrfSessionKey(r *http.Request) (string, bool) {
	if auth.IsAPITokenAuth(r.Context()) {
		return "", false
	}
	key := auth.GetSessionIDFromRequest(r)
	return key, key != ""
}

// registrarError renders the Registrar's own refusals (405, CSRF, a recovered
// panic) in seed's JSON envelope, with the codes and messages seed answered
// before the shared registrar.
func registrarError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	switch code {
	case "method_not_allowed":
		writeError(w, r, status, ErrCodeMethodNotAllowed, message)
	case "csrf_token_missing", "csrf_token_expired", "csrf_token_invalid":
		logging.FromContext(r.Context()).WarnContext(r.Context(), "CSRF validation failed",
			"pattern", r.Pattern, "method", r.Method, "reason", code)
		writeError(w, r, status, ErrCodeForbidden, csrfRefusal(code))
	default: // internal_server_error, csrf_unavailable
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			i18n.FromRequest(r).T("errors.api.internalError"))
	}
}

// csrfRefusal is the message the UI has always received for each CSRF failure.
func csrfRefusal(code string) string {
	switch code {
	case "csrf_token_missing":
		return "CSRF token required"
	case "csrf_token_expired":
		return "CSRF token expired"
	default:
		return "Invalid CSRF token"
	}
}
