package auth_test

// Wave 2 / task #86 — zxcvbn password strength meter.

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/auth"
)

func TestEvaluatePasswordStrength_Weak(t *testing.T) {
	r := auth.EvaluatePasswordStrength("password", nil)
	if r.Score >= auth.MinPasswordScore {
		t.Errorf("expected weak password score < %d, got %d", auth.MinPasswordScore, r.Score)
	}
	if r.Warning == "" {
		t.Error("expected non-empty warning for weak password")
	}
	if len(r.Suggestions) == 0 {
		t.Error("expected at least one suggestion for weak password")
	}
}

func TestEvaluatePasswordStrength_Strong(t *testing.T) {
	// A genuinely strong, non-dictionary, mixed-class passphrase.
	r := auth.EvaluatePasswordStrength("correct-horse-battery-staple-9!Quokka", nil)
	if r.Score < auth.MinPasswordScore {
		t.Errorf("expected strong password score >= %d, got %d (guesses=%d)",
			auth.MinPasswordScore, r.Score, r.EstimatedGuesses)
	}
	if r.CrackTimeSeconds <= 0 {
		t.Errorf("expected positive crack time, got %v", r.CrackTimeSeconds)
	}
}

func TestEvaluatePasswordStrength_PenalizesUserInput(t *testing.T) {
	// Compare the same password evaluated WITHOUT vs WITH the username
	// in userInputs. zxcvbn must score the userInputs-aware case at
	// least as low (and typically lower) because the password derives
	// from the username token.
	const pwd = "kris.armstrong-kris.armstrong"
	without := auth.EvaluatePasswordStrength(pwd, nil)
	with := auth.EvaluatePasswordStrength(pwd, []string{"kris.armstrong"})

	if with.Score > without.Score {
		t.Errorf("userInput-aware score should be <= baseline; got with=%d without=%d",
			with.Score, without.Score)
	}
	if with.EstimatedGuesses > without.EstimatedGuesses {
		t.Errorf("userInput-aware guesses should be <= baseline; got with=%d without=%d",
			with.EstimatedGuesses, without.EstimatedGuesses)
	}
}
