/**
 * API Client Library
 *
 * Provides a centralized HTTP client for communicating with the Seed backend API.
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
 * import { api } from './api';
 *
 * // GET request
 * const data = await api.get<MyType>('/api/v1/endpoint');
 *
 * // POST request with body
 * const result = await api.post<Response>('/api/v1/endpoint', { key: 'value' });
 * ```
 *
 * Session Management:
 * The API client automatically handles 401 Unauthorized responses by attempting
 * token refresh. If refresh fails, invokes session expired callback.
 */

// API base URL - can be overridden via VITE_API_BASE environment variable
const API_BASE: string = import.meta.env.VITE_API_BASE || '';

/** Callback function invoked when session expires (401 response after refresh attempt) */
type SessionExpiredCallback = () => void;

/**
 * Thrown by {@link handleResponse} when a 401 survives a refresh attempt (or
 * refresh itself fails). Callers that need to distinguish "the session ended,
 * the global session-expired flow is already handling it" from any other
 * request failure should check `instanceof SessionExpiredError` rather than
 * matching on the error message.
 */
export class SessionExpiredError extends Error {
  constructor() {
    super('Session expired');
    this.name = 'SessionExpiredError';
  }
}

/** Global session expired callback - set via setSessionExpiredCallback */
let onSessionExpired: SessionExpiredCallback | null = null;

/**
 * Monotonic session generation, incremented by beginSession() on every
 * successful login.
 *
 * A 401 is only evidence that *the session the request was issued under* is
 * gone. Without this counter a response that lands after the user has logged in
 * again expires the session that replaced it — the user is returned to the login
 * form reading "Session expired" moments after a successful login (#2204).
 */
let sessionGeneration = 0;

/** Promise to track ongoing refresh attempts and prevent race conditions */
let refreshPromise: Promise<boolean> | null = null;

/** CSRF token cache - fetched once after login, used for all state-changing requests */
let csrfToken: string | null = null;

/**
 * In-flight CSRF fetch, so concurrent callers share one mint.
 *
 * `GET /auth/csrf` calls foundation's `Manager.Generate`, which *replaces* the
 * session's stored token, so two concurrent fetches leave the first caller
 * holding a token the server has already discarded. Two mutations that both
 * 401 wake from the same refresh and both need a token, which is exactly that
 * race — so they queue behind one fetch, the way refreshPromise queues refreshes.
 */
let csrfFetchPromise: Promise<string | null> | null = null;

/** CSRF token header name - must match backend auth.CSRFHeaderName */
const CSRF_HEADER_NAME = 'X-CSRF-Token';

/**
 * The pre-session endpoints, mirroring `isCSRFExemptPath` in
 * `internal/auth/csrf.go`. A request to one of these runs before the user holds
 * an access token, so there is no session to mint a CSRF token against and no
 * session to expire — the flag suppresses both the token header and the 401
 * refresh-and-retry.
 *
 * It is an exact-match set, not a `/api/v1/auth/` prefix. The prefix also caught
 * the MFA *enrolment* routes, which the server does protect, so `totp/setup`,
 * `totp/verify`, `totp/disable` and `webauthn/register/{begin,finish}` went out
 * bare and came back `403 CSRF token required` — MFA could not be enabled from
 * the UI at all (#2725). Mirror image of #2632, which was the same prefix bug
 * on `/api/v1/sso/`.
 */
const PRE_SESSION_ENDPOINTS: ReadonlySet<string> = new Set([
  '/api/v1/auth/login',
  '/api/v1/auth/refresh',
  '/api/v1/auth/logout',
  '/api/v1/auth/login/totp',
  '/api/v1/auth/webauthn/login/begin',
  '/api/v1/auth/webauthn/login/finish',
]);

/**
 * Whether an endpoint runs before the user holds a session.
 *
 * The query string is dropped before the lookup: passkey sign-in posts to
 * `webauthn/login/finish?username=…`, and matching the raw endpoint would treat
 * the one pre-session route that carries a query as authenticated.
 */
