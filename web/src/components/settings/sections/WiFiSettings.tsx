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

import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Wifi } from "../../ui/Icons";
import { icon as iconTokens, layout, radius } from "../../../styles/theme";
import {
  WiFiSettings as WiFiSettingsType,
  SaveStatus,
} from "../../../types/settings";

interface WiFiSettingsProps {
  wifiSettings: WiFiSettingsType;
  setWifiSettings: React.Dispatch<React.SetStateAction<WiFiSettingsType>>;
  wifiStatus: SaveStatus;
}

export function WiFiSettings({
  wifiSettings,
  setWifiSettings,
  wifiStatus,
}: WiFiSettingsProps) {
  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Wifi className={iconTokens.size.sm} />
          <span>WiFi</span>
          <AutoSaveIndicator status={wifiStatus} />
        </div>
      }
    >
      <div className="stack-sm">
        <div>
          <label className="caption text-text-muted" htmlFor="wifi-interface">
            WiFi Interface
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
              className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
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
              className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
            />
          )}
          <p className="caption text-text-muted mt-1">
            {wifiSettings.isWireless
              ? "Currently monitoring a wireless interface"
              : "No wireless interface detected"}
          </p>
        </div>
      </div>
    </CollapsibleSection>
  );
}
