/**
 * LoginForm Component
 *
 * Authentication form component for The Seed application.
 *
 * Responsibilities:
 * - User login with username/password
 * - SSO provider integration (Google, Microsoft, GitHub)
 * - Error display for authentication failures
 * - Dynamic SSO provider availability checking
 *
 * Features:
 * - Fetches enabled SSO providers from backend
 * - Displays SSO buttons only for enabled providers
 * - Handles SSO error messages from URL parameters
 * - Responsive design with consistent theming
 */

import type React from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { button, cn, input, layout, radius, spacing } from "../styles/theme";

// API base URL - configurable via environment variable
const API_BASE = import.meta.env.VITE_API_BASE || "";

export interface LoginFormProps {
  onLogin: (username: string, password: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

// Helper to extract and clear SSO error from URL
function getAndClearSsoError(): string | null {
  const params = new URLSearchParams(window.location.search);
  const errorParam = params.get("sso_error");
  if (errorParam) {
    // Clean URL without reload
    window.history.replaceState({}, "", window.location.pathname);
    return decodeURIComponent(errorParam.replace(/%20/g, " "));
  }
  return null;
}

// SSO provider info from backend (fixes #769)
interface SsoProvider {
  name: string;
  enabled: boolean;
}

export function LoginForm({ onLogin, isLoading, error }: LoginFormProps) {
  const { t } = useTranslation("common");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  // Initialize SSO error from URL params using lazy initialization
  const [ssoError] = useState<string | null>(getAndClearSsoError);
  // Fetch SSO providers to conditionally show buttons (fixes #769)
  const [ssoProviders, setSsoProviders] = useState<SsoProvider[]>([]);

  // Fetch enabled SSO providers on mount (fixes #769)
  useEffect(() => {
    fetch(`${API_BASE}/api/sso/providers`)
      .then((res) => (res.ok ? res.json() : { providers: [] }))
      .then((data) => setSsoProviders(data.providers || []))
      .catch(() => setSsoProviders([]));
  }, []);

  // Helper to check if a provider is enabled
  const isProviderEnabled = (name: string) =>
    ssoProviders.some((p) => p.name.toLowerCase() === name.toLowerCase() && p.enabled);

  // Check if any SSO provider is enabled
  const hasEnabledSso = ssoProviders.some((p) => p.enabled);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await onLogin(username, password);
  };

  return (
    <div className={cn("min-h-screen", layout.flex.center, "pad")}>
      <div className="w-full max-w-sm">
        <div className={cn("text-center", spacing.margin.bottom.sectionLg)}>
          <div className="w-16 h-16 mx-auto text-brand-primary">
            <svg viewBox="0 0 48 48" fill="none" className="w-full h-full" aria-hidden="true">
              <circle cx="24" cy="24" r="20" stroke="currentColor" strokeWidth="2" opacity="0.3" />
              <circle cx="24" cy="24" r="14" stroke="currentColor" strokeWidth="2" opacity="0.5" />
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
              <line
                x1="14.1"
                y1="14.1"
                x2="19.1"
                y2="19.1"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="28.9"
                y1="28.9"
                x2="33.9"
                y2="33.9"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="33.9"
                y1="14.1"
                x2="28.9"
                y2="19.1"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="14.1"
                y1="33.9"
                x2="19.1"
                y2="28.9"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <circle cx="24" cy="8" r="3" fill="currentColor" />
              <circle cx="24" cy="40" r="3" fill="currentColor" />
              <circle cx="8" cy="24" r="3" fill="currentColor" />
              <circle cx="40" cy="24" r="3" fill="currentColor" />
              <circle cx="12.3" cy="12.3" r="2.5" fill="currentColor" />
              <circle cx="35.7" cy="35.7" r="2.5" fill="currentColor" />
              <circle cx="35.7" cy="12.3" r="2.5" fill="currentColor" />
              <circle cx="12.3" cy="35.7" r="2.5" fill="currentColor" />
            </svg>
          </div>
          <h1 className={cn("heading-1", spacing.margin.top.heading)}>{t("app.title")}</h1>
          <p className={cn("body-small", spacing.margin.top.inline)}>{t("app.tagline")}</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className={cn(
            "bg-surface-raised",
            radius.md,
            "border border-surface-border pad-lg stack-lg",
          )}
        >
          <div>
            <label
              htmlFor="login-username"
              className={cn("label block", spacing.margin.bottom.inline)}
            >
              {t("labels.username")}
            </label>
            <input
              id="login-username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className={cn(
                "w-full",
                input.size.md,
                radius.md,
                "border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary",
              )}
              placeholder="admin"
              required
            />
          </div>

          <div>
            <label
              htmlFor="login-password"
              className={cn("label block", spacing.margin.bottom.inline)}
            >
              {t("labels.password")}
            </label>
            <input
              id="login-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={cn(
                "w-full",
                input.size.md,
                radius.md,
                "border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary",
              )}
              placeholder="••••••••"
              required
            />
          </div>

          {(error || ssoError) && (
            <div
              role="alert"
              aria-live="assertive"
              className={cn(
                "pad-sm bg-status-error/10 border border-status-error/20",
                radius.md,
                "text-status-error body-small",
              )}
            >
              {error || ssoError}
            </div>
          )}

          <button
            type="submit"
            disabled={isLoading}
            className={cn(
              "w-full",
              button.size.md,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "font-medium hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50",
            )}
          >
            {isLoading ? t("status.loggingIn") : t("buttons.login")}
          </button>

          <p className="caption text-text-muted text-center">{t("login.defaultCredentials")}</p>

          {/* SSO Options - only show if any provider is enabled (fixes #769) */}
          {hasEnabledSso && (
            <div className="flex flex-col space-y-3">
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
                  {t("buttons.signInWithGoogle")}
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
                  {t("buttons.signInWithMicrosoft")}
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
                  {t("buttons.signInWithGitHub")}
                </button>
              )}
            </div>
          )}
        </form>
      </div>
    </div>
  );
}
