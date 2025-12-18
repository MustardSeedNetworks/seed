/**
 * SettingsContext.test.tsx - Settings Context Tests
 *
 * Purpose: Comprehensive test suite for SettingsContext and related hooks (useSettings, useSettingsOptional).
 * Tests context provider setup, settings loading/saving, state updates, API integration, and error handling.
 *
 * Key Test Areas:
 * - Provider initialization: default settings loading and context creation
 * - localStorage persistence: settings saved/loaded from browser storage
 * - API integration: fetch calls to /api/config endpoint
 * - useSettings hook: proper context value retrieval
 * - useSettingsOptional hook: graceful handling without provider
 * - State updates: card settings, display options, iperf settings, thresholds
 * - Error handling: API failure and retry logic
 * - Async operations: proper loading/error state transitions
 * - Cleanup: proper unmounting and resource cleanup
 *
 * Test Framework: Vitest with React Testing Library hooks rendering
 * Mocks: localStorage, fetch API calls
 *
 * Usage:
 * ```bash
 * npm test -- SettingsContext.test.tsx
 * ```
 *
 * Dependencies: vitest, @testing-library/react
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { ReactNode } from "react";
import { SettingsProvider } from "./SettingsContext";
import { useSettings, useSettingsOptional } from "./useSettings";
import {
  DEFAULT_CARD_SETTINGS,
  DEFAULT_DISPLAY_OPTIONS,
  DEFAULT_IPERF_SETTINGS,
  DEFAULT_THRESHOLDS,
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
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

      expect(() => {
        renderHook(() => useSettings());
      }).toThrow("useSettings must be used within a SettingsProvider");

      consoleSpy.mockRestore();
    });

    it("provides default values on initial render", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      expect(result.current.cardSettings).toEqual(DEFAULT_CARD_SETTINGS);
      expect(result.current.displayOptions).toEqual(DEFAULT_DISPLAY_OPTIONS);
      expect(result.current.iperfSettings).toEqual(DEFAULT_IPERF_SETTINGS);
      expect(result.current.status.cards).toBe("idle");
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
      expect(result.current?.cardSettings).toEqual(DEFAULT_CARD_SETTINGS);
    });
  });

  describe("API loading", () => {
    it("loads card settings from API", async () => {
      const savedCardSettings = {
        ...DEFAULT_CARD_SETTINGS,
        link: { enabled: true, autoRunOnLink: false },
        dns: { enabled: true, autoRunOnLink: false },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            thresholds: DEFAULT_THRESHOLDS,
            cardSettings: savedCardSettings,
          }),
      });

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.cardSettings.link.autoRunOnLink).toBe(false);
        expect(result.current.cardSettings.dns.autoRunOnLink).toBe(false);
      });
    });

    it("loads display options from API", async () => {
      const savedDisplayOptions = { showPublicIP: false };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            thresholds: DEFAULT_THRESHOLDS,
            displayOptions: savedDisplayOptions,
          }),
      });

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.displayOptions.showPublicIP).toBe(false);
      });
    });

    it("loads iperf settings from API", async () => {
      const savedIperfSettings = {
        ...DEFAULT_IPERF_SETTINGS,
        server: "192.168.1.100",
        port: 5202,
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            thresholds: DEFAULT_THRESHOLDS,
            iperf: savedIperfSettings,
          }),
      });

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.iperfSettings.server).toBe("192.168.1.100");
        expect(result.current.iperfSettings.port).toBe(5202);
      });
    });

    it("handles API error gracefully and uses defaults", async () => {
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      mockFetch.mockRejectedValue(new Error("Network error"));

      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoaded).toBe(true);
      });

      // Should fall back to defaults
      expect(result.current.cardSettings).toEqual(DEFAULT_CARD_SETTINGS);
      consoleSpy.mockRestore();
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
            credentials: "include",
            method: "GET",
          })
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
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
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
    it("updateCardSettings updates state immediately", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateCardSettings({ link: { enabled: true, autoRunOnLink: false } });
      });

      expect(result.current.cardSettings.link.autoRunOnLink).toBe(false);
      // Other options should remain unchanged
      expect(result.current.cardSettings.dns.autoRunOnLink).toBe(
        DEFAULT_CARD_SETTINGS.dns.autoRunOnLink
      );
    });

    it("updateCardSettings sets status to saving", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateCardSettings({ link: { enabled: true, autoRunOnLink: false } });
      });

      expect(result.current.status.cards).toBe("saving");
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
    it("reloads settings from API", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.isLoaded).toBe(true);
      });

      // Set up new response for refresh
      const newCardSettings = {
        ...DEFAULT_CARD_SETTINGS,
        link: { enabled: true, autoRunOnLink: false },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            thresholds: DEFAULT_THRESHOLDS,
            cardSettings: newCardSettings,
          }),
      });

      // Refresh
      await act(async () => {
        await result.current.refreshSettings();
      });

      expect(result.current.cardSettings.link.autoRunOnLink).toBe(false);
    });

    it("fetches all settings from API again", async () => {
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

      expect(mockFetch).toHaveBeenCalledWith("/api/settings", expect.any(Object));
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
      })
    );
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  describe("debounced saves", () => {
    it("saves card settings to backend API after debounce", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      // Clear initial fetch calls
      mockFetch.mockClear();

      act(() => {
        result.current.updateCardSettings({ link: { enabled: true, autoRunOnLink: false } });
      });

      // Not saved yet (no PUT calls)
      const putCalls1 = mockFetch.mock.calls.filter((call) => call[1]?.method === "PUT");
      expect(putCalls1.length).toBe(0);

      // After debounce (DEBOUNCE_MS = 1000ms, so advance 1100ms to be safe)
      await act(async () => {
        vi.advanceTimersByTime(1100);
      });

      const putCalls2 = mockFetch.mock.calls.filter((call) => call[1]?.method === "PUT");
      expect(putCalls2.length).toBe(1);
      expect(putCalls2[0][1].body).toContain('"autoRunOnLink":false');
    });

    it("saves display options to backend API after debounce", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      mockFetch.mockClear();

      act(() => {
        result.current.updateDisplayOptions({ showPublicIP: false });
      });

      await act(async () => {
        vi.advanceTimersByTime(1100);
      });

      const putCalls = mockFetch.mock.calls.filter((call) => call[1]?.method === "PUT");
      expect(putCalls.length).toBe(1);
      expect(putCalls[0][1].body).toContain('"showPublicIP":false');
    });

    it("saves iperf settings to backend API after debounce", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      mockFetch.mockClear();

      act(() => {
        result.current.updateIperfSettings({ server: "10.0.0.1" });
      });

      await act(async () => {
        vi.advanceTimersByTime(1100);
      });

      const putCalls = mockFetch.mock.calls.filter((call) => call[1]?.method === "PUT");
      expect(putCalls.length).toBe(1);
      expect(putCalls[0][1].body).toContain('"server":"10.0.0.1"');
    });

    it("debounces multiple rapid card settings updates", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      mockFetch.mockClear();

      act(() => {
        result.current.updateCardSettings({ link: { enabled: true, autoRunOnLink: false } });
      });

      await act(async () => {
        vi.advanceTimersByTime(400);
      });

      act(() => {
        result.current.updateCardSettings({ dns: { enabled: true, autoRunOnLink: false } });
      });

      await act(async () => {
        vi.advanceTimersByTime(1100);
      });

      // Only the final state should be saved (once)
      const putCalls = mockFetch.mock.calls.filter((call) => call[1]?.method === "PUT");
      expect(putCalls.length).toBe(1);
      // Both link and dns should have autoRunOnLink: false
      const body = putCalls[0][1].body;
      expect(body).toContain('"link"');
      expect(body).toContain('"dns"');
    });

    it("sets status to saved after debounce completes", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateCardSettings({ link: { enabled: true, autoRunOnLink: false } });
      });

      expect(result.current.status.cards).toBe("saving");

      await act(async () => {
        vi.advanceTimersByTime(1100);
      });

      expect(result.current.status.cards).toBe("saved");
    });

    it("resets status to idle after delay", async () => {
      const { result } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateCardSettings({ link: { enabled: true, autoRunOnLink: false } });
      });

      await act(async () => {
        vi.advanceTimersByTime(1100);
      });

      expect(result.current.status.cards).toBe("saved");

      await act(async () => {
        vi.advanceTimersByTime(2100);
      });

      expect(result.current.status.cards).toBe("idle");
    });
  });

  describe("cleanup", () => {
    it("cleans up debounce timers on unmount", async () => {
      const { result, unmount } = renderHook(() => useSettings(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateCardSettings({ link: { enabled: true, autoRunOnLink: false } });
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
