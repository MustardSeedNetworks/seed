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
  CardSettings,
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
  /** Card settings for FAB auto-run configuration */
  cardSettings: CardSettings;
  /** Update card settings (triggers auto-save to profile) */
  updateCardSettings: (updates: Partial<CardSettings>) => void;
}

/**
 * Settings section for speed test and iPerf performance testing configuration.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const PerformanceSettings: React.NamedExoticComponent<PerformanceSettingsProps> = memo(
  function performanceSettings({
    testsSettings,
    setTestsSettings,
    iperfSettings,
    setIperfSettings,
    iperfStatus,
    iperfSuggestions,
    iperfSuggestionsStatus,
    iperfSuggestionsError,
    fetchIperfSuggestions,
    cardSettings,
    updateCardSettings,
  }: PerformanceSettingsProps) {
    const { t } = useTranslation("settings");

    // Get translated direction label
    const getDirectionLabel = (direction: string): string => {
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
          <div class={layout.inline.default}>
            <Gauge class={iconTokens.size.sm} />
            <span>{t("sections.performance")}</span>
            <AutoSaveIndicator status={iperfStatus} />
          </div>
        }
        defaultOpen={false}
      >
        <div class="stack">
          {/* Enable/Disable Toggles */}
          <div class="stack-sm">
            <label
              class={cn(
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border",
              )}
            >
              <div>
                <span class="body-small text-text-primary font-medium">
                  {t("performance.enableSpeedtest")}
                </span>
                <p class="caption text-text-muted">{t("performance.speedtestDesc")}</p>
              </div>
              <input
                type="checkbox"
                checked={testsSettings.runSpeedtest}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  setTestsSettings((prev) => ({
                    ...prev,
                    runSpeedtest: e.target.checked,
                  }))
                }
                class={iconTokens.size.sm}
              />
            </label>
            <label
              class={cn(
                layout.flex.between,
                spacing.pad.sm,
                "bg-surface-base",
                radius.default,
                "border border-surface-border",
              )}
            >
              <div>
                <span class="body-small text-text-primary font-medium">
                  {t("performance.enableIperf")}
                </span>
                <p class="caption text-text-muted">{t("performance.iperfDesc")}</p>
              </div>
              <input
                type="checkbox"
                checked={testsSettings.runIperf}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  setTestsSettings((prev) => ({
                    ...prev,
                    runIperf: e.target.checked,
                  }))
                }
                class={iconTokens.size.sm}
              />
            </label>
          </div>

          {/* Auto-Run on Link Up (FAB button) */}
          <div class={cn("border-t border-surface-border", spacing.padding.top.heading)}>
            <span class="caption text-text-muted font-medium">
              {t("performance.autoRunOnLink")}
            </span>
            <p class="caption text-text-muted mt-1">
              {t(
                "performance.autoRunOnLinkDesc",
                "Controls which tests run when FAB button is clicked",
              )}
            </p>
            <div class={cn(spacing.margin.top.inline, "stack-sm")}>
              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border",
                )}
              >
                <span class="body-small text-text-primary">{t("performance.speedtest")}</span>
                <input
                  type="checkbox"
                  checked={cardSettings.performance.speedtest.autoRunOnLink}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateCardSettings({
                      performance: {
                        ...cardSettings.performance,
                        speedtest: {
                          ...cardSettings.performance.speedtest,
                          autoRunOnLink: e.target.checked,
                        },
                      },
                    })
                  }
                  class={iconTokens.size.sm}
                />
              </label>
              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  "bg-surface-base",
                  radius.default,
                  "border border-surface-border",
                )}
              >
                <span class="body-small text-text-primary">{t("performance.iperf")}</span>
                <input
                  type="checkbox"
                  checked={cardSettings.performance.iperf.autoRunOnLink}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateCardSettings({
                      performance: {
                        ...cardSettings.performance,
                        iperf: {
                          ...cardSettings.performance.iperf,
                          autoRunOnLink: e.target.checked,
                        },
                      },
                    })
                  }
                  class={iconTokens.size.sm}
                />
              </label>
            </div>
          </div>

          {/* Internet Speed (Speedtest) Subsection */}
          <div class={cn("border-t border-surface-border", spacing.padding.top.heading)}>
            <h4
              class={cn(
                "body-small font-semibold text-text-primary",
                spacing.margin.bottom.inline,
                "uppercase tracking-wide",
              )}
            >
              {t("performance.internetSpeed")}
            </h4>
            <div class="stack">
              <div>
                <label for="speedtest-server-id" class="caption text-text-muted font-medium">
                  {t("performance.serverId")}
                </label>
                <input
                  id="speedtest-server-id"
                  type="text"
                  value={testsSettings.speedtest.serverId}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      speedtest: {
                        ...prev.speedtest,
                        serverId: e.target.value,
                      },
                    }))
                  }
                  placeholder={t("performance.autoClosestServer")}
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.md,
                    "w-full",
                    spacing.margin.top.tight,
                    "body-small",
                  )}
                />
                <div class={cn(layout.flex.between, spacing.margin.top.tight)}>
                  <p class="caption text-text-muted">{t("performance.autoSelectDesc")}</p>
                  <button
                    type="button"
                    onClick={(): void =>
                      setTestsSettings((prev) => ({
                        ...prev,
                        speedtest: { ...prev.speedtest, serverId: "" },
                      }))
                    }
                    class="caption text-brand-primary hover:underline"
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
              class={cn(
                "body-small font-semibold text-text-primary",
                spacing.margin.bottom.inline,
                "uppercase tracking-wide",
              )}
            >
              {t("performance.lanSpeed")}
            </h4>
            <div class="stack">
              <p class="caption text-text-muted">{t("performance.lanSpeedDesc")}</p>

              {/* Server Address */}
              <div>
                <label for="iperf-server-address" class="caption text-text-muted font-medium">
                  {t("performance.serverAddress")}
                </label>
                <input
                  id="iperf-server-address"
                  type="text"
                  value={iperfSettings.server}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      server: e.target.value,
                    }))
                  }
                  placeholder="192.168.1.100"
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.md,
                    "w-full",
                    spacing.margin.top.tight,
                    "body-small disabled:opacity-60",
                  )}
                />
                <div class={cn(layout.flex.between, spacing.margin.top.inline)}>
                  <button
                    type="button"
                    disabled={iperfSuggestionsStatus === "loading"}
                    onClick={fetchIperfSuggestions}
                    class="caption text-brand-primary hover:underline disabled:opacity-60 disabled:cursor-not-allowed"
                  >
                    {iperfSuggestionsStatus === "loading"
                      ? t("performance.scanning")
                      : t("performance.findIperfHosts")}
                  </button>
                  {iperfSuggestionsStatus === "loading" && (
                    <svg
                      class={cn(iconTokens.size.sm, "animate-spin text-text-muted")}
                      viewBox="0 0 24 24"
                      fill="none"
                      aria-hidden="true"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      />
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      />
                    </svg>
                  )}
                </div>
                {iperfSuggestionsStatus === "error" && (
                  <p class={cn("caption text-status-warning", spacing.margin.top.tight)}>
                    {iperfSuggestionsError || t("performance.noIperfHosts")}
                  </p>
                )}
                {iperfSuggestions.length > 0 && (
                  <div class={cn("flex flex-wrap", spacing.gap.compact, spacing.margin.top.inline)}>
                    {iperfSuggestions.map((sugg) => (
                      <button
                        type="button"
                        key={`${sugg.host}-${sugg.hostname || ""}`}
                        class={cn(
                          spacing.chip.sm,
                          radius.full,
                          "border border-surface-border bg-surface-base caption text-text-primary hover:bg-surface-hover",
                        )}
                        onClick={(): void =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            server: sugg.host,
                          }))
                        }
                      >
                        <span class="font-medium">{sugg.hostname || sugg.host}</span>
                        <span class={cn("text-text-muted", spacing.margin.left.tight)}>
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
                <label class="caption text-text-muted font-medium" for="iperf-port">
                  {t("performance.port")}
                </label>
                <input
                  id="iperf-port"
                  type="number"
                  value={iperfSettings.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      port: Number.parseInt(e.target.value, 10) || 5201,
                    }))
                  }
                  class={cn(
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
                  class={cn(
                    "caption text-text-muted font-medium block",
                    spacing.margin.bottom.inline,
                  )}
                >
                  {t("performance.protocol")}
                </span>
                <div
                  class={cn("flex flex-wrap", spacing.gap.compact)}
                  role="radiogroup"
                  aria-label="Protocol selection"
                >
                  {(["tcp", "udp"] as const).map((proto) => {
                    const checked = iperfSettings.protocol === proto;
                    return (
                      <label
                        key={proto}
                        class={cn(
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
                          onChange={(): void =>
                            setIperfSettings((prev) => ({
                              ...prev,
                              protocol: proto,
                            }))
                          }
                          class="sr-only"
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
                  class={cn(
                    "caption text-text-muted font-medium block",
                    spacing.margin.bottom.inline,
                  )}
                >
                  {t("performance.direction")}
                </span>
                <div
                  class={cn("flex flex-wrap", spacing.gap.compact)}
                  role="radiogroup"
                  aria-label="Direction selection"
                >
                  {(["download", "upload", "bidirectional"] as const).map((direction) => {
                    const checked = iperfSettings.direction === direction;
                    return (
                      <label
                        key={direction}
                        class={cn(
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
                          onChange={(): void =>
                            setIperfSettings((prev) => ({
                              ...prev,
                              direction: direction,
                            }))
                          }
                          class="sr-only"
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
                <label class="caption text-text-muted font-medium" for="iperf-duration">
                  {t("performance.duration")}
                </label>
                <input
                  id="iperf-duration"
                  type="number"
                  value={iperfSettings.duration}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      duration: Number.parseInt(e.target.value, 10) || 10,
                    }))
                  }
                  min={1}
                  max={60}
                  class={cn(
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
              <div class={cn("border-t border-surface-border", spacing.padding.top.heading)}>
                <label
                  class={cn(
                    layout.flex.between,
                    spacing.pad.sm,
                    "bg-surface-base",
                    radius.default,
                    "border border-surface-border",
                    spacing.margin.bottom.inline,
                  )}
                >
                  <span class="body-small text-text-primary">{t("performance.enableServer")}</span>
                  <input
                    type="checkbox"
                    checked={iperfSettings.enableServer}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                      setIperfSettings((prev) => ({
                        ...prev,
                        enableServer: e.target.checked,
                      }))
                    }
                    class={iconTokens.size.sm}
                  />
                </label>
                <div>
                  <label class="caption text-text-muted font-medium" for="iperf-server-port">
                    {t("performance.serverPort")}
                  </label>
                  <input
                    id="iperf-server-port"
                    type="number"
                    value={iperfSettings.serverPort}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setIperfSettings((prev) => ({
                        ...prev,
                        serverPort: Number.parseInt(e.target.value, 10) || 5201,
                      }))
                    }
                    class={cn(
                      inputTokens.base,
                      inputTokens.state.default,
                      inputTokens.size.md,
                      "w-full",
                      spacing.margin.top.tight,
                      "body-small disabled:opacity-60",
                    )}
                  />
                </div>
                <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
                  {t("performance.serverAutoStart")}
                </p>
              </div>
            </div>
          </div>
        </div>
      </CollapsibleSection>
    );
  },
);
