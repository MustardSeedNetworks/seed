/**
 * LinkSettings Component
 *
 * Purpose: Configure interface link speed and duplex settings.
 * Single dropdown with all speed/duplex preset combinations.
 *
 * Key Features:
 * - Auto-negotiation option
 * - Combined speed/duplex presets (10M-100G)
 * - Copper: 10/100 support half/full duplex
 * - 1G+: Full duplex only (IEEE standard)
 * - Fiber speeds: 25G/40G/100G
 * - Shows available modes for selected interface
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, theme utilities
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Select } from '../../../shared/Select';
import { Toggle } from '../../../shared/Toggle';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import type {
  CardSettings,
  LinkSettings as LinkSettingsType,
  SaveStatus,
} from '../../../types/settings';
import { PlugZap } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

interface LinkSettingsProps {
  linkSettings: LinkSettingsType;
  setLinkSettings: React.Dispatch<React.SetStateAction<LinkSettingsType>>;
  linkStatus: SaveStatus;
  /** Card settings for visibility and FAB configuration */
  cardSettings: CardSettings;
  /** Update card settings (triggers auto-save to profile) */
  updateCardSettings: (updates: Partial<CardSettings>) => void;
}

// Combined speed/duplex mode options matching ethtool output format
// Format: "speed/duplex" (e.g., "10/half", "100/full", "1000/full")
// Note: Half duplex only available at 10/100 Mbps per IEEE standards
const LINK_MODE_OPTIONS: { value: string; label: string }[] = [
  { value: 'auto', label: 'Auto-Negotiate' },
  // 10 Mbps - supports half and full duplex
  { value: '10/half', label: '10 Mbps Half Duplex' },
  { value: '10/full', label: '10 Mbps Full Duplex' },
  // 100 Mbps - supports half and full duplex
  { value: '100/half', label: '100 Mbps Half Duplex' },
  { value: '100/full', label: '100 Mbps Full Duplex' },
  // 1 Gbps+ - full duplex only (IEEE 802.3)
  { value: '1000/full', label: '1 Gbps Full Duplex' },
  { value: '2500/full', label: '2.5 Gbps Full Duplex' },
  { value: '5000/full', label: '5 Gbps Full Duplex' },
  { value: '10000/full', label: '10 Gbps Full Duplex' },
  // Fiber speeds
  { value: '25000/full', label: '25 Gbps Full Duplex' },
  { value: '40000/full', label: '40 Gbps Full Duplex' },
  { value: '100000/full', label: '100 Gbps Full Duplex' },
];

/**
 * Settings section for link speed and duplex configuration.
 * Uses a single dropdown with combined speed/duplex modes.
 */
export function LinkSettings({
  linkSettings,
  setLinkSettings,
  linkStatus,
  cardSettings,
  updateCardSettings,
}: LinkSettingsProps): React.ReactElement {
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');

  // Handle mode change
  const handleModeChange = (mode: string): void => {
    setLinkSettings((prev) => ({
      ...prev,
      mode,
    }));
  };

  // Check if current mode is manual (not auto)
  const isManualMode = linkSettings.mode !== 'auto';

  return (
    <CollapsibleSection
      readOnlyReason={readOnlyReason}
      title={
        <div className={layout.inline.default}>
          <PlugZap className={iconTokens.size.sm} />
          <span>{t('sections.link')}</span>
          <AutoSaveIndicator status={linkStatus} />
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack">
        {/* Card Visibility & FAB Controls */}
        <div className="stack-sm">
          <Toggle
            label={t('common.showCard')}
            description={t('common.showCardDesc')}
            checked={cardSettings.link.enabled}
            onChange={(checked: boolean): void =>
              updateCardSettings({
                link: { ...cardSettings.link, enabled: checked },
              })
            }
            className="rounded-lg border border-surface-border bg-surface-base p-3"
          />
          <Toggle
            label={t('common.runOnFab')}
            description={t('common.runOnFabDesc')}
            checked={cardSettings.link.autoRunOnLink}
            onChange={(checked: boolean): void =>
              updateCardSettings({
                link: { ...cardSettings.link, autoRunOnLink: checked },
              })
            }
            className="rounded-lg border border-surface-border bg-surface-base p-3"
          />
        </div>

        {/* Combined Speed/Duplex Dropdown */}
        <FormField label={t('link.speedDuplex')}>
          {(control) => (
            <Select
              {...control}
              value={linkSettings.mode}
              onValueChange={handleModeChange}
              options={LINK_MODE_OPTIONS}
            />
          )}
        </FormField>

        {/* Warning for manual settings */}
        {isManualMode ? (
          <p className={cn('caption text-status-warning', spacing.margin.top.inline)}>
            {t('link.manualWarning')}
          </p>
        ) : null}

        {/* Available Modes Display */}
        {linkSettings.availableModes.length > 0 ? (
          <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <span className="caption text-text-muted font-medium">{t('link.availableModes')}</span>
            <div className={cn('flex flex-wrap', spacing.gap.tight, spacing.margin.top.inline)}>
              {linkSettings.availableModes.map((mode) => (
                <span
                  key={mode}
                  className={cn(
                    spacing.chip.sm,
                    'bg-surface-base border border-surface-border',
                    radius.default,
                    'caption text-text-muted',
                  )}
                >
                  {mode}
                </span>
              ))}
            </div>
          </div>
        ) : null}

        <p className={cn('caption text-text-muted', spacing.margin.top.inline)}>
          {t('link.requiresRoot')}
        </p>
      </div>
    </CollapsibleSection>
  );
}
