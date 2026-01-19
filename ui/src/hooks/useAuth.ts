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
 * - Session expiration with cleanup callback
 * - Connected state tracking
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
 * const { isAuthenticated, login, logout, expireSession, clearError } = useAuth();
 *
 * const handleLogin = async () => {
 *   const success = await login(username, password);
 *   if (success) {
 *     // User authenticated
 *   }
 * };
 * ```
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { clearCSRFToken } from '../api';
import { LogComponents, logger } from '../lib/logger';

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
  /** Whether connected to the backend */
  connected: boolean;
  /** Attempt to login with credentials. Returns true on success. */
  login: (username: string, password: string) => Promise<boolean>;
  /** Logout and clear authentication state */
  logout: () => void;
  /** Expire the session with an optional message (clears state, shows error) */
  expireSession: (message?: string) => void;
  /** Refresh the access token (for SSE/WebSocket reconnection). Returns new token or null. */
  refreshToken: () => Promise<string | null>;
  /** True while login request is in progress */
  isLoading: boolean;
  /** Error message from failed login attempt */
  error: string | null;
  /** Clear the login error */
  clearError: () => void;
  /** Set connected state */
  setConnected: (connected: boolean) => void;
  /** Polling interval ref (for cleanup on session expire) */
  pollingIntervalRef: React.MutableRefObject<number | null>;
}

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

// localStorage keys to clear on mount (migrated to httpOnly cookies)
const LEGACY_KEYS: string[] = ['seed-token', 'seed-token-expiry', 'seed-username'];

/**
 * Clears old localStorage keys from cookie migration.
 * Runs automatically on mount to clean up legacy token storage.
 */
function clearLegacyStorage(): void {
  for (const key of LEGACY_KEYS) {
    localStorage.removeItem(key);
  }
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
  const [connected, setConnected] = useState(false);
  const pollingIntervalRef = useRef<number | null>(null);

  // Expire session handler - clears state and shows error message
  const expireSession = useCallback((message = 'Session expired. Please sign in again.') => {
    // Clear any polling intervals
    if (pollingIntervalRef.current !== null) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }

    // Clear CSRF token
    clearCSRFToken();

    // Reset authentication state
    setState({
      isAuthenticated: false,
      token: null,
      username: null,
    });
    setConnected(false);
    setError(message);

    logger.warn(LogComponents.Auth, 'Session expired', { message });
  }, []);

  // Clear error handler
  const clearError = useCallback(() => {
    setError(null);
  }, []);

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
      credentials: 'include', // Send cookies
    })
      .then((response) => {
        if (response.ok) {
          // Authenticated - we don't have username from /api/status, will be set on login
          setState({
            isAuthenticated: true,
            token: null, // Will be set on login for SSE
            username: null,
          });
          setConnected(true);
        } else {
          // Not authenticated
          setState({
            isAuthenticated: false,
            token: null,
            username: null,
          });
          setConnected(false);
        }
      })
      .catch((err) => {
        // Error checking auth, assume not authenticated
        // fixes #678 - added logging for auth check errors
        logger.error(LogComponents.Auth, 'Failed to check authentication status', err, {
          endpoint: '/api/v1/status',
        });
        setState({
          isAuthenticated: false,
          token: null,
          username: null,
        });
        setConnected(false);
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
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Receive httpOnly cookies
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        throw new Error('Invalid credentials');
      }

      const data: LoginResponse = await (response.json() as Promise<LoginResponse>);

      // Backend sets httpOnly cookies automatically
      // Store access token in memory ONLY for SSE/WebSocket connections
      setState({
        isAuthenticated: true,
        token: data.token, // Access token for SSE (short-lived, 15min)
        username,
      });
      setConnected(true);

      logger.info(LogComponents.Auth, 'User logged in successfully', {
        username,
      });
      return true;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Login failed';
      setError(errorMessage);
      // fixes #678 - added structured error logging for login failures
      logger.error(LogComponents.Auth, 'Login failed', err, {
        endpoint: '/api/v1/auth/login',
        username,
      });
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    const currentUsername = state.username;

    // Clear any polling intervals
    if (pollingIntervalRef.current !== null) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }

    // Clear in-memory state immediately
    setState({
      isAuthenticated: false,
      token: null,
      username: null,
    });
    setConnected(false);

    // Clear cached CSRF token
    clearCSRFToken();

    // Call logout endpoint to clear httpOnly cookies
    fetch(`${API_BASE}/api/auth/logout`, {
      method: 'POST',
      credentials: 'include', // Send cookies to be cleared
    })
      .then(() => {
        logger.info(LogComponents.Auth, 'User logged out successfully', {
          username: currentUsername,
        });
      })
      .catch((err) => {
        // fixes #678 - added error logging for logout failures
        logger.error(LogComponents.Auth, 'Logout API call failed', err, {
          endpoint: '/api/v1/auth/logout',
          username: currentUsername,
        });
        // Local state already cleared, so continue
      });
  }, [state.username]);

  /**
   * Refresh the access token using the refresh token cookie.
   * Returns the new access token if successful, null otherwise.
   * This is used by WebSocket to get a fresh token for reconnection.
   */
  const refreshToken = useCallback(async (): Promise<string | null> => {
    try {
      const response = await fetch(`${API_BASE}/api/auth/refresh`, {
        method: 'POST',
        credentials: 'include', // Send refresh token cookie
      });

      if (!response.ok) {
        logger.warn(LogComponents.Auth, 'Token refresh failed', {
          status: response.status,
        });
        return null;
      }

      const data: LoginResponse = await (response.json() as Promise<LoginResponse>);

      // Update state with new token
      setState((prev) => ({
        ...prev,
        token: data.token,
      }));

      logger.info(LogComponents.Auth, 'Token refreshed successfully');
      return data.token;
    } catch (err) {
      logger.error(LogComponents.Auth, 'Token refresh error', err);
      return null;
    }
  }, []);

  return {
    isAuthenticated: state.isAuthenticated,
    token: state.token,
    username: state.username,
    connected,
    login,
    logout,
    expireSession,
    refreshToken,
    isLoading,
    error,
    clearError,
    setConnected,
    pollingIntervalRef,
  };
}
