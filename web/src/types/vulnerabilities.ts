export interface Vulnerability {
  cveId: string;
  description: string;
  severity: "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";
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
  cve_database: string;
  nvd_api_key: string;
  update_interval: number;
  severity_threshold: string;
  max_concurrent: number;
}
