// SPDX-License-Identifier: BUSL-1.1

package api

// handlers_api_tokens.go implements the personal-access-token surface
// added in Phase D-2 of LICENSE_STRATEGY: API tokens for programmatic
// access (scripts, monitoring, CI). Minting requires the Pro tier;
// listing / revoking is available to any authenticated user for their
// own tokens.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/license"
	"github.com/krisarmstrong/seed/internal/logging"
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

// MintTokenRequest is the body of POST /api/v1/tokens.
type MintTokenRequest struct {
	Name string `json:"name"`
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
	if r.Method != http.MethodDelete {
		writeAPITokenError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed,
			"Method not allowed")
		return
	}
	s.handleAPITokenRevoke(w, r)
}

func (s *Server) handleAPITokenMint(w http.ResponseWriter, r *http.Request) {
	username := usernameFromContext(r)
	if username == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
			"Authentication required")
		return
	}

	if !s.licenseAllowsAPITokens() {
		writeAPITokenError(w, r, http.StatusPaymentRequired, "TIER_TOO_LOW",
			"API token minting requires the Pro tier. "+
				"Start a Pro trial with `seed license trial` or activate a Pro key.")
		return
	}

	repo := s.services.Auth.APITokens
	if repo == nil {
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			"API token storage is not available")
		return
	}

	var req MintTokenRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeBadRequest,
			"Invalid JSON body")
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
	}
	if insertErr := repo.Insert(r.Context(), rec); insertErr != nil {
		logger := logging.FromContext(r.Context())
		logger.ErrorContext(r.Context(), "failed to insert api token", "error", insertErr)
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to persist token")
		return
	}

	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusCreated, MintTokenResponse{
		ID:        rec.ID,
		Name:      rec.Name,
		Token:     plaintext,
		Prefix:    rec.Prefix,
		CreatedAt: rec.CreatedAt,
	})
}

func (s *Server) handleAPITokenList(w http.ResponseWriter, r *http.Request) {
	username := usernameFromContext(r)
	if username == "" {
		writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
			"Authentication required")
		return
	}

	repo := s.services.Auth.APITokens
	if repo == nil {
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
			[]TokenListItem{})
		return
	}

	rows, err := repo.ListByOwner(r.Context(), username)
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

	repo := s.services.Auth.APITokens
	if repo == nil {
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			"API token storage is not available")
		return
	}
	if err := repo.Revoke(r.Context(), id, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound,
				"Token not found or already revoked")
			return
		}
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to revoke token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// licenseAllowsAPITokens reports whether the active license tier is
// allowed to mint API tokens. Pro grants it; trial mode (which grants
// Pro-equivalent features) grants it. Free / Starter do not.
//
// A nil license manager is treated as "license disabled" (developer
// builds, tests) and permits minting so the feature stays usable
// without forcing a license setup.
func (s *Server) licenseAllowsAPITokens() bool {
	mgr := s.services.Auth.License
	if mgr == nil {
		return true
	}
	st := mgr.GetState()
	if st == nil {
		return false
	}
	return st.IsTrialMode || st.Tier >= license.TierPro
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

// resolveAPIToken returns the owning username for the given plaintext
// token, or "" if no active token matches. Token's last_used_at is
// touched on a successful lookup (best-effort; failures are logged
// but do not block the request).
func resolveAPIToken(ctx context.Context, repo *database.APITokenRepository, plaintext string) string {
	if repo == nil || !strings.HasPrefix(plaintext, APITokenPrefix) {
		return ""
	}
	rec, err := repo.FindActiveByHash(ctx, hashAPIToken(plaintext))
	if err != nil {
		return ""
	}
	if touchErr := repo.TouchLastUsed(ctx, rec.ID); touchErr != nil {
		logging.GetLogger().WarnContext(ctx, "failed to update api token last_used_at",
			"token_id", rec.ID, "error", touchErr)
	}
	return rec.OwnerUsername
}

// apiTokenMiddleware sits in front of the existing JWT auth middleware.
// If the request carries `Authorization: Bearer sd_pat_...`, it looks
// up the token, sets X-Username, and forwards to `next` so downstream
// handlers see an authenticated user. Otherwise it falls through to
// `next` unchanged, letting the JWT middleware handle the request.
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
		owner := resolveAPIToken(r.Context(), repo, token)
		if owner == "" {
			writeAPITokenError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized,
				"Invalid or revoked API token")
			return
		}
		ctx := logging.WithUserID(r.Context(), owner)
		r.Header.Set("X-Username", owner)
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
