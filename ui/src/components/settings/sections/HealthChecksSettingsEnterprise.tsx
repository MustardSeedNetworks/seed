/**
 * Enterprise protocol endpoint sub-sections of HealthChecksSettings.
 *
 * Renders the SQL / file-share (SMB/NFS) / LDAP endpoint editors. Each
 * owns its own useArrayItem CRUD helpers so the parent
 * HealthChecksSettings only forwards testsSettings + setter.
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { useArrayItem } from '../../../hooks/useArrayItem';
import { cn, input, layout, radius, spacing } from '../../../styles/theme';
import type { TestsSettings } from '../../../types/settings';

interface HealthChecksSettingsEnterpriseProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Renders three optional protocol editors (SQL, file share, LDAP); matches original inline structure
export function HealthChecksSettingsEnterprise({
  testsSettings,
  setTestsSettings,
}: HealthChecksSettingsEnterpriseProps): JSX.Element {
  const { t } = useTranslation('settings');

  const {
    add: addSqlEndpoint,
    remove: removeSqlEndpoint,
    update: updateSqlEndpoint,
  } = useArrayItem(setTestsSettings, 'sqlEndpoints', () => ({
    name: '',
    driver: 'postgres' as const,
    host: '',
    port: 5432,
    database: '',
    username: '',
    enabled: true,
    criticality: 7,
  }));

  const {
    add: addFileShareEndpoint,
    remove: removeFileShareEndpoint,
    update: updateFileShareEndpoint,
  } = useArrayItem(setTestsSettings, 'fileShareEndpoints', () => ({
    name: '',
    protocol: 'smb' as const,
    host: '',
    sharePath: '',
    enabled: true,
    criticality: 5,
  }));

  const {
    add: addLdapEndpoint,
    remove: removeLdapEndpoint,
    update: updateLdapEndpoint,
  } = useArrayItem(setTestsSettings, 'ldapEndpoints', () => ({
    name: '',
    host: '',
    port: 389,
    useTls: false,
    baseDn: '',
    enabled: true,
    criticality: 7,
  }));

  return (
    <>
      {/* SQL Database Endpoints */}
      <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span class="caption text-text-muted font-medium">{t('health.sqlEndpoints')}</span>
          <button
            type="button"
            onClick={addSqlEndpoint}
            class="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.sqlDescription')}
        </p>
        {(testsSettings.sqlEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            class={cn(
              spacing.stack.xs,
              spacing.margin.bottom.heading,
              spacing.pad.xs,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div class={cn('flex', spacing.gap.compact)}>
              <input
                type="text"
                value={endpoint.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateSqlEndpoint(endpoint.id ?? '', 'name', e.target.value)
                }
                placeholder={t('common.name')}
                class={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'flex-1 bg-surface-raised',
                )}
              />
              <select
                value={endpoint.driver}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateSqlEndpoint(endpoint.id ?? '', 'driver', e.target.value)
                }
                class={cn(input.base, input.state.default, input.size.md, 'w-28 bg-surface-raised')}
              >
                <option value="postgres">PostgreSQL</option>
                <option value="mysql">MySQL</option>
                <option value="mssql">SQL Server</option>
                <option value="oracle">Oracle</option>
              </select>
              <button
                type="button"
                onClick={(): void => removeSqlEndpoint(endpoint.id ?? '')}
                class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
              >
                {t('common.remove')}
              </button>
            </div>
            <div class={cn('flex', spacing.gap.compact)}>
              <input
                type="text"
                value={endpoint.host}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateSqlEndpoint(endpoint.id ?? '', 'host', e.target.value)
                }
                placeholder={t('common.host')}
                class={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'flex-1 bg-surface-raised',
                )}
              />
              <input
                type="number"
                value={endpoint.port}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateSqlEndpoint(endpoint.id ?? '', 'port', Number.parseInt(e.target.value, 10))
                }
                placeholder={t('common.port')}
                class={cn(input.base, input.state.default, input.size.md, 'w-20 bg-surface-raised')}
              />
            </div>
            <div class={cn('flex', spacing.gap.compact)}>
              <input
                type="text"
                value={endpoint.database}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateSqlEndpoint(endpoint.id ?? '', 'database', e.target.value)
                }
                placeholder={t('health.database')}
                class={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'flex-1 bg-surface-raised',
                )}
              />
              <input
                type="text"
                value={endpoint.username}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateSqlEndpoint(endpoint.id ?? '', 'username', e.target.value)
                }
                placeholder={t('health.username')}
                class={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'flex-1 bg-surface-raised',
                )}
              />
            </div>
          </div>
        ))}
      </div>

      {/* File Share Endpoints (SMB/NFS) */}
      <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span class="caption text-text-muted font-medium">{t('health.fileShareEndpoints')}</span>
          <button
            type="button"
            onClick={addFileShareEndpoint}
            class="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.fileShareDescription')}
        </p>
        {(testsSettings.fileShareEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <input
              type="text"
              value={endpoint.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFileShareEndpoint(endpoint.id ?? '', 'name', e.target.value)
              }
              placeholder={t('common.name')}
              class={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <select
              value={endpoint.protocol}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFileShareEndpoint(endpoint.id ?? '', 'protocol', e.target.value)
              }
              class={cn(input.base, input.state.default, input.size.md, 'w-20')}
            >
              <option value="smb">SMB</option>
              <option value="nfs">NFS</option>
            </select>
            <input
              type="text"
              value={endpoint.host}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFileShareEndpoint(endpoint.id ?? '', 'host', e.target.value)
              }
              placeholder={t('common.host')}
              class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <input
              type="text"
              value={endpoint.sharePath}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFileShareEndpoint(endpoint.id ?? '', 'sharePath', e.target.value)
              }
              placeholder={t('health.sharePath')}
              class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <button
              type="button"
              onClick={(): void => removeFileShareEndpoint(endpoint.id ?? '')}
              class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
            >
              {t('common.remove')}
            </button>
          </div>
        ))}
      </div>

      {/* LDAP Endpoints */}
      <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span class="caption text-text-muted font-medium">{t('health.ldapEndpoints')}</span>
          <button
            type="button"
            onClick={addLdapEndpoint}
            class="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.ldapDescription')}
        </p>
        {(testsSettings.ldapEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            class={cn(
              spacing.stack.xs,
              spacing.margin.bottom.heading,
              spacing.pad.xs,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div class={cn('flex', spacing.gap.compact)}>
              <input
                type="text"
                value={endpoint.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateLdapEndpoint(endpoint.id ?? '', 'name', e.target.value)
                }
                placeholder={t('common.name')}
                class={cn(input.base, input.state.default, input.size.md, 'w-32 bg-surface-raised')}
              />
              <input
                type="text"
                value={endpoint.host}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateLdapEndpoint(endpoint.id ?? '', 'host', e.target.value)
                }
                placeholder={t('common.host')}
                class={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'flex-1 bg-surface-raised',
                )}
              />
              <input
                type="number"
                value={endpoint.port}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateLdapEndpoint(endpoint.id ?? '', 'port', Number.parseInt(e.target.value, 10))
                }
                placeholder={t('common.port')}
                class={cn(input.base, input.state.default, input.size.md, 'w-20 bg-surface-raised')}
              />
              <button
                type="button"
                onClick={(): void => removeLdapEndpoint(endpoint.id ?? '')}
                class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
              >
                {t('common.remove')}
              </button>
            </div>
            <div class={cn('flex items-center', spacing.gap.compact)}>
              <input
                type="text"
                value={endpoint.baseDn}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateLdapEndpoint(endpoint.id ?? '', 'baseDn', e.target.value)
                }
                placeholder={t('health.baseDn')}
                class={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'flex-1 bg-surface-raised',
                )}
              />
              <label
                class={cn('flex items-center', spacing.gap.compact, 'caption text-text-muted')}
              >
                <input
                  type="checkbox"
                  checked={endpoint.useTls}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateLdapEndpoint(endpoint.id ?? '', 'useTls', e.target.checked)
                  }
                />
                TLS
              </label>
            </div>
          </div>
        ))}
      </div>
    </>
  );
}
