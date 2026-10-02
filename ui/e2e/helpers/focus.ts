/**
 * `toBeFocused` with the evidence a failure needs. Playwright reports
 * "inactive" both when another element holds focus and when the page has
 * lost window focus (`document.hasFocus()` false), and a trace records
 * neither. seed#2944 failed that way on WebKit in CI and never reproduced
 * locally, so a failure says which of the two it was.
 */
import { expect, type Locator } from '@playwright/test';

export async function expectFocused(target: Locator): Promise<void> {
  try {
    await expect(target).toBeFocused();
  } catch (error) {
    const state = await target.page().evaluate(() => {
      const active = document.activeElement;
      const testId = active?.getAttribute('data-testid');
      return {
        activeElement: active
          ? `<${active.tagName.toLowerCase()}${testId ? ` data-testid="${testId}"` : ''}>`
          : 'null',
        hasFocus: document.hasFocus(),
      };
    });
    if (error instanceof Error) {
      error.message += `\n\ndocument.activeElement: ${state.activeElement}\ndocument.hasFocus(): ${state.hasFocus}`;
    }
    throw error;
  }
}
