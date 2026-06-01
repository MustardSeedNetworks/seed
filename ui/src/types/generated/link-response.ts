/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface LinkResponse {
  interface: string;
  linkUp: boolean;
  carrier: boolean;
  hasIP: boolean;
  speed: string;
  duplex: string;
  advertisedSpeeds: string[];
  mtu: number;
  autoNeg: boolean;
  flapCount24h: number;
  history?: LinkHistoryEvent[];
  uptimeMs?: number;
  poe?: PoEInfo;
  sfp?: SFPInfo;
}
export interface LinkHistoryEvent {
  state: string;
  timestamp: string;
}
export interface PoEInfo {
  detected: boolean;
  standard?: string;
  class?: number;
  powerMw?: number;
  voltage?: number;
}
export interface SFPInfo {
  present: boolean;
  vendor?: string;
  partNumber?: string;
  serial?: string;
  type?: string;
  wavelength?: number;
  distance?: number;
  connector?: string;
  ddmSupport: boolean;
  ddm?: SFPDDMInfo;
}
export interface SFPDDMInfo {
  temperature: number;
  voltage: number;
  txPowerDbm: number;
  txPowerMw: number;
  rxPowerDbm: number;
  rxPowerMw: number;
  laserBiasMa: number;
  alarms?: string[];
  warnings?: string[];
}
