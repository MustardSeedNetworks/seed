// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Hook complexity is acceptable
/**
 * Generic Fetch Hook
 *
 * A reusable hook for type-safe API fetching with built-in state management.
 * Consolidates common patterns found across useDevices, useNetworkData, and other hooks.
 *
 * Features:
 * - Generic type-safe fetching with <T> generic
 * - Built-in loading, error, and data states
 * - fetch() and refetch() functions
 * - Automatic error logging
 * - Optional auto-fetch on mount
 * - Support for all HTTP methods
 * - Configurable request options
 *
 * Usage:
 * ```typescript
 * // Simple GET request with auto-fetch
 * const { data, loading, error, refetch } = useFetch<DeviceData[]>({
 *   url: '/api/v1/devices',
 *   errorMessage: 'Failed to fetch devices',
 *   autoFetch: true,
 * });
 *
 * // POST request with manual trigger
 * const { data, loading, fetch } = useFetch<ScanResult>({
 *   url: '/api/v1/scan',
 *   method: 'POST',
 *   errorMessage: 'Failed to start scan',
 * });
 *
 * // Trigger with body
 * await fetch({ subnet: '192.168.1.0/24' });
 * ```
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api';
import { type LogComponent, LogComponents, logger } from '../lib/logger';

/** HTTP methods supported by the hook */
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

/** Configuration options for useFetch hook */
export interface UseFetchOptions<T> {
  /** API endpoint URL (relative path) */
  url: string;

  /** HTTP method (defaults to GET) */
  method?: HttpMethod;

  /** Error message for logging when request fails */
  errorMessage: string;

  /** Component name for logging (defaults to 'app') */
  logComponent?: LogComponent;

  /** Automatically fetch on mount (defaults to false) */
  autoFetch?: boolean;

  /** Initial data value before first fetch */
  initialData?: T;

  /** Default request body for POST/PUT/PATCH requests */
  defaultBody?: unknown;

  /** Additional headers for requests */
  headers?: Record<string, string>;

  /** Transform response data before setting state */
  transform?: (response: unknown) => T;
}

/** Return type for useFetch hook */
export interface UseFetchResult<T> {
  /** Fetched data (null before first successful fetch) */
  data: T | null;

  /** Loading state */
  loading: boolean;

  /** Error message (null if no error) */
  error: string | null;

  /** Execute fetch with optional body override */
  fetch: (body?: unknown) => Promise<T | null>;

  /** Re-execute the last fetch (alias for fetch without body) */
  refetch: () => Promise<T | null>;

  /** Clear data and error state */
  reset: () => void;
}

/**
 * Generic hook for type-safe API fetching with state management.
 *
 * @typeParam T - The expected response data type
 * @param options - Configuration options for the fetch operation
 * @returns Fetch state and control functions
 *
 * @example
 * ```typescript
 * const { data, loading, error, fetch, refetch } = useFetch<Device[]>({
 *   url: '/api/v1/shell/devices',
 *   errorMessage: 'Failed to fetch devices',
 *   autoFetch: true,
 *   logComponent: LogComponents.DEVICES,
 * });
 * ```
 */
