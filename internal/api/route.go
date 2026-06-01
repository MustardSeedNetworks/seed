package api

// route.go is the capability registry: routes are declared as data and a single
// register() composes their per-route middleware in one canonical order. This
// replaces hand-wrapping each route at registration, where the wrapper nesting
// was applied inconsistently and could be forgotten (a documented regression
// class). See docs/architecture/decisions/0002-capability-registry.md.

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
)

// route declares an HTTP route and its per-route policy. Authentication and CSRF
// are enforced globally (server_lifecycle.go) and are NOT part of this policy —
// the trio captured here is exactly what was previously hand-wrapped at
// registration: a role gate, a license-feature gate, and rate limiting.
type route struct {
	// path is the full request path (callers pass APIVersionPrefix+"/...").
	path string
	// handler is the terminal handler for the route.
	handler http.HandlerFunc
	// minRole gates state-changing methods. The supported value today is
	// database.RoleOperator, applied via writeGated (safe GET/HEAD/OPTIONS pass;
	// mutating methods require operator+). Empty = no role gate.
	minRole string
	// feature is the license feature required via requireFeature. Empty = none.
	feature string
	// rateLimited wraps the route in the shared endpoint rate limiter.
	rateLimited bool
}

// register installs rt on the mux, composing middleware in ONE canonical order
// for every route: rateLimit → requireFeature → requireRole → handler (rate
// limit outermost, role gate closest to the handler). Composing here — rather
// than at each call site — makes the policy declarative and the ordering
// uniform, and is the single choke point a future audit/CI gate can enforce.
func (s *Server) register(rt route) {
	// Record the route for the /__capabilities manifest before composing.
	s.manifest = append(s.manifest, rt)

	h := rt.handler

	// requireRole closest to the handler.
	if rt.minRole == database.RoleOperator {
		h = s.writeGated(h)
	}
	// requireFeature next.
	if rt.feature != "" {
		h = s.requireFeature(rt.feature, h)
	}
	// rateLimit outermost. RateLimitMiddleware takes and returns http.Handler;
	// an http.HandlerFunc satisfies http.Handler.
	if rt.rateLimited {
		s.mux.Handle(rt.path, s.endpointRateLimiter().RateLimitMiddleware(h))
		return
	}
	s.mux.HandleFunc(rt.path, h)
}

// registerAll installs a slice of routes.
func (s *Server) registerAll(routes []route) {
	for _, rt := range routes {
		s.register(rt)
	}
}

// capabilityView is the JSON-serializable projection of a route's policy for
// the /__capabilities manifest (the handler func itself is not exposed).
type capabilityView struct {
	Path        string `json:"path"`
	MinRole     string `json:"minRole,omitempty"`
	Feature     string `json:"feature,omitempty"`
	RateLimited bool   `json:"rateLimited,omitempty"`
}

// handleCapabilities serves the route-policy manifest: every route registered
// through register() with its per-route policy (role / feature / rate-limit).
// No auth — like /__version, it is a deployment/audit introspection surface.
// Auth and CSRF are global and intentionally not represented here.
func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	views := make([]capabilityView, 0, len(s.manifest))
	for _, rt := range s.manifest {
		views = append(views, capabilityView{
			Path:        rt.path,
			MinRole:     rt.minRole,
			Feature:     rt.feature,
			RateLimited: rt.rateLimited,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(views); err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"failed to encode capabilities manifest", "error", err)
	}
}
