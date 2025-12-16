import { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { ScanSearch } from "../../ui/Icons";
import { icon as iconTokens, layout, radius, spacing } from "../../../styles/theme";
import {
  NetworkDiscoverySettings as NetworkDiscoverySettingsType,
  SubnetConfig,
  SaveStatus,
  DiscoveryProfile,
  DiscoveryServiceStatus,
} from "../../../types/settings";

interface DiscoverySettingsProps {
  networkDiscoverySettings: NetworkDiscoverySettingsType;
  setNetworkDiscoverySettings: React.Dispatch<React.SetStateAction<NetworkDiscoverySettingsType>>;
  networkDiscoveryStatus: SaveStatus;
  subnets: SubnetConfig[];
  subnetsStatus: SaveStatus;
  newSubnetCidr: string;
  setNewSubnetCidr: React.Dispatch<React.SetStateAction<string>>;
  newSubnetName: string;
  setNewSubnetName: React.Dispatch<React.SetStateAction<string>>;
  subnetError: string | null;
  setSubnetError: React.Dispatch<React.SetStateAction<string | null>>;
  addSubnet: () => void;
  toggleSubnet: (cidr: string, enabled: boolean) => void;
  deleteSubnet: (cidr: string) => void;
}

// Profile values for iteration
const PROFILE_VALUES: DiscoveryProfile[] = ["stealth", "standard", "full_scan", "custom"];

/**
 * Settings section for network discovery profiles and subnet management.
 */
export function DiscoverySettings({
  networkDiscoverySettings,
  setNetworkDiscoverySettings,
  networkDiscoveryStatus,
  subnets,
  subnetsStatus,
  newSubnetCidr,
  setNewSubnetCidr,
  newSubnetName,
  setNewSubnetName,
  subnetError,
  setSubnetError,
  addSubnet,
  toggleSubnet,
  deleteSubnet,
}: DiscoverySettingsProps) {
  const { t } = useTranslation("settings");
  const [serviceStatus, setServiceStatus] = useState<DiscoveryServiceStatus | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);

  // Get translated profile label
  const getProfileLabel = (profile: DiscoveryProfile) => {
    switch (profile) {
      case "stealth":
        return t("discovery.profileStealth");
      case "standard":
        return t("discovery.profileStandard");
      case "full_scan":
        return t("discovery.profileFullScan");
      case "custom":
        return t("discovery.profileCustom");
      default:
        return profile;
    }
  };

  // Get translated profile description
  const getProfileDescription = (profile: DiscoveryProfile) => {
    switch (profile) {
      case "stealth":
        return t("discovery.profileStealthDesc");
      case "standard":
        return t("discovery.profileStandardDesc");
      case "full_scan":
        return t("discovery.profileFullScanDesc");
      case "custom":
        return t("discovery.profileCustomDesc");
      default:
        return "";
    }
  };

  // Fetch service status
  const fetchServiceStatus = useCallback(async () => {
    setStatusLoading(true);
    try {
      const response = await fetch("/api/discovery/service/status");
      if (response.ok) {
        const data = await response.json();
        setServiceStatus(data);
      }
    } catch {
      // Silently fail - status display is informational
    } finally {
      setStatusLoading(false);
    }
  }, []);

  // Fetch status on mount and periodically
  useEffect(() => {
    fetchServiceStatus();
    const interval = setInterval(fetchServiceStatus, 10000);
    return () => clearInterval(interval);
  }, [fetchServiceStatus]);

  // Update profile via API
  const handleProfileChange = async (profile: DiscoveryProfile) => {
    setNetworkDiscoverySettings((prev) => ({
      ...prev,
      profile,
    }));

    try {
      await fetch("/api/discovery/profile", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ profile }),
      });
      // Refresh status after profile change
      setTimeout(fetchServiceStatus, 500);
    } catch {
      // Settings auto-save will handle persistence
    }
  };

  const currentProfile = networkDiscoverySettings.profile || "standard";
  const showCustomOptions = currentProfile === "custom";
  const showSubnets = currentProfile === "full_scan" || currentProfile === "custom";

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <ScanSearch className={iconTokens.size.sm} />
          <span>{t("sections.discovery")}</span>
          <AutoSaveIndicator status={networkDiscoveryStatus} />
        </div>
      }
    >
      <div className="stack">
        {/* Enable Toggle */}
        <label
          className={`${layout.flex.between} ${spacing.pad.xs} bg-surface-base ${radius.default} border border-surface-border`}
        >
          <div>
            <span className="body-small text-text-primary font-medium">
              {t("discovery.enableDiscovery")}
            </span>
            <p className="caption text-text-muted">{t("discovery.scanForDevices")}</p>
          </div>
          <input
            type="checkbox"
            checked={networkDiscoverySettings.enabled}
            onChange={(e) =>
              setNetworkDiscoverySettings((prev) => ({
                ...prev,
                enabled: e.target.checked,
              }))
            }
            className={iconTokens.size.sm}
          />
        </label>

        {/* Auto-Scan on Link Up */}
        <label
          className={`${layout.flex.between} ${spacing.pad.xs} bg-surface-base ${radius.default} border border-surface-border`}
        >
          <div>
            <span className="body-small text-text-primary font-medium">
              {t("discovery.autoScanOnLink")}
            </span>
            <p className="caption text-text-muted">{t("discovery.autoScanDesc")}</p>
          </div>
          <input
            type="checkbox"
            checked={networkDiscoverySettings.autoScan}
            onChange={(e) =>
              setNetworkDiscoverySettings((prev) => ({
                ...prev,
                autoScan: e.target.checked,
              }))
            }
            className={iconTokens.size.sm}
          />
        </label>

        {/* Service Status Banner */}
        {serviceStatus && (
          <div
            className={`${spacing.pad.sm} ${radius.lg} border ${
              serviceStatus.running
                ? "bg-status-success/10 border-status-success/30"
                : "bg-status-error/10 border-status-error/30"
            }`}
          >
            <div className={layout.flex.between}>
              <div className={layout.inline.default}>
                <div
                  className={`w-2 h-2 ${radius.full} ${
                    serviceStatus.running
                      ? serviceStatus.scanning
                        ? "bg-status-warning animate-pulse"
                        : "bg-status-success"
                      : "bg-status-error"
                  }`}
                />
                <span className="body-small font-medium text-text-primary">
                  {serviceStatus.running
                    ? serviceStatus.scanning
                      ? t("discovery.serviceStatus.scanning")
                      : t("discovery.serviceStatus.running")
                    : t("discovery.serviceStatus.stopped")}
                </span>
              </div>
              <button
                onClick={fetchServiceStatus}
                disabled={statusLoading}
                className="caption text-text-muted hover:text-text-primary"
              >
                {statusLoading ? "..." : t("discovery.serviceStatus.refresh")}
              </button>
            </div>
            {serviceStatus.running && (
              <div
                className={`${spacing.margin.top.inline} grid grid-cols-2 ${spacing.gap.compact} caption text-text-muted`}
              >
                <div>
                  <span className="font-medium">{t("discovery.serviceStatus.devices")}:</span>{" "}
                  {serviceStatus.deviceCount}
                </div>
                <div>
                  <span className="font-medium">{t("discovery.serviceStatus.interface")}:</span>{" "}
                  {serviceStatus.interface || "auto"}
                </div>
                <div>
                  <span className="font-medium">{t("discovery.serviceStatus.subnet")}:</span>{" "}
                  {serviceStatus.subnet || "..."}
                </div>
                <div>
                  <span className="font-medium">{t("discovery.serviceStatus.localIP")}:</span>{" "}
                  {serviceStatus.localIP || "..."}
                </div>
              </div>
            )}
            {serviceStatus.activeMethods && serviceStatus.activeMethods.length > 0 && (
              <div className={`${spacing.margin.top.inline} flex flex-wrap ${spacing.gap.tight}`}>
                {serviceStatus.activeMethods.map((method) => (
                  <span
                    key={method}
                    className={`${spacing.chip.sm} bg-surface-base ${radius.default} caption text-text-muted`}
                  >
                    {method}
                  </span>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Discovery Profile Selector */}
        <div>
          <label className="caption text-text-muted font-medium">{t("discovery.profile")}</label>
          <div className={`${spacing.margin.top.inline} stack-sm`}>
            {PROFILE_VALUES.map((profile) => (
              <label
                key={profile}
                className={`${layout.inline.default} items-start ${spacing.pad.sm} ${radius.lg} border cursor-pointer transition-colors ${
                  currentProfile === profile
                    ? "border-brand-primary bg-brand-primary/5"
                    : "border-surface-border hover:border-brand-primary/50"
                }`}
              >
                <input
                  type="radio"
                  name="discovery-profile"
                  value={profile}
                  checked={currentProfile === profile}
                  onChange={() => handleProfileChange(profile)}
                  className={spacing.margin.top.tight}
                />
                <div className="flex-1">
                  <div className="body-small font-medium text-text-primary">
                    {getProfileLabel(profile)}
                  </div>
                  <div className={`caption text-text-muted ${spacing.margin.top.tight}`}>
                    {getProfileDescription(profile)}
                  </div>
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* Custom Options (only shown when Custom profile is selected) */}
        {showCustomOptions && (
          <div className={`border-t border-surface-border ${spacing.pad.sm}`}>
            <span className="caption text-text-muted font-medium">
              {t("discovery.customOptions")}
            </span>
            <div className={`${spacing.margin.top.inline} stack-sm`}>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={networkDiscoverySettings.customOptions?.passiveListen ?? true}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        passiveListen: e.target.checked,
                      },
                    }))
                  }
                  className={iconTokens.size.sm}
                />
                <span className="body-small text-text-primary">
                  {t("discovery.passiveListeners")}
                </span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={networkDiscoverySettings.customOptions?.arpScan ?? true}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        arpScan: e.target.checked,
                      },
                    }))
                  }
                  className={iconTokens.size.sm}
                />
                <span className="body-small text-text-primary">{t("discovery.arpScanning")}</span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={networkDiscoverySettings.customOptions?.icmpScan ?? true}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        icmpScan: e.target.checked,
                      },
                    }))
                  }
                  className={iconTokens.size.sm}
                />
                <span className="body-small text-text-primary">{t("discovery.icmpPingSweep")}</span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={networkDiscoverySettings.customOptions?.portScan?.enabled ?? false}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        portScan: {
                          ...prev.customOptions?.portScan,
                          enabled: e.target.checked,
                          ports: prev.customOptions?.portScan?.ports ?? [],
                          topPorts: prev.customOptions?.portScan?.topPorts ?? 100,
                        },
                      },
                    }))
                  }
                  className={iconTokens.size.sm}
                />
                <span className="body-small text-text-primary">{t("discovery.portScanning")}</span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={networkDiscoverySettings.customOptions?.traceroute ?? false}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        traceroute: e.target.checked,
                      },
                    }))
                  }
                  className={iconTokens.size.sm}
                />
                <span className="body-small text-text-primary">{t("discovery.traceroute")}</span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={networkDiscoverySettings.customOptions?.snmpQuery ?? false}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        snmpQuery: e.target.checked,
                      },
                    }))
                  }
                  className={iconTokens.size.sm}
                />
                <span className="body-small text-text-primary">{t("discovery.snmpQueries")}</span>
              </label>
            </div>
          </div>
        )}

        {/* Timing Settings */}
        <div className={`border-t border-surface-border ${spacing.pad.sm}`}>
          <span className="caption text-text-muted font-medium">
            {t("discovery.timingSettings")}
          </span>

          {/* Scan Workers */}
          <div className={spacing.margin.top.inline}>
            <label className="caption text-text-muted" htmlFor="discovery-workers">
              {t("discovery.concurrentWorkers")}
            </label>
            <input
              id="discovery-workers"
              type="number"
              value={networkDiscoverySettings.arpScanWorkers}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  arpScanWorkers: parseInt(e.target.value) || 50,
                }))
              }
              min={1}
              max={100}
              className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
            />
            <p className={`caption text-text-muted ${spacing.margin.top.tight}`}>
              {t("discovery.workersDesc")}
            </p>
          </div>

          {/* Ping Timeout */}
          <div className={spacing.margin.top.content}>
            <label className="caption text-text-muted" htmlFor="discovery-ping-timeout">
              {t("discovery.pingTimeout")}
            </label>
            <input
              id="discovery-ping-timeout"
              type="number"
              value={networkDiscoverySettings.pingTimeoutMs}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  pingTimeoutMs: parseInt(e.target.value) || 500,
                }))
              }
              min={100}
              max={5000}
              className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
            />
          </div>

          {/* Scan Timeout */}
          <div className={spacing.margin.top.content}>
            <label className="caption text-text-muted" htmlFor="discovery-scan-timeout">
              {t("discovery.scanTimeout")}
            </label>
            <input
              id="discovery-scan-timeout"
              type="number"
              value={networkDiscoverySettings.scanTimeoutMs}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  scanTimeoutMs: parseInt(e.target.value) || 30000,
                }))
              }
              min={5000}
              max={120000}
              className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
            />
          </div>

          {/* Rescan Interval */}
          <div className={spacing.margin.top.content}>
            <label className="caption text-text-muted" htmlFor="discovery-rescan-interval">
              {t("discovery.rescanInterval")}
            </label>
            <input
              id="discovery-rescan-interval"
              type="number"
              value={networkDiscoverySettings.scanIntervalMs}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  scanIntervalMs: parseInt(e.target.value) || 0,
                }))
              }
              min={0}
              className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
            />
            <p className={`caption text-text-muted ${spacing.margin.top.tight}`}>
              {t("discovery.rescanIntervalDesc")}
            </p>
          </div>
        </div>

        {/* OUI File Path */}
        <div className={`border-t border-surface-border ${spacing.pad.sm}`}>
          <label className="caption text-text-muted font-medium" htmlFor="discovery-oui-path">
            {t("discovery.ouiFilePath")}
          </label>
          <input
            id="discovery-oui-path"
            type="text"
            value={networkDiscoverySettings.ouiFilePath}
            onChange={(e) =>
              setNetworkDiscoverySettings((prev) => ({
                ...prev,
                ouiFilePath: e.target.value,
              }))
            }
            placeholder="oui.txt"
            className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
          />
          <p className={`caption text-text-muted ${spacing.margin.top.tight}`}>
            {t("discovery.ouiFileDesc")}
          </p>
        </div>

        {/* Target Networks (only for full_scan or custom profile) */}
        {showSubnets && (
          <div className={`border-t border-surface-border ${spacing.pad.sm}`}>
            <div className={`${layout.flex.between} ${spacing.margin.bottom.inline}`}>
              <span className="caption text-text-muted font-medium">
                {t("discovery.targetNetworks")} <AutoSaveIndicator status={subnetsStatus} />
              </span>
            </div>
            <p className={`caption text-text-muted ${spacing.margin.bottom.inline}`}>
              {t("discovery.targetNetworksDesc")}
            </p>

            {/* List of configured subnets */}
            {subnets.length > 0 && (
              <div className={`stack-sm ${spacing.margin.bottom.heading}`}>
                {subnets.map((subnet) => (
                  <div
                    key={subnet.cidr}
                    className={`${layout.flex.between} ${spacing.pad.xs} bg-surface-base ${radius.default} border border-surface-border`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="body-small text-text-primary truncate">
                        {subnet.name || subnet.cidr}
                      </div>
                      <div className="caption text-text-muted">{subnet.cidr}</div>
                    </div>
                    <div className={`${layout.inline.default} ${spacing.margin.left.inline}`}>
                      <input
                        type="checkbox"
                        checked={subnet.enabled}
                        onChange={(e) => toggleSubnet(subnet.cidr, e.target.checked)}
                        className={iconTokens.size.sm}
                        title={
                          subnet.enabled
                            ? t("discovery.disableSubnet")
                            : t("discovery.enableSubnet")
                        }
                      />
                      <button
                        onClick={() => deleteSubnet(subnet.cidr)}
                        className="text-status-error hover:text-status-error/70 body-small"
                        title={t("discovery.removeSubnet")}
                      >
                        X
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Add new subnet form */}
            <div className="stack-sm">
              <input
                type="text"
                value={newSubnetCidr}
                onChange={(e) => {
                  setNewSubnetCidr(e.target.value);
                  setSubnetError(null);
                }}
                placeholder={t("discovery.cidrPlaceholder")}
                className={`w-full ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
              />
              <input
                type="text"
                value={newSubnetName}
                onChange={(e) => setNewSubnetName(e.target.value)}
                placeholder={t("discovery.namePlaceholder")}
                className={`w-full ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
              />
              {subnetError && <p className="caption text-status-error">{subnetError}</p>}
              <button
                onClick={addSubnet}
                className={`w-full ${spacing.pad.sm} bg-brand-primary hover:bg-brand-accent text-text-inverse ${radius.default} body-small`}
              >
                {t("discovery.addSubnet")}
              </button>
            </div>
          </div>
        )}
      </div>
    </CollapsibleSection>
  );
}
