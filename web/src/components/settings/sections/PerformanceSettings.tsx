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

import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Gauge } from "../../ui/Icons";
import { icon as iconTokens, layout, radius } from "../../../styles/theme";
import {
  TestsSettings,
  IperfSettings,
  IperfSuggestion,
  SaveStatus,
} from "../../../types/settings";

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

export function PerformanceSettings({
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
  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Gauge className={iconTokens.size.sm} />
          <span>Performance</span>
          <AutoSaveIndicator status={iperfStatus} />
        </div>
      }
    >
      <div className="stack">
        {/* Enable/Disable Toggles */}
        <div className="stack-sm">
          <label
            className={`${layout.flex.between} p-2.5 bg-surface-base ${radius.default} border border-surface-border`}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                Enable Speedtest
              </span>
              <p className="caption text-text-muted">
                Test internet speed via Speedtest.net
              </p>
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
            className={`${layout.flex.between} p-2.5 bg-surface-base ${radius.default} border border-surface-border`}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                Enable iPerf
              </span>
              <p className="caption text-text-muted">
                Test LAN speed via iperf3
              </p>
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
        <div className="border-t border-surface-border pt-3">
          <span className="caption text-text-muted font-medium">
            Auto-Run on Link Up
          </span>
          <div className="mt-2 stack-sm">
            <label
              className={`${layout.flex.between} p-2.5 bg-surface-base ${radius.default} border border-surface-border`}
            >
              <span className="body-small text-text-primary">Speedtest</span>
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
              className={`${layout.flex.between} p-2.5 bg-surface-base ${radius.default} border border-surface-border`}
            >
              <span className="body-small text-text-primary">iPerf</span>
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
        <div className="border-t border-surface-border pt-3">
          <h4 className="body-small font-semibold text-text-primary mb-2 uppercase tracking-wide">
            Internet Speed (Speedtest)
          </h4>
          <div className="stack pl-1">
            <div>
              <label className="caption text-text-muted font-medium">
                Server ID (optional)
              </label>
              <input
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
                placeholder="Auto (closest server)"
                className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
              />
              <div className={`${layout.flex.between} mt-1`}>
                <p className="caption text-text-muted">
                  Leave empty to auto-select nearest server
                </p>
                <button
                  type="button"
                  onClick={() =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      speedtest: { ...prev.speedtest, serverId: "" },
                    }))
                  }
                  className="caption text-brand-primary hover:underline"
                >
                  Reset to Auto
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* LAN Speed (iperf3) Subsection */}
        <div>
          <h4 className="body-small font-semibold text-text-primary mb-2 uppercase tracking-wide">
            LAN Speed (iperf3)
          </h4>
          <div className="stack pl-1">
            <p className="caption text-text-muted">
              Configure iperf3 client settings for LAN speed tests.
            </p>

            {/* Server Address */}
            <div>
              <label className="caption text-text-muted font-medium">
                Server Address
              </label>
              <input
                type="text"
                value={iperfSettings.server}
                onChange={(e) =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    server: e.target.value,
                  }))
                }
                placeholder="192.168.1.100 or hostname"
                className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary disabled:opacity-60`}
              />
              <div className={`${layout.flex.between} mt-2`}>
                <button
                  type="button"
                  disabled={iperfSuggestionsStatus === "loading"}
                  onClick={fetchIperfSuggestions}
                  className="caption text-brand-primary hover:underline disabled:opacity-60 disabled:cursor-not-allowed"
                >
                  {iperfSuggestionsStatus === "loading"
                    ? "Scanning..."
                    : "Find iperf hosts on LAN"}
                </button>
                {iperfSuggestionsStatus === "loading" && (
                  <svg
                    className={`${iconTokens.size.sm} animate-spin text-text-muted`}
                    viewBox="0 0 24 24"
                    fill="none"
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
                <p className="caption text-status-warning mt-1">
                  {iperfSuggestionsError || "No iperf hosts responded"}
                </p>
              )}
              {iperfSuggestions.length > 0 && (
                <div className="flex flex-wrap gap-2 mt-2">
                  {iperfSuggestions.map((sugg) => (
                    <button
                      type="button"
                      key={`${sugg.host}-${sugg.hostname || ""}`}
                      className={`px-3 py-1 ${radius.full} border border-surface-border bg-surface-base caption text-text-primary hover:bg-surface-hover`}
                      onClick={() =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          server: sugg.host,
                        }))
                      }
                    >
                      <span className="font-medium">
                        {sugg.hostname || sugg.host}
                      </span>
                      <span className="text-text-muted ml-1">
                        {sugg.hostname ? `(${sugg.host})` : ""}
                        {sugg.latencyMs !== undefined
                          ? ` · ${Math.round(sugg.latencyMs)}ms`
                          : ""}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Port */}
            <div>
              <label
                className="caption text-text-muted font-medium"
                htmlFor="iperf-port"
              >
                Port
              </label>
              <input
                id="iperf-port"
                type="number"
                value={iperfSettings.port}
                onChange={(e) =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    port: parseInt(e.target.value) || 5201,
                  }))
                }
                className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary disabled:opacity-60`}
              />
            </div>

            {/* Protocol Toggle */}
            <div>
              <label className="caption text-text-muted font-medium block mb-1">
                Protocol
              </label>
              <div
                className="flex flex-wrap gap-2"
                role="radiogroup"
                aria-label="Protocol selection"
              >
                {(["tcp", "udp"] as const).map((proto) => {
                  const checked = iperfSettings.protocol === proto;
                  return (
                    <label
                      key={proto}
                      className={`cursor-pointer px-3 py-1.5 ${radius.full} border body-small font-medium transition-colors ${
                        checked
                          ? "bg-brand-primary text-text-inverse border-brand-primary"
                          : "bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover"
                      }`}
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
              <label className="caption text-text-muted font-medium block mb-1">
                Direction
              </label>
              <div
                className="flex flex-wrap gap-2"
                role="radiogroup"
                aria-label="Direction selection"
              >
                {(
                  [
                    { value: "download", label: "Download" },
                    { value: "upload", label: "Upload" },
                    { value: "bidirectional", label: "Both" },
                  ] as const
                ).map((option) => {
                  const checked = iperfSettings.direction === option.value;
                  return (
                    <label
                      key={option.value}
                      className={`cursor-pointer px-3 py-1.5 ${radius.full} border body-small font-medium transition-colors ${
                        checked
                          ? "bg-brand-primary text-text-inverse border-brand-primary"
                          : "bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover"
                      }`}
                    >
                      <input
                        type="radio"
                        name="iperf-direction"
                        value={option.value}
                        checked={checked}
                        onChange={() =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            direction: option.value,
                          }))
                        }
                        className="sr-only"
                        aria-label={`${option.label} direction`}
                      />
                      {option.label}
                    </label>
                  );
                })}
              </div>
            </div>

            {/* Duration */}
            <div>
              <label
                className="caption text-text-muted font-medium"
                htmlFor="iperf-duration"
              >
                Duration (seconds)
              </label>
              <input
                id="iperf-duration"
                type="number"
                value={iperfSettings.duration}
                onChange={(e) =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    duration: parseInt(e.target.value) || 10,
                  }))
                }
                min={1}
                max={60}
                className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary disabled:opacity-60`}
              />
            </div>

            {/* Server Mode */}
            <div className="border-t border-surface-border pt-3">
              <label
                className={`${layout.flex.between} p-2.5 bg-surface-base ${radius.default} border border-surface-border mb-2`}
              >
                <span className="body-small text-text-primary">
                  Enable iperf3 Server
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
                <label
                  className="caption text-text-muted font-medium"
                  htmlFor="iperf-server-port"
                >
                  Server Port
                </label>
                <input
                  id="iperf-server-port"
                  type="number"
                  value={iperfSettings.serverPort}
                  onChange={(e) =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      serverPort: parseInt(e.target.value) || 5201,
                    }))
                  }
                  className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary disabled:opacity-60`}
                />
              </div>
              <p className="caption text-text-muted mt-1">
                When enabled, starts iperf3 server automatically
              </p>
            </div>
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
}
