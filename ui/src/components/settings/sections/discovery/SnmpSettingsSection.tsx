import type React from 'react';
import { useTranslation } from 'react-i18next';
import { FormField } from '../../../../shared/FormField';
import { Input } from '../../../../shared/Input';
import { cn, layout, spacing } from '../../../../styles/theme';
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
export function SnmpSettingsSection({
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
      <FormField id="snmp-port" label={t('snmp.port')} className={spacing.margin.bottom.inline}>
        {(control) => (
          <Input
            {...control}
            type="number"
            value={snmpSettings.port}
            onChange={(e): void =>
              setSnmpSettings((prev) => ({
                ...prev,
                port: Number.parseInt(e.target.value, 10) || 161,
              }))
            }
            min="1"
            max="65535"
          />
        )}
      </FormField>
      <FormField
        id="snmp-timeout"
        label={t('snmp.timeout')}
        className={spacing.margin.bottom.inline}
      >
        {(control) => (
          <Input
            {...control}
            type="number"
            value={snmpSettings.timeout / 1000}
            onChange={(e): void =>
              setSnmpSettings((prev) => ({
                ...prev,
                timeout: (Number.parseFloat(e.target.value) || 5) * 1000,
              }))
            }
            min="1"
            max="30"
            step="1"
          />
        )}
      </FormField>
    </div>
  );
}
