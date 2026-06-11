// SPDX-License-Identifier: BUSL-1.1

package api

// tokens.go implements the personal-access-token surface added in Phase D-2 of
// LICENSE_STRATEGY: API tokens for programmatic access (scripts, monitoring,
// CI). Minting requires the Pro tier; listing / revoking is available to any
// authenticated user for their own tokens.
//
// All token-store access in handler bodies goes through s.identityTokens (the
// use-case, ADR-0024). The PAT authentication seam (apiTokenMiddleware /
// resolveAPIToken) remains unchanged — it is wired in server_lifecycle.go
// via s.services.Auth.APITokens and is authentication infrastructure, not
// handler data access.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/tokens"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// APITokenPrefix is the visible prefix that distinguishes a Seed
// personal-access token from a JWT or other Bearer value.
const APITokenPrefix = "sd_pat_"

// apiTokenIDLength is the random byte count for token IDs (16 bytes →
// 32 hex chars). Small enough to be friendly, large enough to avoid
// any collision risk over the lifetime of the deployment.
const apiTokenIDLength = 16

// apiTokenSecretLength is the random byte count for the secret portion
// of the token (32 bytes → 64 hex chars → ~256 bits of entropy).
const apiTokenSecretLength = 32

// apiTokenDisplayPrefix is how many chars of the plaintext token are
// retained in the DB so the UI can identify it without revealing it.
// "sd_pat_" (7) + first 5 chars of the secret = 12.
const apiTokenDisplayPrefix = 12

// apiTokenNameMaxLen caps the user-supplied token label length. The
// limit keeps DB rows compact and prevents UI overflow.
const apiTokenNameMaxLen = 64

// MintTokenRequest is the body of POST /api/v1/tokens. Scope is the
// optional per-token role cap (#1255): omit / empty means the token
// inherits the owner's role at auth time; "viewer"/"operator"/"admin"
// caps the effective role at min(owner.role, scope). A scope above the
// owner's role is rejected up front so the UI surfaces a clear error
// rather than minting a token that silently downgrades.
type MintTokenRequest struct {
	Name  string `json:"name"`
	Scope string `json:"scope,omitempty"`
}

// MintTokenResponse is returned by POST /api/v1/tokens. The Token
// field contains the plaintext value and is shown ONLY at creation —
// it is never persisted in plaintext form.
type MintTokenResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"createdAt"`
	// Scope echoes the requested per-token cap (#1255). Empty when the
	// token inherits the owner's role.
	Scope string `json:"scope,omitempty"`
}

// TokenListItem is a sanitized projection of APITokenRecord for the
// list endpoint — never contains the plaintext token or the hash.
type TokenListItem struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Prefix     string    `json:"prefix"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsedAt time.Time `json:"lastUsedAt,omitzero"`
	RevokedAt  time.Time `json:"revokedAt,omitzero"`
	// Scope is the per-token role cap, empty when inheriting the owner's
	// role (#1255). Surfaced so the UI can show "viewer" / "operator" /
	// "admin" badges next to each token.
	Scope string `json:"scope,omitempty"`
}

// LicenseStatusResponse is returned by GET /api/v1/license. It tells
// the UI which tier is active, whether features should be exposed,
// and how much trial time (if any) remains.
type LicenseStatusResponse struct {
	Tier          string    `json:"tier"`      // "Free" | "Starter" | "Pro" | "Trial"
	TierValue     int       `json:"tierValue"` // numeric (0=Free, 1=Starter, 2=Pro)
	IsTrialMode   bool      `json:"isTrialMode"`
	TrialDaysLeft int       `json:"trialDaysLeft,omitempty"`
	CanMintTokens bool      `json:"canMintTokens"` // true iff Tier >= Pro or active trial
	Activated     bool      `json:"activated"`
	ExpiresAt     time.Time `json:"expiresAt,omitzero"`
}

// handleLicenseStatus exposes the local license state to the UI. The
// unlicensed (Free) state is a valid result, not an error condition.
// Reads the license manager through s.licenseManager() — the single
// Server accessor that D1 will repoint off the service container.
func (s *Server) handleLicenseStatus(w http.ResponseWriter, r *http.Request) {
	resp := LicenseStatusResponse{
		Tier:          license.TierFree.String(),
		TierValue:     int(license.TierFree),
		CanMintTokens: false,
	}

	mgr := s.licenseManager()
	if mgr == nil {
		// License disabled (dev / test build) — allow minting so the
		// feature stays usable. Matches the tokens use-case LicenseGate
		// (a nil manager permits minting).
		resp.CanMintTokens = true
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, resp)
		return
	}

	st := mgr.GetState()
	if st == nil {
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, resp)
		return
	}

	resp.Activated = true
	resp.IsTrialMode = st.IsTrialMode
	resp.ExpiresAt = st.ExpiresAt
	resp.TierValue = int(st.Tier)
	if st.IsTrialMode {
		resp.Tier = "Trial"
		resp.TrialDaysLeft = mgr.TrialDaysRemaining()
		resp.CanMintTokens = true
	} else {
		resp.Tier = st.Tier.String()
		// Route the UI signal through the same catalog lookup the
		// backend gate (tokens.LicenseGate) uses. Keeps the two in
		// lock-step if rest_api ever moves between tiers.
		resp.CanMintTokens = mgr.HasFeature("rest_api")
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, resp)
}

