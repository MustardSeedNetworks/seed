/**
 * i18n TypeScript Types
 *
 * Provides type-safe translation keys for react-i18next.
 * These types are generated from the English locale files.
 *
 * Usage:
 * ```tsx
 * import { useTranslation } from 'react-i18next';
 * import type { CommonKeys } from './i18n/types';
 *
 * function MyComponent() {
 *   const { t } = useTranslation('common');
 *   // TypeScript will autocomplete 'buttons.save', 'status.connected', etc.
 *   return <button>{t('buttons.save')}</button>;
 * }
 * ```
 */

import type enCards from '@locales/en/cards.json';
import type enCommon from '@locales/en/common.json';
import type enErrors from '@locales/en/errors.json';
import type enGlossary from '@locales/en/glossary.json';
import type enHelp from '@locales/en/help.json';
import type enSettings from '@locales/en/settings.json';
import type enSetup from '@locales/en/setup.json';

/**
 * Type definitions for each namespace.
 */
export type CommonTranslations = typeof enCommon;
export type CardsTranslations = typeof enCards;
export type SettingsTranslations = typeof enSettings;
export type ErrorsTranslations = typeof enErrors;
export type GlossaryTranslations = typeof enGlossary;
export type HelpTranslations = typeof enHelp;
export type SetupTranslations = typeof enSetup;

/**
 * All translations combined.
 */
export interface Translations {
  common: CommonTranslations;
  cards: CardsTranslations;
  settings: SettingsTranslations;
  errors: ErrorsTranslations;
  glossary: GlossaryTranslations;
  help: HelpTranslations;
  setup: SetupTranslations;
}

/**
 * Declaration merging for react-i18next.
 * This enables autocomplete for translation keys.
 */
declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'common';
    resources: Translations;
  }
}
