import type React from 'react';
import { useTranslation } from 'react-i18next';
import { Checkbox } from '../../../../shared/Checkbox';
import { Input } from '../../../../shared/Input';
import { Tooltip } from '../../../../shared/Tooltip';
import { cn, layout, radius, spacing } from '../../../../styles/theme';
import type { SaveStatus, SubnetConfig } from '../../../../types/settings';
import { AutoSaveIndicator } from '../AutoSaveIndicator';

interface SubnetManagerProps {
  subnets: SubnetConfig[];
  subnetsStatus: SaveStatus;
  newSubnetCidr: string;
  setNewSubnetCidr: (value: string) => void;
  newSubnetName: string;
  setNewSubnetName: (value: string) => void;
  subnetError: string | null;
  setSubnetError: (value: string | null) => void;
  addSubnet: () => void;
  toggleSubnet: (cidr: string, enabled: boolean) => void;
  deleteSubnet: (cidr: string) => void;
}

/**
 * Manages target networks (subnets) for discovery.
 * Only shown when full_scan or custom profile is selected.
 */
export function SubnetManager({
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
}: SubnetManagerProps): React.ReactElement {
  const { t } = useTranslation('settings');

  return (
    <div className={cn('border-t border-surface-border', spacing.pad.sm)}>
      <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
        <span className="caption text-text-muted font-medium">
          {t('discovery.targetNetworks')} <AutoSaveIndicator status={subnetsStatus} />
        </span>
      </div>
      <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
        {t('discovery.targetNetworksDesc')}
      </p>
      {/* List of configured subnets */}
      {subnets.length > 0 ? (
        <div className={cn('stack-sm', spacing.margin.bottom.heading)}>
          {subnets.map((subnet) => (
            <div
              key={subnet.cidr}
              className={cn(
                layout.flex.between,
                spacing.pad.xs,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div className="flex-1 min-w-0">
                <div className="body-small text-text-primary truncate">
                  {subnet.name || subnet.cidr}
                </div>
                <div className="caption text-text-muted">{subnet.cidr}</div>
              </div>
              <div className={cn(layout.inline.default, spacing.margin.left.inline)}>
                <Tooltip
                  text={
                    subnet.enabled ? t('discovery.disableNetwork') : t('discovery.enableNetwork')
                  }
                >
                  {(description) => (
                    <Checkbox
                      // The name carries the CIDR because every row would
                      // otherwise announce the same words, and a screen-reader
                      // user picking one of several identical "Enable subnet"
                      // checkboxes cannot tell which is which.
                      label={
                        subnet.enabled
                          ? t('discovery.disableNetworkNamed', { cidr: subnet.cidr })
                          : t('discovery.enableNetworkNamed', { cidr: subnet.cidr })
                      }
                      hideLabel
                      aria-describedby={description['aria-describedby']}
                      onClick={description.onClick}
                      checked={subnet.enabled}
                      onChange={(enabled): void => toggleSubnet(subnet.cidr, enabled)}
                    />
                  )}
                </Tooltip>
                <Tooltip text={t('discovery.removeNetwork')}>
                  <button
                    type="button"
                    onClick={(): void => deleteSubnet(subnet.cidr)}
                    className="text-status-error hover:text-status-error/70 body-small"
                  >
                    X
                  </button>
                </Tooltip>
              </div>
            </div>
          ))}
        </div>
      ) : null}
      {/* Add new subnet form */}
      <div className="stack-sm">
        <Input
          value={newSubnetCidr}
          onChange={(e): void => {
            setNewSubnetCidr(e.target.value);
            setSubnetError(null);
          }}
          placeholder={t('discovery.cidrPlaceholder')}
          aria-label={t('discovery.cidrPlaceholder')}
        />
        <Input
          value={newSubnetName}
          onChange={(e): void => setNewSubnetName(e.target.value)}
          placeholder={t('discovery.namePlaceholder')}
          aria-label={t('discovery.namePlaceholder')}
        />
        {subnetError ? <p className="caption text-status-error">{subnetError}</p> : null}
        <button
          type="button"
          onClick={addSubnet}
          className={cn(
            'w-full',
            spacing.pad.sm,
            'bg-brand-primary hover:bg-brand-accent text-on-brand',
            radius.default,
            'body-small',
          )}
        >
          {t('discovery.addNetwork')}
        </button>
      </div>
    </div>
  );
}
