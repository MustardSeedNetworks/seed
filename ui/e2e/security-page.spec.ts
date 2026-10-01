import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';
import { mockBluetoothScanJob } from './helpers/bluetooth';

/**
 * Security Page (/security) E2E
 *
 * Covers the shell module's security posture surface:
 * - Page renders with the proper heading
 * - MFA card and Guest Network Audit card slots are present
 * - Bluetooth card scans (via the jobs spine) and surfaces decoded devices
 */

test.describe('Security Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/security');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Security title', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await expect(page.getByTestId('page-header-description')).toBeVisible();
  });

  test('should land on the /security route', async ({ page }) => {
    await expect(page).toHaveURL(/\/security$/);
  });

  // #2641: the MFA card's buttons carried `btn btn-secondary`, a class defined
  // in no stylesheet, and the passkey button's `display: contents` tooltip
  // wrapper swallowed the stack margin, so the two ran together as one word.
  test('should render the MFA card as two separated buttons', async ({ page }, testInfo) => {
    const totp = page.getByTestId('mfa-setup-totp');
    const passkey = page.getByTestId('mfa-add-passkey');
    await expect(totp).toBeVisible({ timeout: 5000 });
    await expect(passkey).toBeVisible();

    const totpBox = await totp.boundingBox();
    const passkeyBox = await passkey.boundingBox();
    if (!totpBox || !passkeyBox) {
      throw new Error('MFA enrolment buttons are not laid out');
    }
    // Separated, and each is a real control rather than bare text: an unstyled
    // button has no padding, so the box is exactly the text it contains. The row
    // wraps on a narrow card, so the separation is whichever axis they differ on.
    const horizontal = passkeyBox.x - (totpBox.x + totpBox.width);
    const vertical = passkeyBox.y - (totpBox.y + totpBox.height);
    expect(Math.max(horizontal, vertical)).toBeGreaterThanOrEqual(8);
    expect(totpBox.height).toBeGreaterThanOrEqual(28);

    // The row asks for a shot of /security showing two distinct buttons. It is
    // attached as evidence rather than asserted: this suite has no visual
    // baselines, and adding one gate's worth of pixel comparison across three
    // browsers is a different change.
    await testInfo.attach('security-mfa-card.png', {
      body: await page.getByTestId('mfa-setup-totp').locator('xpath=../..').screenshot(),
      contentType: 'image/png',
    });
  });

  test('should render the Guest Network Audit card', async ({ page }) => {
    await expect(page.locator('text=/guest.*network|guest.*audit/i').first()).toBeVisible({
      timeout: 5000,
    });
  });

  test('should render the Bluetooth card with a scan button', async ({ page }) => {
    await expect(page.getByTestId('bluetooth-scan-button')).toBeVisible({ timeout: 5000 });
    // The maximize-to-modal control is disabled until a scan finds devices.
    await expect(page.getByTestId('bluetooth-card-maximize')).toBeDisabled();
  });

  test('should scan and show decoded devices in the full-screen modal', async ({ page }) => {
    await mockBluetoothScanJob(page);

    await page.getByTestId('bluetooth-scan-button').click();

    // Card summarizes the found devices once the job result lands.
    await expect(page.getByTestId('bluetooth-device-count')).toContainText('2', {
      timeout: 5000,
    });

    // Open the full-screen device table and confirm the decoded fields render.
    await page.getByTestId('bluetooth-card-maximize').click();
    const modal = page.getByTestId('bluetooth-modal');
    await expect(modal).toBeVisible();
    await expect(modal).toContainText('AirPods Pro');
    await expect(modal).toContainText('Apple'); // decoded companyName column
    await expect(modal).toContainText('Fitbit Charge');
  });
});
