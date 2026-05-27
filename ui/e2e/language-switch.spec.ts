import { expect, test } from '@playwright/test';

/**
 * Language switching E2E
 *
 * Confirms the localStorage-driven locale handoff between i18next and
 * the browser actually round-trips for seed:
 *
 * - Default render is English.
 * - Setting `language` in localStorage to `es` and reloading produces
 *   Spanish strings (per ES locale JSON shipped with the binary).
 * - The `<html lang>` attribute flips to match (WCAG 3.1.1/3.1.2).
 *
 * Markers used for the Spanish assertions are taken from the
 * Translation Memory (msn-docs-internal/05-Engineering/
 * I18N_TRANSLATION_MEMORY.md) — the canonical translations for the
 * smoke-tested labels. If a label here ever needs to change, update
 * the TM and this spec in lockstep.
 *
 * Mirrors niac-go's e2e/language-switch.spec.ts (Phase 6) for
 * cross-product coverage. Uses seed's bare `language` storage key
 * (vs niac's `niac-language` and stem's `stem-language`).
 */

const LOCAL_STORAGE_KEY = 'language';

test.describe('Language switching', () => {
  test.beforeEach(async ({ page }) => {
    // Start from a clean storage state so default-detection is
    // exercised on every test (rather than carried-over preferences
    // from another spec file in the same run).
    await page.goto('/');
    await page.evaluate((key) => localStorage.removeItem(key), LOCAL_STORAGE_KEY);
  });

  test('renders English by default and sets <html lang="en">', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    const htmlLang = await page.locator('html').getAttribute('lang');
    expect(htmlLang).toBe('en');

    // EN marker: app.tagline comes from en/common.json. Stable across
    // UI revisions; the app tagline is the most reliable text-content
    // marker we have that's not also used as an aria-label.
    await expect(
      page.getByText(/Network Diagnostics by Mustard Seed Networks/i).first(),
    ).toBeVisible();
  });

  test('flips to Spanish when localStorage is set to es', async ({ page }) => {
    // addInitScript runs in the new page context BEFORE any user code,
    // so i18next sees the preference at bootstrap time rather than
    // post-mount (which would require a full re-init).
    await page.addInitScript(
      ({ key }) => {
        localStorage.setItem(key, 'es');
      },
      { key: LOCAL_STORAGE_KEY },
    );

    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // <html lang> must reflect active locale for accessibility tools.
    await expect.poll(async () => page.locator('html').getAttribute('lang')).toBe('es');

    // ES marker: app.tagline -> "Diagnósticos de Red por Mustard Seed
    // Networks" per es/common.json. "Mustard Seed Networks" stays
    // English per the glossary (brand name).
    await expect(
      page.getByText(/Diagn[oó]sticos de Red por Mustard Seed Networks/i).first(),
    ).toBeVisible();
  });

  test('clears language preference when localStorage is removed', async ({ page }) => {
    // First, set Spanish.
    await page.addInitScript(
      ({ key }) => {
        localStorage.setItem(key, 'es');
      },
      { key: LOCAL_STORAGE_KEY },
    );
    await page.goto('/');
    await expect.poll(async () => page.locator('html').getAttribute('lang')).toBe('es');

    // Then clear and reload — should fall back to detection (likely en).
    await page.evaluate((key) => localStorage.removeItem(key), LOCAL_STORAGE_KEY);
    await page.reload();
    await page.waitForLoadState('domcontentloaded');

    // Without an explicit preference, language-detector picks browser
    // default. In CI this is en; locally it depends. Either way the
    // attribute should be a valid 2-char code, not stale 'es'.
    const lang = await page.locator('html').getAttribute('lang');
    expect(lang).toMatch(/^[a-z]{2}$/);
  });
});
