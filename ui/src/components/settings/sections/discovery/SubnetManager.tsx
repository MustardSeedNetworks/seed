import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../../styles/theme';
import type { SaveStatus, SubnetConfig } from '../../../../types/settings';
import { AutoSaveIndicator } from '../AutoSaveIndicator';

interface SubnetManagerProps {
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

/**
 * Manages target networks (subnets) for discovery.
 * Only shown when full_scan or custom profile is selected.
 */
export const SubnetManager: React.NamedExoticComponent<SubnetManagerProps> = memo(
  function SubnetManagerComponent({
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
      <div class={cn('border-t border-surface-border', spacing.pad.sm)}>
        <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span class="caption text-text-muted font-medium">
            {t('discovery.targetNetworks')} <AutoSaveIndicator status={subnetsStatus} />
          </span>
        </div>
        <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('discovery.targetNetworksDesc')}
        </p>

        {/* List of configured subnets */}
        {subnets.length > 0 ? (
          <div class={cn('stack-sm', spacing.margin.bottom.heading)}>
            {subnets.map((subnet) => (
              <div
                key={subnet.cidr}
                class={cn(
                  layout.flex.between,
                  spacing.pad.xs,
                  'bg-surface-base',
                  radius.default,
                  'border border-surface-border',
                )}
              >
                <div class="flex-1 min-w-0">
                  <div class="body-small text-text-primary truncate">
                    {subnet.name || subnet.cidr}
                  </div>
                  <div class="caption text-text-muted">{subnet.cidr}</div>
                </div>
                <div class={cn(layout.inline.default, spacing.margin.left.inline)}>
                  <input
                    type="checkbox"
                    checked={subnet.enabled}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                      toggleSubnet(subnet.cidr, e.target.checked)
                    }
                    class={iconTokens.size.sm}
                    title={
                      subnet.enabled ? t('discovery.disableSubnet') : t('discovery.enableSubnet')
                    }
                  />
                  <button
                    type="button"
                    onClick={(): void => deleteSubnet(subnet.cidr)}
                    class="text-status-error hover:text-status-error/70 body-small"
                    title={t('discovery.removeSubnet')}
                  >
                    X
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : null}

        {/* Add new subnet form */}
        <div class="stack-sm">
          <input
            type="text"
            value={newSubnetCidr}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void => {
              setNewSubnetCidr(e.target.value);
              setSubnetError(null);
            }}
            placeholder={t('discovery.cidrPlaceholder')}
            class={cn(
              'w-full',
              spacing.chip.lg,
              'bg-surface-base border border-surface-border',
              radius.default,
              'body-small text-text-primary',
            )}
          />
          <input
            type="text"
            value={newSubnetName}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              setNewSubnetName(e.target.value)
            }
            placeholder={t('discovery.namePlaceholder')}
            class={cn(
              'w-full',
              spacing.chip.lg,
              'bg-surface-base border border-surface-border',
              radius.default,
              'body-small text-text-primary',
            )}
          />
          {subnetError ? <p class="caption text-status-error">{subnetError}</p> : null}
          <button
            type="button"
            onClick={addSubnet}
            class={cn(
              'w-full',
              spacing.pad.sm,
              'bg-brand-primary hover:bg-brand-accent text-text-inverse',
              radius.default,
              'body-small',
            )}
          >
            {t('discovery.addSubnet')}
          </button>
        </div>
      </div>
    );
  },
);
