import { useState, useEffect } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface SetupWizardProps {
  onComplete: () => void;
  suggestedPassword?: string;
}

interface SetupStatusResponse {
  needsSetup: boolean;
  username?: string;
  suggestedPassword?: string;
}

export function SetupWizard({
  onComplete,
  suggestedPassword,
}: SetupWizardProps) {
  // Default to custom password entry - more secure UX
  const [passwordMode, setPasswordMode] = useState<"generated" | "custom">(
    "custom",
  );
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Update password fields when switching to generated mode
  useEffect(() => {
    if (passwordMode === "generated" && suggestedPassword) {
      setPassword(suggestedPassword);
      setConfirmPassword(suggestedPassword);
    }
  }, [passwordMode, suggestedPassword]);

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

    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await fetch(`${API_BASE}/api/setup/complete`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ password }),
      });

      if (response.ok) {
        onComplete();
      } else {
        const data = await response.json();
        setError(data.error || "Failed to complete setup");
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-surface-base flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
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
          <h1 className="text-2xl font-bold text-text-primary mt-3">
            Welcome to LuminetIQ
          </h1>
          <p className="text-text-muted mt-1">
            Set up your admin password to get started
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-surface-raised rounded-lg border border-surface-border p-6"
        >
          <div className="mb-4">
            <p className="text-sm text-text-muted mb-4">
              Username: <strong>admin</strong> (cannot be changed)
            </p>
          </div>

          {/* Password mode selection */}
          <div className="mb-6 space-y-3">
            <p className="text-sm font-medium text-text-primary mb-2">
              Choose how to set your password:
            </p>

            {/* Custom password option */}
            <label className="flex items-start gap-3 p-3 rounded border border-surface-border cursor-pointer hover:bg-surface-base transition-colors">
              <input
                type="radio"
                name="passwordMode"
                value="custom"
                checked={passwordMode === "custom"}
                onChange={() => handlePasswordModeChange("custom")}
                className="mt-0.5 w-4 h-4 text-brand-primary focus:ring-brand-primary"
              />
              <div>
                <span className="text-sm font-medium text-text-primary">
                  Create my own password
                </span>
                <p className="text-xs text-text-muted mt-0.5">
                  Choose a password you'll remember
                </p>
              </div>
            </label>

            {/* Generated password option */}
            {suggestedPassword && (
              <label className="flex items-start gap-3 p-3 rounded border border-surface-border cursor-pointer hover:bg-surface-base transition-colors">
                <input
                  type="radio"
                  name="passwordMode"
                  value="generated"
                  checked={passwordMode === "generated"}
                  onChange={() => handlePasswordModeChange("generated")}
                  className="mt-0.5 w-4 h-4 text-brand-primary focus:ring-brand-primary"
                />
                <div className="flex-1">
                  <span className="text-sm font-medium text-text-primary">
                    Use generated secure password
                  </span>
                  <p className="text-xs text-text-muted mt-0.5">
                    Automatically generated strong password
                  </p>
                  {passwordMode === "generated" && (
                    <div className="mt-2 p-2 bg-surface-sunken rounded">
                      <div className="flex items-center gap-2">
                        <code className="flex-1 font-mono text-sm text-brand-primary select-all break-all">
                          {suggestedPassword}
                        </code>
                        <button
                          type="button"
                          onClick={() =>
                            navigator.clipboard.writeText(suggestedPassword)
                          }
                          className="px-2 py-1 text-xs text-text-muted hover:text-text-primary border border-surface-border rounded hover:bg-surface-base transition-colors shrink-0"
                        >
                          Copy
                        </button>
                      </div>
                      <p className="text-xs text-status-warning mt-2">
                        Save this password somewhere safe before continuing!
                      </p>
                    </div>
                  )}
                </div>
              </label>
            )}
          </div>

          {passwordMode === "custom" && (
            <>
              <div className="mb-4">
                <label
                  htmlFor="setup-password"
                  className="block text-sm font-medium text-text-primary mb-1"
                >
                  Password
                </label>
                <div className="relative">
                  <input
                    id="setup-password"
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary pr-10"
                    placeholder="Enter admin password"
                    required
                    minLength={8}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                  >
                    {showPassword ? (
                      <svg
                        className="w-5 h-5"
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
                        className="w-5 h-5"
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
                <p className="text-xs text-text-muted mt-1">
                  Minimum 8 characters
                </p>
              </div>

              <div className="mb-6">
                <label
                  htmlFor="setup-confirm-password"
                  className="block text-sm font-medium text-text-primary mb-1"
                >
                  Confirm Password
                </label>
                <input
                  id="setup-confirm-password"
                  type={showPassword ? "text" : "password"}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
                  placeholder="Confirm your password"
                  required
                />
              </div>
            </>
          )}

          {error && (
            <div
              role="alert"
              aria-live="assertive"
              className="mb-4 p-3 bg-status-error/10 border border-status-error/20 rounded text-status-error text-sm"
            >
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
          >
            {isSubmitting ? "Setting up..." : "Complete Setup"}
          </button>
        </form>
      </div>
    </div>
  );
}

export async function checkSetupStatus(): Promise<SetupStatusResponse> {
  try {
    const response = await fetch(`${API_BASE}/api/setup/status`);
    if (response.ok) {
      return await response.json();
    }
  } catch {
    // If we can't reach the API, assume setup is complete
  }
  return { needsSetup: false };
}
