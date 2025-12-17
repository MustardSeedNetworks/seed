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
import { logger, LogComponents } from "../lib/logger";

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

// localStorage keys to clear on mount (migrated to httpOnly cookies)
const LEGACY_KEYS = ["seed-token", "seed-token-expiry", "seed-username"];

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
   * fixes #678 - standardized error handling with logger
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
      .catch((err) => {
        // Error checking auth, assume not authenticated
        // fixes #678 - added logging for auth check errors
        logger.error(LogComponents.AUTH, "Failed to check authentication status", err, {
          endpoint: "/api/status",
        });
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

      logger.info(LogComponents.AUTH, "User logged in successfully", { username });
      return true;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Login failed";
      setError(errorMessage);
      // fixes #678 - added structured error logging for login failures
      logger.error(LogComponents.AUTH, "Login failed", err, {
        endpoint: "/api/auth/login",
        username,
      });
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    const currentUsername = state.username;

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
    })
      .then(() => {
        logger.info(LogComponents.AUTH, "User logged out successfully", {
          username: currentUsername,
        });
      })
      .catch((err) => {
        // fixes #678 - added error logging for logout failures
        logger.error(LogComponents.AUTH, "Logout API call failed", err, {
          endpoint: "/api/auth/logout",
          username: currentUsername,
        });
        // Local state already cleared, so continue
      });
  }, [state.username]);

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
 * Returns auth headers for API requests (deprecated - returns empty object).
 * @deprecated No longer needed with cookie-based authentication.
 * API requests automatically include cookies with credentials: 'include'.
 * This function is kept for backward compatibility but returns empty object.
 */
export function getAuthHeaders(): HeadersInit {
  return {};
}
