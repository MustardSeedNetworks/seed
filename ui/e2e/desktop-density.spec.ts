/**
 * desktop-density.spec.ts — the desktop clarity and density ratchet
 * (UI-SEED-20, seed#2709).
 *
 * Owner directive 2026-09-16, "clear concise easy everywhere": Seed read loose
 * and half-empty on a wide monitor. Three things were measured on `main` and
 * are held here so they cannot come back.
 *
 * 1. The page header footprint. It was 83px on every route — the same figure
 *    stem and niac-go measured before their passes — and the fleet now shares
 *    one page-title scale, so the ceiling is the fleet's.
 * 2. Chrome spent on every page before its own content. The footer repeated
 *    the product name, the company and the version that the rail already
 *    carries, in a four-column block 192px tall, on all twelve routes. On
 *    /wifi that was a third of the whole page.
 * 3. Content fitting its own box. A card grid that lays out four columns for
 *    three cards leaves a dead column on the right at 1440px and squeezes
 *    every card to 232px at 1280px, where card headers paint outside their
 *    own border.
 *
 * The overflow walk is PER ELEMENT rather than `document.scrollWidth`
 * (UI-FLEET-2 finding 1): a card clips or overpaints its own content without
 * the document ever overflowing, which is exactly the failure this row is
 * about. Its exemptions are AUTHORED, never a pixel tolerance:
 * `data-phone-width-exempt` (the fleet attribute), `text-overflow: ellipsis`
 * (truncation is the prescribed fit, and a truncated cell overflows by
 * construction) and `sr-only` (the 1px clip IS the technique).
 */

import { expect, test } from '@playwright/test';

import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';

/** Every `pageRegistry` route. A subset would only prove the subset. */
const ROUTES = [
  '/link',
  '/network',
  '/path',
  '/wifi',
  '/security',
  '/performance',
  '/reports',
  '/logs',
  '/polling-targets',
  '/topology',
  '/alerts',
];

/**
 * The two desktop widths the acceptance names. 1280 is where a four-column
 * grid squeezed cards hardest; 1440 is where the dead right column showed.
 */
const WIDTHS = [1280, 1440];

/**
 * 80px. Measured on main: 83px on every route with the old `text-2xl
 * sm:text-3xl` title and the 32px icon. stem and niac-go hold the same
 * ceiling after the same step, so a regression in any one repo fails here.
 */
const HEADER_CEILING = 80;

/**
 * 96px. The footer was 192px on every route. One quiet line of legal and
 * contact links is what a diagnostic page can afford; the product name,
 * company and version it used to repeat are in the rail.
 */
const FOOTER_CEILING = 96;

test.use({ storageState: AUTH_STORAGE_STATE });

async function settle(page: import('@playwright/test').Page, route: string) {
  await page.goto(route, { waitUntil: 'domcontentloaded' });
  await disableAnimations(page);
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
  await page.waitForFunction(
    () => {
      const current = document.documentElement.scrollHeight;
      const store = window as unknown as { __lastHeight?: number };
      const previous = store.__lastHeight;
      store.__lastHeight = current;

      return previous === current;
    },
    undefined,
    { timeout: 10000, polling: 250 },
  );
}

test.describe('desktop density', () => {
  for (const width of WIDTHS) {
    test.describe(`at ${width}px`, () => {
      for (const route of ROUTES) {
        test(`${route} spends its height on content`, async ({ page }) => {
          await page.setViewportSize({ width, height: 900 });
          await settle(page, route);

          const measured = await page.evaluate(() => {
            const title = document.querySelector('[data-testid="page-header-title"]');
            let header: Element | null = title;
            while (header && !header.classList.contains('animate-fade-in')) {
              header = header.parentElement;
            }
            const footer = document.querySelector('[data-testid="app-footer"]');

            return {
              header: header ? Math.round(header.getBoundingClientRect().height) : -1,
              footer: footer ? Math.round(footer.getBoundingClientRect().height) : -1,
            };
          });

          expect(measured.header, 'page header height').toBeLessThanOrEqual(HEADER_CEILING);
          expect(measured.footer, 'footer height').toBeLessThanOrEqual(FOOTER_CEILING);
        });

        test(`${route} fits its content to its boxes`, async ({ page }) => {
          await page.setViewportSize({ width, height: 900 });
          await settle(page, route);

          const offenders = await page.evaluate(() => {
            const found: string[] = [];
            for (const element of Array.from(document.querySelectorAll('*'))) {
              const el = element as HTMLElement;
              if (el.clientWidth === 0 || el.scrollWidth - el.clientWidth <= 1) {
                continue;
              }
              if (el.classList.contains('sr-only') || el.closest('[data-phone-width-exempt]')) {
                continue;
              }
              if (getComputedStyle(el).textOverflow === 'ellipsis') {
                continue;
              }
              const classes = (el.className || '').toString().split(/\s+/).slice(0, 3).join('.');
              found.push(
                `${el.tagName.toLowerCase()}.${classes} ${el.scrollWidth}>${el.clientWidth} ` +
                  `"${(el.textContent || '').trim().slice(0, 40)}"`,
              );
            }

            return found;
          });

          expect(offenders, 'elements whose content is wider than their box').toEqual([]);
        });
      }
    });
  }
});
