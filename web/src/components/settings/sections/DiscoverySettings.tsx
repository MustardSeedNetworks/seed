import { useState, useEffect, useCallback, memo } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { ScanSearch } from "../../ui/Icons";
import {
  icon as iconTokens,
  layout,
  radius,
  spacing,
  input as inputTokens,
} from "../../../styles/theme";
import {
  NetworkDiscoverySettings as NetworkDiscoverySettingsType,
  SubnetConfig,
  SaveStatus,
  DiscoveryProfile,
  DiscoveryServiceStatus,
  SNMPSettings as SNMPSettingsType,
  SNMPv3Credential,
} from "../../../types/settings";
import { generateId } from "../../../utils/id";

interface DiscoverySettingsProps {
  networkDiscoverySettings: NetworkDiscoverySettingsType;
  setNetworkDiscoverySettings: React.Dispatch<
    React.SetStateAction<NetworkDiscoverySettingsType>
  >;
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
  // SNMP settings (now integrated under Discovery)
  snmpSettings: SNMPSettingsType;
  setSnmpSettings: React.Dispatch<React.SetStateAction<SNMPSettingsType>>;
  snmpStatus: SaveStatus;
}

// Profile values for iteration
const PROFILE_VALUES: DiscoveryProfile[] = [
  "stealth",
  "standard",
  "full_scan",
  "custom",
];

/**
 * Settings section for network discovery profiles and subnet management.
 */
// SNMP protocol values - labels are translated in the component
const AUTH_PROTOCOL_VALUES = [
  "",
  "MD5",
  "SHA",
  "SHA224",
  "SHA256",
  "SHA384",
  "SHA512",
];
const PRIV_PROTOCOL_VALUES = [
  "",
  "DES",
  "AES",
  "AES192",
  "AES256",
  "AES192C",
  "AES256C",
];
const SECURITY_LEVEL_VALUES = [
  "noAuthNoPriv",
  "authNoPriv",
  "authPriv",
] as const;

