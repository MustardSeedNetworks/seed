import { useTranslation } from "react-i18next";
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
  spacing,
} from "../../../styles/theme";

interface ThresholdsSettingsProps {
  thresholds: SettingsThresholds;
  setThresholds: React.Dispatch<React.SetStateAction<SettingsThresholds>>;
  thresholdsStatus: SaveStatus;
}

/**
 * Settings section for configuring alert thresholds across metrics.
 */
export function ThresholdsSettings({
  thresholds,
  setThresholds,
  thresholdsStatus,
}: ThresholdsSettingsProps) {
  const { t } = useTranslation("settings");

  // Type-safe threshold category getter
  function getThresholdCategory(
    prev: SettingsThresholds,
    category: keyof Omit<SettingsThresholds, "httpTimings">
  ) {
    switch (category) {
      case "dns":
        return prev.dns;
      case "gateway":
        return prev.gateway;
      case "wifi":
        return prev.wifi;
      case "customPing":
        return prev.customPing;
      case "customTcp":
        return prev.customTcp;
      case "customHttp":
        return prev.customHttp;
    }
  }

  // Type-safe HTTP timing phase getter
  function getHttpTimingPhase(
    httpTimings: SettingsThresholds["httpTimings"],
    phase: keyof SettingsThresholds["httpTimings"]
  ) {
    switch (phase) {
      case "dns":
        return httpTimings.dns;
      case "tcp":
        return httpTimings.tcp;
      case "tls":
        return httpTimings.tls;
      case "ttfb":
        return httpTimings.ttfb;
    }
  }

  const updateThreshold = (
    category: keyof Omit<SettingsThresholds, "httpTimings">,
    level: "good" | "warning",
    value: number
  ) => {
    setThresholds((prev) => {
      const current = getThresholdCategory(prev, category);
      const updated =
        level === "good" ? { ...current, good: value } : { ...current, warning: value };
      return { ...prev, [category]: updated };
    });
  };

  const updateHttpTimingThreshold = (
    phase: keyof SettingsThresholds["httpTimings"],
    level: "good" | "warning",
    value: number
  ) => {
    setThresholds((prev) => {
      const current = getHttpTimingPhase(prev.httpTimings, phase);
      const updated =
        level === "good" ? { ...current, good: value } : { ...current, warning: value };
      return {
        ...prev,
        httpTimings: { ...prev.httpTimings, [phase]: updated },
      };
    });
  };

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <SlidersHorizontal className={iconTokens.size.sm} />
          <span>{t("sections.thresholds")}</span>
          <AutoSaveIndicator status={thresholdsStatus} />
        </div>
      }
    >
      <div className="stack-sm">
        {/* DNS Thresholds */}
        <div
          className={`${spacing.pad.sm} bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
            <span className="body-small font-medium text-text-primary">
              {t("thresholds.dnsLookup")}
            </span>
            <Tooltip content={THRESHOLD_HELP["DNS Lookup"]} position="top">
              <Info
                className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
              />
            </Tooltip>
          </div>
          <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
            <div>
              <label className="caption text-text-muted" htmlFor="dns-good">
                {t("thresholds.goodLess")}
              </label>
              <input
                id="dns-good"
                type="number"
                value={thresholds.dns.good}
                onChange={(e) => updateThreshold("dns", "good", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="dns-warning">
                {t("thresholds.warningLess")}
              </label>
              <input
                id="dns-warning"
                type="number"
                value={thresholds.dns.warning}
                onChange={(e) => updateThreshold("dns", "warning", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
          </div>
        </div>

        {/* Gateway Thresholds */}
        <div
          className={`${spacing.pad.sm} bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
            <span className="body-small font-medium text-text-primary">
              {t("thresholds.gatewayPing")}
            </span>
            <Tooltip content={THRESHOLD_HELP["Gateway Ping"]} position="top">
              <Info
                className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
              />
            </Tooltip>
          </div>
          <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
            <div>
              <label className="caption text-text-muted" htmlFor="gateway-good">
                {t("thresholds.goodLess")}
              </label>
              <input
                id="gateway-good"
                type="number"
                value={thresholds.gateway.good}
                onChange={(e) => updateThreshold("gateway", "good", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="gateway-warning">
                {t("thresholds.warningLess")}
              </label>
              <input
                id="gateway-warning"
                type="number"
                value={thresholds.gateway.warning}
                onChange={(e) => updateThreshold("gateway", "warning", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
          </div>
        </div>

        {/* Wi-Fi Signal Thresholds */}
        <div
          className={`${spacing.pad.sm} bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
            <span className="body-small font-medium text-text-primary">
              {t("thresholds.wifiSignal")}
            </span>
            <Tooltip content={THRESHOLD_HELP["Wi-Fi Signal"]} position="top">
              <Info
                className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
              />
            </Tooltip>
          </div>
          <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
            <div>
              <label className="caption text-text-muted" htmlFor="wifi-good">
                {t("thresholds.goodGreater")}
              </label>
              <input
                id="wifi-good"
                type="number"
                value={thresholds.wifi.good}
                onChange={(e) => updateThreshold("wifi", "good", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="wifi-warning">
                {t("thresholds.warningGreater")}
              </label>
              <input
                id="wifi-warning"
                type="number"
                value={thresholds.wifi.warning}
                onChange={(e) => updateThreshold("wifi", "warning", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
          </div>
        </div>

        {/* Health Check Ping Thresholds */}
        <div
          className={`${spacing.pad.sm} bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
            <span className="body-small font-medium text-text-primary">
              {t("thresholds.healthPing")}
            </span>
            <Tooltip content={THRESHOLD_HELP["Health Check: Ping"]} position="top">
              <Info
                className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
              />
            </Tooltip>
          </div>
          <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
            <div>
              <label className="caption text-text-muted" htmlFor="ping-good">
                {t("thresholds.goodLess")}
              </label>
              <input
                id="ping-good"
                type="number"
                value={thresholds.customPing.good}
                onChange={(e) => updateThreshold("customPing", "good", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="ping-warning">
                {t("thresholds.warningLess")}
              </label>
              <input
                id="ping-warning"
                type="number"
                value={thresholds.customPing.warning}
                onChange={(e) => updateThreshold("customPing", "warning", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
          </div>
        </div>

        {/* Health Check TCP Thresholds */}
        <div
          className={`${spacing.pad.sm} bg-surface-base ${radius.md} border border-surface-border`}
        >
          <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
            <span className="body-small font-medium text-text-primary">
              {t("thresholds.healthTcp")}
            </span>
            <Tooltip content={THRESHOLD_HELP["Health Check: TCP"]} position="top">
              <Info
                className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
              />
            </Tooltip>
          </div>
          <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
            <div>
              <label className="caption text-text-muted" htmlFor="tcp-good">
                {t("thresholds.goodLess")}
              </label>
              <input
                id="tcp-good"
                type="number"
                value={thresholds.customTcp.good}
                onChange={(e) => updateThreshold("customTcp", "good", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="tcp-warning">
                {t("thresholds.warningLess")}
              </label>
              <input
                id="tcp-warning"
                type="number"
                value={thresholds.customTcp.warning}
                onChange={(e) => updateThreshold("customTcp", "warning", Number(e.target.value))}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
              />
            </div>
          </div>
        </div>

        {/* HTTP Thresholds (Total + Timing Phases) */}
        <div
          className={`${spacing.pad.sm} bg-surface-base ${radius.md} border border-surface-border`}
        >
          <span
            className={`body-small font-medium text-text-primary block ${spacing.margin.bottom.inline}`}
          >
            {t("thresholds.httpThresholds")}
          </span>

          {/* Total */}
          <div className={spacing.margin.bottom.heading}>
            <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
              <span className="caption font-medium text-text-primary">
                {t("thresholds.totalResponseTime")}
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP Total"]} position="top">
                <Info
                  className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
                />
              </Tooltip>
            </div>
            <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
              <div>
                <label className="caption text-text-muted" htmlFor="http-total-good">
                  {t("thresholds.goodLess")}
                </label>
                <input
                  id="http-total-good"
                  type="number"
                  value={thresholds.customHttp.good}
                  onChange={(e) => updateThreshold("customHttp", "good", Number(e.target.value))}
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
              <div>
                <label className="caption text-text-muted" htmlFor="http-total-warning">
                  {t("thresholds.warningLess")}
                </label>
                <input
                  id="http-total-warning"
                  type="number"
                  value={thresholds.customHttp.warning}
                  onChange={(e) => updateThreshold("customHttp", "warning", Number(e.target.value))}
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
            </div>
          </div>

          <p
            className={`caption text-text-muted ${spacing.margin.bottom.heading} border-t border-surface-border ${spacing.pad.sm}`}
          >
            {t("thresholds.perPhaseThresholds")}
          </p>

          {/* DNS */}
          <div className={spacing.margin.bottom.heading}>
            <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
              <span className="caption font-medium text-text-primary">
                {t("thresholds.dnsLookupPhase")}
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP DNS"]} position="top">
                <Info
                  className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
                />
              </Tooltip>
            </div>
            <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
              <div>
                <label className="caption text-text-muted" htmlFor="http-dns-good">
                  {t("thresholds.goodLess")}
                </label>
                <input
                  id="http-dns-good"
                  type="number"
                  value={thresholds.httpTimings.dns.good}
                  onChange={(e) => updateHttpTimingThreshold("dns", "good", Number(e.target.value))}
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
              <div>
                <label className="caption text-text-muted" htmlFor="http-dns-warning">
                  {t("thresholds.warningLess")}
                </label>
                <input
                  id="http-dns-warning"
                  type="number"
                  value={thresholds.httpTimings.dns.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold("dns", "warning", Number(e.target.value))
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
            </div>
          </div>

          {/* TCP */}
          <div className={spacing.margin.bottom.heading}>
            <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
              <span className="caption font-medium text-text-primary">
                {t("thresholds.tcpConnect")}
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TCP"]} position="top">
                <Info
                  className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
                />
              </Tooltip>
            </div>
            <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
              <div>
                <label className="caption text-text-muted" htmlFor="http-tcp-good">
                  {t("thresholds.goodLess")}
                </label>
                <input
                  id="http-tcp-good"
                  type="number"
                  value={thresholds.httpTimings.tcp.good}
                  onChange={(e) => updateHttpTimingThreshold("tcp", "good", Number(e.target.value))}
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
              <div>
                <label className="caption text-text-muted" htmlFor="http-tcp-warning">
                  {t("thresholds.warningLess")}
                </label>
                <input
                  id="http-tcp-warning"
                  type="number"
                  value={thresholds.httpTimings.tcp.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold("tcp", "warning", Number(e.target.value))
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
            </div>
          </div>

          {/* TLS */}
          <div className={spacing.margin.bottom.heading}>
            <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
              <span className="caption font-medium text-text-primary">
                {t("thresholds.tlsHandshake")}
              </span>
              <Tooltip content={THRESHOLD_HELP["HTTP TLS"]} position="top">
                <Info
                  className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
                />
              </Tooltip>
            </div>
            <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
              <div>
                <label className="caption text-text-muted" htmlFor="http-tls-good">
                  {t("thresholds.goodLess")}
                </label>
                <input
                  id="http-tls-good"
                  type="number"
                  value={thresholds.httpTimings.tls.good}
                  onChange={(e) => updateHttpTimingThreshold("tls", "good", Number(e.target.value))}
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
              <div>
                <label className="caption text-text-muted" htmlFor="http-tls-warning">
                  {t("thresholds.warningLess")}
                </label>
                <input
                  id="http-tls-warning"
                  type="number"
                  value={thresholds.httpTimings.tls.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold("tls", "warning", Number(e.target.value))
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
            </div>
          </div>

          {/* TTFB */}
          <div>
            <div className={`${layout.inline.tight} ${spacing.margin.bottom.inline}`}>
              <span className="caption font-medium text-text-primary">{t("thresholds.ttfb")}</span>
              <Tooltip content={THRESHOLD_HELP["HTTP TTFB"]} position="top">
                <Info
                  className={`${iconTokens.size.xs} text-text-muted hover:text-text-secondary cursor-help`}
                />
              </Tooltip>
            </div>
            <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
              <div>
                <label className="caption text-text-muted" htmlFor="http-ttfb-good">
                  {t("thresholds.goodLess")}
                </label>
                <input
                  id="http-ttfb-good"
                  type="number"
                  value={thresholds.httpTimings.ttfb.good}
                  onChange={(e) =>
                    updateHttpTimingThreshold("ttfb", "good", Number(e.target.value))
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
              <div>
                <label className="caption text-text-muted" htmlFor="http-ttfb-warning">
                  {t("thresholds.warningLess")}
                </label>
                <input
                  id="http-ttfb-warning"
                  type="number"
                  value={thresholds.httpTimings.ttfb.warning}
                  onChange={(e) =>
                    updateHttpTimingThreshold("ttfb", "warning", Number(e.target.value))
                  }
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} body-small`}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
}
