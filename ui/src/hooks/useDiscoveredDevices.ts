// biome-ignore-all lint/style/noInferrableTypes: useExplicitType requires types on default params
/**
 * useDiscoveredDevices Hook
 *
 * Fetches and manages discovered network devices for use in the DeviceSelector.
 * This hook provides filtered and grouped device data from the network discovery API.
 *
 * Features:
 * - Fetches discovered devices from the API
 * - Groups devices by type (Router, Switch, Workstation, etc.)
 * - Provides loading and error states
 * - Auto-refresh support
 *
 * Usage:
 * ```typescript
 * const { devices, groupedDevices, isLoading, error, refresh } = useDiscoveredDevices();
 * ```
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";

/** Network device from discovery API with display fields */
export interface DiscoveredDevice {
  ip: string;
  mac: string;
  hostname?: string;
  vendor?: string;
  displayName?: string;
  deviceType?: string;
  isRouter?: boolean;
  lastSeen: string;
  profile?: {
    deviceType?: string;
  };
}

/** Device discovery status */
export interface DiscoveryStatus {
  scanning: boolean;
  deviceCount: number;
  lastScan: string;
}

/** API response from /api/discovery */
interface DiscoveryResponse {
  devices: DiscoveredDevice[];
  status: DiscoveryStatus;
}

/** Grouped devices by type */
export interface GroupedDevices {
  routers: DiscoveredDevice[];
  switches: DiscoveredDevice[];
  servers: DiscoveredDevice[];
  workstations: DiscoveredDevice[];
  printers: DiscoveredDevice[];
  phones: DiscoveredDevice[];
  other: DiscoveredDevice[];
}

/**
 * Categorizes a device based on its profile and metadata.
 * Returns one of: router, switch, server, workstation, printer, phone, other
 */
function categorizeDevice(device: DiscoveredDevice): string {
  const deviceType = device.profile?.deviceType?.toLowerCase() || "";

  // Check router first
  if (device.isRouter || deviceType.includes("router")) {
    return "router";
  }

  // Check for switch
  if (deviceType.includes("switch")) {
    return "switch";
  }

  // Check for printer
  if (deviceType.includes("printer")) {
    return "printer";
  }

  // Check for server
  if (
    deviceType.includes("server") ||
    deviceType.includes("nas") ||
    deviceType.includes("storage")
  ) {
    return "server";
  }

  // Check for phone/mobile
  if (
    deviceType.includes("phone") ||
    deviceType.includes("mobile") ||
    deviceType.includes("smartphone")
  ) {
    return "phone";
  }

  // Check for workstation/computer
  if (
    deviceType.includes("computer") ||
    deviceType.includes("desktop") ||
    deviceType.includes("laptop") ||
    deviceType.includes("workstation") ||
    deviceType.includes("pc")
  ) {
    return "workstation";
  }

  return "other";
}

/**
 * Groups devices by their categorized type.
 */
function groupDevicesByType(devices: DiscoveredDevice[]): GroupedDevices {
  const grouped: GroupedDevices = {
    routers: [],
    switches: [],
    servers: [],
    workstations: [],
    printers: [],
    phones: [],
    other: [],
  };

  for (const device of devices) {
    const category = categorizeDevice(device);
    switch (category) {
      case "router":
        grouped.routers.push(device);
        break;
      case "switch":
        grouped.switches.push(device);
        break;
      case "server":
        grouped.servers.push(device);
        break;
      case "workstation":
        grouped.workstations.push(device);
        break;
      case "printer":
        grouped.printers.push(device);
        break;
      case "phone":
        grouped.phones.push(device);
        break;
      default:
        grouped.other.push(device);
    }
  }

  return grouped;
}

/**
 * Gets a display name for a device, prioritizing hostname over IP.
 */
export function getDeviceDisplayName(device: DiscoveredDevice): string {
  if (device.displayName) {
    return device.displayName;
  }
  if (device.hostname) {
    return device.hostname;
  }
  return device.ip;
}

/**
 * Custom hook for fetching and managing discovered network devices.
 */
export function useDiscoveredDevices(autoRefresh: boolean = false): {
  devices: DiscoveredDevice[];
  groupedDevices: GroupedDevices;
  status: DiscoveryStatus | null;
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
} {
  const [devices, setDevices] = useState<DiscoveredDevice[]>([]);
  const [status, setStatus] = useState<DiscoveryStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  /**
   * Fetches discovered devices from the API.
   */
  const fetchDevices = useCallback(async (): Promise<void> => {
    try {
      setError(null);
      // Use /api/devices endpoint which returns discovered network devices
      // (not /api/discovery which returns LLDP/CDP protocol neighbors)
      const data = await api.get<DiscoveryResponse>("/api/v1/shell/devices");
      setDevices(data.devices || []);
      setStatus(data.status);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch discovered devices";
      setError(message);
      logger.error(LogComponents.Devices, "Failed to fetch discovered devices", err, {
        endpoint: "/api/v1/shell/devices",
      });
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Manually refresh the device list.
   */
  const refresh = useCallback(async () => {
    setIsLoading(true);
    await fetchDevices();
  }, [fetchDevices]);

  // Initial fetch
  useEffect(() => {
    fetchDevices().catch(() => undefined);
  }, [fetchDevices]);

  // Auto-refresh if enabled
  useEffect(() => {
    if (!autoRefresh) {
      return;
    }

    const interval: ReturnType<typeof setInterval> = setInterval(() => {
      fetchDevices().catch(() => undefined);
    }, 10000); // Refresh every 10 seconds

    return (): void => clearInterval(interval);
  }, [autoRefresh, fetchDevices]);

  // Group devices by type
  const groupedDevices = useMemo(() => groupDevicesByType(devices), [devices]);

  return {
    devices,
    groupedDevices,
    status,
    isLoading,
    error,
    refresh,
  };
}
