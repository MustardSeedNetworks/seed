/**
 * Network Data Hook
 *
 * Consolidated hook for fetching various network status and configuration data.
 * Replaces multiple scattered API calls in App.tsx with a single, organized hook.
 *
 * Features:
 * - Fetch link status and interface information
 * - Fetch IP configuration and gateway info
 * - Fetch DNS test results
 * - Fetch VLAN information
 * - Fetch public IP
 * - Fetch system health status
 * - Automatic data refresh with configurable interval
 *
 * Usage:
 * ```typescript
 * const { networkData, isLoading, refresh } = useNetworkData();
 *
 * // Access specific data
 * const { linkStatus, ipConfig, gateway, dnsResults } = networkData;
 * ```
 */

import { useState, useCallback, useEffect, useRef } from "react";
import { api } from "../lib/api";
import { logger, LogComponents } from "../lib/logger";

function isFulfilled<T>(result: PromiseSettledResult<T>): result is PromiseFulfilledResult<T> {
  return result.status === "fulfilled";
}

/** Network interface link status */
export interface LinkStatus {
  interface: string;
  state: "up" | "down" | "unknown";
  speed: number;
  duplex: string;
  mtu: number;
}

/** IP configuration for an interface */
export interface IPConfig {
  interface: string;
  ipv4: string;
  ipv6: string;
  subnet: string;
  dhcp: boolean;
}

/** Gateway information */
export interface GatewayInfo {
  ip: string;
  latency: number;
  reachable: boolean;
  interface: string;
}

/** DNS test result */
export interface DNSResult {
  server: string;
  responseTime: number;
  status: "success" | "timeout" | "error";
  resolvedIP?: string;
}

/** VLAN information */
export interface VLANInfo {
  id: number;
  name: string;
  interface: string;
  tagged: boolean;
}

/** Public IP information */
export interface PublicIPInfo {
  ipv4: string;
  ipv6: string;
  location?: {
    city: string;
    country: string;
    isp: string;
  };
}

/** System health status */
export interface SystemHealth {
  status: "healthy" | "degraded" | "unhealthy";
  uptime: number;
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  services: Array<{
    name: string;
    status: "running" | "stopped" | "error";
  }>;
}

/** Discovery neighbors from LLDP/CDP/EDP */
export interface DiscoveryNeighbor {
  protocol: string;
  chassisId: string;
  portId: string;
  portDescription?: string;
  systemName?: string;
  systemDescription?: string;
  capabilities?: string[];
  managementAddress?: string;
  ttl: number;
}

/** Consolidated network data */
export interface NetworkData {
  linkStatus: LinkStatus | null;
  interfaces: string[];
  currentInterface: string;
  ipConfig: IPConfig | null;
  gateway: GatewayInfo | null;
  dnsResults: DNSResult[];
  vlanInfo: VLANInfo[];
  publicIP: PublicIPInfo | null;
  systemHealth: SystemHealth | null;
  discoveryNeighbors: DiscoveryNeighbor[];
}

/** Hook options */
interface UseNetworkDataOptions {
  /** Auto-refresh interval in milliseconds (0 = disabled) */
  refreshInterval?: number;
  /** Enable auto-refresh on mount */
  autoRefresh?: boolean;
}

/**
 * Custom hook for consolidated network data fetching.
 *
 * @param options - Configuration options
 * @returns Network data state and control functions
 */
