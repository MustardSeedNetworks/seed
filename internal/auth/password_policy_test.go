package auth_test

// Wave 2 / task #86 — combined password policy (class + zxcvbn + HIBP).

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/seed/internal/auth"
)

// hibpAlwaysClean returns a server that says "no match" for everything.
func hibpAlwaysClean(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// A single padding-style row that won't match any real password.
		_, _ = w.Write([]byte("0000000000000000000000000000000000A:0\r\n"))
	}))
}

func TestEnforcePasswordPolicy_AcceptsStrongPassword(t *testing.T) {
	srv := hibpAlwaysClean(t)
	defer srv.Close()
	defer auth.SetHIBPEndpointForTest(srv.URL + "/")()
	t.Setenv("SEED_DISABLE_HIBP", "")

	result, err := auth.EnforcePasswordPolicy(
		context.Background(),
		"correct-horse-battery-staple-9!Quokka",
		[]string{"admin"},
	)
	if err != nil {
		t.Fatalf("expected accept, got error %v (reason=%s detail=%s)",
			err, result.Reason, result.Message)
	}
	if !result.Accepted {
		t.Errorf("expected Accepted=true; result=%+v", result)
	}
}

func TestEnforcePasswordPolicy_RejectsOnClassRule(t *testing.T) {
	// "short" fails length+class — should NOT even reach zxcvbn/HIBP.
	defer auth.SetHIBPEndpointForTest("http://127.0.0.1:1/")()
	t.Setenv("SEED_DISABLE_HIBP", "")

	result, err := auth.EnforcePasswordPolicy(
		context.Background(),
		"short",
		[]string{"admin"},
	)
	if !errors.Is(err, auth.ErrPasswordRejected) {
		t.Fatalf("expected ErrPasswordRejected, got %v", err)
	}
	if result.Reason != auth.PasswordRejectClassRule {
		t.Errorf("expected reason=class_rule, got %q", result.Reason)
	}
}

func TestEnforcePasswordPolicy_RejectsOnWeakScore(t *testing.T) {
	srv := hibpAlwaysClean(t)
	defer srv.Close()
	defer auth.SetHIBPEndpointForTest(srv.URL + "/")()
	t.Setenv("SEED_DISABLE_HIBP", "")

	// Long enough to pass length+class but a well-known weak base.
	result, err := auth.EnforcePasswordPolicy(
		context.Background(),
		"Password1234!",
		[]string{"admin"},
	)
	if !errors.Is(err, auth.ErrPasswordRejected) {
		t.Fatalf("expected ErrPasswordRejected, got %v (score=%d)",
			err, result.Strength.Score)
	}
	if result.Reason != auth.PasswordRejectWeakScore {
		t.Errorf("expected reason=weak_score, got %q (score=%d)",
			result.Reason, result.Strength.Score)
	}
}

func TestEnforcePasswordPolicy_RejectsOnBreached(t *testing.T) {
	// Use the real "password" with a server that responds to its
	// specific suffix — reusing the helper from hibp_test.go.
	srv := hibpServer(t, "Tr0ub4dor&3-Long-Enough-And-Strong-2026!", 12345)
	defer srv.Close()
	defer auth.SetHIBPEndpointForTest(srv.URL + "/")()
	t.Setenv("SEED_DISABLE_HIBP", "")

	result, err := auth.EnforcePasswordPolicy(
		context.Background(),
		"Tr0ub4dor&3-Long-Enough-And-Strong-2026!",
		[]string{"admin"},
	)
	if !errors.Is(err, auth.ErrPasswordRejected) {
		t.Fatalf("expected ErrPasswordRejected, got %v (score=%d)", err, result.Strength.Score)
	}
	if result.Reason != auth.PasswordRejectBreached {
		t.Errorf("expected reason=breached, got %q", result.Reason)
	}
	if result.BreachCount != 12345 {
		t.Errorf("expected BreachCount=12345, got %d", result.BreachCount)
	}
}
