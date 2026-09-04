/**
 * useDeviceScan - triggers and polls a network device scan.
 *
 * Owns the scan poll interval and timeout refs, and now the scan's own
 * error state (#2394), tearing all of it down on unmount. Returns a stable
 * trigger callback plus whether the most recent attempt failed, for
 * App.tsx and other callers to kick off a /api/v1/security/devices/scan
 * run and show its outcome.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { api, SessionExpiredError } from '../api';
import type { NetworkDiscoveryData } from '../components/cards/NetworkDiscoveryCard';
import { LogComponents, logger } from '../lib/logger';

interface UseDeviceScanArgs {
  fetchNetworkDiscovery: () => Promise<void>;
  setNetworkDiscovery: (
    updater: (prev: NetworkDiscoveryData | null) => NetworkDiscoveryData | null,
  ) => void;
}

interface UseDeviceScanReturn {
  triggerDeviceScan: () => Promise<void>;
  /**
   * Whether the most recent scan attempt failed, so the card can show it
   * instead of silently reverting to its previous contents (#2394). Not set
   * for a session-expiry 401: that is already handled by the global
   * session-expired flow (api client's onSessionExpired callback), and a
   * second, generic error here would be redundant noise on a card that is
   * about to unmount.
   */
  scanError: boolean;
}

export function useDeviceScan({
  fetchNetworkDiscovery,
  setNetworkDiscovery,
}: UseDeviceScanArgs): UseDeviceScanReturn {
  const scanPollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const scanTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [scanError, setScanError] = useState(false);

  const triggerDeviceScan = useCallback(async () => {
    try {
      // Clear any existing polling interval/timeout
      if (scanPollIntervalRef.current) {
        clearInterval(scanPollIntervalRef.current);
        scanPollIntervalRef.current = null;
      }
      if (scanTimeoutRef.current) {
        clearTimeout(scanTimeoutRef.current);
        scanTimeoutRef.current = null;
      }

      // A fresh attempt supersedes whatever error the last one left behind.
      setScanError(false);

      // Update status to show scanning
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: true },
            }
          : null,
      );

      await api.post('/api/v1/security/devices/scan');

      // Poll for completion
      scanPollIntervalRef.current = setInterval(async () => {
        try {
          const status = await api.get<{ scanning: boolean }>('/api/v1/security/devices/status');
          if (!status.scanning) {
            if (scanPollIntervalRef.current) {
              clearInterval(scanPollIntervalRef.current);
              scanPollIntervalRef.current = null;
            }
            await fetchNetworkDiscovery();
          }
        } catch {
          // Status check failed, keep polling
        }
      }, 1000);

      // Safety timeout - stop polling after 60 seconds
      scanTimeoutRef.current = setTimeout(() => {
        if (scanPollIntervalRef.current) {
          clearInterval(scanPollIntervalRef.current);
          scanPollIntervalRef.current = null;
        }
      }, 60000);
    } catch (err) {
      logger.error(LogComponents.DEVICES, 'Failed to trigger device scan', err);
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: false },
            }
          : null,
      );
      if (!(err instanceof SessionExpiredError)) {
        setScanError(true);
      }
    }
  }, [fetchNetworkDiscovery, setNetworkDiscovery]);

  // Cleanup device scan polling on unmount
  useEffect(
    () => (): void => {
      if (scanPollIntervalRef.current) {
        clearInterval(scanPollIntervalRef.current);
      }
      if (scanTimeoutRef.current) {
        clearTimeout(scanTimeoutRef.current);
      }
    },
    [],
  );

  return { triggerDeviceScan, scanError };
}
