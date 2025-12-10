/**
 * SettingsContext - Centralized settings management
 *
 * Provides a React Context for settings that are accessed by multiple components.
 * Settings are loaded from localStorage and backend API, with auto-save on changes.
 *
 * Phase 1: FABOptions, DisplayOptions, IperfSettings (most commonly accessed)
 * Future phases will migrate more settings from SettingsDrawer.
 */

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useRef,
  ReactNode,
} from "react";
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
  STORAGE_KEYS,
} from "../types/settings";

// ============================================================================
// Context Types
// ============================================================================

interface SettingsContextValue {
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

  // Full refresh from storage/API
  refreshSettings: () => Promise<void>;

  // Flag to check if initial load is complete
  isLoaded: boolean;
}

const SettingsContext = createContext<SettingsContextValue | null>(null);

// ============================================================================
// Provider Component
// ============================================================================

const API_BASE = import.meta.env.VITE_API_BASE || "";
const DEBOUNCE_MS = 800;

interface SettingsProviderProps {
  children: ReactNode;
}

// Helper to load initial values synchronously
function loadInitialFabOptions(): FABOptions {
  try {
    const saved = localStorage.getItem(STORAGE_KEYS.FAB_OPTIONS);
    if (saved) return { ...DEFAULT_FAB_OPTIONS, ...JSON.parse(saved) };
  } catch {
    /* ignore */
  }
  return DEFAULT_FAB_OPTIONS;
}

function loadInitialDisplayOptions(): DisplayOptions {
  try {
    const saved = localStorage.getItem(STORAGE_KEYS.DISPLAY_OPTIONS);
    if (saved) return { ...DEFAULT_DISPLAY_OPTIONS, ...JSON.parse(saved) };
  } catch {
    /* ignore */
  }
  return DEFAULT_DISPLAY_OPTIONS;
}

function loadInitialIperfSettings(): IperfSettings {
  try {
    const saved = localStorage.getItem(STORAGE_KEYS.IPERF_SETTINGS);
    if (saved) return { ...DEFAULT_IPERF_SETTINGS, ...JSON.parse(saved) };
  } catch {
    /* ignore */
  }
  return DEFAULT_IPERF_SETTINGS;
}

export function SettingsProvider({ children }: SettingsProviderProps) {
  // State - initialized with values from localStorage
  const [fabOptions, setFabOptions] = useState<FABOptions>(
    loadInitialFabOptions,
  );
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>(
    loadInitialDisplayOptions,
  );
  const [iperfSettings, setIperfSettings] = useState<IperfSettings>(
    loadInitialIperfSettings,
  );
  const [thresholds, setThresholds] =
    useState<SettingsThresholds>(DEFAULT_THRESHOLDS);

  // Status indicators
  const [status, setStatus] = useState<SettingsContextValue["status"]>({
    fab: "idle",
    display: "idle",
    iperf: "idle",
    thresholds: "idle",
  });

  // Tracking refs
  const [isLoadedState, setIsLoadedState] = useState(false);
  const debounceTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>(
    {},
  );

  // ============================================================================
  // Load Thresholds from API (only async load needed)
  // ============================================================================

  const loadThresholdsFromAPI = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        if (data.thresholds) {
          setThresholds((prev) => ({ ...prev, ...data.thresholds }));
        }
      }
    } catch (err) {
      console.error("Failed to fetch thresholds:", err);
    }
  }, []);

  const refreshSettings = useCallback(async () => {
    // Re-load from localStorage
    setFabOptions(loadInitialFabOptions());
    setDisplayOptions(loadInitialDisplayOptions());
    setIperfSettings(loadInitialIperfSettings());
    await loadThresholdsFromAPI();
  }, [loadThresholdsFromAPI]);

  // Initial load - only fetch thresholds from API
  useEffect(() => {
    let mounted = true;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Initial data fetch pattern
    loadThresholdsFromAPI().finally(() => {
      if (mounted) {
        setIsLoadedState(true);
      }
    });
    return () => {
      mounted = false;
    };
  }, [loadThresholdsFromAPI]);

  // Cleanup timers on unmount
  useEffect(() => {
    const timers = debounceTimers.current;
    return () => {
      Object.values(timers).forEach(clearTimeout);
    };
  }, []);

  // ============================================================================
  // Debounced Save Helper
  // ============================================================================

  const debounceSave = useCallback(
    (
      key: string,
      saveFn: () => void | Promise<void>,
      delay: number = DEBOUNCE_MS,
    ) => {
      if (debounceTimers.current[key]) {
        clearTimeout(debounceTimers.current[key]);
      }

      setStatus((prev) => ({ ...prev, [key]: "saving" as SaveStatus }));

      debounceTimers.current[key] = setTimeout(async () => {
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
    },
    [],
  );

  // ============================================================================
  // Update Methods - update state and trigger debounced save
  // ============================================================================

  const updateFabOptions = useCallback(
    (updates: Partial<FABOptions>) => {
      setFabOptions((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("fab", () => {
          localStorage.setItem(STORAGE_KEYS.FAB_OPTIONS, JSON.stringify(next));
        });
        return next;
      });
    },
    [debounceSave],
  );

  const updateDisplayOptions = useCallback(
    (updates: Partial<DisplayOptions>) => {
      setDisplayOptions((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("display", () => {
          localStorage.setItem(
            STORAGE_KEYS.DISPLAY_OPTIONS,
            JSON.stringify(next),
          );
        });
        return next;
      });
    },
    [debounceSave],
  );

  const updateIperfSettings = useCallback(
    (updates: Partial<IperfSettings>) => {
      setIperfSettings((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("iperf", () => {
          localStorage.setItem(
            STORAGE_KEYS.IPERF_SETTINGS,
            JSON.stringify(next),
          );
        });
        return next;
      });
    },
    [debounceSave],
  );

  const updateThresholds = useCallback(
    (updates: Partial<SettingsThresholds>) => {
      setThresholds((prev) => {
        const next = { ...prev, ...updates };
        debounceSave("thresholds", async () => {
          const response = await fetch(`${API_BASE}/api/settings`, {
            method: "PUT",
            headers: {
              ...getAuthHeaders(),
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ thresholds: next }),
          });
          if (!response.ok) {
            throw new Error("Failed to save thresholds");
          }
        });
        return next;
      });
    },
    [debounceSave],
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

  return (
    <SettingsContext.Provider value={value}>
      {children}
    </SettingsContext.Provider>
  );
}

// ============================================================================
// Hook
// ============================================================================

export function useSettings(): SettingsContextValue {
  const context = useContext(SettingsContext);
  if (!context) {
    throw new Error("useSettings must be used within a SettingsProvider");
  }
  return context;
}

// Optional: Hook that doesn't throw if used outside provider (for gradual migration)
export function useSettingsOptional(): SettingsContextValue | null {
  return useContext(SettingsContext);
}
