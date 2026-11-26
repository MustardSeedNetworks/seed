/**
 * Network configuration sub-section of SettingsDrawer.
 *
 * Contains the IP mode toggle (DHCP/static), static IP fields, the
 * apply button + status message, the public-IP display toggle, and
 * the VLAN/MTU controls. Pulled out of SettingsDrawer so the main
 * file stays manageable.
 */

import type React from 'react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../contexts/RoleContext';
import { CollapsibleSection } from '../../shared/CollapsibleSection';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Toggle } from '../../shared/Toggle';
import {
  button,
  cn,
  icon as iconTokens,
  radius,
  spacing,
  status as statusColor,
} from '../../styles/theme';
import type { DisplayOptions, IpSettings, SaveStatus } from '../../types/settings';
import { Network } from '../ui/Icons';
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
  ipMessage: string | null;
  displayOptions: DisplayOptions;
  setDisplayOptions: React.Dispatch<React.SetStateAction<DisplayOptions>>;
  displayStatus: SaveStatus;
  isValidIp: (ip: string) => boolean;
}

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
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');

  return (
    <CollapsibleSection
      readOnlyReason={readOnlyReason}
      title={
        <div className={cn('flex items-center', spacing.gap.compact)}>
          <Network className={iconTokens.size.sm} />
          <span>{t('sections.network')}</span>
        </div>
      }
    >
      {/* Network Configuration */}
      <div className="stack">
        <p className="section-title">{t('network.title')}</p>
        {/* Mode Toggle */}
        <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
          <button
            type="button"
            onClick={(): void => setIpSettings((prev) => ({ ...prev, mode: 'dhcp' }))}
            className={cn(
              spacing.tab,
              radius.md,
              'body-small font-medium transition-colors',
              ipSettings.mode === 'dhcp'
                ? 'bg-brand-primary text-on-brand'
                : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover',
            )}
          >
            {t('network.dhcp')}
          </button>
          <button
            type="button"
            onClick={(): void => setIpSettings((prev) => ({ ...prev, mode: 'static' }))}
            className={cn(
              spacing.tab,
              radius.md,
              'body-small font-medium transition-colors',
              ipSettings.mode === 'static'
                ? 'bg-brand-primary text-on-brand'
                : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover',
            )}
          >
            {t('network.static')}
          </button>
        </div>

        {/* Static IP Fields */}
        {ipSettings.mode === 'static' && (
          <div
            className={cn('stack', spacing.padding.top.heading, 'border-t border-surface-border')}
          >
            <FormField id="static-ip-address" label={t('network.ipAddress')} required>
              {(control) => (
                <Input
                  {...control}
                  aria-invalid={
                    ipSettings.address && !isValidIp(ipSettings.address)
                      ? true
                      : control['aria-invalid']
                  }
                  value={ipSettings.address}
                  onChange={(e): void =>
                    setIpSettings((prev) => ({ ...prev, address: e.target.value }))
                  }
                  placeholder="192.168.1.100"
                />
              )}
            </FormField>
            <FormField id="static-subnet-mask" label={t('network.subnetMask')} required>
              {(control) => (
                <Input
                  {...control}
                  value={ipSettings.netmask}
                  onChange={(e): void =>
                    setIpSettings((prev) => ({ ...prev, netmask: e.target.value }))
                  }
                  placeholder="24 or 255.255.255.0"
                />
              )}
            </FormField>
            <FormField id="static-gateway" label={t('network.gateway')}>
              {(control) => (
                <Input
                  {...control}
                  aria-invalid={
                    ipSettings.gateway && !isValidIp(ipSettings.gateway)
                      ? true
                      : control['aria-invalid']
                  }
                  value={ipSettings.gateway}
                  onChange={(e): void =>
                    setIpSettings((prev) => ({ ...prev, gateway: e.target.value }))
                  }
                  placeholder="192.168.1.1"
                />
              )}
            </FormField>
            <FormField id="static-dns-servers" label={t('network.dnsServers')}>
              {(control) => (
                <Input
                  {...control}
                  value={dnsInput}
                  onChange={(e): void => setDnsInput(e.target.value)}
                  placeholder="8.8.8.8, 8.8.4.4"
                />
              )}
            </FormField>
          </div>
        )}

        {/* Apply Button */}
        <button
          type="button"
          onClick={saveIpSettings}
          disabled={savingIp || (ipSettings.mode === 'static' && !ipSettings.address)}
          className={cn(
            'w-full',
            button.size.md,
            'bg-brand-primary text-on-brand',
            radius.md,
            'font-medium hover:bg-brand-accent disabled:control-disabled transition-colors',
          )}
        >
          {savingIp ? t('network.applying') : t('network.applyIpSettings')}
        </button>

        {ipMessage ? (
          <p
            className={cn(
              'caption text-center',
              ipMessage.includes('Failed') || ipMessage.includes('Error')
                ? statusColor.text.error
                : statusColor.text.success,
            )}
          >
            {ipMessage}
          </p>
        ) : null}

        <p className="caption">{t('network.requiresRoot')}</p>
      </div>
      {/* Display Options */}
      <div
        className={cn(
          'border-t border-surface-border',
          spacing.padding.top.heading,
          spacing.margin.top.heading,
        )}
      >
        <p className={cn('caption font-medium', spacing.margin.bottom.inline)}>
          {t('network.displayOptions')} <AutoSaveIndicator status={displayStatus} />
        </p>
        <div className="stack-sm">
          {/* Show Public IP */}
          <Toggle
            label={t('network.showPublicIp')}
            description={t('network.displayInNetworkCard')}
            checked={displayOptions.showPublicIp}
            onChange={(checked: boolean): void =>
              setDisplayOptions((prev) => ({
                ...prev,
                showPublicIp: checked,
              }))
            }
            className="rounded-lg border border-surface-border bg-surface-base p-3"
          />
        </div>
      </div>
      {/* VLAN Configuration */}
      <div
        className={cn(
          'border-t border-surface-border',
          spacing.padding.top.heading,
          spacing.margin.top.heading,
        )}
      >
        <p className={cn('section-title', spacing.margin.bottom.inline)}>{t('network.vlanTag')}</p>
        <VlanControl />
      </div>
      {/* MTU Configuration */}
      <div
        className={cn(
          'border-t border-surface-border',
          spacing.padding.top.heading,
          spacing.margin.top.heading,
        )}
      >
        <p className={cn('section-title', spacing.margin.bottom.inline)}>
          {t('network.mtuSetting')}
        </p>
        <MtuControl />
      </div>
    </CollapsibleSection>
  );
}
