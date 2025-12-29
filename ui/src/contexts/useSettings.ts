/**
 * useSettings Hooks
 *
 * Provides backward-compatible access to settings via ProfileContext.
 * Settings are now stored in the active profile - profiles own everything.
 *
 * This adapter maintains the original SettingsContextValue interface
 * for compatibility with existing components.
 */

import { useMemo } from "react";
import type {
  CardSettings,
  DisplayOptions,
  IperfSettings,
  SaveStatus,
  SettingsThresholds,
} from "../types/settings";
import {
  type SettingsSaveStatus,
  useProfileContext,
  useProfileContextOptional,
} from "./ProfileContext";

/**
 * Context value type for settings management (backward compatibility)
 */
export interface SettingsContextValue {
  // Core settings state
  cardSettings: CardSettings;
  displayOptions: DisplayOptions;
  iperfSettings: IperfSettings;
  thresholds: SettingsThresholds;

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
  updateThresholds: (updates: Partial<SettingsThresholds>) => void;

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
    case "saving":
      return "saving";
    case "saved":
      return "saved";
    case "error":
      return "error";
    default:
      return "idle";
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

  // Create the backward-compatible status object
  const status = useMemo(() => {
    const saveStatus = convertSaveStatus(settingsStatus);
    return {
      cards: saveStatus,
      display: saveStatus,
      iperf: saveStatus,
      thresholds: saveStatus,
    };
  }, [settingsStatus]);

  // Create adapter that maps new types to old types
  return useMemo(
    () => ({
      cardSettings: cardSettings as CardSettings,
      displayOptions: displayOptions as DisplayOptions,
      iperfSettings: iperfSettings as IperfSettings,
      thresholds: thresholds as SettingsThresholds,
      status,
      updateCardSettings: updateCardSettings as (updates: Partial<CardSettings>) => void,
      updateDisplayOptions: updateDisplayOptions as (updates: Partial<DisplayOptions>) => void,
      updateIperfSettings: updateIperfSettings as (updates: Partial<IperfSettings>) => void,
      updateThresholds: updateThresholds as (updates: Partial<SettingsThresholds>) => void,
      refreshSettings,
      isLoaded: isSettingsLoaded,
    }),
    [
      cardSettings,
      displayOptions,
      iperfSettings,
      thresholds,
      status,
      updateCardSettings,
      updateDisplayOptions,
      updateIperfSettings,
      updateThresholds,
      refreshSettings,
      isSettingsLoaded,
    ],
  );
}

/**
 * Hook to optionally access settings context.
 * Returns null if used outside ProfileProvider (for non-critical usage).
 *
 * @returns Settings context value or null
 */
export function useSettingsOptional(): SettingsContextValue | null {
  const context = useProfileContextOptional();

  // Create the backward-compatible status object
  const status = useMemo(() => {
    if (!context) return null;
    const saveStatus = convertSaveStatus(context.settingsStatus);
    return {
      cards: saveStatus,
      display: saveStatus,
      iperf: saveStatus,
      thresholds: saveStatus,
    };
  }, [context]);

  // Create adapter that maps new types to old types
  return useMemo(() => {
    if (!context || !status) return null;
    return {
      cardSettings: context.cardSettings as CardSettings,
      displayOptions: context.displayOptions as DisplayOptions,
      iperfSettings: context.iperfSettings as IperfSettings,
      thresholds: context.thresholds as SettingsThresholds,
      status,
      updateCardSettings: context.updateCardSettings as (updates: Partial<CardSettings>) => void,
      updateDisplayOptions: context.updateDisplayOptions as (
        updates: Partial<DisplayOptions>,
      ) => void,
      updateIperfSettings: context.updateIperfSettings as (updates: Partial<IperfSettings>) => void,
      updateThresholds: context.updateThresholds as (updates: Partial<SettingsThresholds>) => void,
      refreshSettings: context.refreshSettings,
      isLoaded: context.isSettingsLoaded,
    };
  }, [context, status]);
}
