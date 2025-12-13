import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Tooltip } from "../../ui/Tooltip";
import { THRESHOLD_HELP } from "../../help/HelpContent";
import { SettingsThresholds, SaveStatus } from "../../../types/settings";
import { Info, SlidersHorizontal } from "../../ui/Icons";

interface ThresholdsSettingsProps {
  thresholds: SettingsThresholds;
  setThresholds: React.Dispatch<React.SetStateAction<SettingsThresholds>>;
  thresholdsStatus: SaveStatus;
}

export function ThresholdsSettings({
  thresholds,
  setThresholds,
  thresholdsStatus,
}: ThresholdsSettingsProps) {
  const updateThreshold = (
    category: keyof Omit<SettingsThresholds, "httpTimings">,
    level: "good" | "warning",
    value: number,
  ) => {
    setThresholds((prev) => ({
      ...prev,
      [category]: {
        ...prev[category],
        [level]: value,
      },
    }));
  };

  const updateHttpTimingThreshold = (
    phase: keyof SettingsThresholds["httpTimings"],
    level: "good" | "warning",
    value: number,
  ) => {
    setThresholds((prev) => ({
      ...prev,
      httpTimings: {
        ...prev.httpTimings,
        [phase]: {
          ...prev.httpTimings[phase],
          [level]: value,
        },
      },
    }));
  };

  return (
    <CollapsibleSection
      title={
        <div className="flex items-center gap-2">
          <SlidersHorizontal className="w-4 h-4" />
          <span>Thresholds</span>
          <AutoSaveIndicator status={thresholdsStatus} />
        </div>
      }
    >
      <div className="space-y-3">
        {/* DNS Thresholds */}
        <div className="p-3 bg-surface-base rounded border border-surface-border">
          <div className="flex items-center gap-1.5 mb-2">
            <span className="text-sm font-medium text-text-primary">
              DNS Lookup (ms)
            </span>
            <Tooltip content={THRESHOLD_HELP["DNS Lookup"]} position="top">
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-xs text-text-muted">Good (&lt;)</label>
              <input
                type="number"
                value={thresholds.dns.good}
                onChange={(e) =>
                  updateThreshold("dns", "good", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
            <div>
              <label className="text-xs text-text-muted">Warning (&lt;)</label>
              <input
                type="number"
                value={thresholds.dns.warning}
                onChange={(e) =>
                  updateThreshold("dns", "warning", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
          </div>
        </div>

        {/* Gateway Thresholds */}
        <div className="p-3 bg-surface-base rounded border border-surface-border">
          <div className="flex items-center gap-1.5 mb-2">
            <span className="text-sm font-medium text-text-primary">
              Gateway Ping (ms)
            </span>
            <Tooltip content={THRESHOLD_HELP["Gateway Ping"]} position="top">
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-xs text-text-muted">Good (&lt;)</label>
              <input
                type="number"
                value={thresholds.gateway.good}
                onChange={(e) =>
                  updateThreshold("gateway", "good", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
            <div>
              <label className="text-xs text-text-muted">Warning (&lt;)</label>
              <input
                type="number"
                value={thresholds.gateway.warning}
                onChange={(e) =>
                  updateThreshold("gateway", "warning", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
          </div>
        </div>

        {/* Wi-Fi Signal Thresholds */}
        <div className="p-3 bg-surface-base rounded border border-surface-border">
          <div className="flex items-center gap-1.5 mb-2">
            <span className="text-sm font-medium text-text-primary">
              Wi-Fi Signal (dBm)
            </span>
            <Tooltip content={THRESHOLD_HELP["Wi-Fi Signal"]} position="top">
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-xs text-text-muted">Good (&gt;)</label>
              <input
                type="number"
                value={thresholds.wifi.good}
                onChange={(e) =>
                  updateThreshold("wifi", "good", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
            <div>
              <label className="text-xs text-text-muted">Warning (&gt;)</label>
              <input
                type="number"
                value={thresholds.wifi.warning}
                onChange={(e) =>
                  updateThreshold("wifi", "warning", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
          </div>
        </div>

        {/* Health Check Ping Thresholds */}
        <div className="p-3 bg-surface-base rounded border border-surface-border">
          <div className="flex items-center gap-1.5 mb-2">
            <span className="text-sm font-medium text-text-primary">
              Health Check: Ping (ms)
            </span>
            <Tooltip
              content={THRESHOLD_HELP["Health Check: Ping"]}
              position="top"
            >
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-xs text-text-muted">Good (&lt;)</label>
              <input
                type="number"
                value={thresholds.customPing.good}
                onChange={(e) =>
                  updateThreshold("customPing", "good", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
            <div>
              <label className="text-xs text-text-muted">Warning (&lt;)</label>
              <input
                type="number"
                value={thresholds.customPing.warning}
                onChange={(e) =>
                  updateThreshold(
                    "customPing",
                    "warning",
                    Number(e.target.value),
                  )
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
          </div>
        </div>

        {/* Health Check TCP Thresholds */}
        <div className="p-3 bg-surface-base rounded border border-surface-border">
          <div className="flex items-center gap-1.5 mb-2">
            <span className="text-sm font-medium text-text-primary">
              Health Check: TCP (ms)
            </span>
            <Tooltip
              content={THRESHOLD_HELP["Health Check: TCP"]}
              position="top"
            >
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-xs text-text-muted">Good (&lt;)</label>
              <input
                type="number"
                value={thresholds.customTcp.good}
                onChange={(e) =>
                  updateThreshold("customTcp", "good", Number(e.target.value))
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
            <div>
              <label className="text-xs text-text-muted">Warning (&lt;)</label>
              <input
                type="number"
                value={thresholds.customTcp.warning}
                onChange={(e) =>
                  updateThreshold(
                    "customTcp",
                    "warning",
                    Number(e.target.value),
                  )
                }
                className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
              />
            </div>
          </div>
        </div>

        {/* HTTP Thresholds (Total + Timing Phases) */}
        <div className="p-3 bg-surface-base rounded border border-surface-border">
          <span className="text-sm font-medium text-text-primary block mb-2">
            HTTP Thresholds (ms)
          </span>

          {/* Total */}
          <div className="mb-3">
            <div className="flex items-center gap-1.5 mb-1">
              <span className="text-xs font-medium text-text-primary">
                Total Response Time
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP Total"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-xs text-text-muted">Good (&lt;)</label>
                <input
                  type="number"
                  value={thresholds.customHttp.good}
                  onChange={(e) =>
                    updateThreshold(
                      "customHttp",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
              <div>
                <label className="text-xs text-text-muted">
                  Warning (&lt;)
                </label>
                <input
                  type="number"
                  value={thresholds.customHttp.warning}
                  onChange={(e) =>
                    updateThreshold(
                      "customHttp",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
            </div>
          </div>

          <p className="text-xs text-text-muted mb-3 border-t border-surface-border pt-2">
            Per-phase thresholds:
          </p>

          {/* DNS */}
          <div className="mb-3">
            <div className="flex items-center gap-1.5 mb-1">
              <span className="text-xs font-medium text-text-primary">
                DNS Lookup
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP DNS"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-xs text-text-muted">Good (&lt;)</label>
                <input
                  type="number"
                  value={thresholds.httpTimings.dns.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "dns",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
              <div>
                <label className="text-xs text-text-muted">
                  Warning (&lt;)
                </label>
                <input
                  type="number"
                  value={thresholds.httpTimings.dns.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "dns",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
            </div>
          </div>

          {/* TCP */}
          <div className="mb-3">
            <div className="flex items-center gap-1.5 mb-1">
              <span className="text-xs font-medium text-text-primary">
                TCP Connect
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TCP"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-xs text-text-muted">Good (&lt;)</label>
                <input
                  type="number"
                  value={thresholds.httpTimings.tcp.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tcp",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
              <div>
                <label className="text-xs text-text-muted">
                  Warning (&lt;)
                </label>
                <input
                  type="number"
                  value={thresholds.httpTimings.tcp.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tcp",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
            </div>
          </div>

          {/* TLS */}
          <div className="mb-3">
            <div className="flex items-center gap-1.5 mb-1">
              <span className="text-xs font-medium text-text-primary">
                TLS Handshake
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TLS"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-xs text-text-muted">Good (&lt;)</label>
                <input
                  type="number"
                  value={thresholds.httpTimings.tls.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tls",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
              <div>
                <label className="text-xs text-text-muted">
                  Warning (&lt;)
                </label>
                <input
                  type="number"
                  value={thresholds.httpTimings.tls.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tls",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
            </div>
          </div>

          {/* TTFB */}
          <div>
            <div className="flex items-center gap-1.5 mb-1">
              <span className="text-xs font-medium text-text-primary">
                TTFB (Server Wait)
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TTFB"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-xs text-text-muted">Good (&lt;)</label>
                <input
                  type="number"
                  value={thresholds.httpTimings.ttfb.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "ttfb",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
              <div>
                <label className="text-xs text-text-muted">
                  Warning (&lt;)
                </label>
                <input
                  type="number"
                  value={thresholds.httpTimings.ttfb.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "ttfb",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
}
