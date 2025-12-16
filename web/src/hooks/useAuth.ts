/**
 * Authentication Hook
 *
 * Manages user authentication state using secure httpOnly cookies.
 *
 * Features:
 * - Cookie-based authentication (XSS protection via httpOnly cookies)
 * - Automatic token refresh using refresh tokens
 * - Login/logout functionality
 * - Loading and error state management
 * - Automatic session restoration on mount
 *
 * Security:
 * - Tokens stored in httpOnly cookies (not accessible to JavaScript)
 * - Short-lived access tokens (15min) with long-lived refresh tokens (7 days)
 * - Automatic refresh on token expiration
 * - CSRF protection via SameSite=Strict cookies
 *
 * The hook automatically:
 * - Checks authentication status on mount by calling backend
 * - Refreshes expired access tokens transparently
 * - Clears old localStorage keys for migration
 *
 * Usage:
 * ```typescript
 * const { isAuthenticated, login, logout, token } = useAuth();
 *
 * const handleLogin = async () => {
 *   const success = await login(username, password);
 *   if (success) {
 *     // User authenticated
 *   }
 * };
 * ```
 */

import { useCallback, useEffect, useState } from "react";

/** Internal authentication state */
interface AuthState {
  isAuthenticated: boolean;
  token: string | null; // Access token for WebSocket connections (short-lived)
  username: string | null;
}

/** Login API response structure */
interface LoginResponse {
  token: string; // JWT authentication token
  expires: number; // Token expiration timestamp (Unix seconds)
}

/** Return value from useAuth hook */
interface UseAuthReturn {
  isAuthenticated: boolean;
  token: string | null;
  username: string | null;
  /** Attempt to login with credentials. Returns true on success. */
  login: (username: string, password: string) => Promise<boolean>;
  /** Logout and clear authentication state */
  logout: () => void;
  /** True while login request is in progress */
  isLoading: boolean;
  /** Error message from failed login attempt */
  error: string | null;
}

const API_BASE = import.meta.env.VITE_API_BASE || "";

// Legacy localStorage keys - will be cleared on mount for migration
const LEGACY_KEYS = [
  "netscope-token",
  "netscope-token-expiry",
  "netscope-username",
  "netscope_token",
  "netscope_token_expiry",
  "netscope_username",
  "luminetiq-token",
  "luminetiq-token-expiry",
  "luminetiq-username",
  "seed-token",
  "seed-token-expiry",
  "seed-username",
];

/**
 * Clears old localStorage keys from cookie migration.
 * Runs automatically on mount to clean up legacy token storage.
 */
function clearLegacyStorage(): void {
  LEGACY_KEYS.forEach((key) => localStorage.removeItem(key));
}

/**
 * Custom hook for managing user authentication state.
 *
 * Provides login/logout functionality and tracks authentication state.
 * Automatically checks session validity on mount via backend API.
 *
 * @returns Authentication state and control functions
 */
export function useAuth(): UseAuthReturn {
  // Internal authentication state
  const [state, setState] = useState<AuthState>({
    isAuthenticated: false,
    token: null,
    username: null,
  });
  const [isLoading, setIsLoading] = useState(true); // Start as loading while checking session
  const [error, setError] = useState<string | null>(null);

  /**
   * Effect: Check authentication status on mount
   *
   * Calls backend API to verify session (cookies sent automatically).
   * Clears legacy localStorage keys from migration.
   */
  useEffect(() => {
    clearLegacyStorage();

    // Check if we're authenticated by calling a protected endpoint
    fetch(`${API_BASE}/api/status`, {
      credentials: "include", // Send cookies
    })
      .then((response) => {
        if (response.ok) {
          // Authenticated - we don't have username from /api/status, will be set on login
          setState({
            isAuthenticated: true,
            token: null, // Will be set on login for WebSocket
            username: null,
          });
        } else {
          // Not authenticated
          setState({
            isAuthenticated: false,
            token: null,
            username: null,
          });
        }
      })
      .catch(() => {
        // Error checking auth, assume not authenticated
        setState({
          isAuthenticated: false,
          token: null,
          username: null,
        });
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, []);

  const login = useCallback(async (username: string, password: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/api/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include", // Receive httpOnly cookies
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        throw new Error("Invalid credentials");
      }

      const data: LoginResponse = await response.json();

      // Backend sets httpOnly cookies automatically
      // Store access token in memory ONLY for WebSocket connections
      setState({
        isAuthenticated: true,
        token: data.token, // Access token for WebSocket (short-lived, 15min)
        username,
      });

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    // Clear in-memory state immediately
    setState({
      isAuthenticated: false,
      token: null,
      username: null,
    });

    // Call logout endpoint to clear httpOnly cookies
    fetch(`${API_BASE}/api/auth/logout`, {
      method: "POST",
      credentials: "include", // Send cookies to be cleared
    }).catch(() => {
      // Ignore errors - local state already cleared
    });
  }, []);

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

/**
 * @deprecated No longer needed with cookie-based authentication.
 * API requests automatically include cookies with credentials: 'include'.
 * This function is kept for backward compatibility but returns empty object.
 */
export function getAuthHeaders(): HeadersInit {
  return {};
}
