import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { FABOptions, SaveStatus } from "../../../types/settings";

interface FABOptionsSettingsProps {
  fabOptions: FABOptions;
  setFabOptions: React.Dispatch<React.SetStateAction<FABOptions>>;
  fabStatus: SaveStatus;
}

export function FABOptionsSettings({
  fabOptions,
  setFabOptions,
  fabStatus,
}: FABOptionsSettingsProps) {
  return (
    <CollapsibleSection
      title={
        <>
          Run All Tests (FAB)
          <AutoSaveIndicator status={fabStatus} />
        </>
      }
    >
      <div className="space-y-3">
        <p className="text-xs text-text-muted">
          Configure which tests run when the FAB button is pressed. Order
          matches card display.
        </p>

        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">Link</span>
          <input
            type="checkbox"
            checked={fabOptions.runLink}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runLink: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>

        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">Nearest Switch</span>
          <input
            type="checkbox"
            checked={fabOptions.runSwitch}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runSwitch: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>

        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">VLAN</span>
          <input
            type="checkbox"
            checked={fabOptions.runVLAN}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runVLAN: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>

        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">IP Config</span>
          <input
            type="checkbox"
            checked={fabOptions.runIPConfig}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runIPConfig: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>

        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">Gateway</span>
          <input
            type="checkbox"
            checked={fabOptions.runGateway}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runGateway: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>

        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">DNS</span>
          <input
            type="checkbox"
            checked={fabOptions.runDNS}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runDNS: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>

        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">Health Checks</span>
          <input
            type="checkbox"
            checked={fabOptions.runHealthChecks}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runHealthChecks: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>

        {/* Performance tests block */}
        <div className="p-2.5 bg-surface-base rounded border border-surface-border space-y-3">
          <label className="flex items-center justify-between">
            <span className="text-sm font-medium text-text-primary">
              Performance
            </span>
            <input
              type="checkbox"
              checked={fabOptions.runPerformance}
              onChange={(e) =>
                setFabOptions((prev) => ({
                  ...prev,
                  runPerformance: e.target.checked,
                }))
              }
              className="w-4 h-4"
            />
          </label>

          <div className="pl-4 space-y-2 border-l-2 border-surface-border">
            <label className="flex items-center justify-between">
              <span
                className={`text-sm text-text-primary ${!fabOptions.runPerformance ? "opacity-60" : ""}`}
              >
                Internet Speed (Speedtest)
              </span>
              <input
                type="checkbox"
                disabled={!fabOptions.runPerformance}
                checked={fabOptions.runSpeedtest}
                onChange={(e) =>
                  setFabOptions((prev) => ({
                    ...prev,
                    runSpeedtest: e.target.checked,
                  }))
                }
                className="w-4 h-4"
              />
            </label>
            <label className="flex items-center justify-between">
              <span
                className={`text-sm text-text-primary ${!fabOptions.runPerformance ? "opacity-60" : ""}`}
              >
                LAN Speed (iperf3)
              </span>
              <input
                type="checkbox"
                disabled={!fabOptions.runPerformance}
                checked={fabOptions.runIperf}
                onChange={(e) =>
                  setFabOptions((prev) => ({
                    ...prev,
                    runIperf: e.target.checked,
                  }))
                }
                className="w-4 h-4"
              />
            </label>
          </div>
        </div>

        {/* Network discovery */}
        <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">Network Discovery</span>
          <input
            type="checkbox"
            checked={fabOptions.runNetworkDiscovery}
            onChange={(e) =>
              setFabOptions((prev) => ({
                ...prev,
                runNetworkDiscovery: e.target.checked,
              }))
            }
            className="w-4 h-4"
          />
        </label>
      </div>
    </CollapsibleSection>
  );
}
