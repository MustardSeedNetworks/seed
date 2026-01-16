/**
 * i18n Configuration
 *
 * Configures react-i18next for internationalization support.
 * Loads translations from shared locale files at /locales/{lang}/*.json
 *
 * Supported languages:
 * - en: English (default)
 * - es: Spanish
 *
 * Usage:
 * ```tsx
 * import { useTranslation } from 'react-i18next';
 *
 * function MyComponent() {
 *   const { t } = useTranslation('common');
 *   return <button>{t('buttons.save')}</button>;
 * }
 * ```
 */

// Import English locale files
import enCards from "@locales/en/cards.json";
import enCommon from "@locales/en/common.json";
import enErrors from "@locales/en/errors.json";
import enGlossary from "@locales/en/glossary.json";
import enHelp from "@locales/en/help.json";
import enSettings from "@locales/en/settings.json";
import enSetup from "@locales/en/setup.json";
import enSurvey from "@locales/en/survey.json";
// Import Spanish locale files
import esCards from "@locales/es/cards.json";
import esCommon from "@locales/es/common.json";
import esErrors from "@locales/es/errors.json";
import esGlossary from "@locales/es/glossary.json";
import esHelp from "@locales/es/help.json";
import esSettings from "@locales/es/settings.json";
import esSetup from "@locales/es/setup.json";
import esSurvey from "@locales/es/survey.json";
import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";

/**
 * Available languages configuration.
 */
export const languages = [
  { code: "en", label: "English", nativeLabel: "English" },
  { code: "es", label: "Spanish", nativeLabel: "Español" },
] as const;

export type LanguageCode = (typeof languages)[number]["code"];

/**
 * Translation namespaces.
 */
export const namespaces = [
  "common",
  "cards",
  "settings",
  "errors",
  "glossary",
  "help",
  "setup",
  "survey",
] as const;

export type Namespace = (typeof namespaces)[number];

/**
 * Default namespace used when none is specified.
 */
export const defaultNs: Namespace = "common";

/**
 * Resources organized by language and namespace.
 */
const resources: { en: Record<string, unknown>; es: Record<string, unknown> } = {
  en: {
    common: enCommon,
    cards: enCards,
    settings: enSettings,
    errors: enErrors,
    glossary: enGlossary,
    help: enHelp,
    setup: enSetup,
    survey: enSurvey,
  },
  es: {
    common: esCommon,
    cards: esCards,
    settings: esSettings,
    errors: esErrors,
    glossary: esGlossary,
    help: esHelp,
    setup: esSetup,
    survey: esSurvey,
  },
};

i18n
  // Detect user language from browser/localStorage
  .use(LanguageDetector)
  // Pass i18n instance to react-i18next
  .use(initReactI18next)
  // Initialize i18next
  .init({
    resources,
    fallbackLng: "en",
    // biome-ignore lint/style/useNamingConvention: i18next API
    defaultNS: defaultNs,
    ns: namespaces,

    // Language detection options
    detection: {
      // Order of language detection
      order: ["localStorage", "navigator", "htmlTag"],
      // Cache user language in localStorage
      caches: ["localStorage"],
      // Key to store language preference
      lookupLocalStorage: "language",
    },

    interpolation: {
      // React already escapes values
      escapeValue: false,
    },

    // Debug mode in development
    debug: import.meta.env.DEV,
  });

export default i18n;
