/**
 * Enterprise protocol endpoint sub-sections of HealthChecksSettings.
 *
 * Renders the SQL / file-share (SMB/NFS) / LDAP endpoint editors. Each
 * owns its own useArrayItem CRUD helpers so the parent
 * HealthChecksSettings only forwards testsSettings + setter.
 */

import type React from 'react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useArrayItem } from '../../../hooks/useArrayItem';
import { Checkbox } from '../../../shared/Checkbox';
import { Input } from '../../../shared/Input';
import { Select } from '../../../shared/Select';
import { cn, layout, radius, spacing } from '../../../styles/theme';
import type { SqlEndpoint, TestsSettings } from '../../../types/settings';

// Product names, written the same in every locale.
const SQL_DRIVERS: ReadonlyArray<{ value: SqlEndpoint['driver']; label: string }> = [
  { value: 'postgres', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'mssql', label: 'SQL Server' },
  { value: 'oracle', label: 'Oracle' },
];

interface HealthChecksSettingsEnterpriseProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

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
    enabled: true,
  }));

  const {
    add: addFileShareEndpoint,
    remove: removeFileShareEndpoint,
    update: updateFileShareEndpoint,
  } = useArrayItem(setTestsSettings, 'fileShareEndpoints', () => ({
    name: '',
    protocol: 'smb' as const,
    host: '',
    share: '',
    enabled: true,
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
    startTls: false,
    baseDn: '',
    enabled: true,
  }));

  return (
    <>
      {/* SQL Database Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.sqlEndpoints')}</span>
          <button
            type="button"
            onClick={addSqlEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.sqlDescription')}
        </p>
        {(testsSettings.sqlEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn(
              spacing.stack.xs,
              spacing.margin.bottom.heading,
              spacing.pad.xs,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div className={cn('flex', spacing.gap.compact)}>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('common.name')}
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateSqlEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                />
              </div>
              <div className="w-32 shrink-0">
                <Select
                  aria-label={t('health.sqlDriver')}
                  value={endpoint.driver}
                  onValueChange={(value): void =>
                    updateSqlEndpoint(endpoint.id ?? '', 'driver', value as SqlEndpoint['driver'])
                  }
                  options={SQL_DRIVERS}
                />
              </div>
              <button
                type="button"
                onClick={(): void => removeSqlEndpoint(endpoint.id ?? '')}
                className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
              >
                {t('common.remove')}
              </button>
            </div>
            <div className={cn('flex', spacing.gap.compact)}>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('common.host')}
                  value={endpoint.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateSqlEndpoint(endpoint.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                />
              </div>
              <div className="w-20 shrink-0">
                <Input
                  type="number"
                  aria-label={t('common.port')}
                  value={endpoint.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateSqlEndpoint(
                      endpoint.id ?? '',
                      'port',
                      Number.parseInt(e.target.value, 10),
                    )
                  }
                  placeholder={t('common.port')}
                />
              </div>
            </div>
            <div className={cn('flex', spacing.gap.compact)}>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('health.database')}
                  value={endpoint.database}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateSqlEndpoint(endpoint.id ?? '', 'database', e.target.value)
                  }
                  placeholder={t('health.database')}
                />
              </div>
            </div>
          </div>
        ))}
      </div>
      {/* File Share Endpoints (SMB/NFS) */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">
            {t('health.fileShareEndpoints')}
          </span>
          <button
            type="button"
            onClick={addFileShareEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.fileShareDescription')}
        </p>
        {(testsSettings.fileShareEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <div className="w-24 shrink-0">
              <Input
                type="text"
                aria-label={t('common.name')}
                value={endpoint.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateFileShareEndpoint(endpoint.id ?? '', 'name', e.target.value)
                }
                placeholder={t('common.name')}
              />
            </div>
            <div className="w-24 shrink-0">
              <Select
                aria-label={t('health.fileShareProtocol')}
                value={endpoint.protocol}
                onValueChange={(value): void =>
                  updateFileShareEndpoint(endpoint.id ?? '', 'protocol', value as 'smb' | 'nfs')
                }
                options={[
                  { value: 'smb', label: 'SMB' },
                  { value: 'nfs', label: 'NFS' },
                ]}
              />
            </div>
            <div className="min-w-0 flex-1">
              <Input
                type="text"
                aria-label={t('common.host')}
                value={endpoint.host}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateFileShareEndpoint(endpoint.id ?? '', 'host', e.target.value)
                }
                placeholder={t('common.host')}
              />
            </div>
            <div className="min-w-0 flex-1">
              <Input
                type="text"
                aria-label={t('health.sharePath')}
                value={endpoint.share}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateFileShareEndpoint(endpoint.id ?? '', 'share', e.target.value)
                }
                placeholder={t('health.sharePath')}
              />
            </div>
            <button
              type="button"
              onClick={(): void => removeFileShareEndpoint(endpoint.id ?? '')}
              className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
            >
              {t('common.remove')}
            </button>
          </div>
        ))}
      </div>
      {/* LDAP Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.ldapEndpoints')}</span>
          <button
            type="button"
            onClick={addLdapEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.ldapDescription')}
        </p>
        {(testsSettings.ldapEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn(
              spacing.stack.xs,
              spacing.margin.bottom.heading,
              spacing.pad.xs,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div className={cn('flex', spacing.gap.compact)}>
              <div className="w-32 shrink-0">
                <Input
                  type="text"
                  aria-label={t('common.name')}
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateLdapEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                />
              </div>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('common.host')}
                  value={endpoint.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateLdapEndpoint(endpoint.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                />
              </div>
              <div className="w-20 shrink-0">
                <Input
                  type="number"
                  aria-label={t('common.port')}
                  value={endpoint.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateLdapEndpoint(
                      endpoint.id ?? '',
                      'port',
                      Number.parseInt(e.target.value, 10),
                    )
                  }
                  placeholder={t('common.port')}
                />
              </div>
              <button
                type="button"
                onClick={(): void => removeLdapEndpoint(endpoint.id ?? '')}
                className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
              >
                {t('common.remove')}
              </button>
            </div>
            <div className={cn('flex items-center', spacing.gap.compact)}>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('health.baseDn')}
                  value={endpoint.baseDn}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateLdapEndpoint(endpoint.id ?? '', 'baseDn', e.target.value)
                  }
                  placeholder={t('health.baseDn')}
                />
              </div>
              <Checkbox
                label={<>TLS</>}
                checked={endpoint.useTls}
                onChange={(checked: boolean): void =>
                  updateLdapEndpoint(endpoint.id ?? '', 'useTls', checked)
                }
              />
            </div>
          </div>
        ))}
      </div>
    </>
  );
}
