import { mkdirSync } from 'node:fs';
import { test } from '@playwright/test';
import { sidebarSettingsButton } from './helpers/auth';

// Scratch capture script for UI-SEED-50 evidence. Not committed.
const OUT = process.env.SHOTS_DIR ?? '/tmp/shots';
const ROUTES = [
  'dashboard',
  'link',
  'network',
  'path',
  'wifi',
  'security',
  'performance',
  'reports',
  'logs',
  'polling-targets',
  'device-groups',
  'ipam',
  'lifecycle',
  'ip-sla',
  'topology',
  'interfaces',
  'device-configs',
  'flows',
  'alerts',
];
const SIZES = [
  { w: 1440, h: 900 },
  { w: 390, h: 844 },
] as const;

for (const theme of ['dark', 'light'] as const) {
  test(`capture ${theme}`, async ({ browser }) => {
    test.setTimeout(600_000);
    mkdirSync(OUT, { recursive: true });
    const context = await browser.newContext({
      colorScheme: theme,
      ignoreHTTPSErrors: true,
      storageState: 'playwright/.auth/user.json',
    });
    const page = await context.newPage();
    const tag = theme === 'light' ? '-light' : '';
    for (const { w, h } of SIZES) {
      await page.setViewportSize({ width: w, height: h });
      for (const route of [...ROUTES, 'zz-not-a-route']) {
        await page.goto(`/${route}`);
        await page.waitForTimeout(900);
        await page.screenshot({ path: `${OUT}/${route}${tag}-${w}.png` });
      }
      await page.goto('/network');
      await page.getByTestId('page-header-help-button').click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: `${OUT}/help-drawer${tag}-${w}.png` });
      await page.keyboard.press('Escape');
      await page.goto('/link');
      if (w < 768) {
        await page.getByTestId('mobile-menu-toggle').click();
      }
      await sidebarSettingsButton(page).click();
      await page.waitForTimeout(700);
      await page.screenshot({ path: `${OUT}/settings-drawer${tag}-${w}.png` });
      for (const tab of ['Network', 'Tests', 'Access']) {
        await page.getByRole('tab', { name: tab }).click();
        await page.waitForTimeout(400);
        await page.screenshot({ path: `${OUT}/settings-${tab.toLowerCase()}${tag}-${w}.png` });
      }
      await page.keyboard.press('Escape');
    }
    await context.close();
  });
}
