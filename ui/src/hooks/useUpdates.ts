/**
 * Updates Hook
 *
 * Manages application update operations including checking for updates,
 * downloading, applying, and rollback functionality.
 *
 * Features:
 * - Check for available updates via GitHub releases
 * - Download updates with progress tracking
 * - Apply downloaded updates
 * - Rollback to previous version
 * - Configure update settings
 *
 * Usage:
 * ```typescript
 * const { checkForUpdate, downloadUpdate, applyUpdate, status } = useUpdates();
 *
 * // Check for updates
 * const updateInfo = await checkForUpdate();
 * if (updateInfo?.available) {
 *   await downloadUpdate();
 *   await applyUpdate();
 * }
 * ```
 */

import { useCallback, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";
import type {
  UpdateActionResponse,
  UpdateCheckResponse,
  UpdateConfig,
  UpdateConfigRequest,
  UpdateStatusResponse,
} from "../types/update";

/**
 * Custom hook for managing application updates.
 *
 * @returns Update state and control functions
 */
export function useUpdates(): {
  updateInfo: UpdateCheckResponse | null;
  status: UpdateStatusResponse | null;
  config: UpdateConfig | null;
  isChecking: boolean;
  isDownloading: boolean;
  isApplying: boolean;
  error: string | null;
  checkForUpdate: () => Promise<UpdateCheckResponse | null>;
  getStatus: () => Promise<UpdateStatusResponse | null>;
  getUpdateInfo: () => Promise<UpdateCheckResponse | null>;
  downloadUpdate: () => Promise<boolean>;
  applyUpdate: () => Promise<boolean>;
  rollback: () => Promise<boolean>;
  getConfig: () => Promise<UpdateConfig | null>;
  updateConfig: (config: Partial<UpdateConfig>) => Promise<boolean>;
} {
  const [updateInfo, setUpdateInfo] = useState<UpdateCheckResponse | null>(null);
  const [status, setStatus] = useState<UpdateStatusResponse | null>(null);
  const [config, setConfig] = useState<UpdateConfig | null>(null);
  const [isChecking, setIsChecking] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
  const [isApplying, setIsApplying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Checks for available updates.
   */
  const checkForUpdate = useCallback(async (): Promise<UpdateCheckResponse | null> => {
    try {
      setError(null);
      setIsChecking(true);
      const data = await api.get<UpdateCheckResponse>("/api/v1/updates/check");
      setUpdateInfo(data);
      return data;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to check for updates";
      setError(message);
      logger.error(LogComponents.System, "Update check failed", err, {
        endpoint: "/api/v1/updates/check",
      });
      return null;
    } finally {
      setIsChecking(false);
    }
  }, []);

  /**
   * Gets the current update status.
   */
  const getStatus = useCallback(async (): Promise<UpdateStatusResponse | null> => {
    try {
      const data = await api.get<UpdateStatusResponse>("/api/v1/updates/status");
      setStatus(data);
      return data;
    } catch (err) {
      logger.error(LogComponents.System, "Failed to get update status", err, {
        endpoint: "/api/v1/updates/status",
      });
      return null;
    }
  }, []);

  /**
   * Gets information about available updates.
   */
  const getUpdateInfo = useCallback(async (): Promise<UpdateCheckResponse | null> => {
    try {
      const data = await api.get<UpdateCheckResponse>("/api/v1/updates/info");
      setUpdateInfo(data);
      return data;
    } catch (err) {
      logger.error(LogComponents.System, "Failed to get update info", err, {
        endpoint: "/api/v1/updates/info",
      });
      return null;
    }
  }, []);

  /**
   * Downloads the available update.
   */
  const downloadUpdate = useCallback(async (): Promise<boolean> => {
    try {
      setError(null);
      setIsDownloading(true);
      await api.post<UpdateActionResponse>("/api/v1/updates/download");
      // Refresh status after download
      await getStatus();
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to download update";
      setError(message);
      logger.error(LogComponents.System, "Update download failed", err, {
        endpoint: "/api/v1/updates/download",
      });
      return false;
    } finally {
      setIsDownloading(false);
    }
  }, [getStatus]);

  /**
   * Applies the downloaded update.
   */
  const applyUpdate = useCallback(async (): Promise<boolean> => {
    try {
      setError(null);
      setIsApplying(true);
      await api.post<UpdateActionResponse>("/api/v1/updates/apply");
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to apply update";
      setError(message);
      logger.error(LogComponents.System, "Update apply failed", err, {
        endpoint: "/api/v1/updates/apply",
      });
      return false;
    } finally {
      setIsApplying(false);
    }
  }, []);

  /**
   * Rolls back to the previous version.
   */
  const rollback = useCallback(async (): Promise<boolean> => {
    try {
      setError(null);
      await api.post<UpdateActionResponse>("/api/v1/updates/rollback");
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to rollback update";
      setError(message);
      logger.error(LogComponents.System, "Update rollback failed", err, {
        endpoint: "/api/v1/updates/rollback",
      });
      return false;
    }
  }, []);

  /**
   * Gets the update configuration.
   */
  const getConfig = useCallback(async (): Promise<UpdateConfig | null> => {
    try {
      const data = await api.get<UpdateConfig>("/api/v1/updates/config");
      setConfig(data);
      return data;
    } catch (err) {
      logger.error(LogComponents.Config, "Failed to get update config", err, {
        endpoint: "/api/v1/updates/config",
      });
      return null;
    }
  }, []);

  /**
   * Updates the configuration.
   */
  const updateConfig = useCallback(async (updates: UpdateConfigRequest): Promise<boolean> => {
    try {
      const data = await api.patch<UpdateConfig>("/api/v1/updates/config", updates);
      setConfig(data);
      return true;
    } catch (err) {
      logger.error(LogComponents.Config, "Failed to update config", err, {
        endpoint: "/api/v1/updates/config",
        updates,
      });
      return false;
    }
  }, []);

  return {
    // State
    updateInfo,
    status,
    config,
    isChecking,
    isDownloading,
    isApplying,
    error,

    // Operations
    checkForUpdate,
    getStatus,
    getUpdateInfo,
    downloadUpdate,
    applyUpdate,
    rollback,

    // Configuration
    getConfig,
    updateConfig,
  };
}
