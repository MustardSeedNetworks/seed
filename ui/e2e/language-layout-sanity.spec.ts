import { type Page, expect, test } from '@playwright/test';

/**
 * Language layout sanity (ES overflow detection)
 *
 * Spanish UI strings run ~30% longer than English on average. The
 * unit tests / key-resolution tests / language-switch spec only
 * verify that ES text RENDERS — they don't catch when text
 * overflows containers sized for shorter EN text.
 *
 * This spec exercises a small fixed set of high-traffic views in ES
 * and asserts that key text containers don't overflow their parents
 * (causing clipping, mid-word wrap, or horizontal scrollbars).
 *
 * Full pixel-diff visual regression is intentionally NOT in this
 * spec — see the cross-repo discussion in niac-go#733 for rationale.
 *
 * Mirrors niac-go#733 for cross-product coverage. Uses seed's bare
 * `language` localStorage key.
 */

const LOCAL_STORAGE_KEY = 'language';

const setLanguage = async (page: Page, lang: 'en' | 'es'): Promise<void> => {
  await page.addInitScript(
    ({ key, value }) => {
      localStorage.setItem(key, value);
    },
    { key: LOCAL_STORAGE_KEY, value: lang },
  );
};

const noHorizontalScroll = async (page: Page): Promise<void> => {
  const { scrollWidth, clientWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  expect(scrollWidth - clientWidth).toBeLessThan(2);
};

const noClippedText = async (page: Page, selector: string): Promise<void> => {
  const clipped = await page.locator(selector).evaluateAll((els: Element[]) =>
    els
      .filter((el) => {
        const e = el as HTMLElement;
        return e.scrollWidth > e.clientWidth && e.innerText.trim().length > 0;
      })
      .map((el) => (el as HTMLElement).innerText.trim().slice(0, 80)),
  );
  expect(clipped, `${selector} elements with clipped text:\n  ${clipped.join('\n  ')}`).toEqual(
    [],
  );
};

test.describe('Language layout sanity', () => {
  for (const lang of ['en', 'es'] as const) {
    test.describe(`${lang.toUpperCase()} dashboard`, () => {
      test.beforeEach(async ({ page }) => {
        await setLanguage(page, lang);
        await page.goto('/');
        await page.waitForLoadState('domcontentloaded');
      });

      test('no horizontal scrollbar on dashboard', async ({ page }) => {
        await noHorizontalScroll(page);
      });

      test("section headings don't get clipped", async ({ page }) => {
        // Section headings include "Connectivity" → "Conectividad",
        // "Network" → "Red", "Testing & Discovery" → "Pruebas y
        // descubrimiento" (~50% wider). High-risk because chrome is
        // sized for EN word count.
        await noClippedText(page, 'h1, h2, h3');
      });
    });
  }
});
