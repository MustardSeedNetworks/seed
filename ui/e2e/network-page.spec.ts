import { expect, test } from '@playwright/test';
import { mockAuthenticated } from './helpers/auth';

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
    await mockAuthenticated(page);
    await page.goto('/network');
    await expect(page.getByRole('heading', { name: /^network$/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Network title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /^network$/i, level: 1 })).toBeVisible();
    await expect(page.getByText(/dhcp.*gateway.*dns/i)).toBeVisible();
  });

  test('should land on the /network route', async ({ page }) => {
    await expect(page).toHaveURL(/\/network$/);
  });

  test('should render at least one network-config card', async ({ page }) => {
    const cards = page.locator('text=/dhcp|gateway|dns|public.*ip|switch/i');
    await expect(cards.first()).toBeVisible({ timeout: 5000 });
  });
});
