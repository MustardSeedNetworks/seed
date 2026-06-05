/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface BluetoothStatsResponse {
  stats: BluetoothDiscoveryStats;
}
export interface BluetoothDiscoveryStats {
  totalDevices: number;
  classicDevices: number;
  bleDevices: number;
  dualDevices: number;
  connectedDevices: number;
  authorizedCount: number;
  unauthorizedCount: number;
  devicesByClass: {
    [k: string]: number;
  };
  vendorBreakdown: {
    [k: string]: number;
  };
  lastScanTime: string;
}
