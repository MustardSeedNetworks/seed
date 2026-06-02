/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface BluetoothDevicesResponse {
  devices: BluetoothDevice[];
  total: number;
}
export interface BluetoothDevice {
  id: string;
  device_id?: string;
  address: string;
  name: string;
  alias: string;
  vendor: string;
  is_connected: boolean;
  type: string;
  device_class: string;
  appearance: number;
  class_of_device?: number;
  rssi: number;
  tx_power: number;
  est_distance_m: number;
  is_connectable: boolean;
  service_uuids?: string[];
  manufacturer_id?: number;
  manufacturer_data?: string;
  is_authorized: boolean;
  is_trusted: boolean;
  is_paired: boolean;
  is_blocked: boolean;
  first_seen: string;
  last_seen: string;
  metadata?: {};
}
