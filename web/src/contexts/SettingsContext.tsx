/**
 * SettingsContext - Centralized settings management
 *
 * Provides a React Context for settings that are accessed by multiple components.
 * Settings are loaded from and saved to the backend API config file.
 *
 * All settings (CardSettings, DisplayOptions, IperfSettings, Thresholds) are
 * persisted to the backend config file for persistence across sessions.
 */

import { useState, useCallback, useEffect, useRef, ReactNode } from "react";
import { logger, LogComponents } from "../lib/logger";
import { api } from "../lib/api";
import {
  CardSettings,
  DisplayOptions,
  IperfSettings,
  SettingsThresholds,
  SaveStatus,
  DEFAULT_CARD_SETTINGS,
  DEFAULT_DISPLAY_OPTIONS,
  DEFAULT_IPERF_SETTINGS,
  DEFAULT_THRESHOLDS,
} from "../types/settings";
import { SettingsContext, SettingsContextValue } from "./settingsContextDef";

// ============================================================================
// Provider Component
// ============================================================================

// Debounce delay increased to reduce API calls and improve performance (fixes #673)
const DEBOUNCE_MS = 1000;

interface SettingsProviderProps {
  children: ReactNode;
}

/**
 * Context provider that manages application settings state and API synchronization.
 */
export function SettingsProvider({ children }: SettingsProviderProps) {
  // State - initialized with defaults, will be updated from API
  const [cardSettings, setCardSettings] = useState<CardSettings>(DEFAULT_CARD_SETTINGS);
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>(DEFAULT_DISPLAY_OPTIONS);
  const [iperfSettings, setIperfSettings] = useState<IperfSettings>(DEFAULT_IPERF_SETTINGS);
  const [thresholds, setThresholds] = useState<SettingsThresholds>(DEFAULT_THRESHOLDS);

  // Status indicators
  const [status, setStatus] = useState<SettingsContextValue["status"]>({
    cards: "idle",
    display: "idle",
    iperf: "idle",
    thresholds: "idle",
  });

  // Tracking refs
  const isMountedRef = useRef(true);
  const [isLoadedState, setIsLoadedState] = useState(false);
  // Using Map for type-safe dynamic key access
  const debounceTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const statusResetTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const saveControllers = useRef<Map<string, AbortController>>(new Map());

  // ============================================================================
  // Load All Settings from API
  // ============================================================================

  const loadSettingsFromAPI = useCallback(async (signal?: AbortSignal) => {
    try {
      const data = await api.get<Record<string, unknown>>("/api/settings", { signal });

      if (!isMountedRef.current || signal?.aborted) return;

      // Load thresholds
      if (data.thresholds && typeof data.thresholds === "object") {
        setThresholds((prev) => ({ ...prev, ...(data.thresholds as Partial<SettingsThresholds>) }));
      }

      // Load card settings (migrate from old fabOptions if present)
      if (data.cardSettings && typeof data.cardSettings === "object") {
        setCardSettings((prev) => ({ ...prev, ...(data.cardSettings as Partial<CardSettings>) }));
      } else if (data.fabOptions && typeof data.fabOptions === "object") {
        const fabOptions = data.fabOptions as Record<string, unknown>;
        setCardSettings((prev) => ({
          ...prev,
          link: {
            enabled: true,
            autoRunOnLink: (fabOptions.runLink as boolean | undefined) ?? true,
          },
          switch: {
            enabled: true,
            autoRunOnLink: (fabOptions.runSwitch as boolean | undefined) ?? true,
          },
          vlan: {
            enabled: true,
            autoRunOnLink: (fabOptions.runVLAN as boolean | undefined) ?? true,
          },
          network: {
            enabled: true,
            autoRunOnLink: (fabOptions.runIPConfig as boolean | undefined) ?? true,
          },
          gateway: {
            enabled: true,
            autoRunOnLink: (fabOptions.runGateway as boolean | undefined) ?? true,
          },
          dns: { enabled: true, autoRunOnLink: (fabOptions.runDNS as boolean | undefined) ?? true },
          healthChecks: {
            enabled: true,
            autoRunOnLink: (fabOptions.runHealthChecks as boolean | undefined) ?? true,
          },
          networkDiscovery: {
            enabled: (fabOptions.runNetworkDiscovery as boolean | undefined) ?? true,
            autoRunOnLink: (fabOptions.autoScanOnLink as boolean | undefined) ?? true,
          },
          performance: {
            enabled: (fabOptions.runPerformance as boolean | undefined) ?? true,
            autoRunOnLink: (fabOptions.runPerformance as boolean | undefined) ?? true,
            speedtest: {
              enabled: (fabOptions.runSpeedtest as boolean | undefined) ?? true,
              autoRunOnLink: (fabOptions.runSpeedtest as boolean | undefined) ?? true,
            },
            iperf: {
              enabled: (fabOptions.runIperf as boolean | undefined) ?? false,
              autoRunOnLink: (fabOptions.runIperf as boolean | undefined) ?? false,
            },
          },
        }));
      }

      // Load display options
      if (data.displayOptions && typeof data.displayOptions === "object") {
        setDisplayOptions((prev) => ({
          ...prev,
          ...(data.displayOptions as Partial<DisplayOptions>),
        }));
      }

      // Load iperf settings
      if (data.iperf && typeof data.iperf === "object") {
        setIperfSettings((prev) => ({ ...prev, ...(data.iperf as Partial<IperfSettings>) }));
      }
    } catch (err) {
      if (!signal?.aborted) {
        logger.error(LogComponents.CONFIG, "Failed to fetch settings", err);
      }
    }
  }, []);

  const refreshSettings = useCallback(async () => {
    await loadSettingsFromAPI();
  }, [loadSettingsFromAPI]);

  // Initial load - fetch all settings from API
  useEffect(() => {
    const controller = new AbortController();

    loadSettingsFromAPI(controller.signal).finally(() => {
      if (isMountedRef.current && !controller.signal.aborted) {
        setIsLoadedState(true);
      }
    });
    return () => {
      controller.abort();
    };
  }, [loadSettingsFromAPI]);

  // Cleanup timers and in-flight requests on unmount
  useEffect(() => {
    const debounceTimersMap = debounceTimers.current;
    const statusResetTimersMap = statusResetTimers.current;
    const saveControllersMap = saveControllers.current;

    return () => {
      isMountedRef.current = false;

      debounceTimersMap.forEach((timer) => clearTimeout(timer));
      statusResetTimersMap.forEach((timer) => clearTimeout(timer));
      saveControllersMap.forEach((controller) => controller.abort());

      debounceTimersMap.clear();
      statusResetTimersMap.clear();
      saveControllersMap.clear();
    };
  }, []);

  // ============================================================================
  // Debounced Save Helper
  // ============================================================================

  const debounceSave = useCallback(
    (
      key: string,
      saveFn: (signal: AbortSignal) => void | Promise<void>,
      delay: number = DEBOUNCE_MS
    ) => {
      const existingTimer = debounceTimers.current.get(key);
      if (existingTimer) {
        clearTimeout(existingTimer);
      }

      const existingResetTimer = statusResetTimers.current.get(key);
      if (existingResetTimer) {
        clearTimeout(existingResetTimer);
        statusResetTimers.current.delete(key);
      }

      if (isMountedRef.current) {
        setStatus((prev) => ({ ...prev, [key]: "saving" as SaveStatus }));
      }

      const newTimer = setTimeout(async () => {
        debounceTimers.current.delete(key);

        // Cancel any in-flight request for this key before starting a new one.
        const existingController = saveControllers.current.get(key);
        if (existingController) {
          existingController.abort();
          saveControllers.current.delete(key);
        }

        const controller = new AbortController();
        saveControllers.current.set(key, controller);

        try {
          await saveFn(controller.signal);
          if (!isMountedRef.current || controller.signal.aborted) return;

          setStatus((prev) => ({ ...prev, [key]: "saved" as SaveStatus }));

          const resetTimer = setTimeout(() => {
            if (!isMountedRef.current) return;
            setStatus((prev) => ({ ...prev, [key]: "idle" as SaveStatus }));
          }, 2000);
          statusResetTimers.current.set(key, resetTimer);
        } catch {
          if (!isMountedRef.current || controller.signal.aborted) return;
          setStatus((prev) => ({ ...prev, [key]: "error" as SaveStatus }));
        } finally {
          const currentController = saveControllers.current.get(key);
          if (currentController === controller) {
            saveControllers.current.delete(key);
          }
        }
      }, delay);
      debounceTimers.current.set(key, newTimer);
    },
    []
  );

  // ============================================================================
  // Save to Backend API Helper
  // ============================================================================

  const saveToBackend = async (updates: Record<string, unknown>, signal: AbortSignal) => {
    await api.put<{ status: string }>("/api/settings", updates, { signal });
  };

  // ============================================================================
  // Update Methods - update state and trigger debounced save to backend
  // ============================================================================

  const updateCardSettings = useCallback(
    (updates: Partial<CardSettings>) => {
      setCardSettings((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("cards", (signal) => saveToBackend({ cardSettings: next }, signal));
        return next;
      });
    },
    [debounceSave]
  );

  const updateDisplayOptions = useCallback(
    (updates: Partial<DisplayOptions>) => {
      setDisplayOptions((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("display", (signal) => saveToBackend({ displayOptions: next }, signal));
        return next;
      });
    },
    [debounceSave]
  );

  const updateIperfSettings = useCallback(
    (updates: Partial<IperfSettings>) => {
      setIperfSettings((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("iperf", (signal) => saveToBackend({ iperf: next }, signal));
        return next;
      });
    },
    [debounceSave]
  );

  const updateThresholds = useCallback(
    (updates: Partial<SettingsThresholds>) => {
      setThresholds((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("thresholds", (signal) => saveToBackend({ thresholds: next }, signal));
        return next;
      });
    },
    [debounceSave]
  );

  // ============================================================================
  // Context Value
  // ============================================================================

  const value: SettingsContextValue = {
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
    isLoaded: isLoadedState,
  };

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
}