/**
 * Settings section for network discovery profiles and subnet management.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const DiscoverySettings = memo(function DiscoverySettings({
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
  snmpSettings,
  setSnmpSettings,
  snmpStatus,
}: DiscoverySettingsProps) {
  const { t } = useTranslation("settings");
  const [serviceStatus, setServiceStatus] =
    useState<DiscoveryServiceStatus | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [newCommunity, setNewCommunity] = useState("");
  const [expandedCredential, setExpandedCredential] = useState<string | null>(
    null
  );

  // Get translated label for auth protocol
  const getAuthProtocolLabel = (value: string) => {
    if (value === "") return t("snmp.noAuth");
    return value;
  };

  // Get translated label for privacy protocol
  const getPrivProtocolLabel = (value: string) => {
    if (value === "") return t("snmp.noPrivacy");
    return value;
  };

  // Get translated label for security level
  const getSecurityLevelLabel = (value: string) => {
    switch (value) {
      case "noAuthNoPriv":
        return t("snmp.noAuthNoPriv");
      case "authNoPriv":
        return t("snmp.authNoPriv");
      case "authPriv":
        return t("snmp.authPriv");
      default:
        return value;
    }
  };

  const addCommunity = useCallback(() => {
    if (newCommunity.trim() === "") return;
    if (snmpSettings.communities.includes(newCommunity.trim())) {
      setNewCommunity("");
      return;
    }
    setSnmpSettings((prev) => ({
      ...prev,
      communities: [...prev.communities, newCommunity.trim()],
    }));
    setNewCommunity("");
  }, [newCommunity, setSnmpSettings, snmpSettings.communities]);

  const removeCommunity = useCallback(
    (community: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        communities: prev.communities.filter((c) => c !== community),
      }));
    },
    [setSnmpSettings]
  );

  const addV3Credential = useCallback(() => {
    const newCred: SNMPv3Credential = {
      id: generateId(),
      name: t("snmp.newCredential"),
      username: "",
      authProtocol: "",
      authPassword: "",
      privProtocol: "",
      privPassword: "",
      contextName: "",
      securityLevel: "noAuthNoPriv",
    };
    setSnmpSettings((prev) => ({
      ...prev,
      v3Credentials: [...prev.v3Credentials, newCred],
    }));
    setExpandedCredential(newCred.id!);
  }, [setSnmpSettings, t]);

  const removeV3Credential = useCallback(
    (id: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        v3Credentials: prev.v3Credentials.filter((c) => c.id !== id),
      }));
      if (expandedCredential === id) {
        setExpandedCredential(null);
      }
    },
    [setSnmpSettings, expandedCredential]
  );

  const updateV3Credential = useCallback(
    (id: string, field: keyof SNMPv3Credential, value: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        v3Credentials: prev.v3Credentials.map((c) =>
          c.id === id ? { ...c, [field]: value } : c
        ),
      }));
    },
    [setSnmpSettings]
  );

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
  const showSubnets =
    currentProfile === "full_scan" || currentProfile === "custom";

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
            <p className="caption text-text-muted">
              {t("discovery.scanForDevices")}
            </p>
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
            <p className="caption text-text-muted">
              {t("discovery.autoScanDesc")}
            </p>
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
                  <span className="font-medium">
                    {t("discovery.serviceStatus.devices")}:
                  </span>{" "}
                  {serviceStatus.deviceCount}
                </div>
                <div>
                  <span className="font-medium">
                    {t("discovery.serviceStatus.interface")}:
                  </span>{" "}
                  {serviceStatus.interface || "auto"}
                </div>
                <div>
                  <span className="font-medium">
                    {t("discovery.serviceStatus.subnet")}:
                  </span>{" "}
                  {serviceStatus.subnet || "..."}
                </div>
                <div>
                  <span className="font-medium">
                    {t("discovery.serviceStatus.localIP")}:
                  </span>{" "}
                  {serviceStatus.localIP || "..."}
                </div>
              </div>
            )}
            {serviceStatus.activeMethods &&
              serviceStatus.activeMethods.length > 0 && (
                <div
                  className={`${spacing.margin.top.inline} flex flex-wrap ${spacing.gap.tight}`}
                >
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
          <label className="caption text-text-muted font-medium">
            {t("discovery.profile")}
          </label>
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
                  <div
                    className={`caption text-text-muted ${spacing.margin.top.tight}`}
                  >
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
                  checked={
                    networkDiscoverySettings.customOptions?.passiveListen ??
                    true
                  }
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
                  checked={
                    networkDiscoverySettings.customOptions?.arpScan ?? true
                  }
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
                <span className="body-small text-text-primary">
                  {t("discovery.arpScanning")}
                </span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.icmpScan ?? true
                  }
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
                <span className="body-small text-text-primary">
                  {t("discovery.icmpPingSweep")}
                </span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.portScan?.enabled ??
                    false
                  }
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        portScan: {
                          ...prev.customOptions?.portScan,
                          enabled: e.target.checked,
                          ports: prev.customOptions?.portScan?.ports ?? [],
                          topPorts:
                            prev.customOptions?.portScan?.topPorts ?? 100,
                        },
                      },
                    }))
                  }
                  className={iconTokens.size.sm}
                />
                <span className="body-small text-text-primary">
                  {t("discovery.portScanning")}
                </span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.traceroute ?? false
                  }
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
                <span className="body-small text-text-primary">
                  {t("discovery.traceroute")}
                </span>
              </label>
              <label className={layout.inline.default}>
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.snmpQuery ?? false
                  }
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
                <span className="body-small text-text-primary">
                  {t("discovery.snmpQueries")}
                </span>
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
            <label
              className="caption text-text-muted"
              htmlFor="discovery-workers"
            >
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
            <p
              className={`caption text-text-muted ${spacing.margin.top.tight}`}
            >
              {t("discovery.workersDesc")}
            </p>
          </div>

          {/* Ping Timeout */}
          <div className={spacing.margin.top.content}>
            <label
              className="caption text-text-muted"
              htmlFor="discovery-ping-timeout"
            >
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
            <label
              className="caption text-text-muted"
              htmlFor="discovery-scan-timeout"
            >
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
            <label
              className="caption text-text-muted"
              htmlFor="discovery-rescan-interval"
            >
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
            <p
              className={`caption text-text-muted ${spacing.margin.top.tight}`}
            >
              {t("discovery.rescanIntervalDesc")}
            </p>
          </div>
        </div>

        {/* OUI File Path */}
        <div className={`border-t border-surface-border ${spacing.pad.sm}`}>
          <label
            className="caption text-text-muted font-medium"
            htmlFor="discovery-oui-path"
          >
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
            <div
              className={`${layout.flex.between} ${spacing.margin.bottom.inline}`}
            >
              <span className="caption text-text-muted font-medium">
                {t("discovery.targetNetworks")}{" "}
                <AutoSaveIndicator status={subnetsStatus} />
              </span>
            </div>
            <p
              className={`caption text-text-muted ${spacing.margin.bottom.inline}`}
            >
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
                      <div className="caption text-text-muted">
                        {subnet.cidr}
                      </div>
                    </div>
                    <div
                      className={`${layout.inline.default} ${spacing.margin.left.inline}`}
                    >
                      <input
                        type="checkbox"
                        checked={subnet.enabled}
                        onChange={(e) =>
                          toggleSubnet(subnet.cidr, e.target.checked)
                        }
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
              {subnetError && (
                <p className="caption text-status-error">{subnetError}</p>
              )}
              <button
                onClick={addSubnet}
                className={`w-full ${spacing.pad.sm} bg-brand-primary hover:bg-brand-accent text-text-inverse ${radius.default} body-small`}
              >
                {t("discovery.addSubnet")}
              </button>
            </div>
          </div>
        )}

        {/* SNMP Settings Section */}
        <div className={`border-t border-surface-border ${spacing.pad.sm}`}>
          <div
            className={`${layout.flex.between} ${spacing.margin.bottom.inline}`}
          >
            <span className="body-small text-text-primary font-medium">
              {t("sections.snmp")} <AutoSaveIndicator status={snmpStatus} />
            </span>
          </div>
          <p
            className={`caption text-text-muted ${spacing.margin.bottom.inline}`}
          >
            {t(
              "snmp.description",
              "Configure SNMP credentials for enhanced device discovery"
            )}
          </p>

          {/* SNMP Port */}
          <div className={spacing.margin.bottom.inline}>
            <label className="caption text-text-muted" htmlFor="snmp-port">
              {t("snmp.port")}
            </label>
            <input
              id="snmp-port"
              type="number"
              value={snmpSettings.port}
              onChange={(e) =>
                setSnmpSettings((prev) => ({
                  ...prev,
                  port: parseInt(e.target.value, 10) || 161,
                }))
              }
              min="1"
              max="65535"
              className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} ${spacing.margin.top.tight} body-small`}
            />
          </div>

          {/* Timeout */}
          <div className={spacing.margin.bottom.inline}>
            <label className="caption text-text-muted" htmlFor="snmp-timeout">
              {t("snmp.timeout")}
            </label>
            <input
              id="snmp-timeout"
              type="number"
              value={snmpSettings.timeout / 1000}
              onChange={(e) =>
                setSnmpSettings((prev) => ({
                  ...prev,
                  timeout: (parseFloat(e.target.value) || 5) * 1000,
                }))
              }
              min="1"
              max="30"
              step="1"
              className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} ${spacing.margin.top.tight} body-small`}
            />
          </div>

          {/* Community Strings (v1/v2c) */}
          <div
            className={`border-t border-surface-border ${spacing.padding.top.heading} ${spacing.margin.top.inline}`}
          >
            <div
              className={`flex items-center justify-between ${spacing.margin.bottom.inline}`}
            >
              <span className="caption text-text-muted font-medium">
                {t("snmp.communityStrings")}
              </span>
              <button
                onClick={() => addCommunity()}
                className="caption text-brand-primary hover:text-brand-accent"
                aria-label="Add community string"
              >
                {t("common.add")}
              </button>
            </div>
            <div
              className={`flex ${spacing.gap.compact} ${spacing.margin.bottom.inline}`}
            >
              <label className="sr-only" htmlFor="snmp-community-new">
                {t("snmp.communityString")}
              </label>
              <input
                id="snmp-community-new"
                type="text"
                value={newCommunity}
                onChange={(e) => setNewCommunity(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") addCommunity();
                }}
                placeholder={t("snmp.communityString")}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} flex-1 caption`}
              />
            </div>
            {snmpSettings.communities.map((community, index) => (
              <div
                key={`${community}-${index}`}
                className={`flex ${spacing.gap.compact} ${spacing.margin.bottom.inline}`}
              >
                <input
                  aria-label={`Community string ${community}`}
                  type="text"
                  value={community}
                  readOnly
                  className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} flex-1 bg-surface-hover caption`}
                />
                <button
                  onClick={() => removeCommunity(community)}
                  className={`text-status-error hover:text-status-error/80 ${spacing.actionBtn}`}
                  aria-label={t("common.remove")}
                >
                  {t("common.remove")}
                </button>
              </div>
            ))}
          </div>

          {/* SNMPv3 Credentials */}
          <div
            className={`border-t border-surface-border ${spacing.padding.top.heading}`}
          >
            <div
              className={`flex items-center justify-between ${spacing.margin.bottom.inline}`}
            >
              <span className="caption text-text-muted font-medium">
                {t("snmp.v3Credentials")}
              </span>
              <button
                onClick={addV3Credential}
                className="caption text-brand-primary hover:text-brand-accent"
              >
                {t("common.add")}
              </button>
            </div>
            {snmpSettings.v3Credentials.map((cred) => (
              <div
                key={cred.id}
                className={`${spacing.margin.bottom.inline} border border-surface-border ${radius.default} overflow-hidden`}
              >
                <div
                  className={`flex items-center justify-between ${spacing.pad.xs} bg-surface-base cursor-pointer hover:bg-surface-hover`}
                  onClick={() =>
                    setExpandedCredential(
                      expandedCredential === cred.id ? null : cred.id!
                    )
                  }
                >
                  <span className="body-small text-text-primary">
                    {cred.name || t("snmp.unnamedCredential")}
                  </span>
                  <div className={`flex items-center ${spacing.gap.compact}`}>
                    <span className="caption text-text-muted">
                      {cred.username || t("snmp.noUsername")}
                    </span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        removeV3Credential(cred.id!);
                      }}
                      className={`text-status-error hover:text-status-error/80 ${spacing.actionBtn}`}
                      aria-label={t("common.remove")}
                    >
                      {t("common.remove")}
                    </button>
                  </div>
                </div>
                {expandedCredential === cred.id && (
                  <div
                    className={`${spacing.pad.sm} bg-surface-hover stack-sm`}
                  >
                    {/* Name */}
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`cred-name-${cred.id}`}
                      >
                        {t("common.name")}
                      </label>
                      <input
                        id={`cred-name-${cred.id}`}
                        type="text"
                        value={cred.name}
                        onChange={(e) =>
                          updateV3Credential(cred.id!, "name", e.target.value)
                        }
                        placeholder={t("snmp.credentialName")}
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                      />
                    </div>

                    {/* Username */}
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`cred-username-${cred.id}`}
                      >
                        {t("snmp.username")}
                      </label>
                      <input
                        id={`cred-username-${cred.id}`}
                        type="text"
                        value={cred.username}
                        onChange={(e) =>
                          updateV3Credential(
                            cred.id!,
                            "username",
                            e.target.value
                          )
                        }
                        placeholder={t("snmp.snmpv3Username")}
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                      />
                    </div>

                    {/* Security Level */}
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`sec-level-${cred.id}`}
                      >
                        {t("snmp.securityLevel")}
                      </label>
                      <select
                        id={`sec-level-${cred.id}`}
                        value={cred.securityLevel}
                        onChange={(e) =>
                          updateV3Credential(
                            cred.id!,
                            "securityLevel",
                            e.target.value
                          )
                        }
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                      >
                        {SECURITY_LEVEL_VALUES.map((value) => (
                          <option key={value} value={value}>
                            {getSecurityLevelLabel(value)}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Authentication Protocol */}
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`auth-proto-${cred.id}`}
                      >
                        {t("snmp.authProtocol")}
                      </label>
                      <select
                        id={`auth-proto-${cred.id}`}
                        value={cred.authProtocol}
                        onChange={(e) =>
                          updateV3Credential(
                            cred.id!,
                            "authProtocol",
                            e.target.value
                          )
                        }
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                      >
                        {AUTH_PROTOCOL_VALUES.map((value) => (
                          <option key={value} value={value}>
                            {getAuthProtocolLabel(value)}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Authentication Password */}
                    {cred.authProtocol !== "" && (
                      <div>
                        <label
                          className="caption text-text-muted"
                          htmlFor={`auth-pass-${cred.id}`}
                        >
                          {t("snmp.authPassword")}
                        </label>
                        <input
                          id={`auth-pass-${cred.id}`}
                          type="password"
                          value={cred.authPassword}
                          onChange={(e) =>
                            updateV3Credential(
                              cred.id!,
                              "authPassword",
                              e.target.value
                            )
                          }
                          placeholder={t("snmp.authPasswordPlaceholder")}
                          className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                        />
                      </div>
                    )}

                    {/* Privacy Protocol */}
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`priv-proto-${cred.id}`}
                      >
                        {t("snmp.privProtocol")}
                      </label>
                      <select
                        id={`priv-proto-${cred.id}`}
                        value={cred.privProtocol}
                        onChange={(e) =>
                          updateV3Credential(
                            cred.id!,
                            "privProtocol",
                            e.target.value
                          )
                        }
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                      >
                        {PRIV_PROTOCOL_VALUES.map((value) => (
                          <option key={value} value={value}>
                            {getPrivProtocolLabel(value)}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Privacy Password */}
                    {cred.privProtocol !== "" && (
                      <div>
                        <label
                          className="caption text-text-muted"
                          htmlFor={`priv-pass-${cred.id}`}
                        >
                          {t("snmp.privPassword")}
                        </label>
                        <input
                          id={`priv-pass-${cred.id}`}
                          type="password"
                          value={cred.privPassword}
                          onChange={(e) =>
                            updateV3Credential(
                              cred.id!,
                              "privPassword",
                              e.target.value
                            )
                          }
                          placeholder={t("snmp.privPasswordPlaceholder")}
                          className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                        />
                      </div>
                    )}

                    {/* Context Name */}
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`context-name-${cred.id}`}
                      >
                        {t("snmp.contextName")}
                      </label>
                      <input
                        id={`context-name-${cred.id}`}
                        type="text"
                        value={cred.contextName}
                        onChange={(e) =>
                          updateV3Credential(
                            cred.id!,
                            "contextName",
                            e.target.value
                          )
                        }
                        placeholder={t("snmp.snmpContext")}
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} ${spacing.margin.top.tight} caption`}
                      />
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </CollapsibleSection>
  );
});
