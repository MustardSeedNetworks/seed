/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface SystemHealthResponse {
  system: Health;
  application: SystemHealthApplicationResponse;
  services: SystemHealthServicesResponse;
}
export interface Health {
  cpuPercent: number;
  memoryPercent: number;
  memoryUsed: number;
  memoryTotal: number;
  diskPercent: number;
  diskUsed: number;
  diskTotal: number;
  uptime: number;
  loadAvg1: number;
  loadAvg5: number;
  loadAvg15: number;
  goroutines: number;
  processMemory: number;
  hostname: string;
  os: string;
  arch: string;
  numCpu: number;
  topCpuProcesses?: ProcessInfo[];
  topMemoryProcesses?: ProcessInfo[];
}
export interface ProcessInfo {
  name: string;
  pid: number;
  cpuPercent: number;
  memoryMb: number;
}
export interface SystemHealthApplicationResponse {
  version: string;
  uptime: number;
}
export interface SystemHealthServicesResponse {
  discoveryService: boolean;
  linkMonitor: boolean;
  sseHub: boolean;
}
