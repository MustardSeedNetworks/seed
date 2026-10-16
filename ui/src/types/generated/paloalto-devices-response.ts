/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface PaloAltoDevicesResponse {
  readAt?: string;
  devices: Device[];
}
export interface Device {
  serial: string;
  hostname?: string;
  model?: string;
  family?: string;
  swVersion?: string;
  ipAddress?: string;
  managedBy?: string;
  status: 'connected' | 'disconnected';
  haState?: string;
}
