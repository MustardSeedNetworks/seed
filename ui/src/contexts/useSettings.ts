/**
 * useSettings Hooks
 *
 * Provides backward-compatible access to settings via ProfileContext.
 * Settings are now stored in the active profile - profiles own everything.
 *
 * This adapter maintains the original SettingsContextValue interface
 * for compatibility with existing components.
 */

import type { ThresholdSettings } from '../types/generated/settings-response';
import type { CardSettings, DisplayOptions, IperfSettings, SaveStatus } from '../types/settings';
import { type SettingsSaveStatus, useProfileContext } from './profileContext';

/**
 * Context value type for settings management (backward compatibility)
 */
export interface SettingsContextValue {
  // Core settings state
  cardSettings: CardSettings;
  displayOptions: DisplayOptions;
  iperfSettings: IperfSettings;
  thresholds: ThresholdSettings;

  // Save status indicators
  status: {
    cards: SaveStatus;
    display: SaveStatus;
    iperf: SaveStatus;
    thresholds: SaveStatus;
  };

  // Update methods - trigger auto-save with debounce
  updateCardSettings: (updates: Partial<CardSettings>) => void;
  updateDisplayOptions: (updates: Partial<DisplayOptions>) => void;
  updateIperfSettings: (updates: Partial<IperfSettings>) => void;
  updateThresholds: (updates: Partial<ThresholdSettings>) => void;

  // Full refresh from API
  refreshSettings: () => Promise<void>;

  // Flag to check if initial load is complete
  isLoaded: boolean;
}

/**
 * Convert ProfileContext's SettingsSaveStatus to the old SaveStatus type.
 */
function convertSaveStatus(status: SettingsSaveStatus): SaveStatus {
  switch (status) {
    case 'saving':
      return 'saving';
    case 'saved':
      return 'saved';
    case 'error':
      return 'error';
    default:
      return 'idle';
  }
}

/**
 * Hook to access settings via ProfileContext.
 * Maintains backward compatibility with SettingsContext interface.
 *
 * @returns Settings context value
 * @throws Error if used outside ProfileProvider
 */
export function useSettings(): SettingsContextValue {
  const {
    cardSettings,
    displayOptions,
    iperfSettings,
    thresholds,
    settingsStatus,
    isSettingsLoaded,
    updateCardSettings,
    updateDisplayOptions,
    updateIperfSettings,
    updateThresholds,
    refreshSettings,
  } = useProfileContext();

  const saveStatus = convertSaveStatus(settingsStatus);

  return {
    cardSettings: cardSettings as CardSettings,
    displayOptions: displayOptions as DisplayOptions,
    iperfSettings: iperfSettings as IperfSettings,
    thresholds: thresholds as ThresholdSettings,
    status: {
      cards: saveStatus,
      display: saveStatus,
      iperf: saveStatus,
      thresholds: saveStatus,
    },
    updateCardSettings: updateCardSettings as (updates: Partial<CardSettings>) => void,
    updateDisplayOptions: updateDisplayOptions as (updates: Partial<DisplayOptions>) => void,
    updateIperfSettings: updateIperfSettings as (updates: Partial<IperfSettings>) => void,
    updateThresholds: updateThresholds as (updates: Partial<ThresholdSettings>) => void,
    refreshSettings,
    isLoaded: isSettingsLoaded,
  };
}
