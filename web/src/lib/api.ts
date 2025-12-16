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

/** Flag to prevent multiple simultaneous refresh attempts */
let isRefreshing = false;

/**
 * Registers a callback to be invoked when the API returns a 401 Unauthorized response
 * and token refresh fails. Typically used to logout the user and redirect to the login page.
 *
 * @param callback - Function to call when session expires
 */
export function setSessionExpiredCallback(callback: SessionExpiredCallback): void {
  onSessionExpired = callback;
}

/**
 * Attempts to refresh the access token using the refresh token cookie.
 * Returns true if refresh succeeded, false otherwise.
 */
async function refreshAccessToken(): Promise<boolean> {
  if (isRefreshing) {
    // Wait for ongoing refresh attempt
    await new Promise((resolve) => setTimeout(resolve, 100));
    return true; // Assume it succeeded
  }

  isRefreshing = true;
  try {
    const response = await fetch(`${API_BASE}/api/auth/refresh`, {
      method: "POST",
      credentials: "include", // Send refresh token cookie
    });
    return response.ok;
  } catch {
    return false;
  } finally {
    isRefreshing = false;
  }
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
  retryRequest?: () => Promise<Response>
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
  async get<T>(endpoint: string): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const makeRequest = () =>
      fetch(`${API_BASE}${endpoint}`, {
        credentials: "include", // Send httpOnly cookies
      });

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest);
  },

  /**
   * Performs a POST request with optional JSON body.
   *
   * @param endpoint - API endpoint path
   * @param body - Request body (will be JSON serialized)
   * @returns Promise resolving to typed response data
   * @example
   * const result = await api.post<Result>('/api/network/scan', { subnet: '192.168.1.0/24' });
   */
  async post<T>(endpoint: string, body?: unknown): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const makeRequest = () =>
      fetch(`${API_BASE}${endpoint}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include", // Send httpOnly cookies
        body: body ? JSON.stringify(body) : undefined,
      });

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest);
  },

  /**
   * Performs a PUT request with optional JSON body.
   *
   * @param endpoint - API endpoint path
   * @param body - Request body (will be JSON serialized)
   * @returns Promise resolving to typed response data
   * @example
   * await api.put('/api/settings', { theme: 'dark' });
   */
  async put<T>(endpoint: string, body?: unknown): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const makeRequest = () =>
      fetch(`${API_BASE}${endpoint}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include", // Send httpOnly cookies
        body: body ? JSON.stringify(body) : undefined,
      });

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest);
  },

  /**
   * Performs a DELETE request to the specified endpoint.
   *
   * @param endpoint - API endpoint path
   * @returns Promise resolving to typed response data
   * @example
   * await api.delete('/api/devices/12345');
   */
  async delete<T>(endpoint: string): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const makeRequest = () =>
      fetch(`${API_BASE}${endpoint}`, {
        method: "DELETE",
        credentials: "include", // Send httpOnly cookies
      });

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
      headers: {
        ...init?.headers,
      },
    });
  },
};
