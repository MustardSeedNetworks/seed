/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface UserResponse {
  id: number;
  username: string;
  role: string;
  isActive: boolean;
  authProvider: string;
  email?: string;
  displayName?: string;
  lastLogin?: string;
  lockedUntilFuture?: boolean;
  createdAt: string;
  updatedAt: string;
}
