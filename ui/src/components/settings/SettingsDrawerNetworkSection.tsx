/**
 * Network configuration sub-section of SettingsDrawer.
 *
 * Contains the IP mode toggle (DHCP/static), static IP fields, the
 * apply button + status message, the public-IP display toggle, and
 * the VLAN/MTU controls. Pulled out of SettingsDrawer so the main
 * file stays manageable.
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { button, cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import type { DisplayOptions, IpSettings, SaveStatus } from '../../types/settings';
import { CollapsibleSection } from '../ui/CollapsibleSection';
import { Network } from '../ui/icons';
import { AutoSaveIndicator } from './sections/AutoSaveIndicator';
import { MtuControl } from './sections/MtuControl';
import { VlanControl } from './sections/VlanControl';

interface SettingsDrawerNetworkSectionProps {
  ipSettings: IpSettings;
  setIpSettings: React.Dispatch<React.SetStateAction<IpSettings>>;
  dnsInput: string;
  setDnsInput: React.Dispatch<React.SetStateAction<string>>;
  saveIpSettings: () => Promise<void>;
  savingIp: boolean;
  ipMessage: string;
  displayOptions: DisplayOptions;
  setDisplayOptions: React.Dispatch<React.SetStateAction<DisplayOptions>>;
  displayStatus: SaveStatus;
  isValidIp: (ip: string) => boolean;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Renders five sub-blocks (mode toggle, static fields, apply, display options, VLAN/MTU); mirrors original inline structure
export function SettingsDrawerNetworkSection({
  ipSettings,
  setIpSettings,
  dnsInput,
  setDnsInput,
  saveIpSettings,
  savingIp,
  ipMessage,
  displayOptions,
  setDisplayOptions,
  displayStatus,
  isValidIp,
}: SettingsDrawerNetworkSectionProps): JSX.Element {
  const { t } = useTranslation('settings');

  return (
    <CollapsibleSection
      title={
        <div class={cn('flex items-center', spacing.gap.compact)}>
          <Network class={iconTokens.size.sm} />
          <span>{t('sections.network')}</span>
        </div>
      }
    >
      {/* Network Configuration */}
      <div class="stack">
        <p class="section-title">{t('network.title')}</p>
        {/* Mode Toggle */}
        <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
          <button
            type="button"
            onClick={(): void => setIpSettings((prev) => ({ ...prev, mode: 'dhcp' }))}
            class={cn(
              spacing.tab,
              radius.md,
              'body-small font-medium transition-colors',
              ipSettings.mode === 'dhcp'
                ? 'bg-brand-primary text-text-inverse'
                : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover',
            )}
          >
            {t('network.dhcp')}
          </button>
          <button
            type="button"
            onClick={(): void => setIpSettings((prev) => ({ ...prev, mode: 'static' }))}
            class={cn(
              spacing.tab,
              radius.md,
              'body-small font-medium transition-colors',
              ipSettings.mode === 'static'
                ? 'bg-brand-primary text-text-inverse'
                : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover',
            )}
          >
            {t('network.static')}
          </button>
        </div>

        {/* Static IP Fields */}
        {ipSettings.mode === 'static' && (
          <div class={cn('stack', spacing.padding.top.heading, 'border-t border-surface-border')}>
            <div>
              <label for="static-ip-address" class="caption font-medium">
                {t('network.ipAddress')} *
              </label>
              <input
                id="static-ip-address"
                type="text"
                value={ipSettings.address}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setIpSettings((prev) => ({
                    ...prev,
                    address: e.target.value,
                  }))
                }
                placeholder="192.168.1.100"
                class={cn(
                  'w-full',
                  spacing.margin.top.tight,
                  spacing.chip.sm,
                  'bg-surface-base border',
                  radius.md,
                  'body-small text-text-primary',
                  ipSettings.address && !isValidIp(ipSettings.address)
                    ? 'border-status-error'
                    : 'border-surface-border',
                )}
              />
            </div>
            <div>
              <label for="static-subnet-mask" class="caption font-medium">
                {t('network.subnetMask')} *
              </label>
              <input
                id="static-subnet-mask"
                type="text"
                value={ipSettings.netmask}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setIpSettings((prev) => ({
                    ...prev,
                    netmask: e.target.value,
                  }))
                }
                placeholder="24 or 255.255.255.0"
                class={cn(
                  'w-full',
                  spacing.margin.top.tight,
                  spacing.chip.lg,
                  'bg-surface-base border border-surface-border',
                  radius.md,
                  'body-small text-text-primary',
                )}
              />
            </div>
            <div>
              <label for="static-gateway" class="caption font-medium">
                {t('network.gateway')}
              </label>
              <input
                id="static-gateway"
                type="text"
                value={ipSettings.gateway}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setIpSettings((prev) => ({
                    ...prev,
                    gateway: e.target.value,
                  }))
                }
                placeholder="192.168.1.1"
                class={cn(
                  'w-full',
                  spacing.margin.top.tight,
                  spacing.chip.sm,
                  'bg-surface-base border',
                  radius.md,
                  'body-small text-text-primary',
                  ipSettings.gateway && !isValidIp(ipSettings.gateway)
                    ? 'border-status-error'
                    : 'border-surface-border',
                )}
              />
            </div>
            <div>
              <label for="static-dns-servers" class="caption font-medium">
                {t('network.dnsServers')}
              </label>
              <input
                id="static-dns-servers"
                type="text"
                value={dnsInput}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setDnsInput(e.target.value)
                }
                placeholder="8.8.8.8, 8.8.4.4"
                class={cn(
                  'w-full',
                  spacing.margin.top.tight,
                  spacing.chip.lg,
                  'bg-surface-base border border-surface-border',
                  radius.md,
                  'body-small text-text-primary',
                )}
              />
            </div>
          </div>
        )}

        {/* Apply Button */}
        <button
          type="button"
          onClick={saveIpSettings}
          disabled={savingIp || (ipSettings.mode === 'static' && !ipSettings.address)}
          class={cn(
            'w-full',
            button.size.md,
            'bg-brand-primary text-text-inverse',
            radius.md,
            'font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors',
          )}
        >
          {savingIp ? t('network.applying') : t('network.applyIpSettings')}
        </button>

        {ipMessage ? (
          <p
            class={cn(
              'caption text-center',
              ipMessage.includes('Failed') || ipMessage.includes('Error')
                ? 'text-status-error'
                : 'text-status-success',
            )}
          >
            {ipMessage}
          </p>
        ) : null}

        <p class="caption">{t('network.requiresRoot')}</p>
      </div>

      {/* Display Options */}
      <div
        class={cn(
          'border-t border-surface-border',
          spacing.padding.top.heading,
          spacing.margin.top.heading,
        )}
      >
        <p class={cn('caption font-medium', spacing.margin.bottom.inline)}>
          {t('network.displayOptions')} <AutoSaveIndicator status={displayStatus} />
        </p>
        <div class="stack-sm">
          {/* Show Public IP */}
          <label
            class={cn(
              'flex items-center justify-between',
              spacing.pad.xs,
              'bg-surface-base',
              radius.md,
              'border border-surface-border',
            )}
          >
            <div>
              <span class="body-small text-text-primary font-medium">
                {t('network.showPublicIp')}
              </span>
              <p class="caption text-text-muted">{t('network.displayInNetworkCard')}</p>
            </div>
            <input
              type="checkbox"
              checked={displayOptions.showPublicIp}
              onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                setDisplayOptions((prev) => ({
                  ...prev,
                  showPublicIp: e.target.checked,
                }))
              }
              class={iconTokens.size.sm}
            />
          </label>
        </div>
      </div>

      {/* VLAN Configuration */}
      <div
        class={cn(
          'border-t border-surface-border',
          spacing.padding.top.heading,
          spacing.margin.top.heading,
        )}
      >
        <p class={cn('section-title', spacing.margin.bottom.inline)}>{t('network.vlanTag')}</p>
        <VlanControl />
      </div>

      {/* MTU Configuration */}
      <div
        class={cn(
          'border-t border-surface-border',
          spacing.padding.top.heading,
          spacing.margin.top.heading,
        )}
      >
        <p class={cn('section-title', spacing.margin.bottom.inline)}>{t('network.mtuSetting')}</p>
        <MtuControl />
      </div>
    </CollapsibleSection>
  );
}
