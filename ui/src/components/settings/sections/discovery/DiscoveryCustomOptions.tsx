import type React from 'react';
import { memo, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
} from '../../../../styles/theme';
import type { NetworkDiscoverySettings } from '../../../../types/settings';
import { PORT_PRESETS } from './DiscoveryCustomOptions.constants';
import { DiscoveryPerformanceTiming } from './DiscoveryPerformanceTiming';
import { DiscoveryPortScanDetails } from './DiscoveryPortScanDetails';

interface DiscoveryCustomOptionsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Discovery scan method options.
 */
export const DiscoveryCustomOptions: React.NamedExoticComponent<DiscoveryCustomOptionsProps> = memo(
  function discoveryCustomOptions({ settings, onSettingsChange }: DiscoveryCustomOptionsProps) {
    const { t } = useTranslation('settings');

    // Auto-populate ports when preset changes (but not for custom)
    useEffect(() => {
      const preset = settings.options?.portScan?.preset ?? 'common';
      if (preset !== 'custom') {
        const presetConfig = PORT_PRESETS[preset];
        onSettingsChange((prev) => ({
          ...prev,
          options: {
            ...prev.options,
            portScan: {
              ...prev.options?.portScan,
              enabled: prev.options?.portScan?.enabled ?? false,
              preset,
              tcpPorts: presetConfig.tcp,
              udpPorts: presetConfig.udp,
              bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
            },
          },
        }));
      }
      // Only run when preset changes - onSettingsChange is stable from useCallback
    }, [settings.options?.portScan?.preset, onSettingsChange]);

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
                      udpPorts: prev.options?.portScan?.udpPorts ?? '53,161',
                      bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
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

          {/* TCP Probe Settings */}
          <div
            className={cn(
              'border-t border-surface-border',
              spacing.pad.sm,
              spacing.margin.top.inline,
            )}
          >
            <span className="caption text-text-muted font-medium">
              {t('discovery.tcpProbeSettings')}
            </span>
            <p className="caption text-text-muted">{t('discovery.tcpProbeDesc')}</p>
            <div className={cn('grid grid-cols-2', spacing.gap.compact, spacing.margin.top.inline)}>
              <div>
                <label className="caption text-text-muted" htmlFor="tcp-probe-timeout">
                  {t('discovery.tcpProbeTimeout')}
                </label>
                <input
                  id="tcp-probe-timeout"
                  type="number"
                  value={settings.options?.tcpProbe?.timeoutMs ?? 2000}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      options: {
                        ...prev.options,
                        tcpProbe: {
                          ...prev.options?.tcpProbe,
                          timeoutMs: Number.parseInt(e.target.value, 10) || 2000,
                          workers: prev.options?.tcpProbe?.workers ?? 20,
                        },
                      },
                    }))
                  }
                  min={100}
                  max={10000}
                  className={cn(
                    'w-full',
                    spacing.margin.top.tight,
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    'body-small',
                  )}
                />
              </div>
              <div>
                <label className="caption text-text-muted" htmlFor="tcp-probe-workers">
                  {t('discovery.tcpProbeWorkers')}
                </label>
                <input
                  id="tcp-probe-workers"
                  type="number"
                  value={settings.options?.tcpProbe?.workers ?? 20}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      options: {
                        ...prev.options,
                        tcpProbe: {
                          ...prev.options?.tcpProbe,
                          timeoutMs: prev.options?.tcpProbe?.timeoutMs ?? 2000,
                          workers: Number.parseInt(e.target.value, 10) || 20,
                        },
                      },
                    }))
                  }
                  min={1}
                  max={100}
                  className={cn(
                    'w-full',
                    spacing.margin.top.tight,
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    'body-small',
                  )}
                />
              </div>
            </div>
          </div>

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

          <DiscoveryPerformanceTiming settings={settings} onSettingsChange={onSettingsChange} />
        </div>
      </div>
    );
  },
);
