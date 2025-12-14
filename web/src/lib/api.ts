/**
 * API Client Library
 * 
 * Provides a centralized HTTP client for communicating with the LuminetIQ backend API.
 * 
 * Features:
 * - Automatic JWT token injection from localStorage
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
 * The API client automatically handles 401 Unauthorized responses by invoking
 * a registered session expired callback, typically used to redirect to login.
 */

// API base URL - can be overridden via VITE_API_BASE environment variable
const API_BASE = import.meta.env.VITE_API_BASE || "";

/** Callback function invoked when session expires (401 response) */
type SessionExpiredCallback = () => void;

/** Global session expired callback - set via setSessionExpiredCallback */
let onSessionExpired: SessionExpiredCallback | null = null;

/**
 * Registers a callback to be invoked when the API returns a 401 Unauthorized response.
 * Typically used to logout the user and redirect to the login page.
 * 
 * @param callback - Function to call when session expires
 */
export function setSessionExpiredCallback(
  callback: SessionExpiredCallback,
): void {
  onSessionExpired = callback;
}

/**
 * Retrieves the JWT authentication token from localStorage.
 * 
 * @returns JWT token string or null if not authenticated
 */
function getToken(): string | null {
  return localStorage.getItem("netscope-token");
}

/**
 * Constructs HTTP headers with JWT authentication if available.
 * 
 * @returns Headers object with Authorization header if token exists
 */
function getAuthHeaders(): HeadersInit {
  const token = getToken();
  if (token) {
    return {
      Authorization: `Bearer ${token}`,
    };
  }
  return {};
}

/**
 * Handles API response processing including error handling and JSON parsing.
 * 
 * Automatically triggers session expired callback on 401 responses (except for auth endpoints).
 * Throws errors for non-2xx status codes.
 * 
 * @param response - Fetch API Response object
 * @param isAuthEndpoint - If true, skips session expiration handling
 * @returns Parsed JSON response data
 * @throws Error on non-2xx status codes or session expiration
 */
async function handleResponse<T>(
  response: Response,
  isAuthEndpoint: boolean,
): Promise<T> {
  // Check for unauthorized access (session expired)
  if (response.status === 401 && !isAuthEndpoint) {
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
 * All methods automatically include JWT authentication headers.
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
    const response = await fetch(`${API_BASE}${endpoint}`, {
      headers: getAuthHeaders(),
    });
    return handleResponse<T>(response, isAuthEndpoint);
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
    const headers: HeadersInit = {
      ...getAuthHeaders(),
      "Content-Type": "application/json",
    };

    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: "POST",
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    return handleResponse<T>(response, isAuthEndpoint);
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
    const headers: HeadersInit = {
      ...getAuthHeaders(),
      "Content-Type": "application/json",
    };

    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: "PUT",
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    return handleResponse<T>(response, isAuthEndpoint);
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
    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: "DELETE",
      headers: getAuthHeaders(),
    });
    return handleResponse<T>(response, isAuthEndpoint);
  },

  /**
   * Raw fetch method for cases requiring direct Response object access.
   * Automatically includes authentication headers.
   * 
   * @param endpoint - API endpoint path
   * @param init - Optional fetch configuration
   * @returns Promise resolving to Fetch API Response object
   */
  async fetch(endpoint: string, init?: RequestInit): Promise<Response> {
    const headers: HeadersInit = {
      ...getAuthHeaders(),
      ...init?.headers,
    };

    return fetch(`${API_BASE}${endpoint}`, {
      ...init,
      headers,
    });
  },
};
