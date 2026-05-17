/**
 * Initial Setup Wizard Component
 *
 * Guides users through the first-time setup process for The Seed application.
 *
 * Features:
 * - Password setup with validation (minimum 12 characters)
 * - Password confirmation requirement
 * - Generated password suggestion option
 * - Custom password entry mode
 * - Automatic login after setup completion
 * - Error handling and user feedback
 *
 * Flow:
 * 1. User enters password (or accepts suggested password)
 * 2. Confirms password matches
 * 3. SetupWizard sends POST /api/setup/complete with new password
 * 4. Server hashes and stores password
 * 5. Component automatically logs in user
 * 6. Calls onComplete callback to exit setup flow
 *
 * The wizard is shown when the system detects initial setup is needed
 * (no admin password configured). It's displayed before the main application.
 */

import { Activity, Copy, Eye, EyeOff, Lock, Zap } from 'lucide-react';
import type React from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { evaluatePassword, type PasswordRule } from '../../lib/passwordPolicy';
import {
  button,
  buttonClass,
  cardClass,
  cn,
  icon as iconTokens,
  inputClass,
  layout,
  radius,
  spacing,
} from '../../styles/theme';

// API base URL for setup endpoints
const API_BASE: string = import.meta.env.VITE_API_BASE || '';

/**
 * Props for SetupWizard component
 */
interface SetupWizardProps {
  /** Callback invoked when setup is complete and user is logged in */
  onComplete: () => void;
  /** Function to attempt login after password is set */
  onLogin: (username: string, password: string) => Promise<boolean>;
  /** Optional pre-generated password suggestion to offer user */
  suggestedPassword?: string;
  /** Username from config (fixes #768 - no hardcoded 'admin') */
  username?: string;
  /** Security fix #724, #758: One-time setup token required for setup completion */
  setupToken?: string;
}

/**
 * SetupWizard Component
 *
 * Modal-like component that requires user to set admin password before
 * accessing the main application.
 */
// SSO providers from backend (/api/sso/providers returns the names of
// only the providers that are enabled AND have a ClientID configured -
// see internal/api/handlers_oauth.go::initOAuthManager). Fixes #720.

