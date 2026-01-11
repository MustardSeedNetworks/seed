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

import { Activity, Copy, Eye, EyeOff, Lock, Zap } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
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
} from "../../styles/theme";

// API base URL for setup endpoints
const API_BASE = import.meta.env.VITE_API_BASE || "";

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
// SSO provider info from backend
interface SsoProvider {
  name: string;
  enabled: boolean;
}

/**
 * First-run setup flow that forces the user to create credentials before using the app.
 */
export function SetupWizard({
  onComplete,
  onLogin,
  suggestedPassword,
  username = "admin",
  setupToken,
}: SetupWizardProps) {
  const { t } = useTranslation("setup");
  // Default to custom password entry - more secure UX
  const [passwordMode, setPasswordMode] = useState<"generated" | "custom">("custom");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);
  const [ssoProviders, setSsoProviders] = useState<SsoProvider[]>([]);

  // Fetch enabled SSO providers (fixes #769)
  useEffect(() => {
    fetch(`${API_BASE}/api/sso/providers`)
      .then((res) => (res.ok ? res.json() : { providers: [] }))
      .then((data) => setSsoProviders(data.providers || []))
      .catch(() => setSsoProviders([]));
  }, []);

  // Update password fields when switching to generated mode
  useEffect(() => {
    if (passwordMode === "generated" && suggestedPassword) {
      setPassword(suggestedPassword);
      setConfirmPassword(suggestedPassword);
    }
  }, [passwordMode, suggestedPassword]);

  // Helper to check if a provider is enabled
  const isProviderEnabled = (name: string) =>
    ssoProviders.some((p) => p.name.toLowerCase() === name.toLowerCase() && p.enabled);

  // Check if any SSO provider is enabled
  const hasEnabledSso = ssoProviders.some((p) => p.enabled);

  const handleCopyPassword = async (): Promise<void> => {
    if (suggestedPassword) {
      await navigator.clipboard.writeText(suggestedPassword);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handlePasswordModeChange = (mode: "generated" | "custom") => {
    setPasswordMode(mode);
    if (mode === "generated" && suggestedPassword) {
      setPassword(suggestedPassword);
      setConfirmPassword(suggestedPassword);
      setShowPassword(true);
    } else {
      setPassword("");
      setConfirmPassword("");
      setShowPassword(false);
    }
    setError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (password.length < 12) {
      setError(t("errors.passwordTooShort"));
      return;
    }

    if (password !== confirmPassword) {
      setError(t("errors.passwordMismatch"));
      return;
    }

    setIsSubmitting(true);

    try {
      // Step 1: Complete setup (set password on server)
      // Security fix #724, #758: Include the one-time setup token to prevent CSRF attacks
      const response = await fetch(`${API_BASE}/api/setup/complete`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ password, setupToken }),
      });

      if (!response.ok) {
        const data = await response.json();
        setError(data.error || t("errors.setupFailed"));
        return;
      }

      // Step 2: Automatically log in with the new password (fixes #768 - use username from config)
      const loginSuccess = await onLogin(username, password);

      if (!loginSuccess) {
        setError(t("errors.loginFailed"));
        // Still call onComplete to exit setup wizard
        onComplete();
        return;
      }

      // Step 3: Setup complete and user is logged in
      onComplete();
    } catch {
      setError(t("errors.networkError"));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className={cn("min-h-screen bg-surface-base", layout.flex.center, "pad")}>
      <div className="w-full max-w-md">
        <div className={cn("text-center", spacing.margin.bottom.sectionLg)}>
          <div className="w-16 h-16 mx-auto flex items-center justify-center rounded-2xl bg-brand-primary text-text-inverse">
            <Activity className="w-8 h-8" />
          </div>
          <h1 className={cn("heading-2", spacing.margin.top.heading)}>{t("welcome.title")}</h1>
          <p className={cn("body-small", spacing.margin.top.inline)}>{t("welcome.subtitle")}</p>
        </div>

        <form onSubmit={handleSubmit} className={cardClass("default", "lg")}>
          <div className={spacing.margin.bottom.content}>
            <p className={cn("body-small", spacing.margin.bottom.content)}>
              {t("username.label")} <strong>{username}</strong> {t("username.cannotChange")}
            </p>
          </div>

          {/* Password mode selection */}
          <div className={cn(spacing.margin.bottom.section, "stack-sm")}>
            <p
              className={cn(
                "body-small font-medium text-text-primary",
                spacing.margin.bottom.inline,
              )}
            >
              {t("password.chooseMethod")}
            </p>

            {/* Custom password option */}
            <label
              className={cn(
                "flex items-start",
                spacing.gap.default,
                "pad-sm",
                radius.md,
                "border border-surface-border cursor-pointer hover:bg-surface-base transition-colors",
              )}
            >
              <input
                type="radio"
                name="passwordMode"
                value="custom"
                checked={passwordMode === "custom"}
                onChange={() => handlePasswordModeChange("custom")}
                className={cn(
                  spacing.margin.top.inline,
                  iconTokens.size.sm,
                  "text-brand-primary focus:ring-brand-primary",
                )}
              />
              <div>
                <span className="body-small font-medium text-text-primary flex items-center gap-2">
                  <Lock className={iconTokens.size.sm} />
                  {t("password.custom.title")}
                </span>
                <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
                  {t("password.custom.description")}
                </p>
              </div>
            </label>

            {/* Generated password option */}
            {suggestedPassword && (
              <label
                className={cn(
                  "flex items-start",
                  spacing.gap.default,
                  "pad-sm",
                  radius.md,
                  "border border-surface-border cursor-pointer hover:bg-surface-base transition-colors",
                )}
              >
                <input
                  type="radio"
                  name="passwordMode"
                  value="generated"
                  checked={passwordMode === "generated"}
                  onChange={() => handlePasswordModeChange("generated")}
                  className={cn(
                    spacing.margin.top.inline,
                    iconTokens.size.sm,
                    "text-brand-primary focus:ring-brand-primary",
                  )}
                />
                <div className="flex-1">
                  <span className="body-small font-medium text-text-primary flex items-center gap-2">
                    <Zap className={iconTokens.size.sm} />
                    {t("password.generated.title")}
                  </span>
                  <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
                    {t("password.generated.description")}
                  </p>
                  {passwordMode === "generated" && (
                    <div
                      className={cn(
                        spacing.margin.top.inline,
                        spacing.pad.sm,
                        "bg-surface-sunken",
                        radius.default,
                      )}
                    >
                      <div className={cn(layout.inline.default)}>
                        <code className="flex-1 font-mono body-small text-brand-primary select-all break-all">
                          {suggestedPassword}
                        </code>
                        <button
                          type="button"
                          onClick={handleCopyPassword}
                          className={cn(
                            button.size.xs,
                            "text-text-muted hover:text-text-primary border border-surface-border",
                            radius.md,
                            "hover:bg-surface-base transition-colors shrink-0 p-1.5",
                          )}
                          title={t("buttons.copy")}
                        >
                          <Copy className="w-3.5 h-3.5" />
                        </button>
                      </div>
                      {copied && (
                        <p className={cn("caption text-status-success", spacing.margin.top.inline)}>
                          {t("buttons.copied")}
                        </p>
                      )}
                      <p className={cn("caption text-status-warning", spacing.margin.top.inline)}>
                        {t("password.generated.saveWarning")}
                      </p>
                    </div>
                  )}
                </div>
              </label>
            )}
          </div>

          {passwordMode === "custom" && (
            <>
              <div className={spacing.margin.bottom.content}>
                <label
                  htmlFor="setup-password"
                  className={cn(
                    "block body-small font-medium text-text-primary",
                    spacing.margin.bottom.inline,
                  )}
                >
                  {t("password.label")}
                </label>
                <div className="relative">
                  <input
                    id="setup-password"
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className={cn(inputClass("default", "md"), spacing.padding.right.icon)}
                    placeholder={t("password.placeholder")}
                    required
                    minLength={12}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                    aria-label={showPassword ? "Hide password" : "Show password"}
                  >
                    {showPassword ? (
                      <EyeOff className={iconTokens.size.md} />
                    ) : (
                      <Eye className={iconTokens.size.md} />
                    )}
                  </button>
                </div>
                <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
                  {t("password.minLength")}
                </p>
              </div>

              <div className={spacing.margin.bottom.section}>
                <label
                  htmlFor="setup-confirm-password"
                  className={cn(
                    "block body-small font-medium text-text-primary",
                    spacing.margin.bottom.inline,
                  )}
                >
                  {t("password.confirm.label")}
                </label>
                <input
                  id="setup-confirm-password"
                  type={showPassword ? "text" : "password"}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className={inputClass("default", "md")}
                  placeholder={t("password.confirm.placeholder")}
                  required
                />
              </div>
            </>
          )}

          {error && (
            <div
              role="alert"
              aria-live="assertive"
              className={cn(
                spacing.margin.bottom.content,
                "pad-sm bg-status-error/10 border border-status-error/20",
                radius.md,
                "text-status-error body-small",
              )}
            >
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={isSubmitting}
            className={buttonClass("primary", "md", "w-full")}
          >
            {isSubmitting ? t("buttons.settingUp") : t("buttons.completeSetup")}
          </button>

          {/* SSO Options - only show if any provider is enabled (fixes #769) */}
          {hasEnabledSso && (
            <>
              {/* Separator */}
              <div className="relative my-6">
                <div className="absolute inset-0 flex items-center" aria-hidden="true">
                  <div className="w-full border-t border-surface-border" />
                </div>
                <div className="relative flex justify-center">
                  <span className="px-2 bg-surface-raised text-sm text-text-muted">
                    {t("common:or")}
                  </span>
                </div>
              </div>

              <div className="flex flex-col stack-sm">
                {isProviderEnabled("google") && (
                  <button
                    type="button"
                    onClick={() => {
                      window.location.href = `${API_BASE}/api/sso/login?provider=google`;
                    }}
                    className={cn(
                      "w-full",
                      button.size.md,
                      "bg-status-info text-text-inverse",
                      radius.md,
                      "font-medium hover:bg-status-info-dark focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50",
                    )}
                  >
                    {t("common:buttons.signInWithGoogle")}
                  </button>
                )}
                {isProviderEnabled("microsoft") && (
                  <button
                    type="button"
                    onClick={() => {
                      window.location.href = `${API_BASE}/api/sso/login?provider=microsoft`;
                    }}
                    className={cn(
                      "w-full",
                      button.size.md,
                      "bg-brand-secondary text-text-inverse",
                      radius.md,
                      "font-medium hover:bg-brand-secondary-dark focus:outline-none focus:ring-2 focus:ring-brand-secondary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50",
                    )}
                  >
                    {t("common:buttons.signInWithMicrosoft")}
                  </button>
                )}
                {isProviderEnabled("github") && (
                  <button
                    type="button"
                    onClick={() => {
                      window.location.href = `${API_BASE}/api/sso/login?provider=github`;
                    }}
                    className={cn(
                      "w-full",
                      button.size.md,
                      "bg-surface-sunken text-text-primary",
                      radius.md,
                      "font-medium hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-surface-border focus:ring-offset-2 focus:ring-offset-surface-base border border-surface-border disabled:opacity-50",
                    )}
                  >
                    {t("common:buttons.signInWithGitHub")}
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
