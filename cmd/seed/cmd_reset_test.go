package main

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestResetFlagsStruct(t *testing.T) {
	t.Parallel()

	// Test default values
	flags := resetFlags{}

	if flags.preserveAuth {
		t.Error("preserveAuth should default to false")
	}
	if flags.preserveJWT {
		t.Error("preserveJWT should default to false")
	}
	if flags.backup {
		t.Error("backup should default to false")
	}
	if flags.force {
		t.Error("force should default to false")
	}
}

func TestResetFlagsAllTrue(t *testing.T) {
	t.Parallel()

	flags := resetFlags{
		preserveAuth: true,
		preserveJWT:  true,
		backup:       true,
		force:        true,
	}

	if !flags.preserveAuth {
		t.Error("preserveAuth should be true")
	}
	if !flags.preserveJWT {
		t.Error("preserveJWT should be true")
	}
	if !flags.backup {
		t.Error("backup should be true")
	}
	if !flags.force {
		t.Error("force should be true")
	}
}

func TestPreserveExistingCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		newCfg      *config.Config
		existingCfg *config.Config
		flags       resetFlags
		wantAuth    bool
		wantJWT     bool
	}{
		{
			name:   "nil existing config",
			newCfg: &config.Config{},
			flags: resetFlags{
				preserveAuth: true,
				preserveJWT:  true,
			},
			wantAuth: false,
			wantJWT:  false,
		},
		{
			name: "preserve auth only",
			newCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "",
					DefaultPasswordHash: "",
					JWTSecret:           "",
				},
			},
			existingCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "admin",
					DefaultPasswordHash: "existing-hash",
					JWTSecret:           "existing-jwt",
				},
			},
			flags: resetFlags{
				preserveAuth: true,
				preserveJWT:  false,
			},
			wantAuth: true,
			wantJWT:  false,
		},
		{
			name: "preserve jwt only",
			newCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "",
					DefaultPasswordHash: "",
					JWTSecret:           "",
				},
			},
			existingCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "admin",
					DefaultPasswordHash: "existing-hash",
					JWTSecret:           "existing-jwt",
				},
			},
			flags: resetFlags{
				preserveAuth: false,
				preserveJWT:  true,
			},
			wantAuth: false,
			wantJWT:  true,
		},
		{
			name: "preserve both",
			newCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "",
					DefaultPasswordHash: "",
					JWTSecret:           "",
				},
			},
			existingCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "admin",
					DefaultPasswordHash: "existing-hash",
					JWTSecret:           "existing-jwt",
				},
			},
			flags: resetFlags{
				preserveAuth: true,
				preserveJWT:  true,
			},
			wantAuth: true,
			wantJWT:  true,
		},
		{
			name: "preserve neither",
			newCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "",
					DefaultPasswordHash: "",
					JWTSecret:           "",
				},
			},
			existingCfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "admin",
					DefaultPasswordHash: "existing-hash",
					JWTSecret:           "existing-jwt",
				},
			},
			flags: resetFlags{
				preserveAuth: false,
				preserveJWT:  false,
			},
			wantAuth: false,
			wantJWT:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Make a copy to avoid modifying the test case
			newCfg := &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     tc.newCfg.Auth.DefaultUsername,
					DefaultPasswordHash: tc.newCfg.Auth.DefaultPasswordHash,
					JWTSecret:           tc.newCfg.Auth.JWTSecret,
				},
			}

			preserveExistingCredentials(newCfg, tc.existingCfg, tc.flags)
			assertPreservedCredentials(t, tc, newCfg)
		})
	}
}

func assertPreservedCredentials(t *testing.T, tc struct {
	name        string
	newCfg      *config.Config
	existingCfg *config.Config
	flags       resetFlags
	wantAuth    bool
	wantJWT     bool
}, newCfg *config.Config) {
	t.Helper()

	if tc.existingCfg == nil {
		return
	}

	assertAuthPreservation(t, tc, newCfg)
	assertJWTPreservation(t, tc, newCfg)
}

