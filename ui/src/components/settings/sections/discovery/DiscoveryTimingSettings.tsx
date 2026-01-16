import type React from "react";
import { memo } from "react";
import { useTranslation } from "react-i18next";
import { cn, radius, spacing } from "../../../../styles/theme";
import type { NetworkDiscoverySettings } from "../../../../types/settings";

interface DiscoveryTimingSettingsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Timing and performance settings for discovery.
 * Includes workers, timeouts, and rescan intervals.
 */
export const DiscoveryTimingSettings: React.NamedExoticComponent<DiscoveryTimingSettingsProps> =
  memo(function discoveryTimingSettings({
    settings,
    onSettingsChange,
  }: DiscoveryTimingSettingsProps) {
    const { t } = useTranslation("settings");

    return (
      <div class={cn("border-t border-surface-border", spacing.pad.sm)}>
        <span class="caption text-text-muted font-medium">{t("discovery.timingSettings")}</span>

        {/* Scan Workers */}
        <div class={spacing.margin.top.inline}>
          <label class="caption text-text-muted" for="discovery-workers">
            {t("discovery.concurrentWorkers")}
          </label>
          <input
            id="discovery-workers"
            type="number"
            value={settings.arpScanWorkers}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                arpScanWorkers: Number.parseInt(e.target.value, 10) || 50,
              }))
            }
            min={1}
            max={100}
            class={cn(
              "w-full",
              spacing.margin.top.tight,
              spacing.chip.lg,
              "bg-surface-base border border-surface-border",
              radius.default,
              "body-small text-text-primary",
            )}
          />
          <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
            {t("discovery.workersDesc")}
          </p>
        </div>

        {/* Ping Timeout */}
        <div class={spacing.margin.top.content}>
          <label class="caption text-text-muted" for="discovery-ping-timeout">
            {t("discovery.pingTimeout")}
          </label>
          <input
            id="discovery-ping-timeout"
            type="number"
            value={settings.pingTimeoutMs}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                pingTimeoutMs: Number.parseInt(e.target.value, 10) || 500,
              }))
            }
            min={100}
            max={5000}
            class={cn(
              "w-full",
              spacing.margin.top.tight,
              spacing.chip.lg,
              "bg-surface-base border border-surface-border",
              radius.default,
              "body-small text-text-primary",
            )}
          />
        </div>

        {/* Scan Timeout */}
        <div class={spacing.margin.top.content}>
          <label class="caption text-text-muted" for="discovery-scan-timeout">
            {t("discovery.scanTimeout")}
          </label>
          <input
            id="discovery-scan-timeout"
            type="number"
            value={settings.scanTimeoutMs}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                scanTimeoutMs: Number.parseInt(e.target.value, 10) || 30000,
              }))
            }
            min={5000}
            max={120000}
            class={cn(
              "w-full",
              spacing.margin.top.tight,
              spacing.chip.lg,
              "bg-surface-base border border-surface-border",
              radius.default,
              "body-small text-text-primary",
            )}
          />
        </div>

        {/* Rescan Interval */}
        <div class={spacing.margin.top.content}>
          <label class="caption text-text-muted" for="discovery-rescan-interval">
            {t("discovery.rescanInterval")}
          </label>
          <input
            id="discovery-rescan-interval"
            type="number"
            value={settings.scanIntervalMs}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                scanIntervalMs: Number.parseInt(e.target.value, 10) || 0,
              }))
            }
            min={0}
            class={cn(
              "w-full",
              spacing.margin.top.tight,
              spacing.chip.lg,
              "bg-surface-base border border-surface-border",
              radius.default,
              "body-small text-text-primary",
            )}
          />
          <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
            {t("discovery.rescanIntervalDesc")}
          </p>
        </div>

        {/* OUI database is baked into binary at build time - no runtime path needed */}
      </div>
    );
  });
