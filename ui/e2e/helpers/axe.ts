/**
 * axe in a real browser, for the a11y specs (#388). The page must have loaded
 * `axe.source` through `page.addInitScript` first.
 *
 * color-contrast is off for the reason `help-closeout.spec.ts` gives: it
 * belongs to the theme row (UI-SEED-7, seed#2647) with its own acceptance.
 */
import { type Page, test } from '@playwright/test';
import type axe from 'axe-core';

declare global {
  interface Window {
    /** Installed by each spec's init script from `axe.source`. */
    axe: typeof axe;
  }
}

type Violation = { id: string; impact: string | null; nodes: string[] };

/** `within` scopes the run to an overlay whose page is already walked. */
export async function axeViolations(page: Page, within?: string): Promise<Violation[]> {
  const violations = await page.evaluate(async (selector) => {
    const report = await window.axe.run(selector ? { include: [[selector]] } : document, {
      runOnly: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'],
      rules: { 'color-contrast': { enabled: false } },
    });
    return report.violations.map(({ id, impact, nodes }) => ({
      id,
      impact: impact ?? null,
      nodes: nodes.map(({ html }) => html.slice(0, 200)),
    }));
  }, within);
  await test.info().attach('axe-violations.json', {
    body: JSON.stringify(violations, null, 2),
    contentType: 'application/json',
  });
  return violations;
}
