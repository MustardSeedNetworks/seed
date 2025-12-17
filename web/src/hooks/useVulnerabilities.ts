/**
 * Vulnerability Scanner Hook
 *
 * Manages vulnerability scanning for network devices using the NVD (National Vulnerability Database).
 *
 * Features:
 * - Trigger vulnerability scans for all devices or specific IPs
 * - Fetch scan status and progress
 * - Retrieve vulnerability results with optional severity filtering
 * - Configure scanner settings (API keys, database cache)
 *
 * The scanner:
 * - Fingerprints devices from network discovery
 * - Looks up CVEs from NVD database based on device OS/vendor/version
 * - Caches results for performance
 * - Supports filtering by severity (critical, high, medium, low)
 *
 * Usage:
 * ```typescript
 * const { triggerScan, fetchResults, isScanning } = useVulnerabilities();
 *
 * // Scan all discovered devices
 * await triggerScan();
 *
 * // Scan specific device
 * await triggerScan('192.168.1.100');
 *
 * // Get results for critical vulnerabilities
 * const criticalVulns = await fetchResults('critical');
 * ```
 */

import { useState, useCallback } from "react";
import { logger, LogComponents } from "../lib/logger";
import { api } from "../lib/api";
import type {
  DeviceVulnerabilities,
  VulnerabilityScannerStatus,
  VulnerabilityScannerConfig,
} from "../types/vulnerabilities";

type SeverityFilter = "critical" | "high" | "medium" | "low";

/** API response for scan initiation */
interface ScanResponse {
  status: string; // "scan started" on success
  running?: boolean;
}

/** API response for vulnerability results */
interface ResultsResponse {
  results: DeviceVulnerabilities[]; // Array of device vulnerability reports
  count: number; // Total number of results
}

function isValidIPv4(ip: string): boolean {
  const parts = ip.split(".");
  if (parts.length !== 4) return false;
  return parts.every((part) => {
    if (!/^\d{1,3}$/.test(part)) return false;
    const value = Number(part);
    return value >= 0 && value <= 255;
  });
}

function isValidIPv6(ip: string): boolean {
  if (ip === "") return false;

  const [head, ...rest] = ip.split("::");
  if (rest.length > 1) return false;

  const headParts = head ? head.split(":") : [];
  const tailParts = rest.length === 1 && rest[0] ? rest[0].split(":") : [];
  const hasCompression = rest.length === 1;

  const allParts = hasCompression ? [...headParts, ...tailParts] : headParts;
  if (allParts.some((p) => p === "")) return false;

  const lastPart = allParts.at(-1);
  const hasIPv4Tail = lastPart ? lastPart.includes(".") : false;

  const validateHextet = (part: string): boolean => /^[0-9a-fA-F]{1,4}$/.test(part);

  if (hasIPv4Tail) {
    if (!isValidIPv4(lastPart!)) return false;
    const hextets = allParts.slice(0, -1);
    if (!hextets.every(validateHextet)) return false;

    if (hasCompression) {
      return hextets.length <= 6;
    }
    return hextets.length === 6;
  }

  if (!allParts.every(validateHextet)) return false;

  if (hasCompression) {
    return allParts.length < 8;
  }
  return allParts.length === 8;
}

function isValidIP(ip: string): boolean {
  return isValidIPv4(ip) || isValidIPv6(ip);
}

function normalizeSeverityFilter(severity: string): SeverityFilter | null {
  const normalized = severity.trim().toLowerCase();
  switch (normalized) {
    case "critical":
    case "high":
    case "medium":
    case "low":
      return normalized;
    default:
      return null;
  }
}

/**
 * Custom hook for managing vulnerability scanning operations.
 *
 * Provides functions to trigger scans, check status, and retrieve results.
 *
 * @returns Vulnerability scanning state and control functions
 */
export function useVulnerabilities() {
  const [isScanning, setIsScanning] = useState(false);
  const [scanError, setScanError] = useState<string | null>(null);

  const triggerScan = useCallback(async (ip?: string): Promise<boolean> => {
    setIsScanning(true);
    setScanError(null);

    try {
      const params = new URLSearchParams();
      if (ip) {
        const trimmed = ip.trim();
        if (!isValidIP(trimmed)) {
          throw new Error("Invalid IP address");
        }
        params.set("ip", trimmed);
      }

      const endpoint = params.size
        ? `/api/vulnerabilities/scan?${params.toString()}`
        : "/api/vulnerabilities/scan";

      const data = await api.post<ScanResponse>(endpoint);
      return data.status === "scan started" || data.status === "scan already in progress";
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown error";
      setScanError(message);
      return false;
    } finally {
      setIsScanning(false);
    }
  }, []);

  const fetchStatus = useCallback(async (): Promise<VulnerabilityScannerStatus | null> => {
    try {
      return await api.get<VulnerabilityScannerStatus>("/api/vulnerabilities/status");
    } catch (error) {
      logger.error(LogComponents.VULN, "Failed to fetch vulnerability status", error);
      return null;
    }
  }, []);

  const fetchResults = useCallback(async (severity?: string): Promise<DeviceVulnerabilities[]> => {
    try {
      const params = new URLSearchParams();
      if (severity) {
        const validSeverity = normalizeSeverityFilter(severity);
        if (!validSeverity) {
          throw new Error("Invalid severity filter");
        }
        params.set("severity", validSeverity);
      }

      const endpoint = params.size
        ? `/api/vulnerabilities/results?${params.toString()}`
        : "/api/vulnerabilities/results";
      const data = await api.get<ResultsResponse>(endpoint);
      return data.results || [];
    } catch (error) {
      logger.error(LogComponents.VULN, "Failed to fetch vulnerability results", error);
      return [];
    }
  }, []);

  const fetchDeviceVulnerabilities = useCallback(
    async (ip: string): Promise<DeviceVulnerabilities | null> => {
      try {
        const trimmed = ip.trim();
        if (!isValidIP(trimmed)) {
          throw new Error("Invalid IP address");
        }

        const params = new URLSearchParams({ ip: trimmed });
        return await api.get<DeviceVulnerabilities>(
          `/api/vulnerabilities/device?${params.toString()}`
        );
      } catch (error) {
        logger.error(LogComponents.VULN, "Failed to fetch vulnerabilities for device", error, {
          ip,
        });
        return null;
      }
    },
    []
  );

  const fetchSettings = useCallback(async (): Promise<VulnerabilityScannerConfig | null> => {
    try {
      return await api.get<VulnerabilityScannerConfig>("/api/vulnerabilities/settings");
    } catch (error) {
      logger.error(LogComponents.VULN, "Failed to fetch vulnerability settings", error);
      return null;
    }
  }, []);

  const updateSettings = useCallback(
    async (settings: Partial<VulnerabilityScannerConfig>): Promise<boolean> => {
      try {
        await api.put<{ status: string }>("/api/vulnerabilities/settings", settings);
        return true;
      } catch (error) {
        logger.error(LogComponents.VULN, "Failed to update vulnerability settings", error);
        return false;
      }
    },
    []
  );

  return {
    triggerScan,
    fetchStatus,
    fetchResults,
    fetchDeviceVulnerabilities,
    fetchSettings,
    updateSettings,
    isScanning,
    scanError,
  };
}
