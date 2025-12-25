/**
 * useDefaults Hook
 *
 * Fetches and caches default settings from the backend.
 * The backend is the single source of truth for all default values.
 */

import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "../lib/api";
import { logger, LogComponents } from "../lib/logger";
import { DefaultSettings } from "../types/defaults";

// ============================================================================
// Fallback Defaults (used only if API fails)
// ============================================================================

/**
 * These fallback defaults are ONLY used if the API call fails.
 * The backend (/api/settings/defaults) is the single source of truth.
 */
const FALLBACK_DEFAULTS: DefaultSettings = {
  cardSettings: {
    link: { enabled: true, autoRunOnLink: true },
    switch: { enabled: true, autoRunOnLink: true },
    vlan: { enabled: true, autoRunOnLink: true },
    network: { enabled: true, autoRunOnLink: true },
    gateway: { enabled: true, autoRunOnLink: true },
    dns: { enabled: true, autoRunOnLink: true },
    healthChecks: { enabled: true, autoRunOnLink: true },
    networkDiscovery: { enabled: true, autoRunOnLink: true },
    performance: {
      enabled: true,
      autoRunOnLink: true,
      speedtest: { enabled: true, autoRunOnLink: true },
      iperf: { enabled: false, autoRunOnLink: false },
    },
  },
  displayOptions: {
    showPublicIP: true,
    unitSystem: "sae",
  },
  thresholds: {
    dns: { good: 50, warning: 100 },
    gateway: { good: 20, warning: 50 },
    wifi: { good: -50, warning: -70 },
    customPing: { good: 50, warning: 100 },
    customTcp: { good: 100, warning: 200 },
    customHttp: { good: 500, warning: 1000 },
    httpTimings: {
      dns: { good: 50, warning: 100 },
      tcp: { good: 50, warning: 100 },
      tls: { good: 100, warning: 200 },
      ttfb: { good: 200, warning: 500 },
    },
  },
  iperf: {
    server: "",
    port: 5201,
    protocol: "tcp",
    direction: "download",
    duration: 10,
    serverPort: 5201,
    enableServer: true,
    autoRunOnLink: false,
  },
  tests: {
    dnsHostname: "google.com",
    pingTargets: [
      { id: "default-google-dns", name: "Google DNS", host: "8.8.8.8", enabled: true, count: 3 },
      { id: "default-cloudflare", name: "Cloudflare", host: "1.1.1.1", enabled: true, count: 3 },
    ],
    httpEndpoints: [
      { id: "default-google", name: "Google", url: "https://www.google.com", expectedStatus: 200, enabled: true },
    ],
    runPerformance: true,
    runSpeedtest: true,
    runIperf: false,
    runDiscovery: true,
    speedtest: {
      serverId: "",
      autoRunOnLink: true,
    },
  },
  networkDiscovery: {
    enabled: true,
    arpScanWorkers: 50,
    pingTimeoutMs: 500,
    scanTimeoutMs: 30000,
    autoScan: true,
    scanIntervalMs: 600000,
    ouiFilePath: "",
    ipv6Enabled: true,
    options: {
      passiveProtocols: { lldp: true, cdp: true, edp: true, ndp: true },
      arpScan: true,
      icmpScan: true,
      portScan: {
        enabled: false,
        preset: "common",
        tcpPorts: "22,80,443,8080-8100",
        udpPorts: "53,123,161",
        bannerTimeoutMs: 2000,
      },
      tcpProbe: { timeoutMs: 2000, workers: 20 },
      traceroute: false,
      snmpQuery: false,
    },
    timing: {
      probeIntervalMs: 75,
      rescanIntervalMs: 600000,
      workers: 50,
    },
    profiler: {
      enabled: true,
      timeoutMs: 2000,
      maxConcurrent: 5,
      quickPorts: [22, 80, 443, 8080],
    },
    fingerprinting: {
      enabled: false,
      osDetection: false,
      serviceProbes: false,
    },
  },
  snmp: {
    communities: ["public"],
    timeoutMs: 5000,
    retries: 2,
    port: 161,
  },
  link: {
    autoNegotiation: true,
    speed: "auto",
    duplex: "auto",
    availableModes: [],
  },
  cableTest: {
    enabled: true,
    autoRunOnLinkDown: false,
  },
  vulnerability: {
    enabled: true,
    cveDatabase: "nvd",
    nvdApiKey: "",
    updateInterval: 86400,
    severityThreshold: "medium",
    maxConcurrent: 5,
    autoScan: true,
  },
};

// ============================================================================
// Cache for defaults (shared across all hook instances)
// ============================================================================

let cachedDefaults: DefaultSettings | null = null;
let fetchPromise: Promise<DefaultSettings> | null = null;

// ============================================================================
// Hook Interface
// ============================================================================

interface UseDefaultsResult {
  defaults: DefaultSettings;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

// ============================================================================
// Hook Implementation
// ============================================================================

/**
 * Hook to fetch and cache default settings from the backend.
 * Uses a shared cache to avoid redundant API calls across components.
 */
export function useDefaults(): UseDefaultsResult {
  const [defaults, setDefaults] = useState<DefaultSettings>(
    cachedDefaults ?? FALLBACK_DEFAULTS
  );
  const [isLoading, setIsLoading] = useState(!cachedDefaults);
  const [error, setError] = useState<Error | null>(null);
  const isMountedRef = useRef(true);

  const fetchDefaults = useCallback(async () => {
    // If already cached, use cached value
    if (cachedDefaults) {
      setDefaults(cachedDefaults);
      setIsLoading(false);
      return;
    }

    // If already fetching, wait for that promise
    if (fetchPromise) {
      try {
        const result = await fetchPromise;
        if (isMountedRef.current) {
          setDefaults(result);
          setIsLoading(false);
        }
      } catch (err) {
        if (isMountedRef.current) {
          setError(err instanceof Error ? err : new Error(String(err)));
          setIsLoading(false);
        }
      }
      return;
    }

    // Start new fetch
    setIsLoading(true);
    setError(null);

    fetchPromise = api.get<DefaultSettings>("/api/settings/defaults");

    try {
      const result = await fetchPromise;
      cachedDefaults = result;
      if (isMountedRef.current) {
        setDefaults(result);
        setIsLoading(false);
      }
    } catch (err) {
      logger.warn(LogComponents.CONFIG, "Failed to fetch defaults, using fallback", err);
      if (isMountedRef.current) {
        setError(err instanceof Error ? err : new Error(String(err)));
        setDefaults(FALLBACK_DEFAULTS);
        setIsLoading(false);
      }
    } finally {
      fetchPromise = null;
    }
  }, []);

  const refetch = useCallback(async () => {
    cachedDefaults = null;
    fetchPromise = null;
    await fetchDefaults();
  }, [fetchDefaults]);

  useEffect(() => {
    isMountedRef.current = true;
    fetchDefaults();
    return () => {
      isMountedRef.current = false;
    };
  }, [fetchDefaults]);

  return { defaults, isLoading, error, refetch };
}

/**
 * Get cached defaults synchronously.
 * Returns fallback if not yet loaded.
 */
export function getDefaultsSync(): DefaultSettings {
  return cachedDefaults ?? FALLBACK_DEFAULTS;
}

/**
 * Clear the defaults cache (useful for testing).
 */
export function clearDefaultsCache(): void {
  cachedDefaults = null;
  fetchPromise = null;
}
