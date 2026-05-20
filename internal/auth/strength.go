// strength.go — password strength evaluation for package auth.
//
// Wraps github.com/trustelem/zxcvbn to produce a structured strength
// report. The trustelem port intentionally omits zxcvbn's feedback
// messages (per its README), so this layer synthesizes a short,
// score-appropriate warning + suggestion list locally.

package auth

import (
	"github.com/trustelem/zxcvbn"
)

// MinPasswordScore is the minimum zxcvbn score (0-4) required before
// a password is accepted at set/change time. Score 3 corresponds to
// "safely unguessable" per the zxcvbn paper.
const MinPasswordScore = 3

// zxcvbn score thresholds used by strengthFeedback.
const (
	zxcvbnScoreExtremelyGuessable = 0 // "too guessable: risky password"
	zxcvbnScoreVeryGuessable      = 1 // "very guessable: protection from throttled online attacks"
	zxcvbnScoreSomewhatGuessable  = 2 // "somewhat guessable: protection from unthrottled online attacks"
)

// Crack-time scenario used in the public report. Matches zxcvbn's
// canonical "offline_slow_hashing_1e4_per_second" attacker — i.e. an
// attacker running offline against a slow KDF like Argon2id at
// ~10,000 guesses/sec.
const offlineSlowHashRate = 1e4

// PasswordStrength is the structured result of EvaluatePasswordStrength.
//
// Score, EstimatedGuesses and CrackTimeSeconds are sourced from zxcvbn.
// Warning + Suggestions are synthesised locally because the Go port
// omits feedback messages.
type PasswordStrength struct {
	// Score is the zxcvbn score, 0 (worst) through 4 (best).
	Score int `json:"score"`
	// EstimatedGuesses is zxcvbn's guess estimate for the password.
	EstimatedGuesses int64 `json:"estimated_guesses"`
	// Warning is a short human-readable warning (may be empty).
	Warning string `json:"warning,omitempty"`
	// Suggestions are improvement suggestions (may be empty).
	Suggestions []string `json:"suggestions,omitempty"`
	// CrackTimeSeconds is the offline-slow-hash crack time estimate
	// in seconds (zxcvbn's "offline_slow_hashing_1e4_per_second").
	CrackTimeSeconds float64 `json:"crack_time_seconds"`
}

// EvaluatePasswordStrength runs zxcvbn against the given password.
//
// userInputs should be the username, email, and any display-name-like
// strings tied to the account so zxcvbn can penalise passwords that
// derive from them.
func EvaluatePasswordStrength(password string, userInputs []string) PasswordStrength {
	result := zxcvbn.PasswordStrength(password, userInputs)

	// zxcvbn returns Guesses as a float64 to avoid overflow on huge
	// passwords. Clamp to int64 max defensively.
	var guesses int64
	const maxInt64Float = float64(1<<63 - 1)
	switch {
	case result.Guesses < 0:
		guesses = 0
	case result.Guesses > maxInt64Float:
		guesses = int64(1<<63 - 1)
	default:
		guesses = int64(result.Guesses)
	}

	warning, suggestions := strengthFeedback(result.Score)
	crackTime := result.Guesses / offlineSlowHashRate

	return PasswordStrength{
		Score:            result.Score,
		EstimatedGuesses: guesses,
		Warning:          warning,
		Suggestions:      suggestions,
		CrackTimeSeconds: crackTime,
	}
}

// strengthFeedback returns a short warning + suggestion list keyed off
// the zxcvbn score. Kept score-only because the trustelem port does
// not expose pattern-level feedback (top-N-common, dictionary match,
// etc.).
func strengthFeedback(score int) (string, []string) {
	switch score {
	case zxcvbnScoreExtremelyGuessable:
		return "This password is extremely guessable.",
			[]string{
				"Use a longer passphrase made of unrelated words.",
				"Avoid common words, names, and keyboard patterns.",
			}
	case zxcvbnScoreVeryGuessable:
		return "This password is very guessable.",
			[]string{
				"Add more length — 14+ characters is much harder to crack.",
				"Avoid substitutions like 'P@ssw0rd'; modern crackers know them.",
			}
	case zxcvbnScoreSomewhatGuessable:
		return "This password is somewhat guessable.",
			[]string{
				"Use a longer passphrase of 4+ unrelated words.",
				"Mix in symbols or numbers in unpredictable positions.",
			}
	case MinPasswordScore:
		return "", nil
	default:
		return "", nil
	}
}
