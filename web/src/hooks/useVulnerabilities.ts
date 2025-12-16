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
import { getAuthHeaders } from "./useAuth";
import type {
  DeviceVulnerabilities,
  VulnerabilityScannerStatus,
  VulnerabilityScannerConfig,
} from "../types/vulnerabilities";

// API base URL for vulnerability endpoints
const API_BASE = "";

/** API response for scan initiation */
interface ScanResponse {
  status: string; // "scan started" on success
}

/** API response for vulnerability results */
interface ResultsResponse {
  results: DeviceVulnerabilities[]; // Array of device vulnerability reports
  count: number; // Total number of results
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
      const url = ip
        ? `${API_BASE}/api/vulnerabilities/scan?ip=${encodeURIComponent(ip)}`
        : `${API_BASE}/api/vulnerabilities/scan`;

      const response = await fetch(url, {
        method: "POST",
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error || "Failed to start vulnerability scan");
      }

      const data: ScanResponse = await response.json();
      return data.status === "scan started";
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
      const response = await fetch(`${API_BASE}/api/vulnerabilities/status`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        return null;
      }
      return await response.json();
    } catch (error) {
      console.error("Failed to fetch vulnerability status:", error);
      return null;
    }
  }, []);

  const fetchResults = useCallback(async (severity?: string): Promise<DeviceVulnerabilities[]> => {
    try {
      const url = severity
        ? `${API_BASE}/api/vulnerabilities/results?severity=${encodeURIComponent(severity)}`
        : `${API_BASE}/api/vulnerabilities/results`;

      const response = await fetch(url, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        return [];
      }

      const data: ResultsResponse = await response.json();
      return data.results || [];
    } catch (error) {
      console.error("Failed to fetch vulnerability results:", error);
      return [];
    }
  }, []);

  const fetchDeviceVulnerabilities = useCallback(
    async (ip: string): Promise<DeviceVulnerabilities | null> => {
      try {
        const response = await fetch(
          `${API_BASE}/api/vulnerabilities/device?ip=${encodeURIComponent(ip)}`,
          {
            headers: getAuthHeaders(),
          }
        );

        if (!response.ok) {
          return null;
        }

        return await response.json();
      } catch (error) {
        console.error(`Failed to fetch vulnerabilities for device ${ip}:`, error);
        return null;
      }
    },
    []
  );

  const fetchSettings = useCallback(async (): Promise<VulnerabilityScannerConfig | null> => {
    try {
      const response = await fetch(`${API_BASE}/api/vulnerabilities/settings`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        return null;
      }
      return await response.json();
    } catch (error) {
      console.error("Failed to fetch vulnerability settings:", error);
      return null;
    }
  }, []);

  const updateSettings = useCallback(
    async (settings: Partial<VulnerabilityScannerConfig>): Promise<boolean> => {
      try {
        const response = await fetch(`${API_BASE}/api/vulnerabilities/settings`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            ...getAuthHeaders(),
          },
          body: JSON.stringify(settings),
        });

        return response.ok;
      } catch (error) {
        console.error("Failed to update vulnerability settings:", error);
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
