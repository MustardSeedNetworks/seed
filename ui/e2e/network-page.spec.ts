import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Network Page (/network) E2E
 *
 * Covers the sap module's network-config surface:
 * - DHCP / NetworkCard
 * - GatewayCard
 * - DnsCard
 * - PublicIpCard
 * - SwitchCard (LLDP/CDP)
 */

test.describe('Network Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/network');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Network title', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await expect(page.getByTestId('page-header-description')).toBeVisible();
  });

  test('should land on the /network route', async ({ page }) => {
    await expect(page).toHaveURL(/\/network$/);
  });

  test('should render at least one network-config card', async ({ page }) => {
    const cards = page.locator('text=/dhcp|gateway|dns|public.*ip|switch/i');
    await expect(cards.first()).toBeVisible({ timeout: 5000 });
  });

  // #123: interface, MAC, vendor, mode and the lease sit behind one control.
  // Mode is the one row every response carries (the E2E daemon runs on lo).
  test('opens and closes the Network card Details group', async ({ page }) => {
    const toggle = page.getByTestId('network-card-details').getByRole('button');
    await expect(toggle).toHaveAttribute('aria-expanded', 'false', { timeout: 10000 });

    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');
    const body = page.locator(`[id="${await toggle.getAttribute('aria-controls')}"]`);
    await expect(body.getByText('Mode', { exact: true })).toBeVisible();

    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
    await expect(body).toHaveCount(0);
  });
});
