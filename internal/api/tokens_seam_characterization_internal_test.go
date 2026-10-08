// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// PAT authentication seam behaviour, pinned before the token record and its
// lookup moved behind the identity/tokens use-case (seed#2750): a resolved
// token is stamped as used, and a revoked token or an unwired store is 401.

func patRequest(plaintext string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	return req
}

func TestAPITokenSeamStampsLastUsed(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)
	plaintext := insertPAT(t, s, "carol")

	w := httptest.NewRecorder()
	apiTokenMiddleware(s.apiTokens, s.resolveClientID,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, patRequest(plaintext))
	require.Equal(t, http.StatusOK, w.Code)

	listed, err := s.apiTokens.ListByOwner(t.Context(), "carol")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.False(t, listed[0].LastUsedAt.IsZero(), "a resolved PAT must be stamped as used")
}

func TestAPITokenSeamRejectsRevoked(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)
	plaintext := insertPAT(t, s, "carol")
	listed, err := s.apiTokens.ListByOwner(t.Context(), "carol")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.NoError(t, s.apiTokens.Revoke(t.Context(), listed[0].ID, "carol"))

	called := false
	w := httptest.NewRecorder()
	apiTokenMiddleware(s.apiTokens, s.resolveClientID,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })).ServeHTTP(w, patRequest(plaintext))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.False(t, called)
}

func TestAPITokenSeamRejectsWithoutStore(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)
	plaintext := insertPAT(t, s, "carol")
	s.apiTokens = nil

	called := false
	w := httptest.NewRecorder()
	apiTokenMiddleware(s.apiTokens, s.resolveClientID,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })).ServeHTTP(w, patRequest(plaintext))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.False(t, called)
}
