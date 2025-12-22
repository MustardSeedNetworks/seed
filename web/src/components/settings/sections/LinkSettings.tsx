/**
 * LinkSettings Component
 *
 * Purpose: Configure interface link speed and duplex settings.
 * Allows users to select auto-negotiation or fixed speed/duplex modes.
 *
 * Key Features:
 * - Auto-negotiation toggle
 * - Speed selection (10/100/1000/2500/5000/10000 Mbps)
 * - Duplex selection (Full/Half)
 * - Shows available modes for selected interface
 * - AutoSaveIndicator for save status
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, theme utilities
 * State: Manages link configuration settings
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
  LinkSpeed,
  DuplexMode,
  SaveStatus,
} from "../../../types/settings";

interface LinkSettingsProps {
  linkSettings: LinkSettingsType;
  setLinkSettings: React.Dispatch<React.SetStateAction<LinkSettingsType>>;
  linkStatus: SaveStatus;
}

// Speed options with labels
const SPEED_OPTIONS: { value: LinkSpeed; label: string }[] = [
  { value: "auto", label: "Auto" },
  { value: "10", label: "10 Mbps" },
  { value: "100", label: "100 Mbps" },
  { value: "1000", label: "1 Gbps" },
  { value: "2500", label: "2.5 Gbps" },
  { value: "5000", label: "5 Gbps" },
  { value: "10000", label: "10 Gbps" },
];

// Duplex options with labels
const DUPLEX_OPTIONS: { value: DuplexMode; label: string }[] = [
  { value: "auto", label: "Auto" },
  { value: "full", label: "Full Duplex" },
  { value: "half", label: "Half Duplex" },
];

/**
 * Settings section for link speed and duplex configuration.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const LinkSettings = memo(function LinkSettings({
  linkSettings,
  setLinkSettings,
  linkStatus,
}: LinkSettingsProps) {
  const { t } = useTranslation("settings");

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
        {/* Auto-Negotiation Toggle */}
        <label
          className={cn(
            layout.flex.between,
            spacing.pad.sm,
            "bg-surface-base",
            radius.default,
            "border border-surface-border"
          )}
        >
          <div>
            <span className="body-small text-text-primary font-medium">
              {t("link.autoNegotiation", "Auto-Negotiation")}
            </span>
            <p className="caption text-text-muted">
              {t(
                "link.autoNegotiationDesc",
                "Automatically negotiate best speed and duplex"
              )}
            </p>
          </div>
          <input
            type="checkbox"
            checked={linkSettings.autoNegotiation}
            onChange={(e) =>
              setLinkSettings((prev) => ({
                ...prev,
                autoNegotiation: e.target.checked,
                // Reset to auto when enabling auto-negotiation
                speed: e.target.checked ? "auto" : prev.speed,
                duplex: e.target.checked ? "auto" : prev.duplex,
              }))
            }
            className={iconTokens.size.sm}
          />
        </label>

        {/* Manual Speed/Duplex Configuration (only when auto-neg is off) */}
        {!linkSettings.autoNegotiation && (
          <>
            {/* Speed Selection */}
            <div
              className={cn(
                "border-t border-surface-border",
                spacing.padding.top.heading
              )}
            >
              <label
                className="caption text-text-muted font-medium"
                htmlFor="link-speed"
              >
                {t("link.speed", "Link Speed")}
              </label>
              <select
                id="link-speed"
                value={linkSettings.speed}
                onChange={(e) =>
                  setLinkSettings((prev) => ({
                    ...prev,
                    speed: e.target.value as LinkSpeed,
                  }))
                }
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  spacing.chip.lg,
                  "bg-surface-base border border-surface-border",
                  radius.default,
                  "body-small text-text-primary"
                )}
              >
                {SPEED_OPTIONS.filter((opt) => opt.value !== "auto").map(
                  (option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  )
                )}
              </select>
            </div>

            {/* Duplex Selection */}
            <div className={spacing.margin.top.content}>
              <label
                className="caption text-text-muted font-medium"
                htmlFor="link-duplex"
              >
                {t("link.duplex", "Duplex Mode")}
              </label>
              <select
                id="link-duplex"
                value={linkSettings.duplex}
                onChange={(e) =>
                  setLinkSettings((prev) => ({
                    ...prev,
                    duplex: e.target.value as DuplexMode,
                  }))
                }
                className={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  spacing.chip.lg,
                  "bg-surface-base border border-surface-border",
                  radius.default,
                  "body-small text-text-primary"
                )}
              >
                {DUPLEX_OPTIONS.filter((opt) => opt.value !== "auto").map(
                  (option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  )
                )}
              </select>
            </div>

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
          </>
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
