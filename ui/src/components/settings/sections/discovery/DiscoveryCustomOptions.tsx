import type React from "react";
import { memo, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
} from "../../../../styles/theme";
import type { NetworkDiscoverySettings, PortPreset } from "../../../../types/settings";

interface DiscoveryCustomOptionsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Port presets as defined in the plan:
 * - Common Services: OS/application/service identification
 * - Secure Ports: Encrypted/authenticated services
 * - Insecure Ports: Should probably be disabled if running
 * - Custom: User-defined ports
 */
const PORT_PRESETS: Record<PortPreset, { tcp: string; udp: string; description: string }> = {
  common: {
    tcp: "21,22,23,25,53,80,110,111,135,139,143,443,445,993,995,1433,1521,3306,3389,5432,5900,5985,8080,8443",
    udp: "53,67,68,69,123,137,138,161,162,500,514,1900",
    description: "Common service ports for OS/application identification",
  },
  secure: {
    tcp: "22,443,465,587,636,853,993,995,8443,9443",
    udp: "443,500,4500,853",
    description: "Encrypted and authenticated services (recommended)",
  },
  insecure: {
    tcp: "21,23,25,69,80,110,111,135,139,143,445,512,513,514,1099,2049,3389,5800,5900,6000-6009",
    udp: "67,68,69,111,137,138,161,162,514,1900,2049",
    description: "Insecure ports that should be disabled",
  },
  custom: {
    tcp: "",
    udp: "",
    description: "Manually configure port lists",
  },
};

/**
 * Discovery scan method options.
 */
