/**
 * AppearanceSettings Component
 *
 * Purpose: Settings panel for theme (light/dark/system) selection.
 * Allows users to customize the visual appearance of the application.
 *
 * Key Features:
 * - Theme selector: dropdown with Light, Dark, and System options
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
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import i18n, { languages } from '../../../i18n';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Palette } from '../../ui/icons';

interface AppearanceSettingsProps {
  theme: 'light' | 'dark' | 'system';
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  isDark: boolean;
  unitSystem: 'sae' | 'metric';
  setUnitSystem: (unit: 'sae' | 'metric') => void;
}

/**
 * Settings section for theme selection and language preferences.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const AppearanceSettings: React.NamedExoticComponent<AppearanceSettingsProps> = memo(
  function AppearanceSettingsComponent({
    theme,
    setTheme,
    isDark,
    unitSystem,
    setUnitSystem,
  }: AppearanceSettingsProps): React.ReactElement {
    const { t } = useTranslation('settings');
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
        title={
          <div class={layout.inline.default}>
            <Palette class={iconTokens.size.sm} />
            <span>{t('sections.appearance')}</span>
          </div>
        }
        defaultOpen={false}
      >
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
            <span class="body-small text-text-primary">{t('appearance.theme')}</span>
            <select
              value={theme}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTheme(e.target.value as 'light' | 'dark' | 'system')
              }
              class={cn(
                'bg-surface-raised border border-surface-border',
                radius.default,
                spacing.chip.sm,
                'body-small text-text-primary',
              )}
            >
              <option value="light">{t('appearance.themeLight')}</option>
              <option value="dark">{t('appearance.themeDark')}</option>
              <option value="system">{t('appearance.themeSystem')}</option>
            </select>
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
            <span class="body-small text-text-primary">{t('appearance.language')}</span>
            <select
              value={currentLanguage}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                handleLanguageChange(e.target.value)
              }
              class={cn(
                'bg-surface-raised border border-surface-border',
                radius.default,
                spacing.chip.sm,
                'body-small text-text-primary',
              )}
            >
              {languages.map((lang) => (
                <option key={lang.code} value={lang.code}>
                  {lang.nativeLabel}
                </option>
              ))}
            </select>
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
            <span class="body-small text-text-primary">{t('appearance.measurementUnits')}</span>
            <select
              value={unitSystem}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setUnitSystem(e.target.value as 'sae' | 'metric')
              }
              class={cn(
                'bg-surface-raised border border-surface-border',
                radius.default,
                spacing.chip.sm,
                'body-small text-text-primary',
              )}
            >
              <option value="sae">{t('appearance.unitSae')}</option>
              <option value="metric">{t('appearance.unitMetric')}</option>
            </select>
          </label>

          <button
            type="button"
            onClick={(): void => setTheme(isDark ? 'light' : 'dark')}
            class={cn(
              'w-full',
              layout.flex.between,
              spacing.pad.sm,
              'bg-surface-base',
              radius.default,
              'border border-surface-border hover:bg-surface-hover transition-colors',
            )}
          >
            <span class="body-small text-text-primary">{t('appearance.quickToggle')}</span>
            <span class="text-xl">{isDark ? '\u{1F319}' : '\u2600\uFE0F'}</span>
          </button>
        </div>
      </CollapsibleSection>
    );
  },
);
