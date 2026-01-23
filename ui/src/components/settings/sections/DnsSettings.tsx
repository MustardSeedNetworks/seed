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
import { memo, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
} from '../../../styles/theme';
import type { CardSettings, DnsServer, SaveStatus, TestsSettings } from '../../../types/settings';
import { generateId } from '../../../utils/id';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Globe } from '../../ui/icons';
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

export const DnsSettings: React.NamedExoticComponent<DnsSettingsProps> = memo(
  function DnsSettingsComponent({
    testsSettings,
    setTestsSettings,
    testsStatus,
    cardSettings,
    updateCardSettings,
  }: DnsSettingsProps): React.ReactElement {
    const { t } = useTranslation('settings');

    const addDnsServer = useCallback((): void => {
      setTestsSettings((prev) => ({
        ...prev,
        dnsServers: [...prev.dnsServers, { id: generateId(), address: '', enabled: true }],
      }));
    }, [setTestsSettings]);

    const removeDnsServer = useCallback(
      (id: string): void => {
        setTestsSettings((prev) => ({
          ...prev,
          dnsServers: prev.dnsServers.filter((s) => s.id !== id),
        }));
      },
      [setTestsSettings],
    );

    const updateDnsServer = useCallback(
      (id: string, field: keyof DnsServer, value: string | boolean): void => {
        setTestsSettings((prev) => ({
          ...prev,
          dnsServers: prev.dnsServers.map((s) => (s.id === id ? { ...s, [field]: value } : s)),
        }));
      },
      [setTestsSettings],
    );

    return (
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <Globe class={iconTokens.size.sm} />
            <span>{t('sections.dns')}</span>
            <AutoSaveIndicator status={testsStatus} />
          </div>
        }
      >
        <div class="stack">
          {/* Card Visibility & FAB Controls */}
          <div class="stack-sm">
            <label
              class={cn(
                layout.flex.between,
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div>
                <span class="body-small text-text-primary font-medium">
                  {t('common.showCard', 'Show Card')}
                </span>
                <p class="caption text-text-muted">
                  {t('common.showCardDesc', 'Display this card on the dashboard')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.dns.enabled}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    dns: { ...cardSettings.dns, enabled: e.target.checked },
                  })
                }
                class={iconTokens.size.sm}
              />
            </label>
            <label
              class={cn(
                layout.flex.between,
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div>
                <span class="body-small text-text-primary font-medium">
                  {t('common.runOnFab', 'Include in Run All')}
                </span>
                <p class="caption text-text-muted">
                  {t('common.runOnFabDesc', 'Run when FAB button is clicked')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.dns.autoRunOnLink}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    dns: { ...cardSettings.dns, autoRunOnLink: e.target.checked },
                  })
                }
                class={iconTokens.size.sm}
              />
            </label>
          </div>

          {/* DNS Hostname */}
          <div>
            <label for="dns-test-hostname" class="caption text-text-muted">
              {t('dns.testHostname')}
            </label>
            <input
              id="dns-test-hostname"
              type="text"
              value={testsSettings.dnsHostname}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  dnsHostname: e.target.value,
                }))
              }
              placeholder="google.com"
              class={cn(
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.md,
                'w-full',
                spacing.margin.top.tight,
                'body-small',
              )}
            />
            <p class={cn('caption', 'text-text-muted', spacing.margin.top.tight)}>
              {t('dns.testHostnameDesc')}
            </p>
          </div>

          {/* DNS Servers for per-server testing */}
          <div class={cn('border-t', 'border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('dns.additionalServers')}</span>
              <button
                type="button"
                onClick={addDnsServer}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption', 'text-text-muted', spacing.margin.bottom.inline)}>
              {t('dns.serversDescription')}
            </p>
            {testsSettings.dnsServers.map((server) => (
              <div
                key={server.id || server.address}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={server.address}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateDnsServer(server.id ?? '', 'address', e.target.value)
                  }
                  placeholder={t('dns.serverIp')}
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.md,
                    'flex-1',
                    'caption',
                  )}
                />
                <button
                  type="button"
                  onClick={(): void => removeDnsServer(server.id ?? '')}
                  class={cn('text-status-error', 'hover:text-status-error/80', spacing.actionBtn)}
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
  },
);
