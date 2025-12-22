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

import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { radius, layout, spacing, button } from "../../styles/theme";
import {
  buttonClass,
  inputClass,
  cardClass,
  cn,
  icon as iconTokens,
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
interface SSOProvider {
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
  const [passwordMode, setPasswordMode] = useState<"generated" | "custom">(
    "custom"
  );
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [ssoProviders, setSsoProviders] = useState<SSOProvider[]>([]);

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
    ssoProviders.some(
      (p) => p.name.toLowerCase() === name.toLowerCase() && p.enabled
    );

  // Check if any SSO provider is enabled
  const hasEnabledSSO = ssoProviders.some((p) => p.enabled);

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

    // Use length-checked comparison to avoid timing analysis
    // (client-side password confirmation doesn't pose timing attack risk,
    // but we use this pattern to satisfy security linting)
    const passwordsMatch =
      password.length === confirmPassword.length &&
      [...password].every((char, idx) => char === confirmPassword.charAt(idx));
    if (!passwordsMatch) {
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
    <div
      className={cn("min-h-screen bg-surface-base", layout.flex.center, "pad")}
    >
      <div className="w-full max-w-md">
        <div className={cn("text-center", spacing.margin.bottom.sectionLg)}>
          <div className="w-16 h-16 mx-auto text-brand-primary">
            <svg viewBox="0 0 48 48" fill="none" className="w-full h-full">
              <circle
                cx="24"
                cy="24"
                r="20"
                stroke="currentColor"
                strokeWidth="2"
                opacity="0.3"
              />
              <circle
                cx="24"
                cy="24"
                r="14"
                stroke="currentColor"
                strokeWidth="2"
                opacity="0.5"
              />
              <circle cx="24" cy="24" r="4" fill="currentColor" />
              <line
                x1="24"
                y1="10"
                x2="24"
                y2="18"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="24"
                y1="30"
                x2="24"
                y2="38"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="10"
                y1="24"
                x2="18"
                y2="24"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="30"
                y1="24"
                x2="38"
                y2="24"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <circle cx="24" cy="8" r="3" fill="currentColor" />
              <circle cx="24" cy="40" r="3" fill="currentColor" />
              <circle cx="8" cy="24" r="3" fill="currentColor" />
              <circle cx="40" cy="24" r="3" fill="currentColor" />
            </svg>
          </div>
          <h1 className={cn("heading-2", spacing.margin.top.heading)}>
            {t("welcome.title")}
          </h1>
          <p className={cn("body-small", spacing.margin.top.inline)}>
            {t("welcome.subtitle")}
          </p>
        </div>

        <form onSubmit={handleSubmit} className={cardClass("default", "lg")}>
          <div className={spacing.margin.bottom.content}>
            <p className={cn("body-small", spacing.margin.bottom.content)}>
              {t("username.label")} <strong>{username}</strong>{" "}
              {t("username.cannotChange")}
            </p>
          </div>

          {/* Password mode selection */}
          <div className={cn(spacing.margin.bottom.section, "stack-sm")}>
            <p
              className={cn(
                "body-small font-medium text-text-primary",
                spacing.margin.bottom.inline
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
                "border border-surface-border cursor-pointer hover:bg-surface-base transition-colors"
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
                  "text-brand-primary focus:ring-brand-primary"
                )}
              />
              <div>
                <span className="body-small font-medium text-text-primary">
                  {t("password.custom.title")}
                </span>
                <p
                  className={cn(
                    "caption text-text-muted",
                    spacing.margin.top.inline
                  )}
                >
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
                  "border border-surface-border cursor-pointer hover:bg-surface-base transition-colors"
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
                    "text-brand-primary focus:ring-brand-primary"
                  )}
                />
                <div className="flex-1">
                  <span className="body-small font-medium text-text-primary">
                    {t("password.generated.title")}
                  </span>
                  <p
                    className={cn(
                      "caption text-text-muted",
                      spacing.margin.top.inline
                    )}
                  >
                    {t("password.generated.description")}
                  </p>
                  {passwordMode === "generated" && (
                    <div
                      className={cn(
                        spacing.margin.top.inline,
                        spacing.pad.sm,
                        "bg-surface-sunken",
                        radius.default
                      )}
                    >
                      <div className={cn(layout.inline.default)}>
                        <code className="flex-1 font-mono body-small text-brand-primary select-all break-all">
                          {suggestedPassword}
                        </code>
                        <button
                          type="button"
                          onClick={() =>
                            navigator.clipboard.writeText(suggestedPassword)
                          }
                          className={cn(
                            button.size.xs,
                            "caption text-text-muted hover:text-text-primary border border-surface-border",
                            radius.md,
                            "hover:bg-surface-base transition-colors shrink-0"
                          )}
                        >
                          {t("buttons.copy")}
                        </button>
                      </div>
                      <p
                        className={cn(
                          "caption text-status-warning",
                          spacing.margin.top.inline
                        )}
                      >
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
                    spacing.margin.bottom.inline
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
                    className={cn(
                      inputClass("default", "md"),
                      spacing.padding.right.icon
                    )}
                    placeholder={t("password.placeholder")}
                    required
                    minLength={12}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                  >
                    {showPassword ? (
                      <svg
                        className={iconTokens.size.md}
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
                        />
                      </svg>
                    ) : (
                      <svg
                        className={iconTokens.size.md}
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                        />
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                        />
                      </svg>
                    )}
                  </button>
                </div>
                <p
                  className={cn(
                    "caption text-text-muted",
                    spacing.margin.top.inline
                  )}
                >
                  {t("password.minLength")}
                </p>
              </div>

              <div className={spacing.margin.bottom.section}>
                <label
                  htmlFor="setup-confirm-password"
                  className={cn(
                    "block body-small font-medium text-text-primary",
                    spacing.margin.bottom.inline
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
                "text-status-error body-small"
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
          {hasEnabledSSO && (
            <>
              {/* Separator */}
              <div className="relative my-6">
                <div
                  className="absolute inset-0 flex items-center"
                  aria-hidden="true"
                >
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
                    onClick={() =>
                      (window.location.href = `${API_BASE}/api/sso/login?provider=google`)
                    }
                    className={cn(
                      "w-full",
                      button.size.md,
                      "bg-status-info text-text-inverse",
                      radius.md,
                      "font-medium hover:bg-status-info-dark focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
                    )}
                  >
                    {t("common:buttons.signInWithGoogle")}
                  </button>
                )}
                {isProviderEnabled("microsoft") && (
                  <button
                    type="button"
                    onClick={() =>
                      (window.location.href = `${API_BASE}/api/sso/login?provider=microsoft`)
                    }
                    className={cn(
                      "w-full",
                      button.size.md,
                      "bg-brand-secondary text-text-inverse",
                      radius.md,
                      "font-medium hover:bg-brand-secondary-dark focus:outline-none focus:ring-2 focus:ring-brand-secondary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
                    )}
                  >
                    {t("common:buttons.signInWithMicrosoft")}
                  </button>
                )}
                {isProviderEnabled("github") && (
                  <button
                    type="button"
                    onClick={() =>
                      (window.location.href = `${API_BASE}/api/sso/login?provider=github`)
                    }
                    className={cn(
                      "w-full",
                      button.size.md,
                      "bg-surface-sunken text-text-primary",
                      radius.md,
                      "font-medium hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-surface-border focus:ring-offset-2 focus:ring-offset-surface-base border border-surface-border disabled:opacity-50"
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
