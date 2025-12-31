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

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";

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
export interface IpConfig {
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
export interface DnsResult {
  server: string;
  responseTime: number;
  status: "success" | "timeout" | "error";
  resolvedIp?: string;
}

/** VLAN information */
export interface VlanInfo {
  id: number;
  name: string;
  interface: string;
  tagged: boolean;
}

/** Public IP information */
export interface PublicIpInfo {
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
  ipConfig: IpConfig | null;
  gateway: GatewayInfo | null;
  dnsResults: DnsResult[];
  vlanInfo: VlanInfo[];
  publicIp: PublicIpInfo | null;
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
    publicIp: null,
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
        api.get<LinkStatus>("/api/sap/link"),
        api.get<{ interfaces: string[] }>("/api/interfaces"),
        api.get<{ interface: string }>("/api/interface"),
        api.get<IpConfig>("/api/sap/ipconfig"),
        api.get<GatewayInfo>("/api/sap/gateway"),
        api.get<{ results: DnsResult[] }>("/api/sap/dns"),
        api.get<{ vlans: VlanInfo[] }>("/api/sap/vlan"),
        api.get<PublicIpInfo>("/api/sap/publicip"),
        api.get<SystemHealth>("/api/sap/system/health"),
        api.get<{ neighbors: DiscoveryNeighbor[] }>("/api/shell/discovery"),
      ] as const;

      const [
        linkRes,
        interfacesRes,
        currentInterfaceRes,
        ipConfigRes,
        gatewayRes,
        dnsRes,
        vlanRes,
        publicIpRes,
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
        publicIp: isFulfilled(publicIpRes) ? publicIpRes.value : null,
        systemHealth: isFulfilled(healthRes) ? healthRes.value : null,
        discoveryNeighbors:
          isFulfilled(discoveryRes) && discoveryRes.value ? discoveryRes.value.neighbors || [] : [],
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch network data";
      setError(message);
      logger.error(LogComponents.Network, "Failed to refresh network data", err, {
        endpoints: [
          "/api/sap/link",
          "/api/interfaces",
          "/api/interface",
          "/api/sap/ipconfig",
          "/api/sap/gateway",
          "/api/sap/dns",
          "/api/sap/vlan",
          "/api/sap/publicip",
          "/api/sap/system/health",
          "/api/shell/discovery",
        ],
      });
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Fetches only link status (for quick updates).
   */
  const refreshLinkStatus = useCallback(async (): Promise<LinkStatus | null> => {
    try {
      const data = await api.get<LinkStatus>("/api/sap/link");
      setNetworkData((prev) => ({ ...prev, linkStatus: data }));
      return data;
    } catch (err) {
      logger.error(LogComponents.Network, "Failed to refresh link status", err, {
        endpoint: "/api/sap/link",
      });
      return null;
    }
  }, []);

  /**
   * Fetches only gateway info (for connectivity checks).
   */
  const refreshGateway = useCallback(async (): Promise<GatewayInfo | null> => {
    try {
      const data = await api.get<GatewayInfo>("/api/sap/gateway");
      setNetworkData((prev) => ({ ...prev, gateway: data }));
      return data;
    } catch (err) {
      logger.error(LogComponents.Gateway, "Failed to refresh gateway", err, {
        endpoint: "/api/sap/gateway",
      });
      return null;
    }
  }, []);

  /**
   * Fetches only public IP (force refresh).
   */
  const refreshPublicIp = useCallback(async (): Promise<PublicIpInfo | null> => {
    try {
      const data = await api.post<PublicIpInfo>("/api/sap/publicip");
      setNetworkData((prev) => ({ ...prev, publicIp: data }));
      return data;
    } catch (err) {
      logger.error(LogComponents.Publicip, "Failed to refresh public IP", err, {
        endpoint: "/api/sap/publicip",
      });
      return null;
    }
  }, []);

  // Set up auto-refresh interval
  // Fixes #971: Check mounted state before setting interval after initial refresh
  useEffect(() => {
    let isMounted = true;

    if (autoRefresh && refreshInterval > 0) {
      refresh().finally(() => {
        if (isMounted) {
          intervalRef.current = setInterval(refresh, refreshInterval);
        }
      });

      return () => {
        isMounted = false;
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
    refreshPublicIp,
  };
}
