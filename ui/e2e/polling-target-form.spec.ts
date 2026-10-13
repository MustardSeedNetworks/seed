import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * The polling-target dialog by keyboard alone (seed#2524).
 *
 * It was a bare overlay: no focus trap, no Escape, no dialog name, and labels
 * that were spans, so a keyboard user tabbed straight out of the form into the
 * page behind it. Runs against the real daemon; nothing is saved.
 */
test('opens, tabs through every control in order, and escapes by keyboard', async ({ page }) => {
  await skipSetupWizard(page);
  await page.goto('/polling-targets');

  const add = page.getByTestId('target-add');
  await expect(add).toBeEnabled({ timeout: 10000 });
  await add.focus();
  await page.keyboard.press('Enter');

  const dialog = page.getByRole('dialog', { name: 'Add polling target' });
  await expect(dialog).toBeVisible();
  // The credential select is disabled until the vault list answers, and a
  // disabled control is skipped by Tab.
  await expect(dialog.getByLabel('Credential', { exact: true })).toBeEnabled();

  const close = dialog.getByRole('button', { name: 'Close modal' });
  const rest = [
    dialog.getByLabel('Name', { exact: true }),
    dialog.getByLabel('IP address', { exact: true }),
    dialog.getByLabel('SNMP version', { exact: true }),
    dialog.getByLabel('Credential', { exact: true }),
    dialog.getByLabel('Poll interval (seconds)', { exact: true }),
    dialog.getByRole('checkbox'),
    dialog.getByRole('button', { name: 'Cancel' }),
    dialog.getByRole('button', { name: 'Add target' }),
  ];
  await expect(close).toBeFocused();
  for (const control of rest) {
    await page.keyboard.press('Tab');
    await expect(control).toBeFocused();
  }
  // The trap: past the last control, focus wraps rather than leaving.
  await page.keyboard.press('Tab');
  await expect(close).toBeFocused();

  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
  await expect(add).toBeFocused();
});
