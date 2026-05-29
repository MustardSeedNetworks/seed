// SPDX-License-Identifier: BUSL-1.1

package api

// handlers_users.go implements the Settings → Users CRUD surface added
// for seed#1191 (multi_user). The endpoints are admin-only except for
// the self-password change path (operators and viewers may rotate their
// own password). Creating users is Pro-gated via requireFeature; all
// other operations are available on any tier so a Pro→Free downgrade
// doesn't render the existing users unmanageable.

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
)

// timeNow is package-internal indirection so tests can freeze the clock
// for LockedUntilFuture deterministically. Currently it just delegates
// to [time.Now]; the indirection unlocks future test fixtures.
func timeNow() time.Time { return time.Now() }

// usersPathPrefix is the trailing-slash form of /api/v1/users used by
// the path router to extract the {username} suffix.
const usersPathPrefix = APIVersionPrefix + "/users/"

// usernameMinLen / usernameMaxLen mirror the DB CHECK constraint
// added by the hardening migration.
const (
	usernameMinLen = 3
	usernameMaxLen = 64
)

// UserResponse is the sanitized projection returned to the UI. The
// password hash, failed-attempts counter, and lock state are NEVER
// exposed (lock state is internal; the UI displays only Active /
// Disabled / Locked from the boolean lockedUntilFuture flag).
type UserResponse struct {
	ID                int64  `json:"id"`
	Username          string `json:"username"`
	Role              string `json:"role"`
	IsActive          bool   `json:"isActive"`
	AuthProvider      string `json:"authProvider"`
	Email             string `json:"email,omitempty"`
	DisplayName       string `json:"displayName,omitempty"`
	LastLogin         string `json:"lastLogin,omitempty"`         // RFC3339 or empty
	LockedUntilFuture bool   `json:"lockedUntilFuture,omitempty"` // true iff a lock is currently in effect
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// CreateUserRequest is the body of POST /api/v1/users.
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateUserRequest is the body of PATCH /api/v1/users/{username}.
// All fields are optional; only the ones present are applied. Empty
// strings on Password/Role are treated as "leave unchanged".
type UpdateUserRequest struct {
	Password string `json:"password,omitempty"`
	Role     string `json:"role,omitempty"`
	IsActive *bool  `json:"isActive,omitempty"` // pointer so caller can set false explicitly
}

// handleUsers routes /api/v1/users (GET = list, POST = create).
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleUsersList(w, r)
	case http.MethodPost:
		s.handleUserCreate(w, r)
	default:
		writeAPITokenError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
	}
}

// handleUserByName routes /api/v1/users/{username} (GET, PATCH, DELETE).
func (s *Server) handleUserByName(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Path, usersPathPrefix)
	if username == "" || strings.ContainsRune(username, '/') {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Username is required in path")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleUserGet(w, r, username)
	case http.MethodPatch:
		s.handleUserUpdate(w, r, username)
	case http.MethodDelete:
		s.handleUserDelete(w, r, username)
	default:
		writeAPITokenError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
	}
}

// handleCurrentUser implements GET /api/v1/users/me — any authenticated
// caller can read their own record. Used by the UI to display the
// "you" badge on the Users list and to enforce client-side
// self-edit-only paths.
func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPITokenError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
		return
	}
	caller := usernameFromContext(r)
	if caller == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required")
		return
	}
	s.handleUserGet(w, r, caller)
}

func (s *Server) handleUsersList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	db := s.services.Database.DB
	if db == nil {
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "User store unavailable")
		return
	}
	users, err := db.ListUsers(r.Context())
	if err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(), "list users failed", "error", err)
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to list users")
		return
	}
	resp := make([]UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, toUserResponse(u))
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, resp)
}

