/**
 * WiFiSettings Component
 *
 * Purpose: WiFi interface configuration and network connection management.
 *
 * Key Features:
 * - Interface selection: dropdown of available WiFi interfaces
 * - Network scanning: scan for available WiFi networks
 * - Network connection: connect to WiFi networks with password
 * - Connection status: show current connection state
 * - Saved networks: view and manage saved network profiles
 * - AutoSaveIndicator: shows persistent save status
 *
 * Usage:
 * ```typescript
 * <WiFiSettings
 *   wifiSettings={settings}
 *   setWifiSettings={updateSettings}
 *   wifiStatus={saveStatus}
 * />
 * ```
 */

import type React from "react";
import { memo, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../../lib/api";
import { cn, icon as iconTokens, layout, radius, spacing } from "../../../styles/theme";
import type { SaveStatus, WiFiSettings as WiFiSettingsType } from "../../../types/settings";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { Wifi } from "../../ui/Icons";
import { AutoSaveIndicator } from "./AutoSaveIndicator";

// Types for WiFi scanning and connection
interface ScannedNetwork {
  ssid: string;
  bssid: string;
  signal: number;
  channel: number;
  frequency: number;
  security: string;
  channelWidth?: number;
}

interface ConnectionResult {
  success: boolean;
  message: string;
  ssid?: string;
}

interface SavedNetwork {
  ssid: string;
  uuid?: string;
  type?: string;
  device?: string;
}

interface WiFiSettingsProps {
  wifiSettings: WiFiSettingsType;
  setWifiSettings: React.Dispatch<React.SetStateAction<WiFiSettingsType>>;
  wifiStatus: SaveStatus;
}

/**
 * Settings section for WiFi scanning configuration, adapter selection, and connection management.
 */
export const WiFiSettings: React.NamedExoticComponent<WiFiSettingsProps> = memo(
  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex WiFi management component
  function wiFiSettings({ wifiSettings, setWifiSettings, wifiStatus }: WiFiSettingsProps) {
    const { t } = useTranslation("settings");

    // State for network scanning and connection
    const [networks, setNetworks] = useState<ScannedNetwork[]>([]);
    const [scanning, setScanning] = useState(false);
    const [scanError, setScanError] = useState<string | null>(null);
    const [connecting, setConnecting] = useState(false);
    const [connectionStatus, setConnectionStatus] = useState<string | null>(null);
    const [selectedNetwork, setSelectedNetwork] = useState<ScannedNetwork | null>(null);
    const [password, setPassword] = useState("");
    const [savedNetworks, setSavedNetworks] = useState<SavedNetwork[]>([]);
    const [showPassword, setShowPassword] = useState(false);

    // Scan for available networks
    const scanNetworks = useCallback(async () => {
      setScanning(true);
      setScanError(null);
      try {
        const response = await api.get<{ networks: ScannedNetwork[]; error?: string }>(
          `/api/canopy/wifi/scan?interface=${wifiSettings.interface}`,
        );
        if (response?.networks) {
          // Filter out hidden networks (empty SSID) and sort by signal strength
          const visibleNetworks = response.networks
            .filter((n) => n.ssid && n.ssid.trim() !== "")
            .sort((a, b) => b.signal - a.signal);
          setNetworks(visibleNetworks);
        }
        if (response?.error) {
          setScanError(response.error);
        }
      } catch {
        setScanError("Failed to scan networks");
      } finally {
        setScanning(false);
      }
    }, [wifiSettings.interface]);

    // Load saved networks
    const loadSavedNetworks = useCallback(async () => {
      try {
        const response = await api.get<{ networks: SavedNetwork[] }>("/api/v1/canopy/wifi/saved");
        if (response?.networks) {
          setSavedNetworks(response.networks);
        }
      } catch {
        // Ignore errors for saved networks
      }
    }, []);

    // Connect to a network
    const connectToNetwork = useCallback(async () => {
      if (!selectedNetwork) {
        return;
      }

      setConnecting(true);
      setConnectionStatus(null);
      try {
        const response = await api.post<ConnectionResult>("/api/v1/canopy/wifi/connect", {
          ssid: selectedNetwork.ssid,
          password: password,
        });
        if (response?.success) {
          setConnectionStatus(`Connected to ${selectedNetwork.ssid}`);
          setSelectedNetwork(null);
          setPassword("");
          // Refresh saved networks
          await loadSavedNetworks();
        } else {
          setConnectionStatus(response?.message || "Connection failed");
        }
      } catch {
        setConnectionStatus("Connection failed");
      } finally {
        setConnecting(false);
      }
    }, [selectedNetwork, password, loadSavedNetworks]);

    // Disconnect from current network
    const disconnectNetwork = useCallback(async () => {
      setConnecting(true);
      try {
        const response = await api.post<ConnectionResult>("/api/v1/canopy/wifi/disconnect", {});
        if (response?.success) {
          setConnectionStatus("Disconnected");
        } else {
          setConnectionStatus(response?.message || "Disconnect failed");
        }
      } catch {
        setConnectionStatus("Disconnect failed");
      } finally {
        setConnecting(false);
      }
    }, []);

    // Forget a saved network
    const forgetNetwork = useCallback(
      async (ssid: string) => {
        try {
          await api.delete(`/api/canopy/wifi/forget?ssid=${encodeURIComponent(ssid)}`);
          await loadSavedNetworks();
        } catch {
          // Ignore errors
        }
      },
      [loadSavedNetworks],
    );

    // Auto-scan networks and load saved networks on mount, refresh every 30s
    useEffect(() => {
      if (wifiSettings.isWireless) {
        // Initial load
        loadSavedNetworks().catch(() => undefined);
        scanNetworks().catch(() => undefined);

        // Auto-refresh scan every 30 seconds
        const interval = setInterval(() => {
          scanNetworks().catch(() => undefined);
        }, 30000);

        return (): void => clearInterval(interval);
      }
    }, [wifiSettings.isWireless, loadSavedNetworks, scanNetworks]);

    // Get signal strength indicator
    const getSignalBars = (signal: number): string => {
      if (signal >= -50) {
        return "████";
      }
      if (signal >= -60) {
        return "███░";
      }
      if (signal >= -70) {
        return "██░░";
      }
      if (signal >= -80) {
        return "█░░░";
      }
      return "░░░░";
    };

    const getSignalColor = (signal: number): string => {
      if (signal >= -50) {
        return "text-status-success";
      }
      if (signal >= -60) {
        return "text-status-success";
      }
      if (signal >= -70) {
        return "text-status-warning";
      }
      return "text-status-error";
    };

    return (
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <Wifi class={iconTokens.size.sm} />
            <span>{t("sections.wifi")}</span>
            <AutoSaveIndicator status={wifiStatus} />
          </div>
        }
      >
        <div class="stack-md">
          {/* Interface Selection */}
          <div>
            <label class="caption text-text-muted" for="wifi-interface">
              {t("wifi.title")}
            </label>
            {wifiSettings.availableWifi.length > 0 ? (
              <select
                id="wifi-interface"
                value={wifiSettings.interface}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setWifiSettings((prev) => ({
                    ...prev,
                    interface: e.target.value,
                  }))
                }
                class={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  spacing.chip.lg,
                  "bg-surface-base border border-surface-border",
                  radius.default,
                  "body-small text-text-primary",
                )}
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
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setWifiSettings((prev) => ({
                    ...prev,
                    interface: e.target.value,
                  }))
                }
                placeholder="wlan0 or en0"
                class={cn(
                  "w-full",
                  spacing.margin.top.tight,
                  spacing.chip.lg,
                  "bg-surface-base border border-surface-border",
                  radius.default,
                  "body-small text-text-primary",
                )}
              />
            )}
            <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
              {wifiSettings.isWireless ? t("wifi.wirelessMonitoring") : t("wifi.noWireless")}
            </p>
          </div>

          {/* WiFi Network Connection - only show if wireless adapter available */}
          {wifiSettings.isWireless ? (
            <>
              {/* Available Networks */}
              <div class="border-t border-surface-border pt-3">
                <div class="flex items-center justify-between">
                  <span class="body-small font-medium text-text-primary">
                    Available Networks{" "}
                    {scanning ? <span class="text-text-muted">(scanning...)</span> : null}
                  </span>
                  <button
                    type="button"
                    onClick={scanNetworks}
                    disabled={scanning}
                    class={cn(
                      "caption font-medium",
                      spacing.chip.md,
                      radius.default,
                      "bg-surface-hover text-text-primary border border-surface-border",
                      "hover:bg-surface-border disabled:opacity-50",
                    )}
                  >
                    ↻ Refresh
                  </button>
                </div>

                {scanError ? <p class="caption text-status-error mt-1">{scanError}</p> : null}

                {/* Loading state when no networks yet */}
                {networks.length === 0 && scanning && (
                  <p class="caption text-text-muted mt-2">Scanning for networks...</p>
                )}

                {/* No networks found */}
                {networks.length === 0 && !scanning && !scanError && (
                  <p class="caption text-text-muted mt-2">No networks found</p>
                )}

                {/* Network List */}
                {networks.length > 0 && (
                  <div
                    class={cn(
                      "mt-2 max-h-48 overflow-y-auto",
                      "border border-surface-border",
                      radius.default,
                      "bg-surface-base",
                    )}
                  >
                    {networks.map((network) => (
                      <button
                        type="button"
                        key={network.bssid}
                        onClick={(): void => {
                          setSelectedNetwork(network);
                          setPassword("");
                          setConnectionStatus(null);
                        }}
                        class={cn(
                          "w-full text-left px-3 py-2",
                          "border-b border-surface-border last:border-b-0",
                          "hover:bg-surface-hover",
                          selectedNetwork?.bssid === network.bssid && "bg-brand-primary/10",
                        )}
                      >
                        <div class="flex items-center justify-between">
                          <div>
                            <span class="body-small text-text-primary">{network.ssid}</span>
                            <span class="caption text-text-muted ml-2">{network.security}</span>
                          </div>
                          <div class="flex items-center gap-2">
                            <span class="caption text-text-muted">Ch {network.channel}</span>
                            <span class={cn("font-mono caption", getSignalColor(network.signal))}>
                              {getSignalBars(network.signal)}
                            </span>
                          </div>
                        </div>
                      </button>
                    ))}
                  </div>
                )}

                {/* Connection Dialog */}
                {selectedNetwork ? (
                  <div
                    class={cn(
                      "mt-3 p-3",
                      "border border-surface-border",
                      radius.default,
                      "bg-surface-sunken",
                    )}
                  >
                    <div class="flex items-center justify-between mb-2">
                      <span class="body-small font-medium text-text-primary">
                        Connect to {selectedNetwork.ssid}
                      </span>
                      <button
                        type="button"
                        onClick={(): void => {
                          setSelectedNetwork(null);
                          setPassword("");
                        }}
                        class="caption text-text-muted hover:text-text-primary"
                      >
                        Cancel
                      </button>
                    </div>

                    {selectedNetwork.security !== "Open" ? (
                      <div class="relative mb-2">
                        <input
                          type={showPassword ? "text" : "password"}
                          value={password}
                          onChange={(
                            e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                          ): void => setPassword(e.target.value)}
                          placeholder="Password"
                          class={cn(
                            "w-full pr-16",
                            spacing.chip.lg,
                            "bg-surface-base border border-surface-border",
                            radius.default,
                            "body-small text-text-primary",
                          )}
                          onKeyDown={(e: React.KeyboardEvent): void => {
                            if (e.key === "Enter" && password) {
                              connectToNetwork().catch(() => undefined);
                            }
                          }}
                        />
                        <button
                          type="button"
                          onClick={(): void => setShowPassword(!showPassword)}
                          class="absolute right-2 top-1/2 -translate-y-1/2 caption text-text-muted hover:text-text-primary"
                        >
                          {showPassword ? "Hide" : "Show"}
                        </button>
                      </div>
                    ) : null}

                    <button
                      type="button"
                      onClick={connectToNetwork}
                      disabled={connecting || (selectedNetwork.security !== "Open" && !password)}
                      class={cn(
                        "w-full",
                        "body-small font-medium",
                        spacing.chip.lg,
                        radius.default,
                        "bg-brand-primary text-text-inverse",
                        "hover:bg-brand-accent disabled:opacity-50",
                      )}
                    >
                      {connecting ? "Connecting..." : "Connect"}
                    </button>
                  </div>
                ) : null}

                {/* Connection Status */}
                {connectionStatus ? (
                  <p
                    class={cn(
                      "caption mt-2",
                      connectionStatus.includes("Connected")
                        ? "text-status-success"
                        : "text-status-error",
                    )}
                  >
                    {connectionStatus}
                  </p>
                ) : null}
              </div>

              {/* Current Connection / Disconnect */}
              <div class="border-t border-surface-border pt-3">
                <div class="flex items-center justify-between">
                  <span class="body-small font-medium text-text-primary">Connection</span>
                  <button
                    type="button"
                    onClick={disconnectNetwork}
                    disabled={connecting}
                    class={cn(
                      "caption font-medium",
                      spacing.chip.md,
                      radius.default,
                      "bg-status-error/10 text-status-error border border-status-error/20",
                      "hover:bg-status-error/20 disabled:opacity-50",
                    )}
                  >
                    Disconnect
                  </button>
                </div>
              </div>

              {/* Saved Networks */}
              {savedNetworks.length > 0 ? (
                <div class="border-t border-surface-border pt-3">
                  <span class="body-small font-medium text-text-primary block mb-2">
                    Saved Networks
                  </span>
                  <div
                    class={cn(
                      "max-h-32 overflow-y-auto",
                      "border border-surface-border",
                      radius.default,
                      "bg-surface-base",
                    )}
                  >
                    {savedNetworks.map((network) => (
                      <div
                        key={network.uuid || network.ssid}
                        class={cn(
                          "flex items-center justify-between px-3 py-2",
                          "border-b border-surface-border last:border-b-0",
                        )}
                      >
                        <span class="body-small text-text-primary">{network.ssid}</span>
                        <button
                          type="button"
                          onClick={(): void => forgetNetwork(network.ssid)}
                          class="caption text-status-error hover:underline"
                        >
                          Forget
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
            </>
          ) : null}
        </div>
      </CollapsibleSection>
    );
  },
);
