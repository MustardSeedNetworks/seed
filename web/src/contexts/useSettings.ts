/**
 * useSettings Hooks
 *
 * Hooks to access settings context in components.
 * Separated from SettingsContext.tsx for react-refresh compliance.
 */

import { useContext } from "react";
import { SettingsContext, SettingsContextValue } from "./settingsContextDef";

/**
 * Hook to access settings context, throws if used outside SettingsProvider.
 *
 * @returns Settings context value
 * @throws Error if used outside SettingsProvider
 */
export function useSettings(): SettingsContextValue {
  const context = useContext(SettingsContext);
  if (!context) {
    throw new Error("useSettings must be used within a SettingsProvider");
  }
  return context;
}

/**
 * Hook to optionally access settings context, returns null if outside provider.
 *
 * @returns Settings context value or null
 */
export function useSettingsOptional(): SettingsContextValue | null {
  return useContext(SettingsContext);
}
