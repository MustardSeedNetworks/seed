import { memo } from "react";
import { useTranslation } from "react-i18next";
import { cn, spacing, radius } from "../../../../styles/theme";
import type { NetworkDiscoverySettings } from "../../../../types/settings";

interface DiscoveryTimingSettingsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<
    React.SetStateAction<NetworkDiscoverySettings>
  >;
}

/**
 * Timing and performance settings for discovery.
 * Includes workers, timeouts, and rescan intervals.
 */
export const DiscoveryTimingSettings = memo(function DiscoveryTimingSettings({
  settings,
  onSettingsChange,
}: DiscoveryTimingSettingsProps) {
  const { t } = useTranslation("settings");

  return (
    <div className={cn("border-t border-surface-border", spacing.pad.sm)}>
      <span className="caption text-text-muted font-medium">
        {t("discovery.timingSettings")}
      </span>

      {/* Scan Workers */}
      <div className={spacing.margin.top.inline}>
        <label className="caption text-text-muted" htmlFor="discovery-workers">
          {t("discovery.concurrentWorkers")}
        </label>
        <input
          id="discovery-workers"
          type="number"
          value={settings.arpScanWorkers}
          onChange={(e) =>
            onSettingsChange((prev) => ({
              ...prev,
              arpScanWorkers: parseInt(e.target.value) || 50,
            }))
          }
          min={1}
          max={100}
          className={cn(
            "w-full",
            spacing.margin.top.tight,
            spacing.chip.lg,
            "bg-surface-base border border-surface-border",
            radius.default,
            "body-small text-text-primary"
          )}
        />
        <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
          {t("discovery.workersDesc")}
        </p>
      </div>

      {/* Ping Timeout */}
      <div className={spacing.margin.top.content}>
        <label
          className="caption text-text-muted"
          htmlFor="discovery-ping-timeout"
        >
          {t("discovery.pingTimeout")}
        </label>
        <input
          id="discovery-ping-timeout"
          type="number"
          value={settings.pingTimeoutMs}
          onChange={(e) =>
            onSettingsChange((prev) => ({
              ...prev,
              pingTimeoutMs: parseInt(e.target.value) || 500,
            }))
          }
          min={100}
          max={5000}
          className={cn(
            "w-full",
            spacing.margin.top.tight,
            spacing.chip.lg,
            "bg-surface-base border border-surface-border",
            radius.default,
            "body-small text-text-primary"
          )}
        />
      </div>

      {/* Scan Timeout */}
      <div className={spacing.margin.top.content}>
        <label
          className="caption text-text-muted"
          htmlFor="discovery-scan-timeout"
        >
          {t("discovery.scanTimeout")}
        </label>
        <input
          id="discovery-scan-timeout"
          type="number"
          value={settings.scanTimeoutMs}
          onChange={(e) =>
            onSettingsChange((prev) => ({
              ...prev,
              scanTimeoutMs: parseInt(e.target.value) || 30000,
            }))
          }
          min={5000}
          max={120000}
          className={cn(
            "w-full",
            spacing.margin.top.tight,
            spacing.chip.lg,
            "bg-surface-base border border-surface-border",
            radius.default,
            "body-small text-text-primary"
          )}
        />
      </div>

      {/* Rescan Interval */}
      <div className={spacing.margin.top.content}>
        <label
          className="caption text-text-muted"
          htmlFor="discovery-rescan-interval"
        >
          {t("discovery.rescanInterval")}
        </label>
        <input
          id="discovery-rescan-interval"
          type="number"
          value={settings.scanIntervalMs}
          onChange={(e) =>
            onSettingsChange((prev) => ({
              ...prev,
              scanIntervalMs: parseInt(e.target.value) || 0,
            }))
          }
          min={0}
          className={cn(
            "w-full",
            spacing.margin.top.tight,
            spacing.chip.lg,
            "bg-surface-base border border-surface-border",
            radius.default,
            "body-small text-text-primary"
          )}
        />
        <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
          {t("discovery.rescanIntervalDesc")}
        </p>
      </div>

      {/* OUI database is baked into binary at build time - no runtime path needed */}
    </div>
  );
});
