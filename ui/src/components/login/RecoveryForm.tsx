/**
 * RecoveryForm Component
 *
 * Password recovery form for The Seed application.
 * Migrated to react-hook-form + valibot per #1201.
 */

import { valibotResolver } from '@hookform/resolvers/valibot';
import { Eye, EyeOff, KeyRound, Lock, Timer } from 'lucide-react';
import type React from 'react';
import { useEffect, useState } from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { RecoveryCompleteSchema } from '../../schemas/auth';
import {
  alert,
  button,
  cn,
  icon,
  input,
  layout,
  radius,
  spacing,
  status as statusColor,
} from '../../styles/theme';

// API base URL - configurable via environment variable
const API_BASE: string = import.meta.env.VITE_API_BASE || '';

// Minimum password length (matches setup-wizard, mirrored in schema)
const MIN_PASSWORD_LENGTH = 12;

export interface RecoveryFormProps {
  /** Callback when recovery is complete */
  onRecoveryComplete: () => void;
  /** Callback to return to login */
  onBackToLogin: () => void;
  /** Remaining time in seconds */
  remainingTime?: number;
  /** File path instructions */
  tokenFilePath?: string;
}

interface RecoveryInstructions {
  triggerFile: string;
  tokenFile: string;
  expiryTime: string;
  steps: string[];
}

interface RecoveryFormFields {
  token: string;
  password: string;
  confirmPassword: string;
}

