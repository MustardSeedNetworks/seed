/**
 * App Drawers State Hook
 *
 * Manages the open/close state for all drawer and modal components
 * in the application (profiles, settings, help).
 *
 * Extracted from App.tsx to reduce component complexity (#889).
 */

import { useCallback, useState } from "react";

interface UseAppDrawersReturn {
  /** Whether the profiles drawer is open */
  profilesOpen: boolean;
  /** Whether the settings drawer is open */
  settingsOpen: boolean;
  /** Whether the help modal is open */
  helpOpen: boolean;
  /** Open the profiles drawer */
  openProfiles: () => void;
  /** Close the profiles drawer */
  closeProfiles: () => void;
  /** Open the settings drawer */
  openSettings: () => void;
  /** Close the settings drawer */
  closeSettings: () => void;
  /** Open the help modal */
  openHelp: () => void;
  /** Close the help modal */
  closeHelp: () => void;
}

/**
 * Hook for managing drawer and modal visibility state.
 *
 * @returns Object with drawer states and toggle functions
 */
export function useAppDrawers(): UseAppDrawersReturn {
  const [profilesOpen, setProfilesOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);

  const openProfiles = useCallback(() => setProfilesOpen(true), []);
  const closeProfiles = useCallback(() => setProfilesOpen(false), []);
  const openSettings = useCallback(() => setSettingsOpen(true), []);
  const closeSettings = useCallback(() => setSettingsOpen(false), []);
  const openHelp = useCallback(() => setHelpOpen(true), []);
  const closeHelp = useCallback(() => setHelpOpen(false), []);

  return {
    profilesOpen,
    settingsOpen,
    helpOpen,
    openProfiles,
    closeProfiles,
    openSettings,
    closeSettings,
    openHelp,
    closeHelp,
  };
}

export default useAppDrawers;
