import type React from "react";
import { memo } from "react";
import { useTranslation } from "react-i18next";
import { cn, icon as iconTokens, layout, radius, spacing } from "../../../../styles/theme";
import type { NetworkDiscoverySettings } from "../../../../types/settings";

interface DiscoveryTogglesProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Enable/disable toggles for discovery service.
 * Includes main enable toggle and auto-scan on link up option.
 */
export const DiscoveryToggles: React.NamedExoticComponent<DiscoveryTogglesProps> = memo(
  function discoveryToggles({ settings, onSettingsChange }: DiscoveryTogglesProps) {
    const { t } = useTranslation("settings");

    return (
      <>
        {/* Enable Toggle */}
        <label
          class={cn(
            layout.flex.between,
            spacing.pad.xs,
            "bg-surface-base",
            radius.default,
            "border border-surface-border",
          )}
        >
          <div>
            <span class="body-small text-text-primary font-medium">
              {t("discovery.enableDiscovery")}
            </span>
            <p class="caption text-text-muted">{t("discovery.scanForDevices")}</p>
          </div>
          <input
            type="checkbox"
            checked={settings.enabled}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                enabled: e.target.checked,
              }))
            }
            class={iconTokens.size.sm}
          />
        </label>

        {/* Auto-Scan on Link Up */}
        <label
          class={cn(
            layout.flex.between,
            spacing.pad.xs,
            "bg-surface-base",
            radius.default,
            "border border-surface-border",
          )}
        >
          <div>
            <span class="body-small text-text-primary font-medium">
              {t("discovery.autoScanOnLink")}
            </span>
            <p class="caption text-text-muted">{t("discovery.autoScanDesc")}</p>
          </div>
          <input
            type="checkbox"
            checked={settings.autoScan}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                autoScan: e.target.checked,
              }))
            }
            class={iconTokens.size.sm}
          />
        </label>
      </>
    );
  },
);
