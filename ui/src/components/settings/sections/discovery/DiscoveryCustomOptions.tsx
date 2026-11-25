import type React from 'react';
import { useTranslation } from 'react-i18next';
import { Checkbox } from '../../../../shared/Checkbox';
import { cn, radius, spacing } from '../../../../styles/theme';
import type { NetworkDiscoverySettings } from '../../../../types/settings';
import { DiscoveryPortScanDetails } from './DiscoveryPortScanDetails';

interface DiscoveryCustomOptionsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Discovery scan method options.
 */
export function DiscoveryCustomOptions({
  settings,
  onSettingsChange,
}: DiscoveryCustomOptionsProps) {
  const { t } = useTranslation('settings');

  return (
    <div className={cn('border-t border-surface-border', spacing.pad.sm)}>
      <span className="caption text-text-muted font-medium">{t('discovery.scanMethods')}</span>
      <div className={cn(spacing.margin.top.inline, 'stack-sm')}>
        {/* Passive Protocol Details */}
        <div>
          <span className="body-small text-text-primary font-medium">
            {t('discovery.passiveProtocols')}
          </span>
          <div
            className={cn(
              'ml-spacious',
              spacing.pad.xs,
              spacing.margin.top.tight,
              'bg-surface-base',
              radius.default,
              'border border-surface-border',
            )}
          >
            <div className={cn('flex flex-wrap', spacing.gap.compact)}>
              <Checkbox
                label="LLDP"
                checked={settings.options?.passiveProtocols?.lldp ?? true}
                onChange={(checked: boolean): void =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      passiveProtocols: {
                        ...prev.options?.passiveProtocols,
                        lldp: checked,
                        cdp: prev.options?.passiveProtocols?.cdp ?? true,
                        edp: prev.options?.passiveProtocols?.edp ?? true,
                        ndp: prev.options?.passiveProtocols?.ndp ?? true,
                      },
                    },
                  }))
                }
              />
              <Checkbox
                label="CDP"
                checked={settings.options?.passiveProtocols?.cdp ?? true}
                onChange={(checked: boolean): void =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      passiveProtocols: {
                        ...prev.options?.passiveProtocols,
                        lldp: prev.options?.passiveProtocols?.lldp ?? true,
                        cdp: checked,
                        edp: prev.options?.passiveProtocols?.edp ?? true,
                        ndp: prev.options?.passiveProtocols?.ndp ?? true,
                      },
                    },
                  }))
                }
              />
              <Checkbox
                label="EDP"
                checked={settings.options?.passiveProtocols?.edp ?? true}
                onChange={(checked: boolean): void =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      passiveProtocols: {
                        ...prev.options?.passiveProtocols,
                        lldp: prev.options?.passiveProtocols?.lldp ?? true,
                        cdp: prev.options?.passiveProtocols?.cdp ?? true,
                        edp: checked,
                        ndp: prev.options?.passiveProtocols?.ndp ?? true,
                      },
                    },
                  }))
                }
              />
              <Checkbox
                label="NDP"
                checked={settings.options?.passiveProtocols?.ndp ?? true}
                onChange={(checked: boolean): void =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      passiveProtocols: {
                        ...prev.options?.passiveProtocols,
                        lldp: prev.options?.passiveProtocols?.lldp ?? true,
                        cdp: prev.options?.passiveProtocols?.cdp ?? true,
                        edp: prev.options?.passiveProtocols?.edp ?? true,
                        ndp: checked,
                      },
                    },
                  }))
                }
              />
            </div>
          </div>
        </div>

        {/* ARP Scanning */}
        <Checkbox
          label={t('discovery.arpScanning')}
          checked={settings.options?.arpScan ?? true}
          onChange={(checked: boolean): void =>
            onSettingsChange((prev) => ({
              ...prev,
              options: {
                ...prev.options,
                arpScan: checked,
              },
            }))
          }
        />

        {/* ICMP Ping Sweep */}
        <Checkbox
          label={t('discovery.icmpPingSweep')}
          checked={settings.options?.icmpScan ?? true}
          onChange={(checked: boolean): void =>
            onSettingsChange((prev) => ({
              ...prev,
              options: {
                ...prev.options,
                icmpScan: checked,
              },
            }))
          }
        />

        {/* Port Scanning */}
        <Checkbox
          label={t('discovery.portScanning')}
          checked={settings.options?.portScan?.enabled ?? false}
          onChange={(checked: boolean): void =>
            onSettingsChange((prev) => ({
              ...prev,
              options: {
                ...prev.options,
                portScan: {
                  ...prev.options?.portScan,
                  enabled: checked,
                  preset: prev.options?.portScan?.preset ?? 'common',
                  tcpPorts: prev.options?.portScan?.tcpPorts ?? '22,80,443',
                },
              },
            }))
          }
        />

        {/* Port Scan Details (shown when enabled) */}
        {settings.options?.portScan?.enabled ? (
          <DiscoveryPortScanDetails settings={settings} onSettingsChange={onSettingsChange} />
        ) : null}

        {/* Traceroute */}
        <Checkbox
          label={t('discovery.traceroute')}
          checked={settings.options?.traceroute ?? false}
          onChange={(checked: boolean): void =>
            onSettingsChange((prev) => ({
              ...prev,
              options: {
                ...prev.options,
                traceroute: checked,
              },
            }))
          }
        />

        {/* SNMP Queries */}
        <Checkbox
          label={t('discovery.snmpQueries')}
          checked={settings.options?.snmpQuery ?? false}
          onChange={(checked: boolean): void =>
            onSettingsChange((prev) => ({
              ...prev,
              options: {
                ...prev.options,
                snmpQuery: checked,
              },
            }))
          }
        />
      </div>
    </div>
  );
}
