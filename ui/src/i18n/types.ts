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

import type enCards from "@locales/en/cards.json";
import type enCommon from "@locales/en/common.json";
import type enErrors from "@locales/en/errors.json";
import type enHelp from "@locales/en/help.json";
import type enSettings from "@locales/en/settings.json";
import type enSetup from "@locales/en/setup.json";
import type enSurvey from "@locales/en/survey.json";

/**
 * Type definitions for each namespace.
 */
export type CommonTranslations = typeof enCommon;
export type CardsTranslations = typeof enCards;
export type SettingsTranslations = typeof enSettings;
export type ErrorsTranslations = typeof enErrors;
export type HelpTranslations = typeof enHelp;
export type SetupTranslations = typeof enSetup;
export type SurveyTranslations = typeof enSurvey;

/**
 * All translations combined.
 */
export interface Translations {
  common: CommonTranslations;
  cards: CardsTranslations;
  settings: SettingsTranslations;
  errors: ErrorsTranslations;
  help: HelpTranslations;
  setup: SetupTranslations;
  survey: SurveyTranslations;
}

/**
 * Helper type to get nested keys from an object type.
 * Example: NestedKeys<{a: {b: string}}> = 'a.b'
 */
type NestedKeys<T, Prefix extends string = ""> = T extends object
  ? {
      [K in keyof T]: K extends string
        ? T[K] extends object
          ? NestedKeys<T[K], Prefix extends "" ? K : `${Prefix}.${K}`>
          : Prefix extends ""
            ? K
            : `${Prefix}.${K}`
        : never;
    }[keyof T]
  : never;

/**
 * Type-safe translation keys for each namespace.
 */
export type CommonKeys = NestedKeys<CommonTranslations>;
export type CardsKeys = NestedKeys<CardsTranslations>;
export type SettingsKeys = NestedKeys<SettingsTranslations>;
export type ErrorsKeys = NestedKeys<ErrorsTranslations>;
export type HelpKeys = NestedKeys<HelpTranslations>;
export type SetupKeys = NestedKeys<SetupTranslations>;
export type SurveyKeys = NestedKeys<SurveyTranslations>;

/**
 * Declaration merging for react-i18next.
 * This enables autocomplete for translation keys.
 */
declare module "i18next" {
  interface CustomTypeOptions {
    // biome-ignore lint/style/useNamingConvention: Required by i18next library interface
    defaultNS: "common";
    resources: Translations;
  }
}
