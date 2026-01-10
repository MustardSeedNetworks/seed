/**
 * React Query Client Configuration
 *
 * Centralized QueryClient setup with sensible defaults for
 * network-focused application with real-time updates.
 *
 * Related: #890
 */

import { QueryClient } from "@tanstack/react-query";

/**
 * Create a QueryClient with application-specific defaults.
 *
 * Defaults optimized for:
 * - Network monitoring app with frequent data updates
 * - SSE for real-time updates (reduces need for aggressive polling)
 * - User-initiated refetching for manual tests
 */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Data considered fresh for 30 seconds (SSE handles real-time updates)
        staleTime: 30 * 1000,
        // Cache for 5 minutes (profiles don't change frequently)
        gcTime: 5 * 60 * 1000,
        // Retry failed requests up to 3 times with exponential backoff
        retry: 3,
        // Don't refetch on window focus (SSE provides updates)
        refetchOnWindowFocus: false,
        // Don't refetch when reconnecting (SSE handles this)
        refetchOnReconnect: false,
      },
      mutations: {
        // Retry mutations once on failure
        retry: 1,
      },
    },
  });
}

// Singleton QueryClient for the application
let queryClient: QueryClient | null = null;

/**
 * Get or create the singleton QueryClient.
 * Use this to access the client outside of React components.
 */
export function getQueryClient(): QueryClient {
  if (!queryClient) {
    queryClient = createQueryClient();
  }
  return queryClient;
}
