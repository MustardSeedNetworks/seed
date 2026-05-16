/**
 * useDeviceScan - triggers and polls a network device scan.
 *
 * Owns the scan poll interval and timeout refs and tears them down on
 * unmount. Returns a stable trigger callback for App.tsx and other
 * callers to kick off a /api/v1/shell/devices/scan run.
 */

import { useCallback, useEffect, useRef } from 'react';
import { api } from '../api';
import type { NetworkDiscoveryData } from '../components/cards/NetworkDiscoveryCard';
import { LogComponents, logger } from '../lib/logger';

interface UseDeviceScanArgs {
  fetchNetworkDiscovery: () => Promise<void>;
  setNetworkDiscovery: (
    updater: (prev: NetworkDiscoveryData | null) => NetworkDiscoveryData | null,
  ) => void;
}

export function useDeviceScan({
  fetchNetworkDiscovery,
  setNetworkDiscovery,
}: UseDeviceScanArgs): () => Promise<void> {
  const scanPollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const scanTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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

      // Update status to show scanning
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: true },
            }
          : null,
      );

      await api.post('/api/v1/shell/devices/scan');

      // Poll for completion
      scanPollIntervalRef.current = setInterval(async () => {
        try {
          const status = await api.get<{ scanning: boolean }>('/api/v1/shell/devices/status');
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
      logger.error(LogComponents.Devices, 'Failed to trigger device scan', err);
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: false },
            }
          : null,
      );
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

  return triggerDeviceScan;
}
