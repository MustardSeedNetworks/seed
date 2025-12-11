import { useCallback, useEffect, useState } from "react";

interface AuthState {
  isAuthenticated: boolean;
  token: string | null;
  username: string | null;
}

interface LoginResponse {
  token: string;
  expires: number;
}

interface UseAuthReturn {
  isAuthenticated: boolean;
  token: string | null;
  username: string | null;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => void;
  isLoading: boolean;
  error: string | null;
}

// New standardized localStorage keys (hyphen-separated)
const TOKEN_KEY = "netscope-token";
const TOKEN_EXPIRY_KEY = "netscope-token-expiry";
const USERNAME_KEY = "netscope-username";
const API_BASE = import.meta.env.VITE_API_BASE || "";

// Legacy keys for migration (underscore-separated)
const LEGACY_TOKEN_KEY = "netscope_token";
const LEGACY_TOKEN_EXPIRY_KEY = "netscope_token_expiry";
const LEGACY_USERNAME_KEY = "netscope_username";

// One-time migration from old underscore keys to new hyphen keys
function migrateStorageKeys(): void {
  const legacyToken = localStorage.getItem(LEGACY_TOKEN_KEY);
  if (legacyToken) {
    // Migrate old keys to new format
    localStorage.setItem(TOKEN_KEY, legacyToken);
    const legacyExpiry = localStorage.getItem(LEGACY_TOKEN_EXPIRY_KEY);
    if (legacyExpiry) {
      localStorage.setItem(TOKEN_EXPIRY_KEY, legacyExpiry);
    }
    const legacyUsername = localStorage.getItem(LEGACY_USERNAME_KEY);
    if (legacyUsername) {
      localStorage.setItem(USERNAME_KEY, legacyUsername);
    }
    // Remove legacy keys
    localStorage.removeItem(LEGACY_TOKEN_KEY);
    localStorage.removeItem(LEGACY_TOKEN_EXPIRY_KEY);
    localStorage.removeItem(LEGACY_USERNAME_KEY);
  }
}

// Run migration on module load
migrateStorageKeys();

// Check if a stored token has expired
function isTokenExpired(): boolean {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY);
  if (!expiry) {
    return true; // No expiry stored means we can't verify, treat as expired
  }
  // Add 30 second buffer to avoid edge cases where token expires during request
  const expiryTime = parseInt(expiry, 10) * 1000; // Convert seconds to ms
  return Date.now() >= expiryTime - 30000;
}

export function useAuth(): UseAuthReturn {
  const [state, setState] = useState<AuthState>({
    isAuthenticated: false,
    token: null,
    username: null,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Check for existing token on mount and validate expiry
  useEffect(() => {
    const token = localStorage.getItem(TOKEN_KEY);
    const username = localStorage.getItem(USERNAME_KEY);

    if (token) {
      // Check if token has expired
      if (isTokenExpired()) {
        // Token expired, clear storage and stay logged out
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(TOKEN_EXPIRY_KEY);
        localStorage.removeItem(USERNAME_KEY);
        return;
      }

      setState({
        isAuthenticated: true,
        token,
        username,
      });
    }
  }, []);

  const login = useCallback(
    async (username: string, password: string): Promise<boolean> => {
      setIsLoading(true);
      setError(null);

      try {
        const response = await fetch(`${API_BASE}/api/auth/login`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ username, password }),
        });

        if (!response.ok) {
          throw new Error("Invalid credentials");
        }

        const data: LoginResponse = await response.json();

        localStorage.setItem(TOKEN_KEY, data.token);
        localStorage.setItem(TOKEN_EXPIRY_KEY, String(data.expires));
        localStorage.setItem(USERNAME_KEY, username);

        setState({
          isAuthenticated: true,
          token: data.token,
          username,
        });

        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : "Login failed");
        return false;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(TOKEN_EXPIRY_KEY);
    localStorage.removeItem(USERNAME_KEY);

    setState({
      isAuthenticated: false,
      token: null,
      username: null,
    });

    // Call logout endpoint (fire and forget)
    fetch(`${API_BASE}/api/auth/logout`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${state.token}`,
      },
    }).catch(() => {
      // Ignore errors
    });
  }, [state.token]);

  return {
    isAuthenticated: state.isAuthenticated,
    token: state.token,
    username: state.username,
    login,
    logout,
    isLoading,
    error,
  };
}

// Helper to get auth headers for API requests
export function getAuthHeaders(): HeadersInit {
  const token = localStorage.getItem(TOKEN_KEY);
  if (token) {
    return {
      Authorization: `Bearer ${token}`,
    };
  }
  return {};
}
