import { memo } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  layout,
  spacing,
  radius,
  icon as iconTokens,
} from "../../../../styles/theme";
import type { NetworkDiscoverySettings } from "../../../../types/settings";

interface DiscoveryTogglesProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<
    React.SetStateAction<NetworkDiscoverySettings>
  >;
}

/**
 * Enable/disable toggles for discovery service.
 * Includes main enable toggle and auto-scan on link up option.
 */
export const DiscoveryToggles = memo(function DiscoveryToggles({
  settings,
  onSettingsChange,
}: DiscoveryTogglesProps) {
  const { t } = useTranslation("settings");

  return (
    <>
      {/* Enable Toggle */}
      <label
        className={cn(
          layout.flex.between,
          spacing.pad.xs,
          "bg-surface-base",
          radius.default,
          "border border-surface-border"
        )}
      >
        <div>
          <span className="body-small text-text-primary font-medium">
            {t("discovery.enableDiscovery")}
          </span>
          <p className="caption text-text-muted">
            {t("discovery.scanForDevices")}
          </p>
        </div>
        <input
          type="checkbox"
          checked={settings.enabled}
          onChange={(e) =>
            onSettingsChange((prev) => ({
              ...prev,
              enabled: e.target.checked,
            }))
          }
          className={iconTokens.size.sm}
        />
      </label>

      {/* Auto-Scan on Link Up */}
      <label
        className={cn(
          layout.flex.between,
          spacing.pad.xs,
          "bg-surface-base",
          radius.default,
          "border border-surface-border"
        )}
      >
        <div>
          <span className="body-small text-text-primary font-medium">
            {t("discovery.autoScanOnLink")}
          </span>
          <p className="caption text-text-muted">
            {t("discovery.autoScanDesc")}
          </p>
        </div>
        <input
          type="checkbox"
          checked={settings.autoScan}
          onChange={(e) =>
            onSettingsChange((prev) => ({
              ...prev,
              autoScan: e.target.checked,
            }))
          }
          className={iconTokens.size.sm}
        />
      </label>
    </>
  );
});
