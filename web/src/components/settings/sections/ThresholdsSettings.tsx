import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Tooltip } from "../../ui/Tooltip";
import { THRESHOLD_HELP } from "../../help/HelpContent";
import { SettingsThresholds, SaveStatus } from "../../../types/settings";
import { Info, SlidersHorizontal } from "../../ui/Icons";
import {
  layout,
  icon as iconTokens,
  radius,
  input as inputTokens,
} from "../../../styles/theme";

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
        <div className={layout.inline.default}>
          <SlidersHorizontal className={iconTokens.size.sm} />
          <span>Thresholds</span>
          <AutoSaveIndicator status={thresholdsStatus} />
        </div>
      }
    >
      <div className="stack-sm">
        {/* DNS Thresholds */}
        <div
          className={`p-3 bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} mb-2`}>
            <span className="body-small font-medium text-text-primary">
              DNS Lookup (ms)
            </span>
            <Tooltip content={THRESHOLD_HELP["DNS Lookup"]} position="top">
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="caption text-text-muted" htmlFor="dns-good">
                Good (&lt;)
              </label>
              <input
                id="dns-good"
                type="number"
                value={thresholds.dns.good}
                onChange={(e) =>
                  updateThreshold("dns", "good", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="dns-warning">
                Warning (&lt;)
              </label>
              <input
                id="dns-warning"
                type="number"
                value={thresholds.dns.warning}
                onChange={(e) =>
                  updateThreshold("dns", "warning", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
          </div>
        </div>

        {/* Gateway Thresholds */}
        <div
          className={`p-3 bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} mb-2`}>
            <span className="body-small font-medium text-text-primary">
              Gateway Ping (ms)
            </span>
            <Tooltip content={THRESHOLD_HELP["Gateway Ping"]} position="top">
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="caption text-text-muted" htmlFor="gateway-good">
                Good (&lt;)
              </label>
              <input
                id="gateway-good"
                type="number"
                value={thresholds.gateway.good}
                onChange={(e) =>
                  updateThreshold("gateway", "good", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
            <div>
              <label
                className="caption text-text-muted"
                htmlFor="gateway-warning"
              >
                Warning (&lt;)
              </label>
              <input
                id="gateway-warning"
                type="number"
                value={thresholds.gateway.warning}
                onChange={(e) =>
                  updateThreshold("gateway", "warning", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
          </div>
        </div>

        {/* Wi-Fi Signal Thresholds */}
        <div
          className={`p-3 bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} mb-2`}>
            <span className="body-small font-medium text-text-primary">
              Wi-Fi Signal (dBm)
            </span>
            <Tooltip content={THRESHOLD_HELP["Wi-Fi Signal"]} position="top">
              <Info className="w-3.5 h-3.5 text-text-muted hover:text-text-secondary cursor-help" />
            </Tooltip>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="caption text-text-muted" htmlFor="wifi-good">
                Good (&gt;)
              </label>
              <input
                id="wifi-good"
                type="number"
                value={thresholds.wifi.good}
                onChange={(e) =>
                  updateThreshold("wifi", "good", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="wifi-warning">
                Warning (&gt;)
              </label>
              <input
                id="wifi-warning"
                type="number"
                value={thresholds.wifi.warning}
                onChange={(e) =>
                  updateThreshold("wifi", "warning", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
          </div>
        </div>

        {/* Health Check Ping Thresholds */}
        <div
          className={`p-3 bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} mb-2`}>
            <span className="body-small font-medium text-text-primary">
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
              <label className="caption text-text-muted" htmlFor="ping-good">
                Good (&lt;)
              </label>
              <input
                id="ping-good"
                type="number"
                value={thresholds.customPing.good}
                onChange={(e) =>
                  updateThreshold("customPing", "good", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="ping-warning">
                Warning (&lt;)
              </label>
              <input
                id="ping-warning"
                type="number"
                value={thresholds.customPing.warning}
                onChange={(e) =>
                  updateThreshold(
                    "customPing",
                    "warning",
                    Number(e.target.value),
                  )
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
          </div>
        </div>

        {/* Health Check TCP Thresholds */}
        <div
          className={`p-3 bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} mb-2`}>
            <span className="body-small font-medium text-text-primary">
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
              <label className="caption text-text-muted" htmlFor="tcp-good">
                Good (&lt;)
              </label>
              <input
                id="tcp-good"
                type="number"
                value={thresholds.customTcp.good}
                onChange={(e) =>
                  updateThreshold("customTcp", "good", Number(e.target.value))
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="tcp-warning">
                Warning (&lt;)
              </label>
              <input
                id="tcp-warning"
                type="number"
                value={thresholds.customTcp.warning}
                onChange={(e) =>
                  updateThreshold(
                    "customTcp",
                    "warning",
                    Number(e.target.value),
                  )
                }
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
              />
            </div>
          </div>
        </div>

        {/* HTTP Thresholds (Total + Timing Phases) */}
        <div
          className={`p-3 bg-surface-base ${radius.md} border border-surface-border`}
        >
          <span className="body-small font-medium text-text-primary block mb-2">
            HTTP Thresholds (ms)
          </span>

          {/* Total */}
          <div className="mb-3">
            <div className={`${layout.inline.tight} mb-1`}>
              <span className="caption font-medium text-text-primary">
                Total Response Time
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP Total"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-total-good"
                >
                  Good (&lt;)
                </label>
                <input
                  id="http-total-good"
                  type="number"
                  value={thresholds.customHttp.good}
                  onChange={(e) =>
                    updateThreshold(
                      "customHttp",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-total-warning"
                >
                  Warning (&lt;)
                </label>
                <input
                  id="http-total-warning"
                  type="number"
                  value={thresholds.customHttp.warning}
                  onChange={(e) =>
                    updateThreshold(
                      "customHttp",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
            </div>
          </div>

          <p className="caption text-text-muted mb-3 border-t border-surface-border pt-2">
            Per-phase thresholds:
          </p>

          {/* DNS */}
          <div className="mb-3">
            <div className={`${layout.inline.tight} mb-1`}>
              <span className="caption font-medium text-text-primary">
                DNS Lookup
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP DNS"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-dns-good"
                >
                  Good (&lt;)
                </label>
                <input
                  id="http-dns-good"
                  type="number"
                  value={thresholds.httpTimings.dns.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "dns",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-dns-warning"
                >
                  Warning (&lt;)
                </label>
                <input
                  id="http-dns-warning"
                  type="number"
                  value={thresholds.httpTimings.dns.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "dns",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
            </div>
          </div>

          {/* TCP */}
          <div className="mb-3">
            <div className={`${layout.inline.tight} mb-1`}>
              <span className="caption font-medium text-text-primary">
                TCP Connect
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TCP"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-tcp-good"
                >
                  Good (&lt;)
                </label>
                <input
                  id="http-tcp-good"
                  type="number"
                  value={thresholds.httpTimings.tcp.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tcp",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-tcp-warning"
                >
                  Warning (&lt;)
                </label>
                <input
                  id="http-tcp-warning"
                  type="number"
                  value={thresholds.httpTimings.tcp.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tcp",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
            </div>
          </div>

          {/* TLS */}
          <div className="mb-3">
            <div className={`${layout.inline.tight} mb-1`}>
              <span className="caption font-medium text-text-primary">
                TLS Handshake
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TLS"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-tls-good"
                >
                  Good (&lt;)
                </label>
                <input
                  id="http-tls-good"
                  type="number"
                  value={thresholds.httpTimings.tls.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tls",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-tls-warning"
                >
                  Warning (&lt;)
                </label>
                <input
                  id="http-tls-warning"
                  type="number"
                  value={thresholds.httpTimings.tls.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "tls",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
            </div>
          </div>

          {/* TTFB */}
          <div>
            <div className={`${layout.inline.tight} mb-1`}>
              <span className="caption font-medium text-text-primary">
                TTFB (Server Wait)
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TTFB"]} position="top">
                <Info className="w-3 h-3 text-text-muted hover:text-text-secondary cursor-help" />
              </Tooltip>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-ttfb-good"
                >
                  Good (&lt;)
                </label>
                <input
                  id="http-ttfb-good"
                  type="number"
                  value={thresholds.httpTimings.ttfb.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "ttfb",
                      "good",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
              <div>
                <label
                  className="caption text-text-muted"
                  htmlFor="http-ttfb-warning"
                >
                  Warning (&lt;)
                </label>
                <input
                  id="http-ttfb-warning"
                  type="number"
                  value={thresholds.httpTimings.ttfb.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold(
                      "ttfb",
                      "warning",
                      Number(e.target.value),
                    )
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 body-small`}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
}
