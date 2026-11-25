/**
 * AppearanceSettings Component
 *
 * Purpose: Settings panel for theme (light/dark/system) selection.
 * Allows users to customize the visual appearance of the application.
 *
 * Key Features:
 * - Theme selector: Light, Dark and System, as a segmented control
 * - Quick toggle: button to quickly switch between light and dark themes
 * - System detection: respects OS dark mode preference when "System" is selected
 * - Icon feedback: shows moon emoji (🌙) for dark, sun emoji (☀️) for light
 * - CollapsibleSection wrapper: integrates with settings page layout
 * - Palette icon: visual indicator in settings menu
 *
 * Usage:
 * ```typescript
 * <AppearanceSettings
 *   theme="dark"
 *   setTheme={(t) => updateTheme(t)}
 *   isDark={true}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, Icons, theme utilities
 * Props: theme (string), setTheme (callback), isDark (boolean for current state)
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import i18n, { languages } from '../../../i18n';
import { accessibleNameIncluding } from '../../../lib/accessibleName';
import { FormField } from '../../../shared/FormField';
import { SegmentedControl } from '../../../shared/SegmentedControl';
import { Select } from '../../../shared/Select';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Palette } from '../../ui/Icons';

interface AppearanceSettingsProps {
  theme: 'light' | 'dark' | 'system';
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  isDark: boolean;
  unitSystem: 'sae' | 'metric';
  setUnitSystem: (unit: 'sae' | 'metric') => void;
}

function toTheme(value: string): 'light' | 'dark' | 'system' {
  if (value === 'light' || value === 'dark') {
    return value;
  }
  return 'system';
}

/**
 * Settings section for theme selection and language preferences.
 */
export function AppearanceSettings({
  theme,
  setTheme,
  isDark,
  unitSystem,
  setUnitSystem,
}: AppearanceSettingsProps): React.ReactElement {
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');
  // Normalize language code (e.g., "en-US" -> "en") and validate against supported languages
  const detectedLanguage = i18n.language?.split('-')[0] || 'en';
  const supportedCodes = languages.map((l) => l.code);
  const currentLanguage = supportedCodes.includes(
    detectedLanguage as (typeof supportedCodes)[number],
  )
    ? detectedLanguage
    : 'en';

  const handleLanguageChange = (langCode: string): void => {
    i18n.changeLanguage(langCode).catch(() => undefined);
  };

  return (
    <CollapsibleSection
      readOnlyReason={readOnlyReason}
      data-testid="appearance-settings-section"
      title={
        <div className={layout.inline.default}>
          <Palette className={iconTokens.size.sm} />
          <span>{t('sections.appearance')}</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm">
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-hairline bg-surface-base p-3">
          <span className="text-sm text-text-primary">{t('appearance.theme')}</span>
          <SegmentedControl
            label={t('appearance.theme')}
            value={theme}
            onChange={(value): void => setTheme(toTheme(value))}
            options={[
              { value: 'light', label: t('appearance.themeLight') },
              { value: 'dark', label: t('appearance.themeDark') },
              { value: 'system', label: t('appearance.themeSystem') },
            ]}
          />
        </div>

        <FormField label={t('appearance.language')}>
          {(control) => (
            <Select
              {...control}
              value={currentLanguage}
              onValueChange={handleLanguageChange}
              options={languages.map((lang) => ({ value: lang.code, label: lang.nativeLabel }))}
            />
          )}
        </FormField>

        <FormField label={t('appearance.measurementUnits')}>
          {(control) => (
            <Select
              {...control}
              value={unitSystem}
              onValueChange={(value): void => setUnitSystem(value === 'metric' ? 'metric' : 'sae')}
              options={[
                { value: 'sae', label: t('appearance.unitSae') },
                { value: 'metric', label: t('appearance.unitMetric') },
              ]}
            />
          )}
        </FormField>

        <button
          type="button"
          onClick={(): void => setTheme(isDark ? 'light' : 'dark')}
          data-testid="theme-toggle"
          aria-label={accessibleNameIncluding(
            isDark ? t('appearance.switchToLight') : t('appearance.switchToDark'),
            t('appearance.quickToggle'),
          )}
          className={cn(
            'w-full',
            layout.flex.between,
            spacing.pad.sm,
            'bg-surface-base',
            radius.default,
            'border border-surface-border hover:bg-surface-hover transition-colors',
          )}
        >
          <span className="body-small text-text-primary">{t('appearance.quickToggle')}</span>
          <span className="text-xl">{isDark ? '\u{1F319}' : '\u2600\uFE0F'}</span>
        </button>
      </div>
    </CollapsibleSection>
  );
}
