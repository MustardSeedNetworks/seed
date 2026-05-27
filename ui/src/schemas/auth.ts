/**
 * Valibot schemas for seed's auth / recovery / setup forms.
 *
 * Cover the user-input boundary of the login flow: login (username +
 * password), MFA verification, recovery code entry, and the
 * initial-setup wizard. Mirrors stem/src/schemas/auth.ts so the two
 * code bases stay aligned.
 *
 * The Go side validates again on receipt — these schemas exist for
 * the UI to show inline per-field errors before the network call, not
 * as the security boundary.
 */
import * as v from 'valibot';

/** 6-digit TOTP code. Whitespace is trimmed (users often paste with
 * spaces from authenticator apps). Stored without separators. */
export const TotpCodeSchema = v.pipe(
  v.string('Code is required'),
  v.trim(),
  v.regex(/^\d{6}$/, 'Code must be exactly 6 digits'),
);

/** Username for login + recovery flows. */
export const UsernameSchema = v.pipe(
  v.string('Username is required'),
  v.trim(),
  v.minLength(1, 'Username is required'),
  v.maxLength(128, 'Username is too long'),
);

/** Password for login flows. Length-only check; the Go side enforces
 * complexity rules. */
export const PasswordSchema = v.pipe(
  v.string('Password is required'),
  v.minLength(1, 'Password is required'),
  v.maxLength(512, 'Password is too long'),
);

/** Recovery code: 16 hex characters in groups of 4. Accept with or
 * without separators on input; normalize before posting. */
export const RecoveryCodeSchema = v.pipe(
  v.string('Recovery code is required'),
  v.trim(),
  v.transform((s) => s.replace(/[-\s]/g, '').toLowerCase()),
  v.regex(/^[0-9a-f]{16}$/, 'Recovery code must be 16 hex characters'),
);

// =============================================================================
// Form schemas
// =============================================================================

export const LoginSchema = v.object({
  username: UsernameSchema,
  password: PasswordSchema,
});

export const MfaVerifySchema = v.object({
  code: TotpCodeSchema,
});

/**
 * Recovery completion: filesystem-token-based password reset. The
 * admin writes the token to a file on the server, the user pastes the
 * token here, then enters and confirms a new password. Password
 * confirmation is a cross-field check; the resolver surfaces it
 * under formState.errors.root.
 */
export const RecoveryCompleteSchema = v.pipe(
  v.object({
    token: v.pipe(
      v.string('Recovery token is required'),
      v.trim(),
      v.minLength(1, 'Recovery token is required'),
    ),
    password: v.pipe(
      v.string('Password is required'),
      v.minLength(12, 'Password must be at least 12 characters'),
      v.maxLength(512, 'Password is too long'),
    ),
    confirmPassword: v.string(),
  }),
  v.check((c) => c.password === c.confirmPassword, 'Passwords do not match'),
);

/**
 * Setup wizard: initial-admin creation. Username is fixed from the
 * setup status (env-configured), so the schema only covers the
 * password the user actually types.
 */
export const SetupWizardSchema = v.pipe(
  v.object({
    password: v.pipe(
      v.string('Password is required'),
      v.minLength(12, 'Password must be at least 12 characters'),
      v.maxLength(512, 'Password is too long'),
    ),
    confirmPassword: v.string(),
  }),
  v.check((c) => c.password === c.confirmPassword, 'Passwords do not match'),
);

/** Profile editor: name + optional description for survey/path profiles. */
export const ProfileEditorSchema = v.object({
  name: v.pipe(
    v.string('Name is required'),
    v.trim(),
    v.minLength(1, 'Name is required'),
    v.maxLength(64, 'Name is too long (max 64 chars)'),
  ),
  description: v.pipe(v.string(), v.maxLength(256, 'Description is too long (max 256 chars)')),
  isDefault: v.boolean(),
  notes: v.pipe(v.string(), v.maxLength(2048, 'Notes are too long (max 2048 chars)')),
});

/** Wi-Fi survey creation form. */
export const CreateSurveySchema = v.object({
  name: v.pipe(
    v.string('Survey name is required'),
    v.trim(),
    v.minLength(1, 'Survey name is required'),
    v.maxLength(128, 'Survey name is too long (max 128 chars)'),
  ),
  surveyType: v.picklist(['passive', 'active', 'throughput']),
});

/**
 * Path discovery form. Target is a hostname or IPv4/IPv6 address. The
 * Go side does authoritative parsing — this UI check just blocks
 * obviously malformed input (empty, too long, contains spaces).
 */
export const PathDiscoverySchema = v.object({
  target: v.pipe(
    v.string('Target is required'),
    v.trim(),
    v.minLength(1, 'Target is required'),
    v.maxLength(253, 'Target is too long (max 253 chars)'),
    v.regex(/^[^\s]+$/, 'Target cannot contain whitespace'),
  ),
  protocol: v.picklist(['icmp', 'tcp', 'udp']),
  port: v.pipe(
    v.number('Port must be a number'),
    v.integer('Port must be an integer'),
    v.minValue(1, 'Port must be 1-65535'),
    v.maxValue(65535, 'Port must be 1-65535'),
  ),
});
