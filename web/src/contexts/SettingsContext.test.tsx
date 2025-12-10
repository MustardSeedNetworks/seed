import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { ReactNode } from "react";
import {
  SettingsProvider,
  useSettings,
  useSettingsOptional,
} from "./SettingsContext";
import {
  DEFAULT_FAB_OPTIONS,
  DEFAULT_DISPLAY_OPTIONS,
  DEFAULT_IPERF_SETTINGS,
  DEFAULT_THRESHOLDS,
  STORAGE_KEYS,
} from "../types/settings";

// Mock localStorage
const mockLocalStorage = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(window, "localStorage", {
  value: mockLocalStorage,
});

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Mock getAuthHeaders
vi.mock("../hooks/useAuth", () => ({
  getAuthHeaders: () => ({ Authorization: "Bearer test-token" }),
}));

// Helper wrapper component
function createWrapper() {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <SettingsProvider>{children}</SettingsProvider>;
  };
}

describe("SettingsContext", () => {
  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();
    // Default successful API response
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ thresholds: DEFAULT_THRESHOLDS }),
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("useSettings hook", () => {
    it("throws error when used outside provider", () => {
      // Suppress console.error for this test
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      expect(() => {
        renderHook(() => useSettings());
      }).toThrow("useSettings must be used within a SettingsProvider");

      consoleSpy.mockRestore();
    });

    it("provides default values on initial render", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      expect(result.current.fabOptions).toEqual(DEFAULT_FAB_OPTIONS);
      expect(result.current.displayOptions).toEqual(DEFAULT_DISPLAY_OPTIONS);
      expect(result.current.iperfSettings).toEqual(DEFAULT_IPERF_SETTINGS);
      expect(result.current.status.fab).toBe("idle");
      expect(result.current.status.display).toBe("idle");
      expect(result.current.status.iperf).toBe("idle");
      expect(result.current.status.thresholds).toBe("idle");
    });

    it("sets isLoaded to true after initial API fetch", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoaded).toBe(true);
      });
    });
  });

  describe("useSettingsOptional hook", () => {
    it("returns null when used outside provider", () => {
      const { result } = renderHook(() => useSettingsOptional());
      expect(result.current).toBeNull();
    });

    it("returns context value when used inside provider", () => {
      const { result } = renderHook(() => useSettingsOptional(), {
        wrapper: createWrapper(),
      });
      expect(result.current).not.toBeNull();
      expect(result.current?.fabOptions).toEqual(DEFAULT_FAB_OPTIONS);
    });
  });

  describe("localStorage loading", () => {
    it("loads FAB options from localStorage", async () => {
      const savedFabOptions = {
        ...DEFAULT_FAB_OPTIONS,
        runLink: false,
        runDNS: false,
      };
      mockLocalStorage.setItem(
        STORAGE_KEYS.FAB_OPTIONS,
        JSON.stringify(savedFabOptions),
      );

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      expect(result.current.fabOptions.runLink).toBe(false);
      expect(result.current.fabOptions.runDNS).toBe(false);
    });

    it("loads display options from localStorage", async () => {
      const savedDisplayOptions = { showPublicIP: false };
      mockLocalStorage.setItem(
        STORAGE_KEYS.DISPLAY_OPTIONS,
        JSON.stringify(savedDisplayOptions),
      );

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      expect(result.current.displayOptions.showPublicIP).toBe(false);
    });

    it("loads iperf settings from localStorage", async () => {
      const savedIperfSettings = {
        ...DEFAULT_IPERF_SETTINGS,
        server: "192.168.1.100",
        port: 5202,
      };
      mockLocalStorage.setItem(
        STORAGE_KEYS.IPERF_SETTINGS,
        JSON.stringify(savedIperfSettings),
      );

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      expect(result.current.iperfSettings.server).toBe("192.168.1.100");
      expect(result.current.iperfSettings.port).toBe(5202);
    });

    it("handles invalid JSON in localStorage gracefully", () => {
      mockLocalStorage.setItem(STORAGE_KEYS.FAB_OPTIONS, "not-valid-json");

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      // Should fall back to defaults
      expect(result.current.fabOptions).toEqual(DEFAULT_FAB_OPTIONS);
    });
  });

  describe("thresholds API loading", () => {
    it("fetches thresholds from API on mount", async () => {
      renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          "/api/settings",
          expect.objectContaining({
            headers: { Authorization: "Bearer test-token" },
          }),
        );
      });
    });

    it("loads thresholds from API response", async () => {
      const customThresholds = {
        ...DEFAULT_THRESHOLDS,
        dns: { good: 30, warning: 60 },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ thresholds: customThresholds }),
      });

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.thresholds.dns.good).toBe(30);
        expect(result.current.thresholds.dns.warning).toBe(60);
      });
    });

    it("handles API error gracefully", async () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      mockFetch.mockRejectedValue(new Error("Network error"));

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoaded).toBe(true);
      });

      // Should keep defaults on error
      expect(result.current.thresholds).toEqual(DEFAULT_THRESHOLDS);
      consoleSpy.mockRestore();
    });
  });

  describe("update methods", () => {
    it("updateFabOptions updates state immediately", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFabOptions({ runLink: false });
      });

      expect(result.current.fabOptions.runLink).toBe(false);
      // Other options should remain unchanged
      expect(result.current.fabOptions.runDNS).toBe(DEFAULT_FAB_OPTIONS.runDNS);
    });

    it("updateFabOptions sets status to saving", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFabOptions({ runLink: false });
      });

      expect(result.current.status.fab).toBe("saving");
    });

    it("updateDisplayOptions updates state", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateDisplayOptions({ showPublicIP: false });
      });

      expect(result.current.displayOptions.showPublicIP).toBe(false);
    });

    it("updateIperfSettings updates state", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateIperfSettings({
          server: "10.0.0.1",
          port: 5203,
        });
      });

      expect(result.current.iperfSettings.server).toBe("10.0.0.1");
      expect(result.current.iperfSettings.port).toBe(5203);
    });

    it("updateThresholds updates state", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateThresholds({
          dns: { good: 25, warning: 75 },
        });
      });

      expect(result.current.thresholds.dns.good).toBe(25);
      expect(result.current.thresholds.dns.warning).toBe(75);
    });
  });

  describe("refreshSettings", () => {
    it("reloads settings from localStorage", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.isLoaded).toBe(true);
      });

      // Modify localStorage
      const newFabOptions = {
        ...DEFAULT_FAB_OPTIONS,
        runLink: false,
      };
      mockLocalStorage.setItem(
        STORAGE_KEYS.FAB_OPTIONS,
        JSON.stringify(newFabOptions),
      );

      // Refresh
      await act(async () => {
        await result.current.refreshSettings();
      });

      expect(result.current.fabOptions.runLink).toBe(false);
    });

    it("fetches thresholds from API again", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoaded).toBe(true);
      });

      mockFetch.mockClear();
      mockFetch.mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            thresholds: {
              ...DEFAULT_THRESHOLDS,
              dns: { good: 10, warning: 20 },
            },
          }),
      });

      await act(async () => {
        await result.current.refreshSettings();
      });

      expect(mockFetch).toHaveBeenCalledWith(
        "/api/settings",
        expect.any(Object),
      );
    });
  });
});

