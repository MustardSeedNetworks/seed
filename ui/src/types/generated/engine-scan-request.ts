/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface EngineScanRequest {
  scanType: string;
  includeWired: boolean;
  includeWifi: boolean;
  includeBluetooth: boolean;
  includeSnmp: boolean;
  includePortScan: boolean;
  includeVulnScan: boolean;
  freshWiredScan: boolean;
  freshWifiScan: boolean;
  freshBluetoothScan: boolean;
  portScanIntensity?: string;
  portScanCustomPorts?: number[];
  timingProfile?: string;
}
