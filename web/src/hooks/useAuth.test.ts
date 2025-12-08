import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useAuth, getAuthHeaders } from "./useAuth";

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

describe("useAuth", () => {
  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();
  });

  it("starts with unauthenticated state", () => {
    const { result } = renderHook(() => useAuth());

    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.token).toBeNull();
    expect(result.current.username).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("restores auth state from localStorage on mount", async () => {
    mockLocalStorage.setItem("netscope_token", "existing-token");
    mockLocalStorage.setItem("netscope_username", "testuser");

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
      expect(result.current.token).toBe("existing-token");
      expect(result.current.username).toBe("testuser");
    });
  });

  it("login sets authenticated state on success", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ token: "new-token", expires: 3600 }),
    });

    const { result } = renderHook(() => useAuth());

    let loginResult: boolean;
    await act(async () => {
      loginResult = await result.current.login("admin", "password");
    });

    expect(loginResult!).toBe(true);
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.token).toBe("new-token");
    expect(result.current.username).toBe("admin");
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
      "netscope_token",
      "new-token",
    );
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
      "netscope_username",
      "admin",
    );
  });

  it("login sets error on failure", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    });

    const { result } = renderHook(() => useAuth());

    let loginResult: boolean;
    await act(async () => {
      loginResult = await result.current.login("admin", "wrong-password");
    });

    expect(loginResult!).toBe(false);
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.error).toBe("Invalid credentials");
  });

  it("login handles network error", async () => {
    mockFetch.mockRejectedValueOnce(new Error("Network error"));

    const { result } = renderHook(() => useAuth());

    let loginResult: boolean;
    await act(async () => {
      loginResult = await result.current.login("admin", "password");
    });

    expect(loginResult!).toBe(false);
    expect(result.current.error).toBe("Network error");
  });

  it("login sets isLoading during request", async () => {
    let resolvePromise: (value: unknown) => void;
    const promise = new Promise((resolve) => {
      resolvePromise = resolve;
    });

    mockFetch.mockReturnValueOnce(promise);

    const { result } = renderHook(() => useAuth());

    act(() => {
      result.current.login("admin", "password");
    });

    expect(result.current.isLoading).toBe(true);

    await act(async () => {
      resolvePromise!({
        ok: true,
        json: () => Promise.resolve({ token: "token", expires: 3600 }),
      });
    });

    expect(result.current.isLoading).toBe(false);
  });

  it("logout clears auth state", async () => {
    mockLocalStorage.setItem("netscope_token", "token");
    mockLocalStorage.setItem("netscope_username", "user");

    // Mock the logout fetch
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { result } = renderHook(() => useAuth());

    // Wait for initial state restoration
    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    act(() => {
      result.current.logout();
    });

    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.token).toBeNull();
    expect(result.current.username).toBeNull();
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("netscope_token");
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith(
      "netscope_username",
    );
  });
});

describe("getAuthHeaders", () => {
  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();
  });

  it("returns empty object when no token", () => {
    const headers = getAuthHeaders();
    expect(headers).toEqual({});
  });

  it("returns Authorization header when token exists", () => {
    mockLocalStorage.setItem("netscope_token", "test-token");

    const headers = getAuthHeaders();
    expect(headers).toEqual({
      Authorization: "Bearer test-token",
    });
  });
});
