/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface RogueServersResponse {
  servers: RogueServer[];
  rogueCount: number;
  authorizedCount: number;
}
export interface RogueServer {
  ip: string;
  mac: string;
  firstSeen: string;
  lastSeen: string;
  offerCount: number;
  isAuthorized: boolean;
}
