/**
 * API Client Library
 *
 * Provides a centralized HTTP client for communicating with the The Seed backend API.
 *
 * Features:
 * - Cookie-based authentication (httpOnly cookies)
 * - Automatic token refresh on expiration
 * - Session expiration handling with callback mechanism
 * - Type-safe request/response handling
 * - Support for GET, POST, PUT, DELETE operations
 * - Automatic JSON serialization/deserialization
 *
 * Usage:
 * ```typescript
 * import { api } from './lib/api';
 *
 * // GET request
 * const data = await api.get<MyType>('/api/endpoint');
 *
 * // POST request with body
 * const result = await api.post<Response>('/api/endpoint', { key: 'value' });
 * ```
 *
 * Session Management:
 * The API client automatically handles 401 Unauthorized responses by attempting
 * token refresh. If refresh fails, invokes session expired callback.
 */

// API base URL - can be overridden via VITE_API_BASE environment variable
const API_BASE = import.meta.env.VITE_API_BASE || "";

/** Callback function invoked when session expires (401 response after refresh attempt) */
type SessionExpiredCallback = () => void;

/** Global session expired callback - set via setSessionExpiredCallback */
let onSessionExpired: SessionExpiredCallback | null = null;

/** Promise to track ongoing refresh attempts and prevent race conditions */
let refreshPromise: Promise<boolean> | null = null;

/** CSRF token cache - fetched once after login, used for all state-changing requests */
let csrfToken: string | null = null;

/** CSRF token header name - must match backend auth.CSRFHeaderName */
const CSRF_HEADER_NAME = "X-CSRF-Token";

/**
 * Registers a callback to be invoked when the API returns a 401 Unauthorized response
 * and token refresh fails. Typically used to logout the user and redirect to the login page.
 *
 * @param callback - Function to call when session expires
 */
export function setSessionExpiredCallback(callback: SessionExpiredCallback | null): void {
  onSessionExpired = callback;
}

/**
 * Attempts to refresh the access token using the refresh token cookie.
 * Returns true if refresh succeeded, false otherwise.
 *
 * Uses a mutex pattern to prevent multiple simultaneous refresh attempts.
 * If a refresh is already in progress, waits for it to complete instead of
 * making duplicate requests.
 */
async function refreshAccessToken(): Promise<boolean> {
  // If refresh is already in progress, wait for it to complete
  if (refreshPromise) {
    return refreshPromise;
  }

  // Start new refresh attempt
  refreshPromise = (async () => {
    try {
      const response = await fetch(`${API_BASE}/api/auth/refresh`, {
        method: "POST",
        credentials: "include", // Send refresh token cookie
      });
      return response.ok;
    } catch {
      return false;
    }
  })();

  // Clear the promise when done (success or failure)
  const result = await refreshPromise;
  refreshPromise = null;
  return result;
}

/**
 * Fetches a CSRF token from the backend.
 * Called automatically on first state-changing request after login.
 * Token is cached and reused for subsequent requests.
 *
 * @returns Promise resolving to CSRF token or null if fetch fails
 */
async function fetchCSRFToken(): Promise<string | null> {
  try {
    const response = await fetch(`${API_BASE}/api/auth/csrf`, {
      method: "GET",
      credentials: "include", // Send auth cookies
    });
    if (response.ok) {
      const data = (await response.json()) as { token: string };
      csrfToken = data.token;
      return csrfToken;
    }
    return null;
  } catch {
    return null;
  }
}

/**
 * Gets the current CSRF token, fetching if needed.
 * Returns null if user is not authenticated.
 */
async function getCSRFToken(): Promise<string | null> {
  if (csrfToken) {
    return csrfToken;
  }
  return fetchCSRFToken();
}

/**
 * Clears the cached CSRF token. Should be called on logout.
 */
export function clearCSRFToken(): void {
  csrfToken = null;
}

/**
 * Handles API response processing including error handling and JSON parsing.
 *
 * Automatically attempts token refresh on 401 responses (except for auth endpoints).
 * Throws errors for non-2xx status codes.
 *
 * @param response - Fetch API Response object
 * @param isAuthEndpoint - If true, skips token refresh and session expiration handling
 * @param retryRequest - Optional function to retry the original request after token refresh
 * @returns Parsed JSON response data
 * @throws Error on non-2xx status codes or session expiration
 */
