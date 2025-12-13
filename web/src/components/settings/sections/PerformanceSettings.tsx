import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Gauge } from "../../ui/Icons";
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
        <div className="flex items-center gap-2">
          <Gauge className="w-4 h-4" />
          <span>Performance Configuration</span>
          <AutoSaveIndicator status={iperfStatus} />
        </div>
      }
    >
      <div className="space-y-4">
        <p className="text-xs text-text-muted">
          Configure settings for performance tests. Enable/disable tests in Test
          Options.
        </p>

        {/* Internet Speed (Speedtest) Subsection */}
        <div className="border-b border-surface-border pb-4">
          <h4 className="text-sm font-semibold text-text-primary mb-2 uppercase tracking-wide">
            Internet Speed (Speedtest)
          </h4>
          <div className="space-y-3 pl-1">
            <div>
              <label className="text-xs text-text-muted font-medium">
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
                className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
              />
              <div className="flex items-center justify-between mt-1">
                <p className="text-xs text-text-muted">
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
                  className="text-xs text-brand-primary hover:underline"
                >
                  Reset to Auto
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* LAN Speed (iperf3) Subsection */}
        <div>
          <h4 className="text-sm font-semibold text-text-primary mb-2 uppercase tracking-wide">
            LAN Speed (iperf3)
          </h4>
          <div className="space-y-3 pl-1">
            <p className="text-xs text-text-muted">
              Configure iperf3 client settings for LAN speed tests.
            </p>

            {/* Server Address */}
            <div>
              <label className="text-xs text-text-muted font-medium">
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
                className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary disabled:opacity-60"
              />
              <div className="flex items-center justify-between mt-2">
                <button
                  type="button"
                  disabled={iperfSuggestionsStatus === "loading"}
                  onClick={fetchIperfSuggestions}
                  className="text-xs text-brand-primary hover:underline disabled:opacity-60 disabled:cursor-not-allowed"
                >
                  {iperfSuggestionsStatus === "loading"
                    ? "Scanning..."
                    : "Find iperf hosts on LAN"}
                </button>
                {iperfSuggestionsStatus === "loading" && (
                  <svg
                    className="w-4 h-4 animate-spin text-text-muted"
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
                <p className="text-xs text-status-warning mt-1">
                  {iperfSuggestionsError || "No iperf hosts responded"}
                </p>
              )}
              {iperfSuggestions.length > 0 && (
                <div className="flex flex-wrap gap-2 mt-2">
                  {iperfSuggestions.map((sugg) => (
                    <button
                      type="button"
                      key={`${sugg.host}-${sugg.hostname || ""}`}
                      className="px-3 py-1 rounded-full border border-surface-border bg-surface-base text-xs text-text-primary hover:bg-surface-hover"
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
              <label className="text-xs text-text-muted font-medium">
                Port
              </label>
              <input
                type="number"
                value={iperfSettings.port}
                onChange={(e) =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    port: parseInt(e.target.value) || 5201,
                  }))
                }
                className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary disabled:opacity-60"
              />
            </div>

            {/* Protocol Toggle */}
            <div>
              <label className="text-xs text-text-muted font-medium block mb-1">
                Protocol
              </label>
              <div className="flex flex-wrap gap-2">
                {(["tcp", "udp"] as const).map((proto) => (
                  <button
                    key={proto}
                    type="button"
                    onClick={() =>
                      setIperfSettings((prev) => ({
                        ...prev,
                        protocol: proto,
                      }))
                    }
                    aria-pressed={iperfSettings.protocol === proto}
                    className={`px-3 py-1.5 rounded-full border text-sm font-medium transition-colors ${
                      iperfSettings.protocol === proto
                        ? "bg-brand-primary text-text-inverse border-brand-primary"
                        : "bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover"
                    }`}
                  >
                    {proto.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>

            {/* Direction Toggle */}
            <div>
              <label className="text-xs text-text-muted font-medium block mb-1">
                Direction
              </label>
              <div className="flex flex-wrap gap-2">
                {(
                  [
                    { value: "download", label: "Download" },
                    { value: "upload", label: "Upload" },
                    { value: "bidirectional", label: "Both" },
                  ] as const
                ).map((option) => (
                  <button
                    key={option.value}
                    type="button"
                    onClick={() =>
                      setIperfSettings((prev) => ({
                        ...prev,
                        direction: option.value,
                      }))
                    }
                    aria-pressed={iperfSettings.direction === option.value}
                    className={`px-3 py-1.5 rounded-full border text-sm font-medium transition-colors ${
                      iperfSettings.direction === option.value
                        ? "bg-brand-primary text-text-inverse border-brand-primary"
                        : "bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover"
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Duration */}
            <div>
              <label className="text-xs text-text-muted font-medium">
                Duration (seconds)
              </label>
              <input
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
                className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary disabled:opacity-60"
              />
            </div>

            {/* Server Mode */}
            <div className="border-t border-surface-border pt-3">
              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border mb-2">
                <span className="text-sm text-text-primary">
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
                  className="w-4 h-4"
                />
              </label>
              <div>
                <label className="text-xs text-text-muted font-medium">
                  Server Port
                </label>
                <input
                  type="number"
                  value={iperfSettings.serverPort}
                  onChange={(e) =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      serverPort: parseInt(e.target.value) || 5201,
                    }))
                  }
                  className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary disabled:opacity-60"
                />
              </div>
              <p className="text-xs text-text-muted mt-1">
                When enabled, starts iperf3 server automatically
              </p>
            </div>
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
}
