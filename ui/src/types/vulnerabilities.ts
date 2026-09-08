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

import type { DeviceVulnerabilities, Vulnerability } from './generated/engine-discovery-response';

// The scanner's own shapes come from the generated wire types. The hand-typed
// mirrors that used to live here omitted the CISA KEV fields the daemon
// attaches to every finding — `activelyExploited`, `ransomwareRelated`,
// `requiredAction`, `dueDate` — so no view could name them (seed#2393).
export type { DeviceVulnerabilities, Vulnerability };

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
