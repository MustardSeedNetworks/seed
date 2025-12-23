import { memo } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  layout,
  spacing,
  radius,
  icon as iconTokens,
  input as inputTokens,
} from "../../../../styles/theme";
import type { NetworkDiscoverySettings } from "../../../../types/settings";

interface DiscoveryCustomOptionsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<
    React.SetStateAction<NetworkDiscoverySettings>
  >;
}

/**
 * Discovery scan method options.
 */
export const DiscoveryCustomOptions = memo(function DiscoveryCustomOptions({
  settings,
  onSettingsChange,
}: DiscoveryCustomOptionsProps) {
  const { t } = useTranslation("settings");

  return (
    <div className={cn("border-t border-surface-border", spacing.pad.sm)}>
      <span className="caption text-text-muted font-medium">
        {t("discovery.scanMethods", "Scan Methods")}
      </span>
      <div className={cn(spacing.margin.top.inline, "stack-sm")}>
        {/* Passive Protocol Details */}
        <div>
          <span className="body-small text-text-primary font-medium">
            {t("discovery.passiveProtocols", "Passive Protocols")}
          </span>
          <div
            className={cn(
              "ml-6",
              spacing.pad.xs,
              spacing.margin.top.tight,
              "bg-surface-base",
              radius.default,
              "border border-surface-border"
            )}
          >
            <div className={cn("flex flex-wrap", spacing.gap.compact)}>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={settings.options?.passiveProtocols?.lldp ?? true}
                  onChange={(e) =>
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
                  onChange={(e) =>
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
                  onChange={(e) =>
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
                  onChange={(e) =>
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
            onChange={(e) =>
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
          <span className="body-small text-text-primary">
            {t("discovery.arpScanning")}
          </span>
        </label>

        {/* ICMP Ping Sweep */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.icmpScan ?? true}
            onChange={(e) =>
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
          <span className="body-small text-text-primary">
            {t("discovery.icmpPingSweep")}
          </span>
        </label>

        {/* Port Scanning */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.portScan?.enabled ?? false}
            onChange={(e) =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  portScan: {
                    ...prev.options?.portScan,
                    enabled: e.target.checked,
                    preset: prev.options?.portScan?.preset ?? "common",
                    tcpPorts: prev.options?.portScan?.tcpPorts ?? "22,80,443",
                    udpPorts: prev.options?.portScan?.udpPorts ?? "53,161",
                    bannerTimeoutMs:
                      prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                  },
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">
            {t("discovery.portScanning")}
          </span>
        </label>

        {/* Port Scan Details (shown when enabled) */}
        {settings.options?.portScan?.enabled && (
          <div
            className={cn(
              "ml-6 stack-sm",
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border"
            )}
          >
            <div>
              <label
                className="caption text-text-muted"
                htmlFor="port-scan-preset"
              >
                {t("discovery.portScanPreset", "Port Preset")}
              </label>
              <select
                id="port-scan-preset"
                value={settings.options?.portScan?.preset ?? "common"}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      portScan: {
                        ...prev.options?.portScan,
                        enabled: prev.options?.portScan?.enabled ?? false,
                        preset: e.target.value as
                          | "common"
                          | "secure"
                          | "insecure"
                          | "custom",
                        tcpPorts:
                          prev.options?.portScan?.tcpPorts ?? "22,80,443",
                        udpPorts: prev.options?.portScan?.udpPorts ?? "53,161",
                        bannerTimeoutMs:
                          prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                      },
                    },
                  }))
                }
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small"
                )}
              >
                <option value="common">
                  {t("discovery.portPresetCommon", "Common Services")}
                </option>
                <option value="secure">
                  {t("discovery.portPresetSecure", "Secure Ports")}
                </option>
                <option value="insecure">
                  {t("discovery.portPresetInsecure", "Insecure Ports")}
                </option>
                <option value="custom">
                  {t("discovery.portPresetCustom", "Custom")}
                </option>
              </select>
            </div>
            <div>
              <label
                className="caption text-text-muted"
                htmlFor="port-scan-tcp"
              >
                {t("discovery.portScanTcpPorts", "TCP Ports")}
              </label>
              <input
                id="port-scan-tcp"
                type="text"
                value={settings.options?.portScan?.tcpPorts ?? "22,80,443"}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      portScan: {
                        ...prev.options?.portScan,
                        enabled: prev.options?.portScan?.enabled ?? false,
                        preset: prev.options?.portScan?.preset ?? "common",
                        tcpPorts: e.target.value,
                        udpPorts: prev.options?.portScan?.udpPorts ?? "53,161",
                        bannerTimeoutMs:
                          prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                      },
                    },
                  }))
                }
                placeholder="22,80,443,8080-8100"
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small"
                )}
              />
            </div>
            <div>
              <label
                className="caption text-text-muted"
                htmlFor="port-scan-udp"
              >
                {t("discovery.portScanUdpPorts", "UDP Ports")}
              </label>
              <input
                id="port-scan-udp"
                type="text"
                value={settings.options?.portScan?.udpPorts ?? "53,161"}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      portScan: {
                        ...prev.options?.portScan,
                        enabled: prev.options?.portScan?.enabled ?? false,
                        preset: prev.options?.portScan?.preset ?? "common",
                        tcpPorts:
                          prev.options?.portScan?.tcpPorts ?? "22,80,443",
                        udpPorts: e.target.value,
                        bannerTimeoutMs:
                          prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                      },
                    },
                  }))
                }
                placeholder="53,123,161"
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small"
                )}
              />
            </div>
            <div>
              <label
                className="caption text-text-muted"
                htmlFor="port-scan-banner"
              >
                {t("discovery.portScanBannerTimeout", "Banner Timeout (ms)")}
              </label>
              <input
                id="port-scan-banner"
                type="number"
                value={settings.options?.portScan?.bannerTimeoutMs ?? 2000}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      portScan: {
                        ...prev.options?.portScan,
                        enabled: prev.options?.portScan?.enabled ?? false,
                        preset: prev.options?.portScan?.preset ?? "common",
                        tcpPorts:
                          prev.options?.portScan?.tcpPorts ?? "22,80,443",
                        udpPorts: prev.options?.portScan?.udpPorts ?? "53,161",
                        bannerTimeoutMs: parseInt(e.target.value) || 2000,
                      },
                    },
                  }))
                }
                min={100}
                max={10000}
                className={cn(
                  "w-24",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small"
                )}
              />
            </div>
          </div>
        )}

        {/* TCP Probe Settings */}
        <div
          className={cn(
            "border-t border-surface-border",
            spacing.pad.sm,
            spacing.margin.top.inline
          )}
        >
          <span className="caption text-text-muted font-medium">
            {t("discovery.tcpProbeSettings", "TCP Probe Settings")}
          </span>
          <p className="caption text-text-muted">
            {t(
              "discovery.tcpProbeDesc",
              "Configure TCP connection probing for device detection and service discovery"
            )}
          </p>
          <div
            className={cn(
              "grid grid-cols-2",
              spacing.gap.compact,
              spacing.margin.top.inline
            )}
          >
            <div>
              <label
                className="caption text-text-muted"
                htmlFor="tcp-probe-timeout"
              >
                {t("discovery.tcpProbeTimeout", "Timeout (ms)")}
              </label>
              <input
                id="tcp-probe-timeout"
                type="number"
                value={settings.options?.tcpProbe?.timeoutMs ?? 2000}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      tcpProbe: {
                        ...prev.options?.tcpProbe,
                        timeoutMs: parseInt(e.target.value) || 2000,
                        workers: prev.options?.tcpProbe?.workers ?? 20,
                      },
                    },
                  }))
                }
                min={100}
                max={10000}
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small"
                )}
              />
            </div>
            <div>
              <label
                className="caption text-text-muted"
                htmlFor="tcp-probe-workers"
              >
                {t("discovery.tcpProbeWorkers", "Workers")}
              </label>
              <input
                id="tcp-probe-workers"
                type="number"
                value={settings.options?.tcpProbe?.workers ?? 20}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      tcpProbe: {
                        ...prev.options?.tcpProbe,
                        timeoutMs: prev.options?.tcpProbe?.timeoutMs ?? 2000,
                        workers: parseInt(e.target.value) || 20,
                      },
                    },
                  }))
                }
                min={1}
                max={100}
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small"
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
            onChange={(e) =>
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
          <span className="body-small text-text-primary">
            {t("discovery.traceroute")}
          </span>
        </label>

        {/* SNMP Queries */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.options?.snmpQuery ?? false}
            onChange={(e) =>
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
          <span className="body-small text-text-primary">
            {t("discovery.snmpQueries")}
          </span>
        </label>
      </div>
    </div>
  );
});
