/**
 * Update Types
 *
 * Type definitions for the in-app update system.
 */

/** State of the update process */
export type UpdateState =
  | 'idle'
  | 'checking'
  | 'downloading'
  | 'verifying'
  | 'applying'
  | 'restarting'
  | 'complete'
  | 'failed'
  | 'rolled_back';

/** Information about an available update */
export interface UpdateInfo {
  available: boolean;
  currentVersion: string;
  latestVersion: string;
  releaseNotes: string;
  publishedAt: string;
  // biome-ignore lint/style/useNamingConvention: Matches Go JSON field naming
  downloadURL: string;
  downloadSize: number;
  // biome-ignore lint/style/useNamingConvention: Matches Go JSON field naming
  checksumURL: string;
}

/** Current status of the update service */
export interface UpdateStatus {
  state: UpdateState;
  progress: number;
  message: string;
  error: string;
  downloadedBytes: number;
  totalBytes: number;
  startedAt: string;
}

/** Combined status response from the API */
export interface UpdateStatusResponse {
  state: UpdateState;
  progress: number;
  message: string;
  error: string;
  downloadedBytes: number;
  totalBytes: number;
  startedAt: string;
  lastCheck: string;
  updateReady: boolean;
  requiresAction: boolean;
}

/** Update configuration */
export interface UpdateConfig {
  enabled: boolean;
  checkInterval: string;
  autoDownload: boolean;
  autoApply: boolean;
  includePrerelease: boolean;
}

/** Request to update configuration */
export interface UpdateConfigRequest {
  enabled?: boolean;
  checkInterval?: string;
  autoDownload?: boolean;
  autoApply?: boolean;
  includePrerelease?: boolean;
}

/** API response for check/info endpoints */
export interface UpdateCheckResponse {
  available: boolean;
  currentVersion: string;
  latestVersion: string;
  releaseNotes: string;
  publishedAt: string;
  // biome-ignore lint/style/useNamingConvention: Matches Go JSON field naming
  downloadURL: string;
  downloadSize: number;
  // biome-ignore lint/style/useNamingConvention: Matches Go JSON field naming
  checksumURL: string;
}

/** API response for action endpoints */
export interface UpdateActionResponse {
  status: string;
  message: string;
}
