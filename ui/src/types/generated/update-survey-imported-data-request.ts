/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface UpdateSurveyImportedDataRequest {
  apLocations: APLocation[];
  clientLocations: ClientLocation[];
  passFailCriteria: PassFailCriterion[];
}
export interface APLocation {
  id: string;
  x: number;
  y: number;
  label?: string;
  bssid?: string;
  vendor?: string;
  notes?: string;
  imported?: boolean;
}
export interface ClientLocation {
  id: string;
  x: number;
  y: number;
  label?: string;
  mac?: string;
  imported?: boolean;
}
export interface PassFailCriterion {
  option: string;
  name?: string;
  limit: number;
  suffix?: string;
  enabled: boolean;
  mode?: string;
  ap?: number;
  imported?: boolean;
}