function isPreSessionEndpoint(endpoint: string): boolean {
  const queryStart: number = endpoint.indexOf('?');
  return PRE_SESSION_ENDPOINTS.has(queryStart === -1 ? endpoint : endpoint.slice(0, queryStart));
}

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
  const existingPromise: Promise<boolean> | null = refreshPromise;
  if (existingPromise !== null) {
    return existingPromise;
  }

  // Start new refresh attempt
  const newPromise: Promise<boolean> = (async (): Promise<boolean> => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
        method: 'POST',
        credentials: 'include', // Send refresh token cookie
      });
      if (response.ok) {
        // The server keys CSRF tokens by the bearer (auth.GetSessionIDFromRequest
        // → csrf.SessionKey), and a refresh mints a new JWT, so the cached token
        // belongs to a session key that no longer exists. Cleared here rather
        // than after the await so a second caller waiting on this same promise
        // cannot resume and reuse it (#2633).
        csrfToken = null;
      }
      return response.ok;
    } catch {
      return false;
    }
  })();
  refreshPromise = newPromise;

  // Clear the promise when done (success or failure)
  const result: boolean = await newPromise;
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
function fetchCsrfToken(): Promise<string | null> {
  const existingPromise: Promise<string | null> | null = csrfFetchPromise;
  if (existingPromise !== null) {
    return existingPromise;
  }

  const newPromise: Promise<string | null> = (async (): Promise<string | null> => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/auth/csrf`, {
        method: 'GET',
        credentials: 'include', // Send auth cookies
      });
      if (response.ok) {
        const data: { token: string } = await (response.json() as Promise<{ token: string }>);
        csrfToken = data.token;
        return csrfToken;
      }
      return null;
    } catch {
      return null;
    } finally {
      csrfFetchPromise = null;
    }
  })();
  csrfFetchPromise = newPromise;
  return newPromise;
}

/**
 * Gets the current CSRF token, fetching if needed.
 * Returns null if user is not authenticated.
 */
function getCsrfToken(): Promise<string | null> {
  if (csrfToken) {
    return Promise.resolve(csrfToken);
  }
  return fetchCsrfToken();
}

/**
 * Clears the cached CSRF token. Should be called on logout.
 */
export function clearCSRFToken(): void {
  csrfToken = null;
  csrfFetchPromise = null;
}

/**
 * Marks the start of a new authenticated session.
 *
 * Call on every successful login, and when a mount-time probe finds a session
 * already established by a cookie — both are how this client comes to hold
 * one. In-flight requests from a previous session can no longer expire the
 * new one, and a 401 collected before any call to this function is not an
 * expiry at all (#2643).
 */
export function beginSession(): void {
  sessionGeneration++;
}

/**
 * Builds the error for a non-2xx response.
 *
 * The server's own message is carried through. Callers that used a raw fetch
 * read it out of the body themselves and showed it — "CIDR overlaps an
 * existing subnet" is worth more to the operator than "API error: 400" — so
 * dropping it would have made the migration off raw fetch a regression. The
 * `API error: N` prefix is kept so anything matching on it still matches.
 */
async function requestError(response: Response): Promise<Error> {
  let detail = '';
  try {
    const body = (await response.clone().json()) as { error?: string; message?: string };
    detail = body.error ?? body.message ?? '';
  } catch {
    // Not JSON, or already consumed. The status alone will have to do.
  }
  return new Error(
    detail ? `API error: ${response.status}: ${detail}` : `API error: ${response.status}`,
  );
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
  retryRequest: (() => Promise<Response>) | undefined,
  issuedGeneration: number,
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
      // The retry reached the server under a live access token, so anything
      // other than a second 401 is an ordinary request error and must surface
      // as one. Reporting it as an expired session logged the operator out of
      // a valid session — a 403 from a stale CSRF token did exactly that
      // before the cache was cleared on refresh (#2633).
      if (retryResponse.status !== 401) {
        throw await requestError(retryResponse);
      }
    }

    // Refresh failed, or the retry was itself a 401. Only expire a session
    // this client actually held, and only the one this request was issued
    // under.
    //
    // Generation 0 means no session has been established in this page load,
    // so a 401 there is the ordinary unauthenticated answer — not an expiry.
    // Without that clause `issuedGeneration === sessionGeneration` compares
    // 0 to 0 and passes, and every pre-auth fetch (ProfileProvider wraps
    // <App> in main.tsx and queries /api/v1/profiles on mount) told a
    // first-time visitor "Session expired. Please sign in again." before they
    // had ever signed in (#2643).
    //
    // Above 0, a straggler from a session the user has already replaced must
    // not tear down its successor (#2204).
    if (sessionGeneration > 0 && issuedGeneration === sessionGeneration) {
      onSessionExpired?.();
    }
    throw new SessionExpiredError();
  }

  if (!response.ok) {
    throw await requestError(response);
  }

  // Parse and return JSON response
  return response.json();
}

/**
 * Issues a state-changing request, attaching the CSRF token the server expects.
 *
 * The token is read inside `makeRequest`, not captured before the first
 * attempt: a 401 retry runs after `refreshAccessToken` has dropped the cached
 * token, so the retry mints one keyed to the new bearer. Capturing it once sent
 * the old session's token to the retry and earned a 403 (#2633).
 */
async function mutate<T>(
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  endpoint: string,
  body: unknown,
  init: RequestInit | undefined,
): Promise<T> {
  const isAuthEndpoint: boolean = isPreSessionEndpoint(endpoint);
  const issuedGeneration = sessionGeneration;

  const makeRequest = async (): Promise<Response> => {
    const headers: Headers = new Headers(
      method === 'DELETE' ? undefined : { 'Content-Type': 'application/json' },
    );
    const token: string | null = isAuthEndpoint ? null : await getCsrfToken();
    if (token) {
      headers.set(CSRF_HEADER_NAME, token);
    }
    for (const [key, value] of new Headers(init?.headers).entries()) {
      headers.set(key, value);
    }

    return fetch(`${API_BASE}${endpoint}`, {
      ...init,
      method,
      credentials: 'include', // Send httpOnly cookies
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  };

  const response = await makeRequest();
  return handleResponse<T>(response, isAuthEndpoint, makeRequest, issuedGeneration);
}

/**
 * API client object providing HTTP methods for backend communication.
 * All methods automatically include httpOnly cookie credentials.
 */
export const api = {
  /**
   * Performs a GET request to the specified endpoint.
   *
   * @param endpoint - API endpoint path (e.g., '/api/v1/network/status')
   * @returns Promise resolving to typed response data
   * @example
   * const status = await api.get<NetworkStatus>('/api/v1/network/status');
   */
  async get<T>(endpoint: string, init?: RequestInit): Promise<T> {
    const isAuthEndpoint: boolean = isPreSessionEndpoint(endpoint);
    const issuedGeneration = sessionGeneration;
    const makeRequest = (): Promise<Response> =>
      fetch(`${API_BASE}${endpoint}`, {
        ...init,
        method: 'GET',
        credentials: 'include', // Send httpOnly cookies
        headers: new Headers(init?.headers),
      });

    const response = await makeRequest();
    return handleResponse<T>(response, isAuthEndpoint, makeRequest, issuedGeneration);
  },

  /**
   * Performs a POST request with optional JSON body.
   * Automatically includes CSRF token for authenticated requests.
   *
   * @param endpoint - API endpoint path
   * @param body - Request body (will be JSON serialized)
   * @returns Promise resolving to typed response data
   * @example
   * const result = await api.post<Result>('/api/v1/network/scan', { subnet: '192.168.1.0/24' });
   */
  post<T>(endpoint: string, body?: unknown, init?: RequestInit): Promise<T> {
    return mutate<T>('POST', endpoint, body, init);
  },

  /**
   * Performs a PUT request with optional JSON body.
   * Automatically includes CSRF token for authenticated requests.
   *
   * @param endpoint - API endpoint path
   * @param body - Request body (will be JSON serialized)
   * @returns Promise resolving to typed response data
   * @example
   * await api.put('/api/v1/settings', { theme: 'dark' });
   */
  put<T>(endpoint: string, body?: unknown, init?: RequestInit): Promise<T> {
    return mutate<T>('PUT', endpoint, body, init);
  },

  /**
   * Performs a PATCH request with optional JSON body.
   * Automatically includes CSRF token for authenticated requests.
   *
   * @param endpoint - API endpoint path
   * @param body - Request body (will be JSON serialized)
   * @returns Promise resolving to typed response data
   * @example
   * await api.patch('/api/v1/settings', { theme: 'dark' });
   */
  patch<T>(endpoint: string, body?: unknown, init?: RequestInit): Promise<T> {
    return mutate<T>('PATCH', endpoint, body, init);
  },

  /**
   * Performs a DELETE request to the specified endpoint.
   * Automatically includes CSRF token for authenticated requests.
   *
   * @param endpoint - API endpoint path
   * @returns Promise resolving to typed response data
   * @example
   * await api.delete('/api/v1/devices/12345');
   */
  delete<T>(endpoint: string, init?: RequestInit): Promise<T> {
    return mutate<T>('DELETE', endpoint, undefined, init);
  },

  /**
   * Raw fetch method for cases requiring direct Response object access.
   * Automatically includes cookie credentials.
   *
   * @param endpoint - API endpoint path
   * @param init - Optional fetch configuration
   * @returns Promise resolving to Fetch API Response object
   */
  fetch(endpoint: string, init?: RequestInit): Promise<Response> {
    return fetch(`${API_BASE}${endpoint}`, {
      ...init,
      credentials: 'include', // Send httpOnly cookies
      headers: new Headers(init?.headers),
    });
  },
};
