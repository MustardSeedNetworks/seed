import type React from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../../styles/theme';
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
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={settings.options?.passiveProtocols?.lldp ?? true}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      options: {
                        ...prev.options,
                        passiveProtocols: {
                          ...prev.options?.passiveProtocols,
                          lldp: e.target.checked,
                          cdp: prev.options?.passiveProtocols?.cdp ?? true,
                          edp: prev.options?.passiveProtocols?.edp ?? true,
                          ndp: prev.options?.passiveProtocols?.ndp ?? true,
                        },
                      },
                    }))
                  }
                  className={iconTokens.size.xs}
                />
                <span className="caption text-text-primary">LLDP</span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={settings.options?.passiveProtocols?.cdp ?? true}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      options: {
                        ...prev.options,
                        passiveProtocols: {
                          ...prev.options?.passiveProtocols,
                          lldp: prev.options?.passiveProtocols?.lldp ?? true,
                          cdp: e.target.checked,
                          edp: prev.options?.passiveProtocols?.edp ?? true,
                          ndp: prev.options?.passiveProtocols?.ndp ?? true,
                        },
                      },
                    }))
                  }
                  className={iconTokens.size.xs}
                />
                <span className="caption text-text-primary">CDP</span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={settings.options?.passiveProtocols?.edp ?? true}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      options: {
                        ...prev.options,
                        passiveProtocols: {
                          ...prev.options?.passiveProtocols,
                          lldp: prev.options?.passiveProtocols?.lldp ?? true,
                          cdp: prev.options?.passiveProtocols?.cdp ?? true,
                          edp: e.target.checked,
                          ndp: prev.options?.passiveProtocols?.ndp ?? true,
                        },
                      },
                    }))
                  }
                  className={iconTokens.size.xs}
                />
                <span className="caption text-text-primary">EDP</span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={settings.options?.passiveProtocols?.ndp ?? true}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      options: {
                        ...prev.options,
                        passiveProtocols: {
                          ...prev.options?.passiveProtocols,
                          lldp: prev.options?.passiveProtocols?.lldp ?? true,
                          cdp: prev.options?.passiveProtocols?.cdp ?? true,
                          edp: prev.options?.passiveProtocols?.edp ?? true,
                          ndp: e.target.checked,
                        },
                      },
                    }))
                  }
                  className={iconTokens.size.xs}
                />
                <span className="caption text-text-primary">NDP</span>
              </label>
            </div>
          </div>
        </div>

        {/* ARP Scanning */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.arpScan ?? true}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  arpScan: e.target.checked,
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">{t('discovery.arpScanning')}</span>
        </label>

        {/* ICMP Ping Sweep */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.icmpScan ?? true}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  icmpScan: e.target.checked,
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">{t('discovery.icmpPingSweep')}</span>
        </label>

        {/* Port Scanning */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.portScan?.enabled ?? false}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  portScan: {
                    ...prev.options?.portScan,
                    enabled: e.target.checked,
                    preset: prev.options?.portScan?.preset ?? 'common',
                    tcpPorts: prev.options?.portScan?.tcpPorts ?? '22,80,443',
                  },
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">{t('discovery.portScanning')}</span>
        </label>

        {/* Port Scan Details (shown when enabled) */}
        {settings.options?.portScan?.enabled ? (
          <DiscoveryPortScanDetails settings={settings} onSettingsChange={onSettingsChange} />
        ) : null}

        {/* Traceroute */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.traceroute ?? false}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  traceroute: e.target.checked,
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">{t('discovery.traceroute')}</span>
        </label>

        {/* SNMP Queries */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.snmpQuery ?? false}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  snmpQuery: e.target.checked,
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">{t('discovery.snmpQueries')}</span>
        </label>
      </div>
    </div>
  );
}
