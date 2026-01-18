/**
 * Device Scanning Hook
 *
 * Manages device scanning state and the trigger/polling logic
 * for network device discovery.
 *
 * Extracted from App.tsx to reduce component complexity (#889).
 */

import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { NetworkDiscoveryData } from "../components/cards/NetworkDiscoveryCard";
import { api } from "../api";
import { LogComponents, logger } from "../lib/logger";

interface UseDeviceScanningProps {
  /** Function to fetch full network discovery data after scan completes */
  fetchNetworkDiscovery: () => Promise<void>;
}

interface UseDeviceScanningReturn {
  /** Current network discovery data */
  networkDiscovery: NetworkDiscoveryData | null;
  /** Update network discovery data */
  setNetworkDiscovery: React.Dispatch<React.SetStateAction<NetworkDiscoveryData | null>>;
  /** Trigger a new device scan */
  triggerDeviceScan: () => Promise<void>;
  /** Abort controller ref for cleanup */
  networkDiscoveryAbortRef: React.MutableRefObject<AbortController | null>;
}

/**
 * Hook for managing device scanning with automatic polling for completion.
 *
 * @param props - Configuration with fetchNetworkDiscovery callback
 * @returns Object with scanning state and trigger function
 */
export function useDeviceScanning({
  fetchNetworkDiscovery,
}: UseDeviceScanningProps): UseDeviceScanningReturn {
  const [networkDiscovery, setNetworkDiscovery] = useState<NetworkDiscoveryData | null>(null);

  // Refs for polling cleanup
  const scanPollIntervalRef: React.MutableRefObject<ReturnType<typeof setInterval> | null> =
    useRef(null);
  const scanTimeoutRef: React.MutableRefObject<ReturnType<typeof setTimeout> | null> = useRef(null);
  const networkDiscoveryAbortRef: React.MutableRefObject<AbortController | null> = useRef(null);

  // Cleanup on unmount
  useEffect(
    (): (() => void) => (): void => {
      networkDiscoveryAbortRef.current?.abort();
      if (scanPollIntervalRef.current) {
        clearInterval(scanPollIntervalRef.current);
      }
      if (scanTimeoutRef.current) {
        clearTimeout(scanTimeoutRef.current);
      }
    },
    [],
  );

  // Trigger network device scan
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

      await api.post("/api/v1/shell/devices/scan");

      // Poll for completion
      scanPollIntervalRef.current = setInterval(async () => {
        try {
          const status = await api.get<{ scanning: boolean }>("/api/v1/shell/devices/status");
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
      logger.error(LogComponents.Devices, "Failed to trigger device scan", err);
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: false },
            }
          : null,
      );
    }
  }, [fetchNetworkDiscovery]);

  return {
    networkDiscovery,
    setNetworkDiscovery,
    triggerDeviceScan,
    networkDiscoveryAbortRef,
  };
}

export default useDeviceScanning;
