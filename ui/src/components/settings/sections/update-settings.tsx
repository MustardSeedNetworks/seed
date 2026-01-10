/**
 * UpdateSettings Component
 *
 * Purpose: Settings panel for in-app update configuration and manual update actions.
 * Allows users to check for updates, configure automatic updates, and manage update behavior.
 *
 * Key Features:
 * - Check for updates: Manual button to check for new versions
 * - Update status display: Shows current and available versions
 * - Download and apply updates: Manual update workflow
 * - Auto-update settings: Configure automatic checking and downloading
 * - Prerelease option: Include beta/RC versions
 * - Rollback support: Revert to previous version if needed
 *
 * Usage:
 * ```typescript
 * <UpdateSettings
 *   currentVersion="1.0.0"
 *   onUpdateApplied={handleRestart}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, Icons, useUpdates hook
 */

import { memo, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useUpdates } from "../../../hooks/useUpdates";
import { formatBytes } from "../../../lib/format";
import { cn, icon as iconTokens, layout, radius, spacing } from "../../../styles/theme";
import type { UpdateConfig } from "../../../types/update";
import { CollapsibleSection } from "../../ui/collapsible-section";
import { CheckCircle, Download, Loader, RefreshCw, RotateCcw } from "../../ui/icons";

interface UpdateSettingsProps {
  currentVersion?: string;
  onUpdateApplied?: () => void;
}

