/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface CableResponse {
  supported: boolean;
  status: string;
  length?: number;
  lengthFt?: number;
  pairs?: CablePairResult[];
  faults: string[];
  wiringStandard: string;
  pinout?: CablePinout[];
  isCrossover?: boolean;
  driverName?: string;
}
export interface CablePairResult {
  pair: string;
  pairLetter: string;
  status: string;
  lengthM?: number;
  lengthFt?: number;
}
export interface CablePinout {
  pin: number;
  color: string;
  pair: string;
}
