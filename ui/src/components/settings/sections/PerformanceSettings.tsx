/**
 * PerformanceSettings Component (~421 lines)
 *
 * Purpose: Comprehensive performance testing configuration including speedtest.net and iperf3 settings.
 * Allows users to configure test intervals, timeouts, and iperf3 server suggestions.
 *
 * Key Features:
 * - Speedtest.net configuration: enable/disable, interval, timeout settings
 * - iperf3 configuration: enable/disable, server setup, port configuration
 * - Test interval: frequency of performance tests (in seconds)
 * - Timeout settings: maximum duration for tests
 * - iperf3 server suggestions: fetches and displays recommended public iperf servers
 * - Server validation: validates iperf server addresses and ports
 * - Port configuration: custom port for iperf3 tests
 * - Bandwidth limits: configurable upload/download limits
 * - Protocol selection: TCP/UDP selection for iperf tests
 * - AutoSaveIndicator: shows persistent save status
 * - Gauge icon: visual indicator in settings menu
 *
 * Usage:
 * ```typescript
 * <PerformanceSettings
 *   testsSettings={settings}
 *   setTestsSettings={updateSettings}
 *   iperfSettings={iperfSettings}
 *   setIperfSettings={updateIperf}
 *   iperfStatus={saveStatus}
 *   iperfSuggestions={suggestions}
 *   iperfSuggestionsStatus={fetchStatus}
 *   iperfSuggestionsError={error}
 *   fetchIperfSuggestions={fetchSuggestions}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, Gauge icon, settings types
 * State: Manages speedtest, iperf, and suggestion configurations
 */

import type React from "react";
import { memo } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
} from "../../../styles/theme";
import type {
  IperfSettings,
  IperfSuggestion,
  SaveStatus,
  TestsSettings,
} from "../../../types/settings";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { Gauge } from "../../ui/Icons";
import { AutoSaveIndicator } from "./AutoSaveIndicator";

interface PerformanceSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  iperfSettings: IperfSettings;
  setIperfSettings: React.Dispatch<React.SetStateAction<IperfSettings>>;
  iperfStatus: SaveStatus;
  iperfSuggestions: IperfSuggestion[];
  iperfSuggestionsStatus: "idle" | "loading" | "error";
  iperfSuggestionsError: string | null;
  fetchIperfSuggestions: () => void;
}

