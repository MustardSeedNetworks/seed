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
  total_devices: number;
  classic_devices: number;
  ble_devices: number;
  dual_devices: number;
  connected_devices: number;
  authorized_count: number;
  unauthorized_count: number;
  devices_by_class: {
    [k: string]: number;
  };
  vendor_breakdown: {
    [k: string]: number;
  };
  last_scan_time: string;
}
