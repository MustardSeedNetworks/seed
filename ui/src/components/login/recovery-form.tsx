/**
 * RecoveryForm Component
 *
 * Password recovery form for The Seed application.
 *
 * Responsibilities:
 * - Display recovery status and remaining time
 * - Accept recovery token from filesystem
 * - Set new password with confirmation
 * - Validate password strength (min 12 chars)
 *
 * Recovery Flow:
 * 1. Admin creates .recovery file via SSH (proves filesystem access)
 * 2. Server generates token, writes to .recovery-token
 * 3. Admin enters token + new password in this form
 * 4. Server validates, updates password, invalidates all sessions
 */

import { Eye, EyeOff, KeyRound, Lock, Timer } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { alert, button, cn, icon, input, layout, radius, spacing } from "../../styles/theme";

// API base URL - configurable via environment variable
const API_BASE = import.meta.env.VITE_API_BASE || "";

// Minimum password length (matches setup-wizard)
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

export function RecoveryForm({
  onRecoveryComplete,
  onBackToLogin,
  remainingTime: initialRemainingTime = 0,
  tokenFilePath = "",
}: RecoveryFormProps) {
  const { t } = useTranslation("common");
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [remainingTime, setRemainingTime] = useState(initialRemainingTime);
  const [instructions, setInstructions] = useState<RecoveryInstructions | null>(null);

  // Fetch recovery instructions on mount
  useEffect(() => {
    fetch(`${API_BASE}/api/recovery/instructions`)
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
    if (remainingTime <= 0) return;

    const interval = setInterval(() => {
      setRemainingTime((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [remainingTime]);

  // Format remaining time as MM:SS
  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, "0")}`;
  };

  // Password validation
  const passwordValid = password.length >= MIN_PASSWORD_LENGTH;
  const passwordsMatch = password === confirmPassword;
  const canSubmit = token.trim() && passwordValid && passwordsMatch && !isSubmitting;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!passwordValid) {
      setError(t("errors.password.tooShort", { min: MIN_PASSWORD_LENGTH }));
      return;
    }

    if (!passwordsMatch) {
      setError(t("errors.password.mismatch"));
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await fetch(`${API_BASE}/api/recovery/complete`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          token: token.trim(),
          password,
        }),
      });

      const data = await response.json();

      if (response.ok && data.success) {
        onRecoveryComplete();
      } else {
        setError(data.message || data.error || t("errors.recovery.failed"));
      }
    } catch {
      setError(t("errors.network"));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className={cn("min-h-screen", layout.flex.center, "pad")}>
      <div className="w-full max-w-md">
        {/* Header */}
        <div className={cn("text-center", spacing.margin.bottom.section)}>
          <div className={cn("w-16 h-16 mx-auto text-status-warning", layout.flex.center)}>
            <KeyRound className={icon.size["2xl"]} />
          </div>
          <h1 className={cn("heading-1", spacing.margin.top.heading)}>
            {t("recovery.title", "Password Recovery")}
          </h1>
          <p className={cn("body-small", spacing.margin.top.inline)}>
            {t("recovery.subtitle", "Reset your password using filesystem access")}
          </p>
        </div>

        {/* Timer Warning */}
        {remainingTime > 0 && (
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
              {t("recovery.timeRemaining", "Time remaining")}: {formatTime(remainingTime)}
            </span>
          </div>
        )}

        {/* Instructions Panel */}
        {instructions && (
          <div
            className={cn(
              "bg-surface-sunken",
              radius.md,
              "border border-surface-border pad",
              spacing.margin.bottom.section,
            )}
          >
            <h3 className={cn("heading-4", spacing.margin.bottom.inline)}>
              {t("recovery.instructions.title", "Recovery Instructions")}
            </h3>
            <ol className="body-small text-text-secondary space-y-1 list-decimal list-inside">
              {instructions.steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
            {tokenFilePath && (
              <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
                {t("recovery.tokenLocation", "Token file")}:{" "}
                <code className="code">{tokenFilePath}</code>
              </p>
            )}
          </div>
        )}

        {/* Recovery Form */}
        <form
          onSubmit={handleSubmit}
          className={cn(
            "bg-surface-raised",
            radius.md,
            "border border-surface-border pad-lg stack-lg",
          )}
        >
          {/* Token Input */}
          <div>
            <label
              htmlFor="recovery-token"
              className={cn("label block", spacing.margin.bottom.inline)}
            >
              {t("recovery.tokenLabel", "Recovery Token")}
            </label>
            <div className="relative">
              <KeyRound
                className={cn(
                  icon.size.sm,
                  "absolute left-3 top-1/2 -translate-y-1/2 text-text-muted",
                )}
              />
              <input
                id="recovery-token"
                type="text"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                className={cn(
                  "w-full pl-10",
                  input.size.md,
                  radius.md,
                  "border border-surface-border bg-surface-base text-text-primary",
                  "focus:outline-none focus:border-brand-primary font-mono",
                )}
                placeholder={t(
                  "recovery.tokenPlaceholder",
                  "Paste token from .recovery-token file",
                )}
                autoComplete="off"
                spellCheck={false}
                required
              />
            </div>
          </div>

          {/* New Password Input */}
          <div>
            <label
              htmlFor="recovery-password"
              className={cn("label block", spacing.margin.bottom.inline)}
            >
              {t("recovery.newPasswordLabel", "New Password")}
            </label>
            <div className="relative">
              <Lock
                className={cn(
                  icon.size.sm,
                  "absolute left-3 top-1/2 -translate-y-1/2 text-text-muted",
                )}
              />
              <input
                id="recovery-password"
                type={showPassword ? "text" : "password"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={cn(
                  "w-full pl-10 pr-10",
                  input.size.md,
                  radius.md,
                  "border bg-surface-base text-text-primary",
                  password && !passwordValid ? "border-status-error" : "border-surface-border",
                  "focus:outline-none focus:border-brand-primary",
                )}
                placeholder="••••••••••••"
                required
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className={cn(
                  "absolute right-3 top-1/2 -translate-y-1/2 text-text-muted",
                  "hover:text-text-primary focus:outline-none",
                )}
                aria-label={showPassword ? t("buttons.hidePassword") : t("buttons.showPassword")}
              >
                {showPassword ? (
                  <EyeOff className={icon.size.sm} />
                ) : (
                  <Eye className={icon.size.sm} />
                )}
              </button>
            </div>
            <p
              className={cn(
                "caption mt-1",
                password && !passwordValid ? "text-status-error" : "text-text-muted",
              )}
            >
              {t("recovery.passwordRequirement", "Minimum {{min}} characters", {
                min: MIN_PASSWORD_LENGTH,
              })}
            </p>
          </div>

          {/* Confirm Password Input */}
          <div>
            <label
              htmlFor="recovery-confirm-password"
              className={cn("label block", spacing.margin.bottom.inline)}
            >
              {t("recovery.confirmPasswordLabel", "Confirm Password")}
            </label>
            <div className="relative">
              <Lock
                className={cn(
                  icon.size.sm,
                  "absolute left-3 top-1/2 -translate-y-1/2 text-text-muted",
                )}
              />
              <input
                id="recovery-confirm-password"
                type={showConfirmPassword ? "text" : "password"}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className={cn(
                  "w-full pl-10 pr-10",
                  input.size.md,
                  radius.md,
                  "border bg-surface-base text-text-primary",
                  confirmPassword && !passwordsMatch
                    ? "border-status-error"
                    : "border-surface-border",
                  "focus:outline-none focus:border-brand-primary",
                )}
                placeholder="••••••••••••"
                required
              />
              <button
                type="button"
                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                className={cn(
                  "absolute right-3 top-1/2 -translate-y-1/2 text-text-muted",
                  "hover:text-text-primary focus:outline-none",
                )}
                aria-label={
                  showConfirmPassword ? t("buttons.hidePassword") : t("buttons.showPassword")
                }
              >
                {showConfirmPassword ? (
                  <EyeOff className={icon.size.sm} />
                ) : (
                  <Eye className={icon.size.sm} />
                )}
              </button>
            </div>
            {confirmPassword && !passwordsMatch && (
              <p className="caption mt-1 text-status-error">
                {t("errors.password.mismatch", "Passwords do not match")}
              </p>
            )}
          </div>

          {/* Error Display */}
          {error && (
            <div role="alert" aria-live="assertive" className={cn(alert.base, alert.variant.error)}>
              {error}
            </div>
          )}

          {/* Submit Button */}
          <button
            type="submit"
            disabled={!canSubmit}
            className={cn(
              "w-full",
              button.size.md,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "font-medium hover:bg-brand-accent",
              "focus:outline-none focus:ring-2 focus:ring-brand-primary",
              "focus:ring-offset-2 focus:ring-offset-surface-base",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {isSubmitting
              ? t("recovery.submitting", "Resetting Password...")
              : t("recovery.submit", "Reset Password")}
          </button>

          {/* Back to Login Link */}
          <button
            type="button"
            onClick={onBackToLogin}
            className={cn(
              "w-full",
              button.size.sm,
              "text-text-secondary hover:text-text-primary",
              "focus:outline-none focus:underline",
            )}
          >
            {t("recovery.backToLogin", "Back to Login")}
          </button>
        </form>

        {/* Security Note */}
        <p className={cn("caption text-text-muted text-center", spacing.margin.top.section)}>
          {t(
            "recovery.securityNote",
            "Recovery tokens are single-use and expire after 15 minutes.",
          )}
        </p>
      </div>
    </div>
  );
}
