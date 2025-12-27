/**
 * Device Discovery Hook
 *
 * Manages network device discovery operations including scanning, status tracking,
 * and device data management.
 *
 * Features:
 * - Trigger network device scans
 * - Fetch discovered devices and scan status
 * - Manage device discovery settings
 * - Handle subnet configuration
 *
 * Usage:
 * ```typescript
 * const { devices, isScanning, triggerScan, fetchDevices } = useDevices();
 *
 * // Trigger a new scan
 * await triggerScan();
 *
 * // Get all discovered devices
 * const devices = await fetchDevices();
 * ```
 */

import { useState, useCallback } from "react";
import { api } from "../lib/api";
import { logger, LogComponents } from "../lib/logger";

/** Network device discovered by the scanner */
export interface Device {
  ip: string;
  mac: string;
  hostname: string;
  vendor: string;
  discoveryMethod: string;
  firstSeen: string;
  lastSeen: string;
  openPorts?: number[];
  osInfo?: {
    name: string;
    version: string;
    vendor: string;
  };
}

/** Status of the device discovery scanner */
export interface DeviceDiscoveryStatus {
  scanning: boolean;
  lastScan: string;
  deviceCount: number;
  progress?: number;
}

/** Device discovery settings */
export interface DeviceDiscoverySettings {
  enabled: boolean;
  arpScanWorkers: number;
  pingTimeoutMs: number;
  scanTimeoutMs: number;
  autoScan: boolean;
  scanIntervalMs: number;
  // Note: OUI database is baked into binary at build time
}

/** Subnet configuration for scanning */
export interface SubnetConfig {
  cidr: string;
  name: string;
  enabled: boolean;
}

/** API response for devices endpoint */
interface DevicesResponse {
  devices: Device[];
  status: DeviceDiscoveryStatus;
}

/** API response for scan trigger */
interface ScanResponse {
  message: string;
  scanning: boolean;
}

/**
 * Custom hook for managing device discovery operations.
 *
 * Provides functions to scan networks, fetch devices, and manage settings.
 *
 * @returns Device discovery state and control functions
 */
export function useDevices() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [status, setStatus] = useState<DeviceDiscoveryStatus | null>(null);
  const [isScanning, setIsScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Fetches all discovered devices and current status.
   */
  const fetchDevices = useCallback(async (): Promise<Device[]> => {
    try {
      setError(null);
      const data = await api.get<DevicesResponse>("/api/devices");
      setDevices(data.devices || []);
      setStatus(data.status);
      setIsScanning(data.status?.scanning || false);
      return data.devices || [];
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to fetch devices";
      setError(message);
      logger.error(LogComponents.DEVICES, "Failed to fetch devices", err, {
        endpoint: "/api/devices",
      });
      return [];
    }
  }, []);

  /**
   * Triggers a network device scan.
   */
  const triggerScan = useCallback(async (): Promise<boolean> => {
    try {
      setError(null);
      setIsScanning(true);
      const data = await api.post<ScanResponse>("/api/devices/scan");
      setIsScanning(Boolean(data.scanning));
      return data.scanning;
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to start scan";
      setError(message);
      setIsScanning(false);
      logger.error(LogComponents.DEVICES, "Failed to trigger scan", err, {
        endpoint: "/api/devices/scan",
      });
      return false;
    }
  }, []);

  /**
   * Fetches the current scan status.
   */
  const fetchStatus =
    useCallback(async (): Promise<DeviceDiscoveryStatus | null> => {
      try {
        const data = await api.get<DeviceDiscoveryStatus>(
          "/api/devices/status"
        );
        setStatus(data);
        setIsScanning(data.scanning);
        return data;
      } catch (err) {
        logger.error(LogComponents.DEVICES, "Failed to fetch status", err, {
          endpoint: "/api/devices/status",
        });
        return null;
      }
    }, []);

  /**
   * Fetches device discovery settings.
   */
  const fetchSettings =
    useCallback(async (): Promise<DeviceDiscoverySettings | null> => {
      try {
        return await api.get<DeviceDiscoverySettings>("/api/devices/settings");
      } catch (err) {
        logger.error(LogComponents.DEVICES, "Failed to fetch settings", err, {
          endpoint: "/api/devices/settings",
        });
        return null;
      }
    }, []);

  /**
   * Updates device discovery settings.
   */
  const updateSettings = useCallback(
    async (settings: Partial<DeviceDiscoverySettings>): Promise<boolean> => {
      try {
        await api.put("/api/devices/settings", settings);
        return true;
      } catch (err) {
        logger.error(LogComponents.DEVICES, "Failed to update settings", err, {
          endpoint: "/api/devices/settings",
          updates: settings,
        });
        return false;
      }
    },
    []
  );

  /**
   * Fetches configured subnets for scanning.
   */
  const fetchSubnets = useCallback(async (): Promise<SubnetConfig[]> => {
    try {
      return await api.get<SubnetConfig[]>("/api/devices/subnets");
    } catch (err) {
      logger.error(LogComponents.DEVICES, "Failed to fetch subnets", err, {
        endpoint: "/api/devices/subnets",
      });
      return [];
    }
  }, []);

  /**
   * Adds a new subnet for scanning.
   */
  const addSubnet = useCallback(
    async (subnet: SubnetConfig): Promise<boolean> => {
      try {
        await api.post("/api/devices/subnets", subnet);
        return true;
      } catch (err) {
        logger.error(LogComponents.DEVICES, "Failed to add subnet", err, {
          endpoint: "/api/devices/subnets",
          cidr: subnet.cidr,
        });
        return false;
      }
    },
    []
  );

  /**
   * Updates an existing subnet configuration.
   */
  const updateSubnet = useCallback(
    async (subnet: SubnetConfig): Promise<boolean> => {
      try {
        await api.put("/api/devices/subnets", subnet);
        return true;
      } catch (err) {
        logger.error(LogComponents.DEVICES, "Failed to update subnet", err, {
          endpoint: "/api/devices/subnets",
          cidr: subnet.cidr,
        });
        return false;
      }
    },
    []
  );

  /**
   * Deletes a subnet from scanning configuration.
   */
  const deleteSubnet = useCallback(async (cidr: string): Promise<boolean> => {
    try {
      await api.delete(`/api/devices/subnets?cidr=${encodeURIComponent(cidr)}`);
      return true;
    } catch (err) {
      logger.error(LogComponents.DEVICES, "Failed to delete subnet", err, {
        endpoint: "/api/devices/subnets",
        cidr,
      });
      return false;
    }
  }, []);

  return {
    // State
    devices,
    status,
    isScanning,
    error,

    // Device operations
    fetchDevices,
    triggerScan,
    fetchStatus,

    // Settings operations
    fetchSettings,
    updateSettings,

    // Subnet operations
    fetchSubnets,
    addSubnet,
    updateSubnet,
    deleteSubnet,
  };
}
