import type React from 'react';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDefaults } from '../../../../hooks/useDefaults';
import { useLocale } from '../../../../hooks/useLocale';
import { CollapsibleSection } from '../../../../shared/CollapsibleSection';
import { FormField } from '../../../../shared/FormField';
import { Input } from '../../../../shared/Input';
import { cn, spacing } from '../../../../styles/theme';
import type { NetworkDiscoverySettings } from '../../../../types/settings';

const MS_PER_UNIT = { minute: 60_000, second: 1000 } as const;
type Unit = keyof typeof MS_PER_UNIT;

interface DurationFieldProps {
  label: string;
  description: string;
  valueMs: number;
  recommendedMs: number | undefined;
  unit: Unit;
  min: number;
  max: number;
  onChange: (ms: number) => void;
  'data-testid': string;
}

/**
 * A duration edited in a human unit and stored in milliseconds. The server
 * value is only rewritten when the operator types a valid one, so a value
 * that is not a whole number of units (a hand-edited config) survives a save
 * untouched.
 */
function DurationField({
  label,
  description,
  valueMs,
  recommendedMs,
  unit,
  min,
  max,
  onChange,
  'data-testid': testId,
}: DurationFieldProps): React.JSX.Element {
  const { t } = useTranslation('settings');
  const locale = useLocale();
  const id = useId();
  const [draft, setDraft] = useState<string | null>(null);
  const perUnit = MS_PER_UNIT[unit];
  const formatted = new Intl.NumberFormat(locale, { style: 'unit', unit, unitDisplay: 'long' });

  return (
    <div className={spacing.margin.top.content}>
      <FormField
        id={id}
        label={label}
        hint={
          <>
            {description}
            {recommendedMs === undefined
              ? null
              : ` ${t('discovery.recommended', { value: formatted.format(recommendedMs / perUnit) })}`}
          </>
        }
      >
        {(control) => (
          <Input
            {...control}
            data-testid={testId}
            type="number"
            inputMode="numeric"
            min={min}
            max={max}
            step={1}
            value={draft ?? String(valueMs / perUnit)}
            onChange={(e): void => {
              setDraft(e.target.value);
              const units = e.target.valueAsNumber;
              if (Number.isInteger(units) && units >= min && units <= max) {
                onChange(units * perUnit);
              }
            }}
            onBlur={(): void => setDraft(null)}
          />
        )}
      </FormField>
    </div>
  );
}

interface DiscoveryTimingSettingsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * The two discovery timers the daemon reads: how often it rescans, and how
 * long one sweep may run.
 */
export function DiscoveryTimingSettings({
  settings,
  onSettingsChange,
}: DiscoveryTimingSettingsProps) {
  const { t } = useTranslation('settings');
  const { defaults } = useDefaults();
  const recommended = defaults?.networkDiscovery;

  return (
    <div className={cn('border-t border-surface-border', spacing.pad.sm)}>
      <span className="caption text-text-muted font-medium">{t('discovery.timingSettings')}</span>
      <DurationField
        data-testid="discovery-rescan-interval"
        label={t('discovery.rescanInterval')}
        description={t('discovery.rescanIntervalDesc')}
        valueMs={settings.timing.rescanIntervalMs}
        recommendedMs={recommended?.timing.rescanIntervalMs}
        unit="minute"
        min={1}
        max={60}
        onChange={(ms): void =>
          onSettingsChange((prev) => ({
            ...prev,
            timing: { ...prev.timing, rescanIntervalMs: ms },
          }))
        }
      />
      <div className={spacing.margin.top.content}>
        <CollapsibleSection
          title={t('discovery.advancedTiming')}
          variant="compact"
          data-testid="discovery-timing-advanced"
        >
          <DurationField
            data-testid="discovery-scan-timeout"
            label={t('discovery.scanTimeout')}
            description={t('discovery.scanTimeoutDesc')}
            valueMs={settings.scanTimeoutMs}
            recommendedMs={recommended?.scanTimeoutMs}
            unit="second"
            min={5}
            max={120}
            onChange={(ms): void => onSettingsChange((prev) => ({ ...prev, scanTimeoutMs: ms }))}
          />
        </CollapsibleSection>
      </div>
    </div>
  );
}
