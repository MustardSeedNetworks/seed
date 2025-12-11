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
    const futureExpiry = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
    mockLocalStorage.setItem("netscope-token", "existing-token");
    mockLocalStorage.setItem("netscope-token-expiry", String(futureExpiry));
    mockLocalStorage.setItem("netscope-username", "testuser");

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
      "netscope-token",
      "new-token",
    );
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
      "netscope-username",
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
    const futureExpiry = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
    mockLocalStorage.setItem("netscope-token", "token");
    mockLocalStorage.setItem("netscope-token-expiry", String(futureExpiry));
    mockLocalStorage.setItem("netscope-username", "user");

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
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("netscope-token");
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith(
      "netscope-username",
    );
  });

  it("login stores token expiry in localStorage", async () => {
    const futureExpiry = Math.floor(Date.now() / 1000) + 3600;
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve({ token: "new-token", expires: futureExpiry }),
    });

    const { result } = renderHook(() => useAuth());

    await act(async () => {
      await result.current.login("admin", "password");
    });

    expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
      "netscope-token-expiry",
      String(futureExpiry),
    );
  });

  it("clears expired token on mount", async () => {
    // Set token with expired timestamp (1 hour ago)
    const pastExpiry = Math.floor(Date.now() / 1000) - 3600;
    mockLocalStorage.setItem("netscope-token", "expired-token");
    mockLocalStorage.setItem("netscope-token-expiry", String(pastExpiry));
    mockLocalStorage.setItem("netscope-username", "testuser");

    const { result } = renderHook(() => useAuth());

    // Wait for effect to run
    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(false);
    });

    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("netscope-token");
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith(
      "netscope-token-expiry",
    );
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith(
      "netscope-username",
    );
  });

  it("keeps valid token on mount when not expired", async () => {
    // Set token with future expiry (1 hour from now)
    const futureExpiry = Math.floor(Date.now() / 1000) + 3600;
    mockLocalStorage.setItem("netscope-token", "valid-token");
    mockLocalStorage.setItem("netscope-token-expiry", String(futureExpiry));
    mockLocalStorage.setItem("netscope-username", "testuser");

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
      expect(result.current.token).toBe("valid-token");
      expect(result.current.username).toBe("testuser");
    });
  });

  it("treats token as expired when no expiry is stored", async () => {
    // Set token without expiry
    mockLocalStorage.setItem("netscope-token", "token-no-expiry");
    mockLocalStorage.setItem("netscope-username", "testuser");
    // Don't set netscope-token-expiry

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(false);
    });

    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("netscope-token");
  });

  it("logout clears token expiry from localStorage", async () => {
    const futureExpiry = Math.floor(Date.now() / 1000) + 3600;
    mockLocalStorage.setItem("netscope-token", "token");
    mockLocalStorage.setItem("netscope-token-expiry", String(futureExpiry));
    mockLocalStorage.setItem("netscope-username", "user");

    mockFetch.mockResolvedValueOnce({ ok: true });

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    act(() => {
      result.current.logout();
    });

    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith(
      "netscope-token-expiry",
    );
  });

  it("logout calls backend logout endpoint with token", async () => {
    const futureExpiry = Math.floor(Date.now() / 1000) + 3600;
    mockLocalStorage.setItem("netscope-token", "my-token");
    mockLocalStorage.setItem("netscope-token-expiry", String(futureExpiry));
    mockLocalStorage.setItem("netscope-username", "user");

    mockFetch.mockResolvedValueOnce({ ok: true });

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    act(() => {
      result.current.logout();
    });

    expect(mockFetch).toHaveBeenCalledWith(
      "/api/auth/logout",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer my-token",
        }),
      }),
    );
  });

  it("logout handles backend error gracefully", async () => {
    const futureExpiry = Math.floor(Date.now() / 1000) + 3600;
    mockLocalStorage.setItem("netscope-token", "token");
    mockLocalStorage.setItem("netscope-token-expiry", String(futureExpiry));
    mockLocalStorage.setItem("netscope-username", "user");

    // Make logout endpoint fail
    mockFetch.mockRejectedValueOnce(new Error("Network error"));

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    // Should not throw
    act(() => {
      result.current.logout();
    });

    // State should still be cleared
    expect(result.current.isAuthenticated).toBe(false);
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
    mockLocalStorage.setItem("netscope-token", "test-token");

    const headers = getAuthHeaders();
    expect(headers).toEqual({
      Authorization: "Bearer test-token",
    });
  });
});
