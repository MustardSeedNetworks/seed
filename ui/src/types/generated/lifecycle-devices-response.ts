/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface LifecycleDevicesResponse {
  records: number;
  devices: LifecycleDevice[];
}
export interface LifecycleDevice {
  ip: string;
  name?: string;
  components: LifecycleComponent[];
}
export interface LifecycleComponent {
  productId: string;
  name?: string;
  class?: string;
  serialNum?: string;
  status:
    'not_listed' | 'announced' | 'end_of_sale' | 'end_of_sw_maintenance' | 'end_of_security_support' | 'end_of_support';
  record?: EoXRecord;
}
export interface EoXRecord {
  productId: string;
  description?: string;
  endOfSale?: string;
  endOfSwMaintenance?: string;
  endOfSecuritySupport?: string;
  lastDateOfSupport?: string;
  bulletin?: string;
  migrationProductId?: string;
}
