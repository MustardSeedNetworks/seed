/**
 * i18n Configuration
 *
 * Configures react-i18next for internationalization support.
 * Loads translations from shared locale files at /locales/{lang}/*.json.
 * English is bundled, since it is the default and every lookup's fallback;
 * any other language is fetched the first time it is selected.
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
import enCards from '@locales/en/cards.json';
import enCommon from '@locales/en/common.json';
import enErrors from '@locales/en/errors.json';
import enGlossary from '@locales/en/glossary.json';
import enHelp from '@locales/en/help.json';
import enPages from '@locales/en/pages.json';
import enSettings from '@locales/en/settings.json';
import enSetup from '@locales/en/setup.json';
import i18n, { type BackendModule, type ResourceLanguage } from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

/**
 * Available languages configuration.
 */
export const languages = [
  { code: 'en', label: 'English', nativeLabel: 'English' },
  { code: 'es', label: 'Spanish', nativeLabel: 'Español' },
] as const;

export type LanguageCode = (typeof languages)[number]['code'];

/**
 * Translation namespaces.
 */
export const namespaces = [
  'common',
  'cards',
  'settings',
  'errors',
  'glossary',
  'help',
  'setup',
  'pages',
] as const;

export type Namespace = (typeof namespaces)[number];

/**
 * Default namespace used when none is specified.
 */
export const defaultNs: Namespace = 'common';

/**
 * Every language but English loads on demand. `supportedLngs` below keeps
 * i18next from asking for any language this table does not name.
 */
const lazyLanguages: ReadonlyMap<string, () => Promise<ResourceLanguage>> = new Map([
  ['es', () => import('./es').then((m) => m.es)],
]);

const lazyLanguageBackend: BackendModule = {
  type: 'backend',
  init: () => undefined,
  read: (language, namespace, callback) => {
    const load = lazyLanguages.get(language);
    if (!load) {
      // English: bundled, so a reload has nothing to add.
      callback(null, {});
      return;
    }
    load().then(
      (resources) => callback(null, resources[namespace]),
      (error: unknown) =>
        callback(error instanceof Error ? error : new Error(String(error)), false),
    );
  },
};

/**
 * Resolves once the active language's resources are in place. The entry point
 * awaits it before the first render, so a Spanish session never paints English.
 */
export const i18nReady: Promise<unknown> = i18n
  .use(lazyLanguageBackend)
  // Detect user language from browser/localStorage
  .use(LanguageDetector)
  // Pass i18n instance to react-i18next
  .use(initReactI18next)
  // Initialize i18next
  .init({
    resources: {
      en: {
        common: enCommon,
        cards: enCards,
        settings: enSettings,
        errors: enErrors,
        glossary: enGlossary,
        help: enHelp,
        pages: enPages,
        setup: enSetup,
      },
    },
    // Only the languages missing from `resources` go to the backend.
    partialBundledLanguages: true,
    supportedLngs: languages.map((language) => language.code),
    // `es-MX` is served by `es`, as it was when both were bundled.
    nonExplicitSupportedLngs: true,
    load: 'languageOnly',
    fallbackLng: 'en',
    defaultNS: defaultNs,
    ns: namespaces,

    // Language detection options
    detection: {
      // Order of language detection
      order: ['localStorage', 'navigator', 'htmlTag'],
      // Cache user language in localStorage
      caches: ['localStorage'],
      // Key to store language preference
      lookupLocalStorage: 'language',
    },

    interpolation: {
      // React already escapes values
      escapeValue: false,
    },

    // Debug mode in development
    debug: import.meta.env.DEV,

    // Report keys that do not resolve. i18next renders the key itself when a
    // lookup fails, so `settings.mode.reflector` appears in the UI as though
    // it were a label. Reporting costs nothing here -- nothing is persisted --
    // and the test setup turns each report into a failing test (#1942).
    saveMissing: true,
  });

// Keep the document's lang attribute in sync with the active locale.
// Required for screen readers, search engines, browser spell-check, and
// CSS :lang() selectors. Without this, `<html lang="en">` from index.html
// stays English even after the user switches to Spanish — WCAG 3.1.1/3.1.2
// violation. Per msn-docs-internal/05-Engineering/I18N_CONVENTIONS.md.
if (typeof document !== 'undefined') {
  document.documentElement.lang = i18n.language;
  i18n.on('languageChanged', (lng) => {
    document.documentElement.lang = lng;
  });
}

export default i18n;