/**
 * Settings section for application updates.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const UpdateSettings = memo(function UpdateSettings({
  currentVersion = "",
  onUpdateApplied,
}: UpdateSettingsProps) {
  const { t } = useTranslation("settings");
  const {
    updateInfo,
    status,
    config,
    isChecking,
    isDownloading,
    isApplying,
    error,
    checkForUpdate,
    getStatus,
    getConfig,
    downloadUpdate,
    applyUpdate,
    rollback,
    updateConfig,
  } = useUpdates();

  // Local state for settings form
  const [localConfig, setLocalConfig] = useState<UpdateConfig | null>(null);
  const [hasChecked, setHasChecked] = useState(false);

  // Load initial config and status
  useEffect(() => {
    getConfig().catch(() => {
      // Ignore initial load errors
    });
    getStatus().catch(() => {
      // Ignore initial load errors
    });
  }, [getConfig, getStatus]);

  // Sync local config with fetched config
  useEffect(() => {
    if (config) {
      setLocalConfig(config);
    }
  }, [config]);

  const handleCheckForUpdate = useCallback(async () => {
    await checkForUpdate();
    setHasChecked(true);
  }, [checkForUpdate]);

  const handleDownload = useCallback(async () => {
    await downloadUpdate();
  }, [downloadUpdate]);

  const handleApply = useCallback(async () => {
    const success = await applyUpdate();
    if (success) {
      onUpdateApplied?.();
    }
  }, [applyUpdate, onUpdateApplied]);

  const handleRollback = useCallback(async () => {
    await rollback();
  }, [rollback]);

  const handleConfigChange = useCallback(
    async (key: keyof UpdateConfig, value: boolean) => {
      if (!localConfig) return;

      const newConfig = { ...localConfig, [key]: value };
      setLocalConfig(newConfig);
      await updateConfig({ [key]: value });
    },
    [localConfig, updateConfig],
  );

  const formatDate = (dateStr: string): string => {
    if (!dateStr) return "";
    try {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Download className={iconTokens.size.sm} />
          <span>{t("sections.updates", "Updates")}</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm">
        {/* Current Version Display */}
        <div
          className={cn(
            layout.flex.between,
            spacing.pad.sm,
            "bg-surface-base",
            radius.default,
            "border border-surface-border",
          )}
        >
          <span className="body-small text-text-primary">
            {t("updates.currentVersion", "Current Version")}
          </span>
          <span className="body-small text-text-secondary font-mono">
            {currentVersion || updateInfo?.currentVersion || "Unknown"}
          </span>
        </div>

        {/* Check for Updates Button */}
        <button
          type="button"
          onClick={handleCheckForUpdate}
          disabled={isChecking}
          className={cn(
            "w-full",
            layout.flex.between,
            spacing.pad.sm,
            "bg-surface-base",
            radius.default,
            "border border-surface-border hover:bg-surface-hover transition-colors",
            "disabled:opacity-50 disabled:cursor-not-allowed",
          )}
        >
          <span className="body-small text-text-primary">
            {t("updates.checkForUpdates", "Check for Updates")}
          </span>
          {isChecking ? (
            <Loader className={cn(iconTokens.size.sm, "animate-spin")} />
          ) : (
            <RefreshCw className={iconTokens.size.sm} />
          )}
        </button>

        {/* Error Display */}
        {error && (
          <div
            className={cn(spacing.pad.sm, "bg-red-500/10 border border-red-500/30", radius.default)}
          >
            <span className="body-small text-red-500">{error}</span>
          </div>
        )}

        {/* Update Available */}
        {hasChecked && updateInfo?.available && (
          <div
            className={cn(
              "stack-sm",
              spacing.pad.sm,
              "bg-green-500/10 border border-green-500/30",
              radius.default,
            )}
          >
            <div className={layout.flex.between}>
              <span className="body-small text-green-500 font-medium">
                {t("updates.updateAvailable", "Update Available")}
              </span>
              <span className="body-small text-green-500 font-mono">
                v{updateInfo.latestVersion}
              </span>
            </div>

            {updateInfo.publishedAt && (
              <div className="body-small text-text-secondary">
                {t("updates.releasedOn", "Released")}: {formatDate(updateInfo.publishedAt)}
              </div>
            )}

            {updateInfo.downloadSize > 0 && (
              <div className="body-small text-text-secondary">
                {t("updates.downloadSize", "Size")}: {formatBytes(updateInfo.downloadSize)}
              </div>
            )}

            {updateInfo.releaseNotes && (
              <div className="body-small text-text-secondary whitespace-pre-wrap max-h-32 overflow-y-auto">
                {updateInfo.releaseNotes}
              </div>
            )}

            {/* Download Button */}
            {!status?.updateReady && (
              <button
                type="button"
                onClick={handleDownload}
                disabled={isDownloading}
                className={cn(
                  "w-full",
                  layout.flex.center,
                  spacing.gap.sm,
                  spacing.pad.sm,
                  "bg-green-500 hover:bg-green-600 text-white",
                  radius.default,
                  "transition-colors",
                  "disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {isDownloading ? (
                  <>
                    <Loader className={cn(iconTokens.size.sm, "animate-spin")} />
                    <span className="body-small">{t("updates.downloading", "Downloading...")}</span>
                  </>
                ) : (
                  <>
                    <Download className={iconTokens.size.sm} />
                    <span className="body-small">
                      {t("updates.downloadUpdate", "Download Update")}
                    </span>
                  </>
                )}
              </button>
            )}

            {/* Apply Button (shows when downloaded) */}
            {status?.updateReady && (
              <button
                type="button"
                onClick={handleApply}
                disabled={isApplying}
                className={cn(
                  "w-full",
                  layout.flex.center,
                  spacing.gap.sm,
                  spacing.pad.sm,
                  "bg-blue-500 hover:bg-blue-600 text-white",
                  radius.default,
                  "transition-colors",
                  "disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {isApplying ? (
                  <>
                    <Loader className={cn(iconTokens.size.sm, "animate-spin")} />
                    <span className="body-small">{t("updates.applying", "Applying...")}</span>
                  </>
                ) : (
                  <>
                    <CheckCircle className={iconTokens.size.sm} />
                    <span className="body-small">
                      {t("updates.applyUpdate", "Apply Update & Restart")}
                    </span>
                  </>
                )}
              </button>
            )}
          </div>
        )}

        {/* No Update Available */}
        {hasChecked && !updateInfo?.available && !error && (
          <div
            className={cn(
              layout.flex.center,
              spacing.gap.sm,
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border",
            )}
          >
            <CheckCircle className={cn(iconTokens.size.sm, "text-green-500")} />
            <span className="body-small text-text-secondary">
              {t("updates.upToDate", "You're up to date!")}
            </span>
          </div>
        )}

        {/* Last Check Time */}
        {status?.lastCheck && (
          <div className="body-small text-text-muted text-center">
            {t("updates.lastChecked", "Last checked")}: {formatDate(status.lastCheck)}
          </div>
        )}

        {/* Configuration Options */}
        {localConfig && (
          <>
            <div className="border-t border-surface-border my-2" />

            <label
              className={cn(
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border cursor-pointer",
              )}
            >
              <span className="body-small text-text-primary">
                {t("updates.autoCheck", "Automatic Update Checks")}
              </span>
              <input
                type="checkbox"
                checked={localConfig.enabled}
                onChange={(e) => handleConfigChange("enabled", e.target.checked)}
                className="w-4 h-4 accent-primary"
              />
            </label>

            <label
              className={cn(
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border cursor-pointer",
              )}
            >
              <span className="body-small text-text-primary">
                {t("updates.autoDownload", "Auto-Download Updates")}
              </span>
              <input
                type="checkbox"
                checked={localConfig.autoDownload}
                onChange={(e) => handleConfigChange("autoDownload", e.target.checked)}
                className="w-4 h-4 accent-primary"
              />
            </label>

            <label
              className={cn(
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border cursor-pointer",
              )}
            >
              <span className="body-small text-text-primary">
                {t("updates.includePrerelease", "Include Pre-release Versions")}
              </span>
              <input
                type="checkbox"
                checked={localConfig.includePrerelease}
                onChange={(e) => handleConfigChange("includePrerelease", e.target.checked)}
                className="w-4 h-4 accent-primary"
              />
            </label>

            {/* Rollback Option */}
            <button
              type="button"
              onClick={handleRollback}
              className={cn(
                "w-full",
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border hover:bg-surface-hover transition-colors",
                "text-orange-500",
              )}
            >
              <span className="body-small">
                {t("updates.rollback", "Rollback to Previous Version")}
              </span>
              <RotateCcw className={iconTokens.size.sm} />
            </button>
          </>
        )}
      </div>
    </CollapsibleSection>
  );
});
