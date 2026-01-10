import type React from "react";
import { memo, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { logger } from "../../../lib/logger";
import { icon as iconTokens, layout } from "../../../styles/theme";
import type {
  DiscoveryServiceStatus as DiscoveryServiceStatusType,
  NetworkDiscoverySettings as NetworkDiscoverySettingsType,
  SaveStatus,
  SnmpSettings as SnmpSettingsType,
  SubnetConfig,
} from "../../../types/settings";
import { CollapsibleSection } from "../../ui/collapsible-section";
import { ScanSearch } from "../../ui/icons";
import { AutoSaveIndicator } from "./auto-save-indicator";
import { DiscoveryCustomOptions } from "./discovery/discovery-custom-options";
import { DiscoveryServiceStatus } from "./discovery/discovery-service-status";
import { DiscoveryTimingSettings } from "./discovery/discovery-timing-settings";
import { DiscoveryToggles } from "./discovery/discovery-toggles";
import { SnmpSettingsSection } from "./discovery/snmp-settings-section";
import { SubnetManager } from "./discovery/subnet-manager";

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
  // SNMP settings (now integrated under Discovery)
  snmpSettings: SnmpSettingsType;
  setSnmpSettings: React.Dispatch<React.SetStateAction<SnmpSettingsType>>;
  snmpStatus: SaveStatus;
}

/**
 * Settings section for network discovery options and subnet management.
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
  const [serviceStatus, setServiceStatus] = useState<DiscoveryServiceStatusType | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);

  // Fetch service status
  // Fixes #865: Log fetch errors for debugging instead of silently swallowing them
  const fetchServiceStatus = useCallback(async () => {
    setStatusLoading(true);
    try {
      const response = await fetch("/api/v1/shell/discovery/service/status");
      if (response.ok) {
        const data = await response.json();
        setServiceStatus(data);
      } else {
        // Log non-OK responses for debugging
        logger.warn("discovery", "Failed to fetch service status", { status: response.status });
      }
    } catch (err) {
      // Log error for debugging - status display is informational but errors help troubleshoot
      logger.warn("discovery", "Error fetching service status", { error: err });
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

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <ScanSearch className={iconTokens.size.sm} />
          <span>{t("sections.discovery")}</span>
          <AutoSaveIndicator status={networkDiscoveryStatus} />
        </div>
      }
      defaultOpen={false}
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

        {/* Discovery Options */}
        <DiscoveryCustomOptions
          settings={networkDiscoverySettings}
          onSettingsChange={setNetworkDiscoverySettings}
        />

        {/* Timing Settings */}
        <DiscoveryTimingSettings
          settings={networkDiscoverySettings}
          onSettingsChange={setNetworkDiscoverySettings}
        />

        {/* Target Networks */}
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

        {/* SNMP Settings Section */}
        <SnmpSettingsSection
          snmpSettings={snmpSettings}
          setSnmpSettings={setSnmpSettings}
          snmpStatus={snmpStatus}
        />
      </div>
    </CollapsibleSection>
  );
});