func (s *Server) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	// Pro gate — Free + Starter are single-admin only.
	if !s.licenseAllowsMultiUser() {
		writeAPITokenError(
			w,
			r,
			http.StatusPaymentRequired,
			"TIER_TOO_LOW",
			"Adding additional users requires the Pro tier. Activate a Pro key or start a trial with `seed license trial`.",
		)
		return
	}

	var req CreateUserRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Role = strings.TrimSpace(req.Role)
	if req.Role == "" {
		req.Role = database.RoleViewer
	}

	if err := validateUsername(req.Username); err != nil {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	if !database.IsValidRole(req.Role) {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation,
			"role must be one of: admin, operator, viewer")
		return
	}
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to hash password")
		return
	}

	db := s.services.Database.DB
	if db == nil {
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "User store unavailable")
		return
	}
	user, err := db.CreateUser(r.Context(), req.Username, hash, req.Role)
	if err != nil {
		if errors.Is(err, database.ErrUserExists) {
			writeAPITokenError(w, r, http.StatusConflict, "USER_EXISTS", "Username already in use")
			return
		}
		logging.FromContext(r.Context()).ErrorContext(r.Context(), "create user failed", "error", err)
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to create user")
		return
	}

	logging.FromContext(r.Context()).InfoContext(r.Context(), "user created",
		"actor", usernameFromContext(r), "target", user.Username, "role", user.Role,
		"event", "auth.user.created")

	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusCreated, toUserResponse(user))
}

func (s *Server) handleUserGet(w http.ResponseWriter, r *http.Request, target string) {
	caller := usernameFromContext(r)
	if caller == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required")
		return
	}
	// Self-read is always allowed; everyone else needs admin.
	if caller != target && !s.callerIsAdmin(r) {
		writeAPITokenError(w, r, http.StatusForbidden, ErrCodeForbidden, "Admin role required")
		return
	}
	db := s.services.Database.DB
	if db == nil {
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "User store unavailable")
		return
	}
	user, err := db.GetUser(r.Context(), target)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound, "User not found")
			return
		}
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to fetch user")
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, toUserResponse(user))
}

func (s *Server) handleUserUpdate(w http.ResponseWriter, r *http.Request, target string) {
	caller := usernameFromContext(r)
	if caller == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required")
		return
	}

	var req UpdateUserRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}
	if !s.checkUserUpdateAuthorization(w, r, caller, target, &req) {
		return
	}

	db := s.services.Database.DB
	if db == nil {
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "User store unavailable")
		return
	}

	if req.Password != "" && !s.applyPasswordUpdate(w, r, db, target, req.Password) {
		return
	}
	if req.Role != "" && !s.applyRoleUpdate(w, r, db, target, req.Role) {
		return
	}
	if req.IsActive != nil && !*req.IsActive && !s.applyDeactivation(w, r, db, caller, target) {
		return
	}

	updated, err := db.GetUser(r.Context(), target)
	if err != nil {
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to reload user")
		return
	}
	logging.FromContext(r.Context()).InfoContext(r.Context(), "user updated",
		"actor", caller, "target", target, "event", "auth.user.updated")
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, toUserResponse(updated))
}

// checkUserUpdateAuthorization enforces "self may only change own password,
// admin may do anything." Returns false (and writes the error response)
// when the caller may not proceed.
func (s *Server) checkUserUpdateAuthorization(
	w http.ResponseWriter, r *http.Request, caller, target string, req *UpdateUserRequest,
) bool {
	if caller != target && !s.callerIsAdmin(r) {
		writeAPITokenError(w, r, http.StatusForbidden, ErrCodeForbidden, "Admin role required")
		return false
	}
	if caller == target && !s.callerIsAdmin(r) && (req.Role != "" || req.IsActive != nil) {
		writeAPITokenError(w, r, http.StatusForbidden, ErrCodeForbidden,
			"Only an administrator can change role or active state")
		return false
	}
	return true
}

// applyPasswordUpdate validates strength, hashes, and persists. Returns
// false (and writes the error response) on any failure.
func (s *Server) applyPasswordUpdate(
	w http.ResponseWriter, r *http.Request, db *database.DB, target, password string,
) bool {
	if err := auth.ValidatePasswordStrength(password); err != nil {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return false
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to hash password")
		return false
	}
	if updErr := db.UpdateUserPassword(r.Context(), target, hash); updErr != nil {
		if errors.Is(updErr, database.ErrUserNotFound) {
			writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound, "User not found")
			return false
		}
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to update password")
		return false
	}
	return true
}