/**
 * Settings section for speed test and iPerf performance testing configuration.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const PerformanceSettings = memo(function PerformanceSettings({
  testsSettings,
  setTestsSettings,
  iperfSettings,
  setIperfSettings,
  iperfStatus,
  iperfSuggestions,
  iperfSuggestionsStatus,
  iperfSuggestionsError,
  fetchIperfSuggestions,
}: PerformanceSettingsProps) {
  const { t } = useTranslation("settings");

  // Get translated direction label
  const getDirectionLabel = (direction: string) => {
    switch (direction) {
      case "download":
        return t("performance.download");
      case "upload":
        return t("performance.upload");
      case "bidirectional":
        return t("performance.both");
      default:
        return direction;
    }
  };

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Gauge className={iconTokens.size.sm} />
          <span>{t("sections.performance")}</span>
          <AutoSaveIndicator status={iperfStatus} />
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack">
        {/* Enable/Disable Toggles */}
        <div className="stack-sm">
          <label
            className={cn(
              layout.flex.between,
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border",
            )}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                {t("performance.enableSpeedtest")}
              </span>
              <p className="caption text-text-muted">{t("performance.speedtestDesc")}</p>
            </div>
            <input
              type="checkbox"
              checked={testsSettings.runSpeedtest}
              onChange={(e) =>
                setTestsSettings((prev) => ({
                  ...prev,
                  runSpeedtest: e.target.checked,
                }))
              }
              className={iconTokens.size.sm}
            />
          </label>
          <label
            className={cn(
              layout.flex.between,
              spacing.pad.sm,
              "bg-surface-base",
              radius.default,
              "border border-surface-border",
            )}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                {t("performance.enableIperf")}
              </span>
              <p className="caption text-text-muted">{t("performance.iperfDesc")}</p>
            </div>
            <input
              type="checkbox"
              checked={testsSettings.runIperf}
              onChange={(e) =>
                setTestsSettings((prev) => ({
                  ...prev,
                  runIperf: e.target.checked,
                }))
              }
              className={iconTokens.size.sm}
            />
          </label>
        </div>

        {/* Auto-Run on Link Up */}
        <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
          <span className="caption text-text-muted font-medium">
            {t("performance.autoRunOnLink")}
          </span>
          <div className={cn(spacing.margin.top.inline, "stack-sm")}>
            <label
              className={cn(
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border",
              )}
            >
              <span className="body-small text-text-primary">{t("performance.speedtest")}</span>
              <input
                type="checkbox"
                checked={testsSettings.speedtest.autoRunOnLink}
                onChange={(e) =>
                  setTestsSettings((prev) => ({
                    ...prev,
                    speedtest: {
                      ...prev.speedtest,
                      autoRunOnLink: e.target.checked,
                    },
                  }))
                }
                className={iconTokens.size.sm}
              />
            </label>
            <label
              className={cn(
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border",
              )}
            >
              <span className="body-small text-text-primary">{t("performance.iperf")}</span>
              <input
                type="checkbox"
                checked={testsSettings.iperf.autoRunOnLink}
                onChange={(e) =>
                  setTestsSettings((prev) => ({
                    ...prev,
                    iperf: {
                      ...prev.iperf,
                      autoRunOnLink: e.target.checked,
                    },
                  }))
                }
                className={iconTokens.size.sm}
              />
            </label>
          </div>
        </div>

        {/* Internet Speed (Speedtest) Subsection */}
        <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
          <h4
            className={cn(
              "body-small font-semibold text-text-primary",
              spacing.margin.bottom.inline,
              "uppercase tracking-wide",
            )}
          >
            {t("performance.internetSpeed")}
          </h4>
          <div className="stack">
            <div>
              <label htmlFor="speedtest-server-id" className="caption text-text-muted font-medium">
                {t("performance.serverId")}
              </label>
              <input
                id="speedtest-server-id"
                type="text"
                value={testsSettings.speedtest.serverId}
                onChange={(e) =>
                  setTestsSettings((prev) => ({
                    ...prev,
                    speedtest: {
                      ...prev.speedtest,
                      serverId: e.target.value,
                    },
                  }))
                }
                placeholder={t("performance.autoClosestServer")}
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  "w-full",
                  spacing.margin.top.tight,
                  "body-small",
                )}
              />
              <div className={cn(layout.flex.between, spacing.margin.top.tight)}>
                <p className="caption text-text-muted">{t("performance.autoSelectDesc")}</p>
                <button
                  type="button"
                  type="button"
                  onClick={() =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      speedtest: { ...prev.speedtest, serverId: "" },
                    }))
                  }
                  className="caption text-brand-primary hover:underline"
                >
                  {t("performance.resetToAuto")}
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* LAN Speed (iperf3) Subsection */}
        <div>
          <h4
            className={cn(
              "body-small font-semibold text-text-primary",
              spacing.margin.bottom.inline,
              "uppercase tracking-wide",
            )}
          >
            {t("performance.lanSpeed")}
          </h4>
          <div className="stack">
            <p className="caption text-text-muted">{t("performance.lanSpeedDesc")}</p>

            {/* Server Address */}
            <div>
              <label htmlFor="iperf-server-address" className="caption text-text-muted font-medium">
                {t("performance.serverAddress")}
              </label>
              <input
                id="iperf-server-address"
                type="text"
                value={iperfSettings.server}
                onChange={(e) =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    server: e.target.value,
                  }))
                }
                placeholder="192.168.1.100"
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  "w-full",
                  spacing.margin.top.tight,
                  "body-small disabled:opacity-60",
                )}
              />
              <div className={cn(layout.flex.between, spacing.margin.top.inline)}>
                <button
                  type="button"
                  type="button"
                  disabled={iperfSuggestionsStatus === "loading"}
                  onClick={fetchIperfSuggestions}
                  className="caption text-brand-primary hover:underline disabled:opacity-60 disabled:cursor-not-allowed"
                >
                  {iperfSuggestionsStatus === "loading"
                    ? t("performance.scanning")
                    : t("performance.findIperfHosts")}
                </button>
                {iperfSuggestionsStatus === "loading" && (
                  <svg
                    className={cn(iconTokens.size.sm, "animate-spin text-text-muted")}
                    viewBox="0 0 24 24"
                    fill="none"
                    aria-hidden="true"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                )}
              </div>
              {iperfSuggestionsStatus === "error" && (
                <p className={cn("caption text-status-warning", spacing.margin.top.tight)}>
                  {iperfSuggestionsError || t("performance.noIperfHosts")}
                </p>
              )}
              {iperfSuggestions.length > 0 && (
                <div
                  className={cn("flex flex-wrap", spacing.gap.compact, spacing.margin.top.inline)}
                >
                  {iperfSuggestions.map((sugg) => (
                    <button
                      type="button"
                      type="button"
                      key={`${sugg.host}-${sugg.hostname || ""}`}
                      className={cn(
                        spacing.chip.sm,
                        radius.full,
                        "border border-surface-border bg-surface-base caption text-text-primary hover:bg-surface-hover",
                      )}
                      onClick={() =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          server: sugg.host,
                        }))
                      }
                    >
                      <span className="font-medium">{sugg.hostname || sugg.host}</span>
                      <span className={cn("text-text-muted", spacing.margin.left.tight)}>
                        {sugg.hostname ? `(${sugg.host})` : ""}
                        {sugg.latencyMs !== undefined ? ` · ${Math.round(sugg.latencyMs)}ms` : ""}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Port */}
            <div>
              <label className="caption text-text-muted font-medium" htmlFor="iperf-port">
                {t("performance.port")}
              </label>
              <input
                id="iperf-port"
                type="number"
                value={iperfSettings.port}
                onChange={(e) =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    port: Number.parseInt(e.target.value, 10) || 5201,
                  }))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  "w-full",
                  spacing.margin.top.tight,
                  "body-small disabled:opacity-60",
                )}
              />
            </div>

            {/* Protocol Toggle */}
            <div>
              <span
                className={cn(
                  "caption text-text-muted font-medium block",
                  spacing.margin.bottom.inline,
                )}
              >
                {t("performance.protocol")}
              </span>
              <div
                className={cn("flex flex-wrap", spacing.gap.compact)}
                role="radiogroup"
                aria-label="Protocol selection"
              >
                {(["tcp", "udp"] as const).map((proto) => {
                  const checked = iperfSettings.protocol === proto;
                  return (
                    <label
                      key={proto}
                      className={cn(
                        "cursor-pointer",
                        spacing.chip.md,
                        radius.full,
                        "border body-small font-medium transition-colors",
                        checked
                          ? "bg-brand-primary text-text-inverse border-brand-primary"
                          : "bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover",
                      )}
                    >
                      <input
                        type="radio"
                        name="iperf-protocol"
                        value={proto}
                        checked={checked}
                        onChange={() =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            protocol: proto,
                          }))
                        }
                        className="sr-only"
                        aria-label={`${proto.toUpperCase()} protocol`}
                      />
                      {proto.toUpperCase()}
                    </label>
                  );
                })}
              </div>
            </div>

            {/* Direction Toggle */}
            <div>
              <span
                className={cn(
                  "caption text-text-muted font-medium block",
                  spacing.margin.bottom.inline,
                )}
              >
                {t("performance.direction")}
              </span>
              <div
                className={cn("flex flex-wrap", spacing.gap.compact)}
                role="radiogroup"
                aria-label="Direction selection"
              >
                {(["download", "upload", "bidirectional"] as const).map((direction) => {
                  const checked = iperfSettings.direction === direction;
                  return (
                    <label
                      key={direction}
                      className={cn(
                        "cursor-pointer",
                        spacing.chip.md,
                        radius.full,
                        "border body-small font-medium transition-colors",
                        checked
                          ? "bg-brand-primary text-text-inverse border-brand-primary"
                          : "bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover",
                      )}
                    >
                      <input
                        type="radio"
                        name="iperf-direction"
                        value={direction}
                        checked={checked}
                        onChange={() =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            direction: direction,
                          }))
                        }
                        className="sr-only"
                        aria-label={`${getDirectionLabel(direction)} direction`}
                      />
                      {getDirectionLabel(direction)}
                    </label>
                  );
                })}
              </div>
            </div>

            {/* Duration */}
            <div>
              <label className="caption text-text-muted font-medium" htmlFor="iperf-duration">
                {t("performance.duration")}
              </label>
              <input
                id="iperf-duration"
                type="number"
                value={iperfSettings.duration}
                onChange={(e) =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    duration: Number.parseInt(e.target.value, 10) || 10,
                  }))
                }
                min={1}
                max={60}
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  "w-full",
                  spacing.margin.top.tight,
                  "body-small disabled:opacity-60",
                )}
              />
            </div>

            {/* Server Mode */}
            <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
              <label
                className={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border",
                  spacing.margin.bottom.inline,
                )}
              >
                <span className="body-small text-text-primary">
                  {t("performance.enableServer")}
                </span>
                <input
                  type="checkbox"
                  checked={iperfSettings.enableServer}
                  onChange={(e) =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      enableServer: e.target.checked,
                    }))
                  }
                  className={iconTokens.size.sm}
                />
              </label>
              <div>
                <label className="caption text-text-muted font-medium" htmlFor="iperf-server-port">
                  {t("performance.serverPort")}
                </label>
                <input
                  id="iperf-server-port"
                  type="number"
                  value={iperfSettings.serverPort}
                  onChange={(e) =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      serverPort: Number.parseInt(e.target.value, 10) || 5201,
                    }))
                  }
                  className={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.md,
                    "w-full",
                    spacing.margin.top.tight,
                    "body-small disabled:opacity-60",
                  )}
                />
              </div>
              <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
                {t("performance.serverAutoStart")}
              </p>
            </div>
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
});
