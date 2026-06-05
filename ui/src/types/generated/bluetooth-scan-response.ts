/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface BluetoothScanResponse {
  devices: BluetoothDevice[];
  adapterName: string;
  scanType: string;
  scanTime: string;
  scanDurationMs: number;
  stats?: BluetoothDiscoveryStats;
}
export interface BluetoothDevice {
  id: string;
  deviceId?: string;
  address: string;
  name: string;
  alias: string;
  vendor: string;
  isConnected: boolean;
  type: string;
  deviceClass: string;
  appearance: number;
  classOfDevice?: number;
  rssi: number;
  txPower: number;
  estDistanceM: number;
  isConnectable: boolean;
  serviceUuids?: string[];
  serviceNames?: string[];
  manufacturerId?: number;
  companyName?: string;
  appearanceLabel?: string;
  manufacturerData?: string;
  isAuthorized: boolean;
  isTrusted: boolean;
  isPaired: boolean;
  isBlocked: boolean;
  firstSeen: string;
  lastSeen: string;
  metadata?: {};
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
