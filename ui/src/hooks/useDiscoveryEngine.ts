/**
 * useDiscoveryEngine Hook
 *
 * Provides access to the unified Discovery Engine for real-time device discovery
 * across wired, WiFi, and Bluetooth networks.
 *
 * Features:
 * - Fetches all discovered devices from the Engine registry
 * - Triggers quick and full scans
 * - Provides real-time updates via SSE
 * - Engine statistics and capabilities
 *
 * Usage:
 * ```typescript
 * const { devices, stats, capabilities, scan, quickScan, fullScan, isLoading, error } = useDiscoveryEngine();
 * ```
 */

import { useCallback, useEffect, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";

const log: ReturnType<typeof logger> = logger(LogComponents.API);

/** Device from Discovery Engine */
export interface EngineDevice {
  mac: string;
  ip: string;
  hostname?: string;
  vendor?: string;
  deviceType?: string;
  connectionTypes?: string[];
  discoveryMethod?: string[];
  isOnline?: boolean;
  firstSeen?: string;
  lastSeen?: string;
  wifiPresence?: {
    ssid?: string;
    channel?: number;
    signalDbm?: number;
    isAccessPoint?: boolean;
    isAuthorized?: boolean;
  };
  bluetoothPresence?: {
    name?: string;
    type?: string;
    rssi?: number;
    isPaired?: boolean;
    isAuthorized?: boolean;
  };
}

/** Engine statistics */
export interface EngineStats {
  running: boolean;
  deviceCount: number;
  wiredCount: number;
  wifiCount: number;
  bluetoothCount: number;
  multiConnectedCount: number;
  scanCount: number;
  lastScanTime?: string;
  lastScanDuration?: number;
}

/** Engine capabilities */
export interface EngineCapabilities {
  wired: boolean;
  wifi: boolean;
  bluetooth: boolean;
  snmp: boolean;
  portScan: boolean;
  profiling: boolean;
  vulnScan: boolean;
  correlation: boolean;
}

/** Scan result from Engine */
export interface EngineScanResult {
  scanType: string;
  duration: number;
  deviceCount: number;
  phases: string[];
  startTime: string;
  endTime: string;
}

/** Scan options */
export interface EngineScanOptions {
  scanType?: "quick" | "full";
  includeWired?: boolean;
  includeWifi?: boolean;
  includeBluetooth?: boolean;
  includeSnmp?: boolean;
  includePortScan?: boolean;
  includeVulnScan?: boolean;
  freshWiredScan?: boolean;
  freshWifiScan?: boolean;
  freshBluetoothScan?: boolean;
}

/** API response from Engine endpoints */
interface EngineResponse {
  devices: EngineDevice[];
  stats: EngineStats;
  scanResult?: EngineScanResult;
  capabilities: EngineCapabilities;
}

/** Hook state */
interface UseDiscoveryEngineState {
  devices: EngineDevice[];
  stats: EngineStats | null;
  capabilities: EngineCapabilities | null;
  lastScanResult: EngineScanResult | null;
  isLoading: boolean;
  isScanning: boolean;
  error: string | null;
}

const initialState: UseDiscoveryEngineState = {
  devices: [],
  stats: null,
  capabilities: null,
  lastScanResult: null,
  isLoading: false,
  isScanning: false,
  error: null,
};

/**
 * Hook for interacting with the Discovery Engine.
 */
export function useDiscoveryEngine(): {
  devices: EngineDevice[];
  stats: EngineStats | null;
  capabilities: EngineCapabilities | null;
  lastScanResult: EngineScanResult | null;
  isLoading: boolean;
  isScanning: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  scan: (options?: EngineScanOptions) => Promise<EngineScanResult | undefined>;
  quickScan: () => Promise<EngineScanResult | undefined>;
  fullScan: () => Promise<EngineScanResult | undefined>;
  getDevice: (mac: string) => Promise<EngineDevice | null>;
  fetchCapabilities: () => Promise<EngineCapabilities | null>;
  fetchStats: () => Promise<EngineStats | null>;
} {
  const [state, setState] = useState<UseDiscoveryEngineState>(initialState);

  /**
   * Fetch all devices from the Engine.
   */
  const fetchDevices = useCallback(async () => {
    setState((prev) => ({ ...prev, isLoading: true, error: null }));

    try {
      const response = await api.get<EngineResponse>("/api/v1/discovery/engine");
      setState((prev) => ({
        ...prev,
        devices: response.devices || [],
        stats: response.stats,
        capabilities: response.capabilities,
        lastScanResult: response.scanResult || prev.lastScanResult,
        isLoading: false,
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch devices";
      log.error("Failed to fetch devices from Discovery Engine", err);
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error: message,
      }));
    }
  }, []);

  /**
   * Trigger a scan with options.
   */
  const scan = useCallback(async (options?: EngineScanOptions) => {
    setState((prev) => ({ ...prev, isScanning: true, error: null }));

    try {
      const response = await api.post<EngineResponse>(
        "/api/v1/discovery/engine/scan",
        options || {},
      );
      setState((prev) => ({
        ...prev,
        devices: response.devices || [],
        stats: response.stats,
        capabilities: response.capabilities,
        lastScanResult: response.scanResult || null,
        isScanning: false,
      }));
      return response.scanResult;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Scan failed";
      log.error("Discovery Engine scan failed", err);
      setState((prev) => ({
        ...prev,
        isScanning: false,
        error: message,
      }));
      throw err;
    }
  }, []);

  /**
   * Trigger a quick scan (uses cached data, fast correlation).
   */
  const quickScan = useCallback(async () => {
    setState((prev) => ({ ...prev, isScanning: true, error: null }));

    try {
      const response = await api.post<EngineResponse>("/api/v1/discovery/engine/quick");
      setState((prev) => ({
        ...prev,
        devices: response.devices || [],
        stats: response.stats,
        capabilities: response.capabilities,
        lastScanResult: response.scanResult || null,
        isScanning: false,
      }));
      return response.scanResult;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Quick scan failed";
      log.error("Discovery Engine quick scan failed", err);
      setState((prev) => ({
        ...prev,
        isScanning: false,
        error: message,
      }));
      throw err;
    }
  }, []);

  /**
   * Trigger a full scan (fresh discovery + enrichment + assessment).
   */
  const fullScan = useCallback(async () => {
    setState((prev) => ({ ...prev, isScanning: true, error: null }));

    try {
      const response = await api.post<EngineResponse>("/api/v1/discovery/engine/full");
      setState((prev) => ({
        ...prev,
        devices: response.devices || [],
        stats: response.stats,
        capabilities: response.capabilities,
        lastScanResult: response.scanResult || null,
        isScanning: false,
      }));
      return response.scanResult;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Full scan failed";
      log.error("Discovery Engine full scan failed", err);
      setState((prev) => ({
        ...prev,
        isScanning: false,
        error: message,
      }));
      throw err;
    }
  }, []);

  /**
   * Get a specific device by MAC address.
   */
  const getDevice = useCallback(async (mac: string): Promise<EngineDevice | null> => {
    try {
      const device = await api.get<EngineDevice>(
        `/api/v1/discovery/engine/device/${encodeURIComponent(mac)}`,
      );
      return device;
    } catch (err) {
      log.error(`Failed to get device ${mac}`, err);
      return null;
    }
  }, []);

  /**
   * Fetch Engine capabilities.
   */
  const fetchCapabilities = useCallback(async () => {
    try {
      const caps = await api.get<EngineCapabilities>("/api/v1/discovery/engine/capabilities");
      setState((prev) => ({ ...prev, capabilities: caps }));
      return caps;
    } catch (err) {
      log.error("Failed to fetch Engine capabilities", err);
      return null;
    }
  }, []);

  /**
   * Fetch Engine statistics.
   */
  const fetchStats = useCallback(async () => {
    try {
      const stats = await api.get<EngineStats>("/api/v1/discovery/engine/stats");
      setState((prev) => ({ ...prev, stats }));
      return stats;
    } catch (err) {
      log.error("Failed to fetch Engine stats", err);
      return null;
    }
  }, []);

  // Initial fetch on mount
  useEffect(() => {
    fetchDevices().catch(() => undefined);
  }, [fetchDevices]);

  return {
    // State
    devices: state.devices,
    stats: state.stats,
    capabilities: state.capabilities,
    lastScanResult: state.lastScanResult,
    isLoading: state.isLoading,
    isScanning: state.isScanning,
    error: state.error,

    // Actions
    refresh: fetchDevices,
    scan,
    quickScan,
    fullScan,
    getDevice,
    fetchCapabilities,
    fetchStats,
  };
}

export default useDiscoveryEngine;
