/**
 * SSE Polling Hook
 *
 * Manages REST polling as a complement/fallback to SSE real-time updates.
 *
 * When SSE is connected:
 * - Backend pushes card updates every 5 seconds
 * - This hook polls supplementary data (interfaces, WiFi details, etc.) every 60 seconds
 *
 * When SSE is disconnected:
 * - Uses exponential backoff polling (15s, 30s, 60s, 120s, 240s capped)
 * - Polls all card data to maintain dashboard updates
 *
 * This hook was extracted from App.tsx (#892) to:
 * - Simplify the main App component
 * - Make polling logic more maintainable and testable
 * - Centralize polling interval configuration
 *
 * @see Issue #672 - Fallback REST polling
 * @see Issue #892 - Extract complex polling logic
 */

import { useEffect, useRef } from "react";
import type { SseConnectionStatus } from "./useSse";

/** Polling intervals configuration */
const pollingIntervals = {
  /** Supplementary data polling when SSE is connected */
  sseConnectedInterval: 60000, // 60 seconds

  /** Base delay for exponential backoff when SSE disconnected */
  backoffBaseDelay: 15000, // 15 seconds

  /** Maximum polling interval (capped backoff) */
  backoffMaxDelay: 240000, // 4 minutes

  /** Number of backoff steps before resetting */
  backoffMaxAttempts: 5,
} as const;

/** Fetcher functions that can be called during polling */
interface PollingFetchers {
  /** Fetch link/interface status */
  fetchLinkData: () => Promise<void>;
  /** Fetch IP configuration */
  fetchIpConfig: () => Promise<void>;
  /** Fetch available interfaces list */
  fetchInterfaces: () => Promise<void>;
  /** Fetch switch/neighbor discovery data */
  fetchDiscoveryData: () => Promise<void>;
  /** Fetch DNS test data */
  fetchDnsData: () => Promise<void>;
  /** Fetch gateway ping data */
  fetchGatewayData: () => Promise<void>;
  /** Fetch VLAN data */
  fetchVlanData: () => Promise<void>;
  /** Fetch WiFi data */
  fetchWifiData: () => Promise<void>;
  /** Fetch cable test data */
  fetchCableData: () => Promise<void>;
  /** Fetch WiFi channel graph data */
  fetchChannelGraphData: () => Promise<void>;
}

/** Options for useSsePolling hook */
interface UseSsePollingOptions {
  /** Whether user is authenticated */
  isAuthenticated: boolean;
  /** Current SSE connection status */
  sseStatus: SseConnectionStatus;
  /** All fetcher functions */
  fetchers: PollingFetchers;
}

/**
 * Hook for managing REST polling alongside SSE real-time updates.
 *
 * Provides fallback polling when SSE is disconnected and supplementary
 * data polling when SSE is connected (for data not broadcast via SSE).
 *
 * @param options - Polling configuration and fetcher functions
 */
export function useSsePolling({
  isAuthenticated,
  sseStatus,
  fetchers,
}: UseSsePollingOptions): void {
  const {
    fetchLinkData,
    fetchIpConfig,
    fetchInterfaces,
    fetchDiscoveryData,
    fetchDnsData,
    fetchGatewayData,
    fetchVlanData,
    fetchWifiData,
    fetchCableData,
    fetchChannelGraphData,
  } = fetchers;

  // Track backoff attempts with a ref to persist across effect reruns
  const attemptsRef = useRef(0);

  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    // Only poll if SSE is not connected
    if (sseStatus === "connected") {
      // SSE provides real-time updates, no need for aggressive polling
      // Still poll some endpoints that aren't broadcast via SSE
      const slowInterval: ReturnType<typeof setInterval> = setInterval(() => {
        fetchInterfaces().catch(() => undefined);
        fetchWifiData().catch(() => undefined); // WiFi details not broadcast via SSE
        fetchCableData().catch(() => undefined); // Cable test not broadcast via SSE
        fetchChannelGraphData().catch(() => undefined); // Channel graph data for WiFi visualization
      }, pollingIntervals.sseConnectedInterval);

      return (): void => clearInterval(slowInterval);
    }

    // Fallback: Poll when SSE disconnected with exponential backoff
    // Reset attempts when SSE status changes to ensure fresh backoff sequence
    attemptsRef.current = 0;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const poll = (): void => {
      fetchLinkData().catch(() => undefined);
      fetchIpConfig().catch(() => undefined);
      fetchDiscoveryData().catch(() => undefined);
      fetchDnsData().catch(() => undefined);
      fetchGatewayData().catch(() => undefined);
      fetchVlanData().catch(() => undefined);
      fetchWifiData().catch(() => undefined);

      // Schedule next poll with exponential backoff
      const delay = Math.min(
        pollingIntervals.backoffBaseDelay * 2 ** attemptsRef.current,
        pollingIntervals.backoffMaxDelay,
      );

      // Increase attempts up to max, then cap (no reset - continuous max interval)
      if (attemptsRef.current < pollingIntervals.backoffMaxAttempts) {
        attemptsRef.current++;
      }

      timeoutId = setTimeout(poll, delay);
    };

    // Start initial poll with base delay
    timeoutId = setTimeout(poll, pollingIntervals.backoffBaseDelay);

    return () => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [
    isAuthenticated,
    sseStatus,
    fetchLinkData,
    fetchIpConfig,
    fetchInterfaces,
    fetchDiscoveryData,
    fetchDnsData,
    fetchGatewayData,
    fetchVlanData,
    fetchWifiData,
    fetchCableData,
    fetchChannelGraphData,
  ]);
}

export default useSsePolling;