func (s *Server) applyRoleUpdate(
	w http.ResponseWriter, r *http.Request, db *database.DB, target, role string,
) bool {
	if !database.IsValidRole(role) {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation,
			"role must be one of: admin, operator, viewer")
		return false
	}
	err := db.UpdateUserRole(r.Context(), target, role)
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, database.ErrUserNotFound):
		writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound, "User not found")
	case errors.Is(err, database.ErrLastAdmin):
		writeAPITokenError(w, r, http.StatusConflict, "LAST_ADMIN", "Cannot demote the last administrator")
	default:
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to update role")
	}
	return false
}

func (s *Server) applyDeactivation(
	w http.ResponseWriter, r *http.Request, db *database.DB, caller, target string,
) bool {
	if caller == target {
		writeAPITokenError(w, r, http.StatusConflict, "SELF_DEACTIVATE",
			"You cannot deactivate your own account")
		return false
	}
	if err := db.DeactivateUser(r.Context(), target); err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound, "User not found")
			return false
		}
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to deactivate user")
		return false
	}
	return true
}

func (s *Server) handleUserDelete(w http.ResponseWriter, r *http.Request, target string) {
	if !s.requireAdmin(w, r) {
		return
	}
	caller := usernameFromContext(r)
	if caller == target {
		writeAPITokenError(w, r, http.StatusConflict, "SELF_DELETE", "You cannot delete your own account")
		return
	}
	db := s.services.Database.DB
	if db == nil {
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "User store unavailable")
		return
	}
	if err := db.DeleteUser(r.Context(), target); err != nil {
		switch {
		case errors.Is(err, database.ErrUserNotFound):
			writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound, "User not found")
		case errors.Is(err, database.ErrLastAdmin):
			writeAPITokenError(w, r, http.StatusConflict, "LAST_ADMIN",
				"Cannot delete the last administrator")
		default:
			writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to delete user")
		}
		return
	}
	logging.FromContext(r.Context()).InfoContext(r.Context(), "user deleted",
		"actor", caller, "target", target, "event", "auth.user.deleted")
	w.WriteHeader(http.StatusNoContent)
}

// requireAdmin replies 403 (or 401 if no caller) when the requester is
// not an admin. Returns true only if the request may proceed.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	caller := usernameFromContext(r)
	if caller == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required")
		return false
	}
	if !s.callerIsAdmin(r) {
		writeAPITokenError(w, r, http.StatusForbidden, ErrCodeForbidden, "Admin role required")
		return false
	}
	return true
}

// callerRole resolves the requesting user's role. ok is false when the
// role can't be established (no caller, lookup failure, inactive user).
// When no user DB is configured (single-user/env mode) it returns
// admin/true: the lone env-configured operator is implicitly admin, which
// keeps single-user dev builds fully usable.
func (s *Server) callerRole(r *http.Request) (string, bool) {
	db := s.services.Database.DB
	caller := usernameFromContext(r)
	// No user DB attached = no multi-user authorization model in effect,
	// so the gate has nothing to enforce — treat the request as admin.
	// Production single-user still gets X-Username from the auth
	// middleware, so this branch only fires in test/dev contexts that
	// reach the mux without auth middleware, matching the long-standing
	// callerIsAdmin "no DB = admin" tolerance.
	if db == nil {
		return database.RoleAdmin, true
	}
	if caller == "" {
		return "", false
	}
	u, err := db.GetUser(r.Context(), caller)
	if err != nil || !u.IsActive {
		return "", false
	}
	// #1255: PAT auth sets X-Token-Scope to a per-token role cap. The
	// effective role becomes the lower of the owner's role and the
	// token's scope so an automation token minted from an admin owner
	// can't escalate. An invalid/unknown scope value is ignored (rank-0
	// would lock the token out entirely, the wrong failure mode for a
	// malformed header).
	if scope := r.Header.Get("X-Token-Scope"); scope != "" && database.IsValidRole(scope) {
		if roleRank(scope) < roleRank(u.Role) {
			return scope, true
		}
	}
	return u.Role, true
}

// Role ranks order the three roles for >= comparisons. Unknown/unresolved
// roles rank at rankNone so they never satisfy a minimum of viewer or above.
const (
	rankNone     = 0
	rankViewer   = 1
	rankOperator = 2
	rankAdmin    = 3
)

