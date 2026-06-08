/**
 * Main Application Component
 *
 * Root component for The Seed by Mustard Seed Networks. After the B1 refactor
 * this file is intentionally small: it owns only the pre-dashboard GATING —
 * setup wizard, loading, and login — plus session-expiration handling.
 *
 * All authenticated runtime wiring (theme, settings, interfaces, cards, SSE,
 * polling, run-all-tests orchestration) lives in `useAppOrchestration`, and the
 * authenticated UI is rendered by `<AppShell>`. The orchestration hook is called
 * unconditionally here (regardless of auth) so hook/effect timing and global
 * side effects (theme on the login screen, logger auth toggle) are unchanged —
 * the split is purely structural.
 */

import type { JSX } from 'react';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { setSessionExpiredCallback } from './api';
import { AppShell } from './app/AppShell';
import { LoginForm } from './app/LoginForm';
import { useAppOrchestration } from './app/useAppOrchestration';
import { SetupWizard } from './components/setup/SetupWizard';
import { useAuth } from './hooks/useAuth';
import { useSetupState } from './hooks/useSetupState';

/**
 * Main App Component — authentication/setup gating and mount.
 */
function App(): JSX.Element {
  const { t } = useTranslation('common');
  const { isAuthenticated, login, logout, isLoading, error } = useAuth();

  const [sessionExpired, setSessionExpired] = useState(false);

  // Setup wizard state (extracted to hook #889)
  const { needsSetup, suggestedPassword, setupUsername, setupToken, completeSetup } =
    useSetupState();

  // Handle session expiration via API client callback
  useEffect(() => {
    setSessionExpiredCallback(() => {
      setSessionExpired(true);
      logout();
    });
    return (): void => {
      setSessionExpiredCallback(null);
    };
  }, [logout]);

  // All authenticated runtime wiring. Called unconditionally (before any gating
  // return) so hook order is stable and effect timing matches the pre-refactor
  // god component.
  const orchestration = useAppOrchestration({ isAuthenticated });

  const authError = sessionExpired ? 'Session expired. Please log in again.' : error;

  const handleLogin = useCallback(
    async (username: string, password: string) => {
      // Clear the "session expired" banner at the START of every attempt
      // — not only on success. Otherwise a stale session-expired flag
      // (set by the previous logout/timeout) masks the real "Invalid
      // credentials" error from the new attempt, because authError
      // prioritizes sessionExpired over error. The user clicked Login,
      // so they're acknowledging the stale-session notice; whatever
      // happens next should reflect THIS attempt.
      setSessionExpired(false);
      return login(username, password);
    },
    [login],
  );

  // Show setup wizard if needed (before auth check)
  if (needsSetup === true) {
    return (
      <SetupWizard
        onComplete={completeSetup}
        onLogin={login}
        suggestedPassword={suggestedPassword}
        username={setupUsername}
        setupToken={setupToken} // Security fix #724, #758
      />
    );
  }

  // Show loading while checking setup status
  if (needsSetup === null) {
    return (
      <div className="min-h-screen flex-center">
        <div className="text-text-muted">{t('status.loading')}</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginForm onLogin={handleLogin} isLoading={isLoading} error={authError} />;
  }

  return <AppShell orchestration={orchestration} logout={logout} />;
}

export default App;