export function RecoveryForm({
  onRecoveryComplete,
  onBackToLogin,
  remainingTime: initialRemainingTime = 0,
  tokenFilePath = '',
}: RecoveryFormProps): React.ReactElement {
  const { t } = useTranslation('common');
  const { t: tErrors } = useTranslation('errors');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [remainingTime, setRemainingTime] = useState(initialRemainingTime);
  const [instructions, setInstructions] = useState<RecoveryInstructions | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RecoveryFormFields>({
    resolver: valibotResolver(RecoveryCompleteSchema),
    defaultValues: { token: '', password: '', confirmPassword: '' },
    mode: 'onBlur',
  });

  // Fetch recovery instructions on mount
  useEffect(() => {
    fetch(`${API_BASE}/api/v1/recovery/instructions`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data) {
          setInstructions(data);
        }
      })
      .catch(() => {
        // Instructions are optional, don't error
      });
  }, []);

  // Countdown timer for token expiry
  useEffect(() => {
    if (remainingTime <= 0) {
      return;
    }
    const interval = setInterval(() => {
      setRemainingTime((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return (): void => clearInterval(interval);
  }, [remainingTime]);

  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const onSubmit: SubmitHandler<RecoveryFormFields> = async ({ token, password }) => {
    setSubmitError(null);
    try {
      const response = await fetch(`${API_BASE}/api/v1/recovery/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: token.trim(), password }),
      });
      const data = (await response.json()) as {
        success?: boolean;
        message?: string;
        error?: string;
      };
      if (response.ok && data.success) {
        onRecoveryComplete();
      } else {
        setSubmitError(data.message || data.error || tErrors('recovery.failed'));
      }
    } catch {
      setSubmitError(tErrors('network.networkError'));
    }
  };

  // Cross-field error (passwords don't match) from valibot v.check().
  const rootErrors = errors.root;
  const crossFieldError = rootErrors
    ? Object.values(rootErrors).find(
        (e): e is { message: string } =>
          typeof e === 'object' && e !== null && 'message' in e && typeof e.message === 'string',
      )
    : undefined;

  return (
    <div className={cn('min-h-screen', layout.flex.center, 'pad')}>
      <div className="w-full max-w-md">
        {/* Header */}
        <div className={cn('text-center', spacing.margin.bottom.section)}>
          <div className={cn('w-16 h-16 mx-auto text-status-warning', layout.flex.center)}>
            <KeyRound className={icon.size['2xl']} />
          </div>
          <h1 className={cn('heading-1', spacing.margin.top.heading)}>
            {t('recovery.title', 'Password Recovery')}
          </h1>
          <p className={cn('body-small', spacing.margin.top.inline)}>
            {t('recovery.subtitle', 'Reset your password using filesystem access')}
          </p>
        </div>

        {/* Timer Warning */}
        {remainingTime > 0 ? (
          <div
            className={cn(
              alert.base,
              remainingTime < 120 ? alert.variant.warning : alert.variant.info,
              spacing.margin.bottom.section,
              layout.flex.center,
            )}
          >
            <Timer className={icon.size.sm} />
            <span className="ml-2">
              {t('recovery.timeRemaining', 'Time remaining')}: {formatTime(remainingTime)}
            </span>
          </div>
        ) : null}

        {/* Instructions Panel */}
        {instructions ? (
          <div
            className={cn(
              'bg-surface-sunken',
              radius.md,
              'border border-surface-border pad',
              spacing.margin.bottom.section,
            )}
          >
            <h3 className={cn('heading-4', spacing.margin.bottom.inline)}>
              {t('recovery.instructions.title', 'Recovery Instructions')}
            </h3>
            <ol className="body-small text-text-secondary space-y-1 list-decimal list-inside">
              {instructions.steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
            {tokenFilePath ? (
              <p className={cn('caption text-text-muted', spacing.margin.top.inline)}>
                {t('recovery.tokenLocation', 'Token file')}:{' '}
                <code className="code">{tokenFilePath}</code>
              </p>
            ) : null}
          </div>
        ) : null}

        {/* Recovery Form */}
        <form
          onSubmit={handleSubmit(onSubmit)}
          className={cn(
            'bg-surface-raised',
            radius.md,
            'border border-surface-border pad-lg stack-lg',
          )}
        >
          {/* Token Input */}
          <div>
            <label
              htmlFor="recovery-token"
              className={cn('label block', spacing.margin.bottom.inline)}
            >
              {t('recovery.tokenLabel', 'Recovery Token')}
            </label>
            <div className="relative">
              <KeyRound
                className={cn(
                  icon.size.sm,
                  'absolute left-3 top-1/2 -translate-y-1/2 text-text-muted',
                )}
              />
              <input
                id="recovery-token"
                type="text"
                required={true}
                {...register('token')}
                className={cn(
                  'w-full pl-10',
                  input.size.md,
                  radius.md,
                  'border border-surface-border bg-surface-base text-text-primary',
                  'focus:outline-none focus:border-brand-primary font-mono',
                )}
                placeholder={t(
                  'recovery.tokenPlaceholder',
                  'Paste token from .recovery-token file',
                )}
                autoComplete="off"
                spellCheck={false}
              />
            </div>
            {errors.token ? (
              <p className="caption mt-1 text-status-error">{errors.token.message}</p>
            ) : null}
          </div>

          {/* New Password Input */}
          <div>
            <label
              htmlFor="recovery-password"
              className={cn('label block', spacing.margin.bottom.inline)}
            >
              {t('recovery.newPasswordLabel', 'New Password')}
            </label>
            <div className="relative">
              <Lock
                className={cn(
                  icon.size.sm,
                  'absolute left-3 top-1/2 -translate-y-1/2 text-text-muted',
                )}
              />
              <input
                id="recovery-password"
                type={showPassword ? 'text' : 'password'}
                required={true}
                {...register('password')}
                className={cn(
                  'w-full pl-10 pr-10',
                  input.size.md,
                  radius.md,
                  'border bg-surface-base text-text-primary',
                  errors.password ? statusColor.border.error : 'border-surface-border',
                  'focus:outline-none focus:border-brand-primary',
                )}
                placeholder="••••••••••••"
              />
              <button
                type="button"
                onClick={(): void => setShowPassword(!showPassword)}
                className={cn(
                  'absolute right-3 top-1/2 -translate-y-1/2 text-text-muted',
                  'hover:text-text-primary focus:outline-none',
                )}
                aria-label={showPassword ? t('buttons.hidePassword') : t('buttons.showPassword')}
              >
                {showPassword ? (
                  <EyeOff className={icon.size.sm} />
                ) : (
                  <Eye className={icon.size.sm} />
                )}
              </button>
            </div>
            {errors.password ? (
              <p className="caption mt-1 text-status-error">{errors.password.message}</p>
            ) : (
              <p className="caption mt-1 text-text-muted">
                {t('recovery.passwordRequirement', 'Minimum {{min}} characters', {
                  min: MIN_PASSWORD_LENGTH,
                })}
              </p>
            )}
          </div>

          {/* Confirm Password Input */}
          <div>
            <label
              htmlFor="recovery-confirm-password"
              className={cn('label block', spacing.margin.bottom.inline)}
            >
              {t('recovery.confirmPasswordLabel', 'Confirm Password')}
            </label>
            <div className="relative">
              <Lock
                className={cn(
                  icon.size.sm,
                  'absolute left-3 top-1/2 -translate-y-1/2 text-text-muted',
                )}
              />
              <input
                id="recovery-confirm-password"
                type={showConfirmPassword ? 'text' : 'password'}
                required={true}
                {...register('confirmPassword')}
                className={cn(
                  'w-full pl-10 pr-10',
                  input.size.md,
                  radius.md,
                  'border bg-surface-base text-text-primary',
                  errors.confirmPassword || crossFieldError
                    ? statusColor.border.error
                    : 'border-surface-border',
                  'focus:outline-none focus:border-brand-primary',
                )}
                placeholder="••••••••••••"
              />
              <button
                type="button"
                onClick={(): void => setShowConfirmPassword(!showConfirmPassword)}
                className={cn(
                  'absolute right-3 top-1/2 -translate-y-1/2 text-text-muted',
                  'hover:text-text-primary focus:outline-none',
                )}
                aria-label={
                  showConfirmPassword ? t('buttons.hidePassword') : t('buttons.showPassword')
                }
              >
                {showConfirmPassword ? (
                  <EyeOff className={icon.size.sm} />
                ) : (
                  <Eye className={icon.size.sm} />
                )}
              </button>
            </div>
            {errors.confirmPassword ? (
              <p className="caption mt-1 text-status-error">{errors.confirmPassword.message}</p>
            ) : null}
          </div>

          {/* Cross-field error (passwords don't match) */}
          {crossFieldError ? (
            <div role="alert" className={cn(alert.base, alert.variant.error)}>
              {crossFieldError.message}
            </div>
          ) : null}

          {/* Submit error (network / server) */}
          {submitError ? (
            <div role="alert" aria-live="assertive" className={cn(alert.base, alert.variant.error)}>
              {submitError}
            </div>
          ) : null}

          {/* Submit Button */}
          <button
            type="submit"
            disabled={isSubmitting}
            className={cn(
              'w-full',
              button.size.md,
              'bg-brand-primary text-text-inverse',
              radius.md,
              'font-medium hover:bg-brand-accent',
              'focus:outline-none focus:ring-2 focus:ring-brand-primary',
              'focus:ring-offset-2 focus:ring-offset-surface-base',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            {isSubmitting
              ? t('recovery.submitting', 'Resetting Password...')
              : t('recovery.submit', 'Reset Password')}
          </button>

          {/* Back to Login Link */}
          <button
            type="button"
            onClick={onBackToLogin}
            className={cn(
              'w-full',
              button.size.sm,
              'text-text-secondary hover:text-text-primary',
              'focus:outline-none focus:underline',
            )}
          >
            {t('recovery.backToLogin', 'Back to Login')}
          </button>
        </form>

        {/* Security Note */}
        <p className={cn('caption text-text-muted text-center', spacing.margin.top.section)}>
          {t(
            'recovery.securityNote',
            'Recovery tokens are single-use and expire after 15 minutes.',
          )}
        </p>
      </div>
    </div>
  );
}