// roleRank maps a role string to its comparable rank.
func roleRank(role string) int {
	switch role {
	case database.RoleAdmin:
		return rankAdmin
	case database.RoleOperator:
		return rankOperator
	case database.RoleViewer:
		return rankViewer
	default:
		return rankNone
	}
}

// callerIsAdmin reports whether the request's user is an admin. Tolerates
// a missing DB (dev builds) by assuming admin so the panel stays usable.
func (s *Server) callerIsAdmin(r *http.Request) bool {
	role, ok := s.callerRole(r)
	return ok && role == database.RoleAdmin
}

// requireRole replies 401 (no caller) or 403 (under-privileged) unless the
// requester's role ranks at or above min. Returns true when the request
// may proceed.
//
// Denials emit structured `event=auth.unauthorized` / `event=auth.forbidden`
// records (#1257) so SIEM pipelines across seed/stem/niac can filter
// authorization failures uniformly. The fields mirror what niac emits on
// scope mismatch: required role/scope, actual role (if resolved), client
// IP, request path + method, and the caller username.
func (s *Server) requireRole(w http.ResponseWriter, r *http.Request, minRole string) bool {
	role, ok := s.callerRole(r)
	if !ok {
		logging.FromContext(r.Context()).WarnContext(r.Context(),
			"Authentication required for protected endpoint",
			"event", "auth.unauthorized",
			"required_role", minRole,
			"client_ip", GetClientIP(r),
			"path", r.URL.Path,
			"method", r.Method,
		)
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required")
		return false
	}
	if roleRank(role) < roleRank(minRole) {
		logging.FromContext(r.Context()).WarnContext(r.Context(),
			"Forbidden: insufficient role",
			"event", "auth.forbidden",
			"required_role", minRole,
			"actual_role", role,
			"username", usernameFromContext(r),
			"client_ip", GetClientIP(r),
			"path", r.URL.Path,
			"method", r.Method,
		)
		writeAPITokenError(w, r, http.StatusForbidden, ErrCodeForbidden, minRole+" role required")
		return false
	}
	return true
}

// requireWriteAccess gates state-changing operations on operator-or-above;
// viewers are read-only (#1226). Safe (read) methods should skip this.
func (s *Server) requireWriteAccess(w http.ResponseWriter, r *http.Request) bool {
	return s.requireRole(w, r, database.RoleOperator)
}

// writeGated wraps a handler so that state-changing methods require an
// operator-or-above role while safe reads (GET/HEAD/OPTIONS) pass through
// untouched. Applied at route registration to persistent-configuration
// endpoints so viewers stay read-only on them; diagnostic actions
// (scans/pings/traceroute) are deliberately left ungated (#1226).
func (s *Server) writeGated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next(w, r)
		default:
			if !s.requireWriteAccess(w, r) {
				return
			}
			next(w, r)
		}
	}
}

// licenseAllowsMultiUser reports whether the active license tier may
// add users beyond the bootstrap admin. Pro grants it; trial Pro
// grants it; Free + Starter do not. A nil license manager is treated
// as "license disabled" (dev / test) and permits.
func (s *Server) licenseAllowsMultiUser() bool {
	mgr := s.services.Auth.License
	if mgr == nil {
		return true
	}
	return mgr.HasFeature("multi_user")
}

func validateUsername(name string) error {
	if len(name) < usernameMinLen {
		return errors.New("username must be at least 3 characters")
	}
	if len(name) > usernameMaxLen {
		return errors.New("username must be at most 64 characters")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
		default:
			return errors.New("username may contain only letters, digits, dot, dash, underscore")
		}
	}
	return nil
}

func toUserResponse(u *database.User) UserResponse {
	out := UserResponse{
		ID:           u.ID,
		Username:     u.Username,
		Role:         u.Role,
		IsActive:     u.IsActive,
		AuthProvider: u.AuthProvider,
		Email:        u.Email,
		DisplayName:  u.DisplayName,
		CreatedAt:    u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    u.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if u.LastLogin != nil && !u.LastLogin.IsZero() {
		out.LastLogin = u.LastLogin.UTC().Format("2006-01-02T15:04:05Z")
	}
	if u.LockedUntil != nil && u.LockedUntil.After(timeNow()) {
		out.LockedUntilFuture = true
	}
	return out
}
