/**
 * Password policy validation shared by the setup wizard and any other
 * password-entry surface (recovery form, change-password dialog).
 *
 * Rules (fixes #723):
 * - At least MIN_LENGTH characters
 * - Contains at least one uppercase letter
 * - Contains at least one lowercase letter
 * - Contains at least one number
 * - Contains at least one special character
 */

export const PASSWORD_MIN_LENGTH = 12;

export interface PasswordRule {
  /** Stable identifier for translation lookup and React keys. */
  id: 'length' | 'uppercase' | 'lowercase' | 'number' | 'special';
  /** Whether this rule passes for the given password. */
  ok: boolean;
}

export interface PasswordPolicyResult {
  /** True only when every rule passes. */
  valid: boolean;
  /** Per-rule pass/fail, in display order. */
  rules: PasswordRule[];
}

const UPPERCASE_RE = /[A-Z]/;
const LOWERCASE_RE = /[a-z]/;
const NUMBER_RE = /[0-9]/;
// Anything that isn't an ASCII letter or digit counts as "special".
const SPECIAL_RE = /[^A-Za-z0-9]/;

export function evaluatePassword(password: string): PasswordPolicyResult {
  const rules: PasswordRule[] = [
    { id: 'length', ok: password.length >= PASSWORD_MIN_LENGTH },
    { id: 'uppercase', ok: UPPERCASE_RE.test(password) },
    { id: 'lowercase', ok: LOWERCASE_RE.test(password) },
    { id: 'number', ok: NUMBER_RE.test(password) },
    { id: 'special', ok: SPECIAL_RE.test(password) },
  ];
  return {
    valid: rules.every((r) => r.ok),
    rules,
  };
}

/** Convenience: returns true only if every rule passes. */
export function isPasswordValid(password: string): boolean {
  return evaluatePassword(password).valid;
}
