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
  downloadURL: string;
  downloadSize: number;
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

// UpdateStatusResponse, UpdateConfigRequest, and UpdateCheckResponse now come
// from the generated schema (code-first contract) under
// src/types/generated/. The hand-maintained twins had already drifted from the
// backend DTO (e.g. downloadURL vs the real downloadUrl, missing canAutoUpdate/
// requiresRestart) — the generated types are the source of truth. The
// remaining types here (UpdateState, UpdateInfo, UpdateStatus, UpdateConfig,
// UpdateActionResponse) have no generated DTO and stay hand-maintained.

/** Update configuration */
export interface UpdateConfig {
  enabled: boolean;
  checkInterval: string;
  autoDownload: boolean;
  autoApply: boolean;
  includePrerelease: boolean;
}

/** API response for action endpoints */
export interface UpdateActionResponse {
  status: string;
  message: string;
}
