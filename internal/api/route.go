package api

// route.go is the capability registry: routes are declared as data and a single
// register() composes their per-route middleware in one canonical order. This
// replaces hand-wrapping each route at registration, where the wrapper nesting
// was applied inconsistently and could be forgotten (a documented regression
// class). See docs/architecture/decisions/0002-capability-registry.md.

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// route declares an HTTP route and its per-route policy. Authentication and CSRF
// are enforced globally (server_lifecycle.go) and are NOT part of this policy.
// Everything a route's request handling is gated on lives here so the registry
// is the single authoritative source (ADR-0002): allowed methods, body-size
// limit, a role gate, a license-feature gate, and rate limiting.
type route struct {
	// path is the full request path (callers pass APIVersionPrefix+"/..."). It
	// MAY carry a Go 1.22 method prefix ("GET /api/v1/..."), in which case the
	// method is enforced natively by ServeMux and `methods` is left empty.
	path string
	// handler is the terminal handler for the route.
	handler http.HandlerFunc
	// methods is the set of HTTP methods the route accepts. register() rejects
	// any other method with 405 + an Allow header (before feature/role checks).
	// Leave empty ONLY when path carries a method prefix (ServeMux enforces it).
	methods []string
	// maxBodyBytes caps the request body (DoS guard). 0 means the default
	// (MaxBodySizeJSON); set explicitly for larger uploads or tighter limits.
	maxBodyBytes int64
	// minRole gates state-changing methods. The supported value today is
	// database.RoleOperator, applied via writeGated (safe GET/HEAD/OPTIONS pass;
	// mutating methods require operator+). Empty = no role gate.
	minRole string
	// feature is the license feature required via requireFeature. Empty = none.
	feature string
	// rateLimited wraps the route in the shared endpoint rate limiter.
	rateLimited bool
}

// methodFromPath extracts a Go 1.22 method prefix ("GET /path") if present.
func methodFromPath(path string) (string, bool) {
	i := strings.IndexByte(path, ' ')
	if i <= 0 {
		return "", false
	}
	switch m := path[:i]; m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return m, true
	default:
		return "", false
	}
}

// effectiveMethods is the route's declared method set: a method prefix on the
// path (ServeMux-enforced) takes precedence, otherwise the methods field.
func effectiveMethods(rt route) []string {
	if m, ok := methodFromPath(rt.path); ok {
		return []string{m}
	}
	return rt.methods
}

// methodGate rejects any method outside allowed with a 405 + Allow header,
// preserving the project's JSON error envelope.
func (s *Server) methodGate(allowed []string, next http.HandlerFunc) http.HandlerFunc {
	allowHeader := strings.Join(allowed, ", ")
	return func(w http.ResponseWriter, r *http.Request) {
		if slices.Contains(allowed, r.Method) {
			next(w, r)
			return
		}
		w.Header().Set("Allow", allowHeader)
		sendErrorResponseWithDetails(w, logging.FromContext(r.Context()),
			http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
	}
}

// bodyLimited caps the request body before the handler reads it.
func bodyLimited(limit int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next(w, r)
	}
}

// register installs rt on the mux, composing middleware in ONE canonical order
// for every route: rateLimit → requireFeature → requireRole → handler (rate
// limit outermost, role gate closest to the handler). Composing here — rather
// than at each call site — makes the policy declarative and the ordering
// uniform, and is the single choke point a future audit/CI gate can enforce.
func (s *Server) register(rt route) {
	// Resolve the effective policy, then record it for the /__capabilities
	// manifest (and the route-policy test) before composing middleware.
	rt.methods = effectiveMethods(rt)
	if rt.maxBodyBytes == 0 {
		rt.maxBodyBytes = MaxBodySizeJSON
	}
	s.manifest = append(s.manifest, rt)

	h := rt.handler

	// bodyLimit closest to the handler so r.Body is capped before any read.
	h = bodyLimited(rt.maxBodyBytes, h)
	// requireRole next.
	if rt.minRole == database.RoleOperator {
		h = s.writeGated(h)
	}
	// requireFeature next.
	if rt.feature != "" {
		h = s.requireFeature(rt.feature, h)
	}
	// methodGate before feature/role so a wrong method 405s early. Skipped when
	// the path carries a method prefix — ServeMux enforces the method itself.
	if len(rt.methods) > 0 && !hasMethodPrefix(rt.path) {
		h = s.methodGate(rt.methods, h)
	}
	// rateLimit outermost. RateLimitMiddleware takes and returns http.Handler;
	// an http.HandlerFunc satisfies http.Handler.
	if rt.rateLimited {
		s.mux.Handle(rt.path, s.endpointRateLimiter().RateLimitMiddleware(h))
		return
	}
	s.mux.HandleFunc(rt.path, h)
}

// hasMethodPrefix reports whether path carries a Go 1.22 method prefix.
func hasMethodPrefix(path string) bool {
	_, ok := methodFromPath(path)
	return ok
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
	Path         string   `json:"path"`
	Methods      []string `json:"methods,omitempty"`
	MaxBodyBytes int64    `json:"maxBodyBytes,omitempty"`
	MinRole      string   `json:"minRole,omitempty"`
	Feature      string   `json:"feature,omitempty"`
	RateLimited  bool     `json:"rateLimited,omitempty"`
}

// handleCapabilities serves the route-policy manifest: every route registered
// through register() with its per-route policy (role / feature / rate-limit).
// No auth — like /__version, it is a deployment/audit introspection surface.
// Auth and CSRF are global and intentionally not represented here.
func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	views := make([]capabilityView, 0, len(s.manifest))
	for _, rt := range s.manifest {
		views = append(views, capabilityView{
			Path:         rt.path,
			Methods:      rt.methods,
			MaxBodyBytes: rt.maxBodyBytes,
			MinRole:      rt.minRole,
			Feature:      rt.feature,
			RateLimited:  rt.rateLimited,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(views); err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"failed to encode capabilities manifest", "error", err)
	}
}
