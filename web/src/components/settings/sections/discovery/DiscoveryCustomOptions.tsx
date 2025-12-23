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
 * Custom discovery method options.
 * Only displayed when "custom" profile is selected.
 */
export const DiscoveryCustomOptions = memo(function DiscoveryCustomOptions({
  settings,
  onSettingsChange,
}: DiscoveryCustomOptionsProps) {
  const { t } = useTranslation("settings");

  return (
    <div className={cn("border-t border-surface-border", spacing.pad.sm)}>
      <span className="caption text-text-muted font-medium">
        {t("discovery.customOptions")}
      </span>
      <div className={cn(spacing.margin.top.inline, "stack-sm")}>
        {/* Passive Listeners */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.customOptions?.passiveListen ?? true}
            onChange={(e) =>
              onSettingsChange((prev) => ({
                ...prev,
                customOptions: {
                  ...prev.customOptions,
                  passiveListen: e.target.checked,
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">
            {t("discovery.passiveListeners")}
          </span>
        </label>

        {/* Passive Protocol Details (shown when enabled) */}
        {settings.customOptions?.passiveListen && (
          <div
            className={cn(
              "ml-6",
              spacing.pad.xs,
              "bg-surface-base",
              radius.default,
              "border border-surface-border"
            )}
          >
            <span className="caption text-text-muted">
              {t("discovery.passiveProtocols", "Protocols")}:
            </span>
            <div
              className={cn(
                "flex flex-wrap",
                spacing.gap.compact,
                spacing.margin.top.tight
              )}
            >
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={
                    settings.customOptions?.passiveProtocols?.lldp ?? true
                  }
                  onChange={(e) =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        passiveProtocols: {
                          ...prev.customOptions?.passiveProtocols,
                          lldp: e.target.checked,
                          cdp:
                            prev.customOptions?.passiveProtocols?.cdp ?? true,
                          edp:
                            prev.customOptions?.passiveProtocols?.edp ?? true,
                          ndp:
                            prev.customOptions?.passiveProtocols?.ndp ?? true,
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
                  checked={
                    settings.customOptions?.passiveProtocols?.cdp ?? true
                  }
                  onChange={(e) =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        passiveProtocols: {
                          ...prev.customOptions?.passiveProtocols,
                          lldp:
                            prev.customOptions?.passiveProtocols?.lldp ?? true,
                          cdp: e.target.checked,
                          edp:
                            prev.customOptions?.passiveProtocols?.edp ?? true,
                          ndp:
                            prev.customOptions?.passiveProtocols?.ndp ?? true,
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
                  checked={
                    settings.customOptions?.passiveProtocols?.edp ?? true
                  }
                  onChange={(e) =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        passiveProtocols: {
                          ...prev.customOptions?.passiveProtocols,
                          lldp:
                            prev.customOptions?.passiveProtocols?.lldp ?? true,
                          cdp:
                            prev.customOptions?.passiveProtocols?.cdp ?? true,
                          edp: e.target.checked,
                          ndp:
                            prev.customOptions?.passiveProtocols?.ndp ?? true,
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
                  checked={
                    settings.customOptions?.passiveProtocols?.ndp ?? true
                  }
                  onChange={(e) =>
                    onSettingsChange((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        passiveProtocols: {
                          ...prev.customOptions?.passiveProtocols,
                          lldp:
                            prev.customOptions?.passiveProtocols?.lldp ?? true,
                          cdp:
                            prev.customOptions?.passiveProtocols?.cdp ?? true,
                          edp:
                            prev.customOptions?.passiveProtocols?.edp ?? true,
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
        )}

        {/* ARP Scanning */}
        <label className={layout.inline.default}>
          <input
            type="checkbox"
            checked={settings.customOptions?.arpScan ?? true}
            onChange={(e) =>
              onSettingsChange((prev) => ({
                ...prev,
                customOptions: {
                  ...prev.customOptions,
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
            checked={settings.customOptions?.icmpScan ?? true}
            onChange={(e) =>
              onSettingsChange((prev) => ({
                ...prev,
                customOptions: {
                  ...prev.customOptions,
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
            checked={settings.customOptions?.portScan?.enabled ?? false}
            onChange={(e) =>
              onSettingsChange((prev) => ({
                ...prev,
                customOptions: {
                  ...prev.customOptions,
                  portScan: {
                    ...prev.customOptions?.portScan,
                    enabled: e.target.checked,
                    tcpPorts:
                      prev.customOptions?.portScan?.tcpPorts ?? "22,80,443",
                    udpPorts:
                      prev.customOptions?.portScan?.udpPorts ?? "53,161",
                    bannerTimeoutMs:
                      prev.customOptions?.portScan?.bannerTimeoutMs ?? 2000,
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
        {settings.customOptions?.portScan?.enabled && (
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
                htmlFor="port-scan-tcp"
              >
                {t("discovery.portScanTcpPorts", "TCP Ports")}
              </label>
              <input
                id="port-scan-tcp"
                type="text"
                value={
                  settings.customOptions?.portScan?.tcpPorts ?? "22,80,443"
                }
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    customOptions: {
                      ...prev.customOptions,
                      portScan: {
                        ...prev.customOptions?.portScan,
                        enabled: prev.customOptions?.portScan?.enabled ?? false,
                        tcpPorts: e.target.value,
                        udpPorts:
                          prev.customOptions?.portScan?.udpPorts ?? "53,161",
                        bannerTimeoutMs:
                          prev.customOptions?.portScan?.bannerTimeoutMs ?? 2000,
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
                value={settings.customOptions?.portScan?.udpPorts ?? "53,161"}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    customOptions: {
                      ...prev.customOptions,
                      portScan: {
                        ...prev.customOptions?.portScan,
                        enabled: prev.customOptions?.portScan?.enabled ?? false,
                        tcpPorts:
                          prev.customOptions?.portScan?.tcpPorts ?? "22,80,443",
                        udpPorts: e.target.value,
                        bannerTimeoutMs:
                          prev.customOptions?.portScan?.bannerTimeoutMs ?? 2000,
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
                value={
                  settings.customOptions?.portScan?.bannerTimeoutMs ?? 2000
                }
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    customOptions: {
                      ...prev.customOptions,
                      portScan: {
                        ...prev.customOptions?.portScan,
                        enabled: prev.customOptions?.portScan?.enabled ?? false,
                        tcpPorts:
                          prev.customOptions?.portScan?.tcpPorts ?? "22,80,443",
                        udpPorts:
                          prev.customOptions?.portScan?.udpPorts ?? "53,161",
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
                value={settings.customOptions?.tcpProbe?.timeoutMs ?? 2000}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    customOptions: {
                      ...prev.customOptions,
                      tcpProbe: {
                        ...prev.customOptions?.tcpProbe,
                        timeoutMs: parseInt(e.target.value) || 2000,
                        workers: prev.customOptions?.tcpProbe?.workers ?? 20,
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
                value={settings.customOptions?.tcpProbe?.workers ?? 20}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    customOptions: {
                      ...prev.customOptions,
                      tcpProbe: {
                        ...prev.customOptions?.tcpProbe,
                        timeoutMs:
                          prev.customOptions?.tcpProbe?.timeoutMs ?? 2000,
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
            checked={settings.customOptions?.traceroute ?? false}
            onChange={(e) =>
              onSettingsChange((prev) => ({
                ...prev,
                customOptions: {
                  ...prev.customOptions,
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
            checked={settings.customOptions?.snmpQuery ?? false}
            onChange={(e) =>
              onSettingsChange((prev) => ({
                ...prev,
                customOptions: {
                  ...prev.customOptions,
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