export function useNetworkData(options: UseNetworkDataOptions = {}) {
  const { refreshInterval = 30000, autoRefresh = true } = options;

  const [networkData, setNetworkData] = useState<NetworkData>({
    linkStatus: null,
    interfaces: [],
    currentInterface: "",
    ipConfig: null,
    gateway: null,
    dnsResults: [],
    vlanInfo: [],
    publicIP: null,
    systemHealth: null,
    discoveryNeighbors: [],
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  /**
   * Fetches all network data in parallel.
   */
  const refresh = useCallback(async (): Promise<void> => {
    setIsLoading(true);
    setError(null);

    try {
      // Fetch all data in parallel for efficiency
      const promises = [
        api.get<LinkStatus>("/api/link"),
        api.get<{ interfaces: string[] }>("/api/interfaces"),
        api.get<{ interface: string }>("/api/interface"),
        api.get<IPConfig>("/api/ipconfig"),
        api.get<GatewayInfo>("/api/gateway"),
        api.get<{ results: DNSResult[] }>("/api/dns"),
        api.get<{ vlans: VLANInfo[] }>("/api/vlan"),
        api.get<PublicIPInfo>("/api/publicip"),
        api.get<SystemHealth>("/api/system/health"),
        api.get<{ neighbors: DiscoveryNeighbor[] }>("/api/discovery"),
      ] as const;

      const [
        linkRes,
        interfacesRes,
        currentInterfaceRes,
        ipConfigRes,
        gatewayRes,
        dnsRes,
        vlanRes,
        publicIPRes,
        healthRes,
        discoveryRes,
      ] = await Promise.allSettled(promises);

      // Extract data with proper type narrowing for PromiseSettledResult
      setNetworkData({
        linkStatus: isFulfilled(linkRes) ? linkRes.value : null,
        interfaces:
          isFulfilled(interfacesRes) && interfacesRes.value
            ? interfacesRes.value.interfaces || []
            : [],
        currentInterface:
          isFulfilled(currentInterfaceRes) && currentInterfaceRes.value
            ? currentInterfaceRes.value.interface || ""
            : "",
        ipConfig: isFulfilled(ipConfigRes) ? ipConfigRes.value : null,
        gateway: isFulfilled(gatewayRes) ? gatewayRes.value : null,
        dnsResults: isFulfilled(dnsRes) && dnsRes.value ? dnsRes.value.results || [] : [],
        vlanInfo: isFulfilled(vlanRes) && vlanRes.value ? vlanRes.value.vlans || [] : [],
        publicIP: isFulfilled(publicIPRes) ? publicIPRes.value : null,
        systemHealth: isFulfilled(healthRes) ? healthRes.value : null,
        discoveryNeighbors:
          isFulfilled(discoveryRes) && discoveryRes.value ? discoveryRes.value.neighbors || [] : [],
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch network data";
      setError(message);
      logger.error(LogComponents.NETWORK, "Failed to refresh network data", err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Fetches only link status (for quick updates).
   */
  const refreshLinkStatus = useCallback(async (): Promise<LinkStatus | null> => {
    try {
      const data = await api.get<LinkStatus>("/api/link");
      setNetworkData((prev) => ({ ...prev, linkStatus: data }));
      return data;
    } catch (err) {
      logger.error(LogComponents.NETWORK, "Failed to refresh link status", err);
      return null;
    }
  }, []);

  /**
   * Fetches only gateway info (for connectivity checks).
   */
  const refreshGateway = useCallback(async (): Promise<GatewayInfo | null> => {
    try {
      const data = await api.get<GatewayInfo>("/api/gateway");
      setNetworkData((prev) => ({ ...prev, gateway: data }));
      return data;
    } catch (err) {
      logger.error(LogComponents.GATEWAY, "Failed to refresh gateway", err);
      return null;
    }
  }, []);

  /**
   * Fetches only public IP (force refresh).
   */
  const refreshPublicIP = useCallback(async (): Promise<PublicIPInfo | null> => {
    try {
      const data = await api.post<PublicIPInfo>("/api/publicip");
      setNetworkData((prev) => ({ ...prev, publicIP: data }));
      return data;
    } catch (err) {
      logger.error(LogComponents.PUBLICIP, "Failed to refresh public IP", err);
      return null;
    }
  }, []);

  // Set up auto-refresh interval
  useEffect(() => {
    if (autoRefresh && refreshInterval > 0) {
      refresh(); // Initial fetch
      intervalRef.current = setInterval(refresh, refreshInterval);

      return () => {
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
        }
      };
    }
  }, [autoRefresh, refreshInterval, refresh]);

  return {
    // State
    networkData,
    isLoading,
    error,

    // Refresh operations
    refresh,
    refreshLinkStatus,
    refreshGateway,
    refreshPublicIP,
  };
}
