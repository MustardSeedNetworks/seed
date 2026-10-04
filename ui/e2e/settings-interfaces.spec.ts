import { expect, type Page, test } from '@playwright/test';
import { sidebarSettingsButton, skipSetupWizard } from './helpers/auth';

/**
 * Settings → Interfaces: an added interface shows in the list at once
 * (UI-SEED-41, #3066).
 *
 * The section reads its lists through getters it calls while rendering.
 * Those getters used to read a ref and never change identity, so the React
 * Compiler cached their first result and the list stayed as it was until the
 * drawer remounted.
 *
 * The active profile is served from a stateful mock that starts with no
 * interfaces and applies each PUT, so the test needs neither a multi-interface
 * licence nor a change to the shared daemon's profile.
 */

interface ProfileBody {
  id: string;
  config?: { interfaces?: unknown };
}

async function serveActiveProfileWithoutInterfaces(page: Page): Promise<void> {
  let profile: ProfileBody | null = null;

  await page.route('**/api/v1/profiles/active', async (route) => {
    if (!profile) {
      const real = (await (await route.fetch()).json()) as ProfileBody;
      profile = { ...real, config: { ...real.config, interfaces: { ethernet: [], wifi: [] } } };
    }
    await route.fulfill({ json: profile });
  });

  await page.route(/\/api\/v1\/profiles\/[^/]+$/, async (route) => {
    if (route.request().method() !== 'PUT' || !profile) {
      await route.fallback();
      return;
    }
    const body = route.request().postDataJSON() as { config: ProfileBody['config'] };
    profile = { ...profile, config: body.config };
    await route.fulfill({ json: profile });
  });
}

test.describe('Settings — interfaces', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await serveActiveProfileWithoutInterfaces(page);
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await sidebarSettingsButton(page).click();
    await expect(page.getByTestId('settings-drawer')).toBeVisible({ timeout: 5000 });
  });

  test('an added interface appears in the list without reopening Settings', async ({ page }) => {
    const section = page.getByTestId('interfaces-settings');
    await section.getByRole('button', { expanded: false }).click();
    const ethernet = page.getByTestId('interface-group-ethernet');
    await expect(ethernet.getByRole('listitem')).toHaveCount(0);

    await page.getByTestId('add-ethernet-input').fill('e2e-eth9');
    await page.getByTestId('add-ethernet-button').click();

    await expect(page.getByTestId('interface-row-ethernet-e2e-eth9')).toBeVisible();
    await expect(page.getByTestId('add-ethernet-input')).toHaveValue('');
  });
});
