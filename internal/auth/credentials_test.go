package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	fndpw "github.com/MustardSeedNetworks/foundation/pkg/password"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

const credentialsUser = "operator"

// TestAuthenticateCredentials drives both login paths (UserStore-backed
// and in-memory) through foundation's password records. A bcrypt record
// is no longer accepted: pre-v1 installs carry no legacy hashes.
func TestAuthenticateCredentials(t *testing.T) {
	const correct = "Correct-Horse-Battery-9"
	record, err := fndpw.Hash(correct)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	bcryptRecord := "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW"

	tests := []struct {
		name     string
		stored   string
		password string
		wantOK   bool
	}{
		{"correct password", record, correct, true},
		{"wrong password", record, "Wrong-Horse-Battery-9", false},
		{"empty password", record, "", false},
		{"oversize password", record, strings.Repeat("a", fndpw.MaxBytes+1), false},
		{"bcrypt record rejected", bcryptRecord, correct, false},
		{"setup placeholder", auth.SetupModePlaceholder, correct, false},
		{"empty record", "", correct, false},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/store", func(t *testing.T) {
			store := newMockUserStore()
			store.passwords[credentialsUser] = tt.stored
			m := auth.NewManager("test-secret-please-change", 0, credentialsUser, "")
			m.SetUserStore(store)
			assertLogin(t, m, tt.password, tt.wantOK)
		})
		t.Run(tt.name+"/memory", func(t *testing.T) {
			m := auth.NewManager("test-secret-please-change", 0, credentialsUser, tt.stored)
			assertLogin(t, m, tt.password, tt.wantOK)
		})
	}
}

// TestValidatePasswordStrengthMaxBytes keeps the strength rule's upper
// bound at the most foundation will hash, so an oversize password is a
// 400 at set time rather than a hashing failure.
func TestValidatePasswordStrengthMaxBytes(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"at the limit", fndpw.MaxBytes, false},
		{"one byte over", fndpw.MaxBytes + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := "Aa1!" + strings.Repeat("x", tt.length-4)
			if err := auth.ValidatePasswordStrength(candidate); (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePasswordStrength(%d bytes) = %v, wantErr %v", tt.length, err, tt.wantErr)
			}
			if _, err := fndpw.Hash(candidate); (err != nil) != tt.wantErr {
				t.Fatalf("Hash(%d bytes) = %v, wantErr %v", tt.length, err, tt.wantErr)
			}
		})
	}
}

func assertLogin(t *testing.T, m *auth.Manager, password string, wantOK bool) {
	t.Helper()
	ctx := context.Background()
	tok, err := m.Authenticate(ctx, credentialsUser, password)
	onlyErr := m.VerifyPasswordOnly(ctx, credentialsUser, password)
	if wantOK {
		if err != nil || tok == "" || onlyErr != nil {
			t.Fatalf("want success; Authenticate=(%q, %v) VerifyPasswordOnly=%v", tok, err, onlyErr)
		}
		return
	}
	if !errors.Is(err, auth.ErrInvalidCredentials) || !errors.Is(onlyErr, auth.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials; Authenticate=%v VerifyPasswordOnly=%v", err, onlyErr)
	}
}