export function useFetch<T>(options: UseFetchOptions<T>): UseFetchResult<T> {
  const {
    url,
    method = 'GET',
    errorMessage,
    logComponent = LogComponents.APP,
    autoFetch = false,
    initialData,
    defaultBody,
    headers,
    transform,
  } = options;

  const [data, setData] = useState<T | null>(initialData ?? null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Track if component is mounted to avoid state updates after unmount
  const isMountedRef = useRef(true);

  // Track the last body used for refetch
  const lastBodyRef = useRef<unknown>(defaultBody);

  /**
   * Executes the fetch request with optional body override.
   *
   * @param body - Optional request body (overrides defaultBody)
   * @returns Promise resolving to fetched data or null on error
   */
  const executeFetch = useCallback(
    async (body?: unknown): Promise<T | null> => {
      // Store body for potential refetch
      if (body !== undefined) {
        lastBodyRef.current = body;
      }

      if (!isMountedRef.current) {
        return null;
      }

      setLoading(true);
      setError(null);

      try {
        let response: unknown;
        const requestBody = body ?? lastBodyRef.current;

        // Execute appropriate API method based on HTTP method
        switch (method) {
          case 'GET':
            response = await api.get<unknown>(url, headers ? { headers } : undefined);
            break;
          case 'POST':
            response = await api.post<unknown>(url, requestBody, headers ? { headers } : undefined);
            break;
          case 'PUT':
            response = await api.put<unknown>(url, requestBody, headers ? { headers } : undefined);
            break;
          case 'PATCH':
            response = await api.patch<unknown>(
              url,
              requestBody,
              headers ? { headers } : undefined,
            );
            break;
          case 'DELETE':
            response = await api.delete<unknown>(url, headers ? { headers } : undefined);
            break;
          default:
            throw new Error(`Unsupported HTTP method: ${method}`);
        }

        // Apply transform if provided, otherwise cast directly
        const transformedData = transform ? transform(response) : (response as T);

        if (isMountedRef.current) {
          setData(transformedData);
          setLoading(false);
        }

        return transformedData;
      } catch (err) {
        const message = err instanceof Error ? err.message : errorMessage;

        if (isMountedRef.current) {
          setError(message);
          setLoading(false);
        }

        logger.error(logComponent, errorMessage, err, {
          endpoint: url,
          method,
        });

        return null;
      }
    },
    [url, method, errorMessage, logComponent, headers, transform],
  );

  /**
   * Re-executes the last fetch operation.
   */
  const refetch = useCallback((): Promise<T | null> => executeFetch(), [executeFetch]);

  /**
   * Resets data and error state to initial values.
   */
  const reset = useCallback(() => {
    setData(initialData ?? null);
    setError(null);
    setLoading(false);
    lastBodyRef.current = defaultBody;
  }, [initialData, defaultBody]);

  // Handle auto-fetch on mount
  useEffect(() => {
    isMountedRef.current = true;

    if (autoFetch) {
      executeFetch().catch(() => undefined);
    }

    return (): void => {
      isMountedRef.current = false;
    };
  }, [autoFetch, executeFetch]);

  return {
    data,
    loading,
    error,
    fetch: executeFetch,
    refetch,
    reset,
  };
}

/**
 * Simplified hook for GET requests with auto-fetch enabled by default.
 *
 * @typeParam T - The expected response data type
 * @param url - API endpoint URL
 * @param errorMessage - Error message for logging
 * @param options - Additional configuration options
 * @returns Fetch state and control functions
 *
 * @example
 * ```typescript
 * const { data, loading, error, refetch } = useGet<Device[]>(
 *   '/api/v1/shell/devices',
 *   'Failed to fetch devices',
 *   { logComponent: LogComponents.DEVICES }
 * );
 * ```
 */
export function useGet<T>(
  url: string,
  errorMessage: string,
  options: Omit<UseFetchOptions<T>, 'url' | 'errorMessage' | 'method'> = {},
): UseFetchResult<T> {
  return useFetch<T>({
    url,
    errorMessage,
    method: 'GET',
    autoFetch: true,
    ...options,
  });
}

/**
 * Simplified hook for POST requests.
 *
 * @typeParam T - The expected response data type
 * @param url - API endpoint URL
 * @param errorMessage - Error message for logging
 * @param options - Additional configuration options
 * @returns Fetch state and control functions
 *
 * @example
 * ```typescript
 * const { data, loading, fetch } = usePost<ScanResult>(
 *   '/api/v1/shell/devices/scan',
 *   'Failed to start scan'
 * );
 *
 * // Trigger the POST request
 * await fetch({ subnet: '192.168.1.0/24' });
 * ```
 */
export function usePost<T>(
  url: string,
  errorMessage: string,
  options: Omit<UseFetchOptions<T>, 'url' | 'errorMessage' | 'method'> = {},
): UseFetchResult<T> {
  return useFetch<T>({
    url,
    errorMessage,
    method: 'POST',
    autoFetch: false,
    ...options,
  });
}
