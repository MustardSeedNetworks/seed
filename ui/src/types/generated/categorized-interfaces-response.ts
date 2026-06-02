/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface CategorizedInterfacesResponse {
  ethernet: InterfaceInfo[];
  wifi: InterfaceInfo[];
  recommendedEthernet?: string;
  recommendedWifi?: string;
  currentInterface: string;
  currentType: string;
}
export interface InterfaceInfo {
  name: string;
  friendlyName?: string;
  description?: string;
  type: string;
  up: boolean;
  running: boolean;
  hardwareAddr: string;
  mtu: number;
  addresses: string[];
  speed?: number;
  speedDisplay?: string;
  chipsetVendor?: string;
  chipsetModel?: string;
  hasTDR?: boolean;
  hasDOM?: boolean;
  score?: number;
}