func assertAuthPreservation(t *testing.T, tc struct {
	name        string
	newCfg      *config.Config
	existingCfg *config.Config
	flags       resetFlags
	wantAuth    bool
	wantJWT     bool
}, newCfg *config.Config) {
	t.Helper()

	if tc.wantAuth {
		if newCfg.Auth.DefaultUsername != tc.existingCfg.Auth.DefaultUsername {
			t.Errorf(
				"DefaultUsername should be preserved: got %q, want %q",
				newCfg.Auth.DefaultUsername,
				tc.existingCfg.Auth.DefaultUsername,
			)
		}
		if newCfg.Auth.DefaultPasswordHash != tc.existingCfg.Auth.DefaultPasswordHash {
			t.Errorf(
				"DefaultPasswordHash should be preserved: got %q, want %q",
				newCfg.Auth.DefaultPasswordHash,
				tc.existingCfg.Auth.DefaultPasswordHash,
			)
		}
		return
	}

	if newCfg.Auth.DefaultUsername == tc.existingCfg.Auth.DefaultUsername &&
		tc.existingCfg.Auth.DefaultUsername != "" {
		t.Error("DefaultUsername should not be preserved")
	}
	if newCfg.Auth.DefaultPasswordHash == tc.existingCfg.Auth.DefaultPasswordHash &&
		tc.existingCfg.Auth.DefaultPasswordHash != "" {
		t.Error("DefaultPasswordHash should not be preserved")
	}
}

func assertJWTPreservation(t *testing.T, tc struct {
	name        string
	newCfg      *config.Config
	existingCfg *config.Config
	flags       resetFlags
	wantAuth    bool
	wantJWT     bool
}, newCfg *config.Config) {
	t.Helper()

	if tc.wantJWT {
		if newCfg.Auth.JWTSecret != tc.existingCfg.Auth.JWTSecret {
			t.Errorf(
				"JWTSecret should be preserved: got %q, want %q",
				newCfg.Auth.JWTSecret,
				tc.existingCfg.Auth.JWTSecret,
			)
		}
		return
	}

	if newCfg.Auth.JWTSecret == tc.existingCfg.Auth.JWTSecret && tc.existingCfg.Auth.JWTSecret != "" {
		t.Error("JWTSecret should not be preserved")
	}
}

func TestPreserveExistingCredentialsNilExisting(t *testing.T) {
	t.Parallel()

	newCfg := &config.Config{
		Auth: config.AuthConfig{
			DefaultUsername:     "new-user",
			DefaultPasswordHash: "new-hash",
			JWTSecret:           "new-jwt",
		},
	}

	originalUser := newCfg.Auth.DefaultUsername
	originalHash := newCfg.Auth.DefaultPasswordHash
	originalJWT := newCfg.Auth.JWTSecret

	flags := resetFlags{
		preserveAuth: true,
		preserveJWT:  true,
	}

	preserveExistingCredentials(newCfg, nil, flags)

	// With nil existing config, new values should remain unchanged
	if newCfg.Auth.DefaultUsername != originalUser {
		t.Errorf("Username should remain unchanged: got %q, want %q", newCfg.Auth.DefaultUsername, originalUser)
	}
	if newCfg.Auth.DefaultPasswordHash != originalHash {
		t.Errorf("Hash should remain unchanged: got %q, want %q", newCfg.Auth.DefaultPasswordHash, originalHash)
	}
	if newCfg.Auth.JWTSecret != originalJWT {
		t.Errorf("JWT should remain unchanged: got %q, want %q", newCfg.Auth.JWTSecret, originalJWT)
	}
}

func TestResetFlagsCombinations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		preserveAuth bool
		preserveJWT  bool
		backup       bool
		force        bool
	}{
		{"all false", false, false, false, false},
		{"auth only", true, false, false, false},
		{"jwt only", false, true, false, false},
		{"backup only", false, false, true, false},
		{"force only", false, false, false, true},
		{"auth and jwt", true, true, false, false},
		{"backup and force", false, false, true, true},
		{"all true", true, true, true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flags := resetFlags{
				preserveAuth: tc.preserveAuth,
				preserveJWT:  tc.preserveJWT,
				backup:       tc.backup,
				force:        tc.force,
			}

			if flags.preserveAuth != tc.preserveAuth {
				t.Errorf("preserveAuth: got %v, want %v", flags.preserveAuth, tc.preserveAuth)
			}
			if flags.preserveJWT != tc.preserveJWT {
				t.Errorf("preserveJWT: got %v, want %v", flags.preserveJWT, tc.preserveJWT)
			}
			if flags.backup != tc.backup {
				t.Errorf("backup: got %v, want %v", flags.backup, tc.backup)
			}
			if flags.force != tc.force {
				t.Errorf("force: got %v, want %v", flags.force, tc.force)
			}
		})
	}
}
