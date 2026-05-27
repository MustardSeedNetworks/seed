/**
 * useLocale — single source for the active i18next language code.
 *
 * Returns a BCP-47 tag suitable for `Intl.*` constructors. i18next
 * gives us the short code ('en' / 'es'); we normalise to the regional
 * tags 'en-US' / 'es-ES' so number/date formatters get a sensible
 * default (decimal separators, weekday names) without each call site
 * having to pick.
 *
 * Use this whenever you'd otherwise call `Intl.NumberFormat(undefined,
 * …)` or hardcode `'en-US'`. The hook re-renders consumers when the
 * user switches language, so formatted strings flip automatically.
 *
 * Mirrors NIAC/Stem hooks/useLocale.ts (Phase 5/7) for cross-product
 * consistency.
 */

import { useTranslation } from 'react-i18next';

const REGION_BY_LANGUAGE: Record<string, string> = {
  en: 'en-US',
  es: 'es-ES',
};

export function useLocale(): string {
  const { i18n } = useTranslation();
  return REGION_BY_LANGUAGE[i18n.language] ?? i18n.language ?? 'en-US';
}
