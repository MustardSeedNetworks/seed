/**
 * DnsSettings Component
 *
 * Purpose: Allows users to configure custom DNS servers for testing and specify
 * test hostnames and other DNS test parameters.
 *
 * Key Features:
 * - Multiple DNS servers: add/remove custom DNS server addresses
 * - Enable/disable per-server: toggle which servers to test
 * - Test hostname: configurable hostname for DNS resolution testing
 * - IPv6 support: separate options for IPv4 and IPv6 queries
 * - CRUD operations: add new servers, remove existing, update addresses
 * - AutoSaveIndicator: shows save status while persisting changes
 * - Globe icon: visual indicator in settings menu
 * - ID generation: unique IDs for server entries
 *
 * Usage:
 * ```typescript
 * <DnsSettings
 *   testsSettings={settings}
 *   setTestsSettings={updateSettings}
 *   testsStatus={saveStatus}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, Globe icon, utilities for ID generation
 * State: Receives test settings and save status from parent, callbacks for updates
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { Toggle } from '../../../shared/Toggle';
import {
  cn,
  icon as iconTokens,
  layout,
  spacing,
  status as statusColor,
} from '../../../styles/theme';
import type { CardSettings, DnsServer, SaveStatus, TestsSettings } from '../../../types/settings';
import { generateId } from '../../../utils/id';
import { Globe } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

interface DnsSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
  /** Card settings for visibility and FAB configuration */
  cardSettings: CardSettings;
  /** Update card settings (triggers auto-save to profile) */
  updateCardSettings: (updates: Partial<CardSettings>) => void;
}

export function DnsSettings({
  testsSettings,
  setTestsSettings,
  testsStatus,
  cardSettings,
  updateCardSettings,
}: DnsSettingsProps): React.ReactElement {
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');

  const addDnsServer = (): void => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: [...prev.dnsServers, { id: generateId(), address: '', enabled: true }],
    }));
  };

  const removeDnsServer = (id: string): void => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: prev.dnsServers.filter((s) => s.id !== id),
    }));
  };

  const updateDnsServer = (id: string, field: keyof DnsServer, value: string | boolean): void => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: prev.dnsServers.map((s) => (s.id === id ? { ...s, [field]: value } : s)),
    }));
  };

  return (
    <CollapsibleSection
      readOnlyReason={readOnlyReason}
      data-testid="dns-settings-section"
      title={
        <div className={layout.inline.default}>
          <Globe className={iconTokens.size.sm} />
          <span>{t('sections.dns')}</span>
          <AutoSaveIndicator status={testsStatus} />
        </div>
      }
    >
      <div className="stack">
        {/* Card Visibility & FAB Controls */}
        <div className="stack-sm">
          <Toggle
            label={t('common.showCard')}
            description={t('common.showCardDesc')}
            checked={cardSettings.dns.enabled}
            onChange={(checked: boolean): void =>
              updateCardSettings({
                dns: { ...cardSettings.dns, enabled: checked },
              })
            }
            className="rounded-lg border border-surface-border bg-surface-base p-3"
          />
          <Toggle
            label={t('common.runOnFab')}
            description={t('common.runOnFabDesc')}
            checked={cardSettings.dns.autoRunOnLink}
            onChange={(checked: boolean): void =>
              updateCardSettings({
                dns: { ...cardSettings.dns, autoRunOnLink: checked },
              })
            }
            className="rounded-lg border border-surface-border bg-surface-base p-3"
          />
        </div>

        {/* DNS Hostname */}
        <FormField
          id="dns-test-hostname"
          label={t('dns.testHostname')}
          hint={t('dns.testHostnameDesc')}
        >
          {(control) => (
            <Input
              {...control}
              value={testsSettings.dnsHostname}
              onChange={(e): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  dnsHostname: e.target.value,
                }))
              }
              placeholder="google.com"
            />
          )}
        </FormField>

        {/* DNS Servers for per-server testing */}
        <div className={cn('border-t', 'border-surface-border', spacing.padding.top.heading)}>
          <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">
              {t('dns.additionalServers')}
            </span>
            <button
              type="button"
              onClick={addDnsServer}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t('common.add')}
            </button>
          </div>
          <p className={cn('caption', 'text-text-muted', spacing.margin.bottom.inline)}>
            {t('dns.serversDescription')}
          </p>
          {testsSettings.dnsServers.map((server) => (
            <div
              key={server.id || server.address}
              className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
            >
              <Input
                value={server.address}
                onChange={(e): void => updateDnsServer(server.id ?? '', 'address', e.target.value)}
                placeholder={t('dns.serverIp')}
                aria-label={t('dns.serverIp')}
                className="flex-1"
              />
              <button
                type="button"
                onClick={(): void => removeDnsServer(server.id ?? '')}
                className={cn(
                  statusColor.text.error,
                  'hover:text-status-error/80',
                  spacing.actionBtn,
                )}
                aria-label={t('common.remove')}
              >
                {t('common.remove')}
              </button>
            </div>
          ))}
        </div>
      </div>
    </CollapsibleSection>
  );
}
