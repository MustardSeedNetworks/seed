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

import type React from "react";
import { memo, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useUpdates } from "../../../hooks/useUpdates";
import { formatBytes } from "../../../lib/format";
import { cn, icon as iconTokens, layout, radius, spacing } from "../../../styles/theme";
import type { UpdateConfig } from "../../../types/update";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { CheckCircle, Download, Loader, RefreshCw, RotateCcw } from "../../ui/Icons";

interface UpdateSettingsProps {
  currentVersion?: string;
  onUpdateApplied?: () => void;
}

/**
 * Settings section for application updates.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const UpdateSettings: React.NamedExoticComponent<UpdateSettingsProps> = memo(
  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex update management UI with multiple states
  function updateSettings({ currentVersion = "", onUpdateApplied }: UpdateSettingsProps) {
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

    const handleCheckForUpdate = useCallback(async (): Promise<void> => {
      await checkForUpdate();
      setHasChecked(true);
    }, [checkForUpdate]);

    const handleDownload = useCallback(async (): Promise<void> => {
      await downloadUpdate();
    }, [downloadUpdate]);

    const handleApply = useCallback(async (): Promise<void> => {
      const success = await applyUpdate();
      if (success) {
        onUpdateApplied?.();
      }
    }, [applyUpdate, onUpdateApplied]);

    const handleRollback = useCallback(async (): Promise<void> => {
      await rollback();
    }, [rollback]);

    const handleConfigChange = useCallback(
      async (key: keyof UpdateConfig, value: boolean): Promise<void> => {
        if (!localConfig) {
          return;
        }

        const newConfig = { ...localConfig, [key]: value };
        setLocalConfig(newConfig);
        await updateConfig({ [key]: value });
      },
      [localConfig, updateConfig],
    );

    const formatDate = (dateStr: string): string => {
      if (!dateStr) {
        return "";
      }
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
          <div class={layout.inline.default}>
            <Download class={iconTokens.size.sm} />
            <span>{t("sections.updates", "Updates")}</span>
          </div>
        }
        defaultOpen={false}
      >
        <div class="stack-sm">
          {/* Current Version Display */}
          <div
            class={cn(
              layout.flex.between,
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border",
            )}
          >
            <span class="body-small text-text-primary">
              {t("updates.currentVersion", "Current Version")}
            </span>
            <span class="body-small text-text-secondary font-mono">
              {currentVersion || updateInfo?.currentVersion || "Unknown"}
            </span>
          </div>

          {/* Check for Updates Button */}
          <button
            type="button"
            onClick={(): void => {
              handleCheckForUpdate().catch(() => undefined);
            }}
            disabled={isChecking}
            class={cn(
              "w-full",
              layout.flex.between,
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border hover:bg-surface-hover transition-colors",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            <span class="body-small text-text-primary">
              {t("updates.checkForUpdates", "Check for Updates")}
            </span>
            {isChecking ? (
              <Loader class={cn(iconTokens.size.sm, "animate-spin")} />
            ) : (
              <RefreshCw class={iconTokens.size.sm} />
            )}
          </button>

          {/* Error Display */}
          {error ? (
            <div
              class={cn(spacing.pad.sm, "bg-red-500/10 border border-red-500/30", radius.default)}
            >
              <span class="body-small text-red-500">{error}</span>
            </div>
          ) : null}

          {/* Update Available */}
          {hasChecked && updateInfo?.available ? (
            <div
              class={cn(
                "stack-sm",
                spacing.pad.sm,
                "bg-green-500/10 border border-green-500/30",
                radius.default,
              )}
            >
              <div class={layout.flex.between}>
                <span class="body-small text-green-500 font-medium">
                  {t("updates.updateAvailable", "Update Available")}
                </span>
                <span class="body-small text-green-500 font-mono">v{updateInfo.latestVersion}</span>
              </div>

              {updateInfo.publishedAt ? (
                <div class="body-small text-text-secondary">
                  {t("updates.releasedOn", "Released")}: {formatDate(updateInfo.publishedAt)}
                </div>
              ) : null}

              {updateInfo.downloadSize > 0 ? (
                <div class="body-small text-text-secondary">
                  {t("updates.downloadSize", "Size")}: {formatBytes(updateInfo.downloadSize)}
                </div>
              ) : null}

              {updateInfo.releaseNotes ? (
                <div class="body-small text-text-secondary whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {updateInfo.releaseNotes}
                </div>
              ) : null}

              {/* Download Button */}
              {status?.updateReady ? null : (
                <button
                  type="button"
                  onClick={(): void => {
                    handleDownload().catch(() => undefined);
                  }}
                  disabled={isDownloading}
                  class={cn(
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
                      <Loader class={cn(iconTokens.size.sm, "animate-spin")} />
                      <span class="body-small">{t("updates.downloading", "Downloading...")}</span>
                    </>
                  ) : (
                    <>
                      <Download class={iconTokens.size.sm} />
                      <span class="body-small">
                        {t("updates.downloadUpdate", "Download Update")}
                      </span>
                    </>
                  )}
                </button>
              )}

              {/* Apply Button (shows when downloaded) */}
              {status?.updateReady ? (
                <button
                  type="button"
                  onClick={(): void => {
                    handleApply().catch(() => undefined);
                  }}
                  disabled={isApplying}
                  class={cn(
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
                      <Loader class={cn(iconTokens.size.sm, "animate-spin")} />
                      <span class="body-small">{t("updates.applying", "Applying...")}</span>
                    </>
                  ) : (
                    <>
                      <CheckCircle class={iconTokens.size.sm} />
                      <span class="body-small">
                        {t("updates.applyUpdate", "Apply Update & Restart")}
                      </span>
                    </>
                  )}
                </button>
              ) : null}
            </div>
          ) : null}

          {/* No Update Available */}
          {hasChecked && !updateInfo?.available && !error ? (
            <div
              class={cn(
                layout.flex.center,
                spacing.gap.sm,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border",
              )}
            >
              <CheckCircle class={cn(iconTokens.size.sm, "text-green-500")} />
              <span class="body-small text-text-secondary">
                {t("updates.upToDate", "You're up to date!")}
              </span>
            </div>
          ) : null}

          {/* Last Check Time */}
          {status?.lastCheck ? (
            <div class="body-small text-text-muted text-center">
              {t("updates.lastChecked", "Last checked")}: {formatDate(status.lastCheck)}
            </div>
          ) : null}

          {/* Configuration Options */}
          {localConfig ? (
            <>
              <div class="border-t border-surface-border my-2" />

              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border cursor-pointer",
                )}
              >
                <span class="body-small text-text-primary">
                  {t("updates.autoCheck", "Automatic Update Checks")}
                </span>
                <input
                  type="checkbox"
                  checked={localConfig.enabled}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void => {
                    handleConfigChange("enabled", e.target.checked).catch(() => undefined);
                  }}
                  class="w-4 h-4 accent-primary"
                />
              </label>

              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border cursor-pointer",
                )}
              >
                <span class="body-small text-text-primary">
                  {t("updates.autoDownload", "Auto-Download Updates")}
                </span>
                <input
                  type="checkbox"
                  checked={localConfig.autoDownload}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void => {
                    handleConfigChange("autoDownload", e.target.checked).catch(() => undefined);
                  }}
                  class="w-4 h-4 accent-primary"
                />
              </label>

              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border cursor-pointer",
                )}
              >
                <span class="body-small text-text-primary">
                  {t("updates.includePrerelease", "Include Pre-release Versions")}
                </span>
                <input
                  type="checkbox"
                  checked={localConfig.includePrerelease}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void => {
                    handleConfigChange("includePrerelease", e.target.checked).catch(
                      () => undefined,
                    );
                  }}
                  class="w-4 h-4 accent-primary"
                />
              </label>

              {/* Rollback Option */}
              <button
                type="button"
                onClick={(): void => {
                  handleRollback().catch(() => undefined);
                }}
                class={cn(
                  "w-full",
                  layout.flex.between,
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border hover:bg-surface-hover transition-colors",
                  "text-orange-500",
                )}
              >
                <span class="body-small">
                  {t("updates.rollback", "Rollback to Previous Version")}
                </span>
                <RotateCcw class={iconTokens.size.sm} />
              </button>
            </>
          ) : null}
        </div>
      </CollapsibleSection>
    );
  },
);