// handleAPITokens routes /api/v1/tokens (POST = mint, GET = list).
func (s *Server) handleAPITokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleAPITokenMint(w, r)
	case http.MethodGet:
		s.handleAPITokenList(w, r)
	default:
		writeAPITokenError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed,
			"Method not allowed")
	}
}

// handleAPITokenByID routes /api/v1/tokens/<id> (DELETE = revoke).
func (s *Server) handleAPITokenByID(w http.ResponseWriter, r *http.Request) {
	s.handleAPITokenRevoke(w, r)
}

func (s *Server) handleAPITokenMint(w http.ResponseWriter, r *http.Request) {
	username := usernameFromContext(r)
	if username == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
			"Authentication required")
		return
	}

	var req MintTokenRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation,
			"`name` is required")
		return
	}
	if len(req.Name) > apiTokenNameMaxLen {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation,
			"`name` must be 64 characters or fewer")
		return
	}

	// #1255: optional per-token scope. Must be a legal role string and
	// must not exceed the minter's own role — escalating via a PAT
	// would defeat the role gate. Scope validation stays in the handler
	// (authorization edge concern, ADR-0024).
	req.Scope = strings.TrimSpace(req.Scope)
	if req.Scope != "" {
		if !database.IsValidRole(req.Scope) {
			writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation,
				"`scope` must be one of viewer, operator, admin")
			return
		}
		ownerRole, ok := s.callerRole(r)
		if !ok || roleRank(req.Scope) > roleRank(ownerRole) {
			writeAPITokenError(w, r, http.StatusForbidden, ErrCodeForbidden,
				"`scope` may not exceed your own role")
			return
		}
	}

	id, secret, mintErr := mintTokenMaterial()
	if mintErr != nil {
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to generate token material")
		return
	}
	plaintext := APITokenPrefix + secret
	rec := database.APITokenRecord{
		ID:            id,
		OwnerUsername: username,
		Name:          req.Name,
		TokenHash:     hashAPIToken(plaintext),
		Prefix:        plaintext[:apiTokenDisplayPrefix],
		CreatedAt:     time.Now().UTC(),
		Scope:         req.Scope,
	}

	if insertErr := s.identityTokens.Mint(r.Context(), rec); insertErr != nil {
		switch {
		case errors.Is(insertErr, tokens.ErrMintingNotAllowed):
			writeAPITokenError(w, r, http.StatusPaymentRequired, "TIER_TOO_LOW",
				"API token minting requires the Pro tier. "+
					"Start a Pro trial with `seed license trial` or activate a Pro key.")
		case errors.Is(insertErr, tokens.ErrUnavailable):
			writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
				"API token storage is not available")
		default:
			logger := logging.FromContext(r.Context())
			logger.ErrorContext(r.Context(), "failed to insert api token", "error", insertErr)
			writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal,
				"Failed to persist token")
		}
		return
	}

	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusCreated, MintTokenResponse{
		ID:        rec.ID,
		Name:      rec.Name,
		Token:     plaintext,
		Prefix:    rec.Prefix,
		CreatedAt: rec.CreatedAt,
		Scope:     rec.Scope,
	})
}

func (s *Server) handleAPITokenList(w http.ResponseWriter, r *http.Request) {
	username := usernameFromContext(r)
	if username == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
			"Authentication required")
		return
	}

	rows, err := s.identityTokens.List(r.Context(), username)
	if err != nil {
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to list tokens")
		return
	}
	out := make([]TokenListItem, 0, len(rows))
	for _, t := range rows {
		out = append(out, TokenListItem{
			ID:         t.ID,
			Name:       t.Name,
			Prefix:     t.Prefix,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
			RevokedAt:  t.RevokedAt,
			Scope:      t.Scope,
		})
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, out)
}

