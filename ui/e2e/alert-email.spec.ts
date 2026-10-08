import { expect, type Page, test } from '@playwright/test';
import { sidebarSettingsButton, skipSetupWizard } from './helpers/auth';

/**
 * Alert email relay (#3209), against the E2E daemon.
 *
 * The section saves alerts.email through PUT /api/v1/settings, which the
 * running daemon applies without a restart. openEmail navigates afresh, so
 * each reopen reads back what the server stored. A relay that could never
 * deliver is refused with its reason, shown as given; the password is never
 * served back.
 *
 * The suite's daemon is shared, so the test ends with email turned off.
 */

async function openEmail(page: Page): Promise<void> {
  await page.goto('/');
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
  await sidebarSettingsButton(page).click();
  await expect(page.getByTestId('settings-drawer')).toBeVisible({ timeout: 5000 });
  const section = page.getByTestId('alert-email-section');
  await section.getByRole('button', { expanded: false }).first().click();
  await expect(page.getByTestId('alert-email-save')).toBeVisible();
}

async function save(page: Page): Promise<void> {
  const put = page.waitForResponse(
    (r) => r.request().method() === 'PUT' && r.url().endsWith('/api/v1/settings'),
  );
  await page.getByTestId('alert-email-save').click();
  await put;
}

test.describe('Alert email', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('an operator configures, edits and turns off the relay', async ({ page }) => {
    // Three full loads of the Settings drawer; webkit on CI needs ~7 s each.
    test.slow();
    await openEmail(page);
    await expect(page.getByTestId('alert-email-host')).toHaveValue('');

    // A relay with nobody to send to could not deliver: nothing is stored.
    await page.getByTestId('alert-email-host').fill('smtp.example.test');
    await page.getByTestId('alert-email-from').fill('seed@example.test');
    await save(page);
    await expect(page.getByTestId('alert-email-error')).toContainText(
      'at least one recipient is required',
    );

    await page.getByTestId('alert-email-username').fill('seed');
    await page.getByTestId('alert-email-password').fill('relay-password');
    await page.getByTestId('alert-email-to').fill('noc@example.test, oncall@example.test');
    await save(page);
    await expect(page.getByTestId('alert-email-error')).toHaveCount(0);

    await openEmail(page);
    await expect(page.getByTestId('alert-email-host')).toHaveValue('smtp.example.test');
    await expect(page.getByTestId('alert-email-tls')).toHaveValue('starttls');
    await expect(page.getByTestId('alert-email-username')).toHaveValue('seed');
    await expect(page.getByTestId('alert-email-password')).toHaveValue('');
    await expect(page.getByTestId('alert-email-password')).toHaveAttribute(
      'placeholder',
      'A password is stored; type to replace it',
    );
    await expect(page.getByTestId('alert-email-to')).toHaveValue(
      'noc@example.test, oncall@example.test',
    );

    // Edit: implicit TLS on a custom port, keeping the stored password.
    await page.getByTestId('alert-email-tls').selectOption('tls');
    await page.getByTestId('alert-email-port').fill('2465');
    await save(page);

    await openEmail(page);
    await expect(page.getByTestId('alert-email-tls')).toHaveValue('tls');
    await expect(page.getByTestId('alert-email-port')).toHaveValue('2465');
    await expect(page.getByTestId('alert-email-password')).toHaveAttribute(
      'placeholder',
      'A password is stored; type to replace it',
    );

    // Clearing the server turns email off and takes the password with it.
    await page.getByTestId('alert-email-host').fill('');
    await save(page);
    await expect(page.getByTestId('alert-email-error')).toHaveCount(0);

    await openEmail(page);
    await expect(page.getByTestId('alert-email-host')).toHaveValue('');
    await expect(page.getByTestId('alert-email-password')).toHaveAttribute('placeholder', '');
  });
});
