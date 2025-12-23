/**
 * LinkSettings Component
 *
 * Purpose: Configure interface link speed and duplex settings.
 * Single dropdown with all speed/duplex preset combinations.
 *
 * Key Features:
 * - Auto-negotiation option
 * - Combined speed/duplex presets (10M-100G)
 * - Copper: 10/100 support half/full duplex
 * - 1G+: Full duplex only (IEEE standard)
 * - Fiber speeds: 25G/40G/100G
 * - Shows available modes for selected interface
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, theme utilities
 */

import { memo } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { PlugZap } from "../../ui/Icons";
import {
  icon as iconTokens,
  layout,
  radius,
  spacing,
  cn,
} from "../../../styles/theme";
import type {
  LinkSettings as LinkSettingsType,
  SaveStatus,
} from "../../../types/settings";

interface LinkSettingsProps {
  linkSettings: LinkSettingsType;
  setLinkSettings: React.Dispatch<React.SetStateAction<LinkSettingsType>>;
  linkStatus: SaveStatus;
}

// Combined speed/duplex preset options
// Note: Half duplex only available at 10/100 Mbps per IEEE standards
const LINK_PRESETS: {
  value: string;
  label: string;
  speed: string;
  duplex: string;
}[] = [
  { value: "auto", label: "Auto-Negotiate", speed: "auto", duplex: "auto" },
  // 10 Mbps - supports half and full duplex
  {
    value: "10-half",
    label: "10 Mbps Half Duplex",
    speed: "10",
    duplex: "half",
  },
  {
    value: "10-full",
    label: "10 Mbps Full Duplex",
    speed: "10",
    duplex: "full",
  },
  // 100 Mbps - supports half and full duplex
  {
    value: "100-half",
    label: "100 Mbps Half Duplex",
    speed: "100",
    duplex: "half",
  },
  {
    value: "100-full",
    label: "100 Mbps Full Duplex",
    speed: "100",
    duplex: "full",
  },
  // 1 Gbps+ - full duplex only (IEEE 802.3)
  {
    value: "1000-full",
    label: "1 Gbps Full Duplex",
    speed: "1000",
    duplex: "full",
  },
  {
    value: "2500-full",
    label: "2.5 Gbps Full Duplex",
    speed: "2500",
    duplex: "full",
  },
  {
    value: "5000-full",
    label: "5 Gbps Full Duplex",
    speed: "5000",
    duplex: "full",
  },
  {
    value: "10000-full",
    label: "10 Gbps Full Duplex",
    speed: "10000",
    duplex: "full",
  },
  // Fiber speeds
  {
    value: "25000-full",
    label: "25 Gbps Full Duplex",
    speed: "25000",
    duplex: "full",
  },
  {
    value: "40000-full",
    label: "40 Gbps Full Duplex",
    speed: "40000",
    duplex: "full",
  },
  {
    value: "100000-full",
    label: "100 Gbps Full Duplex",
    speed: "100000",
    duplex: "full",
  },
];

/**
 * Settings section for link speed and duplex configuration.
 * Uses a single dropdown with preset speed/duplex combinations.
 */
export const LinkSettings = memo(function LinkSettings({
  linkSettings,
  setLinkSettings,
  linkStatus,
}: LinkSettingsProps) {
  const { t } = useTranslation("settings");

  // Get current preset value from speed/duplex
  const getCurrentPreset = (): string => {
    if (linkSettings.autoNegotiation || linkSettings.speed === "auto") {
      return "auto";
    }
    return `${linkSettings.speed}-${linkSettings.duplex}`;
  };

  // Handle preset change
  const handlePresetChange = (presetValue: string) => {
    const preset = LINK_PRESETS.find((p) => p.value === presetValue);
    if (!preset) return;

    setLinkSettings((prev) => ({
      ...prev,
      autoNegotiation: presetValue === "auto",
      speed: preset.speed as LinkSettingsType["speed"],
      duplex: preset.duplex as LinkSettingsType["duplex"],
    }));
  };

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <PlugZap className={iconTokens.size.sm} />
          <span>{t("sections.link", "Link")}</span>
          <AutoSaveIndicator status={linkStatus} />
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack">
        {/* Combined Speed/Duplex Dropdown */}
        <div>
          <label
            className="caption text-text-muted font-medium"
            htmlFor="link-preset"
          >
            {t("link.speedDuplex", "Speed / Duplex")}
          </label>
          <select
            id="link-preset"
            value={getCurrentPreset()}
            onChange={(e) => handlePresetChange(e.target.value)}
            className={cn(
              "w-full",
              spacing.margin.top.tight,
              spacing.chip.lg,
              "bg-surface-base border border-surface-border",
              radius.default,
              "body-small text-text-primary"
            )}
          >
            {LINK_PRESETS.map((preset) => (
              <option key={preset.value} value={preset.value}>
                {preset.label}
              </option>
            ))}
          </select>
        </div>

        {/* Warning for manual settings */}
        {!linkSettings.autoNegotiation && (
          <p
            className={cn(
              "caption text-status-warning",
              spacing.margin.top.inline
            )}
          >
            {t(
              "link.manualWarning",
              "Manual speed/duplex may cause link issues if mismatched with switch"
            )}
          </p>
        )}

        {/* Available Modes Display */}
        {linkSettings.availableModes.length > 0 && (
          <div
            className={cn(
              "border-t border-surface-border",
              spacing.padding.top.heading
            )}
          >
            <span className="caption text-text-muted font-medium">
              {t("link.availableModes", "Supported Modes")}
            </span>
            <div
              className={cn(
                "flex flex-wrap",
                spacing.gap.tight,
                spacing.margin.top.inline
              )}
            >
              {linkSettings.availableModes.map((mode) => (
                <span
                  key={mode}
                  className={cn(
                    spacing.chip.sm,
                    "bg-surface-base border border-surface-border",
                    radius.default,
                    "caption text-text-muted"
                  )}
                >
                  {mode}
                </span>
              ))}
            </div>
          </div>
        )}

        <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
          {t(
            "link.requiresRoot",
            "Changing link settings requires root privileges"
          )}
        </p>
      </div>
    </CollapsibleSection>
  );
});
