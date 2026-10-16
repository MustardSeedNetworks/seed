/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WMIInventoryResponse {
  targetId: string;
  collectedAt: string;
  inventory: Inventory;
}
export interface Inventory {
  os: OperatingSystem;
  system: ComputerSystem;
  firmware: Firmware;
  processors: Processor[];
  disks: Disk[];
  adapters: NetworkAdapter[];
  software: Program[];
}
export interface OperatingSystem {
  caption: string;
  version: string;
  buildNumber: string;
  architecture: string;
  installDate?: string;
  lastBootUpTime?: string;
}
export interface ComputerSystem {
  manufacturer: string;
  model: string;
  domain: string;
  memoryBytes: number;
}
export interface Firmware {
  serialNumber: string;
  version: string;
}
export interface Processor {
  name: string;
  cores: number;
  logicalCores: number;
}
export interface Disk {
  deviceId: string;
  sizeBytes: number;
  freeBytes: number;
}
export interface NetworkAdapter {
  description: string;
  macAddress: string;
  ipAddresses: string[];
}
export interface Program {
  name: string;
  version: string;
  vendor: string;
}
