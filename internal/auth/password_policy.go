// password_policy.go — composite password-set policy (task #86) for
// package auth.
//
// EnforcePasswordPolicy is the single chokepoint every password-set or
// password-change handler should call before storing a new credential.
// It runs the existing class/length check, the zxcvbn-based strength
// meter, and the HIBP breach-corpus lookup, and returns a structured
// rejection reason that callers translate into either an
// errors.password.weak or an errors.password.breached response.

package auth

import (
	"context"
	"errors"
	"fmt"
)

// PasswordRejectReason classifies why a password was refused at
// set/change time.
type PasswordRejectReason string

const (
	// PasswordRejectNone is the zero value (password accepted).
	PasswordRejectNone PasswordRejectReason = ""
	// PasswordRejectClassRule means the password failed the legacy
	// length+class check (ValidatePasswordStrength).
	PasswordRejectClassRule PasswordRejectReason = "class_rule"
	// PasswordRejectWeakScore means zxcvbn scored below MinPasswordScore.
	PasswordRejectWeakScore PasswordRejectReason = "weak_score"
	// PasswordRejectBreached means the password was found in the HIBP
	// breach corpus.
	PasswordRejectBreached PasswordRejectReason = "breached"
)

// PasswordPolicyResult is what handlers consume — it carries both the
// machine-readable rejection reason and a human-readable message
// suitable for surfacing to operators (the message is intentionally
// non-localised; handlers map it onto i18n keys).
type PasswordPolicyResult struct {
	// Accepted is true when the password passes all checks.
	Accepted bool
	// Reason is non-empty when Accepted is false.
	Reason PasswordRejectReason
	// Message is a short non-localised explanation suitable for logs
	// and for fallback display when no i18n key applies.
	Message string
	// Strength is the zxcvbn report (populated unconditionally so the
	// UI can render a meter even on accept).
	Strength PasswordStrength
	// BreachCount is the HIBP count when Reason == PasswordRejectBreached.
	BreachCount int
}

// ErrPasswordRejected is returned by EnforcePasswordPolicy when the
// password fails policy. Callers should inspect the returned
// PasswordPolicyResult for the specific reason.
var ErrPasswordRejected = errors.New("password rejected by policy")

// EnforcePasswordPolicy runs the full set of checks against a candidate
// password and returns a PasswordPolicyResult. When the password is
// rejected the error wraps ErrPasswordRejected.
//
// userInputs is the list of identity strings (username, email, display
// name) used by zxcvbn to penalise passwords derived from them.
//
// HIBP failures are non-fatal (see hibp.go); they will not produce a
// rejection.
func EnforcePasswordPolicy(
	ctx context.Context,
	password string,
	userInputs []string,
) (PasswordPolicyResult, error) {
	// 1) Legacy class+length rule (12+ chars, mixed classes). Still
	// enforced because zxcvbn's "score 3" alone permits short passwords
	// when they are sufficiently unguessable, which violates our
	// project minimum.
	if classErr := ValidatePasswordStrength(password); classErr != nil {
		return PasswordPolicyResult{
			Accepted: false,
			Reason:   PasswordRejectClassRule,
			Message: fmt.Sprintf(
				"password must be at least %d characters and contain upper, lower, digit, and symbol",
				MinPasswordLength,
			),
			Strength: EvaluatePasswordStrength(password, userInputs),
		}, fmt.Errorf("%w: class rule", ErrPasswordRejected)
	}

	// 2) zxcvbn strength score.
	strength := EvaluatePasswordStrength(password, userInputs)
	if strength.Score < MinPasswordScore {
		msg := fmt.Sprintf(
			"password is too easy to guess (score %d/%d)",
			strength.Score,
			MinPasswordScore,
		)
		if strength.Warning != "" {
			msg += ": " + strength.Warning
		}
		if len(strength.Suggestions) > 0 {
			msg += " — " + strength.Suggestions[0]
		}
		return PasswordPolicyResult{
			Accepted: false,
			Reason:   PasswordRejectWeakScore,
			Message:  msg,
			Strength: strength,
		}, fmt.Errorf("%w: weak score", ErrPasswordRejected)
	}

	// 3) HIBP k-anonymity lookup. Network failures are non-fatal and
	// return (false, 0, nil) — see hibp.go.
	breached, count, _ := CheckPasswordBreached(ctx, password)
	if breached {
		return PasswordPolicyResult{
			Accepted: false,
			Reason:   PasswordRejectBreached,
			Message: fmt.Sprintf(
				"password has appeared in %d known data breaches; choose a different one",
				count,
			),
			Strength:    strength,
			BreachCount: count,
		}, fmt.Errorf("%w: breached corpus", ErrPasswordRejected)
	}

	return PasswordPolicyResult{
		Accepted: true,
		Strength: strength,
	}, nil
}
