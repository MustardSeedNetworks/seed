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
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { setSessionExpiredCallback } from './api';
import { AppShell } from './app/AppShell';
import { LoginForm } from './app/LoginForm';
import { useAppOrchestration } from './app/useAppOrchestration';
import { SetupWizard } from './components/setup/SetupWizard';
import { LicenseProvider } from './contexts/LicenseContext';
import { RoleProvider } from './contexts/RoleContext';
import { useAuth } from './hooks/useAuth';
import { useSetupState } from './hooks/useSetupState';

/**
 * Main App Component — authentication/setup gating and mount.
 */
function App(): JSX.Element {
  const { t } = useTranslation('common');
  const { isAuthenticated, login, completeSecondFactor, logout, expireSession, isLoading, error } =
    useAuth();

  // Setup wizard state (extracted to hook #889)
  const { needsSetup, suggestedPassword, setupUsername, setupToken, completeSetup } =
    useSetupState();

  // Handle session expiration via API client callback.
  //
  // Calls expireSession() (clears state, sets the error banner), NOT logout():
  // a 401 that survived a refresh attempt already means the server considers
  // the session gone, so logout()'s POST /api/v1/auth/logout has nothing left
  // to invalidate. Sending it anyway was actively harmful — background
  // fetches issued before authentication settles (RoleProvider, LicenseProvider)
  // 401 on the login screen and fire this callback, and logout()'s real
  // network call could land *after* a concurrent re-login's response, its
  // Set-Cookie clearing the session the fresh login had just established
  // (seed#2407 CI flake in e2e/auth-complete.spec.ts "should allow re-login
  // after session expiry"). expireSession() only ever touches client state.
  useEffect(() => {
    setSessionExpiredCallback(() => {
      expireSession();
    });
    return (): void => {
      setSessionExpiredCallback(null);
    };
  }, [expireSession]);

  // All authenticated runtime wiring. Called unconditionally (before any gating
  // return) so hook order is stable and effect timing matches the pre-refactor
  // god component.
  const orchestration = useAppOrchestration({ isAuthenticated });

  // Show setup wizard if needed (before auth check)
  let content: JSX.Element;
  if (needsSetup === true) {
    content = (
      <SetupWizard
        onComplete={completeSetup}
        onLogin={login}
        suggestedPassword={suggestedPassword}
        username={setupUsername}
        setupToken={setupToken} // Security fix #724, #758
      />
    );
  } else if (needsSetup === null) {
    // Show loading while checking setup status
    content = (
      <div className="min-h-screen flex-center">
        <div className="text-text-muted">{t('status.loading')}</div>
      </div>
    );
  } else if (!isAuthenticated) {
    content = (
      <LoginForm
        onLogin={login}
        onSecondFactor={completeSecondFactor}
        isLoading={isLoading}
        error={error}
      />
    );
  } else {
    content = <AppShell orchestration={orchestration} logout={logout} />;
  }

  // RoleProvider/LicenseProvider live here — not in main.tsx wrapping <App>
  // from the outside — because only App knows `isAuthenticated`. Gating
  // their fetches on it stops GET /api/v1/users/me and GET /api/v1/license
  // from ever firing while unauthenticated (the setup wizard, the loading
  // screen, and the login form all render above); previously both fetched
  // unconditionally on mount, 401ing on the login screen and — via the API
  // client's automatic refresh-then-session-expired handling — firing real
  // logout traffic that could race a concurrent real login (seed#2422).
  return (
    <LicenseProvider isAuthenticated={isAuthenticated}>
      <RoleProvider isAuthenticated={isAuthenticated}>{content}</RoleProvider>
    </LicenseProvider>
  );
}

export default App;