func (s *Server) handleAPITokenRevoke(w http.ResponseWriter, r *http.Request) {
	username := usernameFromContext(r)
	if username == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
			"Authentication required")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, APIVersionPrefix+"/tokens/")
	if id == "" || strings.ContainsRune(id, '/') {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeBadRequest,
			"Token ID is required in path")
		return
	}

	if err := s.identityTokens.Revoke(r.Context(), id, username); err != nil {
		switch {
		case errors.Is(err, tokens.ErrUnavailable):
			writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
				"API token storage is not available")
		case errors.Is(err, sql.ErrNoRows):
			writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound,
				"Token not found or already revoked")
		default:
			writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal,
				"Failed to revoke token")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// mintTokenMaterial returns (id, secret, err). The ID is a hex string
// used as the public handle; the secret is the high-entropy portion
// appended to APITokenPrefix to form the plaintext.
func mintTokenMaterial() (string, string, error) {
	idBytes := make([]byte, apiTokenIDLength)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", err
	}
	secretBytes := make([]byte, apiTokenSecretLength)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(idBytes), hex.EncodeToString(secretBytes), nil
}

// hashAPIToken returns the SHA-256 hex digest of the plaintext token.
// We use SHA-256 (not bcrypt/argon) because tokens are high-entropy
// random values; the goal is "DB compromise doesn't yield usable
// tokens," not slowing down password guessing.
func hashAPIToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// resolveAPIToken returns the matched record for the given plaintext
// token, or a zero record if no active token matches. The token's
// last_used_at is touched on a successful lookup (best-effort; failures
// are logged but do not block the request). Callers check the returned
// OwnerUsername == "" to detect no-match; #1255 callers also read Scope
// to clamp the effective role.
//
// This function is part of the PAT authentication seam and takes
// *database.APITokenRepository directly — it is not a handler data-access
// path and is intentionally excluded from the use-case strangle (ADR-0024).
func resolveAPIToken(ctx context.Context, repo *database.APITokenRepository, plaintext string) database.APITokenRecord {
	if repo == nil || !strings.HasPrefix(plaintext, APITokenPrefix) {
		return database.APITokenRecord{}
	}
	rec, err := repo.FindActiveByHash(ctx, hashAPIToken(plaintext))
	if err != nil {
		return database.APITokenRecord{}
	}
	if touchErr := repo.TouchLastUsed(ctx, rec.ID); touchErr != nil {
		logging.GetLogger().WarnContext(ctx, "failed to update api token last_used_at",
			"token_id", rec.ID, "error", touchErr)
	}
	return rec
}

// apiTokenMiddleware sits in front of the existing JWT auth middleware.
// If the request carries `Authorization: Bearer sd_pat_...`, it looks
// up the token, sets X-Username, and forwards to `next` so downstream
// handlers see an authenticated user. Otherwise it falls through to
// `next` unchanged, letting the JWT middleware handle the request.
//
// This middleware is part of the PAT authentication seam — it takes
// *database.APITokenRepository directly and is wired in server_lifecycle.go.
// It is intentionally excluded from the use-case strangle (ADR-0024).
func apiTokenMiddleware(repo *database.APITokenRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only attempt token auth for API paths; static assets and
		// auth endpoints are handled by the existing JWT bypass.
		if !strings.HasPrefix(r.URL.Path, APIVersionPrefix) {
			next.ServeHTTP(w, r)
			return
		}
		authz := r.Header.Get("Authorization")
		if authz == "" {
			next.ServeHTTP(w, r)
			return
		}
		const bearer = "Bearer "
		if !strings.HasPrefix(authz, bearer) {
			next.ServeHTTP(w, r)
			return
		}
		token := strings.TrimPrefix(authz, bearer)
		if !strings.HasPrefix(token, APITokenPrefix) {
			// Not a PAT — let the JWT middleware handle it.
			next.ServeHTTP(w, r)
			return
		}
		rec := resolveAPIToken(r.Context(), repo, token)
		if rec.OwnerUsername == "" {
			writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
				"Invalid or revoked API token")
			return
		}
		ctx := logging.WithUserID(r.Context(), rec.OwnerUsername)
		r.Header.Set("X-Username", rec.OwnerUsername)
		// #1255: thread the per-token scope so callerRole can clamp the
		// effective role at min(owner.role, token.scope). Empty scope
		// (legacy tokens / no-cap mints) leaves the header unset and
		// the owner's full role applies.
		if rec.Scope != "" {
			r.Header.Set("X-Token-Scope", rec.Scope)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// writeAPITokenError is a thin wrapper around sendErrorResponseWithDetails
// that pulls the logger + localizer from the request context.
func writeAPITokenError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	logger := logging.FromContext(r.Context())
	_ = i18n.FromRequest(r) // reserved for future localization
	sendErrorResponseWithDetails(w, logger, status, code, message, "")
}
