// SPDX-License-Identifier: BUSL-1.1

package auth_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// A 24h SessionTimeout must not yield a 24h access JWT: the cookie lives 15
// minutes and the refresh token extends the session.
func TestAccessTokenLifetimeIsCapped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"default 24h session is capped", 24 * time.Hour, auth.AccessTokenDuration},
		{"shorter configured timeout is honoured", 5 * time.Minute, 5 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := auth.NewManager("test-secret-test-secret-test-secret", tc.timeout, "admin", "")
			t.Cleanup(m.Stop)

			for name, mint := range map[string]func() (string, error){
				"GenerateToken":       func() (string, error) { return m.GenerateToken(t.Context(), "admin") },
				"GenerateAccessToken": func() (string, error) { return m.GenerateAccessToken(t.Context(), "admin") },
			} {
				tok, err := mint()
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				claims, err := m.ValidateToken(t.Context(), tok)
				if err != nil {
					t.Fatalf("%s validate: %v", name, err)
				}
				if got := time.Until(claims.ExpiresAt.Time); got > tc.want || got < tc.want-time.Minute {
					t.Errorf("%s lifetime = %v, want about %v", name, got, tc.want)
				}
			}
		})
	}
}
