import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Wifi } from "../../ui/Icons";
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
        <div className="flex items-center gap-2">
          <Wifi className="w-4 h-4" />
          <span>WiFi</span>
          <AutoSaveIndicator status={wifiStatus} />
        </div>
      }
    >
      <div className="space-y-3">
        <div>
          <label className="text-xs text-text-muted">WiFi Interface</label>
          {wifiSettings.availableWifi.length > 0 ? (
            <select
              value={wifiSettings.interface}
              onChange={(e) =>
                setWifiSettings((prev) => ({
                  ...prev,
                  interface: e.target.value,
                }))
              }
              className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
            >
              {wifiSettings.availableWifi.map((iface) => (
                <option key={iface} value={iface}>
                  {iface}
                </option>
              ))}
            </select>
          ) : (
            <input
              type="text"
              value={wifiSettings.interface}
              onChange={(e) =>
                setWifiSettings((prev) => ({
                  ...prev,
                  interface: e.target.value,
                }))
              }
              placeholder="wlan0 or en0"
              className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
            />
          )}
          <p className="text-xs text-text-muted mt-1">
            {wifiSettings.isWireless
              ? "Currently monitoring a wireless interface"
              : "No wireless interface detected"}
          </p>
        </div>
      </div>
    </CollapsibleSection>
  );
}
