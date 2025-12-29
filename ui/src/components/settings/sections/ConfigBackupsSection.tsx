/**
 * ConfigBackupsSection Component
 *
 * Purpose: Settings panel for configuration backup and restore functionality.
 * Allows users to create, list, restore, and delete configuration backups.
 *
 * Key Features:
 * - List backups: Shows available configuration backups with timestamps
 * - Create backup: Button to create a manual backup of current config
 * - Restore backup: Ability to restore config from any backup
 * - Delete backup: Remove unwanted backups
 * - Version info: Shows current and latest config version
 * - Confirmation dialog: Prevents accidental restores
 *
 * Usage:
 * ```typescript
 * <ConfigBackupsSection />
 * ```
 *
 * Dependencies: CollapsibleSection, getAuthHeaders, theme tokens
 * State: backups list, loading states, error messages
 */

import { memo, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { button, cn, icon as iconTokens, layout, radius, spacing } from "../../../styles/theme";
import { CollapsibleSection } from "../../ui/CollapsibleSection";

const ApiBase = import.meta.env.VITE_API_BASE || "";

interface BackupInfo {
  name: string;
  path: string;
  size: number;
  createdAt: string;
  version: number;
}

interface ConfigVersion {
  current: number;
  latest: number;
  needsMigration: boolean;
}

/**
 * Settings section for configuration backup and restore.
 */
export const ConfigBackupsSection = memo(function ConfigBackupsSection() {
  const { t } = useTranslation("settings");
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [version, setVersion] = useState<ConfigVersion | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [restoreConfirm, setRestoreConfirm] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchBackups = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [backupsRes, versionRes] = await Promise.all([
        fetch(`${ApiBase}/api/config/backups`, { credentials: "include" }),
        fetch(`${ApiBase}/api/config/version`, { credentials: "include" }),
      ]);

      if (backupsRes.ok) {
        const data = await backupsRes.json();
        setBackups(data.backups || []);
      } else {
        setError(t("configBackups.fetchError"));
      }

      if (versionRes.ok) {
        setVersion(await versionRes.json());
      }
    } catch {
      setError(t("configBackups.networkError"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchBackups();
  }, [fetchBackups]);

  const createBackup = async () => {
    setActionLoading("create");
    setError(null);
    try {
      const response = await fetch(`${ApiBase}/api/config/backup`, {
        method: "POST",
        credentials: "include",
      });
      if (response.ok) {
        await fetchBackups();
      } else {
        setError(t("configBackups.createError"));
      }
    } catch {
      setError(t("configBackups.networkError"));
    } finally {
      setActionLoading(null);
    }
  };

  const restoreBackup = async (backupName: string) => {
    setActionLoading(backupName);
    setError(null);
    try {
      const response = await fetch(`${ApiBase}/api/config/restore`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ backupName }),
      });
      if (response.ok) {
        setRestoreConfirm(null);
        // Reload page to apply restored config
        window.location.reload();
      } else {
        const text = await response.text();
        setError(text || t("configBackups.restoreError"));
      }
    } catch {
      setError(t("configBackups.networkError"));
    } finally {
      setActionLoading(null);
    }
  };

  const deleteBackup = async (backupName: string) => {
    setActionLoading(backupName);
    setError(null);
    try {
      const response = await fetch(
        `${ApiBase}/api/config/backup/delete?name=${encodeURIComponent(backupName)}`,
        {
          method: "DELETE",
          credentials: "include",
        },
      );
      if (response.ok) {
        await fetchBackups();
      } else {
        setError(t("configBackups.deleteError"));
      }
    } catch {
      setError(t("configBackups.networkError"));
    } finally {
      setActionLoading(null);
    }
  };

  const formatDate = (dateStr: string) => {
    try {
      return new Date(dateStr).toLocaleString();
    } catch {
      return dateStr;
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <svg
            className={iconTokens.size.sm}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2"
            />
          </svg>
          <span>{t("configBackups.title")}</span>
        </div>
      }
    >
      <div className="stack-sm">
        {/* Version Info */}
        {version && (
          <div
            className={cn(
              layout.flex.between,
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border",
            )}
          >
            <span className="body-small text-text-muted">{t("configBackups.version")}</span>
            <span className="body-small text-text-primary">
              v{version.current}
              {version.needsMigration && (
                <span className="ml-2 text-status-warning">
                  ({t("configBackups.needsMigration")})
                </span>
              )}
            </span>
          </div>
        )}

        {/* Create Backup Button */}
        <button
          type="button"
          onClick={createBackup}
          disabled={actionLoading === "create"}
          className={cn(
            "w-full",
            button.size.md,
            "bg-brand-primary text-text-inverse",
            radius.md,
            "font-medium hover:bg-brand-primary-hover transition-colors flex items-center justify-center",
            spacing.gap.compact,
            "touch-manipulation disabled:opacity-50",
          )}
        >
          {actionLoading === "create" ? (
            t("configBackups.creating")
          ) : (
            <>
              <svg
                className={iconTokens.size.sm}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 4v16m8-8H4"
                />
              </svg>
              {t("configBackups.createBackup")}
            </>
          )}
        </button>

        {/* Error Message */}
        {error && <p className="caption text-status-error">{error}</p>}

        {/* Backups List */}
        {loading ? (
          <p className="caption text-text-muted">{t("configBackups.loading")}</p>
        ) : backups.length === 0 ? (
          <p className="caption text-text-muted">{t("configBackups.noBackups")}</p>
        ) : (
          <div className="stack-xs">
            <p className="caption text-text-muted">
              {t("configBackups.available", { count: backups.length })}
            </p>
            {backups.map((backup) => (
              <div
                key={backup.name}
                className={cn(
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border",
                )}
              >
                <div className={layout.flex.between}>
                  <div>
                    <p className="body-small text-text-primary font-medium">
                      {formatDate(backup.createdAt)}
                    </p>
                    <p className="caption text-text-muted">
                      {formatSize(backup.size)} • v{backup.version || "?"}
                    </p>
                  </div>
                  <div className={cn("flex", spacing.gap.compact)}>
                    {restoreConfirm === backup.name ? (
                      <>
                        <button
                          type="button"
                          onClick={() => restoreBackup(backup.name)}
                          disabled={!!actionLoading}
                          className={cn(
                            spacing.chip.sm,
                            radius.md,
                            "bg-status-warning text-text-inverse caption hover:opacity-90 disabled:opacity-50",
                          )}
                        >
                          {actionLoading === backup.name
                            ? t("configBackups.restoring")
                            : t("configBackups.confirm")}
                        </button>
                        <button
                          type="button"
                          onClick={() => setRestoreConfirm(null)}
                          disabled={!!actionLoading}
                          className={cn(
                            spacing.chip.sm,
                            radius.md,
                            "border border-surface-border caption text-text-muted hover:text-text-primary disabled:opacity-50",
                          )}
                        >
                          {t("configBackups.cancel")}
                        </button>
                      </>
                    ) : (
                      <>
                        <button
                          type="button"
                          onClick={() => setRestoreConfirm(backup.name)}
                          disabled={!!actionLoading}
                          className={cn(
                            spacing.chip.sm,
                            radius.md,
                            "border border-surface-border caption text-text-muted hover:text-text-primary disabled:opacity-50",
                          )}
                          title={t("configBackups.restoreTooltip")}
                        >
                          {t("configBackups.restore")}
                        </button>
                        <button
                          type="button"
                          onClick={() => deleteBackup(backup.name)}
                          disabled={!!actionLoading}
                          className={cn(
                            spacing.chip.sm,
                            radius.md,
                            "border border-status-error caption text-status-error hover:bg-status-error hover:text-text-inverse disabled:opacity-50",
                          )}
                          title={t("configBackups.deleteTooltip")}
                        >
                          {actionLoading === backup.name ? "..." : t("configBackups.delete")}
                        </button>
                      </>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
          {t("configBackups.description")}
        </p>
      </div>
    </CollapsibleSection>
  );
});
