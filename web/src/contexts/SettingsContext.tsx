/**
 * SettingsContext - Centralized settings management
 *
 * Provides a React Context for settings that are accessed by multiple components.
 * Settings are loaded from and saved to the backend API config file.
 *
 * All settings (FABOptions, DisplayOptions, IperfSettings, Thresholds) are
 * persisted to the backend config file for persistence across sessions.
 */

import { useState, useCallback, useEffect, useRef, ReactNode } from "react";
import { getAuthHeaders } from "../hooks/useAuth";
import {
  FABOptions,
  DisplayOptions,
  IperfSettings,
  SettingsThresholds,
  SaveStatus,
  DEFAULT_FAB_OPTIONS,
  DEFAULT_DISPLAY_OPTIONS,
  DEFAULT_IPERF_SETTINGS,
  DEFAULT_THRESHOLDS,
} from "../types/settings";
import { SettingsContext, SettingsContextValue } from "./settingsContextDef";

// ============================================================================
// Provider Component
// ============================================================================

const API_BASE = import.meta.env.VITE_API_BASE || "";
const DEBOUNCE_MS = 800;

interface SettingsProviderProps {
  children: ReactNode;
}

/**
 * Context provider that manages application settings state and API synchronization.
 */
export function SettingsProvider({ children }: SettingsProviderProps) {
  // State - initialized with defaults, will be updated from API
  const [fabOptions, setFabOptions] = useState<FABOptions>(DEFAULT_FAB_OPTIONS);
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>(DEFAULT_DISPLAY_OPTIONS);
  const [iperfSettings, setIperfSettings] = useState<IperfSettings>(DEFAULT_IPERF_SETTINGS);
  const [thresholds, setThresholds] = useState<SettingsThresholds>(DEFAULT_THRESHOLDS);

  // Status indicators
  const [status, setStatus] = useState<SettingsContextValue["status"]>({
    fab: "idle",
    display: "idle",
    iperf: "idle",
    thresholds: "idle",
  });

  // Tracking refs
  const [isLoadedState, setIsLoadedState] = useState(false);
  // Using Map for type-safe dynamic key access
  const debounceTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  // ============================================================================
  // Load All Settings from API
  // ============================================================================

  const loadSettingsFromAPI = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();

        // Load thresholds
        if (data.thresholds) {
          setThresholds((prev) => ({ ...prev, ...data.thresholds }));
        }

        // Load FAB options
        if (data.fabOptions) {
          setFabOptions((prev) => ({ ...prev, ...data.fabOptions }));
        }

        // Load display options
        if (data.displayOptions) {
          setDisplayOptions((prev) => ({ ...prev, ...data.displayOptions }));
        }

        // Load iperf settings
        if (data.iperf) {
          setIperfSettings((prev) => ({ ...prev, ...data.iperf }));
        }
      }
    } catch (err) {
      console.error("Failed to fetch settings:", err);
    }
  }, []);

  const refreshSettings = useCallback(async () => {
    await loadSettingsFromAPI();
  }, [loadSettingsFromAPI]);

  // Initial load - fetch all settings from API
  useEffect(() => {
    let mounted = true;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Initial data fetch pattern
    loadSettingsFromAPI().finally(() => {
      if (mounted) {
        setIsLoadedState(true);
      }
    });
    return () => {
      mounted = false;
    };
  }, [loadSettingsFromAPI]);

  // Cleanup timers on unmount
  useEffect(() => {
    const timers = debounceTimers.current;
    return () => {
      timers.forEach((timer) => clearTimeout(timer));
    };
  }, []);

  // ============================================================================
  // Debounced Save Helper
  // ============================================================================

  const debounceSave = useCallback(
    (key: string, saveFn: () => void | Promise<void>, delay: number = DEBOUNCE_MS) => {
      const existingTimer = debounceTimers.current.get(key);
      if (existingTimer) {
        clearTimeout(existingTimer);
      }

      setStatus((prev) => ({ ...prev, [key]: "saving" as SaveStatus }));

      const newTimer = setTimeout(async () => {
        try {
          await saveFn();
          setStatus((prev) => ({ ...prev, [key]: "saved" as SaveStatus }));
          setTimeout(() => {
            setStatus((prev) => ({ ...prev, [key]: "idle" as SaveStatus }));
          }, 2000);
        } catch {
          setStatus((prev) => ({ ...prev, [key]: "error" as SaveStatus }));
        }
      }, delay);
      debounceTimers.current.set(key, newTimer);
    },
    []
  );

  // ============================================================================
  // Save to Backend API Helper
  // ============================================================================

  const saveToBackend = async (updates: Record<string, unknown>) => {
    const response = await fetch(`${API_BASE}/api/settings`, {
      method: "PUT",
      headers: {
        ...getAuthHeaders(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(updates),
    });
    if (!response.ok) {
      throw new Error("Failed to save settings");
    }
  };

  // ============================================================================
  // Update Methods - update state and trigger debounced save to backend
  // ============================================================================

  const updateFabOptions = useCallback(
    (updates: Partial<FABOptions>) => {
      setFabOptions((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("fab", () => saveToBackend({ fabOptions: next }));
        return next;
      });
    },
    [debounceSave]
  );

  const updateDisplayOptions = useCallback(
    (updates: Partial<DisplayOptions>) => {
      setDisplayOptions((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("display", () => saveToBackend({ displayOptions: next }));
        return next;
      });
    },
    [debounceSave]
  );

  const updateIperfSettings = useCallback(
    (updates: Partial<IperfSettings>) => {
      setIperfSettings((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("iperf", () => saveToBackend({ iperf: next }));
        return next;
      });
    },
    [debounceSave]
  );

  const updateThresholds = useCallback(
    (updates: Partial<SettingsThresholds>) => {
      setThresholds((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("thresholds", () => saveToBackend({ thresholds: next }));
        return next;
      });
    },
    [debounceSave]
  );

  // ============================================================================
  // Context Value
  // ============================================================================

  const value: SettingsContextValue = {
    fabOptions,
    displayOptions,
    iperfSettings,
    thresholds,
    status,
    updateFabOptions,
    updateDisplayOptions,
    updateIperfSettings,
    updateThresholds,
    refreshSettings,
    isLoaded: isLoadedState,
  };

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
}
