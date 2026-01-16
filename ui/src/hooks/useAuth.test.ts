// biome-ignore-all lint/nursery/useAwaitThenable: act() returns Promise when given async callback
// biome-ignore-all lint/suspicious/useAwait: async callbacks in act() are required for React Testing Library even without await
/**
 * useAuth.test.ts - Authentication Hook Tests
 *
 * Purpose: Test suite for the useAuth hook with cookie-based authentication.
 * Tests authentication state management, cookie handling, and session management.
 *
 * Key Test Areas:
 * - Cookie-based authentication (httpOnly cookies set by backend)
 * - Session restoration via API call on mount
 * - Login/logout flow with cookies
 * - Token returned for WebSocket connections only
 * - Legacy localStorage cleanup
 *
 * Test Framework: Vitest with React Testing Library hooks
 * Mocks: localStorage, fetch API
 *
 * Usage:
 * ```bash
 * npm test -- useAuth.test.ts
 * ```
 *
 * Dependencies: vitest, @testing-library/react
 */

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuth } from "./useAuth";

// Mock localStorage
interface MockLocalStorage {
  getItem: ReturnType<typeof vi.fn>;
  setItem: ReturnType<typeof vi.fn>;
  removeItem: ReturnType<typeof vi.fn>;
  clear: () => void;
}

const mockLocalStorage: MockLocalStorage = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: (): void => {
      store = {};
    },
  };
})();

Object.defineProperty(window, "localStorage", {
  value: mockLocalStorage,
});

// Mock fetch
const mockFetch: ReturnType<typeof vi.fn> = vi.fn();
global.fetch = mockFetch;

describe("useAuth", () => {
  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();
  });

  it("starts with loading state and checks auth status", async () => {
    // Mock /api/status to return 401 (not authenticated)
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
    });

    const { result } = renderHook(() => useAuth());

    // Should start loading
    expect(result.current.isLoading).toBe(true);

    // Wait for status check to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.token).toBeNull();
    expect(result.current.username).toBeNull();
  });

  it("restores auth state when backend confirms session", async () => {
    // Mock /api/status to return 200 (authenticated)
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
    });

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.isAuthenticated).toBe(true);
    expect(mockFetch).toHaveBeenCalledWith(
      "/api/status",
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("clears legacy localStorage keys on mount", async () => {
    // Set legacy keys (migrated to httpOnly cookies)
    mockLocalStorage.setItem("seed-token", "old-token");
    mockLocalStorage.setItem("seed-token-expiry", "123456");
    mockLocalStorage.setItem("seed-username", "olduser");

    mockFetch.mockResolvedValueOnce({ ok: false });

    renderHook(() => useAuth());

    await waitFor(() => {
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("seed-token");
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("seed-token-expiry");
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("seed-username");
    });
  });

  it("login sets authenticated state on success", async () => {
    // Mock initial status check
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { result } = renderHook(() => useAuth());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    // Mock successful login
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ token: "access-token", expires: 3600 }),
    });

    let loginResult: boolean;
    await act(async () => {
      loginResult = await result.current.login("admin", "password");
    });

    expect(loginResult).toBe(true);
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.token).toBe("access-token"); // For WebSocket
    expect(result.current.username).toBe("admin");
    expect(mockFetch).toHaveBeenLastCalledWith(
      "/api/auth/login",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: JSON.stringify({ username: "admin", password: "password" }),
      }),
    );
  });

  it("login sets error on failure", async () => {
    mockFetch.mockResolvedValueOnce({ ok: false }); // Initial status check

    const { result } = renderHook(() => useAuth());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
    });

    let loginResult: boolean;
    await act(async () => {
      loginResult = await result.current.login("admin", "wrongpassword");
    });

    expect(loginResult).toBe(false);
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.error).toBeTruthy();
  });

  it("login handles network error", async () => {
    mockFetch.mockResolvedValueOnce({ ok: false }); // Initial status check

    const { result } = renderHook(() => useAuth());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    mockFetch.mockRejectedValueOnce(new Error("Network error"));

    let loginResult: boolean;
    await act(async () => {
      loginResult = await result.current.login("admin", "password");
    });

    expect(loginResult).toBe(false);
    expect(result.current.error).toBe("Network error");
  });

  it("login sets isLoading during request", async () => {
    mockFetch.mockResolvedValueOnce({ ok: false }); // Initial status check

    const { result } = renderHook(() => useAuth());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    // Mock login that takes time
    let resolveLogin:
      | ((value: { ok: boolean; json: () => Promise<{ token: string; expires: number }> }) => void)
      | undefined;
    const loginPromise = new Promise<{
      ok: boolean;
      json: () => Promise<{ token: string; expires: number }>;
    }>((resolve) => {
      resolveLogin = resolve;
    });
    mockFetch.mockReturnValueOnce(loginPromise);

    act(() => {
      result.current.login("admin", "password");
    });

    // Should be loading
    expect(result.current.isLoading).toBe(true);

    // Resolve login
    if (resolveLogin) {
      resolveLogin({
        ok: true,
        json: () => Promise.resolve({ token: "token", expires: 3600 }),
      });
    }

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
  });

  it("logout clears auth state", async () => {
    // Mock initial authenticated state
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    // Mock logout endpoint
    mockFetch.mockResolvedValueOnce({ ok: true });

    await act(async () => {
      result.current.logout();
    });

    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.token).toBeNull();
    expect(result.current.username).toBeNull();
  });

  it("logout calls backend endpoint", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true }); // Initial status

    const { result } = renderHook(() => useAuth());

    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    mockFetch.mockResolvedValueOnce({ ok: true });

    await act(async () => {
      result.current.logout();
    });

    // Should have called logout endpoint with credentials
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        "/api/auth/logout",
        expect.objectContaining({
          method: "POST",
          credentials: "include",
        }),
      );
    });
  });

  it("logout handles backend error gracefully", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { result } = renderHook(() => useAuth());

    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    // Mock logout failure
    mockFetch.mockRejectedValueOnce(new Error("Network error"));

    await act(async () => {
      result.current.logout();
    });

    // Should still clear local state
    expect(result.current.isAuthenticated).toBe(false);
  });
});
