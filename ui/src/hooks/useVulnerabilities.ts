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
 * A scan is a `vuln-scan` job (ADR-0005): triggerScan submits it and resolves
 * once the job reaches a terminal state, so callers reload on completion
 * rather than after a guessed delay (#2960).
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

import { useRef, useState } from 'react';
import { api } from '../api';
import { getJob, isTerminalJobState, submitJob } from '../lib/jobsClient';
import { LogComponents, logger } from '../lib/logger';
import type { JobResponse } from '../types/generated/job-response';
import type { VulnScanRequest } from '../types/generated/vuln-scan-request';
import type {
  DeviceVulnerabilities,
  VulnerabilityScannerConfig,
  VulnerabilityScannerStatus,
} from '../types/vulnerabilities';
import { useJobEvents } from './useJobEvents';

type SeverityFilter = 'critical' | 'high' | 'medium' | 'low';

/** API response for vulnerability results */
interface ResultsResponse {
  results: DeviceVulnerabilities[]; // Array of device vulnerability reports
  count: number; // Total number of results
}

function isValidIpv4(ip: string): boolean {
  const parts = ip.split('.');
  if (parts.length !== 4) {
    return false;
  }
  return parts.every((part) => {
    if (!/^\d{1,3}$/.test(part)) {
      return false;
    }
    const value = Number(part);
    return value >= 0 && value <= 255;
  });
}

function isValidIpv6(ip: string): boolean {
  if (ip === '') {
    return false;
  }

  const [head, ...rest] = ip.split('::');
  if (rest.length > 1) {
    return false;
  }

  const headParts = head ? head.split(':') : [];
  const tailParts = rest.length === 1 && rest[0] ? rest[0].split(':') : [];
  const hasCompression = rest.length === 1;

  const allParts = hasCompression ? [...headParts, ...tailParts] : headParts;
  if (allParts.some((p) => p === '')) {
    return false;
  }

  const lastPart = allParts.at(-1);
  const hasIpv4Tail = lastPart ? lastPart.includes('.') : false;

  const validateHextet = (part: string): boolean => /^[0-9a-fA-F]{1,4}$/.test(part);

  if (hasIpv4Tail) {
    if (!(lastPart && isValidIpv4(lastPart))) {
      return false;
    }
    const hextets = allParts.slice(0, -1);
    if (!hextets.every(validateHextet)) {
      return false;
    }

    if (hasCompression) {
      return hextets.length <= 6;
    }
    return hextets.length === 6;
  }

  if (!allParts.every(validateHextet)) {
    return false;
  }

  if (hasCompression) {
    return allParts.length < 8;
  }
  return allParts.length === 8;
}

function isValidIp(ip: string): boolean {
  return isValidIpv4(ip) || isValidIpv6(ip);
}

function normalizeSeverityFilter(severity: string): SeverityFilter | null {
  const normalized = severity.trim().toLowerCase();
  switch (normalized) {
    case 'critical':
    case 'high':
    case 'medium':
    case 'low':
      return normalized;
    default:
      return null;
  }
}

async function fetchStatus(): Promise<VulnerabilityScannerStatus | null> {
  try {
    return await api.get<VulnerabilityScannerStatus>('/api/v1/security/vulnerabilities/status');
  } catch (error) {
    logger.error(LogComponents.VULN, 'Failed to fetch vulnerability status', error, {
      endpoint: '/api/v1/security/vulnerabilities/status',
    });
    return null;
  }
}

async function fetchResults(severity?: string): Promise<DeviceVulnerabilities[]> {
  try {
    const params = new URLSearchParams();
    if (severity) {
      const validSeverity = normalizeSeverityFilter(severity);
      if (!validSeverity) {
        throw new Error('Invalid severity filter');
      }
      params.set('severity', validSeverity);
    }

    const endpoint =
      params.size > 0
        ? `/api/v1/security/vulnerabilities/results?${params.toString()}`
        : '/api/v1/security/vulnerabilities/results';
    const data = await api.get<ResultsResponse>(endpoint);
    return data.results || [];
  } catch (error) {
    logger.error(LogComponents.VULN, 'Failed to fetch vulnerability results', error, {
      endpoint: '/api/v1/security/vulnerabilities/results',
      severity,
    });
    return [];
  }
}

async function fetchDeviceVulnerabilities(ip: string): Promise<DeviceVulnerabilities | null> {
  try {
    const trimmed = ip.trim();
    if (!isValidIp(trimmed)) {
      throw new Error('Invalid IP address');
    }

    const params = new URLSearchParams({ ip: trimmed });
    return await api.get<DeviceVulnerabilities>(
      `/api/v1/security/vulnerabilities/device?${params.toString()}`,
    );
  } catch (error) {
    logger.error(LogComponents.VULN, 'Failed to fetch vulnerabilities for device', error, {
      ip,
    });
    return null;
  }
}

async function fetchSettings(): Promise<VulnerabilityScannerConfig | null> {
  try {
    return await api.get<VulnerabilityScannerConfig>('/api/v1/security/vulnerabilities/settings');
  } catch (error) {
    logger.error(LogComponents.VULN, 'Failed to fetch vulnerability settings', error, {
      endpoint: '/api/v1/security/vulnerabilities/settings',
    });
    return null;
  }
}

async function updateSettings(settings: Partial<VulnerabilityScannerConfig>): Promise<boolean> {
  try {
    await api.put<{ status: string }>('/api/v1/security/vulnerabilities/settings', settings);
    return true;
  } catch (error) {
    logger.error(LogComponents.VULN, 'Failed to update vulnerability settings', error, {
      endpoint: '/api/v1/security/vulnerabilities/settings',
      updates: settings,
    });
    return false;
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

  // Resolvers for submitted scans, keyed by job id, settled by the job stream.
  const waitersRef = useRef(new Map<string, (job: JobResponse) => void>());

  useJobEvents((job: JobResponse) => {
    const settle = waitersRef.current.get(job.id);
    if (settle && isTerminalJobState(job.state)) {
      waitersRef.current.delete(job.id);
      settle(job);
    }
  });

  // finished resolves with the job's terminal snapshot. The stream is
  // live-only, so a job that finished before its waiter was registered is
  // caught by the one GET issued after registration.
  const finished = (submitted: JobResponse): Promise<JobResponse> => {
    if (isTerminalJobState(submitted.state)) {
      return Promise.resolve(submitted);
    }
    const waiters = waitersRef.current;
    return new Promise<JobResponse>((resolve, reject) => {
      waiters.set(submitted.id, resolve);
      getJob(submitted.id)
        .then((job) => {
          if (isTerminalJobState(job.state) && waiters.delete(job.id)) {
            resolve(job);
          }
        })
        .catch((error: unknown) => {
          waiters.delete(submitted.id);
          reject(error);
        });
    });
  };

  const triggerScan = async (ip?: string): Promise<boolean> => {
    setIsScanning(true);
    setScanError(null);

    const scan = async (): Promise<boolean> => {
      const params: VulnScanRequest = {};
      if (ip) {
        const trimmed = ip.trim();
        if (!isValidIp(trimmed)) {
          throw new Error('Invalid IP address');
        }
        params.ip = trimmed;
      }

      const job = await finished(await submitJob({ kind: 'vuln-scan', params }));
      if (job.state !== 'succeeded') {
        throw new Error(job.error ?? `Vulnerability scan ${job.state}`);
      }
      return true;
    };

    const ok = await scan().catch((error: unknown) => {
      setScanError(error instanceof Error ? error.message : 'Unknown error');
      return false;
    });
    setIsScanning(false);
    return ok;
  };

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