async function handleResponse<T>(
  response: Response,
  isAuthEndpoint: boolean,
  retryRequest?: () => Promise<Response>,
): Promise<T> {
  // Check for unauthorized access (token expired)
  if (response.status === 401 && !isAuthEndpoint) {
    // Attempt to refresh access token using refresh token cookie
    const refreshed = await refreshAccessToken();

    if (refreshed && retryRequest) {
      // Token refreshed successfully, retry original request
      const retryResponse = await retryRequest();
      if (retryResponse.ok) {
        return retryResponse.json();
      }
    }

    // Refresh failed or retry failed - session truly expired
    onSessionExpired?.();
    throw new Error("Session expired");
  }

  // Handle non-success responses
  if (!response.ok) {
    throw new Error(`API error: ${response.status}`);
  }

  // Parse and return JSON response
  return response.json();
}

/**
 * API client object providing HTTP methods for backend communication.
 * All methods automatically include httpOnly cookie credentials.
 */
export const api = {
  /**
   * Performs a GET request to the specified endpoint.
   *
   * @param endpoint - API endpoint path (e.g., '/api/network/status')
   * @returns Promise resolving to typed response data
   * @example
   * const status = await api.get<NetworkStatus>('/api/network/status');
   */
  async get<T>(endpoint: string, init?: RequestInit): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const makeRequest = () =>
      fetch(`${API_BASE}${endpoint}`, {
        ...init,
        method: "GET",
        credentials: "include", // Send httpOnly cookies
        headers: new Headers(init?.headers),
      });

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest);
  },

  /**
   * Performs a POST request with optional JSON body.
   * Automatically includes CSRF token for authenticated requests.
   *
   * @param endpoint - API endpoint path
   * @param body - Request body (will be JSON serialized)
   * @returns Promise resolving to typed response data
   * @example
   * const result = await api.post<Result>('/api/network/scan', { subnet: '192.168.1.0/24' });
   */
  async post<T>(endpoint: string, body?: unknown, init?: RequestInit): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    // Get CSRF token for non-auth endpoints (state-changing requests)
    const token = !isAuthEndpoint ? await getCSRFToken() : null;

    const makeRequest = () => {
      const headers = new Headers({ "Content-Type": "application/json" });
      if (token) {
        headers.set(CSRF_HEADER_NAME, token);
      }
      for (const [key, value] of new Headers(init?.headers).entries()) {
        headers.set(key, value);
      }

      return fetch(`${API_BASE}${endpoint}`, {
        ...init,
        method: "POST",
        credentials: "include", // Send httpOnly cookies
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    };

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest);
  },

  /**
   * Performs a PUT request with optional JSON body.
   * Automatically includes CSRF token for authenticated requests.
   *
   * @param endpoint - API endpoint path
   * @param body - Request body (will be JSON serialized)
   * @returns Promise resolving to typed response data
   * @example
   * await api.put('/api/settings', { theme: 'dark' });
   */
  async put<T>(endpoint: string, body?: unknown, init?: RequestInit): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    // Get CSRF token for non-auth endpoints (state-changing requests)
    const token = !isAuthEndpoint ? await getCSRFToken() : null;

    const makeRequest = () => {
      const headers = new Headers({ "Content-Type": "application/json" });
      if (token) {
        headers.set(CSRF_HEADER_NAME, token);
      }
      for (const [key, value] of new Headers(init?.headers).entries()) {
        headers.set(key, value);
      }

      return fetch(`${API_BASE}${endpoint}`, {
        ...init,
        method: "PUT",
        credentials: "include", // Send httpOnly cookies
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    };

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest);
  },

  /**
   * Performs a DELETE request to the specified endpoint.
   * Automatically includes CSRF token for authenticated requests.
   *
   * @param endpoint - API endpoint path
   * @returns Promise resolving to typed response data
   * @example
   * await api.delete('/api/devices/12345');
   */
  async delete<T>(endpoint: string, init?: RequestInit): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    // Get CSRF token for non-auth endpoints (state-changing requests)
    const token = !isAuthEndpoint ? await getCSRFToken() : null;

    const makeRequest = () => {
      const headers = new Headers(init?.headers);
      if (token) {
        headers.set(CSRF_HEADER_NAME, token);
      }

      return fetch(`${API_BASE}${endpoint}`, {
        ...init,
        method: "DELETE",
        credentials: "include", // Send httpOnly cookies
        headers,
      });
    };

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest);
  },

  /**
   * Raw fetch method for cases requiring direct Response object access.
   * Automatically includes cookie credentials.
   *
   * @param endpoint - API endpoint path
   * @param init - Optional fetch configuration
   * @returns Promise resolving to Fetch API Response object
   */
  async fetch(endpoint: string, init?: RequestInit): Promise<Response> {
    return fetch(`${API_BASE}${endpoint}`, {
      ...init,
      credentials: "include", // Send httpOnly cookies
      headers: new Headers(init?.headers),
    });
  },
};
