import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, input as inputTokens, layout, spacing } from '../../../../styles/theme';
import type { SaveStatus, SnmpSettings } from '../../../../types/settings';
import { AutoSaveIndicator } from '../AutoSaveIndicator';

interface SnmpSettingsSectionProps {
  snmpSettings: SnmpSettings;
  setSnmpSettings: React.Dispatch<React.SetStateAction<SnmpSettings>>;
  snmpStatus: SaveStatus;
}

/**
 * SNMP transport settings within Discovery Settings.
 *
 * Community strings and v3 credentials are not edited here: they live in the
 * encrypted device-credential vault, which has its own section (#1799).
 */
export const SnmpSettingsSection: React.NamedExoticComponent<SnmpSettingsSectionProps> = memo(
  function SnmpSettingsSectionComponent({
    snmpSettings,
    setSnmpSettings,
    snmpStatus,
  }: SnmpSettingsSectionProps): React.ReactElement {
    const { t } = useTranslation('settings');

    return (
      <div className={cn('border-t border-surface-border', spacing.pad.sm)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="body-small text-text-primary font-medium">
            {t('sections.snmp')} <AutoSaveIndicator status={snmpStatus} />
          </span>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('snmp.description')}
        </p>
        {/* SNMP Port */}
        <div className={spacing.margin.bottom.inline}>
          <label className="caption text-text-muted" htmlFor="snmp-port">
            {t('snmp.port')}
          </label>
          <input
            id="snmp-port"
            type="number"
            value={snmpSettings.port}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              setSnmpSettings((prev) => ({
                ...prev,
                port: Number.parseInt(e.target.value, 10) || 161,
              }))
            }
            min="1"
            max="65535"
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.md,
              spacing.margin.top.tight,
              'body-small',
            )}
          />
        </div>
        {/* Timeout */}
        <div className={spacing.margin.bottom.inline}>
          <label className="caption text-text-muted" htmlFor="snmp-timeout">
            {t('snmp.timeout')}
          </label>
          <input
            id="snmp-timeout"
            type="number"
            value={snmpSettings.timeout / 1000}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              setSnmpSettings((prev) => ({
                ...prev,
                timeout: (Number.parseFloat(e.target.value) || 5) * 1000,
              }))
            }
            min="1"
            max="30"
            step="1"
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.md,
              spacing.margin.top.tight,
              'body-small',
            )}
          />
        </div>
      </div>
    );
  },
);