/**
 * First-run setup flow that forces the user to create credentials before using the app.
 */
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Single-screen wizard with several conditional UI blocks; refactoring split is tracked separately.
export function SetupWizard({
  onComplete,
  onLogin,
  suggestedPassword,
  username = 'admin',
  setupToken,
}: SetupWizardProps): React.JSX.Element {
  const { t } = useTranslation('setup');
  // Default to custom password entry - more secure UX
  const [passwordMode, setPasswordMode] = useState<'generated' | 'custom'>('custom');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);
  const [ssoProviders, setSsoProviders] = useState<string[]>([]);

  // Fetch enabled SSO providers (fixes #769, #720)
  useEffect(() => {
    fetch(`${API_BASE}/api/sso/providers`)
      .then((res) => (res.ok ? res.json() : { providers: [] }))
      .then((data: { providers?: string[] }) => setSsoProviders(data.providers ?? []))
      .catch(() => setSsoProviders([]));
  }, []);

  // Update password fields when switching to generated mode
  useEffect(() => {
    if (passwordMode === 'generated' && suggestedPassword) {
      setPassword(suggestedPassword);
      setConfirmPassword(suggestedPassword);
    }
  }, [passwordMode, suggestedPassword]);

  // Helper to check if a provider is enabled. The backend only lists
  // providers that have Enabled=true AND a non-empty ClientID, so
  // presence in the list is sufficient (#720).
  const isProviderEnabled = (name: string): boolean =>
    ssoProviders.some((p) => p.toLowerCase() === name.toLowerCase());

  // Check if any SSO provider is enabled
  const hasEnabledSso = ssoProviders.length > 0;

  const handleCopyPassword = async (): Promise<void> => {
    if (suggestedPassword) {
      // biome-ignore lint/nursery/useAwaitThenable: navigator.clipboard.writeText returns a Promise
      await navigator.clipboard.writeText(suggestedPassword);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handlePasswordModeChange = (mode: 'generated' | 'custom'): void => {
    setPasswordMode(mode);
    if (mode === 'generated' && suggestedPassword) {
      setPassword(suggestedPassword);
      setConfirmPassword(suggestedPassword);
      setShowPassword(true);
    } else {
      setPassword('');
      setConfirmPassword('');
      setShowPassword(false);
    }
    setError(null);
  };

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    setError(null);

    // Fixes #723: enforce complexity rules, not just length.
    const policy = evaluatePassword(password);
    if (!policy.valid) {
      setError(t('errors.passwordTooShort'));
      return;
    }

    if (password !== confirmPassword) {
      setError(t('errors.passwordMismatch'));
      return;
    }

    setIsSubmitting(true);

    try {
      // Step 1: Complete setup (set password on server)
      // Security fix #724, #758: Include the one-time setup token to prevent CSRF attacks
      const response = await fetch(`${API_BASE}/api/setup/complete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ password, setupToken }),
      });

      if (!response.ok) {
        // biome-ignore lint/nursery/useAwaitThenable: response.json() returns a Promise
        const data = await response.json();
        setError(data.error || t('errors.setupFailed'));
        return;
      }

      // Step 2: Automatically log in with the new password (fixes #768 - use username from config)
      const loginSuccess = await onLogin(username, password);

      if (!loginSuccess) {
        // Fixes #719: do not exit the wizard when auto-login fails - the user
        // would land in the main UI unauthenticated. Keep the wizard open with
        // the error visible so they can retry or surface the real failure.
        setError(t('errors.loginFailed'));
        return;
      }

      // Step 3: Setup complete and user is logged in
      onComplete();
    } catch {
      setError(t('errors.networkError'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div class={cn('min-h-screen bg-surface-base', layout.flex.center, 'pad')}>
      <div class="w-full max-w-md">
        <div class={cn('text-center', spacing.margin.bottom.sectionLg)}>
          <div class="w-16 h-16 mx-auto flex items-center justify-center rounded-2xl bg-brand-primary text-text-inverse">
            <Activity class="w-8 h-8" />
          </div>
          <h1 class={cn('heading-2', spacing.margin.top.heading)}>{t('welcome.title')}</h1>
          <p class={cn('body-small', spacing.margin.top.inline)}>{t('welcome.subtitle')}</p>
        </div>

        <form onSubmit={handleSubmit} class={cardClass('default', 'lg')}>
          <div class={spacing.margin.bottom.content}>
            <p class={cn('body-small', spacing.margin.bottom.content)}>
              {t('username.label')} <strong>{username}</strong> {t('username.cannotChange')}
            </p>
          </div>

          {/* Password mode selection */}
          <div class={cn(spacing.margin.bottom.section, 'stack-sm')}>
            <p class={cn('body-small font-medium text-text-primary', spacing.margin.bottom.inline)}>
              {t('password.chooseMethod')}
            </p>

            {/* Custom password option */}
            <label
              class={cn(
                'flex items-start',
                spacing.gap.default,
                'pad-sm',
                radius.md,
                'border border-surface-border cursor-pointer hover:bg-surface-base transition-colors',
              )}
            >
              <input
                type="radio"
                name="passwordMode"
                value="custom"
                checked={passwordMode === 'custom'}
                onChange={(): void => handlePasswordModeChange('custom')}
                class={cn(
                  spacing.margin.top.inline,
                  iconTokens.size.sm,
                  'text-brand-primary focus:ring-brand-primary',
                )}
              />
              <div>
                <span class="body-small font-medium text-text-primary flex items-center gap-2">
                  <Lock class={iconTokens.size.sm} />
                  {t('password.custom.title')}
                </span>
                <p class={cn('caption text-text-muted', spacing.margin.top.inline)}>
                  {t('password.custom.description')}
                </p>
              </div>
            </label>

            {/* Generated password option */}
            {suggestedPassword ? (
              <label
                class={cn(
                  'flex items-start',
                  spacing.gap.default,
                  'pad-sm',
                  radius.md,
                  'border border-surface-border cursor-pointer hover:bg-surface-base transition-colors',
                )}
              >
                <input
                  type="radio"
                  name="passwordMode"
                  value="generated"
                  checked={passwordMode === 'generated'}
                  onChange={(): void => handlePasswordModeChange('generated')}
                  class={cn(
                    spacing.margin.top.inline,
                    iconTokens.size.sm,
                    'text-brand-primary focus:ring-brand-primary',
                  )}
                />
                <div class="flex-1">
                  <span class="body-small font-medium text-text-primary flex items-center gap-2">
                    <Zap class={iconTokens.size.sm} />
                    {t('password.generated.title')}
                  </span>
                  <p class={cn('caption text-text-muted', spacing.margin.top.inline)}>
                    {t('password.generated.description')}
                  </p>
                  {passwordMode === 'generated' && (
                    <div
                      class={cn(
                        spacing.margin.top.inline,
                        spacing.pad.sm,
                        'bg-surface-sunken',
                        radius.default,
                      )}
                    >
                      <div class={cn(layout.inline.default)}>
                        <code class="flex-1 font-mono body-small text-brand-primary select-all break-all">
                          {suggestedPassword}
                        </code>
                        <button
                          type="button"
                          onClick={handleCopyPassword}
                          class={cn(
                            button.size.xs,
                            'text-text-muted hover:text-text-primary border border-surface-border',
                            radius.md,
                            'hover:bg-surface-base transition-colors shrink-0 p-1.5',
                          )}
                          title={t('buttons.copy')}
                        >
                          <Copy class="w-3.5 h-3.5" />
                        </button>
                      </div>
                      {copied ? (
                        <p class={cn('caption text-status-success', spacing.margin.top.inline)}>
                          {t('buttons.copied')}
                        </p>
                      ) : null}
                      <p class={cn('caption text-status-warning', spacing.margin.top.inline)}>
                        {t('password.generated.saveWarning')}
                      </p>
                    </div>
                  )}
                </div>
              </label>
            ) : null}
          </div>

          {passwordMode === 'custom' && (
            <>
              <div class={spacing.margin.bottom.content}>
                <label
                  for="setup-password"
                  class={cn(
                    'block body-small font-medium text-text-primary',
                    spacing.margin.bottom.inline,
                  )}
                >
                  {t('password.label')}
                </label>
                <div class="relative">
                  <input
                    id="setup-password"
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setPassword(e.target.value)
                    }
                    class={cn(inputClass('default', 'md'), spacing.padding.right.icon)}
                    placeholder={t('password.placeholder')}
                    required={true}
                    minLength={12}
                  />
                  <button
                    type="button"
                    onClick={(): void => setShowPassword(!showPassword)}
                    class="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                    aria-label={showPassword ? 'Hide password' : 'Show password'}
                  >
                    {showPassword ? (
                      <EyeOff class={iconTokens.size.md} />
                    ) : (
                      <Eye class={iconTokens.size.md} />
                    )}
                  </button>
                </div>
                <p class={cn('caption text-text-muted', spacing.margin.top.inline)}>
                  {t('password.minLength')}
                </p>
                {/* Fixes #723: live complexity-rule checklist so the user
                    can see exactly which constraints they still need to meet. */}
                {password.length > 0 ? (
                  <ul
                    aria-label={t('password.rulesLabel')}
                    class={cn('caption stack-xs', spacing.margin.top.inline)}
                  >
                    {evaluatePassword(password).rules.map((rule: PasswordRule) => (
                      <li
                        key={rule.id}
                        class={cn(
                          'flex items-center gap-2',
                          rule.ok ? 'text-status-success' : 'text-text-muted',
                        )}
                      >
                        <span aria-hidden="true">{rule.ok ? '✓' : '○'}</span>
                        <span>{t(`password.rules.${rule.id}`)}</span>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>

              <div class={spacing.margin.bottom.section}>
                <label
                  for="setup-confirm-password"
                  class={cn(
                    'block body-small font-medium text-text-primary',
                    spacing.margin.bottom.inline,
                  )}
                >
                  {t('password.confirm.label')}
                </label>
                <input
                  id="setup-confirm-password"
                  type={showPassword ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setConfirmPassword(e.target.value)
                  }
                  class={inputClass('default', 'md')}
                  placeholder={t('password.confirm.placeholder')}
                  required={true}
                />
              </div>
            </>
          )}

          {error ? (
            <div
              role="alert"
              aria-live="assertive"
              class={cn(
                spacing.margin.bottom.content,
                'pad-sm bg-status-error/10 border border-status-error/20',
                radius.md,
                'text-status-error body-small',
              )}
            >
              {error}
            </div>
          ) : null}

          <button
            type="submit"
            disabled={isSubmitting}
            class={buttonClass('primary', 'md', 'w-full')}
          >
            {isSubmitting ? t('buttons.settingUp') : t('buttons.completeSetup')}
          </button>

          {/* SSO Options - only show if any provider is enabled (fixes #769) */}
          {hasEnabledSso && (
            <>
              {/* Separator */}
              <div class="relative my-6">
                <div class="absolute inset-0 flex items-center" aria-hidden="true">
                  <div class="w-full border-t border-surface-border" />
                </div>
                <div class="relative flex justify-center">
                  <span class="px-2 bg-surface-raised text-sm text-text-muted">
                    {t('common:or')}
                  </span>
                </div>
              </div>

              <div class="flex flex-col stack-sm">
                {isProviderEnabled('google') && (
                  <button
                    type="button"
                    onClick={() => {
                      window.location.href = `${API_BASE}/api/sso/login?provider=google`;
                    }}
                    class={cn(
                      'w-full',
                      button.size.md,
                      'bg-status-info text-text-inverse',
                      radius.md,
                      'font-medium hover:bg-status-info-dark focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50',
                    )}
                  >
                    {t('common:buttons.signInWithGoogle')}
                  </button>
                )}
                {isProviderEnabled('microsoft') && (
                  <button
                    type="button"
                    onClick={() => {
                      window.location.href = `${API_BASE}/api/sso/login?provider=microsoft`;
                    }}
                    class={cn(
                      'w-full',
                      button.size.md,
                      'bg-brand-secondary text-text-inverse',
                      radius.md,
                      'font-medium hover:bg-brand-secondary-dark focus:outline-none focus:ring-2 focus:ring-brand-secondary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50',
                    )}
                  >
                    {t('common:buttons.signInWithMicrosoft')}
                  </button>
                )}
                {isProviderEnabled('github') && (
                  <button
                    type="button"
                    onClick={() => {
                      window.location.href = `${API_BASE}/api/sso/login?provider=github`;
                    }}
                    class={cn(
                      'w-full',
                      button.size.md,
                      'bg-surface-sunken text-text-primary',
                      radius.md,
                      'font-medium hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-surface-border focus:ring-offset-2 focus:ring-offset-surface-base border border-surface-border disabled:opacity-50',
                    )}
                  >
                    {t('common:buttons.signInWithGitHub')}
                  </button>
                )}
              </div>
            </>
          )}
        </form>
      </div>
    </div>
  );
}
