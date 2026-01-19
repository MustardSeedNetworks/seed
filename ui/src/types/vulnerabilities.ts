/**
 * Vulnerabilities Type Definitions
 *
 * Purpose: TypeScript interfaces for vulnerability scanning data structures.
 * Defines types for CVE data, device vulnerability results, and scanner status.
 *
 * Key Types:
 * - Vulnerability: Individual CVE with severity, score, description, references
 * - DeviceVulnerabilities: Vulnerabilities for a single device with device metadata
 * - VulnerabilityScannerStatus: Scanner operational status and statistics
 * - VulnerabilityScanRequest: Request payload for triggering vulnerability scans
 *
 * Usage:
 * ```typescript
 * import type { DeviceVulnerabilities, Vulnerability } from './vulnerabilities';
 *
 * const deviceVulns: DeviceVulnerabilities = {
 *   deviceIp: '192.168.1.100',
 *   vulnerabilities: [...]
 * };
 * ```
 *
 * Dependencies: None (pure type definitions)
 * Data Source: Vulnerability scanner API endpoints
 */

export interface Vulnerability {
  cveId: string;
  description: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  score: number;
  published: string;
  modified: string;
  references: string[];
  affectedCpe: string;
}

export interface DeviceVulnerabilities {
  deviceIp: string;
  mac: string;
  hostname: string;
  vendor: string;
  product: string;
  version: string;
  vulnerabilities: Vulnerability[];
  scanTime: string;
  error?: string;
}

export interface VulnerabilityScannerStatus {
  enabled: boolean;
  scanning: boolean;
  stats: {
    enabled: boolean;
    running: boolean;
    devicesScanned: number;
    totalVulns: number;
    criticalCount: number;
    highCount: number;
    mediumCount: number;
    lowCount: number;
    lastUpdate: string;
    cveDatabase: string;
  };
  severityFilter: string;
}

export interface VulnerabilityScannerConfig {
  enabled: boolean;
  cveDatabase: string;
  nvdApiKey: string;
  updateInterval: number;
  severityThreshold: string;
  maxConcurrent: number;
}