export const DiscoveryCustomOptions = memo(function DiscoveryCustomOptions({
  settings,
  onSettingsChange,
}: DiscoveryCustomOptionsProps) {
  const { t } = useTranslation("settings");

  // Auto-populate ports when preset changes (but not for custom)
  useEffect(() => {
    const preset = settings.options?.portScan?.preset ?? "common";
    if (preset !== "custom") {
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
              "border border-surface-border",
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
          <span className="body-small text-text-primary">{t("discovery.arpScanning")}</span>
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
          <span className="body-small text-text-primary">{t("discovery.icmpPingSweep")}</span>
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
                    bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                  },
                },
              }))
            }
            className={iconTokens.size.sm}
          />
          <span className="body-small text-text-primary">{t("discovery.portScanning")}</span>
        </label>

        {/* Port Scan Details (shown when enabled) */}
        {settings.options?.portScan?.enabled && (
          <div
            className={cn(
              "ml-6 stack-sm",
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border",
            )}
          >
            <div>
              <label className="caption text-text-muted" htmlFor="port-scan-preset">
                {t("discovery.portScanPreset", "Port Preset")}
              </label>
              <select
                id="port-scan-preset"
                value={settings.options?.portScan?.preset ?? "common"}
                onChange={(e) => {
                  const newPreset = e.target.value as PortPreset;
                  const presetConfig = PORT_PRESETS[newPreset];
                  onSettingsChange((prev) => ({
                    ...prev,
                    options: {
                      ...prev.options,
                      portScan: {
                        ...prev.options?.portScan,
                        enabled: prev.options?.portScan?.enabled ?? false,
                        preset: newPreset,
                        // Auto-populate for non-custom presets
                        tcpPorts:
                          newPreset === "custom"
                            ? (prev.options?.portScan?.tcpPorts ?? "22,80,443")
                            : presetConfig.tcp,
                        udpPorts:
                          newPreset === "custom"
                            ? (prev.options?.portScan?.udpPorts ?? "53,161")
                            : presetConfig.udp,
                        bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                      },
                    },
                  }));
                }}
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small",
                )}
              >
                <option value="common">{t("discovery.portPresetCommon", "Common Services")}</option>
                <option value="secure">{t("discovery.portPresetSecure", "Secure Ports")}</option>
                <option value="insecure">
                  {t("discovery.portPresetInsecure", "Insecure Ports")}
                </option>
                <option value="custom">{t("discovery.portPresetCustom", "Custom")}</option>
              </select>
              {/* Description for selected preset */}
              <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
                {PORT_PRESETS[settings.options?.portScan?.preset ?? "common"].description}
              </p>
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="port-scan-tcp">
                {t("discovery.portScanTcpPorts", "TCP Ports")}
                {(settings.options?.portScan?.preset ?? "common") !== "custom" && (
                  <span className="ml-2 text-text-muted italic">
                    {t("discovery.portPresetReadOnly", "(read-only, select Custom to edit)")}
                  </span>
                )}
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
                        bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                      },
                    },
                  }))
                }
                placeholder="22,80,443,8080-8100"
                readOnly={(settings.options?.portScan?.preset ?? "common") !== "custom"}
                disabled={(settings.options?.portScan?.preset ?? "common") !== "custom"}
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  (settings.options?.portScan?.preset ?? "common") !== "custom"
                    ? "bg-surface-muted cursor-not-allowed opacity-60"
                    : inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small",
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="port-scan-udp">
                {t("discovery.portScanUdpPorts", "UDP Ports")}
                {(settings.options?.portScan?.preset ?? "common") !== "custom" && (
                  <span className="ml-2 text-text-muted italic">
                    {t("discovery.portPresetReadOnly", "(read-only, select Custom to edit)")}
                  </span>
                )}
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
                        tcpPorts: prev.options?.portScan?.tcpPorts ?? "22,80,443",
                        udpPorts: e.target.value,
                        bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                      },
                    },
                  }))
                }
                placeholder="53,123,161"
                readOnly={(settings.options?.portScan?.preset ?? "common") !== "custom"}
                disabled={(settings.options?.portScan?.preset ?? "common") !== "custom"}
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  inputTokens.base,
                  (settings.options?.portScan?.preset ?? "common") !== "custom"
                    ? "bg-surface-muted cursor-not-allowed opacity-60"
                    : inputTokens.state.default,
                  inputTokens.size.sm,
                  "body-small",
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="port-scan-banner">
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
                        tcpPorts: prev.options?.portScan?.tcpPorts ?? "22,80,443",
                        udpPorts: prev.options?.portScan?.udpPorts ?? "53,161",
                        bannerTimeoutMs: Number.parseInt(e.target.value, 10) || 2000,
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
                  "body-small",
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
            spacing.margin.top.inline,
          )}
        >
          <span className="caption text-text-muted font-medium">
            {t("discovery.tcpProbeSettings", "TCP Probe Settings")}
          </span>
          <p className="caption text-text-muted">
            {t(
              "discovery.tcpProbeDesc",
              "Configure TCP connection probing for device detection and service discovery",
            )}
          </p>
          <div className={cn("grid grid-cols-2", spacing.gap.compact, spacing.margin.top.inline)}>
            <div>
              <label className="caption text-text-muted" htmlFor="tcp-probe-timeout">
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
                        timeoutMs: Number.parseInt(e.target.value, 10) || 2000,
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
                  "body-small",
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="tcp-probe-workers">
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
                        workers: Number.parseInt(e.target.value, 10) || 20,
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
                  "body-small",
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
          <span className="body-small text-text-primary">{t("discovery.traceroute")}</span>
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
          <span className="body-small text-text-primary">{t("discovery.snmpQueries")}</span>
        </label>

        {/* Performance & Timing Section */}
        <div
          className={cn(
            "border-t border-surface-border",
            spacing.pad.sm,
            spacing.margin.top.inline,
          )}
        >
          <span className="caption text-text-muted font-medium">
            {t("discovery.performanceTiming", "Performance & Timing")}
          </span>
          <p className="caption text-text-muted">
            {t("discovery.performanceTimingDesc", "Adjust discovery speed and resource usage")}
          </p>
          <div className={cn("stack-sm", spacing.margin.top.inline)}>
            {/* Probe Interval Slider */}
            <div>
              <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
                <label htmlFor="probe-interval-slider" className="caption text-text-muted">
                  {t("discovery.probeInterval", "Probe Interval")}
                </label>
                <span className="caption text-text-primary font-medium">
                  {settings.timing?.probeIntervalMs ?? 75}ms
                </span>
              </div>
              <input
                id="probe-interval-slider"
                type="range"
                min={25}
                max={500}
                step={25}
                value={settings.timing?.probeIntervalMs ?? 75}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    timing: {
                      ...prev.timing,
                      probeIntervalMs: Number.parseInt(e.target.value, 10),
                      rescanIntervalMs: prev.timing?.rescanIntervalMs ?? 600000,
                      workers: prev.timing?.workers ?? 50,
                    },
                  }))
                }
                className="w-full"
              />
              <div
                className={cn(
                  layout.flex.between,
                  "caption text-text-muted",
                  spacing.margin.top.tight,
                )}
              >
                <span>{t("discovery.slower", "Slower")}</span>
                <span>{t("discovery.faster", "Faster")}</span>
              </div>
            </div>

            {/* Scan Timeout Slider */}
            <div>
              <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
                <label htmlFor="scan-timeout-slider" className="caption text-text-muted">
                  {t("discovery.scanTimeout", "Scan Timeout")}
                </label>
                <span className="caption text-text-primary font-medium">
                  {settings.scanTimeoutMs ?? 2000}ms
                </span>
              </div>
              <input
                id="scan-timeout-slider"
                type="range"
                min={500}
                max={10000}
                step={500}
                value={settings.scanTimeoutMs ?? 2000}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    scanTimeoutMs: Number.parseInt(e.target.value, 10),
                  }))
                }
                className="w-full"
              />
              <div
                className={cn(
                  layout.flex.between,
                  "caption text-text-muted",
                  spacing.margin.top.tight,
                )}
              >
                <span>500ms</span>
                <span>10s</span>
              </div>
            </div>

            {/* Workers Slider */}
            <div>
              <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
                <label htmlFor="workers-slider" className="caption text-text-muted">
                  {t("discovery.workers", "Workers")}
                </label>
                <span className="caption text-text-primary font-medium">
                  {settings.timing?.workers ?? 20}
                </span>
              </div>
              <input
                id="workers-slider"
                type="range"
                min={5}
                max={100}
                step={5}
                value={settings.timing?.workers ?? 20}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    timing: {
                      ...prev.timing,
                      probeIntervalMs: prev.timing?.probeIntervalMs ?? 75,
                      rescanIntervalMs: prev.timing?.rescanIntervalMs ?? 600000,
                      workers: Number.parseInt(e.target.value, 10),
                    },
                  }))
                }
                className="w-full"
              />
              <div
                className={cn(
                  layout.flex.between,
                  "caption text-text-muted",
                  spacing.margin.top.tight,
                )}
              >
                <span>{t("discovery.gentler", "Gentler")}</span>
                <span>{t("discovery.aggressive", "Aggressive")}</span>
              </div>
            </div>

            {/* Rescan Interval Slider */}
            <div>
              <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
                <label htmlFor="rescan-interval-slider" className="caption text-text-muted">
                  {t("discovery.rescanInterval", "Rescan Interval")}
                </label>
                <span className="caption text-text-primary font-medium">
                  {Math.round((settings.timing?.rescanIntervalMs ?? 600000) / 60000)}m
                </span>
              </div>
              <input
                id="rescan-interval-slider"
                type="range"
                min={60}
                max={3600}
                step={60}
                value={(settings.timing?.rescanIntervalMs ?? 600000) / 1000}
                onChange={(e) =>
                  onSettingsChange((prev) => ({
                    ...prev,
                    timing: {
                      ...prev.timing,
                      probeIntervalMs: prev.timing?.probeIntervalMs ?? 75,
                      rescanIntervalMs: Number.parseInt(e.target.value, 10) * 1000,
                      workers: prev.timing?.workers ?? 50,
                    },
                  }))
                }
                className="w-full"
              />
              <div
                className={cn(
                  layout.flex.between,
                  "caption text-text-muted",
                  spacing.margin.top.tight,
                )}
              >
                <span>1m</span>
                <span>60m</span>
              </div>
            </div>

            {/* Banner Timeout Slider (only shown when port scanning is enabled) */}
            {settings.options?.portScan?.enabled && (
              <div>
                <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
                  <label htmlFor="banner-timeout-slider" className="caption text-text-muted">
                    {t("discovery.bannerTimeout", "Banner Timeout")}
                  </label>
                  <span className="caption text-text-primary font-medium">
                    {settings.options?.portScan?.bannerTimeoutMs ?? 2000}ms
                  </span>
                </div>
                <input
                  id="banner-timeout-slider"
                  type="range"
                  min={500}
                  max={10000}
                  step={500}
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
                          tcpPorts: prev.options?.portScan?.tcpPorts ?? "22,80,443",
                          udpPorts: prev.options?.portScan?.udpPorts ?? "53,161",
                          bannerTimeoutMs: Number.parseInt(e.target.value, 10),
                        },
                      },
                    }))
                  }
                  className="w-full"
                />
                <div
                  className={cn(
                    layout.flex.between,
                    "caption text-text-muted",
                    spacing.margin.top.tight,
                  )}
                >
                  <span>500ms</span>
                  <span>10s</span>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
});
