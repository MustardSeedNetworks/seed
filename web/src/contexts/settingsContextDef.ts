/**
 * Settings Context
 *
 * Provides the React Context for application settings.
 * Separated from SettingsContext.tsx for react-refresh compliance.
 */

import { createContext } from "react";
import {
  FABOptions,
  DisplayOptions,
  IperfSettings,
  SettingsThresholds,
  SaveStatus,
} from "../types/settings";

/**
 * Context value type for settings management
 */
export interface SettingsContextValue {
  // Core settings state
  fabOptions: FABOptions;
  displayOptions: DisplayOptions;
  iperfSettings: IperfSettings;
  thresholds: SettingsThresholds;

  // Save status indicators
  status: {
    fab: SaveStatus;
    display: SaveStatus;
    iperf: SaveStatus;
    thresholds: SaveStatus;
  };

  // Update methods - trigger auto-save with debounce
  updateFabOptions: (updates: Partial<FABOptions>) => void;
  updateDisplayOptions: (updates: Partial<DisplayOptions>) => void;
  updateIperfSettings: (updates: Partial<IperfSettings>) => void;
  updateThresholds: (updates: Partial<SettingsThresholds>) => void;

  // Full refresh from API
  refreshSettings: () => Promise<void>;

  // Flag to check if initial load is complete
  isLoaded: boolean;
}

/** React Context for application settings */
export const SettingsContext = createContext<SettingsContextValue | null>(null);