// Tests that require fake timers for debounce testing
describe("SettingsContext with fake timers", () => {
  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();
    vi.useFakeTimers();
    // Default successful API response
    mockFetch.mockImplementation(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ thresholds: DEFAULT_THRESHOLDS }),
      }),
    );
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  describe("debounced saves", () => {
    it("saves FAB options to localStorage after debounce", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFabOptions({ runLink: false });
      });

      // Not saved yet
      const calls1 = mockLocalStorage.setItem.mock.calls.filter(
        ([key]: [string]) => key === STORAGE_KEYS.FAB_OPTIONS,
      );
      expect(calls1.length).toBe(0);

      // After debounce
      await act(async () => {
        vi.advanceTimersByTime(900);
      });

      const calls2 = mockLocalStorage.setItem.mock.calls.filter(
        ([key]: [string]) => key === STORAGE_KEYS.FAB_OPTIONS,
      );
      expect(calls2.length).toBe(1);
      expect(calls2[0][1]).toContain('"runLink":false');
    });

    it("saves display options to localStorage after debounce", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateDisplayOptions({ showPublicIP: false });
      });

      await act(async () => {
        vi.advanceTimersByTime(900);
      });

      const calls = mockLocalStorage.setItem.mock.calls.filter(
        ([key]: [string]) => key === STORAGE_KEYS.DISPLAY_OPTIONS,
      );
      expect(calls.length).toBe(1);
      expect(calls[0][1]).toBe('{"showPublicIP":false}');
    });

    it("saves iperf settings to localStorage after debounce", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateIperfSettings({ server: "10.0.0.1" });
      });

      await act(async () => {
        vi.advanceTimersByTime(900);
      });

      const calls = mockLocalStorage.setItem.mock.calls.filter(
        ([key]: [string]) => key === STORAGE_KEYS.IPERF_SETTINGS,
      );
      expect(calls.length).toBe(1);
      expect(calls[0][1]).toContain('"server":"10.0.0.1"');
    });

    it("debounces multiple rapid FAB updates", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFabOptions({ runLink: false });
      });

      await act(async () => {
        vi.advanceTimersByTime(400);
      });

      act(() => {
        result.current.updateFabOptions({ runDNS: false });
      });

      await act(async () => {
        vi.advanceTimersByTime(900);
      });

      // Only the final state should be saved (once)
      const calls = mockLocalStorage.setItem.mock.calls.filter(
        ([key]: [string]) => key === STORAGE_KEYS.FAB_OPTIONS,
      );
      expect(calls.length).toBe(1);
      expect(calls[0][1]).toContain('"runLink":false');
      expect(calls[0][1]).toContain('"runDNS":false');
    });

    it("sets status to saved after debounce completes", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFabOptions({ runLink: false });
      });

      expect(result.current.status.fab).toBe("saving");

      await act(async () => {
        vi.advanceTimersByTime(900);
      });

      expect(result.current.status.fab).toBe("saved");
    });

    it("resets status to idle after delay", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFabOptions({ runLink: false });
      });

      await act(async () => {
        vi.advanceTimersByTime(900);
      });

      expect(result.current.status.fab).toBe("saved");

      await act(async () => {
        vi.advanceTimersByTime(2100);
      });

      expect(result.current.status.fab).toBe("idle");
    });
  });

  describe("cleanup", () => {
    it("cleans up debounce timers on unmount", async () => {
      const { result, unmount } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFabOptions({ runLink: false });
      });

      // Unmount before debounce completes
      unmount();

      // Should not throw or cause issues
      await act(async () => {
        vi.advanceTimersByTime(1000);
      });
    });
  });
});
