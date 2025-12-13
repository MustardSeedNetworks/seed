import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Settings } from "../../ui/Icons";
import { TestsSettings, FABOptions, SaveStatus } from "../../../types/settings";

interface TestOptionsSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  fabOptions: FABOptions;
  setFabOptions: React.Dispatch<React.SetStateAction<FABOptions>>;
  testsStatus: SaveStatus;
  fabStatus: SaveStatus;
}

export function TestOptionsSettings({
  testsSettings,
  setTestsSettings,
  fabOptions,
  setFabOptions,
  testsStatus,
  fabStatus,
}: TestOptionsSettingsProps) {
  return (
    <CollapsibleSection
      title={
        <div className="flex items-center gap-2">
          <Settings className="w-4 h-4" />
          <span>Test Options</span>
          <AutoSaveIndicator
            status={
              testsStatus === "saving" || fabStatus === "saving"
                ? "saving"
                : testsStatus === "saved" && fabStatus === "saved"
                  ? "saved"
                  : testsStatus === "error" || fabStatus === "error"
                    ? "error"
                    : "idle"
            }
          />
        </div>
      }
    >
      <div className="space-y-4">
        {/* Enable/Disable Tests */}
        <div>
          <h4 className="text-sm font-semibold text-text-primary mb-2 uppercase tracking-wide">
            Enable Tests
          </h4>
          <p className="text-xs text-text-muted mb-3">
            Master toggles to enable or disable each test type globally.
          </p>
          <div className="space-y-2">
            <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
              <div>
                <span className="text-sm text-text-primary">
                  Internet Speed Test
                </span>
                <p className="text-xs text-text-muted">Speedtest.net</p>
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
                className="w-4 h-4"
              />
            </label>

            <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
              <div>
                <span className="text-sm text-text-primary">
                  LAN Speed Test
                </span>
                <p className="text-xs text-text-muted">iperf3</p>
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
                className="w-4 h-4"
              />
            </label>

            <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
              <div>
                <span className="text-sm text-text-primary">
                  Network Discovery
                </span>
                <p className="text-xs text-text-muted">
                  ARP scan, device detection
                </p>
              </div>
              <input
                type="checkbox"
                checked={testsSettings.runDiscovery}
                onChange={(e) =>
                  setTestsSettings((prev) => ({
                    ...prev,
                    runDiscovery: e.target.checked,
                  }))
                }
                className="w-4 h-4"
              />
            </label>
          </div>
        </div>

        {/* Auto-Run on Link Up */}
        <div className="border-t border-surface-border pt-4">
          <h4 className="text-sm font-semibold text-text-primary mb-2 uppercase tracking-wide">
            Auto-Run on Link Up
          </h4>
          <p className="text-xs text-text-muted mb-3">
            Automatically run these tests when the network interface comes up.
          </p>
          <div className="space-y-2">
            <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
              <div>
                <span
                  className={`text-sm text-text-primary ${!testsSettings.runSpeedtest ? "opacity-60" : ""}`}
                >
                  Internet Speed Test
                </span>
                <p
                  className={`text-xs text-text-muted ${!testsSettings.runSpeedtest ? "opacity-60" : ""}`}
                >
                  Run Speedtest when link detected
                </p>
              </div>
              <input
                type="checkbox"
                disabled={!testsSettings.runSpeedtest}
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
                className="w-4 h-4"
              />
            </label>

            <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
              <div>
                <span
                  className={`text-sm text-text-primary ${!testsSettings.runIperf ? "opacity-60" : ""}`}
                >
                  LAN Speed Test
                </span>
                <p
                  className={`text-xs text-text-muted ${!testsSettings.runIperf ? "opacity-60" : ""}`}
                >
                  Run iperf3 when link detected
                </p>
              </div>
              <input
                type="checkbox"
                disabled={!testsSettings.runIperf}
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
                className="w-4 h-4"
              />
            </label>

            <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
              <div>
                <span
                  className={`text-sm text-text-primary ${!testsSettings.runDiscovery ? "opacity-60" : ""}`}
                >
                  Network Discovery
                </span>
                <p
                  className={`text-xs text-text-muted ${!testsSettings.runDiscovery ? "opacity-60" : ""}`}
                >
                  Scan network when link detected
                </p>
              </div>
              <input
                type="checkbox"
                disabled={!testsSettings.runDiscovery}
                checked={fabOptions.autoScanOnLink}
                onChange={(e) =>
                  setFabOptions((prev) => ({
                    ...prev,
                    autoScanOnLink: e.target.checked,
                  }))
                }
                className="w-4 h-4"
              />
            </label>
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
}
