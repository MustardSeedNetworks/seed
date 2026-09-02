/**
 * minimum-width.spec.ts — the UI has to work at the width we publish.
 *
 * `EDITIONS.md` §5 states a minimum supported width of **480px** (comfortable
 * target 768px+), because a Lite unit is reached from a phone or tablet over
 * its AP-mode SSID. Owner confirmed 2026-09-02 that this stands (#244).
 *
 * This is a *narrow viewport* obligation, not a mobile-device one — it runs on
 * the same chromium/webkit projects as everything else, by setting a viewport.
 * `mobile-chrome` / `mobile-safari` device projects stay deleted; see
 * E2E_CONVENTIONS.md, which previously conflated the two.
 *
 * The assertion is horizontal overflow, because that is the failure a narrow
 * viewport actually produces and the one an operator cannot work around: a page
 * that scrolls sideways hides controls off the right edge with no affordance
 * saying so. Everything else about a cramped layout is a judgement call; this
 * is not.
 *
 * Verified failable before landing: injecting a 2000px-wide div made every case
 * fail with "dashboard overflows 480px by 1520px. Widest offenders: [...]". A
 * layout assertion that has never been seen to fail is not evidence of anything.
 */

import { expect, test } from '@playwright/test';

import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';

/** The published minimum, and the comfortable target beside it. */
const WIDTHS = [
  { name: 'minimum', width: 480 },
  { name: 'comfortable', width: 768 },
];

/** Every page in pageRegistry, plus the dashboard. A subset would only prove
 *  the subset. */
const PAGES = [
  { name: 'dashboard', path: '/' },
  { name: 'link', path: '/link' },
  { name: 'network', path: '/network' },
  { name: 'path', path: '/path' },
  { name: 'wifi', path: '/wifi' },
  { name: 'security', path: '/security' },
  { name: 'performance', path: '/performance' },
  { name: 'reports', path: '/reports' },
  { name: 'logs', path: '/logs' },
  { name: 'polling-targets', path: '/polling-targets' },
  { name: 'topology', path: '/topology' },
  { name: 'alerts', path: '/alerts' },
];

test.use({ storageState: AUTH_STORAGE_STATE });

for (const { name: widthName, width } of WIDTHS) {
  test.describe(`at the ${widthName} width (${width}px)`, () => {
    for (const page_ of PAGES) {
      test(`${page_.name} does not scroll sideways`, async ({ page }) => {
        await disableAnimations(page);
        await page.setViewportSize({ width, height: 900 });
        await page.goto(page_.path, { waitUntil: 'domcontentloaded' });
        await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });

        // scrollWidth beyond clientWidth is the definition of sideways scroll.
        // Compared on documentElement rather than body: body can be narrower
        // than its overflowing children and report no overflow at all.
        const overflow = await page.evaluate(() => {
          const root = document.documentElement;

          return {
            scrollWidth: root.scrollWidth,
            clientWidth: root.clientWidth,
            // The widest offending element, so a failure names what to fix
            // rather than only that something is too wide.
            widest: Array.from(document.querySelectorAll('*'))
              .map((el) => ({
                selector: `${el.tagName.toLowerCase()}${el.className && typeof el.className === 'string' ? `.${el.className.split(' ').slice(0, 3).join('.')}` : ''}`,
                right: el.getBoundingClientRect().right,
              }))
              .filter((entry) => entry.right > root.clientWidth + 1)
              .sort((a, b) => b.right - a.right)
              .slice(0, 3),
          };
        });

        expect(
          overflow.scrollWidth,
          `${page_.name} overflows ${width}px by ${overflow.scrollWidth - overflow.clientWidth}px. ` +
            `Widest offenders: ${JSON.stringify(overflow.widest)}`,
        ).toBeLessThanOrEqual(overflow.clientWidth + 1);
      });
    }
  });
}

