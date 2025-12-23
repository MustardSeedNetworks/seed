import { useState, useEffect, useCallback, memo } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { ScanSearch } from "../../ui/Icons";
import { icon as iconTokens, layout } from "../../../styles/theme";
import {
  NetworkDiscoverySettings as NetworkDiscoverySettingsType,
  SubnetConfig,
  SaveStatus,
  DiscoveryServiceStatus as DiscoveryServiceStatusType,
  SNMPSettings as SNMPSettingsType,
} from "../../../types/settings";
import {
  DiscoveryServiceStatus,
  DiscoveryToggles,
  DiscoveryProfileSelector,
  DiscoveryCustomOptions,
  DiscoveryTimingSettings,
  SubnetManager,
  SNMPSettingsSection,
} from "./discovery";

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

/**
 * Settings section for network discovery profiles and subnet management.
 * Refactored into sub-components for better maintainability.
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
    useState<DiscoveryServiceStatusType | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);

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

  // Handle profile change
  const handleProfileChange = useCallback(
    (profile: NetworkDiscoverySettingsType["profile"]) => {
      setNetworkDiscoverySettings((prev) => ({
        ...prev,
        profile,
      }));
    },
    [setNetworkDiscoverySettings]
  );

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
      defaultOpen={true}
    >
      <div className="stack">
        {/* Enable/Auto-scan Toggles */}
        <DiscoveryToggles
          settings={networkDiscoverySettings}
          onSettingsChange={setNetworkDiscoverySettings}
        />

        {/* Service Status Banner */}
        <DiscoveryServiceStatus
          status={serviceStatus}
          loading={statusLoading}
          onRefresh={fetchServiceStatus}
        />

        {/* Discovery Profile Selector */}
        <DiscoveryProfileSelector
          currentProfile={currentProfile}
          onProfileChange={handleProfileChange}
          onStatusRefresh={fetchServiceStatus}
        />

        {/* Custom Options (only shown when Custom profile is selected) */}
        {showCustomOptions && (
          <DiscoveryCustomOptions
            settings={networkDiscoverySettings}
            onSettingsChange={setNetworkDiscoverySettings}
          />
        )}

        {/* Timing Settings */}
        <DiscoveryTimingSettings
          settings={networkDiscoverySettings}
          onSettingsChange={setNetworkDiscoverySettings}
        />

        {/* Target Networks (only for full_scan or custom profile) */}
        {showSubnets && (
          <SubnetManager
            subnets={subnets}
            subnetsStatus={subnetsStatus}
            newSubnetCidr={newSubnetCidr}
            setNewSubnetCidr={setNewSubnetCidr}
            newSubnetName={newSubnetName}
            setNewSubnetName={setNewSubnetName}
            subnetError={subnetError}
            setSubnetError={setSubnetError}
            addSubnet={addSubnet}
            toggleSubnet={toggleSubnet}
            deleteSubnet={deleteSubnet}
          />
        )}

        {/* SNMP Settings Section */}
        <SNMPSettingsSection
          snmpSettings={snmpSettings}
          setSnmpSettings={setSnmpSettings}
          snmpStatus={snmpStatus}
        />
      </div>
    </CollapsibleSection>
  );
});
