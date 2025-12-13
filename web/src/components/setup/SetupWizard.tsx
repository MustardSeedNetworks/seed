import { useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface SetupWizardProps {
  onComplete: () => void;
}

interface SetupStatusResponse {
  needsSetup: boolean;
  username?: string;
}

export function SetupWizard({ onComplete }: SetupWizardProps) {
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

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

          <div className="mb-4">
            <label
              htmlFor="setup-password"
              className="block text-sm font-medium text-text-primary mb-1"
            >
              Password
            </label>
            <input
              id="setup-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="Enter admin password"
              required
              minLength={8}
            />
            <p className="text-xs text-text-muted mt-1">Minimum 8 characters</p>
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
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="Confirm your password"
              required
            />
          </div>

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