/**
 * Touch targets at the minimum width.
 *
 * The other half of #244's ask. WCAG 2.5.8 (AA, 2.2) puts the floor at 24x24
 * CSS pixels; this asserts that rather than the 44px iOS guideline, because 24
 * is the standard the accessibility gate already holds the rest of the UI to
 * and inventing a stricter local rule would be a different argument.
 *
 * Two exemptions, both from the spec and both implemented rather than assumed,
 * because the first run found examples of each:
 *
 *  - **Inline.** WCAG 2.5.8 exempts a target inside a sentence, because
 *    enlarging it would break the text. The support address in the footer is
 *    one; padding a mailto link to 24px tall would put a gap in the paragraph.
 *  - **Visually hidden until focused.** The skip link measures 1x1 while
 *    hidden. What matters is its size when it is reachable, so it is measured
 *    focused, not parked.
 *
 * And one correction to what gets measured: a control wrapped in its own
 * `<label>` is targeted by the whole label, not the 13x13 native checkbox
 * inside it. Measuring the input alone reported failures for controls that are
 * comfortably tappable, and "fix" would have meant inflating checkboxes to
 * 24x24 for no benefit.
 */
test.describe('at the minimum width (480px)', () => {
  for (const page_ of PAGES) {
    test(`${page_.name} keeps its controls tappable`, async ({ page }) => {
      await disableAnimations(page);
      await page.setViewportSize({ width: 480, height: 900 });
      await page.goto(page_.path, { waitUntil: 'domcontentloaded' });
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });

      const tooSmall = await page.evaluate(() => {
        const MINIMUM = 24;

        /** A control rendered inline inside running text (WCAG 2.5.8 exempt). */
        const isInlineInText = (el: Element): boolean => {
          const { display } = window.getComputedStyle(el);
          if (display !== 'inline' && display !== 'inline-block') {
            return false;
          }
          const parent = el.parentElement;
          if (parent === null) {
            return false;
          }

          // Text either side of the control is what makes it "in a sentence"
          // rather than a button that merely happens to be inline.
          return Array.from(parent.childNodes).some(
            (node) => node.nodeType === Node.TEXT_NODE && (node.textContent ?? '').trim() !== '',
          );
        };

        /** Visually hidden until focused — the skip-link pattern. */
        const isHiddenUntilFocused = (el: Element): boolean => {
          const style = window.getComputedStyle(el);

          return (
            style.clipPath !== 'none' ||
            style.clip !== 'auto' ||
            (style.position === 'absolute' && style.overflow === 'hidden')
          );
        };

        /** The box a finger actually has to hit. */
        const effectiveTarget = (el: Element): DOMRect => {
          // A control inside its own <label> is targeted by the whole label:
          // clicking the text toggles it. Measuring the bare input reports a
          // 13x13 checkbox as a failure when the thing you tap is far larger.
          const label = el.closest('label');
          if (label?.contains(el) === true) {
            return label.getBoundingClientRect();
          }

          return el.getBoundingClientRect();
        };

        return Array.from(document.querySelectorAll('button, a[href], input, select'))
          .filter((el) => {
            const style = window.getComputedStyle(el);
            if (style.display === 'none' || style.visibility === 'hidden') {
              return false;
            }
            const box = effectiveTarget(el);

            // A zero box is a control that is not laid out — collapsed panel,
            // closed drawer. Not a touch-target failure.
            if (box.width === 0 && box.height === 0) {
              return false;
            }
            if (isInlineInText(el) || isHiddenUntilFocused(el)) {
              return false;
            }

            return box.width < MINIMUM || box.height < MINIMUM;
          })
          .map((el) => ({
            tag: el.tagName.toLowerCase(),
            label: (
              el.getAttribute('aria-label') ??
              el.closest('label')?.textContent ??
              el.textContent ??
              ''
            )
              .trim()
              .slice(0, 40),
            width: Math.round(effectiveTarget(el).width),
            height: Math.round(effectiveTarget(el).height),
          }))
          .slice(0, 8);
      });

      expect(
        tooSmall,
        `${page_.name} has controls under 24x24 at 480px: ${JSON.stringify(tooSmall, null, 2)}`,
      ).toEqual([]);
    });
  }
});
