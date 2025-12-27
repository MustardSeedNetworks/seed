/**
 * SettingsContext - Centralized settings management
 *
 * Provides a React Context for settings that are accessed by multiple components.
 * Settings are loaded from and saved to the backend API config file.
 *
 * All settings (CardSettings, DisplayOptions, IperfSettings, Thresholds) are
 * persisted to the backend config file for persistence across sessions.
 *
 * NOTE: The backend is the single source of truth for default values.
 * Defaults are loaded from /api/settings/defaults instead of hardcoded constants.
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
} from "../types/settings";
import { SettingsContext, SettingsContextValue } from "./settingsContextDef";
import { useDefaults } from "../hooks/useDefaults";

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
 * Uses defaults from the backend via /api/settings/defaults (single source of truth).
 */
export function SettingsProvider({ children }: SettingsProviderProps) {
  // Load defaults from the backend (single source of truth)
  const { defaults } = useDefaults();

  // Convert backend defaults to the CardSettings format
  const defaultCardSettings: CardSettings = {
    link: defaults.cardSettings.link,
    switch: defaults.cardSettings.switch,
    vlan: defaults.cardSettings.vlan,
    network: defaults.cardSettings.network,
    gateway: defaults.cardSettings.gateway,
    dns: defaults.cardSettings.dns,
    healthChecks: defaults.cardSettings.healthChecks,
    networkDiscovery: defaults.cardSettings.networkDiscovery,
    performance: defaults.cardSettings.performance,
  };

  const defaultDisplayOptions: DisplayOptions = {
    showPublicIP: defaults.displayOptions.showPublicIP,
    unitSystem: defaults.displayOptions.unitSystem as "sae" | "metric",
  };

  const defaultIperfSettings: IperfSettings = {
    server: defaults.iperf.server,
    port: defaults.iperf.port,
    protocol: defaults.iperf.protocol as "tcp" | "udp",
    direction: defaults.iperf.direction as "download" | "upload" | "both",
    duration: defaults.iperf.duration,
    serverPort: defaults.iperf.serverPort,
    enableServer: defaults.iperf.enableServer,
  };

  const defaultThresholds: SettingsThresholds = {
    dns: defaults.thresholds.dns,
    gateway: defaults.thresholds.gateway,
    wifi: defaults.thresholds.wifi,
    customPing: defaults.thresholds.customPing,
    customTcp: defaults.thresholds.customTcp,
    customHttp: defaults.thresholds.customHttp,
    httpTimings: defaults.thresholds.httpTimings,
  };

  // State - initialized with backend defaults, will be updated from API
  const [cardSettings, setCardSettings] =
    useState<CardSettings>(defaultCardSettings);
  const [displayOptions, setDisplayOptions] =
    useState<DisplayOptions>(defaultDisplayOptions);
  const [iperfSettings, setIperfSettings] =
    useState<IperfSettings>(defaultIperfSettings);
  const [thresholds, setThresholds] =
    useState<SettingsThresholds>(defaultThresholds);

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
  const debounceTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map()
  );
  const statusResetTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map()
  );
  const saveControllers = useRef<Map<string, AbortController>>(new Map());

  // ============================================================================
  // Load All Settings from API
  // ============================================================================

  const loadSettingsFromAPI = useCallback(async (signal?: AbortSignal) => {
    try {
      const data = await api.get<Record<string, unknown>>("/api/settings", {
        signal,
      });

      if (!isMountedRef.current || signal?.aborted) return;

      // Load thresholds
      if (data.thresholds && typeof data.thresholds === "object") {
        setThresholds((prev) => ({
          ...prev,
          ...(data.thresholds as Partial<SettingsThresholds>),
        }));
      }

      // Load card settings from new API structure
      if (data.cardSettings && typeof data.cardSettings === "object") {
        const cs = data.cardSettings as Record<string, unknown>;
        setCardSettings((prev) => ({
          ...prev,
          link: {
            enabled: (cs.link as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.link as { autoRunOnLink?: boolean })?.autoRunOnLink ?? true,
          },
          switch: {
            enabled: (cs.switch as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.switch as { autoRunOnLink?: boolean })?.autoRunOnLink ?? true,
          },
          vlan: {
            enabled: (cs.vlan as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.vlan as { autoRunOnLink?: boolean })?.autoRunOnLink ?? true,
          },
          network: {
            enabled: (cs.network as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.network as { autoRunOnLink?: boolean })?.autoRunOnLink ??
              true,
          },
          gateway: {
            enabled: (cs.gateway as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.gateway as { autoRunOnLink?: boolean })?.autoRunOnLink ??
              true,
          },
          dns: {
            enabled: (cs.dns as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.dns as { autoRunOnLink?: boolean })?.autoRunOnLink ?? true,
          },
          healthChecks: {
            enabled:
              (cs.healthChecks as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.healthChecks as { autoRunOnLink?: boolean })?.autoRunOnLink ??
              true,
          },
          networkDiscovery: {
            enabled:
              (cs.networkDiscovery as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.networkDiscovery as { autoRunOnLink?: boolean })
                ?.autoRunOnLink ?? true,
          },
          performance: {
            enabled: (cs.performance as { visible?: boolean })?.visible ?? true,
            autoRunOnLink:
              (cs.performance as { autoRunOnLink?: boolean })?.autoRunOnLink ??
              true,
            speedtest: {
              enabled:
                (
                  (cs.performance as { speedtest?: { enabled?: boolean } })
                    ?.speedtest as { enabled?: boolean }
                )?.enabled ?? true,
              autoRunOnLink:
                (
                  (
                    cs.performance as {
                      speedtest?: { autoRunOnLink?: boolean };
                    }
                  )?.speedtest as { autoRunOnLink?: boolean }
                )?.autoRunOnLink ?? true,
            },
            iperf: {
              enabled:
                (
                  (cs.performance as { iperf?: { enabled?: boolean } })
                    ?.iperf as { enabled?: boolean }
                )?.enabled ?? false,
              autoRunOnLink:
                (
                  (cs.performance as { iperf?: { autoRunOnLink?: boolean } })
                    ?.iperf as { autoRunOnLink?: boolean }
                )?.autoRunOnLink ?? false,
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
        setIperfSettings((prev) => ({
          ...prev,
          ...(data.iperf as Partial<IperfSettings>),
        }));
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

          // Fixes #912: Reset error status after a delay so user can see it
          const errorResetTimer = setTimeout(() => {
            if (!isMountedRef.current) return;
            setStatus((prev) => ({ ...prev, [key]: "idle" as SaveStatus }));
          }, 5000); // Longer delay for errors
          statusResetTimers.current.set(key, errorResetTimer);
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

  const saveToBackend = async (
    updates: Record<string, unknown>,
    signal: AbortSignal
  ) => {
    await api.put<{ status: string }>("/api/settings", updates, { signal });
  };

  // ============================================================================
  // Update Methods - update state and trigger debounced save to backend
  // ============================================================================

  const updateCardSettings = useCallback(
    (updates: Partial<CardSettings>) => {
      setCardSettings((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("cards", (signal) =>
          saveToBackend({ cardSettings: next }, signal)
        );
        return next;
      });
    },
    [debounceSave]
  );

  const updateDisplayOptions = useCallback(
    (updates: Partial<DisplayOptions>) => {
      setDisplayOptions((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("display", (signal) =>
          saveToBackend({ displayOptions: next }, signal)
        );
        return next;
      });
    },
    [debounceSave]
  );

  const updateIperfSettings = useCallback(
    (updates: Partial<IperfSettings>) => {
      setIperfSettings((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("iperf", (signal) =>
          saveToBackend({ iperf: next }, signal)
        );
        return next;
      });
    },
    [debounceSave]
  );

  const updateThresholds = useCallback(
    (updates: Partial<SettingsThresholds>) => {
      setThresholds((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("thresholds", (signal) =>
          saveToBackend({ thresholds: next }, signal)
        );
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

  return (
    <SettingsContext.Provider value={value}>
      {children}
    </SettingsContext.Provider>
  );
}
