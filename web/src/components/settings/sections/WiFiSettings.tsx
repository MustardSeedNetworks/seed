/**
 * WiFiSettings Component
 *
 * Purpose: WiFi interface configuration allowing users to select the active WiFi
 * interface for network discovery and testing.
 *
 * Key Features:
 * - Interface selection: dropdown of available WiFi interfaces
 * - Dynamic list: shows only interfaces available on the system
 * - Enable/disable: toggle WiFi scanning on/off
 * - Survey interval: frequency of WiFi signal surveys
 * - Channel configuration: select specific channels or auto-detect
 * - Signal threshold: minimum RSSI for device detection
 * - AutoSaveIndicator: shows persistent save status
 * - WiFi icon: visual indicator in settings menu
 * - Fallback message: shows message when no WiFi interfaces available
 *
 * Usage:
 * ```typescript
 * <WiFiSettings
 *   wifiSettings={settings}
 *   setWifiSettings={updateSettings}
 *   wifiStatus={saveStatus}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, Wifi icon, settings types
 * State: Manages active interface selection and WiFi-specific configurations
 */

import { memo } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Wifi } from "../../ui/Icons";
import {
  icon as iconTokens,
  layout,
  radius,
  spacing,
} from "../../../styles/theme";
import {
  WiFiSettings as WiFiSettingsType,
  SaveStatus,
} from "../../../types/settings";

interface WiFiSettingsProps {
  wifiSettings: WiFiSettingsType;
  setWifiSettings: React.Dispatch<React.SetStateAction<WiFiSettingsType>>;
  wifiStatus: SaveStatus;
}

/**
 * Settings section for WiFi scanning configuration and adapter selection.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const WiFiSettings = memo(function WiFiSettings({
  wifiSettings,
  setWifiSettings,
  wifiStatus,
}: WiFiSettingsProps) {
  const { t } = useTranslation("settings");

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Wifi className={iconTokens.size.sm} />
          <span>{t("sections.wifi")}</span>
          <AutoSaveIndicator status={wifiStatus} />
        </div>
      }
    >
      <div className="stack-sm">
        <div>
          <label className="caption text-text-muted" htmlFor="wifi-interface">
            {t("wifi.title")}
          </label>
          {wifiSettings.availableWifi.length > 0 ? (
            <select
              id="wifi-interface"
              value={wifiSettings.interface}
              onChange={(e) =>
                setWifiSettings((prev) => ({
                  ...prev,
                  interface: e.target.value,
                }))
              }
              className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
            >
              {wifiSettings.availableWifi.map((iface) => (
                <option key={iface} value={iface}>
                  {iface}
                </option>
              ))}
            </select>
          ) : (
            <input
              id="wifi-interface-input"
              type="text"
              value={wifiSettings.interface}
              onChange={(e) =>
                setWifiSettings((prev) => ({
                  ...prev,
                  interface: e.target.value,
                }))
              }
              placeholder="wlan0 or en0"
              className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
            />
          )}
          <p className={`caption text-text-muted ${spacing.margin.top.tight}`}>
            {wifiSettings.isWireless
              ? t("wifi.wirelessMonitoring")
              : t("wifi.noWireless")}
          </p>
        </div>
      </div>
    </CollapsibleSection>
  );
});
