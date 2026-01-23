import type React from 'react';
import { memo, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { logger } from '../../../lib/logger';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import type {
  CardSettings,
  DiscoveryServiceStatus as DiscoveryServiceStatusType,
  NetworkDiscoverySettings as NetworkDiscoverySettingsType,
  SaveStatus,
  SnmpSettings as SnmpSettingsType,
  SubnetConfig,
} from '../../../types/settings';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { ScanSearch } from '../../ui/icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';
import { DiscoveryCustomOptions } from './discovery/DiscoveryCustomOptions';
import { DiscoveryServiceStatus } from './discovery/DiscoveryServiceStatus';
import { DiscoveryTimingSettings } from './discovery/DiscoveryTimingSettings';
import { DiscoveryToggles } from './discovery/DiscoveryToggles';
import { SnmpSettingsSection } from './discovery/SnmpSettingsSection';
import { SubnetManager } from './discovery/SubnetManager';

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
  /** Card settings for visibility and FAB configuration */
  cardSettings: CardSettings;
  /** Update card settings (triggers auto-save to profile) */
  updateCardSettings: (updates: Partial<CardSettings>) => void;
}

/**
 * Settings section for network discovery options and subnet management.
 * Refactored into sub-components for better maintainability.
 */
export const DiscoverySettings: React.NamedExoticComponent<DiscoverySettingsProps> = memo(
  function DiscoverySettingsComponent({
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
    cardSettings,
    updateCardSettings,
  }: DiscoverySettingsProps): React.ReactElement {
    const { t } = useTranslation('settings');
    const [serviceStatus, setServiceStatus] = useState<DiscoveryServiceStatusType | null>(null);
    const [statusLoading, setStatusLoading] = useState(false);

    // Fetch service status
    // Fixes #865: Log fetch errors for debugging instead of silently swallowing them
    const fetchServiceStatus = useCallback(async (): Promise<void> => {
      setStatusLoading(true);
      try {
        const response = await fetch('/api/v1/shell/discovery/service/status');
        if (response.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() is a Promise
          const data = (await response.json()) as DiscoveryServiceStatusType;
          setServiceStatus(data);
        } else {
          // Log non-OK responses for debugging
          logger.warn('discovery', 'Failed to fetch service status', { status: response.status });
        }
      } catch (err) {
        // Log error for debugging - status display is informational but errors help troubleshoot
        logger.warn('discovery', 'Error fetching service status', { error: err });
      } finally {
        setStatusLoading(false);
      }
    }, []);

    // Fetch status on mount and periodically
    useEffect((): (() => void) => {
      fetchServiceStatus().catch(() => undefined);
      const interval = setInterval((): void => {
        fetchServiceStatus().catch(() => undefined);
      }, 10000);
      return (): void => clearInterval(interval);
    }, [fetchServiceStatus]);

    return (
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <ScanSearch class={iconTokens.size.sm} />
            <span>{t('sections.discovery')}</span>
            <AutoSaveIndicator status={networkDiscoveryStatus} />
          </div>
        }
        defaultOpen={false}
      >
        <div class="stack">
          {/* Card Visibility & FAB Controls */}
          <div class="stack-sm">
            <label
              class={cn(
                layout.flex.between,
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div>
                <span class="body-small text-text-primary font-medium">
                  {t('common.showCard', 'Show Card')}
                </span>
                <p class="caption text-text-muted">
                  {t('common.showCardDesc', 'Display this card on the dashboard')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.networkDiscovery.enabled}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    networkDiscovery: {
                      ...cardSettings.networkDiscovery,
                      enabled: e.target.checked,
                    },
                  })
                }
                class={iconTokens.size.sm}
              />
            </label>
            <label
              class={cn(
                layout.flex.between,
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div>
                <span class="body-small text-text-primary font-medium">
                  {t('common.runOnFab', 'Include in Run All')}
                </span>
                <p class="caption text-text-muted">
                  {t('common.runOnFabDesc', 'Run when FAB button is clicked')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.networkDiscovery.autoRunOnLink}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    networkDiscovery: {
                      ...cardSettings.networkDiscovery,
                      autoRunOnLink: e.target.checked,
                    },
                  })
                }
                class={iconTokens.size.sm}
              />
            </label>
          </div>

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
  },
);
