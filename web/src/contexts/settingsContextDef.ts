/**
 * Settings Context
 *
 * Provides the React Context for application settings.
 * Separated from SettingsContext.tsx for react-refresh compliance.
 */

import { createContext } from "react";
import {
  CardSettings,
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

/** React Context for application settings */
export const SettingsContext = createContext<SettingsContextValue | null>(null);
