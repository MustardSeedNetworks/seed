/**
 * Setup State Hook
 *
 * Manages state for the initial setup wizard flow including
 * setup status, suggested password, and setup token.
 *
 * Extracted from app.tsx to reduce component complexity (#889).
 */

import { useEffect, useState } from "react";
import { checkSetupStatus } from "../components/setup/setupApi";

interface UseSetupStateReturn {
  /** Whether the system needs initial setup (null = loading) */
  needsSetup: boolean | null;
  /** Suggested password from the backend (shown to user during setup) */
  suggestedPassword: string | undefined;
  /** Username from config (fixes #768) */
  setupUsername: string | undefined;
  /** Setup token for secure setup completion (security fix #724, #758) */
  setupToken: string | undefined;
  /** Mark setup as complete (clears setup state) */
  completeSetup: () => void;
}

/**
 * Hook for managing setup wizard state.
 * Automatically checks setup status on mount.
 *
 * @returns Object with setup state and completion handler
 */
export function useSetupState(): UseSetupStateReturn {
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
  const [suggestedPassword, setSuggestedPassword] = useState<string | undefined>(undefined);
  const [setupUsername, setSetupUsername] = useState<string | undefined>(undefined);
  const [setupToken, setSetupToken] = useState<string | undefined>(undefined);

  // Check if setup is needed on mount
  useEffect(() => {
    checkSetupStatus().then((status) => {
      setNeedsSetup(status.needsSetup);
      if (status.suggestedPassword) {
        setSuggestedPassword(status.suggestedPassword);
      }
      if (status.username) {
        setSetupUsername(status.username);
      }
      // Security fix #724, #758: Capture setup token for setup completion
      if (status.setupToken) {
        setSetupToken(status.setupToken);
      }
    });
  }, []);

  // Mark setup as complete
  const completeSetup = () => {
    setNeedsSetup(false);
    setSuggestedPassword(undefined);
    setSetupToken(undefined);
  };

  return {
    needsSetup,
    suggestedPassword,
    setupUsername,
    setupToken,
    completeSetup,
  };
}

export default useSetupState;
