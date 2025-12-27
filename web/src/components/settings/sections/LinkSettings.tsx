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

// Combined speed/duplex mode options matching ethtool output format
// Format: "speed/duplex" (e.g., "10/half", "100/full", "1000/full")
// Note: Half duplex only available at 10/100 Mbps per IEEE standards
const LINK_MODE_OPTIONS: { value: string; label: string }[] = [
  { value: "auto", label: "Auto-Negotiate" },
  // 10 Mbps - supports half and full duplex
  { value: "10/half", label: "10 Mbps Half Duplex" },
  { value: "10/full", label: "10 Mbps Full Duplex" },
  // 100 Mbps - supports half and full duplex
  { value: "100/half", label: "100 Mbps Half Duplex" },
  { value: "100/full", label: "100 Mbps Full Duplex" },
  // 1 Gbps+ - full duplex only (IEEE 802.3)
  { value: "1000/full", label: "1 Gbps Full Duplex" },
  { value: "2500/full", label: "2.5 Gbps Full Duplex" },
  { value: "5000/full", label: "5 Gbps Full Duplex" },
  { value: "10000/full", label: "10 Gbps Full Duplex" },
  // Fiber speeds
  { value: "25000/full", label: "25 Gbps Full Duplex" },
  { value: "40000/full", label: "40 Gbps Full Duplex" },
  { value: "100000/full", label: "100 Gbps Full Duplex" },
];

/**
 * Settings section for link speed and duplex configuration.
 * Uses a single dropdown with combined speed/duplex modes.
 */
export const LinkSettings = memo(function LinkSettings({
  linkSettings,
  setLinkSettings,
  linkStatus,
}: LinkSettingsProps) {
  const { t } = useTranslation("settings");

  // Handle mode change
  const handleModeChange = (mode: string) => {
    setLinkSettings((prev) => ({
      ...prev,
      mode,
    }));
  };

  // Check if current mode is manual (not auto)
  const isManualMode = linkSettings.mode !== "auto";

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
            htmlFor="link-mode"
          >
            {t("link.speedDuplex", "Speed / Duplex")}
          </label>
          <select
            id="link-mode"
            value={linkSettings.mode}
            onChange={(e) => handleModeChange(e.target.value)}
            className={cn(
              "w-full",
              spacing.margin.top.tight,
              spacing.chip.lg,
              "bg-surface-base border border-surface-border",
              radius.default,
              "body-small text-text-primary"
            )}
          >
            {LINK_MODE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        {/* Warning for manual settings */}
        {isManualMode && (
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
