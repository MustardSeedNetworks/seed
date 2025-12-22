/**
 * CableTestSettings Component
 *
 * Purpose: Configure cable test (TDR) settings.
 * Allows users to enable/disable cable testing and configure test behavior.
 *
 * Key Features:
 * - Enable/disable cable test card
 * - Auto-run on link down option
 * - Length unit selection (feet/meters)
 * - TDR support status display
 * - AutoSaveIndicator for save status
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, theme utilities
 * State: Manages cable test configuration settings
 */

import { memo, useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Cable } from "../../ui/Icons";
import {
  icon as iconTokens,
  layout,
  radius,
  spacing,
  cn,
} from "../../../styles/theme";
import type {
  CableTestSettings as CableTestSettingsType,
  SaveStatus,
} from "../../../types/settings";

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface CableTestSettingsProps {
  cableTestSettings: CableTestSettingsType;
  setCableTestSettings: React.Dispatch<
    React.SetStateAction<CableTestSettingsType>
  >;
  cableTestStatus: SaveStatus;
}

interface TDRSupportStatus {
  supported: boolean;
  driver?: string;
  message?: string;
}

/**
 * Settings section for cable test (TDR) configuration.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const CableTestSettings = memo(function CableTestSettings({
  cableTestSettings,
  setCableTestSettings,
  cableTestStatus,
}: CableTestSettingsProps) {
  const { t } = useTranslation("settings");
  const [tdrSupport, setTdrSupport] = useState<TDRSupportStatus | null>(null);
  const [checkingSupport, setCheckingSupport] = useState(false);

  // Check TDR support on mount
  const checkTDRSupport = useCallback(async () => {
    setCheckingSupport(true);
    try {
      const response = await fetch(`${API_BASE}/api/cable/support`, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setTdrSupport(data);
      } else {
        setTdrSupport({ supported: false, message: "Unable to check support" });
      }
    } catch {
      setTdrSupport({ supported: false, message: "Network error" });
    } finally {
      setCheckingSupport(false);
    }
  }, []);

  useEffect(() => {
    checkTDRSupport();
  }, [checkTDRSupport]);

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Cable className={iconTokens.size.sm} />
          <span>{t("sections.cableTest", "Cable Test")}</span>
          <AutoSaveIndicator status={cableTestStatus} />
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack">
        {/* TDR Support Status */}
        <div
          className={cn(
            spacing.pad.sm,
            radius.lg,
            "border",
            tdrSupport?.supported
              ? "bg-status-success/10 border-status-success/30"
              : "bg-surface-base border-surface-border"
          )}
        >
          <div className={layout.flex.between}>
            <div className={layout.inline.default}>
              <div
                className={cn(
                  "w-2 h-2",
                  radius.full,
                  checkingSupport
                    ? "bg-status-warning animate-pulse"
                    : tdrSupport?.supported
                      ? "bg-status-success"
                      : "bg-text-muted"
                )}
              />
              <span className="body-small font-medium text-text-primary">
                {checkingSupport
                  ? t("cableTest.checkingSupport", "Checking TDR support...")
                  : tdrSupport?.supported
                    ? t("cableTest.supported", "TDR Supported")
                    : t("cableTest.notSupported", "TDR Not Supported")}
              </span>
            </div>
            <button
              onClick={checkTDRSupport}
              disabled={checkingSupport}
              className="caption text-text-muted hover:text-text-primary"
            >
              {checkingSupport ? "..." : t("common.refresh", "Refresh")}
            </button>
          </div>
          {tdrSupport?.driver && (
            <p
              className={cn(
                "caption text-text-muted",
                spacing.margin.top.tight
              )}
            >
              {t("cableTest.driver", "Driver")}: {tdrSupport.driver}
            </p>
          )}
          {!tdrSupport?.supported && tdrSupport?.message && (
            <p
              className={cn(
                "caption text-text-muted",
                spacing.margin.top.tight
              )}
            >
              {tdrSupport.message}
            </p>
          )}
        </div>

        {/* Enable Cable Test Card */}
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
              {t("cableTest.enableCard", "Show Cable Test Card")}
            </span>
            <p className="caption text-text-muted">
              {t(
                "cableTest.enableCardDesc",
                "Display cable test card on dashboard"
              )}
            </p>
          </div>
          <input
            type="checkbox"
            checked={cableTestSettings.enabled}
            onChange={(e) =>
              setCableTestSettings((prev) => ({
                ...prev,
                enabled: e.target.checked,
              }))
            }
            className={iconTokens.size.sm}
          />
        </label>

        {/* Auto-Run on Link Down */}
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
              {t("cableTest.autoRunOnLinkDown", "Auto-Test on Link Down")}
            </span>
            <p className="caption text-text-muted">
              {t(
                "cableTest.autoRunOnLinkDownDesc",
                "Automatically run cable test when link goes down"
              )}
            </p>
          </div>
          <input
            type="checkbox"
            checked={cableTestSettings.autoRunOnLinkDown}
            onChange={(e) =>
              setCableTestSettings((prev) => ({
                ...prev,
                autoRunOnLinkDown: e.target.checked,
              }))
            }
            className={iconTokens.size.sm}
            disabled={!tdrSupport?.supported}
          />
        </label>

        {/* Length Unit Selection */}
        <div
          className={cn(
            "border-t border-surface-border",
            spacing.padding.top.heading
          )}
        >
          <label
            className="caption text-text-muted font-medium"
            htmlFor="cable-length-unit"
          >
            {t("cableTest.lengthUnit", "Length Unit")}
          </label>
          <div
            className={cn(
              "flex",
              spacing.gap.compact,
              spacing.margin.top.inline
            )}
          >
            {(["feet", "meters"] as const).map((unit) => {
              const checked = cableTestSettings.lengthUnit === unit;
              return (
                <label
                  key={unit}
                  className={cn(
                    "cursor-pointer flex-1 text-center",
                    spacing.chip.md,
                    radius.default,
                    "border body-small font-medium transition-colors",
                    checked
                      ? "bg-brand-primary text-text-inverse border-brand-primary"
                      : "bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover"
                  )}
                >
                  <input
                    type="radio"
                    name="cable-length-unit"
                    value={unit}
                    checked={checked}
                    onChange={() =>
                      setCableTestSettings((prev) => ({
                        ...prev,
                        lengthUnit: unit,
                      }))
                    }
                    className="sr-only"
                  />
                  {unit === "feet"
                    ? t("cableTest.feet", "Feet")
                    : t("cableTest.meters", "Meters")}
                </label>
              );
            })}
          </div>
        </div>

        <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
          {t(
            "cableTest.tdrNote",
            "TDR cable testing requires compatible network hardware and drivers"
          )}
        </p>
      </div>
    </CollapsibleSection>
  );
});
