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

import type React from 'react';
import { memo, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { formatBytes } from '../../../lib/format';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import { CollapsibleSection } from '../../ui/CollapsibleSection';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

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
export const ConfigBackupsSection: React.NamedExoticComponent<Record<string, never>> = memo(
  function ConfigBackupsSectionComponent(): React.ReactElement {
    const { t } = useTranslation('settings');
    const [backups, setBackups] = useState<BackupInfo[]>([]);
    const [version, setVersion] = useState<ConfigVersion | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [restoreConfirm, setRestoreConfirm] = useState<string | null>(null);
    const [actionLoading, setActionLoading] = useState<string | null>(null);

    const fetchBackups = useCallback(async (): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        const [backupsRes, versionRes] = await Promise.all([
          fetch(`${API_BASE}/api/config/backups`, { credentials: 'include' }),
          fetch(`${API_BASE}/api/config/version`, { credentials: 'include' }),
        ]);

        if (backupsRes.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() is a Promise
          const data = await backupsRes.json();
          setBackups(data.backups || []);
        } else {
          setError(t('configBackups.fetchError'));
        }

        if (versionRes.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() is a Promise
          setVersion(await versionRes.json());
        }
      } catch {
        setError(t('configBackups.networkError'));
      } finally {
        setLoading(false);
      }
    }, [t]);

    useEffect((): void => {
      fetchBackups().catch(() => undefined);
    }, [fetchBackups]);

    const createBackup = async (): Promise<void> => {
      setActionLoading('create');
      setError(null);
      try {
        const response = await fetch(`${API_BASE}/api/config/backup`, {
          method: 'POST',
          credentials: 'include',
        });
        if (response.ok) {
          await fetchBackups();
        } else {
          setError(t('configBackups.createError'));
        }
      } catch {
        setError(t('configBackups.networkError'));
      } finally {
        setActionLoading(null);
      }
    };

    const restoreBackup = async (backupName: string): Promise<void> => {
      setActionLoading(backupName);
      setError(null);
      try {
        const response = await fetch(`${API_BASE}/api/config/restore`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ backupName }),
        });
        if (response.ok) {
          setRestoreConfirm(null);
          // Reload page to apply restored config
          window.location.reload();
        } else {
          // biome-ignore lint/nursery/useAwaitThenable: response.text() is a Promise
          const text = await response.text();
          setError(text || t('configBackups.restoreError'));
        }
      } catch {
        setError(t('configBackups.networkError'));
      } finally {
        setActionLoading(null);
      }
    };

    const deleteBackup = async (backupName: string): Promise<void> => {
      setActionLoading(backupName);
      setError(null);
      try {
        const response = await fetch(
          `${API_BASE}/api/config/backup/delete?name=${encodeURIComponent(backupName)}`,
          {
            method: 'DELETE',
            credentials: 'include',
          },
        );
        if (response.ok) {
          await fetchBackups();
        } else {
          setError(t('configBackups.deleteError'));
        }
      } catch {
        setError(t('configBackups.networkError'));
      } finally {
        setActionLoading(null);
      }
    };

    const formatDate = (dateStr: string): string => {
      try {
        return new Date(dateStr).toLocaleString();
      } catch {
        return dateStr;
      }
    };

    // Helper function to render backups list content based on loading/data state
    const renderBackupsContent = (): React.ReactElement => {
      if (loading) {
        return <p class="caption text-text-muted">{t('configBackups.loading')}</p>;
      }
      if (backups.length === 0) {
        return <p class="caption text-text-muted">{t('configBackups.noBackups')}</p>;
      }
      return (
        <div class="stack-xs">
          <p class="caption text-text-muted">
            {t('configBackups.available', { count: backups.length })}
          </p>
          {backups.map((backup) => (
            <div
              key={backup.name}
              class={cn(
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div class={layout.flex.between}>
                <div>
                  <p class="body-small text-text-primary font-medium">
                    {formatDate(backup.createdAt)}
                  </p>
                  <p class="caption text-text-muted">
                    {formatBytes(backup.size)} • v{backup.version || '?'}
                  </p>
                </div>
                <div class={cn('flex', spacing.gap.compact)}>
                  {restoreConfirm === backup.name ? (
                    <>
                      <button
                        type="button"
                        onClick={(): void => {
                          restoreBackup(backup.name).catch(() => undefined);
                        }}
                        disabled={!!actionLoading}
                        class={cn(
                          spacing.chip.sm,
                          radius.md,
                          'bg-status-warning text-text-inverse caption hover:opacity-90 disabled:opacity-50',
                        )}
                      >
                        {actionLoading === backup.name
                          ? t('configBackups.restoring')
                          : t('configBackups.confirm')}
                      </button>
                      <button
                        type="button"
                        onClick={(): void => setRestoreConfirm(null)}
                        disabled={!!actionLoading}
                        class={cn(
                          spacing.chip.sm,
                          radius.md,
                          'border border-surface-border caption text-text-muted hover:text-text-primary disabled:opacity-50',
                        )}
                      >
                        {t('configBackups.cancel')}
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        type="button"
                        onClick={(): void => setRestoreConfirm(backup.name)}
                        disabled={!!actionLoading}
                        class={cn(
                          spacing.chip.sm,
                          radius.md,
                          'border border-surface-border caption text-text-muted hover:text-text-primary disabled:opacity-50',
                        )}
                        title={t('configBackups.restoreTooltip')}
                      >
                        {t('configBackups.restore')}
                      </button>
                      <button
                        type="button"
                        onClick={(): void => {
                          deleteBackup(backup.name).catch(() => undefined);
                        }}
                        disabled={!!actionLoading}
                        class={cn(
                          spacing.chip.sm,
                          radius.md,
                          'border border-status-error caption text-status-error hover:bg-status-error hover:text-text-inverse disabled:opacity-50',
                        )}
                        title={t('configBackups.deleteTooltip')}
                      >
                        {actionLoading === backup.name ? '...' : t('configBackups.delete')}
                      </button>
                    </>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      );
    };

    return (
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <svg
              class={iconTokens.size.sm}
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
            <span>{t('configBackups.title')}</span>
          </div>
        }
      >
        <div class="stack-sm">
          {/* Version Info */}
          {version ? (
            <div
              class={cn(
                layout.flex.between,
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <span class="body-small text-text-muted">{t('configBackups.version')}</span>
              <span class="body-small text-text-primary">
                v{version.current}
                {version.needsMigration ? (
                  <span class="ml-2 text-status-warning">
                    ({t('configBackups.needsMigration')})
                  </span>
                ) : null}
              </span>
            </div>
          ) : null}

          {/* Create Backup Button */}
          <button
            type="button"
            onClick={(): void => {
              createBackup().catch(() => undefined);
            }}
            disabled={actionLoading === 'create'}
            class={cn(
              'w-full',
              button.size.md,
              'bg-brand-primary text-text-inverse',
              radius.md,
              'font-medium hover:bg-brand-primary-hover transition-colors flex items-center justify-center',
              spacing.gap.compact,
              'touch-manipulation disabled:opacity-50',
            )}
          >
            {actionLoading === 'create' ? (
              t('configBackups.creating')
            ) : (
              <>
                <svg
                  class={iconTokens.size.sm}
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
                {t('configBackups.createBackup')}
              </>
            )}
          </button>

          {/* Error Message */}
          {error ? <p class="caption text-status-error">{error}</p> : null}

          {/* Backups List */}
          {renderBackupsContent()}

          <p class={cn('caption text-text-muted', spacing.margin.top.inline)}>
            {t('configBackups.description')}
          </p>
        </div>
      </CollapsibleSection>
    );
  },
);
